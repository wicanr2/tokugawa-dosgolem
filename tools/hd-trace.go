package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type hdPending struct {
	Return, Stack uint32
	Name          string
	BeginStep     int
}

type hdTrace struct {
	Events         []map[string]any `json:"events"`
	Dropped        uint64           `json:"dropped_events"`
	Pending        []hdPending      `json:"pending_returns"`
	LastReturn     map[string]any   `json:"-"`
	ReturnSequence uint64           `json:"-"`
}

func (h *hdTrace) event(e map[string]any) {
	if len(h.Events) == 256 {
		h.Dropped++
		return
	}
	h.Events = append(h.Events, e)
}

// 只讀零基底 LE 的 backing RAM，避免診斷讀取觸發 VGA latch 或 CPU 行為。
func (h *hdTrace) observe(step int, eip, esp uint32, mem []byte) {
	if len(h.Pending) > 0 {
		p := h.Pending[len(h.Pending)-1]
		if eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
			h.Pending = h.Pending[:len(h.Pending)-1]
			e := map[string]any{"kind": "load_return", "step": step, "linear_eip": eip, "name": p.Name, "begin_step": p.BeginStep}
			if len(mem) < 0x5b1bc {
				e["observation_error"] = "dimension globals outside RAM"
			} else {
				w, height := binary.LittleEndian.Uint32(mem[0x5b1b8:]), binary.LittleEndian.Uint32(mem[0x5b1b4:])
				e["raw_columns"], e["raw_rows"], e["pitch_bytes"] = w, height, 80
				planes := []map[string]any{}
				for _, at := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
					base := binary.LittleEndian.Uint32(mem[at:])
					plane := map[string]any{"pointer_global_linear": at, "base_linear": base}
					end := uint64(base) + uint64(height-1)*80 + uint64(w)
					if w < 1 || w > 80 || height < 1 || height > 400 || end > uint64(len(mem)) || (base < 0xc0000 && end > 0xa0000) {
						plane["observation_error"] = "unsupported dimensions or RAM bounds"
					} else {
						sum := sha256.New()
						for y := uint32(0); y < height; y++ {
							offset := uint64(base) + uint64(y)*80
							sum.Write(mem[offset : offset+uint64(w)])
						}
						plane["sha256"] = fmt.Sprintf("%x", sum.Sum(nil))
					}
					planes = append(planes, plane)
				}
				e["planes"] = planes
			}
			h.LastReturn = e
			h.ReturnSequence++
			h.event(e)
		}
	}
	if eip != 0x1bf44 {
		return
	}
	e := map[string]any{"kind": "load_enter", "step": step, "linear_eip": eip, "stack_linear": esp}
	if uint64(esp)+8 > uint64(len(mem)) {
		e["observation_error"] = "stack outside RAM"
		h.event(e)
		return
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	ptr := binary.LittleEndian.Uint32(mem[esp+4:])
	e["name_pointer_linear"], e["return_linear"] = ptr, ret
	name := make([]byte, 0, 64)
	valid := false
	for i := uint64(0); i < 64 && uint64(ptr)+i < uint64(len(mem)); i++ {
		b := mem[uint64(ptr)+i]
		if b == 0 {
			valid = len(name) > 0
			break
		}
		if b < 32 || b > 126 {
			break
		}
		name = append(name, b)
	}
	if !valid || len(h.Pending) == 16 || uint64(ret) >= uint64(len(mem)) {
		e["observation_error"] = "invalid name, return address or pending depth"
	} else {
		e["name"] = string(name)
		h.Pending = append(h.Pending, hdPending{ret, esp, string(name), step})
	}
	h.event(e)
}
