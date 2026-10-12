package cpu386

import (
	"encoding/binary"
	"testing"
)

func TestIntegrationDwordRegisterTESTImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct{ value, mask, flags uint32 }{
			{0, 0xffffffff, ZF | PF}, {0xffffffff, 0, ZF | PF},
			{0xffffffff, 0x80000000, SF | PF}, {1, 1, 0},
			{0xffffffff, 0xfe, 0}, {0x80000001, 0x80000001, SF},
			{0xffffffff, 0xffffffff, SF | PF}, {2, 1, ZF | PF},
		} {
			code := binary.LittleEndian.AppendUint32([]byte{0xf7, 0xc0 | reg}, v.mask)
			mem := testBus(append([]byte(nil), code...))
			c := New(mem)
			for i := range c.R {
				c.R[i] = 0x12340000 + uint32(i)*0x10001
			}
			c.R[reg] = v.value
			before := c.R
			c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != before || string(mem) != string(code) || c.EFlags != IF|DF|v.flags || c.EIP != 6 {
				t.Fatalf("TEST reg=%d flags=%x", reg, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0xf7}, {0xf7, 0xc3}, {0xf7, 0xc3, 2, 0, 0}, {0xf7, 0xcb, 2, 0, 0, 0}, {0x26, 0xf7, 0xc3, 2, 0, 0, 0}, {0xf3, 0xf7, 0xc3, 2, 0, 0, 0}} {
		c := New(testBus(code))
		c.R[EBX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected TEST published state")
		}
	}
}

func TestIntegrationDwordCMPMemoryRegister(t *testing.T) {
	for _, v := range []struct{ left, right, flags uint32 }{
		{0, 0, ZF | PF}, {0, 1, CF | AF | SF | PF},
		{0x80000000, 1, OF | AF | PF}, {0x7fffffff, 0xffffffff, CF | OF | SF | PF},
		{0xffffffff, 0xffffffff, ZF | PF}, {0x10, 1, AF | PF}, {20, 0, PF},
	} {
		for source := byte(0); source < 8; source++ {
			for mode := 0; mode < 3; mode++ {
				if mode != 0 && source != EAX {
					continue
				}
				for _, boundsReject := range []bool{false, true} {
					mem := testBus(make([]byte, 256))
					code := []byte{0x39, 0x05 | source<<3, 16, 0, 0, 0}
					address := 144
					if mode == 1 {
						code = []byte{0x39, 0x45, 0xe8}
						address = 208
					}
					if mode == 2 {
						code = []byte{0x39, 0x44, 0x7e, 4}
					}
					copy(mem, code)
					binary.LittleEndian.PutUint32(mem[address:], v.left)
					mem[address-1], mem[address+4] = 0xaa, 0xbb
					c := New(mem)
					for i := range c.R {
						c.R[i] = 0x12340000 + uint32(i)*0x10001
					}
					c.Seg[SegDS], c.Seg[SegSS] = 0x30, 0x38
					limit := uint32(63)
					if boundsReject {
						limit = 18
					}
					c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit})
					c.SetDescriptor(0x38, Descriptor{Base: 192, Limit: limit})
					if mode == 1 {
						c.R[EBP] = 40
					}
					if mode == 2 {
						c.R[ESI], c.R[EDI] = 4, 4
					}
					c.R[source] = v.right
					before := c.R
					memoryBefore := append([]byte(nil), mem...)
					c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
					flagsBefore := c.EFlags
					err := c.Step()
					if c.R != before || string(mem) != string(memoryBefore) {
						t.Fatal("CMP changed registers or memory")
					}
					if boundsReject {
						if err == nil || c.EFlags != flagsBefore {
							t.Fatal("rejected CMP published flags")
						}
					} else if err != nil || c.EFlags != IF|DF|v.flags || c.EIP != uint32(len(code)) {
						t.Fatalf("CMP source=%d mode=%d flags=%x err=%v", source, mode, c.EFlags, err)
					}
				}
			}
		}
	}
	for _, code := range [][]byte{{0x39}, {0x39, 0x05, 16, 0}, {0x66, 0x39, 0x05, 16, 0, 0, 0}, {0x26, 0x39, 0x05, 16, 0, 0, 0}, {0xf3, 0x39, 0x05, 16, 0, 0, 0}} {
		c := New(testBus(code))
		c.R[EAX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected CMP form published state")
		}
	}
}

func TestIntegrationWordNOTRegister(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct{ value, result uint16 }{{0, 0xffff}, {0xffff, 0}, {0x5555, 0xaaaa}, {0xaaaa, 0x5555}, {0x8000, 0x7fff}, {0x7fff, 0x8000}} {
			for _, flags := range []uint32{IF | DF, IF | DF | CF | OF | AF | SF | ZF | PF} {
				c := New(testBus{0x66, 0xf7, 0xd0 | reg})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[reg] = 0xa5a50000 | uint32(v.value)
				want := c.R
				want[reg] = 0xa5a50000 | uint32(v.result)
				c.EFlags = flags
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != flags || c.EIP != 3 {
					t.Fatalf("NOT word reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0xf7}, {0x66, 0xf7, 0x10}, {0x26, 0x66, 0xf7, 0xd3}, {0xf3, 0x66, 0xf7, 0xd3}, {0x66, 0x66, 0xf7, 0xd3}} {
		c := New(testBus(code))
		c.R[EBX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected NOT published state")
		}
	}
}

func TestIntegrationByteNOTRegister(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct{ value, result byte }{{0, 0xff}, {0xff, 0}, {0x55, 0xaa}, {0xaa, 0x55}, {0x80, 0x7f}, {0x7f, 0x80}} {
			for _, flags := range []uint32{IF | DF, IF | DF | CF | OF | AF | SF | ZF | PF} {
				c := New(testBus{0xf6, 0xd0 | reg})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				index, shift := reg&3, uint(0)
				if reg >= 4 {
					shift = 8
				}
				c.R[index] = c.R[index]&^(0xff<<shift) | uint32(v.value)<<shift
				want := c.R
				want[index] = want[index]&^(0xff<<shift) | uint32(v.result)<<shift
				c.EFlags = flags
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != flags || c.EIP != 2 {
					t.Fatalf("NOT byte reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0xf6}, {0xf6, 0x10}, {0x66, 0xf6, 0xd0}, {0x26, 0xf6, 0xd0}, {0xf3, 0xf6, 0xd0}} {
		c := New(testBus(code))
		c.R[EAX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected byte NOT published state")
		}
	}
}

func TestIntegrationWordADDSignExtendedImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct {
			value, result uint16
			imm           byte
			flags         uint32
		}{
			{0xffff, 0, 1, CF | AF | ZF | PF}, {0x7fff, 0x8000, 1, OF | AF | SF | PF}, {1, 0, 0xff, CF | AF | ZF | PF}, {0, 0xff80, 0x80, SF}, {0x80, 0, 0x80, CF | ZF | PF}, {0x8000, 0x7fff, 0xff, CF | OF | PF}, {0x0f, 0x10, 1, AF}, {0x1000, 0x1006, 6, PF},
		} {
			c := New(testBus{0x66, 0x83, 0xc0 | reg, v.imm})
			for i := range c.R {
				c.R[i] = 0x11223344 + uint32(i)*0x10000
			}
			c.R[reg] = 0xa5a50000 | uint32(v.value)
			want := c.R
			want[reg] = 0xa5a50000 | uint32(v.result)
			c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 4 {
				t.Fatalf("ADD word imm reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x83, 0xc7}, {0x66, 0x83, 0x07, 6}, {0x26, 0x66, 0x83, 0xc7, 6}, {0xf3, 0x66, 0x83, 0xc7, 6}} {
		c := New(testBus(code))
		c.R[EDI] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected ADD word imm published state")
		}
	}
}

func TestIntegrationWordSUBImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct {
			value, imm, result uint16
			flags              uint32
		}{
			{0, 1, 0xffff, CF | AF | SF | PF}, {0x8000, 1, 0x7fff, OF | AF | PF}, {0x7fff, 0xffff, 0x8000, CF | OF | SF | PF}, {0xffff, 0xffff, 0, ZF | PF}, {0x1000, 6, 0x0ffa, AF | PF}, {0, 0, 0, ZF | PF}, {0x8000, 0x8000, 0, ZF | PF},
		} {
			c := New(testBus{0x66, 0x81, 0xe8 | reg, byte(v.imm), byte(v.imm >> 8)})
			for i := range c.R {
				c.R[i] = 0x11223344 + uint32(i)*0x10000
			}
			c.R[reg] = 0xa5a50000 | uint32(v.value)
			want := c.R
			want[reg] = 0xa5a50000 | uint32(v.result)
			c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 5 {
				t.Fatalf("SUB word imm reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x81, 0xef}, {0x66, 0x81, 0xef, 0x40}, {0x66, 0x81, 0x2f, 0x40, 1}, {0x26, 0x66, 0x81, 0xef, 0x40, 1}, {0xf3, 0x66, 0x81, 0xef, 0x40, 1}} {
		c := New(testBus(code))
		c.R[EDI] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected SUB word imm published state")
		}
	}
}

func TestIntegrationWordADCRegister(t *testing.T) {
	for dst := byte(0); dst < 8; dst++ {
		for src := byte(0); src < 8; src++ {
			for _, v := range []struct {
				left, right, result uint16
				carry, flags        uint32
			}{
				{0, 0, 0, 0, ZF | PF}, {0, 0, 1, CF, 0}, {0xffff, 0, 0, CF, CF | AF | ZF | PF}, {0x7fff, 0, 0x8000, CF, AF | OF | SF | PF}, {0x8000, 0x8000, 0, 0, CF | OF | ZF | PF}, {0x8000, 0xffff, 0x8000, CF, CF | AF | SF | PF}, {0x0f, 0, 0x10, CF, AF}, {0x5555, 0x5555, 0xaaaa, 0, OF | SF | PF}, {1, 1, 3, CF, PF},
			} {
				if dst == src && v.left != v.right {
					continue
				}
				c := New(testBus{0x66, 0x13, 0xc0 | dst<<3 | src})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[dst] = 0xaaaa0000 | uint32(v.left)
				c.R[src] = 0xbbbb0000 | uint32(v.right)
				want := c.R
				want[dst] = want[dst]&0xffff0000 | uint32(v.result)
				c.EFlags = IF | DF | OF | AF | SF | ZF | PF | v.carry
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 3 {
					t.Fatalf("ADC word dst=%d src=%d registers=%x flags=%x", dst, src, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x13}, {0x66, 0x13, 0x03}, {0x26, 0x66, 0x13, 0xc3}, {0xf3, 0x66, 0x13, 0xc3}} {
		c := New(testBus(code))
		c.R[EAX], c.R[EBX] = 0xabcdef12, 0x12345678
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected word ADC published state")
		}
	}
}

func TestIntegrationSSWordCMPImmediate(t *testing.T) {
	for form := 0; form < 3; form++ {
		for _, v := range []struct {
			value uint16
			imm   byte
			flags uint32
		}{
			{0, 1, CF | AF | SF | PF}, {0x8000, 0xff, CF | AF | SF}, {0x7fff, 0xff, CF | OF | SF | PF}, {1, 1, ZF | PF}, {0, 0, ZF | PF}, {0x8000, 1, OF | AF | PF},
		} {
			for _, reject := range []bool{false, true} {
				code := []byte{0x66, 0x36, 0x83, 0x3d, 32, 0, 0, 0, v.imm}
				if form == 1 {
					code = []byte{0x66, 0x36, 0x83, 0x7d, 0xfc, v.imm}
				}
				if form == 2 {
					code = []byte{0x66, 0x36, 0x83, 0x7c, 0x73, 8, v.imm}
				}
				mem := testBus(make([]byte, 224))
				copy(mem, code)
				c := New(mem)
				c.Seg[SegDS], c.Seg[SegSS] = 0x28, 0x30
				c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
				limit := uint32(63)
				if reject {
					limit = 32
				}
				c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit})
				for i := range c.R {
					c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
				}
				c.R[EBP], c.R[EBX], c.R[ESI] = 36, 16, 4
				binary.LittleEndian.PutUint16(mem[96:], 0x7654)
				binary.LittleEndian.PutUint16(mem[160:], v.value)
				before := c.R
				beforeMem := string(mem)
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				flags := uint32(IF | DF | v.flags)
				if reject {
					flags = c.EFlags
				}
				err := c.Step()
				if (err != nil) != reject || c.R != before || c.EFlags != flags || string(mem) != beforeMem {
					t.Fatalf("SS CMP form=%d reject=%v flags=%x err=%v", form, reject, c.EFlags, err)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x36, 0x83, 0x3d, 32, 0}, {0x66, 0x36, 0x83, 0x3d, 32, 0, 0, 0}, {0x66, 0x36, 0x83, 0xff, 0}, {0x66, 0x36, 0x83, 0x07, 0}, {0x36, 0x83, 0x3d, 32, 0, 0, 0, 0}, {0x66, 0x26, 0x83, 0x3d, 32, 0, 0, 0, 0}} {
		c := New(testBus(code))
		c.R[EDI] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected SS CMP published state")
		}
	}
}

func TestIntegrationDwordANDMemoryImmediate(t *testing.T) {
	for form := 0; form < 3; form++ {
		for _, v := range []struct {
			value, result uint32
			imm           byte
			flags         uint32
		}{
			{0xffffffff, 0xfffffffe, 0xfe, SF}, {0x80000001, 0x80000001, 0xff, SF}, {0x80, 0, 0x7f, ZF | PF}, {0x55, 0x55, 0x55, PF}, {0x12345678, 0, 0, ZF | PF}, {0xffffffff, 0xffffff80, 0x80, SF}, {0x80000000, 0, 0x7f, ZF | PF},
		} {
			for reject := 0; reject < 3; reject++ {
				code := []byte{0x83, 0x25, 32, 0, 0, 0, v.imm}
				if form == 1 {
					code = []byte{0x83, 0x65, 0xfc, v.imm}
				}
				if form == 2 {
					code = []byte{0x83, 0x64, 0x73, 8, v.imm}
				}
				mem := testBus(make([]byte, 224))
				copy(mem, code)
				c := New(mem)
				c.Seg[SegDS], c.Seg[SegSS] = 0x28, 0x30
				limit := uint32(63)
				if reject == 2 {
					limit = 34
				}
				c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
				c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit, Writable: reject != 1})
				for i := range c.R {
					c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
				}
				c.R[EBP], c.R[EBX], c.R[ESI] = 36, 16, 4
				source := 96
				if form == 1 {
					source = 160
				}
				binary.LittleEndian.PutUint32(mem[source:], v.value)
				before := c.R
				wantMem := append([]byte(nil), mem...)
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				flags := c.EFlags
				if reject == 0 {
					binary.LittleEndian.PutUint32(wantMem[source:], v.result)
					flags = IF | DF | v.flags
				}
				err := c.Step()
				if (err != nil) != (reject != 0) || c.R != before || c.EFlags != flags || string(mem) != string(wantMem) {
					t.Fatalf("AND dword mem form=%d reject=%d flags=%x err=%v", form, reject, c.EFlags, err)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x83, 0x25, 32, 0}, {0x83, 0x25, 32, 0, 0, 0}, {0x66, 0x83, 0x65, 0xfc, 0xfe}, {0x26, 0x83, 0x65, 0xfc, 0xfe}, {0xf3, 0x83, 0x65, 0xfc, 0xfe}} {
		c := New(testBus(code))
		c.R[EBP] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected AND memory published state")
		}
	}
}

func TestIntegrationByteXCHGMemory(t *testing.T) {
	for form := 0; form < 3; form++ {
		for reg := byte(0); reg < 8; reg++ {
			if form == 1 && reg != EDX {
				continue
			}
			for reject := 0; reject < 3; reject++ {
				code := []byte{0x86, 0x05 | reg<<3, 32, 0, 0, 0}
				if form == 1 {
					code = []byte{0x86, 0x45 | reg<<3, 0xfc}
				}
				if form == 2 {
					code = append([]byte{0x26}, code...)
				}
				mem := testBus(make([]byte, 224))
				copy(mem, code)
				c := New(mem)
				c.Seg[SegDS], c.Seg[SegSS], c.Seg[SegES] = 0x28, 0x30, 0x30
				limit := uint32(63)
				if reject == 2 {
					limit = 31
				}
				c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
				c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit, Writable: reject != 1})
				for i := range c.R {
					c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
				}
				c.R[EBP] = 36
				index, shift := reg&3, uint(0)
				if reg >= 4 {
					shift = 8
				}
				c.R[index] = c.R[index]&^(0xff<<shift) | 0x55<<shift
				source := 96
				if form != 0 {
					source = 160
				}
				mem[source] = 0x42
				beforeMem := append([]byte(nil), mem...)
				wantMem := append([]byte(nil), mem...)
				want := c.R
				if reject == 0 {
					want[index] = want[index]&^(0xff<<shift) | 0x42<<shift
					wantMem[source] = 0x55
				}
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				flags := c.EFlags
				err := c.Step()
				if (err != nil) != (reject != 0) || c.R != want || c.EFlags != flags || string(mem) != string(wantMem) {
					t.Fatalf("XCHG byte form=%d reg=%d reject=%d registers=%x err=%v", form, reg, reject, c.R, err)
				}
				if reject != 0 && string(mem) != string(beforeMem) {
					t.Fatal("rejected XCHG changed memory")
				}
			}
		}
	}
	for _, code := range [][]byte{{0x26, 0x86}, {0x26, 0x86, 0x05, 32, 0}, {0x26, 0x86, 0xc1}, {0x66, 0x26, 0x86, 0x0f}, {0xf3, 0x26, 0x86, 0x0f}} {
		c := New(testBus(code))
		c.R[ECX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected XCHG published state")
		}
	}
}

func TestIntegrationByteRotateRightOnce(t *testing.T) {
	for reg := 0; reg < 8; reg++ {
		for _, v := range []struct {
			value, result byte
			flags         uint32
		}{{0, 0, 0}, {1, 0x80, CF | OF}, {2, 1, 0}, {0x80, 0x40, OF}, {0x81, 0xc0, CF}, {0xff, 0xff, CF}} {
			c := New(testBus{0xd0, byte(0xc8 + reg)})
			for i := range c.R {
				c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
			}
			index, shift := reg&3, uint(0)
			if reg >= 4 {
				shift = 8
			}
			c.R[index] = c.R[index]&^(0xff<<shift) | uint32(v.value)<<shift
			want := c.R
			want[index] = want[index]&^(0xff<<shift) | uint32(v.result)<<shift
			preserved := uint32(IF | DF | AF | SF | ZF | PF)
			c.EFlags = preserved | CF | OF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != want || c.EFlags != preserved|v.flags || c.EIP != 2 {
				t.Fatalf("ROR byte reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0xd0}, {0xd0, 0x08}, {0x66, 0xd0, 0xcd}, {0xf3, 0xd0, 0xcd}, {0x26, 0xd0, 0xcd}} {
		c := New(testBus(code))
		c.R[ECX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected ROR published state")
		}
	}
}

func TestIntegrationADCSignExtendedImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct {
			value, carry, result, flags uint32
			imm                         byte
		}{
			{0, 0, 0, ZF | PF, 0}, {0, 1, 1, 0, 0}, {0xffffffff, 1, 0, CF | AF | ZF | PF, 0}, {0x7fffffff, 1, 0x80000000, AF | OF | SF | PF, 0}, {0, 1, 0, CF | AF | ZF | PF, 0xff}, {0x80000000, 1, 0x80000000, CF | AF | SF | PF, 0xff}, {0, 0, 0xffffff80, SF, 0x80}, {0x80, 0, 0, CF | ZF | PF, 0x80}, {5, 1, 7, 0, 1},
		} {
			c := New(testBus{0x83, 0xd0 | reg, v.imm})
			for i := range c.R {
				c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
			}
			c.R[reg] = v.value
			want := c.R
			want[reg] = v.result
			c.EFlags = IF | DF | OF | AF | SF | ZF | PF | v.carry
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 3 {
				t.Fatalf("ADC imm reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0x83, 0xd7}, {0x66, 0x83, 0xd7, 0}, {0x83, 0x17, 0}, {0x26, 0x83, 0xd7, 0}, {0xf3, 0x83, 0xd7, 0}} {
		c := New(testBus(code))
		c.R[EDI] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected ADC published state")
		}
	}
}

func TestIntegrationByteRotateOnce(t *testing.T) {
	for reg := 0; reg < 8; reg++ {
		for _, v := range []struct {
			value, result byte
			flags         uint32
		}{{0, 0, 0}, {1, 2, 0}, {0x40, 0x80, OF}, {0x80, 1, CF | OF}, {0x81, 3, CF | OF}, {0xc0, 0x81, CF}, {0xff, 0xff, CF}} {
			for _, preserved := range []uint32{IF | DF, IF | DF | SF | ZF | PF | AF} {
				c := New(testBus{0xd0, byte(0xc0 + reg)})
				for i := range c.R {
					c.R[i] = 0xabcd1234 + uint32(i)*0x10000
				}
				index, shift := reg&3, uint(0)
				if reg >= 4 {
					shift = 8
				}
				c.R[index] = c.R[index]&^(0xff<<shift) | uint32(v.value)<<shift
				want := c.R
				want[index] = want[index]&^(0xff<<shift) | uint32(v.result)<<shift
				c.EFlags = preserved | CF | OF
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != preserved|v.flags || c.EIP != 2 {
					t.Fatalf("ROL reg=%d value=%x registers=%x flags=%x", reg, v.value, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0xd0}, {0xd0, 0x00}, {0x66, 0xd0, 0xc2}, {0xf3, 0xd0, 0xc2}, {0x26, 0xd0, 0xc2}, {0xd2, 0xc2}, {0xc0, 0xc2, 1}} {
		c := New(testBus(code))
		c.R[EDX] = 0xabcdef80
		before := c.R
		c.EFlags = IF | AF | CF | OF
		if c.Step() == nil || c.R != before || c.EFlags != IF|AF|CF|OF {
			t.Fatal("rejected ROL published state")
		}
	}
}

func TestIntegrationWordADDMemory(t *testing.T) {
	for form := 0; form < 3; form++ {
		for dst := byte(0); dst < 8; dst++ {
			if form != 0 && dst != EDX {
				continue
			}
			for _, v := range []struct {
				left, right, result uint16
				flags               uint32
			}{
				{0xffff, 1, 0, CF | AF | ZF | PF}, {0x7fff, 1, 0x8000, OF | AF | SF | PF}, {0x8000, 0x8000, 0, CF | OF | ZF | PF}, {1, 1, 2, 0}, {0x0f, 1, 0x10, AF},
			} {
				for _, reject := range []bool{false, true} {
					code := []byte{0x66, 0x03, 0x05 | dst<<3, 32, 0, 0, 0}
					if form == 1 {
						code = []byte{0x66, 0x03, 0x45 | dst<<3, 20}
					}
					if form == 2 {
						code = []byte{0x66, 0x03, 0x44 | dst<<3, 0x73, 8}
					}
					mem := testBus(make([]byte, 224))
					copy(mem, code)
					c := New(mem)
					c.Seg[SegDS], c.Seg[SegSS] = 0x28, 0x30
					limit := uint32(63)
					if reject {
						limit = 32
					}
					c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit})
					c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit})
					for i := range c.R {
						c.R[i] = 0x11223344 + uint32(i)*0x10000
					}
					c.R[EBP], c.R[EBX], c.R[ESI] = 12, 16, 4
					c.R[dst] = 0xa5a50000 | uint32(v.left)
					source := 96
					if form == 1 {
						source = 160
					}
					binary.LittleEndian.PutUint16(mem[source:], v.right)
					beforeMem := string(mem)
					before := c.R
					want := before
					want[dst] = 0xa5a50000 | uint32(v.result)
					c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
					flags := uint32(IF | DF | v.flags)
					if reject {
						want = before
						flags = c.EFlags
					}
					err := c.Step()
					if (err != nil) != reject || c.R != want || c.EFlags != flags || string(mem) != beforeMem {
						t.Fatalf("ADD word memory form=%d dst=%d reject=%v registers=%x flags=%x err=%v", form, dst, reject, c.R, c.EFlags, err)
					}
				}
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x03, 0x05, 32, 0}, {0x26, 0x66, 0x03, 0xc2}, {0xf3, 0x66, 0x03, 0xc2}} {
		c := New(testBus(code))
		c.R[EAX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected word ADD published state")
		}
	}
}

func TestIntegrationByteADDMemory(t *testing.T) {
	for form := 0; form < 3; form++ {
		for dst := byte(0); dst < 8; dst++ {
			if form != 0 && dst != EDX {
				continue
			}
			for _, v := range []struct {
				left, right, result byte
				flags               uint32
			}{
				{0xff, 1, 0, CF | AF | ZF | PF}, {0x7f, 1, 0x80, OF | AF | SF}, {0x80, 0x80, 0, CF | OF | ZF | PF}, {1, 1, 2, 0}, {0x0f, 1, 0x10, AF},
			} {
				for _, reject := range []bool{false, true} {
					code := []byte{0x02, 0x05 | dst<<3, 32, 0, 0, 0}
					if form == 1 {
						code = []byte{0x02, 0x45 | dst<<3, 0xfc}
					}
					if form == 2 {
						code = []byte{0x02, 0x44 | dst<<3, 0x73, 8}
					}
					mem := testBus(make([]byte, 224))
					copy(mem, code)
					c := New(mem)
					c.Seg[SegDS], c.Seg[SegSS] = 0x28, 0x30
					limit := uint32(63)
					if reject {
						limit = 31
					}
					c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit})
					c.SetDescriptor(0x30, Descriptor{Base: 128, Limit: limit})
					for i := range c.R {
						c.R[i] = 0xa5a55a5a + uint32(i)*0x10000
					}
					c.R[EBP], c.R[EBX], c.R[ESI] = 36, 16, 4
					index, shift := dst&3, uint(0)
					if dst >= 4 {
						shift = 8
					}
					c.R[index] = c.R[index]&^(0xff<<shift) | uint32(v.left)<<shift
					source := 96
					if form == 1 {
						source = 160
					}
					mem[source] = v.right
					beforeMem := string(mem)
					before := c.R
					want := before
					want[index] = want[index]&^(0xff<<shift) | uint32(v.result)<<shift
					c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
					flags := uint32(IF | DF | v.flags)
					if reject {
						want = before
						flags = c.EFlags
					}
					err := c.Step()
					if (err != nil) != reject || c.R != want || c.EFlags != flags || string(mem) != beforeMem {
						t.Fatalf("ADD byte form=%d dst=%d reject=%v registers=%x flags=%x err=%v", form, dst, reject, c.R, c.EFlags, err)
					}
				}
			}
		}
	}
	for _, code := range [][]byte{{0x02}, {0x02, 0x05, 32, 0}, {0x22, 0x05, 32, 0, 0, 0}, {0x26, 0x02, 0xc2}, {0x66, 0x02, 0xc2}, {0xf3, 0x02, 0xc2}} {
		c := New(testBus(code))
		c.R[EAX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected byte ADD published state")
		}
	}
}

func TestIntegrationWordRotateCL(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		if reg == ECX {
			continue
		}
		for _, v := range []struct {
			value, result uint16
			count         byte
			flags         uint32
		}{
			{0x8001, 0x8001, 0, CF}, {0x8001, 3, 1, CF | OF}, {0x8001, 0xc000, 15, 0}, {0x8001, 0x8001, 16, CF}, {0x8001, 3, 17, CF}, {0x8001, 0xc000, 31, 0}, {0x8001, 0x8001, 32, CF}, {0x8001, 3, 33, CF | OF}, {0x7fff, 0xfffe, 1, OF}, {0xffff, 0xffff, 1, CF}, {1, 2, 1, 0},
		} {
			for _, oldOF := range []uint32{0, OF} {
				c := New(testBus{0x66, 0xd3, 0xc0 | reg})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[reg] = 0xa5a50000 | uint32(v.value)
				c.R[ECX] = 0x55550000 | uint32(v.count)
				want := c.R
				want[reg] = 0xa5a50000 | uint32(v.result)
				preserved := uint32(IF | DF | SF | ZF | AF | PF)
				c.EFlags = preserved | oldOF | CF
				flags := preserved | v.flags
				if v.count&31 != 1 {
					flags |= oldOF
				}
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != flags || c.EIP != 3 {
					t.Fatalf("ROL word reg=%d count=%d registers=%x flags=%x", reg, v.count, c.R, c.EFlags)
				}
			}
		}
	}
	c := New(testBus{0x66, 0xd3, 0xc1})
	c.R[ECX] = 0xaaaa0001
	c.EFlags = IF | CF | OF
	if err := c.Step(); err != nil || c.R[ECX] != 0xaaaa0002 || c.EFlags != IF {
		t.Fatal("ROL CX uses original CL", err)
	}
	for _, code := range [][]byte{{0x66, 0xd3}, {0x66, 0xd3, 0x00}, {0x66, 0xd3, 0xe0}, {0x26, 0x66, 0xd3, 0xc0}, {0xf3, 0x66, 0xd3, 0xc0}} {
		c := New(testBus(code))
		c.R[EAX], c.R[ECX] = 0xabcdef12, 1
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected word ROL published state")
		}
	}
}

func TestIntegrationWordXORImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, v := range []struct {
			value, imm, result uint16
			flags              uint32
		}{
			{0, 0xffff, 0xffff, SF | PF}, {0xffff, 0xffff, 0, ZF | PF}, {0x1234, 0xffff, 0xedcb, SF}, {0x8000, 1, 0x8001, SF}, {0x55, 0, 0x55, PF},
		} {
			code := []byte{0x66, 0x81, 0xf0 | reg, byte(v.imm), byte(v.imm >> 8)}
			c := New(testBus(code))
			for i := range c.R {
				c.R[i] = 0x12345678 + uint32(i)*0x10000
			}
			c.R[reg] = 0xa5a50000 | uint32(v.value)
			want := c.R
			want[reg] = 0xa5a50000 | uint32(v.result)
			c.EFlags = IF | DF | AF | CF | OF | SF | ZF | PF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 5 {
				t.Fatalf("XOR word reg=%d result=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x81, 0xf2}, {0x66, 0x81, 0xf2, 0xff}, {0x66, 0x81, 0x32, 0xff, 0xff}, {0x26, 0x66, 0x81, 0xf2, 0xff, 0xff}, {0xf3, 0x66, 0x81, 0xf2, 0xff, 0xff}} {
		c := New(testBus(code))
		c.R[EDX] = 0xabcdef12
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected word XOR published state")
		}
	}
}

func TestIntegrationWordANDReverseRegisterEncoding(t *testing.T) {
	for dst := byte(0); dst < 8; dst++ {
		for src := byte(0); src < 8; src++ {
			for _, v := range []struct {
				left, right, result uint16
				flags               uint32
			}{
				{0xf0f0, 0x0f0f, 0, ZF | PF}, {0x8001, 0xffff, 0x8001, SF}, {0xffff, 0x55, 0x55, PF}, {0x1234, 0x1111, 0x1010, 0}, {0x8000, 0x8000, 0x8000, SF | PF},
			} {
				if dst == src && v.left != v.right {
					continue
				}
				c := New(testBus{0x66, 0x23, 0xc0 | dst<<3 | src})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[dst] = 0xaaaa0000 | uint32(v.left)
				c.R[src] = 0xbbbb0000 | uint32(v.right)
				want := c.R
				want[dst] = want[dst]&0xffff0000 | uint32(v.result)
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 3 {
					t.Fatalf("AND dst=%d src=%d registers=%x flags=%x", dst, src, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x66, 0x23}, {0x66, 0x23, 0x02}, {0xf3, 0x66, 0x23, 0xc2}, {0x26, 0x66, 0x23, 0xc2}} {
		c := New(testBus(code))
		c.R[EAX], c.R[EDX] = 0xabcdef12, 0x1234ffff
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected word AND published state")
		}
	}
}

func TestIntegrationDwordANDReverseRegisterEncoding(t *testing.T) {
	for dst := byte(0); dst < 8; dst++ {
		for src := byte(0); src < 8; src++ {
			for _, v := range []struct{ left, right, result, flags uint32 }{
				{0xf0f0f0f0, 0x0f0f0f0f, 0, ZF | PF}, {0x80000001, 0xffffffff, 0x80000001, SF}, {0xffffffff, 0x55, 0x55, PF}, {0x12345678, 0x11111111, 0x10101010, 0}, {0x80000000, 0x80000000, 0x80000000, SF | PF}, {0, 0, 0, ZF | PF},
			} {
				if dst == src && v.left != v.right {
					continue
				}
				c := New(testBus{0x23, 0xc0 | dst<<3 | src})
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[dst], c.R[src] = v.left, v.right
				want := c.R
				want[dst] = v.result
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 2 {
					t.Fatalf("AND32 dst=%d src=%d registers=%x flags=%x", dst, src, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x23}, {0x23, 0x02}, {0xf3, 0x23, 0xc2}, {0x26, 0x23, 0xc2}} {
		c := New(testBus(code))
		c.R[EAX], c.R[EDX] = 0xabcdef12, 0x1234ffff
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected dword AND published state")
		}
	}
}

func TestIntegrationRegisterXORSignExtendedImmediate(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, word := range []bool{false, true} {
			for _, v := range []struct {
				value, result, flags uint32
				imm                  byte
			}{
				{0, 0, ZF | PF, 0},
				{0, 0xffffffff, SF | PF, 0xff},
				{0xffffffff, 0, ZF | PF, 0xff},
				{0x80008000, 0x80008001, SF, 1},
				{0x7fff7fff, 0x8000807f, SF, 0x80},
				{0x55, 0, ZF | PF, 0x55},
				{0x55, 0x55, PF, 0},
			} {
				code := []byte{0x83, 0xf0 | reg, v.imm}
				if word {
					code = append([]byte{0x66}, code...)
				}
				c := New(testBus(code))
				for i := range c.R {
					c.R[i] = 0x11223344 + uint32(i)*0x10000
				}
				c.R[reg] = v.value
				if word {
					c.R[reg] = 0xa5a50000 | (v.value & 0xffff)
				}
				want := c.R
				want[reg] = v.result
				if word {
					want[reg] = 0xa5a50000 | (v.result & 0xffff)
				}
				c.EFlags = IF | DF | CF | OF | AF | SF | ZF | PF
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != uint32(len(code)) {
					t.Fatalf("XOR imm8 reg=%d word=%v registers=%x flags=%x", reg, word, c.R, c.EFlags)
				}
			}
		}
	}
	for _, code := range [][]byte{{0x83, 0xf0}, {0x66, 0x83, 0xf2}, {0x83, 0x30, 1}, {0x26, 0x83, 0xf2, 1}, {0xf3, 0x83, 0xf2, 1}, {0x66, 0x66, 0x83, 0xf2, 1}} {
		c := New(testBus(code))
		c.R[EAX], c.R[EDX] = 0xabcd1234, 0x56789abc
		before := c.R
		c.EFlags = IF | CF | OF | AF
		if c.Step() == nil || c.R != before || c.EFlags != IF|CF|OF|AF {
			t.Fatal("rejected XOR immediate published state")
		}
	}
}

func TestIntegrationByteXORImmediate(t *testing.T) {
	for reg := 0; reg < 8; reg++ {
		for _, v := range []struct {
			value, imm, result byte
			flags              uint32
		}{{0x55, 0x55, 0, ZF | PF}, {0, 0xff, 0xff, SF | PF}, {0xf0, 0xf1, 1, 0}} {
			c := New(testBus{0x80, byte(0xf0 + reg), v.imm})
			for i := range c.R {
				c.R[i] = 0x12345678
			}
			c.setReg8(reg, v.value)
			before := c.R
			c.EFlags = IF | DF | CF | OF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			want := before
			index := reg & 3
			shift := uint(0)
			if reg >= 4 {
				shift = 8
			}
			want[index] = (want[index] &^ (0xff << shift)) | uint32(v.result)<<shift
			if c.R != want || c.EFlags != IF|DF|v.flags || c.EIP != 3 {
				t.Fatalf("byte XOR: reg=%d registers=%x flags=%x", reg, c.R, c.EFlags)
			}
		}
	}
	for _, code := range [][]byte{{0x80, 0xf2}, {0x26, 0x80, 0xf2, 0xff}, {0xf3, 0x80, 0xf2, 0xff}} {
		c := New(testBus(code))
		c.R[EDX] = 0xabcd1234
		c.EFlags = IF | CF | OF
		if c.Step() == nil || c.R[EDX] != 0xabcd1234 || c.EFlags != IF|CF|OF {
			t.Fatal("invalid XOR published state")
		}
	}
}

func TestIntegrationCSIndirectCALL(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x2e, 0xff, 0x13}) // LE 核心以線性 EIP 取指令；CS base 只在資料覆寫驗證。
		binary.LittleEndian.PutUint32(mem[48:], 8)
		c := New(mem)
		c.Seg[SegCS] = 8
		c.Seg[SegSS] = 16
		limit := uint32(63)
		if reject == 1 {
			limit = 18
		}
		c.SetDescriptor(8, Descriptor{Base: 32, Limit: limit})
		c.SetDescriptor(16, Descriptor{Base: 96, Limit: 31, Writable: reject != 2})
		c.R[EBX] = 16
		c.R[ESP] = 24
		c.EFlags = IF | CF | OF
		err := c.Step()
		if reject > 0 {
			if err == nil || c.R[ESP] != 24 || binary.LittleEndian.Uint32(mem[116:]) != 0 {
				t.Fatal("rejected CS CALL published stack")
			}
		} else if err != nil || c.EIP != 8 || c.R[ESP] != 20 || binary.LittleEndian.Uint32(mem[116:]) != 3 {
			t.Fatal("CS CALL target/return", err)
		}
		if c.R[EBX] != 16 || c.EFlags != IF|CF|OF {
			t.Fatal("CALL flags/source")
		}
	}
}

func TestIntegrationByteShiftCL(t *testing.T) {
	for _, v := range []struct {
		code, value, count, result byte
		flags                      uint32
	}{{0xe4, 1, 0, 1, CF | OF}, {0xe4, 1, 3, 8, OF}, {0xe4, 0x80, 1, 0, CF | OF | ZF | PF}, {0xe4, 1, 1, 2, 0}, {0xec, 0x80, 1, 0x40, OF}, {0xec, 3, 1, 1, CF}, {0xe4, 1, 32, 1, CF | OF}, {0xe4, 1, 33, 2, 0}} {
		c := New(testBus{0xd2, v.code})
		c.R[EAX] = 0xabcd0077 | uint32(v.value)<<8
		c.R[ECX] = uint32(v.count)
		c.EFlags = IF | DF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != 0xabcd0077|uint32(v.result)<<8 || c.R[ECX] != uint32(v.count) || c.EFlags != IF|DF|v.flags {
			t.Fatalf("byte shift CL width flags=%x", c.EFlags)
		}
	}
}

func TestIntegrationESByteStore(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x26, 0x88, 0x4c, 0x06, 2})
		mem[80] = 0xaa
		mem[81] = 0xbb
		c := New(mem)
		c.R[ESI] = 12
		c.R[EAX] = 2
		c.R[ECX] = 0xabcdef12
		c.Seg[SegES] = 0x28
		limit := uint32(63)
		if reject == 2 {
			limit = 15
		}
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
		c.EFlags = IF | CF | OF
		before := c.R
		err := c.Step()
		if reject > 0 {
			if err == nil || mem[80] != 0xaa {
				t.Fatal("rejected ES byte store published")
			}
		} else if err != nil || mem[80] != 0x12 {
			t.Fatal("ES byte store", err)
		}
		if c.R != before || mem[81] != 0xbb || c.EFlags != IF|CF|OF {
			t.Fatal("ES byte store width/flags")
		}
	}
}

func TestIntegrationLFSStackPointer(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x0f, 0xb4, 0x75, 0xf6})
		binary.LittleEndian.PutUint32(mem[86:], 0xabcdef12)
		binary.LittleEndian.PutUint16(mem[90:], 0x30)
		c := New(mem)
		c.R[EBP] = 32
		c.R[ESI] = 0x12345678
		c.Seg[SegSS] = 0x28
		c.Seg[SegFS] = 0
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
		if reject == 1 {
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 26})
		}
		if reject != 2 {
			c.SetDescriptor(0x30, Descriptor{Limit: 100})
		}
		c.EFlags = IF | DF | CF
		err := c.Step()
		if reject > 0 {
			if err == nil || c.R[ESI] != 0x12345678 || c.Seg[SegFS] != 0 {
				t.Fatal("rejected LFS published state")
			}
		} else if err != nil || c.R[ESI] != 0xabcdef12 || c.Seg[SegFS] != 0x30 {
			t.Fatal("LFS far pointer", err)
		}
		if c.EFlags != IF|DF|CF || c.R[EBP] != 32 {
			t.Fatal("LFS unrelated state")
		}
	}
}

func TestIntegrationADC(t *testing.T) {
	for _, v := range []struct{ left, right, carry, result, flags uint32 }{{0, 0, 0, 0, ZF | PF}, {0xffffffff, 0, 1, 0, CF | AF | ZF | PF}, {0x7fffffff, 0, 1, 0x80000000, AF | OF | SF | PF}, {0x80000000, 0xffffffff, 1, 0x80000000, CF | AF | SF | PF}, {1, 0xffffffff, 1, 1, CF | AF}, {0x80000000, 0x80000000, 0, 0, CF | OF | ZF | PF}} {
		for _, immediate := range []bool{false, true} {
			if immediate && v.right != uint32(int32(int8(v.right))) {
				continue
			}
			code := []byte{0x13, 0xc1}
			if immediate {
				code = []byte{0x83, 0xd0, byte(v.right)}
			}
			c := New(testBus(code))
			c.R[EAX] = v.left
			c.R[ECX] = v.right
			c.EFlags = IF | DF | v.carry
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R[EAX] != v.result || c.R[ECX] != v.right || c.EFlags != IF|DF|v.flags {
				t.Fatalf("ADC registers=%x flags=%x", c.R, c.EFlags)
			}
		}
	}
}

func TestIntegrationROLImmediate(t *testing.T) {
	for _, v := range []struct {
		value         uint32
		count         byte
		result, flags uint32
	}{{0x80000001, 0, 0x80000001, CF | OF}, {0x80000001, 32, 0x80000001, CF | OF}, {0x80000001, 1, 3, CF | OF}, {0x40000000, 1, 0x80000000, OF}, {1, 1, 2, 0}, {0x12345678, 8, 0x34567812, OF}, {0x12345678, 31, 0x091a2b3c, OF}} {
		c := New(testBus{0xc1, 0xc2, v.count})
		c.R[EDX] = v.value
		c.EFlags = IF | DF | AF | ZF | SF | PF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EDX] != v.result || c.EFlags != IF|DF|AF|ZF|SF|PF|v.flags {
			t.Fatal("ROL immediate flags/result")
		}
	}
}

func TestIntegrationRCR1(t *testing.T) {
	for _, v := range []struct{ value, carry, result, flags uint32 }{{0, 1, 0x80000000, OF}, {1, 0, 0, CF}, {0x80000000, 1, 0xc0000000, 0}, {0x80000001, 0, 0x40000000, CF | OF}, {0, 0, 0, 0}} {
		c := New(testBus{0xd1, 0xda})
		c.R[EDX] = v.value
		c.EFlags = IF | DF | AF | ZF | SF | PF | OF | v.carry
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EDX] != v.result || c.EFlags != IF|DF|AF|ZF|SF|PF|v.flags {
			t.Fatal("RCR1 carry/overflow")
		}
	}
}

func TestIntegrationJSAndADCAccumulator(t *testing.T) {
	for _, sign := range []bool{false, true} {
		c := New(testBus{0x78, 4})
		c.EFlags = IF | CF
		if sign {
			c.EFlags |= SF
		}
		flags := c.EFlags
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		want := uint32(2)
		if sign {
			want = 6
		}
		if c.EIP != want || c.EFlags != flags {
			t.Fatal("JS condition")
		}
	}
	for _, v := range []struct{ a, b, carry, result, flags uint32 }{{0xffffffff, 0, 1, 0, CF | AF | ZF | PF}, {0x7fffffff, 0, 1, 0x80000000, AF | OF | SF | PF}, {1, 0xffffffff, 1, 1, CF | AF}} {
		code := make([]byte, 5)
		code[0] = 0x15
		binary.LittleEndian.PutUint32(code[1:], v.b)
		c := New(testBus(code))
		c.R[EAX] = v.a
		c.EFlags = IF | v.carry
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != v.result || c.EFlags != IF|v.flags {
			t.Fatal("ADC accumulator")
		}
	}
}

func TestIntegrationBitRuntime(t *testing.T) {
	for _, v := range []struct{ source, result uint32 }{{0, 0xabcd1234}, {1, 0}, {639, 9}, {0x80000000, 31}, {0xffffffff, 31}} {
		c := New(testBus{0x0f, 0xbd, 0xc8})
		c.R[EAX] = v.source
		c.R[ECX] = 0xabcd1234
		c.EFlags = IF | CF | OF | ZF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		want := uint32(IF | CF | OF)
		if v.source == 0 {
			want |= ZF
		}
		if c.R[EAX] != v.source || c.R[ECX] != v.result || c.EFlags != want {
			t.Fatal("BSR result/flags")
		}
	}
	for _, v := range []struct {
		value, result byte
		flags         uint32
	}{{0, 0, ZF | PF}, {1, 0xff, CF | AF | SF | PF}, {0x80, 0x80, CF | OF | SF}, {0xff, 1, CF | AF}} {
		c := New(testBus{0xf6, 0xd9})
		c.R[ECX] = 0xabcd1200 | uint32(v.value)
		c.EFlags = IF | DF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[ECX] != 0xabcd1200|uint32(v.result) || c.EFlags != IF|DF|v.flags {
			t.Fatal("NEG byte width/flags")
		}
	}
	for _, v := range []struct {
		value         uint32
		count         byte
		result, flags uint32
	}{{0x80000001, 0, 0x80000001, CF | OF}, {0x80000001, 32, 0x80000001, CF | OF}, {0x80000001, 1, 3, CF | OF}, {0x40000000, 1, 0x80000000, OF}, {1, 1, 2, 0}, {0x12345678, 8, 0x34567812, OF}, {0x12345678, 31, 0x091a2b3c, OF}} {
		c := New(testBus{0xd3, 0xc0})
		c.R[EAX] = v.value
		c.R[ECX] = uint32(v.count)
		c.EFlags = IF | DF | AF | ZF | SF | PF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != v.result || c.R[ECX] != uint32(v.count) || c.EFlags != IF|DF|AF|ZF|SF|PF|v.flags {
			t.Fatal("ROL CL flags/result")
		}
	}
}

func TestIntegrationJNS(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, delta := range []byte{3, 0xfc} {
			mem := testBus(make([]byte, 16))
			copy(mem[4:], []byte{0x79, delta})
			c := New(mem)
			c.EIP = 4
			c.EFlags = IF | CF
			if negative {
				c.EFlags |= SF
			}
			flags := c.EFlags
			before := c.R
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			want := uint32(6)
			if !negative {
				want = uint32(6 + int32(int8(delta)))
			}
			if c.EIP != want || c.EFlags != flags || c.R != before {
				t.Fatal("JNS condition/target")
			}
		}
	}
}

func TestIntegrationJNO(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		for _, delta := range []byte{3, 0xfc} {
			mem := testBus(make([]byte, 16))
			copy(mem[4:], []byte{0x71, delta})
			c := New(mem)
			c.EIP = 4
			c.EFlags = IF | CF
			if overflow {
				c.EFlags |= OF
			}
			flags := c.EFlags
			before := c.R
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			want := uint32(6)
			if !overflow {
				want = uint32(6 + int32(int8(delta)))
			}
			if c.EIP != want || c.EFlags != flags || c.R != before {
				t.Fatal("JNO condition/target")
			}
		}
	}
}

func TestIntegrationWordORMemory(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 9, 5, 16, 0, 0, 0})
		binary.LittleEndian.PutUint16(mem[80:], 0x8000)
		mem[82] = 0xaa
		c := New(mem)
		c.R[EAX] = 0xabcd0003
		c.Seg[SegDS] = 0x28
		limit := uint32(63)
		if reject == 2 {
			limit = 16
		}
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
		c.EFlags = IF | DF | CF | OF
		before := c.R
		err := c.Step()
		want := uint32(IF | DF | SF | PF)
		if reject > 0 {
			want = IF | DF | CF | OF
			if err == nil || binary.LittleEndian.Uint16(mem[80:]) != 0x8000 {
				t.Fatal("rejected OR16 published state")
			}
		} else if err != nil || binary.LittleEndian.Uint16(mem[80:]) != 0x8003 {
			t.Fatal("OR16 memory", err)
		}
		if c.R != before || c.EFlags != want || mem[82] != 0xaa {
			t.Fatal("OR16 width/flags")
		}
	}
}

func TestIntegrationSegmentLogicByte(t *testing.T) {
	for _, v := range []struct {
		modrm, imm, initial, result byte
		flags                       uint32
	}{{0x20, 0xcf, 0xff, 0xcf, SF | PF}, {8, 0x20, 0x10, 0x30, PF}, {8, 0x80, 0, 0x80, SF}, {8, 0, 0, 0, ZF | PF}} {
		for _, seg := range []struct {
			prefix byte
			index  int
		}{{0x26, SegES}, {0x64, SegFS}} {
			for _, reject := range []int{0, 1, 2} {
				mem := testBus(make([]byte, 128))
				copy(mem, []byte{seg.prefix, 0x80, v.modrm, v.imm})
				mem[80] = v.initial
				mem[81] = 0xaa
				c := New(mem)
				c.R[EAX] = 16
				c.Seg[seg.index] = 0x28
				limit := uint32(63)
				if reject == 2 {
					limit = 15
				}
				c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
				c.EFlags = IF | DF | CF | OF
				before := c.R
				err := c.Step()
				want := uint32(IF|DF) | v.flags
				if reject > 0 {
					want = IF | DF | CF | OF
					if err == nil || mem[80] != v.initial {
						t.Fatal("rejected logic published state")
					}
				} else if err != nil || mem[80] != v.result {
					t.Fatal("segment logic", err)
				}
				if c.R != before || c.EFlags != want || mem[81] != 0xaa {
					t.Fatal("segment logic width/flags")
				}
			}
		}
	}
	for _, code := range [][]byte{{0x64, 0x80, 0x30, 1}, {0x64, 0x80, 0xc8, 1}, {0x64, 0x90}} {
		c := New(testBus(code))
		c.R[EAX] = 0x1234
		c.EFlags = IF | CF
		if c.Step() == nil || c.R[EAX] != 0x1234 || c.EFlags != IF|CF {
			t.Fatal("unverified FS form accepted")
		}
	}
}

func TestIntegrationTESTDwordMemory(t *testing.T) {
	for _, reject := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x85, 0x14, 0x85, 64, 0, 0, 0})
		binary.LittleEndian.PutUint32(mem[76:], 0x80000001)
		c := New(mem)
		c.R[EAX] = 3
		c.R[EDX] = 0x80000000
		c.Seg[SegDS] = 0x28
		limit := uint32(127)
		if reject {
			limit = 78
		}
		c.SetDescriptor(0x28, Descriptor{Limit: limit})
		c.EFlags = IF | CF | OF
		before := c.R
		err := c.Step()
		want := uint32(IF | SF | PF)
		if reject {
			want = IF | CF | OF
			if err == nil {
				t.Fatal("TEST boundary")
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if c.R != before || c.EFlags != want || binary.LittleEndian.Uint32(mem[76:]) != 0x80000001 {
			t.Fatal("TEST memory state")
		}
	}
}

func TestIntegrationALArithmetic(t *testing.T) {
	for _, v := range []struct {
		op, a, imm, result byte
		flags              uint32
	}{{4, 0x7f, 1, 0x80, OF | AF | SF}, {4, 0xff, 1, 0, CF | AF | ZF | PF}, {0x2c, 0, 1, 0xff, CF | AF | SF | PF}, {0x2c, 0x80, 1, 0x7f, OF | AF}} {
		c := New(testBus([]byte{v.op, v.imm}))
		c.R[EAX] = 0xabcdef00 | uint32(v.a)
		c.EFlags = IF | DF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != 0xabcdef00|uint32(v.result) || c.EFlags != IF|DF|v.flags {
			t.Fatal("AL arithmetic width/flags")
		}
	}
}

func TestIntegrationORByteDestination(t *testing.T) {
	for dst := 0; dst < 8; dst++ {
		for src := 0; src < 8; src++ {
			c := New(testBus([]byte{0x08, byte(0xc0 | src<<3 | dst)}))
			for i := range c.R {
				c.R[i] = 0xabcda501
			}
			c.EFlags = IF | DF | CF | OF
			want := c.R
			dreg, shift := dst, 0
			if dst >= 4 {
				dreg = dst - 4
				shift = 8
			}
			value := c.reg8(dst) | c.reg8(src)
			want[dreg] = want[dreg]&^(0xff<<shift) | uint32(value)<<shift
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			flags := uint32(IF | DF)
			if value&0x80 != 0 {
				flags |= SF
			}
			if value == 0 {
				flags |= ZF
			}
			if parity(value) {
				flags |= PF
			}
			if c.R != want || c.EFlags != flags {
				t.Fatal("OR byte width/flags")
			}
		}
	}
}

func TestIntegrationCMPWordMemory(t *testing.T) {
	for _, reject := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0x3b, 0x55, 0x20})
		binary.LittleEndian.PutUint16(mem[96:], 1)
		c := New(mem)
		c.R[EBP] = 0
		c.R[EDX] = 0xaaaa0000
		c.Seg[SegSS] = 0x28
		limit := uint32(63)
		if reject {
			limit = 32
		}
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit})
		c.EFlags = IF | OF
		before := c.R
		err := c.Step()
		want := uint32(IF | CF | AF | SF | PF)
		if reject {
			want = IF | OF
			if err == nil {
				t.Fatal("CMP boundary accepted")
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if c.EFlags != want || c.R != before || binary.LittleEndian.Uint16(mem[96:]) != 1 {
			t.Fatal("CMP memory state")
		}
	}
}

func TestIntegrationESMOVSXWord(t *testing.T) {
	for _, v := range []struct {
		word   uint16
		result uint32
	}{{0x8000, 0xffff8000}, {0xffff, 0xffffffff}, {0x7fff, 0x7fff}} {
		for _, reject := range []bool{false, true} {
			mem := testBus(make([]byte, 128))
			copy(mem, []byte{0x26, 0x0f, 0xbf, 0x32})
			binary.LittleEndian.PutUint16(mem[80:], v.word)
			c := New(mem)
			c.R[EDX] = 16
			c.R[ESI] = 0xabcd1234
			c.Seg[SegES] = 0x28
			limit := uint32(63)
			if reject {
				limit = 16
			}
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit})
			c.EFlags = IF | CF | OF
			err := c.Step()
			if reject {
				if err == nil || c.R[ESI] != 0xabcd1234 {
					t.Fatal("rejected MOVSX changed destination")
				}
			} else if err != nil || c.R[ESI] != v.result {
				t.Fatal("MOVSX sign", err)
			}
			if c.EFlags != IF|CF|OF {
				t.Fatal("MOVSX flags")
			}
		}
	}
}

func TestIntegrationESWordStore(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0x26, 0x89, 0x1a})
		mem[82] = 0x55
		mem[83] = 0xaa
		mem[84] = 0xcc
		c := New(mem)
		c.R[EDX] = 18
		c.R[EBX] = 0xabcd1234
		c.Seg[SegES] = 0x28
		limit := uint32(63)
		if reject == 2 {
			limit = 18
		}
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
		c.EFlags = IF | CF | OF
		before := c.R
		err := c.Step()
		if reject == 0 {
			if err != nil || mem[82] != 0x34 || mem[83] != 0x12 {
				t.Fatal("ES word store", err)
			}
		} else if err == nil || mem[82] != 0x55 || mem[83] != 0xaa {
			t.Fatal("rejected ES store changed bytes")
		}
		if c.R != before || c.EFlags != IF|CF|OF || mem[84] != 0xcc {
			t.Fatal("ES store published extra state")
		}
	}
}

func TestIntegrationCMPWordRegister(t *testing.T) {
	for _, v := range []struct{ a, b, flags uint32 }{{0xaaaa0000, 0xbbbb0000, ZF | PF}, {0xffff8000, 1, OF | AF | PF}, {0xaaaa0000, 0xbbbb0001, CF | AF | SF | PF}} {
		c := New(testBus([]byte{0x66, 0x39, 0xc8}))
		c.R[EAX] = v.a
		c.R[ECX] = v.b
		c.EFlags = IF | DF
		before := c.R
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R != before || c.EFlags != IF|DF|v.flags {
			t.Fatalf("CMP word flags %x", c.EFlags)
		}
	}
}

func TestIntegrationIMULWord(t *testing.T) {
	for _, v := range []struct {
		a, b, result uint16
		overflow     bool
	}{{2, 0xfffd, 0xfffa, false}, {0x8000, 0xffff, 0x8000, true}, {0x7fff, 2, 0xfffe, true}, {0, 0x8000, 0, false}} {
		for _, memory := range []bool{false, true} {
			mem := testBus(make([]byte, 128))
			copy(mem, []byte{0x66, 0x0f, 0xaf, 0xc1})
			c := New(mem)
			c.R[EAX] = 0xabcd0000 | uint32(v.a)
			c.R[ECX] = uint32(v.b)
			if memory {
				copy(mem, []byte{0x66, 0x0f, 0xaf, 5, 64, 0, 0, 0})
				binary.LittleEndian.PutUint16(mem[64:], v.b)
				c.Seg[SegDS] = 0x28
				c.SetDescriptor(0x28, Descriptor{Limit: 127})
			}
			c.EFlags = IF | ZF | AF | CF | OF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			want := uint32(IF | ZF | AF)
			if v.overflow {
				want |= CF | OF
			}
			if c.R[EAX] != 0xabcd0000|uint32(v.result) || c.EFlags != want {
				t.Fatal("IMUL word width/overflow")
			}
		}
	}
	mem := testBus([]byte{0x66, 0x0f, 0xaf, 5, 64, 0, 0, 0})
	c := New(mem)
	c.R[EAX] = 0xabcd1234
	c.Seg[SegDS] = 0x28
	c.SetDescriptor(0x28, Descriptor{Limit: 64})
	c.EFlags = IF | CF
	if c.Step() == nil || c.R[EAX] != 0xabcd1234 || c.EFlags != IF|CF {
		t.Fatal("IMUL bound failure published state")
	}
}

func TestIntegrationLES(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0xc4, 0x5d, 0xfa})
		binary.LittleEndian.PutUint32(mem[90:], 0x12345678)
		binary.LittleEndian.PutUint16(mem[94:], 0x28)
		c := New(mem)
		c.Seg[SegSS] = 0x30
		c.Seg[SegES] = 0x10
		c.R[EBP] = 32
		c.R[EBX] = 0xabcdef12
		limit := uint32(63)
		if reject == 1 {
			limit = 30
		}
		c.SetDescriptor(0x30, Descriptor{Base: 64, Limit: limit})
		if reject != 2 {
			c.SetDescriptor(0x28, Descriptor{Limit: 1023})
		}
		c.EFlags = IF | CF | OF
		err := c.Step()
		if reject == 0 {
			if err != nil || c.R[EBX] != 0x12345678 || c.Seg[SegES] != 0x28 {
				t.Fatal("LES far pointer", err)
			}
		} else if err == nil || c.R[EBX] != 0xabcdef12 || c.Seg[SegES] != 0x10 {
			t.Fatal("rejected LES changed destination")
		}
		if c.EFlags != IF|CF|OF {
			t.Fatal("LES flags")
		}
	}
}

func TestIntegrationWordOUT(t *testing.T) {
	for _, accept := range []bool{false, true} {
		c := New(testBus([]byte{0x66, 0xef}))
		c.R[EAX] = 0xabcd1234
		c.R[EDX] = 0xbeef03c4
		c.EFlags = IF | CF | OF
		before := c.R
		calls := 0
		c.PortOut16 = func(p, v uint16) bool {
			calls++
			if p != 0x3c4 || v != 0x1234 {
				t.Fatal("OUT word width")
			}
			return accept
		}
		err := c.Step()
		if (err == nil) != accept || calls != 1 || c.R != before || c.EFlags != IF|CF|OF {
			t.Fatal("word OUT contract", err)
		}
	}
	c := New(testBus([]byte{0x66, 0xef}))
	if c.Step() == nil {
		t.Fatal("missing word platform callback accepted")
	}
}

func TestIntegrationTESTWordRegister(t *testing.T) {
	for _, v := range []struct{ a, b, flags uint32 }{{0x80000000, 0xffffffff, ZF | PF}, {0xffff8001, 0xaaaa8000, SF | PF}, {0xffff0003, 0xffff0001, 0}} {
		mem := testBus([]byte{0x66, 0x85, 0xc8})
		c := New(mem)
		c.R[EAX] = v.a
		c.R[ECX] = v.b
		c.EFlags = IF | DF | CF | OF
		before := c.R
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R != before || c.EFlags != IF|DF|v.flags {
			t.Fatalf("TEST word changed operands or incorrect flags %x", c.EFlags)
		}
	}
}

func TestIntegrationCMPSB(t *testing.T) {
	for _, v := range []struct {
		prefix      byte
		count       uint32
		reverse     bool
		left, right []byte
		remaining   uint32
		equal       bool
	}{{0xf3, 3, false, []byte{1, 2, 3}, []byte{1, 2, 3}, 0, true}, {0xf3, 3, false, []byte{1, 2, 3}, []byte{1, 4, 3}, 1, false}, {0xf2, 3, false, []byte{1, 2, 3}, []byte{3, 2, 1}, 1, true}, {0xf3, 3, true, []byte{1, 2, 3}, []byte{1, 2, 3}, 0, true}, {0xf3, 0, false, nil, nil, 0, false}} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{v.prefix, 0xa6})
		copy(mem[32:], v.left)
		copy(mem[64:], v.right)
		c := New(mem)
		c.Seg[SegDS] = 0x28
		c.Seg[SegES] = 0x30
		c.SetDescriptor(0x28, Descriptor{Limit: 127})
		c.SetDescriptor(0x30, Descriptor{Limit: 127})
		c.R[ESI] = 32
		c.R[EDI] = 64
		c.R[ECX] = v.count
		c.EFlags = IF | CF
		if v.reverse {
			c.R[ESI] += 2
			c.R[EDI] += 2
			c.EFlags |= DF
		}
		before := c.R
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		used := v.count - v.remaining
		delta := used
		if v.reverse {
			delta = 0 - used
		}
		if c.R[ECX] != v.remaining || c.R[ESI] != before[ESI]+delta || c.R[EDI] != before[EDI]+delta || (c.EFlags&ZF != 0) != v.equal || c.EFlags&IF == 0 {
			t.Fatal("CMPSB count/direction/equality")
		}
		if v.count == 0 && c.EFlags != IF|CF {
			t.Fatal("zero count flags")
		}
	}
	c := New(testBus{0xf3, 0xa6})
	c.R[ECX] = 1000001
	c.EFlags = IF | CF
	if c.Step() == nil || c.R[ECX] != 1000001 || c.EFlags != IF|CF {
		t.Fatal("unbounded compare accepted")
	}
}

func TestIntegrationXCHGAccumulator(t *testing.T) {
	for rm := byte(1); rm < 8; rm++ {
		for _, word := range []bool{false, true} {
			code := []byte{0x90 + rm}
			if word {
				code = append([]byte{0x66}, code...)
			}
			c := New(testBus(code))
			c.R[EAX] = 0x12345678
			c.R[rm] = 0xabcd9876
			c.EFlags = IF | CF | OF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			wantA, wantR := uint32(0xabcd9876), uint32(0x12345678)
			if word {
				wantA, wantR = 0x12349876, 0xabcd5678
			}
			if c.R[EAX] != wantA || c.R[rm] != wantR || c.EFlags != IF|CF|OF {
				t.Fatal("XCHG accumulator width/flags")
			}
		}
	}
}

func TestIntegrationESMOVWordIndirect(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0x26, 0x8b, 0x10})
		mem[80] = 0x34
		mem[81] = 0x80
		c := New(mem)
		c.R[EAX] = 16
		c.R[EDX] = 0xabcdffff
		c.Seg[SegES] = 0x28
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 17})
		if invalid {
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 16})
		}
		c.EFlags = IF | CF | OF
		err := c.Step()
		if invalid {
			if err == nil || c.R[EDX] != 0xabcdffff {
				t.Fatal("invalid word load published")
			}
		} else if err != nil || c.R[EDX] != 0xabcd8034 {
			t.Fatal("ES MOV word width", err)
		}
		if c.EFlags != IF|CF|OF {
			t.Fatal("MOV changed flags")
		}
	}
}

func TestIntegrationESMOVZXByteWord(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0x26, 0x0f, 0xb6, 0x08})
		mem[80] = 0xff
		c := New(mem)
		c.R[EAX] = 16
		c.R[ECX] = 0xabcdffff
		c.Seg[SegES] = 0x28
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
		if invalid {
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 15})
		}
		c.EFlags = IF | CF | OF
		before := c.R[ECX]
		err := c.Step()
		if invalid {
			if err == nil || c.R[ECX] != before {
				t.Fatal("invalid ES MOVZX published")
			}
		} else if err != nil || c.R[ECX] != 0xabcd00ff || c.R[EAX] != 16 {
			t.Fatal("ES MOVZX width", err)
		}
		if c.EFlags != IF|CF|OF {
			t.Fatal("MOVZX changed flags")
		}
	}
}

func TestIntegrationCompareIndirectByte(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x38, 0x10})
		mem[80] = 0x7f
		c := New(mem)
		c.R[EAX] = 16
		c.R[EDX] = 0x80
		c.Seg[SegDS] = 0x28
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 16})
		if invalid {
			c.R[EAX] = 17
		}
		before := c.R
		c.EFlags = IF | DF | CF
		err := c.Step()
		if invalid {
			if err == nil || c.EFlags != IF|DF|CF {
				t.Fatal("rejected CMP changed flags")
			}
		} else if err != nil || c.EFlags != IF|DF|CF|OF|SF|PF {
			t.Fatalf("CMP flags=%x err=%v", c.EFlags, err)
		}
		if c.R != before || mem[80] != 0x7f {
			t.Fatal("CMP changed operands")
		}
	}
}

func TestIntegrationROR32(t *testing.T) {
	for _, v := range []struct {
		value         uint32
		count         byte
		result, flags uint32
	}{{0x12345678, 8, 0x78123456, OF}, {0x80000001, 1, 0xc0000000, CF}, {1, 1, 0x80000000, CF | OF}, {0x80000001, 31, 3, OF}, {0x12345678, 32, 0x12345678, CF | OF}, {1, 33, 0x80000000, CF | OF}} {
		c := New(testBus{0xc1, 0xca, v.count})
		c.R[EDX] = v.value
		c.EFlags = IF | DF | AF | ZF | PF | SF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EDX] != v.result || c.EFlags != IF|DF|AF|ZF|PF|SF|v.flags {
			t.Fatalf("ROR count=%d result=%x flags=%x", v.count, c.R[EDX], c.EFlags)
		}
	}
}

func TestIntegrationENTERZeroNesting(t *testing.T) {
	for _, allocation := range []byte{0, 8, 40} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0xc8, allocation, 0, 0})
		c := New(mem)
		c.R[ESP] = 32
		c.R[EBP] = 0x12345678
		c.Seg[SegSS] = 0x28
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63, Writable: true})
		c.EFlags = IF | CF | OF
		err := c.Step()
		if allocation == 40 {
			if err == nil || c.R[ESP] != 32 || c.R[EBP] != 0x12345678 || mem[92] != 0 {
				t.Fatal("ENTER underflow published frame")
			}
		} else if err != nil || c.R[EBP] != 28 || c.R[ESP] != 28-uint32(allocation) || binary.LittleEndian.Uint32(mem[92:]) != 0x12345678 {
			t.Fatal("ENTER frame", err)
		}
		if c.EFlags != IF|CF|OF {
			t.Fatal("ENTER flags")
		}
	}
	c := New(testBus{0xc8, 0, 0, 1})
	c.R[ESP] = 32
	if c.Step() == nil || c.R[ESP] != 32 {
		t.Fatal("unverified nesting accepted")
	}
}

func TestIntegrationSegmentStoreIndirect(t *testing.T) {
	for _, word := range []bool{false, true} {
		for _, writable := range []bool{false, true} {
			mem := testBus(make([]byte, 128))
			code := []byte{0x8c, 0x03}
			if word {
				code = append([]byte{0x66}, code...)
			}
			copy(mem, code)
			mem[80] = 0xaa
			mem[81] = 0xbb
			mem[82] = 0xcc
			c := New(mem)
			c.R[EBX] = 16
			c.Seg[SegDS] = 0x28
			c.Seg[SegES] = 0x160
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63, Writable: writable})
			c.EFlags = IF | CF | OF
			err := c.Step()
			if writable {
				if err != nil || mem[80] != 0x60 || mem[81] != 1 || mem[82] != 0xcc {
					t.Fatal("indirect word selector store", err)
				}
			} else if err == nil || mem[80] != 0xaa || mem[81] != 0xbb {
				t.Fatal("readonly indirect store published")
			}
			if c.EFlags != IF|CF|OF {
				t.Fatal("segment store changed flags")
			}
		}
	}
}

func TestIntegrationPOPMemoryCommit(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 160))
		copy(mem, []byte{0x8f, 0x47, 4})
		binary.LittleEndian.PutUint32(mem[112:], 0x12345678)
		c := New(mem)
		c.Seg[SegSS] = 0x28
		c.Seg[SegDS] = 0x30
		c.R[ESP] = 16
		c.R[EDI] = 4
		c.SetDescriptor(0x28, Descriptor{Base: 96, Limit: 63})
		limit := uint32(63)
		if reject == 2 {
			limit = 10
		}
		c.SetDescriptor(0x30, Descriptor{Base: 32, Limit: limit, Writable: reject != 1})
		c.EFlags = IF | CF | OF
		err := c.Step()
		if reject == 0 {
			if err != nil || c.R[ESP] != 20 || binary.LittleEndian.Uint32(mem[40:]) != 0x12345678 {
				t.Fatal("POP memory result", err)
			}
		} else if err == nil || c.R[ESP] != 16 || binary.LittleEndian.Uint32(mem[40:]) != 0 {
			t.Fatal("failed POP published state")
		}
		if c.EFlags != IF|CF|OF || binary.LittleEndian.Uint32(mem[112:]) != 0x12345678 {
			t.Fatal("POP changed flags/source")
		}
	}
}

func TestIntegrationCSLEAIgnoresSegmentBase(t *testing.T) {
	mem := testBus([]byte{0x2e, 0x8d, 0x86, 0xc0, 0xaf, 2, 0})
	c := New(mem)
	c.R[ESI] = 141
	c.Seg[SegCS] = 0x08
	c.SetDescriptor(0x08, Descriptor{Base: 0x100000, Limit: 0})
	c.EFlags = IF | CF | OF
	if err := c.Step(); err != nil {
		t.Fatal(err)
	}
	if c.R[EAX] != 0x2afc0+141 || c.EFlags != IF|CF|OF || c.EIP != 7 {
		t.Fatal("LEA consulted segment or flags")
	}
}

func TestIntegrationSegmentLoadIndirectBase(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x8e, 0x03})
		mem[80] = 0x30
		c := New(mem)
		c.R[EBX] = 16
		c.Seg[SegDS] = 0x28
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 17})
		if !invalid {
			c.SetDescriptor(0x30, Descriptor{Limit: 127})
		}
		c.EFlags = IF | CF | OF
		err := c.Step()
		if invalid {
			if err == nil || c.Seg[SegES] != 0 {
				t.Fatal("invalid indirect selector accepted")
			}
		} else if err != nil || c.Seg[SegES] != 0x30 {
			t.Fatal("indirect selector", err)
		}
		if c.EFlags != IF|CF|OF || c.R[EBX] != 16 {
			t.Fatal("indirect MOV changed unrelated state")
		}
	}
}

func TestIntegrationSegmentLoadESP(t *testing.T) {
	for _, invalid := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x8e, 0x44, 0x24, 4})
		mem[84] = 0x30
		mem[85] = 0
		mem[86] = 0xff
		c := New(mem)
		c.R[ESP] = 16
		c.Seg[SegSS] = 0x28
		c.Seg[SegES] = 0x10
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
		c.SetDescriptor(0x30, Descriptor{Limit: 127})
		if invalid == 1 {
			delete(c.Descriptors, 0x30)
		}
		if invalid == 2 {
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 20})
		}
		c.EFlags = IF | CF | OF
		err := c.Step()
		if invalid == 0 {
			if err != nil || c.Seg[SegES] != 0x30 {
				t.Fatal("ESP selector load", err)
			}
		} else if err == nil || c.Seg[SegES] != 0x10 {
			t.Fatal("invalid ESP selector accepted")
		}
		if c.R[ESP] != 16 || c.EFlags != IF|CF|OF {
			t.Fatal("MOV segment changed unrelated state")
		}
	}
}

func TestIntegrationSegmentStoreESP(t *testing.T) {
	for _, reject := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x8c, 0x5c, 0x24, 4})
		mem[84] = 0xaa
		mem[85] = 0xbb
		mem[86] = 0xcc
		c := New(mem)
		c.R[ESP] = 16
		c.Seg[SegDS] = 0x160
		c.Seg[SegSS] = 0x28
		limit := uint32(63)
		if reject == 2 {
			limit = 20
		}
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
		c.EFlags = IF | CF | OF
		err := c.Step()
		if reject == 0 {
			if err != nil || mem[84] != 0x60 || mem[85] != 1 || mem[86] != 0xcc {
				t.Fatal("ESP segment word store", err)
			}
		} else if err == nil || mem[84] != 0xaa || mem[85] != 0xbb {
			t.Fatal("rejected segment store published bytes")
		}
		if c.EFlags != IF|CF|OF {
			t.Fatal("segment MOV changed flags")
		}
	}
}

func TestIntegrationSBBImmediate(t *testing.T) {
	for _, v := range []struct {
		a                    uint32
		imm                  byte
		carry, result, flags uint32
	}{{3, 1, CF, 1, 0}, {0, 0, CF, 0xffffffff, CF | AF | SF | PF}, {0x80000000, 0x7f, CF, 0x7fffff80, OF | AF}, {0, 0xff, CF, 0, CF | AF | ZF | PF}, {0xffffffff, 0xff, CF, 0xffffffff, CF | AF | SF | PF}, {5, 2, 0, 3, PF}} {
		for reg := 0; reg < 8; reg++ {
			mem := testBus([]byte{0x83, byte(0xd8 | reg), v.imm})
			c := New(mem)
			c.R[reg] = v.a
			c.EFlags = IF | DF | v.carry
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R[reg] != v.result || c.EFlags != IF|DF|v.flags {
				t.Fatalf("SBB %08x,%02x CF=%d: %08x flags=%x", v.a, v.imm, v.carry, c.R[reg], c.EFlags)
			}
		}
	}
}

func TestIntegrationSARMemoryOne(t *testing.T) {
	for _, v := range []struct{ a, result, flags uint32 }{{1, 0, CF | ZF | PF}, {0x80000001, 0xc0000000, CF | SF | PF}, {0x7fffffff, 0x3fffffff, CF | PF}, {0, 0, ZF | PF}, {0xffffffff, 0xffffffff, CF | SF | PF}} {
		for _, reject := range []int{0, 1, 2} {
			mem := testBus(make([]byte, 128))
			copy(mem, []byte{0xd1, 0x7d, 0xf0})
			binary.LittleEndian.PutUint32(mem[80:], v.a)
			mem[84] = 0xaa
			c := New(mem)
			c.R[EBP] = 32
			c.Seg[SegSS] = 0x30
			limit := uint32(63)
			if reject == 2 {
				limit = 18
			}
			c.SetDescriptor(0x30, Descriptor{Base: 64, Limit: limit, Writable: reject != 1})
			c.EFlags = IF | DF | CF | OF
			err := c.Step()
			if reject > 0 {
				if err == nil || binary.LittleEndian.Uint32(mem[80:]) != v.a || c.EFlags != IF|DF|CF|OF {
					t.Fatal("rejected SAR published state")
				}
				continue
			}
			if err != nil || binary.LittleEndian.Uint32(mem[80:]) != v.result || mem[84] != 0xaa || c.EFlags != IF|DF|v.flags {
				t.Fatal("SAR sign/width/flags", err)
			}
		}
	}
}

func TestIntegrationTESTWordMemory(t *testing.T) {
	for _, v := range []struct {
		a, mask uint16
		flags   uint32
	}{{0x8001, 0xfff0, SF | PF}, {3, 1, 0}, {0, 0xffff, ZF | PF}} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0xf7, 0x05, 16, 0, 0, 0, byte(v.mask), byte(v.mask >> 8)})
		binary.LittleEndian.PutUint16(mem[80:], v.a)
		mem[82] = 0xff
		c := New(mem)
		c.Seg[SegDS] = 0x30
		c.SetDescriptor(0x30, Descriptor{Base: 64, Limit: 17})
		c.EFlags = IF | DF | CF | OF
		regs := c.R
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R != regs || binary.LittleEndian.Uint16(mem[80:]) != v.a || c.EFlags != IF|DF|v.flags || c.EIP != 9 {
			t.Fatal("TEST word width/flags")
		}
		c.EIP = 0
		c.EFlags = IF | CF | OF
		c.SetDescriptor(0x30, Descriptor{Base: 64, Limit: 16})
		if c.Step() == nil || c.EFlags != IF|CF|OF {
			t.Fatal("TEST word bounds published flags")
		}
	}
}

func TestIntegrationORDwordImmediateMemory(t *testing.T) {
	for _, v := range []struct {
		a             uint32
		imm           byte
		result, flags uint32
	}{{0x12340000, 1, 0x12340001, 0}, {0, 0x80, 0xffffff80, SF}, {0, 0, 0, ZF | PF}, {0x80000000, 0x7f, 0x8000007f, SF}} {
		for _, reject := range []int{0, 1, 2} {
			mem := testBus(make([]byte, 128))
			copy(mem, []byte{0x83, 0x0e, v.imm})
			binary.LittleEndian.PutUint32(mem[64:], v.a)
			mem[68] = 0xaa
			c := New(mem)
			c.Seg[SegDS] = 0x30
			c.R[ESI] = 32
			limit := uint32(95)
			if reject == 2 {
				limit = 34
			}
			c.SetDescriptor(0x30, Descriptor{Base: 32, Limit: limit, Writable: reject != 1})
			c.EFlags = IF | DF | CF | OF
			err := c.Step()
			if reject > 0 {
				if err == nil || binary.LittleEndian.Uint32(mem[64:]) != v.a || c.EFlags != IF|DF|CF|OF {
					t.Fatal("rejected OR changed state")
				}
				continue
			}
			if err != nil || binary.LittleEndian.Uint32(mem[64:]) != v.result || mem[68] != 0xaa || c.EFlags != IF|DF|v.flags {
				t.Fatalf("OR immediate width/flags: %x %x %v", mem[64:69], c.EFlags, err)
			}
		}
	}
}

func TestIntegrationANDRegisters(t *testing.T) {
	for _, v := range []struct{ a, b, result, flags uint32 }{{0xffffffff, 0x80000003, 0x80000003, SF | PF}, {0x100, 1, 0, ZF | PF}, {3, 1, 1, 0}} {
		c := New(testBus{0x21, 0xc8})
		c.R[EAX] = v.a
		c.R[ECX] = v.b
		c.EFlags = IF | DF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != v.result || c.R[ECX] != v.b || c.EFlags != IF|DF|v.flags {
			t.Fatal("AND32 flags/source")
		}
	}
}

func TestIntegrationSBBWordEncodings(t *testing.T) {
	for _, v := range []struct{ a, b, carry, result, flags uint32 }{{0, 0, 1, 0xffff, CF | AF | SF | PF}, {0, 0xffff, 1, 0, CF | AF | ZF | PF}, {0x8000, 0x7fff, 1, 0, OF | AF | ZF | PF}, {3, 1, 1, 1, 0}} {
		for _, op := range []byte{0x19, 0x1b} {
			modrm := byte(0xc8)
			if op == 0x1b {
				modrm = 0xc1
			}
			c := New(testBus{0x66, op, modrm})
			c.R[EAX] = 0xabcd0000 | v.a
			c.R[ECX] = 0x12340000 | v.b
			c.EFlags = IF | DF | v.carry
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R[EAX] != 0xabcd0000|v.result || c.R[ECX] != 0x12340000|v.b || c.EFlags != IF|DF|v.flags {
				t.Fatalf("SBB16 result=%x flags=%x", c.R, c.EFlags)
			}
		}
	}
}

func TestIntegrationCMCOnlyChangesCarry(t *testing.T) {
	for flags := uint32(0); flags < 4096; flags++ {
		c := New(testBus{0xf5})
		c.EFlags = flags
		c.R[EAX] = 0x12345678
		before := c.R
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.EFlags != flags^CF || c.R != before || c.EIP != 1 {
			t.Fatal("CMC changed unrelated state")
		}
	}
}

func TestIntegrationXCHGRegisterPairs(t *testing.T) {
	for a := byte(0); a < 8; a++ {
		for b := byte(0); b < 8; b++ {
			for _, word := range []bool{false, true} {
				code := []byte{0x87, 0xc0 | (a << 3) | b}
				if word {
					code = append([]byte{0x66}, code...)
				}
				c := New(testBus(code))
				for i := range c.R {
					c.R[i] = 0x12340000 + uint32(i)*0x11110001
				}
				before := c.R
				c.EFlags = IF | DF | CF | OF
				if err := c.Step(); err != nil {
					t.Fatal(err)
				}
				want := before
				if word {
					want[a] = before[a]&0xffff0000 | before[b]&0xffff
					want[b] = before[b]&0xffff0000 | before[a]&0xffff
				} else {
					want[a], want[b] = before[b], before[a]
				}
				if c.R != want || c.EFlags != IF|DF|CF|OF {
					t.Fatal("XCHG width/alias/flags")
				}
			}
		}
	}
}

func TestIntegrationSBBReverseEncoding(t *testing.T) {
	for _, v := range []struct{ a, b, carry, result, flags uint32 }{
		{3, 1, 1, 1, 0}, {0, 0, 1, 0xffffffff, CF | AF | SF | PF}, {0x80000000, 0x7fffffff, 1, 0, OF | AF | ZF | PF}, {0, 0xffffffff, 1, 0, CF | AF | ZF | PF}, {5, 2, 0, 3, PF},
	} {
		c := New(testBus{0x19, 0xc8})
		c.R[EAX] = v.a
		c.R[ECX] = v.b
		c.EFlags = IF | DF | v.carry
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EAX] != v.result || c.R[ECX] != v.b || c.EFlags != IF|DF|v.flags {
			t.Fatalf("SBB %x flags=%x", c.R, c.EFlags)
		}
	}
}

func TestIntegrationPushPopGS(t *testing.T) {
	for _, selector := range []uint16{0, 0x30} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x0f, 0xa8, 0x0f, 0xa9})
		c := New(mem)
		c.Seg[SegSS] = 0x28
		c.Seg[SegGS] = selector
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63, Writable: true})
		c.SetDescriptor(0x30, Descriptor{Limit: 127})
		c.R[ESP] = 32
		c.EFlags = IF | OF | CF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[ESP] != 28 || mem[92] != byte(selector) || mem[93] != 0 || c.EFlags != IF|OF|CF {
			t.Fatal("PUSH GS slot/flags")
		}
		c.Seg[SegGS] = 0
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[ESP] != 32 || c.Seg[SegGS] != selector || c.EFlags != IF|OF|CF {
			t.Fatal("POP GS result/flags")
		}
	}
	mem := testBus(make([]byte, 128))
	copy(mem, []byte{0x0f, 0xa9})
	mem[92] = 0x38
	c := New(mem)
	c.R[ESP] = 28
	c.Seg[SegSS] = 0x28
	c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
	if c.Step() == nil || c.R[ESP] != 28 || c.Seg[SegGS] != 0 {
		t.Fatal("invalid GS selector was accepted")
	}
}

func TestIntegrationESCompareAndLoadByte(t *testing.T) {
	for _, load := range []bool{false, true} {
		for _, invalid := range []bool{false, true} {
			mem := testBus(make([]byte, 128))
			op := byte(0x3a)
			if load {
				op = 0x8a
			}
			copy(mem, []byte{0x26, op, 0x22})
			mem[68] = 0x80
			c := New(mem)
			c.R[EAX] = 0x12345678
			c.R[EDX] = 4
			c.setReg8(4, 0x7f)
			c.Seg[SegES] = 0x28
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 31})
			if invalid {
				c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 3})
			}
			c.EFlags = IF | CF | OF
			before := c.R
			err := c.Step()
			if invalid {
				if err == nil || c.R != before || c.EFlags != IF|CF|OF {
					t.Fatal("rejected ES byte read changed state")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if load {
				if c.R[EAX] != 0x12348078 || c.EFlags != IF|CF|OF {
					t.Fatal("ES MOV byte width/flags")
				}
			} else if c.R != before || c.EFlags != IF|CF|OF|SF|PF {
				t.Fatalf("ES CMP registers=%x flags=%x", c.R, c.EFlags)
			}
			if mem[68] != 0x80 {
				t.Fatal("read changed source")
			}
		}
	}
}

func TestIntegrationSegmentLoadWithWordPrefixES(t *testing.T) {
	for _, invalid := range []int{0, 1, 2} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x26, 0x66, 0x8e, 0x1d, 16, 0, 0, 0})
		mem[80] = 0x30
		mem[81] = 0
		mem[82] = 0xff
		c := New(mem)
		c.Seg[SegES] = 0x28
		c.Seg[SegDS] = 0x10
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63})
		c.SetDescriptor(0x30, Descriptor{Base: 100, Limit: 27})
		c.EFlags = IF | CF | OF
		if invalid == 1 {
			c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 16})
		}
		if invalid == 2 {
			delete(c.Descriptors, 0x30)
		}
		err := c.Step()
		if invalid == 0 {
			if err != nil || c.Seg[SegDS] != 0x30 {
				t.Fatal("ES word selector load", err)
			}
		} else if err == nil || c.Seg[SegDS] != 0x10 {
			t.Fatal("rejected load published segment")
		}
		if c.EFlags != IF|CF|OF || mem[82] != 0xff {
			t.Fatal("MOV segment altered unrelated state")
		}
	}
}

func TestIntegrationByteSUBRegisters(t *testing.T) {
	for _, v := range []struct {
		a, b, result byte
		flags        uint32
	}{
		{0, 0, 0, ZF | PF}, {0, 1, 255, CF | AF | SF | PF}, {128, 1, 127, OF | AF}, {127, 255, 128, CF | OF | SF},
	} {
		for _, high := range []bool{false, true} {
			modrm := byte(0xc8)
			if high {
				modrm = 0xec
			}
			c := New(testBus{0x28, modrm})
			c.R[EAX] = 0x12345678
			c.R[ECX] = 0x9abcdef0
			dst, src := 0, 1
			if high {
				dst, src = 4, 5
			}
			c.setReg8(dst, v.a)
			c.setReg8(src, v.b)
			before := c.R
			c.EFlags = IF | DF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.reg8(dst) != v.result || c.R[ECX] != before[ECX] || c.EFlags != IF|DF|v.flags {
				t.Fatalf("SUB8 %x flags=%x", c.R, c.EFlags)
			}
			mask := uint32(0xffffff00)
			if high {
				mask = 0xffff00ff
			}
			if c.R[EAX]&mask != before[EAX]&mask {
				t.Fatal("SUB8 clobbered other bits")
			}
		}
	}
}

func TestIntegrationRedundantDSOnImmediateMOV(t *testing.T) {
	for reg := byte(0); reg < 8; reg++ {
		for _, word := range []bool{false, true} {
			code := []byte{0x3e, 0xb8 + reg, 0x34, 0x12, 0x78, 0x56}
			want := uint32(0x56781234)
			if word {
				code = []byte{0x66, 0x3e, 0xb8 + reg, 0x34, 0x12}
				want = 0xabcd1234
			}
			c := New(testBus(code))
			c.R[reg] = 0xabcd0000
			c.EFlags = IF | CF | OF
			if err := c.Step(); err != nil {
				t.Fatal(err)
			}
			if c.R[reg] != want || c.EFlags != IF|CF|OF || c.EIP != uint32(len(code)) {
				t.Fatal("redundant DS changed MOV semantics")
			}
		}
	}
	for _, code := range [][]byte{{0x3e, 0x3e, 0xba, 1, 0, 0, 0}, {0x3e, 0x8b, 0x02}} {
		c := New(testBus(code))
		if c.Step() == nil {
			t.Fatal("unverified DS form accepted")
		}
	}
}

func TestIntegrationWordORRegisters(t *testing.T) {
	for _, v := range []struct{ a, b, result, flags uint32 }{
		{0x12348000, 0xabcd0003, 0x12348003, SF | PF},
		{0x12340000, 0xabcd0000, 0x12340000, ZF | PF},
		{0x12340001, 0xabcd0000, 0x12340001, 0},
	} {
		c := New(testBus{0x66, 0x09, 0xca})
		c.R[EDX], c.R[ECX] = v.a, v.b
		c.EFlags = IF | DF | CF | OF
		if err := c.Step(); err != nil {
			t.Fatal(err)
		}
		if c.R[EDX] != v.result || c.R[ECX] != v.b || c.EFlags != IF|DF|v.flags || c.EIP != 3 {
			t.Fatalf("OR16 registers=%X flags=%X", c.R, c.EFlags)
		}
	}
	for _, code := range [][]byte{{0x26, 0x66, 0x09, 0x02}, {0xf3, 0x66, 0x09, 0xca}} {
		c := New(testBus(code))
		c.R[EDX] = 0x12340001
		c.R[ECX] = 2
		c.EFlags = IF | CF | OF
		if c.Step() == nil {
			t.Fatal("unsupported form accepted")
		}
		if c.R[EDX] != 0x12340001 || c.R[ECX] != 2 || c.EFlags != IF|CF|OF {
			t.Fatal("rejected form published state")
		}
	}
}

func TestIntegrationSegmentStoreWithWordPrefixES(t *testing.T) {
	for _, writable := range []bool{false, true} {
		mem := testBus(make([]byte, 128))
		copy(mem, []byte{0x66, 0x26, 0x8c, 0x1d, 16, 0, 0, 0})
		mem[80], mem[81], mem[82] = 0xaa, 0xbb, 0xcc
		c := New(mem)
		c.Seg[SegES] = 0x28
		c.Seg[SegDS] = 0x160
		c.SetDescriptor(0x28, Descriptor{Base: 64, Limit: 63, Writable: writable})
		c.EFlags = IF | OF | CF
		err := c.Step()
		if writable {
			if err != nil || mem[80] != 0x60 || mem[81] != 1 || mem[82] != 0xcc {
				t.Fatal("ES word store width/base", err)
			}
		} else if err == nil || mem[80] != 0xaa || mem[81] != 0xbb {
			t.Fatal("readonly store published")
		}
		if c.EFlags != IF|OF|CF {
			t.Fatal("MOV changed flags")
		}
	}
}
