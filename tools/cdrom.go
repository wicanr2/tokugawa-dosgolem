package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/cpu386"
)

type cueTrack struct {
	Number     int    `json:"number"`
	Kind       string `json:"kind"`
	FileSector uint32 `json:"file_sector"`
	Pregap     uint32 `json:"pregap_sectors"`
	DiscSector uint32 `json:"disc_sector"`
}

type cueDisc struct {
	File                *os.File         `json:"-"`
	Tracks              []cueTrack       `json:"tracks"`
	RawSectors          uint32           `json:"raw_sectors"`
	VolumeSectors       uint32           `json:"volume_sectors"`
	RootSector          uint32           `json:"root_sector"`
	RootBytes           uint32           `json:"root_bytes"`
	SHA256              string           `json:"bin_sha256"`
	LeadOut             uint32           `json:"lead_out_lba"`
	AudioRejections     []map[string]any `json:"audio_rejections"`
	AudioRejectionCount uint64           `json:"audio_rejection_count"`
	AudioIdleStopCount  uint64           `json:"audio_idle_stop_count"`
}

func cueFrames(value string) (uint32, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid CUE time")
	}
	values := [3]uint64{}
	for i, p := range parts {
		v, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return 0, err
		}
		values[i] = v
	}
	if values[1] >= 60 || values[2] >= 75 {
		return 0, fmt.Errorf("invalid CUE second/frame")
	}
	return uint32((values[0]*60+values[1])*75 + values[2]), nil
}

// 本切片只支援單一 BINARY 檔、MODE1/2352 第一軌與後續 AUDIO 軌。
func openCueDisc(path string) (*cueDisc, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d := &cueDisc{}
	name := ""
	indexed := []bool{}
	for _, line := range strings.Split(string(text), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "FILE":
			if name != "" || len(fields) != 3 || fields[2] != "BINARY" {
				return nil, fmt.Errorf("unsupported CUE FILE")
			}
			name = strings.Trim(fields[1], "\"")
			if name == "" || strings.ContainsAny(name, "/\\:") || name == "." || name == ".." {
				return nil, fmt.Errorf("invalid CUE filename")
			}
		case "TRACK":
			if name == "" || len(fields) != 3 {
				return nil, fmt.Errorf("invalid CUE TRACK")
			}
			n, e := strconv.Atoi(fields[1])
			if e != nil || n != len(d.Tracks)+1 || n > 99 {
				return nil, fmt.Errorf("invalid CUE track number")
			}
			kind := fields[2]
			if (n == 1 && kind != "MODE1/2352") || (n > 1 && kind != "AUDIO") {
				return nil, fmt.Errorf("unsupported CUE track kind")
			}
			d.Tracks = append(d.Tracks, cueTrack{Number: n, Kind: kind})
			indexed = append(indexed, false)
		case "INDEX":
			if len(d.Tracks) == 0 || len(fields) != 3 || fields[1] != "01" {
				return nil, fmt.Errorf("unsupported CUE INDEX")
			}
			i := len(d.Tracks) - 1
			if indexed[i] {
				return nil, fmt.Errorf("duplicate CUE INDEX")
			}
			v, e := cueFrames(fields[2])
			if e != nil {
				return nil, e
			}
			d.Tracks[i].FileSector = v
			indexed[i] = true
		case "PREGAP":
			if len(d.Tracks) == 0 || len(fields) != 2 {
				return nil, fmt.Errorf("invalid CUE PREGAP")
			}
			v, e := cueFrames(fields[1])
			if e != nil {
				return nil, e
			}
			d.Tracks[len(d.Tracks)-1].Pregap = v
		default:
			return nil, fmt.Errorf("unsupported CUE directive %q", fields[0])
		}
	}
	if len(d.Tracks) == 0 || d.Tracks[0].FileSector != 0 {
		return nil, fmt.Errorf("invalid first data track")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	resolved := ""
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			if resolved != "" || !entry.Type().IsRegular() {
				return nil, fmt.Errorf("ambiguous or nonregular CUE input")
			}
			resolved = filepath.Join(filepath.Dir(path), entry.Name())
		}
	}
	if resolved == "" {
		return nil, fmt.Errorf("CUE BIN not found")
	}
	f, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	d.File = f
	fail := func(err error) (*cueDisc, error) { f.Close(); return nil, err }
	info, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if info.Size()%2352 != 0 || info.Size()/2352 > 0xffffffff {
		return fail(fmt.Errorf("invalid raw CD size"))
	}
	d.RawSectors = uint32(info.Size() / 2352)
	for i, t := range d.Tracks {
		if !indexed[i] || t.FileSector >= d.RawSectors || (i > 0 && t.FileSector <= d.Tracks[i-1].FileSector) {
			return fail(fmt.Errorf("CUE track bounds"))
		}
	}
	var gaps uint32
	for i := range d.Tracks {
		gaps += d.Tracks[i].Pregap
		d.Tracks[i].DiscSector = d.Tracks[i].FileSector + gaps
	}
	d.LeadOut = d.RawSectors + gaps
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fail(err)
	}
	d.SHA256 = fmt.Sprintf("%x", h.Sum(nil))
	pvd, err := d.readDataSector(16)
	if err != nil {
		return fail(err)
	}
	if string(pvd[:7]) != "\x01CD001\x01" || binary.LittleEndian.Uint16(pvd[128:]) != 2048 || binary.BigEndian.Uint16(pvd[130:]) != 2048 {
		return fail(fmt.Errorf("unsupported ISO descriptor"))
	}
	d.VolumeSectors = binary.LittleEndian.Uint32(pvd[80:])
	if d.VolumeSectors != binary.BigEndian.Uint32(pvd[84:]) || d.VolumeSectors > d.RawSectors || (len(d.Tracks) > 1 && d.VolumeSectors > d.Tracks[1].FileSector) {
		return fail(fmt.Errorf("ISO volume bounds"))
	}
	d.RootSector = binary.LittleEndian.Uint32(pvd[158:])
	d.RootBytes = binary.LittleEndian.Uint32(pvd[166:])
	if uint64(d.RootSector)*2048+uint64(d.RootBytes) > uint64(d.VolumeSectors)*2048 {
		return fail(fmt.Errorf("ISO root bounds"))
	}
	return d, nil
}

func (d *cueDisc) readDataSector(lba uint32) ([]byte, error) {
	if lba >= d.RawSectors || (len(d.Tracks) > 1 && lba >= d.Tracks[1].FileSector) {
		return nil, fmt.Errorf("not a data sector")
	}
	raw := make([]byte, 2352)
	if _, err := d.File.ReadAt(raw, int64(lba)*2352); err != nil {
		return nil, err
	}
	if raw[0] != 0 || raw[11] != 0 || raw[15] != 1 {
		return nil, fmt.Errorf("invalid MODE1 sector")
	}
	for _, v := range raw[1:11] {
		if v != 255 {
			return nil, fmt.Errorf("invalid CD sync")
		}
	}
	return raw[16:2064], nil
}

func (d *cueDisc) handle(c *cpu386.CPU) bool {
	switch uint16(c.R[cpu386.EAX]) {
	case 0x1500:
		c.R[cpu386.EBX] = c.R[cpu386.EBX]&0xffff0000 | 1
		c.R[cpu386.ECX] = c.R[cpu386.ECX]&0xffff0000 | 3
		return true
	case 0x150c:
		c.R[cpu386.EBX] = c.R[cpu386.EBX]&0xffff0000 | 0x020a
		return true
	case 0x150d:
		return c.WriteSegmentBytes(c.Seg[cpu386.SegES], c.R[cpu386.EBX], []byte{3})
	case 0x1510:
		return d.request(c)
	}
	return false
}

func redBook(lba uint32) []byte {
	frames := lba + 150
	return []byte{byte(frames % 75), byte(frames / 75 % 60), byte(frames / 4500), 0}
}

func (d *cueDisc) request(c *cpu386.CPU) bool {
	if uint16(c.R[cpu386.ECX]) != 3 {
		return false
	}
	base := c.R[cpu386.EBX]
	seg := c.Seg[cpu386.SegES]
	packet := make([]byte, 13)
	for i := range packet {
		v, ok := c.ReadSegment8(seg, base+uint32(i))
		if !ok {
			return false
		}
		packet[i] = v
	}
	if packet[1] != 0 || !(packet[0] == 26 && packet[2] == 3 || packet[0] == 22 && packet[2] == 0x84 || packet[0] == 13 && packet[2] == 0x85) {
		return false
	}
	desc, ok := c.Descriptors[seg]
	if !ok || !desc.Writable || uint64(base)+uint64(packet[0]) > uint64(desc.Limit)+1 {
		return false
	}
	length := int(packet[0])
	packet = append(packet, make([]byte, length-len(packet))...)
	for i := 13; i < len(packet); i++ {
		a := uint64(base) + uint64(i)
		if a > 0xffffffff {
			return false
		}
		v, ok := c.ReadSegment8(seg, uint32(a))
		if !ok {
			return false
		}
		packet[i] = v
	}
	if packet[2] == 0x85 {
		// Positive-length PLAY remains unhandled, so this subset is always idle.
		// When playback is added, STOP must operate on its actual state.
		if !c.WriteSegmentBytes(seg, base+3, []byte{0, 1}) {
			return false
		}
		d.AudioIdleStopCount++
		return true
	}
	if packet[2] == 0x84 {
		count := binary.LittleEndian.Uint32(packet[18:])
		if count != 0 || packet[13] > 1 {
			return false
		}
		if !c.WriteSegmentBytes(seg, base+3, []byte{0, 0x80}) {
			return false
		}
		d.AudioRejectionCount++
		if len(d.AudioRejections) == 64 {
			copy(d.AudioRejections, d.AudioRejections[1:])
			d.AudioRejections = d.AudioRejections[:63]
		}
		d.AudioRejections = append(d.AudioRejections, map[string]any{"command": 0x84, "addressing_mode": packet[13], "start_raw": binary.LittleEndian.Uint32(packet[14:]), "sectors": count, "status": 0x8000, "reason": "zero-length PLAY AUDIO; DOSBox Staging v0.82.2 compatibility error; no audio produced"})
		return true
	}
	address := uint32(binary.LittleEndian.Uint16(packet[14:])) + 16*uint32(binary.LittleEndian.Uint16(packet[16:]))
	count := uint32(binary.LittleEndian.Uint16(packet[18:]))
	if count < 1 || count > 32 {
		return false
	}
	// 轉送地址為實模式 far pointer。專用 selector 只用於主機邊界檢查。
	target := uint16(0xffe0)
	if _, exists := c.Descriptors[target]; exists {
		return false
	}
	c.SetDescriptor(target, cpu386.Descriptor{Base: address, Limit: count - 1, Writable: true})
	defer delete(c.Descriptors, target)
	control, ok := c.ReadSegment8(target, 0)
	if !ok {
		return false
	}
	var output []byte
	switch control {
	case 10:
		if count != 7 {
			return false
		}
		output = []byte{10, byte(d.Tracks[0].Number), byte(d.Tracks[len(d.Tracks)-1].Number)}
		output = append(output, redBook(d.LeadOut)...)
	case 11:
		if count != 7 {
			return false
		}
		track, ok := c.ReadSegment8(target, 1)
		if !ok {
			return false
		}
		if track < 1 || int(track) > len(d.Tracks) {
			return c.WriteSegmentBytes(seg, base+3, []byte{8, 0x81})
		}
		t := d.Tracks[track-1]
		output = []byte{11, track}
		output = append(output, redBook(t.DiscSector)...)
		attr := byte(0)
		if t.Kind != "AUDIO" {
			attr = 0x40
		}
		output = append(output, attr)
	default:
		return false
	}
	if !c.WriteSegmentBytes(target, 0, output) {
		return false
	}
	return c.WriteSegmentBytes(seg, base+3, []byte{0, 1})
}

type realModeBus struct{ bus cpu.Bus }

func (b realModeBus) Read8(a uint32) (uint8, error) {
	if a >= 0x100000 {
		return 0, fmt.Errorf("real mode address out of bounds")
	}
	return b.bus.Read8(a), nil
}
func (b realModeBus) Write8(a uint32, v uint8) error {
	if a >= 0x100000 {
		return fmt.Errorf("real mode address out of bounds")
	}
	b.bus.Write8(a, v)
	return nil
}

func (d *cueDisc) realMode(r *cpu.CPU, n uint8) bool {
	if n != 0x2f {
		return false
	}
	c := cpu386.New(realModeBus{r.Bus})
	for i := range c.R {
		c.R[i] = uint32(r.R[i]) | uint32(r.Hi[i])<<16
	}
	c.EFlags = uint32(r.Flags)
	c.Seg[cpu386.SegES] = 8
	c.SetDescriptor(8, cpu386.Descriptor{Base: uint32(r.Seg[cpu.ES]) * 16, Limit: 0xffff, Writable: true})
	if !d.handle(c) {
		return false
	}
	for i := range c.R {
		r.R[i] = uint16(c.R[i])
		r.Hi[i] = uint16(c.R[i] >> 16)
	}
	r.SetFlags(uint16(c.EFlags))
	return true
}
