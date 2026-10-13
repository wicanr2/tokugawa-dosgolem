package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func scrollFixture(t *testing.T, scale int) (*image.NRGBA, *image.NRGBA, *image.NRGBA, *hdScroll) {
	t.Helper()
	source := image.NewNRGBA(image.Rect(0, 0, 640, 150))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.NRGBA{245, 234, 210, 255}}, image.Point{}, draw.Src)
	asset := image.NewNRGBA(image.Rect(0, 0, 1280, 300))
	draw.Draw(asset, asset.Bounds(), &image.Uniform{C: color.NRGBA{230, 220, 190, 255}}, image.Point{}, draw.Src)
	frame := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.NRGBA{110, 50, 40, 255}}, image.Point{}, draw.Src)
	draw.Draw(frame, scrollRect, source, image.Point{}, draw.Src)
	out, _, err := previewA0(frame, nil, nil, scale, false)
	if err != nil {
		t.Fatal(err)
	}
	return frame, source, out, newHDScroll(source, asset, scale)
}

func TestHDScrollPreservesForegroundAndOutside(t *testing.T) {
	for _, scale := range []int{1, 2, 4} {
		frame, source, out, h := scrollFixture(t, scale)
		ink := color.NRGBA{0, 0, 0, 255}
		frame.SetNRGBA(90, 280, ink)
		frame.SetNRGBA(575, 370, ink)
		frameBefore := append([]byte(nil), frame.Pix...)
		sourceBefore := append([]byte(nil), source.Pix...)
		before := append([]byte(nil), out.Pix...)
		mode, n := h.apply(out, frame, true)
		if mode != "hd" || n != 2 {
			t.Fatal(mode, n)
		}
		for y := 0; y < 400*scale; y++ {
			for x := 0; x < 640*scale; x++ {
				if y < 250*scale {
					i := out.PixOffset(x, y)
					if !bytes.Equal(out.Pix[i:i+4], before[i:i+4]) {
						t.Fatal("outside changed", x, y)
					}
				}
			}
		}
		for _, p := range []image.Point{{90, 280}, {575, 370}} {
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					if out.NRGBAAt(p.X*scale+dx, p.Y*scale+dy) != ink {
						t.Fatal("foreground changed")
					}
				}
			}
		}
		if !bytes.Equal(frame.Pix, frameBefore) || !bytes.Equal(source.Pix, sourceBefore) {
			t.Fatal("input changed")
		}
	}
}

func TestHDScrollRejectsMismatchAndMissing(t *testing.T) {
	frame, source, out, h := scrollFixture(t, 2)
	before := append([]byte(nil), out.Pix...)
	frame.SetNRGBA(32, 280, color.NRGBA{1, 2, 3, 255})
	if mode, _ := h.apply(out, frame, true); mode != "fallback_source_mismatch" || !bytes.Equal(out.Pix, before) {
		t.Fatal(mode)
	}
	if mode, _ := h.apply(out, frame, false); mode != "original" || !bytes.Equal(out.Pix, before) {
		t.Fatal(mode)
	}
	if mode, _ := newHDScroll(source, nil, 2).apply(out, frame, true); mode != "fallback_missing_asset" {
		t.Fatal(mode)
	}
	for _, c := range []struct {
		source, asset image.Image
		scale         int
		want          string
	}{
		{nil, source, 2, "fallback_source_shape"}, {source, source, 0, "fallback_scale"},
		{source, image.NewNRGBA(image.Rect(0, 0, 640, 400)), 2, "fallback_hd_aspect"},
		{source, image.NewNRGBA(image.Rect(0, 0, 640, 150)), 2, "fallback_hd_transparency"},
		{source, image.NewNRGBA(image.Rect(0, 0, 600, 150)), 2, "fallback_hd_shape"},
	} {
		if got := newHDScroll(c.source, c.asset, c.scale).Reason; got != c.want {
			t.Fatal(got, c.want)
		}
	}
}

func TestHDScrollDrawLifecycle(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	ret := uint32(0x1e9b0)
	binary.LittleEndian.PutUint32(m[esp:], ret)
	before := append([]byte(nil), m...)
	h := hdLive{LastName: "VISUAL.OZM", ScrollValidated: true}
	h.observe(1, 0x1c084, esp, m)
	if h.ScrollActive || h.ScrollPending == nil {
		t.Fatal("must wait for complete copy")
	}
	h.observe(2, ret, esp, m)
	if h.ScrollActive {
		t.Fatal("wrong stack")
	}
	h.observe(3, ret, esp+4, m)
	if !h.ScrollActive || h.ScrollPending != nil {
		t.Fatal("return did not activate")
	}
	h.observe(4, 0x1bf44, esp, m)
	if !h.ScrollActive || h.ScrollValidated {
		t.Fatal("portrait load must retain drawn background only")
	}
	h.LastName = "A-0.OZM"
	h.observe(5, 0x1c084, esp, m)
	if h.ScrollActive || h.ScrollPending != nil {
		t.Fatal("other full screen copy must invalidate")
	}
	if !bytes.Equal(m, before) {
		t.Fatal("guest changed")
	}
	h = hdLive{LastName: "VISUAL.OZM", ScrollValidated: true}
	h.observe(6, 0x1c084, ^uint32(0)-1, m)
	if h.ScrollPending != nil {
		t.Fatal("stack bounds")
	}
}
