//go:build desktop

// 可撤回的桌面試作，正式素材包格式與發行入口另定。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var desktopSpeed = flag.Int("desktop-speed", 1, "desktop virtual-time speed, 1–60; validation acceleration only")
var desktopSource = flag.String("hd-source", "/project/workplace/hd-prototype/asset-capture/A-0-original.png", "verified local original A-0 PNG")
var desktopAsset = flag.String("hd-asset", "/project/workplace/hd-prototype/A-0-hd-b-v1.png", "local single-asset HD PNG; missing asset falls back")
var desktopScrollSource = flag.String("hd-scroll-source", "/project/workplace/hd-prototype/scroll/scroll-original.png", "verified local VISUAL scroll PNG")
var desktopScrollAsset = flag.String("hd-scroll-asset", "/project/workplace/hd-prototype/scroll/scroll-hd-b-v1.png", "local HD scroll PNG; missing asset falls back individually")

type desktopGame struct {
	c                           *desktopContext
	source, asset               image.Image
	texture                     *ebiten.Image
	pixels                      *image.NRGBA
	enabled, paused             bool
	mode                        string
	updates, draws, mouseEvents int
	lastMouse                   [3]int
	events                      []map[string]any
	assetInfo                   map[string]any
	bar                         windowBar
	barFont                     barFont
	barTexture                  *ebiten.Image
	barEvents                   []map[string]any
	blockMouse                  bool
	barPointer                  image.Point
	scroll                      *hdScroll
	scrollMode, portraitMode    string
	scrollForeground            int
}

func init() { runDesktop = runDesktopWindow }

func runDesktopWindow(c *desktopContext) error {
	if *desktopSpeed < 1 || *desktopSpeed > 60 {
		return fmt.Errorf("invalid desktop speed")
	}
	g := &desktopGame{c: c, lastMouse: [3]int{-1, -1, 0}, assetInfo: map[string]any{}, bar: newWindowBar()}
	font, err := loadBarFont("/project/tools/toolbar-font.hex.gz")
	if err != nil {
		return err
	}
	g.barFont = font
	if data, err := os.ReadFile(*desktopSource); err == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		g.assetInfo["source_sha256"] = sum
		if sum == "0b7a25670a568f2e8e61ecfdf4d7ea264dac13caf67dea7ddf9218b370f1239c" {
			g.source, err = previewPNG(*desktopSource)
			if err != nil {
				return err
			}
		}
	}
	if data, err := os.ReadFile(*desktopAsset); err == nil {
		g.assetInfo["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
		g.asset, err = previewPNG(*desktopAsset)
		if err != nil {
			g.assetInfo["hd_error"] = err.Error()
			g.asset = nil
		}
	}
	// 在啟動時縮到呈現矩形，避免每幀重新掃描大圖。
	if g.asset != nil {
		b := g.asset.Bounds()
		delta := b.Dx()*150 - b.Dy()*320
		if delta < 0 {
			delta = -delta
		}
		if b.Dx() < 320 || b.Dy() < 150 || delta*100 > b.Dy()*320 {
			g.assetInfo["hd_error"] = "shape or aspect mismatch"
			g.asset = nil
		} else {
			opaque := true
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					if rgbaAt(g.asset, x, y).A != 255 {
						opaque = false
					}
				}
			}
			if !opaque {
				g.assetInfo["hd_error"] = "transparent asset unsupported"
				g.asset = nil
			} else {
				resized := image.NewNRGBA(image.Rect(0, 0, 640, 300))
				for y := 0; y < 300; y++ {
					for x := 0; x < 640; x++ {
						resized.SetNRGBA(x, y, rgbaAt(g.asset, x*b.Dx()/640, y*b.Dy()/300))
					}
				}
				g.asset = resized
			}
		}
	}
	var scrollSource, scrollAsset image.Image
	scrollInfo := map[string]any{}
	if data, err := os.ReadFile(*desktopScrollSource); err == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		scrollInfo["source_sha256"] = sum
		if sum == "e8abcce3af8bc8bd4243ad27aecc0f8b07122813af23dd79d958650de04f6690" {
			scrollSource, err = previewPNG(*desktopScrollSource)
			if err != nil {
				return err
			}
		}
	}
	if data, err := os.ReadFile(*desktopScrollAsset); err == nil {
		scrollInfo["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
		scrollAsset, err = previewPNG(*desktopScrollAsset)
		if err != nil {
			scrollInfo["hd_error"] = err.Error()
			scrollAsset = nil
		}
	}
	g.scroll = newHDScroll(scrollSource, scrollAsset, 2)
	scrollInfo["validation"] = g.scroll.Reason
	g.assetInfo["scroll"] = scrollInfo
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("德川家康｜原版｜F6 切換 HD｜F7 暫停")
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	ebiten.SetRunnableOnUnfocused(true)
	g.texture = ebiten.NewImage(1280, 800)
	g.barTexture = ebiten.NewImage(640, barHeight+2*barRowHeight)
	err = ebiten.RunGame(g)
	c.Report = map[string]any{"classification": "Linux desktop prototype; source-matched A-0 portrait and story scroll only", "updates": g.updates, "draws": g.draws, "events": g.events, "windowbar_events": g.barEvents, "hd_observation": c.Live, "physical_mouse_events": g.mouseEvents, "scripted_mouse": c.Scripted, "hd_enabled": g.enabled, "last_mode": g.mode, "portrait_mode": g.portraitMode, "scroll_mode": g.scrollMode, "scroll_foreground_pixels": g.scrollForeground, "assets": g.assetInfo, "virtual_speed": *desktopSpeed, "audio_output": "PCM receipt only; desktop playback not enabled", "listening_verified": false}
	return err
}

func (g *desktopGame) render() error {
	s, err := g.c.Snapshot()
	if err != nil {
		return err
	}
	if s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 400) {
		g.pixels = image.NewNRGBA(image.Rect(0, 0, 1280, 800))
		g.mode = "waiting_for_640x400"
		return nil
	}
	out, mode, err := previewA0(s.Image, g.source, g.asset, 2, g.enabled && g.c.Live.Active)
	if err != nil {
		return err
	}
	if g.enabled && !g.c.Live.Active {
		mode = "fallback_no_original_draw"
	}
	g.portraitMode = mode
	g.scrollMode, g.scrollForeground = g.scroll.apply(out, s.Image, g.enabled && g.c.Live.ScrollActive)
	if g.enabled && !g.c.Live.ScrollActive {
		g.scrollMode = "fallback_no_original_background"
	}
	if g.scrollMode == "hd" {
		mode = "hd"
	}
	compositeDesktopCursor(out, s, 2)
	g.pixels = out
	g.mode = mode
	return nil
}

func (g *desktopGame) saveFrame(label string) (map[string]any, error) {
	if err := g.render(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/output/startup-probe/desktop-%d-%s.png", g.c.Steps(), label)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	err = png.Encode(f, g.pixels)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "png_sha256": fmt.Sprintf("%x", sha256.Sum256(b)), "rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(g.pixels.Pix)), "mode": g.mode}, nil
}

func (g *desktopGame) setHD(enabled bool) error {
	if enabled != g.enabled {
		before := g.c.Proof()
		g.enabled = enabled
		f, err := g.saveFrame(fmt.Sprintf("switch-%d", len(g.events)))
		if err != nil {
			return err
		}
		after := g.c.Proof()
		same := reflect.DeepEqual(before, after)
		if len(g.events) < 128 {
			g.events = append(g.events, map[string]any{"kind": "toggle", "step": g.c.Steps(), "enabled": g.enabled, "paused": g.paused, "before": before, "after": after, "guest_unchanged": same, "frame": f})
		}
		if !same {
			return fmt.Errorf("presentation changed original state")
		}
	}
	return nil
}

func (g *desktopGame) capture() error {
	f, err := g.saveFrame(fmt.Sprintf("capture-%d", len(g.events)))
	if err != nil {
		return err
	}
	if len(g.events) < 128 {
		g.events = append(g.events, map[string]any{"kind": "capture", "step": g.c.Steps(), "paused": g.paused, "proof": g.c.Proof(), "frame": f})
	}
	return nil
}

func (g *desktopGame) Update() error {
	g.updates++
	consumed, err := g.updateBar()
	if err != nil {
		return err
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		if err := g.setHD(!g.enabled); err != nil {
			return err
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		if err := g.capture(); err != nil {
			return err
		}
	}
	mouseBlocked := true
	if !consumed {
		mouseBlocked = g.mouseBlocked()
	}
	if !g.c.Scripted && !consumed && !mouseBlocked {
		x, y := ebiten.CursorPosition()
		p, inside := barGamePoint(x, y, g.bar.Visible)
		buttons := 0
		if ebiten.IsFocused() && inside {
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				buttons |= 1
			}
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
				buttons |= 2
			}
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
				buttons |= 4
			}
			now := [3]int{p.X, p.Y, buttons}
			if now != g.lastMouse {
				g.c.Mouse(now[0], now[1], now[2])
				g.lastMouse = now
				g.mouseEvents++
			}
		} else if g.lastMouse[2] != 0 {
			g.c.Mouse(g.lastMouse[0], g.lastMouse[1], 0)
			g.lastMouse[2] = 0
			g.mouseEvents++
		}
	}
	running := true
	if !g.paused && !consumed && !mouseBlocked {
		n := int((int64(g.updates)*1000000*int64(*desktopSpeed))/60 - (int64(g.updates-1)*1000000*int64(*desktopSpeed))/60)
		running = g.c.Advance(n)
	}
	if err := g.render(); err != nil {
		return err
	}
	g.texture.WritePixels(g.pixels.Pix)
	if g.bar.Visible {
		g.barTexture.WritePixels(g.bar.paint(g.barFont, g.selectedBar(), g.hdAvailable()).Pix)
	}
	title := "原版"
	if g.mode == "hd" {
		title = "HD"
	}
	if g.paused {
		title += " 暫停"
	}
	ebiten.SetWindowTitle("德川家康｜" + title + "｜頂端設定／Esc｜F6 HD｜F7 暫停")
	w, h := ebiten.WindowSize()
	status := map[string]any{"step": g.c.Steps(), "mode": g.mode, "eligible": g.c.Live.Active, "scroll_eligible": g.c.Live.ScrollActive, "scroll_mode": g.scrollMode, "scroll_foreground_pixels": g.scrollForeground, "portrait_mode": g.portraitMode, "enabled": g.enabled, "paused": g.paused, "events": g.events, "mouse_events": g.mouseEvents, "bar_visible": g.bar.Visible, "bar_pinned": g.bar.Pinned, "bar_menu": g.bar.Menu, "bar_focus": g.bar.Focus, "bar_hover": g.bar.Hover, "bar_events": g.barEvents, "window_size": []int{w, h}, "mouse_blocked": g.blockMouse}
	b, err := json.Marshal(status)
	if err != nil {
		return err
	}
	if err = os.WriteFile("/output/desktop-status.pending", b, 0600); err != nil {
		return err
	}
	if err = os.Rename("/output/desktop-status.pending", "/output/desktop-status.json"); err != nil {
		return err
	}
	if !running {
		return ebiten.Termination
	}
	return nil
}

func (g *desktopGame) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	if g.bar.Visible {
		op.GeoM.Translate(0, barHeight*2)
	}
	screen.DrawImage(g.texture, op)
	if g.bar.Visible {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(g.barTexture, op)
	}
	g.draws++
}

func (g *desktopGame) Layout(_, _ int) (int, int) {
	if g.bar.Visible {
		return 1280, 800 + barHeight*2
	}
	return 1280, 800
}

func (g *desktopGame) DrawFinalScreen(dst ebiten.FinalScreen, src *ebiten.Image, geo ebiten.GeoM) {
	dst.DrawImage(src, &ebiten.DrawImageOptions{GeoM: geo, Filter: ebiten.FilterNearest})
}
