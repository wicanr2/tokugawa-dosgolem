package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
)

type terrainScenePixel struct {
	Floor, Backdrop int
	Order, Owner    uint64
	Color           color.NRGBA
	Known           bool
	Affected        bool
	VisibleSprite   bool
}

type hdTerrainSceneReport struct {
	Reason                string   `json:"reason"`
	FloorModes            []string `json:"floor_modes"`
	SpriteModes           []string `json:"sprite_modes"`
	FloorHD               int      `json:"floor_hd_count"`
	TreeHD                int      `json:"tree_hd_count"`
	ObjectHD              int      `json:"object_hd_count"`
	UnitHD                int      `json:"unit_hd_count"`
	FloorNativePixels     int      `json:"floor_native_pixels"`
	ProtectedNativePixels int      `json:"protected_original_foreground_pixels"`
	UIOutputPixels        int      `json:"preserved_UI_output_pixels"`
	UIHDOutputPixels      int      `json:"HD_frame_underlay_output_pixels"`
}

type hdTerrainScene struct {
	Pixels     []terrainScenePixel
	ModelValid bool
}

type sceneBackground func(x, y, dx, dy int, index byte) (color.NRGBA, bool)

func spriteBackgroundValid(s hdTerrainSpriteDraw, source *image.NRGBA) bool {
	return s.BackgroundVerified && s.Order > 0 && s.BeginStep < s.EndStep && s.Background != nil &&
		s.Background.Bounds() == image.Rect(0, 0, 48, 48) && source != nil &&
		s.BackgroundSHA == fmt.Sprintf("%x", sha256.Sum256(s.Background.Pix)) &&
		s.SourceSHA == fmt.Sprintf("%x", sha256.Sum256(source.Pix))
}

// 此畫布只讀原版 native、來源與已自然返回的 journal，不回寫 guest。
func (h *hdTerrainScene) apply(out *image.NRGBA, native image.Image, state hdTerrainState, floors map[int]hdTerrainPair, pairs map[hdTerrainSpriteKey]hdTerrainSpritePair, sources map[hdTerrainSpriteKey]*image.NRGBA, scale int, enabled bool, backgroundsHD ...sceneBackground) hdTerrainSceneReport {
	h.ModelValid = false
	r := hdTerrainSceneReport{Reason: "original", FloorModes: make([]string, len(state.History)), SpriteModes: make([]string, len(state.Sprites))}
	fail := func(reason string) hdTerrainSceneReport {
		r.Reason = reason
		for i := range r.FloorModes {
			r.FloorModes[i] = reason
		}
		for i := range r.SpriteModes {
			r.SpriteModes[i] = reason
		}
		return r
	}
	if !enabled {
		return fail("original")
	}
	if len(state.History) == 0 {
		return fail("fallback_no_floor_history")
	}
	if state.Pending != nil {
		return fail("fallback_pending_draw")
	}
	if scale < 1 || scale > 4 || out == nil || native == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*scale, 400*scale) || len(state.History) > 512 || len(state.Sprites) > 512 {
		return fail("fallback_geometry")
	}
	if len(h.Pixels) != 640*400 {
		h.Pixels = make([]terrainScenePixel, 640*400)
	}
	for i := range h.Pixels {
		h.Pixels[i] = terrainScenePixel{Floor: -1, Backdrop: -1}
	}
	last := uint64(0)
	orders := map[uint64]bool{}
	for i, d := range state.History {
		p := floors[d.Index]
		if d.Order <= last || d.Rect.Dx() != 48 || d.Rect.Dy() != 32 || !d.Rect.In(native.Bounds()) || p.Source == nil || p.Source.Bounds() != image.Rect(0, 0, 48, 32) {
			return fail("fallback_floor_journal")
		}
		last = d.Order
		orders[d.Order] = true
		for y := d.Rect.Min.Y; y < d.Rect.Max.Y; y++ {
			for x := d.Rect.Min.X; x < d.Rect.Max.X; x++ {
				c := p.Source.NRGBAAt(x-d.Rect.Min.X, y-d.Rect.Min.Y)
				if c.A != 255 {
					return fail("fallback_background_source")
				}
				h.Pixels[y*640+x] = terrainScenePixel{Floor: i, Backdrop: -1, Order: d.Order, Owner: d.Order, Color: c, Known: true}
			}
		}
	}
	last = 0
	backgrounds := make([]bool, len(state.Sprites))
	indexedBackgrounds := make([]bool, len(state.Sprites))
	for i, s := range state.Sprites {
		im := sources[hdTerrainSpriteKey{s.Group, s.Index}]
		if s.Order <= last || orders[s.Order] || s.Rect.Dx() != 48 || s.Rect.Dy() != 48 || !s.Rect.In(native.Bounds()) || im == nil || im.Bounds() != image.Rect(0, 0, 48, 48) {
			return fail("fallback_sprite_journal")
		}
		last = s.Order
		orders[s.Order] = true
		backgrounds[i] = spriteBackgroundValid(s, im)
		indexedBackgrounds[i] = len(s.BackgroundIndexed) == 48*48 && s.BackgroundIndexedSHA == fmt.Sprintf("%x", sha256.Sum256(s.BackgroundIndexed))
		if !backgrounds[i] {
			continue
		}
		for y := s.Rect.Min.Y; y < s.Rect.Max.Y; y++ {
			for x := s.Rect.Min.X; x < s.Rect.Max.X; x++ {
				p := &h.Pixels[y*640+x]
				if p.Known {
					continue
				}
				c := s.Background.NRGBAAt(x-s.Rect.Min.X, y-s.Rect.Min.Y)
				if c.A != 255 {
					return fail("fallback_background_source")
				}
				*p = terrainScenePixel{Floor: -1, Backdrop: i, Order: s.Order - 1, Color: c, Known: true}
			}
		}
	}
	for _, s := range state.Sprites {
		im := sources[hdTerrainSpriteKey{s.Group, s.Index}]
		for y := s.Rect.Min.Y; y < s.Rect.Max.Y; y++ {
			for x := s.Rect.Min.X; x < s.Rect.Max.X; x++ {
				p := &h.Pixels[y*640+x]
				if !p.Known {
					continue
				}
				c := im.NRGBAAt(x-s.Rect.Min.X, y-s.Rect.Min.Y)
				if c.A != 0 && c.A != 255 {
					return fail("fallback_foreground_source")
				}
				if c.A == 0 {
					continue
				}
				if s.Order <= p.Order {
					if p.Floor < 0 {
						if c == p.Color {
							p.Owner = s.Order
						} else {
							p.Owner = 0
						}
					}
					continue
				}
				p.Color, p.Owner = c, s.Order
			}
		}
	}
	match := func(rect image.Rectangle) bool {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				p := h.Pixels[y*640+x]
				if !p.Known || rgbaAt(native, x, y) != p.Color {
					return false
				}
			}
		}
		return true
	}
	h.ModelValid = true
	for _, s := range state.Sprites {
		if !match(s.Rect) {
			continue
		}
		for y := s.Rect.Min.Y; y < s.Rect.Max.Y; y++ {
			for x := s.Rect.Min.X; x < s.Rect.Max.X; x++ {
				p := &h.Pixels[y*640+x]
				if p.Owner == s.Order {
					p.VisibleSprite = true
				}
			}
		}
	}
	for i, d := range state.History {
		p := floors[d.Index]
		mode := p.Reason
		if mode == "ready" {
			if p.Asset == nil || p.Asset.Bounds() != image.Rect(0, 0, 48*scale, 32*scale) {
				mode = "fallback_asset_geometry"
			} else if !match(d.Rect) {
				mode = "fallback_scene_mismatch"
			} else {
				visible := 0
				for y := d.Rect.Min.Y; y < d.Rect.Max.Y; y++ {
					for x := d.Rect.Min.X; x < d.Rect.Max.X; x++ {
						if h.Pixels[y*640+x].Floor == i {
							visible++
						}
					}
				}
				mode = "fallback_overwritten_floor"
				if visible > 0 {
					mode = "hd"
					r.FloorHD++
					r.FloorNativePixels += visible
				}
			}
		}
		r.FloorModes[i] = mode
	}
	approved := map[uint64]int{}
	for i, s := range state.Sprites {
		key := hdTerrainSpriteKey{s.Group, s.Index}
		p, ok := pairs[key]
		mode := "fallback_unapproved_source"
		if ok {
			mode = p.Reason
		}
		if mode == "ready" {
			backgroundAvailable := true
			for y := s.Rect.Min.Y; y < s.Rect.Max.Y; y++ {
				for x := s.Rect.Min.X; x < s.Rect.Max.X; x++ {
					pixel := h.Pixels[y*640+x]
					if pixel.Floor < 0 && (pixel.Backdrop < 0 || state.Sprites[pixel.Backdrop].Order > s.Order) {
						backgroundAvailable = false
					}
				}
			}
			if p.Key != key || p.Asset == nil || p.Asset.Bounds() != image.Rect(0, 0, 48*scale, 48*scale) || p.Source == nil || !bytes.Equal(p.Source.Pix, sources[key].Pix) {
				mode = "fallback_target_source"
			} else if !backgroundAvailable {
				mode = "fallback_background_unavailable"
			} else if !match(s.Rect) {
				mode = "fallback_scene_mismatch"
			} else {
				visible := 0
				for y := s.Rect.Min.Y; y < s.Rect.Max.Y; y++ {
					for x := s.Rect.Min.X; x < s.Rect.Max.X; x++ {
						if h.Pixels[y*640+x].Owner == s.Order {
							visible++
						}
					}
				}
				mode = "fallback_overwritten_sprite"
				if visible > 0 {
					mode = "hd"
					approved[s.Order] = i
					if s.Group == "units" {
						r.UnitHD++
					} else if s.Group == "objects" {
						r.ObjectHD++
						if s.Index == 0 {
							r.TreeHD++
						}
					}
				}
			}
		}
		r.SpriteModes[i] = mode
	}
	if r.FloorHD+r.ObjectHD+r.UnitHD == 0 {
		r.Reason = "fallback_no_qualified_layers"
		return r
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			p := h.Pixels[y*640+x]
			floorHD := p.Floor >= 0 && r.FloorModes[p.Floor] == "hd"
			affected := floorHD
			for _, i := range approved {
				if image.Pt(x, y).In(state.Sprites[i].Rect) {
					affected = true
					break
				}
			}
			if !affected || !p.Known {
				continue
			}
			h.Pixels[y*640+x].Affected = true
			ownerIndex, ownerHD := approved[p.Owner]
			foreground := p.Owner != 0 && (p.Floor < 0 || p.Owner != p.Order)
			if foreground && !ownerHD {
				r.ProtectedNativePixels++
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					var c color.NRGBA
					var hdBackground color.NRGBA
					hasHDBackground := false
					if p.Floor >= 0 {
						d := state.History[p.Floor]
						pair := floors[d.Index]
						c = pair.Source.NRGBAAt(x-d.Rect.Min.X, y-d.Rect.Min.Y)
						if floorHD {
							c = pair.Asset.NRGBAAt((x-d.Rect.Min.X)*scale+dx, (y-d.Rect.Min.Y)*scale+dy)
						}
					} else {
						s := state.Sprites[p.Backdrop]
						c = s.Background.NRGBAAt(x-s.Rect.Min.X, y-s.Rect.Min.Y)
						if len(backgroundsHD) == 1 && backgroundsHD[0] != nil && indexedBackgrounds[p.Backdrop] {
							n := (y-s.Rect.Min.Y)*48 + x - s.Rect.Min.X
							hdBackground, hasHDBackground = backgroundsHD[0](x, y, dx, dy, s.BackgroundIndexed[n])
							if hasHDBackground {
								c = hdBackground
							}
						}
					}
					for i, s := range state.Sprites {
						if s.Order <= p.Order || !image.Pt(x, y).In(s.Rect) {
							continue
						}
						key := hdTerrainSpriteKey{s.Group, s.Index}
						source := sources[key].NRGBAAt(x-s.Rect.Min.X, y-s.Rect.Min.Y)
						if r.SpriteModes[i] == "hd" {
							if ownerHD && i == ownerIndex {
								continue
							}
							if p.Floor < 0 && source.A == 0 {
								continue
							}
							c = blendTerrainAlpha(c, pairs[key].Asset.NRGBAAt((x-s.Rect.Min.X)*scale+dx, (y-s.Rect.Min.Y)*scale+dy))
						} else if source.A == 255 {
							c = source
						}
					}
					if ownerHD {
						s := state.Sprites[ownerIndex]
						key := hdTerrainSpriteKey{s.Group, s.Index}
						c = blendTerrainAlpha(c, pairs[key].Asset.NRGBAAt((x-s.Rect.Min.X)*scale+dx, (y-s.Rect.Min.Y)*scale+dy))
					} else if foreground {
						c = p.Color
					}
					if p.Floor < 0 && p.Owner == 0 {
						if hasHDBackground {
							c = hdBackground
							r.UIHDOutputPixels++
						} else {
							c = p.Color
							r.UIOutputPixels++
						}
					}
					out.SetNRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	r.Reason = "hd"
	return r
}
