package main

import (
	"encoding/binary"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

func peekKey(m *machine.LEMachine) (uint16, bool, bool) {
	if m.Keyboard == nil || len(m.Mem) < 0x43e {
		return 0, false, false
	}
	h := binary.LittleEndian.Uint16(m.Mem[0x41a:])
	t := binary.LittleEndian.Uint16(m.Mem[0x41c:])
	if h < 0x1e || h >= 0x3e || t < 0x1e || t >= 0x3e || h&1 != 0 || t&1 != 0 {
		return 0, false, false
	}
	if h == t {
		return 0, false, true
	}
	value := binary.LittleEndian.Uint16(m.Mem[0x400+int(h):])
	if value&255 == 0xf0 && value>>8 != 0 {
		value &= 0xff00
	}
	return value, true, true
}

func keyboardInterrupt(c *cpu386.CPU, m *machine.LEMachine, n byte) bool {
	fn := byte(c.R[cpu386.EAX] >> 8)
	if n == 0x21 && fn == 0x0b {
		_, available, ok := peekKey(m)
		if !ok {
			return false
		}
		c.R[cpu386.EAX] &^= 0xff
		if available {
			c.R[cpu386.EAX] |= 0xff
		}
		c.EFlags &^= cpu386.CF
		return true
	}
	if n != 0x16 {
		return false
	}
	switch fn {
	case 0, 0x10:
		if m.Keyboard == nil {
			return false
		}
		value, available, err := m.Keyboard.ReadEnhanced()
		if err != nil || !available {
			return false
		}
		lowWord(c, cpu386.EAX, value)
		return true
	case 1, 0x11:
		value, available, ok := peekKey(m)
		if !ok {
			return false
		}
		if available {
			lowWord(c, cpu386.EAX, value)
			c.EFlags &^= cpu386.ZF
		} else {
			c.EFlags |= cpu386.ZF
		}
		return true
	case 2:
		if len(m.Mem) <= 0x417 {
			return false
		}
		c.R[cpu386.EAX] = c.R[cpu386.EAX]&^0xff | uint32(m.Mem[0x417])
		return true
	}
	return false
}
