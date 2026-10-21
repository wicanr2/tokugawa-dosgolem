package main

import (
	"io"
	"sync"
)

// 主機依音效裝置消費樣本，DOS 裝置仍依自己的虛擬時間前進。
type cdStream struct {
	sync.Mutex
	disc                                                         *cueDisc
	position, end, frames, silence, commands, virtualCompletions uint64
	playing, paused, closed                                      bool
	pending                                                      [4]byte
	pendingOffset, pendingBytes                                  int
}

func newCDStream(disc *cueDisc) *cdStream { return &cdStream{disc: disc} }

func (s *cdStream) command(kind string, start, end uint64) {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	switch kind {
	case "play":
		s.position, s.end = start, end
		s.playing, s.paused = true, false
		s.pendingOffset, s.pendingBytes = 0, 0
		s.commands++
	case "pause":
		s.playing, s.paused = false, true
		s.commands++
	case "idle_stop":
		s.playing, s.paused = false, false
		s.pendingOffset, s.pendingBytes = 0, 0
		s.commands++
	case "resume":
		if s.paused && (s.position < s.end || s.pendingBytes > 0) {
			s.playing, s.paused = true, false
		}
		s.commands++
	case "complete":
		// 加速 guest 提早完成不截斷主機尚未消費的音樂。
		s.virtualCompletions++
	}
}

func (s *cdStream) Read(data []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return 0, io.EOF
	}
	for i := 0; i < len(data); {
		if !s.playing {
			clear(data[i:])
			s.silence += uint64(len(data) - i)
			return len(data), nil
		}
		if s.pendingBytes > 0 {
			n := copy(data[i:], s.pending[s.pendingOffset:s.pendingOffset+s.pendingBytes])
			i += n
			s.pendingOffset += n
			s.pendingBytes -= n
			continue
		}
		if s.position >= s.end {
			s.playing = false
			continue
		}
		offset, span, ok := s.disc.audioSpan(s.position)
		if !ok {
			return i, io.ErrUnexpectedEOF
		}
		count := min(span, s.end-s.position, uint64((len(data)-i)/4))
		dst := data[i:]
		partial := count == 0
		if partial {
			count, dst = 1, s.pending[:]
		} else {
			dst = dst[:int(count)*4]
		}
		if offset < 0 {
			clear(dst)
		} else if _, err := s.disc.File.ReadAt(dst, offset); err != nil {
			return i, err
		}
		s.position += count
		s.frames += count
		if partial {
			s.pendingOffset, s.pendingBytes = 0, 4
		} else {
			i += len(dst)
		}
	}
	return len(data), nil
}

func (s *cdStream) Close() error { s.Lock(); defer s.Unlock(); s.closed = true; return nil }

func (s *cdStream) report() map[string]any {
	s.Lock()
	defer s.Unlock()
	return map[string]any{"clock": "audio device; independent of guest instructions", "source_frames": s.frames,
		"disc_sample_frame": s.position, "end_sample_frame": s.end, "source_playing": s.playing,
		"source_paused": s.paused, "idle_silent_bytes": s.silence, "transport_commands": s.commands,
		"guest_completions_ignored": s.virtualCompletions, "dropped_oldest_frames": uint64(0)}
}
