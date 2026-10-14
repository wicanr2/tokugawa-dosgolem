package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"testing"
)

func terrainMemory(t *testing.T) []byte {
	t.Helper()
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx")
	if err != nil {
		t.Skip("local original atlas unavailable")
	}
	if len(idx) != 640*400 {
		t.Fatal("atlas shape")
	}
	mem := make([]byte, 0x120000)
	for bit, global := range []int{0x42174, 0x42168, 0x42170, 0x4216c} {
		base := 0x100000 + bit*0x8000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		for n, index := range idx {
			if index&(1<<bit) != 0 {
				mem[base+n/8] |= 1 << uint(7-n%8)
			}
		}
	}
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe2a)
	return mem
}

func TestTerrainOriginalPlanesReturnsAndInvalidation(t *testing.T) {
	mem := terrainMemory(t)
	before := sha256.Sum256(mem)
	var state hdTerrainState
	state.observe(1, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	if state.Pending == nil || len(state.Draws) != 0 {
		t.Fatal("early activation")
	}
	state.observe(2, 0x1fe2a, 0x1000, 0, 0, mem)
	if state.Pending == nil {
		t.Fatal("wrong stack completed")
	}
	state.observe(3, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 1 || state.Draws[0].Index != 0 || state.Draws[0].Rect != image.Rect(64, 56, 112, 88) {
		t.Fatal("natural floor draw")
	}
	state.observe(4, 0x1fc07, 0x1000, 6, 56*80+9, mem)
	if len(state.Draws) != 0 {
		t.Fatal("overlap not invalidated before return")
	}
	state.observe(5, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 1 || state.Draws[0].Index != 1 {
		t.Fatal("replacement")
	}
	state.observe(6, 0x1c084, 0, 0, 0, mem)
	if len(state.Draws) != 0 || state.Pending != nil {
		t.Fatal("full frame retained stale ground")
	}
	if before != sha256.Sum256(mem) {
		t.Fatal("observer wrote original RAM")
	}
	state.observe(7, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	mem[0x100000] ^= 1
	state.observe(8, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 0 {
		t.Fatal("changed plane accepted at return")
	}
	mem[0x100000] ^= 1
	binary.LittleEndian.PutUint32(mem[0x35984:], 1)
	binary.LittleEndian.PutUint32(mem[0x4217c:], 1)
	state.observe(9, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	if state.Pending != nil {
		t.Fatal("original checker mask accepted")
	}
	for _, pair := range [][2]uint32{{78, 0}, {256 * 80, 0}, {1, 0}, {80, 0}, {0, 75}, {0, 399 * 80}, {0xffffffff, 0}, {0, 0xffffffff}} {
		if _, ok := hdTerrainDecode(pair[0], pair[1]); ok {
			t.Fatal("invalid geometry", pair)
		}
	}
}

func TestTerrainPreparedSourcesAndFullCellProtection(t *testing.T) {
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/terrain/prepared/profiles.json")
	if err != nil {
		t.Skip("local floor fixtures unavailable")
	}
	var manifest struct {
		ToolSHA string `json:"tool_sha256"`
		Records []struct {
			Index int    `json:"index"`
			SHA   string `json:"source_png_sha256"`
		} `json:"records"`
	}
	if err = json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	tool, err := os.ReadFile("/project/tools/hd-terrain-prepare.py")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(tool)) != manifest.ToolSHA || len(manifest.Records) != 104 || len(hdTerrainProfiles) != 104 {
		t.Fatal("stale floor preparation")
	}
	approved := 0
	for _, r := range manifest.Records {
		t.Run(fmt.Sprint(r.Index), func(t *testing.T) {
			p := hdTerrainProfiles[r.Index]
			path := fmt.Sprintf("/project/workplace/hd-prototype/catalog/terrain/prepared/sources/floor-%03d-original.png", r.Index)
			data, err := os.ReadFile(path)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.SourceSHA || p.SourceSHA != r.SHA {
				t.Fatal("source fingerprint")
			}
			source, err := previewPNG(path)
			if err != nil {
				t.Fatal(err)
			}
			native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
			rect := image.Rect(80, 80, 128, 112)
			for y := 0; y < 32; y++ {
				for x := 0; x < 48; x++ {
					native.SetNRGBA(80+x, 80+y, rgbaAt(source, x, y))
				}
			}
			before := append([]byte(nil), native.Pix...)
			out, _, _ := previewA0(native, nil, nil, 2, false)
			original := append([]byte(nil), out.Pix...)
			missing := newHDTerrainPair(source, nil, 2)
			if missing.apply(out, native, rect, 2, true) != "fallback_missing_asset" || !bytes.Equal(original, out.Pix) {
				t.Fatal("missing floor changed presentation")
			}
			if p.Asset == "" {
				return
			}
			approved++
			path = "/project/workplace/hd-prototype/catalog/terrain/art/" + p.Asset
			data, err = os.ReadFile(path)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.AssetSHA {
				t.Fatal("HD fingerprint")
			}
			asset, err := previewPNG(path)
			if err != nil {
				t.Fatal(err)
			}
			pair := newHDTerrainPair(source, asset, 2)
			if pair.apply(out, native, rect, 2, true) != "hd" || bytes.Equal(original, out.Pix) || !bytes.Equal(before, native.Pix) {
				t.Fatal("valid HD floor or original immutability")
			}
			for y := 0; y < 800; y++ {
				for x := 0; x < 1280; x++ {
					if x >= 160 && x < 256 && y >= 160 && y < 224 {
						continue
					}
					i := (y*1280 + x) * 4
					if !bytes.Equal(out.Pix[i:i+4], original[i:i+4]) {
						t.Fatal("outside floor changed")
					}
				}
			}
			native.Pix[(90*640+90)*4] ^= 1
			out, _, _ = previewA0(native, nil, nil, 2, false)
			occluded := append([]byte(nil), out.Pix...)
			if pair.apply(out, native, rect, 2, true) != "fallback_source_mismatch" || !bytes.Equal(occluded, out.Pix) {
				t.Fatal("occluded floor did not keep entire original cell")
			}
		})
	}
	if approved != 8 {
		t.Fatal("reviewed source count", approved)
	}
}
