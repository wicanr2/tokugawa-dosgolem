// 固定正常玩家路徑的原版背景擷取，不是獨立 OZM 解碼器。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

func scrollExtractSHA(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func scrollExtractRead(p string) []byte {
	b, e := os.ReadFile(p)
	if e != nil {
		panic(e)
	}
	return b
}
func scrollExtractRequire(ok bool, s string) {
	if !ok {
		panic(s)
	}
}

func main() {
	_, e := os.Stat("/.dockerenv")
	scrollExtractRequire(e == nil, "只允許在 Docker 執行")
	root := "/output"
	raw := scrollExtractRead(filepath.Join(root, "startup-probe/latest.json"))
	var receipt struct {
		Steps    uint64 `json:"successful_steps"`
		Stop     string `json:"stop"`
		InputSHA string `json:"input_sha256"`
		MouseSHA string `json:"mouse_input_sha256"`
		Trace    struct {
			Events []struct {
				Kind, Name string
				Columns    uint32 `json:"raw_columns"`
				Rows       uint32 `json:"raw_rows"`
				Pitch      uint32 `json:"pitch_bytes"`
				Planes     []struct {
					Base   uint32 `json:"base_linear"`
					Global uint32 `json:"pointer_global_linear"`
					SHA    string `json:"sha256"`
				}
			}
		} `json:"hd_asset_trace"`
	}
	if e = json.Unmarshal(raw, &receipt); e != nil {
		panic(e)
	}
	scrollExtractRequire(receipt.Steps == 136769993 && receipt.Stop == "instruction_budget", "停止條件不符")
	scrollExtractRequire(receipt.InputSHA == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a", "EXE 不符")
	scrollExtractRequire(receipt.MouseSHA == "c31fc431659940533d5f7046a571268e6bbf94d4b1835360b7b9d4c6c2c4f7a9", "普通輸入前綴不符")
	events := receipt.Trace.Events
	scrollExtractRequire(len(events) > 0, "缺少載入事件")
	last := events[len(events)-1]
	scrollExtractRequire(last.Kind == "load_return" && last.Name == "VISUAL.OZM" && last.Columns == 80 && last.Rows == 400 && last.Pitch == 80 && len(last.Planes) == 4, "最後載入不符")
	ram := scrollExtractRead(filepath.Join(root, "startup-memory.bin"))
	indices := make([]byte, 640*400)
	for i, p := range last.Planes {
		scrollExtractRequire(p.Base >= 0xc0000 && uint64(p.Base)+32000 <= uint64(len(ram)), "平面超界")
		scrollExtractRequire(p.Global == []uint32{0x38c00, 0x38bf8, 0x38c04, 0x38bfc}[i] && binary.LittleEndian.Uint32(ram[p.Global:]) == p.Base, "平面指標不符")
		data := ram[p.Base : p.Base+32000]
		scrollExtractRequire(scrollExtractSHA(data) == p.SHA, "原版平面雜湊不符")
		for y := 0; y < 400; y++ {
			for x := 0; x < 640; x++ {
				indices[y*640+x] |= ((data[y*80+x/8] >> uint(7-x%8)) & 1) << uint(i)
			}
		}
	}
	// 由已驗證的同版本 150M 正常畫面取得 RGB，排除肖像與文字前景。
	oldRoot := "/project/workplace/hd-prototype/asset-capture"
	oldRaw := scrollExtractRead(filepath.Join(oldRoot, "startup-probe/latest.json"))
	var old struct {
		Frame struct {
			Path string
			SHA  string `json:"png_sha256"`
		}
		Steps uint64 `json:"successful_steps"`
	}
	if e = json.Unmarshal(oldRaw, &old); e != nil {
		panic(e)
	}
	scrollExtractRequire(old.Steps == 150000000, "RGB 基準步數不符")
	frameBytes := scrollExtractRead(filepath.Join(oldRoot, "startup-probe", filepath.Base(old.Frame.Path)))
	scrollExtractRequire(scrollExtractSHA(frameBytes) == old.Frame.SHA, "RGB 基準雜湊不符")
	frame, e := png.Decode(bytes.NewReader(frameBytes))
	if e != nil {
		panic(e)
	}
	scrollExtractRequire(frame.Bounds() == image.Rect(0, 0, 640, 400), "畫面尺寸不符")
	portrait, paper := image.Rect(160, 50, 480, 200), image.Rect(80, 268, 590, 382)
	var palette [16]color.NRGBA
	var seen [16]bool
	matched := 0
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			if image.Pt(x, y).In(portrait) || image.Pt(x, y).In(paper) {
				continue
			}
			i := indices[y*640+x]
			c := color.NRGBAModel.Convert(frame.At(x, y)).(color.NRGBA)
			scrollExtractRequire(!seen[i] || palette[i] == c, fmt.Sprintf("背景與原版色號衝突 (%d,%d)", x, y))
			palette[i], seen[i] = c, true
			matched++
		}
	}
	visual := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			i := indices[y*640+x]
			scrollExtractRequire(seen[i], "背景色號未觀測")
			visual.SetNRGBA(x, y, palette[i])
		}
	}
	scroll := image.NewNRGBA(image.Rect(0, 0, 640, 150))
	draw.Draw(scroll, scroll.Bounds(), visual, image.Pt(0, 250), draw.Src)
	foreground := 0
	changed := image.Rectangle{}
	for y := 250; y < 400; y++ {
		for x := 0; x < 640; x++ {
			if visual.NRGBAAt(x, y) != color.NRGBAModel.Convert(frame.At(x, y)).(color.NRGBA) {
				scrollExtractRequire(image.Pt(x, y).In(paper), "捲軸外框出現前景差異")
				foreground++
				r := image.Rect(x, y, x+1, y+1)
				if changed.Empty() {
					changed = r
				} else {
					changed = changed.Union(r)
				}
			}
		}
	}
	dat := scrollExtractRead("/original/DX5.DAT")
	scrollExtractRequire(scrollExtractSHA(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37", "DAT 不符")
	resource := dat[2692940 : 2692940+81152]
	scrollExtractRequire(scrollExtractSHA(resource) == "d749447e8f9a19a172d7cc51e6bba5db2fc8017db5b4752844ee2ed22d87d4e9", "VISUAL payload 不符")
	files := map[string]string{}
	for name, img := range map[string]image.Image{"VISUAL-original.png": visual, "scroll-original.png": scroll} {
		p := filepath.Join(root, name)
		_, e = os.Lstat(p)
		scrollExtractRequire(os.IsNotExist(e), "拒絕覆寫既有擷取")
		var b bytes.Buffer
		if e = png.Encode(&b, img); e != nil {
			panic(e)
		}
		if e = os.WriteFile(p, b.Bytes(), 0644); e != nil {
			panic(e)
		}
		files[name] = scrollExtractSHA(b.Bytes())
	}
	summary := map[string]any{"classification": "original resource extracted by dosgolem normal path; local only", "tool": "Go RAM planes + independent native RGB mapping", "go_version": runtime.Version(), "tool_sha256": scrollExtractSHA(scrollExtractRead("/project/tools/hd-scroll-extract.go")), "exe_sha256": receipt.InputSHA, "dat_sha256": scrollExtractSHA(dat), "resource_name": "VISUAL.OZM", "resource_file_offset": 2692940, "resource_length": 81152, "resource_sha256": scrollExtractSHA(resource), "receipt_sha256": scrollExtractSHA(raw), "ram_sha256": scrollExtractSHA(ram), "successful_steps": receipt.Steps, "mouse_input_sha256": receipt.MouseSHA, "planes": last.Planes, "native_reference_receipt_sha256": scrollExtractSHA(oldRaw), "native_reference_png_sha256": old.Frame.SHA, "palette": palette, "palette_observed": seen, "background_pixels_matched": matched, "foreground_pixels": foreground, "foreground_bounds": changed, "logical_rect": []int{0, 250, 640, 150}, "paper_rect": []int{80, 268, 510, 114}, "files": files}
	b, e := json.MarshalIndent(summary, "", "  ")
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile(filepath.Join(root, "scroll-extraction.json"), append(b, '\n'), 0644); e != nil {
		panic(e)
	}
	fmt.Printf("VISUAL background: %d pixels matched; scroll foreground: %d pixels; %v\n", matched, foreground, changed)
}
