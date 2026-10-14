package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"image"
	"image/color"
	"image/png"
	"os"
)

type mouseCursor struct {
	DisplayCount int
	Defined      bool
	HotX, HotY   int
	Mask         [32]uint16
}

func (m *mouseCursor) define(c *cpu386.CPU) bool {
	x, y := int(int16(c.R[cpu386.EBX])), int(int16(c.R[cpu386.ECX]))
	if x < -16 || x > 16 || y < -16 || y > 16 {
		return false
	}
	var b [64]byte
	for i := range b {
		a := uint64(c.R[cpu386.EDX]) + uint64(i)
		if a > 0xffffffff {
			return false
		}
		value, ok := c.ReadSegment8(c.Seg[cpu386.SegES], uint32(a))
		if !ok {
			return false
		}
		b[i] = value
	}
	var mask [32]uint16
	for i := range mask {
		mask[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	m.Mask = mask
	m.HotX = x
	m.HotY = y
	m.Defined = true
	return true
}

func (m *mouseCursor) composite(px []byte, w, h, x, y int, maxColor byte) int {
	if m == nil || !m.Defined || m.DisplayCount < 0 || w < 1 || h < 1 || len(px) != w*h {
		return 0
	}
	drawn := 0
	for row := 0; row < 16; row++ {
		py := y - m.HotY + row
		if py < 0 || py >= h {
			continue
		}
		for col := 0; col < 16; col++ {
			pX := x - m.HotX + col
			if pX < 0 || pX >= w {
				continue
			}
			bit := uint16(0x8000) >> col
			index := py*w + pX
			if m.Mask[row]&bit == 0 {
				px[index] = 0
			}
			if m.Mask[row+16]&bit != 0 {
				px[index] ^= maxColor
			}
			drawn++
		}
	}
	return drawn
}

func captureFrame(v *bankVideo, cursor *mouseCursor, x, y int) (map[string]any, error) {
	if v.Legacy == nil {
		return nil, nil
	}
	w, h := v.Width, v.Height
	var px []byte
	kind := "VBE"
	palette := v.Legacy.Palette()
	if v.Mode != 0 {
		if w < 1 || h < 1 || w > 4096 || h > 4096 || w*h > len(v.Pixels) {
			return nil, fmt.Errorf("invalid VBE frame size")
		}
		px = v.Pixels[:w*h]
	} else {
		kind = "VGA"
		w, h = v.Legacy.PlanarSize()
		if w == 0 || h == 0 {
			return nil, nil
		}
		px = v.Legacy.VGA.Pixels(w, h)
	}
	if len(px) != w*h {
		return nil, fmt.Errorf("frame pixel count mismatch")
	}
	rawHash := fmt.Sprintf("%x", sha256.Sum256(px))
	px = append([]byte(nil), px...)
	maxColor := byte(255)
	if kind == "VGA" {
		maxColor = 15
	}
	cursorPixels := cursor.composite(px, w, h, x, y, maxColor)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	colors := map[[3]byte]bool{}
	nonBlack := 0
	for i, p := range px {
		if kind == "VGA" {
			p = v.Legacy.VGA.DACIndex(p)
		}
		rgb := palette[p]
		colors[rgb] = true
		if rgb != [3]byte{} {
			nonBlack++
		}
		img.SetRGBA(i%w, i/w, color.RGBA{rgb[0], rgb[1], rgb[2], 255})
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return nil, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes()))
	name := "startup-probe/frame-" + digest + ".png"
	if err := os.WriteFile("/output/"+name, encoded.Bytes(), 0600); err != nil {
		return nil, err
	}
	return map[string]any{"path": "workplace/" + name, "png_sha256": digest, "rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(img.Pix)), "raw_indexed_sha256": rawHash, "cursor_overlay_pixels": cursorPixels, "cursor_overlay_classification": "display-layer approximation; no TSR VRAM writeback", "width": w, "height": h, "display": kind, "color_count": len(colors), "non_black_pixels": nonBlack, "source": "dosgolem native VRAM and DAC", "visually_verified": false}, nil
}

func paletteBIOS(c *cpu386.CPU, m *machine.Machine) bool {
	if uint16(c.R[cpu386.EAX]) == 0x1002 {
		var data [17]byte
		for i := range data {
			a := uint64(c.R[cpu386.EDX]) + uint64(i)
			if a > 0xffffffff {
				return false
			}
			b, ok := c.ReadSegment8(c.Seg[cpu386.SegES], uint32(a))
			if !ok {
				return false
			}
			data[i] = b & 63
		}
		for i := 0; i < 16; i++ {
			m.VGA.SetPal(i, data[i])
		}
		m.VGA.SetOverscan(data[16])
		return true
	}
	if uint16(c.R[cpu386.EAX]) == 0x1009 {
		data := make([]byte, 17)
		for i := 0; i < 16; i++ {
			data[i] = m.VGA.Pal(i)
		}
		data[16] = m.VGA.Overscan()
		return c.WriteSegmentBytes(c.Seg[cpu386.SegES], c.R[cpu386.EDX], data)
	}
	if uint16(c.R[cpu386.EAX]) == 0x1012 {
		return paletteBlockBIOS(c, m)
	}
	switch uint16(c.R[cpu386.EAX]) {
	case 0x1000, 0x1001, 0x1010:
	default:
		return false
	}
	return legacyQuery(c, m, 0x10)
}

func paletteBlockBIOS(c *cpu386.CPU, m *machine.Machine) bool {
	if uint16(c.R[cpu386.EAX]) != 0x1012 {
		return false
	}
	first, count := int(uint16(c.R[cpu386.EBX])), int(uint16(c.R[cpu386.ECX]))
	if first >= 256 || count > 256-first {
		return false
	}
	data := make([]byte, count*3)
	for i := range data {
		a := uint64(c.R[cpu386.EDX]) + uint64(i)
		if a > 0xffffffff {
			return false
		}
		b, ok := c.ReadSegment8(c.Seg[cpu386.SegES], uint32(a))
		if !ok {
			return false
		}
		data[i] = b & 63
	}
	copy(m.DAC[first*3:], data)
	return true
}

func paletteRealMode(r *cpu.CPU, n uint8, m *machine.Machine) bool {
	if n != 0x10 || r.R[cpu.AX] != 0x1012 {
		return false
	}
	c := cpu386.New(realModeBus{r.Bus})
	for i := range c.R {
		c.R[i] = uint32(r.R[i]) | uint32(r.Hi[i])<<16
	}
	c.Seg[cpu386.SegES] = 8
	c.SetDescriptor(8, cpu386.Descriptor{Base: uint32(r.Seg[cpu.ES]) * 16, Limit: 0xffff})
	return paletteBlockBIOS(c, m)
}

func setVideoBIOS(c *cpu386.CPU, m *machine.LEMachine, v *bankVideo, p *videoPorts) bool {
	mode := uint16(c.R[cpu386.EAX])
	if (mode != 0x12 && mode != 3) || v.Legacy == nil || len(m.Mem) < 0x500 {
		return false
	}
	if mode == 3 && (len(m.Mem) < 0xc0000 || len(v.Legacy.Mem) < 0xc0000) {
		return false
	}
	copy(v.Legacy.Mem[0x400:0x500], m.Mem[0x400:0x500])
	v.Legacy.SetVideoMode(byte(mode))
	if mode == 3 {
		bda := v.Legacy.Mem
		binary.LittleEndian.PutUint16(bda[0x44a:], 80)
		binary.LittleEndian.PutUint16(bda[0x44c:], 0x1000)
		clear(bda[0x44e:0x460])
		bda[0x460], bda[0x461] = 7, 6
		bda[0x462], bda[0x484] = 0, 24
		binary.LittleEndian.PutUint16(bda[0x485:], 16)
		for a := 0xb8000; a < 0xc0000; a += 2 {
			bda[a], bda[a+1] = ' ', 7
		}
		copy(m.Mem[0xb8000:0xc0000], bda[0xb8000:0xc0000])
	}
	copy(m.Mem[0x400:0x500], v.Legacy.Mem[0x400:0x500])
	v.Mode = 0
	v.Bank = 0
	v.Width = 0
	v.Height = 0
	p.seq = 0
	p.crtc = 0
	return true
}

func cursorBIOS(c *cpu386.CPU, m *machine.LEMachine) bool {
	fn := byte(c.R[cpu386.EAX] >> 8)
	if fn < 1 || fn > 3 || len(m.Mem) < 0x462 {
		return false
	}
	if fn == 1 {
		m.Mem[0x460] = byte(c.R[cpu386.ECX])
		m.Mem[0x461] = byte(c.R[cpu386.ECX] >> 8)
		return true
	}
	page := byte(c.R[cpu386.EBX] >> 8)
	if page > 7 {
		return false
	}
	a := 0x450 + int(page)*2
	if fn == 2 {
		m.Mem[a] = byte(c.R[cpu386.EDX])
		m.Mem[a+1] = byte(c.R[cpu386.EDX] >> 8)
	} else {
		lowWord(c, cpu386.EDX, uint16(m.Mem[a])|uint16(m.Mem[a+1])<<8)
		lowWord(c, cpu386.ECX, uint16(m.Mem[0x460])|uint16(m.Mem[0x461])<<8)
	}
	return true
}

// 標準 VGA 埠轉接。保留完整索引，避免未安裝的 SVGA 擴充別名到 VGA。
type videoPorts struct {
	m         *machine.Machine
	seq, crtc byte
}

func (v *videoPorts) advance(micros uint64) {
	v.m.CRTCTiming = true
	v.m.Steps = uint64(float64(micros) * machine.StepsPerSecond() / 1_000_000)
}

func (v *videoPorts) outputWord(p, value uint16) bool {
	switch p {
	case 0x3c4, 0x3d4, 0x3b4, 0x3ce:
	default:
		return false
	}
	v.output(p, byte(value))
	v.output(p+1, byte(value>>8))
	return true
}

func (v *videoPorts) output(p uint16, b byte) bool {
	switch p {
	case 0x3c4:
		v.seq = b
		if b <= 4 {
			v.m.Out8(p, b)
		}
	case 0x3c5:
		if v.seq <= 4 {
			v.m.Out8(p, b)
		}
	case 0x3d4, 0x3b4:
		v.crtc = b
		if b <= 0x18 {
			v.m.Out8(p, b)
		}
	case 0x3d5, 0x3b5:
		if v.crtc <= 0x18 {
			v.m.Out8(p, b)
		}
	case 0x3cd: // 沒有 Tseng bank register。
	case 0x3c0, 0x3c2, 0x3c6, 0x3c7, 0x3c8, 0x3c9, 0x3ce, 0x3cf:
		v.m.Out8(p, b)
	default:
		return false
	}
	return true
}

func (v *videoPorts) input(p uint16) (byte, bool) {
	switch p {
	case 0x3c4:
		return v.seq, true
	case 0x3c5:
		if v.seq > 4 {
			return 0xff, true
		}
	case 0x3d4, 0x3b4:
		return v.crtc, true
	case 0x3d5, 0x3b5:
		if v.crtc > 0x18 {
			return 0xff, true
		}
	case 0x3cd:
		return 0xff, true
	case 0x3c1, 0x3cc, 0x3c9, 0x3ce, 0x3cf, 0x3da:
	default:
		return 0, false
	}
	return v.m.In8(p), true
}
