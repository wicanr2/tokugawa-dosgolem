package main

import (
	"bytes"
	"encoding/binary"
	"image"
)

// IDA 9.4 原始入口與普通歷史簡介來源證據見 RESEARCH-LOG。
func openingPartialEntry(eip uint32) bool {
	return eip == 0x11e37 || eip == 0x11ba6 || eip == 0x11c16
}

func openingPartialDestination(eip uint32) image.Rectangle {
	switch eip {
	case 0x11e37:
		return image.Rect(248, 50, 376, 169)
	case 0x11ba6:
		return image.Rect(216, 30, 424, 230)
	case 0x11c16:
		return image.Rect(216, 98, 288, 125)
	}
	return image.Rectangle{}
}

func openingPartialArgs(eip, esp uint32, mem []byte) ([5]uint32, bool) {
	var args [5]uint32
	if !openingPartialEntry(eip) || uint64(esp)+8 > uint64(len(mem)) {
		return args, false
	}
	offset := binary.LittleEndian.Uint32(mem[uint64(esp)+4:])
	var offsets []uint32
	switch eip {
	case 0x11e37:
		args, offsets = [5]uint32{offset, 16, 119, 80, 0xa0fbf}, []uint32{0, 16, 32}
	case 0x11ba6:
		args, offsets = [5]uint32{offset, 26, 200, 80, 0xa097b}, []uint32{0, 26, 52, 16000, 16026}
	case 0x11c16:
		args, offsets = [5]uint32{offset, 9, 27, 80, 0xa1ebb}, []uint32{16052, 18212, 20372, 22532}
	}
	for _, known := range offsets {
		if offset == known {
			return args, true
		}
	}
	return args, false
}

func openingPartialSource(eip uint32) string {
	if eip == 0x11e37 {
		return "OP2_P.OZM"
	}
	if eip == 0x11ba6 || eip == 0x11c16 {
		return "OP9.OZM"
	}
	return ""
}

func openingStorySource(name string, mem []byte) ([]byte, bool) {
	if _, partial := hdPrefixProfiles[name]; partial {
		current, ok := prefixStorySource(name, mem, true)
		return current.Indexed, ok
	}
	profile, known := hdPortraitProfiles[name]
	if !known || profile.Columns != 40 || !openingPlaneAlias(mem) {
		return nil, false
	}
	source := graphicsSourceProfile{Name: name, Width: 320, Height: 150, Planes: profile.Planes[:]}
	_, expected, ok := graphicsSource(mem, []graphicsSourceProfile{source})
	return expected, ok
}

func openingStoryMatches(name string, mem []byte, snapshot func() (nativeSnapshot, error)) bool {
	expected, ok := openingStorySource(name, mem)
	if !ok || snapshot == nil {
		return false
	}
	native, err := snapshot()
	return err == nil && bytes.Equal(expected, graphicsCrop(native.Indexed, 640, image.Rect(160, 50, 480, 200)))
}

// 三次交鋒與六次 OP9 的原版返回，完整 48 通道分別一致。
// partial/source-verification.json SHA e179e94ac069b7b8a255a38cd4cb0afcc1d3c33e64bd1c0b8ea9ecf0862f5245。
var hdOpeningPartialPalettes = map[string][16][3]byte{
	"OP2_P.OZM": {
		{255, 0, 255}, {85, 48, 48}, {195, 85, 73}, {85, 60, 60},
		{255, 0, 0}, {48, 85, 0}, {0, 0, 0}, {195, 195, 73},
		{73, 48, 48}, {97, 85, 73}, {231, 195, 97}, {219, 0, 24},
		{255, 109, 109}, {255, 0, 0}, {255, 243, 219}, {255, 255, 255},
	},
	"OP9.OZM": {
		{0, 0, 0}, {36, 36, 36}, {195, 73, 48}, {85, 24, 24},
		{255, 219, 24}, {48, 85, 0}, {24, 24, 231}, {231, 231, 60},
		{0, 0, 0}, {85, 85, 85}, {231, 195, 97}, {219, 0, 0},
		{255, 85, 85}, {48, 219, 48}, {36, 231, 255}, {255, 255, 255},
	},
}
