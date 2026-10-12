package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestHDTraceReadOnlyLoaderAndPlanes(t *testing.T) {
	mem := make([]byte, 0x80000)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x12345)
	binary.LittleEndian.PutUint32(mem[0x1004:], 0x2000)
	copy(mem[0x2000:], "BATTLE.OZM\x00")
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 2)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 2)
	for _, at := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		binary.LittleEndian.PutUint32(mem[at:], 0x60000)
	}
	copy(mem[0x60000:], []byte{1, 2})
	copy(mem[0x60050:], []byte{3, 4})
	before := append([]byte(nil), mem...)
	var h hdTrace
	h.observe(10, 0x1bf44, 0x1000, mem)
	h.observe(11, 0x12345, 0x0ffc, mem) // 同位址但錯誤的返回堆疊不可誤判。
	if len(h.Events) != 1 || len(h.Pending) != 1 {
		t.Fatal("premature return")
	}
	h.observe(20, 0x12345, 0x1004, mem)
	if !bytes.Equal(mem, before) || len(h.Pending) != 0 || len(h.Events) != 2 {
		t.Fatal("observer changed memory or lost return")
	}
	want := fmt.Sprintf("%x", sha256.Sum256([]byte{1, 2, 3, 4}))
	planes := h.Events[1]["planes"].([]map[string]any)
	if h.Events[0]["name"] != "BATTLE.OZM" || len(planes) != 4 || planes[0]["sha256"] != want {
		t.Fatal("source or pitch mapping")
	}
}

func TestHDTraceBoundsAndEventLimit(t *testing.T) {
	var h hdTrace
	mem := make([]byte, 8)
	before := append([]byte(nil), mem...)
	for i := 0; i < 300; i++ {
		h.observe(i, 0x1bf44, 0xffffffff, mem)
	}
	if !bytes.Equal(before, mem) || len(h.Events) != 256 || h.Dropped != 44 || len(h.Pending) != 0 {
		t.Fatal("unbounded or mutating observer")
	}
	// 返回時尚未有 globals 的小記憶體也必須保持有界。
	h = hdTrace{Pending: []hdPending{{1, 0, "test", 0}}}
	h.observe(1, 1, 4, mem)
	if h.Events[0]["observation_error"] == nil {
		t.Fatal("missing RAM diagnostic")
	}
	mem = make([]byte, 0xc0000)
	binary.LittleEndian.PutUint32(mem[0x5b1b8:], 1)
	binary.LittleEndian.PutUint32(mem[0x5b1b4:], 1)
	binary.LittleEndian.PutUint32(mem[0x38c00:], 0xa0000)
	h = hdTrace{Pending: []hdPending{{1, 0, "test", 0}}}
	h.observe(1, 1, 4, mem)
	if h.Events[0]["planes"].([]map[string]any)[0]["observation_error"] == nil {
		t.Fatal("VGA backing bytes misreported as RAM")
	}
}
