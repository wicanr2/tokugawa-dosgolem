package main

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func TestTerrainTraceReadOnlyReturnAndLimits(t *testing.T) {
	mem := make([]byte, 0x120000)
	for i, global := range []int{0x42174, 0x42168, 0x42170, 0x4216c} {
		binary.LittleEndian.PutUint32(mem[global:], uint32(0x100000+i*0x8000))
	}
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe2a)
	before := sha256.Sum256(mem)
	var trace terrainTrace
	trace.observe(1, 0x1fc07, 0x1000, 6, 0x1045, mem)
	trace.observe(2, 0x1fe2a, 0x1000, 0, 0, mem)
	if trace.Pending == nil || len(trace.Events) != 1 {
		t.Fatal("wrong stack completed return")
	}
	trace.observe(3, 0x1fe2a, 0x1004, 0, 0, mem)
	if trace.Pending != nil || len(trace.Events) != 2 || before != sha256.Sum256(mem) {
		t.Fatal("natural return or immutable backing memory")
	}
	trace.observe(4, 0x1f9d0, 0x1000, 0x5000, 0x1045, mem)
	trace.observe(5, 0x1fe2a, 0x1004, 0, 0, mem)
	if trace.Events[2]["kind"] != "sprite_enter" || trace.Events[3]["kind"] != "sprite_return" || before != sha256.Sum256(mem) {
		t.Fatal("sprite return or immutable backing memory")
	}
	trace.Events = make([]map[string]any, 512)
	trace.observe(4, 0x1fc07, 0x1000, 6, 0x1045, mem)
	trace.observe(5, 0x1fe2a, 0x1004, 0, 0, mem)
	if len(trace.Events) != 512 || trace.Dropped != 2 || trace.Pending != nil {
		t.Fatal("bounded audit affected return lifecycle")
	}
	trace.observe(6, 0x1fc07, 0xffffffff, 0, 0, mem)
	if trace.Pending != nil {
		t.Fatal("out-of-range stack accepted")
	}
	if terrainPlaneHashes(mem[:100]) != nil {
		t.Fatal("out-of-range globals accepted")
	}
	binary.LittleEndian.PutUint32(mem[0x42174:], 0xa0000)
	planes := terrainPlaneHashes(mem)
	if planes[0]["observation_error"] == nil {
		t.Fatal("VGA aperture accepted as ordinary RAM")
	}
}
