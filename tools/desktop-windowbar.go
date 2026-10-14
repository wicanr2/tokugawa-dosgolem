//go:build desktop

package main

import (
	"fmt"
	"image"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *desktopGame) resizeBar(old bool) {
	if old == g.bar.Visible {
		return
	}
	w, h := ebiten.WindowSize()
	d := barHeightDelta(w, h, old)
	if g.bar.Visible {
		h += d
	} else {
		h -= d
	}
	ebiten.SetWindowSize(w, max(h, 100))
}

func (g *desktopGame) selectedBar() [3]int {
	var selected [3]int
	if g.enabled {
		selected[0] = 1
	}
	if g.paused {
		selected[1] = 1
	}
	return selected
}

func (g *desktopGame) hdAvailable() bool {
	if g.terrainTree.Reason == "ready" {
		return true
	}
	for _, pair := range g.terrain {
		if pair.Reason == "ready" {
			return true
		}
	}
	for _, pair := range g.kao {
		if pair.Asset != nil && pair.Source != nil {
			return true
		}
	}
	for _, pair := range g.portraits {
		if pair.Asset != nil && pair.Source != nil {
			return true
		}
	}
	return (g.asset != nil && g.source != nil) || (g.scroll != nil && g.scroll.Reason == "ready") || (g.frame != nil && g.frame.Reason == "ready") || (g.window != nil && g.window.Reason == "ready")
}

func (g *desktopGame) chooseBar(field, row int) error {
	if field < 0 || field > 2 || row < 0 || row >= len(barLists[field]) {
		return nil
	}
	if field == 0 && row == 1 && !g.hdAvailable() {
		return nil
	}
	before := g.c.Proof()
	switch field {
	case 0:
		if err := g.setHD(row == 1); err != nil {
			return err
		}
	case 1:
		g.paused = row == 1
	case 2:
		if row == 1 {
			return ebiten.Termination
		}
		if err := g.capture(); err != nil {
			return err
		}
	}
	after := g.c.Proof()
	same := reflect.DeepEqual(before, after)
	if len(g.barEvents) < 128 {
		g.barEvents = append(g.barEvents, map[string]any{"kind": "option", "field": field, "row": row, "before": before, "after": after, "guest_unchanged": same})
	}
	if !same {
		return fmt.Errorf("window bar changed original state")
	}
	return nil
}

// 功能列先收走輸入，再決定是否步進原版。關閉的當幀也不轉送。
func (g *desktopGame) updateBar() (bool, error) {
	x, y := ebiten.CursorPosition()
	p := image.Pt(x/2, y/2)
	if !ebiten.IsFocused() || x < 0 || y < 0 {
		p = image.Pt(-1, -1)
	}
	old := g.bar.Visible
	esc := inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	if esc && (ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)) {
		return true, ebiten.Termination
	}
	changed := g.bar.reveal(esc, p)
	if changed {
		g.resizeBar(old)
		if len(g.barEvents) < 128 {
			g.barEvents = append(g.barEvents, map[string]any{"kind": "visibility", "visible": g.bar.Visible, "pinned": g.bar.Pinned, "step": g.c.Steps()})
		}
	}
	if g.bar.Visible {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}
	if old || g.bar.Visible || changed {
		g.blockMouse = true
	}
	if esc || changed {
		return true, nil
	}
	if !g.bar.Visible {
		return false, nil
	}
	field, row := g.bar.hit(p)
	if field == g.bar.Menu && row >= 0 && p != g.barPointer {
		g.bar.Hover = row
	}
	g.barPointer = p
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case field == 3:
			g.bar.close()
			g.resizeBar(true)
		case row >= 0:
			if err := g.chooseBar(field, row); err != nil {
				return true, err
			}
			g.bar.Menu = -1
		case field >= 0:
			g.bar.Focus = field
			g.bar.Hover = g.selectedBar()[field]
			if g.bar.Menu == field {
				g.bar.Menu = -1
			} else {
				g.bar.Menu = field
			}
		default:
			g.bar.Menu = -1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.bar.Focus = (g.bar.Focus + 1) % 3
		g.bar.Menu = -1
	}
	down, up := inpututil.IsKeyJustPressed(ebiten.KeyDown), inpututil.IsKeyJustPressed(ebiten.KeyUp)
	if down || up {
		if g.bar.Menu < 0 {
			g.bar.Menu = g.bar.Focus
			g.bar.Hover = g.selectedBar()[g.bar.Focus]
		} else {
			d := 1
			if up {
				d = -1
			}
			n := len(barLists[g.bar.Menu])
			g.bar.Hover = (g.bar.Hover + d + n) % n
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.bar.Menu < 0 {
			g.bar.Menu = g.bar.Focus
			g.bar.Hover = g.selectedBar()[g.bar.Focus]
		} else {
			if err := g.chooseBar(g.bar.Menu, g.bar.Hover); err != nil {
				return true, err
			}
			g.bar.Menu = -1
		}
	}
	return true, nil
}

func (g *desktopGame) mouseBlocked() bool {
	if !g.blockMouse {
		return false
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		g.blockMouse = false
	}
	return true
}
