package machine

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu386"
)

func newMCBTest(t *testing.T) (*LEMachine, *DPMIHost) {
	t.Helper()
	m, h := newDPMITest(t)
	for i := range m.Mem {
		m.Mem[i] = 0xa5
	}
	if err := h.EnableDOSMCBMemory(0x80); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.Mem[:0x1000], bytes.Repeat([]byte{0xa5}, 0x1000)) {
		t.Fatal("MCB initialization clobbered image")
	}
	return m, h
}

func mcbAlloc(t *testing.T, h *DPMIHost, paras uint16) (uint16, uint16) {
	t.Helper()
	c := h.m.CPU
	c.R[cpu386.EBX] = 0xabcd0000 | uint32(paras)
	c.R[cpu386.EAX], c.R[cpu386.EDX] = 0x12340000, 0x56780000
	if !dpmiCall(h, c, 0x0100) || c.EFlags&cpu386.CF != 0 {
		t.Fatalf("allocate %d: AX=%x", paras, c.R[cpu386.EAX])
	}
	if c.R[cpu386.EAX]>>16 != 0x1234 || c.R[cpu386.EDX]>>16 != 0x5678 || c.R[cpu386.EBX]>>16 != 0xabcd {
		t.Fatal("high register bits changed")
	}
	return uint16(c.R[cpu386.EAX]), uint16(c.R[cpu386.EDX])
}

func mcbFree(t *testing.T, h *DPMIHost, sel uint16) {
	t.Helper()
	h.m.CPU.R[cpu386.EDX] = uint32(sel)
	if !dpmiCall(h, h.m.CPU, 0x0101) || h.m.CPU.EFlags&cpu386.CF != 0 {
		t.Fatal("free failed")
	}
}

func TestDOSMCBGeometryReuseAndCoalescing(t *testing.T) {
	m, h := newMCBTest(t)
	a, sa := mcbAlloc(t, h, 1)
	b, sb := mcbAlloc(t, h, 2)
	if a != 0x101 || b != 0x103 {
		t.Fatalf("segments %x %x; need one MCB per block", a, b)
	}
	head := m.Mem[0x1000:0x1010]
	if head[0] != 'M' || binary.LittleEndian.Uint16(head[1:]) != 0x80 || binary.LittleEndian.Uint16(head[3:]) != 1 || string(head[8:]) != "        " {
		t.Fatalf("header %x", head)
	}
	d := m.CPU.Descriptors[sb]
	if d.Base != 0x1030 || d.Limit != 31 || !d.Writable {
		t.Fatalf("descriptor %+v", d)
	}
	if !m.CPU.WriteSegmentBytes(sb, 31, []byte{0x5a}) || m.CPU.WriteSegmentBytes(sb, 32, []byte{0x5a}) {
		t.Fatal("descriptor boundary")
	}
	m.CPU.Seg[cpu386.SegDS] = sa
	mcbFree(t, h, sa)
	if m.CPU.Seg[cpu386.SegDS] != 0 {
		t.Fatal("freed selector remains loaded")
	}
	c, sc := mcbAlloc(t, h, 1)
	if c != a {
		t.Fatal("first-fit did not reuse freed block")
	}
	mcbFree(t, h, sb)
	mcbFree(t, h, sc)
	if len(h.dosMCB.blocks) != 1 || h.largestDOSMCB() != 0x9eff || m.Mem[0x1000] != 'Z' {
		t.Fatal("coalescing lost MCB paragraphs")
	}
	whole, sw := mcbAlloc(t, h, 0x9eff)
	if whole != a {
		t.Fatal("full arena start changed")
	}
	m.CPU.R[cpu386.EBX] = 1
	dpmiCall(h, m.CPU, 0x0100)
	if m.CPU.EFlags&cpu386.CF == 0 || uint16(m.CPU.R[cpu386.EAX]) != 8 || uint16(m.CPU.R[cpu386.EBX]) != 0 {
		t.Fatal("exhausted arena error")
	}
	mcbFree(t, h, sw)
}

func TestDOSMCBResizeFragmentationAndLargestBlock(t *testing.T) {
	m, h := newMCBTest(t)
	a, sa := mcbAlloc(t, h, 4)
	_, sb := mcbAlloc(t, h, 4)
	_, sc := mcbAlloc(t, h, 4)
	mcbFree(t, h, sb)
	m.CPU.R[cpu386.EDX], m.CPU.R[cpu386.EBX] = uint32(sa), 20
	dpmiCall(h, m.CPU, 0x0102)
	if m.CPU.EFlags&cpu386.CF == 0 || uint16(m.CPU.R[cpu386.EAX]) != 8 || uint16(m.CPU.R[cpu386.EBX]) != 9 || h.dosBlocks[sa].Paras != 9 || m.CPU.Descriptors[sa].Limit != 143 {
		t.Fatal("failed growth did not report and synchronize adjacent maximum")
	}
	m.CPU.R[cpu386.EBX] = 2
	dpmiCall(h, m.CPU, 0x0102)
	if m.CPU.EFlags&cpu386.CF != 0 || h.dosBlocks[sa].Base != uint32(a)*16 || m.CPU.Descriptors[sa].Limit != 31 {
		t.Fatal("shrink moved block or failed")
	}
	_, gap := mcbAlloc(t, h, 6)
	mcbFree(t, h, gap)
	mcbFree(t, h, sc)
	m.CPU.R[cpu386.EBX] = 0xffff
	dpmiCall(h, m.CPU, 0x0100)
	available := uint16(m.CPU.R[cpu386.EBX])
	if available != 0x9efc {
		t.Fatalf("largest allocatable data block = %x", available)
	}
	mcbAlloc(t, h, available)
}

func TestDOSMCBErrorsAndSelectorExhaustionAreNonDestructive(t *testing.T) {
	m, h := newMCBTest(t)
	before := append([]byte(nil), m.Mem...)
	for sel := uint32(0x100); sel <= 0xfff8; sel += 8 {
		m.CPU.SetDescriptor(uint16(sel), cpu386.Descriptor{})
	}
	m.CPU.R[cpu386.EBX] = 1
	dpmiCall(h, m.CPU, 0x0100)
	if uint16(m.CPU.R[cpu386.EAX]) != 0x8011 || !bytes.Equal(before, m.Mem) || len(h.dosBlocks) != 0 {
		t.Fatal("descriptor exhaustion mutated arena")
	}
	delete(m.CPU.Descriptors, 0x100)
	a, sa := mcbAlloc(t, h, 1)
	m.Mem[int(a-1)*16] = 0
	corrupt := append([]byte(nil), m.Mem...)
	m.CPU.R[cpu386.EDX] = uint32(sa)
	dpmiCall(h, m.CPU, 0x0101)
	if uint16(m.CPU.R[cpu386.EAX]) != 7 || !bytes.Equal(corrupt, m.Mem) || len(h.dosBlocks) != 1 {
		t.Fatal("damaged chain not rejected intact")
	}
	if err := h.EnableDOSMCBMemory(0x80); err == nil {
		t.Fatal("mode switch accepted after use")
	}
}

func TestDOSMCBStackAndLinearAllocationsStaySeparate(t *testing.T) {
	m, h := newMCBTest(t)
	stack, ok := h.allocDOS(256)
	if !ok {
		t.Fatal("stack allocation")
	}
	a, sa := mcbAlloc(t, h, 0x1001)
	if uint32(a)*16 != stack+4096+16 || m.CPU.Descriptors[sa].Limit != 0x1000f {
		t.Fatal("stack overlap or 32-bit descriptor truncated")
	}
	m.CPU.R[cpu386.EBX], m.CPU.R[cpu386.ECX] = 0, 4096
	dpmiCall(h, m.CPU, 0x0501)
	base := uint32(uint16(m.CPU.R[cpu386.EBX]))<<16 | uint32(uint16(m.CPU.R[cpu386.ECX]))
	if base < 0x100000 {
		t.Fatal("linear allocation entered DOS pool")
	}
	m.CPU.R[cpu386.EDX] = 0xbeef
	dpmiCall(h, m.CPU, 0x0101)
	if uint16(m.CPU.R[cpu386.EAX]) != 0x8022 {
		t.Fatal("unknown selector")
	}
	m.CPU.R[cpu386.EBX] = 0
	dpmiCall(h, m.CPU, 0x0100)
	if uint16(m.CPU.R[cpu386.EAX]) != 0x8021 {
		t.Fatal("zero allocation")
	}
}
