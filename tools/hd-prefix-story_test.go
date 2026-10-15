package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

// 已知 bytes 來自實際擷取；high 是 synthetic test input，絕非原版尾端證據。
func prefixStoryFixture(t *testing.T, name string, high byte) ([]byte, nativeSnapshot) {
	t.Helper()
	owner := "B-6.OZM"
	if name == "E-4.OZM" {
		owner = name
	}
	root := "/project/workplace/hd-prototype/catalog/remaining/source-boundary/art-prefix/"
	mem := make([]byte, 2<<20)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 40)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 150)
	indexed := make([]byte, 48000)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		data, err := os.ReadFile(root + fmt.Sprintf("%s-plane-%d-known.bin", owner, bit))
		if err != nil || len(data) != (map[bool]int{false: 6000, true: 5999}[bit == 3]) {
			t.Fatal("actual known-plane input", err)
		}
		if prefixStorySHA(data) != hdPrefixProfiles[name].Planes[bit] {
			t.Fatal("known-plane hash")
		}
		if bit == 3 {
			data = append(data, high)
		}
		base := uint32(0x100000 + bit*0x20000)
		binary.LittleEndian.PutUint32(mem[global:], base)
		binary.LittleEndian.PutUint32(mem[0x386d4+bit*4:], base)
		for y := 0; y < 150; y++ {
			copy(mem[int(base)+y*80:int(base)+y*80+40], data[y*40:y*40+40])
		}
		for i, value := range data {
			for x := 0; x < 8; x++ {
				indexed[i*8+x] |= ((value >> uint(7-x)) & 1) << uint(bit)
			}
		}
	}
	s := nativeSnapshot{Indexed: make([]byte, 256000), Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), MaxColor: 15}
	palette := graphicsNativePalette(hdGraphicsProfile{})
	for i, c := range palette {
		s.DAC[i] = byte(i)
		s.Palette[i] = [3]byte{c.R, c.G, c.B}
	}
	for y := 0; y < 150; y++ {
		copy(s.Indexed[(y+50)*640+160:(y+50)*640+480], indexed[y*320:(y+1)*320])
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			rgb := s.Palette[s.DAC[s.Indexed[y*640+x]]]
			s.Image.SetNRGBA(x, y, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
		}
	}
	binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
	binary.LittleEndian.PutUint32(mem[0x60004:], 0x61000)
	copy(mem[0x61000:], append([]byte(name), 0))
	return mem, s
}

func prefixStoryLoad(t *testing.T, name string, mem []byte, s nativeSnapshot) hdLive {
	t.Helper()
	h := hdLive{}
	before := sha256.Sum256(mem)
	h.observe(1, 0x1bf44, 0x60000, mem)
	if h.Validated || h.Active {
		t.Fatal("qualified before loader return")
	}
	h.observe(2, 0x12345, 0x60004, mem)
	if !h.Validated || h.LastName != name || sha256.Sum256(mem) != before {
		t.Fatal("loader known-region validation or guest write", name)
	}
	return h
}

func prefixStoryDrawArgs(mem []byte) {
	for i, value := range []uint32{0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(mem[0x60004+i*4:], value)
	}
}

func TestPrefixStoryAllUnknownBytesAndNaturalConsumers(t *testing.T) {
	for _, name := range []string{"B-6.OZM", "E-4.OZM"} {
		mem, s := prefixStoryFixture(t, name, 0)
		for value := 0; value < 256; value++ {
			mem[0x160000+149*80+39] = byte(value)
			for x := 0; x < 8; x++ {
				s.Indexed[199*640+472+x] = 7 | (((byte(value) >> uint(7-x)) & 1) << 3)
			}
			binary.LittleEndian.PutUint32(mem[0x60004:], 0x61000)
			h := prefixStoryLoad(t, name, mem, s)
			prefixStoryDrawArgs(mem)
			for _, entry := range []uint32{0x1c1bc, 0x11dcb} {
				h.Active = false
				before := sha256.Sum256(mem)
				h.observe(3, entry, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
				if h.Pending == nil || h.Active {
					t.Fatal("entry qualification", name, value, entry)
				}
				h.observe(4, 0x12345, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
				if h.Active || h.Pending == nil {
					t.Fatal("wrong return stack")
				}
				h.observe(5, 0x12345, 0x60004, mem, func() (nativeSnapshot, error) { return s, nil })
				if !h.Active || h.Pending != nil || sha256.Sum256(mem) != before {
					t.Fatal("unknown byte fixed or natural guard failed", name, value, entry)
				}
			}
		}
	}
}

func TestPrefixStoryFourNamesKnownRegionAndNativePreservation(t *testing.T) {
	if len(hdPrefixProfiles) != 4 {
		t.Fatal("four partial source names")
	}
	for name, profile := range hdPrefixProfiles {
		mem, s := prefixStoryFixture(t, name, 0xa5)
		current, ok := prefixStorySource(name, mem, true)
		if !ok || !bytes.Equal(current.Indexed, graphicsCrop(s.Indexed, 640, a0Rect)) {
			t.Fatal("actual planes and native indices")
		}
		data, err := os.ReadFile(profile.Source)
		if err != nil {
			t.Fatal(err)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		pair := newHDPrefixPair(profile, data, asset)
		out, _, err := previewA0(s.Image, nil, nil, 2, false)
		if err != nil {
			t.Fatal(err)
		}
		original := append([]byte(nil), out.Pix...)
		if pair.apply(out, s, name, true) != "hd" {
			t.Fatal("known-region presentation", name, pair.Reason)
		}
		for y := 0; y < 800; y++ {
			for x := 0; x < 1280; x++ {
				want := rgbaAt(s.Image, x/2, y/2)
				if x >= 320 && x < 960 && y >= 100 && y < 400 && !image.Pt((x-320)/2, (y-100)/2).In(image.Rect(312, 149, 320, 150)) {
					want = rgbaAt(pair.Asset, x-320, y-100)
				}
				if out.NRGBAAt(x, y) != want {
					t.Fatal("HD formula, unknown pixel or outside rectangle", name, x, y)
				}
			}
		}
		for _, reject := range []string{"disabled", "missing", "known_native", "palette", "wrong_source_name"} {
			copy(out.Pix, original)
			candidate, bad, resource, enabled := pair, s, name, true
			switch reject {
			case "disabled":
				enabled = false
			case "missing":
				candidate = newHDPrefixPair(profile, data, nil)
			case "known_native":
				bad.Indexed = append([]byte(nil), s.Indexed...)
				bad.Indexed[50*640+160] ^= 1
			case "palette":
				bad.Palette[bad.DAC[2]][0] ^= 1
			case "wrong_source_name":
				resource = "unknown.OZM"
			}
			if candidate.apply(out, bad, resource, enabled) == "hd" || sha256.Sum256(out.Pix) != sha256.Sum256(original) {
				t.Fatal("fallback changed output", name, reject)
			}
		}
		data[0] ^= 1
		copy(out.Pix, original)
		if pair.apply(out, s, name, true) != "hd" {
			t.Fatal("pair retained mutable source input")
		}
	}
}

func TestPrefixStoryReturnSourceAndInputGuards(t *testing.T) {
	for _, entry := range []uint32{0x1c1bc, 0x11dcb} {
		for _, reject := range []string{"known_before", "alias_before", "known_after", "unknown_after", "base_after", "native_known", "native_unknown", "alias_after", "no_snapshot"} {
			if (reject == "alias_before" || reject == "alias_after") && entry != 0x11dcb {
				continue
			}
			mem, s := prefixStoryFixture(t, "B-6.OZM", 0xa5)
			h := prefixStoryLoad(t, "B-6.OZM", mem, s)
			prefixStoryDrawArgs(mem)
			if reject == "known_before" {
				mem[0x100000] ^= 1
			}
			if reject == "alias_before" {
				mem[0x386d4] ^= 1
			}
			snapshot := func() (nativeSnapshot, error) { return s, nil }
			h.observe(3, entry, 0x60000, mem, snapshot)
			if reject == "known_before" || reject == "alias_before" {
				if h.Pending != nil {
					t.Fatal("unmatched entry accepted", reject)
				}
				continue
			}
			switch reject {
			case "known_after":
				mem[0x100000] ^= 1
			case "unknown_after":
				mem[0x160000+149*80+39] ^= 1
				s.Indexed[199*640+479] ^= 8
			case "base_after":
				copy(mem[0x170000:0x170000+12000], mem[0x100000:0x100000+12000])
				binary.LittleEndian.PutUint32(mem[0x38c00:], 0x170000)
				binary.LittleEndian.PutUint32(mem[0x386d4:], 0x170000)
			case "native_known":
				s.Indexed[50*640+160] ^= 1
			case "native_unknown":
				s.Indexed[199*640+479] ^= 8
			case "alias_after":
				mem[0x386d4] ^= 1
			case "no_snapshot":
				snapshot = nil
			}
			h.observe(4, 0x12345, 0x60004, mem, snapshot)
			if h.Active {
				t.Fatal("failed source/copy/guard retained qualification", entry, reject)
			}
		}
	}
	for _, argument := range []int{0, 1, 2, 3, 4} {
		mem, s := prefixStoryFixture(t, "E-4.OZM", 0)
		h := prefixStoryLoad(t, "E-4.OZM", mem, s)
		prefixStoryDrawArgs(mem)
		mem[0x60004+argument*4] ^= 1
		h.observe(3, 0x1c1bc, 0x60000, mem, func() (nativeSnapshot, error) { return s, nil })
		if h.Pending != nil || h.Active {
			t.Fatal("wrong story argument")
		}
	}
}
