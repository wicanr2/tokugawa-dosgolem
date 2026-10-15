package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

// DRAFT diagnostics only. No replacement state is passed to the HD presenter.
type compositeRecord struct {
	Entry, Return, Stack                  uint32
	BeginStep, EndStep                    int
	Kind, Source                          string
	Args                                  [4]uint32
	SourceRect, DestRect                  image.Rectangle
	SourceSHA, BeforeSHA, ExpectedSHA     string
	ActualSHA                             string
	SourcePath, BeforePath, AfterPath     string
	NonzeroSource, MixedOR, ChangedPixels int
	Verified                              bool   `json:"verified"`
	Error                                 string `json:"observation_error,omitempty"`
}

type compositePending struct {
	Record   compositeRecord
	Expected []byte
	Selected []byte
}

type compositeProbe struct {
	Profiles  []graphicsSourceProfile `json:"-"`
	Directory string                  `json:"-"`
	Records   []compositeRecord       `json:"records"`
	Pending   *compositePending       `json:"-"`
	Dropped   int                     `json:"dropped_events"`
}

func compositeBlob(directory, prefix string, data []byte) (string, error) {
	if directory == "" {
		return "", nil
	}
	path := filepath.Join(directory, fmt.Sprintf("%s-%x.idx", prefix, sha256.Sum256(data)))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		old, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(old, data) {
			return "", fmt.Errorf("different existing diagnostic data: %s", path)
		}
		return path, nil
	}
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	return path, closeErr
}

func (t *compositeProbe) observe(step int, eip, esp uint32, mem []byte, snapshot func() (nativeSnapshot, error)) error {
	if p := t.Pending; p != nil && eip == p.Record.Return && uint64(esp) == uint64(p.Record.Stack)+4 {
		r := p.Record
		r.EndStep = step
		profile, source, ok := graphicsSource(mem, t.Profiles)
		if !ok || profile.Name != r.Source || fmt.Sprintf("%x", sha256.Sum256(source)) != r.SourceSHA {
			r.Error = "source changed before natural return"
		} else if snapshot == nil {
			r.Error = "native snapshot unavailable"
		} else {
			s, err := snapshot()
			if err != nil {
				return err
			}
			actual := graphicsCrop(s.Indexed, 640, r.DestRect)
			r.ActualSHA = fmt.Sprintf("%x", sha256.Sum256(actual))
			r.Verified = s.MaxColor == 15 && len(s.Indexed) == 640*400 && bytes.Equal(actual, p.Expected)
			if !r.Verified {
				r.Error = "composite destination differs from expected indices"
			}
			r.AfterPath, err = compositeBlob(t.Directory, "target-after", actual)
			if err != nil {
				return err
			}
		}
		t.Records = append(t.Records, r)
		t.Pending = nil
		return nil
	}
	if eip != 0x1c291 && eip != 0x1c35b {
		return nil
	}
	if t.Pending != nil {
		r := t.Pending.Record
		r.Error = "nested composite entry before observed return"
		t.Records = append(t.Records, r)
		t.Pending = nil
	}
	if len(t.Records) >= 512 {
		t.Dropped++
		return nil
	}
	r := compositeRecord{Entry: eip, Stack: esp, BeginStep: step, Kind: "or_full"}
	reject := func(reason string) error { r.Error = reason; t.Records = append(t.Records, r); return nil }
	stackBytes := uint64(4)
	if eip == 0x1c35b {
		r.Kind, stackBytes = "masked_rect", 20
	}
	if uint64(esp)+stackBytes > uint64(len(mem)) {
		return reject("stack outside RAM")
	}
	r.Return = binary.LittleEndian.Uint32(mem[esp:])
	if uint64(r.Return) >= uint64(len(mem)) {
		return reject("return outside RAM")
	}
	profile, source, ok := graphicsSource(mem, t.Profiles)
	if !ok {
		return reject("unmatched complete RAM source")
	}
	r.Source, r.SourceSHA = profile.Name, fmt.Sprintf("%x", sha256.Sum256(source))
	if r.Kind == "or_full" {
		if profile.Width != 640 || profile.Height != 400 {
			return reject("full OR source geometry")
		}
		r.SourceRect, r.DestRect = image.Rect(0, 0, 640, 400), image.Rect(0, 0, 640, 400)
	} else {
		for i := range r.Args {
			r.Args[i] = binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):])
		}
		a := r.Args
		if a[0] < 1 || a[0] > 40 || a[1] < 1 || a[1] > 400 || a[2] >= 32000 || a[3] >= 32000 {
			return reject("masked rectangle parameters outside bounds")
		}
		sx, sy, dx, dy := int(a[2]%80)*8, int(a[2]/80), int(a[3]%80)*8, int(a[3]/80)
		width, height := int(a[0])*16, int(a[1])
		r.SourceRect, r.DestRect = image.Rect(sx, sy, sx+width, sy+height), image.Rect(dx, dy, dx+width, dy+height)
	}
	selected := graphicsCrop(source, profile.Width, r.SourceRect)
	if selected == nil || !r.DestRect.In(image.Rect(0, 0, 640, 400)) {
		return reject("source or destination rectangle outside geometry")
	}
	if snapshot == nil {
		return reject("native snapshot unavailable")
	}
	s, err := snapshot()
	if err != nil {
		return err
	}
	before := graphicsCrop(s.Indexed, 640, r.DestRect)
	if s.MaxColor != 15 || len(s.Indexed) != 640*400 || len(before) != len(selected) {
		return reject("native indexed display unavailable")
	}
	expected := append([]byte(nil), before...)
	for i, v := range selected {
		if v != 0 {
			r.NonzeroSource++
		}
		if r.Kind == "or_full" {
			expected[i] |= v
		} else if v != 0 {
			expected[i] = v
		}
		if v != 0 && expected[i] != v {
			r.MixedOR++
		}
		if expected[i] != before[i] {
			r.ChangedPixels++
		}
	}
	r.BeforeSHA, r.ExpectedSHA = fmt.Sprintf("%x", sha256.Sum256(before)), fmt.Sprintf("%x", sha256.Sum256(expected))
	r.SourcePath, err = compositeBlob(t.Directory, profile.Name, source)
	if err != nil {
		return err
	}
	r.BeforePath, err = compositeBlob(t.Directory, "target-before", before)
	if err != nil {
		return err
	}
	t.Pending = &compositePending{Record: r, Expected: expected, Selected: append([]byte(nil), selected...)}
	return nil
}
