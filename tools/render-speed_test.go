package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestNativeScaleMatchesOriginalPixelsAndStride(t *testing.T) {
	parent := image.NewNRGBA(image.Rect(0, 0, 647, 405))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			parent.SetNRGBA(x, y, color.NRGBA{byte(x), byte(y), byte(x + y), byte(x*7 + y)})
		}
	}
	source := parent.SubImage(image.Rect(0, 0, 640, 400)).(*image.NRGBA)
	other := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			other.Set(x, y, source.At(x, y))
		}
	}
	for _, im := range []image.Image{source, other} {
		for scale := 1; scale <= 4; scale++ {
			got := scaleNativeFrame(im, scale)
			want := image.NewNRGBA(got.Bounds())
			for y := 0; y < want.Rect.Dy(); y++ {
				for x := 0; x < want.Rect.Dx(); x++ {
					want.SetNRGBA(x, y, color.NRGBAModel.Convert(im.At(x/scale, y/scale)).(color.NRGBA))
				}
			}
			if !bytes.Equal(got.Pix, want.Pix) {
				t.Fatalf("type %T scale %d changed pixels", im, scale)
			}
		}
	}
}

func TestSpeedMenuBoundsFontAndEveryHitRow(t *testing.T) {
	font, err := loadBarFont("/project/tools/toolbar-font.hex.gz")
	if err != nil {
		t.Fatal(err)
	}
	bar := newWindowBar()
	bar.Visible = true
	bar.Menu = 1
	paint := bar.paint(font, [3]int{}, true, 4)
	menu := barMenuRect(1)
	if !menu.In(paint.Bounds()) || menu.Max.Y > 400 {
		t.Fatal(menu, paint.Bounds())
	}
	for row, label := range barLists[1] {
		field, hit := bar.hit(image.Pt(menu.Min.X+5, barHeight+row*barRowHeight+12))
		if field != 1 || hit != row {
			t.Fatal(field, hit, row)
		}
		width := font['✓'].Width + font[' '].Width
		for _, r := range label {
			width += font[r].Width
		}
		if menu.Min.X+8+width > menu.Max.X || barHeight+row*barRowHeight+4+16 > menu.Max.Y {
			t.Fatal("speed label clipped", row, label)
		}
	}
}
