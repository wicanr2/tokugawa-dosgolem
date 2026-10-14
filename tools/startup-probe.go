// 有界 LE 啟動研究探針。只提供平台 ABI，不跳過遊戲或 C runtime 指令。
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
)

type rawTrace struct {
	step      int
	eip       uint32
	bytes     [16]byte
	size      int
	registers [8]uint32
	segments  [6]uint16
	flags     uint32
}

func main() {
	budget := flag.Int("steps", 3000000, "successful-instruction budget")
	mouseScript := flag.String("mouse", "[]", "JSON array of mouse states at successful-instruction boundaries")
	mouseFile := flag.String("mouse-file", "", "read mouse state array from a file")
	cdDiagnostic := flag.Bool("cd-trace", false, "record bounded read-only DPMI/CD request diagnostics")
	hdDiagnostic := flag.Bool("hd-trace", false, "record bounded read-only original asset loader diagnostics")
	terrainDiagnostic := flag.Bool("hd-terrain-trace", false, "record bounded read-only original floor/sprite draw registers and returns")
	backdropDiagnostic := flag.Bool("hd-backdrop-trace", false, "capture bounded read-only native sprite backgrounds and verify original binary-alpha primitive")
	battleFrameDiagnostic := flag.Bool("hd-battle-frame-trace", false, "verify MAPGAMEN full native draw at its natural return; no HD replacement")
	hdExport := flag.Bool("hd-export", false, "copy validated loader-return planes and DAC to local research output; implies hd-trace")
	desktop := flag.Bool("desktop", false, "run isolated desktop presentation prototype")
	flag.Parse()
	if *hdExport {
		*hdDiagnostic = true
	}
	var assetTrace hdTrace
	var floorTrace terrainTrace
	var backdropTrace spriteBackdropTrace
	var live hdLive
	var desktopReceipt map[string]any
	if *desktop && runDesktop == nil {
		panic("desktop build tag is required")
	}
	if *budget < 1 || *budget > 750000000 {
		panic("instruction budget out of range")
	}
	inputSource := "inline"
	if *mouseFile != "" {
		if *mouseScript != "[]" {
			panic("mouse and mouse-file are mutually exclusive")
		}
		b, err := os.ReadFile(*mouseFile)
		if err != nil {
			panic(err)
		}
		*mouseScript = string(b)
		inputSource = *mouseFile
	}
	mouseEvents, err := parseMouseEvents(*mouseScript, *budget)
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile("/original/DX5.EXE")
	if err != nil {
		panic(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if digest != "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a" {
		panic("DX5.EXE hash mismatch")
	}
	m, err := machine.LoadLE(data)
	if err != nil {
		panic(err)
	}
	// PSP 與環境放在 LE objects 之前的空白區；selector 是主機標籤。
	for _, value := range m.Mem[0x8000:0x8202] {
		if value != 0 {
			panic("PSP region occupied")
		}
	}
	m.CPU.SetDescriptor(0x28, cpu386.Descriptor{Base: 0x8000, Limit: 255, Writable: true})
	environment := []byte("\x00\x00\x01\x00C:\\DX5.EXE\x00")
	m.CPU.SetDescriptor(0x30, cpu386.Descriptor{Base: 0x8100, Limit: uint32(len(environment) - 1)})
	copy(m.Mem[0x8100:], environment)
	m.Mem[0x802c] = 0x30
	m.Mem[0x8080] = 0
	m.Mem[0x8081] = 13
	m.CPU.Seg[cpu386.SegES] = 0x28
	fileState, err := os.MkdirTemp("/output/startup-probe", "state-")
	if err != nil {
		panic(err)
	}
	files, err := machine.OpenDirectoryOverlayFiles("/original", fileState)
	if err != nil {
		panic(err)
	}
	defer files.Close()
	services := machine.NewFD2StartupDOS(files)
	services.AttachMachine(m)
	defer services.Close()
	if err := services.DPMI.EnableDOSMCBMemory(0x0800); err != nil {
		panic(err)
	}
	disc, err := openCueDisc("/original/DX5.cue")
	if err != nil {
		panic(err)
	}
	defer disc.File.Close()
	audioFile, err := os.CreateTemp("/output/startup-probe", "cdda-*.pcm")
	if err != nil {
		panic(err)
	}
	defer audioFile.Close()
	audioHash := sha256.New()
	audioWriter := bufio.NewWriterSize(io.MultiWriter(audioFile, audioHash), 64*1024)
	disc.Audio.Sink = audioWriter
	screen := newBankVideo()
	m.CPU.Bus = videoBus{m, screen}
	mouseMachine := machine.New()
	services.DPMI.RealModeInterrupt = func(r *cpu.CPU, n uint8) bool {
		if n == 0x21 && r.R[cpu.AX] == 0x6300 {
			r.R[cpu.AX] &= 0xff00
			r.Seg[cpu.DS] = 0x820
			r.R[cpu.SI] = 0
			r.SetFlags(r.Flags &^ cpu.CF)
			return true
		}
		if paletteRealMode(r, n, mouseMachine) {
			return true
		}
		if screen.realMode(r, n) {
			return true
		}
		return disc.realMode(r, n)
	}
	screen.Legacy = mouseMachine
	copy(m.Mem[0x400:0x500], mouseMachine.Mem[0x400:0x500])
	if !machine.InstallLEBIOSKeyboard(m) {
		panic("BIOS keyboard installation failed")
	}
	mouseServices := dos.New(mouseMachine, "")
	mouseServices.Install()
	defer mouseServices.Close()
	video := videoPorts{m: mouseMachine}
	oldPortOut := m.CPU.PortOut
	m.CPU.PortOut = func(p uint16, b byte) bool {
		if video.output(p, b) {
			oldPortOut(p, b)
			return true
		}
		return oldPortOut(p, b)
	}
	m.CPU.PortIn = video.input
	m.CPU.PortOut16 = func(p uint16, value uint16) bool {
		if !video.outputWord(p, value) {
			return false
		}
		oldPortOut(p, byte(value))
		oldPortOut(p+1, byte(value>>8))
		return true
	}
	if disc.SHA256 != "3f511d4c2fb819f950d881697f4e771d37ccfa2b0fd9eca6c4bb48fff6be2448" {
		panic("disc hash mismatch")
	}
	interrupts := []map[string]any{}
	counts := map[string]int{}
	steps := 0
	cursor := mouseCursor{DisplayCount: -1}
	traceRing := make([]rawTrace, 1024)
	traceCount := 0
	cdTrace := []map[string]any{}
	cdTraceDropped := 0
	m.CPU.IntHook = func(c *cpu386.CPU, number uint8) bool {
		if number == 0x2f || number == 0x31 && uint16(c.R[cpu386.EAX]) == 0x0300 && byte(c.R[cpu386.EBX]) == 0x2f {
			if err := disc.advanceAudio(uint64(steps)); err != nil {
				panic(err)
			}
		}
		before := c.R
		beforeFlags := c.EFlags
		function := uint16(c.R[cpu386.EAX])
		var cdEvent map[string]any
		if *cdDiagnostic && number == 0x31 && (function == 0x0100 || function == 0x0300 && byte(c.R[cpu386.EBX]) == 0x2f) {
			if len(cdTrace) < 256 {
				context := []map[string]any{}
				first := traceCount - 64
				if first < 0 {
					first = 0
				}
				for i := first; i < traceCount; i++ {
					r := traceRing[i%len(traceRing)]
					context = append(context, map[string]any{"step": r.step, "linear_eip": fmt.Sprintf("0x%X", r.eip), "bytes": hex.EncodeToString(r.bytes[:r.size]), "registers": r.registers, "segments": r.segments, "flags": r.flags})
				}
				cdEvent = map[string]any{"step": steps, "function_ax": fmt.Sprintf("0x%04X", function), "before": captureCDDiagnostic(c, m, services.DPMI), "instruction_context": context}
			} else {
				cdTraceDropped++
			}
		}
		handled := false
		switch {
		case number == 0x10 && byte(function>>8) == 0:
			handled = setVideoBIOS(c, m, screen, &video)
		case number == 0x10 && byte(function>>8) == 0x10:
			handled = paletteBIOS(c, mouseMachine)
		case number == 0x21 && function == 0x4b00:
			handled = executeAbsentFile(c, files)
		case number == 0x16 || number == 0x21 && byte(function>>8) == 0x0b:
			handled = keyboardInterrupt(c, m, number)
		case number == 0x10 && byte(function>>8) >= 1 && byte(function>>8) <= 3:
			handled = cursorBIOS(c, m)
		case number == 0x10 && function>>8 == 0x4f:
			handled = screen.handle(c)
		case number == 0x33:
			handled = mouseInterrupt(c, mouseMachine, &cursor)
		case number == 0x10 && (uint8(function>>8) == 0x0f || function == 0x1a00 || function == 0x6f00 || function == 0x5f00 || (uint8(function>>8) == 0x12 && uint8(c.R[cpu386.EBX]) == 0x10)):
			copy(mouseMachine.Mem[0x400:0x500], m.Mem[0x400:0x500])
			handled = legacyQuery(c, mouseMachine, 0x10)
		case number == 0x2f:
			handled = disc.handle(c)
		case number == 0x21 && function == 0x3700:
			c.R[cpu386.EAX] &^= 0xff
			c.R[cpu386.EDX] = c.R[cpu386.EDX]&^0xff | '/'
			c.EFlags &^= cpu386.CF
			handled = true
		case number == 0x21 && uint8(function>>8) == 0x30:
			// 固定測試主機的 DOS 6.22；高字不宣告 Phar Lap。
			c.R[cpu386.EAX] = 0x1606
			c.R[cpu386.EBX] = 0
			c.R[cpu386.ECX] = 0
			c.EFlags &^= cpu386.CF
			handled = true
		case number == 0x21 && function == 0xff00 && uint16(c.R[cpu386.EDX]) == 0x78:
			// Watcom 公開啟動契約：AL 非零表示 DOS/4G 相容主機。
			c.R[cpu386.EAX] = 0xffff
			c.EFlags &^= cpu386.CF
			handled = true
		case number == 0x31:
			handled = services.DPMI.Handle(c)
		case number == 0x21 && uint8(function>>8) == 0x2c:
			centiseconds := uint32(steps / 10000)
			c.R[cpu386.ECX] = c.R[cpu386.ECX]&0xffff0000 | ((centiseconds/360000)%24)<<8 | (centiseconds/6000)%60
			c.R[cpu386.EDX] = c.R[cpu386.EDX]&0xffff0000 | ((centiseconds/100)%60)<<8 | centiseconds%100
			handled = true
		case number == 0x21:
			// 只轉接既有通用 DOS 服務，禁止進入 FD2 啟動狀態機。
			switch uint8(function >> 8) {
			case 0x00, 0x02, 0x09, 0x19, 0x25, 0x35, 0x3d, 0x3e, 0x3f, 0x40, 0x42, 0x44, 0x4c:
				handled = services.Handle(c, number)
			}
		}
		key := fmt.Sprintf("%02X:%04X", number, function)
		counts[key]++
		interrupts = append(interrupts, map[string]any{"number": fmt.Sprintf("0x%02X", number), "function_ax": fmt.Sprintf("0x%04X", function), "linear_eip_after_int": fmt.Sprintf("0x%X", c.EIP), "before": before, "after": c.R, "flags_before": beforeFlags, "flags_after": c.EFlags, "handled": handled})
		if len(interrupts) > 256 {
			interrupts = interrupts[1:]
		}
		if cdEvent != nil {
			cdEvent["after"] = captureCDDiagnostic(c, m, services.DPMI)
			cdEvent["handled"] = handled
			cdTrace = append(cdTrace, cdEvent)
		}
		return handled
	}
	stop := "instruction_budget"
	var before uint32
	inputEvents := []map[string]any{}
	inputFrames := []map[string]any{}
	nextMouse := 0
	advance := func(maxSteps int) bool {
		end := steps + maxSteps
		for steps < *budget && steps < end {
			if steps&1023 == 0 {
				if err := disc.advanceAudio(uint64(steps)); err != nil {
					panic(err)
				}
			}
			video.advance(uint64(steps))
			if nextMouse < len(mouseEvents) && mouseEvents[nextMouse].Step == steps {
				e := mouseEvents[nextMouse]
				if e.Buttons == 0 && mouseServices.Mouse.Buttons == 0 {
					f, err := captureFrame(screen, &cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
					if err != nil {
						panic(err)
					}
					inputFrames = append(inputFrames, map[string]any{"step": steps, "frame": f})
				}
				applyMouseEvent(mouseServices, e)
				inputEvents = append(inputEvents, map[string]any{"scheduled_step": e.Step, "actual_step": steps, "requested_x": e.X, "requested_y": e.Y, "x": mouseServices.Mouse.X, "y": mouseServices.Mouse.Y, "buttons": mouseServices.Mouse.Buttons})
				nextMouse++
			}
			before = m.CPU.EIP
			if *hdDiagnostic {
				previous := len(assetTrace.Events)
				assetTrace.observe(steps, before, m.CPU.R[cpu386.ESP], m.Mem)
				if *hdExport && len(assetTrace.Events) > previous {
					event := assetTrace.Events[len(assetTrace.Events)-1]
					if event["kind"] == "load_return" {
						var palette [16]color.NRGBA
						dac := screen.Legacy.Palette()
						for i := range palette {
							rgb := dac[screen.Legacy.VGA.DACIndex(byte(i))]
							palette[i] = color.NRGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}
						}
						if err := captureLoadedAsset(event, m.Mem, palette); err != nil {
							panic(err)
						}
					}
				}
			}
			if *desktop {
				live.Terrain.observe(steps, before, m.CPU.R[cpu386.ESP], m.CPU.R[cpu386.ESI], m.CPU.R[cpu386.EDI], m.Mem, func() (nativeSnapshot, error) {
					return snapshotNative(screen, cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
				})
			}
			if *desktop || *battleFrameDiagnostic {
				live.observe(steps, before, m.CPU.R[cpu386.ESP], m.Mem, func() (nativeSnapshot, error) {
					return snapshotNative(screen, cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
				})
			}
			if *terrainDiagnostic {
				floorTrace.observe(steps, before, m.CPU.R[cpu386.ESP], m.CPU.R[cpu386.ESI], m.CPU.R[cpu386.EDI], m.Mem)
			}
			if *backdropDiagnostic {
				backdropTrace.observe(steps, before, m.CPU.R[cpu386.ESP], m.CPU.R[cpu386.ESI], m.CPU.R[cpu386.EDI], m.Mem, func() (nativeSnapshot, error) {
					return snapshotNative(screen, cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
				})
			}
			row := rawTrace{step: steps, eip: before, registers: m.CPU.R, segments: m.CPU.Seg, flags: m.CPU.EFlags}
			for offset := uint32(0); offset < 16; offset++ {
				b, e := m.CPU.Bus.Read8(before + offset)
				if e != nil {
					break
				}
				row.bytes[row.size] = b
				row.size++
			}
			traceRing[traceCount%len(traceRing)] = row
			traceCount++
			if err := m.CPU.Step(); err != nil {
				stop = err.Error()
				if m.Keyboard.Waiting {
					stop = "waiting_for_keyboard"
				}
				break
			}
			steps++
			if services.Exited {
				stop = "dos_exit"
				break
			}
		}
		return steps < *budget && stop == "instruction_budget"
	}
	if *desktop {
		ctx := &desktopContext{
			Advance: advance, Steps: func() int { return steps }, Live: &live, Scripted: len(mouseEvents) > 0,
			Snapshot: func() (nativeSnapshot, error) {
				return snapshotNative(screen, cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
			},
			Mouse: func(x, y, buttons int) {
				e := scheduledMouse{Step: steps, X: x, Y: y, Buttons: buttons}
				applyMouseEvent(mouseServices, e)
				if len(inputEvents) < 1024 {
					inputEvents = append(inputEvents, map[string]any{"source": "desktop", "actual_step": steps, "requested_x": x, "requested_y": y, "x": mouseServices.Mouse.X, "y": mouseServices.Mouse.Y, "buttons": buttons})
				}
			},
			Proof: func() map[string]any {
				s, err := snapshotNative(screen, cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
				if err != nil {
					panic(err)
				}
				return map[string]any{"step": steps, "ram_sha256": fmt.Sprintf("%x", sha256.Sum256(m.Mem)), "eip": m.CPU.EIP, "registers": m.CPU.R, "segments": m.CPU.Seg, "flags": m.CPU.EFlags, "raw_indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(s.Indexed)), "palette": s.Palette, "cursor": cursor, "mouse_x": s.X, "mouse_y": s.Y}
			},
		}
		if err := runDesktop(ctx); err != nil {
			panic(err)
		}
		desktopReceipt = ctx.Report
		if steps < *budget && stop == "instruction_budget" {
			stop = "desktop_closed"
		}
	} else {
		advance(*budget)
	}
	if err := disc.advanceAudio(uint64(steps)); err != nil {
		panic(err)
	}
	if err := audioWriter.Flush(); err != nil {
		panic(err)
	}
	if err := audioFile.Close(); err != nil {
		panic(err)
	}
	audioDigest := fmt.Sprintf("%x", audioHash.Sum(nil))
	audioPath := "/output/startup-probe/audio-" + audioDigest + ".pcm"
	if err := os.Rename(audioFile.Name(), audioPath); err != nil {
		panic(err)
	}
	trace := []map[string]any{}
	first := traceCount - len(traceRing)
	if first < 0 {
		first = 0
	}
	for i := first; i < traceCount; i++ {
		r := traceRing[i%len(traceRing)]
		trace = append(trace, map[string]any{"step": r.step, "linear_eip": fmt.Sprintf("0x%X", r.eip), "bytes": hex.EncodeToString(r.bytes[:r.size]), "registers": r.registers, "segments": r.segments, "flags": r.flags})
	}
	script, err := os.ReadFile("/project/tools/startup-probe.go")
	if err != nil {
		panic(err)
	}
	patch, err := os.ReadFile("/project/tools/dosgolem.patch")
	if err != nil {
		panic(err)
	}
	cpuSource, err := os.ReadFile("/output/engine/internal/cpu386/cpu.go")
	if err != nil {
		panic(err)
	}
	toolHashes := map[string]string{}
	for _, name := range []string{"tools/hd-battle-frame.go", "tools/hd-battle-frame_test.go", "tools/hd-battle-frame-render.go", "tools/hd-battle-frame-render_test.go", "tools/hd-battle-frame-profile.go"} {
		b, e := os.ReadFile("/project/" + name)
		if e != nil {
			panic(e)
		}
		toolHashes[name] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	for _, name := range []string{"tools/hd-terrain-scene.go", "tools/hd-terrain-scene_test.go", "tools/hd-terrain-unit-profiles.go"} {
		b, e := os.ReadFile("/project/" + name)
		if e != nil {
			panic(e)
		}
		toolHashes[name] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	for _, name := range []string{"tools/startup-probe.go", "tools/cdrom.go", "tools/cdrom_test.go", "tools/cdda.go", "tools/cdda_test.go", "tools/hd-capture.go", "tools/hd-trace.go", "tools/hd-trace_test.go", "tools/hd-terrain-trace.go", "tools/hd-terrain-trace_test.go", "tools/hd-terrain.go", "tools/hd-terrain-profiles.go", "tools/hd-terrain_test.go", "tools/hd-story_test.go", "tools/hd-portraits.go", "tools/hd-kao.go", "tools/hd-kao_test.go", "tools/hd-kao-profiles.go", "tools/hd-live.go", "tools/hd-live_test.go", "tools/hd-layer.go", "tools/hd-scroll.go", "tools/hd-scroll_test.go", "tools/hd-frame.go", "tools/hd-frame_test.go", "tools/hd-preview_test.go", "tools/presentation.go", "tools/presentation_test.go", "tools/desktop.go", "tools/windowbar.go", "tools/windowbar_test.go", "tools/desktop-windowbar.go", "tools/toolbar-font.hex.gz", "tools/LICENSE-toolbar-font.txt", "tools/run-desktop.sh", "tools/desktop.mod", "tools/desktop.sum", "tools/video.go", "tools/video_test.go", "tools/vbe.go", "tools/vbe_test.go", "tools/keyboard.go", "tools/keyboard_test.go", "tools/dos_test.go", "tools/dosgolem.patch", "tools/dpmi-mcb.go", "tools/dpmi-mcb_test.go", "tools/cpu386-integration_test.go", "tools/run-probes.sh"} {
		b, e := os.ReadFile("/project/" + name)
		if e != nil {
			panic(e)
		}
		toolHashes[name] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	for _, name := range []string{"tools/hd-window.go", "tools/hd-window_test.go", "tools/hd-story_test.go", "tools/hd-terrain-sprite-profiles.go", "tools/hd-terrain-sprites.go", "tools/hd-terrain-sprites_test.go", "tools/hd-terrain-tree-profile.go", "tools/hd-backdrop.go", "tools/hd-backdrop_test.go"} {
		b, e := os.ReadFile("/project/" + name)
		if e != nil {
			panic(e)
		}
		toolHashes[name] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	frame, err := captureFrame(screen, &cursor, int(mouseServices.Mouse.X), int(mouseServices.Mouse.Y))
	if err != nil {
		panic(err)
	}
	var backdropRecords []map[string]any
	var battleFrameCapture map[string]any
	if *battleFrameDiagnostic {
		battleFrameCapture, err = live.BattleFrame.export("/output")
		if err != nil {
			panic(err)
		}
	}
	if *backdropDiagnostic {
		backdropRecords, err = backdropTrace.export("/output")
		if err != nil {
			panic(err)
		}
	}
	receipt := map[string]any{
		"dosgolem_head": "a9714ebdab2ad6b529f81225472680f2b11f2842", "integration_input_sha256": toolHashes,
		"input_file": "DX5.EXE", "input_sha256": digest,
		"file_state_directory": fileState, "original_read_only": true,
		"dos_memory_model": "explicit 32-bit DPMI MCB arena; first fit, reusable; owner PSP 0800h",
		"tool":             "dosgolem native LE + bounded DOS4G-compatible startup research probe", "go_version": runtime.Version(),
		"probe_sha256": fmt.Sprintf("%x", sha256.Sum256(script)), "address_space": "dosgolem relocated flat linear memory; selectors are host labels",
		"patch_sha256": fmt.Sprintf("%x", sha256.Sum256(patch)), "cpu_source_sha256": fmt.Sprintf("%x", sha256.Sum256(cpuSource)),
		"successful_steps": steps, "instruction_budget": *budget, "stop_instruction_linear_eip": fmt.Sprintf("0x%X", before), "stop": stop,
		"interrupts_tail": interrupts, "interrupt_counts": counts, "trace_tail": trace,
		"dpmi_calls": services.DPMI.Calls, "dpmi_unimplemented": services.DPMI.Unimplemented,
		"real_mode_history":     services.DPMI.RealModeHistory,
		"real_mode_last":        services.DPMI.RealModeLast,
		"cd_diagnostic_enabled": *cdDiagnostic, "cd_diagnostic_events": cdTrace, "cd_diagnostic_dropped": cdTraceDropped,
		"hd_diagnostic_enabled": *hdDiagnostic, "hd_asset_trace": assetTrace, "hd_export_enabled": *hdExport,
		"terrain_diagnostic_enabled": *terrainDiagnostic, "terrain_draw_trace": floorTrace,
		"sprite_backdrop_diagnostic_enabled": *backdropDiagnostic, "sprite_backdrop_records": backdropRecords, "sprite_backdrop_dropped": backdropTrace.Dropped,
		"battle_frame_diagnostic_enabled": *battleFrameDiagnostic, "battle_frame_observation": live.BattleFrame, "battle_frame_capture": battleFrameCapture,
		"desktop":            desktopReceipt,
		"mouse_calls":        mouseServices.Mouse.Calls,
		"mouse_cursor":       cursor,
		"mouse_input_events": inputEvents, "mouse_input_schedule": mouseEvents,
		"mouse_input_source": inputSource, "mouse_input_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(*mouseScript))),
		"input_boundary_frames": inputFrames,
		"console":               string(services.Console), "exit_code": services.ExitCode, "memory_bytes": len(m.Mem),
		"fd2_startup_calls": services.Calls(), "native_runtime_instructions_skipped": false,
		"virtual_clock": map[string]any{"initial_time": "00:00:00.00", "microseconds_per_successful_instruction": 1, "elapsed_microseconds": steps, "classification": "hardware-spec approximation"},
		"mounted_disc":  disc, "cd_supported_functions": []string{"1500", "150C", "150D"}, "cd_audio_verified": false,
		"cd_supported_requests":       []string{"1510/IOCTL INPUT/10", "1510/IOCTL INPUT/11", "1510/IOCTL INPUT/15", "1510/PLAY AUDIO", "1510/STOP AUDIO", "1510/RESUME AUDIO"},
		"cd_pcm":                      map[string]any{"path": audioPath, "sha256": audioDigest, "sample_rate": 44100, "channels": 2, "format": "s16le", "stereo_frames": disc.Audio.Frames, "nonzero_channel_samples": disc.Audio.NonzeroSamples, "absolute_sample_peak": disc.Audio.Peak, "idle_intervals": "omitted from PCM; reconstruct from virtual_us events", "timing": "hardware-spec approximation", "listening_verified": false},
		"test_environment":            map[string]any{"dos_version": "6.22", "command_line": "", "program_name": "C:\\DX5.EXE", "dos4g_detection": "AX=FFFFh, GS=0; compatible host, not original extender emulation"},
		"normal_player_path_verified": false, "game_screen_verified": false,
		"vbe_mode": screen.Mode, "vbe_width": screen.Width, "vbe_height": screen.Height, "vbe_bank": screen.Bank,
		"frame": frame, "vga_mode": mouseMachine.VideoMode(),
		"keyboard_reads": m.Keyboard.Reads, "keyboard_enqueued": m.Keyboard.Enqueued, "keyboard_waiting": m.Keyboard.Waiting,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		panic(err)
	}
	if err := os.WriteFile("/output/startup-memory.bin", m.Mem, 0600); err != nil {
		panic(err)
	}
}

// Read backing RAM only. Reading through the device bus could alter device latches.
func captureCDDiagnostic(c *cpu386.CPU, m *machine.LEMachine, h *machine.DPMIHost) map[string]any {
	read := func(address uint64, size uint64) map[string]any {
		out := map[string]any{"linear_address": fmt.Sprintf("0x%X", address), "requested_bytes": size}
		if size > 128 {
			size = 128
		}
		if address > uint64(len(m.Mem)) || size > uint64(len(m.Mem))-address {
			out["error"] = "outside backing RAM"
			return out
		}
		out["bytes"] = hex.EncodeToString(m.Mem[int(address):int(address+size)])
		return out
	}
	blocks := h.DOSMemory()
	buffers := []map[string]any{}
	for _, block := range blocks {
		buffers = append(buffers, map[string]any{"block": block, "allocated_memory": read(uint64(block.Base), uint64(block.Paras)*16), "at_segment_as_flat_pointer": read(uint64(block.Segment), 64)})
	}
	out := map[string]any{"registers": c.R, "segments": c.Seg, "flags": c.EFlags, "linear_eip_after_int": fmt.Sprintf("0x%X", c.EIP), "dos_buffers": buffers}
	if uint16(c.R[cpu386.EAX]) != 0x0300 {
		return out
	}
	d, ok := c.Descriptors[c.Seg[cpu386.SegES]]
	offset := uint64(c.R[cpu386.EDI])
	address := uint64(d.Base) + offset
	if !ok || offset+50 > uint64(d.Limit)+1 || address+50 > uint64(len(m.Mem)) {
		out["real_mode_packet_error"] = "packet outside descriptor or backing RAM"
		return out
	}
	p := m.Mem[int(address):int(address+50)]
	out["real_mode_packet"] = read(address, 50)
	if binary.LittleEndian.Uint16(p[28:]) != 0x1510 {
		return out
	}
	request := uint64(binary.LittleEndian.Uint16(p[34:]))*16 + uint64(binary.LittleEndian.Uint16(p[16:]))
	out["cd_request"] = read(request, 64)
	if request+20 <= uint64(len(m.Mem)) && m.Mem[int(request)+2] == 3 {
		q := m.Mem[int(request):int(request+20)]
		transfer := uint64(binary.LittleEndian.Uint16(q[16:]))*16 + uint64(binary.LittleEndian.Uint16(q[14:]))
		out["ioctl_transfer"] = read(transfer, uint64(binary.LittleEndian.Uint16(q[18:])))
	}
	return out
}

type scheduledMouse struct {
	Step    int `json:"step"`
	X       int `json:"x"`
	Y       int `json:"y"`
	Buttons int `json:"buttons"`
}

func parseMouseEvents(s string, budget int) ([]scheduledMouse, error) {
	var events []scheduledMouse
	d := json.NewDecoder(strings.NewReader(s))
	d.DisallowUnknownFields()
	if err := d.Decode(&events); err != nil {
		return nil, err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing mouse input")
	}
	if len(events) > 1024 {
		return nil, fmt.Errorf("too many mouse events")
	}
	previous := -1
	for _, e := range events {
		if e.Step <= previous || e.Step < 0 || e.Step >= budget || e.X < 0 || e.X > 65535 || e.Y < 0 || e.Y > 65535 || e.Buttons < 0 || e.Buttons > 7 {
			return nil, fmt.Errorf("invalid mouse event: %+v", e)
		}
		previous = e.Step
	}
	return events, nil
}

func applyMouseEvent(d *dos.DOS, e scheduledMouse) {
	d.MoveMouse(e.X, e.Y)
	previous := int(d.Mouse.Buttons)
	for button := 0; button < 3; button++ {
		mask := 1 << button
		if previous&mask == e.Buttons&mask {
			continue
		}
		if e.Buttons&mask != 0 {
			d.PressMouse(button)
		} else {
			d.ReleaseMouse(button)
		}
	}
}

func mouseInterrupt(c *cpu386.CPU, m *machine.Machine, cursor *mouseCursor) bool {
	fn := uint16(c.R[cpu386.EAX])
	if cursor == nil {
		return false
	}
	if fn == 9 {
		return cursor.define(c)
	}
	if fn == 1 {
		if !cursor.Defined {
			return false
		}
		if cursor.DisplayCount < 0 {
			cursor.DisplayCount++
		}
		return true
	}
	if fn == 2 {
		if cursor.DisplayCount <= -32768 {
			return false
		}
		if !legacyQuery(c, m, 0x33) {
			return false
		}
		cursor.DisplayCount--
		return true
	}
	switch uint16(c.R[cpu386.EAX]) {
	case 0, 3, 4, 7, 8, 0xb:
	default:
		return false
	}
	if !legacyQuery(c, m, 0x33) {
		return false
	}
	if fn == 0 {
		*cursor = mouseCursor{DisplayCount: -1}
	}
	return true
}

func executeAbsentFile(c *cpu386.CPU, files machine.ReadOnlyFileProvider) bool {
	name := []byte{}
	terminated := false
	for i := uint32(0); i < 260; i++ {
		a := c.R[cpu386.EDX]
		if a > ^uint32(0)-i {
			return false
		}
		b, ok := c.ReadSegment8(c.Seg[cpu386.SegDS], a+i)
		if !ok {
			return false
		}
		if b == 0 {
			terminated = true
			break
		}
		name = append(name, b)
	}
	if !terminated {
		return false
	}
	path := strings.TrimPrefix(string(name), ".\\")
	if path == "" || path == "." || path == ".." || strings.ContainsAny(path, "/\\:") {
		return false
	}
	f, err := files.OpenRead(path)
	if err == nil {
		f.Close()
		return false
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	lowWord(c, cpu386.EAX, 2)
	c.EFlags |= cpu386.CF
	return true
}

func legacyQuery(c *cpu386.CPU, m *machine.Machine, n uint8) bool {
	r := m.CPU
	if r == nil || r.IntHook == nil {
		return false
	}
	for i := range r.R {
		r.R[i] = uint16(c.R[i])
	}
	r.SetFlags(uint16(c.EFlags))
	if !r.IntHook(r, n) {
		return false
	}
	for i := range r.R {
		if i != cpu.SP {
			c.R[i] = c.R[i]&0xffff0000 | uint32(r.R[i])
		}
	}
	return true
}
