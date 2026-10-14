package main

import "encoding/binary"

type hdDrawReturn struct{ Return, Stack uint32 }

// 固定 EXE 的原版入口與返回觀測；不修改 guest 或取代指令。
type hdLive struct {
	Window          hdWindowState    `json:"window"`
	Kao             hdKaoState       `json:"kao"`
	Trace           hdTrace          `json:"loads"`
	LastName        string           `json:"last_resource"`
	Validated       bool             `json:"source_validated"`
	Active          bool             `json:"draw_completed"`
	Pending         *hdDrawReturn    `json:"pending_draw"`
	Events          []map[string]any `json:"draw_events"`
	ScrollValidated bool             `json:"scroll_source_validated"`
	ScrollActive    bool             `json:"scroll_draw_completed"`
	ScrollPending   *hdDrawReturn    `json:"pending_scroll_draw"`
}

func (h *hdLive) observe(step int, eip, esp uint32, mem []byte) {
	h.Window.observe(step, eip, esp, mem)
	h.Kao.observe(step, eip, esp, mem)
	if h.ScrollPending != nil && eip == h.ScrollPending.Return && uint64(esp) == uint64(h.ScrollPending.Stack)+4 {
		h.ScrollActive = true
		h.ScrollPending = nil
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "scroll_draw_return", "step": step, "linear_eip": eip, "eligible": true})
		}
	}
	if h.Pending != nil && eip == h.Pending.Return && uint64(esp) == uint64(h.Pending.Stack)+4 {
		_, known := hdPortraitProfiles[h.LastName]
		h.Active = h.Validated && known
		h.Pending = nil
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "draw_return", "step": step, "linear_eip": eip, "eligible": h.Active})
		}
	}
	if eip == 0x1bf44 {
		h.ScrollValidated = false
		h.ScrollPending = nil
		h.Active = false
		h.Validated = false
		h.LastName = ""
		h.Pending = nil
	}
	previous := h.Trace.ReturnSequence
	h.Trace.observe(step, eip, esp, mem)
	if h.Trace.ReturnSequence != previous {
		e := h.Trace.LastReturn
		if e["kind"] == "load_return" {
			h.LastName, _ = e["name"].(string)
			if h.LastName == "VISUAL.OZM" && e["raw_columns"] == uint32(80) && e["raw_rows"] == uint32(400) {
				planes, ok := e["planes"].([]map[string]any)
				want := []string{"ea6c9f57424df636a60d144e1405fa2467429d14e87f2f854d47fe48b910acb5", "655790f2a54acccb050c4fbe0b29b3fb510976611807edbb1b4653656f4a29c6", "7fccd2bd1a06751fe2c043ac4693aa8cd088871bc742e12a861689fa360f4ab5", "b87ca90699e63248f74cac792f3f53b7576056af0c8427a8d0a6bcc1b1abd76a"}
				h.ScrollValidated = ok && len(planes) == 4
				if h.ScrollValidated {
					for i := range planes {
						if planes[i]["sha256"] != want[i] {
							h.ScrollValidated = false
						}
					}
				}
			}
			if profile, known := hdPortraitProfiles[h.LastName]; known && e["raw_columns"] == profile.Columns && e["raw_rows"] == uint32(150) {
				p, ok := e["planes"].([]map[string]any)
				want := profile.Planes
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
	if eip == 0x1c084 {
		h.ScrollActive = false
		h.ScrollPending = nil
		h.Active = false
		h.Pending = nil
		if h.ScrollValidated && h.LastName == "VISUAL.OZM" && uint64(esp)+4 <= uint64(len(mem)) {
			ret := binary.LittleEndian.Uint32(mem[esp:])
			if uint64(ret) < uint64(len(mem)) {
				h.ScrollPending = &hdDrawReturn{ret, esp}
				if len(h.Events) < 128 {
					h.Events = append(h.Events, map[string]any{"kind": "scroll_draw_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp, "logical_rect": []int{0, 250, 640, 150}})
				}
			}
		}
	}
	if eip != 0x1c1bc {
		return
	}
	h.Active = false
	h.Pending = nil
	_, known := hdPortraitProfiles[h.LastName]
	if !h.Validated || !known || uint64(esp)+24 > uint64(len(mem)) {
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
