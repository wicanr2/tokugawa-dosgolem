package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"os"
)

type graphicsSourceProfile struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Width, Height int
	Planes        []string `json:"plane_sha256"`
}

type graphicsDrawRecord struct {
	Name                   string `json:"resource"`
	Kind                   string `json:"kind"`
	BeginStep, EndStep     int
	Return, Stack          uint32
	Args                   [5]uint32 `json:"raw_args"`
	SourceRect, DestRect   image.Rectangle
	ExpectedSHA, ActualSHA string
	Verified               bool              `json:"verified"`
	Error                  string            `json:"observation_error,omitempty"`
	VisibleRects           []image.Rectangle `json:"visible_rectangles,omitempty"`
}

type graphicsPending struct {
	Record   graphicsDrawRecord
	Expected []byte
}
type graphicsTrace struct {
	Profiles []graphicsSourceProfile `json:"-"`
	Records  []graphicsDrawRecord    `json:"records"`
	Pending  *graphicsPending        `json:"-"`
	Dropped  uint64                  `json:"dropped_events"`
}

func newGraphicsTrace(path string) (graphicsTrace, error) {
	var inventory struct {
		Records []graphicsSourceProfile `json:"records"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return graphicsTrace{}, err
	}
	if err = json.Unmarshal(data, &inventory); err != nil {
		return graphicsTrace{}, err
	}
	var t graphicsTrace
	for _, p := range inventory.Records {
		if p.Status == "decoded" {
			t.Profiles = append(t.Profiles, p)
		}
	}
	if len(t.Profiles) != 107 {
		return t, fmt.Errorf("expected 107 decoded sources, got %d", len(t.Profiles))
	}
	for _, profile := range hdGraphicsOpeningProfiles {
		t.Profiles = append(t.Profiles, profile.Source)
	}
	return t, nil
}

func openingPlaneAlias(mem []byte) bool {
	if len(mem) < 0x38c08 {
		return false
	}
	for i, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		if binary.LittleEndian.Uint32(mem[0x386d4+i*4:]) != binary.LittleEndian.Uint32(mem[global:]) {
			return false
		}
	}
	return true
}

func graphicsSource(mem []byte, profiles []graphicsSourceProfile) (graphicsSourceProfile, []byte, bool) {
	if len(mem) < 0x5b1bc {
		return graphicsSourceProfile{}, nil, false
	}
	cols, rows := binary.LittleEndian.Uint32(mem[0x5b1b8:]), binary.LittleEndian.Uint32(mem[0x5b1b4:])
	if cols < 1 || cols > 80 || rows < 1 || rows > 400 {
		return graphicsSourceProfile{}, nil, false
	}
	var planes [4][]byte
	var hashes [4]string
	for i, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := binary.LittleEndian.Uint32(mem[global:])
		end := uint64(base) + uint64(rows-1)*80 + uint64(cols)
		if end > uint64(len(mem)) || (base < 0xc0000 && end > 0xa0000) {
			return graphicsSourceProfile{}, nil, false
		}
		planes[i] = make([]byte, int(cols*rows))
		for y := uint32(0); y < rows; y++ {
			copy(planes[i][y*cols:(y+1)*cols], mem[uint64(base)+uint64(y)*80:uint64(base)+uint64(y)*80+uint64(cols)])
		}
		hashes[i] = fmt.Sprintf("%x", sha256.Sum256(planes[i]))
	}
	for _, p := range profiles {
		if p.Width != int(cols)*8 || p.Height != int(rows) || len(p.Planes) != 4 {
			continue
		}
		match := true
		for i, want := range p.Planes {
			if hashes[i] != want {
				match = false
			}
		}
		if !match {
			continue
		}
		idx := make([]byte, p.Width*p.Height)
		for y := 0; y < p.Height; y++ {
			for x := 0; x < p.Width; x++ {
				for bit, plane := range planes {
					idx[y*p.Width+x] |= ((plane[y*int(cols)+x/8] >> uint(7-x%8)) & 1) << uint(bit)
				}
			}
		}
		return p, idx, true
	}
	return graphicsSourceProfile{}, nil, false
}

func graphicsCrop(idx []byte, width int, r image.Rectangle) []byte {
	if width < 1 || len(idx)%width != 0 || r.Empty() || !r.In(image.Rect(0, 0, width, len(idx)/width)) {
		return nil
	}
	out := make([]byte, r.Dx()*r.Dy())
	for y := 0; y < r.Dy(); y++ {
		copy(out[y*r.Dx():(y+1)*r.Dx()], idx[(r.Min.Y+y)*width+r.Min.X:(r.Min.Y+y)*width+r.Max.X])
	}
	return out
}

func (t *graphicsTrace) observe(step int, eip, esp uint32, mem []byte, snapshot func() (nativeSnapshot, error)) {
	if p := t.Pending; p != nil && eip == p.Record.Return && uint64(esp) == uint64(p.Record.Stack)+4 {
		r := p.Record
		r.EndStep = step
		profile, _, ok := graphicsSource(mem, t.Profiles)
		if (r.Kind == "opening_full" || r.Kind == "opening_rect") && !openingPlaneAlias(mem) {
			r.Error = "opening source alias changed before natural return"
		} else if !ok || profile.Name != r.Name {
			r.Error = "source changed before natural return"
		} else {
			var s nativeSnapshot
			var err error
			if snapshot == nil {
				err = fmt.Errorf("native snapshot unavailable")
			} else {
				s, err = snapshot()
			}
			if err != nil {
				r.Error = err.Error()
			} else {
				actual := graphicsCrop(s.Indexed, 640, r.DestRect)
				r.ActualSHA = fmt.Sprintf("%x", sha256.Sum256(actual))
				r.Verified = len(actual) == len(p.Expected) && r.ActualSHA == r.ExpectedSHA
				if !r.Verified {
					r.Error = "opaque destination differs from source"
				}
			}
		}
		t.Records = append(t.Records, r)
		t.Pending = nil
	}
	if eip != 0x1c1bc && eip != 0x1c084 && eip != 0x11f14 && !openingPartialEntry(eip) {
		return
	}
	if len(t.Records) >= 512 {
		t.Dropped++
		t.Pending = nil
		return
	}
	r := graphicsDrawRecord{BeginStep: step, Stack: esp, Kind: "opaque_rect"}
	stackBytes := uint64(24)
	if openingPartialEntry(eip) {
		stackBytes = 8
	}
	if uint64(esp)+stackBytes > uint64(len(mem)) {
		r.Error = "stack outside RAM"
		t.Records = append(t.Records, r)
		return
	}
	r.Return = binary.LittleEndian.Uint32(mem[esp:])
	if uint64(r.Return) >= uint64(len(mem)) {
		return
	}
	if eip == 0x1c084 || eip == 0x11f14 {
		r.Kind = "opaque_full"
		if eip == 0x11f14 {
			r.Kind = "opening_full"
			if !openingPlaneAlias(mem) {
				r.Error = "opening source pointers are not verified loader aliases"
				t.Records = append(t.Records, r)
				return
			}
		}
		r.Args = [5]uint32{0, 80, 400, 80, 0xa0000}
	} else if openingPartialEntry(eip) {
		r.Kind = "opening_rect"
		var valid bool
		r.Args, valid = openingPartialArgs(eip, esp, mem)
		if !valid || !openingPlaneAlias(mem) {
			r.Error = "unsupported opening offset or source aliases"
			t.Records = append(t.Records, r)
			return
		}
	} else {
		for i := range r.Args {
			r.Args[i] = binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):])
		}
	}
	p, idx, ok := graphicsSource(mem, t.Profiles)
	if !ok {
		r.Error = "unmatched RAM source"
		t.Records = append(t.Records, r)
		return
	}
	r.Name = p.Name
	if openingPartialEntry(eip) && r.Name != openingPartialSource(eip) {
		r.Error = "opening entry does not match exact source"
		t.Records = append(t.Records, r)
		return
	}
	a := r.Args
	sy, sx := a[0]/80, a[0]%80
	if a[1] < 1 || a[1] > 80 || a[2] < 1 || a[2] > 400 || a[3] != 80 || a[4] < 0xa0000 || a[4] >= 0xa0000+32000 {
		r.Error = "unsupported parameters"
		t.Records = append(t.Records, r)
		return
	}
	d := a[4] - 0xa0000
	dy, dx := d/80, d%80
	r.SourceRect = image.Rect(int(sx*8), int(sy), int((sx+a[1])*8), int(sy+a[2]))
	r.DestRect = image.Rect(int(dx*8), int(dy), int((dx+a[1])*8), int(dy+a[2]))
	expected := graphicsCrop(idx, p.Width, r.SourceRect)
	if expected == nil || !r.DestRect.In(image.Rect(0, 0, 640, 400)) {
		r.Error = "source or destination rectangle outside bounds"
		t.Records = append(t.Records, r)
		return
	}
	r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(expected))
	t.Pending = &graphicsPending{Record: r, Expected: expected}
}
