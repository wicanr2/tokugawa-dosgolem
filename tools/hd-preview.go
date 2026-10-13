// 可丟棄的離線呈現試作。來源識別只支援已驗證的 A-0 矩形。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
)

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
