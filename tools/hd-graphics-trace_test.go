package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"testing"
)

func graphicsFixture() ([]byte, graphicsSourceProfile, []byte) {
	const width, height = 144, 74
	mem := make([]byte, 0x150000)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], width/8)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], height)
	idx := make([]byte, width*height)
	for i := range idx {
		idx[i] = byte((i/width + i%width) % 16)
	}
	p := graphicsSourceProfile{Name: "fixture.OZM", Width: width, Height: height}
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x10000
		binary.LittleEndian.PutUint32(mem[global:], uint32(base))
		packed := make([]byte, width/8*height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				v := ((idx[y*width+x] >> uint(bit)) & 1) << uint(7-x%8)
				mem[base+y*80+x/8] |= v
				packed[y*(width/8)+x/8] |= v
			}
		}
		p.Planes = append(p.Planes, fmt.Sprintf("%x", sha256.Sum256(packed)))
	}
	binary.LittleEndian.PutUint32(mem[0x60000:], 0x12345)
	for i, v := range [5]uint32{0, 18, 74, 80, 0xa5b2a} {
		binary.LittleEndian.PutUint32(mem[0x60004+i*4:], v)
	}
	return mem, p, idx
}

func TestGraphicsOpaqueNaturalReturnReadOnly(t *testing.T) {
	mem, p, idx := graphicsFixture()
	before := sha256.Sum256(mem)
	trace := graphicsTrace{Profiles: []graphicsSourceProfile{p}}
	frame := make([]byte, 640*400)
	for y := 0; y < 74; y++ {
		copy(frame[(291+y)*640+464:], idx[y*144:(y+1)*144])
	}
	calls := 0
	snapshot := func() (nativeSnapshot, error) { calls++; return nativeSnapshot{Indexed: frame}, nil }
	trace.observe(1, 0x1c1bc, 0x60000, mem, snapshot)
	trace.observe(2, 0x12345, 0x60000, mem, snapshot)
	if trace.Pending == nil || calls != 0 {
		t.Fatal("non-natural return accepted")
	}
	trace.observe(3, 0x12345, 0x60004, mem, snapshot)
	if len(trace.Records) != 1 || !trace.Records[0].Verified || trace.Records[0].DestRect != image.Rect(464, 291, 608, 365) || calls != 1 || sha256.Sum256(mem) != before {
		t.Fatal("source, geometry, return or read-only contract", trace.Records)
	}
	trace.observe(4, 0x1c1bc, 0x60000, mem, snapshot)
	frame[291*640+464] ^= 1
	trace.observe(5, 0x12345, 0x60004, mem, snapshot)
	if trace.Records[1].Verified || trace.Records[1].Error == "" {
		t.Fatal("different destination accepted")
	}
	trace.observe(6, 0x1c1bc, 0x60000, mem, snapshot)
	mem[0x100000] ^= 1
	trace.observe(7, 0x12345, 0x60004, mem, snapshot)
	if trace.Records[2].Verified || trace.Records[2].Error != "source changed before natural return" {
		t.Fatal("changed source accepted")
	}
}

func TestGraphicsGeometryBoundsAndEventCap(t *testing.T) {
	for _, args := range [][5]uint32{{0, 18, 74, 79, 0xa5b2a}, {0, 18, 74, 80, 0xa7cff}, {0, 81, 74, 80, 0xa0000}, {0xffffffff, 18, 74, 80, 0xa0000}, {0, 18, 75, 80, 0xa0000}} {
		mem, p, _ := graphicsFixture()
		for i, v := range args {
			binary.LittleEndian.PutUint32(mem[0x60004+i*4:], v)
		}
		trace := graphicsTrace{Profiles: []graphicsSourceProfile{p}}
		trace.observe(1, 0x1c1bc, 0x60000, mem, nil)
		if trace.Pending != nil || len(trace.Records) != 1 || trace.Records[0].Error == "" {
			t.Fatal("invalid geometry accepted", args)
		}
	}
	mem, p, _ := graphicsFixture()
	trace := graphicsTrace{Profiles: []graphicsSourceProfile{p}, Records: make([]graphicsDrawRecord, 512)}
	trace.observe(1, 0x1c1bc, 0x60000, mem, nil)
	if trace.Pending != nil || trace.Dropped != 1 || len(trace.Records) != 512 {
		t.Fatal("event cap not enforced")
	}
	binary.LittleEndian.PutUint32(mem[0x38c00:], 0xffffffff)
	if _, _, ok := graphicsSource(mem, []graphicsSourceProfile{p}); ok {
		t.Fatal("source outside RAM accepted")
	}
}
