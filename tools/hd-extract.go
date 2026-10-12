// 研究入口：只擷取固定正常路徑最後一張 A-0.OZM，不是正式 OZM 解碼器。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

type capturePlane struct {
	Base   uint32 `json:"base_linear"`
	Global uint32 `json:"pointer_global_linear"`
	SHA    string `json:"sha256"`
	Error  string `json:"observation_error"`
}

type captureEvent struct {
	Kind, Name string
	Columns    uint32 `json:"raw_columns"`
	Rows       uint32 `json:"raw_rows"`
	Pitch      uint32 `json:"pitch_bytes"`
	Planes     []capturePlane
}

func digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func required(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func read(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	input := flag.String("input", "/output", "正常路徑擷取目錄")
	flag.Parse()
	_, err := os.Stat("/.dockerenv")
	required(err == nil, "只允許在 Docker 執行")
	raw := read(filepath.Join(*input, "startup-probe/latest.json"))
	var receipt struct {
		InputSHA string `json:"input_sha256"`
		Steps    uint64 `json:"successful_steps"`
		MouseSHA string `json:"mouse_input_sha256"`
		Stop     string `json:"stop"`
		Trace    struct {
			Events  []captureEvent
			Dropped uint64 `json:"dropped_events"`
			Pending []any  `json:"pending_returns"`
		} `json:"hd_asset_trace"`
		Frame struct {
			Path string
			SHA  string `json:"png_sha256"`
		} `json:"frame"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		panic(err)
	}
	required(receipt.InputSHA == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a", "EXE 版本不符")
	required(receipt.Steps == 150000000 && receipt.Stop == "instruction_budget", "擷取停止條件不符")
	required(receipt.MouseSHA == "cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736", "玩家操作前綴不符")
	required(receipt.Trace.Dropped == 0 && len(receipt.Trace.Pending) == 0, "載入觀測不完整")
	var last captureEvent
	for _, event := range receipt.Trace.Events {
		if event.Kind == "load_return" {
			last = event
		}
	}
	required(last.Name == "A-0.OZM" && last.Columns == 40 && last.Rows == 150 && last.Pitch == 80 && len(last.Planes) == 4, "最後素材不符")
	ram := read(filepath.Join(*input, "startup-memory.bin"))
	indices := make([]byte, 320*150)
	for i, plane := range last.Planes {
		required(plane.Error == "" && uint64(plane.Base)+149*80+40 <= uint64(len(ram)), "平面超界")
		required(plane.Base >= 0xc0000, "拒絕 VGA 與低位記憶體來源")
		required(plane.Global == []uint32{0x38c00, 0x38bf8, 0x38c04, 0x38bfc}[i], "平面來源順序不符")
		required(binary.LittleEndian.Uint32(ram[plane.Global:]) == plane.Base, "平面指標已改變")
		packed := make([]byte, 0, 40*150)
		for y := 0; y < 150; y++ {
			row := ram[uint64(plane.Base)+uint64(y*80) : uint64(plane.Base)+uint64(y*80+40)]
			packed = append(packed, row...)
			for x := 0; x < 320; x++ {
				indices[y*320+x] |= ((row[x/8] >> uint(7-x%8)) & 1) << uint(i)
			}
		}
		required(digest(packed) == plane.SHA, "緩衝區在載入後已改變")
	}
	frameBytes := read(filepath.Join(*input, "startup-probe", filepath.Base(receipt.Frame.Path)))
	required(digest(frameBytes) == receipt.Frame.SHA, "原生 PNG 雜湊不符")
	frame, err := png.Decode(bytes.NewReader(frameBytes))
	if err != nil {
		panic(err)
	}
	required(frame.Bounds() == image.Rect(0, 0, 640, 400), "原生畫面尺寸不符")
	var palette [16]color.NRGBA
	var seen [16]bool
	for y := 0; y < 150; y++ {
		for x := 0; x < 320; x++ {
			i := indices[y*320+x]
			c := color.NRGBAModel.Convert(frame.At(160+x, 50+y)).(color.NRGBA)
			required(!seen[i] || palette[i] == c, "候選矩形與平面色號不一致")
			palette[i], seen[i] = c, true
		}
	}
	asset := image.NewNRGBA(image.Rect(0, 0, 320, 150))
	for y := 0; y < 150; y++ {
		for x := 0; x < 320; x++ {
			asset.SetNRGBA(x, y, palette[indices[y*320+x]])
		}
	}
	dat := read("/original/DX5.DAT")
	required(digest(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37", "DAT 版本不符")
	resource := dat[566431 : 566431+15505]
	required(binary.LittleEndian.Uint16(resource) == 40 && binary.LittleEndian.Uint16(resource[2:]) == 150, "來源標頭不符")
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, asset); err != nil {
		panic(err)
	}
	summary := map[string]any{
		"status": "verified original asset extraction; not a production HD pack",
		"tool":   "Go standard library PNG and RAM extraction", "go_version": runtime.Version(),
		"tool_sha256":    digest(read("/project/tools/hd-extract.go")),
		"receipt_sha256": digest(raw), "ram_sha256": digest(ram), "frame_sha256": digest(frameBytes),
		"exe_sha256": receipt.InputSHA, "mouse_input_sha256": receipt.MouseSHA, "successful_steps": receipt.Steps,
		"dat_sha256": digest(dat), "resource_name": "A-0.OZM", "resource_file_offset": 566431, "resource_length": 15505,
		"resource_sha256": digest(resource), "planes": last.Planes,
		"logical_rect": []int{160, 50, 320, 150}, "source_size": []int{320, 150},
		"raw_indexed_sha256": digest(indices), "palette": palette, "palette_observed": seen,
		"png_sha256": digest(encoded.Bytes()), "pixels_verified": len(indices), "pixel_mismatches": 0,
		"scope":  "this normal-path capture only; draw-call parameters and occlusion require separate evidence",
		"rights": "original-derived; local only",
	}
	manifest, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		panic(err)
	}
	for _, name := range []string{"A-0-original.png", "A-0-extraction.json"} {
		_, err := os.Lstat(filepath.Join(*input, name))
		required(os.IsNotExist(err), "拒絕覆寫既有擷取輸出")
	}
	if err := os.WriteFile(filepath.Join(*input, "A-0-original.png"), encoded.Bytes(), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(*input, "A-0-extraction.json"), append(manifest, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("A-0.OZM 320x150: %d pixels verified; PNG SHA-256 %s\n", len(indices), digest(encoded.Bytes()))
}
