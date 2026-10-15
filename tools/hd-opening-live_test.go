package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
)

func openingSourceFixture(t *testing.T, name string) ([]byte, graphicsSourceProfile, []byte) {
	t.Helper()
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + name + ".idx")
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 640*400 {
		t.Fatal("full source geometry")
	}
	p := graphicsSourceProfile{Name: name, Width: 640, Height: 400, Planes: hdScrollSourcePlanes}
	if name != "VISUAL.OZM" {
		p = hdGraphicsOpeningProfiles[0].Source
	}
	mem := make([]byte, 2<<20)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 80)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 400)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x20000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		binary.LittleEndian.PutUint32(mem[0x386d4+bit*4:], uint32(base))
		for y := 0; y < 400; y++ {
			for x := 0; x < 640; x++ {
				mem[base+y*80+x/8] |= ((idx[y*640+x] >> uint(bit)) & 1) << uint(7-x%8)
			}
		}
		if fmt.Sprintf("%x", sha256.Sum256(mem[base:base+32000])) != p.Planes[bit] {
			t.Fatal("actual source plane differs from profile", name, bit)
		}
	}
	binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
	return mem, p, idx
}

func TestOpeningCompleteCopyNaturalReturnAndSourceGuards(t *testing.T) {
	for _, reject := range []string{"", "alias_before", "alias_after", "tail_before", "source_after", "native_after"} {
		mem, profile, indices := openingSourceFixture(t, "OP-FULL.OZM")
		trace := graphicsTrace{Profiles: []graphicsSourceProfile{profile}}
		snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: indices}, nil }
		if reject == "alias_before" {
			mem[0x386d4] ^= 1
		}
		if reject == "tail_before" {
			mem[0x160000+31999] ^= 1
		}
		before := sha256.Sum256(mem)
		trace.observe(1, 0x11f14, 0x60000, mem, snapshot)
		if sha256.Sum256(mem) != before {
			t.Fatal("entry changed RAM")
		}
		if reject == "alias_before" || reject == "tail_before" {
			if trace.Pending != nil || len(trace.Records) != 1 || trace.Records[0].Verified || trace.Records[0].Error == "" {
				t.Fatal("invalid entry source accepted", reject)
			}
			continue
		}
		trace.observe(2, 0x12345, 0x60000, mem, snapshot)
		if trace.Pending == nil || len(trace.Records) != 0 {
			t.Fatal("incorrect return stack accepted")
		}
		switch reject {
		case "alias_after":
			mem[0x386d4] ^= 1
		case "source_after":
			mem[0x100000] ^= 1
		case "native_after":
			indices = append([]byte(nil), indices...)
			indices[0] ^= 1
		}
		before = sha256.Sum256(mem)
		trace.observe(3, 0x12345, 0x60004, mem, snapshot)
		if len(trace.Records) != 1 || trace.Records[0].Verified != (reject == "") || sha256.Sum256(mem) != before {
			t.Fatal("complete-copy qualification or RAM mutation", reject, trace.Records)
		}
		if reject == "" {
			r := trace.Records[0]
			if r.Kind != "opening_full" || r.Args != [5]uint32{0, 80, 400, 80, 0xa0000} || r.EndStep != 3 || r.ExpectedSHA != hdGraphicsOpeningProfiles[0].IndicesSHA {
				t.Fatal("complete-copy provenance or source")
			}
		}
	}
	if openingPlaneAlias(make([]byte, 3)) {
		t.Fatal("short pointer array accepted")
	}
}

func TestOpeningCompleteCopyRevokesOldLayers(t *testing.T) {
	mem, _, indices := openingSourceFixture(t, "OP-FULL.OZM")
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: indices}, nil }
	h := hdLive{}
	h.Graphics.Draws = []graphicsDrawRecord{{Name: "old"}}
	h.Window.Draws = []hdWindowDraw{{}}
	h.Kao.Draws = []hdKaoDraw{{}}
	h.observe(1, 0x11f14, 0x60000, mem, snapshot)
	if len(h.Graphics.Draws) != 0 || len(h.Window.Draws) != 0 || len(h.Kao.Draws) != 0 {
		t.Fatal("old layers remained during complete copy")
	}
	h.observe(2, 0x12345, 0x60004, mem, snapshot)
	if len(h.Graphics.Draws) != 1 || h.Graphics.Draws[0].Name != "OP-FULL.OZM" {
		t.Fatal("new exact opening source not retained")
	}
}

func TestOpeningScrollUsesActualCompleteSource(t *testing.T) {
	mem, _, indices := openingSourceFixture(t, "VISUAL.OZM")
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: indices}, nil }
	before := sha256.Sum256(mem)
	h := hdLive{ScrollValidated: true, LastName: "VISUAL.OZM"}
	h.observe(1, 0x11f14, 0x60000, mem, snapshot)
	if h.ScrollPending == nil || h.ScrollActive {
		t.Fatal("scroll accepted before return")
	}
	h.observe(2, 0x12345, 0x60004, mem, snapshot)
	if !h.ScrollActive || sha256.Sum256(mem) != before {
		t.Fatal("exact scroll source or guest mutation")
	}
	indices[0] ^= 1
	if openingScrollMatches(mem, snapshot) || openingScrollMatches(mem, nil) {
		t.Fatal("unknown native scroll or absent snapshot accepted")
	}
	indices[0] ^= 1
	mem[0x386d4] ^= 1
	if openingScrollMatches(mem, snapshot) {
		t.Fatal("different scroll source pointers accepted")
	}
	mem[0x386d4] ^= 1
	mem[0x100000] ^= 1
	if openingScrollMatches(mem, snapshot) {
		t.Fatal("different scroll source accepted")
	}
}
