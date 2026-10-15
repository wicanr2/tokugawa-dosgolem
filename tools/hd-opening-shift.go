package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
)

// IDA 9.4 LE 線性 EA、十三筆普通原版來源與 READY 見 RESEARCH-LOG。
type openingShiftSpec struct {
	Name                      string
	X, Columns, Rows, Maximum int
	Dest                      image.Rectangle
}

func openingShiftEntry(eip uint32) bool { return eip == 0x11ea4 || eip == 0x11c83 }

func openingShiftDefinition(eip uint32) (openingShiftSpec, bool) {
	switch eip {
	case 0x11ea4:
		return openingShiftSpec{"OP2.OZM", 240, 30, 150, 240, image.Rect(200, 50, 440, 200)}, true
	case 0x11c83:
		return openingShiftSpec{"OP8.OZM", 440, 26, 200, 440, image.Rect(216, 30, 424, 230)}, true
	}
	return openingShiftSpec{}, false
}

func openingShiftByName(name string) (openingShiftSpec, bool) {
	for _, entry := range []uint32{0x11ea4, 0x11c83} {
		s, _ := openingShiftDefinition(entry)
		if s.Name == name {
			return s, true
		}
	}
	return openingShiftSpec{}, false
}

type openingShiftState struct {
	Name     string
	Bases    [4]uint32
	Original []byte
	Count    int
}

type openingShiftGuard struct {
	Bases  [4]uint32
	RawSHA string
}

// OP8 的線性複製含來源外一 byte；觀測這個 byte 不授予美術身份。
func openingShiftCurrent(mem []byte, spec openingShiftSpec) ([4]uint32, []byte, string, bool) {
	var bases [4]uint32
	if len(mem) < 0x5b1bc || !openingPlaneAlias(mem) || binary.LittleEndian.Uint32(mem[0x5b1b8:]) != 80 || binary.LittleEndian.Uint32(mem[0x5b1b4:]) != 200 {
		return bases, nil, "", false
	}
	length := 16000
	if spec.Name == "OP8.OZM" {
		length++
	}
	raw := make([]byte, length*4)
	current := make([]byte, length*8)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := binary.LittleEndian.Uint32(mem[global:])
		end := uint64(base) + uint64(length)
		if end > uint64(len(mem)) || (base < 0xc0000 && end > 0xa0000) {
			return bases, nil, "", false
		}
		bases[bit] = base
		plane := mem[uint64(base):end]
		copy(raw[bit*length:], plane)
		for i, value := range plane {
			for x := 0; x < 8; x++ {
				current[i*8+x] |= ((value >> uint(7-x)) & 1) << uint(bit)
			}
		}
	}
	return bases, current, fmt.Sprintf("%x", sha256.Sum256(raw)), true
}

func openingShiftIndex(position, count int) int {
	if position < 16 {
		return position
	}
	return max(15, position-count)
}

// 255 明示來源外未知，與原生 0..15 索引分開。
func openingShiftQualified(original []byte, spec openingShiftSpec, count int) []byte {
	if len(original) != 128000 || count < 0 || count > spec.Maximum {
		return nil
	}
	out := make([]byte, spec.Columns*8*spec.Rows)
	for y := 0; y < spec.Rows; y++ {
		for x := 0; x < spec.Columns*8; x++ {
			index := openingShiftIndex(y*640+spec.X+x, count)
			value := byte(255)
			if index < len(original) {
				value = original[index]
			}
			out[y*spec.Columns*8+x] = value
		}
	}
	return out
}

func (t *graphicsTrace) openingShiftPrepare(step int, eip, esp uint32, mem []byte) {
	spec, _ := openingShiftDefinition(eip)
	r := graphicsDrawRecord{Name: spec.Name, Kind: "opening_shift", BeginStep: step, Stack: esp,
		SourceRect: image.Rect(spec.X, 0, spec.X+spec.Columns*8, spec.Rows), DestRect: spec.Dest,
		Args: [5]uint32{uint32(spec.X / 8), uint32(spec.Columns), uint32(spec.Rows), 80, uint32(0xa0000 + spec.Dest.Min.Y*80 + spec.Dest.Min.X/8)}}
	fail := func(reason string) {
		r.Error = reason
		t.Records = append(t.Records, r)
		t.Pending = nil
		t.Shift = nil
	}
	if uint64(esp)+4 > uint64(len(mem)) {
		fail("shift stack outside RAM")
		return
	}
	r.Return = binary.LittleEndian.Uint32(mem[esp:])
	if uint64(r.Return) >= uint64(len(mem)) {
		fail("shift return outside RAM")
		return
	}
	bases, current, rawSHA, ok := openingShiftCurrent(mem, spec)
	if !ok {
		fail("shift source geometry or aliases invalid")
		return
	}
	state := t.Shift
	count := 0
	valid := state != nil && state.Name == spec.Name && state.Bases == bases && len(state.Original) == 128000 && state.Count < spec.Maximum
	if valid {
		count = state.Count + 1
		for i := 0; i < 128000; i++ {
			if current[i] != state.Original[openingShiftIndex(i, count)] {
				valid = false
				break
			}
		}
	}
	if !valid {
		// 重新開始僅接受原始完整四平面，不搜尋可匹配的位移數。
		var profile graphicsSourceProfile
		for _, p := range t.Profiles {
			if p.Name == spec.Name {
				profile = p
				break
			}
		}
		_, original, matched := graphicsSource(mem, []graphicsSourceProfile{profile})
		if !matched || len(original) != 128000 {
			fail("unmatched exact or consecutive shifted source")
			return
		}
		state = &openingShiftState{Name: spec.Name, Bases: bases, Original: original}
		count = 0
	}
	qualified := openingShiftQualified(state.Original, spec, count)
	expected := make([]byte, len(qualified))
	for y := 0; y < spec.Rows; y++ {
		for x := 0; x < spec.Columns*8; x++ {
			i := y*spec.Columns*8 + x
			actual := current[y*640+spec.X+x]
			if qualified[i] != 255 && qualified[i] != actual {
				fail("shift guard differs from known source")
				return
			}
			expected[i] = actual
		}
	}
	r.SourceShift = count
	r.QualifiedSHA = fmt.Sprintf("%x", sha256.Sum256(qualified))
	r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(expected))
	state.Count = count
	t.Shift = state
	t.Pending = &graphicsPending{Record: r, Expected: expected, Shift: &openingShiftGuard{Bases: bases, RawSHA: rawSHA}}
}

func openingShiftReturn(p *graphicsPending, mem []byte) bool {
	spec, ok := openingShiftByName(p.Record.Name)
	if !ok || p.Shift == nil {
		return false
	}
	bases, _, digest, valid := openingShiftCurrent(mem, spec)
	return valid && bases == p.Shift.Bases && digest == p.Shift.RawSHA
}

func (p hdGraphicsPair) applyOpeningShift(out *image.NRGBA, s nativeSnapshot, draw graphicsDrawRecord, r hdGraphicsReport) hdGraphicsReport {
	r.Mode = "fallback_geometry"
	spec, ok := openingShiftByName(draw.Name)
	if !ok || out == nil || out.Bounds() != image.Rect(0, 0, 640*p.Scale, 400*p.Scale) || s.Image == nil || s.Image.Bounds() != image.Rect(0, 0, 640, 400) || len(s.Indexed) != 256000 || p.Profile.Source.Width != 640 || p.Profile.Source.Height != 200 || draw.DestRect != spec.Dest || draw.SourceRect != image.Rect(spec.X, 0, spec.X+spec.Columns*8, spec.Rows) {
		return r
	}
	r.Mode = "fallback_no_verified_draw"
	if !draw.Verified || draw.Name != p.Profile.Source.Name {
		return r
	}
	qualified := openingShiftQualified(p.Source, spec, draw.SourceShift)
	if qualified == nil || fmt.Sprintf("%x", sha256.Sum256(qualified)) != draw.QualifiedSHA {
		return r
	}
	r.Mode = "fallback_palette"
	if s.MaxColor != 15 {
		return r
	}
	effect := graphicsPaletteEffect(p.Profile, s)
	if !effect.Valid {
		return r
	}
	r.Effect = &effect
	visible := draw.VisibleRects
	if visible == nil {
		visible = []image.Rectangle{spec.Dest}
	}
	r.Mode = "fallback_visibility"
	if len(visible) > 256 {
		return r
	}
	if len(visible) == 0 {
		r.Mode = "fallback_no_visible_art"
		return r
	}
	for i, f := range visible {
		if f.Empty() || !f.In(spec.Dest) {
			return r
		}
		for _, old := range visible[:i] {
			if f.Overlaps(old) {
				return r
			}
		}
	}
	r.Mode = "fallback_source_mismatch"
	for _, f := range visible {
		for y := f.Min.Y; y < f.Max.Y; y++ {
			for x := f.Min.X; x < f.Max.X; x++ {
				want := qualified[(y-spec.Dest.Min.Y)*spec.Dest.Dx()+x-spec.Dest.Min.X]
				if want != 255 && s.Indexed[y*640+x] != want {
					return r
				}
			}
		}
	}
	painted := func(point image.Point) bool {
		if len(p.Profile.PaintRects) > 0 {
			for _, rect := range p.Profile.PaintRects {
				if point.In(rect) {
					return true
				}
			}
			return false
		}
		return p.Profile.PaintRect.Empty() || point.In(p.Profile.PaintRect)
	}
	for _, f := range visible {
		for y := f.Min.Y; y < f.Max.Y; y++ {
			for x := f.Min.X; x < f.Max.X; x++ {
				position := openingShiftIndex((y-spec.Dest.Min.Y)*640+spec.X+x-spec.Dest.Min.X, draw.SourceShift)
				known := position < len(p.Source)
				sx, sy := position%640, position/640
				preserve := !known || !painted(image.Pt(sx, sy))
				if known && p.preserves(sx, sy) {
					preserve = true
				}
				if preserve {
					r.PreservedNative++
				} else {
					r.PaintedNative++
				}
				for dy := 0; dy < p.Scale; dy++ {
					for dx := 0; dx < p.Scale; dx++ {
						pixel := rgbaAt(s.Image, x, y)
						if !preserve {
							pixel = effect.colour(rgbaAt(p.Asset, sx*p.Scale+dx, sy*p.Scale+dy))
						}
						out.SetNRGBA(x*p.Scale+dx, y*p.Scale+dy, pixel)
					}
				}
			}
		}
	}
	r.VisibleRects = append([]image.Rectangle(nil), visible...)
	r.Mode = "hd"
	return r
}
