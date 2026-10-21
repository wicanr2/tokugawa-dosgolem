package main

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

func TestCDStreamDeviceClockSurvivesGuestCompletion(t *testing.T) {
	d, pcm, guest := syntheticAudio(t)
	s := newCDStream(d)
	d.Audio.HostEvent = s.command
	if d.playAudio(0, 4, 4) != 0x300 {
		t.Fatal("play")
	}
	// Guest 跑完全部四個 sector，主機尚未讀取任何樣本。
	advanceCD(t, d, 1000000)
	if d.Audio.Playing || !bytes.Equal(guest.Bytes(), pcm[2*2352:]) {
		t.Fatal("guest changed")
	}
	if s.position != 4*588 || s.frames != 0 || !s.playing {
		t.Fatal(s.report())
	}
	got := make([]byte, 4*2352)
	for i := 0; i < len(got); {
		n := min(137, len(got)-i)
		if _, err := io.ReadFull(s, got[i:i+n]); err != nil {
			t.Fatal(err)
		}
		i += n
	}
	if !bytes.Equal(got, pcm[2*2352:]) {
		t.Fatal("host skipped or stretched samples")
	}
	zero := make([]byte, 40)
	io.ReadFull(s, zero)
	if !bytes.Equal(zero, make([]byte, 40)) || s.playing {
		t.Fatal("natural host end")
	}
}

func TestCDStreamPauseResumeReplacementAndEventLimit(t *testing.T) {
	d, pcm, _ := syntheticAudio(t)
	s := newCDStream(d)
	d.Audio.HostEvent = s.command
	// 事件收據已滿仍須收到新命令。
	d.Audio.Events = make([]map[string]any, 256)
	d.playAudio(0, 4, 2)
	a := make([]byte, 7)
	io.ReadFull(s, a)
	if !bytes.Equal(a, pcm[2*2352:2*2352+7]) {
		t.Fatal(a)
	}
	d.stopAudio()
	zero := make([]byte, 101)
	io.ReadFull(s, zero)
	if !bytes.Equal(zero, make([]byte, 101)) {
		t.Fatal("pause")
	}
	d.resumeAudio()
	b := make([]byte, 19)
	io.ReadFull(s, b)
	if !bytes.Equal(b, pcm[2*2352+7:2*2352+26]) {
		t.Fatal("resume changed partial frame")
	}
	d.playAudio(0, 6, 1)
	io.ReadFull(s, b)
	if !bytes.Equal(b, pcm[4*2352:4*2352+19]) {
		t.Fatal("replacement")
	}
	if s.commands != 4 || d.Audio.DroppedEvents < 4 {
		t.Fatal(s.report(), d.Audio.DroppedEvents)
	}
	s.Close()
	if _, err := s.Read(b); err != io.EOF {
		t.Fatal(err)
	}
}

func TestCDStreamPregapAndConcurrentTransport(t *testing.T) {
	d, pcm, _ := syntheticAudio(t)
	s := newCDStream(d)
	s.command("play", 2*588, 6*588)
	got := make([]byte, 4*2352)
	io.ReadFull(s, got)
	if !bytes.Equal(got[:2*2352], make([]byte, 2*2352)) || !bytes.Equal(got[2*2352:], pcm[2*2352:4*2352]) {
		t.Fatal("pregap mapping")
	}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			s.command("play", 4*588, 8*588)
			s.command("pause", 0, 0)
			s.command("resume", 0, 0)
		}
	}()
	go func() {
		defer workers.Done()
		data := make([]byte, 127)
		for i := 0; i < 1000; i++ {
			if _, err := s.Read(data); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	workers.Wait()
}
