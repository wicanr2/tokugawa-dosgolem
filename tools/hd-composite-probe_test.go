package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCompositeMaskedNaturalReturnAndReadOnly(t *testing.T) {
	mem, profile, idx := graphicsFixture()
	for i, v := range [4]uint32{9, 74, 0, 0x5b2a} {
		binary.LittleEndian.PutUint32(mem[0x60004+i*4:], v)
	}
	ram := sha256.Sum256(mem)
	frame := bytes.Repeat([]byte{10}, 640*400)
	probe := compositeProbe{Profiles: []graphicsSourceProfile{profile}, Directory: t.TempDir()}
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame, MaxColor: 15}, nil }
	if err := probe.observe(1, 0x1c35b, 0x60000, mem, snapshot); err != nil {
		t.Fatal(err)
	}
	probe.observe(2, 0x12345, 0x60000, mem, snapshot)
	if probe.Pending == nil || len(probe.Records) != 0 || sha256.Sum256(mem) != ram {
		t.Fatal("incorrect return or RAM mutation")
	}
	for y := 0; y < 74; y++ {
		for x := 0; x < 144; x++ {
			if v := idx[y*144+x]; v != 0 {
				frame[(291+y)*640+464+x] = v
			}
		}
	}
	if err := probe.observe(3, 0x12345, 0x60004, mem, snapshot); err != nil {
		t.Fatal(err)
	}
	r := probe.Records[0]
	if !r.Verified || r.MixedOR != 0 || sha256.Sum256(mem) != ram {
		t.Fatal("transparent copy or read-only contract", r)
	}
	saved, err := os.ReadFile(r.SourcePath)
	if err != nil || !bytes.Equal(saved, idx) {
		t.Fatal("actual source evidence missing", err)
	}
	probe.observe(4, 0x1c35b, 0x60000, mem, snapshot)
	frame[291*640+464] ^= 1
	probe.observe(5, 0x12345, 0x60004, mem, snapshot)
	if probe.Records[1].Verified || probe.Records[1].Error == "" {
		t.Fatal("destination mutation accepted")
	}
	probe.observe(6, 0x1c35b, 0x60000, mem, snapshot)
	mem[0x100000] ^= 1
	probe.observe(7, 0x12345, 0x60004, mem, snapshot)
	if probe.Records[2].Verified || probe.Records[2].Error != "source changed before natural return" {
		t.Fatal("source mutation accepted")
	}
}

func TestCompositeFullORComplementsAndReadOnly(t *testing.T) {
	mem := make([]byte, 0x200000)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 80)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 400)
	binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
	profile := graphicsSourceProfile{Name: "synthetic-full", Width: 640, Height: 400}
	frame := make([]byte, 640*400)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x20000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		plane := make([]byte, 32000)
		for i := range frame {
			v := byte(i % 16)
			frame[i] = 15 ^ v
			plane[i/8] |= (v >> uint(bit) & 1) << uint(7-i%8)
		}
		copy(mem[base:], plane)
		profile.Planes = append(profile.Planes, fmt.Sprintf("%x", sha256.Sum256(plane)))
	}
	ram := sha256.Sum256(mem)
	probe := compositeProbe{Profiles: []graphicsSourceProfile{profile}}
	snapshot := func() (nativeSnapshot, error) { return nativeSnapshot{Indexed: frame, MaxColor: 15}, nil }
	probe.observe(1, 0x1c291, 0x60000, mem, snapshot)
	frame = bytes.Repeat([]byte{15}, len(frame))
	probe.observe(2, 0x12345, 0x60004, mem, snapshot)
	if len(probe.Records) != 1 || !probe.Records[0].Verified || probe.Records[0].MixedOR != 224000 || sha256.Sum256(mem) != ram {
		t.Fatal("full OR, mixed colors or read-only contract", probe.Records)
	}
}

func TestCompositeBoundsAndEventLimit(t *testing.T) {
	for _, args := range [][4]uint32{{41, 74, 0, 0}, {9, 0, 0, 0}, {9, 75, 0, 0}, {9, 74, 0xffffffff, 0}, {9, 74, 0, 31999}} {
		mem, p, _ := graphicsFixture()
		for i, v := range args {
			binary.LittleEndian.PutUint32(mem[0x60004+i*4:], v)
		}
		probe := compositeProbe{Profiles: []graphicsSourceProfile{p}}
		probe.observe(1, 0x1c35b, 0x60000, mem, nil)
		if probe.Pending != nil || len(probe.Records) != 1 || probe.Records[0].Error == "" {
			t.Fatal("invalid geometry accepted", args)
		}
	}
	mem, p, _ := graphicsFixture()
	probe := compositeProbe{Profiles: []graphicsSourceProfile{p}, Records: make([]compositeRecord, 512)}
	probe.observe(1, 0x1c35b, 0x60000, mem, nil)
	if probe.Pending != nil || len(probe.Records) != 512 || probe.Dropped != 1 {
		t.Fatal("event cap ignored")
	}
	probe = compositeProbe{Profiles: []graphicsSourceProfile{p}}
	probe.observe(1, 0x1c35b, uint32(len(mem)-2), mem, nil)
	if !strings.Contains(probe.Records[0].Error, "stack") {
		t.Fatal("stack bounds ignored")
	}
}
