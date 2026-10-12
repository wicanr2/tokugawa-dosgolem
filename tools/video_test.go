package main

import (
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"math"
	"testing"
)

func TestRetraceVirtualClock(t *testing.T) {
	m := machine.New()
	m.SetVideoMode(0x12)
	v := videoPorts{m: m}
	timing := m.VGA.Timing()
	v.advance(0)
	a, ok := v.input(0x3da)
	if !ok || a&8 != 0 {
		t.Fatal("retrace at frame start")
	}
	v.advance(uint64(math.Ceil(float64(timing.VRetraceStart)*1e6/timing.LineHz)) + 1)
	a, ok = v.input(0x3da)
	if !ok || a&8 == 0 {
		t.Fatal("retrace start")
	}
	v.advance(uint64(math.Ceil(float64(timing.VTotal)*1e6/timing.LineHz)) + 1)
	a, ok = v.input(0x3da)
	if !ok || a&8 != 0 {
		t.Fatal("retrace frame wrap")
	}
	le := &machine.LEMachine{Mem: []byte{0xec, 0x24, 8, 0x75, 0xfb, 0xec, 0x24, 8, 0x74, 0xf6}}
	c := cpu386.New(le)
	c.R[cpu386.EDX] = 0x3da
	c.PortIn = v.input
	steps := 0
	for c.EIP < 10 && steps < 50000 {
		v.advance(uint64(steps))
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		steps++
	}
	if c.EIP != 10 || steps < 100 {
		t.Fatalf("paired retrace wait did not run normally, eip=%x steps=%d", c.EIP, steps)
	}
}

func TestMouseScriptPlatformInput(t *testing.T) {
	for _, s := range []string{`[{"step":4,"x":1,"y":2,"buttons":8}]`, `[{"step":4,"x":1,"y":2},{"step":4,"x":1,"y":2}]`, `[{"step":10,"x":1,"y":2}]`, `[{"step":4,"x":-1,"y":2}]`, `[{"step":4,"x":1,"y":2,"address":16}]`, `[] []`} {
		if _, err := parseMouseEvents(s, 10); err == nil {
			t.Fatalf("invalid script accepted: %s", s)
		}
	}
	events, err := parseMouseEvents(`[{"step":2,"x":900,"y":172,"buttons":1},{"step":3,"x":300,"y":172,"buttons":1},{"step":4,"x":300,"y":172,"buttons":0}]`, 10)
	if err != nil {
		t.Fatal(err)
	}
	m := machine.New()
	m.SetVideoMode(0x12)
	d := dos.New(m, "")
	d.Install()
	defer d.Close()
	d.Mouse.MaxX = 639
	d.Mouse.MaxY = 399
	c := cpu386.New(&machine.LEMachine{Mem: make([]byte, 256)})
	cursor := mouseCursor{DisplayCount: -1}
	applyMouseEvent(d, events[0])
	c.R[cpu386.EAX] = 3
	if !mouseInterrupt(c, m, &cursor) || uint16(c.R[cpu386.EBX]) != 1 || uint16(c.R[cpu386.ECX]) != 639 || uint16(c.R[cpu386.EDX]) != 172 {
		t.Fatal("ordinary INT33 polling did not observe clamped input")
	}
	applyMouseEvent(d, events[1])
	applyMouseEvent(d, events[2])
	c.R[cpu386.EAX] = 3
	if !mouseInterrupt(c, m, &cursor) || uint16(c.R[cpu386.EBX]) != 0 || uint16(c.R[cpu386.ECX]) != 300 || d.Mouse.Press[0] != 1 || d.Mouse.Release[0] != 1 || d.Mouse.PressAt[0] != [2]uint16{639, 172} || d.Mouse.ReleaseAt[0] != [2]uint16{300, 172} {
		t.Fatal("mouse button transitions or positions")
	}
}

func TestHiddenMouseState(t *testing.T) {
	m := machine.New()
	d := dos.New(m, "")
	d.Install()
	defer d.Close()
	cursor := mouseCursor{DisplayCount: -1}
	c := cpu386.New(nil)
	c.R[cpu386.EAX] = 0xabcd0002
	c.R[cpu386.EBX] = 0x12345678
	c.EFlags = cpu386.IF | cpu386.CF | cpu386.OF
	before := c.R
	if !mouseInterrupt(c, m, &cursor) || cursor.DisplayCount != -2 || c.R != before || c.EFlags != cpu386.IF|cpu386.CF|cpu386.OF {
		t.Fatal("hide mouse state")
	}
	c.R[cpu386.EAX] = 1
	if mouseInterrupt(c, m, &cursor) || cursor.DisplayCount != -2 {
		t.Fatal("unimplemented visible cursor accepted")
	}
	c.R[cpu386.EAX] = 0xabcd0000
	if !mouseInterrupt(c, m, &cursor) || cursor.DisplayCount != -1 || c.R[cpu386.EAX] != 0xabcdffff || c.R[cpu386.EBX] != 0x12340002 {
		t.Fatal("mouse reset")
	}
}

func TestMouseMaskAndComposition(t *testing.T) {
	for _, reject := range []bool{false, true} {
		le := &machine.LEMachine{Mem: make([]byte, 128)}
		le.Mem[32] = 0
		le.Mem[33] = 0x80
		le.Mem[64] = 0
		le.Mem[65] = 0xc0
		c := cpu386.New(le)
		c.Seg[cpu386.SegES] = 8
		limit := uint32(127)
		if reject {
			limit = 94
		}
		c.SetDescriptor(8, cpu386.Descriptor{Limit: limit})
		c.R[cpu386.EAX] = 9
		c.R[cpu386.EDX] = 32
		cursor := mouseCursor{DisplayCount: -1}
		if cursor.define(c) == reject {
			t.Fatal("cursor source bounds")
		}
		if reject {
			if cursor.Defined {
				t.Fatal("partial cursor published")
			}
			continue
		}
		if cursor.Mask[0] != 0x8000 || cursor.Mask[16] != 0xc000 {
			t.Fatal("cursor word order")
		}
		c.R[cpu386.EAX] = 1
		if !mouseInterrupt(c, nil, &cursor) || cursor.DisplayCount != 0 {
			t.Fatal("show defined cursor")
		}
		px := []byte{5, 5, 5, 5}
		if cursor.composite(px, 2, 2, 0, 0, 15) != 4 || px[0] != 10 || px[1] != 15 || px[2] != 0 {
			t.Fatal("cursor AND XOR order")
		}
		cursor.DisplayCount = -1
		unchanged := []byte{5, 5, 5, 5}
		if cursor.composite(unchanged, 2, 2, 0, 0, 15) != 0 || unchanged[0] != 5 {
			t.Fatal("hidden cursor modified frame")
		}
		cursor.DisplayCount = 0
		cursor.HotX = 1
		cursor.HotY = 1
		clipped := []byte{5}
		if cursor.composite(clipped, 1, 1, 0, 0, 255) != 1 || clipped[0] != 0 {
			t.Fatal("cursor hotspot clipping")
		}
	}
}

func TestWriteAttributePaletteLE(t *testing.T) {
	for _, reject := range []bool{false, true} {
		le := &machine.LEMachine{Mem: make([]byte, 128)}
		for i := 0; i < 17; i++ {
			le.Mem[80+i] = byte(80 + i)
		}
		m := machine.New()
		m.VGA.SetPal(5, 7)
		c := cpu386.New(le)
		c.Seg[cpu386.SegES] = 0x28
		limit := uint32(63)
		if reject {
			limit = 31
		}
		c.SetDescriptor(0x28, cpu386.Descriptor{Base: 64, Limit: limit})
		c.R[cpu386.EAX] = 0x1002
		c.R[cpu386.EDX] = 16
		c.EFlags = cpu386.IF | cpu386.CF
		before := c.R
		ok := paletteBIOS(c, m)
		if reject {
			if ok || m.VGA.Pal(5) != 7 {
				t.Fatal("partial attribute palette published")
			}
		} else if !ok || m.VGA.Pal(5) != 21 || m.VGA.Overscan() != 32 {
			t.Fatal("write attribute palette")
		}
		if c.R != before || c.EFlags != cpu386.IF|cpu386.CF {
			t.Fatal("write palette unrelated state")
		}
	}
}

func TestReadAttributePaletteLE(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		le := &machine.LEMachine{Mem: make([]byte, 128)}
		for i := range le.Mem[80:98] {
			le.Mem[80+i] = 0xaa
		}
		m := machine.New()
		m.VGA.SetPal(5, 63)
		m.VGA.SetOverscan(27)
		c := cpu386.New(le)
		c.Seg[cpu386.SegES] = 0x28
		limit := uint32(63)
		if reject == 2 {
			limit = 31
		}
		c.SetDescriptor(0x28, cpu386.Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
		c.R[cpu386.EAX] = 0x1009
		c.R[cpu386.EDX] = 16
		c.EFlags = cpu386.IF | cpu386.CF
		before := c.R
		ok := paletteBIOS(c, m)
		if reject > 0 {
			if ok {
				t.Fatal("invalid palette output accepted")
			}
			for _, b := range le.Mem[80:97] {
				if b != 0xaa {
					t.Fatal("partial palette output")
				}
			}
		} else if !ok || le.Mem[85] != 63 || le.Mem[96] != 27 {
			t.Fatal("LE attribute palette output")
		}
		if c.R != before || c.EFlags != cpu386.IF|cpu386.CF || le.Mem[97] != 0xaa {
			t.Fatal("palette output unrelated state")
		}
	}
}

func TestPaletteBIOS(t *testing.T) {
	m := machine.New()
	d := dos.New(m, "")
	d.Install()
	defer d.Close()
	c := cpu386.New(nil)
	c.R[cpu386.EAX] = 0xabcd1010
	c.R[cpu386.EBX] = 0x98760010
	c.R[cpu386.ECX] = 0x12340102
	c.R[cpu386.EDX] = 0x43213f00
	c.EFlags = cpu386.IF | cpu386.CF | cpu386.OF
	before := c.R
	if !paletteBIOS(c, m) || c.R != before || c.EFlags != cpu386.IF|cpu386.CF|cpu386.OF || m.DAC[48] != 63 || m.DAC[49] != 1 || m.DAC[50] != 2 {
		t.Fatal("DAC scalar BIOS")
	}
	c.R[cpu386.EAX] = 0x1000
	c.R[cpu386.EBX] = 0x3f05
	if !paletteBIOS(c, m) || m.VGA.Pal(5) != 63 {
		t.Fatal("attribute palette")
	}
	c.R[cpu386.EAX] = 0x1001
	c.R[cpu386.EBX] = 0x1b00
	if !paletteBIOS(c, m) || m.VGA.Overscan() != 27 {
		t.Fatal("overscan")
	}
	c.R[cpu386.EAX] = 0x1017
	if paletteBIOS(c, m) {
		t.Fatal("pointer palette service accepted without LE adapter")
	}
}

func TestPaletteBlockLE(t *testing.T) {
	for _, reject := range []bool{false, true} {
		le := &machine.LEMachine{Mem: make([]byte, 128)}
		copy(le.Mem[80:], []byte{63, 64, 255, 1, 2, 3})
		m := machine.New()
		m.DAC[48] = 7
		c := cpu386.New(le)
		c.Seg[cpu386.SegES] = 0x28
		limit := uint32(63)
		if reject {
			limit = 20
		}
		c.SetDescriptor(0x28, cpu386.Descriptor{Base: 64, Limit: limit})
		c.R[cpu386.EAX] = 0x1012
		c.R[cpu386.EBX] = 16
		c.R[cpu386.ECX] = 2
		c.R[cpu386.EDX] = 16
		c.EFlags = cpu386.IF | cpu386.CF
		before := c.R
		ok := paletteBlockBIOS(c, m)
		if reject {
			if ok || m.DAC[48] != 7 {
				t.Fatal("partial palette published")
			}
		} else if !ok || m.DAC[48] != 63 || m.DAC[49] != 0 || m.DAC[50] != 63 || m.DAC[53] != 3 {
			t.Fatal("LE block palette")
		}
		if c.R != before || c.EFlags != cpu386.IF|cpu386.CF {
			t.Fatal("block palette registers/flags")
		}
		c.R[cpu386.EBX] = 255
		if paletteBlockBIOS(c, m) {
			t.Fatal("palette range overflow accepted")
		}
	}
}

func TestVGAModeAndLEBus(t *testing.T) {
	le := &machine.LEMachine{Mem: make([]byte, 1<<20)}
	legacy := machine.New()
	v := newBankVideo()
	v.Legacy = legacy
	v.Mode = 0x100
	p := videoPorts{m: legacy, seq: 6, crtc: 0x30}
	c := cpu386.New(videoBus{le, v})
	c.R[cpu386.EAX] = 0xabcd0012
	c.EFlags = cpu386.IF | cpu386.CF | cpu386.OF
	before := c.R
	le.Mem[0x417] = 3
	legacy.VGA.Planes[0][0] = 0xff
	if !setVideoBIOS(c, le, v, &p) || c.R != before || c.EFlags != cpu386.IF|cpu386.CF|cpu386.OF || le.Mem[0x449] != 0x12 || le.Mem[0x417] != 3 || v.Mode != 0 || p.seq != 0 || legacy.VGA.Planes[0][0] != 0 {
		t.Fatal("VGA mode transition")
	}
	p.outputWord(0x3c4, 0x0402)
	if err := c.Bus.Write8(0xa0000, 0x80); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		want := byte(0)
		if i == 2 {
			want = 0x80
		}
		if legacy.VGA.Planes[i][0] != want {
			t.Fatal("VGA plane mapping")
		}
	}
	if le.Mem[0xa0000] != 0 {
		t.Fatal("VGA write leaked into LE RAM")
	}
	p.outputWord(0x3ce, 0x0204)
	value, err := c.Bus.Read8(0xa0000)
	if err != nil || value != 0x80 {
		t.Fatal("VGA read map")
	}
	_, _, latch := legacy.VGA.Regs()
	if latch[2] != 0x80 || latch[0] != 0 {
		t.Fatal("VGA latch")
	}
	c.R[cpu386.EAX] = 0x13
	if setVideoBIOS(c, le, v, &p) || le.Mem[0x449] != 0x12 {
		t.Fatal("unverified mode accepted")
	}
}

func TestCursorBIOSState(t *testing.T) {
	m := &machine.LEMachine{Mem: make([]byte, 0x500)}
	c := cpu386.New(m)
	c.R[cpu386.EAX] = 0xabcd0200
	c.R[cpu386.EBX] = 0x12340300
	c.R[cpu386.EDX] = 0xbeef1122
	c.EFlags = cpu386.IF | cpu386.CF
	before := c.R
	if !cursorBIOS(c, m) || c.R != before || m.Mem[0x456] != 0x22 || m.Mem[0x457] != 0x11 {
		t.Fatal("cursor set")
	}
	m.Mem[0x460] = 7
	m.Mem[0x461] = 6
	c.R[cpu386.EAX] = 0x0300
	c.R[cpu386.EDX] = 0x98760000
	c.R[cpu386.ECX] = 0x56780000
	if !cursorBIOS(c, m) || c.R[cpu386.EDX] != 0x98761122 || c.R[cpu386.ECX] != 0x56780607 || c.EFlags != cpu386.IF|cpu386.CF {
		t.Fatal("cursor query")
	}
	c.R[cpu386.EBX] = 0x0800
	if cursorBIOS(c, m) {
		t.Fatal("invalid page accepted")
	}
}

func TestVideoPortsIsolation(t *testing.T) {
	v := videoPorts{m: machine.New()}
	for _, p := range []struct {
		index, data      uint16
		standard, vendor byte
	}{{0x3c4, 0x3c5, 2, 6}, {0x3d4, 0x3d5, 0x10, 0x30}} {
		v.output(p.index, p.standard)
		v.output(p.data, 0x55)
		v.output(p.index, p.vendor)
		v.output(p.data, 0xaa)
		if b, ok := v.input(p.data); !ok || b != 0xff {
			t.Fatal("vendor register appeared")
		}
		v.output(p.index, p.standard)
		if b, ok := v.input(p.data); !ok || b != 0x55 {
			t.Fatal("vendor index corrupted standard register")
		}
	}
	v.output(0x3cd, 0x55)
	if b, ok := v.input(0x3cd); !ok || b != 0xff {
		t.Fatal("uninstalled Tseng port")
	}
	if _, ok := v.input(0x1234); ok || v.output(0x1234, 1) {
		t.Fatal("unknown port accepted")
	}
	v.output(0x3c8, 5)
	for _, b := range []byte{1, 2, 63} {
		v.output(0x3c9, b)
	}
	v.output(0x3c7, 5)
	for _, want := range []byte{1, 2, 63} {
		if b, ok := v.input(0x3c9); !ok || b != want {
			t.Fatal("DAC round trip")
		}
	}
}
