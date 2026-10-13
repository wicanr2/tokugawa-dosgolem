package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestDesktopCursorTransparencyAndOrder(t *testing.T) {
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), X: 160, Y: 50, MaxColor: 15}
	s.Cursor = mouseCursor{Defined: true, DisplayCount: 0}
	for i := 0; i < 16; i++ {
		s.Cursor.Mask[i] = 0xffff
		s.DAC[i] = byte(i)
	}
	s.Palette[0] = [3]byte{10, 20, 30}
	s.Palette[15] = [3]byte{200, 210, 220}
	s.Cursor.Mask[0] &^= 0x8000 // 黑色點。
	s.Cursor.Mask[16] = 0x4000  // 原版 XOR 點。
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			out.SetNRGBA(x, y, color.NRGBA{70, 80, 90, 255})
		}
	}
	original := append([]byte(nil), s.Indexed...)
	compositeDesktopCursor(out, s, 2)
	if out.NRGBAAt(320, 100) != (color.NRGBA{10, 20, 30, 255}) || out.NRGBAAt(322, 100) != (color.NRGBA{200, 210, 220, 255}) {
		t.Fatal("cursor did not draw above HD")
	}
	if out.NRGBAAt(324, 100) != (color.NRGBA{70, 80, 90, 255}) {
		t.Fatal("transparent cursor erased HD")
	}
	if !bytes.Equal(s.Indexed, original) {
		t.Fatal("native indices mutated")
	}
	s.Cursor.HotX = 16
	s.Cursor.HotY = 16
	s.X = 0
	s.Y = 0
	compositeDesktopCursor(out, s, 2) // 完全在畫面外，不能越界。
}
