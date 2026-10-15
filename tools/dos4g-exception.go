package main

import (
	"encoding/binary"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

type dos4gExceptionTrace struct {
	Events  []map[string]any `json:"events"`
	Dropped int              `json:"dropped"`
}

// DOS/4GW normalizes CPU exception frames for handlers installed by DOS AH=25h.
// The guest handler executes normally and returns using IRET.
func deliverDOS4GGeneralProtection(c *cpu386.CPU, selector uint16, target, faultEIP uint32) bool {
	if c.LimitFault == nil {
		return false
	}
	d, ok := c.Descriptors[selector]
	old, oldOK := c.Descriptors[c.Seg[cpu386.SegCS]]
	if !ok || !oldOK || d.Base != 0 || old.Base != 0 || target > d.Limit || faultEIP > old.Limit || c.R[cpu386.ESP] < 12 {
		return false
	}
	var frame [12]byte
	binary.LittleEndian.PutUint32(frame[:], faultEIP)
	binary.LittleEndian.PutUint32(frame[4:], uint32(c.Seg[cpu386.SegCS]))
	binary.LittleEndian.PutUint32(frame[8:], c.EFlags)
	sp := c.R[cpu386.ESP] - 12
	if !c.WriteSegmentBytes(c.Seg[cpu386.SegSS], sp, frame[:]) {
		return false
	}
	c.R[cpu386.ESP], c.Seg[cpu386.SegCS], c.EIP = sp, selector, target
	c.EFlags &^= cpu386.IF | (1 << 8)
	return true
}

func (t *dos4gExceptionTrace) deliver(c *cpu386.CPU, services *machine.FD2StartupDOS, step int, faultEIP uint32) bool {
	if c.LimitFault == nil {
		return false
	}
	fault := *c.LimitFault
	selector, target, ok := services.ProtectedInterruptVector(13)
	if !ok || !deliverDOS4GGeneralProtection(c, selector, target, faultEIP) {
		return false
	}
	if len(t.Events) < 256 {
		t.Events = append(t.Events, map[string]any{"step": step, "interrupt": 13, "fault_eip": faultEIP,
			"memory_fault": fault, "handler_selector": selector, "handler_eip": target, "stack_pointer": c.R[cpu386.ESP],
			"classification": "DOS4GW host ABI; original handler executes; no instruction skip"})
	} else {
		t.Dropped++
	}
	return true
}
