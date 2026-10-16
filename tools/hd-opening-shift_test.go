package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"testing"
)

type openingShiftFixture struct {
	Source  string      `json:"source"`
	Sample  int         `json:"sample_id"`
	Entry   uint32      `json:"entry_linear"`
	Count   int         `json:"shift_count"`
	Palette [16][3]byte `json:"active_palette"`
	Unknown [][2]int    `json:"outside_formal_source_pixels"`
}

func openingShiftFixtures(t *testing.T) []openingShiftFixture {
	t.Helper()
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/remaining/history/scrolling/source-verification.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof struct {
		Verified bool
		Records  []openingShiftFixture
	}
	if json.Unmarshal(data, &proof) != nil || !proof.Verified || len(proof.Records) != 13 {
		t.Fatal("thirteen ordinary original fixtures")
	}
	return proof.Records
}

func openingShiftFixtureMemory(t *testing.T, f openingShiftFixture) ([]byte, nativeSnapshot, hdGraphicsProfile, []byte) {
	t.Helper()
	mem, s := openingPartialMemory(t, openingPartialFixture{Source: f.Source, Sample: f.Sample, Palette: f.Palette})
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 200)
	var profile hdGraphicsProfile
	for _, p := range hdGraphicsProfiles {
		if p.Source.Name == f.Source {
			profile = p
		}
	}
	original, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + f.Source + ".idx")
	if err != nil || len(original) != 128000 {
		t.Fatal("original source", err)
	}
	return mem, s, profile, original
}

func openingShiftFixtureState(f openingShiftFixture, original []byte) *openingShiftState {
	if f.Count == 0 {
		return nil
	}
	return &openingShiftState{Name: f.Source, Original: original, Count: f.Count - 1, Bases: [4]uint32{0x100000, 0x120000, 0x140000, 0x160000}}
}

func TestOpeningShiftActualOriginalFixtures(t *testing.T) {
	for _, f := range openingShiftFixtures(t) {
		mem, s, profile, original := openingShiftFixtureMemory(t, f)
		h := hdGraphicsState{Shift: openingShiftFixtureState(f, original)}
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		before := sha256.Sum256(mem)
		h.observe(1, f.Entry, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 {
			t.Fatal("entry source", f.Source, f.Count)
		}
		h.observe(2, 0x12345, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 {
			t.Fatal("wrong return stack accepted")
		}
		h.observe(3, 0x12345, 0x60004, mem, snapshot)
		if len(h.Draws) != 1 || h.Completed != 1 || sha256.Sum256(mem) != before {
			t.Fatal("natural source/copy or guest changed", f.Source, f.Count)
		}
		draw := h.Draws[0]
		if draw.SourceShift != f.Count || !draw.Verified {
			t.Fatal("actual shift", draw)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, original, asset, 2)
		base, _ := prepareHDImage(s.Image, 640, 400, 2)
		out := base.(*image.NRGBA)
		native := append([]byte(nil), out.Pix...)
		report := pair.apply(out, s, draw, true)
		if report.Mode != "hd" || report.PreservedNative != len(f.Unknown) || report.PaintedNative+report.PreservedNative != draw.DestRect.Dx()*draw.DestRect.Dy() {
			t.Fatal("actual paint and unknown", f.Source, f.Count, report)
		}
		spec, _ := openingShiftDefinition(f.Entry)
		for y := 0; y < 800; y++ {
			for x := 0; x < 1280; x++ {
				nx, ny := x/2, y/2
				want := rgbaAt(s.Image, nx, ny)
				if image.Pt(nx, ny).In(spec.Dest) {
					position := (ny-spec.Dest.Min.Y)*640 + spec.X + nx - spec.Dest.Min.X
					if position >= 16 {
						position = max(15, position-f.Count)
					}
					if position < len(original) {
						want = report.Effect.colour(rgbaAt(pair.Asset, position%640*2+x%2, position/640*2+y%2))
					}
				}
				if out.NRGBAAt(x, y) != want {
					t.Fatal("independent row-major HD formula", f.Source, f.Count, x, y)
				}
			}
		}
		for _, reject := range []string{"missing", "palette", "native", "count", "source_hash", "visibility"} {
			copy(out.Pix, native)
			candidate, bad, record := pair, s, draw
			switch reject {
			case "missing":
				candidate = newHDGraphicsPair(profile, original, nil, 2)
			case "palette":
				bad.Palette[bad.DAC[2]][0] ^= 1
			case "native":
				bad.Indexed = append([]byte(nil), s.Indexed...)
				bad.Indexed[spec.Dest.Min.Y*640+spec.Dest.Min.X] ^= 1
			case "count":
				record.SourceShift = -1
			case "source_hash":
				record.QualifiedSHA = "unknown"
			case "visibility":
				record.VisibleRects = []image.Rectangle{image.Rect(0, 0, 640, 400)}
			}
			if candidate.apply(out, bad, record, true).Mode == "hd" || sha256.Sum256(out.Pix) != sha256.Sum256(native) {
				t.Fatal("fallback altered output", f.Source, f.Count, reject)
			}
		}
	}
}

func TestOpeningShiftConsecutiveSequenceAndLimit(t *testing.T) {
	fixtures := openingShiftFixtures(t)
	for _, index := range []int{0, 6} {
		f := fixtures[index]
		mem, s, _, original := openingShiftFixtureMemory(t, f)
		spec, _ := openingShiftDefinition(f.Entry)
		initial := append(append([]byte(nil), original...), 0, 1, 2, 3, 4, 5, 6, 7)
		h := hdGraphicsState{}
		for count := 0; count <= spec.Maximum+1; count++ {
			current := make([]byte, 128008)
			for i := range current {
				from := i
				if i >= 16 {
					from = max(15, i-count)
				}
				current[i] = initial[from]
			}
			for bit := 0; bit < 4; bit++ {
				base := 0x100000 + bit*0x20000
				for i := 0; i < 16001; i++ {
					var value byte
					for x := 0; x < 8; x++ {
						value |= ((current[i*8+x] >> uint(bit)) & 1) << uint(7-x)
					}
					mem[base+i] = value
				}
			}
			for y := 0; y < spec.Rows; y++ {
				copy(s.Indexed[(y+spec.Dest.Min.Y)*640+spec.Dest.Min.X:], current[y*640+spec.X:y*640+spec.X+spec.Columns*8])
			}
			before := sha256.Sum256(mem)
			h.observe(count*10+1, f.Entry, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
			if count > spec.Maximum {
				if h.Pending != nil || h.Shift != nil || len(h.Draws) != 0 {
					t.Fatal("out of evidence range accepted")
				}
				break
			}
			h.observe(count*10+2, 0x12345, 0x60004, mem, func() (nativeSnapshot, error) { return s, nil })
			if len(h.Draws) != 1 || h.Draws[0].SourceShift != count || h.Completed != uint64(count+1) || sha256.Sum256(mem) != before {
				t.Fatal("consecutive transformation", f.Source, count)
			}
		}
	}
}

func TestOpeningShiftSourceAndReturnGuards(t *testing.T) {
	for _, f := range openingShiftFixtures(t) {
		for _, reject := range []string{"alias_before", "source_before", "alias_after", "source_after", "native_after", "no_snapshot", "guard_after"} {
			if reject == "guard_after" && f.Source != "OP8.OZM" {
				continue
			}
			mem, s, _, original := openingShiftFixtureMemory(t, f)
			h := hdGraphicsState{Shift: openingShiftFixtureState(f, original)}
			snapshot := func() (nativeSnapshot, error) { return s, nil }
			if reject == "alias_before" {
				mem[0x386d4] ^= 1
			}
			if reject == "source_before" {
				mem[0x100000] ^= 1
			}
			h.observe(1, f.Entry, 0x60000, mem, snapshot)
			if reject == "alias_before" || reject == "source_before" {
				if h.Pending != nil || h.Shift != nil {
					t.Fatal("unknown source", reject)
				}
				continue
			}
			spec, _ := openingShiftDefinition(f.Entry)
			switch reject {
			case "alias_after":
				mem[0x386d4] ^= 1
			case "source_after":
				mem[0x100000] ^= 1
			case "guard_after":
				mem[0x100000+16000] ^= 1
			case "native_after":
				s.Indexed[spec.Dest.Min.Y*640+spec.Dest.Min.X] ^= 1
			case "no_snapshot":
				snapshot = nil
			}
			h.observe(2, 0x12345, 0x60004, mem, snapshot)
			if len(h.Draws) != 0 || h.Shift != nil {
				t.Fatal("failed natural return retained qualification", f.Source, f.Count, reject)
			}
		}
	}
	mem := make([]byte, 32)
	tx := graphicsTrace{}
	tx.observe(1, 0x11c83, 0xfffffff0, mem, nil)
	if tx.Pending != nil || tx.Shift != nil {
		t.Fatal("overflowing stack")
	}
}

const endingProofRoot = "/project/workplace/hd-prototype/catalog/remaining/consumers/battle/r24/ready/"

type endingProofState struct {
	Count      int    `json:"count"`
	IndicesSHA string `json:"indices_sha256"`
	CropSHA    string `json:"display_crop_sha256"`
	PlanesSHA  string `json:"planes_sha256"`
	Path       string `json:"planes_path"`
}

func endingProofStates(t *testing.T) []endingProofState {
	t.Helper()
	data, err := os.ReadFile(endingProofRoot + "ending-source-verification.json")
	var proof struct {
		Verified bool
		ProbeSHA string `json:"probe_source_sha256"`
		CPU_SHA  string `json:"cpu_source_sha256"`
		States   []endingProofState
		Normal   bool `json:"normal_ending_verified"`
	}
	if err != nil || json.Unmarshal(data, &proof) != nil || !proof.Verified || proof.Normal || len(proof.States) != 241 {
		t.Fatal("bounded original-instruction evidence", err)
	}
	for path, want := range map[string]string{"/project/tools/ending-source-probe.go": proof.ProbeSHA, "/output/engine/internal/cpu386/cpu.go": proof.CPU_SHA} {
		data, err := os.ReadFile(path)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != want {
			t.Fatal("ending evidence source changed", path, err)
		}
	}
	return proof.States
}

func endingFixtureMemory(t *testing.T, state endingProofState) ([]byte, nativeSnapshot, hdGraphicsProfile, []byte) {
	t.Helper()
	original, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/END-2.OZM.idx")
	if err != nil || len(original) != 256000 {
		t.Fatal("ending source", err)
	}
	var profile hdGraphicsProfile
	for _, p := range hdGraphicsProfiles {
		if p.Source.Name == "END-2.OZM" {
			profile = p
		}
	}
	if fmt.Sprintf("%x", sha256.Sum256(original)) != profile.IndicesSHA {
		t.Fatal("ending source fingerprint")
	}
	packed, err := os.ReadFile(endingProofRoot + state.Path)
	if err != nil || len(packed) != 128000 || fmt.Sprintf("%x", sha256.Sum256(packed)) != state.PlanesSHA {
		t.Fatal("original CPU planes", err)
	}
	mem := make([]byte, 2<<20)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x10000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		copy(mem[base:], packed[bit*32000:(bit+1)*32000])
	}
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 80)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 400)
	ret := uint32(0x187f2)
	if state.Count > 0 {
		ret = 0x18822
	}
	binary.LittleEndian.PutUint32(mem[0x60000:], ret)
	_, current, _, ok := endingShiftCurrent(mem)
	if !ok || fmt.Sprintf("%x", sha256.Sum256(current)) != state.IndicesSHA {
		t.Fatal("actual CPU indices")
	}
	s := nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 256000), MaxColor: 15}
	for i := 3; i < len(s.Image.Pix); i += 4 {
		s.Image.Pix[i] = 255
	}
	palette := graphicsNativePalette(profile)
	for i, c := range palette {
		s.DAC[i], s.Palette[i] = byte(i), [3]byte{c.R, c.G, c.B}
	}
	for y := 0; y < 150; y++ {
		for x := 0; x < 400; x++ {
			value := current[y*640+x]
			s.Indexed[y*640+120+x] = value
			s.Image.SetNRGBA(120+x, y, palette[value])
		}
	}
	return mem, s, profile, original
}

func TestEndingShiftOriginalInstructionFixturesAndPaint(t *testing.T) {
	for _, state := range endingProofStates(t) {
		if state.Path == "" {
			continue
		}
		mem, s, profile, original := endingFixtureMemory(t, state)
		h := hdGraphicsState{}
		if state.Count > 0 {
			h.Shift = &openingShiftState{Name: "END-2.OZM", Bases: [4]uint32{0x100000, 0x110000, 0x120000, 0x130000}, Original: original, Count: state.Count - 1}
		}
		before := sha256.Sum256(mem)
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		h.observe(1, 0x185a7, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 || h.Pending.Record.ExpectedSHA != state.CropSHA {
			t.Fatal("ending entry", state.Count)
		}
		ret := h.Pending.Record.Return
		h.observe(2, ret, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 {
			t.Fatal("ending wrong stack")
		}
		h.observe(3, ret, 0x60004, mem, snapshot)
		if len(h.Draws) != 1 || h.Draws[0].SourceShift != state.Count || sha256.Sum256(mem) != before {
			t.Fatal("ending verified return", state.Count)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, original, asset, 2)
		base, reason := prepareHDImage(s.Image, 640, 400, 2)
		if reason != "" {
			t.Fatal("ending native fixture", reason)
		}
		out := base.(*image.NRGBA)
		native := append([]byte(nil), out.Pix...)
		draw := h.Draws[0]
		report := pair.apply(out, s, draw, true)
		if report.Mode != "hd" || report.PaintedNative != 60000 || report.PreservedNative != 0 {
			t.Fatal("ending HD", state.Count, report)
		}
		for y := 0; y < 800; y++ {
			for x := 0; x < 1280; x++ {
				want := rgbaAt(s.Image, x/2, y/2)
				if x >= 240 && x < 1040 && y < 300 {
					want = rgbaAt(pair.Asset, x-240+state.Count*2, y)
				}
				if out.NRGBAAt(x, y) != want {
					t.Fatal("ending independent HD crop", state.Count, x, y)
				}
			}
		}
		for _, reject := range []string{"original", "missing", "palette", "native", "count", "hash", "caller", "visibility"} {
			copy(out.Pix, native)
			candidate, bad, record, enabled := pair, s, draw, true
			switch reject {
			case "original":
				enabled = false
			case "missing":
				candidate = newHDGraphicsPair(profile, original, nil, 2)
			case "palette":
				bad.Palette[bad.DAC[2]][0] ^= 1
			case "native":
				bad.Indexed = append([]byte(nil), s.Indexed...)
				bad.Indexed[120] ^= 1
			case "count":
				record.SourceShift = 241
			case "hash":
				record.QualifiedSHA = "unknown"
			case "caller":
				record.Return = 0x12345
			case "visibility":
				record.VisibleRects = []image.Rectangle{image.Rect(0, 0, 640, 400)}
			}
			if candidate.apply(out, bad, record, enabled).Mode == "hd" || sha256.Sum256(out.Pix) != sha256.Sum256(native) {
				t.Fatal("ending fallback wrote output", state.Count, reject)
			}
		}
		copy(out.Pix, native)
		clipped := graphicsOcclude(draw, []hdWindowDraw{{Rect: image.Rect(120, 0, 136, 16), EndStep: 4}}, nil)
		s.Indexed = append([]byte(nil), s.Indexed...)
		s.Indexed[120] ^= 1
		if pair.apply(out, s, clipped, true).PaintedNative != 60000-256 || out.NRGBAAt(240, 0) != rgbaAt(s.Image, 120, 0) {
			t.Fatal("ending later window visibility")
		}
	}
}

func TestEndingShiftAll241StatesAndSourceGuards(t *testing.T) {
	states := endingProofStates(t)
	mem, s, _, original := endingFixtureMemory(t, states[0])
	h := hdGraphicsState{}
	for count, state := range states {
		current := append([]byte(nil), original...)
		for i := 0; i < 96000; i++ {
			current[i] = original[min(i+count, 96000)]
		}
		if fmt.Sprintf("%x", sha256.Sum256(current)) != state.IndicesSHA {
			t.Fatal("all original CPU states", count)
		}
		for bit := 0; bit < 4; bit++ {
			base := 0x100000 + bit*0x10000
			clear(mem[base : base+32000])
			for i, value := range current {
				mem[base+i/8] |= ((value >> uint(bit)) & 1) << uint(7-i%8)
			}
		}
		for y := 0; y < 150; y++ {
			copy(s.Indexed[y*640+120:], current[y*640:y*640+400])
		}
		ret := uint32(0x187f2)
		if count > 0 {
			ret = 0x18822
		}
		binary.LittleEndian.PutUint32(mem[0x60000:], ret)
		before := sha256.Sum256(mem)
		h.observe(count*10+1, 0x185a7, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
		if h.Pending == nil || h.Pending.Record.ExpectedSHA != state.CropSHA {
			t.Fatal("ending consecutive entry", count)
		}
		h.observe(count*10+2, ret, 0x60004, mem, func() (nativeSnapshot, error) { return s, nil })
		if len(h.Draws) != 1 || h.Draws[0].SourceShift != count || sha256.Sum256(mem) != before {
			t.Fatal("ending consecutive natural return", count)
		}
	}
	h.observe(3000, 0x185a7, 0x60000, mem, nil)
	if h.Pending != nil || h.Shift != nil || len(h.Draws) != 0 {
		t.Fatal("ending exceeded 240")
	}
	for _, reject := range []string{"source_before", "dimensions", "pointer_before", "source_after", "pointer_after", "native_after", "no_snapshot", "skip_initial", "skip_shift"} {
		mem, s, _, original = endingFixtureMemory(t, states[0])
		h = hdGraphicsState{}
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		switch reject {
		case "source_before":
			mem[0x100000] ^= 1
		case "dimensions":
			binary.LittleEndian.PutUint32(mem[0x5b1b4:], 399)
		case "pointer_before":
			binary.LittleEndian.PutUint32(mem[0x38c00:], 0xfffffff0)
		case "skip_initial":
			binary.LittleEndian.PutUint32(mem[0x60000:], 0x18822)
		case "skip_shift":
			mem, s, _, original = endingFixtureMemory(t, states[120])
			h.Shift = &openingShiftState{Name: "END-2.OZM", Bases: [4]uint32{0x100000, 0x110000, 0x120000, 0x130000}, Original: original, Count: 0}
		}
		h.observe(1, 0x185a7, 0x60000, mem, snapshot)
		if reject == "source_before" || reject == "dimensions" || reject == "pointer_before" || reject == "skip_initial" || reject == "skip_shift" {
			if h.Pending != nil || h.Shift != nil {
				t.Fatal("ending rejected unknown source", reject)
			}
			continue
		}
		switch reject {
		case "source_after":
			mem[0x100000+31000] ^= 1
		case "pointer_after":
			mem[0x38c00] ^= 1
		case "native_after":
			s.Indexed[120] ^= 1
		case "no_snapshot":
			snapshot = nil
		}
		h.observe(2, 0x187f2, 0x60004, mem, snapshot)
		if h.Pending != nil || h.Shift != nil || len(h.Draws) != 0 {
			t.Fatal("ending rejected bad return", reject)
		}
	}
}
