# 德川家康：DOS 原版與 dosgolem

## 開工入口

本專案保存《德川家康》的 jsdos 來源與 dosgolem 接入證據。下載完成不代表遊戲已在 dosgolem 可玩。本次沒有建立 remake 引擎。

1. 先讀 [CONTEXT.md](CONTEXT.md)，以其中唯一的目前狀態表為準。
2. 依任務查 `/home/anr2/.codex/knowledge-base/knowledge-router.md`，只載入命中的入口。回覆載明命中結果。
3. 接入 dosgolem 前讀 `/home/anr2/cht/dosgolem/README.md`、`CLAUDE.md`、`docs/spec/000-index.md` 與 `docs/spec/081-dos4gw-capability-matrix.md`。能力矩陣中的其他遊戲收據不能證明本作可執行。
4. 核對 [來源收據](original/source.json) 與檔案雜湊。先查證玩家阻塞點，再決定研究範圍。

面向使用者的回覆與文件用繁體中文。檔名、識別字、命令與工具名稱保留原文。

## 路徑與文件職責

| 路徑 | 職責 |
|---|---|
| `/home/anr2/cht/德川家康` | 本專案根目錄 |
| [original/德川家康.jsdos.zip](original/德川家康.jsdos.zip) | 網站原始 jsdos ZIP，保持位元組不變 |
| [original/source.json](original/source.json) | 下載網址、API 回應、工具版本、大小與逐檔 SHA-256 |
| [original/game/](original/game/) | 原樣解包的遊戲、光碟與 jsdos 設定，執行時唯讀掛載 |
| [tools/fetch-jsdos.py](tools/fetch-jsdos.py) | 固定版本下載與解包；拒絕覆寫既有原始檔 |
| [tools/verify-source.py](tools/verify-source.py) | 離線核對 ZIP、解包檔與擁有權 |
| [tools/le-entry-probe.go](tools/le-entry-probe.go) | 不帶 runtime 的原始 LE 入口有界探針 |
| [tools/startup-probe.go](tools/startup-probe.go) | 原版指令自然執行的 DOS/4G 啟動研究探針 |
| [tools/ida-entry.py](tools/ida-entry.py) | 保留原名與位址的 IDA 入口證據；可傳入 `起點:終點` 匯出至多 16 KiB 的窄範圍 |
| [tools/dosgolem.patch](tools/dosgolem.patch) | 固定 dosgolem 基準的通用平台修正，尚未回填外部工作樹 |
| [tools/cpu386-integration_test.go](tools/cpu386-integration_test.go) | 平台修正的獨立回歸測試 |
| [tools/run-probes.sh](tools/run-probes.sh) | 在 Docker 中套用修正、跑平台測試並重生啟動收據 |
| [tools/cdrom.go](tools/cdrom.go) | 真實 CUE/BIN 掛載與 MSCDEX 查詢子集 |
| [tools/cdrom_test.go](tools/cdrom_test.go) | 光碟幾何、來源雜湊與傳輸界限測試 |
| [tools/aux-cd-trace.py](tools/aux-cd-trace.py) | 固定 DOSBox-X image 中查證包內 DOS/4GW 的配置與 TOC 回寫，只作輔助證據 |
| [tools/video.go](tools/video.go) | 標準 VGA 埠轉接、模式切換、擴充索引隔離與原生畫面擷取 |
| [tools/video_test.go](tools/video_test.go) | VGA 暫存器隔離與 DAC 回歸測試 |
| [tools/vbe.go](tools/vbe.go) | VBE 分頁 VRAM 與 BIOS 子集 |
| [tools/vbe_test.go](tools/vbe_test.go) | VBE 資訊、分頁與界限測試 |
| [tools/keyboard.go](tools/keyboard.go) | 共用 BDA 按鍵緩衝的 DOS／BIOS 鍵盤子集 |
| [tools/keyboard_test.go](tools/keyboard_test.go) | 按鍵查詢、消耗與空緩衝測試 |
| [tools/player-path.json](tools/player-path.json) | 原版入口、角色建立與敘事推進的普通滑鼠時間表；已驗證範圍見 RESEARCH-LOG |
| [tools/setup-github.sh](tools/setup-github.sh) | 主機 gh 建立 private repo 與六筆 Issues；Docker 擷取本文，`--prepare` 不存取 GitHub |
| `workplace/github-setup/` | 從 WORKLIST 重生的待送出 Issue 本文，不作遠端狀態台帳 |
| `workplace/cd-trace/` | CD TOC／PLAY 的隔離診斷收據、IDA 窄範圍證據與可重生工具副本；`aux-toc/` 保存額外 TOC 查詢的 DOSBox-X 輔助證據 |
| [tools/dos_test.go](tools/dos_test.go) | DOS EXEC 缺檔錯誤與不假執行測試 |
| [RESEARCH-LOG.md](RESEARCH-LOG.md) | 輸入雜湊、位址空間、平台契約與研究證據 |
| [README.md](README.md) | 專案用途、穩定入口與執行方式摘要 |
| [WORKLIST.md](WORKLIST.md) | dosgolem 接入待辦、CD 音樂／四語／HD 規劃、完成條件與待送出的 GitHub 建立內容 |
| [workplace/](workplace/) | 探針、快取、截圖、實驗與其他中間產物 |
| `workplace/startup-probe/state-*` | 各次執行的可寫檔案副本；由既有 DirectoryOverlayFiles 建立，不回寫原版 |
| `workplace/startup-probe/frame-<SHA-256>.png` | dosgolem 原生 VRAM 與 DAC 生成的畫面；是否為遊戲畫面須另行核對 |
| `workplace/fdos-int33.lst` | FreeDOS mouse 專案的滑鼠介面參考快照；來源與 SHA-256 見 RESEARCH-LOG |
| `workplace/mscdex21.txt`、`workplace/dosbox-staging-v0.82.2-*.cpp` | MSCDEX 規格及固定版本的光碟平台參考快照；來源與 SHA-256 見 RESEARCH-LOG |
| [workplace/leprobe/stdout.log](workplace/leprobe/stdout.log) | 本次 dosgolem LE 格式與 relocation preview 收據 |
| [workplace/leprobe/stderr.log](workplace/leprobe/stderr.log) | 探針錯誤輸出 |
| [workplace/leprobe/toolchain.txt](workplace/leprobe/toolchain.txt) | 探針使用的 Go 版本 |
| `workplace/ida/range-<起點>-<終點>.json` | IDA 9.4 的窄範圍原始指令、bytes、檔案偏移與交叉參照；語意結論另見 RESEARCH-LOG |
| [workplace/intake-verification.json](workplace/intake-verification.json) | 最近一次來源完整性與工具雜湊核對結果 |
| [CONTEXT.md](CONTEXT.md) | 目前狀態、有效結論、限制與下一個驗證入口 |
| [WORKLOG.md](WORKLOG.md) | 按時間追加的工作歷程與清理結果 |
| [.gitignore](.gitignore) | 排除本機原版資料、探針輸出與快取 |

新增目錄或文件前讀 `/home/anr2/.codex/knowledge-base/local/project-document-responsibilities.md`，並更新本表。未有可執行待辦前不另建 WORKLIST；需要時只建立一份。README 只放穩定入口，工作歷程寫入 WORKLOG。

## 原版啟動契約

來源頁：[德川家康](https://dos.zczc.cz/games/%E5%BE%B7%E5%B7%9D%E5%AE%B6%E5%BA%B7/)。實際下載網址與 API 記錄見 `original/source.json`。

包內 [dosbox.conf](original/game/.jsdos/dosbox.conf) 的啟動順序如下。`PLAY.BAT` 的內容是 `DX5.EXE`。

```ini
[autoexec]
echo off
mount c .
c:
imgmount d ./DX5.cue -t iso
PLAY.BAT
```

- DOSBox 的工作目錄須對應 `original/game/`。保留 `DX5.EXE`、`DX5.DAT`、`DOS4GW.EXE` 與 CUE/BIN 的相對位置。
- `DX5.EXE` 是 MZ 內嵌 LE，LE 標頭位於**檔案偏移** `0x2C90`。不可將普通 MZ stub 的執行結果當成 32 位遊戲已啟動。
- `DX5.cue` 宣告一軌 `MODE1/2352` 資料與十五軌 CD 音樂。實際映像檔名是 `DX5.bin`，CUE 引用 `DX5.BIN`。保留下載原樣；若工具要求大小寫一致，在可寫工作副本處理並記錄，不改原始資料。
- CD 音樂不以 Sound Blaster 或 AdLib 合成代替。光碟掛載、資料讀取與音軌播放各自驗證，不能用其中一項代表全部支援。
- 使用包內 `DX5.EXE` 作為網站版本基準。它與 `DX5.EXE.BAK` 不同；包內附畫面亮度修正說明。未確認之前，不把 `.BAK` 稱為未修改的出廠原版，不切換版本或再次套補丁。
- 已發現的差異包含亮度修正區，以及**檔案偏移** `0x19395` 的 `EB`／`74` 差異。IDA 已確認網站 EXE 在此略過 TOC 末軌為 16 的檢查；原始定位與勘誤見 RESEARCH-LOG，完整雜湊見 CONTEXT。這不證明 `.BAK` 的來源或其餘差異。

## Docker 邊界

輔助 DOSBox 基準腳本為 [tools/aux-dosbox.sh](tools/aux-dosbox.sh)。只在既有 `san5-dosbox-xvfb:bookworm-20261009-r1` 容器內執行，輸入掛在 `/original` 唯讀，工作樹唯讀掛在 `/project`，`workplace/` 掛在 `/output`。使用 UID/GID 1000、`--network none`、1 GiB、2 CPU、128 程序與 `timeout 55s sh /project/tools/aux-dosbox.sh`。輸出 `workplace/dosbox-aux/`，用途只限輔助查證。

CD 配置的輔助除錯使用 [tools/aux-cd-trace.py](tools/aux-cd-trace.py)，沿用既有 DOSBox-X image `psychicwar-dosboxx@sha256:79a3e2df79738a5821a7a168d15e73894ad3ce7e7c6e2d4cdd53c09550316d46`。實際程式版本為 2026.06.02、commit `d75a57c72a74a6d87250e4b2556e70ea02995e51`，不能由 image tag 推定版本。只觀測第一次 0100h 配置與 0300h TOC 回寫，不注入滑鼠、不改 guest 記憶體或暫存器。執行前確認 `original/game/`、`workplace/cd-trace/` 與腳本存在，再以同樣資源限制啟動容器；工作樹掛 `/project` 唯讀、原版掛 `/original` 唯讀、`workplace/cd-trace/` 掛 `/output` 可寫，執行 `timeout -k 5s 90s python3 /project/tools/aux-cd-trace.py`。輸出在 `workplace/cd-trace/aux-debug/`，含前後記憶體、暫存器、原始終端、記錄檔及雜湊 manifest；這些不能充當 dosgolem 原版收據。Xvfb 與 debugger 由腳本有界管理並清理。

額外 TOC 查詢使用同一掛載與 image，執行 `timeout -k 5s 180s python3 /project/tools/aux-cd-trace.py --toc-queries 2`，外層期限設為 190 秒。兩筆查詢各保存實模式封包、請求與傳輸緩衝區的前後 bytes；輸出在 `workplace/cd-trace/aux-toc/`。數量限 0 至 16，擴大範圍前須相應調整有界期限。

下載、分析、解包、建置、測試、遊戲執行與抓圖都在 Docker。主機只作 Docker／Git 控制、檔案編輯與必要狀態檢查。不把主機 Python、Go、Wine 或 DOSBox 當成替代工具鏈。

- 沿用已驗證的 `python:3.13.7-slim` 與 `golang:1.24.13-bookworm`，下列命令固定映像 digest。缺件時先修正可重現工具鏈。
- 每個掛載來源先以 `test -f`／`test -d` 確認。使用 `--mount`，避免不存在的來源被 dockerd 建成 root-owned 目錄。
- 一次性容器使用 `--rm`、外層逾時、CPU／記憶體／程序數上限，以及目前使用者的 UID/GID。
- 預設 `--network none`；只有來源下載明確使用 `--network bridge`。
- 遊戲與 dosgolem 原始碼唯讀掛載。存檔、快取與收據寫入 `workplace/` 的指定輸出目錄。
- 每批收尾在容器內檢查 `find /project \( -user root -o -type d -name '*.md' \) -print`。只修正確認過的個別問題，不遞迴改整個專案擁有權。
- 用 `docker ps -a --filter name=tokugawa` 查本專案容器，移除本次留下的資源。不清理其他專案，不執行全域 prune。

## 已驗證的操作入口

在專案根目錄執行。主機 shell 只負責檢查路徑與啟動 Docker。

GitHub 建立工具另依使用者明確授權使用主機 `/home/anr2/.local/bin/gh`。在主機終端執行 `sh tools/setup-github.sh`，先由 Docker 從 WORKLIST 擷取本文，再核對主機登入、建立或沿用 private repository，以及建立缺少的六筆 Issues。重跑前讀取所有頁的遠端現況，同題 Issue 不重建，既有 Issue 不覆寫。工具不提交遊戲資料，也不初始化本機 `.git`。僅預覽待送出本文時執行 `sh tools/setup-github.sh --prepare`。遠端建立尚未實測，最新驗證狀態以 CONTEXT 為準。

來源完整性核對：

```sh
test -f tools/verify-source.py && test -d original/game && test -d workplace
timeout 90s docker run --rm --name tokugawa-verify --network none \
  --memory 512m --cpus 1 --pids-limit 64 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/workplace,target=/output" \
  python@sha256:5f55cdf0c5d9dc1a415637a5ccc4a9e18663ad203673173b8cda8f8dcacef689 \
  python3 /project/tools/verify-source.py
```

dosgolem 的唯讀 LE 解析與 relocation preview：

```sh
test -d /home/anr2/cht/dosgolem && test -f original/game/DX5.EXE
mkdir -p workplace/leprobe
timeout 180s docker run --rm --name tokugawa-leprobe --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount type=bind,source=/home/anr2/cht/dosgolem,target=/dosgolem,readonly \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace/leprobe,target=/output" \
  --env HOME=/tmp --env GOCACHE=/output/gocache --env GOMODCACHE=/output/gomodcache \
  --env GOFLAGS=-mod=readonly --env GOTOOLCHAIN=local --env CGO_ENABLED=0 \
  --workdir /dosgolem \
  golang@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac \
  sh -c 'go version > /output/toolchain.txt && go run ./cmd/leprobe -exe /original/DX5.EXE > /output/stdout.log 2> /output/stderr.log'
```

`cmd/leprobe` 預設只解析格式與預覽 relocation。`execution_support=partial` 不能解讀成《德川家康》已可執行。**不可加 `-execute-entry-prefix`**；目前該分支使用 FD2 專用 runtime 與硬編碼位址，證據是 `/home/anr2/cht/dosgolem/cmd/leprobe/main.go` 的 `executePrefix`。

原版自然啟動研究探針與平台回歸：

```sh
test -d /home/anr2/cht/dosgolem && test -f original/game/DX5.EXE
test -d workplace && test -f tools/run-probes.sh
timeout 630s docker run --rm --name tokugawa-repro-probe --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount type=bind,source=/home/anr2/cht/dosgolem,target=/dosgolem,readonly \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace,target=/output" \
  --env HOME=/tmp --env GOCACHE=/output/leprobe/gocache \
  --env GOMODCACHE=/output/leprobe/gomodcache --env GOTOOLCHAIN=local \
  --env CGO_ENABLED=0 --env GOFLAGS=-mod=readonly --workdir /dosgolem \
  golang@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac \
  timeout -k 10s 600s sh /project/tools/run-probes.sh
```

先核對 `workplace/startup-probe/platform-test.log`，再看 `latest.json` 的停止原因。未支援服務與指令保持失敗即停止。`instruction_budget` 也不能解讀成可玩。`fd2_startup_calls` 必須為零；探針沒有安裝 FD2 函式攔截。歷次收據在 `workplace/startup-probe/receipts/`。

命令最後的 `tools/run-probes.sh` 可接 `-steps 100000000`，保留原版等待並跑到主選單。普通滑鼠輸入用 `-mouse '<JSON>'` 傳入事件陣列，每筆含 `step`、`x`、`y`、`buttons`。`step` 是成功指令數，與虛擬微秒一致；`buttons` 的 bit0 為左鍵、bit1 為右鍵、bit2 為中鍵。按下與放開分兩筆，時間嚴格遞增，全部落在預算內。實際套用結果保存在 `latest.json` 的 `mouse_input_events`。輸入只走 dosgolem 的 MoveMouse／PressMouse／ReleaseMouse，不帶遊戲位址。目視核對與已測操作見 RESEARCH-LOG。

查證 CD 資料流可加 `-cd-trace`，記錄有界的 DPMI 配置與 INT 2Fh 轉送前後資料。診斷只讀 backing memory，預設關閉。將上述 Docker 命令的 `/output` 掛載改為已存在的 `workplace/cd-trace/`，以隔離診斷輸出；正式啟動收據仍留原目錄。開關對照與資料流結論見 RESEARCH-LOG 的「CD 只讀診斷切片」。

要重跑保存的正常玩家路徑，在同一 Docker 命令最後使用 `tools/run-probes.sh -steps 750000000 -mouse-file /project/tools/player-path.json`。`-mouse-file` 與 `-mouse` 擇一，輸入雜湊記入 `mouse_input_sha256`。路徑的驗證進度以 CONTEXT 為準，不以輸入檔存在宣稱通過。

長流程若被外層工作階段終止，將同一命令的 `docker run --rm` 改成 `docker run --detach --rm`，容器內仍保留 600 秒期限與 10 秒強制終止緩衝。以 `docker ps -a --filter name=tokugawa-repro-probe` 核對是否結束。執行中 `latest.json` 仍是前次完整收據；本次只先寫 `pending.json`，成功才替換。結束後必須核對收據的 `mouse_input_sha256`、`integration_input_sha256` 與本次工具及輸入檔，不能沿用前次結果。受限工具不支援外層 `timeout docker` 時，使用單一 Docker 命令並由呼叫工具監督外層期限。

首次下載或在新的空白工作目錄重建來源時，使用 `tools/fetch-jsdos.py`。原有下載存在就停止，不先刪除以繞過保護：

```sh
test -f tools/fetch-jsdos.py
timeout 240s docker run --rm --name tokugawa-download --network bridge \
  --memory 512m --cpus 1 --pids-limit 64 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project" \
  python@sha256:5f55cdf0c5d9dc1a415637a5ccc4a9e18663ad203673173b8cda8f8dcacef689 \
  python3 /project/tools/fetch-jsdos.py
```

腳本固定本次 API 指定的 SHA-256 與大小，支援未完成 `.part` 續傳。API 換版時先核對來源，不直接放寬檢查。

## dosgolem 接入與證據規則

- 已有 LE 載入、原始入口與 DOS/4G 啟動研究探針。最新停止點見 CONTEXT 與 `workplace/startup-probe/latest.json`；正常啟動、輸入、畫面、存讀檔與音訊仍須分項驗證。
- 新接入先區分 LE loader／386 指令／DPMI／DOS 檔案服務／光碟與 MSCDEX／顯示／輸入／音訊缺口。沒有本作執行證據的項目標為未知。
- 程式專用位址、流程與攔截點留在本作層。通用平台能力才進 dosgolem；`apps/fd2`、`apps/rich2` 等其他遊戲的位址與 runtime 初始化不可套用本作。
- 後續要改玩法、資料格式、玩家路徑或忠實度時，依序走「RE 證據 → DRAFT → 證據審查 → READY → 實作 → 同狀態驗證 → CONFORMED」。缺口回到證據與規格，不猜補。
- 正式原版收據由 dosgolem 自行產生。DOSBox／DOSBox-X 只補足缺口與交叉驗證，擷取須標為輔助基準。補齊後回到 dosgolem 重跑。
- 亂數測試在執行前固定兩側條件，記錄 seed、設定方式、版本、初始狀態與輸入。不重擲挑結果，不假定同一 seed 數字必定產生同一序列。
- 使用原始 bytes、位址與資料流作證據。語意分「已證實／強推論／假說／未知」，保留原始函式名與位址，不以推測性改名取代定位。
- 區分檔案偏移、LE object offset、執行期線性位址及 IDA 位址。本次 `0x2C90`、`0x19395` 均為檔案偏移。
- 硬體時序依公開規格近似，標為 `hardware-spec approximation`。不為逐週期一致深挖音訊 driver、ISR 或 busy-wait。

## 範圍與資料權利

目前授權範圍是下載 jsdos、建立 private GitHub repository 與 Issues，並將原版遊戲整合到 dosgolem 執行。使用者指定先建立 repo、登記 Issues，再開工。使用者明確表示不是 remake。可自主補齊本作所需的平台能力與測試；玩法修改、版本切換、公開發行或授權變更需另有明確任務。

後續規劃已確認桌面版、繁中／簡中／英文／日文，以及可切換的 HD 素材包。瀏覽器排除。現行決定見 [CONTEXT.md](CONTEXT.md)，階段與驗收草案見 [WORKLIST.md](WORKLIST.md)。原版繼續負責玩法與存檔，譯文與 HD 圖層在桌面呈現；完整擷取介面、素材格式與美術風格尚未定案，不把規劃寫成既有能力。

原版 ZIP、EXE、DAT、光碟映像、音軌與原包附帶的補丁只留本機，不加入 Git、GitHub Release 或公開雲端。`.gitignore` 已排除 `original/` 與 `workplace/`。自製的平台修正、測試與文件可加入已授權的 private repository，包含 `tools/dosgolem.patch`。未確認散布權前不製作公開資料包。
