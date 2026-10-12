#!/bin/sh
# 僅作輔助基準，圖片不能充當 dosgolem 收據。
set -eu
test -f /.dockerenv
test "$(id -u)" = 1000
test "$(stat -c %u /output)" = 1000
mkdir -p /output/dosbox-aux/game
for f in /original/*; do
  case "$f" in *.bin|*.cue) continue;; esac
  if test -f "$f"; then
    target="/output/dosbox-aux/game/$(basename "$f")"
    if test -f "$target"; then test "$(stat -c %u "$target")" = 1000; chmod u+w "$target"; fi
    cp "$f" "$target"
    chmod u+w "$target"
  fi
done
if test -f /output/dosbox-aux/game/DX5.cue; then
  test "$(stat -c %u /output/dosbox-aux/game/DX5.cue)" = 1000
  chmod u+w /output/dosbox-aux/game/DX5.cue
fi
cp /original/DX5.cue /output/dosbox-aux/game/
ln -sf /original/DX5.bin /output/dosbox-aux/game/DX5.BIN
trap 'kill ${game_pid:-} ${x_pid:-} 2>/dev/null || true; wait 2>/dev/null || true' EXIT INT TERM
Xvfb :91 -screen 0 1024x768x24 >/output/dosbox-aux/xvfb.log 2>&1 &
x_pid=$!
sleep 1
cd /output/dosbox-aux/game
dosbox -conf /original/.jsdos/dosbox.conf >/output/dosbox-aux/dosbox.log 2>&1 &
game_pid=$!
sleep 10
import -window root /output/dosbox-aux/10s.png
sleep 15
import -window root /output/dosbox-aux/25s.png
if test "${1:-}" = --input; then
  window_id=$(xdotool search --name DOSBox | head -n 1)
  xdotool getwindowgeometry --shell "$window_id" > /output/dosbox-aux/window-geometry.txt
  xdotool key --window "$window_id" Return
  sleep 3
  import -window root /output/dosbox-aux/input-28s.png
  xdotool mousemove --window "$window_id" 300 172 click 1
  sleep 3
  import -window root /output/dosbox-aux/input-31s.png
  xdotool mousemove --window "$window_id" 130 43 click 1
  sleep 3
  import -window root /output/dosbox-aux/input-34s.png
fi
find /output/dosbox-aux \( -user root -o -type d -name '*.md' \) -print
