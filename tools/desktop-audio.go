//go:build desktop

package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type desktopPCMPlayer struct {
	stream *cdStream
	player *audio.Player
	paused bool
}

func init() {
	newDesktopPCMOutput = func(disc *cueDisc) (desktopPCMOutput, error) {
		stream := newCDStream(disc)
		ctx := audio.NewContext(int(cdSampleRate))
		player, err := ctx.NewPlayer(stream)
		if err != nil {
			stream.Close()
			return nil, err
		}
		player.SetBufferSize(100 * time.Millisecond)
		player.Play()
		disc.Audio.HostEvent = stream.command
		return &desktopPCMPlayer{stream: stream, player: player}, nil
	}
}

func (p *desktopPCMPlayer) Close() error { err := p.player.Close(); p.stream.Close(); return err }
func (p *desktopPCMPlayer) SetPaused(paused bool) {
	if p.paused == paused {
		return
	}
	p.paused = paused
	if paused {
		p.player.Pause()
	} else {
		p.player.Play()
	}
}
func (p *desktopPCMPlayer) Report() map[string]any {
	r := p.stream.report()
	r["enabled"] = true
	r["sample_rate"] = 44100
	r["format"] = "s16le stereo"
	r["paused"], r["playing"], r["device_position_ns"] = p.paused, p.player.IsPlaying(), p.player.Position().Nanoseconds()
	r["classification"] = "device-clock playback; original guest virtual clock and PCM unchanged"
	r["listening_verified"] = false
	return r
}
