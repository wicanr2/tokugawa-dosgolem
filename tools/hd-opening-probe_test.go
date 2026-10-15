package main

import (
	"crypto/sha256"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestOpeningProbeReadOnlyNaturalReturn(t *testing.T) {
	mem := make([]byte, 2<<20)
	for i := range mem {
		mem[i] = byte(i * 37)
	}
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(mem[0x386d4+i*4:], uint32(0xc0000+i*32000))
	}
	binary.LittleEndian.PutUint32(mem[0x600:], 0x100)
	before := sha256.Sum256(mem)
	probe := openingProbe{Directory: t.TempDir()}
	snapshot := func() (nativeSnapshot, error) {
		return nativeSnapshot{Image: image.NewNRGBA(image.Rect(0, 0, 640, 400)), Indexed: make([]byte, 640*400)}, nil
	}
	load := map[string]any{"name": "OP2.OZM", "step": 12}
	if err := probe.observe(20, 0x11ea4, 0x600, mem, load, snapshot); err != nil {
		t.Fatal(err)
	}
	if len(probe.Pending) != 1 || len(probe.Records) != 0 {
		t.Fatal("entry accepted as completed return")
	}
	if err := probe.observe(21, 0x100, 0x600, mem, load, snapshot); err != nil || len(probe.Records) != 0 {
		t.Fatal("wrong return stack accepted")
	}
	if err := probe.observe(22, 0x100, 0x604, mem, load, snapshot); err != nil {
		t.Fatal(err)
	}
	if len(probe.Pending) != 0 || len(probe.Records) != 1 || probe.Records[0]["end_step"] != 22 || sha256.Sum256(mem) != before {
		t.Fatal("return observation or guest mutation")
	}
	for _, phase := range []string{"before", "after"} {
		path := filepath.Join(probe.Directory, "sample-001-"+phase+".planes.bin")
		b, err := os.ReadFile(path)
		if err != nil || len(b) != 128000 || string(b[:32000]) != string(mem[0xc0000:0xc0000+32000]) {
			t.Fatal("raw candidate planes")
		}
	}
}

func TestOpeningProbeBoundsAndSampling(t *testing.T) {
	probe := openingProbe{Directory: t.TempDir()}
	mem := make([]byte, 0x40000)
	before := sha256.Sum256(mem)
	for n := 0; n < 1050; n++ {
		if err := probe.observe(n, 0x11aeb, 0xffffffff, mem, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(probe.Records) != 9 || len(probe.Pending) != 0 || sha256.Sum256(mem) != before {
		t.Fatal("sample selection, bounds or guest mutation")
	}
	for _, base := range []uint32{0, 0xa0000, 0xfffffff0} {
		binary.LittleEndian.PutUint32(mem[0x386d4:], base)
		capture, err := probe.capture(1, "invalid", mem, nil)
		if err != nil || capture["source_error"] == nil {
			t.Fatal("unsupported candidate pointer accepted")
		}
	}
	probe.Records = make([]map[string]any, 128)
	probe.Counts = nil
	if err := probe.observe(1, 0x11aeb, 0, mem, nil, nil); err != nil || probe.Dropped != 1 || len(probe.Records) != 128 {
		t.Fatal("record limit")
	}
	probe.Records = nil
	probe.Pending = make([]openingProbePending, 32)
	probe.Counts = nil
	if err := probe.observe(1, 0x11aeb, 20, mem, nil, nil); err != nil || probe.Dropped != 2 || len(probe.Pending) != 32 {
		t.Fatal("pending limit")
	}
}
