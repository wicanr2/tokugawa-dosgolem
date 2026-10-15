// 可丟棄的出版商標誌呈現原型，未接遊戲或改變啟動順序。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if os.Getuid() != 1000 {
		panic("Docker UID 1000 required")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		panic(err)
	}
	paths := []string{"/project/workplace/hd-prototype/catalog/remaining/source-boundary/RAINEND-original-reference.png", "/project/assets/hd-b/art/RAINEND-hd-b-v1.png"}
	want := []string{"8e5d4ff9a5b677ddeab48a33a9502b28941529625c10a7ca3c6c73dbf3c42be7", "d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93"}
	var inputs [2]image.Image
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != want[i] {
			panic("input SHA mismatch")
		}
		inputs[i], err = png.Decode(bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
	}
	if inputs[0].Bounds() != image.Rect(0, 0, 640, 480) {
		panic("original PCX reference geometry")
	}
	b := inputs[1].Bounds()
	if b.Min != (image.Point{}) || b.Dx() < 640 || b.Dy() < 480 || b.Dx()*480 != b.Dy()*640 {
		panic("HD geometry")
	}
	out := image.NewNRGBA(image.Rect(0, 0, 1280, 960))
	for y := 0; y < 960; y++ {
		for x := 0; x < 1280; x++ {
			pixel := inputs[1].At(x*b.Dx()/1280, y*b.Dy()/960)
			if y >= 449*2 {
				pixel = inputs[0].At(x/2, y/2)
			}
			out.SetNRGBA(x, y, color.NRGBAModel.Convert(pixel).(color.NRGBA))
		}
	}
	path := "/output/publisher-HD-preview.png"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, out); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	tool, err := os.ReadFile("/project/tools/hd-publisher-preview.go")
	if err != nil {
		panic(err)
	}
	receipt := map[string]any{"classification": "offline presentation prototype; PCX executable not run, original game startup unchanged", "source_sha256": want[0], "HD_sha256": want[1], "output_sha256": fmt.Sprintf("%x", sha256.Sum256(data)), "tool_sha256": fmt.Sprintf("%x", sha256.Sum256(tool)), "size": []int{1280, 960}, "native_preserve_rect": []int{0, 449, 640, 480}, "normal_player_path_verified": false, "rights": "local only; contains original copyright bitmap strip"}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("/output/preview.json", append(encoded, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
