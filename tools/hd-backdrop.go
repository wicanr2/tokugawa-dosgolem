package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
)

type spriteBackdropRecord struct {
	Draw                              hdTerrainSpriteDraw `json:"draw"`
	BeginStep, EndStep                int
	Return, Stack, SourceOffset       uint32
	Before, Source, After             *image.NRGBA `json:"-"`
	BeforeIndexed                     []byte       `json:"-"`
	BeforeIndexedSHA                  string
	BeforeSHA, SourceSHA, AfterSHA    string
	SourceIndexedSHA, AfterIndexedSHA string
	Verified                          bool   `json:"primitive_verified"`
	Error                             string `json:"observation_error,omitempty"`
}

type spriteBackdropTrace struct {
	Records []spriteBackdropRecord `json:"records"`
	Pending *spriteBackdropRecord  `json:"-"`
	Dropped uint64                 `json:"dropped_events"`
}

func cropBackdrop(native image.Image, r image.Rectangle) *image.NRGBA {
	if native == nil || native.Bounds() != image.Rect(0, 0, 640, 400) || r.Dx() != 48 || r.Dy() != 48 || !r.In(native.Bounds()) {
		return nil
	}
	im := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			im.SetNRGBA(x, y, rgbaAt(native, r.Min.X+x, r.Min.Y+y))
		}
	}
	return im
}

func cropBackdropIndices(indices []byte, r image.Rectangle) []byte {
	if len(indices) != 640*400 || r.Dx() != 48 || r.Dy() != 48 || !r.In(image.Rect(0, 0, 640, 400)) {
		return nil
	}
	out := make([]byte, 48*48)
	for y := 0; y < 48; y++ {
		copy(out[y*48:(y+1)*48], indices[(r.Min.Y+y)*640+r.Min.X:(r.Min.Y+y)*640+r.Max.X])
	}
	return out
}

func originalSpriteIndices(mem []byte, source uint32) []byte {
	if !hdTerrainSpriteSourceValid(mem) {
		return nil
	}
	var planes [4]uint32
	for i, global := range []int{0x42174, 0x42168, 0x42170, 0x4216c} {
		planes[i] = binary.LittleEndian.Uint32(mem[global:])
	}
	indices := make([]byte, 48*48)
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			off := uint64(source) + uint64(y*80+x/8)
			var index byte
			for bit, base := range planes {
				addr := uint64(base) + off
				if addr >= uint64(len(mem)) {
					return nil
				}
				if mem[addr]&(0x80>>uint(x%8)) != 0 {
					index |= 1 << bit
				}
			}
			indices[y*48+x] = index
		}
	}
	return indices
}

func originalSpriteSource(mem []byte, source uint32, s nativeSnapshot) *image.NRGBA {
	indices := originalSpriteIndices(mem, source)
	if len(indices) != 48*48 || s.MaxColor != 15 {
		return nil
	}
	im := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	for n, index := range indices {
		if index != 0 {
			rgb := s.Palette[s.DAC[index]]
			im.SetNRGBA(n%48, n/48, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
		}
	}
	return im
}

func (p *spriteBackdropRecord) finish(step int, mem []byte, snapshot func() (nativeSnapshot, error)) {
	if p.Before == nil || p.Source == nil || snapshot == nil {
		return
	}
	p.EndStep = step
	s, e := snapshot()
	if e != nil {
		p.Error = e.Error()
	} else {
		p.After = cropBackdrop(s.Image, p.Draw.Rect)
		current := originalSpriteSource(mem, p.SourceOffset, s)
		if p.After == nil || current == nil || !bytes.Equal(current.Pix, p.Source.Pix) {
			p.Error = "return source, palette or geometry changed"
		} else {
			p.AfterSHA = fmt.Sprintf("%x", sha256.Sum256(p.After.Pix))
			expected := image.NewNRGBA(p.Before.Bounds())
			copy(expected.Pix, p.Before.Pix)
			for y := 0; y < 48; y++ {
				for x := 0; x < 48; x++ {
					if c := p.Source.NRGBAAt(x, y); c.A == 255 {
						expected.SetNRGBA(x, y, c)
					}
				}
			}
			p.Verified = bytes.Equal(expected.Pix, p.After.Pix)
			if p.Draw.CombatReturn != 0 {
				indices := originalSpriteIndices(mem, p.SourceOffset)
				afterIndices := cropBackdropIndices(s.Indexed, p.Draw.Rect)
				p.AfterIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(afterIndices))
				if len(indices) != 48*48 || len(p.BeforeIndexed) != 48*48 || len(afterIndices) != 48*48 || fmt.Sprintf("%x", sha256.Sum256(indices)) != p.SourceIndexedSHA {
					p.Verified = false
				} else {
					want := append([]byte(nil), p.BeforeIndexed...)
					for n, index := range indices {
						if index != 0 {
							want[n] = index
						}
					}
					p.Verified = p.Verified && bytes.Equal(want, afterIndices)
				}
			}
			if !p.Verified {
				p.Error = "original binary-alpha primitive mismatch"
			}
		}
	}
}

func beginSpriteBackdrop(step int, esp, esi, edi uint32, mem []byte, snapshot func() (nativeSnapshot, error)) *spriteBackdropRecord {
	p := &spriteBackdropRecord{BeginStep: step, Stack: esp, SourceOffset: esi}
	if uint64(esp)+4 > uint64(len(mem)) {
		p.Error = "stack outside backing RAM"
		return p
	}
	p.Return = binary.LittleEndian.Uint32(mem[esp:])
	var ok bool
	p.Draw, ok = hdTerrainSpriteDecode(esi, edi, p.Return)
	if !ok || !hdTerrainSpriteSourceValid(mem) || snapshot == nil {
		p.Error = "unverified source, flags or call"
		return p
	}
	s, e := snapshot()
	if e != nil {
		p.Error = e.Error()
		return p
	}
	p.Before = cropBackdrop(s.Image, p.Draw.Rect)
	p.BeforeIndexed = cropBackdropIndices(s.Indexed, p.Draw.Rect)
	if len(p.BeforeIndexed) == 48*48 {
		p.BeforeIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(p.BeforeIndexed))
	}
	p.Source = originalSpriteSource(mem, esi, s)
	if p.Before == nil || p.Source == nil {
		p.Error = "source or snapshot unavailable"
		return p
	}
	p.BeforeSHA = fmt.Sprintf("%x", sha256.Sum256(p.Before.Pix))
	p.SourceSHA = fmt.Sprintf("%x", sha256.Sum256(p.Source.Pix))
	if p.Draw.CombatReturn != 0 {
		p.SourceIndexedSHA = fmt.Sprintf("%x", sha256.Sum256(originalSpriteIndices(mem, esi)))
	}
	return p
}

func (t *spriteBackdropTrace) observe(step int, eip, esp, esi, edi uint32, mem []byte, snapshot func() (nativeSnapshot, error)) {
	if p := t.Pending; p != nil && eip == p.Return && uint64(esp) == uint64(p.Stack)+4 {
		p.finish(step, mem, snapshot)
		t.Records = append(t.Records, *p)
		t.Pending = nil
	}
	if eip != 0x1f9d0 {
		return
	}
	if len(t.Records) >= 64 {
		t.Dropped++
		t.Pending = nil
		return
	}
	p := beginSpriteBackdrop(step, esp, esi, edi, mem, snapshot)
	if p.Error != "" {
		t.Records = append(t.Records, *p)
		t.Pending = nil
		return
	}
	t.Pending = p
}

func (t *spriteBackdropTrace) export(root string) ([]map[string]any, error) {
	dir := filepath.Join(root, "sprite-backgrounds")
	if err := makeRuntimeDirectories(dir, 0750); err != nil {
		return nil, err
	}
	var records []map[string]any
	for i, r := range t.Records {
		row := map[string]any{"draw": r.Draw, "begin_step": r.BeginStep, "end_step": r.EndStep, "return_linear": r.Return, "stack_linear": r.Stack, "source_offset": r.SourceOffset, "primitive_verified": r.Verified, "observation_error": r.Error}
		for _, item := range []struct {
			Name  string
			Image *image.NRGBA
		}{{"before", r.Before}, {"source", r.Source}, {"after", r.After}} {
			if item.Image == nil {
				continue
			}
			var b bytes.Buffer
			if err := png.Encode(&b, item.Image); err != nil {
				return nil, err
			}
			name := fmt.Sprintf("record-%02d-%s.png", i, item.Name)
			path := filepath.Join(dir, name)
			if old, err := readRuntimeFile(path); err == nil && !bytes.Equal(old, b.Bytes()) {
				return nil, fmt.Errorf("refuse changed export %s", name)
			}
			if err := writeRuntimeFile(path, b.Bytes(), 0640); err != nil {
				return nil, err
			}
			row[item.Name] = map[string]any{"path": "sprite-backgrounds/" + name, "png_sha256": fmt.Sprintf("%x", sha256.Sum256(b.Bytes())), "rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(item.Image.Pix))}
		}
		records = append(records, row)
	}
	return records, nil
}
