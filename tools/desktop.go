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
	"path/filepath"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var desktopSpeed = flag.Int("desktop-speed", 1, "desktop virtual-time speed, 1–60; validation acceleration only")
var desktopSource = flag.String("hd-source", "/project/workplace/hd-prototype/asset-capture/A-0-original.png", "verified local original A-0 PNG")
var desktopAsset = flag.String("hd-asset", "/project/assets/hd-b/A-0-hd-b-v1.png", "local single-asset HD PNG; missing asset falls back")
var desktopPortraitDir = flag.String("hd-portrait-dir", "/project/assets/hd-b/art", "reviewed source-specific story portraits from the private art archive")
var desktopKaoSourceDir = flag.String("hd-kao-source-dir", "/project/workplace/hd-prototype/catalog/kao", "verified original 80x80 portrait PNG directory")
var desktopKaoDir = flag.String("hd-kao-dir", "/project/workplace/hd-prototype/catalog/kao/selected", "local HD 80x80 portrait replacement directory")
var desktopScrollSource = flag.String("hd-scroll-source", "/project/workplace/hd-prototype/scroll/scroll-original.png", "verified local VISUAL scroll PNG")
var desktopScrollAsset = flag.String("hd-scroll-asset", "/project/workplace/hd-prototype/scroll/scroll-hd-b-v1.png", "local HD scroll PNG; missing asset falls back individually")
var desktopFrameSource = flag.String("hd-frame-source", "/project/workplace/hd-prototype/scroll/frame/frame-original.png", "verified original upper VISUAL frame PNG")
var desktopFrameAsset = flag.String("hd-frame-asset", "/project/workplace/hd-prototype/scroll/frame/frame-hd-b-v2.png", "local HD upper frame PNG; central story layer is preserved")
var desktopWindowSource = flag.String("hd-window-source", "/project/workplace/hd-prototype/catalog/window/WINDOW-original.png", "verified original nine-tile window PNG")
var desktopWindowAsset = flag.String("hd-window-asset", "/project/assets/hd-b/window/window-skin-hd-b-v1.png", "HD window nine-slice PNG; original foreground is preserved")
var desktopTerrainSourceDir = flag.String("hd-terrain-source-dir", "/project/workplace/hd-prototype/catalog/terrain/prepared/sources", "verified original floor PNGs with normal battle DAC")
var desktopTerrainDir = flag.String("hd-terrain-dir", "/project/assets/hd-b/terrain", "approved source-specific floor HD art; each missing item falls back")
var desktopTerrainSpriteSourceDir = flag.String("hd-terrain-sprite-source-dir", "/project/workplace/hd-prototype/catalog/terrain/objects/sources", "verified index-zero masks and native object/unit pixels")
var desktopTerrainTreeAsset = flag.String("hd-terrain-tree-asset", "/project/assets/hd-b/terrain/object-000-hd-b-v2.png", "approved transparent forest-clump HD art")
var desktopTerrainUnitDir = flag.String("hd-terrain-unit-dir", "/project/assets/hd-b/terrain", "approved source-specific unit HD art")
var desktopTerrainScene = flag.Bool("hd-terrain-scene", true, "compose the complete observed floor journal and qualified overlapping sprites")

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
	portraits                   map[string]desktopPortrait
	kao                         [81]desktopPortrait
	kaoModes                    map[int]string
	kaoHD                       int
	frame                       *hdFrame
	frameMode                   string
	window                      *hdWindow
	windowModes                 []string
	windowHD, windowForeground  int
	terrain                     map[int]hdTerrainPair
	terrainModes                []string
	terrainHD                   int
	terrainSprites              map[hdTerrainSpriteKey]*image.NRGBA
	terrainForeground           int
	terrainTree                 hdTerrainTreePair
	treeModes                   []string
	treeHD, treeProtected       int
	terrainUnits                map[hdTerrainSpriteKey]hdTerrainSpritePair
	scene                       hdTerrainScene
	sceneReport                 hdTerrainSceneReport
}

type desktopPortrait struct {
	Source, Asset image.Image
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
	if g.asset != nil {
		var reason string
		g.asset, reason = prepareHDPortrait(g.asset)
		if reason != "" {
			g.assetInfo["hd_error"] = reason
		}
	}
	g.portraits = map[string]desktopPortrait{}
	portraitInfo := map[string]any{}
	for name, profile := range hdPortraitProfiles {
		if name == "A-0.OZM" {
			continue
		}
		pair := desktopPortrait{}
		info := map[string]any{}
		if data, e := os.ReadFile(profile.Source); e == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			info["source_sha256"] = sum
			if sum == profile.SourceSHA {
				pair.Source, e = previewPNG(profile.Source)
				if e != nil {
					info["source_error"] = e.Error()
				}
			}
		}
		path := filepath.Join(*desktopPortraitDir, profile.Asset)
		if data, e := os.ReadFile(path); e == nil {
			info["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
			pair.Asset, e = previewPNG(path)
			if e != nil {
				info["hd_error"] = e.Error()
				pair.Asset = nil
			} else {
				var reason string
				pair.Asset, reason = prepareHDPortrait(pair.Asset)
				if reason != "" {
					info["hd_error"] = reason
				}
			}
		}
		g.portraits[name], portraitInfo[name] = pair, info
	}
	g.assetInfo["story_portraits"] = portraitInfo
	kaoInfo := map[int]any{}
	for index, profile := range hdKaoProfiles {
		pair := desktopPortrait{}
		info := map[string]any{}
		sourcePath := filepath.Join(*desktopKaoSourceDir, fmt.Sprintf("kao-%02d-original.png", index))
		if data, e := os.ReadFile(sourcePath); e == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			info["source_sha256"] = sum
			if sum == profile.SourceSHA {
				pair.Source, e = previewPNG(sourcePath)
				if e != nil {
					info["source_error"] = e.Error()
				}
			}
		}
		path := filepath.Join(*desktopKaoDir, fmt.Sprintf("kao-%02d-hd-b-v1.png", index))
		if data, e := os.ReadFile(path); e == nil {
			info["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
			pair.Asset, e = previewPNG(path)
			if e != nil {
				info["hd_error"] = e.Error()
				pair.Asset = nil
			} else {
				var reason string
				pair.Asset, reason = prepareHDImage(pair.Asset, 80, 80, 2)
				if reason != "" {
					info["hd_error"] = reason
				}
			}
		}
		g.kao[index], kaoInfo[index] = pair, info
	}
	g.assetInfo["kao"] = kaoInfo
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
	var frameSource, frameAsset image.Image
	frameInfo := map[string]any{}
	if data, e := os.ReadFile(*desktopFrameSource); e == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		frameInfo["source_sha256"] = sum
		if sum == "7686cb21cbf195893ac0f75e495fa304446fd6d1e4f41f1d78300890ffdb9116" {
			frameSource, e = previewPNG(*desktopFrameSource)
			if e != nil {
				frameInfo["source_error"] = e.Error()
			}
		}
	}
	if data, e := os.ReadFile(*desktopFrameAsset); e == nil {
		frameInfo["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
		frameAsset, e = previewPNG(*desktopFrameAsset)
		if e != nil {
			frameInfo["hd_error"] = e.Error()
			frameAsset = nil
		}
	}
	g.frame = newHDFrame(frameSource, frameAsset, 2)
	frameInfo["validation"] = g.frame.Reason
	g.assetInfo["frame"] = frameInfo
	var windowSource, windowAsset image.Image
	windowInfo := map[string]any{}
	if data, e := os.ReadFile(*desktopWindowSource); e == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		windowInfo["source_sha256"] = sum
		if sum == "8a4c3c581d76b131644f69733b61783fafde14541263cfde1d3dfe594d66978b" {
			windowSource, e = previewPNG(*desktopWindowSource)
			if e != nil {
				windowInfo["source_error"] = e.Error()
			}
		}
	}
	if data, e := os.ReadFile(*desktopWindowAsset); e == nil {
		windowInfo["hd_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
		windowAsset, e = previewPNG(*desktopWindowAsset)
		if e != nil {
			windowInfo["hd_error"] = e.Error()
			windowAsset = nil
		}
	}
	g.window = newHDWindow(windowSource, windowAsset, 2)
	windowInfo["validation"] = g.window.Reason
	g.assetInfo["window"] = windowInfo
	g.terrain = map[int]hdTerrainPair{}
	terrainInfo := map[int]any{}
	for index, profile := range hdTerrainProfiles {
		info := map[string]any{}
		var source, asset image.Image
		path := filepath.Join(*desktopTerrainSourceDir, fmt.Sprintf("floor-%03d-original.png", index))
		if data, e := os.ReadFile(path); e == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			info["source_sha256"] = sum
			if sum == profile.SourceSHA {
				source, e = previewPNG(path)
				if e != nil {
					info["source_error"] = e.Error()
				}
			}
		}
		if profile.Asset != "" {
			path = filepath.Join(*desktopTerrainDir, profile.Asset)
			if data, e := os.ReadFile(path); e == nil {
				sum := fmt.Sprintf("%x", sha256.Sum256(data))
				info["hd_sha256"] = sum
				if sum == profile.AssetSHA {
					asset, e = previewPNG(path)
					if e != nil {
						info["hd_error"] = e.Error()
					}
				} else {
					info["hd_error"] = "unapproved HD fingerprint"
				}
			}
		}
		pair := newHDTerrainPair(source, asset, 2)
		info["validation"] = pair.Reason
		g.terrain[index], terrainInfo[index] = pair, info
	}
	g.assetInfo["terrain"] = terrainInfo
	g.terrainSprites = map[hdTerrainSpriteKey]*image.NRGBA{}
	spriteInfo := map[string]any{}
	for _, profile := range hdTerrainSpriteProfiles {
		key := hdTerrainSpriteKey{profile.Group, profile.Index}
		name := fmt.Sprintf("%s-%03d-original.png", profile.Group, profile.Index)
		path := filepath.Join(*desktopTerrainSpriteSourceDir, name)
		info := map[string]any{"validation": "fallback_missing_source"}
		if data, e := os.ReadFile(path); e == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			info["source_sha256"] = sum
			if sum == profile.SourceSHA {
				if source, e := previewPNG(path); e == nil {
					if validated := newHDTerrainSpriteSource(source); validated != nil {
						g.terrainSprites[key] = validated
						info["validation"] = "ready_native_mask"
					}
				}
			}
		}
		spriteInfo[fmt.Sprintf("%s-%03d", profile.Group, profile.Index)] = info
	}
	g.assetInfo["terrain_sprite_sources"] = spriteInfo
	treeInfo := map[string]any{}
	var treeSource image.Image
	if source := g.terrainSprites[hdTerrainSpriteKey{"objects", 0}]; source != nil {
		treeSource = source
	}
	if data, e := os.ReadFile(filepath.Join(*desktopTerrainSpriteSourceDir, "objects-000-original.png")); e == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		treeInfo["source_sha256"] = sum
		if sum != hdTerrainTreeSourceSHA {
			treeSource = nil
		}
	} else {
		treeSource = nil
	}
	var treeAsset image.Image
	if data, e := os.ReadFile(*desktopTerrainTreeAsset); e == nil {
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		treeInfo["hd_sha256"] = sum
		if sum == hdTerrainTreeAssetSHA {
			treeAsset, _ = previewPNG(*desktopTerrainTreeAsset)
		} else {
			treeInfo["hd_error"] = "unapproved tree fingerprint"
		}
	}
	g.terrainTree = newHDTerrainTreePair(treeSource, treeAsset, 2)
	treeInfo["validation"] = g.terrainTree.Reason
	g.assetInfo["terrain_tree"] = treeInfo
	g.terrainUnits = map[hdTerrainSpriteKey]hdTerrainSpritePair{}
	unitInfo := map[int]any{}
	for index, profile := range hdTerrainUnitProfiles {
		key := hdTerrainSpriteKey{"units", index}
		info := map[string]any{}
		if entry, ok := spriteInfo[fmt.Sprintf("units-%03d", index)].(map[string]any); ok {
			if sum, ok := entry["source_sha256"]; ok {
				info["source_sha256"] = sum
			}
		}
		var source, asset image.Image
		if im := g.terrainSprites[key]; im != nil {
			source = im
		}
		path := filepath.Join(*desktopTerrainUnitDir, profile.Asset)
		if data, e := os.ReadFile(path); e == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			info["hd_sha256"] = sum
			if sum == profile.AssetSHA {
				asset, _ = previewPNG(path)
			} else {
				info["hd_error"] = "unapproved unit fingerprint"
			}
		}
		pair := newHDTerrainSpritePair(source, asset, 2, key, profile.SourceBounds, profile.Crop)
		info["validation"] = pair.Reason
		g.terrainUnits[key], unitInfo[index] = pair, info
	}
	g.assetInfo["terrain_units"] = unitInfo
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("德川家康｜原版｜F6 切換 HD｜F7 暫停")
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	ebiten.SetRunnableOnUnfocused(true)
	g.texture = ebiten.NewImage(1280, 800)
	g.barTexture = ebiten.NewImage(640, barHeight+2*barRowHeight)
	err = ebiten.RunGame(g)
	c.Report = map[string]any{"classification": "desktop prototype; source-matched story portraits, scroll, upper frame and KAO portraits", "updates": g.updates, "draws": g.draws, "events": g.events, "windowbar_events": g.barEvents, "hd_observation": c.Live, "physical_mouse_events": g.mouseEvents, "scripted_mouse": c.Scripted, "hd_enabled": g.enabled, "last_mode": g.mode, "portrait_mode": g.portraitMode, "scroll_mode": g.scrollMode, "frame_mode": g.frameMode, "scroll_foreground_pixels": g.scrollForeground, "kao_modes": g.kaoModes, "kao_hd_count": g.kaoHD, "assets": g.assetInfo, "virtual_speed": *desktopSpeed, "audio_output": "PCM receipt only; desktop playback not enabled", "listening_verified": false}
	c.Report["window_modes"], c.Report["window_hd_count"], c.Report["window_foreground_pixels"] = g.windowModes, g.windowHD, g.windowForeground
	c.Report["terrain_modes"], c.Report["terrain_hd_count"] = g.terrainModes, g.terrainHD
	c.Report["terrain_foreground_pixels"] = g.terrainForeground
	c.Report["tree_modes"], c.Report["tree_hd_count"], c.Report["tree_protected_pixels"] = g.treeModes, g.treeHD, g.treeProtected
	c.Report["terrain_scene"], c.Report["terrain_scene_enabled"] = g.sceneReport, *desktopTerrainScene
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
	source, asset := g.source, g.asset
	if g.c.Live.LastName != "A-0.OZM" {
		pair := g.portraits[g.c.Live.LastName]
		source, asset = pair.Source, pair.Asset
	}
	out, mode, err := previewA0(s.Image, source, asset, 2, g.enabled && g.c.Live.Active)
	if err != nil {
		return err
	}
	if g.enabled && !g.c.Live.Active {
		mode = "fallback_no_original_draw"
	}
	g.portraitMode = mode
	if *desktopTerrainScene {
		pairs := map[hdTerrainSpriteKey]hdTerrainSpritePair{{"objects", 0}: g.terrainTree}
		for key, pair := range g.terrainUnits {
			pairs[key] = pair
		}
		g.sceneReport = g.scene.apply(out, s.Image, g.c.Live.Terrain, g.terrain, pairs, g.terrainSprites, 2, g.enabled)
		g.terrainModes, g.terrainHD, g.terrainForeground = g.sceneReport.FloorModes, g.sceneReport.FloorHD, g.sceneReport.ProtectedNativePixels
		g.treeModes = append(g.treeModes[:0], g.sceneReport.SpriteModes...)
		for i, d := range g.c.Live.Terrain.Sprites {
			if d.Group == "units" {
				g.treeModes[i] = "original_unit_separate_layer"
			}
		}
		g.treeHD, g.treeProtected = g.sceneReport.TreeHD, g.sceneReport.ProtectedNativePixels
		if g.sceneReport.Reason == "hd" {
			mode = "hd"
		}
	} else {
		g.terrainModes, g.terrainHD, g.terrainForeground = nil, 0, 0
		for _, draw := range g.c.Live.Terrain.Draws {
			result, foreground := "fallback_pending_draw", 0
			if g.c.Live.Terrain.Pending == nil {
				result, foreground = g.terrain[draw.Index].applyForeground(out, s.Image, draw, g.c.Live.Terrain.Sprites, g.terrainSprites, 2, g.enabled)
			}
			g.terrainModes = append(g.terrainModes, result)
			if result == "hd" {
				g.terrainHD++
				g.terrainForeground += foreground
				mode = "hd"
			}
		}
		g.treeModes, g.treeHD, g.treeProtected = nil, 0, 0
		var replacedTrees []image.Rectangle
		for _, sprite := range g.c.Live.Terrain.Sprites {
			result, protected := g.terrainTree.apply(out, s.Image, sprite, g.c.Live.Terrain, g.terrain, g.terrainModes, g.terrainSprites, replacedTrees, 2, g.enabled)
			g.treeModes = append(g.treeModes, result)
			if result == "hd" {
				replacedTrees = append(replacedTrees, sprite.Rect)
				g.treeHD++
				g.treeProtected += protected
				mode = "hd"
			}
		}
	}
	if mode == "hd" {
		preservePortraitLabels(out, s.Image, g.c.Live.LastName, 2)
	}
	g.frameMode = g.frame.apply(out, s.Image, g.enabled && g.c.Live.ScrollActive)
	if g.enabled && !g.c.Live.ScrollActive {
		g.frameMode = "fallback_no_original_background"
	}
	if g.frameMode == "hd" {
		mode = "hd"
	}
	g.scrollMode, g.scrollForeground = g.scroll.apply(out, s.Image, g.enabled && g.c.Live.ScrollActive)
	if g.enabled && !g.c.Live.ScrollActive {
		g.scrollMode = "fallback_no_original_background"
	}
	if g.scrollMode == "hd" {
		mode = "hd"
	}
	g.windowModes, g.windowHD, g.windowForeground = nil, 0, 0
	for _, draw := range g.c.Live.Window.Draws {
		result, foreground := g.window.apply(out, s.Image, draw.Rect, g.enabled)
		g.windowModes = append(g.windowModes, result)
		g.windowForeground += foreground
		if result == "hd" {
			g.windowHD++
			mode = "hd"
		}
	}
	g.kaoModes, g.kaoHD = map[int]string{}, 0
	for _, draw := range g.c.Live.Kao.Draws {
		pair := g.kao[draw.Index]
		result := previewKao(out, s.Image, pair.Source, pair.Asset, draw.Rect, 2, g.enabled)
		g.kaoModes[draw.Index] = result
		if result == "hd" {
			g.kaoHD++
			mode = "hd"
		}
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
	status := map[string]any{"step": g.c.Steps(), "mode": g.mode, "eligible": g.c.Live.Active, "portrait_resource": g.c.Live.LastName, "scroll_eligible": g.c.Live.ScrollActive, "scroll_mode": g.scrollMode, "frame_mode": g.frameMode, "scroll_foreground_pixels": g.scrollForeground, "portrait_mode": g.portraitMode, "kao_modes": g.kaoModes, "kao_hd_count": g.kaoHD, "kao_draws": g.c.Live.Kao.Draws, "enabled": g.enabled, "paused": g.paused, "events": g.events, "mouse_events": g.mouseEvents, "bar_visible": g.bar.Visible, "bar_pinned": g.bar.Pinned, "bar_menu": g.bar.Menu, "bar_focus": g.bar.Focus, "bar_hover": g.bar.Hover, "bar_events": g.barEvents, "window_size": []int{w, h}, "mouse_blocked": g.blockMouse}
	status["window_modes"], status["window_hd_count"], status["window_foreground_pixels"], status["window_draws"] = g.windowModes, g.windowHD, g.windowForeground, g.c.Live.Window.Draws
	status["terrain_modes"], status["terrain_hd_count"], status["terrain_draws"] = g.terrainModes, g.terrainHD, g.c.Live.Terrain.Draws
	status["terrain_foreground_pixels"], status["terrain_sprites"] = g.terrainForeground, g.c.Live.Terrain.Sprites
	status["tree_modes"], status["tree_hd_count"], status["tree_protected_pixels"], status["tree_draws"] = g.treeModes, g.treeHD, g.treeProtected, g.c.Live.Terrain.Sprites
	status["terrain_scene"], status["terrain_scene_enabled"], status["terrain_history"] = g.sceneReport, *desktopTerrainScene, g.c.Live.Terrain.History
	if *desktopTerrainScene {
		status["terrain_draws"] = g.c.Live.Terrain.History
	}
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
