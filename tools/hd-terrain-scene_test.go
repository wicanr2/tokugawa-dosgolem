package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"testing"
)

func TestFloorJournalRetainsVisibleFragmentsAndBound(t *testing.T) {
	mem := terrainMemory(t)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x200)
	before := sha256.Sum256(mem)
	var state hdTerrainState
	state.observe(1, 0x1fc07, 0x1000, 0, 80*100+10, mem)
	state.observe(2, 0x200, 0x1004, 0, 0, mem)
	state.observe(3, 0x1fc07, 0x1000, 6, 80*100+13, mem)
	state.observe(4, 0x200, 0x1004, 0, 0, mem)
	if len(state.Draws) != 1 || len(state.History) != 2 || state.History[0].Order != 1 || state.History[1].Order != 2 || state.History[0].Rect.Intersect(state.History[1].Rect).Dx() != 24 || before != sha256.Sum256(mem) {
		t.Fatal("partial floor journal or read-only state")
	}
	state.History = make([]hdTerrainDraw, 512)
	state.observe(5, 0x1fc07, 0x1000, 0, 80*100+10, mem)
	state.observe(6, 0x200, 0x1004, 0, 0, mem)
	if state.Pending != nil || len(state.History) != 0 || len(state.Draws) != 0 {
		t.Fatal("history bound must invalidate eligibility")
	}
}

func TestSceneOverlappingHDVisibleOwnerAlphaAndPartialFloors(t *testing.T) {
	grass, blue, purple, yellow, ui, ink, green, red := color.NRGBA{30, 180, 40, 255}, color.NRGBA{20, 40, 200, 255}, color.NRGBA{120, 30, 140, 255}, color.NRGBA{180, 170, 30, 255}, color.NRGBA{210, 120, 20, 255}, color.NRGBA{10, 10, 10, 255}, color.NRGBA{40, 120, 50, 255}, color.NRGBA{220, 20, 30, 128}
	solid := func(w, h int, c color.NRGBA) *image.NRGBA {
		im := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(im, im.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
		return im
	}
	unitSource := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	treeSource := image.NewNRGBA(unitSource.Bounds())
	for _, x := range []int{8, 10, 14, 30} {
		unitSource.SetNRGBA(x, 20, ink)
	}
	for _, x := range []int{10, 12, 30} {
		treeSource.SetNRGBA(x, 20, ink)
	}
	unitArt := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	treeArt := image.NewNRGBA(unitArt.Bounds())
	paint := func(im *image.NRGBA, x, y int, c color.NRGBA) {
		for dy := 0; dy < 2; dy++ {
			for dx := 0; dx < 2; dx++ {
				im.SetNRGBA(x*2+dx, y*2+dy, c)
			}
		}
	}
	paint(unitArt, 8, 20, red)
	paint(unitArt, 10, 20, blue)
	paint(unitArt, 30, 20, red)
	paint(treeArt, 8, 20, green)
	paint(treeArt, 14, 20, green)
	paint(treeArt, 30, 20, green)
	paint(treeArt, 0, 0, green)
	bg := solid(48, 48, ui)
	for y := 16; y < 48; y++ {
		for x := 0; x < 48; x++ {
			bg.SetNRGBA(x, y, grass)
		}
	}
	withBackground := func(s hdTerrainSpriteDraw, source, before *image.NRGBA) hdTerrainSpriteDraw {
		s.Background = before
		s.BackgroundVerified = true
		s.BeginStep = 1
		s.EndStep = 2
		s.BackgroundSHA = fmt.Sprintf("%x", sha256.Sum256(before.Pix))
		s.SourceSHA = fmt.Sprintf("%x", sha256.Sum256(source.Pix))
		return s
	}
	unit := withBackground(hdTerrainSpriteDraw{Group: "units", Index: 1, Rect: image.Rect(80, 64, 128, 112), Order: 2}, unitSource, bg)
	treeBG := solid(48, 48, ui)
	copy(treeBG.Pix, bg.Pix)
	draw.Draw(treeBG, treeBG.Bounds(), unitSource, image.Point{}, draw.Over)
	tree := withBackground(hdTerrainSpriteDraw{Group: "objects", Index: 0, Rect: unit.Rect, Order: 3}, treeSource, treeBG)
	state := hdTerrainState{History: []hdTerrainDraw{{Index: 0, Rect: image.Rect(80, 80, 128, 112), Order: 1}, {Index: 1, Rect: image.Rect(104, 80, 152, 112), Order: 5}}, Sprites: []hdTerrainSpriteDraw{unit, tree}}
	floors := map[int]hdTerrainPair{0: newHDTerrainPair(solid(48, 32, grass), solid(96, 64, purple), 2), 1: newHDTerrainPair(solid(48, 32, blue), solid(96, 64, yellow), 2)}
	sources := map[hdTerrainSpriteKey]*image.NRGBA{{"units", 1}: unitSource, {"objects", 0}: treeSource}
	pairs := map[hdTerrainSpriteKey]hdTerrainSpritePair{{"units", 1}: {Source: unitSource, Asset: unitArt, Reason: "ready", Key: hdTerrainSpriteKey{"units", 1}}, {"objects", 0}: {Source: treeSource, Asset: treeArt, Reason: "ready", Key: hdTerrainSpriteKey{"objects", 0}}}
	native := solid(640, 400, ui)
	draw.Draw(native, state.History[0].Rect, floors[0].Source, image.Point{}, draw.Src)
	draw.Draw(native, unit.Rect, unitSource, image.Point{}, draw.Over)
	draw.Draw(native, tree.Rect, treeSource, image.Point{}, draw.Over)
	draw.Draw(native, state.History[1].Rect, floors[1].Source, image.Point{}, draw.Src)
	nativeBefore := append([]byte(nil), native.Pix...)
	unitBefore := append([]byte(nil), unitSource.Pix...)
	out, _, _ := previewA0(native, nil, nil, 2, false)
	original := append([]byte(nil), out.Pix...)
	var scene hdTerrainScene
	report := scene.apply(out, native, state, floors, pairs, sources, 2, true)
	if report.FloorHD != 2 || report.FloorNativePixels != 2304 || report.UnitHD != 1 || report.TreeHD != 1 {
		t.Fatal("visible fragments or overlapping HD sprites", report)
	}
	if out.NRGBAAt(176, 168) != blendTerrainAlpha(green, red) {
		t.Fatal("earlier visible unit must blend once above expanded later tree")
	}
	if out.NRGBAAt(180, 168) != blue || out.NRGBAAt(184, 168) != purple || out.NRGBAAt(188, 168) != green {
		t.Fatal("transparent outlines did not reveal correct HD layers")
	}
	if out.NRGBAAt(220, 168) != yellow || out.NRGBAAt(160, 128) != ui {
		t.Fatal("later floor or original transparent UI overwritten")
	}
	if !bytes.Equal(nativeBefore, native.Pix) || !bytes.Equal(unitBefore, unitSource.Pix) {
		t.Fatal("renderer changed original input")
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			if image.Pt(x, y).In(image.Rect(160, 128, 304, 224)) {
				continue
			}
			n := (y*1280 + x) * 4
			if !bytes.Equal(out.Pix[n:n+4], original[n:n+4]) {
				t.Fatal("outside scene changed")
			}
		}
	}
	missing := map[hdTerrainSpriteKey]hdTerrainSpritePair{}
	for k, p := range pairs {
		p.Asset = nil
		p.Reason = "fallback_missing_asset"
		missing[k] = p
	}
	missingFloors := map[int]hdTerrainPair{}
	for k, p := range floors {
		p.Asset = nil
		p.Reason = "fallback_missing_asset"
		missingFloors[k] = p
	}
	for _, enabled := range []bool{false, true} {
		out, _, _ = previewA0(native, nil, nil, 2, false)
		before := append([]byte(nil), out.Pix...)
		r := scene.apply(out, native, state, missingFloors, missing, sources, 2, enabled)
		if r.FloorHD+r.UnitHD+r.TreeHD != 0 || !bytes.Equal(before, out.Pix) {
			t.Fatal("missing/disabled changed presentation")
		}
	}
	unitMissing := map[hdTerrainSpriteKey]hdTerrainSpritePair{}
	for k, p := range pairs {
		unitMissing[k] = p
	}
	p := unitMissing[hdTerrainSpriteKey{"units", 1}]
	p.Asset = nil
	p.Reason = "fallback_missing_asset"
	unitMissing[hdTerrainSpriteKey{"units", 1}] = p
	out, _, _ = previewA0(native, nil, nil, 2, false)
	report = scene.apply(out, native, state, floors, unitMissing, sources, 2, true)
	if report.UnitHD != 0 || report.TreeHD != 1 || report.ProtectedNativePixels != 2 || out.NRGBAAt(176, 168) != ink || out.NRGBAAt(188, 168) != ink {
		t.Fatal("missing unit must preserve original visible owner", report)
	}
	native.SetNRGBA(81, 81, ink)
	out, _, _ = previewA0(native, nil, nil, 2, false)
	report = scene.apply(out, native, state, floors, pairs, sources, 2, true)
	if report.FloorHD != 1 || report.TreeHD+report.UnitHD != 0 || out.NRGBAAt(162, 162) != ink {
		t.Fatal("unknown UI overwrite accepted", report)
	}
	state.Pending = &hdTerrainPending{}
	out, _, _ = previewA0(native, nil, nil, 2, false)
	before := append([]byte(nil), out.Pix...)
	if scene.apply(out, native, state, floors, pairs, sources, 2, true).Reason != "fallback_pending_draw" || !bytes.Equal(before, out.Pix) {
		t.Fatal("pending draw changed scene")
	}
}

func TestSceneApprovedUnitArtCropAndSourceGeometry(t *testing.T) {
	for index, profile := range hdTerrainUnitProfiles {
		path := "/project/assets/hd-b/terrain/" + profile.Asset
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skip("local approved art unavailable")
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != profile.AssetSHA {
			t.Fatal("unit asset fingerprint")
		}
		asset, err := previewPNG(path)
		if err != nil {
			t.Fatal(err)
		}
		source, err := previewPNG(fmt.Sprintf("/project/workplace/hd-prototype/catalog/terrain/objects/sources/units-%03d-original.png", index))
		if err != nil {
			t.Fatal(err)
		}
		p := newHDTerrainSpritePair(source, asset, 2, hdTerrainSpriteKey{"units", index}, profile.SourceBounds, profile.Crop)
		if p.Reason != "ready" {
			t.Fatal(index, p.Reason)
		}
		visible := 0
		for y := 0; y < 96; y++ {
			for x := 0; x < 96; x++ {
				c := p.Asset.NRGBAAt(x, y)
				if !image.Pt(x/2, y/2).In(profile.SourceBounds) && c.A != 0 {
					t.Fatal("prepared art escaped original bounds", index, x, y)
				}
				if c.A != 0 {
					visible++
				}
			}
		}
		if visible == 0 {
			t.Fatal("empty prepared unit")
		}
		if newHDTerrainSpritePair(source, nil, 2, hdTerrainSpriteKey{"units", index}, profile.SourceBounds, profile.Crop).Reason != "fallback_missing_asset" {
			t.Fatal("unit missing fallback")
		}
	}
}
