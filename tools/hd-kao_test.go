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

func TestKaoPresentationPreservesOcclusionAndOtherRegions(t *testing.T) {
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	source := image.NewNRGBA(image.Rect(0, 0, 80, 80))
	asset := image.NewNRGBA(image.Rect(0, 0, 160, 160))
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			c := color.NRGBA{uint8(x), uint8(y), 80, 255}
			source.SetNRGBA(x, y, c)
			native.SetNRGBA(96+x, 86+y, c)
		}
	}
	for y := 0; y < 160; y++ {
		for x := 0; x < 160; x++ {
			asset.SetNRGBA(x, y, color.NRGBA{180, 130, 80, 255})
		}
	}
	before := append([]byte(nil), native.Pix...)
	rect := image.Rect(96, 86, 176, 166)
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	if got := previewKao(out, native, source, asset, rect, 2, true); got != "hd" {
		t.Fatal(got)
	}
	if out.NRGBAAt(191, 172).A != 0 || out.NRGBAAt(192, 172) != (color.NRGBA{180, 130, 80, 255}) {
		t.Fatal("replacement escaped its original rectangle")
	}
	if !bytes.Equal(before, native.Pix) {
		t.Fatal("native frame changed")
	}
	// 一個新前景像素也必須撤銷整張頭像，避免蓋住原版選項。
	native.SetNRGBA(120, 100, color.NRGBA{255, 255, 255, 255})
	clear(out.Pix)
	if got := previewKao(out, native, source, asset, rect, 2, true); got != "fallback_source_mismatch" {
		t.Fatal(got)
	}
	if !bytes.Equal(out.Pix, make([]byte, len(out.Pix))) {
		t.Fatal("fallback painted over original foreground")
	}
	if got := previewKao(out, native, source, nil, rect, 2, true); got != "fallback_missing_asset" {
		t.Fatal(got)
	}
}

func TestKaoReadOnlySourceAndReturn(t *testing.T) {
	data, err := os.ReadFile("/original/DX5.DAT")
	if err != nil {
		t.Skip("local original input is unavailable")
	}
	// DAT 目錄定位保持原始 bytes。第 12 筆為固定版本的 KAO.GRP。
	at := 12 + 12*21
	if len(data) < at+21 {
		t.Fatal("truncated original resource table")
	}
	if string(bytes.TrimRight(data[at:at+13], "\x00")) != "KAO.GRP" {
		t.Fatal("original resource table mismatch")
	}
	offset := binary.LittleEndian.Uint32(data[at+17:])
	length := binary.LittleEndian.Uint32(data[at+13:])
	if length != 81*3200 || uint64(offset)+uint64(length) > uint64(len(data)) {
		t.Fatal("original portrait group bounds")
	}
	for index, profile := range hdKaoProfiles {
		start := int(offset) + index*3200
		if fmt.Sprintf("%x", sha256.Sum256(data[start:start+3200])) != profile.RawSHA {
			t.Fatalf("original source fingerprint differs at portrait %d", index)
		}
	}
	mem := make([]byte, 0x100000+int(length))
	copy(mem[0x100000:], data[offset:offset+length])
	binary.LittleEndian.PutUint32(mem[0x39888:], 0x100000)
	esp := uint32(0x1000)
	for i, value := range []uint32{0x19000, 76, 12, 86} {
		binary.LittleEndian.PutUint32(mem[int(esp)+i*4:], value)
	}
	before := append([]byte(nil), mem...)
	h := hdKaoState{}
	h.observe(1, 0x1c6e8, esp, mem)
	if h.Pending == nil || len(h.Draws) != 0 {
		t.Fatal("source-matched entry must wait for original return")
	}
	h.observe(2, 0x19000, esp, mem)
	if len(h.Draws) != 0 {
		t.Fatal("wrong stack activated portrait")
	}
	h.observe(3, 0x19000, esp+4, mem)
	if len(h.Draws) != 1 || h.Draws[0].Index != 76 || h.Draws[0].Rect != image.Rect(96, 86, 176, 166) {
		t.Fatal("original return did not register expected rectangle")
	}
	if !bytes.Equal(before, mem) {
		t.Fatal("observation changed original RAM")
	}
	binary.LittleEndian.PutUint32(mem[esp+4:], 81)
	h.observe(4, 0x1c6e8, esp, mem)
	if h.Pending != nil || len(h.Draws) != 0 {
		t.Fatal("invalid new source retained old portrait at the same destination")
	}
	h.Draws = []hdKaoDraw{{76, image.Rect(96, 86, 176, 166)}}
	h.observe(5, 0x1c084, esp, mem)
	if len(h.Draws) != 0 {
		t.Fatal("full original background copy retained stale portraits")
	}
	binary.LittleEndian.PutUint32(mem[esp+4:], 76)
	mem[0x100000+76*3200] ^= 1
	h.observe(6, 0x1c6e8, esp, mem)
	if h.Pending != nil {
		t.Fatal("changed original source was accepted")
	}
}
