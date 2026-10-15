package main

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu386"
)

type exceptionTestBus []byte

func (b exceptionTestBus) Read8(a uint32) (byte, error) {
	if uint64(a) >= uint64(len(b)) {
		return 0, fmt.Errorf("read outside test memory")
	}
	return b[a], nil
}
func (b exceptionTestBus) Write8(a uint32, v byte) error {
	if uint64(a) >= uint64(len(b)) {
		return fmt.Errorf("write outside test memory")
	}
	b[a] = v
	return nil
}

func exceptionTestCPU() (*cpu386.CPU, exceptionTestBus) {
	mem := exceptionTestBus(make([]byte, 512))
	c := cpu386.New(mem)
	c.Seg[cpu386.SegCS], c.Seg[cpu386.SegDS], c.Seg[cpu386.SegES], c.Seg[cpu386.SegSS] = 8, 16, 24, 32
	c.SetDescriptor(8, cpu386.Descriptor{Limit: 511})
	c.SetDescriptor(16, cpu386.Descriptor{Limit: 511, Writable: true})
	c.SetDescriptor(24, cpu386.Descriptor{Base: 256, Limit: 7, Writable: true})
	c.SetDescriptor(32, cpu386.Descriptor{Limit: 511, Writable: true})
	c.R[cpu386.ESP] = 480
	return c, mem
}

func TestDOS4GPartialREPExecutesHandlerAndResumes(t *testing.T) {
	c, mem := exceptionTestCPU()
	copy(mem, []byte{0xf3, 0xa5})
	// Synthetic handler: MOV EDI,0; IRET. The original handler is run separately.
	copy(mem[32:], []byte{0xbf, 0, 0, 0, 0, 0xcf})
	copy(mem[128:], []byte("abcdefghijkl"))
	c.R[cpu386.ESI], c.R[cpu386.ECX], c.EFlags = 128, 3, cpu386.IF|cpu386.CF|2
	err := c.Step()
	if err == nil || c.LimitFault == nil || c.LimitFault.Offset != 8 || c.LimitFault.Size != 4 || !c.LimitFault.Write || c.R[cpu386.ECX] != 1 || c.R[cpu386.ESI] != 136 || c.R[cpu386.EDI] != 8 || string(mem[256:264]) != "abcdefgh" {
		t.Fatalf("partial REP did not retain progress: %v %+v", err, c.LimitFault)
	}
	if !deliverDOS4GGeneralProtection(c, 8, 32, 0) {
		t.Fatal("exception not delivered")
	}
	if c.R[cpu386.ESP] != 468 || c.EIP != 32 || c.EFlags&cpu386.IF != 0 || binary.LittleEndian.Uint32(mem[468:]) != 0 || binary.LittleEndian.Uint32(mem[472:]) != 8 || binary.LittleEndian.Uint32(mem[476:]) != cpu386.IF|cpu386.CF|2 {
		t.Fatal("incorrect normalized IRET frame")
	}
	if err = c.Step(); err != nil || c.LimitFault != nil || c.R[cpu386.EDI] != 0 {
		t.Fatalf("handler did not execute: %v", err)
	}
	if err = c.Step(); err != nil || c.EIP != 0 || c.R[cpu386.ESP] != 480 || c.EFlags != cpu386.IF|cpu386.CF|2 {
		t.Fatalf("IRET did not restore frame: %v", err)
	}
	if err = c.Step(); err != nil || c.R[cpu386.ECX] != 0 || c.R[cpu386.ESI] != 140 || string(mem[256:264]) != "ijklefgh" {
		t.Fatalf("REP restarted instead of resuming: %v", err)
	}
}

func TestDOS4GRejectsInvalidFramesWithoutPublishingState(t *testing.T) {
	for _, reason := range []string{"no_fault", "no_handler", "target_limit", "nonflat_handler", "short_stack", "readonly_stack", "return_limit"} {
		c, mem := exceptionTestCPU()
		c.LimitFault = &cpu386.LimitFault{Selector: 24, Offset: 8, Size: 4, Write: true}
		selector, target, faultEIP := uint16(8), uint32(32), uint32(0)
		switch reason {
		case "no_fault":
			c.LimitFault = nil
		case "no_handler":
			selector = 0
		case "target_limit":
			target = 512
		case "nonflat_handler":
			c.SetDescriptor(8, cpu386.Descriptor{Base: 1, Limit: 511})
		case "short_stack":
			c.R[cpu386.ESP] = 8
		case "readonly_stack":
			c.SetDescriptor(32, cpu386.Descriptor{Limit: 511})
		case "return_limit":
			faultEIP = 512
		}
		before := append([]byte(nil), mem...)
		regs, flags, eip, segments := c.R, c.EFlags, c.EIP, c.Seg
		if deliverDOS4GGeneralProtection(c, selector, target, faultEIP) || c.R != regs || c.EFlags != flags || c.EIP != eip || c.Seg != segments || string(mem) != string(before) {
			t.Fatalf("invalid %s published a frame", reason)
		}
	}
	for _, reason := range []string{"short_stack", "unknown_cs", "target_limit", "nonflat_cs", "virtual_8086", "nested_task"} {
		c, mem := exceptionTestCPU()
		mem[0] = 0xcf
		binary.LittleEndian.PutUint32(mem[480:], 32)
		binary.LittleEndian.PutUint32(mem[484:], 8)
		binary.LittleEndian.PutUint32(mem[488:], cpu386.IF|2)
		switch reason {
		case "short_stack":
			c.R[cpu386.ESP] = 508
		case "unknown_cs":
			binary.LittleEndian.PutUint32(mem[484:], 0x888)
		case "target_limit":
			binary.LittleEndian.PutUint32(mem[480:], 512)
		case "nonflat_cs":
			c.SetDescriptor(8, cpu386.Descriptor{Base: 1, Limit: 511})
		case "virtual_8086":
			binary.LittleEndian.PutUint32(mem[488:], 1<<17)
		case "nested_task":
			binary.LittleEndian.PutUint32(mem[488:], 1<<14)
		}
		before := append([]byte(nil), mem...)
		regs, flags, segments := c.R, c.EFlags, c.Seg
		if c.Step() == nil || c.R != regs || c.EFlags != flags || c.Seg != segments || string(mem) != string(before) {
			t.Fatalf("invalid IRET %s published state", reason)
		}
	}
}

func TestDOS4GCodeSegmentCMPAndFaultClassification(t *testing.T) {
	c, mem := exceptionTestCPU()
	copy(mem, []byte{0x66, 0x2e, 0x3b, 0x05, 0x80, 0, 0, 0})
	binary.LittleEndian.PutUint16(mem[128:], 0x1234)
	binary.LittleEndian.PutUint16(mem[192:], 0x5678)
	c.SetDescriptor(16, cpu386.Descriptor{Base: 64, Limit: 255, Writable: true})
	c.R[cpu386.EAX], c.EFlags = 0xabcd1234, cpu386.IF|cpu386.CF|cpu386.OF
	regs := c.R
	before := append([]byte(nil), mem...)
	if err := c.Step(); err != nil || c.R != regs || c.EFlags != cpu386.IF|cpu386.ZF|cpu386.PF || string(mem) != string(before) {
		t.Fatalf("CS CMP ignored the code selector or changed input: %v", err)
	}
	// A prior limit failure cannot turn an unknown opcode into a new exception.
	c.LimitFault = &cpu386.LimitFault{Selector: 24, Offset: 8, Size: 4}
	mem[c.EIP] = 0xf1
	if c.Step() == nil || c.LimitFault != nil {
		t.Fatal("stale fault escaped Step reset")
	}
}
