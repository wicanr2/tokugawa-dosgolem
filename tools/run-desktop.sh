#!/bin/sh
# 只在 AGENTS 指定的既有 GUI image 執行；原始 dosgolem 與遊戲唯讀。
set -eu
test -f /.dockerenv
test "$(id -u)" = 1000
test "$(stat -c %u /output)" = 1000
export CGO_ENABLED=1 GOMAXPROCS=2
export GOFLAGS=-p=2
TOKUGAWA_PREPARE_ONLY=1 sh /project/tools/run-probes.sh
cp /project/tools/desktop.mod /output/desktop.mod
cp /project/tools/desktop.sum /output/desktop.sum
cd /dosgolem
go build -mod=readonly -modfile /output/desktop.mod -tags desktop \
  -overlay /output/startup-probe/repro-overlay.json \
  -o /output/tokugawa-desktop-prototype ./cmd/leprobe
go version > /output/toolchain.txt
go version -m /output/tokugawa-desktop-prototype > /output/build-info.txt
if test "${TOKUGAWA_BUILD_ONLY:-0}" = 1; then exit 0; fi
/output/tokugawa-desktop-prototype -desktop "$@" \
  > /output/startup-probe/pending.json 2> /output/startup-probe/latest-stderr.log
mv /output/startup-probe/pending.json /output/startup-probe/latest.json
