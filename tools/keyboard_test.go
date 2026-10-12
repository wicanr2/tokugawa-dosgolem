package main

import (
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"testing"
)

func TestKeyboardSharedQueue(t *testing.T) {
	real := machine.New()
	m := &machine.LEMachine{Mem: append([]byte(nil), real.Mem...)}
	if !machine.InstallLEBIOSKeyboard(m) {
		t.Fatal("keyboard install")
	}
	c := cpu386.New(m)
	c.R[cpu386.EAX] = 0xabcd0b55
	c.EFlags = cpu386.IF | cpu386.CF
	if !keyboardInterrupt(c, m, 0x21) || c.R[cpu386.EAX] != 0xabcd0b00 || c.EFlags != cpu386.IF {
		t.Fatal("empty DOS status")
	}
	if err := m.Keyboard.Enqueue(0x1c0d); err != nil {
		t.Fatal(err)
	}
	c.R[cpu386.EAX] = 0xabcd0b00
	keyboardInterrupt(c, m, 0x21)
	if c.R[cpu386.EAX] != 0xabcd0bff || m.Keyboard.Reads != 0 {
		t.Fatal("DOS status consumed key")
	}
	c.R[cpu386.EAX] = 0xabcd0100
	c.EFlags |= cpu386.ZF
	if !keyboardInterrupt(c, m, 0x16) || c.R[cpu386.EAX] != 0xabcd1c0d || c.EFlags&cpu386.ZF != 0 || m.Keyboard.Reads != 0 {
		t.Fatal("BIOS peek")
	}
	c.R[cpu386.EAX] = 0xabcd1000
	if !keyboardInterrupt(c, m, 0x16) || c.R[cpu386.EAX] != 0xabcd1c0d || m.Keyboard.Reads != 1 {
		t.Fatal("BIOS read")
	}
	c.R[cpu386.EAX] = 0x1000
	if keyboardInterrupt(c, m, 0x16) || !m.Keyboard.Waiting {
		t.Fatal("empty blocking read faked success")
	}
	c.R[cpu386.EAX] = 0x9900
	if keyboardInterrupt(c, m, 0x16) {
		t.Fatal("unknown keyboard service")
	}
}
