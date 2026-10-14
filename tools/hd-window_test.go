package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"
)

func windowTestImages() (*image.NRGBA, *image.NRGBA) {
	source := image.NewNRGBA(image.Rect(0, 0, 144, 16))
	asset := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	for tile := 0; tile < 9; tile++ {
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				source.SetNRGBA(tile*16+x, y, color.NRGBA{uint8(10 + tile), uint8(x), uint8(y), 255})
			}
		}
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				asset.SetNRGBA(tile%3*32+x, tile/3*32+y, color.NRGBA{uint8(150 + tile), uint8(x), uint8(y), 255})
			}
		}
	}
	return source, asset
}

func paintOriginalWindow(native, source *image.NRGBA, rect image.Rectangle) {
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			tile := hdWindowTile(x, y, rect.Dx(), rect.Dy())
			native.SetNRGBA(rect.Min.X+x, rect.Min.Y+y, source.NRGBAAt(tile*16+x%16, y%16))
		}
	}
}

func TestWindowNineTilesAndForegroundPreservation(t *testing.T) {
	source, asset := windowTestImages()
	h := newHDWindow(source, asset, 2)
	if h.Reason != "ready" {
		t.Fatal(h.Reason)
	}
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	rect := image.Rect(80, 68, 192, 180)
	paintOriginalWindow(native, source, rect)
	foreground := []image.Point{{120, 100}, {120, 164}, {120, 165}}
	for _, p := range foreground {
		native.SetNRGBA(p.X, p.Y, color.NRGBA{0, 0, 0, 255})
	}
	before := append([]byte(nil), native.Pix...)
	sourceBefore := append([]byte(nil), source.Pix...)
	assetBefore := append([]byte(nil), asset.Pix...)
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	// 模擬已存在的較低 HD 圖層，前景必須主動覆回原色。
	for y := rect.Min.Y * 2; y < rect.Max.Y*2; y++ {
		for x := rect.Min.X * 2; x < rect.Max.X*2; x++ {
			out.SetNRGBA(x, y, color.NRGBA{255, 0, 255, 255})
		}
	}
	mode, count := h.apply(out, native, rect, true)
	if mode != "hd" || count != len(foreground)*4 {
		t.Fatal(mode, count)
	}
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			if !image.Pt(x/2, y/2).In(rect) {
				if out.NRGBAAt(x, y) != (color.NRGBA{}) {
					t.Fatal("changed outside window", x, y)
				}
				continue
			}
			lx, ly := x/2-rect.Min.X, y/2-rect.Min.Y
			tile := hdWindowTile(lx, ly, rect.Dx(), rect.Dy())
			want := asset.NRGBAAt(tile%3*32+lx%16*2+x%2, tile/3*32+ly%16*2+y%2)
			for _, p := range foreground {
				if p == image.Pt(x/2, y/2) {
					want = native.NRGBAAt(p.X, p.Y)
				}
			}
			if out.NRGBAAt(x, y) != want {
				t.Fatal("tile or foreground mismatch", x, y)
			}
		}
	}
	if !bytes.Equal(before, native.Pix) || !bytes.Equal(sourceBefore, source.Pix) || !bytes.Equal(assetBefore, asset.Pix) {
		t.Fatal("presentation mutated input")
	}
	// 四角出現前景時，完整還原原生視窗，不殘留低層 HD。
	native.SetNRGBA(rect.Min.X, rect.Min.Y, color.NRGBA{255, 255, 255, 255})
	if mode, _ := h.apply(out, native, rect, true); mode != "fallback_source_mismatch" {
		t.Fatal(mode)
	}
	for y := rect.Min.Y * 2; y < rect.Max.Y*2; y++ {
		for x := rect.Min.X * 2; x < rect.Max.X*2; x++ {
			if out.NRGBAAt(x, y) != native.NRGBAAt(x/2, y/2) {
				t.Fatal("fallback failed to restore native foreground")
			}
		}
	}
	if mode, _ := h.apply(out, native, rect, false); mode != "original" {
		t.Fatal(mode)
	}
}

func TestWindowAssetAndGeometryFallback(t *testing.T) {
	source, asset := windowTestImages()
	if h := newHDWindow(source, nil, 2); h.Reason != "fallback_missing_asset" {
		t.Fatal(h.Reason)
	}
	if h := newHDWindow(nil, asset, 2); h.Reason != "fallback_source_shape" {
		t.Fatal(h.Reason)
	}
	asset.SetNRGBA(40, 40, color.NRGBA{})
	if h := newHDWindow(source, asset, 2); h.Reason != "fallback_hd_invalid" {
		t.Fatal(h.Reason)
	}
	if h := newHDWindow(source, image.NewNRGBA(image.Rect(0, 0, 100, 100)), 2); h.Reason != "fallback_hd_shape" {
		t.Fatal(h.Reason)
	}
	source, asset = windowTestImages()
	h := newHDWindow(source, asset, 2)
	if mode, _ := h.apply(image.NewNRGBA(image.Rect(0, 0, 1280, 800)), image.NewNRGBA(image.Rect(0, 0, 640, 400)), image.Rect(600, 380, 648, 428), true); mode != "fallback_geometry" {
		t.Fatal(mode)
	}
}

func TestWindowOriginalReturnAndSaveRestore(t *testing.T) {
	packed, err := os.ReadFile("/project/workplace/hd-prototype/catalog/window/WINDOW-packed.bin")
	if err != nil {
		t.Skip("local original window input unavailable")
	}
	mem := make([]byte, 0x110000)
	copy(mem[0x100000:], packed)
	binary.LittleEndian.PutUint32(mem[0x35153:], 0x100000)
	if !hdWindowSourceValid(mem) {
		t.Fatal("actual original source fingerprint")
	}
	esp := uint32(0x1000)
	put := func(values ...uint32) {
		for i, v := range values {
			binary.LittleEndian.PutUint32(mem[int(esp)+i*4:], v)
		}
	}
	h := hdWindowState{}
	put(0x19000, 10, 20, 8, 3)
	before := append([]byte(nil), mem...)
	h.observe(1, 0x1b00c, esp, mem)
	h.observe(2, 0x19000, esp, mem)
	if h.PendingDraw == nil || len(h.Draws) != 0 {
		t.Fatal("activated before original return")
	}
	h.observe(3, 0x19000, esp+4, mem)
	if len(h.Draws) != 1 || h.Draws[0].Rect != image.Rect(80, 20, 208, 68) || h.Draws[0].EndStep != 3 || !bytes.Equal(before, mem) {
		t.Fatal("original draw or read-only observation")
	}
	initial := append([]hdWindowDraw(nil), h.Draws...)
	h.observe(4, 0x1b165, esp, mem)
	binary.LittleEndian.PutUint32(mem[0x35157:], 1)
	h.observe(5, 0x19000, esp+4, mem)
	put(0x19000, 10, 68, 7, 7)
	h.observe(6, 0x1b00c, esp, mem)
	h.observe(7, 0x19000, esp+4, mem)
	if len(h.Draws) != 2 {
		t.Fatal("new modal window was not registered")
	}
	// 第二層保存後重繪全畫面；還原只能在原版返回及計數相符後。
	h.observe(8, 0x1b165, esp, mem)
	binary.LittleEndian.PutUint32(mem[0x35157:], 2)
	h.observe(9, 0x19000, esp+4, mem)
	h.observe(10, 0x1c084, esp, mem)
	if len(h.Draws) != 0 {
		t.Fatal("full background retained window")
	}
	h.observe(11, 0x1b245, esp, mem)
	h.observe(12, 0x19000, esp+4, mem)
	if len(h.Draws) != 0 {
		t.Fatal("wrong save level restored a window")
	}
	// 上一次未符合的返回已撤銷待還原。以另一個完整保存／還原重測。
	h.Draws = append([]hdWindowDraw(nil), initial...)
	binary.LittleEndian.PutUint32(mem[0x35157:], 1)
	h.observe(13, 0x1b245, esp, mem)
	if len(h.Draws) != 0 || h.PendingRestore == nil {
		t.Fatal("restore entry must fence presentation")
	}
	binary.LittleEndian.PutUint32(mem[0x35157:], 0)
	before = append(before[:0], mem...)
	h.observe(14, 0x19000, esp+4, mem)
	if !reflect.DeepEqual(h.Draws, initial) || !bytes.Equal(before, mem) {
		t.Fatal("original restore did not restore presentation metadata")
	}
	put(0x19000, 10, 20, 2, 3)
	h.observe(15, 0x1b00c, esp, mem)
	if len(h.Draws) != 0 || h.PendingDraw != nil {
		t.Fatal("invalid dimensions retained old window")
	}
	put(0x19000, 10, 20, 8, 3)
	h.observe(16, 0x1b00c, esp, mem)
	mem[0x100000] ^= 1
	h.observe(17, 0x19000, esp+4, mem)
	if len(h.Draws) != 0 {
		t.Fatal("changed original source activated at return")
	}
}
