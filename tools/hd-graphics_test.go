package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

func TestHDGraphicsLogoUsesOnlyObservedSteadyPalette(t *testing.T) {
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/graphics/static/palette-probe/palette-captures.json")
	if err != nil {
		t.Skip("local original palette captures unavailable")
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "cbe61f72fc33ce5a3e2ed5873ac5d06fa9fdba96d9688c478ce579c1f1f73936" {
		t.Fatal("original palette evidence changed")
	}
	var events []struct {
		Step  int
		Frame struct {
			Native struct {
				DAC     []byte
				Palette [][3]byte
			} `json:"native_indices"`
		}
	}
	if err := json.Unmarshal(data, &events); err != nil || len(events) != 7 {
		t.Fatal("seven original captures required", err)
	}
	profile := hdGraphicsLogoProfile
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/OP1.OZM.idx")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
	if err != nil {
		t.Fatal(err)
	}
	pair := newHDGraphicsPair(profile, idx, asset, 2)
	draw := graphicsDrawRecord{Name: profile.Source.Name, SourceRect: image.Rect(0, 0, 640, 400), DestRect: image.Rect(0, 0, 640, 400), ExpectedSHA: profile.IndicesSHA, Verified: true}
	for i, event := range events {
		s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: idx, MaxColor: 15}
		if len(event.Frame.Native.DAC) != 16 || len(event.Frame.Native.Palette) != 256 {
			t.Fatal("invalid original palette", event.Step)
		}
		copy(s.DAC[:], event.Frame.Native.DAC)
		copy(s.Palette[:], event.Frame.Native.Palette)
		out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		before := append([]byte(nil), out.Pix...)
		r := pair.apply(out, s, draw, true)
		steady := i == 4 || i == 5
		if steady {
			if r.Mode != "hd" || r.PaintedNative != 234*57 {
				t.Fatal("steady original publisher palette rejected", event.Step, r)
			}
			other := profile
			other.Source.Name = "unregistered-palette.OZM"
			d := draw
			d.Name = other.Source.Name
			if newHDGraphicsPair(other, idx, asset, 2).apply(out, s, d, true).Mode != "fallback_palette" {
				t.Fatal("publisher palette accepted for another source")
			}
		} else if r.Mode != "fallback_palette" || !bytes.Equal(before, out.Pix) {
			t.Fatal("fade palette changed output", event.Step, r)
		}
	}
}

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

func TestHDGraphicsAllApprovedSourceRegionsPaletteAndCrop(t *testing.T) {
	if len(hdGraphicsProfiles) != 29 {
		t.Fatal("eight existing and twenty reviewed source profiles required")
	}
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
		for i, c := range graphicsNativePalette(profile) {
			s.DAC[i] = byte(i)
			s.Palette[i] = [3]byte{c.R, c.G, c.B}
		}
		// 用非零來源 offset 驗證共同裁切，不只驗證全圖。
		region := profile.PaintRect
		if region.Empty() {
			region = image.Rect(0, 0, profile.Source.Width, profile.Source.Height)
		}
		sx := (region.Min.X+7)/8*8 + 8
		sy := region.Min.Y + 3
		width, height := min(128, region.Max.X-sx), min(67, region.Max.Y-sy-3)
		src := image.Rect(sx, sy, sx+width, sy+height)
		dst := image.Rect(464, 291, 464+width, 291+height)
		expected := graphicsCrop(idx, profile.Source.Width, src)
		for y := 0; y < dst.Dy(); y++ {
			copy(s.Indexed[(dst.Min.Y+y)*640+dst.Min.X:], expected[y*dst.Dx():(y+1)*dst.Dx()])
		}
		draw := graphicsDrawRecord{Name: profile.Source.Name, SourceRect: src, DestRect: dst, ExpectedSHA: fmt.Sprintf("%x", sha256.Sum256(expected)), Verified: true}
		out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		if r := pair.apply(out, s, draw, true); r.Mode != "hd" || r.PaintedNative != width*height {
			t.Fatal(r)
		}
		if out.NRGBAAt(928, 582) != rgbaAt(pair.Asset, src.Min.X*2, src.Min.Y*2) || out.NRGBAAt(927, 582) != (color.NRGBA{}) {
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

func TestHDGraphicsVisibleFragmentsAndSameIndexUnknownCopy(t *testing.T) {
	mem, source, idx := graphicsFixture()
	oldProfiles := hdGraphicsSources
	hdGraphicsSources = []graphicsSourceProfile{source}
	defer func() { hdGraphicsSources = oldProfiles }()
	frame := make([]byte, 640*400)
	for y := 0; y < 74; y++ {
		copy(frame[(291+y)*640+464:], idx[y*144:(y+1)*144])
	}
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame}, nil }
	var state hdGraphicsState
	state.observe(1, 0x1c1bc, 0x60000, mem, snapshot)
	state.observe(2, 0x12345, 0x60004, mem, snapshot)
	original := state.Draws[0]
	// 來源未知但矩形契約已知，即使目的索引剛好相同也保留前景。
	mem[0x100000] ^= 1
	for i, value := range [5]uint32{0, 1, 20, 80, 0xa0000 + 300*80 + 65} {
		binary.LittleEndian.PutUint32(mem[0x60004+i*4:], value)
	}
	beforeMem := sha256.Sum256(mem)
	state.observe(3, 0x1c1bc, 0x60000, mem, snapshot)
	if state.Pending != nil || len(state.Draws) != 1 || len(state.Draws[0].VisibleRects) != 4 || sha256.Sum256(mem) != beforeMem {
		t.Fatal("partial unknown copy or read-only lifetime", state)
	}
	d := state.Draws[0]
	if d.Args != original.Args || d.SourceRect != original.SourceRect || d.DestRect != original.DestRect || d.ExpectedSHA != original.ExpectedSHA {
		t.Fatal("visibility clipping replaced original evidence")
	}
	asset := image.NewNRGBA(image.Rect(0, 0, 288, 148))
	for i := 0; i < len(asset.Pix); i += 4 {
		copy(asset.Pix[i:i+4], []byte{190, 60, 30, 255})
	}
	profile := hdGraphicsProfile{Source: source, IndicesSHA: fmt.Sprintf("%x", sha256.Sum256(idx))}
	pair := newHDGraphicsPair(profile, idx, asset, 2)
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: frame, MaxColor: 15}
	for i, c := range battleFrameNativePalette {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
	}
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	r := pair.apply(out, s, d, true)
	if r.Mode != "hd" || r.PaintedNative != 144*74-8*20 || out.NRGBAAt(1040, 600) != (color.NRGBA{}) || out.NRGBAAt(928, 582).A != 255 {
		t.Fatal("unknown foreground overwritten or visible background lost", r)
	}
	frame[291*640+464] ^= 1
	before := append([]byte(nil), out.Pix...)
	if pair.apply(out, s, d, true).Mode != "fallback_source_mismatch" || !bytes.Equal(before, out.Pix) {
		t.Fatal("fragment validation painted partial output before failure")
	}
	frame[291*640+464] ^= 1
	for _, visible := range [][]image.Rectangle{{image.Rect(0, 0, 1, 1)}, {d.DestRect, d.DestRect}, {image.Rectangle{}}} {
		bad := d
		bad.VisibleRects = visible
		if pair.apply(out, s, bad, true).Mode != "fallback_visibility" || !bytes.Equal(before, out.Pix) {
			t.Fatal("malformed visibility accepted")
		}
	}
	// 多次切割的片段數也有上限。
	state.Draws = []graphicsDrawRecord{{DestRect: image.Rect(0, 0, 640, 400)}}
	for y := 0; y < 256; y++ {
		state.Draws[0].VisibleRects = append(state.Draws[0].VisibleRects, image.Rect(60, y, 80, y+1))
	}
	for i, value := range [5]uint32{0, 1, 400, 80, 0xa0008} {
		binary.LittleEndian.PutUint32(mem[0x60004+i*4:], value)
	}
	state.observe(4, 0x1c1bc, 0x60000, mem, snapshot)
	if len(state.Draws) != 0 {
		t.Fatal("fragment bound not enforced")
	}
}

func TestHDGraphicsPaintRegionPreservesBuffer(t *testing.T) {
	idx := make([]byte, 48*32)
	profile := hdGraphicsProfile{Source: graphicsSourceProfile{Name: "buffer.OZM", Width: 48, Height: 32},
		IndicesSHA: fmt.Sprintf("%x", sha256.Sum256(idx)), PaintRect: image.Rect(8, 4, 32, 24)}
	asset := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for i := 0; i < len(asset.Pix); i += 4 {
		copy(asset.Pix[i:i+4], []byte{100, 80, 60, 255})
	}
	pair := newHDGraphicsPair(profile, idx, asset, 2)
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), MaxColor: 15}
	for i, c := range battleFrameNativePalette {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
	}
	draw := graphicsDrawRecord{Name: profile.Source.Name, SourceRect: image.Rect(0, 0, 48, 32), DestRect: image.Rect(80, 100, 128, 132),
		ExpectedSHA: profile.IndicesSHA, Verified: true}
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	if r := pair.apply(out, s, draw, true); r.Mode != "hd" || r.PaintedNative != 24*20 || out.NRGBAAt(160, 200) != (color.NRGBA{}) || out.NRGBAAt(176, 208).A != 255 {
		t.Fatal("unused buffer overwritten", r)
	}
	profile.PaintRect = image.Rect(0, 0, 49, 32)
	if newHDGraphicsPair(profile, idx, asset, 2).Reason != "fallback_paint_region" {
		t.Fatal("invalid paint region accepted")
	}
}
