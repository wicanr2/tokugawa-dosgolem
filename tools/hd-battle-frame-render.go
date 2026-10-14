package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
)

type hdBattleFrame struct {
	Source, Mask                     []byte
	Asset                            *image.NRGBA
	Reason, PreparedReason           string
	Prepared, CrestOpen, WeatherOpen bool
	Scale                            int
}

type hdBattleFrameReport struct {
	Mode                      string `json:"mode"`
	CrestHD                   bool   `json:"crest_HD"`
	WeatherPlaceholderHD      bool   `json:"weather_placeholder_HD"`
	PaintedNative             int    `json:"painted_native_pixels"`
	PreservedNativeForeground int    `json:"preserved_native_foreground_pixels"`
	SkippedScene              int    `json:"skipped_scene_native_pixels"`
}

func makeBattleFrameMask(source []byte) []byte {
	if len(source) != 640*400 {
		return nil
	}
	mask, seen := make([]byte, len(source)), make([]bool, len(source))
	black := func(n int) bool { return source[n] == 0 || source[n] == 8 }
	for n := range source {
		if !black(n) {
			mask[n] = 1
		}
	}
	queue := make([]int, 0, len(source))
	for start := range source {
		if !black(start) || seen[start] {
			continue
		}
		queue = queue[:0]
		queue = append(queue, start)
		seen[start] = true
		edge := false
		for head := 0; head < len(queue); head++ {
			n := queue[head]
			x, y := n%640, n/640
			edge = edge || x == 0 || x == 639 || y == 0 || y == 399
			for _, p := range [4]image.Point{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				if p.X < 0 || p.X >= 640 || p.Y < 0 || p.Y >= 400 {
					continue
				}
				nn := p.Y*640 + p.X
				if black(nn) && !seen[nn] {
					seen[nn] = true
					queue = append(queue, nn)
				}
			}
		}
		if !edge && len(queue) <= 256 {
			for _, n := range queue {
				mask[n] = 1
			}
		}
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			if image.Pt(x, y).In(battleFrameCrestRect) || image.Pt(x, y).In(battleFrameMinimapRect) {
				mask[y*640+x] = 0
			}
		}
	}
	return mask
}

func newHDBattleFrame(indices []byte, asset image.Image, scale int) *hdBattleFrame {
	p := &hdBattleFrame{Reason: "fallback_source", Scale: scale}
	if len(indices) != 640*400 || fmt.Sprintf("%x", sha256.Sum256(indices)) != battleFrameIndexedSHA || scale < 1 || scale > 4 {
		return p
	}
	p.Source = append([]byte(nil), indices...)
	p.Mask = makeBattleFrameMask(indices)
	if fmt.Sprintf("%x", sha256.Sum256(p.Mask)) != battleFrameMaskSHA {
		p.Reason = "fallback_source_mask"
		return p
	}
	p.Reason = "fallback_missing_asset"
	if asset == nil {
		return p
	}
	prepared, reason := prepareHDImage(asset, 640, 400, scale)
	if reason != "" {
		p.Reason = "fallback_asset_geometry"
		return p
	}
	p.Asset = prepared.(*image.NRGBA)
	p.Reason = "ready"
	return p
}

func (p *hdBattleFrame) update(s nativeSnapshot, active, enabled bool) {
	p.Prepared, p.CrestOpen, p.WeatherOpen = false, false, false
	p.PreparedReason = "original"
	if !enabled {
		return
	}
	p.PreparedReason = p.Reason
	if p.Reason != "ready" {
		return
	}
	p.PreparedReason = "fallback_no_original_full_copy"
	if !active {
		return
	}
	p.PreparedReason = "fallback_native_shape"
	if s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 400) || len(s.Indexed) != 640*400 || s.MaxColor != 15 {
		return
	}
	p.PreparedReason = "fallback_palette"
	for i, want := range battleFrameNativePalette {
		rgb := s.Palette[s.DAC[i]]
		if want != (color.NRGBA{rgb[0], rgb[1], rgb[2], 255}) {
			return
		}
	}
	match := func(r image.Rectangle) bool {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				n := y*640 + x
				if s.Indexed[n] != p.Source[n] {
					return false
				}
			}
		}
		return true
	}
	p.CrestOpen, p.WeatherOpen = match(battleFrameCrestRect), match(battleFrameWeatherRect)
	p.Prepared, p.PreparedReason = true, "ready"
}

func (p *hdBattleFrame) background(x, y, dx, dy int, index byte) (color.NRGBA, bool) {
	if !p.Prepared || x < 0 || x >= 640 || y < 0 || y >= 400 || dx < 0 || dx >= p.Scale || dy < 0 || dy >= p.Scale {
		return color.NRGBA{}, false
	}
	n := y*640 + x
	if p.Mask[n] == 0 || p.Source[n] != index || (image.Pt(x, y).In(battleFrameWeatherRect) && !p.WeatherOpen) {
		return color.NRGBA{}, false
	}
	return p.Asset.NRGBAAt(x*p.Scale+dx, y*p.Scale+dy), true
}

// 後畫框架只處理仍屬於原版框架的索引，避開共同場景及同色原版人物。
func (p *hdBattleFrame) paint(out *image.NRGBA, s nativeSnapshot, state hdTerrainState, scene *hdTerrainScene) hdBattleFrameReport {
	r := hdBattleFrameReport{Mode: p.PreparedReason}
	if !p.Prepared {
		return r
	}
	if out == nil || out.Bounds() != image.Rect(0, 0, 640*p.Scale, 400*p.Scale) {
		r.Mode = "fallback_geometry"
		return r
	}
	if len(state.History) > 0 && (scene == nil || !scene.ModelValid || len(scene.Pixels) != 640*400) {
		r.Mode = "fallback_scene_model"
		return r
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			n := y*640 + x
			crest := image.Pt(x, y).In(battleFrameCrestRect)
			if p.Mask[n] == 0 && !(p.CrestOpen && crest) {
				continue
			}
			if image.Pt(x, y).In(battleFrameWeatherRect) && !p.WeatherOpen {
				continue
			}
			if s.Indexed[n] != p.Source[n] {
				r.PreservedNativeForeground++
				continue
			}
			if len(state.History) > 0 && (scene.Pixels[n].Affected || scene.Pixels[n].VisibleSprite) {
				r.SkippedScene++
				continue
			}
			for dy := 0; dy < p.Scale; dy++ {
				for dx := 0; dx < p.Scale; dx++ {
					out.SetNRGBA(x*p.Scale+dx, y*p.Scale+dy, p.Asset.NRGBAAt(x*p.Scale+dx, y*p.Scale+dy))
				}
			}
			r.PaintedNative++
			if crest {
				r.CrestHD = true
			}
			if image.Pt(x, y).In(battleFrameWeatherRect) {
				r.WeatherPlaceholderHD = true
			}
		}
	}
	r.Mode = "fallback_no_visible_frame"
	if r.PaintedNative > 0 {
		r.Mode = "hd"
	}
	return r
}
