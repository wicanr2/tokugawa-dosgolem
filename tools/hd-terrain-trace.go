package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type terrainTracePending struct {
	Return, Stack uint32
	Source, Dest  uint32
	BeginStep     int
}

// 原始暫存器與 backing RAM 的有界診斷。未參與正式呈現。
type terrainTrace struct {
	Events  []map[string]any     `json:"events"`
	Dropped uint64               `json:"dropped_events"`
	Pending *terrainTracePending `json:"pending_return"`
}

func (t *terrainTrace) event(e map[string]any) {
	if len(t.Events) >= 512 {
		t.Dropped++
		return
	}
	t.Events = append(t.Events, e)
}

func terrainPlaneHashes(mem []byte) []map[string]any {
	if len(mem) < 0x42178 {
		return nil
	}
	var result []map[string]any
	for _, global := range []int{0x42174, 0x42168, 0x42170, 0x4216c} {
		base := binary.LittleEndian.Uint32(mem[global:])
		p := map[string]any{"pointer_global_linear": global, "base_linear": base}
		if base < 0xc0000 || uint64(base)+32000 > uint64(len(mem)) {
			p["observation_error"] = "plane outside ordinary backing RAM"
		} else {
			p["sha256"] = fmt.Sprintf("%x", sha256.Sum256(mem[base:uint64(base)+32000]))
		}
		result = append(result, p)
	}
	return result
}

func (t *terrainTrace) observe(step int, eip, esp, esi, edi uint32, mem []byte) {
	if p := t.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		t.event(map[string]any{"kind": "floor_return", "step": step, "begin_step": p.BeginStep,
			"linear_eip": eip, "raw_source_offset": p.Source, "raw_destination_offset": p.Dest,
			"planes": terrainPlaneHashes(mem)})
		t.Pending = nil
	}
	if eip != 0x1fc07 {
		return
	}
	t.Pending = nil
	e := map[string]any{"kind": "floor_enter", "step": step, "linear_eip": eip,
		"stack_linear": esp, "raw_source_offset": esi, "raw_destination_offset": edi,
		"planes": terrainPlaneHashes(mem)}
	if uint64(esp)+4 > uint64(len(mem)) {
		e["observation_error"] = "stack outside backing RAM"
	} else {
		ret := binary.LittleEndian.Uint32(mem[esp:])
		e["return_linear"] = ret
		if uint64(ret) >= uint64(len(mem)) {
			e["observation_error"] = "return outside backing RAM"
		} else {
			t.Pending = &terrainTracePending{ret, esp, esi, edi, step}
		}
	}
	if len(mem) >= 0x42180 {
		e["raw_35984"] = binary.LittleEndian.Uint32(mem[0x35984:])
		e["raw_4217c"] = binary.LittleEndian.Uint32(mem[0x4217c:])
	}
	t.event(e)
}
