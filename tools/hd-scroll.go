package main

import (
	"image"
)

var scrollRect = image.Rect(0, 250, 640, 400)
var scrollPaperRect = image.Rect(80, 268, 590, 382)

type hdScroll struct {
	Source, Asset *image.NRGBA
	Scale         int
	Reason        string
}

func newHDScroll(source, asset image.Image, scale int) *hdScroll {
	h := &hdScroll{Scale: scale, Reason: "fallback_missing_asset"}
	if scale < 1 || scale > 4 {
		h.Reason = "fallback_scale"
		return h
	}
	if source == nil || source.Bounds() != image.Rect(0, 0, 640, 150) {
		h.Reason = "fallback_source_shape"
		return h
	}
	h.Source = image.NewNRGBA(source.Bounds())
	for y := 0; y < 150; y++ {
		for x := 0; x < 640; x++ {
			c := rgbaAt(source, x, y)
			if c.A != 255 {
				h.Reason = "fallback_source_transparency"
				return h
			}
			h.Source.SetNRGBA(x, y, c)
		}
	}
	if asset == nil {
		return h
	}
	b := asset.Bounds()
	if b.Min != image.Pt(0, 0) || b.Dx() < 640 || b.Dy() < 150 || b.Dx() > 8192 || b.Dy() > 8192 {
		h.Reason = "fallback_hd_shape"
		return h
	}
	d := b.Dx()*150 - b.Dy()*640
	if d < 0 {
		d = -d
	}
	if d*100 > b.Dy()*640 {
		h.Reason = "fallback_hd_aspect"
		return h
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if rgbaAt(asset, x, y).A != 255 {
				h.Reason = "fallback_hd_transparency"
				return h
			}
		}
	}
	h.Asset = image.NewNRGBA(image.Rect(0, 0, 640*scale, 150*scale))
	for y := 0; y < 150*scale; y++ {
		for x := 0; x < 640*scale; x++ {
			h.Asset.SetNRGBA(x, y, rgbaAt(asset, x*b.Dx()/(640*scale), y*b.Dy()/(150*scale)))
		}
	}
	h.Reason = "ready"
	return h
}

// 前景是與底圖不同的原生像素，沒有把它升級成已解析的文字。
func (h *hdScroll) apply(out *image.NRGBA, frame image.Image, enabled bool) (string, int) {
	if !enabled {
		return "original", 0
	}
	if h == nil {
		return "fallback_missing_asset", 0
	}
	if h.Reason != "ready" {
		return h.Reason, 0
	}
	if frame == nil || frame.Bounds() != image.Rect(0, 0, 640, 400) || out == nil || out.Bounds() != image.Rect(0, 0, 640*h.Scale, 400*h.Scale) {
		return "fallback_frame_shape", 0
	}
	foreground := 0
	for y := 0; y < 150; y++ {
		for x := 0; x < 640; x++ {
			if rgbaAt(frame, x, y+250) != h.Source.NRGBAAt(x, y) {
				if !image.Pt(x, y+250).In(scrollPaperRect) {
					return "fallback_source_mismatch", 0
				}
				foreground++
			}
		}
	}
	for y := 0; y < 150*h.Scale; y++ {
		a, b := h.Asset.PixOffset(0, y), out.PixOffset(0, 250*h.Scale+y)
		copy(out.Pix[b:b+640*h.Scale*4], h.Asset.Pix[a:a+640*h.Scale*4])
	}
	for y := 0; y < 150; y++ {
		for x := 0; x < 640; x++ {
			c := rgbaAt(frame, x, y+250)
			if c == h.Source.NRGBAAt(x, y) {
				continue
			}
			for dy := 0; dy < h.Scale; dy++ {
				for dx := 0; dx < h.Scale; dx++ {
					out.SetNRGBA(x*h.Scale+dx, (y+250)*h.Scale+dy, c)
				}
			}
		}
	}
	return "hd", foreground
}
