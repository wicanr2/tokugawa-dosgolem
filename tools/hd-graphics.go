package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

type hdGraphicsState struct {
	Draws     []graphicsDrawRecord `json:"verified_draws"`
	Pending   *graphicsPending     `json:"-"`
	Completed uint64               `json:"completed_returns"`
}

// 只扣除後畫的不透明矩形，原始繪圖參數與來源／目的矩形仍保留。
func graphicsSubtract(r, cut image.Rectangle) []image.Rectangle {
	c := r.Intersect(cut)
	if c.Empty() {
		return []image.Rectangle{r}
	}
	var result []image.Rectangle
	for _, q := range []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, c.Min.Y),
		image.Rect(r.Min.X, c.Max.Y, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, c.Min.Y, c.Min.X, c.Max.Y),
		image.Rect(c.Max.X, c.Min.Y, r.Max.X, c.Max.Y),
	} {
		if !q.Empty() {
			result = append(result, q)
		}
	}
	return result
}

func graphicsFragmentCount(draws []graphicsDrawRecord) int {
	n := 0
	for _, d := range draws {
		n += len(d.VisibleRects)
	}
	return n
}

// 使用已核對的較晚自然返回；切割呈現副本，不改原始繪圖紀錄。
func graphicsOcclude(draw graphicsDrawRecord, windows []hdWindowDraw, portraits []hdKaoDraw) graphicsDrawRecord {
	visible := append([]image.Rectangle(nil), draw.VisibleRects...)
	if draw.VisibleRects == nil {
		visible = []image.Rectangle{draw.DestRect}
	}
	cut := func(rect image.Rectangle, step int) {
		if step <= draw.EndStep {
			return
		}
		if rect.Empty() || !rect.In(image.Rect(0, 0, 640, 400)) {
			visible = []image.Rectangle{}
			return
		}
		var next []image.Rectangle
		for _, r := range visible {
			next = append(next, graphicsSubtract(r, rect)...)
		}
		if len(next) > 256 {
			next = nil
		}
		visible = append([]image.Rectangle{}, next...)
	}
	for _, window := range windows {
		cut(window.Rect, window.EndStep)
	}
	for _, portrait := range portraits {
		cut(portrait.Rect, portrait.EndStep)
	}
	draw.VisibleRects = visible
	return draw
}

func graphicsDestination(esp uint32, mem []byte) (image.Rectangle, bool) {
	if uint64(esp)+24 > uint64(len(mem)) {
		return image.Rectangle{}, false
	}
	cols := binary.LittleEndian.Uint32(mem[uint64(esp)+8:])
	rows := binary.LittleEndian.Uint32(mem[uint64(esp)+12:])
	pitch := binary.LittleEndian.Uint32(mem[uint64(esp)+16:])
	dest := binary.LittleEndian.Uint32(mem[uint64(esp)+20:])
	if cols < 1 || cols > 80 || rows < 1 || rows > 400 || pitch != 80 || dest < 0xa0000 || dest >= 0xa0000+32000 {
		return image.Rectangle{}, false
	}
	x, y := (dest-0xa0000)%80*8, (dest-0xa0000)/80
	r := image.Rect(int(x), int(y), int(x+cols*8), int(y+rows))
	return r, r.In(image.Rect(0, 0, 640, 400))
}

// 只觀測原版來源及自然返回，所有圖像替換留在 host 畫布。
func (h *hdGraphicsState) observe(step int, eip, esp uint32, mem []byte, snapshot func() (nativeSnapshot, error)) {
	entry := eip == 0x1c1bc || eip == 0x1c084 || eip == 0x11f14 || openingPartialEntry(eip)
	returned := h.Pending != nil && eip == h.Pending.Record.Return && uint64(esp) == uint64(h.Pending.Record.Stack)+4
	if !entry && !returned {
		return
	}
	if entry {
		h.Pending = nil
		if eip == 0x1c084 || eip == 0x11f14 {
			h.Draws = nil
		} else {
			r, ok := graphicsDestination(esp, mem)
			if openingPartialEntry(eip) {
				r, ok = openingPartialDestination(eip), true
			}
			if !ok {
				h.Draws = nil
			} else {
				keep := h.Draws[:0]
				for _, draw := range h.Draws {
					visible := draw.VisibleRects
					if visible == nil {
						visible = []image.Rectangle{draw.DestRect}
					}
					draw.VisibleRects = nil
					for _, old := range visible {
						draw.VisibleRects = append(draw.VisibleRects, graphicsSubtract(old, r)...)
					}
					if len(draw.VisibleRects) > 0 {
						keep = append(keep, draw)
					}
				}
				h.Draws = keep
				if graphicsFragmentCount(h.Draws) > 256 {
					h.Draws = nil
				}
			}
		}
	}
	trace := graphicsTrace{Profiles: hdGraphicsSources, Pending: h.Pending}
	trace.observe(step, eip, esp, mem, snapshot)
	h.Pending = trace.Pending
	for _, draw := range trace.Records {
		if draw.Verified {
			draw.VisibleRects = []image.Rectangle{draw.DestRect}
			if len(h.Draws) == 64 {
				h.Draws = h.Draws[1:]
			}
			h.Draws = append(h.Draws, draw)
			if graphicsFragmentCount(h.Draws) > 256 {
				h.Draws = nil
			}
			h.Completed++
		}
	}
}

type hdGraphicsPair struct {
	Profile hdGraphicsProfile
	Source  []byte
	Asset   image.Image
	Scale   int
	Reason  string
}

type hdGraphicsReport struct {
	Resource        string            `json:"resource"`
	Rect            image.Rectangle   `json:"rect"`
	Mode            string            `json:"mode"`
	PaintedNative   int               `json:"painted_native_pixels"`
	PreservedNative int               `json:"preserved_native_pixels,omitempty"`
	VisibleRects    []image.Rectangle `json:"visible_rectangles,omitempty"`
	Effect          *graphicsEffect   `json:"palette_effect,omitempty"`
}

func newHDGraphicsPair(profile hdGraphicsProfile, source []byte, asset image.Image, scale int) hdGraphicsPair {
	p := hdGraphicsPair{Profile: profile, Scale: scale, Reason: "fallback_source"}
	if len(source) != profile.Source.Width*profile.Source.Height || fmt.Sprintf("%x", sha256.Sum256(source)) != profile.IndicesSHA {
		return p
	}
	p.Source = append([]byte(nil), source...)
	p.Profile.PaintRects = append([]image.Rectangle(nil), profile.PaintRects...)
	p.Profile.Preserve = append([]hdGraphicsPreserve(nil), profile.Preserve...)
	if profile.PaintRect != (image.Rectangle{}) && (profile.PaintRect.Empty() || !profile.PaintRect.In(image.Rect(0, 0, profile.Source.Width, profile.Source.Height))) {
		p.Reason = "fallback_paint_region"
		return p
	}
	if len(profile.PaintRects) > 64 || len(profile.Preserve) > 64 || (len(profile.PaintRects) > 0 && profile.PaintRect != (image.Rectangle{})) {
		p.Reason = "fallback_paint_region"
		return p
	}
	for i, rect := range profile.PaintRects {
		if rect.Empty() || !rect.In(image.Rect(0, 0, profile.Source.Width, profile.Source.Height)) {
			p.Reason = "fallback_paint_region"
			return p
		}
		for _, old := range profile.PaintRects[:i] {
			if rect.Overlaps(old) {
				p.Reason = "fallback_paint_region"
				return p
			}
		}
	}
	for _, region := range profile.Preserve {
		if region.Rect.Empty() || !region.Rect.In(image.Rect(0, 0, profile.Source.Width, profile.Source.Height)) || region.Indices == 0 {
			p.Reason = "fallback_preserve_region"
			return p
		}
	}
	p.Reason = "fallback_missing_asset"
	if asset == nil {
		return p
	}
	var reason string
	p.Asset, reason = prepareHDImage(asset, profile.Source.Width, profile.Source.Height, scale)
	if reason != "" {
		p.Reason = "fallback_asset_geometry"
		return p
	}
	p.Reason = "ready"
	return p
}

func (p hdGraphicsPair) preserves(x, y int) bool {
	index := p.Source[y*p.Profile.Source.Width+x]
	for _, region := range p.Profile.Preserve {
		if image.Pt(x, y).In(region.Rect) && region.Indices&(uint16(1)<<index) != 0 {
			return true
		}
	}
	return false
}

func (p hdGraphicsPair) apply(out *image.NRGBA, s nativeSnapshot, draw graphicsDrawRecord, enabled bool) hdGraphicsReport {
	r := hdGraphicsReport{Resource: draw.Name, Rect: draw.DestRect, Mode: "original"}
	if !enabled {
		return r
	}
	r.Mode = p.Reason
	if p.Reason != "ready" {
		return r
	}
	r.Mode = "fallback_geometry"
	if out == nil || out.Bounds() != image.Rect(0, 0, 640*p.Scale, 400*p.Scale) || s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 400) || len(s.Indexed) != 640*400 || !draw.DestRect.In(image.Rect(0, 0, 640, 400)) || draw.DestRect.Empty() || draw.DestRect.Size() != draw.SourceRect.Size() {
		return r
	}
	r.Mode = "fallback_no_verified_draw"
	if !draw.Verified || draw.Name != p.Profile.Source.Name {
		return r
	}
	expected := graphicsCrop(p.Source, p.Profile.Source.Width, draw.SourceRect)
	if expected == nil || fmt.Sprintf("%x", sha256.Sum256(expected)) != draw.ExpectedSHA {
		return r
	}
	r.Mode = "fallback_palette"
	if s.MaxColor != 15 {
		return r
	}
	effect := graphicsPaletteEffect(p.Profile, s)
	if !effect.Valid {
		return r
	}
	if graphicsEffectSource(p.Profile.Source.Name) {
		r.Effect = &effect
	}
	visible := draw.VisibleRects
	if visible == nil {
		visible = []image.Rectangle{draw.DestRect}
	}
	r.Mode = "fallback_visibility"
	if len(visible) > 256 {
		return r
	}
	paints := p.Profile.PaintRects
	if len(paints) == 0 {
		paint := p.Profile.PaintRect
		if paint.Empty() {
			paint = image.Rect(0, 0, p.Profile.Source.Width, p.Profile.Source.Height)
		}
		paints = []image.Rectangle{paint}
	}
	var fragments []image.Rectangle
	for i, f := range visible {
		if f.Empty() || !f.In(draw.DestRect) {
			return r
		}
		for _, old := range visible[:i] {
			if !old.Intersect(f).Empty() {
				return r
			}
		}
		for _, paint := range paints {
			allowed := draw.SourceRect.Intersect(paint).Add(draw.DestRect.Min.Sub(draw.SourceRect.Min))
			if rect := f.Intersect(allowed); !rect.Empty() {
				fragments = append(fragments, rect)
			}
		}
	}
	if len(fragments) > 256 {
		return r
	}
	r.Mode = "fallback_no_visible_art"
	if len(fragments) == 0 {
		return r
	}
	r.Mode = "fallback_source_mismatch"
	for _, f := range fragments {
		src := f.Add(draw.SourceRect.Min.Sub(draw.DestRect.Min))
		want, actual := graphicsCrop(p.Source, p.Profile.Source.Width, src), graphicsCrop(s.Indexed, 640, f)
		if len(actual) != len(want) || fmt.Sprintf("%x", sha256.Sum256(actual)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
			return r
		}
	}
	for _, f := range fragments {
		src := f.Add(draw.SourceRect.Min.Sub(draw.DestRect.Min))
		for y := 0; y < f.Dy(); y++ {
			for x := 0; x < f.Dx(); x++ {
				preserved := p.preserves(src.Min.X+x, src.Min.Y+y)
				if preserved {
					r.PreservedNative++
				} else {
					r.PaintedNative++
				}
				for dy := 0; dy < p.Scale; dy++ {
					for dx := 0; dx < p.Scale; dx++ {
						pixel := rgbaAt(p.Asset, (src.Min.X+x)*p.Scale+dx, (src.Min.Y+y)*p.Scale+dy)
						if preserved {
							pixel = rgbaAt(s.Image, f.Min.X+x, f.Min.Y+y)
						} else {
							pixel = effect.colour(pixel)
						}
						out.SetNRGBA((f.Min.X+x)*p.Scale+dx, (f.Min.Y+y)*p.Scale+dy, pixel)
					}
				}
			}
		}
	}
	r.VisibleRects = fragments
	r.Mode = "hd"
	return r
}
