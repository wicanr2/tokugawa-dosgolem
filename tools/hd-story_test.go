package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
)

type storySourceRecord struct {
	Resource string `json:"resource"`
	Asset    string `json:"asset"`
	AssetSHA string `json:"asset_sha256"`
	Columns  uint32 `json:"columns"`
}

func storySources(t *testing.T) []storySourceRecord {
	t.Helper()
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/story-expanded/profiles.json")
	if err != nil {
		t.Skip("local original story sources unavailable")
	}
	var manifest struct {
		ToolSHA string              `json:"tool_sha256"`
		Records []storySourceRecord `json:"records"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	tool, err := os.ReadFile("/project/tools/hd-story-prepare.py")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(tool)) != manifest.ToolSHA {
		t.Fatal("stale source preparation")
	}
	if len(manifest.Records) != 75 || len(hdPortraitProfiles) != 75 {
		t.Fatal("reviewed source count")
	}
	return manifest.Records
}

func storyMemory(t *testing.T, record storySourceRecord) []byte {
	t.Helper()
	profile := hdPortraitProfiles[record.Resource]
	if profile.Columns != record.Columns || profile.Asset != record.Asset {
		t.Fatal("compiled source differs from preparation", record.Resource)
	}
	indices, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/" + record.Resource + ".idx")
	if err != nil || len(indices) != int(record.Columns)*8*150 {
		t.Fatal("source indices", record.Resource, err)
	}
	mem := make([]byte, 0x120000)
	for plane, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + plane*0x8000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		for y := 0; y < 150; y++ {
			for x := 0; x < int(record.Columns)*8; x++ {
				if indices[y*int(record.Columns)*8+x]&(1<<plane) != 0 {
					mem[base+y*80+x/8] |= 1 << uint(7-x%8)
				}
			}
		}
	}
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], record.Columns)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 150)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1bf9a)
	binary.LittleEndian.PutUint32(mem[0x1004:], 0x8000)
	copy(mem[0x8000:], record.Resource+"\x00")
	return mem
}

func TestStoryProfilesOriginalSourcesAndReturnLifecycle(t *testing.T) {
	for _, record := range storySources(t) {
		t.Run(record.Resource, func(t *testing.T) {
			mem := storyMemory(t, record)
			before := sha256.Sum256(mem)
			var h hdLive
			h.observe(1, 0x1bf44, 0x1000, mem)
			h.observe(2, 0x1bf9a, 0x1004, mem)
			if h.Active || !h.Validated || h.LastName != record.Resource {
				t.Fatal("source-matched load did not qualify", h.LastName, h.Validated)
			}
			for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
				binary.LittleEndian.PutUint32(mem[0x2000+4*i:], v)
			}
			afterSetup := sha256.Sum256(mem)
			h.observe(3, 0x1c1bc, 0x2000, mem)
			if h.Pending == nil || h.Active {
				t.Fatal("draw activated before original return")
			}
			h.observe(4, 0x1803d, 0x2000, mem)
			if h.Active {
				t.Fatal("wrong stack accepted")
			}
			h.observe(5, 0x1803d, 0x2004, mem)
			if !h.Active || h.Pending != nil || afterSetup != sha256.Sum256(mem) {
				t.Fatal("draw return or read-only observation")
			}
			// 新來源必須撤銷資格；同名但一個 byte 不符也不能沿用。
			h.observe(6, 0x1bf44, 0x1000, mem)
			mem[0x100000] ^= 1
			h.observe(7, 0x1bf9a, 0x1004, mem)
			if h.Validated || h.Active {
				t.Fatal("modified source retained eligibility")
			}
			mem[0x100000] ^= 1
			binary.LittleEndian.PutUint32(mem[0x5b1b8:], record.Columns+1)
			h.observe(8, 0x1bf44, 0x1000, mem)
			h.observe(9, 0x1bf9a, 0x1004, mem)
			if h.Validated {
				t.Fatal("wrong source width accepted")
			}
			if before == afterSetup {
				t.Fatal("fixture did not install draw parameters")
			}
		})
	}
}

func TestStoryDiagnosticLimitDoesNotDisablePresentation(t *testing.T) {
	var record storySourceRecord
	for _, r := range storySources(t) {
		if r.Resource == "A-0.OZM" {
			record = r
		}
	}
	mem := storyMemory(t, record)
	before := sha256.Sum256(mem)
	h := hdLive{Trace: hdTrace{Events: make([]map[string]any, 256)}}
	h.observe(1, 0x1bf44, 0x1000, mem)
	h.observe(2, 0x1bf9a, 0x1004, mem)
	if !h.Validated || h.LastName != "A-0.OZM" || len(h.Trace.Events) != 256 || h.Trace.Dropped != 2 || h.Trace.ReturnSequence != 1 {
		t.Fatal("diagnostic cap leaked into presentation")
	}
	mem[0x100000] ^= 1
	h.observe(3, 0x1bf44, 0x1000, mem)
	h.observe(4, 0x1bf9a, 0x1004, mem)
	if h.Validated || h.Trace.ReturnSequence != 2 {
		t.Fatal("full audit log retained stale source")
	}
	mem[0x100000] ^= 1
	if before != sha256.Sum256(mem) {
		t.Fatal("observer mutated input")
	}
}

func TestStoryPreparedArtCropLabelsAndFallback(t *testing.T) {
	for _, record := range storySources(t) {
		t.Run(record.Resource, func(t *testing.T) {
			profile := hdPortraitProfiles[record.Resource]
			data, err := os.ReadFile(profile.Source)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != profile.SourceSHA {
				t.Fatal("source PNG fingerprint", err)
			}
			source, err := previewPNG(profile.Source)
			if err != nil || source.Bounds() != image.Rect(0, 0, 320, 150) {
				t.Fatal("visible source crop", err)
			}
			assetPath := filepath.Join("/project/assets/hd-b/art", profile.Asset)
			if record.Resource == "A-0.OZM" {
				assetPath = "/project/assets/hd-b/A-0-hd-b-v1.png"
			}
			data, err = os.ReadFile(assetPath)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != record.AssetSHA {
				t.Fatal("reviewed HD image fingerprint", err)
			}
			asset, err := previewPNG(assetPath)
			if err != nil {
				t.Fatal(err)
			}
			prepared, reason := prepareHDPortrait(asset)
			if reason != "" {
				t.Fatal(reason)
			}
			native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
			for y := 0; y < 150; y++ {
				for x := 0; x < 320; x++ {
					native.SetNRGBA(160+x, 50+y, rgbaAt(source, x, y))
				}
			}
			before := append([]byte(nil), native.Pix...)
			out, mode, err := previewA0(native, source, prepared, 2, true)
			if err != nil || mode != "hd" {
				t.Fatal(mode, err)
			}
			preservePortraitLabels(out, native, record.Resource, 2)
			for _, rect := range portraitLabelRects[record.Resource] {
				for y := rect.Min.Y * 2; y < rect.Max.Y*2; y++ {
					for x := rect.Min.X * 2; x < rect.Max.X*2; x++ {
						if out.NRGBAAt(320+x, 100+y) != rgbaAt(source, x/2, y/2) {
							t.Fatal("original map lettering changed")
						}
					}
				}
			}
			if out.NRGBAAt(319, 100).A != 0 || out.NRGBAAt(960, 100).A != 0 || !bytes.Equal(before, native.Pix) {
				t.Fatal("presentation escaped rectangle or changed native image")
			}
			if _, mode, err := previewA0(native, source, nil, 2, true); err != nil || mode != "fallback_missing_asset" {
				t.Fatal("missing individual asset", mode, err)
			}
			native.Pix[native.PixOffset(200, 100)] ^= 1
			if _, mode, err := previewA0(native, source, prepared, 2, true); err != nil || mode != "fallback_source_mismatch" {
				t.Fatal("foreground occlusion", mode, err)
			}
		})
	}
}
