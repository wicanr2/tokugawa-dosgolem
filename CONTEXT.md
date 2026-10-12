# 德川家康目前狀態

更新日期：2026-10-12。規則入口：[AGENTS.md](AGENTS.md)。歷程：[WORKLOG.md](WORKLOG.md)。待辦：[WORKLIST.md](WORKLIST.md)。證據：[RESEARCH-LOG.md](RESEARCH-LOG.md)。

## 唯一目前狀態表

| 項目 | 現況 | 證據 |
|---|---|---|
| jsdos 下載 | 完成，214,665,581 bytes | `original/德川家康.jsdos.zip`、[source.json](original/source.json) |
| ZIP 與解包 | CRC 通過；14 筆 ZIP entries，12 個檔案；解包檔共 235,521,461 bytes | `original/source.json` 的逐檔大小與 SHA-256 |
| 原始檔保護 | ZIP 與解包檔設為唯讀；擁有者 UID/GID 1000:1000 | `original/`；最近驗證結果見下方入口 |
| 網站啟動契約 | 掛載 C: 工作目錄、D: CUE 光碟，再由 `PLAY.BAT` 呼叫 `DX5.EXE` | `original/game/.jsdos/dosbox.conf`、`PLAY.BAT` |
| 執行檔格式 | 已證實為 MZ 內嵌 LE；LE header 檔案偏移 `0x2C90` | 原始 bytes；[LE 探針](workplace/leprobe/stdout.log) |
| dosgolem 格式解析 | 完成；40 pages、2 objects、4,831 fixup records；relocation preview 成功 | `workplace/leprobe/stdout.log`，stderr 為空，exit 0 |
| dosgolem 執行本作 | 整合啟動驗收通過；原版由 LE 入口走通出陣，顯示地形與人物單位，持續執行戰場滑鼠查詢。750,000,000 步只因預算結束，沒有未支援指令或 DPMI 服務 | [完整收據](workplace/startup-probe/receipts/307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4.json)、[原生戰場畫面](workplace/startup-probe/frame-b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846.png)、[驗收範圍](RESEARCH-LOG.md) |
| 通用平台修正 | 在隔離副本實作 CPU 指令形式與 DPMI 實模式主機服務轉接；外部 dosgolem 工作樹未改 | [dosgolem.patch](tools/dosgolem.patch)、[回歸測試](tools/cpu386-integration_test.go) |
| 平台回歸 | CPU386、machine、dosfile、研究入口四個套件通過；每次修改後重跑 | [platform-test.log](workplace/startup-probe/platform-test.log) |
| 原版時間等待 | DOS 時間與 VGA 回掃共用一微秒／成功指令的虛擬時間；原版自行校準等待迴圈 | 最新收據 `virtual_clock`；[回掃契約](RESEARCH-LOG.md)；未略過指令 |
| 光碟 | 真實 CUE/BIN 掛載與 TOC 子集已測；只讀診斷確認 DPMI 段號被原版當作平面指標，TOC 與請求緩衝區因此重疊；與零長度 PLAY 的因果關係仍待確認，正長度音訊未支援 | [cdrom.go](tools/cdrom.go)、[RESEARCH-LOG.md](RESEARCH-LOG.md)；`workplace/cd-trace/trace-on.json` |
| CD 只讀診斷 | 開關對照通過；同一 3,000,000 步的完整記憶體、畫面、停止位置、中斷計數與原始 trace 一致；未修改原版或 DPMI 回傳契約 | [startup-probe.go](tools/startup-probe.go) 的 `-cd-trace`；`workplace/cd-trace/trace-on.json`、`trace-off.json` |
| 滑鼠與 BIOS | 普通滑鼠已推進選單、角色建立與敘事；已接原版遮罩與畫面游標層，未實作 TSR 的 VRAM 背景讀回 | [video.go](tools/video.go)；`latest.json` 的輸入與邊界畫面 |
| VGA／VESA 顯示 | 原版模式 12h 與 CRTC 自行設定 640×400，開場、標題與主選單已目視確認；VBE 分頁子集另有測試 | [原生標題與選單](workplace/startup-probe/frame-7df7211c989bca30c3e82077011a96e7ef7142c9029c515aeb5dda5e461697e7.png)、[video.go](tools/video.go) |
| DOSBox 輔助基準 | 可顯示原版標題；DOSBox-X 搭配包內 DOS/4GW 也確認段號被誤用為指標，兩個 MCB 配置讓段號相差 2；只作輔助證據，未作音訊驗證 | [25s.png](workplace/dosbox-aux/25s.png)、[aux-cd-trace.py](tools/aux-cd-trace.py)、`workplace/cd-trace/aux-debug/manifest.json`；詳見 RESEARCH-LOG |
| 正常玩家路徑、存讀檔與音訊 | 部分驗證；選單、角色建立、敘事、選項與出陣確認通過，地形、人物單位與小地圖已目視確認；單位操作、戰鬥與存讀檔未驗證，CD 正長度音訊未支援 | [普通輸入時間表](tools/player-path.json)；精確畫面雜湊與操作見 RESEARCH-LOG |
| remake | 排除 | 使用者明確指定原版與 dosgolem 整合 |
| 後續功能範圍 | 已確認 CD 音樂、繁中／簡中／英文／日文，以及可切換的 HD 素材包；已授權在 repo 與 Issues 建立後開工，尚未實作 | 本次使用者選擇；階段與待送出 Issue 內容見 [WORKLIST.md](WORKLIST.md) |
| 後續使用介面 | 桌面版；瀏覽器排除 | 使用者明確表示「不考慮瀏覽器」；桌面作業系統與打包驗收清單尚未定案 |
| private GitHub repository 與 Issues | 主機 gh 已確認 [wicanr2/tokugawa-dosgolem](https://github.com/wicanr2/tokugawa-dosgolem) 為 private，六筆 Issues 均已存在且開啟；從 [CD-TRACE #1](https://github.com/wicanr2/tokugawa-dosgolem/issues/1) 開工 | 2026-10-12 主機 `gh auth status`、`repo view` 與分頁 Issues API；不重複建立 |
| GitHub 建立工具驗證 | shell 語法、六筆本文擷取與隔離假 API 測試通過；遠端未實測 | `--prepare` 不呼叫 GitHub；查重、相依 URL、登入／列舉失敗、公有同名 repo 與重複 Issues 的停止行為已測。歷程見 WORKLOG |
| Docker 收尾 | 本輪原生重跑與驗證容器均已結束，無本專案殘留；未建新 image | `docker ps -a --filter name=tokugawa` 為空；擁有權檢查通過 |

## 固定來源與版本差異

| 檔案 | SHA-256 |
|---|---|
| `德川家康.jsdos.zip` | `e35bb236ae48f095a6dbda3236a59e5a5b903c82fe9d5564233c15c1793b425a` |
| `DX5.EXE` | `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a` |
| `DX5.EXE.BAK` | `95bffc6fac92974bc678ddefa6e86f601676e78dd2c2d0ff0b7ac2da189512f9` |
| `DX5.bin` | `3f511d4c2fb819f950d881697f4e771d37ccfa2b0fd9eca6c4bb48fff6be2448` |

來源頁、API 回應、最終下載網址與其餘檔案雜湊集中在 `original/source.json`，不以遊戲名稱辨認 binary 版本。

- 已證實：`DX5.EXE` 與 `.BAK` 在檔案偏移 `0x17459`、`0x17582` 分別為 `3F` 與 `15`，以及調色盤範圍 `0x2FAD4–0x2FBF3` 存在差異。包內 UTF-8 修正說明記載亮度調整方法。保留網站使用的 EXE 作為此次基準。
- 已證實：檔案偏移 `0x19395` 的 active／BAK bytes 為 `EB`／`74`。語意未知，不宣稱所有差異只有亮度修正。
- 已證實：`DX5.cue` 宣告 16 軌，第一軌 `MODE1/2352`，其餘 15 軌 `AUDIO`。尚未驗證實際音軌播放與遊戲的選曲行為。
- 未知：`.BAK` 的版本來源、額外差異用途，以及尚未測到的戰鬥、存讀檔與音訊相容性。

## 探針版本與限制

| 項目 | 值 |
|---|---|
| dosgolem 路徑 | `/home/anr2/cht/dosgolem`，執行時唯讀掛載 |
| Git HEAD | `a9714ebdab2ad6b529f81225472680f2b11f2842` |
| 既有工作樹差異 | `cmd/probe/main.go` 已修改，本次沒有改動；本次跑 `cmd/leprobe` |
| Go | `go1.24.13 linux/amd64` |
| Go image | `golang@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac` |
| 下載／驗證工具 | Python 3.13.7；固定 image 與腳本雜湊見來源／驗證收據 |
| 位址空間 | LE header 為檔案偏移；entry 是 object 1 offset `0x8DB4`，dosgolem relocation 後線性位址 `0x18DB4` |
| 原始入口探針 | 11 步停在 dosgolem 線性 `0x18E51`，`INT 21h AH=30h`；[收據](workplace/entry-probe/receipt.json) |
| IDA | Pro 9.4；[入口匯出](workplace/ida/intake.json) 與 `workplace/ida/DX5.i64`，不改名 |
| CPU 修正基準 | 原始 `internal/cpu386/cpu.go` SHA-256 `932441d673ed5af723ff0b44db2fbf928072831750a1eccf85f41cb07cbdce20` |
| 執行方式 | Go overlay，原始 dosgolem 與遊戲唯讀；可寫副本只在 `workplace/engine/` |

`cmd/leprobe` 的 `runtime_applied=false` 表示這份輸出是 relocation preview。不能用 `execution_support=partial` 宣稱本作可執行。`-execute-entry-prefix` 使用 FD2 專用 runtime 與位址，不適用本作。

最近來源完整性與工具輸入雜湊見 [intake-verification.json](workplace/intake-verification.json)。可重跑命令放在 AGENTS。

探針不自動判讀畫面。收據的 `visually_verified`、`game_screen_verified`、`normal_player_path_verified` 保持預設 false；特定步數與 PNG 雜湊的人工目視核對另記在 RESEARCH-LOG，完成度以本表的已驗證範圍為準。

目前任務是把原版接到 dosgolem 執行。本輪已完成原始 LE 入口至新遊戲戰場的整合啟動驗收；四套件回歸與完整收據核對通過。這個範圍不等於全遊戲相容。後續單位操作、戰鬥、光碟音訊與存讀檔依實際阻塞點分項查證，不重新開啟已通過的啟動切片。

2026-10-12 環境已開放主機存取，`/home/anr2/.local/bin/gh auth status` 成功，使用帳號 `wicanr2`。遠端核對確認 repository 為 private，六筆 Issues 已建立，#1 尚無留言，首次讀取時 repository 尚無預設分支。CD TOC／PLAY 的唯讀資料流查證依 #1 繼續。先前工具政策拒絕的歷程保留在 WORKLOG；原版資料仍不得入 Git。

## 後續擴充的已確認決定

- 原版 DOS 程式繼續負責玩法、狀態與存檔。本次沒有改成 remake。
- 使用者選擇可切換 HD 素材包，逐項替換人物、場景與介面。原版顯示保留作對照與缺圖備援；單純放大原圖不作為 HD 素材包完成標準。
- 第一批語言為繁中、簡中、英文與日文。原文與譯文均以 UTF-8 TSV catalog 管理，穩定 key 的規則依 `/home/anr2/.codex/knowledge-base/local/localization-display-semantic-isolation.md`；翻譯只進顯示路徑，不改原版比較、資源名稱或存檔欄位。
- 桌面版為本次目標。瀏覽器與 WebAssembly 不列入規劃或後備路線。
- 使用者已授權依 [WORKLIST.md](WORKLIST.md) 的功能範圍開工。技術切片仍依證據審查成為 READY；桌面工具版本、完整文字／繪圖擷取位置、HD 美術風格與顯示比例仍待可行性驗證或 prototype。尚未開始正式功能實作，沒有把未知寫成既有能力。
