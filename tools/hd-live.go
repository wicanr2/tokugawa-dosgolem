package main

import "encoding/binary"

type hdDrawReturn struct{ Return, Stack uint32 }

// 固定 EXE 的原版入口與返回觀測；不修改 guest 或取代指令。
type hdLive struct {
	Trace     hdTrace          `json:"loads"`
	LastName  string           `json:"last_resource"`
	Validated bool             `json:"source_validated"`
	Active    bool             `json:"draw_completed"`
	Pending   *hdDrawReturn    `json:"pending_draw"`
	Events    []map[string]any `json:"draw_events"`
}

func (h *hdLive) observe(step int, eip, esp uint32, mem []byte) {
	if h.Pending != nil && eip == h.Pending.Return && uint64(esp) == uint64(h.Pending.Stack)+4 {
		h.Active = h.Validated && h.LastName == "A-0.OZM"
		h.Pending = nil
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "draw_return", "step": step, "linear_eip": eip, "eligible": h.Active})
		}
	}
	if eip == 0x1bf44 {
		h.Active = false
		h.Validated = false
		h.LastName = ""
		h.Pending = nil
	}
	previous := len(h.Trace.Events)
	h.Trace.observe(step, eip, esp, mem)
	if len(h.Trace.Events) > previous {
		e := h.Trace.Events[len(h.Trace.Events)-1]
		if e["kind"] == "load_return" {
			h.LastName, _ = e["name"].(string)
			if h.LastName == "A-0.OZM" && e["raw_columns"] == uint32(40) && e["raw_rows"] == uint32(150) {
				p, ok := e["planes"].([]map[string]any)
				want := []string{"12708ee4fb617a539e53a49105266d9c3c78d5c6e0faaa2bde5ff439b56a5bf4", "8e7e24a520cf1af872b11f4836d50675025a7b0c241b8366f3093a7f2ecd6ebd", "4f5066cff4859e342b31d6975fea9614de47484f29ee5fb59c4cb21bd68416b4", "e0b4f317b56fb730e5cca0b8eed04f435b039a665cb1b69dc639ca79e339aa3f"}
				h.Validated = ok && len(p) == 4
				if h.Validated {
					for i := range p {
						if p[i]["sha256"] != want[i] {
							h.Validated = false
						}
					}
				}
			}
		}
	}
	if eip != 0x1c1bc {
		return
	}
	h.Active = false
	h.Pending = nil
	if !h.Validated || h.LastName != "A-0.OZM" || uint64(esp)+24 > uint64(len(mem)) {
		return
	}
	args := make([]uint32, 5)
	for i := range args {
		args[i] = binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):])
	}
	want := []uint32{0, 40, 150, 80, 0xa0fb4}
	for i := range args {
		if args[i] != want[i] {
			return
		}
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	if uint64(ret) >= uint64(len(mem)) {
		return
	}
	h.Pending = &hdDrawReturn{ret, esp}
	if len(h.Events) < 128 {
		h.Events = append(h.Events, map[string]any{"kind": "draw_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp, "raw_args": args, "logical_rect": []int{160, 50, 320, 150}})
	}
}
