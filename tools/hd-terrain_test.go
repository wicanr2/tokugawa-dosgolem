package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
)

func terrainMemory(t *testing.T) []byte {
	t.Helper()
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx")
	if err != nil {
		t.Skip("local original atlas unavailable")
	}
	if len(idx) != 640*400 {
		t.Fatal("atlas shape")
	}
	mem := make([]byte, 0x120000)
	for bit, global := range []int{0x42174, 0x42168, 0x42170, 0x4216c} {
		base := 0x100000 + bit*0x8000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		for n, index := range idx {
			if index&(1<<bit) != 0 {
				mem[base+n/8] |= 1 << uint(7-n%8)
			}
		}
	}
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe2a)
	binary.LittleEndian.PutUint32(mem[0x3598c:], 0x5000)
	binary.LittleEndian.PutUint32(mem[0x35990:], 0x6e00)
	return mem
}

func TestTerrainSpriteNaturalOrderBoundsAndInvalidation(t *testing.T) {
	mem := terrainMemory(t)
	before := sha256.Sum256(mem)
	var h hdTerrainState
	h.observe(1, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	h.observe(2, 0x1fe2a, 0x1004, 0, 0, mem)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe69)
	h.observe(3, 0x1f9d0, 0x1000, 0x5000, 36*80+8, mem)
	if h.Pending == nil || len(h.Sprites) != 0 {
		t.Fatal("sprite early activation")
	}
	h.observe(4, 0x1fe69, 0x1000, 0, 0, mem)
	if h.Pending == nil {
		t.Fatal("wrong return stack")
	}
	h.observe(5, 0x1fe69, 0x1004, 0, 0, mem)
	if len(h.Sprites) != 1 || h.Sprites[0].Group != "objects" || h.Sprites[0].Index != 0 || h.Sprites[0].Order <= h.Draws[0].Order || h.SpriteReturns != 1 {
		t.Fatal("sprite source/order")
	}
	// 部分重畫地面不能刪掉仍覆蓋其他格的前景。
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe2a)
	h.observe(6, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	h.observe(7, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(h.Sprites) != 1 || h.Draws[0].Order <= h.Sprites[0].Order {
		t.Fatal("foreground lifetime/order")
	}
	// 只有觀測外的測試設定可改 RAM。
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe2a)
	if before != sha256.Sum256(mem) {
		t.Fatal("observer wrote guest")
	}
	h.observe(8, 0x1e93f, 0, 0, 0, mem)
	if len(h.Draws)+len(h.Sprites) != 0 || h.Pending != nil {
		t.Fatal("allocation retained stale layers")
	}
	for _, tc := range [][3]uint32{{0x5000, 0, 0x1fe69}, {0x5f00 + 72, 0, 0x1fe69}, {0x6e00 + 72, 0, 0x1fea2}} {
		if _, ok := hdTerrainSpriteDecode(tc[0], tc[1], tc[2]); !ok {
			t.Fatal("valid sprite rejected", tc)
		}
	}
	for _, dest := range []uint32{60*80 + 30, 60*80 + 50} {
		for index := uint32(0); index < 13; index++ {
			d, ok := hdTerrainSpriteDecode(0x6e00+index*6, dest, 0x1f9be)
			if !ok || d.Group != "units" || d.Index != int(index) || d.CombatReturn != 0x1f9be {
				t.Fatal("combat wrapper source", index, dest)
			}
		}
	}
	for _, tc := range [][3]uint32{{0x6e00, 60*80 + 29, 0x1f9be}, {0x6e00, 61*80 + 30, 0x1f9be}, {0x5000, 60*80 + 30, 0x1f9be}, {0x6e00 + 78, 60*80 + 30, 0x1f9be}} {
		if _, ok := hdTerrainSpriteDecode(tc[0], tc[1], tc[2]); ok {
			t.Fatal("unapproved combat wrapper", tc)
		}
	}
	for _, tc := range [][3]uint32{{0x5000, 0, 0x1fea2}, {0x6e00, 0, 0x1fe69}, {0x5000 + 78, 0, 0x1fe69}, {0x5050, 0, 0x1fe69}, {0x5000, 353 * 80, 0x1fe69}, {0x5000, 75, 0x1fe69}, {0xffffffff, 0, 0x1fe69}, {0x5000, 0xffffffff, 0x1fe69}} {
		if _, ok := hdTerrainSpriteDecode(tc[0], tc[1], tc[2]); ok {
			t.Fatal("invalid sprite accepted", tc)
		}
	}
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe69)
	h.Sprites = make([]hdTerrainSpriteDraw, 512)
	h.observe(9, 0x1f9d0, 0x1000, 0x5000, 0, mem)
	h.observe(10, 0x1fe69, 0x1004, 0, 0, mem)
	if len(h.Sprites) != 0 || h.Pending != nil {
		t.Fatal("foreground cap retained stale layers")
	}
	h.observe(11, 0x1f9d0, 0x1000, 0x5000, 0, mem)
	mem[0x100000] ^= 1
	h.observe(12, 0x1fe69, 0x1004, 0, 0, mem)
	if len(h.Sprites) != 0 || h.Pending != nil {
		t.Fatal("changed source completed")
	}
}

func TestTerrainForegroundSameColourHolesOrderAndUnknownOverlay(t *testing.T) {
	floor := image.NewNRGBA(image.Rect(0, 0, 48, 32))
	hd := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	grass, painted, ink := color.NRGBA{44, 215, 44, 255}, color.NRGBA{77, 99, 66, 255}, color.NRGBA{32, 32, 32, 255}
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			floor.SetNRGBA(x, y, grass)
		}
	}
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			hd.SetNRGBA(x, y, painted)
		}
	}
	pair := newHDTerrainPair(floor, hd, 2)
	mask := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	mask.SetNRGBA(0, 0, grass) // 同色仍是前景。
	mask.SetNRGBA(1, 0, ink)
	mask.SetNRGBA(3, 0, color.NRGBA{0, 0, 0, 255}) // 非零索引的黑色必須保留。
	key := hdTerrainSpriteKey{"objects", 0}
	sources := map[hdTerrainSpriteKey]*image.NRGBA{key: newHDTerrainSpriteSource(mask)}
	draw := hdTerrainDraw{Rect: image.Rect(80, 80, 128, 112), Order: 1}
	sprite := hdTerrainSpriteDraw{Group: "objects", Index: 0, Rect: image.Rect(80, 80, 128, 128), Order: 2}
	native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			native.SetNRGBA(80+x, 80+y, grass)
		}
	}
	native.SetNRGBA(81, 80, ink)
	native.SetNRGBA(83, 80, color.NRGBA{0, 0, 0, 255})
	before := append([]byte(nil), native.Pix...)
	out, _, _ := previewA0(native, nil, nil, 2, false)
	reason, n := pair.applyForeground(out, native, draw, []hdTerrainSpriteDraw{sprite}, sources, 2, true)
	if reason != "hd" || n != 3 || out.NRGBAAt(160, 160) != grass || out.NRGBAAt(162, 160) != ink || out.NRGBAAt(164, 160) != painted || out.NRGBAAt(166, 160) != (color.NRGBA{0, 0, 0, 255}) || !bytes.Equal(before, native.Pix) {
		t.Fatal("alpha/colour/holes", reason, n)
	}
	// 晚畫前景依原版順序覆蓋早畫前景。
	second := image.NewNRGBA(mask.Bounds())
	second.SetNRGBA(0, 0, ink)
	sources[hdTerrainSpriteKey{"units", 1}] = second
	native.SetNRGBA(80, 80, ink)
	other := hdTerrainSpriteDraw{Group: "units", Index: 1, Rect: sprite.Rect, Order: 3}
	if reason, n = pair.applyForeground(out, native, draw, []hdTerrainSpriteDraw{sprite, other}, sources, 2, true); reason != "hd" || n != 3 || out.NRGBAAt(160, 160) != ink {
		t.Fatal("layer order", reason, n)
	}
	// 若地面晚於前景，舊前景不得重新浮現。
	draw.Order = 4
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			native.SetNRGBA(80+x, 80+y, grass)
		}
	}
	if reason, n = pair.applyForeground(out, native, draw, []hdTerrainSpriteDraw{sprite, other}, sources, 2, true); reason != "hd" || n != 0 || out.NRGBAAt(160, 160) != painted {
		t.Fatal("overwritten foreground restored", reason, n)
	}
	draw.Order = 1
	out, _, _ = previewA0(native, nil, nil, 2, false)
	unchanged := append([]byte(nil), out.Pix...)
	if reason, _ = pair.applyForeground(out, native, draw, []hdTerrainSpriteDraw{sprite}, nil, 2, true); reason != "fallback_foreground_source" || !bytes.Equal(unchanged, out.Pix) {
		t.Fatal("missing mask changed presentation")
	}
	native.SetNRGBA(84, 80, ink)
	native.SetNRGBA(81, 80, ink)
	native.SetNRGBA(83, 80, color.NRGBA{0, 0, 0, 255})
	out, _, _ = previewA0(native, nil, nil, 2, false)
	unchanged = append([]byte(nil), out.Pix...)
	if reason, _ = pair.applyForeground(out, native, draw, []hdTerrainSpriteDraw{sprite}, sources, 2, true); reason != "fallback_source_mismatch" || !bytes.Equal(unchanged, out.Pix) {
		t.Fatal("unknown overlay not protected")
	}
	mask.SetNRGBA(0, 0, color.NRGBA{44, 215, 44, 128})
	if newHDTerrainSpriteSource(mask) != nil {
		t.Fatal("partial native alpha accepted")
	}
}

func TestTerrainRealSpriteMasksAndNormalComposition(t *testing.T) {
	root := "/project/workplace/hd-prototype/catalog/terrain/objects/"
	data, err := os.ReadFile(root + "source-profiles.json")
	if err != nil {
		t.Skip("local originals unavailable")
	}
	var profiles struct {
		Records []struct {
			Group string `json:"group"`
			Index int    `json:"index"`
			SHA   string `json:"source_png_sha256"`
		} `json:"records"`
	}
	if json.Unmarshal(data, &profiles) != nil || len(profiles.Records) != 39 || len(hdTerrainSpriteProfiles) != 39 {
		t.Fatal("39 native mask profiles")
	}
	idx, err := os.ReadFile("/project/workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx")
	if err != nil {
		t.Fatal(err)
	}
	sources := map[hdTerrainSpriteKey]*image.NRGBA{}
	for n, p := range profiles.Records {
		path := fmt.Sprintf("%ssources/%s-%03d-original.png", root, p.Group, p.Index)
		data, err := os.ReadFile(path)
		compiled := hdTerrainSpriteProfiles[n]
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.SHA || compiled.SourceSHA != p.SHA || compiled.Group != p.Group || compiled.Index != p.Index {
			t.Fatal("sprite fingerprint", p.Group, p.Index)
		}
		im, err := previewPNG(path)
		if err != nil {
			t.Fatal(err)
		}
		source := newHDTerrainSpriteSource(im)
		if source == nil {
			t.Fatal("native mask shape/alpha")
		}
		y0 := 256 + p.Index/13*48
		if p.Group == "units" {
			y0 = 352
		}
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				opaque := source.NRGBAAt(x, y).A == 255
				want := idx[(y0+y)*640+p.Index%13*48+x] != 0
				if opaque != want {
					t.Fatal("index-zero mask", p.Group, p.Index, x, y)
				}
			}
		}
		sources[hdTerrainSpriteKey{p.Group, p.Index}] = source
	}
	data, err = os.ReadFile(root + "trace/verification.json")
	if err != nil {
		t.Fatal(err)
	}
	type traced struct {
		Group      string `json:"group"`
		Index      int    `json:"index"`
		Rect       [4]int `json:"rect"`
		Order      uint64 `json:"order"`
		Foreground int    `json:"foreground_pixels"`
	}
	var evidence struct {
		Floors  []traced `json:"current_floors"`
		Sprites []traced `json:"sprites"`
		Matched []traced `json:"matched_rectangles"`
	}
	if json.Unmarshal(data, &evidence) != nil || len(evidence.Matched) != 64 || len(evidence.Sprites) != 11 {
		t.Fatal("normal trace fixtures")
	}
	native, err := previewPNG("/project/workplace/startup-probe/frame-b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846.png")
	if err != nil {
		t.Fatal(err)
	}
	var sprites []hdTerrainSpriteDraw
	for _, s := range evidence.Sprites {
		sprites = append(sprites, hdTerrainSpriteDraw{Group: s.Group, Index: s.Index, Rect: image.Rect(s.Rect[0], s.Rect[1], s.Rect[2], s.Rect[3]), Order: s.Order})
	}
	out, _, _ := previewA0(native, nil, nil, 2, false)
	count, foreground := 0, 0
	for _, d := range evidence.Floors {
		source, err := previewPNG(fmt.Sprintf("/project/workplace/hd-prototype/catalog/terrain/prepared/sources/floor-%03d-original.png", d.Index))
		if err != nil {
			t.Fatal(err)
		}
		// 使用固定純色 HD 區分呈現，驗證來源與遮罩，不把測試圖當成已製作美術。
		asset := image.NewNRGBA(image.Rect(0, 0, 96, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 96; x++ {
				asset.SetNRGBA(x, y, color.NRGBA{77, 99, 66, 255})
			}
		}
		pair := newHDTerrainPair(source, asset, 2)
		draw := hdTerrainDraw{Index: d.Index, Rect: image.Rect(d.Rect[0], d.Rect[1], d.Rect[2], d.Rect[3]), Order: d.Order}
		reason, n := pair.applyForeground(out, native, draw, sprites, sources, 2, true)
		if reason == "hd" {
			count++
			foreground += n
		} else if reason != "fallback_source_mismatch" {
			t.Fatal("normal compose", reason)
		}
	}
	if count != 64 || foreground != 6202 {
		t.Fatal("normal source/foreground cases", count, foreground)
	}
}

func TestTerrainOriginalPlanesReturnsAndInvalidation(t *testing.T) {
	mem := terrainMemory(t)
	before := sha256.Sum256(mem)
	var state hdTerrainState
	state.observe(1, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	if state.Pending == nil || len(state.Draws) != 0 {
		t.Fatal("early activation")
	}
	state.observe(2, 0x1fe2a, 0x1000, 0, 0, mem)
	if state.Pending == nil {
		t.Fatal("wrong stack completed")
	}
	state.observe(3, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 1 || state.Draws[0].Index != 0 || state.Draws[0].Rect != image.Rect(64, 56, 112, 88) {
		t.Fatal("natural floor draw")
	}
	state.observe(4, 0x1fc07, 0x1000, 6, 56*80+9, mem)
	if len(state.Draws) != 0 {
		t.Fatal("overlap not invalidated before return")
	}
	state.observe(5, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 1 || state.Draws[0].Index != 1 {
		t.Fatal("replacement")
	}
	state.observe(6, 0x1c084, 0, 0, 0, mem)
	if len(state.Draws) != 0 || state.Pending != nil {
		t.Fatal("full frame retained stale ground")
	}
	if before != sha256.Sum256(mem) {
		t.Fatal("observer wrote original RAM")
	}
	state.observe(7, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	mem[0x100000] ^= 1
	state.observe(8, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(state.Draws) != 0 {
		t.Fatal("changed plane accepted at return")
	}
	mem[0x100000] ^= 1
	binary.LittleEndian.PutUint32(mem[0x35984:], 1)
	binary.LittleEndian.PutUint32(mem[0x4217c:], 1)
	state.observe(9, 0x1fc07, 0x1000, 0, 56*80+8, mem)
	if state.Pending != nil {
		t.Fatal("original checker mask accepted")
	}
	for _, pair := range [][2]uint32{{78, 0}, {256 * 80, 0}, {1, 0}, {80, 0}, {0, 75}, {0, 399 * 80}, {0xffffffff, 0}, {0, 0xffffffff}} {
		if _, ok := hdTerrainDecode(pair[0], pair[1]); ok {
			t.Fatal("invalid geometry", pair)
		}
	}
}

func TestTerrainPreparedSourcesAndFullCellProtection(t *testing.T) {
	data, err := os.ReadFile("/project/workplace/hd-prototype/catalog/terrain/prepared/profiles.json")
	if err != nil {
		t.Skip("local floor fixtures unavailable")
	}
	var manifest struct {
		ToolSHA string `json:"tool_sha256"`
		Records []struct {
			Index int    `json:"index"`
			SHA   string `json:"source_png_sha256"`
		} `json:"records"`
	}
	if err = json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	tool, err := os.ReadFile("/project/tools/hd-terrain-prepare.py")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(tool)) != manifest.ToolSHA || len(manifest.Records) != 104 || len(hdTerrainProfiles) != 104 {
		t.Fatal("stale floor preparation")
	}
	approved := 0
	for _, r := range manifest.Records {
		t.Run(fmt.Sprint(r.Index), func(t *testing.T) {
			p := hdTerrainProfiles[r.Index]
			path := fmt.Sprintf("/project/workplace/hd-prototype/catalog/terrain/prepared/sources/floor-%03d-original.png", r.Index)
			data, err := os.ReadFile(path)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.SourceSHA || p.SourceSHA != r.SHA {
				t.Fatal("source fingerprint")
			}
			source, err := previewPNG(path)
			if err != nil {
				t.Fatal(err)
			}
			native := image.NewNRGBA(image.Rect(0, 0, 640, 400))
			rect := image.Rect(80, 80, 128, 112)
			for y := 0; y < 32; y++ {
				for x := 0; x < 48; x++ {
					native.SetNRGBA(80+x, 80+y, rgbaAt(source, x, y))
				}
			}
			before := append([]byte(nil), native.Pix...)
			out, _, _ := previewA0(native, nil, nil, 2, false)
			original := append([]byte(nil), out.Pix...)
			missing := newHDTerrainPair(source, nil, 2)
			if missing.apply(out, native, rect, 2, true) != "fallback_missing_asset" || !bytes.Equal(original, out.Pix) {
				t.Fatal("missing floor changed presentation")
			}
			if p.Asset == "" {
				return
			}
			approved++
			path = "/project/assets/hd-b/terrain/" + p.Asset
			data, err = os.ReadFile(path)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != p.AssetSHA {
				t.Fatal("HD fingerprint")
			}
			asset, err := previewPNG(path)
			if err != nil {
				t.Fatal(err)
			}
			pair := newHDTerrainPair(source, asset, 2)
			if pair.apply(out, native, rect, 2, true) != "hd" || bytes.Equal(original, out.Pix) || !bytes.Equal(before, native.Pix) {
				t.Fatal("valid HD floor or original immutability")
			}
			for y := 0; y < 800; y++ {
				for x := 0; x < 1280; x++ {
					if x >= 160 && x < 256 && y >= 160 && y < 224 {
						continue
					}
					i := (y*1280 + x) * 4
					if !bytes.Equal(out.Pix[i:i+4], original[i:i+4]) {
						t.Fatal("outside floor changed")
					}
				}
			}
			native.Pix[(90*640+90)*4] ^= 1
			out, _, _ = previewA0(native, nil, nil, 2, false)
			occluded := append([]byte(nil), out.Pix...)
			if pair.apply(out, native, rect, 2, true) != "fallback_source_mismatch" || !bytes.Equal(occluded, out.Pix) {
				t.Fatal("occluded floor did not keep entire original cell")
			}
		})
	}
	if approved != 104 {
		t.Fatal("reviewed source count", approved)
	}
}
