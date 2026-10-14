package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

const hdWindowRawSHA = "26d936f2a599e4c6a552de7a07a890e8ccbec3d51750e22383c7f2fde8ce9e50"

type hdWindowDraw struct {
	Rect    image.Rectangle `json:"logical_rect"`
	EndStep int             `json:"completed_step"`
}

type hdWindowPending struct {
	Return, Stack uint32
	Level         uint32
	Draw          hdWindowDraw
	Snapshot      []hdWindowDraw
}

type hdWindowState struct {
	Draws          []hdWindowDraw            `json:"draws"`
	PendingDraw    *hdWindowPending          `json:"pending_draw"`
	PendingSave    *hdWindowPending          `json:"pending_save"`
	PendingRestore *hdWindowPending          `json:"pending_restore"`
	Saved          map[uint32][]hdWindowDraw `json:"saved"`
	Events         []map[string]any          `json:"events"`
}

func hdWindowSourceValid(mem []byte) bool {
	if len(mem) < 0x3515b {
		return false
	}
	base := uint64(binary.LittleEndian.Uint32(mem[0x35153:]))
	return base >= 0xc0000 && base+1152 <= uint64(len(mem)) && fmt.Sprintf("%x", sha256.Sum256(mem[base:base+1152])) == hdWindowRawSHA
}

func (h *hdWindowState) event(kind string, step int, eip uint32, p *hdWindowPending) {
	if len(h.Events) < 256 {
		h.Events = append(h.Events, map[string]any{"kind": kind, "step": step, "linear_eip": eip, "return_linear": p.Return, "stack_linear": p.Stack, "level": p.Level, "logical_rect": p.Draw.Rect})
	}
}

// 只讀原始堆疊與 backing RAM。資格必須等原版自然返回後才生效。
func (h *hdWindowState) observe(step int, eip, esp uint32, mem []byte) {
	returned := func(p *hdWindowPending) bool {
		return p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4
	}
	if p := h.PendingDraw; returned(p) {
		if hdWindowSourceValid(mem) && len(h.Draws) < 64 {
			p.Draw.EndStep = step
			h.Draws = append(h.Draws, p.Draw)
			h.event("window_draw_return", step, eip, p)
		} else {
			h.Draws = nil
		}
		h.PendingDraw = nil
	}
	if p := h.PendingSave; returned(p) {
		if hdWindowSourceValid(mem) && binary.LittleEndian.Uint32(mem[0x35157:]) == p.Level+1 {
			if h.Saved == nil {
				h.Saved = map[uint32][]hdWindowDraw{}
			}
			h.Saved[p.Level] = p.Snapshot
			h.event("window_save_return", step, eip, p)
		} else {
			h.Saved = nil
		}
		h.PendingSave = nil
	}
	if p := h.PendingRestore; returned(p) {
		if hdWindowSourceValid(mem) && binary.LittleEndian.Uint32(mem[0x35157:]) == p.Level {
			h.Draws = p.Snapshot
			h.event("window_restore_return", step, eip, p)
		}
		h.PendingRestore = nil
	}
	switch eip {
	case 0x1afb0:
		h.Draws, h.Saved = nil, nil
		h.PendingDraw, h.PendingSave, h.PendingRestore = nil, nil, nil
	case 0x1c084:
		h.Draws, h.PendingDraw = nil, nil
	case 0x1b00c, 0x1b165, 0x1b245:
		if !hdWindowSourceValid(mem) || uint64(esp)+4 > uint64(len(mem)) {
			h.Draws, h.Saved = nil, nil
			h.PendingDraw, h.PendingSave, h.PendingRestore = nil, nil, nil
			return
		}
		ret := binary.LittleEndian.Uint32(mem[esp:])
		if uint64(ret) >= uint64(len(mem)) {
			h.Draws = nil
			h.PendingDraw, h.PendingSave, h.PendingRestore = nil, nil, nil
			return
		}
		p := &hdWindowPending{Return: ret, Stack: esp}
		if eip == 0x1b165 {
			h.PendingSave = nil
			p.Level = binary.LittleEndian.Uint32(mem[0x35157:])
			if p.Level >= 64 {
				h.Saved = nil
				return
			}
			p.Snapshot = append([]hdWindowDraw(nil), h.Draws...)
			delete(h.Saved, p.Level)
			h.PendingSave = p
			h.event("window_save_enter", step, eip, p)
			return
		}
		if eip == 0x1b245 {
			h.Draws, h.PendingDraw, h.PendingRestore = nil, nil, nil
			level := binary.LittleEndian.Uint32(mem[0x35157:])
			if level == 0 || level > 64 {
				h.Saved = nil
				return
			}
			p.Level = level - 1
			snapshot, ok := h.Saved[p.Level]
			if !ok {
				return
			}
			p.Snapshot = append([]hdWindowDraw(nil), snapshot...)
			delete(h.Saved, p.Level)
			h.PendingRestore = p
			h.event("window_restore_enter", step, eip, p)
			return
		}
		h.PendingDraw = nil
		if uint64(esp)+20 > uint64(len(mem)) {
			h.Draws = nil
			return
		}
		x := binary.LittleEndian.Uint32(mem[uint64(esp)+4:])
		y := binary.LittleEndian.Uint32(mem[uint64(esp)+8:])
		cols := binary.LittleEndian.Uint32(mem[uint64(esp)+12:])
		rows := binary.LittleEndian.Uint32(mem[uint64(esp)+16:])
		if x > 80 || y > 400 || cols < 3 || cols > 40 || rows < 3 || rows > 25 || x*8+cols*16 > 640 || y+rows*16 > 400 {
			h.Draws = nil
			return
		}
		p.Draw.Rect = image.Rect(int(x*8), int(y), int(x*8+cols*16), int(y+rows*16))
		kept := h.Draws[:0]
		for _, draw := range h.Draws {
			if draw.Rect != p.Draw.Rect {
				kept = append(kept, draw)
			}
		}
		h.Draws = kept
		h.PendingDraw = p
		h.event("window_draw_enter", step, eip, p)
	}
}

type hdWindow struct {
	Source, Asset *image.NRGBA
	Scale         int
	Reason        string
}

func newHDWindow(source, asset image.Image, scale int) *hdWindow {
	h := &hdWindow{Scale: scale, Reason: "fallback_source_shape"}
	if scale < 1 || scale > 4 || source == nil || source.Bounds() != image.Rect(0, 0, 144, 16) {
		return h
	}
	h.Source = image.NewNRGBA(source.Bounds())
	for y := 0; y < 16; y++ {
		for x := 0; x < 144; x++ {
			c := rgbaAt(source, x, y)
			if c.A != 255 {
				h.Reason = "fallback_source_transparency"
				return h
			}
			h.Source.SetNRGBA(x, y, c)
		}
	}
	h.Reason = "fallback_missing_asset"
	if asset == nil {
		return h
	}
	b := asset.Bounds()
	if b.Min != image.Pt(0, 0) || b.Dx() != b.Dy() || b.Dx()%3 != 0 || b.Dx() < 48*scale || b.Dx() > 8192 {
		h.Reason = "fallback_hd_shape"
		return h
	}
	prepared, reason := prepareHDImage(asset, 48, 48, scale)
	if reason != "" {
		h.Reason = "fallback_hd_invalid"
		return h
	}
	h.Asset, h.Reason = prepared.(*image.NRGBA), "ready"
	return h
}

func hdWindowTile(x, y, width, height int) int {
	col, row := 1, 1
	if x < 16 {
		col = 0
	} else if x >= width-16 {
		col = 2
	}
	if y < 16 {
		row = 0
	} else if y >= height-16 {
		row = 2
	}
	return row*3 + col
}

func (h *hdWindow) apply(out *image.NRGBA, native image.Image, rect image.Rectangle, enabled bool) (string, int) {
	if !enabled {
		return "original", 0
	}
	if h == nil {
		return "fallback_missing_asset", 0
	}
	if h.Reason != "ready" {
		return h.Reason, 0
	}
	if out == nil || native == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*h.Scale, 400*h.Scale) || rect.Dx() < 48 || rect.Dy() < 48 || rect.Dx()%16 != 0 || rect.Dy()%16 != 0 || !rect.In(native.Bounds()) {
		return "fallback_geometry", 0
	}
	w, height, scale := rect.Dx(), rect.Dy(), h.Scale
	// 原版文字可在內頁或邊框上；四角則必須完整符合已證實的來源。
	for _, corner := range []image.Point{{0, 0}, {w - 16, 0}, {0, height - 16}, {w - 16, height - 16}} {
		for y := corner.Y; y < corner.Y+16; y++ {
			for x := corner.X; x < corner.X+16; x++ {
				tile := hdWindowTile(x, y, w, height)
				if rgbaAt(native, rect.Min.X+x, rect.Min.Y+y) != h.Source.NRGBAAt(tile*16+x%16, y%16) {
					for ny := rect.Min.Y; ny < rect.Max.Y; ny++ {
						for nx := rect.Min.X; nx < rect.Max.X; nx++ {
							for dy := 0; dy < scale; dy++ {
								for dx := 0; dx < scale; dx++ {
									out.SetNRGBA(nx*scale+dx, ny*scale+dy, rgbaAt(native, nx, ny))
								}
							}
						}
					}
					return "fallback_source_mismatch", 0
				}
			}
		}
	}
	foreground := 0
	for y := 0; y < height; y++ {
		for x := 0; x < w; x++ {
			tile := hdWindowTile(x, y, w, height)
			c := rgbaAt(native, rect.Min.X+x, rect.Min.Y+y)
			isForeground := c != h.Source.NRGBAAt(tile*16+x%16, y%16)
			if isForeground {
				foreground += scale * scale
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					pixel := c
					if !isForeground {
						pixel = h.Asset.NRGBAAt((tile%3*16+x%16)*scale+dx, (tile/3*16+y%16)*scale+dy)
					}
					out.SetNRGBA((rect.Min.X+x)*scale+dx, (rect.Min.Y+y)*scale+dy, pixel)
				}
			}
		}
	}
	return "hd", foreground
}
