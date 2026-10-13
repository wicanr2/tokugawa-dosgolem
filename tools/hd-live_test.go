package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestHDLiveDrawReturnAndInvalidation(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(m[int(esp)+4*i:], v)
	}
	before := append([]byte(nil), m...)
	h := hdLive{LastName: "A-0.OZM", Validated: true}
	h.observe(1, 0x1c1bc, esp, m)
	if h.Active || h.Pending == nil {
		t.Fatal("draw entry must wait for return")
	}
	h.observe(2, 0x1803d, esp, m)
	if h.Active {
		t.Fatal("wrong stack activated replacement")
	}
	h.observe(3, 0x1803d, esp+4, m)
	if !h.Active || h.Pending != nil {
		t.Fatal("matching return did not activate")
	}
	if !bytes.Equal(m, before) {
		t.Fatal("guest memory changed")
	}
	h.observe(4, 0x1bf44, uint32(len(m)), m)
	if h.Active || h.Validated {
		t.Fatal("new asset load retained old HD qualification")
	}
}

func TestHDLiveRejectsOtherDraws(t *testing.T) {
	m := make([]byte, 0x60000)
	esp := uint32(0x1000)
	for i, v := range []uint32{0x1803d, 0, 40, 150, 80, 0xa0fb4} {
		binary.LittleEndian.PutUint32(m[int(esp)+4*i:], v)
	}
	for i := 0; i < 5; i++ {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			copyRAM := append([]byte(nil), m...)
			binary.LittleEndian.PutUint32(copyRAM[int(esp)+4+4*i:], 123)
			h := hdLive{LastName: "A-0.OZM", Validated: true, Active: true}
			h.observe(1, 0x1c1bc, esp, copyRAM)
			if h.Active || h.Pending != nil {
				t.Fatal("unknown geometry accepted")
			}
		})
	}
	h := hdLive{LastName: "A-1.OZM", Validated: true}
	h.observe(1, 0x1c1bc, esp, m)
	if h.Pending != nil {
		t.Fatal("different resource accepted")
	}
	h = hdLive{LastName: "A-0.OZM", Validated: true}
	h.observe(1, 0x1c1bc, ^uint32(0)-2, m)
	if h.Pending != nil {
		t.Fatal("out-of-bounds stack accepted")
	}
}
