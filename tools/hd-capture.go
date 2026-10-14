// 研究輸出：在原版載入返回時複製平面與 DAC。沒有 guest 寫入。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func captureLoadedAsset(event map[string]any, mem []byte, palette [16]color.NRGBA) error {
	name, ok := event["name"].(string)
	if !ok || filepath.Base(name) != name || filepath.Ext(name) != ".OZM" {
		return fmt.Errorf("invalid original asset name")
	}
	w, wok := event["raw_columns"].(uint32)
	h, hok := event["raw_rows"].(uint32)
	planes, pok := event["planes"].([]map[string]any)
	if !wok || !hok || !pok || w < 1 || w > 80 || h < 1 || h > 400 || len(planes) != 4 {
		return fmt.Errorf("invalid original asset dimensions")
	}
	indices := make([]byte, int(w*8*h))
	for i, p := range planes {
		base, bok := p["base_linear"].(uint32)
		global, gok := p["pointer_global_linear"].(int)
		want, sok := p["sha256"].(string)
		if !bok || !gok || !sok || global != []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc}[i] || base < 0xc0000 || uint64(base)+uint64(h-1)*80+uint64(w) > uint64(len(mem)) || global+4 > len(mem) || binary.LittleEndian.Uint32(mem[global:]) != base {
			return fmt.Errorf("invalid original asset plane")
		}
		packed := make([]byte, 0, int(w*h))
		for y := uint32(0); y < h; y++ {
			row := mem[uint64(base)+uint64(y)*80 : uint64(base)+uint64(y)*80+uint64(w)]
			packed = append(packed, row...)
			for x := uint32(0); x < w*8; x++ {
				indices[y*w*8+x] |= (row[x/8] >> (7 - x%8) & 1) << i
			}
		}
		if fmt.Sprintf("%x", sha256.Sum256(packed)) != want {
			return fmt.Errorf("original asset plane changed")
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, int(w*8), int(h)))
	for i, index := range indices {
		img.SetNRGBA(i%int(w*8), i/int(w*8), palette[index])
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return err
	}
	key := fmt.Sprintf("%s-%x", name, sha256.Sum256(indices))
	root := "/output/hd-assets"
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	metadata := map[string]any{"classification": "normal-path loader-return capture; local only; draw and palette-at-display not yet verified", "event": event, "palette_at_loader_return": palette, "indices_sha256": fmt.Sprintf("%x", sha256.Sum256(indices)), "png_sha256": fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes())), "width": w * 8, "height": h}
	j, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	for ext, data := range map[string][]byte{".idx": indices, ".png": encoded.Bytes(), ".json": append(j, '\n')} {
		path := filepath.Join(root, key+ext)
		if old, err := os.ReadFile(path); err == nil {
			// 重載同圖時保留第一筆來源收據；索引／PNG 必須相同。
			if ext != ".json" && !bytes.Equal(old, data) {
				return fmt.Errorf("refusing changed capture: %s", path)
			}
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
