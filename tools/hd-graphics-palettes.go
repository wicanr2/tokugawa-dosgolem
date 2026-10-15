package main

import "image/color"

// OP1 的 22M 與 26.333333M 正常開場擷取相同，淡入／淡出不套用此調色盤。
// 證據：graphics/static/palette-probe/palette-captures.json
// SHA-256 cbe61f72fc33ce5a3e2ed5873ac5d06fa9fdba96d9688c478ce579c1f1f73936。
// 原版來源指紋仍由 hdGraphicsLogoProfile 獨立核對。
var hdGraphicsSourcePalettes = map[string][16]color.NRGBA{
	"OP-FULL.OZM": {
		{0, 0, 0, 255}, {32, 32, 32, 255}, {190, 69, 44, 255}, {81, 20, 20, 255},
		{251, 215, 215, 255}, {44, 81, 93, 255}, {20, 20, 227, 255}, {227, 227, 93, 255},
		{0, 0, 0, 255}, {81, 81, 81, 255}, {227, 190, 93, 255}, {215, 0, 0, 255},
		{251, 81, 81, 255}, {32, 227, 251, 255}, {56, 56, 56, 255}, {251, 251, 251, 255},
	},
	"OP1.OZM": {
		{0, 0, 0, 255}, {56, 0, 251, 255}, {32, 0, 93, 255}, {0, 0, 32, 255},
		{0, 190, 251, 255}, {44, 81, 0, 255}, {20, 20, 227, 255}, {227, 227, 56, 255},
		{0, 0, 0, 255}, {81, 81, 81, 255}, {227, 190, 93, 255}, {215, 0, 0, 255},
		{251, 81, 81, 255}, {44, 215, 44, 255}, {32, 227, 251, 255}, {251, 251, 251, 255},
	},
}

func graphicsNativePalette(profile hdGraphicsProfile) [16]color.NRGBA {
	if palette, ok := hdGraphicsSourcePalettes[profile.Source.Name]; ok {
		return palette
	}
	return battleFrameNativePalette
}
