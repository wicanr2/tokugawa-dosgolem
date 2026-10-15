package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

type hdPrefixProfile struct {
	Source, SourceSHA, Asset, AssetSHA string
	Planes                             [4]string
}

type prefixStorySnapshot struct {
	Bases   [4]uint32
	Indexed []byte
}

func prefixStorySHA(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

// 第四平面最後 byte 是未知來源；讀取實際值供本次繪圖 guard，從不固定該值。
func prefixStorySource(name string, mem []byte, aliases bool) (prefixStorySnapshot, bool) {
	var current prefixStorySnapshot
	profile, known := hdPrefixProfiles[name]
	if !known || len(mem) < 0x5b1bc || binary.LittleEndian.Uint32(mem[0x5b1b8:]) != 40 || binary.LittleEndian.Uint32(mem[0x5b1b4:]) != 150 || (aliases && !openingPlaneAlias(mem)) {
		return current, false
	}
	current.Indexed = make([]byte, 48000)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := binary.LittleEndian.Uint32(mem[global:])
		end := uint64(base) + 149*80 + 40
		if end > uint64(len(mem)) || (base < 0xc0000 && end > 0xa0000) {
			return prefixStorySnapshot{}, false
		}
		current.Bases[bit] = base
		plane := make([]byte, 6000)
		for y := 0; y < 150; y++ {
			copy(plane[y*40:(y+1)*40], mem[uint64(base)+uint64(y*80):uint64(base)+uint64(y*80+40)])
		}
		length := 6000
		if bit == 3 {
			length = 5999
		}
		if prefixStorySHA(plane[:length]) != profile.Planes[bit] {
			return prefixStorySnapshot{}, false
		}
		for i, value := range plane {
			for x := 0; x < 8; x++ {
				current.Indexed[i*8+x] |= ((value >> uint(7-x)) & 1) << uint(bit)
			}
		}
	}
	return current, true
}

func prefixStoryKnown(name string) bool {
	_, full := hdPortraitProfiles[name]
	_, partial := hdPrefixProfiles[name]
	return full || partial
}

func prefixStoryReturned(p *hdDrawReturn, name string, mem []byte, snapshot func() (nativeSnapshot, error)) bool {
	if p.SourceName != name || snapshot == nil {
		return false
	}
	current, ok := prefixStorySource(name, mem, p.Entry == 0x11dcb)
	if !ok || current.Bases != p.SourceBases || prefixStorySHA(current.Indexed) != p.SourceGuard {
		return false
	}
	s, err := snapshot()
	return err == nil && bytes.Equal(current.Indexed, graphicsCrop(s.Indexed, 640, a0Rect))
}

type hdPrefixPair struct {
	Profile hdPrefixProfile
	Source  []byte
	Asset   image.Image
	Reason  string
}

func newHDPrefixPair(profile hdPrefixProfile, source []byte, asset image.Image) hdPrefixPair {
	p := hdPrefixPair{Profile: profile, Reason: "fallback_source"}
	if len(source) != 48000 || prefixStorySHA(source) != profile.SourceSHA {
		return p
	}
	for i, value := range source {
		if (i < 47992 && value > 15) || (i >= 47992 && value != 255) {
			return p
		}
	}
	p.Source = append([]byte(nil), source...)
	p.Reason = "fallback_missing_asset"
	if asset == nil {
		return p
	}
	var reason string
	p.Asset, reason = prepareHDPortrait(asset)
	if reason != "" {
		p.Reason = "fallback_asset_geometry"
		return p
	}
	p.Reason = "ready"
	return p
}

func (p hdPrefixPair) apply(out *image.NRGBA, s nativeSnapshot, name string, enabled bool) string {
	if !enabled {
		return "original"
	}
	if p.Reason != "ready" {
		return p.Reason
	}
	if registered, known := hdPrefixProfiles[name]; !known || registered != p.Profile {
		return "fallback_source"
	}
	if out == nil || out.Bounds() != image.Rect(0, 0, 1280, 800) || s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 400) || len(s.Indexed) != 256000 {
		return "fallback_geometry"
	}
	if s.MaxColor != 15 || !graphicsPaletteEffect(hdGraphicsProfile{Source: graphicsSourceProfile{Name: name}}, s).Valid {
		return "fallback_palette"
	}
	for y := 0; y < 150; y++ {
		for x := 0; x < 320; x++ {
			want := p.Source[y*320+x]
			if want != 255 && s.Indexed[(y+50)*640+x+160] != want {
				return "fallback_source_mismatch"
			}
		}
	}
	for y := 0; y < 300; y++ {
		for x := 0; x < 640; x++ {
			pixel := rgbaAt(p.Asset, x, y)
			if p.Source[y/2*320+x/2] == 255 {
				pixel = rgbaAt(s.Image, 160+x/2, 50+y/2)
			}
			out.SetNRGBA(320+x, 100+y, pixel)
		}
	}
	return "hd"
}
