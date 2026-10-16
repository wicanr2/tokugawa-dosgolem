package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"path/filepath"
)

const battleFrameIndexedSHA = "99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c"

var battleFramePlaneSHA = [4]string{
	"a0d55e424ddaa60e187a8cc6765b3463b1ca2a24e3df0a97f41cb08e2838345e",
	"5ea8f4b86004dd213e2bf25280385c858c01327a32a7b6438f758b56ac25d8e1",
	"ce640d6999ec1a78f7ece33027cf99deb0251364c66f7c134bbfeb0f8be0f1f1",
	"95a2ca150fff61219c0b2d2fecf0155b0b342808e7e371f6ae567084b11f92d7",
}

type battleFrameReturn struct {
	Return, Stack uint32
	BeginStep     int
}

type battleFrameState struct {
	SourceReady bool               `json:"source_validated"`
	Active      bool               `json:"draw_completed"`
	Pending     *battleFrameReturn `json:"pending_return"`
	Events      []map[string]any   `json:"events"`
	Dropped     uint64             `json:"dropped_events"`
	Capture     nativeSnapshot     `json:"-"`
	CaptureStep int                `json:"capture_step"`
}

func (b *battleFrameState) loaded(e map[string]any) {
	b.SourceReady = false
	if e["name"] != "MAPGAMEN.OZM" || e["raw_columns"] != uint32(80) || e["raw_rows"] != uint32(400) {
		return
	}
	planes, ok := e["planes"].([]map[string]any)
	if !ok || len(planes) != 4 {
		return
	}
	for i, p := range planes {
		if p["sha256"] != battleFramePlaneSHA[i] {
			return
		}
	}
	b.SourceReady = true
}

func (b *battleFrameState) event(e map[string]any) {
	if len(b.Events) >= 64 {
		b.Dropped++
		return
	}
	b.Events = append(b.Events, e)
}

func (b *battleFrameState) observe(step int, eip, esp uint32, mem []byte, snapshot func() (nativeSnapshot, error)) {
	if p := b.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		b.Pending = nil
		e := map[string]any{"kind": "battle_frame_return", "begin_step": p.BeginStep, "step": step, "linear_eip": eip, "stack_linear": esp, "eligible": false}
		if snapshot != nil {
			s, err := snapshot()
			if err != nil {
				e["observation_error"] = err.Error()
			} else {
				digest := fmt.Sprintf("%x", sha256.Sum256(s.Indexed))
				e["native_indexed_sha256"] = digest
				b.Active = s.Image != nil && s.Image.Bounds() == image.Rect(0, 0, 640, 400) && s.MaxColor == 15 && len(s.Indexed) == 640*400 && digest == battleFrameIndexedSHA
				e["eligible"] = b.Active
				if b.Active {
					b.Capture = s
					b.CaptureStep = step
					e["palette"], e["dac"] = s.Palette, s.DAC
				}
			}
		}
		b.event(e)
	}
	if eip == 0x1bf44 {
		b.SourceReady = false
	}
	if eip == 0x11f14 {
		b.Active, b.Pending = false, nil
		return
	}
	if eip != 0x1c084 {
		return
	}
	wasActive := b.Active
	b.Active = false
	b.Pending = nil
	if !b.SourceReady || snapshot == nil || uint64(esp)+4 > uint64(len(mem)) {
		if wasActive {
			b.event(map[string]any{"kind": "battle_frame_invalidated", "step": step, "linear_eip": eip})
		}
		return
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	if uint64(ret) >= uint64(len(mem)) {
		return
	}
	b.Pending = &battleFrameReturn{Return: ret, Stack: esp, BeginStep: step}
	b.event(map[string]any{"kind": "battle_frame_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp})
}

func (b *battleFrameState) export(root string) (map[string]any, error) {
	if b.Capture.Image == nil {
		return nil, nil
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, b.Capture.Image); err != nil {
		return nil, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes()))
	name := "battle-frame-return-" + digest + ".png"
	for _, file := range []struct {
		Name string
		Data []byte
	}{{name, buffer.Bytes()}, {"battle-frame-" + battleFrameIndexedSHA + ".idx", b.Capture.Indexed}} {
		path := filepath.Join(root, file.Name)
		if old, err := readRuntimeFile(path); err == nil && !bytes.Equal(old, file.Data) {
			return nil, fmt.Errorf("refuse changed frame export %s", file.Name)
		}
		if err := writeRuntimeFile(path, file.Data, 0640); err != nil {
			return nil, err
		}
	}
	return map[string]any{"path": name, "png_sha256": digest, "indices_path": "battle-frame-" + battleFrameIndexedSHA + ".idx", "indices_sha256": battleFrameIndexedSHA, "capture_step": b.CaptureStep}, nil
}
