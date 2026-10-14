package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestStoryMapLabelsKeepOriginalPixels(t *testing.T) {
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			native.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	before := append([]byte(nil), native.Pix...)
	for name, regions := range portraitLabelRects {
		out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		preservePortraitLabels(out, native, name, 2)
		if out.NRGBAAt(320, 100).A != 0 {
			t.Fatal("preservation touched non-label art")
		}
		for _, rect := range regions {
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				for x := rect.Min.X; x < rect.Max.X; x++ {
					want := native.NRGBAAt(160+x, 50+y)
					for dy := 0; dy < 2; dy++ {
						for dx := 0; dx < 2; dx++ {
							if out.NRGBAAt((160+x)*2+dx, (50+y)*2+dy) != want {
								t.Fatal("original map label changed")
							}
						}
					}
				}
			}
		}
	}
	if !bytes.Equal(before, native.Pix) {
		t.Fatal("native image changed")
	}
}

func TestPreviewA0ScopeAndFallback(t *testing.T) {
	frame := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	source := image.NewNRGBA(image.Rect(0, 0, 320, 150))
	hd := image.NewNRGBA(image.Rect(0, 0, 640, 300))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	for y := 0; y < 150; y++ {
		for x := 0; x < 320; x++ {
			source.SetNRGBA(x, y, frame.NRGBAAt(160+x, 50+y))
		}
	}
	for y := 0; y < 300; y++ {
		for x := 0; x < 640; x++ {
			hd.SetNRGBA(x, y, color.NRGBA{9, 8, 7, 255})
		}
	}
	originalFrame := append([]byte(nil), frame.Pix...)
	originalSource := append([]byte(nil), source.Pix...)
	originalHD := append([]byte(nil), hd.Pix...)
	out, mode, err := previewA0(frame, source, hd, 2, true)
	if err != nil || mode != "hd" {
		t.Fatalf("%s %v", mode, err)
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			want := frame.NRGBAAt(x/2, y/2)
			if image.Pt(x/2, y/2).In(a0Rect) {
				want = color.NRGBA{9, 8, 7, 255}
			}
			if out.NRGBAAt(x, y) != want {
				t.Fatalf("unexpected pixel %d,%d", x, y)
			}
		}
	}
	if !bytes.Equal(frame.Pix, originalFrame) || !bytes.Equal(source.Pix, originalSource) || !bytes.Equal(hd.Pix, originalHD) {
		t.Fatal("input mutated")
	}
	for _, tc := range []struct {
		name    string
		enabled bool
		asset   image.Image
		want    string
	}{
		{"off", false, hd, "original"}, {"missing", true, nil, "fallback_missing_asset"},
		{"wrong_aspect", true, image.NewNRGBA(image.Rect(0, 0, 640, 400)), "fallback_hd_aspect"},
		{"transparent", true, image.NewNRGBA(image.Rect(0, 0, 640, 300)), "fallback_hd_transparency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, mode, err := previewA0(frame, source, tc.asset, 1, tc.enabled)
			if err != nil || mode != tc.want || !bytes.Equal(got.Pix, frame.Pix) {
				t.Fatalf("fallback failed: %s %v", mode, err)
			}
		})
	}
	source.SetNRGBA(319, 149, color.NRGBA{1, 2, 3, 255})
	got, mode, err := previewA0(frame, source, hd, 1, true)
	if err != nil || mode != "fallback_source_mismatch" || !bytes.Equal(got.Pix, frame.Pix) {
		t.Fatal("mismatched source was replaced")
	}
}

func TestPreviewA0Bounds(t *testing.T) {
	for _, scale := range []int{0, 5} {
		if _, _, err := previewA0(image.NewNRGBA(image.Rect(0, 0, 640, 400)), nil, nil, scale, false); err == nil {
			t.Fatal("bad scale accepted")
		}
	}
	if _, _, err := previewA0(image.NewNRGBA(image.Rect(0, 0, 320, 200)), nil, nil, 1, false); err == nil {
		t.Fatal("bad frame accepted")
	}
}
