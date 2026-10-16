package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

func TestEndingFinalActualOriginalPalette(t *testing.T) {
	root := "/project/workplace/hd-prototype/catalog/remaining/consumers/battle/r25/ready/"
	data, err := os.ReadFile(root + "native/receipt.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Stages []struct {
			Label       string             `json:"label"`
			IndicesPath string             `json:"indices_path"`
			Palette     [256][3]byte       `json:"palette"`
			DAC         [16]byte           `json:"DAC"`
			Draw        graphicsDrawRecord `json:"last_graphics_return"`
		} `json:"ending_validation_stages"`
	}
	if json.Unmarshal(data, &receipt) != nil {
		t.Fatal("original ending palette evidence")
	}
	seen := map[string]bool{}
	for _, stage := range receipt.Stages {
		name := stage.Draw.Name
		if (name != "END-11.OZM" && name != "END-11B.OZM") || seen[name] {
			continue
		}
		seen[name] = true
		var profile hdGraphicsProfile
		for _, p := range hdGraphicsProfiles {
			if p.Source.Name == name {
				profile = p
			}
		}
		indices, err := os.ReadFile(root + "native/" + stage.IndicesPath)
		if err != nil || len(indices) != 256000 {
			t.Fatal("original native indices", err)
		}
		s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: indices, Palette: stage.Palette, DAC: stage.DAC, MaxColor: 15}
		for i, index := range indices {
			c := s.Palette[s.DAC[index]]
			s.Image.SetNRGBA(i%640, i/640, color.NRGBA{c[0], c[1], c[2], 255})
		}
		source, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + name + ".idx")
		if err != nil {
			t.Fatal(err)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, source, asset, 2)
		base, reason := prepareHDImage(s.Image, 640, 400, 2)
		if reason != "" {
			t.Fatal(reason)
		}
		out := base.(*image.NRGBA)
		before := append([]byte(nil), out.Pix...)
		if !stage.Draw.Verified || pair.apply(out, s, stage.Draw, true).Mode != "hd" {
			t.Fatal("actual ending palette rejected", name)
		}
		if !bytes.Equal(out.Pix[798*out.Stride:], before[798*out.Stride:]) {
			t.Fatal("unknown ending row replaced")
		}
		for _, unknown := range []bool{false, true} {
			bad := s
			if unknown {
				bad.Palette[bad.DAC[1]][0] ^= 1
			} else {
				for i, c := range battleFrameNativePalette {
					bad.Palette[bad.DAC[i]] = [3]byte{c.R, c.G, c.B}
				}
			}
			copy(out.Pix, before)
			if pair.apply(out, bad, stage.Draw, true).Mode != "fallback_palette" || !bytes.Equal(out.Pix, before) {
				t.Fatal("ending accepted unrelated or unknown palette", name)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatal("both actual ending sources required")
	}
}

func remainingGraphicsSnapshot(profile hdGraphicsProfile, source []byte, rect image.Rectangle) (nativeSnapshot, graphicsDrawRecord) {
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), MaxColor: 15}
	palette := graphicsNativePalette(profile)
	for i, c := range palette {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
	}
	expected := graphicsCrop(source, profile.Source.Width, rect)
	for y := 0; y < rect.Dy(); y++ {
		copy(s.Indexed[y*640:], expected[y*rect.Dx():(y+1)*rect.Dx()])
		for x := 0; x < rect.Dx(); x++ {
			s.Image.SetNRGBA(x, y, palette[expected[y*rect.Dx()+x]])
		}
	}
	return s, graphicsDrawRecord{Name: profile.Source.Name, SourceRect: rect,
		DestRect: image.Rect(0, 0, rect.Dx(), rect.Dy()), EndStep: 10,
		ExpectedSHA: fmt.Sprintf("%x", sha256.Sum256(expected)), Verified: true}
}

func TestRemainingGraphicsAllSourceFramesAndFixedForeground(t *testing.T) {
	if len(hdGraphicsRemainingProfiles) != 20 {
		t.Fatal("twenty reviewed sources required")
	}
	frames, sky, duel, eyes, protected := 0, 0, 0, 0, 0
	for _, profile := range hdGraphicsRemainingProfiles {
		source, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + profile.Source.Name + ".idx")
		if err != nil {
			t.Skip("local original indexed sources unavailable")
		}
		path := "/project/assets/hd-b/art/" + profile.Asset
		encoded, err := os.ReadFile(path)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(encoded)) != profile.AssetSHA {
			t.Fatal("reviewed asset fingerprint", profile.Source.Name, err)
		}
		asset, err := previewPNG(path)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, source, asset, 2)
		if pair.Reason != "ready" {
			t.Fatal(profile.Source.Name, pair.Reason)
		}
		cases := []image.Rectangle{image.Rect(0, 0, profile.Source.Width, profile.Source.Height)}
		if len(profile.Source.Name) >= 5 && profile.Source.Name[:5] == "ANIME" {
			for i := 0; i < 6; i++ {
				cases = append(cases, image.Rect(i%3*208, i/3*150, (i%3+1)*208, (i/3+1)*150))
				frames++
			}
		}
		if profile.Source.Name == "END-2B.OZM" {
			for i := 0; i < 8; i++ {
				cases = append(cases, image.Rect(i%2*256, i/2*84, (i%2+1)*256, (i/2+1)*84))
				sky++
			}
		}
		if profile.Source.Name == "OP2_P.OZM" {
			for i := 0; i < 3; i++ {
				cases = append(cases, image.Rect(i*128, 0, (i+1)*128, 119))
				duel++
			}
		}
		if profile.Source.Name == "OP9.OZM" {
			for i := 0; i < 4; i++ {
				cases = append(cases, image.Rect(416, 200+i*27, 488, 227+i*27))
				eyes++
			}
		}
		beforeSource := sha256.Sum256(source)
		for _, rect := range cases {
			s, draw := remainingGraphicsSnapshot(profile, source, rect)
			beforeNative := sha256.Sum256(s.Indexed)
			out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
			for i := 0; i < len(out.Pix); i += 4 {
				copy(out.Pix[i:i+4], []byte{17, 19, 23, 255})
			}
			report := pair.apply(out, s, draw, true)
			if report.Mode != "hd" {
				t.Fatal(profile.Source.Name, rect, report)
			}
			painted, kept := 0, 0
			paints := profile.PaintRects
			if len(paints) == 0 {
				paints = []image.Rectangle{profile.PaintRect}
			}
			for y := 0; y < rect.Dy(); y++ {
				for x := 0; x < rect.Dx(); x++ {
					sx, sy := x+rect.Min.X, y+rect.Min.Y
					paint, preserve := false, false
					for _, area := range paints {
						if image.Pt(sx, sy).In(area) {
							paint = true
						}
					}
					index := source[sy*profile.Source.Width+sx]
					for _, region := range profile.Preserve {
						if image.Pt(sx, sy).In(region.Rect) && region.Indices&(uint16(1)<<index) != 0 {
							preserve = true
						}
					}
					if paint {
						if preserve {
							kept++
						} else {
							painted++
						}
					}
					for dy := 0; dy < 2; dy++ {
						for dx := 0; dx < 2; dx++ {
							want := color.NRGBA{17, 19, 23, 255}
							if paint {
								want = rgbaAt(pair.Asset, sx*2+dx, sy*2+dy)
								if preserve {
									want = graphicsNativePalette(profile)[index]
								}
							}
							if out.NRGBAAt(x*2+dx, y*2+dy) != want {
								t.Fatal("source crop, fixed glyph or reserved buffer", profile.Source.Name, rect, x, y)
							}
						}
					}
				}
			}
			if report.PaintedNative != painted || report.PreservedNative != kept || sha256.Sum256(s.Indexed) != beforeNative || sha256.Sum256(source) != beforeSource {
				t.Fatal("counts or original source mutation", profile.Source.Name, report, painted, kept)
			}
			protected += kept
			if rect.Dy() < 400 && out.NRGBAAt(0, rect.Dy()*2) != (color.NRGBA{17, 19, 23, 255}) {
				t.Fatal("outside destination")
			}
		}
		s, draw := remainingGraphicsSnapshot(profile, source, cases[0])
		out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		before := sha256.Sum256(out.Pix)
		if newHDGraphicsPair(profile, source, nil, 2).apply(out, s, draw, true).Mode != "fallback_missing_asset" || sha256.Sum256(out.Pix) != before {
			t.Fatal("per-source missing asset changed output")
		}
		s.Palette[s.DAC[2]][0]++
		if pair.apply(out, s, draw, true).Mode != "fallback_palette" || sha256.Sum256(out.Pix) != before {
			t.Fatal("unknown palette changed output")
		}
	}
	if frames != 48 || sky != 8 || duel != 3 || eyes != 4 || protected == 0 {
		t.Fatal(frames, sky, duel, eyes, protected)
	}
}

func TestGraphicsFixedForegroundDoesNotAcceptUnknownWrites(t *testing.T) {
	profile := hdGraphicsProfile{Source: graphicsSourceProfile{Name: "fixed.OZM", Width: 32, Height: 16}, Preserve: []hdGraphicsPreserve{{Rect: image.Rect(0, 0, 8, 8), Indices: 1}}}
	source := make([]byte, 32*16)
	profile.IndicesSHA = fmt.Sprintf("%x", sha256.Sum256(source))
	asset := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for i := 0; i < len(asset.Pix); i += 4 {
		copy(asset.Pix[i:i+4], []byte{77, 88, 99, 255})
	}
	pair := newHDGraphicsPair(profile, source, asset, 2)
	s, draw := remainingGraphicsSnapshot(profile, source, image.Rect(0, 0, 32, 16))
	s.Indexed[0] = 1
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	before := append([]byte(nil), out.Pix...)
	if pair.apply(out, s, draw, true).Mode != "fallback_source_mismatch" || !bytes.Equal(out.Pix, before) {
		t.Fatal("fixed foreground incorrectly bypassed destination validation")
	}
	for _, bad := range []hdGraphicsProfile{
		{Source: profile.Source, IndicesSHA: profile.IndicesSHA, PaintRect: image.Rect(0, 0, 8, 8), PaintRects: []image.Rectangle{image.Rect(0, 0, 8, 8)}},
		{Source: profile.Source, IndicesSHA: profile.IndicesSHA, PaintRects: []image.Rectangle{image.Rect(0, 0, 8, 8), image.Rect(4, 4, 12, 12)}},
		{Source: profile.Source, IndicesSHA: profile.IndicesSHA, PaintRects: []image.Rectangle{image.Rect(0, 0, 33, 16)}},
		{Source: profile.Source, IndicesSHA: profile.IndicesSHA, Preserve: []hdGraphicsPreserve{{Rect: image.Rect(0, 0, 8, 8), Indices: 0}}},
	} {
		if newHDGraphicsPair(bad, source, asset, 2).Reason == "ready" {
			t.Fatal("malformed profile accepted")
		}
	}
}

func TestGraphicsLaterWindowAndPortraitOcclusion(t *testing.T) {
	profile := hdGraphicsProfile{Source: graphicsSourceProfile{Name: "background.OZM", Width: 64, Height: 48}}
	source := make([]byte, 64*48)
	profile.IndicesSHA = fmt.Sprintf("%x", sha256.Sum256(source))
	asset := image.NewNRGBA(image.Rect(0, 0, 128, 96))
	for i := 0; i < len(asset.Pix); i += 4 {
		copy(asset.Pix[i:i+4], []byte{55, 66, 77, 255})
	}
	pair := newHDGraphicsPair(profile, source, asset, 2)
	s, draw := remainingGraphicsSnapshot(profile, source, image.Rect(0, 0, 64, 48))
	window, portrait := image.Rect(8, 8, 24, 24), image.Rect(32, 16, 48, 32)
	clipped := graphicsOcclude(draw, []hdWindowDraw{{Rect: window, EndStep: 20}}, []hdKaoDraw{{Rect: portrait, EndStep: 30}})
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	marker := color.NRGBA{111, 99, 88, 255}
	for _, r := range []image.Rectangle{window, portrait} {
		for y := r.Min.Y * 2; y < r.Max.Y*2; y++ {
			for x := r.Min.X * 2; x < r.Max.X*2; x++ {
				out.SetNRGBA(x, y, marker)
			}
		}
	}
	report := pair.apply(out, s, clipped, true)
	if report.Mode != "hd" || report.PaintedNative != 64*48-2*16*16 {
		t.Fatal(report)
	}
	for y := 0; y < 48*2; y++ {
		for x := 0; x < 64*2; x++ {
			want := color.NRGBA{55, 66, 77, 255}
			if image.Pt(x/2, y/2).In(window) || image.Pt(x/2, y/2).In(portrait) {
				want = marker
			}
			if out.NRGBAAt(x, y) != want {
				t.Fatal("same-index later foreground overwritten", x, y)
			}
		}
	}
	if draw.VisibleRects != nil || draw.SourceRect != clipped.SourceRect || draw.DestRect != clipped.DestRect || draw.ExpectedSHA != clipped.ExpectedSHA {
		t.Fatal("original evidence changed")
	}
	older := graphicsOcclude(draw, []hdWindowDraw{{Rect: window, EndStep: 5}}, []hdKaoDraw{{Rect: portrait, EndStep: 8}})
	if r := pair.apply(out, s, older, true); r.PaintedNative != 64*48 {
		t.Fatal("older foreground blocked newer background", r)
	}
	bad := graphicsOcclude(draw, []hdWindowDraw{{Rect: image.Rect(600, 380, 700, 430), EndStep: 20}}, nil)
	if pair.apply(out, s, bad, true).Mode != "fallback_no_visible_art" {
		t.Fatal("invalid occlusion accepted")
	}
	large := draw
	large.VisibleRects = []image.Rectangle{}
	for y := 0; y < 256; y++ {
		large.VisibleRects = append(large.VisibleRects, image.Rect(0, y, 32, y+1))
	}
	if len(graphicsOcclude(large, []hdWindowDraw{{Rect: image.Rect(8, 0, 16, 256), EndStep: 20}}, nil).VisibleRects) != 0 {
		t.Fatal("occlusion fragment cap")
	}
}
