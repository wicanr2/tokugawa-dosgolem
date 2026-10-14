package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

func battleFrameFixture(t *testing.T) ([]byte, image.Image, nativeSnapshot) {
	t.Helper()
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/hd-assets/MAPGAMEN.OZM-" + battleFrameIndexedSHA + ".idx")
	if err != nil {
		t.Skip("local source unavailable")
	}
	path := "/project/workplace/hd-prototype/catalog/battle-frame/" + battleFrameAsset
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("local prepared frame unavailable")
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != battleFrameAssetSHA {
		t.Fatal("prepared frame fingerprint")
	}
	asset, err := previewPNG(path)
	if err != nil {
		t.Fatal(err)
	}
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: append([]byte(nil), idx...), MaxColor: 15}
	for i, c := range battleFrameNativePalette {
		s.DAC[i] = byte(i)
		s.Palette[i] = [3]byte{c.R, c.G, c.B}
	}
	for n, index := range idx {
		s.Image.SetNRGBA(n%640, n/640, battleFrameNativePalette[index])
	}
	return idx, asset, s
}

func TestBattleFrameActualGeometryWholeRegionGatesAndPalette(t *testing.T) {
	idx, asset, s := battleFrameFixture(t)
	beforeIndex := sha256.Sum256(idx)
	beforeNative := sha256.Sum256(s.Image.Pix)
	p := newHDBattleFrame(idx, asset, 2)
	if p.Reason != "ready" {
		t.Fatal(p.Reason)
	}
	count := 0
	for _, v := range p.Mask {
		count += int(v)
	}
	if count != 100853 {
		t.Fatal("source frame mask", count)
	}
	p.update(s, true, true)
	out, _, _ := previewA0(s.Image, nil, nil, 2, false)
	report := p.paint(out, s, hdTerrainState{}, nil)
	if report.Mode != "hd" || !report.CrestHD || !report.WeatherPlaceholderHD || report.PaintedNative != 165853 {
		t.Fatal("intro frame and whole-region gates", report)
	}
	if beforeIndex != sha256.Sum256(idx) || beforeNative != sha256.Sum256(s.Image.Pix) {
		t.Fatal("frame changed input")
	}
	crestPoint, weatherPoint, textPoint := image.Pt(230, 190), image.Pt(500, 320), image.Pt(500, 250)
	for _, point := range []image.Point{crestPoint, weatherPoint, textPoint} {
		n := point.Y*640 + point.X
		s.Indexed[n] = (s.Indexed[n] + 1) % 16
		s.Image.SetNRGBA(point.X, point.Y, battleFrameNativePalette[s.Indexed[n]])
	}
	p.update(s, true, true)
	out, _, _ = previewA0(s.Image, nil, nil, 2, false)
	report = p.paint(out, s, hdTerrainState{}, nil)
	if report.CrestHD || report.WeatherPlaceholderHD {
		t.Fatal("partial source match accepted whole crest/weather", report)
	}
	for _, rect := range []image.Rectangle{battleFrameCrestRect, battleFrameWeatherRect} {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				if out.NRGBAAt(x*2, y*2) != s.Image.NRGBAAt(x, y) {
					t.Fatal("dynamic region overwritten", x, y)
				}
			}
		}
	}
	if out.NRGBAAt(textPoint.X*2, textPoint.Y*2) != s.Image.NRGBAAt(textPoint.X, textPoint.Y) {
		t.Fatal("original text index overwritten")
	}
	for _, enabled := range []bool{false, true} {
		changed := s
		changed.Palette = s.Palette
		changed.Palette[2][0]++
		p.update(changed, true, enabled)
		out, _, _ = previewA0(s.Image, nil, nil, 2, false)
		before := append([]byte(nil), out.Pix...)
		if p.paint(out, changed, hdTerrainState{}, nil).Mode == "hd" || !bytes.Equal(before, out.Pix) {
			t.Fatal("disabled or unknown fade palette changed output")
		}
	}
	if newHDBattleFrame(idx, nil, 2).Reason != "fallback_missing_asset" || newHDBattleFrame(nil, asset, 2).Reason != "fallback_source" {
		t.Fatal("source/missing frame fallback")
	}
}

func TestBattleFrameProtectsSameIndexOriginalSpriteAndHDScene(t *testing.T) {
	idx, asset, s := battleFrameFixture(t)
	p := newHDBattleFrame(idx, asset, 2)
	p.update(s, true, true)
	n := -1
	for i, index := range idx {
		if index != 0 && p.Mask[i] != 0 && p.Asset.NRGBAAt(i%640*2, i/640*2) != battleFrameNativePalette[index] {
			n = i
			break
		}
	}
	if n < 0 {
		t.Fatal("same-index opaque source fixture unavailable")
	}
	state := hdTerrainState{History: []hdTerrainDraw{{Index: 0}}}
	scene := hdTerrainScene{Pixels: make([]terrainScenePixel, 640*400), ModelValid: true}
	scene.Pixels[n].VisibleSprite = true
	out, _, _ := previewA0(s.Image, nil, nil, 2, false)
	p.paint(out, s, state, &scene)
	if out.NRGBAAt(n%640*2, n/640*2) != battleFrameNativePalette[idx[n]] {
		t.Fatal("same-index original foreground overwritten")
	}
	scene.Pixels[n].VisibleSprite = false
	scene.Pixels[n].Affected = true
	paint := color.NRGBA{101, 202, 93, 255}
	out.SetNRGBA(n%640*2, n/640*2, paint)
	p.paint(out, s, state, &scene)
	if out.NRGBAAt(n%640*2, n/640*2) != paint {
		t.Fatal("qualified HD scene overwritten")
	}
	if c, ok := p.background(n%640, n/640, 0, 0, idx[n]); !ok || c != p.Asset.NRGBAAt(n%640*2, n/640*2) {
		t.Fatal("verified indexed frame underlay")
	}
	if _, ok := p.background(n%640, n/640, 0, 0, (idx[n]+1)%16); ok {
		t.Fatal("unknown indexed background accepted")
	}
	if p.paint(out, s, state, nil).Mode != "fallback_scene_model" {
		t.Fatal("missing scene model accepted")
	}
}
