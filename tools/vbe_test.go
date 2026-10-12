package main

import (
	"encoding/binary"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"testing"
)

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
