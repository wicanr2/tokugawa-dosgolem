package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

func syntheticAudio(t *testing.T) (*cueDisc, []byte, *bytes.Buffer) {
	t.Helper()
	pcm := make([]byte, 6*2352)
	for i := 0; i < len(pcm); i += 4 {
		v := int16(i/4%30000 + 1)
		binary.LittleEndian.PutUint16(pcm[i:], uint16(v))
		binary.LittleEndian.PutUint16(pcm[i+2:], uint16(-v))
	}
	path := filepath.Join(t.TempDir(), "audio.bin")
	if err := os.WriteFile(path, pcm, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	var sink bytes.Buffer
	d := &cueDisc{File: f, RawSectors: 6, LeadOut: 8, Tracks: []cueTrack{
		{Number: 1, Kind: "MODE1/2352", FileSector: 0, DiscSector: 0},
		{Number: 2, Kind: "AUDIO", FileSector: 2, DiscSector: 4, Pregap: 2},
		{Number: 3, Kind: "AUDIO", FileSector: 4, DiscSector: 6},
	}}
	d.Audio.Sink = &sink
	return d, pcm, &sink
}

func advanceCD(t *testing.T, d *cueDisc, us uint64) {
	t.Helper()
	if err := d.advanceAudio(us); err != nil {
		t.Fatal(err)
	}
}

func TestCDDAExactPCMAndChunkIndependentCompletion(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		d, pcm, out := syntheticAudio(t)
		if status := d.playAudio(0, 5, 2); status != 0x300 {
			t.Fatal(status)
		}
		if chunked {
			for us := uint64(1); us < 30000; us += 137 {
				advanceCD(t, d, us)
			}
		}
		advanceCD(t, d, 30000)
		if !bytes.Equal(out.Bytes(), pcm[3*2352:5*2352]) || d.Audio.Frames != 1176 || d.Audio.NonzeroSamples != 2352 || d.Audio.Peak == 0 {
			t.Fatal("PCM mapping or duration differs")
		}
		if d.Audio.Playing || d.Audio.Paused || d.Audio.End != 0 {
			t.Fatal("natural completion state")
		}
		e := d.Audio.Events[len(d.Audio.Events)-1]
		if e["kind"] != "complete" || e["virtual_us"] != uint64(26667) {
			t.Fatalf("completion event %+v", e)
		}
		if d.resumeAudio() != 0x810c {
			t.Fatal("completed audio resumed")
		}
	}
}

func TestCDDAPauseResumeReplaceAndBounds(t *testing.T) {
	d, pcm, out := syntheticAudio(t)
	if d.playAudio(1, 0x000204, 2) != 0x300 {
		t.Fatal("MSF play")
	}
	advanceCD(t, d, 10001)
	if d.stopAudio() != 0x100 || !d.Audio.Paused || d.Audio.Playing {
		t.Fatal("pause")
	}
	status := d.audioStatusInfo()
	if status[1] != 1 || !bytes.Equal(status[7:], redBook(6)) {
		t.Fatal("paused IOCTL status")
	}
	advanceCD(t, d, 1000001)
	if d.Audio.Frames != 441 {
		t.Fatal("paused time advanced playback")
	}
	if d.resumeAudio() != 0x300 {
		t.Fatal("resume")
	}
	advanceCD(t, d, 1006667)
	if d.Audio.Frames != 735 || !bytes.Equal(out.Bytes(), pcm[2*2352:2*2352+735*4]) {
		t.Fatal("resume lost fractional clock or bytes")
	}
	if d.playAudio(0, 6, 1) != 0x300 {
		t.Fatal("replace")
	}
	advanceCD(t, d, 1020001)
	if !bytes.Equal(out.Bytes()[735*4:], pcm[4*2352:5*2352]) {
		t.Fatal("replace did not seek to new source")
	}
	for _, v := range [][3]uint32{{0, 0, 1}, {0, 2, 3}, {0, 7, 2}, {0, 0xffffffff, 2}, {1, 0x02004b, 1}, {2, 4, 1}} {
		if d.playAudio(byte(v[0]), v[1], v[2]) != 0x8108 {
			t.Fatalf("invalid range accepted %v", v)
		}
	}
	if d.playAudio(0, 4, 0) != 0x8000 {
		t.Fatal("zero-length compatibility error")
	}
	if d.playAudio(0, 4, 2) != 0x300 {
		t.Fatal("new play")
	}
	d.stopAudio()
	d.stopAudio()
	if d.resumeAudio() != 0x810c || d.Audio.Paused {
		t.Fatal("second STOP retained resume state")
	}
	if err := d.advanceAudio(0); err == nil {
		t.Fatal("backwards clock accepted")
	}
}

func TestCDDARequestsAndStatus(t *testing.T) {
	d, _, _ := syntheticAudio(t)
	m := &machine.LEMachine{Mem: make([]byte, 4096)}
	c := cpu386.New(m)
	c.Seg[cpu386.SegES] = 8
	c.SetDescriptor(8, cpu386.Descriptor{Limit: 4095, Writable: true})
	c.R[cpu386.EAX], c.R[cpu386.EBX], c.R[cpu386.ECX] = 0x1510, 0x100, 3
	p := m.Mem[0x100:0x11a]
	p[0], p[2] = 22, 0x84
	binary.LittleEndian.PutUint32(p[14:], 4)
	binary.LittleEndian.PutUint32(p[18:], 2)
	if !d.handle(c) || binary.LittleEndian.Uint16(p[3:]) != 0x300 {
		t.Fatal("PLAY request")
	}
	advanceCD(t, d, 1000)
	p[0], p[2] = 13, 0x85
	if !d.handle(c) || binary.LittleEndian.Uint16(p[3:]) != 0x100 {
		t.Fatal("STOP request")
	}
	p[0], p[2] = 26, 3
	binary.LittleEndian.PutUint32(p[14:], 0x200)
	binary.LittleEndian.PutUint16(p[18:], 11)
	m.Mem[0x200] = 15
	if !d.handle(c) || m.Mem[0x201] != 1 {
		t.Fatal("audio status request")
	}
	p[0], p[2] = 13, 0x88
	if !d.handle(c) || binary.LittleEndian.Uint16(p[3:]) != 0x300 {
		t.Fatal("RESUME request")
	}
}

func TestCDDAOriginalBINPCM(t *testing.T) {
	if _, err := os.Stat("/original/DX5.cue"); os.IsNotExist(err) {
		t.Skip("original disc not mounted")
	}
	d, err := openCueDisc("/original/DX5.cue")
	if err != nil {
		t.Fatal(err)
	}
	defer d.File.Close()
	var out bytes.Buffer
	d.Audio.Sink = &out
	track := d.Tracks[9]
	if d.playAudio(0, track.DiscSector+750, 75) != 0x300 {
		t.Fatal("original track range")
	}
	advanceCD(t, d, 1000000)
	want := make([]byte, 44100*4)
	if _, err := d.File.ReadAt(want, int64(track.FileSector+750)*2352); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) || d.Audio.Frames != 44100 || d.Audio.NonzeroSamples == 0 || d.Audio.Playing {
		t.Fatal("original PCM sample mismatch or silence")
	}
}
