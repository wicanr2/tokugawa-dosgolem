package main

import "image"

type hdFrame struct {
	Source, Asset *image.NRGBA
	Scale         int
	Reason        string
}

func newHDFrame(source, asset image.Image, scale int) *hdFrame {
	h := &hdFrame{Scale: scale, Reason: "fallback_source_shape"}
	if scale < 1 || scale > 4 || source == nil || source.Bounds() != image.Rect(0, 0, 640, 250) {
		return h
	}
	h.Source = image.NewNRGBA(source.Bounds())
	for y := 0; y < 250; y++ {
		for x := 0; x < 640; x++ {
			c := rgbaAt(source, x, y)
			if c.A != 255 {
				h.Reason = "fallback_source_transparency"
				return h
			}
			h.Source.SetNRGBA(x, y, c)
		}
	}
	h.Reason = "fallback_missing_asset"
	if asset == nil {
		return h
	}
	prepared, reason := prepareHDImage(asset, 640, 250, scale)
	if reason != "" {
		h.Reason = "fallback_hd_invalid"
		return h
	}
	h.Asset = prepared.(*image.NRGBA)
	h.Reason = "ready"
	return h
}

// 中央劇情圖另有自己的來源與圖層，框架只能畫它之外的背景。
func (h *hdFrame) apply(out *image.NRGBA, native image.Image, enabled bool) string {
	if !enabled {
		return "original"
	}
	if h == nil {
		return "fallback_missing_asset"
	}
	if h.Reason != "ready" {
		return h.Reason
	}
	if native == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || out == nil || out.Bounds() != image.Rect(0, 0, 640*h.Scale, 400*h.Scale) {
		return "fallback_frame_shape"
	}
	for y := 0; y < 250; y++ {
		for x := 0; x < 640; x++ {
			if !image.Pt(x, y).In(a0Rect) && rgbaAt(native, x, y) != h.Source.NRGBAAt(x, y) {
				return "fallback_source_mismatch"
			}
		}
	}
	for y := 0; y < 250*h.Scale; y++ {
		src, dst := h.Asset.PixOffset(0, y), out.PixOffset(0, y)
		if y < a0Rect.Min.Y*h.Scale || y >= a0Rect.Max.Y*h.Scale {
			copy(out.Pix[dst:dst+640*h.Scale*4], h.Asset.Pix[src:src+640*h.Scale*4])
		} else {
			left, right := a0Rect.Min.X*h.Scale*4, a0Rect.Max.X*h.Scale*4
			copy(out.Pix[dst:dst+left], h.Asset.Pix[src:src+left])
			copy(out.Pix[dst+right:dst+640*h.Scale*4], h.Asset.Pix[src+right:src+640*h.Scale*4])
		}
	}
	return "hd"
}
