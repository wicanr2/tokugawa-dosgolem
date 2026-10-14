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

func TestTreeBackgroundRestoreVisibleUnitsAndSafety(t *testing.T) {
	grass, ink, blue, painted := color.NRGBA{44, 215, 44, 255}, color.NRGBA{32, 32, 32, 255}, color.NRGBA{20, 20, 227, 255}, color.NRGBA{80, 100, 50, 255}
	floor := image.NewNRGBA(image.Rect(0, 0, 48, 32))
	source := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	hd := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			floor.SetNRGBA(x, y, grass)
		}
	}
	for y := 0; y < 47; y++ {
		for x := 0; x < 47; x++ {
			source.SetNRGBA(x, y, ink)
		}
	}
	source.SetNRGBA(8, 8, color.NRGBA{})
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			hd.SetNRGBA(x, y, painted)
		}
	}
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			hd.SetNRGBA(20+dx, 20+dy, color.NRGBA{})
		}
	}
	beforeUnit := image.NewNRGBA(source.Bounds())
	beforeUnit.SetNRGBA(8, 8, blue)
	afterUnit := image.NewNRGBA(source.Bounds())
	afterUnit.SetNRGBA(9, 9, blue)
	sources := map[hdTerrainSpriteKey]*image.NRGBA{{"objects", 0}: source, {"units", 1}: beforeUnit, {"units", 3}: afterUnit}
	target := hdTerrainSpriteDraw{Group: "objects", Index: 0, Rect: image.Rect(80, 80, 128, 128), Order: 4}
	state := hdTerrainState{Draws: []hdTerrainDraw{{Index: 0, Rect: image.Rect(80, 80, 128, 112), Order: 1}, {Index: 0, Rect: image.Rect(80, 112, 128, 144), Order: 2}}, Sprites: []hdTerrainSpriteDraw{{Group: "units", Index: 1, Rect: target.Rect, Order: 3}, target, {Group: "units", Index: 3, Rect: target.Rect, Order: 5}}}
	floors := map[int]hdTerrainPair{0: newHDTerrainPair(floor, nil, 2)}
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 80; y < 128; y++ {
		for x := 80; x < 128; x++ {
			c := grass
			for _, sprite := range state.Sprites {
				if v := sources[hdTerrainSpriteKey{sprite.Group, sprite.Index}].NRGBAAt(x-80, y-80); v.A == 255 {
					c = v
				}
			}
			native.SetNRGBA(x, y, c)
		}
	}
	pair := hdTerrainTreePair{Source: source, Asset: hd, Reason: "ready"}
	out, _, _ := previewA0(native, nil, nil, 2, false)
	original := append([]byte(nil), out.Pix...)
	before := append([]byte(nil), native.Pix...)
	reason, protected := pair.apply(out, native, target, state, floors, []string{"original", "original"}, sources, nil, 2, true)
	if reason != "hd" || protected != 2 || out.NRGBAAt(176, 176) != blue || out.NRGBAAt(178, 178) != blue || out.NRGBAAt(180, 180) != grass || out.NRGBAAt(182, 182) != painted || !bytes.Equal(before, native.Pix) {
		t.Fatal("visible units/background restore/HD", reason, protected)
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			if x >= 160 && x < 256 && y >= 160 && y < 256 {
				continue
			}
			i := (y*1280 + x) * 4
			if !bytes.Equal(out.Pix[i:i+4], original[i:i+4]) {
				t.Fatal("outside tree changed")
			}
		}
	}
	check := func(want string, p hdTerrainTreePair, s hdTerrainState, m map[hdTerrainSpriteKey]*image.NRGBA, replaced []image.Rectangle) {
		t.Helper()
		out, _, _ := previewA0(native, nil, nil, 2, false)
		before := append([]byte(nil), out.Pix...)
		got, _ := p.apply(out, native, target, s, floors, []string{"original", "original"}, m, replaced, 2, true)
		if got != want || !bytes.Equal(before, out.Pix) {
			t.Fatal("unsafe fallback", want, got)
		}
	}
	check("fallback_replaced_overlap", pair, state, sources, []image.Rectangle{target.Rect})
	missing := pair
	missing.Asset = nil
	missing.Reason = "fallback_missing_asset"
	check("fallback_missing_asset", missing, state, sources, nil)
	pending := state
	pending.Pending = &hdTerrainPending{}
	check("fallback_pending_draw", pair, pending, sources, nil)
	incomplete := state
	incomplete.Draws = append([]hdTerrainDraw(nil), state.Draws...)
	incomplete.Draws[1].Rect = incomplete.Draws[1].Rect.Add(image.Pt(48, 0))
	check("fallback_background_incomplete", pair, incomplete, sources, nil)
	check("fallback_foreground_source", pair, state, nil, nil)
	native.SetNRGBA(80, 80, blue)
	check("fallback_scene_mismatch", pair, state, sources, nil)
	for y := 80; y < 128; y++ {
		for x := 80; x < 128; x++ {
			native.SetNRGBA(x, y, grass)
		}
	}
	overwritten := state
	overwritten.Draws = append([]hdTerrainDraw(nil), state.Draws...)
	overwritten.Draws[0].Order = 6
	overwritten.Draws[1].Order = 7
	check("fallback_overwritten_sprite", pair, overwritten, sources, nil)
	if blendTerrainAlpha(color.NRGBA{0, 0, 255, 255}, color.NRGBA{255, 0, 0, 128}) != (color.NRGBA{128, 0, 127, 255}) {
		t.Fatal("straight-alpha contract")
	}
}

func TestTreeActualArtGeometryAndThreeNormalScenes(t *testing.T) {
	root := "/project/workplace/hd-prototype/catalog/terrain/"
	data, err := os.ReadFile(root + "objects/desktop/desktop-terrain.json")
	if err != nil {
		t.Skip("normal original scene unavailable")
	}
	var receipt struct {
		Desktop struct {
			Observation struct {
				Terrain hdTerrainState `json:"terrain"`
			} `json:"hd_observation"`
		} `json:"desktop"`
	}
	if json.Unmarshal(data, &receipt) != nil {
		t.Fatal("original receipt")
	}
	state := receipt.Desktop.Observation.Terrain
	sources := map[hdTerrainSpriteKey]*image.NRGBA{}
	for _, p := range hdTerrainSpriteProfiles {
		im, e := previewPNG(fmt.Sprintf("%sobjects/sources/%s-%03d-original.png", root, p.Group, p.Index))
		if e != nil {
			t.Fatal(e)
		}
		sources[hdTerrainSpriteKey{p.Group, p.Index}] = newHDTerrainSpriteSource(im)
	}
	source := sources[hdTerrainSpriteKey{"objects", 0}]
	assetPath := "/project/assets/hd-b/terrain/" + hdTerrainTreeAsset
	data, err = os.ReadFile(assetPath)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != hdTerrainTreeAssetSHA {
		t.Fatal("selected tree fingerprint")
	}
	asset, err := previewPNG(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	pair := newHDTerrainTreePair(source, asset, 2)
	if pair.Reason != "ready" {
		t.Fatal(pair.Reason)
	}
	prepared, err := previewPNG(root + "trees/tree-prepared-v2.png")
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			if pair.Asset.NRGBAAt(x, y) != rgbaAt(prepared, x, y) {
				t.Fatal("prepared alpha geometry", x, y)
			}
		}
	}
	if newHDTerrainTreePair(source, nil, 2).Reason != "fallback_missing_asset" || newHDTerrainTreePair(nil, asset, 2).Reason != "fallback_source_shape" {
		t.Fatal("source/asset fallback")
	}
	bad := image.NewNRGBA(image.Rect(0, 0, 1254, 1254))
	bad.SetNRGBA(0, 1253, color.NRGBA{1, 1, 1, 255})
	if newHDTerrainTreePair(source, bad, 2).Reason != "fallback_unapproved_crop" {
		t.Fatal("unreviewed alpha outside crop")
	}
	floors := map[int]hdTerrainPair{}
	for index := range hdTerrainProfiles {
		im, e := previewPNG(fmt.Sprintf("%sprepared/sources/floor-%03d-original.png", root, index))
		if e != nil {
			t.Fatal(e)
		}
		floors[index] = newHDTerrainPair(im, nil, 2)
	}
	nativeImage, err := previewPNG("/project/workplace/startup-probe/frame-b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846.png")
	if err != nil {
		t.Fatal(err)
	}
	native := image.NewNRGBA(nativeImage.Bounds())
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			native.SetNRGBA(x, y, rgbaAt(nativeImage, x, y))
		}
	}
	// 來源合成 fixture 去除最後游標；其原始游標重建已由獨立 Python 證據核對。
	for _, floor := range state.Draws {
		for y := floor.Rect.Min.Y; y < floor.Rect.Max.Y; y++ {
			for x := floor.Rect.Min.X; x < floor.Rect.Max.X; x++ {
				c := floors[floor.Index].Source.NRGBAAt(x-floor.Rect.Min.X, y-floor.Rect.Min.Y)
				for _, s := range state.Sprites {
					if s.Order > floor.Order && image.Pt(x, y).In(s.Rect) {
						if v := sources[hdTerrainSpriteKey{s.Group, s.Index}].NRGBAAt(x-s.Rect.Min.X, y-s.Rect.Min.Y); v.A == 255 {
							c = v
						}
					}
				}
				native.SetNRGBA(x, y, c)
			}
		}
	}
	out, _, _ := previewA0(native, nil, nil, 2, false)
	before := append([]byte(nil), native.Pix...)
	modes := make([]string, len(state.Draws))
	for i := range modes {
		modes[i] = "original"
	}
	var replaced []image.Rectangle
	for _, s := range state.Sprites {
		reason, _ := pair.apply(out, native, s, state, floors, modes, sources, replaced, 2, true)
		if reason == "hd" {
			replaced = append(replaced, s.Rect)
		}
	}
	if len(replaced) != 3 || !bytes.Equal(before, native.Pix) {
		t.Fatal("normal tree scenes/native immutability", len(replaced))
	}
}
