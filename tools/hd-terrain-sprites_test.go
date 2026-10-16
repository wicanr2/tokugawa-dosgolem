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

func TestBattleUnitActualSourcesTransparencyFallbackAndOcclusion(t *testing.T) {
	profile := openingEffectProfile(t, "BATTLE.OZM")
	indexed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/BATTLE.OZM.idx")
	if err != nil {
		t.Fatal(err)
	}
	art, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
	if err != nil {
		t.Fatal(err)
	}
	battle := newHDGraphicsPair(profile, indexed, art, 2)
	background := graphicsDrawRecord{Name: "BATTLE.OZM", Kind: "or_full", Verified: true,
		SourceRect: image.Rect(0, 0, 640, 400), DestRect: image.Rect(0, 0, 640, 400),
		ExpectedSHA: "original composite", ActualSHA: "original composite", SourceCropSHA: profile.IndicesSHA, CompositeSourceSHA: profile.IndicesSHA}
	for index, unitProfile := range hdTerrainUnitProfiles {
		original, err := previewPNG(fmt.Sprintf("/project/workplace/hd-prototype/catalog/terrain/objects/sources/units-%03d-original.png", index))
		if err != nil {
			t.Fatal(err)
		}
		asset, err := previewPNG("/project/assets/hd-b/terrain/" + unitProfile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDTerrainSpritePair(original, asset, 2, hdTerrainSpriteKey{"units", index}, unitProfile.SourceBounds, unitProfile.Crop)
		for _, x := range []int{240, 400} {
			r := image.Rect(x, 60, x+48, 108)
			s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: append([]byte(nil), indexed...), MaxColor: 15}
			palette := graphicsNativePalette(profile)
			for i, c := range palette {
				s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
			}
			for i, value := range indexed {
				s.Image.SetNRGBA(i%640, i/640, palette[value])
			}
			before := cropBackdrop(s.Image, r)
			sourceIndices := make([]byte, 48*48)
			for y := 0; y < 48; y++ {
				for dx := 0; dx < 48; dx++ {
					if c := pair.Source.NRGBAAt(dx, y); c.A == 255 {
						s.Image.SetNRGBA(x+dx, 60+y, c)
						for i := 1; i < 16; i++ {
							if c == palette[i] {
								sourceIndices[y*48+dx] = byte(i)
								s.Indexed[(60+y)*640+x+dx] = byte(i)
							}
						}
					}
				}
			}
			target := hdTerrainSpriteDraw{Group: "units", Index: index, Rect: r, CombatReturn: 0x1f9be, BeginStep: 10, EndStep: 20,
				Background: before, BackgroundVerified: true, BackgroundIndexed: graphicsCrop(indexed, 640, r)}
			target.BackgroundIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(target.BackgroundIndexed))
			target.BackgroundSHA = fmt.Sprintf("%x", sha256.Sum256(before.Pix))
			target.SourceSHA = fmt.Sprintf("%x", sha256.Sum256(pair.Source.Pix))
			target.AfterSHA = fmt.Sprintf("%x", sha256.Sum256(cropBackdrop(s.Image, r).Pix))
			target.SourceIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(sourceIndices))
			target.AfterIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(graphicsCrop(s.Indexed, 640, r)))
			// 原版繪製時的 RGB 可早於淡入；保持索引及保存內容的完整性。
			for i := range before.Pix {
				if i%4 != 3 {
					before.Pix[i] = 0
				}
			}
			target.BackgroundSHA = fmt.Sprintf("%x", sha256.Sum256(before.Pix))
			target.SourceSHA, target.AfterSHA = "earlier palette source", "earlier palette result"
			out, _, _ := previewA0(s.Image, nil, nil, 2, false)
			prior, native := append([]byte(nil), out.Pix...), append([]byte(nil), s.Image.Pix...)
			if mode := pair.applyBattle(out, s, target, battle, background, true, false); mode != "hd" {
				t.Fatal("actual unit rejected", index, x, mode)
			}
			for y := 0; y < 96; y++ {
				for dx := 0; dx < 96; dx++ {
					base := rgbaAt(battle.Asset, x*2+dx, 120+y)
					front := pair.Asset.NRGBAAt(dx, y)
					a := int(front.A)
					channel := func(b, f byte) byte { return byte((int(f)*a + int(b)*(255-a) + 127) / 255) }
					want := color.NRGBA{channel(base.R, front.R), channel(base.G, front.G), channel(base.B, front.B), 255}
					if out.NRGBAAt(x*2+dx, 120+y) != want {
						t.Fatal("independent alpha formula", index, dx, y)
					}
				}
			}
			for i := 0; i < len(out.Pix); i += 4 {
				if !image.Pt(i/4%1280, i/4/1280).In(image.Rect(x*2, 120, (x+48)*2, 216)) && !bytes.Equal(out.Pix[i:i+4], prior[i:i+4]) {
					t.Fatal("outside unit changed")
				}
			}
			if !bytes.Equal(native, s.Image.Pix) {
				t.Fatal("native image changed")
			}
			for _, failure := range []string{"disabled", "occluded", "missing", "background", "return", "source", "palette", "scene"} {
				unit, draw, snap := pair, target, s
				if failure == "missing" {
					unit.Reason = "fallback_missing_asset"
				}
				if failure == "background" {
					draw.BackgroundIndexedSHA = "wrong"
				}
				if failure == "return" {
					draw.CombatReturn = 0x1fea2
				}
				if failure == "source" {
					draw.SourceIndexedSHA = "wrong"
				}
				if failure == "palette" {
					snap.Palette[0][0] ^= 1
				}
				if failure == "scene" {
					copyImage := image.NewNRGBA(s.Image.Bounds())
					copy(copyImage.Pix, s.Image.Pix)
					copyImage.SetNRGBA(x, 60, color.NRGBA{1, 2, 3, 255})
					snap.Image = copyImage
				}
				copy(out.Pix, prior)
				mode := unit.applyBattle(out, snap, draw, battle, background, failure != "disabled", failure == "occluded")
				if mode == "hd" || !bytes.Equal(out.Pix, prior) {
					t.Fatal("unsafe fallback", index, failure, mode)
				}
			}
		}
	}
}

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
	data, err = os.ReadFile(root + "backgrounds/startup-probe/latest.json")
	if err != nil {
		t.Skip("normal pre-draw background evidence unavailable")
	}
	var backgrounds struct {
		Records []struct {
			Draw     hdTerrainSpriteDraw `json:"draw"`
			Begin    int                 `json:"begin_step"`
			EndStep  int                 `json:"end_step"`
			Verified bool                `json:"primitive_verified"`
			Before   struct {
				Path string `json:"path"`
				RGBA string `json:"rgba_sha256"`
			} `json:"before"`
			Source struct {
				SHA string `json:"rgba_sha256"`
			} `json:"source"`
		} `json:"sprite_backdrop_records"`
	}
	if json.Unmarshal(data, &backgrounds) != nil {
		t.Fatal("background receipt")
	}
	for i := range state.Sprites {
		for _, b := range backgrounds.Records {
			if !b.Verified || b.Draw.Group != state.Sprites[i].Group || b.Draw.Index != state.Sprites[i].Index || b.Draw.Rect != state.Sprites[i].Rect {
				continue
			}
			im, e := previewPNG(root + "backgrounds/" + b.Before.Path)
			if e != nil {
				t.Fatal(e)
			}
			bg := image.NewNRGBA(image.Rect(0, 0, 48, 48))
			for y := 0; y < 48; y++ {
				for x := 0; x < 48; x++ {
					bg.SetNRGBA(x, y, rgbaAt(im, x, y))
				}
			}
			s := &state.Sprites[i]
			s.Background, s.BackgroundVerified, s.BackgroundSHA = bg, true, b.Before.RGBA
			s.SourceSHA, s.BeginStep, s.EndStep = b.Source.SHA, b.Begin, b.EndStep
		}
	}
	replaced = nil
	for _, s := range state.Sprites {
		reason, _ := pair.apply(out, native, s, state, floors, modes, sources, replaced, 2, true)
		if reason == "hd" {
			replaced = append(replaced, s.Rect)
		}
	}
	if len(replaced) != 4 || !bytes.Equal(before, native.Pix) {
		t.Fatal("four normal trees with actual captured background", len(replaced))
	}
}

func TestTreeObservedBackgroundPreservesUIAndRestoresOldOutline(t *testing.T) {
	ui, ink, paint := color.NRGBA{200, 140, 20, 255}, color.NRGBA{10, 10, 10, 255}, color.NRGBA{40, 90, 30, 255}
	bg := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	source := image.NewNRGBA(bg.Bounds())
	asset := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			bg.SetNRGBA(x, y, ui)
		}
	}
	source.SetNRGBA(10, 10, ink)
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			asset.SetNRGBA(x, y, paint)
		}
	}
	for y := 20; y < 22; y++ {
		for x := 20; x < 22; x++ {
			asset.SetNRGBA(x, y, color.NRGBA{})
		}
	}
	target := hdTerrainSpriteDraw{Group: "objects", Index: 0, Rect: image.Rect(80, 80, 128, 128), Order: 2, Background: bg, BackgroundVerified: true, BeginStep: 1, EndStep: 3, BackgroundSHA: fmt.Sprintf("%x", sha256.Sum256(bg.Pix)), SourceSHA: fmt.Sprintf("%x", sha256.Sum256(source.Pix))}
	state := hdTerrainState{Sprites: []hdTerrainSpriteDraw{target}}
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			native.SetNRGBA(80+x, 80+y, ui)
		}
	}
	native.SetNRGBA(90, 90, ink)
	pair := hdTerrainTreePair{Source: source, Asset: asset, Reason: "ready"}
	sources := map[hdTerrainSpriteKey]*image.NRGBA{{"objects", 0}: source}
	out, _, _ := previewA0(native, nil, nil, 2, false)
	if reason, _ := pair.apply(out, native, target, state, nil, nil, sources, nil, 2, true); reason != "hd" || out.NRGBAAt(160, 160) != ui || out.NRGBAAt(180, 180) != ui {
		t.Fatal("UI preserved and old opaque source erased", reason)
	}
	native.SetNRGBA(81, 81, ink)
	out, _, _ = previewA0(native, nil, nil, 2, false)
	before := append([]byte(nil), out.Pix...)
	if reason, _ := pair.apply(out, native, target, state, nil, nil, sources, nil, 2, true); reason != "fallback_scene_mismatch" || !bytes.Equal(before, out.Pix) {
		t.Fatal("unobserved UI overwrite accepted", reason)
	}
}
