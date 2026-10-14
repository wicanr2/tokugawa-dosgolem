#!/bin/sh
# 僅在文件列出的 Go Docker image 內執行。
set -eu
test -f /.dockerenv || { printf '%s\n' '請在 Docker 容器內執行。' >&2; exit 2; }
test "$(id -u)" = 1000
test "$(stat -c %u /output)" = 1000
test "$(git -C /dosgolem rev-parse HEAD)" = a9714ebdab2ad6b529f81225472680f2b11f2842
test "$(sha256sum /dosgolem/internal/cpu386/cpu.go | cut -d ' ' -f 1)" = 932441d673ed5af723ff0b44db2fbf928072831750a1eccf85f41cb07cbdce20
test "$(sha256sum /dosgolem/internal/cpu386/cpu_test.go | cut -d ' ' -f 1)" = 0f34a8ef87f6f369e784699fad035525391a5eede6f9ef2ab6f34a6c5f0bd4b5
test "$(sha256sum /dosgolem/internal/machine/dpmi_real_mode.go | cut -d ' ' -f 1)" = b936e3b415d1e3b2b6da53737500992a949d2e400dee36979ac545df8d6b03e0
test "$(sha256sum /dosgolem/internal/machine/dpmi.go | cut -d ' ' -f 1)" = f0b579ae477e767c3ada50a68a123f8839c5062e6b741a2864cd1143e37805cc
mkdir -p /output/engine/internal/cpu386 /output/engine/internal/machine /output/startup-probe/receipts
if test -s /output/startup-probe/latest.json; then
  receipt_hash=$(sha256sum /output/startup-probe/latest.json | cut -d ' ' -f 1)
  cp /output/startup-probe/latest.json "/output/startup-probe/receipts/$receipt_hash.json"
fi
cp /dosgolem/internal/cpu386/cpu.go /output/engine/internal/cpu386/cpu.go
cp /dosgolem/internal/cpu386/cpu_test.go /output/engine/internal/cpu386/cpu_test.go
cp /project/tools/cpu386-integration_test.go /output/engine/internal/cpu386/integration_test.go
cp /dosgolem/internal/machine/dpmi_real_mode.go /output/engine/internal/machine/dpmi_real_mode.go
cp /dosgolem/internal/machine/dpmi.go /output/engine/internal/machine/dpmi.go
cd /output/engine
git apply --check /project/tools/dosgolem.patch
git apply /project/tools/dosgolem.patch
gofmt -w /output/engine/internal/cpu386/cpu.go /output/engine/internal/cpu386/integration_test.go /output/engine/internal/machine/dpmi_real_mode.go /output/engine/internal/machine/dpmi.go
cat > /output/startup-probe/repro-overlay.json <<'JSON'
{"Replace":{
  "/dosgolem/internal/cpu386/cpu.go":"/output/engine/internal/cpu386/cpu.go",
  "/dosgolem/internal/cpu386/cpu_test.go":"/output/engine/internal/cpu386/cpu_test.go",
  "/dosgolem/internal/cpu386/integration_test.go":"/output/engine/internal/cpu386/integration_test.go",
  "/dosgolem/internal/machine/dpmi_real_mode.go":"/output/engine/internal/machine/dpmi_real_mode.go",
  "/dosgolem/internal/machine/dpmi.go":"/output/engine/internal/machine/dpmi.go",
  "/dosgolem/internal/machine/dpmi_mcb.go":"/project/tools/dpmi-mcb.go",
  "/dosgolem/internal/machine/dpmi_mcb_test.go":"/project/tools/dpmi-mcb_test.go",
  "/dosgolem/cmd/leprobe/main.go":"/project/tools/startup-probe.go",
  "/dosgolem/cmd/leprobe/cdrom.go":"/project/tools/cdrom.go",
  "/dosgolem/cmd/leprobe/cdrom_test.go":"/project/tools/cdrom_test.go",
  "/dosgolem/cmd/leprobe/cdda.go":"/project/tools/cdda.go",
  "/dosgolem/cmd/leprobe/cdda_test.go":"/project/tools/cdda_test.go",
  "/dosgolem/cmd/leprobe/hd_trace.go":"/project/tools/hd-trace.go",
  "/dosgolem/cmd/leprobe/hd_capture.go":"/project/tools/hd-capture.go",
  "/dosgolem/cmd/leprobe/hd_trace_test.go":"/project/tools/hd-trace_test.go",
  "/dosgolem/cmd/leprobe/hd_live.go":"/project/tools/hd-live.go",
  "/dosgolem/cmd/leprobe/hd_portraits.go":"/project/tools/hd-portraits.go",
  "/dosgolem/cmd/leprobe/hd_kao.go":"/project/tools/hd-kao.go",
  "/dosgolem/cmd/leprobe/hd_kao_test.go":"/project/tools/hd-kao_test.go",
  "/dosgolem/cmd/leprobe/hd_kao_profiles.go":"/project/tools/hd-kao-profiles.go",
  "/dosgolem/cmd/leprobe/hd_live_test.go":"/project/tools/hd-live_test.go",
  "/dosgolem/cmd/leprobe/hd_layer.go":"/project/tools/hd-layer.go",
  "/dosgolem/cmd/leprobe/hd_layer_test.go":"/project/tools/hd-preview_test.go",
  "/dosgolem/cmd/leprobe/hd_scroll.go":"/project/tools/hd-scroll.go",
  "/dosgolem/cmd/leprobe/hd_scroll_test.go":"/project/tools/hd-scroll_test.go",
  "/dosgolem/cmd/leprobe/hd_frame.go":"/project/tools/hd-frame.go",
  "/dosgolem/cmd/leprobe/hd_frame_test.go":"/project/tools/hd-frame_test.go",
  "/dosgolem/cmd/leprobe/hd_window.go":"/project/tools/hd-window.go",
  "/dosgolem/cmd/leprobe/hd_window_test.go":"/project/tools/hd-window_test.go",
  "/dosgolem/cmd/leprobe/presentation.go":"/project/tools/presentation.go",
  "/dosgolem/cmd/leprobe/presentation_test.go":"/project/tools/presentation_test.go",
  "/dosgolem/cmd/leprobe/desktop.go":"/project/tools/desktop.go",
  "/dosgolem/cmd/leprobe/windowbar.go":"/project/tools/windowbar.go",
  "/dosgolem/cmd/leprobe/windowbar_test.go":"/project/tools/windowbar_test.go",
  "/dosgolem/cmd/leprobe/desktop_windowbar.go":"/project/tools/desktop-windowbar.go",
  "/dosgolem/cmd/leprobe/video.go":"/project/tools/video.go",
  "/dosgolem/cmd/leprobe/video_test.go":"/project/tools/video_test.go",
  "/dosgolem/cmd/leprobe/vbe.go":"/project/tools/vbe.go",
  "/dosgolem/cmd/leprobe/vbe_test.go":"/project/tools/vbe_test.go",
  "/dosgolem/cmd/leprobe/keyboard.go":"/project/tools/keyboard.go",
  "/dosgolem/cmd/leprobe/keyboard_test.go":"/project/tools/keyboard_test.go",
  "/dosgolem/cmd/leprobe/dos_test.go":"/project/tools/dos_test.go"
}}
JSON
cd /dosgolem
go test -count=1 -overlay /output/startup-probe/repro-overlay.json ./internal/cpu386 ./internal/machine ./internal/dosfile ./cmd/leprobe > /output/startup-probe/platform-test.log 2>&1
if test "${TOKUGAWA_PREPARE_ONLY:-0}" = 1; then exit 0; fi
go run -overlay /output/startup-probe/repro-overlay.json ./cmd/leprobe "$@" > /output/startup-probe/pending.json 2> /output/startup-probe/latest-stderr.log
mv /output/startup-probe/pending.json /output/startup-probe/latest.json
find /output/engine /output/startup-probe \( -user root -o -type d -name '*.md' \) -print
