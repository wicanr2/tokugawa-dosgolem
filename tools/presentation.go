package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
)

type nativeSnapshot struct {
	Image    *image.NRGBA
	Cursor   mouseCursor
	X, Y     int
	Indexed  []byte
	Palette  [256][3]byte
	DAC      [16]byte
	MaxColor byte
}

type desktopContext struct {
	Program       string
	Mark          *markProbeTrace
	MarkQualified func(nativeSnapshot) bool
	Advance       func(int) bool
	Steps         func() int
	Snapshot      func() (nativeSnapshot, error)
	Mouse         func(int, int, int)
	Proof         func() map[string]any
	Live          *hdLive
	Scripted      bool
	PendingScript func() bool
	Composite     *compositeProbe
	Audio         desktopPCMOutput
	Report        map[string]any
}

var runDesktop func(*desktopContext) error
var runDesktopPrelude func() (map[string]any, bool, error)

type desktopPCMOutput interface {
	io.WriteCloser
	SetPaused(bool)
	Report() map[string]any
}

var newDesktopPCMOutput func() (desktopPCMOutput, error)

func snapshotNative(v *bankVideo, cursor mouseCursor, x, y int) (nativeSnapshot, error) {
	s := nativeSnapshot{Cursor: cursor, X: x, Y: y, MaxColor: 15}
	if v.Legacy == nil {
		return s, nil
	}
	w, h := v.Width, v.Height
	if v.Mode != 0 {
		if w < 1 || h < 1 || w*h > len(v.Pixels) {
			return s, fmt.Errorf("invalid VBE display")
		}
		s.Indexed = append([]byte(nil), v.Pixels[:w*h]...)
		s.MaxColor = 255
		for i := range s.DAC {
			s.DAC[i] = byte(i)
		}
	} else {
		w, h = v.Legacy.PlanarSize()
		if w == 0 || h == 0 {
			return s, nil
		}
		s.Indexed = append([]byte(nil), v.Legacy.VGA.Pixels(w, h)...)
		for i := range s.DAC {
			s.DAC[i] = v.Legacy.VGA.DACIndex(byte(i))
		}
	}
	if len(s.Indexed) != w*h {
		return s, fmt.Errorf("invalid native pixel count")
	}
	s.Palette = v.Legacy.Palette()
	s.Image = image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, p := range s.Indexed {
		if v.Mode == 0 {
			p = s.DAC[p&15]
		}
		rgb := s.Palette[p]
		s.Image.SetNRGBA(i%w, i/w, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
	}
	return s, nil
}

// 最後合成原版游標。透明遮罩保留 HD；XOR 採原生底圖的色號結果。
func compositeDesktopCursor(out *image.NRGBA, s nativeSnapshot, scale int) {
	m := s.Cursor
	if !m.Defined || m.DisplayCount < 0 || s.Image == nil {
		return
	}
	w, h := s.Image.Bounds().Dx(), s.Image.Bounds().Dy()
	for row := 0; row < 16; row++ {
		for col := 0; col < 16; col++ {
			x, y := s.X-m.HotX+col, s.Y-m.HotY+row
			if x < 0 || y < 0 || x >= w || y >= h {
				continue
			}
			bit := uint16(0x8000) >> col
			if m.Mask[row]&bit != 0 && m.Mask[row+16]&bit == 0 {
				continue
			}
			p := s.Indexed[y*w+x]
			if m.Mask[row]&bit == 0 {
				p = 0
			}
			if m.Mask[row+16]&bit != 0 {
				p ^= s.MaxColor
			}
			if s.MaxColor == 15 {
				p = s.DAC[p&15]
			}
			v := s.Palette[p]
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					out.SetNRGBA(x*scale+dx, y*scale+dy, color.NRGBA{v[0], v[1], v[2], 255})
				}
			}
		}
	}
}
