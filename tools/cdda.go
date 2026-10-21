package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const cdSampleRate = uint64(44100)
const cdFramesPerSector = uint64(588)

// 主機播放器的有界樣本框佇列。丟棄完整框，不改 DOS 的 PCM 來源或時間。
type pcmQueue struct {
	sync.Mutex
	frames                              [][4]byte
	head, count, offset                 int
	current                             [4]byte
	closed                              bool
	received, consumed, silent, dropped uint64
}

func newPCMQueue(capacity int) *pcmQueue {
	if capacity < 1 {
		panic("invalid PCM capacity")
	}
	return &pcmQueue{frames: make([][4]byte, capacity), offset: 4}
}

func (q *pcmQueue) Write(data []byte) (int, error) {
	q.Lock()
	defer q.Unlock()
	if q.closed {
		return 0, io.ErrClosedPipe
	}
	if len(data)%4 != 0 {
		return 0, fmt.Errorf("PCM write must contain complete stereo frames")
	}
	for i := 0; i < len(data); i += 4 {
		if q.count == len(q.frames) {
			q.head = (q.head + 1) % len(q.frames)
			q.count--
			q.dropped++
		}
		copy(q.frames[(q.head+q.count)%len(q.frames)][:], data[i:i+4])
		q.count++
		q.received++
	}
	return len(data), nil
}

func (q *pcmQueue) Read(data []byte) (int, error) {
	q.Lock()
	defer q.Unlock()
	for i := 0; i < len(data); {
		if q.offset == 4 {
			if q.count > 0 {
				q.current = q.frames[q.head]
				q.head = (q.head + 1) % len(q.frames)
				q.count--
				q.consumed++
			} else {
				if q.closed {
					return i, io.EOF
				}
				q.current = [4]byte{}
				q.silent++
			}
			q.offset = 0
		}
		n := copy(data[i:], q.current[q.offset:])
		i += n
		q.offset += n
	}
	return len(data), nil
}

func (q *pcmQueue) Close() error { q.Lock(); q.closed = true; q.Unlock(); return nil }
func (q *pcmQueue) report() map[string]any {
	q.Lock()
	defer q.Unlock()
	return map[string]any{"frame_capacity": len(q.frames), "queued_frames": q.count, "received_frames": q.received,
		"consumed_source_frames": q.consumed, "underflow_silent_frames": q.silent, "dropped_oldest_frames": q.dropped}
}

type cdAudio struct {
	NowUS          uint64                       `json:"virtual_us"`
	Position       uint64                       `json:"disc_sample_frame"`
	End            uint64                       `json:"end_sample_frame"`
	LastStart      uint64                       `json:"last_start_sample_frame"`
	RequestMode    byte                         `json:"request_address_mode"`
	RequestStart   uint32                       `json:"request_raw_start"`
	RequestSectors uint32                       `json:"request_sectors"`
	RequestedEnd   uint64                       `json:"requested_end_sample_frame"`
	ClippedAtEnd   bool                         `json:"clipped_at_lead_out"`
	Playing        bool                         `json:"playing"`
	Paused         bool                         `json:"paused"`
	Fraction       uint64                       `json:"fractional_sample_millionths"`
	Sink           io.Writer                    `json:"-"`
	HostEvent      func(string, uint64, uint64) `json:"-"`
	Frames         uint64                       `json:"output_stereo_frames"`
	NonzeroSamples uint64                       `json:"nonzero_channel_samples"`
	Peak           uint32                       `json:"absolute_sample_peak"`
	Events         []map[string]any             `json:"events"`
	DroppedEvents  uint64                       `json:"dropped_events"`
}

func (a *cdAudio) event(kind string) {
	if a.HostEvent != nil {
		a.HostEvent(kind, a.Position, a.End)
	}
	if len(a.Events) >= 256 {
		a.DroppedEvents++
		return
	}
	a.Events = append(a.Events, map[string]any{"kind": kind, "virtual_us": a.NowUS, "output_frame": a.Frames, "disc_sample_frame": a.Position, "end_sample_frame": a.End,
		"request_address_mode": a.RequestMode, "request_raw_start": a.RequestStart, "request_sectors": a.RequestSectors,
		"requested_end_sample_frame": a.RequestedEnd, "clipped_at_lead_out": a.ClippedAtEnd})
}

// 音軌邊界以 disc sector 計算，檔案位址以 file sector 計算。
// offset=-1 表示顯式 AUDIO PREGAP，依 CUE 契約輸出靜音。
func (d *cueDisc) audioSpan(frame uint64) (offset int64, available uint64, ok bool) {
	for i, t := range d.Tracks {
		start := uint64(t.DiscSector) * cdFramesPerSector
		gapStart := uint64(t.DiscSector-t.Pregap) * cdFramesPerSector
		if t.Kind == "AUDIO" && frame >= gapStart && frame < start {
			return -1, start - frame, true
		}
		endSector := d.LeadOut
		if i+1 < len(d.Tracks) {
			endSector = d.Tracks[i+1].DiscSector - d.Tracks[i+1].Pregap
		}
		end := uint64(endSector) * cdFramesPerSector
		if frame >= start && frame < end {
			if t.Kind != "AUDIO" {
				return 0, 0, false
			}
			return int64(uint64(t.FileSector)*2352 + (frame-start)*4), end - frame, true
		}
	}
	return 0, 0, false
}

func (d *cueDisc) validAudioRange(start, end uint64) bool {
	if start >= end || end > uint64(d.LeadOut)*cdFramesPerSector {
		return false
	}
	for start < end {
		_, n, ok := d.audioSpan(start)
		if !ok {
			return false
		}
		if n > end-start {
			n = end - start
		}
		start += n
	}
	return true
}

func (d *cueDisc) advanceAudio(now uint64) error {
	a := &d.Audio
	if now < a.NowUS {
		return fmt.Errorf("CD virtual clock moved backwards")
	}
	delta := now - a.NowUS
	previousUS, previousFraction := a.NowUS, a.Fraction
	a.NowUS = now
	if !a.Playing {
		return nil
	}
	remaining := a.End - a.Position
	if delta > (^uint64(0)-a.Fraction)/cdSampleRate {
		return fmt.Errorf("CD clock overflow")
	}
	units := delta*cdSampleRate + a.Fraction
	n := units / 1000000
	a.Fraction = units % 1000000
	if n > a.End-a.Position {
		n = a.End - a.Position
	}
	var buffer [16384]byte
	for n > 0 {
		offset, span, ok := d.audioSpan(a.Position)
		if !ok {
			return fmt.Errorf("CD position outside supported audio span")
		}
		count := n
		if count > span {
			count = span
		}
		if count > uint64(len(buffer)/4) {
			count = uint64(len(buffer) / 4)
		}
		pcm := buffer[:int(count)*4]
		if offset < 0 {
			clear(pcm)
		} else {
			if _, err := d.File.ReadAt(pcm, offset); err != nil {
				return err
			}
		}
		if a.Sink != nil {
			written, err := a.Sink.Write(pcm)
			if err != nil {
				return err
			}
			if written != len(pcm) {
				return io.ErrShortWrite
			}
		}
		for i := 0; i < len(pcm); i += 2 {
			v := int32(int16(binary.LittleEndian.Uint16(pcm[i:])))
			if v != 0 {
				a.NonzeroSamples++
			}
			if v < 0 {
				v = -v
			}
			if uint32(v) > a.Peak {
				a.Peak = uint32(v)
			}
		}
		a.Position += count
		a.Frames += count
		n -= count
	}
	if a.Position == a.End {
		a.NowUS = previousUS + (remaining*1000000-previousFraction+cdSampleRate-1)/cdSampleRate
		a.event("complete")
		a.NowUS = now
		a.Playing = false
		a.Paused = false
		a.LastStart = 0
		a.End = 0
		a.Fraction = 0
	}
	return nil
}

func (d *cueDisc) playAudio(mode byte, raw, sectors uint32) uint16 {
	start := uint64(raw)
	if mode > 1 {
		return 0x8108
	}
	if sectors == 0 {
		return 0x8000
	}
	if mode == 1 {
		f, s, m := uint64(byte(raw)), uint64(byte(raw>>8)), uint64(byte(raw>>16))
		if f >= 75 || s >= 60 || m*4500+s*75+f < 150 {
			return 0x8108
		}
		start = m*4500 + s*75 + f - 150
	}
	requestedEnd := start + uint64(sectors)
	end := requestedEnd
	if end > uint64(d.LeadOut) {
		end = uint64(d.LeadOut)
	}
	if !d.validAudioRange(start*cdFramesPerSector, end*cdFramesPerSector) {
		return 0x8108
	}
	a := &d.Audio
	if a.Playing || a.Paused {
		a.event("replace")
	}
	a.Position = start * cdFramesPerSector
	a.LastStart = a.Position
	a.End = end * cdFramesPerSector
	a.RequestMode, a.RequestStart, a.RequestSectors = mode, raw, sectors
	a.RequestedEnd = requestedEnd * cdFramesPerSector
	a.ClippedAtEnd = requestedEnd != end
	a.Playing = true
	a.Paused = false
	a.Fraction = 0
	a.event("play")
	return 0x0300
}

func (d *cueDisc) stopAudio() uint16 {
	a := &d.Audio
	if a.Playing {
		a.Playing = false
		a.Paused = true
		a.LastStart = a.Position
		a.event("pause")
	} else {
		a.Paused = false
		a.LastStart = 0
		a.End = 0
		a.Fraction = 0
		d.AudioIdleStopCount++
		a.event("idle_stop")
	}
	return 0x0100
}

func (d *cueDisc) resumeAudio() uint16 {
	a := &d.Audio
	if !a.Paused {
		return 0x810c
	}
	a.Playing = true
	a.Paused = false
	a.event("resume")
	return 0x0300
}

func (d *cueDisc) audioStatusWord() []byte {
	status := byte(1)
	if d.Audio.Playing {
		status |= 2
	}
	return []byte{0, status}
}

func (d *cueDisc) audioStatusInfo() []byte {
	a := &d.Audio
	out := make([]byte, 11)
	out[0] = 15
	if a.Paused {
		out[1] = 1
	}
	if a.Playing || a.Paused {
		copy(out[3:], redBook(uint32(a.LastStart/cdFramesPerSector)))
		copy(out[7:], redBook(uint32(a.End/cdFramesPerSector)))
	}
	return out
}
