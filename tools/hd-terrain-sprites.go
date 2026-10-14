package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

type hdTerrainTreePair struct {
	Source, Asset *image.NRGBA
	Reason        string
}

func newHDTerrainTreePair(source, asset image.Image, scale int) hdTerrainTreePair {
	p := hdTerrainTreePair{Source: newHDTerrainSpriteSource(source), Reason: "fallback_source_shape"}
	if p.Source == nil || scale < 1 || scale > 4 {
		return p
	}
	var bounds image.Rectangle
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			if p.Source.NRGBAAt(x, y).A == 255 {
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if bounds != hdTerrainTreeSourceBounds {
		return p
	}
	p.Reason = "fallback_missing_asset"
	if asset == nil {
		return p
	}
	b := asset.Bounds()
	if b.Min != image.Pt(0, 0) || b.Dx() != b.Dy() || b.Dx() < 48 || b.Dx() > 8192 || hdTerrainTreeCrop.Empty() || !hdTerrainTreeCrop.In(b) {
		p.Reason = "fallback_asset_geometry"
		return p
	}
	transparent, visible := false, false
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			a := rgbaAt(asset, x, y).A
			transparent = transparent || a == 0
			visible = visible || a > 0
			if !image.Pt(x, y).In(hdTerrainTreeCrop) && a > 7 {
				p.Reason = "fallback_unapproved_crop"
				return p
			}
		}
	}
	if !transparent || !visible {
		p.Reason = "fallback_asset_alpha"
		return p
	}
	p.Asset = image.NewNRGBA(image.Rect(0, 0, 48*scale, 48*scale))
	target := hdTerrainTreeSourceBounds
	for y := 0; y < target.Dy()*scale; y++ {
		for x := 0; x < target.Dx()*scale; x++ {
			c := rgbaAt(asset, hdTerrainTreeCrop.Min.X+x*hdTerrainTreeCrop.Dx()/(target.Dx()*scale), hdTerrainTreeCrop.Min.Y+y*hdTerrainTreeCrop.Dy()/(target.Dy()*scale))
			p.Asset.SetNRGBA(target.Min.X*scale+x, target.Min.Y*scale+y, c)
		}
	}
	p.Reason = "ready"
	return p
}

func blendTerrainAlpha(back, front color.NRGBA) color.NRGBA {
	a := uint32(front.A)
	channel := func(b, f uint8) uint8 { return uint8((uint32(f)*a + uint32(b)*(255-a) + 127) / 255) }
	return color.NRGBA{channel(back.R, front.R), channel(back.G, front.G), channel(back.B, front.B), 255}
}

// 完整背景與原始前景逐點核對後，在新畫布重建。此函式不改 native 或原版狀態。
func (p hdTerrainTreePair) apply(out *image.NRGBA, native image.Image, target hdTerrainSpriteDraw, state hdTerrainState, floors map[int]hdTerrainPair, floorModes []string, sources map[hdTerrainSpriteKey]*image.NRGBA, replaced []image.Rectangle, scale int, enabled bool) (string, int) {
	if !enabled {
		return "original", 0
	}
	if target.Group != "objects" || target.Index != 0 {
		return "fallback_unapproved_source", 0
	}
	if p.Reason != "ready" {
		return p.Reason, 0
	}
	if state.Pending != nil {
		return "fallback_pending_draw", 0
	}
	r := target.Rect
	if native == nil || out == nil || scale < 1 || scale > 4 || native.Bounds() != image.Rect(0, 0, 640, 400) || out.Bounds() != image.Rect(0, 0, 640*scale, 400*scale) || r.Dx() != 48 || r.Dy() != 48 || !r.In(native.Bounds()) || p.Asset == nil || p.Asset.Bounds() != image.Rect(0, 0, 48*scale, 48*scale) || len(floorModes) != len(state.Draws) {
		return "fallback_geometry", 0
	}
	for _, old := range replaced {
		if old.Overlaps(r) {
			return "fallback_replaced_overlap", 0
		}
	}
	observedBackground := target.BackgroundVerified && target.Background != nil && target.Background.Bounds() == image.Rect(0, 0, 48, 48) &&
		target.BeginStep < target.EndStep && target.BackgroundSHA == fmt.Sprintf("%x", sha256.Sum256(target.Background.Pix)) &&
		target.SourceSHA == fmt.Sprintf("%x", sha256.Sum256(p.Source.Pix))
	var base [48 * 48]int
	var expected [48 * 48]color.NRGBA
	var floorOrder, owner [48 * 48]uint64
	for n := range base {
		base[n] = -1
	}
	for i, floor := range state.Draws {
		if !floor.Rect.Overlaps(r) {
			continue
		}
		pair := floors[floor.Index]
		if pair.Source == nil || pair.Source.Bounds() != image.Rect(0, 0, 48, 32) || floor.Rect.Dx() != 48 || floor.Rect.Dy() != 32 {
			return "fallback_background_source", 0
		}
		if floorModes[i] == "hd" && (pair.Asset == nil || pair.Asset.Bounds() != image.Rect(0, 0, 48*scale, 32*scale)) {
			return "fallback_background_source", 0
		}
		inter := floor.Rect.Intersect(r)
		for y := inter.Min.Y; y < inter.Max.Y; y++ {
			for x := inter.Min.X; x < inter.Max.X; x++ {
				n := (y-r.Min.Y)*48 + x - r.Min.X
				if base[n] != -1 {
					return "fallback_ambiguous_background", 0
				}
				c := pair.Source.NRGBAAt(x-floor.Rect.Min.X, y-floor.Rect.Min.Y)
				if c.A != 255 {
					return "fallback_background_source", 0
				}
				base[n], expected[n], floorOrder[n], owner[n] = i, c, floor.Order, floor.Order
			}
		}
	}
	for n, i := range base {
		if i < 0 {
			if !observedBackground || target.Order == 0 {
				return "fallback_background_incomplete", 0
			}
			expected[n] = target.Background.NRGBAAt(n%48, n/48)
			if expected[n].A != 255 {
				return "fallback_background_source", 0
			}
			floorOrder[n] = target.Order - 1
			wx, wy := r.Min.X+n%48, r.Min.Y+n/48
			for _, s := range state.Sprites {
				if s.Order >= target.Order || !image.Pt(wx, wy).In(s.Rect) {
					continue
				}
				im := sources[hdTerrainSpriteKey{s.Group, s.Index}]
				if im == nil || im.Bounds() != image.Rect(0, 0, 48, 48) {
					return "fallback_foreground_source", 0
				}
				if c := im.NRGBAAt(wx-s.Rect.Min.X, wy-s.Rect.Min.Y); c.A == 255 {
					owner[n] = 0
					if c == expected[n] {
						owner[n] = s.Order
					}
				}
			}
		}
	}
	found := false
	lastOrder := uint64(0)
	for _, sprite := range state.Sprites {
		if sprite.Order <= lastOrder {
			return "fallback_draw_order", 0
		}
		lastOrder = sprite.Order
		if !sprite.Rect.Overlaps(r) {
			continue
		}
		source := sources[hdTerrainSpriteKey{sprite.Group, sprite.Index}]
		if source == nil || source.Bounds() != image.Rect(0, 0, 48, 48) || sprite.Rect.Dx() != 48 || sprite.Rect.Dy() != 48 || !sprite.Rect.In(native.Bounds()) {
			return "fallback_foreground_source", 0
		}
		if sprite.Order == target.Order {
			if sprite.Group != target.Group || sprite.Index != target.Index || sprite.Rect != target.Rect || !bytes.Equal(source.Pix, p.Source.Pix) {
				return "fallback_target_source", 0
			}
			found = true
		}
		inter := sprite.Rect.Intersect(r)
		for y := inter.Min.Y; y < inter.Max.Y; y++ {
			for x := inter.Min.X; x < inter.Max.X; x++ {
				n := (y-r.Min.Y)*48 + x - r.Min.X
				if sprite.Order <= floorOrder[n] {
					continue
				}
				c := source.NRGBAAt(x-sprite.Rect.Min.X, y-sprite.Rect.Min.Y)
				if c.A == 255 {
					expected[n], owner[n] = c, sprite.Order
				} else if c.A != 0 {
					return "fallback_foreground_source", 0
				}
			}
		}
	}
	if !found {
		return "fallback_unobserved_target", 0
	}
	visible, protected := 0, 0
	for n, c := range expected {
		if rgbaAt(native, r.Min.X+n%48, r.Min.Y+n/48) != c {
			return "fallback_scene_mismatch", 0
		}
		if owner[n] == target.Order {
			visible++
		} else if owner[n] != 0 && (base[n] < 0 || owner[n] != floorOrder[n]) {
			protected++
		}
	}
	if visible == 0 {
		return "fallback_overwritten_sprite", 0
	}
	patch := image.NewNRGBA(image.Rect(0, 0, 48*scale, 48*scale))
	for n, i := range base {
		wx, wy := r.Min.X+n%48, r.Min.Y+n/48
		for dy := 0; dy < scale; dy++ {
			for dx := 0; dx < scale; dx++ {
				var c color.NRGBA
				if i < 0 {
					c = target.Background.NRGBAAt(n%48, n/48)
				} else {
					floor := state.Draws[i]
					pair := floors[floor.Index]
					c = pair.Source.NRGBAAt(wx-floor.Rect.Min.X, wy-floor.Rect.Min.Y)
					if floorModes[i] == "hd" {
						c = pair.Asset.NRGBAAt((wx-floor.Rect.Min.X)*scale+dx, (wy-floor.Rect.Min.Y)*scale+dy)
					}
				}
				patch.SetNRGBA(n%48*scale+dx, n/48*scale+dy, c)
			}
		}
	}
	for _, sprite := range state.Sprites {
		if !sprite.Rect.Overlaps(r) {
			continue
		}
		source := sources[hdTerrainSpriteKey{sprite.Group, sprite.Index}]
		inter := sprite.Rect.Intersect(r)
		for y := inter.Min.Y; y < inter.Max.Y; y++ {
			for x := inter.Min.X; x < inter.Max.X; x++ {
				n := (y-r.Min.Y)*48 + x - r.Min.X
				if sprite.Order <= floorOrder[n] {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px, py := (x-r.Min.X)*scale+dx, (y-r.Min.Y)*scale+dy
						if sprite.Order == target.Order {
							if base[n] < 0 && p.Source.NRGBAAt(x-r.Min.X, y-r.Min.Y).A == 0 {
								continue
							}
							patch.SetNRGBA(px, py, blendTerrainAlpha(patch.NRGBAAt(px, py), p.Asset.NRGBAAt(px, py)))
						} else if c := source.NRGBAAt(x-sprite.Rect.Min.X, y-sprite.Rect.Min.Y); c.A == 255 {
							patch.SetNRGBA(px, py, c)
						}
					}
				}
			}
		}
	}
	for n, order := range owner {
		if order == 0 || order == target.Order || (base[n] >= 0 && order == floorOrder[n]) {
			continue
		}
		for dy := 0; dy < scale; dy++ {
			for dx := 0; dx < scale; dx++ {
				patch.SetNRGBA(n%48*scale+dx, n/48*scale+dy, expected[n])
			}
		}
	}
	draw.Draw(out, image.Rect(r.Min.X*scale, r.Min.Y*scale, r.Max.X*scale, r.Max.Y*scale), patch, image.Point{}, draw.Src)
	return "hd", protected
}
