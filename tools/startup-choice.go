package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

var startupButtons = [2]image.Rectangle{image.Rect(80, 172, 304, 244), image.Rect(336, 172, 560, 244)}

type startupChoice struct {
	Pending   bool
	Focus     int
	Selection string
}

func newStartupChoice(display string) (startupChoice, error) {
	switch display {
	case "ask":
		return startupChoice{Pending: true}, nil
	case "original", "hd":
		return startupChoice{Selection: display}, nil
	default:
		return startupChoice{}, fmt.Errorf("startup-display must be ask, original or hd")
	}
}

func (s *startupChoice) confirm(index int) bool {
	if !s.Pending || index < 0 || index > 1 {
		return false
	}
	s.Pending, s.Focus = false, index
	s.Selection = "original"
	if index == 1 {
		s.Selection = "hd"
	}
	return true
}

func (s startupChoice) paint(font barFont) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	fill := func(r image.Rectangle, c color.NRGBA) {
		draw.Draw(out, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	fg := color.NRGBA{238, 231, 211, 255}
	fill(out.Bounds(), color.NRGBA{24, 29, 38, 255})
	font.text(out, 256, 80, "德川家康", fg)
	font.text(out, 256, 116, "選擇顯示", fg)
	for i, r := range startupButtons {
		border := color.NRGBA{70, 78, 91, 255}
		if i == s.Focus {
			border = color.NRGBA{196, 155, 82, 255}
		}
		fill(r, border)
		fill(r.Inset(2), color.NRGBA{44, 52, 65, 255})
		label := "原版"
		if i == 1 {
			label = "HD 圖像"
		}
		font.text(out, r.Min.X+72, r.Min.Y+28, label, fg)
	}
	font.text(out, 192, 280, "遊戲中可從頂端功能列切換", fg)
	font.text(out, 200, 320, "滑鼠選擇  Tab / Enter", color.NRGBA{157, 168, 184, 255})
	return out
}
