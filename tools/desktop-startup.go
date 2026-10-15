//go:build desktop

package main

import (
	"flag"
	"fmt"
	"image"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var desktopStartupDisplay = flag.String("startup-display", "ask", "initial desktop display: ask, original or hd; automatic verification must choose explicitly")

func (g *desktopGame) updateStartupChoice() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.choice.Focus = 1 - g.choice.Focus
	}
	index := -1
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		index = g.choice.Focus
	}
	if ebiten.IsFocused() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		for i, r := range startupButtons {
			if image.Pt(x/2, y/2).In(r) {
				index = i
			}
		}
	}
	if index < 0 {
		return nil
	}
	before := g.c.Proof()
	if !g.choice.confirm(index) {
		return nil
	}
	g.enabled = g.choice.Selection == "hd"
	g.startupFence, g.blockMouse = true, true
	after := g.c.Proof()
	unchanged := reflect.DeepEqual(before, after)
	g.choiceEvents = append(g.choiceEvents, map[string]any{"selection": g.choice.Selection, "before": before, "after": after, "guest_unchanged": unchanged})
	if !unchanged {
		return fmt.Errorf("startup selection changed original state")
	}
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	return nil
}

// Keep the entire confirming gesture outside DOS, including held keys and buttons.
func (g *desktopGame) startupInputBlocked() bool {
	if !g.startupFence {
		return false
	}
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeySpace, ebiten.KeyTab, ebiten.KeyArrowLeft, ebiten.KeyArrowRight} {
		if ebiten.IsKeyPressed(key) {
			return true
		}
	}
	for _, button := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if ebiten.IsMouseButtonPressed(button) {
			return true
		}
	}
	g.startupFence = false
	return true // The release frame is consumed as well.
}

func (g *desktopGame) renderStartupChoice() {
	src := g.choice.paint(g.barFont)
	g.pixels = image.NewNRGBA(image.Rect(0, 0, 1280, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 1280; x++ {
			g.pixels.SetNRGBA(x, y, src.NRGBAAt(x/2, y/2))
		}
	}
	g.mode = "startup_choice"
}
