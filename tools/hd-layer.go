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

func preservePortraitLabels(out *image.NRGBA, native image.Image, name string, scale int) {
	preservePortraitLabelsAt(out, native, name, a0Rect, scale)
}

func preservePortraitLabelsAt(out *image.NRGBA, native image.Image, name string, destination image.Rectangle, scale int) {
	for _, rect := range portraitLabelRects[name] {
		for y := rect.Min.Y * scale; y < rect.Max.Y*scale; y++ {
			for x := rect.Min.X * scale; x < rect.Max.X*scale; x++ {
				out.SetNRGBA(destination.Min.X*scale+x, destination.Min.Y*scale+y, rgbaAt(native, destination.Min.X+x/scale, destination.Min.Y+y/scale))
			}
		}
	}
}

func rgbaAt(im image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
}

// 啟動時驗證並縮到呈現尺寸，不在每幀掃描大圖。
func prepareHDPortrait(asset image.Image) (image.Image, string) {
	return prepareHDImage(asset, 320, 150, 2)
}

func prepareHDImage(asset image.Image, width, height, scale int) (image.Image, string) {
	if asset == nil {
		return nil, "missing asset"
	}
	if width < 1 || height < 1 || scale < 1 || scale > 4 {
		return nil, "invalid target dimensions"
	}
	b := asset.Bounds()
	delta := b.Dx()*height - b.Dy()*width
	if delta < 0 {
		delta = -delta
	}
	if b.Min != image.Pt(0, 0) || b.Dx() < width || b.Dy() < height || b.Dx() > 8192 || b.Dy() > 8192 || delta*100 > b.Dy()*width {
		return nil, "shape or aspect mismatch"
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if rgbaAt(asset, x, y).A != 255 {
				return nil, "transparent asset unsupported"
			}
		}
	}
	resized := image.NewNRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := 0; y < height*scale; y++ {
		for x := 0; x < width*scale; x++ {
			resized.SetNRGBA(x, y, rgbaAt(asset, x*b.Dx()/(width*scale), y*b.Dy()/(height*scale)))
		}
	}
	return resized, ""
}

// 僅讀輸入，輸出全新畫布。識別失敗只保留原圖。
func previewA0(frame, source, hd image.Image, scale int, enabled bool) (*image.NRGBA, string, error) {
	return previewPortraitAt(frame, source, hd, a0Rect, scale, enabled)
}

func previewPortraitAt(frame, source, hd image.Image, destination image.Rectangle, scale int, enabled bool) (*image.NRGBA, string, error) {
	if frame == nil || frame.Bounds() != image.Rect(0, 0, 640, 400) || scale < 1 || scale > 4 {
		return nil, "", fmt.Errorf("原生畫面或放大倍率不符")
	}
	if destination != a0Rect && destination != endingStoryRect {
		return nil, "", fmt.Errorf("未驗證的劇情目的矩形")
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
			if rgbaAt(source, x, y) != rgbaAt(frame, destination.Min.X+x, destination.Min.Y+y) {
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
			out.SetNRGBA(destination.Min.X*scale+x, destination.Min.Y*scale+y, rgbaAt(hd, x*b.Dx()/(320*scale), y*b.Dy()/(150*scale)))
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
