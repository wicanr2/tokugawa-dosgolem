package main

import (
	"bytes"
	"encoding/binary"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"os"
	"testing"
)

func TestCDIdleStop(t *testing.T) {
	for _, reject := range []string{"", "readonly", "short", "length", "subunit", "drive"} {
		t.Run(reject, func(t *testing.T) {
			m := &machine.LEMachine{Mem: bytes.Repeat([]byte{0xa5}, 128)}
			c := cpu386.New(m)
			c.Seg[cpu386.SegES] = 8
			limit := uint32(63)
			if reject == "short" {
				limit = 15
			}
			c.SetDescriptor(8, cpu386.Descriptor{Base: 32, Limit: limit, Writable: reject != "readonly"})
			c.R[cpu386.EAX] = 0x1510
			c.R[cpu386.ECX] = 3
			c.R[cpu386.EBX] = 4
			c.EFlags = cpu386.IF | cpu386.CF
			packet := m.Mem[36:49]
			packet[0], packet[1], packet[2] = 13, 0, 0x85
			if reject == "length" {
				packet[0] = 14
			}
			if reject == "subunit" {
				packet[1] = 1
			}
			if reject == "drive" {
				c.R[cpu386.ECX] = 2
			}
			before := append([]byte(nil), m.Mem...)
			regs := c.R
			d := &cueDisc{}
			handled := d.handle(c)
			if reject == "" {
				before[39], before[40] = 0, 1
				if !handled || d.AudioIdleStopCount != 1 || !d.handle(c) || d.AudioIdleStopCount != 2 {
					t.Fatal("idle stop failed or was not repeatable")
				}
			} else if handled || d.AudioIdleStopCount != 0 {
				t.Fatal("invalid idle stop was consumed")
			}
			if !bytes.Equal(before, m.Mem) || c.R != regs || c.EFlags != cpu386.IF|cpu386.CF {
				t.Fatal("idle stop changed unrelated state or partly published a rejected request")
			}
		})
	}
}

func TestCDZeroLengthPlayError(t *testing.T) {
	for _, reject := range []int{0, 1, 2, 3, 4} {
		m := &machine.LEMachine{Mem: make([]byte, 128)}
		c := cpu386.New(m)
		c.Seg[cpu386.SegES] = 8
		limit := uint32(63)
		if reject == 2 {
			limit = 24
		}
		c.SetDescriptor(8, cpu386.Descriptor{Base: 32, Limit: limit, Writable: reject != 1})
		c.R[cpu386.EAX] = 0x1510
		c.R[cpu386.ECX] = 3
		c.R[cpu386.EBX] = 4
		c.EFlags = cpu386.IF | cpu386.CF
		packet := m.Mem[36:58]
		packet[0] = 22
		packet[2] = 0x84
		packet[13] = 1
		binary.LittleEndian.PutUint32(packet[14:], 0x81080300)
		m.Mem[58] = 0xaa
		if reject == 3 {
			binary.LittleEndian.PutUint32(packet[18:], 1)
		}
		if reject == 4 {
			packet[13] = 2
		}
		d := &cueDisc{}
		before := c.R
		handled := d.handle(c)
		if reject == 0 {
			if !handled || binary.LittleEndian.Uint16(packet[3:]) != 0x8000 || len(d.AudioRejections) != 1 {
				t.Fatal("zero-length play did not report a real error")
			}
		} else if reject >= 3 {
			if !handled || binary.LittleEndian.Uint16(packet[3:]) != 0x8108 || len(d.AudioRejections) != 1 {
				t.Fatal("invalid audio range did not return an error")
			}
		} else if handled || packet[3] != 0 || packet[4] != 0 || len(d.AudioRejections) != 0 {
			t.Fatal("invalid packet was consumed")
		}
		if c.R != before || c.EFlags != cpu386.IF|cpu386.CF || m.Mem[58] != 0xaa || binary.LittleEndian.Uint32(packet[14:]) != 0x81080300 {
			t.Fatal("play failure changed unrelated state")
		}
	}
}

func TestOriginalDiscMountAndMSCDEX(t *testing.T) {
	if _, err := os.Stat("/original/DX5.cue"); os.IsNotExist(err) {
		t.Skip("original disc not mounted")
	}
	d, err := openCueDisc("/original/DX5.cue")
	if err != nil {
		t.Fatal(err)
	}
	defer d.File.Close()
	if d.SHA256 != "3f511d4c2fb819f950d881697f4e771d37ccfa2b0fd9eca6c4bb48fff6be2448" || len(d.Tracks) != 16 || d.VolumeSectors != 2309 || d.RootSector != 26 {
		t.Fatal("disc identity/geometry mismatch")
	}
	m := &machine.LEMachine{Mem: make([]byte, 128)}
	c := cpu386.New(m)
	c.R[cpu386.EAX] = 0x1500
	c.EFlags = cpu386.IF | cpu386.CF
	if !d.handle(c) || c.R[cpu386.EBX] != 1 || c.R[cpu386.ECX] != 3 || c.EFlags != cpu386.IF|cpu386.CF {
		t.Fatal("MSCDEX drive query")
	}
	c.R[cpu386.EAX] = 0x150d
	c.R[cpu386.EBX] = 4
	c.Seg[cpu386.SegES] = 0x28
	c.SetDescriptor(0x28, cpu386.Descriptor{Base: 64, Limit: 63, Writable: true})
	if !d.handle(c) || m.Mem[68] != 3 {
		t.Fatal("drive list address")
	}
	c.SetDescriptor(0x28, cpu386.Descriptor{Base: 64, Limit: 3, Writable: true})
	if d.handle(c) {
		t.Fatal("invalid transfer accepted")
	}
	c.R[cpu386.EAX] = 0x1510
	if d.handle(c) {
		t.Fatal("unimplemented CD request accepted")
	}
}

func TestCUETimeBoundaries(t *testing.T) {
	v, err := cueFrames("00:35:61")
	if err != nil || v != 2686 {
		t.Fatal(v, err)
	}
	for _, s := range []string{"00:60:00", "00:00:75", "-1:00:00", "00:00", "bad"} {
		if _, err := cueFrames(s); err == nil {
			t.Fatal("invalid time accepted", s)
		}
	}
}

func TestCDTOCAndDPMIHostBridge(t *testing.T) {
	if _, err := os.Stat("/original/DX5.cue"); os.IsNotExist(err) {
		t.Skip("original disc not mounted")
	}
	d, err := openCueDisc("/original/DX5.cue")
	if err != nil {
		t.Fatal(err)
	}
	defer d.File.Close()
	if d.Tracks[1].DiscSector != 2836 || d.LeadOut != 98158 {
		t.Fatalf("pregap mapping: %+v lead=%d", d.Tracks[1], d.LeadOut)
	}
	m := &machine.LEMachine{Mem: make([]byte, 0x100000)}
	m.CPU = cpu386.New(m)
	c := m.CPU
	h := machine.NewDPMIHost(m)
	h.RealModeInterrupt = d.realMode
	c.Seg[cpu386.SegES] = 16
	c.SetDescriptor(16, cpu386.Descriptor{Limit: 0xfffff, Writable: true})
	c.R[cpu386.EAX] = 0x300
	c.R[cpu386.EBX] = 0x2f
	c.R[cpu386.EDI] = 0x100
	c.EFlags = cpu386.CF | cpu386.IF
	p := m.Mem[0x100:0x132]
	binary.LittleEndian.PutUint32(p[16:], 0x200)
	binary.LittleEndian.PutUint32(p[24:], 3)
	binary.LittleEndian.PutUint32(p[28:], 0x1510)
	binary.LittleEndian.PutUint16(p[32:], 0x202)
	req := m.Mem[0x200:0x21a]
	req[0] = 26
	req[2] = 3
	binary.LittleEndian.PutUint32(req[14:], 0x300)
	binary.LittleEndian.PutUint16(req[18:], 7)
	m.Mem[0x300] = 10
	before := c.R
	if !h.Handle(c) {
		t.Fatalf("DPMI bridge: %+v", h.RealModeLast)
	}
	if c.R != before || c.EFlags != cpu386.IF || req[3] != 0 || req[4] != 1 || m.Mem[0x301] != 1 || m.Mem[0x302] != 16 {
		t.Fatal("host bridge state/output")
	}
	want := redBook(d.LeadOut)
	for i, v := range want {
		if m.Mem[0x303+i] != v {
			t.Fatal("lead-out MSF")
		}
	}
	m.Mem[0x300] = 11
	m.Mem[0x301] = 2
	req[3] = 0
	req[4] = 0
	if !h.Handle(c) || m.Mem[0x302] != 61 || m.Mem[0x303] != 39 || m.Mem[0x304] != 0 || m.Mem[0x306] != 0 {
		t.Fatal("track2 MSF/control")
	}
	calls := 0
	h.RealModeInterrupt = func(_ *cpu.CPU, _ uint8) bool { calls++; return true }
	c.SetDescriptor(16, cpu386.Descriptor{Limit: 0xfffff})
	if h.Handle(c) || calls != 0 {
		t.Fatal("readonly output dispatched host service")
	}
}
