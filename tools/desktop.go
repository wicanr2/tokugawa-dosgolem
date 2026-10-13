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
}

func init() { runDesktop = runDesktopWindow }

func runDesktopWindow(c *desktopContext) error {
	if *desktopSpeed < 1 || *desktopSpeed > 60 {
		return fmt.Errorf("invalid desktop speed")
	}
	g := &desktopGame{c: c, lastMouse: [3]int{-1, -1, 0}, assetInfo: map[string]any{}}
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
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("德川家康｜原版｜F6 切換 HD｜F7 暫停")
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	ebiten.SetRunnableOnUnfocused(true)
	g.texture = ebiten.NewImage(1280, 800)
	err := ebiten.RunGame(g)
	c.Report = map[string]any{"classification": "Linux desktop prototype; one source-matched asset only", "updates": g.updates, "draws": g.draws, "events": g.events, "hd_observation": c.Live, "physical_mouse_events": g.mouseEvents, "scripted_mouse": c.Scripted, "hd_enabled": g.enabled, "last_mode": g.mode, "assets": g.assetInfo, "virtual_speed": *desktopSpeed, "audio_output": "PCM receipt only; desktop playback not enabled", "listening_verified": false}
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

func (g *desktopGame) Update() error {
	g.updates++
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) && len(g.events) < 128 {
		before := g.c.Proof()
		g.enabled = !g.enabled
		f, err := g.saveFrame(fmt.Sprintf("switch-%d", len(g.events)))
		if err != nil {
			return err
		}
		after := g.c.Proof()
		same := reflect.DeepEqual(before, after)
		g.events = append(g.events, map[string]any{"kind": "toggle", "step": g.c.Steps(), "enabled": g.enabled, "paused": g.paused, "before": before, "after": after, "guest_unchanged": same, "frame": f})
		if !same {
			return fmt.Errorf("presentation changed original state")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) && len(g.events) < 128 {
		f, err := g.saveFrame(fmt.Sprintf("capture-%d", len(g.events)))
		if err != nil {
			return err
		}
		g.events = append(g.events, map[string]any{"kind": "capture", "step": g.c.Steps(), "paused": g.paused, "frame": f})
	}
	if !g.c.Scripted {
		x, y := ebiten.CursorPosition()
		buttons := 0
		if ebiten.IsFocused() && x >= 0 && y >= 0 && x < 1280 && y < 800 {
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				buttons |= 1
			}
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
				buttons |= 2
			}
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
				buttons |= 4
			}
			now := [3]int{x / 2, y / 2, buttons}
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
	if !g.paused {
		n := int((int64(g.updates)*1000000*int64(*desktopSpeed))/60 - (int64(g.updates-1)*1000000*int64(*desktopSpeed))/60)
		running = g.c.Advance(n)
	}
	if err := g.render(); err != nil {
		return err
	}
	g.texture.WritePixels(g.pixels.Pix)
	title := "原版"
	if g.mode == "hd" {
		title = "HD"
	}
	if g.paused {
		title += " 暫停"
	}
	ebiten.SetWindowTitle("德川家康｜" + title + "｜F6 切換 HD｜F7 暫停")
	status := map[string]any{"step": g.c.Steps(), "mode": g.mode, "eligible": g.c.Live.Active, "enabled": g.enabled, "paused": g.paused, "events": g.events, "mouse_events": g.mouseEvents}
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

func (g *desktopGame) Draw(screen *ebiten.Image)  { screen.DrawImage(g.texture, nil); g.draws++ }
func (g *desktopGame) Layout(_, _ int) (int, int) { return 1280, 800 }
