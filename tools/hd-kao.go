package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

type hdKaoDraw struct {
	Index   int             `json:"index"`
	Rect    image.Rectangle `json:"logical_rect"`
	EndStep int             `json:"completed_step"`
}

type hdKaoPending struct {
	Return, Stack uint32
	Draw          hdKaoDraw
}

type hdKaoState struct {
	Pending *hdKaoPending    `json:"pending_draw"`
	Draws   []hdKaoDraw      `json:"draws"`
	Events  []map[string]any `json:"events"`
}

func (h *hdKaoState) observe(step int, eip, esp uint32, mem []byte) {
	if p := h.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		if len(h.Draws) < 64 {
			p.Draw.EndStep = step
			h.Draws = append(h.Draws, p.Draw)
		}
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "kao_draw_return", "step": step, "linear_eip": eip, "index": p.Draw.Index, "logical_rect": p.Draw.Rect})
		}
		h.Pending = nil
	}
	if eip == 0x1c084 {
		h.Draws, h.Pending = nil, nil
	}
	if eip != 0x1c6e8 {
		return
	}
	h.Pending = nil
	if uint64(esp)+16 > uint64(len(mem)) || len(mem) < 0x3988c {
		h.Draws = nil
		return
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	index := binary.LittleEndian.Uint32(mem[uint64(esp)+4:])
	xbyte := binary.LittleEndian.Uint32(mem[uint64(esp)+8:])
	y := binary.LittleEndian.Uint32(mem[uint64(esp)+12:])
	if xbyte > 70 || y > 320 {
		h.Draws = nil
		return
	}
	rect := image.Rect(int(xbyte)*8, int(y), int(xbyte)*8+80, int(y)+80)
	kept := h.Draws[:0]
	for _, draw := range h.Draws {
		if !draw.Rect.Overlaps(rect) {
			kept = append(kept, draw)
		}
	}
	h.Draws = kept
	if uint64(index) >= uint64(len(hdKaoProfiles)) || uint64(ret) >= uint64(len(mem)) {
		return
	}
	base := uint64(binary.LittleEndian.Uint32(mem[0x39888:])) + uint64(index)*3200
	if base < 0xc0000 || base+3200 > uint64(len(mem)) {
		return
	}
	if fmt.Sprintf("%x", sha256.Sum256(mem[base:base+3200])) != hdKaoProfiles[index].RawSHA {
		return
	}
	h.Pending = &hdKaoPending{ret, esp, hdKaoDraw{Index: int(index), Rect: rect}}
	if len(h.Events) < 128 {
		h.Events = append(h.Events, map[string]any{"kind": "kao_draw_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp, "raw_args": []uint32{index, xbyte, y}, "source_linear": base, "source_sha256": hdKaoProfiles[index].RawSHA, "logical_rect": rect})
	}
}

// 顯示前逐點核對原版頭像。任何遮擋或未知來源都保留原版。
func previewKao(out *image.NRGBA, native, source, asset image.Image, rect image.Rectangle, scale int, enabled bool) string {
	if !enabled {
		return "original"
	}
	if asset == nil {
		return "fallback_missing_asset"
	}
	if out == nil || native == nil || scale < 1 || scale > 4 || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*scale, 400*scale) || rect.Dx() != 80 || rect.Dy() != 80 || !rect.In(native.Bounds()) {
		return "fallback_geometry"
	}
	if source == nil || source.Bounds() != image.Rect(0, 0, 80, 80) {
		return "fallback_source_shape"
	}
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			if rgbaAt(native, rect.Min.X+x, rect.Min.Y+y) != rgbaAt(source, x, y) {
				return "fallback_source_mismatch"
			}
		}
	}
	b := asset.Bounds()
	if b.Min != image.Pt(0, 0) || b.Dx() < 80 || b.Dy() < 80 || b.Dx() > 8192 || b.Dy() > 8192 {
		return "fallback_hd_shape"
	}
	difference := b.Dx() - b.Dy()
	if difference < 0 {
		difference = -difference
	}
	if difference*100 > b.Dy() {
		return "fallback_hd_aspect"
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if rgbaAt(asset, x, y).A != 255 {
				return "fallback_hd_transparency"
			}
		}
	}
	for y := 0; y < 80*scale; y++ {
		for x := 0; x < 80*scale; x++ {
			out.SetNRGBA(rect.Min.X*scale+x, rect.Min.Y*scale+y, rgbaAt(asset, x*b.Dx()/(80*scale), y*b.Dy()/(80*scale)))
		}
	}
	return "hd"
}
