# 德川家康目前狀態

更新日期：2026-10-14。規則入口：[AGENTS.md](AGENTS.md)。歷程：[WORKLOG.md](WORKLOG.md)。待辦：[WORKLIST.md](WORKLIST.md)。證據：[RESEARCH-LOG.md](RESEARCH-LOG.md)。

## 唯一目前狀態表

| 項目 | 現況 | 證據 |
|---|---|---|
| jsdos 下載 | 完成，214,665,581 bytes | `original/德川家康.jsdos.zip`、[source.json](original/source.json) |
| ZIP 與解包 | CRC 通過；14 筆 ZIP entries，12 個檔案；解包檔共 235,521,461 bytes | `original/source.json` 的逐檔大小與 SHA-256 |
| 原始檔保護 | ZIP 與解包檔設為唯讀；擁有者 UID/GID 1000:1000 | `original/`；最近驗證結果見下方入口 |
| 網站啟動契約 | 掛載 C: 工作目錄、D: CUE 光碟，再由 `PLAY.BAT` 呼叫 `DX5.EXE` | `original/game/.jsdos/dosbox.conf`、`PLAY.BAT` |
| 執行檔格式 | 已證實為 MZ 內嵌 LE；LE header 檔案偏移 `0x2C90` | 原始 bytes；[LE 探針](workplace/leprobe/stdout.log) |
| dosgolem 格式解析 | 完成；40 pages、2 objects、4,831 fixup records；relocation preview 成功 | `workplace/leprobe/stdout.log`，stderr 為空，exit 0 |
| dosgolem 執行本作 | 桌面步進重構後通過原版入口至首場戰場；750,000,000 步、390 筆普通滑鼠事件，只因預算結束。全部 RAM、trace、原生畫面、輸入與 CD PCM 和先前完整驗收一致 | [最近完整收據](workplace/desktop-prototype/headless/startup-probe/latest.json)，SHA-256 `1eb3fe503bac5ecfb0261e0f94cf762a765895d5d4fb9b4a0e1e712ee210ac7c`；[原生戰場畫面](workplace/startup-probe/frame-b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846.png) |
| 通用平台修正 | 在隔離副本實作 CPU 指令形式、DPMI 主機服務轉接與明確啟用的 MCB 配置池；支援標頭、重用、合併與縮放，保留外部預設配置器 | [dosgolem.patch](tools/dosgolem.patch)、[MCB](tools/dpmi-mcb.go)、[回歸](tools/dpmi-mcb_test.go) |
| 平台回歸 | CPU386、machine、dosfile、研究入口四套件通過；涵蓋 104 個地面來源、八項審查素材、75 個劇情來源、診斷滿額、頭像、共用視窗、自然返回、保存／還原、遮擋與備援 | [platform-test.log](workplace/hd-prototype/catalog/terrain/runtime/startup-probe/platform-test.log) |
| 原版時間等待 | DOS 時間與 VGA 回掃共用一微秒／成功指令的虛擬時間；原版自行校準等待迴圈 | 最新收據 `virtual_clock`；[回掃契約](RESEARCH-LOG.md)；未略過指令 |
| 光碟 | AUDIO PREGAP 依 CUE 輸出靜音；有效起點的超長請求在 lead-out 停止，保存原始長度與實際終點。原版末軌表格重疊仍保留，沒有改 EXE 或表格 | [cdrom.go](tools/cdrom.go)、[cdda.go](tools/cdda.go)、[因果與契約](RESEARCH-LOG.md) |
| CDDA 技術驗證 | 完成 PLAY／STOP／RESUME、狀態、自然結束、分塊、跨軌、PREGAP、極大長度與非法起點測試；十五條音軌全段與 BIN 一致。正常流程八次選曲無拒絕，輸出 19,818,728 個立體聲樣本框。桌面與人耳確認未完成 | [CDDA 測試](tools/cdda_test.go)、`workplace/startup-probe/cdda-boundary-test.log`、[獨立核對](workplace/startup-probe/cdda-verification.json)；79,274,912 bytes 逐段與 BIN 及時間公式相符 |
| CD 只讀診斷 | 開關對照通過；同一 3,000,000 步的完整記憶體、畫面、停止位置、中斷計數與原始 trace 一致；未修改原版或 DPMI 回傳契約 | [startup-probe.go](tools/startup-probe.go) 的 `-cd-trace`；`workplace/cd-trace/trace-on.json`、`trace-off.json` |
| 滑鼠與 BIOS | 普通滑鼠已推進選單、角色建立與敘事；已接原版遮罩與畫面游標層，未實作 TSR 的 VRAM 背景讀回 | [video.go](tools/video.go)；`latest.json` 的輸入與邊界畫面 |
| VGA／VESA 顯示 | 原版模式 12h 與 CRTC 自行設定 640×400，開場、標題與主選單已目視確認；VBE 分頁子集另有測試 | [原生標題與選單](workplace/startup-probe/frame-7df7211c989bca30c3e82077011a96e7ef7142c9029c515aeb5dda5e461697e7.png)、[video.go](tools/video.go) |
| DOSBox 輔助基準 | 可顯示原版標題；實際 DOS/4GW 的 MCB 配置可重用，段號相差 2。第 1、2 軌 MSF 起點保留，控制欄位仍遭重疊覆蓋；未作音訊驗證 | [25s.png](workplace/dosbox-aux/25s.png)、[aux-cd-trace.py](tools/aux-cd-trace.py)、`workplace/cd-trace/aux-toc/manifest.json`；詳見 RESEARCH-LOG |
| 正常玩家路徑、存讀檔與音訊 | 原版入口、角色建立、敘事、選項、出陣與首場戰場顯示保持通過，原版選曲／換曲與 PCM 同步通過；單位操作、戰鬥、存讀檔及桌面聽覺未驗證 | [普通輸入時間表](tools/player-path.json)；精確畫面、音訊雜湊與範圍見 RESEARCH-LOG |
| remake | 排除 | 使用者明確指定原版與 dosgolem 整合 |
| 後續功能範圍 | CDDA 裝置與 PCM 已實作；桌面播放、繁中／簡中／英文／日文與可切換 HD 素材包待後續切片 | 已確認範圍與驗收見 [WORKLIST.md](WORKLIST.md) |
| 後續使用介面 | 桌面版；全遊戲 HD 後打包 Windows、Linux、macOS；瀏覽器排除 | 使用者已確認；目前只驗證 Linux 容器桌面 |
| HD 風格試作 | B 戰國手繪風。75 個劇情來源、捲軸、紅金框架、共用視窗及 81 張頭像的共同繪圖端已接入；正常路徑抽測九張劇情與四張頭像。F6 與頂端列切換、逐項缺圖備援通過 | [來源與正常路徑總表](workplace/hd-prototype/catalog/story-expanded/verification-summary.json)、[HD 驗證](workplace/hd-prototype/catalog/story-expanded/desktop/verification-portraits.json)、[缺圖驗證](workplace/hd-prototype/catalog/story-expanded/desktop-missing/verification-portraits-missing.json) |
| HD 素材定位 | DAT 201 筆、112 個 OZM。靜態解碼 107 筆，17 個正常路徑來源的平面與索引一致；五筆尾端少一 byte，保留未知。KAO.GRP 的 81 張頭像索引與繪圖端已確認 | [解碼器](tools/hd-ozm-probe.py)、[頭像擷取](tools/hd-kao-extract.py)、[覆蓋清單](workplace/hd-prototype/catalog/art-coverage.json) |
| 全遊戲 HD 覆蓋 | 製作中。60 張已審查劇情、城郭、軍議及結局圖對應 75 個 OZM，另有共用視窗、捲軸及 81 張頭像。75 個劇情來源已接共同繪圖端。MAP_CHIP 已細分 104 地面、26 物件、13 單位來源格，含空白。地面八項已審查 HD 接入，首戰抽測六種、50 個完整矩形；十種單位為未接入草稿。其他介面、動畫、地形及單位待完成，正式打包尚未開始 | [覆蓋清單](workplace/hd-prototype/catalog/art-coverage.json)、[圖片與提示詞](assets/hd-b/README.md)；資源筆數不等於圖集圖塊數 |
| 戰場地面 HD | 地面切片通過。104 個來源對照、八項審查素材，正常首戰抽測 0／2／6／13／15／22，共 50 個完整矩形；62／67 只有繪製來源與呈現測試。HD／缺圖各自然完成 750M、390 筆普通輸入，完整 RAM、15 個原版欄位、原生畫面與 CD PCM 符合基準，三次切換不改原版。遮擋、不同 DAC 或缺圖整格回原圖，物件／單位與其背景待後續 | [驗收總表](workplace/hd-prototype/catalog/terrain/runtime/verification-summary.json)、[真實視窗](workplace/hd-prototype/catalog/terrain/runtime/desktop/gui-terrain-enabled.png)、[生成圖與提示詞](assets/hd-b/terrain/) |
| HD 劇情與頭像驗證 | 先前 75 個來源通過資料與共同繪圖端測試，其中 A-0 至 A-7、GUNGI、頭像 76..79 的真實視窗切換與缺圖通過，地名保留原版。該版本兩側各自然跑完 750M，RAM、原生畫面、輸入與 CD PCM 符合基準；其餘 66 個劇情來源與其餘頭像未逐張在正常路徑抽測 | [HD 收據](workplace/hd-prototype/catalog/story-expanded/desktop/desktop-portraits.json)、[缺圖收據](workplace/hd-prototype/catalog/story-expanded/desktop-missing/desktop-portraits-missing.json)；每次 21 次切換的原版狀態不變 |
| 桌面呈現與普通滑鼠 | Linux 容器內 Ebitengine 2.9.9 試作通過。正常入口與普通時間表走到首張敘事；視窗縮為 1000×700 後，普通 X11 左鍵正確對應原版 `(300,171)` 並開始新遊戲。支援 F6 切換、F7 暫停、F9 擷取；現行功能列與關窗操作見下一列；一般鍵盤、桌面音訊及其他 OS 未驗收 | [建置入口](tools/run-desktop.sh)、[普通滑鼠核對](workplace/desktop-prototype/verification-mouse.json)、[新遊戲開場](workplace/desktop-prototype/gui-mouse-after-click.png)；操作及限制見 AGENTS |
| 劇情捲軸 HD | 完成此介面切片。VISUAL.OZM 原版平面與捲軸底圖已核對，紙張、捲邊及外框以 HD 替換；文字前景逐點保留。兩頁劇情保留 24324／4036 個呈現前景像素。HD、缺捲軸圖及只有捲軸 HD 三模式的 RAM、原生畫面與 CD PCM 均符合同一 150M 基準；頂端功能列整合亦通過。字型重繪及其他介面仍待後續 | [驗收總表](workplace/hd-prototype/scroll/verification-summary.json)、[實際畫面](workplace/hd-prototype/scroll/desktop/gui-scroll-enabled.png)、[第二頁](workplace/hd-prototype/scroll/desktop/gui-scroll-second-enabled.png)、[素材](workplace/hd-prototype/scroll/scroll-hd-b-v1.png)、[完整提示詞](workplace/hd-prototype/scroll/scroll-hd-b-v1.prompt.txt) |
| 劇情上方框架 HD | 完成此切片。完整 HD、缺框架圖、只有框架 HD 三模式及只有框架時的頂端列通過；中央圖與文字維持各自圖層。四次 150M 的原版 RAM、畫面、輸入及 PCM 符合基準 | [驗收總表](workplace/hd-prototype/scroll/frame/verification-summary.json)、[實際畫面](workplace/hd-prototype/scroll/frame/desktop/gui-frame-enabled.png) |
| 共用視窗 HD | 完成已抽測切片。九圖塊金框與紙張沿用原版幾何，文字及邊框前景保留。只有視窗、搭配頭像、缺圖、確認及能力視窗、頂端列五種驗證通過，每次 150M 的完整 RAM、原生畫面、輸入與 PCM 符合基準；其他視窗與平台未逐項驗收 | [驗收總表](workplace/hd-prototype/catalog/window/verification-summary.json)、[角色選擇](workplace/hd-prototype/catalog/window/desktop-window/gui-window-enabled.png)、[確認框](workplace/hd-prototype/catalog/window/desktop-window-modal/gui-window-modal-confirmation-enabled.png) |
| 生成美術的私人保存 | 使用者於 2026-10-14 明確授權生成圖保存於 private repo。目前保存 183 張 PNG、95 份提示詞及來源／裁切索引，包含修正前草稿。原版擷取與執行收據保持本機 | [HD 素材](assets/hd-b/README.md)、[索引](assets/hd-b/index.json)、[保存工具](tools/archive-hd-art.py)；遠端結果見 WORKLOG |
| 頂端隱藏功能列 | Linux 容器試作通過。參考 softworld_san：頂端滑鼠展開、移開收起，Esc 固定／收起，Shift+Esc 離開。提供原版／HD 圖像、繼續／暫停、截圖／離開。顯示與收起手勢期間隔離原版步進及輸入；視窗擴高而保持遊戲比例。HD、缺圖兩次 150,000,000 步的完整 RAM、原生畫面與 PCM 符合原基準；普通滑鼠未誤觸，縮放後仍可開始新遊戲 | [驗收總表](workplace/desktop-prototype/toolbar/verification-summary.json)、[功能列畫面](workplace/desktop-prototype/toolbar/windowbar-hd-hd.png)、[原圖選單](workplace/desktop-prototype/toolbar/windowbar-hd-display-menu.png)、[普通滑鼠驗證](workplace/desktop-prototype/toolbar/verification-windowbar-mouse.json)；四語及音訊開關待功能接妥 |
| private GitHub repository 與 Issues | 主機 gh 已確認 [wicanr2/tokugawa-dosgolem](https://github.com/wicanr2/tokugawa-dosgolem) 為 private，六筆 Issues 已建立；#2 技術切片驗收後已關閉，HD 接續 #5，桌面與聽覺驗收留 #3 | [CDDA 完成記錄](https://github.com/wicanr2/tokugawa-dosgolem/issues/2#issuecomment-6121828738)、[DESKTOP #3](https://github.com/wicanr2/tokugawa-dosgolem/issues/3)、[HD #5](https://github.com/wicanr2/tokugawa-dosgolem/issues/5)；Issue 即時狀態直接查 GitHub |
| Git 提交範圍 | 已建立本機 `main` 並推送自製工具、測試、文件及使用者授權的生成美術；原版與執行產物排除 | `.gitignore` 與 Git 實際檔案清單；遠端提交以 GitHub 為準 |
| GitHub 建立工具驗證 | shell 語法、六筆本文擷取與隔離假 API 測試通過；遠端未實測 | `--prepare` 不呼叫 GitHub；查重、相依 URL、登入／列舉失敗、公有同名 repo 與重複 Issues 的停止行為已測。歷程見 WORKLOG |
| Docker 收尾 | 地面切片的完整 HD 與缺圖驗證容器均已結束，無本專案殘留；沿用既有 image | docker ps -a --filter name=tokugawa 為空；擁有權核對見 WORKLOG |

## 固定來源與版本差異

| 檔案 | SHA-256 |
|---|---|
| `德川家康.jsdos.zip` | `e35bb236ae48f095a6dbda3236a59e5a5b903c82fe9d5564233c15c1793b425a` |
| `DX5.EXE` | `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a` |
| `DX5.EXE.BAK` | `95bffc6fac92974bc678ddefa6e86f601676e78dd2c2d0ff0b7ac2da189512f9` |
| `DX5.bin` | `3f511d4c2fb819f950d881697f4e771d37ccfa2b0fd9eca6c4bb48fff6be2448` |

來源頁、API 回應、最終下載網址與其餘檔案雜湊集中在 `original/source.json`，不以遊戲名稱辨認 binary 版本。

- 已證實：`DX5.EXE` 與 `.BAK` 在檔案偏移 `0x17459`、`0x17582` 分別為 `3F` 與 `15`，以及調色盤範圍 `0x2FAD4–0x2FBF3` 存在差異。包內 UTF-8 修正說明記載亮度調整方法。保留網站使用的 EXE 作為此次基準。
- 已證實：檔案偏移 `0x19395` 的 active／BAK bytes 為 `EB`／`74`。IDA 新證據確認網站 EXE 在此無條件略過 TOC 末軌是否等於 16 的失敗出口；勘誤與原始定位見 RESEARCH-LOG。不宣稱所有差異只有亮度修正，也不切換 `.BAK`。
- 已證實：`DX5.cue` 宣告 16 軌，第一軌 `MODE1/2352`，其餘 15 軌 `AUDIO`。原版首筆 PLAY 自行選取第 10 軌，LBA 51,515、6,811 個磁區；PCM 解碼通過，人耳確認待桌面驗收。
- 未知：`.BAK` 的版本來源、額外差異用途，以及尚未測到的戰鬥、存讀檔與音訊相容性。

## 探針版本與限制

| 項目 | 值 |
|---|---|
| dosgolem 路徑 | `/home/anr2/cht/dosgolem` 為外部工作樹；本輪固定副本為 `workplace/hd-prototype/scroll/dosgolem-base/`，執行時唯讀掛載 |
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

主機 `/home/anr2/.local/bin/gh` 已登入 `wicanr2`；private repository、`main` 與六筆 Issues 已建立。Issue 現況直接查 GitHub，先前存取受阻與建立歷程留在 WORKLOG。原版資料仍不得入 Git。

## 後續擴充的已確認決定

- 原版 DOS 程式繼續負責玩法、狀態與存檔。本次沒有改成 remake。
- 使用者選擇可切換 HD 素材包，逐項替換人物、場景與介面。原版顯示保留作對照與缺圖備援；單純放大原圖不作為 HD 素材包完成標準。
- 使用者已選 B 戰國手繪風，依同場景比較的自然地形與手繪人物、介面方向製作。生成概念圖不作正式版面或幾何來源，原版邏輯座標仍為 640×400。
- 第一批語言為繁中、簡中、英文與日文。原文與譯文均以 UTF-8 TSV catalog 管理，穩定 key 的規則依 `/home/anr2/.codex/knowledge-base/local/localization-display-semantic-isolation.md`；翻譯只進顯示路徑，不改原版比較、資源名稱或存檔欄位。
- 使用者確認頂端功能列後，另指定捲軸介面也要 HD；劇情捲軸已接入既有 HD 開關，維持原版文字及位置。
- 使用者指定頂端隱藏功能列參考 `/home/anr2/cht/softworld_san`，目前已接入顯示、執行與工具三欄；四語與音訊待功能實作後接入。
- 桌面版為本次目標。使用者已確認全遊戲美術 HD 完成後打包 Windows、Linux、macOS；各平台建置與啟動驗收分開記錄。瀏覽器與 WebAssembly 不列入規劃或後備路線。
- 使用者指定完成 CD 音訊後先執行 HD 化。四語需求保留，實作順序移到 HD 首個場景之後；桌面畫面與音訊試作作為必要依賴一起推進。
- 使用者已授權依 [WORKLIST.md](WORKLIST.md) 的功能範圍開工。MCB 與 CDDA 切片已有 READY 契約並實作；HD 美術風格已選 B。桌面工具版本、完整文字／繪圖擷取位置與顯示比例仍待可行性驗證或 prototype。
