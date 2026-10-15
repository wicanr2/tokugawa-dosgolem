package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

// DRAFT 只讀觀測，不參與任何 HD 呈現資格。
type openingProbe struct {
	Directory string                `json:"-"`
	Counts    map[string]int        `json:"entry_counts"`
	Records   []map[string]any      `json:"sampled_returns"`
	Pending   []openingProbePending `json:"-"`
	Dropped   int                   `json:"dropped_samples"`
}

type openingProbePending struct {
	Return, Stack uint32
	Record        map[string]any
}

func openingProbeEntry(eip uint32) bool {
	switch eip {
	case 0x11aeb, 0x11ba6, 0x11c16, 0x11c83, 0x11cf3, 0x11d62, 0x11dcb, 0x11e37, 0x11ea4, 0x11f14, 0x11f7a:
		return true
	}
	return false
}

func openingProbeSample(n int) bool {
	switch n {
	case 1, 2, 3, 8, 32, 128, 256, 512, 1024:
		return true
	}
	return false
}

func (p *openingProbe) capture(id int, phase string, mem []byte, snapshot func() (nativeSnapshot, error)) (map[string]any, error) {
	result := map[string]any{}
	if len(mem) < 0x386e4 {
		result["source_error"] = "candidate pointer array outside backing RAM"
		return result, nil
	}
	var packed []byte
	var bases [4]uint32
	for i := range bases {
		base := binary.LittleEndian.Uint32(mem[0x386d4+i*4:])
		bases[i] = base
		end := uint64(base) + 32000
		if base < 0xc0000 || end > uint64(len(mem)) {
			result["source_error"] = "candidate plane outside supported backing RAM"
			result["candidate_bases"] = bases
			return result, nil
		}
		packed = append(packed, mem[base:end]...)
	}
	result["candidate_bases"] = bases
	name := fmt.Sprintf("sample-%03d-%s", id, phase)
	planes := name + ".planes.bin"
	if err := os.WriteFile(filepath.Join(p.Directory, planes), packed, 0600); err != nil {
		return nil, err
	}
	result["candidate_planes"] = map[string]any{"path": planes, "sha256": fmt.Sprintf("%x", sha256.Sum256(packed)), "plane_bytes": 32000, "interpretation": "raw bytes from candidate array; source ownership unverified"}
	if snapshot == nil {
		result["native_error"] = "snapshot unavailable"
		return result, nil
	}
	s, err := snapshot()
	if err != nil {
		result["native_error"] = err.Error()
		return result, nil
	}
	if s.Image == nil || len(s.Indexed) != 640*400 {
		result["native_error"] = "unsupported native snapshot geometry"
		return result, nil
	}
	indexPath := name + ".idx"
	if err := os.WriteFile(filepath.Join(p.Directory, indexPath), s.Indexed, 0600); err != nil {
		return nil, err
	}
	imagePath := name + ".png"
	f, err := os.OpenFile(filepath.Join(p.Directory, imagePath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	encodeErr := png.Encode(f, s.Image)
	closeErr := f.Close()
	if encodeErr != nil {
		return nil, encodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	result["native_indices"] = map[string]any{"path": indexPath, "sha256": fmt.Sprintf("%x", sha256.Sum256(s.Indexed))}
	result["native_png"] = imagePath
	result["palette"], result["DAC"], result["max_color"] = s.Palette, s.DAC, s.MaxColor
	return result, nil
}

func (p *openingProbe) observe(step int, eip, esp uint32, mem []byte, lastLoad map[string]any, snapshot func() (nativeSnapshot, error)) error {
	if len(p.Pending) > 0 {
		pending := p.Pending[len(p.Pending)-1]
		if eip == pending.Return && uint64(esp) == uint64(pending.Stack)+4 {
			p.Pending = p.Pending[:len(p.Pending)-1]
			pending.Record["end_step"] = step
			capture, err := p.capture(pending.Record["sample_id"].(int), "after", mem, snapshot)
			if err != nil {
				return err
			}
			pending.Record["after"] = capture
			p.Records = append(p.Records, pending.Record)
		}
	}
	if !openingProbeEntry(eip) {
		return nil
	}
	if p.Counts == nil {
		p.Counts = map[string]int{}
	}
	name, _ := lastLoad["name"].(string)
	key := fmt.Sprintf("%s:0x%X", name, eip)
	p.Counts[key]++
	if !openingProbeSample(p.Counts[key]) {
		return nil
	}
	if len(p.Records)+len(p.Pending) >= 128 || len(p.Pending) >= 32 {
		p.Dropped++
		return nil
	}
	if uint64(esp)+36 > uint64(len(mem)) {
		p.Records = append(p.Records, map[string]any{"begin_step": step, "entry_linear": eip, "observation_error": "stack outside RAM"})
		return nil
	}
	ret := binary.LittleEndian.Uint32(mem[esp:])
	var args [8]uint32
	for i := range args {
		args[i] = binary.LittleEndian.Uint32(mem[uint64(esp)+4+uint64(i*4):])
	}
	id := len(p.Records) + len(p.Pending) + 1
	record := map[string]any{"sample_id": id, "entry_count": p.Counts[key], "begin_step": step, "entry_linear": eip,
		"return_linear": ret, "stack_linear": esp, "raw_stack_words": args, "last_load_name_hint": name,
		"last_load_step_hint": lastLoad["step"], "classification": "raw read-only observation; source identity and geometry unverified"}
	if uint64(ret) >= uint64(len(mem)) {
		record["observation_error"] = "return outside RAM"
		p.Records = append(p.Records, record)
		return nil
	}
	capture, err := p.capture(id, "before", mem, snapshot)
	if err != nil {
		return err
	}
	record["before"] = capture
	p.Pending = append(p.Pending, openingProbePending{ret, esp, record})
	return nil
}
