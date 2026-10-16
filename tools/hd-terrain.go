package main

import (
	"encoding/binary"
	"image"
)

type hdTerrainDraw struct {
	Index int             `json:"index"`
	Rect  image.Rectangle `json:"logical_rect"`
	Order uint64          `json:"draw_order"`
}

type hdTerrainSpriteDraw struct {
	Group                string          `json:"group"`
	Index                int             `json:"index"`
	Rect                 image.Rectangle `json:"logical_rect"`
	Order                uint64          `json:"draw_order"`
	CombatReturn         uint32          `json:"combat_return_linear,omitempty"`
	Background           *image.NRGBA    `json:"-"`
	BackgroundIndexed    []byte          `json:"-"`
	BackgroundIndexedSHA string          `json:"background_indexed_sha256,omitempty"`
	BackgroundVerified   bool            `json:"background_verified,omitempty"`
	BackgroundSHA        string          `json:"background_rgba_sha256,omitempty"`
	SourceSHA            string          `json:"source_rgba_sha256,omitempty"`
	AfterSHA             string          `json:"after_rgba_sha256,omitempty"`
	SourceIndexedSHA     string          `json:"source_indexed_sha256,omitempty"`
	AfterIndexedSHA      string          `json:"after_indexed_sha256,omitempty"`
	BeginStep, EndStep   int
}

type hdTerrainPending struct {
	Return, Stack uint32
	Draw          hdTerrainDraw
	Sprite        *hdTerrainSpriteDraw
	Backdrop      *spriteBackdropRecord `json:"-"`
}

type hdTerrainState struct {
	Draws         []hdTerrainDraw       `json:"draws"`
	History       []hdTerrainDraw       `json:"draw_history"`
	Pending       *hdTerrainPending     `json:"pending_return"`
	Returns       uint64                `json:"completed_returns"`
	Sprites       []hdTerrainSpriteDraw `json:"sprites"`
	SpriteReturns uint64                `json:"completed_sprite_returns"`
	Sequence      uint64                `json:"draw_sequence"`
}

func (h *hdTerrainState) invalidate() { h.Draws, h.History, h.Sprites, h.Pending = nil, nil, nil, nil }

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
	return hdTerrainDraw{Index: int(sy/32*13 + sx/6), Rect: image.Rect(int(x), int(y), int(x+48), int(y+32))}, true
}

func hdTerrainSpriteDecode(source, dest, ret uint32) (hdTerrainSpriteDraw, bool) {
	sy, sx := source/80, source%80
	y, x := dest/80, dest%80*8
	d := hdTerrainSpriteDraw{Rect: image.Rect(int(x), int(y), int(x+48), int(y+48))}
	if sx >= 78 || sx%6 != 0 || x+48 > 640 || y+48 > 400 {
		return d, false
	}
	switch {
	case (sy == 256 || sy == 304) && ret == 0x1fe69:
		d.Group, d.Index = "objects", int((sy-256)/48*13+sx/6)
	case sy == 352 && ret == 0x1fea2:
		d.Group, d.Index = "units", int(sx/6)
	case sy == 352 && ret == 0x1f9be && (dest == 60*80+30 || dest == 60*80+50):
		d.Group, d.Index, d.CombatReturn = "units", int(sx/6), ret
	default:
		return d, false
	}
	return d, true
}

func hdTerrainSpriteSourceValid(mem []byte) bool {
	return hdTerrainSourceValid(mem) && binary.LittleEndian.Uint32(mem[0x3598c:]) == 0x5000 && binary.LittleEndian.Uint32(mem[0x35990:]) == 0x6e00
}

// 暫存器參數只讀。新繪製先撤銷重疊項，自然返回後才啟用。
func (h *hdTerrainState) observe(step int, eip, esp, esi, edi uint32, mem []byte, snapshots ...func() (nativeSnapshot, error)) {
	if eip == 0x1c084 || eip == 0x11f14 || eip == 0x1e93f {
		h.invalidate()
		return
	}
	if p := h.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		if p.Sprite != nil {
			if hdTerrainSpriteSourceValid(mem) && len(h.Sprites) < 512 {
				h.Sequence++
				d := *p.Sprite
				d.Order = h.Sequence
				if p.Backdrop != nil && len(snapshots) == 1 {
					p.Backdrop.finish(step, mem, snapshots[0])
					if p.Backdrop.Verified {
						b := p.Backdrop
						d.Background, d.BackgroundVerified = b.Before, true
						d.BackgroundIndexed, d.BackgroundIndexedSHA = b.BeforeIndexed, b.BeforeIndexedSHA
						d.BackgroundSHA, d.SourceSHA, d.AfterSHA = b.BeforeSHA, b.SourceSHA, b.AfterSHA
						d.BeginStep, d.EndStep = b.BeginStep, b.EndStep
						if d.CombatReturn != 0 {
							d.SourceIndexedSHA, d.AfterIndexedSHA = b.SourceIndexedSHA, b.AfterIndexedSHA
						}
					}
				}
				h.Sprites = append(h.Sprites, d)
				h.SpriteReturns++
			} else {
				h.invalidate()
			}
		} else if hdTerrainSourceValid(mem) && len(h.Draws) < 256 && len(h.History) < 512 {
			h.Sequence++
			p.Draw.Order = h.Sequence
			h.Draws = append(h.Draws, p.Draw)
			h.History = append(h.History, p.Draw)
			h.Returns++
		} else {
			h.invalidate()
		}
		h.Pending = nil
	}
	if eip == 0x1f9d0 {
		h.Pending = nil
		if !hdTerrainSpriteSourceValid(mem) || uint64(esp)+4 > uint64(len(mem)) {
			h.invalidate()
			return
		}
		ret := binary.LittleEndian.Uint32(mem[esp:])
		d, ok := hdTerrainSpriteDecode(esi, edi, ret)
		if !ok {
			h.invalidate()
			return
		}
		h.Pending = &hdTerrainPending{Return: ret, Stack: esp, Sprite: &d}
		if len(snapshots) == 1 {
			h.Pending.Backdrop = beginSpriteBackdrop(step, esp, esi, edi, mem, snapshots[0])
		}
		return
	}
	if eip != 0x1fc07 {
		return
	}
	h.Pending = nil
	draw, ok := hdTerrainDecode(esi, edi)
	if !ok || !hdTerrainSourceValid(mem) || uint64(esp)+4 > uint64(len(mem)) {
		h.invalidate()
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
	h.Pending = &hdTerrainPending{Return: ret, Stack: esp, Draw: draw}
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
	result, _ := p.applyForeground(out, native, hdTerrainDraw{Rect: rect}, nil, nil, scale, enabled)
	return result
}

type hdTerrainSpriteKey struct {
	Group string
	Index int
}

func newHDTerrainSpriteSource(source image.Image) *image.NRGBA {
	if source == nil || source.Bounds() != image.Rect(0, 0, 48, 48) {
		return nil
	}
	out := image.NewNRGBA(source.Bounds())
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			c := rgbaAt(source, x, y)
			if c.A != 0 && c.A != 255 {
				return nil
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}

// 依原始索引零遮罩合成預期畫面。全部吻合後才換地面，前景保持原生像素。
func (p hdTerrainPair) applyForeground(out *image.NRGBA, native image.Image, draw hdTerrainDraw, sprites []hdTerrainSpriteDraw, sources map[hdTerrainSpriteKey]*image.NRGBA, scale int, enabled bool) (string, int) {
	rect := draw.Rect
	if !enabled {
		return "original", 0
	}
	if p.Reason != "ready" {
		return p.Reason, 0
	}
	if native == nil || out == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*scale, 400*scale) || rect.Dx() != 48 || rect.Dy() != 32 || !rect.In(native.Bounds()) || p.Asset == nil || p.Asset.Bounds() != image.Rect(0, 0, 48*scale, 32*scale) {
		return "fallback_geometry", 0
	}
	expected := image.NewNRGBA(image.Rect(0, 0, 48, 32))
	copy(expected.Pix, p.Source.Pix)
	var foreground [48 * 32]bool
	for _, sprite := range sprites {
		if sprite.Order <= draw.Order || !sprite.Rect.Overlaps(rect) {
			continue
		}
		source := sources[hdTerrainSpriteKey{sprite.Group, sprite.Index}]
		if source == nil || source.Bounds() != image.Rect(0, 0, 48, 48) || sprite.Rect.Dx() != 48 || sprite.Rect.Dy() != 48 || !sprite.Rect.In(native.Bounds()) {
			return "fallback_foreground_source", 0
		}
		intersection := rect.Intersect(sprite.Rect)
		for y := intersection.Min.Y; y < intersection.Max.Y; y++ {
			for x := intersection.Min.X; x < intersection.Max.X; x++ {
				c := source.NRGBAAt(x-sprite.Rect.Min.X, y-sprite.Rect.Min.Y)
				if c.A == 255 {
					expected.SetNRGBA(x-rect.Min.X, y-rect.Min.Y, c)
					foreground[(y-rect.Min.Y)*48+x-rect.Min.X] = true
				}
			}
		}
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			if rgbaAt(native, rect.Min.X+x, rect.Min.Y+y) != expected.NRGBAAt(x, y) {
				return "fallback_source_mismatch", 0
			}
		}
	}
	for y := 0; y < 32*scale; y++ {
		for x := 0; x < 48*scale; x++ {
			c := p.Asset.NRGBAAt(x, y)
			if foreground[y/scale*48+x/scale] {
				c = rgbaAt(native, rect.Min.X+x/scale, rect.Min.Y+y/scale)
			}
			out.SetNRGBA(rect.Min.X*scale+x, rect.Min.Y*scale+y, c)
		}
	}
	count := 0
	for _, opaque := range foreground {
		if opaque {
			count++
		}
	}
	return "hd", count
}
