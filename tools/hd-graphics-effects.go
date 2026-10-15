package main

import "image/color"

type graphicsEffect struct {
	Kind        string `json:"kind"`
	NativeLimit byte   `json:"native_limit"`
	HDLimit     byte   `json:"hd_limit"`
	CycleSteps  int    `json:"cycle_steps"`
	Valid       bool   `json:"-"`
}

func graphicsEffectSource(name string) bool {
	return name == "OP4.OZM" || name == "OP-FULL.OZM"
}

func graphicsRGB6(value byte) byte { return value<<2 | value>>4 }

// 原版淡入最後為非零目標減一；零目標由循環取樣獨立確認。
func graphicsCycleTarget(value byte) byte {
	if value == 0 {
		return 0
	}
	return min(byte(63), (value>>2)+1)
}

func graphicsCycleChannel(target byte, steps int) byte {
	value := int(target) + steps
	if target <= 21 {
		value = min(value, 21)
	}
	return graphicsRGB6(byte(value) & 63)
}

func graphicsPaletteEffect(profile hdGraphicsProfile, s nativeSnapshot) graphicsEffect {
	want := graphicsNativePalette(profile)
	var actual [16][3]byte
	exact := true
	var peak, maximum byte
	for i, c := range want {
		actual[i] = s.Palette[s.DAC[i]]
		rgb := [3]byte{c.R, c.G, c.B}
		if actual[i] != rgb {
			exact = false
		}
		for channel := 0; channel < 3; channel++ {
			peak = max(peak, actual[i][channel])
			maximum = max(maximum, rgb[channel])
		}
	}
	if !graphicsEffectSource(profile.Source.Name) {
		extra, known := hdOpeningPartialPalettes[profile.Source.Name]
		return graphicsEffect{Valid: exact || (known && actual == extra)}
	}
	if exact {
		return graphicsEffect{Kind: "clamp", NativeLimit: peak, HDLimit: 255, Valid: true}
	}
	if maximum != 0 && peak <= maximum && graphicsRGB6(peak>>2) == peak {
		clamped := true
		for i, c := range want {
			if actual[i] != [3]byte{min(c.R, peak), min(c.G, peak), min(c.B, peak)} {
				clamped = false
			}
		}
		if clamped {
			limit := byte(uint16(peak) * 255 / uint16(maximum))
			return graphicsEffect{Kind: "clamp", NativeLimit: peak, HDLimit: limit, Valid: true}
		}
	}
	if profile.Source.Name == "OP-FULL.OZM" {
		for steps := 0; steps <= 21; steps++ {
			matched := true
			for i, c := range want {
				if actual[i] != [3]byte{graphicsCycleChannel(graphicsCycleTarget(c.R), steps), graphicsCycleChannel(graphicsCycleTarget(c.G), steps), graphicsCycleChannel(graphicsCycleTarget(c.B), steps)} {
					matched = false
				}
			}
			if matched {
				return graphicsEffect{Kind: "cycle", CycleSteps: steps, Valid: true}
			}
		}
	}
	return graphicsEffect{}
}

// 新 24-bit 素材的顯示轉換近似；不修改原版 DAC 或時間。
func (e graphicsEffect) colour(c color.NRGBA) color.NRGBA {
	switch e.Kind {
	case "clamp":
		c.R, c.G, c.B = min(c.R, e.HDLimit), min(c.G, e.HDLimit), min(c.B, e.HDLimit)
	case "cycle":
		c.R = graphicsCycleChannel(graphicsCycleTarget(c.R), e.CycleSteps)
		c.G = graphicsCycleChannel(graphicsCycleTarget(c.G), e.CycleSteps)
		c.B = graphicsCycleChannel(graphicsCycleTarget(c.B), e.CycleSteps)
	}
	return c
}
