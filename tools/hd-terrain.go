package main

import (
	"encoding/binary"
	"image"
)

type hdTerrainDraw struct {
	Index int             `json:"index"`
	Rect  image.Rectangle `json:"logical_rect"`
}

type hdTerrainPending struct {
	Return, Stack uint32
	Draw          hdTerrainDraw
}

type hdTerrainState struct {
	Draws   []hdTerrainDraw   `json:"draws"`
	Pending *hdTerrainPending `json:"pending_return"`
	Returns uint64            `json:"completed_returns"`
}

func hdTerrainSourceValid(mem []byte) bool {
	if len(mem) < 0x42180 || binary.LittleEndian.Uint32(mem[0x35988:]) != 0 {
		return false
	}
	if binary.LittleEndian.Uint32(mem[0x35984:]) != 0 && binary.LittleEndian.Uint32(mem[0x4217c:]) != 0 {
		return false
	}
	planes := terrainPlaneHashes(mem)
	if len(planes) != 4 {
		return false
	}
	for i, plane := range planes {
		if plane["sha256"] != hdTerrainPlaneSHA[i] {
			return false
		}
	}
	return true
}

func hdTerrainDecode(source, dest uint32) (hdTerrainDraw, bool) {
	sy, sx := source/80, source%80
	y, x := dest/80, dest%80*8
	if sy >= 256 || sy%32 != 0 || sx >= 78 || sx%6 != 0 || x+48 > 640 || y+32 > 400 {
		return hdTerrainDraw{}, false
	}
	return hdTerrainDraw{int(sy/32*13 + sx/6), image.Rect(int(x), int(y), int(x+48), int(y+32))}, true
}

// 暫存器參數只讀。新繪製先撤銷重疊項，自然返回後才啟用。
func (h *hdTerrainState) observe(step int, eip, esp, esi, edi uint32, mem []byte) {
	if eip == 0x1c084 || eip == 0x1e93f {
		h.Draws, h.Pending = nil, nil
		return
	}
	if p := h.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		if hdTerrainSourceValid(mem) && len(h.Draws) < 256 {
			h.Draws = append(h.Draws, p.Draw)
			h.Returns++
		} else {
			h.Draws = nil
		}
		h.Pending = nil
	}
	if eip != 0x1fc07 {
		return
	}
	h.Pending = nil
	draw, ok := hdTerrainDecode(esi, edi)
	if !ok || !hdTerrainSourceValid(mem) || uint64(esp)+4 > uint64(len(mem)) {
		h.Draws = nil
		return
	}
	kept := h.Draws[:0]
	for _, old := range h.Draws {
		if !old.Rect.Overlaps(draw.Rect) {
			kept = append(kept, old)
		}
	}
	h.Draws = kept
	ret := binary.LittleEndian.Uint32(mem[esp:])
	if uint64(ret) >= uint64(len(mem)) {
		return
	}
	h.Pending = &hdTerrainPending{ret, esp, draw}
}

type hdTerrainPair struct {
	Source, Asset *image.NRGBA
	Reason        string
}

func newHDTerrainPair(source, asset image.Image, scale int) hdTerrainPair {
	p := hdTerrainPair{Reason: "fallback_source_shape"}
	if source == nil || source.Bounds() != image.Rect(0, 0, 48, 32) || scale < 1 || scale > 4 {
		return p
	}
	p.Source = image.NewNRGBA(source.Bounds())
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			c := rgbaAt(source, x, y)
			if c.A != 255 {
				p.Reason = "fallback_source_transparency"
				return p
			}
			p.Source.SetNRGBA(x, y, c)
		}
	}
	p.Reason = "fallback_missing_asset"
	if asset == nil {
		return p
	}
	prepared, reason := prepareHDImage(asset, 48, 32, scale)
	if reason != "" {
		p.Reason = reason
		return p
	}
	p.Asset, p.Reason = prepared.(*image.NRGBA), "ready"
	return p
}

func (p hdTerrainPair) apply(out *image.NRGBA, native image.Image, rect image.Rectangle, scale int, enabled bool) string {
	if !enabled {
		return "original"
	}
	if p.Reason != "ready" {
		return p.Reason
	}
	if native == nil || out == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*scale, 400*scale) || rect.Dx() != 48 || rect.Dy() != 32 || !rect.In(native.Bounds()) || p.Asset == nil || p.Asset.Bounds() != image.Rect(0, 0, 48*scale, 32*scale) {
		return "fallback_geometry"
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			if rgbaAt(native, rect.Min.X+x, rect.Min.Y+y) != p.Source.NRGBAAt(x, y) {
				return "fallback_source_mismatch"
			}
		}
	}
	for y := 0; y < 32*scale; y++ {
		for x := 0; x < 48*scale; x++ {
			out.SetNRGBA(rect.Min.X*scale+x, rect.Min.Y*scale+y, p.Asset.NRGBAAt(x, y))
		}
	}
	return "hd"
}
