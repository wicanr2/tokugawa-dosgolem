# 德川家康：DOS 原版與 dosgolem

完整單位圖集入口為 [tools/hd-unit-prepare.py](tools/hd-unit-prepare.py)，研究輸出在 `workplace/hd-prototype/catalog/terrain/units/`。`pose-revisions.json` 保存 002／007 的姿勢修正版與提示詞來源；`draft-profiles.json`、`review-page-*.png` 保存 13 原版索引的來源／裁切與原圖對照；`review.json` 為逐項檢視，`profiles.json` 為審查後的十張非空圖與三張空白來源。工具預設做裁切對照，`--integrate` 核對 review 才更新 `tools/hd-terrain-unit-profiles.go`；不以推測兵種或武將命名。生成圖及提示詞保存在 `assets/hd-b/terrain/`，純 HD 總覽與 metadata 在其 `units/` 子目錄，原圖對照只留本機。固定 GUI image 中專案掛 `/project` 可寫、`units/` 掛 `/output`，執行 `python3 /project/tools/hd-unit-prepare.py`。原版來源依 `terrain/objects/source-profiles.json` 及 `tools/hd-terrain-sprite-profiles.go` 的索引零 alpha 契約核對。

單位圖第一次裁切對照保存於 `units/attempt-01/`，其中 `prepare-source.txt` 與 `pose-revisions.json` 保留當次輸入；008 的單尖槍頭與原版分叉輪廓不符，該次不加入正式 profile。`units/pose-revisions-v2.json` 加入 008 的局部修正版，沿原檔名重生目前裁切對照，舊目錄及圖像不覆寫。

完整單位驗證入口為 [tools/verify-hd-units.py](tools/verify-hd-units.py)。`units/runtime/` 保存固定 GUI image／`tools/run-desktop.sh` 的執行檔、四套件回歸與來源快照；`units/desktop/` 沿 `tools/verify-hd-weather.py --mode weather` 自然執行原版入口至首戰。固定 GUI image 中專案唯讀掛 `/project`、`units/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-units.py`，輸出 `units/verification-summary.json`。十項來源及共同場景測試、三項原始空白與正常首戰兩項單位分開核對；正常完整畫面須等於地面全圖集版本。

地面全圖集材質製作入口為 [tools/hd-terrain-materials.py](tools/hd-terrain-materials.py)，保存於 `workplace/hd-prototype/catalog/terrain/materials/`。`generation.json` 記錄新手繪材質及完整提示詞；`drafts/`、`draft-manifest.json` 及 `review-page-*.png` 保存第一版草稿。`drafts-v2/`、`draft-manifest-v2.json`、`review-v2-page-*.png` 保存清除孤立色點後的第二版；`review-v2.json` 記錄逐項檢視，`profiles.json` 為審查後的來源與 HD 對照。原版八項已通過素材與 `prepared/profiles.json` 保持歷史基準。工具預設只做草稿；`--integrate` 必須核對 review 與 manifest 才更新 `tools/hd-terrain-profiles.go`。固定 GUI image 中專案掛 `/project`、`terrain/` 掛 `/output` 可寫，原始資料只讀。生成材質、衍生圖塊及提示詞依授權同步至 `assets/hd-b/terrain/`，原版索引與遮罩維持本機。

地面全圖集驗證入口為 [tools/verify-hd-terrain-materials.py](tools/verify-hd-terrain-materials.py)。`materials/runtime/` 保存沿固定 GUI image 及 `tools/run-desktop.sh` 重建的執行檔、四套件回歸與來源快照；`materials/desktop/` 使用 `tools/verify-hd-weather.py --mode weather` 跑原版入口至首戰，完整視窗另與前次天氣收據核對。固定 GUI image 中專案唯讀掛 `/project`、`materials/` 可寫掛 `/output`，執行該驗證工具，輸出 `materials/verification-summary.json`。104 個來源的圖形及共同呈現測試與正常首戰六種來源分開記錄；不把材質分類當作原版地形規則。

共同 OZM 診斷的獨立核對入口為 [tools/verify-hd-graphics-source.py](tools/verify-hd-graphics-source.py)。固定 GUI image 中專案唯讀掛 `/project`、`workplace/hd-prototype/catalog/graphics/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-graphics-source.py`。`graphics/source-probe/source-snapshot/tools/` 保存當次來源；`graphics/source-verification.json` 記錄靜態來源矩形、自然返回、完整 RAM 與正常路徑基準核對。

共同不透明 OZM 的呈現入口為 [tools/hd-graphics.go](tools/hd-graphics.go)，固定來源與天氣素材指紋見 [tools/hd-graphics-profiles.go](tools/hd-graphics-profiles.go)，邊界與備援測試見 [tools/hd-graphics_test.go](tools/hd-graphics_test.go)。沿固定 GUI image 與 `tools/run-desktop.sh`，建置寫入 `workplace/hd-prototype/catalog/graphics/runtime/`；`graphics/desktop/` 與 `graphics/desktop-missing/` 保存真實視窗及來源快照。驗證入口 [tools/verify-hd-weather.py](tools/verify-hd-weather.py) 及 [tools/verify-hd-weather-summary.py](tools/verify-hd-weather-summary.py)，天氣來源及生成圖對應保存在 `graphics/weather-profiles.json`，兩模式總表為 `graphics/verification-summary.json`。生成圖與提示詞另保存 `assets/hd-b/art/weather-generation.json`，原圖及收據不進 Git。

開場 OP2／OP8 手繪草稿及完整提示詞保存於 `workplace/hd-prototype/catalog/art/`，生成 metadata 為該目錄的 `opening-generation.json`，私人索引副本為 `assets/hd-b/art/opening-generation.json`。原始生成尺寸分別 2172×724、2171×724，尚須依原版 640×200 校準；OP2 未使用的藍底區另外保護。草稿不當作已接入或正常路徑驗收。

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
| [tools/mark-probe-prepare.py](tools/mark-probe-prepare.py) | 產生研究專用 MARK LE 入口探針與隔離 overlay，拒絕遊戲專用 HD 觀測；不修改正式啟動路徑 |
| [tools/mark-cpu.patch](tools/mark-cpu.patch) | 接在 dosgolem.patch 後的標準記憶體 byte SHR 與 DPMI 新 data descriptor 可寫性修正；保留原始界限，契約及實際 MARK 阻塞證據見 RESEARCH-LOG |
| [tools/dos4g-exception.patch](tools/dos4g-exception.patch)、[tools/dos4g-vector.go](tools/dos4g-vector.go) | 在隔離副本擴充段界限失敗回報、平坦同權限 IRET 與 DOS 向量唯讀查詢，契約及限制見 RESEARCH-LOG |
| [tools/dos4g-exception.go](tools/dos4g-exception.go)、[tools/dos4g-exception_test.go](tools/dos4g-exception_test.go) | 依公開 DOS/4GW 契約將真實段界限例外交付原版已登記的 INT 0Dh handler；不略過原版 handler 或放寬 VRAM selector |
| [tools/mark-trace.go](tools/mark-trace.go) | 固定 MARK 雜湊的唯讀繪圖、目標色盤與 RAM guard 觀測，供獨立研究探針及桌面 HD 資格；保存原生漸入／漸出畫面，沒有 guest 寫入 |
| [tools/program.go](tools/program.go) | 固定 DX5／MARK 來源雜湊與程式名稱，不接受任意路徑；兩個原版 LE 執行入口的回報與觀測分開 |
| [tools/hd-publisher.go](tools/hd-publisher.go)、[tools/hd-publisher_test.go](tools/hd-publisher_test.go) | MARK 的來源／自然繪圖及原版色盤資格、HD 標誌與下方原版版權保留；未知狀態或缺圖回原版 |
| `workplace/hd-prototype/catalog/remaining/consumers/mark-probe/` | 包內 MARK.EXE.BAK 的自然執行、原生畫面及顯示契約研究；原版資料與執行收據只留本機 |
| [tools/startup-choice.go](tools/startup-choice.go)、[tools/desktop-startup.go](tools/desktop-startup.go) | 桌面啟動前的原版／HD 選擇及輸入隔離；與頂端功能列共用顯示設定 |
| [tools/verify-startup-choice.py](tools/verify-startup-choice.py) | Docker X11 驗證啟動選擇、輸入隔離與進入後的功能列切換 |
| [tools/verify-startup-integration.py](tools/verify-startup-integration.py) | 核對同一建置的啟動選擇、首戰回歸、93 項凍結來源與全部 RAM；固定 GUI image 內專案唯讀掛 /project，startup-choice/ 可寫掛 /output 執行 |
| `workplace/desktop-prototype/startup-choice/` | 啟動選擇切片的隔離建置、X11 收據及本機截圖 |
| `workplace/desktop-prototype/publisher/` | MARK 原生與 HD 桌面的隔離建置、缺圖、功能列及完整原版狀態驗證 |
| [tools/verify-publisher.py](tools/verify-publisher.py) | 在既有 GUI image 內以 X11 操作 MARK 的啟動選擇、原圖／HD、功能列與自然退出；收據及原圖保持本機 |
| [tools/verify-publisher-integration.py](tools/verify-publisher-integration.py) | 固定 GUI image 內專案唯讀掛 /project、publisher/ 可寫掛 /output，執行後核對 MARK 與首戰、獨立 PCX 解碼、HD 採樣及來源快照；歷史核對可加 --source-root /output/source-snapshot |
| [tools/desktop-publisher.go](tools/desktop-publisher.go) | 桌面啟動器依序執行固定 MARK 與 DX5，沿用最後顯示設定；關窗停止並保存各自原版收據，不偽造 DOS EXEC |
| [tools/verify-startup-chain.py](tools/verify-startup-chain.py) | 固定 GUI image 內的預設完整啟動鏈、雙向設定沿用、實際 X11、原版 RAM／欄位及正常關窗驗證；模式為 original、hd 或 cancel |
| [tools/verify-startup-chain-integration.py](tools/verify-startup-chain-integration.py) | 固定 GUI image 內專案唯讀掛 /project、startup-chain/ 可寫掛 /output，重查三模式實際收據、完整 RAM、X11 與 103 項來源並輸出總表；歷史核對可加 --source-root /output/source-snapshot |
| `workplace/desktop-prototype/startup-chain/` | 完整啟動鏈的凍結建置與三模式隔離驗證；原版資料及執行收據只留本機 |
| [tools/ida-entry.py](tools/ida-entry.py) | 保留原名與位址的 IDA 入口證據；可傳入 `起點:終點` 匯出至多 16 KiB 的窄範圍 |
| [tools/hd-asset-probe.py](tools/hd-asset-probe.py) | 唯讀盤點 DAT 的候選資源目錄；保存原始欄位、檔案偏移與逐項雜湊，不作正式 HD 識別器 |
| [tools/ida-asset-probe.py](tools/ida-asset-probe.py) | IDA 9.4 的資源字串、直接交叉參照及有界函式證據；只接受固定 SHA 的 DX5.EXE 或包內 MARK.EXE.BAK，另核對 LE header，保留原名與位址 |
| [tools/ida-art-consumers.py](tools/ida-art-consumers.py) | IDA 9.4 的圖像資料讀取端 caller 視窗，只匯出原始定位與指令；從正式資料庫的隔離副本執行 |
| `catalog/remaining/consumers/ida/` | PCX 及剩餘動畫消費端的 IDA 9.4 副本、最小輸出探針與窄範圍證據。沿用已存在的 4ac62de83339 image、UID/GID 1000，正式 workplace/ida/DX5.i64 與原版唯讀；只寫此目錄，副本在查詢前核對雜湊，不把資料庫或匯出收據加入 Git |
| `catalog/remaining/consumers/ida/mark/` | 由唯讀 MARK.EXE.BAK 新建的一次性 IDA 9.4 資料庫與 PCX 名稱直接使用者證據；僅研究包內該 SHA 的檔案，不啟用或改名原始 .BAK，不推定出廠版本 |
| [tools/hd-publisher-preview.go](tools/hd-publisher-preview.go)、`catalog/remaining/consumers/publisher-preview/` | 離線出版商標誌 HD 原型，依既有圖像幾何取樣並保留原版底部版權文字。固定 GUI image 內專案唯讀、publisher-preview 可寫掛 /output，工作目錄指定 /output，以 `go run /project/tools/hd-publisher-preview.go` 執行；不是正常執行收據，含原版文字的合成 PNG 只留本機，不加入 Git |
| [tools/hd-trace.go](tools/hd-trace.go)、[tools/hd-trace_test.go](tools/hd-trace_test.go) | 固定 EXE 的唯讀素材載入觀測及記憶體不變、錯誤界限、事件上限測試；不替換素材 |
| [tools/hd-composite-probe.go](tools/hd-composite-probe.go)、[tools/hd-composite-probe_test.go](tools/hd-composite-probe_test.go) | 預設關閉的原版 OR／透明矩形唯讀探針，核對自然返回、完整四平面與目的索引；只作研究，不替換 HD |
| [tools/battle-cpu.patch](tools/battle-cpu.patch) | 原版正常攻擊及回合分支的 D1 /4 32 位暫存器左移、C1 count-one OF、39 /r 16 位元記憶體比較、81 /7 16 位元立即值比較、AND AX,imm16、單次 SCASW、83 /6 32 位元記憶體 XOR 與 C1 /5 32 位元記憶體 SHR；依 Intel 規格及差分測試，套於 DOS/4GW 補丁後，外部 dosgolem 不改 |
| [tools/battle-instruction-probe.go](tools/battle-instruction-probe.go) | 在 Docker 的原有 cmd/leprobe main overlay 執行：先以 LE loader 核對 IDA 載入後 bytes，再在合成 CPU 狀態單步列出窄範圍的指令形式缺口。--rows 指定 IDA 匯出，預設 /output/range-1f400-1f800.json，上限 8192 列；--cpu-source 必填。只作平台能力篩查，不作正常玩家路徑、玩法或 HD 驗收；輸出保存 consumers/ida/ 及各批次 capabilities/，操作見 RESEARCH-LOG |
| [tools/explore-battle.py](tools/explore-battle.py) | 固定 GUI image 內從普通輸入或原版存檔選單接續真實 X11；commands.json 支援 click／drag／run／capture／toggle／quit。--mouse-file、--prefix-step 與 --desktop-speed 指定研究前綴；存檔輸入必須以 --input-manifest 核對逐檔 SHA。--timeout 預設 3600 秒，到期及錯誤先正常關窗保存收據；早期工具短測另加 --harness-only，不作玩家證據。原版與 Go 來源保持凍結，不直接進入戰鬥函式 |
| [tools/verify-composite-probe.py](tools/verify-composite-probe.py) | 固定 GUI image 內專案唯讀掛 /project、指定 battle 批次可寫掛 /output；--control-only 核對 750M 基準，--harness-close-only 於 r7 核對例外／到期正常收尾。--native-only 於 r10/native-turn-extended/ 或 native-turns/ 核對原版存檔 manifest、十九筆前綴、全部普通輸入與獨立 OR／透明索引公式。沒有合成返回時明示 INCOMPLETE。完整桌面模式另核對 X11，拒絕 harness_only 作玩家證據；--source-root 可指定歷史快照，複用控制組核對 reference.json，不計為新執行 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/` | 原版正常戰場操作、OR／透明矩形素材與有界桌面研究；來源快照、原生圖及執行收據只留本機。根目錄保留 D1 停止批次，r2/ 保留回歸失敗，r3/ 保留 CMP 停止與控制組，r4/ 保留測試條件缺漏，r5/ 保留立即值比較停止與控制組，r6/ 保留 AND 停止與控制組。r7/ 保存最新平台控制組、互動到期失敗及 cleanup-check*/ 工具測試，r8/ 沿用同一二進位接續；複用控制組須另記來源及雜湊，不稱新測次 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r8/save-import/`、`load-check/` | 原版「儲存進度」產生的 DAT 工作副本及普通標題選單讀檔驗證；保存寫入前後 SHA、來源、逐檔匯入 manifest 與普通輸入。不改 original/，匯入資料不冒稱下載基準 DAT，不逆向或另擬存檔格式；全部只留本機 |
| [tools/verify-original-save.py](tools/verify-original-save.py) | 固定 GUI image，專案唯讀掛 /project、r8 可寫掛 /output；核對原版存檔只改 I003.SAV、其他 200 個資源不變，以及正常標題／FILE 3／確認讀回的完整原生索引及 RGBA。歷史來源用 --source-root /project/workplace/hd-prototype/catalog/remaining/consumers/battle/r8/source-snapshot；加 --desktop /project/workplace/hd-prototype/catalog/remaining/consumers/battle/r9 核對五張 X11 及 HD 切換狀態。只驗收此存檔樣本，不宣稱解出存檔格式或所有欄位／平台 |
| [tools/import-original-save.py](tools/import-original-save.py) | 固定 GUI image，專案唯讀掛 /project、原版存檔探針批次可寫掛 /output；--source-root 必須指定該二進位的凍結來源。核對全部普通輸入、完整來源、原版工作副本 DAT、唯一 I003.SAV 差異及其他 200 資源不變後，建立拒絕覆寫的唯讀 save-import/game 與逐檔 manifest。只複製原版程式實際寫出的資料，不解碼或猜補存檔欄位 |
| 第三回合存讀檔核對 | tools/verify-original-save.py --imported：將 r12/load-turn3/ 可寫掛 /output，--source-root 指向 r12/source-snapshot；依 input-data.json 與匯入 manifest 核對來源、九筆普通讀檔、原生索引與 RGBA。索引相同但色盤不同時明示 PASS_NATIVE_INDICES_ONLY，不宣稱 RGBA 相同。新回合 probe 用 verify-composite-probe.py --native-only --prefix-reference /project/workplace/hd-prototype/catalog/remaining/consumers/battle/r12/load-turn3/receipt.json |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r9/` | 沿用 r7 二進位，從 r8 原版存檔以正常標題／欄位／確認操作接續；桌面速度 1，輸入與資料 manifest 明示保存，讀檔及交鋒各自驗證。不同初始存檔不與下載基準混作同狀態 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r8/native-turn/` | 原版存檔、九筆正常讀檔前綴與普通右鍵／回合確認輸入的無頭探針；只從原始 LE 入口執行，記錄完整輸入、資料雜湊及 OR／透明返回。無頭來源證據與 HD 真實視窗驗收分開 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r10/` | 單次 SCASW 修正後的 106 項凍結來源、四套件與桌面建置；native-turn/ 保存同一十九筆原版讀檔／回合輸入的 200M 收據，native-turn-extended/ 以相同輸入延長自然執行。首批執行輸出的 RAM、畫面與音訊位於 r10 根及 startup-probe/，receipt.json 位於 native-turn/，output-layout.json 記錄此掛載布局。各次輸出保持分開，不將預算耗盡稱為回合完成 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r10/native-turns/`、`control/` | 同一 SCASW 版本的多回合普通滑鼠探針與下載基準的 750M／390 事件控制組；資料、來源快照及輸入明示核對，分開保存。control/mark/ 為同版 MARK 的獨立執行，不混用存檔 DAT 或兩個原版環境 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r11/` | 83 /6 32 位元記憶體 XOR 的新凍結來源、四套件、建置及同一多回合普通輸入重播；保留 r10 原停止，不回寫舊快照。native-turns/output-layout.json 明示執行時 r11 根掛 /output，原版輸出位於根目錄；核對時 native-turns 掛 /output，另傳 --artifact-root /project/workplace/hd-prototype/catalog/remaining/consumers/battle/r11。capabilities/ 保存指定 IDA 窄範圍的形式篩查，只作平台支援診斷 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r11/save-turn3/` | 使用 r11 原版二進位、唯讀 source-snapshot 與前三回合普通輸入，再從正常選單儲存進度，目標是縮短後續交鋒查證的讀檔前綴；執行時專案唯讀掛 /project，source-snapshot/tools 另唯讀掛 /project/tools，106 項編譯輸入全部在 tools 下，主工作樹可另行實作。初期環境失敗留根目錄，ready/ 停在原版確認框且 DAT 未改；ready-r2/ 延長普通確認按下時間。資料與存檔只留本機 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r12/` | C1 /5 32 位元記憶體 SHR 的凍結來源、四套件與原版普通輸入重播；正常交鋒證據不足前不加入正式 HD 合成 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r12/load-turn3/` | 由 r11/save-turn3/ready-r2/save-import/ 讀入第三回合原版存檔，以九筆普通標題／FILE 3／確認輸入核對 140M 畫面；不同存檔不與第一回合混作同狀態 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r13/`、`r14/` | ADC AL,imm8 與 83 /2 word register 的新凍結來源及四套件；r13 保留舊否定測試仍拒絕新形式的失敗，r14 改以仍未支援的 word memory 核對拒絕。第三回合正常讀檔重播以 r12/load-turn3 的獨立收據作前綴，保留資料 manifest 與兩種不同存檔的界限 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r14/desktop-original/` | 正常讀回第三回合、結束回合後的原版交鋒 X11 擷取；先核對原版 UI、合成與動畫，當次二進位及 106 項來源保持凍結。正式 HD 消費端另開新來源批次 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r15/` | 依 RESEARCH-LOG 的 READY 契約接 BATTLE OR、JIMEN 透明與原版前景保留，保存凍結來源、四套件、正常原版控制及 X11 原圖／HD／缺圖驗證；頂端單位縮圖維持獨立待辦 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r16/`、`r17/` | 共用視窗原版矩形保存／還原與 graphics／KAO 呈現生命週期修正，先核對原版四平面保存內容及返回目的，再恢復該區 host 圖層；區域外新繪圖保留。r16 保留測試編譯失敗，r17 的 verification-summary.json 為十四張正常 X11、九次切換、個別缺圖及動畫還原驗收。unit-diagnostic/ 另存同版正常十九筆輸入的 215M 地形／前景唯讀診斷，原版資料只留本機 |
| [tools/verify-hd-combat.py](tools/verify-hd-combat.py) | 固定 GUI image，專案唯讀掛 /project、指定 battle 批次可寫掛 /output；核對 600M 完整 RAM／十七欄、實際 X11、原版／HD 切換、OR 混色、文字、個別缺地面及動畫還原。r17 用舊版工具全文重查；r19 加 --units 核對兩縮圖、索引、獨立 alpha、動畫遮擋／還原及個別缺單位。--source-root 可指定歷史快照；個案 ready/ 保存環境重跑。跨個案缺圖比較要求完整原生可見畫面相同，各次切換仍逐項核對完整 guest proof；輸出 verification-summary.json |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r18/` | 頂端兩個 48×48 單位縮圖的 READY 接入、固定來源、四套件、原版控制與正常／缺圖／動畫 X11；重用既有單位素材，來源及原版畫面只留本機。consumers/ida/battle-unit-primitives.i64 為正式資料庫的隔離副本，原始 bytes 核對見 r17/unit-diagnostic/ |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r19/` | 縮圖改用原生索引核對跨淡入來源與自然返回後的獨立建置及正常 GUI；r18 保存 RGB 跨時點資格拒絕及完整原版控制，舊來源和收據不覆寫 |
| `workplace/hd-prototype/catalog/remaining/consumers/battle/r20/` | 已證實的 399 列結局完整複製資格與縮圖外側紙張呈現修正；原版最後一列保持。desktop-animation/ 保留新視窗只畫到一半的早期擷取，ready/ 用 wait_restore 的 steady_resource 等待已核准背景恢復後取樣，二進位與 106 項編譯來源不變。來源、共同消費端測試與正常交鋒 GUI 分開保存，不將共同測試稱為正常結局驗收 |
| `workplace/hd-prototype/catalog/remaining/consumers/ida/remaining-consumer-audit.json` | 20 個來源與 23 個 EXE 名稱索引、DAT 目錄索引、正常 RAM 與 IDA 原始 bytes 核對；兩種索引以完整檔名配對，不互換。新匯出 range-18a80-18dc0、range-1c084-1c200、range-185a7-189c8、range-1e9e0-1ebc0 的用途與 READY 範圍見 RESEARCH-LOG |
| [tools/hd-extract.go](tools/hd-extract.go) | 從正常路徑 RAM 擷取 A-0.OZM，核對四平面雜湊與原生畫面矩形，輸出本機 PNG 及收據；不是正式素材包格式 |
| [tools/hd-capture.go](tools/hd-capture.go) | `-hd-export` 在原版載入返回時輸出已核對平面的索引圖、當時 DAC 與收據；沒有 guest 寫入，不代表繪圖或遮擋已驗收 |
| [tools/hd-ozm-probe.py](tools/hd-ozm-probe.py) | 依 IDA 9.4 原始指令做離線資源盤點，只接受獨立解碼命令；遇到 VGA 來源命令明確拒絕。以正常路徑四平面與索引雜湊交叉核對 |
| [tools/hd-kao-extract.py](tools/hd-kao-extract.py) | 擷取 81 張 80×80 KAO.GRP 頭像、九頁來源圖集與收據；核對正常路徑載入的完整資料及第 76 張的 6400 原生像素 |
| [tools/hd-kao-prepare.py](tools/hd-kao-prepare.py) | 依已檢視的實際格線裁切九張 HD 圖集，補黑底保持人物比例；輸出 `workplace/hd-prototype/catalog/kao/selected/` 的 81 張素材、manifest 與總覽 |
| [tools/hd-art-inventory.py](tools/hd-art-inventory.py) | 重生 `workplace/hd-prototype/catalog/art-coverage.json`，區分已製作、逐項替換、正常路徑驗證與未知素材；歷史收據核對當時驗證器的精確快照及 SHA-256，來源列於 verification_tool_sources；資源數不當成圖集完成百分比 |
| [tools/hd-remaining-intake.py](tools/hd-remaining-intake.py) | 從唯一覆蓋清單核對剩餘已解碼 OZM，保存來源、原圖總覽及動畫逐格參考；不猜補五項解碼未知 |
| [tools/hd-remaining-prepare.py](tools/hd-remaining-prepare.py) | 依原始格線準備 20 項美術，逐格驗證 16 項鏡像、保留動畫空格、黑色及彩色緩衝區。固定 GUI image 內專案唯讀掛 `/project`、remaining 可寫覆掛原路徑，`--version 3` 產生 draft-v3；審查及核對通過後加 `--approve` 選定美術，不修改 Go 執行器。draft-v1／2 保留單像素緩衝區邊界及 BATTLE 美術末列修正前版本 |
| [tools/verify-hd-remaining.py](tools/verify-hd-remaining.py) | 固定 GUI image 與相同掛載執行 `--version 3`，核對 20 張圖、48 動畫格、16 鏡像、空格、緩衝區及來源。輸出 art-verification，不作正常玩家路徑或執行器接入收據 |
| [tools/hd-remaining-runtime.py](tools/hd-remaining-runtime.py)、[tools/hd-graphics-remaining-profiles.go](tools/hd-graphics-remaining-profiles.go)、[測試](tools/hd-graphics-remaining_test.go) | 依選定美術及 READY 契約註冊 20 項共用不透明來源，保留來源指紋、有效區與原圖固定前景；不變更原版字模或動態文字規則 |
| [tools/verify-hd-remaining-runtime.py](tools/verify-hd-remaining-runtime.py) | 固定 GUI image 內專案及原版唯讀，desktop-title 或 desktop-title-missing 可寫掛 `/output`，分別用 `--mode title`／`--mode title-missing` 驗證正常開場；remaining 掛 `/output` 並用 `--mode summary` 核對來源、首戰及限制 |
| [tools/verify-hd-menu-platform.py](tools/verify-hd-menu-platform.py) | 固定 GUI image 內專案唯讀、remaining 可寫掛 `/output`，以 `--revision platform` 或 `rcr` 核對匹配來源快照、正常標題／首戰及普通歷史／離開收據；不同版本的缺圖證據保留其版本範圍 |
| [tools/hd-boundary-probe.py](tools/hd-boundary-probe.py) | 唯讀查證三組未完整解碼 OZM 的實際停止位置、已知平面及 PCX 尺寸，輸出 remaining/source-boundary；不補 bytes、不改原解碼器，原版 PNG 只留本機 |
| [tools/hd-tail-recover.py](tools/hd-tail-recover.py)、`remaining/source-boundary/OP-FULL-runtime/` | 以正常載入的四平面 SHA 核對三個完整平面，枚舉唯一未知 byte 並要求唯一匹配；只保存該次原版 RAM 的索引與證據，不修改 DAT 或靜態解碼器，不推廣到其他未觀測狀態 |
| `workplace/hd-prototype/catalog/remaining/source-boundary/` | 五項 OZM 的三組 payload EOF 診斷及 RAINEND.PCX 原版參考；menu-history／menu-exit 用普通選單輸入查證原版來源及停止點，保存匹配來源快照。未完成 HD GUI 驗證的原版收據不當作新美術完成證據 |
| `workplace/hd-prototype/catalog/remaining/runtime/`、`remaining/desktop-title/`、`remaining/desktop-title-missing/`、`remaining/desktop-regression/` | 本批隔離建置、正常標題／缺圖切換與首戰回歸，保存匹配來源快照；生成素材沿用既有私人素材庫，原版收據留本機 |
| 上列目錄內的 `platform/`、`remaining/source-boundary/menu-history/platform/`、`menu-exit/platform/` | STD 與文字模式恢復修正後的獨立版本；保留先前停止收據，不覆寫已驗證建置 |
| `remaining/runtime/platform/ready/` | 修正倒序 MOVSB 測試缺少合法 DS／ES 描述子的驗證環境後，乾淨重建平台版本；上層保留首次失敗結果 |
| `remaining/source-boundary/menu-history/platform/ready/`、`menu-exit/platform/ready/` | 補建探針必要的 startup-probe 輸出目錄後，重跑同一版本及普通輸入；上層保留漏建目錄的環境錯誤 |
| `remaining/runtime/rcr/`、`remaining/source-boundary/menu-history/rcr/`、`remaining/desktop-regression/rcr/`、`remaining/desktop-title/rcr/` | 補齊普通歷史簡介實際使用的 word RCR reg,1 後，保存新建置、匹配來源快照及普通路徑回歸；不覆寫 STD 版本 |
| `workplace/hd-prototype/catalog/remaining/` | 剩餘開場、動畫與介面的原版參考、生成圖、提示詞、逐項準備及隔離驗證；原圖與收據留本機，生成美術另保存至 assets/hd-b |
| `workplace/hd-prototype/catalog/remaining/history/` | 普通歷史簡介的特殊繪圖證據、只讀診斷及後續 HD 驗證；`ida/` 只操作正式資料庫的隔離副本，原版收據不加入 Git |
| [tools/hd-opening-probe.go](tools/hd-opening-probe.go)、[測試](tools/hd-opening-probe_test.go)、`remaining/history/probe/` | DRAFT 專用 `-hd-opening-trace`，在 IDA 已定位入口／自然返回保存有界原始參數、候選平面及原生畫面；不替換圖、不改 guest，不把最後載入名稱當來源證明 |
| [tools/verify-hd-opening-source.py](tools/verify-hd-opening-source.py)、`remaining/history/source-verification.json` | 在固定 GUI image 中以專案唯讀、history 可寫覆掛原路徑，核對 600M 完整 RAM、十五欄、78 項來源快照、四次完整複製及位移前後 bytes；沒有 HD 呈現完成聲明 |
| [tools/hd-opening-prepare.py](tools/hd-opening-prepare.py)、[註冊表](tools/hd-graphics-opening-profiles.go)、`remaining/history/art/` | 核對兩次正常 RAM 來源及已生成 OP-FULL，全幅校準至 1280×800；來源與工具全文分開保存，`--register` 只在本批 READY 後註冊 runtime_observed 精確來源 |
| `remaining/history/art-v2/` | 同一準備工具以 `--version 2 --register` 校準藍白雲帶修正版；第一版、當時工具全文與來源 sidecar 保持原樣，選定素材的 Go 註冊表更新至第二版 |
| [tools/hd-opening-live_test.go](tools/hd-opening-live_test.go) | 真實 OP-FULL／VISUAL 索引的全畫面入口、指標別名、來源變動、目的變動、自然返回、舊圖層撤銷及捲軸資格測試；不取代普通 GUI 驗證 |
| `catalog/decoded/OP-FULL.OZM.idx`、`catalog/decoded/OP-FULL.OZM-runtime-source.json` | 正常 RAM 來源的獨立索引及證據 sidecar，供既有來源目錄讀取；不來自靜態解碼器，不改 inventory 的五項靜態未知 |
| `remaining/history/runtime/`、`remaining/history/desktop/`、`remaining/history/desktop-missing/`、`remaining/history/regression/`、`remaining/history/title/` | 完整複製接入的隔離建置與匹配來源快照；普通歷史簡介、OP-FULL 缺圖、標題及首戰的真實視窗收據留本機 |
| [tools/verify-hd-opening.py](tools/verify-hd-opening.py) | 固定 GUI image 內專案及原版唯讀、history/desktop/palette、desktop-missing/palette 或 effects 可寫掛 `/output`，以 `--mode opening`／`opening-missing`／`opening-effects` 驗證 OP4 與 OP-FULL；缺圖沿已驗證正常步數，效果模式抽測六個亮度及兩個循環時點。三模式都走普通歷史簡介，啟動時保存驗證器全文 |
| `remaining/history/palette/` | `verify-hd-opening.py --mode opening-palette` 以同一普通輸入觀測 OP4／OP-FULL 各三次原生 DAC 與實際視窗；保留初次一般 palette 不能涵蓋整段開場的收據，不把短暫匹配當穩定驗收 |
| `remaining/history/palette-late/` | 同一工具的 `--mode opening-palette-late` 核對 OP-FULL 晚期兩次原生色盤與完整 RAM，辨別亮度上限變化及來源專用目標色盤 |
| [tools/verify-hd-opening-palette.py](tools/verify-hd-opening-palette.py)、`remaining/history/palette-effects.json` | 核對六次亮度取樣、兩次循環取樣與 IDA 的遊戲色盤常式；原生 48 通道契約及 HD 色彩轉換近似分開記錄 |
| [tools/hd-graphics-effects.go](tools/hd-graphics-effects.go)、[測試](tools/hd-graphics-effects_test.go) | 只接受 OP4／OP-FULL 已核對的完整色盤族，在 host 畫布重現亮度上限及循環；新 24-bit 圖的色彩轉換為明示近似，未知色盤退回原版 |
| `remaining/history/runtime/palette/`、`remaining/history/desktop/palette/`、`remaining/history/desktop-missing/palette/`、`remaining/history/regression/palette/`、`remaining/history/title/palette/`、`remaining/history/effects/` | 專用色盤、第二版雲帶及色彩效果接入後的獨立版本與匹配來源；原始失敗、色盤診斷及先前建置保持原樣 |
| [tools/hd-graphics-effects.go](tools/hd-graphics-effects.go)、[測試](tools/hd-graphics-effects_test.go) | OP4／OP-FULL 的精確原生色盤族識別與 HD 顯示轉換近似；其他來源及未知色盤維持原版備援，全部亮度級與 22 循環級另做測試 |
| [tools/verify-hd-opening-summary.py](tools/verify-hd-opening-summary.py)、`remaining/history/verification-summary.json` | 固定 GUI image 內專案唯讀、history 可寫覆掛相同路徑，彙整完整複製、八時點色彩效果、缺圖、正常標題及首戰的匹配來源與收據；不宣稱局部／捲動路徑完成 |
| `remaining/history/effects/ready/` | 最後一次 GUI 暫停由 549M 跨至色盤更新邊界，保留上層失敗的七組擷取與驗證器；此目錄在已確認的 548M 起等待，仍要求全部 48 通道匹配，不放寬產品條件 |
| [tools/verify-hd-opening-partial-source.py](tools/verify-hd-opening-partial-source.py)、`remaining/history/partial/` | 固定 GUI image 內專案唯讀、partial 可寫覆掛原路徑；以既有普通 600M 的匹配來源快照核對交鋒、歷史劇情、武將及眼部的來源／目的索引、裁切、矩形外不變與 OP9 保留緩衝區。只作原版證據，不宣稱 HD 完成 |
| [tools/hd-opening-partial.go](tools/hd-opening-partial.go)、[測試](tools/hd-opening-partial_test.go) | 未位移開場局部繪圖的固定參數、原版來源偏移及兩組精確色盤；歷史劇情沿原有呈現。維持完整來源指紋，READY 與抽樣限制見 RESEARCH-LOG |
| `remaining/history/partial/runtime/`、`partial/desktop/`、`partial/desktop-missing/`、`partial/title/`、`partial/regression/` | 本批隔離建置、匹配來源快照及普通 GUI 收據；既有完整複製與原版診斷不覆寫 |
| `remaining/history/partial/runtime/ready/` | 首輪測試 fixture 將既有 NRGBA 指標誤作介面轉型，保留上層編譯錯誤；修正測試型別後的新建置與 84 項來源快照，產品契約不改 |
| [tools/verify-hd-opening-partial.py](tools/verify-hd-opening-partial.py) | 固定 GUI image 中專案／原版唯讀，乾淨擷取目錄可寫掛 /output；普通歷史簡介局部 HD／缺圖的逐格擷取，短武將段落使用既有 F7 逐次暫停控制。完整慢速擷取外層期限至少 3000 秒；正式彙整使用下列 capture 與 full 工具 |
| [tools/hd-prefix-extract.py](tools/hd-prefix-extract.py)、`remaining/source-boundary/art-prefix/` | 固定 GUI image 中專案與原版唯讀、art-prefix 可寫掛 /output；僅擷取四項未完整解碼 OZM 的已知平面與索引位元，參考 PNG 將未知八像素設透明。不補來源 byte，不宣稱正常載入或正式素材資格 |
| [tools/hd-prefix-prepare.py](tools/hd-prefix-prepare.py)、`catalog/remaining/partial-story/` | 固定 GUI image 中專案唯讀、partial-story 可寫掛 /output；核對兩組已知來源及既有生成圖，準備四項來源的共用素材、已知索引及幾何。原圖對照僅留本機；來源外八像素以未知標記保留，READY 後才接入執行器 |
| [tools/hd-prefix-story.go](tools/hd-prefix-story.go)、[測試](tools/hd-prefix-story_test.go)、[對照表](tools/hd-prefix-profiles.go)、[桌面載入](tools/desktop-prefix.go) | 四項部分來源故事圖的三完整平面／第四前綴識別、完整自然返回 guard、已知原生索引與來源外八像素保留；只有既有已證實故事消費端取得資格，不冒稱四項正常路徑已抽樣 |
| `catalog/remaining/partial-story/runtime/`、`partial-story/desktop/`、`partial-story/history/` | 部分來源故事接入的隔離建置與同一凍結來源之首戰／歷史 GUI 回歸；原版來源與已知索引只留本機 |
| [tools/verify-hd-prefix-runtime.py](tools/verify-hd-prefix-runtime.py)、`partial-story/verification-summary.json` | 固定 GUI image、原版及專案唯讀，history 可寫掛 /output，以 --mode history 驗證普通歷史 600M 及四項素材載入；改掛 partial-story、--mode summary 彙整首戰與 90 項來源。執行前複製同一建置的 engine、startup-probe、二進位及完整 tools 來源快照；不宣稱四項場景已正常抽樣 |
| `assets/hd-b/art/partial-story/` | 部分來源故事圖的生成圖衍生素材、純 HD 總覽與準備 metadata；由 archive-hd-art.py 保存，沒有原版來源 PNG、索引或平面 |
| `catalog/art/RAINEND-hd-b-v1.png`、`catalog/art/RAINEND-generation-v1.json` | RAINEND.PCX 的出版商標誌 HD 草稿與完整提示詞；保持原構圖及品牌字樣，底部原版文字另留，PCX 正常消費端與顯示模式未確認，不接 production path。生成圖同步 assets/hd-b/art |
| `catalog/art/E-4-generation-v1.json`、`B-6-generation-v1.json` | 四項尾端未知 OZM 的兩組手繪草稿、有序參考圖用途與 SHA；生成圖及完整提示詞同步私人 assets/hd-b/art，未知八像素、幾何與來源資格另列，不自動計入 prepared |
| [tools/verify-hd-opening-scroll-source.py](tools/verify-hd-opening-scroll-source.py)、`remaining/history/scrolling/` | 固定 GUI image 中專案唯讀、scrolling 可寫掛 /output；從既有正常原版抽樣核對 OP2／OP8 的逐位來源位移、線性跨列複製及實際色盤。超出正式來源的像素另記未知，不據此推測美術或啟用正式替換 |
| [tools/hd-opening-shift.go](tools/hd-opening-shift.go)、[測試](tools/hd-opening-shift_test.go) | OP2／OP8 原版自然繪圖的線性位移來源資格、跨列 HD 取樣與來源外像素保護；規格及限制見 RESEARCH-LOG，沿既有 GUI 建置入口 |
| [tools/verify-hd-opening-shift.py](tools/verify-hd-opening-shift.py) | 固定 GUI image 中專案及原版唯讀，scrolling/desktop 或 desktop-missing 可寫掛 /output；先從 scrolling/runtime 的同一建置複製 engine、startup-probe 與二進位。再在容器執行 `python3 -c 'from shutil import copytree; copytree("/project/tools", "/output/source-snapshot/tools")'` 保存來源及授權快照，之後執行本工具，以 --missing 切換個別缺 OP2／OP8。抽測早晚位移及未知尾端，並完成普通歷史 600M；每次外層期限 1800 秒，程序由工具有界清理 |
| [tools/verify-hd-opening-shift-summary.py](tools/verify-hd-opening-shift-summary.py)、`scrolling/verification-summary.json` | 固定 GUI image 中專案唯讀、scrolling 可寫掛 /output；核對兩次捲動視窗完整收據、86 項凍結來源、所有配對圖片及首戰回歸，總表只涵蓋已驗證切片 |
| `remaining/history/scrolling/runtime/`、`scrolling/desktop/`、`scrolling/desktop-missing/`、`scrolling/regression/` | 捲動開場的隔離建置、完整普通歷史狀態、配對畫面與首戰回歸；原版及固定 dosgolem 唯讀，同一建置凍結後不改 Go |
| `remaining/history/scrolling/regression/ready/` | 補齊 86 項凍結來源快照後，以相同二進位與命令乾淨重跑首戰；上層保存第一次缺字型授權快照的驗證器失敗 |
| [tools/verify-hd-opening-partial-full.py](tools/verify-hd-opening-partial-full.py)、`partial/full/`、`partial/full-missing/` | 同一二進位及 84 項來源的快速普通歷史簡介完整 600M 對照，使用既有原圖／HD 按鍵。局部擷取另驗收，避免把短暫畫格的操作競態或收尾逾時當產品缺陷 |
| [tools/verify-hd-opening-partial-captures.py](tools/verify-hd-opening-partial-captures.py)、`partial/capture-verification.json` | 重核已保存的正常 14 組與缺圖 17 組 X11／呈現 PNG、原生來源裁切、HD 公式與切換 proof；兩次提前停止如實記錄，完整原版狀態須另由 full／full-missing 收據證明 |
| [tools/hd-frame-prepare.py](tools/hd-frame-prepare.py) | 既有手繪 VISUAL 上方框架依原版矩形分區縮放；輸出 `workplace/hd-prototype/scroll/frame/` 的來源、HD 圖與幾何收據，遊戲內驗證另列 |
| [tools/hd-frame.go](tools/hd-frame.go)、[tools/hd-frame_test.go](tools/hd-frame_test.go) | VISUAL 上方紅金框架的來源核對與呈現；中央劇情圖及下方捲軸保持獨立，遮擋時完整備援 |
| [tools/hd-window-extract.py](tools/hd-window-extract.py) | 核對 WINDOW.OZM 九個 16×16 圖塊、原版 buffer 與正常視窗；輸出 `workplace/hd-prototype/catalog/window/` 的來源、原始圖塊、手繪參考與來源收據 |
| [tools/hd-window.go](tools/hd-window.go)、[tools/hd-window_test.go](tools/hd-window_test.go) | 共用視窗繪製及保存／還原的唯讀觀測、九圖塊 HD 呈現與原版前景保留；READY 契約及驗收見 RESEARCH-LOG |
| [tools/verify-hd-window.py](tools/verify-hd-window.py) | 核對五次共用視窗 GUI 收據、完整 RAM、原版狀態及建置來源；固定 Python Docker 中專案唯讀掛 `/project`、`workplace/hd-prototype/catalog/window/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-window.py` |
| [tools/archive-hd-art.py](tools/archive-hd-art.py) | 依 2026-10-14 使用者授權保存生成的 HD 圖與提示詞；固定 GUI Docker 內以專案掛 `/project` 可寫執行，`--verify-only` 核對索引與檔案 |
| [assets/hd-b/README.md](assets/hd-b/README.md)、[圖片索引](assets/hd-b/index.json) | 私人儲存庫內的生成圖、生成圖裁切素材與提示詞。使用者已明確授權保存；不是正式發行包或公開散布授權 |
| [tools/hd-kao-profiles.go](tools/hd-kao-profiles.go) | KAO.GRP 原版 0..80 索引的逐張指紋，不以推測武將姓名命名 |
| [tools/hd-object-prepare.py](tools/hd-object-prepare.py)、[tools/hd-terrain-object-profiles.go](tools/hd-terrain-object-profiles.go) | 26 項物件來源的 alpha、裁切及審查；只有 review 的 approved_indices 進入共同呈現，兩個空白格保留 |
| [tools/verify-hd-objects.py](tools/verify-hd-objects.py) | 固定 GUI Docker 內逐項核對物件來源索引、alpha、載入 profile、完整 RAM、切換及前版真實視窗；專案唯讀掛 /project、objects/expanded 可寫掛 /output |
| `workplace/hd-prototype/catalog/terrain/objects/expanded/` | 物件裁切、原圖對照、審查及隔離 runtime／desktop 收據；生成圖、裁切與提示詞保存於 assets/hd-b/terrain/objects |
| [tools/hd-wall-prepare.py](tools/hd-wall-prepare.py) | 以已生成的橫牆、直牆、城門及斷牆分區校準 16 項牆件，保留原生透明形狀；來源索引與像素角色不作玩法推論 |
| [tools/hd-background-prepare.py](tools/hd-background-prepare.py)、[tools/hd-graphics-static-profiles.go](tools/hd-graphics-static-profiles.go) | 校準 BATTLE01／BATTLE02／OP2／OP4／OP8 的實際美術區，保留原版緩衝區；共用不透明繪圖端使用來源指紋、原始矩形及可見片段 |
| [tools/hd-logo-prepare.py](tools/hd-logo-prepare.py)、[tools/hd-graphics-logo-profile.go](tools/hd-graphics-logo-profile.go) | 校準 OP1 的 Family Soft 字標，保留品牌字樣及原版位置；輸出與審查位於 graphics/static/logo，生成素材及提示詞保存至 assets/hd-b/art |
| [tools/hd-graphics-palettes.go](tools/hd-graphics-palettes.go) | 依 palette-probe 七次原版擷取限定 OP1 的獨立穩定調色盤；其他來源維持既有十六色，淡入／淡出保留原圖 |
| [tools/verify-hd-background.py](tools/verify-hd-background.py) | 固定 GUI image 內專案唯讀掛 `/project`，`graphics/static/desktop-logo/palette/ready/` 可寫掛 `/output`，以 `--mode logo` 等待原版獨立調色盤穩定再驗證無輸入開場字標；上層保留首次結果及驗證器欄位錯誤。改掛 `graphics/static/` 並省略旗標核對六圖、留白及獨立 GUI 收據。只聲明實際驗證的正常來源 |
| `workplace/hd-prototype/catalog/graphics/static/` | 六項美術校準與原圖對照；normal-intro 保存無輸入查證，runtime 保存隔離建置，desktop 保存首戰回歸，desktop-logo 保存開場字標切換，三者的 palette 子目錄保存獨立調色盤修正後版本。palette-probe 保存原版 OP1 自然淡入／穩定／淡出擷取。生成圖衍生物與完整提示詞另存 assets/hd-b/art/static |
| [tools/verify-hd-walls.py](tools/verify-hd-walls.py) | 核對完整 24 項非空物件、校準圖重用、原有八項不變及新版正常首戰；固定 GUI Docker，專案唯讀掛 /project、objects/walls 可寫掛 /output |
| `workplace/hd-prototype/catalog/terrain/objects/walls/` | 城牆分區、接縫審查與隔離驗證；每版草稿分目錄保留，生成材質衍生圖與幾何紀錄另存 assets/hd-b/terrain/objects/walls |
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
| [tools/vbe-timing.go](tools/vbe-timing.go) | VBE 0100h／0101h BIOS 模式沿用 dosgolem 已有標準 VGA 掃描時序表；只設定時序，限制及近似等級見 RESEARCH-LOG |
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
| [.gitattributes](.gitattributes) | 僅對已登記的平台補丁排除 unified diff context 前綴的空白警告；其餘檢查保留。套用後另由 git apply、gofmt 與平台測試驗證 |
| [.gitattributes](.gitattributes) | dosgolem 統一差異的 context 行保留格式；只排除該補丁檔的 space-before-tab 誤報 |

新增目錄或文件前讀 `/home/anr2/.codex/knowledge-base/local/project-document-responsibilities.md`，並更新本表。未有可執行待辦前不另建 WORKLIST；需要時只建立一份。README 只放穩定入口，工作歷程寫入 WORKLOG。

## 全美術 HD 研究入口

戰鬥背景與開場徽記的原始生成紀錄為 `workplace/hd-prototype/catalog/art/remaining-generation-v1.json`，原版參考沿用 `catalog/graphics/source-references.json`。BATTLE01／BATTLE02／OP4 的原始生成圖與提示詞保存至 `assets/hd-b/art/`；此紀錄只證明製作，裁切、文字與 runtime 整合仍需分項驗證。

物件擴充入口為 `tools/hd-object-prepare.py`、`tools/hd-terrain-object-profiles.go` 與 `tools/verify-hd-objects.py`。`workplace/hd-prototype/catalog/terrain/objects/expanded/` 保存 26 項來源的裁切審查、原圖對照、準備後圖片及隔離 runtime／desktop 驗證；生成圖的裁切與提示詞另存 `assets/hd-b/terrain/objects/`。原版來源仍留在既有 `objects/sources/`，24、25 空白不得補圖。這批沿用共同前景 renderer 的來源、alpha、遮擋及備援契約，不推定地圖物件用途。

`workplace/hd-prototype/catalog/terrain/backgrounds/` 保存原版圖塊繪製前後的唯讀背景診斷、來源及後續 GUI 驗證。`tools/hd-backdrop.go`／`tools/hd-backdrop_test.go` 的 `-hd-backdrop-trace` 只在原版 sub_1F9D0 入口與自然返回讀取 native snapshot、原始四平面及 DAC，輸出有界的原始背景／來源／返回圖。`tools/verify-hd-backdrops.py` 獨立核對十一項來源、原版透明合成、十五欄基準、完整 RAM 與當次來源快照。沿用固定 GUI image，專案唯讀掛 `/project`，該目錄可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-backdrops.py`。READY 呈現契約見 RESEARCH-LOG；診斷收據本身不代表 HD GUI 已驗收。

該目錄的 `runtime/` 是後續呈現建置與測試；`desktop/`、`desktop-only/`、`desktop-missing/` 保存四個樹叢的三模式真實視窗與來源快照。頂層診斷及 `sprite-backgrounds/` 不覆寫。`tools/verify-hd-backdrop-gui.py` 核對三模式、原版狀態與實際擷取背景，使用固定 GUI image，該目錄可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-backdrop-gui.py`。

`workplace/hd-prototype/catalog/terrain/scene/` 保存完整地面順序、可見碎片及重疊人物的研究與後續驗證。`tools/hd-scene-probe.py` 從既有正常路徑 145 個自然返回重建原版來源與遮罩，對照原版畫面，游標區單列排除；不把重建圖當原版收據。固定 GUI image 內將專案唯讀掛 `/project`、該目錄可寫掛 `/output`，執行 `python3 /project/tools/hd-scene-probe.py`。呈現契約與各推論等級見 RESEARCH-LOG。

`tools/hd-terrain-scene.go`／`tools/hd-terrain-scene_test.go` 保存地面完整返回順序與重疊前景的共同呈現及安全測試；`tools/hd-terrain-unit-profiles.go` 只批准 units-001／003 的原始來源範圍、生成圖 SHA 與明示裁切，其他單位不沿用。`scene/runtime/`、`scene/desktop/`、`scene/desktop-only/`、`scene/desktop-missing/` 分別保存隔離建置與三模式 GUI。建置沿 run-desktop.sh；正常路徑驗證沿 verify-desktop.py 的 scene 三模式，完整收據與獨立核對完成前不稱全遊戲 HD。

`tools/verify_hd_scene_pixels.py` 是 scene 真實視窗驗證的獨立 Pillow 像素核對器，依原始來源、完整順序、明示裁切與 alpha 契約計算畫面。隨 verify-desktop.py 複製至同一可寫輸出目錄，來源唯讀；游標區只從此像素公式核對排除，真實 X11／呈現圖及原版 Proof 仍完整核對。

`tools/verify-hd-scene.py` 彙整 scene 三模式、完整 RAM、十五欄基準、來源快照、原始順序及素材 SHA。固定 GUI image 內專案唯讀掛 `/project`，`scene/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-scene.py`。歷史重核可用 `--runtime-source-root` 指向當次含 tools/ 的來源快照，`--verify-only` 不改寫總表。

`workplace/hd-prototype/catalog/art/ui-generation.json` 保存戰場介面生成草稿的來源、參考、完整提示詞與 SHA；生成原圖與提示詞同步至 `assets/hd-b/art/`，metadata 由 archive-hd-art.py 保存。圖像布局及正式呈現資格另驗證，生成草稿不算原版介面已接入。

`workplace/hd-prototype/catalog/battle-frame/` 保存 MAPGAMEN 原版全畫面繪製的自然返回、當時 DAC、原生索引與框架製作。`tools/hd-battle-frame.go`／`tools/hd-battle-frame_test.go` 目前只觀測已知載入與 sub_1C084 自然返回，完整索引 SHA 不符不授予資格，不先替換畫面。固定 Go image 內該目錄可寫掛 `/output`，沿 run-probes.sh 加 `-hd-battle-frame-trace` 重跑普通路徑；其 `source-probe/` 保留診斷，後續建置及 GUI 另列，不能覆寫來源收據。

`tools/verify-hd-battle-frame-source.py` 獨立核對來源索引、實際 DAC／PNG、自然返回、十五欄與完整 RAM。固定 GUI image 內專案唯讀掛 `/project`、`battle-frame/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-battle-frame-source.py`。來源快照綁定當次工具，不把來源驗證當作 HD 框架已完成。

`tools/hd-battle-frame-prepare.py` 以明示分區座標校準已生成的 MAPGAMEN 框架，保存純生成圖的 HD skin、來源遮罩、幾何及本機預覽；原版遮罩與含原版的預覽不加入 Git。固定 GUI image 內專案唯讀掛 `/project`、`battle-frame/` 可寫掛 `/output`，執行 `python3 /project/tools/hd-battle-frame-prepare.py`。這是可丟棄幾何試作，READY 及正常 GUI 驗收另列。

`tools/hd-battle-frame-profile.go` 固定選定 v2 skin、來源遮罩與十六色；`tools/hd-battle-frame-render.go`／`tools/hd-battle-frame-render_test.go` 實作框架、徽記／天氣整區資格、逐索引前景及共同場景底圖。`battle-frame/runtime/`、`desktop/`、`desktop-only/`、`desktop-missing/` 保存後續隔離建置與真實視窗，來源探針及草稿版本保留不覆寫。

`assets/hd-b/battle-frame/` 只保存生成圖校準出的 v1／v2 skin、完整提示詞、幾何 metadata 及 v1 工具快照。archive-hd-art.py 核對純生成產物與來源關係；不保存原版索引、來源遮罩及含原版的預覽。正式呈現與正常路徑資格見 CONTEXT。

`tools/verify_hd_battle_frame_pixels.py` 依原版索引、實際 palette、固定幾何與場景影響區獨立計算框架。真實視窗驗證開啟 `-hd-battle-frame-proof`，擷取原生及 48×48 背景索引至同一輸出目錄；它們維持本機，不加入素材索引。此核對器與 verify_hd_scene_pixels.py 隨 verify-desktop.py 複製到各 GUI 輸出目錄。

`tools/verify-hd-battle-frame.py` 彙整兩個正常畫面、三種 GUI、來源與幾何、工具快照、完整 RAM 及十五欄基準。固定 GUI image 內專案唯讀掛 `/project`、`battle-frame/` 可寫掛 `/output`，執行 `python3 /project/tools/verify-hd-battle-frame.py`。歷史重核可指定 `--runtime-source-root` 與 `--verify-only`。

`workplace/hd-prototype/catalog/graphics/` 保存其餘 OZM 的原圖、生成參考、共同繪圖診斷及後續驗證。`tools/hd-remaining-art-extract.py` 只讀已解碼且未製作的來源，依已核對十六色重生本機 PNG、放大參考與總覽；不填補未支援的五個 OZM。固定 GUI image 內專案唯讀掛 `/project`、該目錄可寫掛 `/output`，執行 `python3 /project/tools/hd-remaining-art-extract.py`。原圖與混合總覽不加入 Git，生成美術沿 assets/hd-b/art 保存。

`tools/hd-graphics-trace.go`／`tools/hd-graphics-trace_test.go` 是共用 OZM 不透明繪圖的唯讀診斷：以實際 RAM 四平面指紋識別來源，記錄 sub_1C1BC／sub_1C084 的入口與自然返回，逐索引核對輸出。`graphics/source-probe/` 保存有界普通玩家路徑、來源快照及診斷。固定 Go image 內該目錄可寫掛 `/output`，沿 run-probes.sh 加 `-hd-graphics-trace`；診斷只讀，正式呈現由 hd-graphics.go 依已審查契約處理。

`battle-frame/desktop-attempt-01/`、`desktop-only-attempt-01/`、`desktop-missing-attempt-01/` 保留首輪驗證器階段假設與切換次數不符的輸出；不是產品缺陷或成功收據。正常 658M 後畫面有能力／確認視窗遮住徽記，應驗證徽記整區備援，不能假設已是裸框架。

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

桌面程式只有使用 `-desktop` 才開視窗，預設仍為無頭。固定 `-program mark -steps 75000000` 可獨立顯示包內 MARK 標誌與 HD；沿用同一 Docker 桌面命令及功能列，640×480 時畫布為 1280×960。預設 `-program game` 仍直接進 DX5，完整啟動器串接待做。桌面互動預設 `-startup-display ask`，在原版成功指令數為零時選原版／HD；自動驗證須明示 `-startup-display original` 或 `hd`。選擇畫面及確認手勢放開前不步進 DOS、不轉送按鍵或滑鼠點擊；進入後與功能列共用顯示設定。F6 切換原圖／HD，F7 暫停試作步進，F9 擷取強化畫面。滑鼠移到頂端 6 個原版邏輯像素會展開功能列，移開自動收起。Esc 固定／收起；叉號只收起，Shift+Esc 或工具選單可關窗並輸出原生收據。功能列顯示時原版停止步進，畫面下移並增加視窗高度；收起後滑鼠按鍵全部放開才恢復步進與滑鼠轉送。Tab 切換欄位，方向鍵選取，Enter 展開／套用。顯示提供原版及 HD 圖像，肖像與捲軸分項缺圖回原圖；兩項素材都無效時 HD 列停用；執行提供繼續／暫停，工具提供截圖／離開。

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
