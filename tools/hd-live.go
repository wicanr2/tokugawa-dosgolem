package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

type hdWindowLayersRegion struct {
	Rect     image.Rectangle
	Indexed  []byte
	Graphics []graphicsDrawRecord
	Kao      []hdKaoDraw
}

type hdWindowLayersPending struct {
	Return, Stack, Level uint32
	Region               *hdWindowLayersRegion
}

type hdWindowLayersState struct {
	Saved          map[uint32]*hdWindowLayersRegion `json:"-"`
	PendingSave    *hdWindowLayersPending           `json:"-"`
	PendingRestore *hdWindowLayersPending           `json:"-"`
	Events         []map[string]any                 `json:"events"`
}

func windowLayersRect(offset, cols, rows uint32) (image.Rectangle, bool) {
	if offset >= 32000 || cols < 1 || cols > 40 || rows < 1 || rows > 25 {
		return image.Rectangle{}, false
	}
	x, y := int(offset%80)*8, int(offset/80)
	rect := image.Rect(x, y, x+int(cols)*16, y+int(rows)*16)
	return rect, rect.In(image.Rect(0, 0, 640, 400))
}

func windowLayersSlot(mem []byte, level uint32) (image.Rectangle, []byte, bool) {
	base := uint64(0x34ed3) + uint64(level)*16
	if level >= 64 || base+16 > uint64(len(mem)) {
		return image.Rectangle{}, nil, false
	}
	offset := binary.LittleEndian.Uint32(mem[base:])
	cols, rows := binary.LittleEndian.Uint32(mem[base+4:]), binary.LittleEndian.Uint32(mem[base+8:])
	rect, ok := windowLayersRect(offset, cols, rows)
	if !ok {
		return image.Rectangle{}, nil, false
	}
	pointer := uint64(binary.LittleEndian.Uint32(mem[base+12:]))
	planeBytes := uint64(rect.Dx() / 8 * rect.Dy())
	if pointer == 0 || pointer+planeBytes*4 > uint64(len(mem)) {
		return rect, nil, false
	}
	indexed := make([]byte, rect.Dx()*rect.Dy())
	for i := range indexed {
		for bit := uint64(0); bit < 4; bit++ {
			indexed[i] |= (mem[pointer+bit*planeBytes+uint64(i/8)] >> uint(7-i%8) & 1) << bit
		}
	}
	return rect, indexed, true
}

func (h *hdLive) observeWindowLayers(step int, eip, esp uint32, mem []byte, snapshot func() (nativeSnapshot, error)) {
	w := &h.WindowLayers
	if eip != 0x1afb0 && eip != 0x1b165 && eip != 0x1b245 &&
		(w.PendingSave == nil || eip != w.PendingSave.Return || uint64(esp) != uint64(w.PendingSave.Stack)+4) &&
		(w.PendingRestore == nil || eip != w.PendingRestore.Return || uint64(esp) != uint64(w.PendingRestore.Stack)+4) {
		return
	}
	event := func(kind string, p *hdWindowLayersPending, verified bool, indexed []byte) {
		if len(w.Events) < 256 {
			w.Events = append(w.Events, map[string]any{"kind": kind, "step": step, "linear_eip": eip,
				"return_linear": p.Return, "stack_linear": p.Stack, "level": p.Level, "logical_rect": p.Region.Rect,
				"verified": verified, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(indexed))})
		}
	}
	returned := func(p *hdWindowLayersPending) bool {
		return p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4
	}
	if p := w.PendingSave; returned(p) {
		rect, indexed, ok := windowLayersSlot(mem, p.Level)
		ok = ok && hdWindowSourceValid(mem) && binary.LittleEndian.Uint32(mem[0x35157:]) == p.Level+1 && rect == p.Region.Rect && bytes.Equal(indexed, p.Region.Indexed)
		if ok {
			if w.Saved == nil {
				w.Saved = map[uint32]*hdWindowLayersRegion{}
			}
			w.Saved[p.Level] = p.Region
		}
		event("save_return", p, ok, indexed)
		w.PendingSave = nil
	}
	if p := w.PendingRestore; returned(p) {
		var indexed []byte
		ok := hdWindowSourceValid(mem) && binary.LittleEndian.Uint32(mem[0x35157:]) == p.Level && snapshot != nil
		if ok {
			native, err := snapshot()
			indexed = graphicsCrop(native.Indexed, 640, p.Region.Rect)
			ok = err == nil && native.MaxColor == 15 && len(native.Indexed) == 640*400 && bytes.Equal(indexed, p.Region.Indexed)
		}
		if ok {
			h.Graphics.Draws = graphicsRestoreRegion(h.Graphics.Draws, p.Region.Graphics)
			h.Kao.Draws = append(h.Kao.Draws, p.Region.Kao...)
			if len(h.Kao.Draws) > 64 {
				h.Kao.Draws = nil
			}
		}
		event("restore_return", p, ok, indexed)
		w.PendingRestore = nil
	}
	if eip == 0x1afb0 {
		w.Saved, w.PendingSave, w.PendingRestore = nil, nil, nil
	}
	if eip != 0x1b165 && eip != 0x1b245 {
		return
	}
	if !hdWindowSourceValid(mem) || uint64(esp)+4 > uint64(len(mem)) {
		w.Saved, w.PendingSave, w.PendingRestore = nil, nil, nil
		h.Graphics.Draws, h.Kao.Draws = nil, nil
		return
	}
	ret, level := binary.LittleEndian.Uint32(mem[esp:]), binary.LittleEndian.Uint32(mem[0x35157:])
	if uint64(ret) >= uint64(len(mem)) || level > 64 {
		w.Saved, w.PendingSave, w.PendingRestore = nil, nil, nil
		h.Graphics.Draws, h.Kao.Draws = nil, nil
		return
	}
	if eip == 0x1b165 {
		w.PendingSave = nil
		delete(w.Saved, level)
		if level >= 64 || uint64(esp)+20 > uint64(len(mem)) || snapshot == nil {
			return
		}
		x, y := binary.LittleEndian.Uint32(mem[esp+4:]), binary.LittleEndian.Uint32(mem[esp+8:])
		cols, rows := binary.LittleEndian.Uint32(mem[esp+12:]), binary.LittleEndian.Uint32(mem[esp+16:])
		if x >= 80 || y >= 400 {
			return
		}
		rect, ok := windowLayersRect(x+y*80, cols, rows)
		if !ok {
			return
		}
		native, err := snapshot()
		indexed := graphicsCrop(native.Indexed, 640, rect)
		if err != nil || native.MaxColor != 15 || len(native.Indexed) != 640*400 || indexed == nil {
			return
		}
		region := &hdWindowLayersRegion{Rect: rect, Indexed: indexed, Graphics: graphicsClipRegion(h.Graphics.Draws, rect, true)}
		for _, draw := range h.Kao.Draws {
			if draw.Rect.In(rect) {
				region.Kao = append(region.Kao, draw)
			}
		}
		w.PendingSave = &hdWindowLayersPending{Return: ret, Stack: esp, Level: level, Region: region}
		event("save_enter", w.PendingSave, false, indexed)
		return
	}
	w.PendingRestore = nil
	if level == 0 {
		w.Saved = nil
		h.Graphics.Draws, h.Kao.Draws = nil, nil
		return
	}
	level--
	rect, indexed, valid := windowLayersSlot(mem, level)
	region, known := w.Saved[level]
	delete(w.Saved, level)
	if rect.Empty() {
		h.Graphics.Draws, h.Kao.Draws = nil, nil
		return
	}
	h.Graphics.Draws = graphicsClipRegion(h.Graphics.Draws, rect, false)
	h.Graphics.Pending, h.Graphics.Composite, h.Kao.Pending = nil, nil, nil
	kao := []hdKaoDraw{}
	for _, draw := range h.Kao.Draws {
		if !draw.Rect.Overlaps(rect) {
			kao = append(kao, draw)
		}
	}
	h.Kao.Draws = kao
	if valid && known && region.Rect == rect && bytes.Equal(indexed, region.Indexed) {
		w.PendingRestore = &hdWindowLayersPending{Return: ret, Stack: esp, Level: level, Region: region}
		event("restore_enter", w.PendingRestore, false, indexed)
	}
}

type hdDrawReturn struct {
	Return, Stack, Entry uint32
	Destination          image.Rectangle `json:"destination,omitempty"`
	SourceName           string          `json:"source_name,omitempty"`
	SourceGuard          string          `json:"source_guard_sha256,omitempty"`
	SourceBases          [4]uint32       `json:"source_bases,omitempty"`
}

var hdScrollSourcePlanes = []string{"ea6c9f57424df636a60d144e1405fa2467429d14e87f2f854d47fe48b910acb5", "655790f2a54acccb050c4fbe0b29b3fb510976611807edbb1b4653656f4a29c6", "7fccd2bd1a06751fe2c043ac4693aa8cd088871bc742e12a861689fa360f4ab5", "b87ca90699e63248f74cac792f3f53b7576056af0c8427a8d0a6bcc1b1abd76a"}

var endingStoryRect = image.Rect(16, 16, 336, 166)

func endingStoryCaller(ret uint32) (string, uint32) {
	switch ret {
	case 0x18a07:
		return "END-4.OZM", 0x1c1bc
	case 0x18a6e:
		return "END-5.OZM", 0x1c1bc
	case 0x18ad5:
		return "END-6.OZM", 0x1c1bc
	case 0x18b3c:
		return "END-7.OZM", 0x1c1bc
	case 0x18ba3:
		return "END-8.OZM", 0x1c1bc
	case 0x18c19:
		return "END-9.OZM", 0x1c10b
	case 0x18c8f:
		return "END-10.OZM", 0x1c10b
	}
	return "", 0
}

func endingStoryCall(eip, esp uint32, mem []byte, name string) bool {
	if eip != 0x1c1bc && eip != 0x1c10b {
		return false
	}
	length := uint64(24)
	if eip == 0x1c10b {
		length = 36
	}
	if uint64(esp)+length > uint64(len(mem)) {
		return false
	}
	want, entry := endingStoryCaller(binary.LittleEndian.Uint32(mem[esp:]))
	if want == "" || want != name || entry != eip {
		return false
	}
	for i, value := range []uint32{0, 40, 150, 80, 0xa0502} {
		if binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):]) != value {
			return false
		}
	}
	if eip == 0x1c10b {
		for i := 0; i < 4; i++ {
			if binary.LittleEndian.Uint32(mem[uint64(esp)+20+uint64(i*4):]) != 0xa0502 {
				return false
			}
		}
	}
	return true
}

func endingStoryCurrent(mem []byte, name string) ([4]uint32, string, bool) {
	var bases [4]uint32
	profile, known := hdPortraitProfiles[name]
	if !known || profile.Columns != 40 || len(mem) < 0x5b1bc {
		return bases, "", false
	}
	_, indices, ok := graphicsSource(mem, []graphicsSourceProfile{{Name: name, Width: 320, Height: 150, Planes: profile.Planes[:]}})
	if !ok {
		return bases, "", false
	}
	for i, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		bases[i] = binary.LittleEndian.Uint32(mem[global:])
		if bases[i] < 0xc0000 {
			return bases, "", false
		}
	}
	return bases, fmt.Sprintf("%x", sha256.Sum256(indices)), true
}

func endingStoryReturned(p *hdDrawReturn, name string, mem []byte, snapshot func() (nativeSnapshot, error)) bool {
	want, entry := endingStoryCaller(p.Return)
	if p.Destination != endingStoryRect || want != name || p.SourceName != name || p.Entry != entry || snapshot == nil {
		return false
	}
	bases, digest, ok := endingStoryCurrent(mem, name)
	if !ok || bases != p.SourceBases || digest != p.SourceGuard {
		return false
	}
	s, err := snapshot()
	if err != nil {
		return false
	}
	actual := graphicsCrop(s.Indexed, 640, endingStoryRect)
	return len(actual) == 48000 && fmt.Sprintf("%x", sha256.Sum256(actual)) == digest
}

func (h *hdLive) storyDestination() image.Rectangle {
	if h.StoryRect.Empty() {
		return a0Rect
	}
	return h.StoryRect
}

func openingScrollMatches(mem []byte, snapshot func() (nativeSnapshot, error)) bool {
	if !openingPlaneAlias(mem) || snapshot == nil {
		return false
	}
	profile := graphicsSourceProfile{Name: "VISUAL.OZM", Width: 640, Height: 400, Planes: hdScrollSourcePlanes}
	_, source, ok := graphicsSource(mem, []graphicsSourceProfile{profile})
	if !ok {
		return false
	}
	native, err := snapshot()
	return err == nil && len(native.Indexed) == len(source) && bytes.Equal(native.Indexed, source)
}

// 固定 EXE 的原版入口與返回觀測；不修改 guest 或取代指令。
type hdLive struct {
	Graphics        hdGraphicsState     `json:"graphics"`
	BattleFrame     battleFrameState    `json:"battle_frame"`
	Terrain         hdTerrainState      `json:"terrain"`
	Window          hdWindowState       `json:"window"`
	WindowLayers    hdWindowLayersState `json:"window_layers"`
	Kao             hdKaoState          `json:"kao"`
	Trace           hdTrace             `json:"loads"`
	LastName        string              `json:"last_resource"`
	Validated       bool                `json:"source_validated"`
	Active          bool                `json:"draw_completed"`
	StoryRect       image.Rectangle     `json:"story_destination"`
	Pending         *hdDrawReturn       `json:"pending_draw"`
	Events          []map[string]any    `json:"draw_events"`
	ScrollValidated bool                `json:"scroll_source_validated"`
	ScrollActive    bool                `json:"scroll_draw_completed"`
	ScrollPending   *hdDrawReturn       `json:"pending_scroll_draw"`
}

func (h *hdLive) observe(step int, eip, esp uint32, mem []byte, snapshots ...func() (nativeSnapshot, error)) {
	var snapshot func() (nativeSnapshot, error)
	if len(snapshots) == 1 {
		snapshot = snapshots[0]
	}
	h.observeWindowLayers(step, eip, esp, mem, snapshot)
	h.Graphics.observe(step, eip, esp, mem, snapshot)
	h.BattleFrame.observe(step, eip, esp, mem, snapshot)
	h.Window.observe(step, eip, esp, mem)
	h.Kao.observe(step, eip, esp, mem)
	if h.ScrollPending != nil && eip == h.ScrollPending.Return && uint64(esp) == uint64(h.ScrollPending.Stack)+4 {
		h.ScrollActive = true
		if h.ScrollPending.Entry == 0x11f14 {
			h.ScrollActive = openingScrollMatches(mem, snapshot)
		}
		h.ScrollPending = nil
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "scroll_draw_return", "step": step, "linear_eip": eip, "eligible": h.ScrollActive})
		}
	}
	if h.Pending != nil && eip == h.Pending.Return && uint64(esp) == uint64(h.Pending.Stack)+4 {
		known := prefixStoryKnown(h.LastName)
		h.Active = h.Validated && known
		if h.Pending.Destination == endingStoryRect {
			h.Active = h.Active && endingStoryReturned(h.Pending, h.LastName, mem, snapshot)
		} else if h.Pending.SourceGuard != "" {
			h.Active = h.Active && prefixStoryReturned(h.Pending, h.LastName, mem, snapshot)
		}
		if h.Pending.Entry == 0x11dcb {
			h.Active = h.Active && openingStoryMatches(h.LastName, mem, snapshot)
		}
		if h.Active {
			h.StoryRect = h.Pending.Destination
		}
		h.Pending = nil
		if len(h.Events) < 128 {
			h.Events = append(h.Events, map[string]any{"kind": "draw_return", "step": step, "linear_eip": eip, "eligible": h.Active})
		}
	}
	if eip == 0x1bf44 {
		h.StoryRect = image.Rectangle{}
		h.Graphics.Shift = nil
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
			h.BattleFrame.loaded(e)
			h.LastName, _ = e["name"].(string)
			if h.LastName == "VISUAL.OZM" && e["raw_columns"] == uint32(80) && e["raw_rows"] == uint32(400) {
				planes, ok := e["planes"].([]map[string]any)
				want := hdScrollSourcePlanes
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
			if _, partial := hdPrefixProfiles[h.LastName]; partial {
				_, h.Validated = prefixStorySource(h.LastName, mem, false)
			}
		}
	}
	if eip == 0x1c084 || eip == 0x11f14 {
		h.ScrollActive = false
		h.ScrollPending = nil
		h.Active = false
		h.Pending = nil
		if h.ScrollValidated && h.LastName == "VISUAL.OZM" && uint64(esp)+4 <= uint64(len(mem)) && (eip != 0x11f14 || openingPlaneAlias(mem)) {
			ret := binary.LittleEndian.Uint32(mem[esp:])
			if uint64(ret) < uint64(len(mem)) {
				h.ScrollPending = &hdDrawReturn{Return: ret, Stack: esp, Entry: eip}
				if len(h.Events) < 128 {
					h.Events = append(h.Events, map[string]any{"kind": "scroll_draw_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp, "logical_rect": []int{0, 250, 640, 150}})
				}
			}
		}
	}
	if openingPartialEntry(eip) || openingShiftEntry(eip) {
		h.Active, h.Pending = false, nil
	}
	ending := endingStoryCall(eip, esp, mem, h.LastName)
	if eip != 0x1c1bc && eip != 0x11dcb && !(eip == 0x1c10b && ending) {
		return
	}
	h.Active = false
	h.Pending = nil
	known := prefixStoryKnown(h.LastName)
	stackBytes := uint64(24)
	if eip == 0x11dcb {
		stackBytes = 4
	}
	if !h.Validated || !known || uint64(esp)+stackBytes > uint64(len(mem)) {
		return
	}
	args := make([]uint32, 5)
	if eip == 0x11dcb {
		if _, valid := openingStorySource(h.LastName, mem); !valid {
			return
		}
		copy(args, []uint32{0, 40, 150, 80, 0xa0fb4})
	} else {
		for i := range args {
			args[i] = binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):])
		}
	}
	want := []uint32{0, 40, 150, 80, 0xa0fb4}
	if ending {
		want[4] = 0xa0502
	}
	for i := range args {
		if args[i] != want[i] {
			return
		}
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	if uint64(ret) >= uint64(len(mem)) {
		return
	}
	h.Pending = &hdDrawReturn{Return: ret, Stack: esp, Entry: eip, Destination: a0Rect}
	if ending {
		bases, digest, valid := endingStoryCurrent(mem, h.LastName)
		if !valid {
			h.Pending = nil
			return
		}
		h.Pending.Destination = endingStoryRect
		h.Pending.SourceName, h.Pending.SourceGuard, h.Pending.SourceBases = h.LastName, digest, bases
	}
	if _, partial := hdPrefixProfiles[h.LastName]; partial {
		current, valid := prefixStorySource(h.LastName, mem, eip == 0x11dcb)
		if !valid {
			h.Pending = nil
			return
		}
		h.Pending.SourceName = h.LastName
		h.Pending.SourceGuard = prefixStorySHA(current.Indexed)
		h.Pending.SourceBases = current.Bases
	}
	if len(h.Events) < 128 {
		dest := h.Pending.Destination
		event := map[string]any{"kind": "draw_enter", "step": step, "linear_eip": eip, "return_linear": ret, "stack_linear": esp, "raw_args": args, "logical_rect": []int{dest.Min.X, dest.Min.Y, dest.Dx(), dest.Dy()}}
		if ending {
			event["source_name"], event["source_indices_sha256"], event["source_plane_bases"] = h.LastName, h.Pending.SourceGuard, h.Pending.SourceBases
			if eip == 0x1c10b {
				event["raw_plane_destinations"] = []uint32{0xa0502, 0xa0502, 0xa0502, 0xa0502}
			}
		}
		h.Events = append(h.Events, event)
	}
}
