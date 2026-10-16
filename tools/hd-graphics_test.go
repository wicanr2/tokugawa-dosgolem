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

func TestEnding399KnownRowsFromOriginalFullCopy(t *testing.T) {
	for _, name := range []string{"END-11.OZM", "END-11B.OZM"} {
		profile := openingEffectProfile(t, name)
		indexed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + name + ".idx")
		if err != nil || profile.Source.Width != 640 || profile.Source.Height != 399 {
			t.Fatal("actual ending source unavailable", name, err)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, indexed, asset, 2)
		mem := graphicsCompositeRAM(t, profile.Source, indexed)
		ret := uint32(0x18d55)
		if name == "END-11B.OZM" {
			ret = 0x18daf
		}
		binary.LittleEndian.PutUint32(mem[0x1000:], ret)
		beforeRAM := sha256.Sum256(mem)
		s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), MaxColor: 15}
		copy(s.Indexed, indexed)
		palette := graphicsNativePalette(profile)
		for i, c := range palette {
			s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
		}
		for x := 0; x < 640; x++ {
			s.Indexed[399*640+x] = byte(x % 16)
		}
		for i, value := range s.Indexed {
			s.Image.SetNRGBA(i%640, i/640, palette[value])
		}
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		var state hdGraphicsState
		state.observe(1, 0x1c084, 0x1000, mem, snapshot)
		if state.Pending == nil || len(state.Draws) != 0 || state.Pending.Record.Args[2] != 400 || state.Pending.Record.SourceRect != image.Rect(0, 0, 640, 399) {
			t.Fatal("original arguments or known source range", name)
		}
		state.observe(2, ret, 0x1000, mem, snapshot)
		if state.Pending == nil {
			t.Fatal("wrong return stack accepted")
		}
		state.observe(3, ret, 0x1004, mem, snapshot)
		if len(state.Draws) != 1 || !state.Draws[0].Verified || state.Draws[0].Kind != "opaque_full_known_rows" || beforeRAM != sha256.Sum256(mem) {
			t.Fatal("known full-copy return or guest mutation", name)
		}
		out, _, _ := previewA0(s.Image, nil, nil, 2, false)
		original := append([]byte(nil), out.Pix...)
		if report := pair.apply(out, s, state.Draws[0], true); report.Mode != "hd" || report.Rect != image.Rect(0, 0, 640, 399) {
			t.Fatal("ending known rows not rendered", name, report)
		}
		if !bytes.Equal(out.Pix[798*out.Stride:], original[798*out.Stride:]) {
			t.Fatal("unknown last row changed", name)
		}
		// 未知原版末列可變，資格只涵蓋已核對的前 399 列。
		s.Indexed[399*640] ^= 15
		s.Image.SetNRGBA(0, 399, palette[s.Indexed[399*640]])
		var trace graphicsTrace
		trace.Profiles = []graphicsSourceProfile{profile.Source}
		trace.observe(4, 0x1c084, 0x1000, mem, snapshot)
		trace.observe(5, ret, 0x1004, mem, snapshot)
		if len(trace.Records) != 1 || !trace.Records[0].Verified {
			t.Fatal("unknown row blocked known prefix", name)
		}
		missing := newHDGraphicsPair(profile, indexed, nil, 2)
		copy(out.Pix, original)
		if missing.apply(out, s, state.Draws[0], true).Mode != "fallback_missing_asset" || !bytes.Equal(out.Pix, original) {
			t.Fatal("missing ending art changed pixels")
		}
		s.Indexed[398*640] ^= 1
		trace.Pending, trace.Records = nil, nil
		trace.observe(6, 0x1c084, 0x1000, mem, snapshot)
		trace.observe(7, ret, 0x1004, mem, snapshot)
		if len(trace.Records) != 1 || trace.Records[0].Verified {
			t.Fatal("changed known destination accepted", name)
		}
		unsupported := profile.Source
		unsupported.Name = "UNKNOWN-399.OZM"
		trace = graphicsTrace{Profiles: []graphicsSourceProfile{unsupported}}
		trace.observe(8, 0x1c084, 0x1000, mem, snapshot)
		if trace.Pending != nil {
			t.Fatal("unapproved short full-copy source accepted")
		}
	}
}

func TestWakuCachedFrameActualPlanesAndUnderlay(t *testing.T) {
	profile := openingEffectProfile(t, "MAP_WAKU.OZM")
	mem, err := os.ReadFile("/project/workplace/hd-prototype/catalog/remaining/consumers/battle/r19/native-control/startup-memory.bin")
	if err != nil {
		t.Fatal(err)
	}
	base := binary.LittleEndian.Uint32(mem[0x42188:])
	indexed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/MAP_WAKU.OZM.idx")
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range []uint32{0x1ec31, base + 0x3180, 0xa0005} {
		binary.LittleEndian.PutUint32(mem[0x1000+i*4:], value)
	}
	beforeRAM := sha256.Sum256(mem)
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400), MaxColor: 15}
	palette := graphicsNativePalette(profile)
	for i, c := range palette {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
	}
	for y := 0; y < 88; y++ {
		for x := 0; x < 384; x++ {
			value := indexed[y*384+x]
			s.Indexed[y*640+40+x] = value
			s.Image.SetNRGBA(40+x, y, palette[value])
		}
	}
	snapshot := func() (nativeSnapshot, error) { return s, nil }
	var state hdGraphicsState
	state.observe(1, 0x1fff7, 0x1000, mem, snapshot)
	if state.Pending == nil || len(state.Draws) != 0 {
		t.Fatal("cached frame activated before fourth plane return")
	}
	state.observe(2, 0x1ec31, 0x1000, mem, snapshot)
	if state.Pending == nil {
		t.Fatal("wrong cached return stack accepted")
	}
	state.observe(3, 0x1ec31, 0x1004, mem, snapshot)
	if len(state.Draws) != 1 || !state.Draws[0].Verified || state.Draws[0].Kind != "cached_frame" || beforeRAM != sha256.Sum256(mem) {
		t.Fatal("actual cached source/natural return or read-only contract")
	}
	draw := state.Draws[0]
	if draw.CacheBase != base || len(draw.RawHelperArgs) != 2 || draw.RawHelperArgs[0] != base+0x3180 || draw.RawHelperArgs[1] != 0xa0005 {
		t.Fatal("raw cached helper arguments lost")
	}
	asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
	if err != nil {
		t.Fatal(err)
	}
	pair := newHDGraphicsPair(profile, indexed, asset, 2)
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	underlay := color.NRGBA{21, 67, 102, 255}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			out.SetNRGBA(x, y, underlay)
		}
	}
	original := append([]byte(nil), out.Pix...)
	// 後畫的場景／文字在原始黑區改變索引，已存在的 host 畫布必須保留。
	black, nonblack := -1, -1
	for i, v := range indexed[:384*88] {
		if v == 8 && black < 0 {
			black = i
		}
		if v != 8 && nonblack < 0 {
			nonblack = i
		}
	}
	if black < 0 || nonblack < 0 {
		t.Fatal("actual source mask")
	}
	s.Indexed[black/384*640+40+black%384] = 15
	report := pair.apply(out, s, draw, true)
	if report.Mode != "hd" || report.PaintedNative+report.PreservedNative != 384*88 {
		t.Fatal("cached frame rendering", report)
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			want := underlay
			if x >= 80 && x < 848 && y < 176 && indexed[y/2*384+(x-80)/2] != 8 {
				want = rgbaAt(pair.Asset, x-80, y)
			}
			if out.NRGBAAt(x, y) != want {
				t.Fatal("cached frame/source-black underlay or outside pixels", x, y)
			}
		}
	}
	for _, failure := range []string{"disabled", "missing", "palette", "changed_border", "raw_args", "unknown_source"} {
		copy(out.Pix, original)
		p, d, native := pair, draw, s
		if failure == "missing" {
			p.Reason = "fallback_missing_asset"
		}
		if failure == "palette" {
			native.Palette[0][0] ^= 1
		}
		if failure == "changed_border" {
			native.Indexed = append([]byte(nil), s.Indexed...)
			native.Indexed[nonblack/384*640+40+nonblack%384] ^= 1
		}
		if failure == "raw_args" {
			d.RawHelperArgs = []uint32{base, 0xa0005}
		}
		if failure == "unknown_source" {
			d.Name = "OTHER.OZM"
		}
		r := p.apply(out, native, d, failure != "disabled")
		if r.Mode == "hd" || !bytes.Equal(out.Pix, original) {
			t.Fatal("unsafe cached fallback", failure, r)
		}
	}
	s.Indexed[black/384*640+40+black%384] = 8
	control := graphicsTrace{Profiles: []graphicsSourceProfile{profile.Source}}
	control.observe(4, 0x1fff7, 0x1000, mem, snapshot)
	control.observe(5, 0x1ec31, 0x1004, mem, snapshot)
	if len(control.Records) != 1 || !control.Records[0].Verified {
		t.Fatal("negative cases require an independently valid native destination")
	}
	for _, failure := range []string{"source_changed", "base_changed", "destination_changed", "missing_profile", "destination_return_changed"} {
		trace := graphicsTrace{Profiles: []graphicsSourceProfile{profile.Source}}
		if failure == "missing_profile" {
			trace.Profiles = nil
		}
		trace.observe(4, 0x1fff7, 0x1000, mem, snapshot)
		if failure == "missing_profile" {
			if trace.Pending != nil {
				t.Fatal("missing source profile accepted")
			}
			continue
		}
		if trace.Pending == nil {
			t.Fatal("valid cached source rejected")
		}
		if failure == "source_changed" {
			mem[base] ^= 1
		}
		if failure == "base_changed" {
			binary.LittleEndian.PutUint32(mem[0x42188:], base+1)
		}
		if failure == "destination_return_changed" {
			s.Indexed[nonblack/384*640+40+nonblack%384] ^= 1
		}
		if failure == "destination_changed" {
			binary.LittleEndian.PutUint32(mem[0x1008:], 0xa0006)
			trace.Pending = nil
			trace.observe(5, 0x1fff7, 0x1000, mem, snapshot)
			if trace.Pending != nil {
				t.Fatal("unknown destination accepted")
			}
			binary.LittleEndian.PutUint32(mem[0x1008:], 0xa0005)
			continue
		}
		trace.observe(6, 0x1ec31, 0x1004, mem, snapshot)
		if len(trace.Records) != 1 || trace.Records[0].Verified {
			t.Fatal("corrupt cached return accepted", failure)
		}
		if failure == "source_changed" {
			mem[base] ^= 1
		}
		if failure == "base_changed" {
			binary.LittleEndian.PutUint32(mem[0x42188:], base)
		}
		if failure == "destination_return_changed" {
			s.Indexed[nonblack/384*640+40+nonblack%384] ^= 1
		}
	}
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

func TestHDGraphicsCompositeForegroundAndFallback(t *testing.T) {
	var profile hdGraphicsProfile
	for _, candidate := range hdGraphicsProfiles {
		if candidate.Source.Name == "BATTLE.OZM" {
			profile = candidate
		}
	}
	indexed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/BATTLE.OZM.idx")
	if err != nil {
		t.Skip("local original battle source unavailable")
	}
	asset := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	for i := 0; i < len(asset.Pix); i += 4 {
		copy(asset.Pix[i:i+4], []byte{190, 50, 30, 255})
	}
	pair := newHDGraphicsPair(profile, indexed, asset, 2)
	draw := graphicsDrawRecord{Name: "BATTLE.OZM", Kind: "or_full", SourceRect: image.Rect(0, 0, 640, 400), DestRect: image.Rect(0, 0, 640, 400),
		Verified: true, ExpectedSHA: "synthetic-native-result", ActualSHA: "synthetic-native-result",
		SourceCropSHA: profile.IndicesSHA, CompositeSourceSHA: profile.IndicesSHA}
	var foreground, ordinary, transparent, unit image.Point
	found := [4]bool{}
	for y := 0; y < 247; y++ {
		for x := 0; x < 640; x++ {
			point, value := image.Pt(x, y), indexed[y*640+x]
			if value != 0 && graphicsCompositeForeground(draw, point) && !graphicsBattleUnit(point) && !found[0] {
				foreground, found[0] = point, true
			}
			if value != 0 && !graphicsCompositeForeground(draw, point) && !graphicsBattleUnit(point) && !found[1] {
				ordinary, found[1] = point, true
			}
			if value == 0 && !found[2] {
				transparent, found[2] = point, true
			}
			if value != 0 && graphicsBattleUnit(point) && !found[3] {
				unit, found[3] = point, true
			}
		}
	}
	if found != [4]bool{true, true, true, true} {
		t.Fatal("actual source test regions absent", found)
	}
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: append([]byte(nil), indexed...), MaxColor: 15}
	for i, value := range graphicsNativePalette(profile) {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{value.R, value.G, value.B}
	}
	s.Indexed[foreground.Y*640+foreground.X] = 0
	nativeColour := color.NRGBA{12, 34, 56, 255}
	s.Image.SetNRGBA(foreground.X, foreground.Y, nativeColour)
	draw.VisibleRects = []image.Rectangle{image.Rectangle{Min: foreground, Max: foreground.Add(image.Pt(1, 1))}}
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	if report := pair.apply(out, s, draw, true); report.Mode != "hd" || report.PreservedNative != 1 || report.PaintedNative != 0 || out.NRGBAAt(foreground.X*2, foreground.Y*2) != nativeColour {
		t.Fatal("original foreground colour or count", report)
	}
	for _, kind := range []string{"disabled", "missing", "source", "composite", "unknown-write", "transparent", "unit"} {
		d, selected := draw, pair
		if kind == "missing" {
			selected = newHDGraphicsPair(profile, indexed, nil, 2)
		}
		if kind == "source" {
			d.SourceCropSHA = "wrong"
		}
		if kind == "composite" {
			d.ActualSHA = "wrong"
		}
		point := foreground
		if kind == "unknown-write" {
			point = ordinary
			s.Indexed[point.Y*640+point.X] = indexed[point.Y*640+point.X] ^ 1
		} else if kind == "transparent" {
			point = transparent
		} else if kind == "unit" {
			point = unit
		}
		d.VisibleRects = []image.Rectangle{{Min: point, Max: point.Add(image.Pt(1, 1))}}
		before := append([]byte(nil), out.Pix...)
		if report := selected.apply(out, s, d, kind != "disabled"); report.Mode == "hd" || !bytes.Equal(before, out.Pix) {
			t.Fatal("fallback modified output", kind, report)
		}
	}
}

func TestHDWindowLayersRestoreRegionAndRemoveTemporaryPortrait(t *testing.T) {
	packed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/window/WINDOW-packed.bin")
	if err != nil {
		t.Skip("local original window source unavailable")
	}
	mem := make([]byte, 0x200000)
	copy(mem[0x100000:], packed)
	binary.LittleEndian.PutUint32(mem[0x35153:], 0x100000)
	esp, ret := uint32(0x1000), uint32(0x19000)
	rect := image.Rect(128, 286, 208, 366)
	for i, value := range []uint32{ret, 16, 286, 5, 5} {
		binary.LittleEndian.PutUint32(mem[esp+uint32(i*4):], value)
	}
	frame := make([]byte, 640*400)
	for i := range frame {
		frame[i] = byte((i/640 + i%640) % 16)
	}
	original := graphicsDrawRecord{Name: "synthetic-background", Kind: "opaque_full", BeginStep: 1, EndStep: 2, Verified: true, SourceRect: image.Rect(0, 0, 640, 400), DestRect: image.Rect(0, 0, 640, 400)}
	outsideRect := image.Rect(464, 291, 608, 365)
	newOutside := graphicsDrawRecord{Name: "synthetic-later-weather", Kind: "opaque_rect", BeginStep: 6, EndStep: 7, Verified: true, SourceRect: image.Rect(0, 0, 144, 74), DestRect: outsideRect, VisibleRects: []image.Rectangle{outsideRect}}
	portrait := hdKaoDraw{Index: 0, Rect: image.Rect(40, 66, 120, 146), EndStep: 3}
	h := hdLive{Graphics: hdGraphicsState{Draws: []graphicsDrawRecord{original}}, Kao: hdKaoState{Draws: []hdKaoDraw{portrait}}}
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame, MaxColor: 15}, nil }
	before := graphicsCrop(frame, 640, rect)
	ram := sha256.Sum256(mem)
	h.observeWindowLayers(4, 0x1b165, esp, mem, snapshot)
	if h.WindowLayers.PendingSave == nil || sha256.Sum256(mem) != ram {
		t.Fatal("save entry guard or RAM mutation")
	}
	// 合成原版保存動作，並以獨立四平面編碼檢查讀取端。
	for bit := 0; bit < 4; bit++ {
		for i, value := range before {
			mem[0x120000+bit*len(before)/8+i/8] |= (value >> uint(bit) & 1) << uint(7-i%8)
		}
	}
	for i, value := range []uint32{16 + 286*80, 5, 5, 0x120000} {
		binary.LittleEndian.PutUint32(mem[0x34ed3+i*4:], value)
	}
	binary.LittleEndian.PutUint32(mem[0x35157:], 1)
	ram = sha256.Sum256(mem)
	h.observeWindowLayers(5, ret, esp, mem, snapshot)
	if h.WindowLayers.Saved != nil {
		t.Fatal("wrong stack activated saved layers")
	}
	h.observeWindowLayers(5, ret, esp+4, mem, snapshot)
	if h.WindowLayers.Saved[0] == nil || sha256.Sum256(mem) != ram {
		t.Fatal("saved original buffer guard or RAM mutation")
	}
	saved := h.WindowLayers.Saved[0]
	h.Graphics.Draws = graphicsClipRegion([]graphicsDrawRecord{original}, rect, false)
	h.Graphics.Draws = graphicsClipRegion(h.Graphics.Draws, outsideRect, false)
	h.Graphics.Draws = append(h.Graphics.Draws, graphicsDrawRecord{Name: "synthetic-animation", DestRect: rect, VisibleRects: []image.Rectangle{rect}}, newOutside)
	h.Kao.Draws = append(h.Kao.Draws, hdKaoDraw{Index: 22, Rect: rect, EndStep: 8})
	h.observeWindowLayers(9, 0x1b245, esp, mem, snapshot)
	if len(h.Kao.Draws) != 1 || h.Kao.Draws[0] != portrait || h.WindowLayers.PendingRestore == nil {
		t.Fatal("temporary portrait was not revoked")
	}
	binary.LittleEndian.PutUint32(mem[0x35157:], 0)
	ram = sha256.Sum256(mem)
	h.observeWindowLayers(10, ret, esp+4, mem, snapshot)
	if len(h.Graphics.Draws) != 2 || h.WindowLayers.PendingRestore != nil || sha256.Sum256(mem) != ram || h.Graphics.Draws[1].Name != newOutside.Name {
		t.Fatal("restoration lost newer outside drawing or changed RAM", h.Graphics.Draws)
	}
	area := 0
	for _, visible := range h.Graphics.Draws[0].VisibleRects {
		if visible.Overlaps(outsideRect) {
			t.Fatal("restored old background over newer outside drawing")
		}
		area += visible.Dx() * visible.Dy()
	}
	if area != 640*400-outsideRect.Dx()*outsideRect.Dy() {
		t.Fatal("restored region has a gap", area)
	}
	// 相同返回但原生目的不同時，不能恢復保存的圖層。
	h.Graphics.Draws = nil
	h.WindowLayers.PendingRestore = &hdWindowLayersPending{Return: ret, Stack: esp, Level: 0, Region: saved}
	frame[rect.Min.Y*640+rect.Min.X] ^= 1
	h.observeWindowLayers(11, ret, esp+4, mem, snapshot)
	if len(h.Graphics.Draws) != 0 || h.WindowLayers.Events[len(h.WindowLayers.Events)-1]["verified"] != false {
		t.Fatal("changed native restoration resurrected saved HD")
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

func TestHDGraphicsCachedFrameOcclusionByNaturalTerrainReturn(t *testing.T) {
	draw := graphicsDrawRecord{Name: "MAP_WAKU.OZM", Kind: "cached_frame", EndStep: 10, DestRect: image.Rect(40, 0, 424, 88)}
	old := image.Rect(40, 0, 88, 32)
	floor := image.Rect(64, 56, 112, 88)
	sprite := image.Rect(40, 68, 88, 116)
	terrain := hdTerrainState{History: []hdTerrainDraw{{Rect: old, EndStep: 9}, {Rect: floor, EndStep: 11}}, Sprites: []hdTerrainSpriteDraw{{Rect: sprite, EndStep: 12}}}
	clipped := graphicsOcclude(draw, nil, nil, terrain)
	for y := 0; y < 88; y++ {
		for x := 40; x < 424; x++ {
			point, visible := image.Pt(x, y), 0
			for _, rect := range clipped.VisibleRects {
				if point.In(rect) {
					visible++
				}
			}
			want := 1
			if point.In(floor) || point.In(sprite) {
				want = 0
			}
			if visible != want {
				t.Fatal("later original rectangle or earlier return ordering", point, visible, want)
			}
		}
	}
	// 後畫來源即使碰巧保留相同索引，也不能讓舊框架覆蓋它。
	if len(clipped.VisibleRects) == 0 || clipped.DestRect != draw.DestRect {
		t.Fatal("occlusion discarded independent frame evidence")
	}
	unknown := hdTerrainState{History: []hdTerrainDraw{{Rect: image.Rect(-1, 0, 1, 1), EndStep: 11}}}
	if graphicsOcclude(draw, nil, nil, unknown).VisibleRects == nil || len(graphicsOcclude(draw, nil, nil, unknown).VisibleRects) != 0 {
		t.Fatal("invalid later rectangle kept cached frame")
	}
	draw.Kind = "opaque_rect"
	unchanged := graphicsOcclude(draw, nil, nil, terrain)
	if len(unchanged.VisibleRects) != 1 || unchanged.VisibleRects[0] != draw.DestRect {
		t.Fatal("terrain changed an unrelated OZM contract")
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
