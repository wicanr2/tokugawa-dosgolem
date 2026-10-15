package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

type openingPartialFixture struct {
	Source     string      `json:"source"`
	Sample     int         `json:"sample_id"`
	Entry      uint32      `json:"entry_linear"`
	SourceRect [4]int      `json:"source_rect"`
	DestRect   [4]int      `json:"dest_rect"`
	Palette    [16][3]byte `json:"active_palette"`
}

func openingPartialFixtures(t *testing.T) []openingPartialFixture {
	t.Helper()
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/remaining/history/partial/source-verification.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof struct {
		Verified bool
		Records  []openingPartialFixture
	}
	if err = json.Unmarshal(data, &proof); err != nil || !proof.Verified || len(proof.Records) != 14 {
		t.Fatal("fourteen original partial-copy fixtures", err)
	}
	return proof.Records
}

func openingPartialMemory(t *testing.T, fixture openingPartialFixture) ([]byte, nativeSnapshot) {
	t.Helper()
	root := "/project/workplace/hd-prototype/catalog/remaining/history/probe/opening-samples/"
	data, err := os.ReadFile(root + fmtSample(fixture.Sample) + "-before.planes.bin")
	if err != nil || len(data) != 128000 {
		t.Fatal("original planes", err)
	}
	native, err := os.ReadFile(root + fmtSample(fixture.Sample) + "-after.idx")
	if err != nil || len(native) != 256000 {
		t.Fatal("original native frame", err)
	}
	width, height := 640, 400
	if fixture.Source == "OP2_P.OZM" {
		width, height = 384, 119
	}
	if fixture.Entry == 0x11dcb {
		width, height = 320, 150
	}
	mem := make([]byte, 2<<20)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], uint32(width/8))
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], uint32(height))
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := uint32(0x100000 + bit*0x20000)
		binary.LittleEndian.PutUint32(mem[global:], base)
		binary.LittleEndian.PutUint32(mem[0x386d4+bit*4:], base)
		copy(mem[base:base+32000], data[bit*32000:(bit+1)*32000])
	}
	binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
	offset := fixture.SourceRect[1]*80 + fixture.SourceRect[0]/8
	binary.LittleEndian.PutUint32(mem[0x60004:], uint32(offset))
	s := nativeSnapshot{Indexed: native, Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), MaxColor: 15}
	for i, rgb := range fixture.Palette {
		s.DAC[i], s.Palette[i] = byte(i), rgb
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			rgb := fixture.Palette[native[y*640+x]]
			s.Image.SetNRGBA(x, y, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
		}
	}
	return mem, s
}

func fmtSample(number int) string {
	return fmt.Sprintf("sample-%03d", number)
}

func TestOpeningPartialOriginalFixturesAndPresentation(t *testing.T) {
	for _, fixture := range openingPartialFixtures(t) {
		mem, s := openingPartialMemory(t, fixture)
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		before := sha256.Sum256(mem)
		if fixture.Entry == 0x11dcb {
			h := hdLive{LastName: fixture.Source, Validated: true}
			h.observe(1, fixture.Entry, 0x60000, mem, snapshot)
			if h.Pending == nil || h.Active {
				t.Fatal("story accepted before return", fixture.Source)
			}
			h.observe(2, 0x12345, 0x60000, mem, snapshot)
			if h.Active || h.Pending == nil {
				t.Fatal("wrong story return stack")
			}
			h.observe(3, 0x12345, 0x60004, mem, snapshot)
			if !h.Active || sha256.Sum256(mem) != before {
				t.Fatal("actual story source or guest mutation", fixture.Source)
			}
			continue
		}
		h := hdGraphicsState{}
		h.observe(1, fixture.Entry, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 {
			t.Fatal("partial accepted before return", fixture.Source)
		}
		h.observe(2, 0x12345, 0x60004, mem, snapshot)
		if len(h.Draws) != 1 || sha256.Sum256(mem) != before {
			t.Fatal("actual partial source or guest mutation", fixture.Source)
		}
		draw := h.Draws[0]
		src, dst := fixture.SourceRect, fixture.DestRect
		if draw.SourceRect != image.Rect(src[0], src[1], src[2], src[3]) || draw.DestRect != image.Rect(dst[0], dst[1], dst[2], dst[3]) || !draw.Verified {
			t.Fatal("original source or destination geometry", fixture.Sample, draw)
		}
		var profile hdGraphicsProfile
		for _, p := range hdGraphicsProfiles {
			if p.Source.Name == fixture.Source {
				profile = p
			}
		}
		indices, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + fixture.Source + ".idx")
		if err != nil {
			t.Fatal(err)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDGraphicsPair(profile, indices, asset, 2)
		outImage, _ := prepareHDImage(s.Image, 640, 400, 2)
		out := outImage.(*image.NRGBA)
		original := append([]byte(nil), out.Pix...)
		report := pair.apply(out, s, draw, true)
		if report.Mode != "hd" || report.PaintedNative != draw.DestRect.Dx()*draw.DestRect.Dy() {
			t.Fatal("actual palette, crop or paint count", fixture.Sample, report)
		}
		for y := 0; y < 800; y++ {
			for x := 0; x < 1280; x++ {
				inside := image.Pt(x/2, y/2).In(draw.DestRect)
				if inside {
					sx, sy := x-dst[0]*2+src[0]*2, y-dst[1]*2+src[1]*2
					if out.NRGBAAt(x, y) != rgbaAt(pair.Asset, sx, sy) {
						t.Fatal("HD source crop", fixture.Sample, x, y)
					}
				} else {
					i := out.PixOffset(x, y)
					for c := 0; c < 4; c++ {
						if out.Pix[i+c] != original[i+c] {
							t.Fatal("outside destination", fixture.Sample, x, y)
						}
					}
				}
			}
		}
		for _, reject := range []string{"missing", "palette", "native"} {
			copy(out.Pix, original)
			bad := s
			candidate := pair
			if reject == "missing" {
				candidate = newHDGraphicsPair(profile, indices, nil, 2)
			}
			if reject == "palette" {
				bad.Palette[bad.DAC[2]][0] ^= 1
			}
			if reject == "native" {
				bad.Indexed = append([]byte(nil), s.Indexed...)
				bad.Indexed[dst[1]*640+dst[0]] ^= 1
			}
			if candidate.apply(out, bad, draw, true).Mode == "hd" || sha256.Sum256(out.Pix) != sha256.Sum256(original) {
				t.Fatal("invalid presentation changed output", fixture.Sample, reject)
			}
		}
	}
}

func TestOpeningPartialAllDeclaredOffsetsAndGuards(t *testing.T) {
	for entry, offsets := range map[uint32][]uint32{0x11e37: {0, 16, 32}, 0x11ba6: {0, 26, 52, 16000, 16026}, 0x11c16: {16052, 18212, 20372, 22532}} {
		mem := make([]byte, 32)
		for _, offset := range offsets {
			binary.LittleEndian.PutUint32(mem[4:], offset)
			args, ok := openingPartialArgs(entry, 0, mem)
			if !ok || args[0] != offset {
				t.Fatal("declared original offset", entry, offset)
			}
		}
		for _, bad := range []uint32{1, 15, 0xffffffff} {
			binary.LittleEndian.PutUint32(mem[4:], bad)
			if _, ok := openingPartialArgs(entry, 0, mem); ok {
				t.Fatal("unknown offset accepted")
			}
		}
		if _, ok := openingPartialArgs(entry, 0xfffffff0, mem); ok {
			t.Fatal("overflowing stack accepted")
		}
	}
	fixture := openingPartialFixtures(t)[0]
	for _, reject := range []string{"alias_before", "source_before", "alias_after", "source_after", "native_after"} {
		mem, s := openingPartialMemory(t, fixture)
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		if reject == "alias_before" {
			mem[0x386d4] ^= 1
		}
		if reject == "source_before" {
			mem[0x100000] ^= 1
		}
		h := hdGraphicsState{}
		h.observe(1, fixture.Entry, 0x60000, mem, snapshot)
		if reject == "alias_before" || reject == "source_before" {
			if h.Pending != nil {
				t.Fatal("unknown entry source accepted", reject)
			}
			continue
		}
		h.observe(2, 0x12345, 0x60000, mem, snapshot)
		if h.Pending == nil || len(h.Draws) != 0 {
			t.Fatal("wrong return stack accepted")
		}
		if reject == "alias_after" {
			mem[0x386d4] ^= 1
		}
		if reject == "source_after" {
			mem[0x100000] ^= 1
		}
		if reject == "native_after" {
			s.Indexed[50*640+248] ^= 1
		}
		h.observe(3, 0x12345, 0x60004, mem, snapshot)
		if len(h.Draws) != 0 {
			t.Fatal("unknown returned source accepted", reject)
		}
	}
	mem, s := openingPartialMemory(t, fixture)
	binary.LittleEndian.PutUint32(mem[0x60004:], 1)
	h := hdGraphicsState{Draws: []graphicsDrawRecord{{Name: "old", DestRect: image.Rect(0, 0, 640, 400)}}}
	h.observe(1, fixture.Entry, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
	area := 0
	for _, d := range h.Draws {
		for _, r := range d.VisibleRects {
			area += r.Dx() * r.Dy()
			if r.Overlaps(openingPartialDestination(fixture.Entry)) {
				t.Fatal("invalid new draw resurrected old pixels")
			}
		}
	}
	if h.Pending != nil || area != 640*400-128*119 {
		t.Fatal("unknown partial destination occlusion")
	}
}

func TestOpeningStorySourceAndNativeGuards(t *testing.T) {
	fixture := openingPartialFixtures(t)[3]
	for _, reject := range []string{"alias_before", "source_before", "alias_after", "source_after", "native_after", "no_snapshot"} {
		mem, s := openingPartialMemory(t, fixture)
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		if reject == "alias_before" {
			mem[0x386d4] ^= 1
		}
		if reject == "source_before" {
			mem[0x100000] ^= 1
		}
		h := hdLive{LastName: fixture.Source, Validated: true}
		h.observe(1, 0x11dcb, 0x60000, mem, snapshot)
		if reject == "alias_before" || reject == "source_before" {
			if h.Pending != nil {
				t.Fatal("unknown story entry accepted")
			}
			continue
		}
		if reject == "alias_after" {
			mem[0x386d4] ^= 1
		}
		if reject == "source_after" {
			mem[0x100000] ^= 1
		}
		if reject == "native_after" {
			s.Indexed[50*640+160] ^= 1
		}
		if reject == "no_snapshot" {
			snapshot = nil
		}
		h.observe(2, 0x12345, 0x60004, mem, snapshot)
		if h.Active {
			t.Fatal("unknown story return accepted", reject)
		}
	}
}
