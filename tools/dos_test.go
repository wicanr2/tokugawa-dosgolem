package main

import (
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
	"os"
	"path/filepath"
	"testing"
)

func TestEXECAbsentFile(t *testing.T) {
	dir := t.TempDir()
	files, err := machine.OpenDirectoryReadOnlyFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	m := &machine.LEMachine{Mem: make([]byte, 512)}
	copy(m.Mem[64:], ".\\COMMAND.COM\x00")
	c := cpu386.New(m)
	c.Seg[cpu386.SegDS] = 0x28
	c.SetDescriptor(0x28, cpu386.Descriptor{Limit: 511})
	c.R[cpu386.EAX] = 0xabcd4b00
	c.R[cpu386.EDX] = 64
	c.EFlags = cpu386.IF | cpu386.ZF
	if !executeAbsentFile(c, files) || c.R[cpu386.EAX] != 0xabcd0002 || c.EFlags != cpu386.IF|cpu386.ZF|cpu386.CF {
		t.Fatal("EXEC missing file status")
	}
	if err := os.WriteFile(filepath.Join(dir, "COMMAND.COM"), []byte{0xc3}, 0600); err != nil {
		t.Fatal(err)
	}
	c.R[cpu386.EAX] = 0x4b00
	if executeAbsentFile(c, files) || c.R[cpu386.EAX] != 0x4b00 {
		t.Fatal("existing executable faked success")
	}
	copy(m.Mem[64:], "..\\COMMAND.COM\x00")
	if executeAbsentFile(c, files) {
		t.Fatal("unsupported path accepted")
	}
}
