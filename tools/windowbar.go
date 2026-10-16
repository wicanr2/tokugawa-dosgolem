package main

import (
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

const barHeight, barRowHeight = 32, 24

var barFields = [4]image.Rectangle{
	image.Rect(8, 4, 192, 28), image.Rect(200, 4, 368, 28),
	image.Rect(376, 4, 592, 28), image.Rect(600, 4, 632, 28),
}
var barLabels = [3]string{"顯示", "執行", "工具"}
var barLists = [3][]string{{"原版", "HD 圖像"}, {"繼續", "暫停"}, {"擷取畫面 F9", "離開程式"}}
var barText = "顯示執行工具原版肖像繼續暫停擷取畫面離開程式缺圖設定×▾✓德川家康選擇遊戲中可從頂端功能列切換滑鼠"

type windowBar struct {
	Visible, Pinned    bool
	Menu, Hover, Focus int
	blockHover         bool
}

func newWindowBar() windowBar { return windowBar{Menu: -1, Hover: -1} }

func (b *windowBar) close() {
	b.Visible, b.Pinned, b.Menu = false, false, -1
	b.blockHover = true
}

func (b *windowBar) reveal(escape bool, p image.Point) bool {
	old := b.Visible
	if escape {
		b.Visible = !b.Visible
		b.Pinned, b.blockHover, b.Menu = b.Visible, !b.Visible, -1
		return old != b.Visible
	}
	top := p.In(image.Rect(0, 0, 640, 6))
	if !top {
		b.blockHover = false
	}
	if b.Pinned {
		return false
	}
	if b.Visible {
		inside := p.In(image.Rect(0, 0, 640, barHeight))
		if b.Menu >= 0 {
			inside = inside || p.In(barMenuRect(b.Menu))
		}
		b.Visible = inside
	} else if top && !b.blockHover {
		b.Visible = true
	}
	if !b.Visible {
		b.Menu = -1
	}
	return old != b.Visible
}

func barMenuRect(field int) image.Rectangle {
	if field < 0 || field >= len(barLists) {
		return image.Rectangle{}
	}
	f := barFields[field]
	return image.Rect(f.Min.X, barHeight, f.Max.X, barHeight+len(barLists[field])*barRowHeight)
}

func (b *windowBar) hit(p image.Point) (field, row int) {
	if !b.Visible {
		return -1, -1
	}
	if b.Menu >= 0 && p.In(barMenuRect(b.Menu)) {
		return b.Menu, (p.Y - barHeight) / barRowHeight
	}
	for i, r := range barFields {
		if p.In(r) {
			return i, -1
		}
	}
	return -1, -1
}

func barHeightDelta(width, height int, visible bool) int {
	return barHeightDeltaFor(width, height, visible, 400)
}

func barHeightDeltaFor(width, height int, visible bool, gameHeight int) int {
	logicalHeight := gameHeight
	if visible {
		logicalHeight += barHeight
	}
	scale := min(float64(width)/640, float64(height)/float64(logicalHeight))
	return int(float64(barHeight) * scale)
}

// Ebitengine 已將視窗縮放與留黑邊轉成邏輯座標。
func barGamePoint(x, y int, visible bool) (image.Point, bool) {
	return barGamePointFor(x, y, visible, 400)
}

func barGamePointFor(x, y int, visible bool, gameHeight int) (image.Point, bool) {
	if visible {
		y -= barHeight * 2
	}
	p := image.Pt(x, y)
	return image.Pt(x/2, y/2), p.In(image.Rect(0, 0, 1280, gameHeight*2))
}

type barGlyph struct {
	Width int
	Bits  []byte
}
type barFont map[rune]barGlyph

func loadBarFont(path string) (barFont, error) {
	f, err := openRuntimeFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	wanted := map[rune]bool{}
	for r := rune(32); r <= 126; r++ {
		wanted[r] = true
	}
	for _, r := range barText {
		wanted[r] = true
	}
	font := barFont{}
	s := bufio.NewScanner(z)
	for s.Scan() {
		code, bits, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}
		n, e := strconv.ParseInt(code, 16, 32)
		if e != nil || !wanted[rune(n)] {
			continue
		}
		data, e := hex.DecodeString(bits)
		if e != nil || (len(data) != 16 && len(data) != 32) {
			return nil, fmt.Errorf("invalid toolbar glyph %s", code)
		}
		font[rune(n)] = barGlyph{len(data) / 2, data}
	}
	if err = s.Err(); err != nil {
		return nil, err
	}
	for r := range wanted {
		if _, ok := font[r]; !ok {
			return nil, fmt.Errorf("missing toolbar glyph %U", r)
		}
	}
	return font, nil
}

func (font barFont) text(dst *image.NRGBA, x, y int, text string, c color.NRGBA) {
	for _, r := range text {
		g := font[r]
		for gy := 0; gy < 16; gy++ {
			for gx := 0; gx < g.Width; gx++ {
				if g.Bits[gy*(g.Width/8)+gx/8]&(0x80>>uint(gx%8)) != 0 {
					dst.SetNRGBA(x+gx, y+gy, c)
				}
			}
		}
		x += g.Width
	}
}

func (b *windowBar) paint(font barFont, selected [3]int, available bool) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, 640, barHeight+2*barRowHeight))
	bg, fg := color.NRGBA{24, 29, 38, 255}, color.NRGBA{230, 234, 241, 255}
	button, active, muted := color.NRGBA{44, 52, 65, 255}, color.NRGBA{44, 92, 112, 255}, color.NRGBA{133, 143, 158, 255}
	fill := func(r image.Rectangle, c color.NRGBA) {
		draw.Draw(out, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	fill(image.Rect(0, 0, 640, barHeight), bg)
	for i, r := range barFields {
		c := button
		if i == b.Menu || i == b.Focus {
			c = active
		}
		fill(r, c)
		if i == 3 {
			font.text(out, r.Min.X+8, 8, "×", fg)
			continue
		}
		value := barLists[i][selected[i]]
		if i == 2 {
			value = ""
		}
		font.text(out, r.Min.X+8, 8, strings.TrimSpace(barLabels[i]+" "+value), fg)
		font.text(out, r.Max.X-16, 8, "▾", fg)
	}
	if b.Menu >= 0 {
		r := barMenuRect(b.Menu)
		fill(r, bg)
		for i, label := range barLists[b.Menu] {
			y := barHeight + i*barRowHeight
			if i == b.Hover {
				fill(image.Rect(r.Min.X, y, r.Max.X, y+barRowHeight), active)
			}
			c := fg
			if b.Menu == 0 && i == 1 && !available {
				label = "HD 缺圖"
				c = muted
			}
			mark := " "
			if b.Menu < 2 && i == selected[b.Menu] {
				mark = "✓"
			}
			font.text(out, r.Min.X+8, y+4, mark+" "+label, c)
		}
	}
	return out
}
