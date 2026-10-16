package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"testing"
)

func endingStoryFixture(t *testing.T, name string, ret uint32) ([]byte, nativeSnapshot, []byte) {
	t.Helper()
	root := "/project/workplace/hd-prototype/catalog/remaining/consumers/battle/r25/ready/native/"
	data, err := os.ReadFile(root + "receipt.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Stages []struct {
			Label   string
			Path    string `json:"indices_path"`
			Palette [256][3]byte
			DAC     [16]byte
		} `json:"ending_validation_stages"`
	}
	if json.Unmarshal(data, &receipt) != nil {
		t.Fatal("controlled original fixtures")
	}
	var s nativeSnapshot
	for _, stage := range receipt.Stages {
		if len(stage.Label) >= len(name) && stage.Label[:len(name)] == name {
			s.Indexed, err = os.ReadFile(root + stage.Path)
			if err != nil {
				t.Fatal(err)
			}
			s.Palette, s.DAC = stage.Palette, stage.DAC
			break
		}
	}
	if len(s.Indexed) != 256000 {
		t.Fatal("actual original destination")
	}
	s.Image = image.NewNRGBA(image.Rect(0, 0, 640, 400))
	s.MaxColor = 15
	for i, index := range s.Indexed {
		c := s.Palette[s.DAC[index]]
		s.Image.SetNRGBA(i%640, i/640, color.NRGBA{c[0], c[1], c[2], 255})
	}
	source, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + name + ".idx")
	if err != nil || len(source) != 48000 {
		t.Fatal(err)
	}
	if !bytes.Equal(graphicsCrop(s.Indexed, 640, endingStoryRect), source) {
		t.Fatal("independent original crop")
	}
	// 明示合成 backing RAM，只測入口／返回 guard；上方目的取自原始 CPU 執行。
	mem := make([]byte, 2<<20)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x20000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		for y := 0; y < 150; y++ {
			for x := 0; x < 320; x++ {
				mem[base+y*80+x/8] |= ((source[y*320+x] >> uint(bit)) & 1) << uint(7-x%8)
			}
		}
	}
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 40)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 150)
	for i, value := range []uint32{ret, 0, 40, 150, 80, 0xa0502, 0xa0502, 0xa0502, 0xa0502} {
		binary.LittleEndian.PutUint32(mem[0x60000+i*4:], value)
	}
	return mem, s, source
}

func TestEndingStoryActualDestinationAndGuards(t *testing.T) {
	for _, ret := range []uint32{0x18a07, 0x18a6e, 0x18ad5, 0x18b3c, 0x18ba3, 0x18c19, 0x18c8f} {
		name, entry := endingStoryCaller(ret)
		mem, s, _ := endingStoryFixture(t, name, ret)
		h := hdLive{LastName: name, Validated: true}
		snapshot := func() (nativeSnapshot, error) { return s, nil }
		before := sha256.Sum256(mem)
		h.observe(1, entry, 0x60000, mem, snapshot)
		if h.Pending == nil || h.Active || h.Pending.Destination != endingStoryRect {
			t.Fatal("ending entry", name)
		}
		h.observe(2, ret, 0x60000, mem, snapshot)
		if h.Active || h.Pending == nil {
			t.Fatal("wrong ending return stack")
		}
		h.observe(3, ret, 0x60004, mem, snapshot)
		if !h.Active || h.storyDestination() != endingStoryRect || h.Pending != nil || sha256.Sum256(mem) != before {
			t.Fatal("ending return or guest changed", name)
		}
		profile := hdPortraitProfiles[name]
		source, err := previewPNG(profile.Source)
		if err != nil {
			t.Fatal(err)
		}
		asset, err := previewPNG("/project/assets/hd-b/art/" + profile.Asset)
		if err != nil {
			t.Fatal(err)
		}
		prepared, reason := prepareHDPortrait(asset)
		if reason != "" {
			t.Fatal(reason)
		}
		out, mode, err := previewPortraitAt(s.Image, source, prepared, h.storyDestination(), 2, true)
		if err != nil || mode != "hd" {
			t.Fatal("ending positioned art", name, mode, err)
		}
		preservePortraitLabelsAt(out, s.Image, name, endingStoryRect, 2)
		for y := 0; y < 800; y++ {
			for x := 0; x < 1280; x++ {
				want := rgbaAt(s.Image, x/2, y/2)
				if image.Pt(x/2, y/2).In(endingStoryRect) {
					want = rgbaAt(prepared, x-32, y-32)
					for _, label := range portraitLabelRects[name] {
						if image.Pt(x/2-16, y/2-16).In(label) {
							want = rgbaAt(s.Image, x/2, y/2)
						}
					}
				}
				if out.NRGBAAt(x, y) != want {
					t.Fatal("ending art, labels or outside", name, x, y)
				}
			}
		}
		missing, mode, err := previewPortraitAt(s.Image, source, nil, endingStoryRect, 2, true)
		if err != nil || mode != "fallback_missing_asset" || missing.NRGBAAt(32, 32) != rgbaAt(s.Image, 16, 16) {
			t.Fatal("ending missing asset", name)
		}
		for _, reject := range []string{"caller", "destination", "source_before", "source_after", "pointer_after", "native_after", "name_after", "no_snapshot"} {
			mem, s, _ = endingStoryFixture(t, name, ret)
			h = hdLive{LastName: name, Validated: true}
			snapshot = func() (nativeSnapshot, error) { return s, nil }
			switch reject {
			case "caller":
				binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
			case "destination":
				offset := 0x60014
				if entry == 0x1c10b {
					offset = 0x6001c
				}
				binary.LittleEndian.PutUint32(mem[offset:], 0x100000)
			case "source_before":
				mem[0x100000] ^= 1
			}
			h.observe(1, entry, 0x60000, mem, snapshot)
			if reject == "caller" || reject == "destination" || reject == "source_before" {
				if h.Pending != nil || h.Active {
					t.Fatal("unknown ending entry", name, reject)
				}
				continue
			}
			switch reject {
			case "source_after":
				mem[0x100000] ^= 1
			case "pointer_after":
				mem[0x38c00] ^= 1
			case "native_after":
				s.Indexed[16*640+16] ^= 1
			case "name_after":
				h.LastName = "A-7.OZM"
			case "no_snapshot":
				snapshot = nil
			}
			h.observe(2, ret, 0x60004, mem, snapshot)
			if h.Active {
				t.Fatal("unverified ending return", name, reject)
			}
		}
	}
	if endingStoryCall(0x1c10b, 0xfffffff0, make([]byte, 64), "END-9.OZM") {
		t.Fatal("ending stack overflow")
	}
	if _, _, err := previewPortraitAt(image.NewNRGBA(image.Rect(0, 0, 640, 400)), nil, nil, image.Rect(0, 0, 320, 150), 2, true); err == nil {
		t.Fatal("unverified destination accepted")
	}
}

func TestHDLiveDrawReturnAndInvalidation(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(m[int(esp)+4*i:], v)
	}
	before := append([]byte(nil), m...)
	h := hdLive{LastName: "A-0.OZM", Validated: true}
	h.observe(1, 0x1c1bc, esp, m)
	if h.Active || h.Pending == nil {
		t.Fatal("draw entry must wait for return")
	}
	h.observe(2, 0x1803d, esp, m)
	if h.Active {
		t.Fatal("wrong stack activated replacement")
	}
	h.observe(3, 0x1803d, esp+4, m)
	if !h.Active || h.Pending != nil {
		t.Fatal("matching return did not activate")
	}
	if !bytes.Equal(m, before) {
		t.Fatal("guest memory changed")
	}
	h.observe(4, 0x1bf44, uint32(len(m)), m)
	if h.Active || h.Validated {
		t.Fatal("new asset load retained old HD qualification")
	}
}

func TestHDLiveNextPortraitHasItsOwnLifecycle(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(m[int(esp)+4*i:], v)
	}
	h := hdLive{LastName: "A-1.OZM", Validated: true}
	h.observe(1, 0x1c1bc, esp, m)
	if h.Active || h.Pending == nil {
		t.Fatal("next portrait must wait for its original draw return")
	}
	h.observe(2, 0x1803d, esp+4, m)
	if !h.Active || h.LastName != "A-1.OZM" {
		t.Fatal("validated next portrait did not activate")
	}
	h.observe(3, 0x1bf44, uint32(len(m)), m)
	if h.Active || h.Validated || h.LastName != "" {
		t.Fatal("new load kept previous portrait eligibility")
	}
}

func TestHDLiveRejectsOtherDraws(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(m[int(esp)+4*i:], v)
	}
	for i := 0; i < 5; i++ {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			copyRAM := append([]byte(nil), m...)
			binary.LittleEndian.PutUint32(copyRAM[int(esp)+4+4*i:], 123)
			h := hdLive{LastName: "A-0.OZM", Validated: true, Active: true}
			h.observe(1, 0x1c1bc, esp, copyRAM)
			if h.Active || h.Pending != nil {
				t.Fatal("unknown geometry accepted")
			}
		})
	}
	h := hdLive{LastName: "A-9.OZM", Validated: true}
	h.observe(1, 0x1c1bc, esp, m)
	if h.Pending != nil {
		t.Fatal("different resource accepted")
	}
	h = hdLive{LastName: "A-0.OZM", Validated: true}
	h.observe(1, 0x1c1bc, ^uint32(0)-2, m)
	if h.Pending != nil {
		t.Fatal("out-of-bounds stack accepted")
	}
}
