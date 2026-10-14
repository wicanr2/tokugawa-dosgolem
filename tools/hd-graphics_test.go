package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

func TestHDGraphicsDrawLifetimeAndUnknownOverlap(t *testing.T) {
	mem, p, idx := graphicsFixture()
	originalProfiles := hdGraphicsSources
	hdGraphicsSources = []graphicsSourceProfile{p}
	defer func() { hdGraphicsSources = originalProfiles }()
	frame := make([]byte, 640*400)
	for y := 0; y < 74; y++ {
		copy(frame[(291+y)*640+464:], idx[y*144:(y+1)*144])
	}
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame}, nil }
	var state hdGraphicsState
	state.observe(1, 0x1c1bc, 0x60000, mem, snapshot)
	state.observe(2, 0x12345, 0x60004, mem, snapshot)
	if len(state.Draws) != 1 || state.Completed != 1 {
		t.Fatal("natural source draw not retained")
	}
	// 未知的非交疊矩形不撤銷已顯示天氣；未知來源的交疊複製撤銷。
	mem[0x100000] ^= 1
	binary.LittleEndian.PutUint32(mem[0x60014:], 0xa0000)
	state.observe(3, 0x1c1bc, 0x60000, mem, snapshot)
	if len(state.Draws) != 1 || state.Pending != nil {
		t.Fatal("unrelated unknown copy altered weather lifetime")
	}
	binary.LittleEndian.PutUint32(mem[0x60014:], 0xa5b2a)
	state.observe(4, 0x1c1bc, 0x60000, mem, snapshot)
	if len(state.Draws) != 0 {
		t.Fatal("overlapping unknown copy kept weather active")
	}
	mem[0x100000] ^= 1
	state.observe(5, 0x1c1bc, 0x60000, mem, snapshot)
	state.observe(6, 0x12345, 0x60004, mem, snapshot)
	state.observe(7, 0x1c084, 0x60000, mem, snapshot)
	if len(state.Draws) != 0 {
		t.Fatal("full copy kept obsolete weather")
	}
}

func TestHDGraphicsWeatherWholeRegionPaletteAndCrop(t *testing.T) {
	for _, profile := range hdGraphicsProfiles {
		idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + profile.Source.Name + ".idx")
		if err != nil {
			t.Skip("local decoded weather sources unavailable")
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, idx, asset, 2)
		if pair.Reason != "ready" {
			t.Fatal(pair.Reason)
		}
		s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), MaxColor: 15}
		for i, c := range battleFrameNativePalette {
			s.DAC[i] = byte(i)
			s.Palette[i] = [3]byte{c.R, c.G, c.B}
		}
		// 用非零來源 offset 驗證共同裁切，不只驗證全圖。
		src, dst := image.Rect(8, 3, 136, 70), image.Rect(464, 291, 592, 358)
		expected := graphicsCrop(idx, 144, src)
		for y := 0; y < dst.Dy(); y++ {
			copy(s.Indexed[(dst.Min.Y+y)*640+dst.Min.X:], expected[y*dst.Dx():(y+1)*dst.Dx()])
		}
		draw := graphicsDrawRecord{Name: profile.Source.Name, SourceRect: src, DestRect: dst, ExpectedSHA: fmt.Sprintf("%x", sha256.Sum256(expected)), Verified: true}
		out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		if r := pair.apply(out, s, draw, true); r.Mode != "hd" || r.PaintedNative != 128*67 {
			t.Fatal(r)
		}
		if out.NRGBAAt(928, 582) != rgbaAt(pair.Asset, 16, 6) || out.NRGBAAt(927, 582) != (color.NRGBA{}) {
			t.Fatal("crop or outside geometry")
		}
		before := append([]byte(nil), out.Pix...)
		for _, kind := range []string{"disabled", "palette", "index", "unverified"} {
			changed := s
			changed.Indexed = append([]byte(nil), s.Indexed...)
			d := draw
			if kind == "palette" {
				changed.Palette[2][0]++
			}
			if kind == "index" {
				changed.Indexed[291*640+464] ^= 1
			}
			if kind == "unverified" {
				d.Verified = false
			}
			r := pair.apply(out, changed, d, kind != "disabled")
			if r.Mode == "hd" || !bytes.Equal(before, out.Pix) {
				t.Fatal("fallback changed output", kind, r)
			}
		}
		if newHDGraphicsPair(profile, idx, nil, 2).Reason != "fallback_missing_asset" || newHDGraphicsPair(profile, nil, asset, 2).Reason != "fallback_source" {
			t.Fatal("missing/source fallback")
		}
	}
}
