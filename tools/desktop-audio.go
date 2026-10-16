//go:build desktop

package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type desktopPCMPlayer struct {
	queue  *pcmQueue
	player *audio.Player
	paused bool
}

func init() {
	newDesktopPCMOutput = func() (desktopPCMOutput, error) {
		queue := newPCMQueue(int(cdSampleRate) / 2)
		ctx := audio.NewContext(int(cdSampleRate))
		player, err := ctx.NewPlayer(queue)
		if err != nil {
			queue.Close()
			return nil, err
		}
		player.SetBufferSize(100 * time.Millisecond)
		player.Play()
		return &desktopPCMPlayer{queue: queue, player: player}, nil
	}
}

func (p *desktopPCMPlayer) Write(data []byte) (int, error) { return p.queue.Write(data) }
func (p *desktopPCMPlayer) Close() error                   { err := p.player.Close(); p.queue.Close(); return err }
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
	r := p.queue.report()
	r["enabled"] = true
	r["sample_rate"] = 44100
	r["format"] = "s16le stereo"
	r["paused"], r["playing"], r["device_position_ns"] = p.paused, p.player.IsPlaying(), p.player.Position().Nanoseconds()
	r["classification"] = "host playback approximation; original virtual clock and PCM unchanged"
	r["listening_verified"] = false
	return r
}
