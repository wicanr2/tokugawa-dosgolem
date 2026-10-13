// 單張素材呈現的共用試作；識別與缺圖備援不改輸入。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
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
