package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestFramePreservesStoryScrollAndOriginal(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 640, 250))
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	asset := image.NewNRGBA(image.Rect(0, 0, 1280, 500))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			c := color.NRGBA{uint8(x), uint8(y), 15, 255}
			native.SetNRGBA(x, y, c)
			if y < 250 {
				source.SetNRGBA(x, y, c)
			}
		}
	}
	for y := 0; y < 500; y++ {
		for x := 0; x < 1280; x++ {
			asset.SetNRGBA(x, y, color.NRGBA{190, 40, 20, 255})
		}
	}
	h := newHDFrame(source, asset, 2)
	if h.Reason != "ready" {
		t.Fatal(h.Reason)
	}
	out, _, err := previewA0(native, nil, nil, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	// 已經畫好的 HD 劇情圖也不能被框架覆蓋。
	out.SetNRGBA(400, 120, color.NRGBA{90, 20, 150, 255})
	before := append([]byte(nil), out.Pix...)
	original := append([]byte(nil), native.Pix...)
	if mode := h.apply(out, native, true); mode != "hd" {
		t.Fatal(mode)
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			i := out.PixOffset(x, y)
			if y >= 500 || image.Pt(x/2, y/2).In(a0Rect) {
				if !bytes.Equal(out.Pix[i:i+4], before[i:i+4]) {
					t.Fatalf("changed independent layer at %d,%d", x, y)
				}
			} else if out.NRGBAAt(x, y) != (color.NRGBA{190, 40, 20, 255}) {
				t.Fatalf("missing frame pixel at %d,%d", x, y)
			}
		}
	}
	if !bytes.Equal(original, native.Pix) {
		t.Fatal("native frame changed")
	}
	// 框架上新增一個前景像素，必須完整回原圖。
	native.SetNRGBA(40, 50, color.NRGBA{255, 255, 255, 255})
	copy(out.Pix, before)
	if mode := h.apply(out, native, true); mode != "fallback_source_mismatch" || !bytes.Equal(out.Pix, before) {
		t.Fatal("occluded frame did not fall back without painting", mode)
	}
	if got := newHDFrame(source, nil, 2).apply(out, native, true); got != "fallback_missing_asset" {
		t.Fatal(got)
	}
	asset.SetNRGBA(0, 0, color.NRGBA{})
	if got := newHDFrame(source, asset, 2); got.Reason != "fallback_hd_invalid" || got.Asset != nil {
		t.Fatal("transparent HD frame was accepted")
	}
}
