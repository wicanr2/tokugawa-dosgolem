package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestBackdropOriginalPrimitiveAndReadOnlyReturn(t *testing.T) {
	mem := terrainMemory(t)
	binary.LittleEndian.PutUint32(mem[0x1000:], 0x1fe69)
	beforeMemory := sha256.Sum256(mem)
	frame := image.NewNRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{80, 60, 90, 255})
		}
	}
	snap := nativeSnapshot{Image: frame, MaxColor: 15}
	for i := 0; i < 16; i++ {
		snap.DAC[i] = byte(i)
		snap.Palette[i] = [3]byte{byte(i * 15), byte(i * 10), byte(i * 5)}
	}
	source := originalSpriteSource(mem, 0x5000, snap)
	if source == nil {
		t.Fatal("original source unavailable")
	}
	var trace spriteBackdropTrace
	calls := 0
	get := func() (nativeSnapshot, error) { calls++; return snap, nil }
	trace.observe(1, 0x1f9d0, 0x1000, 0x5000, 80*100+10, mem, get)
	var state hdTerrainState
	state.observe(1, 0x1f9d0, 0x1000, 0x5000, 80*100+10, mem, func() (nativeSnapshot, error) { return snap, nil })
	if trace.Pending == nil || len(trace.Records) != 0 || calls != 1 {
		t.Fatal("entry capture")
	}
	copyBefore := append([]byte(nil), trace.Pending.Before.Pix...)
	draw.Draw(frame, image.Rect(80, 100, 128, 148), source, image.Point{}, draw.Over)
	trace.observe(2, 0x1fe69, 0x1000, 0, 0, mem, get)
	if trace.Pending == nil || calls != 1 {
		t.Fatal("wrong stack consumed capture")
	}
	trace.observe(3, 0x1fe69, 0x1004, 0, 0, mem, get)
	state.observe(3, 0x1fe69, 0x1004, 0, 0, mem, func() (nativeSnapshot, error) { return snap, nil })
	if trace.Pending != nil || len(trace.Records) != 1 || !trace.Records[0].Verified || calls != 2 || beforeMemory != sha256.Sum256(mem) || !bytes.Equal(copyBefore, trace.Records[0].Before.Pix) {
		t.Fatal("primitive/return/input immutability")
	}
	if len(state.Sprites) != 1 || !state.Sprites[0].BackgroundVerified || state.Sprites[0].BeginStep != 1 || state.Sprites[0].EndStep != 3 || !bytes.Equal(state.Sprites[0].Background.Pix, copyBefore) {
		t.Fatal("production observer lost verified pre-draw background")
	}
	state.observe(4, 0x1f9d0, 0x1000, 0x5000, 80*100+10, mem, func() (nativeSnapshot, error) { return snap, nil })
	frame.SetNRGBA(80, 100, color.NRGBA{255, 254, 253, 255})
	state.observe(5, 0x1fe69, 0x1004, 0, 0, mem, func() (nativeSnapshot, error) { return snap, nil })
	if len(state.Sprites) != 2 || state.Sprites[1].BackgroundVerified || state.Sprites[1].Background != nil {
		t.Fatal("unverified return gained background eligibility")
	}
	exported, err := trace.export(t.TempDir())
	if err != nil || len(exported) != 1 {
		t.Fatal("bounded evidence export", err)
	}
	trace.Records = make([]spriteBackdropRecord, 64)
	trace.observe(4, 0x1f9d0, 0x1000, 0x5000, 0, mem, get)
	if trace.Dropped != 1 || calls != 2 || trace.Pending != nil {
		t.Fatal("diagnostic bound")
	}
}
