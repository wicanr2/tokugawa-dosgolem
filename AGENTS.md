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
| [tools/hd-asset-probe.py](tools/hd-asset-probe.py) | 唯讀盤點 DAT 的候選資源目錄；保存原始欄位、檔案偏移與逐項雜湊，不作正式 HD 識別器 |
| [tools/ida-asset-probe.py](tools/ida-asset-probe.py) | IDA 9.4 的資源字串、直接交叉參照及有界函式證據；保留原名與位址 |
| [tools/ida-art-consumers.py](tools/ida-art-consumers.py) | IDA 9.4 的圖像資料讀取端 caller 視窗，只匯出原始定位與指令；從正式資料庫的隔離副本執行 |
| [tools/hd-trace.go](tools/hd-trace.go)、[tools/hd-trace_test.go](tools/hd-trace_test.go) | 固定 EXE 的唯讀素材載入觀測及記憶體不變、錯誤界限、事件上限測試；不替換素材 |
| [tools/hd-extract.go](tools/hd-extract.go) | 從正常路徑 RAM 擷取 A-0.OZM，核對四平面雜湊與原生畫面矩形，輸出本機 PNG 及收據；不是正式素材包格式 |
| [tools/hd-capture.go](tools/hd-capture.go) | `-hd-export` 在原版載入返回時輸出已核對平面的索引圖、當時 DAC 與收據；沒有 guest 寫入，不代表繪圖或遮擋已驗收 |
| [tools/hd-ozm-probe.py](tools/hd-ozm-probe.py) | 依 IDA 9.4 原始指令做離線資源盤點，只接受獨立解碼命令；遇到 VGA 來源命令明確拒絕。以正常路徑四平面與索引雜湊交叉核對 |
| [tools/hd-kao-extract.py](tools/hd-kao-extract.py) | 擷取 81 張 80×80 KAO.GRP 頭像、九頁來源圖集與收據；核對正常路徑載入的完整資料及第 76 張的 6400 原生像素 |
| [tools/hd-kao-prepare.py](tools/hd-kao-prepare.py) | 依已檢視的實際格線裁切九張 HD 圖集，補黑底保持人物比例；輸出 `workplace/hd-prototype/catalog/kao/selected/` 的 81 張素材、manifest 與總覽 |
| [tools/hd-art-inventory.py](tools/hd-art-inventory.py) | 重生 `workplace/hd-prototype/catalog/art-coverage.json`，區分已製作、逐項替換、正常路徑驗證與未知素材；資源數不當成圖集完成百分比 |
| [tools/hd-frame-prepare.py](tools/hd-frame-prepare.py) | 既有手繪 VISUAL 上方框架依原版矩形分區縮放；輸出 `workplace/hd-prototype/scroll/frame/` 的來源、HD 圖與幾何收據，遊戲內驗證另列 |
| [tools/hd-frame.go](tools/hd-frame.go)、[tools/hd-frame_test.go](tools/hd-frame_test.go) | VISUAL 上方紅金框架的來源核對與呈現；中央劇情圖及下方捲軸保持獨立，遮擋時完整備援 |
| [tools/hd-window-extract.py](tools/hd-window-extract.py) | 核對 WINDOW.OZM 九個 16×16 圖塊、原版 buffer 與正常視窗；輸出 `workplace/hd-prototype/catalog/window/` 的來源、原始圖塊、手繪參考與來源收據 |
| [tools/hd-window.go](tools/hd-window.go)、[tools/hd-window_test.go](tools/hd-window_test.go) | 共用視窗繪製及保存／還原的唯讀觀測、九圖塊 HD 呈現與原版前景保留；READY 契約及驗收見 RESEARCH-LOG |
| [tools/verify-hd-window.py](tools/verify-hd-window.py) | 核對五次共用視窗 GUI 收據、完整 RAM、原版狀態及建置來源；固定 Python Docker 中專案唯讀掛 `/project`、`workplace/hd-prototype/catalog/window/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-window.py` |
| [tools/archive-hd-art.py](tools/archive-hd-art.py) | 依 2026-10-14 使用者授權保存生成的 HD 圖與提示詞；固定 GUI Docker 內以專案掛 `/project` 可寫執行，`--verify-only` 核對索引與檔案 |
| [assets/hd-b/README.md](assets/hd-b/README.md)、[圖片索引](assets/hd-b/index.json) | 私人儲存庫內的生成圖、生成圖裁切素材與提示詞。使用者已明確授權保存；不是正式發行包或公開散布授權 |
| [tools/hd-kao-profiles.go](tools/hd-kao-profiles.go) | KAO.GRP 原版 0..80 索引的逐張指紋，不以推測武將姓名命名 |
| [tools/hd-kao.go](tools/hd-kao.go)、[tools/hd-kao_test.go](tools/hd-kao_test.go) | 原版頭像繪製的唯讀參數／返回觀測與來源、遮擋、缺圖備援；原版全畫面複製撤銷舊頭像 |
| `workplace/hd-prototype/catalog/kao/` | 本機原版頭像、HD 製作副本與來源證據；生成圖依使用者授權同步至 assets/hd-b |
| `workplace/hd-prototype/catalog/` | 本機 HD 覆蓋盤點、原版擷取、格式診斷與驗證；已生成圖像另存 assets/hd-b |
| [tools/hd-preview.go](tools/hd-preview.go)、[tools/hd-preview_test.go](tools/hd-preview_test.go) | 單張 A-0 的離線原圖／HD／缺圖預覽與範圍保護測試；不接管遊戲或代表桌面執行中切換 |
| [tools/hd-layer.go](tools/hd-layer.go) | 離線預覽與桌面試作共用的單張素材呈現與來源像素保護 |
| [tools/hd-portraits.go](tools/hd-portraits.go) | 由 hd-story-prepare.py 生成的 75 個劇情來源、40／41 欄指紋、原圖、HD 檔名與圖片文字矩形；正常路徑抽測 A-0 至 A-7 及 GUNGI |
| [tools/hd-story-prepare.py](tools/hd-story-prepare.py) | 核對已審查劇情圖的 DAT、四平面、精確共用、可見裁切及圖片文字矩形，產生 `catalog/story-expanded/` 原圖與 profiles.json；固定 GUI Docker，專案掛 `/project`、原版唯讀掛 `/original`、catalog 可寫掛 `/output`，`--go-output` 指定對照表輸出 |
| `workplace/hd-prototype/catalog/story-expanded/` | 75 個劇情來源的準備、隔離建置及驗證；53 來源歷史收據保留於 catalog/story |
| [tools/hd-terrain-extract.py](tools/hd-terrain-extract.py) | 核對 MAP_CHIP 的正常 RAM 四平面、IDA 圖塊讀取端及來源索引，細分 104 地面、26 物件、13 單位來源格；不是 HD 素材或正式遊戲索引集合。固定 GUI Docker，專案唯讀掛 `/project`、原版唯讀掛 `/original`、catalog 可寫掛 `/output`，執行 `python3 /project/tools/hd-terrain-extract.py` |
| `workplace/hd-prototype/catalog/terrain/` | 原版地形、物件與單位來源格、總覽及擷取收據；原版研究輸出只留本機。art/ 保存生成素材與完整提示詞，再依既有授權同步至 assets/hd-b/terrain |
| [tools/hd-terrain-trace.go](tools/hd-terrain-trace.go)、[測試](tools/hd-terrain-trace_test.go) | `-hd-terrain-trace` 的唯讀地面入口暫存器、來源平面與自然返回診斷，上限 512 事件；未參與桌面呈現。沿用 run-probes 的固定 Docker，在 `catalog/terrain/` 的獨立輸出執行正常 750M 時間表 |
| [tools/hd-terrain-prepare.py](tools/hd-terrain-prepare.py)、[來源對照](tools/hd-terrain-profiles.go) | 以正常 DAC 準備 104 個來源及審查後素材的編譯期指紋，輸出 terrain/prepared；固定 GUI Docker，專案掛 `/project` 可寫、terrain 掛 `/output` 可寫，執行 Python 工具後 gofmt 對照表；原版擷取保持原樣 |
| [tools/hd-terrain.go](tools/hd-terrain.go)、[測試](tools/hd-terrain_test.go) | 已核對的地面入口／返回、完整平面指紋與整格像素資格；遮擋、缺圖或調色盤不符均保留原圖，契約見 RESEARCH-LOG |
| [tools/verify-terrain-evidence.py](tools/verify-terrain-evidence.py) | 核對地面 104 來源、八項審查素材與兩次正常 750M GUI；固定 Python Docker，專案唯讀掛 `/project`、terrain/runtime 可寫掛 `/output`，執行 `python3 /project/tools/verify-terrain-evidence.py` |
| `workplace/hd-prototype/catalog/terrain/runtime/` | 地面切片的隔離建置及 GUI 收據；trace/ 保存呈現前的唯讀觀測與探針來源快照 |
| [tools/hd-story_test.go](tools/hd-story_test.go) | 批次劇情來源、原版載入／繪製返回、診斷上限、裁切、地名、缺圖與遮擋測試；來源測試不作正常劇情實玩證據 |
| [tools/verify-hd-story.py](tools/verify-hd-story.py) | 核對兩次 750M 收據、75 個來源、實際素材、二進位與建置輸入；固定 Python Docker 中專案唯讀掛 `/project`、`catalog/story-expanded/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-story.py` |
| [tools/hd-live.go](tools/hd-live.go)、[tools/hd-live_test.go](tools/hd-live_test.go) | 只讀原版素材、A-0 矩形與 VISUAL 全畫面繪製入口／返回，限制桌面替換資格 |
| `workplace/hd-prototype/scroll/dosgolem-base/` | 固定 a9714eb 的獨立來源副本，執行時唯讀；外部 dosgolem 前進時保留既有驗證基準 |
| [tools/presentation.go](tools/presentation.go)、[tools/presentation_test.go](tools/presentation_test.go) | 桌面試作的步進、畫面、普通滑鼠及原版狀態核對介面與游標層測試；預設無頭 |
| [tools/desktop.go](tools/desktop.go) | 只在 desktop 建置標籤啟用的 Ebitengine 視窗、HD 開關與證據擷取 |
| [tools/hd-scroll-extract.go](tools/hd-scroll-extract.go) | 固定正常路徑的 VISUAL.OZM 原始平面、捲軸底圖與文字區核對 |
| [tools/hd-scroll-prepare.go](tools/hd-scroll-prepare.go) | 從生成的完整畫布取出既定捲軸區，保存產物及提示詞雜湊 |
| [tools/hd-scroll.go](tools/hd-scroll.go)、[tools/hd-scroll_test.go](tools/hd-scroll_test.go) | 捲軸 HD 呈現、原版前景保留、來源及缺圖備援 |
| [tools/verify-hd-scroll.py](tools/verify-hd-scroll.py) | 真實視窗的捲軸原圖／HD 切換、文字與原版狀態核對 |
| `workplace/hd-prototype/scroll/` | 本機捲軸來源、製作副本與隔離驗證；生成素材及提示詞另存 assets/hd-b/scroll |
| [tools/windowbar.go](tools/windowbar.go)、[tools/windowbar_test.go](tools/windowbar_test.go) | 頂端隱藏列的狀態、幾何、點陣字型與純畫面測試 |
| [tools/desktop-windowbar.go](tools/desktop-windowbar.go) | 桌面功能列輸入、視窗高度與原版輸入隔離 |
| [tools/toolbar-font.hex.gz](tools/toolbar-font.hex.gz)、[tools/LICENSE-toolbar-font.txt](tools/LICENSE-toolbar-font.txt) | 參考專案的完整 Unifont 點陣字型與原授權全文；執行時讀取，不使用原版遊戲字模 |
| [tools/verify-windowbar.py](tools/verify-windowbar.py) | 真實 X11 頂端顯示、固定、選單、鍵盤與誤觸隔離驗證 |
| `workplace/desktop-prototype/toolbar/` | 本次功能列隔離建置、GUI 收據與截圖；保留先前桌面驗證產物 |
| [tools/run-desktop.sh](tools/run-desktop.sh) | 固定既有 GUI image 的隔離建置入口；原版與外部 dosgolem 唯讀 |
| [tools/desktop.mod](tools/desktop.mod)、[tools/desktop.sum](tools/desktop.sum) | 桌面試作的 Ebitengine 2.9.9 與傳遞依賴鎖定；不修改外部 dosgolem go.mod |
| [tools/verify-desktop.py](tools/verify-desktop.py) | 有界 Xvfb 真實視窗／切換與普通滑鼠驗證、像素比較及程序清理 |
| `workplace/desktop-prototype/headless/` | 步進重構後的完整無頭玩家路徑，獨立於 GUI 試作收據 |
| `workplace/desktop-prototype/` | 桌面試作的建置、模組鎖定、正常路徑收據與 GUI 驗證；不是正式交付目錄 |
| `workplace/hd-prototype/` | 可丟棄 HD 風格概念圖、完整提示詞、候選資源盤點及 IDA 證據；不作已接入遊戲的證明 |
| `workplace/hd-prototype/asset-capture/` | 正常玩家輸入前綴的原版素材 RAM、畫面與獨立擷取收據；與完整路徑收據分開保存 |
| [tools/dosgolem.patch](tools/dosgolem.patch) | 固定 dosgolem 基準的通用平台修正，尚未回填外部工作樹 |
| [tools/dpmi-mcb.go](tools/dpmi-mcb.go) | 可明確啟用的 32 位 DPMI MCB 配置、釋放重用與縮放；契約見 RESEARCH-LOG 的 DOS MCB 節 |
| [tools/dpmi-mcb_test.go](tools/dpmi-mcb_test.go) | MCB 標頭、重用、合併、界限、錯誤與描述子耗盡測試 |
| [tools/cpu386-integration_test.go](tools/cpu386-integration_test.go) | 平台修正的獨立回歸測試 |
| [tools/run-probes.sh](tools/run-probes.sh) | 在 Docker 中套用修正、跑平台測試並重生啟動收據 |
| [tools/cdrom.go](tools/cdrom.go) | 真實 CUE/BIN 掛載與 MSCDEX 查詢子集 |
| [tools/cdrom_test.go](tools/cdrom_test.go) | 光碟幾何、來源雜湊與傳輸界限測試 |
| [tools/cdda.go](tools/cdda.go) | 依虛擬時間推進的 CDDA 狀態與 PCM 串流，包含 PREGAP 靜音與 lead-out 停止 |
| [tools/cdda_test.go](tools/cdda_test.go) | 播放／停止／續播、換曲、自然結束、分塊獨立性、間隙、超長請求與十五軌 BIN 測試 |
| `workplace/startup-probe/cdda-boundary-test.log`、`cdda-verification.json` | 十五軌界限測試與正常流程全部 PCM 的獨立來源核對；驗證範圍及雜湊見 RESEARCH-LOG |
| [tools/aux-cd-trace.py](tools/aux-cd-trace.py) | 固定 DOSBox-X image 中查證包內 DOS/4GW 的配置與 TOC 回寫，只作輔助證據 |
| [tools/video.go](tools/video.go) | 標準 VGA 埠轉接、模式切換、擴充索引隔離與原生畫面擷取 |
| [tools/video_test.go](tools/video_test.go) | VGA 暫存器隔離與 DAC 回歸測試 |
| [tools/vbe.go](tools/vbe.go) | VBE 分頁 VRAM 與 BIOS 子集 |
| [tools/vbe_test.go](tools/vbe_test.go) | VBE 資訊、分頁與界限測試 |
| [tools/keyboard.go](tools/keyboard.go) | 共用 BDA 按鍵緩衝的 DOS／BIOS 鍵盤子集 |
| [tools/keyboard_test.go](tools/keyboard_test.go) | 按鍵查詢、消耗與空緩衝測試 |
| [tools/player-path.json](tools/player-path.json) | 原版入口、角色建立與敘事推進的普通滑鼠時間表；已驗證範圍見 RESEARCH-LOG |
| [tools/setup-github.sh](tools/setup-github.sh) | 主機 gh 建立 private repo 與六筆 Issues；Docker 擷取本文，`--prepare` 不存取 GitHub |
| `workplace/github-setup/` | 從 WORKLIST 重生的 Issue 本文及已授權進度留言草稿，不作遠端狀態台帳 |
| `workplace/cd-trace/` | CD TOC／PLAY 的隔離診斷收據、IDA 窄範圍證據與可重生工具副本；`aux-toc/` 保存額外 TOC 查詢的 DOSBox-X 輔助證據 |
| [tools/dos_test.go](tools/dos_test.go) | DOS EXEC 缺檔錯誤與不假執行測試 |
| [RESEARCH-LOG.md](RESEARCH-LOG.md) | 輸入雜湊、位址空間、平台契約與研究證據 |
| [README.md](README.md) | 專案用途、穩定入口與執行方式摘要 |
| [WORKLIST.md](WORKLIST.md) | dosgolem 接入待辦、CD 音樂／四語／HD 規劃、完成條件與待送出的 GitHub 建立內容 |
| [workplace/](workplace/) | 探針、快取、截圖、實驗與其他中間產物 |
| `workplace/startup-probe/state-*` | 各次執行的可寫檔案副本；由既有 DirectoryOverlayFiles 建立，不回寫原版 |
| `workplace/startup-probe/frame-<SHA-256>.png` | dosgolem 原生 VRAM 與 DAC 生成的畫面；是否為遊戲畫面須另行核對 |
| `workplace/startup-probe/audio-<SHA-256>.pcm` | 本機 CDDA 原始 PCM，44.1 kHz、16-bit little-endian、雙聲道；閒置區間依收據事件還原，不加入 Git |
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

## 全美術 HD 研究入口

`workplace/hd-prototype/catalog/terrain/backgrounds/` 保存原版圖塊繪製前後的唯讀背景診斷、來源及後續 GUI 驗證。`tools/hd-backdrop.go`／`tools/hd-backdrop_test.go` 的 `-hd-backdrop-trace` 只在原版 sub_1F9D0 入口與自然返回讀取 native snapshot、原始四平面及 DAC，輸出有界的原始背景／來源／返回圖。`tools/verify-hd-backdrops.py` 獨立核對十一項來源、原版透明合成、十五欄基準、完整 RAM 與當次來源快照。沿用固定 GUI image，專案唯讀掛 `/project`，該目錄可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-backdrops.py`。READY 呈現契約見 RESEARCH-LOG；診斷收據本身不代表 HD GUI 已驗收。

該目錄的 `runtime/` 是後續呈現建置與測試；`desktop/`、`desktop-only/`、`desktop-missing/` 保存四個樹叢的三模式真實視窗與來源快照。頂層診斷及 `sprite-backgrounds/` 不覆寫。`tools/verify-hd-backdrop-gui.py` 核對三模式、原版狀態與實際擷取背景，使用固定 GUI image，該目錄可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-backdrop-gui.py`。

`workplace/hd-prototype/catalog/terrain/objects/` 保存物件／單位的唯讀繪製順序研究、來源遮罩、隔離建置與 GUI 驗證。舊地面收據留在 `terrain/trace/` 及 `terrain/runtime/`，不覆寫。`-hd-terrain-trace` 同時記錄原版 `sub_1FC07` 地面及 `sub_1F9D0` 透明圖塊的原始暫存器、自然返回與平面指紋；診斷不決定呈現資格。

`workplace/hd-prototype/catalog/terrain/trees/` 保存 objects-000 樹木的背景覆蓋研究、透明素材幾何、可丟棄預覽與獨立 GUI 驗證。原版前景來源沿用 `terrain/objects/sources/`，該輪收據保持不變。`tools/hd-terrain-tree-prepare.py` 核對原始順序、完整背景及實際游標，生成樹木素材指紋與裁切幾何；固定 GUI image 內專案掛 `/project` 可寫、`terrain/trees/` 掛 `/output` 可寫後執行。輪廓之外的低 alpha 雜點只在明示裁切範圍外排除，生成原始 PNG 保留。

`tools/hd-terrain-tree-profile.go` 為準備工具生成的選定 v2 SHA 與裁切幾何。`tools/hd-terrain-sprites.go`／`tools/hd-terrain-sprites_test.go` 實作完整背景核對、原版輪廓移除、透明合成與其他可見前景保留。只批准 objects-000，介面邊緣或未知背景備援；準備工具更新後在固定 Go image 內對 profile 執行 gofmt。

`tools/verify-desktop.py --mode trees`／`trees-only`／`trees-missing` 驗證樹叢三模式，需先使用 `tools/run-desktop.sh` 建置至 `terrain/trees/` 並將相同二進位、engine 與驗證器快照放入各自 desktop 輸出。`tools/verify-hd-trees.py` 在固定 GUI image 中核對三模式、來源、建置輸入與完整原版狀態；專案唯讀掛 `/project`、`terrain/trees/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-trees.py`。既有非樹叢 GUI 模式明確停用樹叢素材，保留各切片驗證的隔離範圍。

[tools/hd-terrain-object-prepare.py](tools/hd-terrain-object-prepare.py) 以正常 750M 的唯讀繪製順序及既有 IDA 遮罩證據，核對原生地面／透明前景合成，準備 39 個原版物件／單位來源及 [tools/hd-terrain-sprite-profiles.go](tools/hd-terrain-sprite-profiles.go) 指紋。固定 GUI image 內專案掛 `/project` 可寫、`catalog/terrain/objects/` 掛 `/output` 可寫，執行 `python3 /project/tools/hd-terrain-object-prepare.py`；只輸出原始來源遮罩，不啟用單位 HD 草稿。

[tools/verify-terrain-evidence.py](tools/verify-terrain-evidence.py) 加 `--foreground` 核對 `terrain/objects/` 的 HD／缺圖正常 GUI、39 個來源、原版狀態及前景像素。固定 GUI image 內專案唯讀掛 `/project`、`terrain/objects/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-terrain-evidence.py --foreground`。不加旗標則核對舊地面切片，須使用該次來源快照。來源準備後用固定 Go image 對生成的 profiles 執行 gofmt，不在主機執行 Go。

樹叢接入後，現行正常流程驗證入口改為 `tools/verify-hd-trees.py`。重核先前原版透明前景切片時，於上述命令加 `--runtime-source-root /project/workplace/hd-prototype/catalog/terrain/objects/desktop/source-snapshot --verify-only`，核對當次來源且保留原總表，不以目前原始碼誤判歷史證據失效。

唯一覆蓋清單為 `workplace/hd-prototype/catalog/art-coverage.json`，由 `tools/hd-art-inventory.py` 重生。圖集資源數不代表內部圖塊完成數。`tools/hd-kao-extract.py` 擷取原版來源；`tools/hd-kao-prepare.py --root /output/kao` 裁切已保存的手繪圖集。這些是本機研究產物，正式素材包格式另列。

沿用固定 GUI image，內含 Pillow 9.4.0。掛載專案至 `/project` 唯讀、`original/game/` 至 `/original` 唯讀、`workplace/hd-prototype/catalog/` 至 `/output` 可寫，再依序執行上述擷取、裁切及覆蓋工具。Docker 的 UID、資源、網路與路徑檢查依本檔「Docker 邊界」。頭像桌面旗標為 `-hd-kao-source-dir` 與 `-hd-kao-dir`，預設分別讀 `catalog/kao/` 及 `catalog/kao/selected/`。原版逐張指紋保存在 `tools/hd-kao-profiles.go`。

完整 GUI 驗證使用 `tools/verify-desktop.py --mode portraits` 或 `--mode portraits-missing`；加 `--include-council` 抽測軍議圖，必須先用 `tools/run-desktop.sh` 建置，保留輸出中的 engine、startup-probe 與執行檔。長流程依本檔既有 detached Docker 方法執行，容器內維持期限。已驗證範圍及最近收據見 CONTEXT。

## 生成 HD 素材的私人儲存庫保存

2026-10-14 使用者明確指定「目前這些圖片都很棒 記得保存在 github repo 當中喔」。生成的 HD 圖、生成圖的裁切素材、總覽與完整提示詞可保存於本 private repository 的 `assets/hd-b/`。這項新授權取代先前對這些生成圖的僅限本機規則；原版 EXE、DAT、光碟、音軌、擷取原圖與執行收據維持原有邊界。

每批生成圖或裁切素材完成後，更新 [保存工具](tools/archive-hd-art.py) 的來源清單，在 Docker 執行保存與 `--verify-only`，再提交至已確認仍為 private 的儲存庫。圖片與提示詞保持可回查的檔名、SHA-256 及裁切依據。這是素材來源保存，正式素材包格式及公開發行另列。

```sh
test -d workplace/hd-prototype && test -f tools/archive-hd-art.py
timeout 120s docker run --rm --name tokugawa-art-archive --network none \
  --memory 1g --cpus 1 --pids-limit 64 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project" \
  sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4 \
  python3 /project/tools/archive-hd-art.py
```

核對時使用同一 image，把專案掛載加上 `readonly`，命令末尾加 `--verify-only`。工具目前涵蓋劇情圖、81 張頭像及九頁圖集、捲軸、紅金框架、視窗 skin、地面、透明單位草稿與風格比較；新增類別先擴充清單。它拒絕覆寫內容不同的既有圖片，變體使用新檔名。

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

探針固定 dosgolem commit `a9714ebdab2ad6b529f81225472680f2b11f2842`。外部工作樹 HEAD 已前進，捲軸試作改用 `workplace/hd-prototype/scroll/dosgolem-base/` 的獨立副本，沒有回退或修改 `/home/anr2/cht/dosgolem`。後文命令已將 `/dosgolem` 掛載來源固定為 `$PWD/workplace/hd-prototype/scroll/dosgolem-base`。副本已存在就核對 HEAD 並沿用，不重新覆寫。

只在副本不存在且輸出父目錄已建立時，用 Docker 重建固定來源：

```sh
test -d /home/anr2/cht/dosgolem/.git
test -d workplace/hd-prototype/scroll
test ! -e workplace/hd-prototype/scroll/dosgolem-base
timeout 60s docker run --rm --name tokugawa-pinned-source --network none \
  --memory 1g --cpus 2 --pids-limit 64 --user "$(id -u):$(id -g)" \
  --mount type=bind,source=/home/anr2/cht/dosgolem,target=/current,readonly \
  --mount "type=bind,source=$PWD/workplace/hd-prototype/scroll,target=/output" \
  --env HOME=/tmp \
  golang@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac \
  sh -c 'git clone --quiet --no-hardlinks --no-checkout /current /output/dosgolem-base && git -C /output/dosgolem-base checkout --quiet --detach a9714ebdab2ad6b529f81225472680f2b11f2842'
```

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
test -d workplace/hd-prototype/scroll/dosgolem-base && test -f original/game/DX5.EXE
mkdir -p workplace/leprobe
timeout 180s docker run --rm --name tokugawa-leprobe --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/workplace/hd-prototype/scroll/dosgolem-base,target=/dosgolem,readonly" \
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
test -d workplace/hd-prototype/scroll/dosgolem-base && test -f original/game/DX5.EXE
test -d workplace && test -f tools/run-probes.sh
timeout 630s docker run --rm --name tokugawa-repro-probe --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/workplace/hd-prototype/scroll/dosgolem-base,target=/dosgolem,readonly" \
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

探針明確啟用 [MCB 配置模式](tools/dpmi-mcb.go)，既有 dosgolem 預設不變。光碟音訊由 [CDDA 裝置](tools/cdda.go) 依虛擬時間串流，收據 `cd_pcm` 記錄路徑、樣本框數與 SHA-256，`mounted_disc.audio.events` 記錄播放、換曲、停止、續播與原始請求。顯式 AUDIO PREGAP 輸出等長靜音；有效起點的超長請求在 lead-out 停止，保存請求與實際終點。資料軌或 lead-out 以後的起點回錯誤。契約及驗證見 RESEARCH-LOG 的「CDDA 靜音間隙與映像結尾契約」。解碼通過與人耳確認分開記錄，後者目前未通過。

命令最後的 `tools/run-probes.sh` 可接 `-steps 100000000`，保留原版等待並跑到主選單。普通滑鼠輸入用 `-mouse '<JSON>'` 傳入事件陣列，每筆含 `step`、`x`、`y`、`buttons`。`step` 是成功指令數，與虛擬微秒一致；`buttons` 的 bit0 為左鍵、bit1 為右鍵、bit2 為中鍵。按下與放開分兩筆，時間嚴格遞增，全部落在預算內。實際套用結果保存在 `latest.json` 的 `mouse_input_events`。輸入只走 dosgolem 的 MoveMouse／PressMouse／ReleaseMouse，不帶遊戲位址。目視核對與已測操作見 RESEARCH-LOG。

查證 CD 資料流可加 `-cd-trace`，記錄有界的 DPMI 配置與 INT 2Fh 轉送前後資料。診斷只讀 backing memory，預設關閉。將上述 Docker 命令的 `/output` 掛載改為已存在的 `workplace/cd-trace/`，以隔離診斷輸出；正式啟動收據仍留原目錄。開關對照與資料流結論見 RESEARCH-LOG 的「CD 只讀診斷切片」。

HD 素材載入觀測可加 `-hd-trace`，契約見 RESEARCH-LOG 的「HD 首個素材入口與唯讀觀測契約」。以同一 Go Docker 命令將 `/output` 改掛已存在的 `workplace/hd-prototype/`，其他隔離及資源限制保持相同；輸出 `startup-probe/latest.json` 的 `hd_asset_trace`。只記原版載入入口、返回、指標與平面雜湊，不替換素材。`-steps 10000000` 可核對啟動前四次載入；完整路徑沿用 `-steps 750000000 -mouse-file /project/tools/player-path.json -hd-trace`。CD 正式收據仍留原輸出目錄。

DAT 盤點沿用固定 Python Docker 命令，唯讀掛專案及 `original/game/`，後者掛 `/original`；將已存在的 `workplace/hd-prototype/` 掛為可寫 `/output`，執行 `python3 /project/tools/hd-asset-probe.py`。輸出是研究用候選目錄，不是正式 HD 素材包。

首張素材擷取的完整輸入、位址、尺寸與雜湊見 RESEARCH-LOG 的「第一張劇情素材的擷取契約」。重生時在固定 Python Docker 內，唯讀掛專案為 `/project`、可寫掛已存在的 `workplace/hd-prototype/asset-capture/` 為 `/output`，由 `tools/player-path.json` 篩選 `step < 150000000` 的事件，以 `json.dumps(events, ensure_ascii=False, indent=2) + "\n"` 寫出 `player-prefix.json`。寫入前核對原時間表 SHA-256 `f621bd1f66e0d4bb492570317ec82b898ec95d3a1f4139a6ee5c451011b30395`；前綴必須有 33 筆，SHA-256 為 `cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736`。拒絕覆寫既有檔案。

再沿用 Go Docker 的完整隔離命令，以該目錄掛 `/output`，執行 `sh /project/tools/run-probes.sh -steps 150000000 -mouse-file /output/player-prefix.json -hd-trace`。完成後用相同 Go image、原版唯讀掛 `/original`、專案唯讀掛 `/project`、擷取目錄可寫掛 `/output`，執行 `go run /project/tools/hd-extract.go -input /output`。工具拒絕覆寫既有擷取檔，不能用完整戰場的 RAM 代替 A-0 擷取。

離線預覽則將整個 `workplace/hd-prototype/` 掛為 `/output`，在同一 Go Docker 執行：

```sh
go test /project/tools/hd-preview.go /project/tools/hd-layer.go /project/tools/hd-portraits.go /project/tools/hd-preview_test.go
go run /project/tools/hd-preview.go /project/tools/hd-layer.go /project/tools/hd-portraits.go \
  -frame /output/asset-capture/startup-probe/frame-f8b23041cc0ab4866e40dd4d14eb92489b8c3896d2595ea5cba11f1338f0db62.png \
  -source /output/asset-capture/A-0-original.png \
  -hd /output/A-0-hd-b-v1.png -out /output/A-0-hd-preview-v1.png
```

Go 預覽工作設定 `GOMAXPROCS=2` 與 `GOFLAGS=-p=2`，配合 2 CPU／64 程序限制。`-hd` 留空顯示原圖，指向不存在的 PNG 驗證缺圖備援；輸出檔名必須尚未存在。生成美術與提示詞已依授權保存於 `assets/hd-b/`，以 `tools/archive-hd-art.py --verify-only` 核對索引。以上命令驗證離線呈現；桌面切換驗證見下節及 CONTEXT。

桌面首張試作的契約與實際驗證見 RESEARCH-LOG 的「首張 HD 桌面試作契約」。沿用已存在的 GUI image ID，實際版本為 Go 1.24.13／Ebitengine 2.9.9；不重建 image，不改外部 dosgolem。下列輸出目錄與快取須已存在且為 UID/GID 1000:1000。

```sh
test -d /home/anr2/cht/dosgolem && test -d original/game
test -d workplace/desktop-prototype && test -d workplace/leprobe/gocache
timeout 180s docker run --rm --name tokugawa-desktop-build --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/workplace/hd-prototype/scroll/dosgolem-base,target=/dosgolem,readonly" \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace/desktop-prototype,target=/output" \
  --mount "type=bind,source=$PWD/workplace/leprobe/gocache,target=/gocache" \
  --env HOME=/tmp --env GOCACHE=/gocache --env GOTOOLCHAIN=local \
  --env TOKUGAWA_BUILD_ONLY=1 --workdir /dosgolem \
  sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4 \
  timeout -k 5s 160s sh /project/tools/run-desktop.sh
```

真實視窗驗證由容器內 Xvfb 管理，不在主機啟動 GUI。以下 `--mode hd` 可改為 `missing` 或 `mouse`，分別核對缺圖備援及縮放後的普通滑鼠。腳本管理並清理 Xvfb 與桌面程序，輸出 `desktop-<mode>.json`、`verification-<mode>.json` 與真實視窗 PNG。HD 及缺圖模式使用 33 筆正常輸入前綴，路徑與雜湊見擷取契約；mouse 模式不帶時間表。

```sh
test -f workplace/desktop-prototype/tokugawa-desktop-prototype
test -f tools/verify-desktop.py && test -d original/game
timeout 300s docker run --rm --name tokugawa-desktop-gui --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace/desktop-prototype,target=/output" \
  --env HOME=/tmp --env GOMAXPROCS=2 \
  sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4 \
  timeout -k 5s 280s python3 /project/tools/verify-desktop.py --mode hd
```

桌面程式只有使用 `-desktop` 才開視窗，預設仍為無頭。F6 切換原圖／HD，F7 暫停試作步進，F9 擷取強化畫面。滑鼠移到頂端 6 個原版邏輯像素會展開功能列，移開自動收起。Esc 固定／收起；叉號只收起，Shift+Esc 或工具選單可關窗並輸出原生收據。功能列顯示時原版停止步進，畫面下移並增加視窗高度；收起後滑鼠按鍵全部放開才恢復步進與滑鼠轉送。Tab 切換欄位，方向鍵選取，Enter 展開／套用。顯示提供原版及 HD 圖像，肖像與捲軸分項缺圖回原圖；兩項素材都無效時 HD 列停用；執行提供繼續／暫停，工具提供截圖／離開。

`-hd-source`／`-hd-asset` 指定肖像原圖及 HD 圖；`-hd-scroll-source`／`-hd-scroll-asset` 指定捲軸原圖及 HD 圖。兩份原圖各自固定雜湊，缺圖分項備援。捲軸的紙張、捲邊與外框換成 HD，文字前景維持原版像素與位置。`-desktop-speed` 預設 1，驗證用 20 不改虛擬時間公式。普通滑鼠只在沒有輸入時間表時啟用；一般鍵盤轉送與桌面音訊尚未接入。四語及音訊設定待功能接妥再加入。二進位留在 `workplace/` 作中間產物，不能稱為正式平台交付。

功能列的獨立驗證沿用上述 GUI image 與建置命令，將可寫掛載改成已存在的 `workplace/desktop-prototype/toolbar/`，保留先前 GUI 收據。建置後執行下列命令；`--mode hd` 可換為 `mouse` 或 `missing`。腳本輸出 `windowbar-<mode>.json`、`verification-windowbar-<mode>.json` 及真實視窗 PNG，並清理所有子程序。

```sh
test -f workplace/desktop-prototype/toolbar/tokugawa-desktop-prototype
test -f tools/verify-windowbar.py && test -d original/game
timeout 300s docker run --rm --name tokugawa-toolbar-gui --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace/desktop-prototype/toolbar,target=/output" \
  --env HOME=/tmp --env GOMAXPROCS=2 \
  sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4 \
  timeout -k 5s 280s python3 /project/tools/verify-windowbar.py --mode hd
```

捲軸的原始資料與生成素材放在 `workplace/hd-prototype/scroll/`，桌面驗證另放其 `desktop/` 子目錄。來源擷取依 RESEARCH-LOG 的「劇情捲軸 HD 契約」：30 筆普通輸入前綴、136769993 步，最後載入 VISUAL.OZM；擷取由 `go run /project/tools/hd-scroll-extract.go` 完成。生成完整畫布及提示詞保存後，由 `go run /project/tools/hd-scroll-prepare.go` 取出既定下方捲軸區，兩工具都只在固定 Go Docker 中執行，拒絕覆寫既有 PNG。工具路徑、輸入與輸出雜湊見該目錄的 `scroll-extraction.json` 及 `scroll-art.json`。

捲軸桌面建置沿用 `tools/run-desktop.sh`，以固定副本掛 `/dosgolem`、`scroll/desktop/` 掛 `/output`。確認素材與二進位已存在後執行兩頁劇情的真實視窗驗證；`--mode hd` 可換為 `missing` 或 `only`，分別測缺捲軸圖及只有捲軸 HD。驗證用虛擬速度 5，以便在第二頁擷取；不改原版時間公式。

```sh
test -f workplace/hd-prototype/scroll/desktop/tokugawa-desktop-prototype
test -f tools/verify-hd-scroll.py && test -d original/game
timeout 300s docker run --rm --name tokugawa-scroll-gui --network none \
  --memory 2g --cpus 2 --pids-limit 128 --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/project,readonly" \
  --mount "type=bind,source=$PWD/original/game,target=/original,readonly" \
  --mount "type=bind,source=$PWD/workplace/hd-prototype/scroll/desktop,target=/output" \
  --env HOME=/tmp --env GOMAXPROCS=2 \
  sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4 \
  timeout -k 5s 280s python3 /project/tools/verify-hd-scroll.py --mode hd
```

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

後續規劃已確認桌面版、繁中／簡中／英文／日文，以及可切換的 HD 素材包。瀏覽器排除。現行決定見 [CONTEXT.md](CONTEXT.md)，階段與驗收草案見 [WORKLIST.md](WORKLIST.md)。原版繼續負責玩法與存檔，譯文與 HD 圖層在桌面呈現。美術風格已選 B 戰國手繪風，首張肖像、劇情捲軸及參考 `/home/anr2/cht/softworld_san` 的頂端隱藏列已接入 Linux 桌面試作；完整擷取介面與素材包格式仍待定案。

原版 ZIP、EXE、DAT、光碟映像、音軌與原包附帶的補丁只留本機，不加入 Git、GitHub Release 或公開雲端。`.gitignore` 已排除 `original/` 與 `workplace/`。自製的平台修正、測試與文件可加入已授權的 private repository，包含 `tools/dosgolem.patch`。未確認散布權前不製作公開資料包。
