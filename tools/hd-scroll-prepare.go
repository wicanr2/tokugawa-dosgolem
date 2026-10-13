// 將生成畫布的既定捲軸區取為素材，不以生成圖反推版面。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	if _, e := os.Stat("/.dockerenv"); e != nil {
		panic("只允許在 Docker 執行")
	}
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			panic(e)
		}
		return b
	}
	sha := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	input := read("/output/scroll-full-hd-b-v1.png")
	im, e := png.Decode(bytes.NewReader(input))
	if e != nil {
		panic(e)
	}
	b := im.Bounds()
	d := b.Dx()*400 - b.Dy()*640
	if d < 0 {
		d = -d
	}
	if b.Min != image.Pt(0, 0) || b.Dx() < 1280 || b.Dy() < 800 || b.Dx() > 8192 || b.Dy() > 8192 || d*100 > b.Dy()*640 {
		panic("生成畫布尺寸或比例不符")
	}
	top := b.Dy() * 250 / 400
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()-top))
	draw.Draw(out, out.Bounds(), im, image.Pt(0, top), draw.Src)
	var encoded bytes.Buffer
	if e = png.Encode(&encoded, out); e != nil {
		panic(e)
	}
	p := "/output/scroll-hd-b-v1.png"
	if _, e = os.Lstat(p); !os.IsNotExist(e) {
		panic("拒絕覆寫既有 HD 素材")
	}
	if e = os.WriteFile(p, encoded.Bytes(), 0644); e != nil {
		panic(e)
	}
	metadata := map[string]any{"generator": "built-in image_gen.imagegen", "generation_mode": "style-transfer; original geometry reference; approved B style", "full_generated_sha256": sha(input), "full_size": []int{b.Dx(), b.Dy()}, "crop_pixels": []int{0, top, b.Dx(), b.Dy() - top}, "logical_rect": []int{0, 250, 640, 150}, "output_sha256": sha(encoded.Bytes()), "output_size": []int{out.Bounds().Dx(), out.Bounds().Dy()}, "prompt_sha256": sha(read("/output/scroll-hd-b-v1.prompt.txt")), "tool_sha256": sha(read("/project/tools/hd-scroll-prepare.go")), "original_reference_sha256": sha(read("/output/VISUAL-original.png")), "style_reference_sha256": sha(read("/project/workplace/hd-prototype/A-0-hd-b-v1.png")), "rights": "original-derived; local only", "scope": "lower scroll only; generated upper frame is unused"}
	data, e := json.MarshalIndent(metadata, "", "  ")
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile("/output/scroll-art.json", append(data, '\n'), 0644); e != nil {
		panic(e)
	}
	fmt.Println("HD scroll prepared", out.Bounds(), metadata["output_sha256"])
}
