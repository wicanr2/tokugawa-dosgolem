package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
)

const publisherIndexedSHA = "fb43b08b3202d43f0e502528683ea9ecc91a8b43778b680bd5c5c5a69228c5d4"
const publisherTargetSHA = "778051183911d4dd5aebe339a3e2187ecde8c63e425a1314f83504f3b4a63f8a"

func (t *markProbeTrace) captureTarget(ip, pointer uint32, mem []byte) {
	if ip != 0x11a50 || t.TargetReady || !t.DrawReady || uint64(pointer)+768 > uint64(len(mem)) {
		return
	}
	data := mem[pointer : pointer+768]
	if fmt.Sprintf("%x", sha256.Sum256(data)) != publisherTargetSHA {
		return
	}
	copy(t.Target[:], data)
	t.TargetReady, t.TargetPointer = true, pointer
}

func (t *markProbeTrace) qualified(s nativeSnapshot, mem []byte) bool {
	if !t.DrawReady || !t.TargetReady || len(t.Source) != 640*480 || s.MaxColor != 255 || len(s.Indexed) != 640*480 ||
		uint64(t.SourcePointer)+640*480 > uint64(len(mem)) || uint64(t.TargetPointer)+768 > uint64(len(mem)) {
		return false
	}
	return fmt.Sprintf("%x", sha256.Sum256(t.Source)) == publisherIndexedSHA && bytes.Equal(s.Indexed, t.Source) &&
		bytes.Equal(mem[t.SourcePointer:t.SourcePointer+640*480], t.Source) && bytes.Equal(mem[t.TargetPointer:t.TargetPointer+768], t.Target[:])
}

// This observer is installed only by the separate fixed-hash MARK research tool.
type markProbeTrace struct {
	Events                 []map[string]any `json:"events"`
	Return, Stack          uint32           `json:"pending_return,omitempty"`
	Source                 []byte           `json:"-"`
	Seen                   map[uint32]bool  `json:"-"`
	SourcePointer          uint32           `json:"source_pointer"`
	Target                 [768]byte        `json:"-"`
	TargetPointer          uint32           `json:"target_palette_pointer"`
	DrawReady, TargetReady bool             `json:"-"`
}

func (t *markProbeTrace) observe(step int, ip, sp uint32, mem []byte, video *bankVideo, cursor *mouseCursor) error {
	if t.Seen == nil {
		t.Seen = map[uint32]bool{}
	}
	if ip == 0x183ba {
		if t.Seen[ip] || uint64(sp)+24 > uint64(len(mem)) {
			return fmt.Errorf("unexpected MARK draw entry")
		}
		t.Seen[ip] = true
		var words [6]uint32
		for i := range words {
			words[i] = binary.LittleEndian.Uint32(mem[sp+uint32(i*4):])
		}
		if words[0] != 0x10265 || words[1] != 0 || words[2] != 0 || words[3] != 640 || words[4] != 480 || uint64(words[5])+640*480 > uint64(len(mem)) {
			return fmt.Errorf("unverified MARK draw parameters: %v", words)
		}
		t.Return, t.Stack = words[0], sp
		t.Source = append([]byte(nil), mem[words[5]:words[5]+640*480]...)
		t.SourcePointer = words[5]
		t.Events = append(t.Events, map[string]any{"kind": "draw_entry", "step": step, "eip": ip, "stack_words": words, "source_indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(t.Source))})
		if err := os.WriteFile("/output/mark-source.idx", t.Source, 0600); err != nil {
			return err
		}
	}
	if ip == t.Return && t.Return != 0 && sp == t.Stack+4 {
		if video.Mode != 0x101 || video.Width != 640 || video.Height != 480 {
			return fmt.Errorf("MARK draw returned with wrong display mode")
		}
		indexed := video.Pixels[:640*480]
		if string(indexed) != string(t.Source) {
			return fmt.Errorf("MARK draw destination differs from decoded source")
		}
		if err := os.WriteFile("/output/mark-native.idx", indexed, 0600); err != nil {
			return err
		}
		t.Events = append(t.Events, map[string]any{"kind": "draw_return", "step": step, "eip": ip, "source_matches_destination": true, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(indexed))})
		t.Return, t.Stack = 0, 0
		t.DrawReady = true
	}
	for _, point := range []struct {
		IP   uint32
		Name string
	}{{0x10279, "fade_in_return"}, {0x10297, "wait_return"}, {0x102a8, "fade_out_return"}, {0x102bc, "cleanup"}} {
		if ip != point.IP || t.Seen[ip] {
			continue
		}
		t.Seen[ip] = true
		frame, err := captureFrame(video, cursor, 0, 0)
		if err != nil {
			return err
		}
		t.Events = append(t.Events, map[string]any{"kind": point.Name, "step": step, "eip": ip, "frame": frame, "palette": video.Legacy.Palette()})
	}
	return nil
}
