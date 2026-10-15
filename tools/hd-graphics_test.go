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

func graphicsCompositeRAM(t *testing.T, profile graphicsSourceProfile, indexed []byte) []byte {
	t.Helper()
	mem := make([]byte, 0x200000)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], uint32(profile.Width/8))
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], uint32(profile.Height))
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base, stride := 0x100000+bit*0x20000, profile.Width/8
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		plane := make([]byte, stride*profile.Height)
		for i, value := range indexed {
			plane[i/8] |= (value >> uint(bit) & 1) << uint(7-i%8)
		}
		if fmt.Sprintf("%x", sha256.Sum256(plane)) != profile.Planes[bit] {
			t.Fatal("actual source planes differ", profile.Name, bit)
		}
		for y := 0; y < profile.Height; y++ {
			copy(mem[base+y*80:], plane[y*stride:(y+1)*stride])
		}
	}
	return mem
}

func TestHDGraphicsCompositeActualSourcesAndVisibility(t *testing.T) {
	root := "/project/workplace/hd-prototype/catalog/remaining/consumers/battle/r14/native-turn3/"
	data, err := os.ReadFile(root + "receipt.json")
	if err != nil {
		t.Skip("local original combat receipt unavailable")
	}
	var receipt struct {
		Composite struct{ Records []compositeRecord } `json:"composite_draw_trace"`
	}
	if json.Unmarshal(data, &receipt) != nil || len(receipt.Composite.Records) != 12 {
		t.Fatal("four original combat samples required")
	}
	readBlob := func(path string) []byte {
		blob, err := os.ReadFile(root + path[len("/output/"):])
		if err != nil {
			t.Fatal(err)
		}
		return blob
	}
	first := receipt.Composite.Records[0]
	frame := append([]byte(nil), readBlob(first.BeforePath)...)
	state := hdGraphicsState{Draws: []graphicsDrawRecord{{Name: "BATTLE02.OZM", SourceRect: image.Rect(0, 0, 640, 400), DestRect: image.Rect(0, 0, 640, 400), Verified: true, ExpectedSHA: first.BeforeSHA}}}
	owners := bytes.Repeat([]byte{1}, 640*400)
	profiles := map[string]hdGraphicsProfile{}
	for _, profile := range hdGraphicsProfiles {
		profiles[profile.Source.Name] = profile
	}
	for sequence, record := range receipt.Composite.Records[:3] {
		profile := profiles[record.Source]
		indexed := readBlob(record.SourcePath)
		mem := graphicsCompositeRAM(t, profile.Source, indexed)
		binary.LittleEndian.PutUint32(mem[record.Stack:], record.Return)
		for i, arg := range record.Args {
			binary.LittleEndian.PutUint32(mem[record.Stack+4+uint32(i*4):], arg)
		}
		before, after := readBlob(record.BeforePath), readBlob(record.AfterPath)
		for y := 0; y < record.DestRect.Dy(); y++ {
			copy(frame[(record.DestRect.Min.Y+y)*640+record.DestRect.Min.X:], before[y*record.DestRect.Dx():(y+1)*record.DestRect.Dx()])
		}
		snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame, MaxColor: 15}, nil }
		ram := sha256.Sum256(mem)
		state.observe(record.BeginStep, record.Entry, record.Stack, mem, snapshot)
		state.observe(record.BeginStep+1, record.Return, record.Stack, mem, snapshot)
		if state.Composite == nil || state.Completed != uint64(sequence) {
			t.Fatal("incorrect return accepted")
		}
		for y := 0; y < record.DestRect.Dy(); y++ {
			copy(frame[(record.DestRect.Min.Y+y)*640+record.DestRect.Min.X:], after[y*record.DestRect.Dx():(y+1)*record.DestRect.Dx()])
		}
		state.observe(record.EndStep, record.Return, record.Stack+4, mem, snapshot)
		if state.Composite != nil || state.Completed != uint64(sequence+1) || sha256.Sum256(mem) != ram || len(state.Draws) != sequence+2 {
			t.Fatal("actual composite return, read-only state or fragment limit", sequence, len(state.Draws), graphicsFragmentCount(state.Draws))
		}
		selected := graphicsCrop(indexed, profile.Source.Width, record.SourceRect)
		for i, value := range selected {
			x, y := record.DestRect.Min.X+i%record.DestRect.Dx(), record.DestRect.Min.Y+i/record.DestRect.Dx()
			if value != 0 {
				owners[y*640+x] = 0
				if value == after[i] && !(record.Source == "BATTLE.OZM" && graphicsBattleUnit(image.Pt(x, y))) {
					owners[y*640+x] = byte(sequence + 2)
				}
			}
		}
		actual := make([]byte, 640*400)
		for layer, draw := range state.Draws {
			visible := draw.VisibleRects
			if visible == nil {
				visible = []image.Rectangle{draw.DestRect}
			}
			for _, rect := range visible {
				for y := rect.Min.Y; y < rect.Max.Y; y++ {
					for x := rect.Min.X; x < rect.Max.X; x++ {
						if actual[y*640+x] != 0 {
							t.Fatal("overlapping composite ownership", x, y)
						}
						actual[y*640+x] = byte(layer + 1)
					}
				}
			}
		}
		if !bytes.Equal(actual, owners) || graphicsFragmentCount(state.Draws) > 256 {
			t.Fatal("native source-zero or mixed-OR visibility differs")
		}
	}
	// 缺圖不改已撤銷的較早圖層；後續全畫面原版複製必須撤銷合成。
	state.observe(999999999, 0x1c084, 0, make([]byte, 32), nil)
	if len(state.Draws) != 0 || state.Composite != nil {
		t.Fatal("full copy resurrected an old composite")
	}
}

func TestHDGraphicsMaskBoundsEmptyAndFragmentLimit(t *testing.T) {
	if rects, ok := graphicsMaskRectangles(make([]bool, 16), image.Rect(0, 0, 4, 4)); !ok || rects == nil || len(rects) != 0 {
		t.Fatal("empty mask became a full source")
	}
	mask := make([]bool, 640*400)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			mask[y*640+x] = (x+y)&1 == 0
		}
	}
	if _, ok := graphicsMaskRectangles(mask, image.Rect(0, 0, 640, 400)); ok {
		t.Fatal("unbounded mask fragments accepted")
	}
	if _, ok := graphicsMaskRectangles(nil, image.Rect(-1, 0, 2, 2)); ok {
		t.Fatal("invalid mask geometry accepted")
	}
}

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
