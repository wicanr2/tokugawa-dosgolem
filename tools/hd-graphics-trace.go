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
	SourceCropSHA          string            `json:"source_crop_sha256,omitempty"`
	CompositeSourceSHA     string            `json:"composite_source_sha256,omitempty"`
	Verified               bool              `json:"verified"`
	SourceShift            int               `json:"source_shift,omitempty"`
	QualifiedSHA           string            `json:"known_source_sha256,omitempty"`
	Error                  string            `json:"observation_error,omitempty"`
	VisibleRects           []image.Rectangle `json:"visible_rectangles,omitempty"`
	RawHelperArgs          []uint32          `json:"raw_helper_args,omitempty"`
	CacheBase              uint32            `json:"raw_cache_base_linear,omitempty"`
}

func wakuFrameEntry(eip, esp uint32, mem []byte) bool {
	return eip == 0x1fff7 && uint64(esp)+4 <= uint64(len(mem)) && binary.LittleEndian.Uint32(mem[esp:]) == 0x1ec31
}

const wakuCachedLowerSHA = "a1c858f9c927dc56b81277328d84b2ffb00eb280ed91d6162c7bde374b608b9d"

type wakuCornerSpec struct {
	Return, Destination uint32
	Tile                int
}

var wakuCorners = []wakuCornerSpec{
	{0x1ecb4, 0xa2585, 0}, {0x1ed37, 0xa3985, 0}, {0x1edba, 0xa4d85, 0}, {0x1ee3d, 0xa6185, 2},
	{0x1eec0, 0xa25b1, 1}, {0x1ef43, 0xa39b1, 1}, {0x1efc6, 0xa4db1, 1}, {0x1f049, 0xa61b1, 3},
}

var wakuCornerCropSHA = [4]string{
	"efbd133e2c4cbbfad9298274c95812054155dae571810e34197e5f548c84f5d2",
	"d097a97cc80cfb18223fa626ecd6c187315ed373e83c08c58fbffcca7f7ff11f",
	"4d370301420d7cd334cf4da39ac2fecd5d93cc810d3ef6dcd4e74ccff966ee3f",
	"49c1db8d57df9f1e763b9f28077d4a0efc25989b3ef0452fac74feef75b62296",
}

func wakuCornerByReturn(ret uint32) (wakuCornerSpec, bool) {
	for _, spec := range wakuCorners {
		if spec.Return == ret {
			return spec, true
		}
	}
	return wakuCornerSpec{}, false
}

func wakuCornerEntry(eip, esp uint32, mem []byte) (wakuCornerSpec, bool) {
	if eip != 0x20020 || uint64(esp)+4 > uint64(len(mem)) {
		return wakuCornerSpec{}, false
	}
	return wakuCornerByReturn(binary.LittleEndian.Uint32(mem[esp:]))
}

func (s wakuCornerSpec) sourceRect() image.Rectangle {
	return image.Rect(s.Tile*32, 88, s.Tile*32+32, 120)
}
func (s wakuCornerSpec) destRect() image.Rectangle {
	d := s.Destination - 0xa0000
	return image.Rect(int(d%80*8), int(d/80), int(d%80*8)+32, int(d/80)+32)
}

// 對靜態來源獨立重建原版實際 reader；不將 0x4080 改成相鄰圖塊平面。
func wakuCornerIndices(source []byte, tile int) []byte {
	if len(source) != 384*120 || tile < 0 || tile > 3 {
		return nil
	}
	indices := graphicsCrop(source, 384, image.Rect(tile*32, 88, tile*32+32, 120))
	if tile == 0 {
		for i, value := range indices {
			indices[i] = (value &^ 2) | ((source[80*384+i] >> 3 & 1) << 1)
		}
	}
	return indices
}

func wakuCornerSource(mem []byte, profiles []graphicsSourceProfile, spec wakuCornerSpec) (graphicsSourceProfile, []byte, uint32, bool) {
	profile, _, base, ok := wakuFrameSource(mem, profiles)
	if !ok || spec.Tile < 0 || spec.Tile > 3 || uint64(base)+0x4a00 > uint64(len(mem)) {
		return graphicsSourceProfile{}, nil, 0, false
	}
	if fmt.Sprintf("%x", sha256.Sum256(mem[base+0x4200:base+0x4a00])) != wakuCachedLowerSHA {
		return graphicsSourceProfile{}, nil, 0, false
	}
	indices := make([]byte, 1024)
	for bit := 0; bit < 4; bit++ {
		offset := uint32(0x4200 + spec.Tile*0x200 + bit*128)
		if spec.Tile == 0 && bit == 1 {
			offset = 0x4080
		}
		plane := mem[base+offset : base+offset+128]
		for i := range indices {
			indices[i] |= ((plane[i/8] >> uint(7-i%8)) & 1) << uint(bit)
		}
	}
	return profile, indices, base, true
}

// 原始 dword_42188 的四個 packed plane；指紋來自完整 MAP_WAKU 前 88 列。
func wakuFrameSource(mem []byte, profiles []graphicsSourceProfile) (graphicsSourceProfile, []byte, uint32, bool) {
	if len(mem) < 0x4218c {
		return graphicsSourceProfile{}, nil, 0, false
	}
	base := binary.LittleEndian.Uint32(mem[0x42188:])
	if base < 0xc0000 || uint64(base)+0x4200 > uint64(len(mem)) {
		return graphicsSourceProfile{}, nil, 0, false
	}
	want := [4]string{
		"a5a88414fb10c4e9a154d055c202b825b263a945706d0cdf897ea25f106ad746",
		"0694dc10a24b46b0bff849fc469957072b5b45c21d68bc6afe845d491d01a2ed",
		"7f772998f44526537b3bf799122a3faeb6ce249697e2fcc9400e7cfa301d153a",
		"bf94d263064055bf14cb82191c973f5035224ac5faba6b6849d5c667365b3235",
	}
	var planes [4][]byte
	for bit := range planes {
		start := uint64(base) + uint64(bit)*0x1080
		planes[bit] = mem[start : start+0x1080]
		if fmt.Sprintf("%x", sha256.Sum256(planes[bit])) != want[bit] {
			return graphicsSourceProfile{}, nil, 0, false
		}
	}
	for _, profile := range profiles {
		if profile.Name != "MAP_WAKU.OZM" || profile.Width != 384 || profile.Height != 120 || len(profile.Planes) != 4 {
			continue
		}
		indices := make([]byte, 384*88)
		for n := range indices {
			for bit, plane := range planes {
				indices[n] |= ((plane[n/8] >> uint(7-n%8)) & 1) << uint(bit)
			}
		}
		return profile, indices, base, true
	}
	return graphicsSourceProfile{}, nil, 0, false
}

type graphicsPending struct {
	Record   graphicsDrawRecord
	Expected []byte
	Shift    *openingShiftGuard
}
type graphicsTrace struct {
	Profiles []graphicsSourceProfile `json:"-"`
	Records  []graphicsDrawRecord    `json:"records"`
	Pending  *graphicsPending        `json:"-"`
	Dropped  uint64                  `json:"dropped_events"`
	Shift    *openingShiftState      `json:"-"`
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
		if r.Kind == "cached_frame" {
			var base uint32
			profile, _, base, ok = wakuFrameSource(mem, t.Profiles)
			ok = ok && base == r.CacheBase
		}
		if r.Kind == "cached_corner" {
			spec, matched := wakuCornerByReturn(r.Return)
			var base uint32
			profile, _, base, ok = wakuCornerSource(mem, t.Profiles, spec)
			ok = ok && matched && base == r.CacheBase
		}
		if r.Kind == "ending_shift" {
			ok = endingShiftReturn(p, mem)
			profile.Name = r.Name
			if !ok {
				t.Shift = nil
			}
		}
		if r.Kind == "opening_shift" {
			ok = openingShiftReturn(p, mem)
			profile.Name = r.Name
			if !ok {
				t.Shift = nil
			}
		}
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
		if (r.Kind == "opening_shift" || r.Kind == "ending_shift") && !r.Verified {
			t.Shift = nil
		}
		t.Records = append(t.Records, r)
		t.Pending = nil
	}
	cached := wakuFrameEntry(eip, esp, mem)
	corner, cachedCorner := wakuCornerEntry(eip, esp, mem)
	if eip != 0x1c1bc && eip != 0x1c084 && eip != 0x11f14 && !openingPartialEntry(eip) && !openingShiftEntry(eip) && !endingShiftEntry(eip) && !cached && !cachedCorner {
		return
	}
	used, limit := 0, 512
	if cached || cachedCorner {
		limit = 1024
	}
	for _, record := range t.Records {
		isCache := record.Kind == "cached_frame" || record.Kind == "cached_corner"
		if isCache == (cached || cachedCorner) {
			used++
		}
	}
	if used >= limit {
		t.Dropped++
		t.Pending = nil
		return
	}
	if openingShiftEntry(eip) {
		t.openingShiftPrepare(step, eip, esp, mem)
		return
	}
	if endingShiftEntry(eip) {
		t.endingShiftPrepare(step, esp, mem)
		return
	}
	r := graphicsDrawRecord{BeginStep: step, Stack: esp, Kind: "opaque_rect"}
	stackBytes := uint64(24)
	if openingPartialEntry(eip) {
		stackBytes = 8
	}
	if cached || cachedCorner {
		stackBytes = 12
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
	if cachedCorner {
		r.Kind, r.Name = "cached_corner", "MAP_WAKU.OZM"
		r.RawHelperArgs = []uint32{binary.LittleEndian.Uint32(mem[esp+4:]), binary.LittleEndian.Uint32(mem[esp+8:])}
		_, indices, base, ok := wakuCornerSource(mem, t.Profiles, corner)
		if !ok || uint64(r.RawHelperArgs[0]) != uint64(base)+uint64(0x4380+corner.Tile*0x200) || r.RawHelperArgs[1] != corner.Destination {
			r.Error = "unmatched cached corner source or fourth-plane arguments"
			t.Records = append(t.Records, r)
			return
		}
		r.CacheBase = base
		r.Args = [5]uint32{uint32(88*80 + corner.Tile*4), 4, 32, 80, corner.Destination}
		r.SourceRect, r.DestRect = corner.sourceRect(), corner.destRect()
		r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(indices))
		r.SourceCropSHA = wakuCornerCropSHA[corner.Tile]
		r.CompositeSourceSHA = "1d0ee5b9b52142beb6813ae52f1db215e5224fae1935a64ea83fe07b97fe9a6b"
		r.QualifiedSHA = wakuCachedLowerSHA
		t.Pending = &graphicsPending{Record: r, Expected: indices}
		return
	}
	if cached {
		r.Kind, r.Name = "cached_frame", "MAP_WAKU.OZM"
		r.RawHelperArgs = []uint32{binary.LittleEndian.Uint32(mem[esp+4:]), binary.LittleEndian.Uint32(mem[esp+8:])}
		_, indices, base, ok := wakuFrameSource(mem, t.Profiles)
		if !ok || uint64(r.RawHelperArgs[0]) != uint64(base)+0x3180 || r.RawHelperArgs[1] != 0xa0005 {
			r.Error = "unmatched cached source or fourth-plane arguments"
			t.Records = append(t.Records, r)
			return
		}
		r.CacheBase = base
		r.Args = [5]uint32{0, 48, 88, 80, 0xa0005}
		r.SourceRect, r.DestRect = image.Rect(0, 0, 384, 88), image.Rect(40, 0, 424, 88)
		r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(indices))
		t.Pending = &graphicsPending{Record: r, Expected: indices}
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
	knownRows := a[2]
	if eip == 0x1c084 && p.Width == 640 && p.Height == 399 && (p.Name == "END-11.OZM" || p.Name == "END-11B.OZM") {
		knownRows, r.Kind = 399, "opaque_full_known_rows"
	}
	r.SourceRect = image.Rect(int(sx*8), int(sy), int((sx+a[1])*8), int(sy+knownRows))
	r.DestRect = image.Rect(int(dx*8), int(dy), int((dx+a[1])*8), int(dy+knownRows))
	expected := graphicsCrop(idx, p.Width, r.SourceRect)
	if expected == nil || !r.DestRect.In(image.Rect(0, 0, 640, 400)) {
		r.Error = "source or destination rectangle outside bounds"
		t.Records = append(t.Records, r)
		return
	}
	r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(expected))
	t.Pending = &graphicsPending{Record: r, Expected: expected}
}
