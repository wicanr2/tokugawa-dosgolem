// 可丟棄的離線呈現試作。來源識別只支援已驗證的 A-0 矩形。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
)

var a0Rect = image.Rect(160, 50, 480, 200)

func rgbaAt(im image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
}

// 僅讀輸入，輸出全新畫布。識別失敗只保留原圖。
func previewA0(frame, source, hd image.Image, scale int, enabled bool) (*image.NRGBA, string, error) {
	if frame == nil || frame.Bounds() != image.Rect(0, 0, 640, 400) || scale < 1 || scale > 4 {
		return nil, "", fmt.Errorf("原生畫面或放大倍率不符")
	}
	out := image.NewNRGBA(image.Rect(0, 0, 640*scale, 400*scale))
	for y := 0; y < 400*scale; y++ {
		for x := 0; x < 640*scale; x++ {
			out.SetNRGBA(x, y, rgbaAt(frame, x/scale, y/scale))
		}
	}
	if !enabled {
		return out, "original", nil
	}
	if hd == nil {
		return out, "fallback_missing_asset", nil
	}
	if source == nil || source.Bounds() != image.Rect(0, 0, 320, 150) {
		return out, "fallback_source_shape", nil
	}
	for y := 0; y < 150; y++ {
		for x := 0; x < 320; x++ {
			if rgbaAt(source, x, y) != rgbaAt(frame, 160+x, 50+y) {
				return out, "fallback_source_mismatch", nil
			}
		}
	}
	b := hd.Bounds()
	if b.Min != image.Pt(0, 0) || b.Dx() < 320 || b.Dy() < 150 || b.Dx() > 8192 || b.Dy() > 8192 {
		return out, "fallback_hd_shape", nil
	}
	// 允許生成器的整數像素尺寸四捨五入；大於 1% 的比例差拒絕。
	difference := b.Dx()*150 - b.Dy()*320
	if difference < 0 {
		difference = -difference
	}
	if difference*100 > b.Dy()*320 {
		return out, "fallback_hd_aspect", nil
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if rgbaAt(hd, x, y).A != 255 {
				return out, "fallback_hd_transparency", nil
			}
		}
	}
	for y := 0; y < 150*scale; y++ {
		for x := 0; x < 320*scale; x++ {
			out.SetNRGBA(160*scale+x, 50*scale+y, rgbaAt(hd, x*b.Dx()/(320*scale), y*b.Dy()/(150*scale)))
		}
	}
	return out, "hd", nil
}

func previewPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 {
		return nil, fmt.Errorf("PNG 尺寸超界")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return nil, err
	}
	return png.Decode(f)
}

func main() {
	framePath := flag.String("frame", "", "原生畫面 PNG")
	sourcePath := flag.String("source", "", "已驗證 A-0 PNG")
	hdPath := flag.String("hd", "", "HD PNG；留空顯示原版，缺圖回原圖")
	outPath := flag.String("out", "", "輸出預覽 PNG")
	scale := flag.Int("scale", 2, "邏輯畫面放大倍率 1–4")
	flag.Parse()
	if _, err := os.Stat("/.dockerenv"); err != nil {
		panic("只允許在 Docker 執行")
	}
	frame, err := previewPNG(*framePath)
	if err != nil {
		panic(err)
	}
	source, err := previewPNG(*sourcePath)
	if err != nil {
		panic(err)
	}
	var hd image.Image
	if *hdPath != "" {
		hd, err = previewPNG(*hdPath)
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}
	out, mode, err := previewA0(frame, source, hd, *scale, *hdPath != "")
	if err != nil {
		panic(err)
	}
	f, err := os.OpenFile(*outPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, out); err != nil {
		f.Close()
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	changed, outside := 0, 0
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			if out.NRGBAAt(x, y) != rgbaAt(frame, x / *scale, y / *scale) {
				changed++
				if !image.Pt(x / *scale, y / *scale).In(a0Rect) {
					outside++
				}
			}
		}
	}
	b, err := os.ReadFile(*outPath)
	if err != nil {
		panic(err)
	}
	metadata := map[string]any{"mode": mode, "classification": "offline presentation prototype; runtime switching not implemented", "go_version": runtime.Version(), "scale": *scale, "logical_rect": []int{160, 50, 320, 150}, "png_sha256": fmt.Sprintf("%x", sha256.Sum256(b)), "rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(out.Pix)), "changed_pixels": changed, "changed_pixels_outside_rect": outside, "rights": "original-derived; local only"}
	m, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		panic(err)
	}
	f, err = os.OpenFile(*outPath+".json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	if _, err = f.Write(append(m, '\n')); err != nil {
		f.Close()
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("%s: changed=%d outside=%d PNG SHA-256 %x\n", mode, changed, outside, sha256.Sum256(b))
}
