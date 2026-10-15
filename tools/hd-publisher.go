package main

import (
	"image"
	"image/color"
)

const publisherAssetSHA = "d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93"

type hdPublisher struct {
	Image  *image.NRGBA
	Reason string
}

func newHDPublisher(asset image.Image) *hdPublisher {
	p := &hdPublisher{Reason: "missing_publisher_asset"}
	if asset == nil {
		return p
	}
	b := asset.Bounds()
	if b.Min != (image.Point{}) || b.Dx() < 1280 || b.Dy() < 960 || b.Dx()*3 != b.Dy()*4 {
		p.Reason = "invalid_publisher_geometry"
		return p
	}
	p.Image = image.NewNRGBA(image.Rect(0, 0, 1280, 960))
	for y := 0; y < 960; y++ {
		for x := 0; x < 1280; x++ {
			p.Image.Set(x, y, asset.At(x*b.Dx()/1280, y*b.Dy()/960))
		}
	}
	p.Reason = "ready"
	return p
}

// The original palette is untouched. Only recognizable fade shapes affect HD RGB.
func publisherPaletteEffect(target [768]byte, current [256][3]byte) (string, byte, bool) {
	level, offset := byte(0), byte(0)
	for i, want := range target {
		now := current[i/3][i%3] >> 2
		if now > level {
			level = now
		}
		if now > want {
			return "", 0, false
		}
		if want-now > offset {
			offset = want - now
		}
	}
	clamp, subtract := true, true
	for i, want := range target {
		now := current[i/3][i%3] >> 2
		if now != min(want, level) {
			clamp = false
		}
		remaining := byte(0)
		if want > offset {
			remaining = want - offset
		}
		if now != remaining {
			subtract = false
		}
	}
	if clamp {
		return "clamp", level<<2 | level>>4, true
	}
	if subtract {
		return "subtract", offset<<2 | offset>>4, true
	}
	return "", 0, false
}

func (p *hdPublisher) render(s nativeSnapshot, mark *markProbeTrace, qualified, enabled bool) (*image.NRGBA, string) {
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 960))
	if s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 480) {
		return out, "waiting_for_mark_display"
	}
	for y := 0; y < 960; y++ {
		for x := 0; x < 1280; x++ {
			out.SetNRGBA(x, y, s.Image.NRGBAAt(x/2, y/2))
		}
	}
	if !enabled {
		return out, "original"
	}
	if p == nil || p.Reason != "ready" {
		return out, "fallback_missing_publisher_asset"
	}
	if !qualified || mark == nil || !mark.TargetReady {
		return out, "fallback_unqualified_publisher"
	}
	effect, level, ok := publisherPaletteEffect(mark.Target, s.Palette)
	if !ok {
		return out, "fallback_unknown_publisher_palette"
	}
	for y := 0; y < 449*2; y++ {
		for x := 0; x < 1280; x++ {
			c := p.Image.NRGBAAt(x, y)
			if effect == "clamp" {
				c.R, c.G, c.B = min(c.R, level), min(c.G, level), min(c.B, level)
			} else {
				lower := func(v byte) byte {
					if v <= level {
						return 0
					}
					return v - level
				}
				c.R, c.G, c.B = lower(c.R), lower(c.G), lower(c.B)
			}
			c.A = 255
			out.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, c.A})
		}
	}
	return out, "hd"
}
