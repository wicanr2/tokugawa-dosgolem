package main

import (
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/draw"
	"os"
	"testing"
)

func TestBattleFrameSourceNaturalReturnAndUnknownCopy(t *testing.T) {
	indices, err := os.ReadFile("/project/workplace/hd-prototype/catalog/hd-assets/MAPGAMEN.OZM-" + battleFrameIndexedSHA + ".idx")
	if err != nil {
		t.Skip("local source frame unavailable")
	}
	im, err := previewPNG("/project/workplace/hd-prototype/catalog/hd-assets/MAPGAMEN.OZM-" + battleFrameIndexedSHA + ".png")
	if err != nil {
		t.Fatal(err)
	}
	frame := image.NewNRGBA(im.Bounds())
	draw.Draw(frame, frame.Bounds(), im, image.Point{}, draw.Src)
	mem := make([]byte, 0x10000)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x200)
	before := sha256.Sum256(mem)
	planes := []map[string]any{}
	for _, sum := range battleFramePlaneSHA {
		planes = append(planes, map[string]any{"sha256": sum})
	}
	event := map[string]any{"name": "MAPGAMEN.OZM", "raw_columns": uint32(80), "raw_rows": uint32(400), "planes": planes}
	var state battleFrameState
	state.loaded(event)
	if !state.SourceReady {
		t.Fatal("source fingerprint not accepted")
	}
	calls := 0
	get := func() (nativeSnapshot, error) {
		calls++
		return nativeSnapshot{Image: frame, Indexed: indices, MaxColor: 15}, nil
	}
	state.observe(1, 0x1c084, 0x1000, mem, get)
	state.observe(2, 0x200, 0x1000, mem, get)
	if state.Pending == nil || state.Active || calls != 0 {
		t.Fatal("non-natural return consumed frame")
	}
	state.observe(3, 0x200, 0x1004, mem, get)
	if !state.Active || state.Pending != nil || calls != 1 || state.CaptureStep != 3 || before != sha256.Sum256(mem) {
		t.Fatal("full native frame or read-only return")
	}
	if info, err := state.export(t.TempDir()); err != nil || info["indices_sha256"] != battleFrameIndexedSHA {
		t.Fatal("source export", err)
	}
	state.observe(4, 0x1bf44, 0x1000, mem, get)
	if state.SourceReady || !state.Active {
		t.Fatal("loading alone changed original frame eligibility")
	}
	state.observe(5, 0x1c084, 0x1000, mem, get)
	if state.Active || state.Pending != nil {
		t.Fatal("unknown full copy kept old frame active")
	}
	state.loaded(event)
	state.observe(6, 0x1c084, 0x1000, mem, get)
	bad := append([]byte(nil), indices...)
	bad[0] ^= 1
	state.observe(7, 0x200, 0x1004, mem, func() (nativeSnapshot, error) { return nativeSnapshot{Image: frame, Indexed: bad, MaxColor: 15}, nil })
	if state.Active {
		t.Fatal("wrong native full frame accepted")
	}
	planes[0]["sha256"] = "wrong"
	state.loaded(event)
	if state.SourceReady {
		t.Fatal("wrong loaded source accepted")
	}
}
