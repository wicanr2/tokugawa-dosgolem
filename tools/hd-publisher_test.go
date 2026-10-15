package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func publisherFixture(t *testing.T) (nativeSnapshot, *markProbeTrace, []byte) {
	t.Helper()
	base := "/project/workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8/"
	indices, err := os.ReadFile(base + "mark-source.idx")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(indices)) != publisherIndexedSHA {
		t.Fatal("unverified native publisher source")
	}
	data, err := os.ReadFile(base + "startup-probe/receipt-75m.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Mark struct {
			Events []struct {
				Kind    string       `json:"kind"`
				Palette [256][3]byte `json:"palette"`
			} `json:"events"`
		} `json:"mark_observation"`
	}
	if err = json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	s := nativeSnapshot{Indexed: append([]byte(nil), indices...), MaxColor: 255, Image: image.NewNRGBA(image.Rect(0, 0, 640, 480))}
	for _, e := range receipt.Mark.Events {
		if e.Kind == "fade_in_return" {
			s.Palette = e.Palette
		}
	}
	mark := &markProbeTrace{Source: append([]byte(nil), indices...), SourcePointer: 16, TargetPointer: 307232, DrawReady: true, TargetReady: true}
	for i := range mark.Target {
		mark.Target[i] = s.Palette[i/3][i%3] >> 2
	}
	if fmt.Sprintf("%x", sha256.Sum256(mark.Target[:])) != publisherTargetSHA {
		t.Fatal("unverified runtime target palette")
	}
	mem := make([]byte, 308000)
	copy(mem[mark.SourcePointer:], indices)
	copy(mem[mark.TargetPointer:], mark.Target[:])
	for i, index := range s.Indexed {
		rgb := s.Palette[index]
		s.Image.SetNRGBA(i%640, i/640, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
	}
	return s, mark, mem
}

func TestPublisherHDUsesOriginalDrawAndPreservesCopyright(t *testing.T) {
	s, mark, mem := publisherFixture(t)
	if !mark.qualified(s, mem) {
		t.Fatal("actual publisher source did not qualify")
	}
	data, err := os.ReadFile("/project/assets/hd-b/art/RAINEND-hd-b-v1.png")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != publisherAssetSHA {
		t.Fatal("unregistered HD publisher asset")
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	p := newHDPublisher(im)
	before := sha256.Sum256(mem)
	original, mode := p.render(s, mark, true, false)
	if mode != "original" {
		t.Fatal(mode)
	}
	hd, mode := p.render(s, mark, true, true)
	if mode != "hd" || bytes.Equal(hd.Pix, original.Pix) {
		t.Fatal("publisher was not replaced")
	}
	for y := 898; y < 960; y++ {
		for x := 0; x < 1280; x++ {
			if hd.NRGBAAt(x, y) != original.NRGBAAt(x, y) {
				t.Fatal("original copyright changed")
			}
		}
	}
	if before != sha256.Sum256(mem) {
		t.Fatal("presentation wrote original memory")
	}
	for _, qualified := range []bool{false, true} {
		pair := p
		if qualified {
			pair = newHDPublisher(nil)
		}
		fallback, reason := pair.render(s, mark, qualified, true)
		if reason == "hd" || !bytes.Equal(fallback.Pix, original.Pix) {
			t.Fatal("invalid/missing publisher did not fall back")
		}
	}
	mem[mark.SourcePointer] ^= 1
	if mark.qualified(s, mem) {
		t.Fatal("changed source guard qualified")
	}
	mem[mark.SourcePointer] ^= 1
	mem[mark.TargetPointer] ^= 1
	if mark.qualified(s, mem) {
		t.Fatal("changed palette guard qualified")
	}
	mem[mark.TargetPointer] ^= 1
	s.Indexed[0] ^= 1
	if mark.qualified(s, mem) {
		t.Fatal("changed destination qualified")
	}
}

func TestPublisherPaletteFadesAndUnknownPalette(t *testing.T) {
	var target [768]byte
	for i := range target {
		target[i] = byte(i % 64)
	}
	for _, subtract := range []bool{false, true} {
		for _, level := range []byte{1, 31} {
			var palette [256][3]byte
			for i, value := range target {
				if subtract {
					if value > level {
						value -= level
					} else {
						value = 0
					}
				} else {
					value = min(value, level)
				}
				palette[i/3][i%3] = value<<2 | value>>4
			}
			kind, value, ok := publisherPaletteEffect(target, palette)
			want := "clamp"
			if subtract {
				want = "subtract"
			}
			if !ok || kind != want || value != level<<2|level>>4 {
				t.Fatal("recognized fade rejected")
			}
		}
	}
	var unknown [256][3]byte
	unknown[0][0] = 255
	if _, _, ok := publisherPaletteEffect(target, unknown); ok {
		t.Fatal("unknown palette accepted")
	}
}

func TestPublisherViewportKeepsToolbarGeometry(t *testing.T) {
	for _, visible := range []bool{false, true} {
		y := 959
		if visible {
			y += barHeight * 2
		}
		p, inside := barGamePointFor(1279, y, visible, 480)
		if !inside || p != image.Pt(639, 479) {
			t.Fatal("publisher viewport lost its bottom edge")
		}
		if _, inside = barGamePointFor(1279, y+1, visible, 480); inside {
			t.Fatal("toolbar coordinates entered outside publisher")
		}
	}
	if barHeightDeltaFor(1280, 960, false, 480) != 64 || barHeightDeltaFor(1280, 1024, true, 480) != 64 {
		t.Fatal("publisher toolbar changes game scale")
	}
}
