package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
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
