//go:build desktop

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var desktopPublisherIntro = flag.Bool("publisher-intro", true, "run the original MARK before the desktop game; false isolates DX5 verification")
var desktopPublisherSpeed = flag.Int("publisher-speed", 0, "publisher validation speed, 0 inherits desktop-speed; otherwise 1–60")

func init() { runDesktopPrelude = runPublisherPrelude }

// Ebitengine permits one RunGame per process. MARK and DX5 keep separate DOS state.
func runPublisherPrelude() (map[string]any, bool, error) {
	if !*desktopPublisherIntro {
		return nil, true, nil
	}
	speed := *desktopPublisherSpeed
	if speed == 0 {
		speed = *desktopSpeed
	}
	if speed < 1 || speed > 60 || *desktopSpeed < 1 || *desktopSpeed > 60 {
		return nil, false, fmt.Errorf("invalid desktop or publisher speed")
	}
	if _, err := newStartupChoice(*desktopStartupDisplay); err != nil {
		return nil, false, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, false, err
	}
	out, err := os.CreateTemp("/output/startup-probe", "mark-prelude-*.json")
	if err != nil {
		return nil, false, err
	}
	defer out.Close()
	stderr, err := os.CreateTemp("/output/startup-probe", "mark-stderr-*.log")
	if err != nil {
		return nil, false, err
	}
	defer stderr.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-program", "mark", "-desktop", "-steps", "75000000",
		"-desktop-speed", strconv.Itoa(speed), "-startup-display", *desktopStartupDisplay,
		"-hd-publisher-asset", *desktopPublisherAsset)
	command.Stdout, command.Stderr = out, stderr
	command.WaitDelay = 5 * time.Second
	commandErr := command.Run()
	if err := out.Close(); err != nil {
		return nil, false, err
	}
	if err := stderr.Close(); err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(out.Name())
	if err != nil {
		return nil, false, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	path := "/output/startup-probe/publisher-receipt-" + digest + ".json"
	if err := os.Rename(out.Name(), path); err != nil {
		return nil, false, err
	}
	if commandErr != nil {
		return nil, false, fmt.Errorf("MARK failed: %w; receipt %s; stderr %s", commandErr, path, stderr.Name())
	}
	var receipt map[string]any
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, false, fmt.Errorf("invalid MARK receipt %s: %w", path, err)
	}
	profile, _ := selectProgram("mark")
	if receipt["program"] != "mark" || receipt["input_file"] != profile.File || receipt["input_sha256"] != profile.SHA || receipt["fd2_startup_calls"] != float64(0) {
		return nil, false, fmt.Errorf("unqualified MARK receipt: %s", path)
	}
	desktop, ok := receipt["desktop"].(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("MARK desktop receipt missing: %s", path)
	}
	enabled, ok := desktop["hd_enabled"].(bool)
	if !ok {
		return nil, false, fmt.Errorf("MARK display setting missing: %s", path)
	}
	report := map[string]any{"receipt_path": path, "receipt_sha256": digest, "stderr_path": stderr.Name(), "receipt": receipt,
		"session_boundary": "independent LE memory and virtual time; no emulated DOS EXEC"}
	if err := os.Rename("/output/desktop-status.json", "/output/mark-desktop-status.json"); err != nil {
		return report, false, err
	}
	if receipt["stop"] == "desktop_closed" {
		report["continued"] = false
		return report, false, nil
	}
	if receipt["stop"] != "dos_exit" || receipt["exit_code"] != float64(0) {
		return report, false, fmt.Errorf("MARK did not exit normally: %s", path)
	}
	*desktopStartupDisplay = "original"
	if enabled {
		*desktopStartupDisplay = "hd"
	}
	report["continued"], report["next_display"] = true, *desktopStartupDisplay
	return report, true, nil
}
