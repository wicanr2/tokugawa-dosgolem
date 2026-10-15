package main

import (
	"encoding/binary"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"math"
	"testing"
)

func TestVBEBIOSInitializesStandardRetraceWithoutChangingPalette(t *testing.T) {
	for _, tc := range []struct {
		mode uint16
		hz   float64
	}{{0x101, 25175000.0 / (800 * 525)}, {0x100, 25175000.0 / (800 * 449)}} {
		v := newBankVideo()
		v.Legacy = machine.New()
		v.Legacy.DAC[3] = 17
		palette := v.Legacy.Palette()
		v.Pixels[0] = 0x77
		c := cpu386.New(nil)
		c.R[cpu386.EAX], c.R[cpu386.EBX] = 0x4f02, uint32(tc.mode|0x8000)
		if !v.handle(c) || uint16(c.R[cpu386.EAX]) != 0x004f || v.Legacy.Palette() != palette || v.Pixels[0] != 0x77 {
			t.Fatal("VBE timing setup changed palette or preserved pixels")
		}
		timing := v.Legacy.VGA.Timing()
		if math.Abs(timing.FrameHz-tc.hz) > 0.000001 || timing.CharWidth != 8 || timing.HDisplayEnd != 80 {
			t.Fatalf("unexpected VBE timing: %+v", timing)
		}
		ports := videoPorts{m: v.Legacy}
		seen := map[byte]bool{}
		edges := 0
		old := byte(0)
		for micros := uint64(0); micros < 40000; micros++ {
			ports.advance(micros)
			value, ok := ports.input(0x3da)
			if !ok {
				t.Fatal("VBE retrace port unavailable")
			}
			bit := value & 8
			seen[bit] = true
			if bit != old {
				edges++
				old = bit
			}
		}
		if !seen[0] || !seen[8] || edges < 2 {
			t.Fatal("VBE retrace remained constant")
		}
		before := v.Legacy.VGA.Timing()
		if v.Legacy.VGA.SetVBEStandardTiming(0x102) || v.Legacy.VGA.Timing() != before {
			t.Fatal("unknown mode changed timing")
		}
	}
}

func TestVBEInfoAndBanking(t *testing.T) {
	le := &machine.LEMachine{Mem: make([]byte, 1<<20)}
	v := newBankVideo()
	bus := videoBus{le, v}
	c := cpu386.New(bus)
	c.Seg[cpu386.SegES] = 0x28
	c.SetDescriptor(0x28, cpu386.Descriptor{Base: 0x9000, Limit: 511, Writable: true})
	c.R[cpu386.EDI] = 16
	c.R[cpu386.EAX] = 0xbeef4f00
	c.EFlags = cpu386.IF | cpu386.CF
	if !v.handle(c) || c.R[cpu386.EAX] != 0xbeef004f || c.EFlags != cpu386.IF|cpu386.CF {
		t.Fatal("VBE status width/flags")
	}
	b := le.Mem[0x9010:]
	if string(b[:4]) != "VESA" || binary.LittleEndian.Uint16(b[4:]) != 0x0102 {
		t.Fatal("VBE identity")
	}
	for _, off := range []int{6, 14} {
		p := binary.LittleEndian.Uint32(b[off:])
		a := (p>>16)*16 + (p & 0xffff)
		if a < 0x9010 || a >= 0x9110 {
			t.Fatal("far pointer outside output block")
		}
	}
	c.R[cpu386.EAX] = 0x4f01
	c.R[cpu386.ECX] = 0x100
	if !v.handle(c) || binary.LittleEndian.Uint16(b[20:]) != 400 || b[25] != 8 || b[2] != 7 || binary.LittleEndian.Uint32(b[12:]) != 0 {
		t.Fatal("mode block")
	}
	c.R[cpu386.EAX] = 0x4f02
	c.R[cpu386.EBX] = 0x100
	if !v.handle(c) || v.Width != 640 || v.Height != 400 {
		t.Fatal("set mode")
	}
	bus.Write8(0xa0000, 0x55)
	c.R[cpu386.EAX] = 0x4f05
	c.R[cpu386.EBX] = 0
	c.R[cpu386.EDX] = 1
	v.handle(c)
	bus.Write8(0xa0000, 0xaa)
	if v.Pixels[0] != 0x55 || v.Pixels[65536] != 0xaa || le.Mem[0xa0000] != 0 {
		t.Fatal("banks aliased RAM")
	}
	c.R[cpu386.EAX] = 0x4f05
	c.R[cpu386.EDX] = 16
	v.handle(c)
	if uint16(c.R[cpu386.EAX]) != 0x014f || v.Bank != 1 {
		t.Fatal("invalid bank accepted")
	}
	c.R[cpu386.EAX] = 0x4f02
	c.R[cpu386.EBX] = 0x8101
	v.handle(c)
	if v.Pixels[0] != 0x55 || v.Height != 480 || v.Bank != 0 {
		t.Fatal("preserve display memory")
	}
	c.R[cpu386.EAX] = 0x4f02
	c.R[cpu386.EBX] = 0x101
	v.handle(c)
	if v.Pixels[0] != 0 {
		t.Fatal("clear display memory")
	}
	c.SetDescriptor(0x28, cpu386.Descriptor{Base: 0x9000, Limit: 511})
	before := append([]byte(nil), b[:256]...)
	c.R[cpu386.EAX] = 0x4f00
	if v.handle(c) || string(b[:256]) != string(before) || c.R[cpu386.EAX] != 0x4f00 {
		t.Fatal("readonly info accepted")
	}
}
