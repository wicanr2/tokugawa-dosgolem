package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

func openingEffectProfile(t *testing.T, name string) hdGraphicsProfile {
	t.Helper()
	for _, profile := range hdGraphicsProfiles {
		if profile.Source.Name == name {
			return profile
		}
	}
	t.Fatal("missing effect source", name)
	return hdGraphicsProfile{}
}

func TestGraphicsEffectsActualNativePaletteSamples(t *testing.T) {
	root := "/project/workplace/hd-prototype/catalog/remaining/history/"
	for _, folder := range []string{"palette", "palette-late"} {
		path := root + folder + "/verification-opening-" + folder + ".json"
		var proof struct {
			Verified bool
			Stages   map[string]struct {
				Palette [16][3]byte `json:"active_palette"`
			}
		}
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &proof) != nil || !proof.Verified {
			t.Fatal("native palette oracle unavailable", path, err)
		}
		for name, sample := range proof.Stages {
			profile := openingEffectProfile(t, strings.Split(name, "@")[0])
			s := nativeSnapshot{MaxColor: 15}
			for i, c := range sample.Palette {
				s.DAC[i] = byte(i)
				s.Palette[i] = c
			}
			effect := graphicsPaletteEffect(profile, s)
			if !effect.Valid || (folder == "palette" && effect.Kind != "clamp") || (folder == "palette-late" && effect.Kind != "cycle") {
				t.Fatal("actual native palette rejected", name, effect)
			}
			if folder == "palette-late" {
				for i, c := range graphicsNativePalette(profile) {
					got := effect.colour(c)
					if [3]byte{got.R, got.G, got.B} != sample.Palette[i] || got.A != c.A {
						t.Fatal("native palette-colour transfer mismatch", name, i)
					}
				}
			}
		}
	}
}

func TestGraphicsEffectsAllClampLevelsAndUnknownPalettes(t *testing.T) {
	for _, name := range []string{"OP4.OZM", "OP-FULL.OZM"} {
		profile := openingEffectProfile(t, name)
		base := graphicsNativePalette(profile)
		var maximum byte
		for _, c := range base {
			maximum = max(maximum, c.R, c.G, c.B)
		}
		for level := 0; level < 64; level++ {
			limit := byte(level<<2 | level>>4)
			s := nativeSnapshot{MaxColor: 15}
			for i, c := range base {
				s.DAC[i] = byte(i)
				s.Palette[i] = [3]byte{min(c.R, limit), min(c.G, limit), min(c.B, limit)}
			}
			e := graphicsPaletteEffect(profile, s)
			want := byte(uint16(min(limit, maximum)) * 255 / uint16(maximum))
			c := color.NRGBA{255, 133, 7, 42}
			if !e.Valid || e.Kind != "clamp" || e.HDLimit != want || e.colour(c) != (color.NRGBA{min(c.R, want), min(c.G, want), min(c.B, want), 42}) {
				t.Fatal("clamp level, HD range or alpha", name, level, e)
			}
			s.Palette[0][0] = 1
			if graphicsPaletteEffect(profile, s).Valid {
				t.Fatal("unknown palette accepted", name, level)
			}
		}
	}
	profile := openingEffectProfile(t, "WEATHER1.OZM")
	if graphicsPaletteEffect(profile, nativeSnapshot{}).Valid {
		t.Fatal("effect family widened to other sources")
	}
}

func TestGraphicsEffectsAllCycleSteps(t *testing.T) {
	var oracle struct {
		Records map[string]struct {
			Target [16][3]byte `json:"target_palette_dac6"`
		}
	}
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/remaining/history/palette-effects.json")
	if err != nil || json.Unmarshal(data, &oracle) != nil {
		t.Fatal("cycle target oracle unavailable", err)
	}
	target := oracle.Records["OP-FULL.OZM"].Target
	profile := openingEffectProfile(t, "OP-FULL.OZM")
	for steps := 0; steps <= 21; steps++ {
		s := nativeSnapshot{MaxColor: 15}
		for i, rgb := range target {
			s.DAC[i] = byte(i)
			for channel, value := range rgb {
				for n := 0; n < steps; n++ {
					if value != 21 {
						value++
					}
				}
				value &= 63
				s.Palette[i][channel] = value<<2 | value>>4
			}
		}
		e := graphicsPaletteEffect(profile, s)
		if !e.Valid || e.Kind != "cycle" || e.CycleSteps != steps {
			t.Fatal("cycle step", steps, e)
		}
		for i, c := range graphicsNativePalette(profile) {
			got := e.colour(c)
			if [3]byte{got.R, got.G, got.B} != s.Palette[i] {
				t.Fatal("cycle colour", steps, i)
			}
		}
	}
}

func TestGraphicsEffectIsAppliedToActualHDCanvas(t *testing.T) {
	profile := openingEffectProfile(t, "OP-FULL.OZM")
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/OP-FULL.OZM.idx")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
	if err != nil {
		t.Fatal(err)
	}
	pair := newHDGraphicsPair(profile, idx, asset, 2)
	region := image.Rect(400, 180, 408, 188)
	indices := graphicsCrop(idx, 640, region)
	draw := graphicsDrawRecord{Name: profile.Source.Name, SourceRect: region, DestRect: region,
		ExpectedSHA: fmt.Sprintf("%x", sha256.Sum256(indices)), Verified: true}
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: idx, MaxColor: 15}
	for i, c := range graphicsNativePalette(profile) {
		s.DAC[i] = byte(i)
		s.Palette[i] = [3]byte{min(c.R, 69), min(c.G, 69), min(c.B, 69)}
	}
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	report := pair.apply(out, s, draw, true)
	if report.Mode != "hd" || report.Effect == nil || report.Effect.Kind != "clamp" || report.Effect.HDLimit != 70 {
		t.Fatal("effect not wired into rendering", report)
	}
	c := rgbaAt(pair.Asset, 800, 360)
	if out.NRGBAAt(800, 360) != (color.NRGBA{min(c.R, 70), min(c.G, 70), min(c.B, 70), c.A}) || out.NRGBAAt(799, 360) != (color.NRGBA{}) {
		t.Fatal("actual canvas effect or region protection")
	}
}
