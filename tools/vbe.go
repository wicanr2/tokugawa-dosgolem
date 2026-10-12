package main

import (
	"encoding/binary"
	"fmt"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

type bankVideo struct {
	Mode, Bank    uint16
	Width, Height int
	Pixels        []byte
	Legacy        *machine.Machine
}

func newBankVideo() *bankVideo { return &bankVideo{Pixels: make([]byte, 1<<20)} }
func vbeSize(mode uint16) (int, int) {
	switch mode {
	case 0x100:
		return 640, 400
	case 0x101:
		return 640, 480
	}
	return 0, 0
}
func lowWord(c *cpu386.CPU, reg int, value uint16) { c.R[reg] = c.R[reg]&0xffff0000 | uint32(value) }
func farPointer(linear uint32) uint32              { return (linear>>4)<<16 | linear&15 }

func (v *bankVideo) handle(c *cpu386.CPU) bool {
	fn := uint16(c.R[cpu386.EAX])
	if fn>>8 != 0x4f {
		return false
	}
	status := uint16(0x014f)
	switch fn {
	case 0x4f00:
		d, ok := c.Descriptors[c.Seg[cpu386.SegES]]
		if !ok {
			return false
		}
		a := uint64(d.Base) + uint64(c.R[cpu386.EDI])
		if a+256 > 0x100000 {
			return false
		}
		b := make([]byte, 256)
		copy(b, "VESA")
		binary.LittleEndian.PutUint16(b[4:], 0x0102)
		binary.LittleEndian.PutUint32(b[6:], farPointer(uint32(a)+64))
		binary.LittleEndian.PutUint32(b[14:], farPointer(uint32(a)+96))
		binary.LittleEndian.PutUint16(b[18:], 16)
		copy(b[64:], "dosgolem banked display\x00")
		binary.LittleEndian.PutUint16(b[96:], 0x100)
		binary.LittleEndian.PutUint16(b[98:], 0x101)
		binary.LittleEndian.PutUint16(b[100:], 0xffff)
		if !c.WriteSegmentBytes(c.Seg[cpu386.SegES], c.R[cpu386.EDI], b) {
			return false
		}
		status = 0x004f
	case 0x4f01:
		w, h := vbeSize(uint16(c.R[cpu386.ECX]))
		if w == 0 {
			break
		}
		b := make([]byte, 256)
		binary.LittleEndian.PutUint16(b, 0x1b)
		b[2] = 7
		binary.LittleEndian.PutUint16(b[4:], 64)
		binary.LittleEndian.PutUint16(b[6:], 64)
		binary.LittleEndian.PutUint16(b[8:], 0xa000)
		binary.LittleEndian.PutUint16(b[16:], uint16(w))
		binary.LittleEndian.PutUint16(b[18:], uint16(w))
		binary.LittleEndian.PutUint16(b[20:], uint16(h))
		b[22] = 8
		b[23] = 16
		b[24] = 1
		b[25] = 8
		b[26] = 16
		b[27] = 4
		b[28] = 64
		b[29] = byte(len(v.Pixels)/(w*h) - 1)
		b[30] = 1
		if !c.WriteSegmentBytes(c.Seg[cpu386.SegES], c.R[cpu386.EDI], b) {
			return false
		}
		status = 0x004f
	case 0x4f02:
		mode := uint16(c.R[cpu386.EBX])
		w, h := vbeSize(mode & 0x7fff)
		if w == 0 {
			break
		}
		if mode&0x8000 == 0 {
			clear(v.Pixels)
		}
		v.Mode = mode & 0x7fff
		v.Bank = 0
		v.Width = w
		v.Height = h
		status = 0x004f
	case 0x4f03:
		lowWord(c, cpu386.EBX, v.Mode)
		status = 0x004f
	case 0x4f05:
		if v.Mode == 0 || byte(c.R[cpu386.EBX]) != 0 {
			break
		}
		switch byte(c.R[cpu386.EBX] >> 8) {
		case 0:
			bank := uint16(c.R[cpu386.EDX])
			if int(bank)*65536 >= len(v.Pixels) {
				break
			}
			v.Bank = bank
			status = 0x004f
		case 1:
			lowWord(c, cpu386.EDX, v.Bank)
			status = 0x004f
		}
	}
	lowWord(c, cpu386.EAX, status)
	return true
}

func (v *bankVideo) realMode(r *cpu.CPU, n uint8) bool {
	if n != 0x10 || r.R[cpu.AX]>>8 != 0x4f {
		return false
	}
	c := cpu386.New(realModeBus{r.Bus})
	for i := range c.R {
		c.R[i] = uint32(r.R[i]) | uint32(r.Hi[i])<<16
	}
	c.EFlags = uint32(r.Flags)
	c.Seg[cpu386.SegES] = 8
	c.SetDescriptor(8, cpu386.Descriptor{Base: uint32(r.Seg[cpu.ES]) * 16, Limit: 0xffff, Writable: true})
	if !v.handle(c) {
		return false
	}
	for i := range c.R {
		r.R[i] = uint16(c.R[i])
		r.Hi[i] = uint16(c.R[i] >> 16)
	}
	r.SetFlags(uint16(c.EFlags))
	return true
}

type videoBus struct {
	le    *machine.LEMachine
	video *bankVideo
}

func (b videoBus) Read8(a uint32) (byte, error) {
	if a >= 0xa0000 && a < 0xb0000 {
		if b.video.Mode != 0 {
			i := uint32(b.video.Bank)*65536 + a - 0xa0000
			if uint64(i) >= uint64(len(b.video.Pixels)) {
				return 0, fmt.Errorf("VBE read out of bounds")
			}
			return b.video.Pixels[i], nil
		}
		if b.video.Legacy != nil && b.video.Legacy.VideoMode() == 0x12 {
			return b.video.Legacy.Read8(a), nil
		}
	}
	return b.le.Read8(a)
}
func (b videoBus) Write8(a uint32, value byte) error {
	if a >= 0xa0000 && a < 0xb0000 {
		if b.video.Mode != 0 {
			i := uint32(b.video.Bank)*65536 + a - 0xa0000
			if uint64(i) >= uint64(len(b.video.Pixels)) {
				return fmt.Errorf("VBE write out of bounds")
			}
			b.video.Pixels[i] = value
			return nil
		}
		if b.video.Legacy != nil && b.video.Legacy.VideoMode() == 0x12 {
			b.video.Legacy.Write8(a, value)
			return nil
		}
	}
	return b.le.Write8(a, value)
}
