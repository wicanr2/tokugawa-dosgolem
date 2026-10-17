# 工作歷程

入口：[AGENTS.md](AGENTS.md)。目前狀態：[CONTEXT.md](CONTEXT.md)。

## 2026-10-11：取得 jsdos 與建立 DOS／dosgolem 工作規範

- 目標：下載指定遊戲頁實際使用的 jsdos，撰寫本專案 AGENTS.md。
- 路由：載入 dosgolem README／CLAUDE、規格索引、DOS/4GW 能力矩陣與文件職責規範；採用 `reverse-engineer-retro-game-remake` 的原始檔保護與證據界線。
- 來源追查：遊戲頁為前端應用。由其 JS 確認 API 與 binary 網域，再由 API 取得 `PLAY.BAT` 與固定 ZIP 名稱。完整來源與逐檔雜湊保存在 `original/source.json`。
- 新目錄理由：初始工作目錄沒有專案檔。`original/` 保存本機原始輸入；`tools/` 保存可重現的下載／驗證腳本；`workplace/` 保存可重生探針與快取，符合文件職責規範。沒有建立發行目錄。
- 下載：最初 128 MiB 防護上限中止。經 HTTP HEAD 確認來源 ZIP 實際為 214,665,581 bytes，再以精確大小與 SHA-256 固定來源，續傳剩餘部分。這是下載大小限制，沒有修改遊戲內容。
- 結果：完整 ZIP SHA-256 符合 API 檔名，CRC 通過。檢查路徑穿越、symlink 與大小寫重複後解包；14 筆 entries、12 個檔案。原始檔設為唯讀，擁有者為 1000:1000。
- 啟動盤點：`PLAY.BAT` 呼叫 `DX5.EXE`；jsdos 設定另掛 `DX5.cue`。主程式為 MZ 內嵌 LE，光碟宣告一軌資料與十五軌音樂。保留亮度修正、BAK 與已發現但語意未知的額外差異。
- dosgolem：唯讀掛載既有 checkout，使用 Go 1.24.13 跑 `go run ./cmd/leprobe -exe /original/DX5.EXE`。exit 0、stderr 空；解析 40 pages、2 objects、4,831 fixup records，完成 relocation preview。沒有執行 entry-prefix 或遊戲。收據在 `workplace/leprobe/`，命令在 AGENTS。
- 文件：建立 AGENTS、CONTEXT 與本歷程；AGENTS 收錄所有新增文件、工具與證據入口。`.gitignore` 排除原版檔、光碟、音軌與工作產物。
- 環境：合併 shell 命令、heredoc 與外層 timeout 包裝曾因 Docker socket 權限失敗；單一已核准 Docker 命令可用。提升權限請求被環境政策拒絕，改用單一 Docker 命令完成；工作負載維持在容器。LE 探針另由工具層監督外層期限，容器內有 170 秒 timeout。
- 收尾：完整性、擁有權與工具雜湊由 `tools/verify-source.py` 重查，結果存入 `workplace/intake-verification.json`。本批容器均使用 `--rm`，最後核對無 `tokugawa` 殘留；沿用既有 image，未建新 image。
- 未完成範圍：本作 dosgolem 執行、正常玩家路徑、存讀檔、畫面與音訊都未驗證；未建立 remake。這些不在此次下載與文件任務內。
- Git：工作目錄尚未初始化為有效 Git 儲存庫，沒有 commit、push 或公開發布。

## 2026-10-11：原版與 dosgolem 接入

- 使用者澄清目標是把原版整合到 dosgolem 執行，排除 remake；已授權建立 private repository。AGENTS 與待辦依此更新。
- 查讀 DOS/4GW 能力矩陣，確認既有 entry-prefix 使用 FD2 專用位址與 runtime，另建不跳過遊戲指令的有界啟動探針。固定同一 EXE 與 dosgolem HEAD，以 Go overlay 套用本專案保存的通用修正。
- IDA Pro 9.4 入口匯出保留原名、bytes 與原始位址。每個平台切片先在 RESEARCH-LOG 記錄 READY 契約，再補 CPU 指令與獨立回歸測試。沒有修改原版 bytes，也沒有回寫唯讀的 dosgolem 工作樹。
- 啟動探針已通過 PSP、環境、DOS 版本、DOS/4G 偵測、DPMI 配置與原版兩秒等待。虛擬時間採一微秒／成功指令的硬體規格近似，不跳過 busy-wait。
- 真實 CUE/BIN 掛載與 ISO 主描述子驗證通過。MSCDEX 子集處理掛載、drive list 與 IOCTL TOC；CUE PREGAP、file sector、disc LBA 與 Red Book 原點分開。非法 track 依規格回錯誤，不替原版改成有效值。CD 音訊未驗證。
- DPMI 0300h 先驗證封包，再轉接實模式主機服務；未知服務回到原有 guest handler。補上空 DBCS 表、既有滑鼠狀態查詢、BIOS 顯示偵測與 VGA 埠。VBE 分頁 VRAM 的資訊、模式、分頁與清除行為有獨立測試，仍待原版畫面驗證。
- DOSBox 輔助啟動先因工作副本 DAT 維持唯讀而失敗。只修正工作副本的 user-write 權限，乾淨重跑後看到標題畫面。截圖保留為輔助基準，未登錄成 dosgolem 收據，音訊使用 dummy。
- 回歸中的 CMPSB 初次失敗是測試沒有建立來源段 descriptor；修正測試後以同一工具鏈重跑。平台回歸持續核對 CPU386、machine、dosfile 與研究入口四套件。最新停止點與測試結果在 CONTEXT 所連的收據，不以步數當作可玩證據。
- GitHub：主機 `gh auth status` 的權限提升被自動審核拒絕，政策不允許 sandbox approval。受限環境的驗證結果不能判定主機憑證失效。GitHub plugin 已提示，但尚無可用工具；專案 `.git` 亦唯讀，沒有建立儲存庫或推送。
- Docker：沿用既有 Go、Python、IDA 與 DOSBox image。工作容器有資源上限、逾時、UID/GID 1000:1000 與 `--rm`。各批核對擁有權；目前沒有本專案殘留容器，未建立新 image。

## 2026-10-12：原生畫面、普通輸入與新遊戲敘事

- 範圍維持原版與 dosgolem 整合。平台修正保存在 `tools/dosgolem.patch` 與本專案轉接層，使用 Go overlay 重跑，外部 dosgolem 的既有 `cmd/probe/main.go` 修改未碰。
- 顯示：以公開 VGA 契約與 dosgolem 既有 CRTC 時序模型補回掃、BIOS 調色盤與游標服務。原版自行顯示 FamilySoft 開場、640×400 標題與主選單。所有正式畫面來自 dosgolem VRAM／DAC，DOSBox 擷取仍只列輔助證據。
- 輸入：建立有界普通滑鼠時間表，走通重新開始、預設姓名、肖像、能力編輯與兩次確認。原版可顯示人物與繁體中文敘事，普通左鍵逐頁推進。每次未按鍵的操作邊界保存原生畫面與精確雜湊。
- 查證：姓名編輯的邊界座標、能力編輯與連續兩次確認都依 IDA 9.4 的窄範圍匯出確認。沒有改遊戲位址、跳過指令或注入流程狀態；最初點擊無反應與看似重複確認的原因已在 RESEARCH-LOG 追加更正。
- 光碟：依 Microsoft MSCDEX 與固定 DOSBox Staging 參考補零長度播放的真實錯誤，以及閒置 STOP AUDIO。正長度播放仍未支援；TOC buffer 重疊與音訊驗證保留為獨立限制，不以原版繼續執行宣稱修好。
- 驗證：CPU386、machine、dosfile 與整合入口四套件通過。五億條指令重跑只因預算結束，仍為正常序章對話；下一次延長到七億五千萬步觀察後續。每次收據保存工具與原始輸入雜湊、輸入時間表、停止原因與原生畫面。
- 來源複驗：ZIP CRC、十二檔逐檔 SHA-256、唯讀與 UID/GID 1000:1000 通過。驗證工具原先用 stat 跟隨輔助目錄的容器絕對符號連結，造成 FileNotFoundError；改以 lstat 檢查連結本身擁有權，乾淨重跑通過。這是驗證工具問題，沒有修改遊戲資料。
- 文件：AGENTS 收錄全部工具與文件；README 只保留穩定入口，現況集中在 CONTEXT。沒有新增發行目錄或 remake 文件。
- GitHub：private repository 已獲授權，但主機 GitHub 權限提升仍被自動核准審核拒絕，且沒有可用 GitHub connector；尚未建立 repo、commit 或 push。
- 後續驗證：七億五千萬步抵達第一個劇情選項。原先對話換頁座標落在選項框外，所以保持等待；已保存畫面雜湊，改以框內座標選取第一項重跑。這是普通輸入路徑調整，沒有修改遊戲判斷。
- 長流程環境：選項重跑的外層程序以 143 結束，stderr 空，尚未完成收據；前次完整收據保留，沒有將此次中斷列為產品缺陷。改用同一 image 與命令的 detach 形式，容器內維持 600 秒期限、10 秒強制終止緩衝與 `--rm`。Docker wait 在受限環境無 socket 權限，改以已可用的 Docker ps 核對，不搬運憑證或在主機執行遊戲。
- 乾淨重跑結果：選項確實引出原版回應，並自然進入出陣選擇。七億五千萬步完整收據的 mouse_input_sha256 與本次輸入檔相符，四套件通過，DPMI 未支援計數為空。原生畫面與雜湊另記在 RESEARCH-LOG；下一次只探索首列武將選擇與右鍵結束。

## 2026-10-12：出陣操作與戰場載入指令

- 普通輸入已推進武將能力頁、上方確認、兵種與兵力清單，以及下方出擊確認。三個確認框的位置依原生畫面核對，沒有把框外點擊當作輸入失效。兵力清單不證明實際編組對應，過度描述已在 RESEARCH-LOG 追加勘誤。
- 695,100,000 步點下方 Yes 後，原版開始載入下一場景。依自然停止與 IDA 9.4 的窄範圍原始 bytes，補 83h register XOR imm8、D0h byte ROL、81h word XOR imm16、23h word／dword AND、03h word ADD memory、D3h word ROL CL 與 02h byte ADD memory。每項先記錄 READY 平台契約，再實作與測試。
- 回歸涵蓋暫存器高位與旗標、來源與目的相同、DS／SS／SIB、來源記憶體保存、截短指令與段界限。CPU386、machine、dosfile、整合入口四套件通過。補丁保存在本專案，不修改外部 dosgolem 或原版 EXE。
- 自然重跑依次通過已遇到的 XOR、ROL、AND 與 word ADD 缺口。最近一次完整停止在 695,114,712 步的 byte ADD memory；修正後正在以相同輸入檔、原始入口與虛擬時間重跑。尚未將載入途中背景列為戰場驗證。
- 文件職責與知識路由已複核。AGENTS 另澄清原包附帶補丁留本機，自製 dosgolem 平台修正可加入已授權的 private repository。GitHub 存取與 `.git` 寫入仍受環境限制，未建立 repo、commit 或 push。
- 後續自然通過 byte ADD，開始執行原版生成的繪圖程式。在 dosgolem 線性 `0x5C384` 遇到 ES byte XCHG；同段生成 bytes 還有 byte ROR 與 ADC imm8。三項通用形式補齊，獨立回歸與四套件通過。生成程式證據另標為執行期線性位址與記憶體雜湊，不誤列成原始 EXE 檔案偏移。
- 後續自然顯示戰場外框、小地圖及中央地形。依實際停止補 word NOT、word ADD imm8、word SUB imm16、word ADC、SS word CMP memory、byte NOT 與 dword AND memory imm8。各切片都有 IDA 9.4 的原始 bytes、檔案偏移、READY 契約及獨立旗標／狀態保存測試；四套件乾淨重跑通過。
- byte NOT 的首輪回歸發現舊拒絕樣本仍把 NOT DL 列為不支援。保留該拒絕測試，將樣本改為尚未支援的 IMUL DL；新增 NOT 正反向向量另行驗證。未因此擴張 IMUL 或刪除失敗檢查。
- 完整戰場地形已目視確認，精確 PNG 與原始色號雜湊記在 RESEARCH-LOG。這只證明地形繪製，單位操作、戰鬥、存讀檔與音訊尚未驗證。最近停止在 712,500,240 步的 AND memory，修正後以相同 390 筆普通輸入重跑。
- 後續出現人物單位與 `1／20 下午`，自然停止先後揭露 39h dword CMP memory register 與 F7h dword TEST register immediate。依 Intel 規格及固定 EXE 的 IDA 原始 bytes 補齊；獨立旗標、界限與失敗保存測試及四套件通過。
- 本輪整合啟動驗收：由原版 LE 入口完整執行 750,000,000 步，只因預算結束；390 筆普通滑鼠事件全部套用。末段持續執行原版戰場滑鼠查詢，六個後續邊界畫面與末幀均符合已目視確認的戰場。收據 SHA-256 `307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4`，工具、補丁與輸入雜湊逐項相符，無未支援 DPMI，無 FD2 攔截與指令略過。
- 驗證與清理：四套件通過；Python／shell 語法、文件索引與相對連結、UID/GID 及 root-owned／異常 `.md` 目錄檢查通過。路由再次命中 dosgolem、平台規格優先及文件職責，載入對應入口。Docker 無本專案執行中或已停止容器，沒有新增 image；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異。
- 交接限制：單位選取、移動、戰鬥與存讀檔尚未驗證，CD 正長度音訊未支援。本輪只完成原版啟動至戰場的整合驗收。GitHub 工具仍不可用；主機 GitHub 權限提升被自動核准審核拒絕，且 `.git` 唯讀，private repository、commit 與 push 均未建立。

## 2026-10-12：規劃 CD 音樂、四語與 HD 素材包

- 使用者要求後續支援規劃，並確認可切換 HD 素材包、繁中／簡中／英文／日文，以及桌面版；另明確排除瀏覽器。沿用原版 DOS 程式與 dosgolem，不建立 remake。
- 路由載入 dosgolem 入口、平台規格優先、本地化的顯示／語意隔離及文件職責；使用 grill-me／grilling 釐清範圍。已確認選擇記在 CONTEXT，技術方案與驗收仍以 WORKLIST 的 DRAFT 草案呈現。
- 事實複核：既有收據八次 CD PLAY 均為零長度。計畫先查 TOC buffer 重疊與播放長度來源，再接正常播放狀態與 PCM 輸出；沒有把播放器解碼成功當成原版選曲已修好。
- 文字查證：固定 EXE 與 DAT 找到部分已知文字的 Big5 bytes，偏移、雜湊與限制追加於 RESEARCH-LOG。LE 繪字與 HD 事件擷取仍待查證，未套用 16-bit Oracle 呼叫介面或取代原版指令。
- 規劃順序：桌面與擷取可行性、CD 音樂、四語顯示、同場景 HD 試作，再共同驗收。尚未選定美術風格、顯示比例、正式 schema、桌面工具版本或打包平台；未開始正式功能實作。
- 本輪只作唯讀查證與既有文件更新。Docker 一次性容器使用 `--rm`、網路隔離、資源限制與 UID/GID 1000:1000；未建新 image，收尾檢查容器、擁有權與文件連結。GitHub 狀態未變。

## 2026-10-12：GitHub 與 Issue 建立準備

- 使用者授權依序建立 private repo、登記 Issues、開工。功能範圍沿用桌面 CD 音樂、繁中／簡中／英文／日文與可切換 HD 素材包；開工第一項為 TOC／PLAY 資料流查證，正式技術切片仍須 READY 契約。
- 路由命中文件職責與工作清單入口，載入 `local/project-document-responsibilities.md` 及 `sources/claude/rulebook/61-worklist-as-data-with-verify.md`。沿用既有文件，不新增平行 Issue 狀態台帳。
- 依主機規則呼叫 `/home/anr2/.local/bin/gh auth status` 並指定 `require_escalated`。工具在執行前拒絕，政策顯示 `sandbox_approval: false`；本輪沒有執行成功的主機登入檢查，不能判定憑證失效。沒有改用容器 gh 或搬運憑證。
- GitHub plugin 查詢為未安裝，沒有可呼叫的 GitHub repository／Issue 工具。既有連接提示仍在，不重複提示安裝。
- 目前檔案狀態核實：`git status --short --branch` 回報非 Git 儲存庫；Docker 唯讀檢查確認 `.git` 不存在。前文的「`.git` 唯讀」描述的是環境路徑權限，不代表存在有效 `.git`。未初始化、commit、push 或建立遠端物件。
- 在 WORKLIST 準備 `tokugawa-dosgolem` 的 private 設定與六筆未送出 Issue 內容，涵蓋 CD 請求查證、CDDA、桌面、四語、HD 與共同驗收。尚無遠端 URL 或編號；恢復存取後先核對 GitHub 現況，再建立缺少的項目。
- 本輪僅整理建立內容與核實環境，未越過「repo → Issues → 開工」的順序開始正式功能實作。既有啟動收據與原版資料維持不變。
- 收尾：六筆 Issue 草稿、五份文件的相對連結、索引正對照、UID/GID、root-owned／異常 `.md` 目錄檢查通過；既有啟動收據 SHA-256 未變。沒有執行產品測試，沒有本專案殘留容器，未建新 image。

## 2026-10-12：主機 gh 建立工具

- 使用者再次明確授權使用主機 gh。主機 `gh auth status` 的提升權限仍在執行前被政策拒絕，沒有新增遠端物件；授權本身已足夠。
- 路由命中文件職責入口，新增 `tools/setup-github.sh` 並同時收錄於 AGENTS。沿用 `workplace/` 保存可重生的待送出本文，不新增平行 Issue 台帳。
- 腳本只讓 Docker 擷取 WORKLIST 的六筆本文；登入、repository 與 Issue 操作均使用主機 `/home/anr2/.local/bin/gh`。不搬運憑證，不使用容器 gh，不初始化 `.git` 或上傳原版資料。
- 依 GitHub CLI 官方手冊確認 private 建立、`--body-file` 與 API 分頁契約。重跑先查遠端所有頁，private 才繼續；同題 Issue 沿用、不覆寫；新 Issue 依序附上已建立的相依 URL。`--prepare` 不存取 GitHub。
- 遠端建立仍待在可使用主機 gh 的終端執行；不能把工具完成當成 repository 或 Issues 已建立。
- 驗證：固定 Python Docker 內檢查 shell 語法，以假 gh／Docker 命令驗證 private 建立、六筆 Issues、真實格式相依 URL、重跑不重建或覆寫，以及登入失敗、API 列舉失敗、公有同名 repo、多筆同題 Issues 時停止。另執行原封不動的本文擷取程式，產生六筆待送出本文並核對 UID/GID；既有啟動收據 SHA-256 未變。假 API 測試不等於 GitHub 實測。
- 首次測試的假 API key 正規式未含數字，將 I18N 誤讀為 I；修正測試工具後在同一 image 與命令乾淨重跑，全部通過。Docker 行內長指令另遇 socket 權限限制，改為唯讀掛載測試腳本，沒有回到主機執行測試。
- 收尾：文件連結、索引正對照、檔案擁有權、root-owned／異常 `.md` 目錄檢查通過；本機沒有初始化 `.git`。容器全部結束並移除，未建新 image；臨時測試腳本移除，待送出本文保留在已忽略的 `workplace/github-setup/`。

## 2026-10-12：repo 回報與 CD 只讀資料流

- 使用者回報已建立 repo，並提供 `https://github.com/wicanr2/tokugawa-dosgolem`。README、CONTEXT 與 WORKLIST 已登記網址，不宣稱已核對 private 設定或六筆 Issues。再次透過主機 gh 提升權限驗證登入，工具在執行前因 `sandbox_approval: false` 拒絕；沒有遠端讀寫。本機 `.git` 仍不存在，沒有 commit 或 push。
- 已詢問六筆 Issues 是否也完成，尚待回答。沿用使用者的 repo → Issues → 開工順序；本輪進行唯讀診斷與證據工具，尚未實作正式 CD 播放、四語或 HD 功能。
- 路由命中 dosgolem、IDA、平台規格、文件職責與 DOSBox／Xvfb 試玩入口；載入對應最小入口，使用 `use-ida-pro-9-4` 技能保留原始名稱與位址。沒有新增 remake、瀏覽器或公開發行範圍。
- 新增預設關閉的 `-cd-trace`。先記錄 READY 觀測契約，再以同一 Go image、原始輸入、3,000,000 步比較開／關。完整記憶體、原始色號、停止位置、trace 與中斷計數一致，四個平台套件通過；收據及雜湊追加於 RESEARCH-LOG，觀測切片達 CONFORMED。正式 750,000,000 步的啟動收據保持原樣。
- IDA 9.4 窄匯出與 dosgolem 前後快照確認：原版配置包裝保留 DPMI 回傳段號，呼叫者把它當平面指標；IOCTL 查詢與請求因此重疊。首筆音軌查詢的軌號被 `0x1A` 覆蓋，裝置回報 `0x8108`。尚未證實至零長度 PLAY 的完整因果鏈。
- 沿用既有固定 digest 的 DOSBox-X image，新增 `tools/aux-cd-trace.py` 並同時掛進 AGENTS。包內實際 DOS/4GW 返回段號 `07F1h`、`07F3h`，MCB dump 確認兩份配置資料前各有一個 paragraph。原版仍把段號當指標，TOC 回寫也覆蓋請求。這是輔助證據，沒有換名成 dosgolem 收據，沒有作聽覺驗收。
- DOSBox-X 的 PTY 命令最初失敗；讀固定版本 debugger 原始碼後修正 LF、終端模式與按鍵輸入，以同一 image 乾淨重跑通過。複合主機命令另遇 Docker socket 限制，改為已確認路徑的單一 Docker 呼叫。兩者均屬環境／驗證工具問題。
- 沒有改 DPMI 段號回傳、配置間距、原版 EXE、軌號或播放長度。下一步查證 MCB 回收／重配、軌道查詢與 PLAY 的資料流，再建立正式平台切片契約。
- 來源重新核對通過 ZIP CRC、12 檔雜湊、唯讀與 UID/GID 1000:1000。Docker 收尾與文件驗證結果接在下方。
- 最終驗證：輔助腳本乾淨重跑退出 0，暫存器及記憶體擷取一致，manifest 的逐檔雜湊通過；Python／shell 語法、文件連結與索引正對照通過。正式原生啟動收據 SHA-256 仍為 `307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4`。所有本輪檔案 UID/GID 1000:1000，root-owned／異常 `.md` 目錄檢查為空；本專案容器全部結束並移除，沒有新增 image。

## 2026-10-12：主機 gh 核對與 CD-TRACE 開工

- 使用者再次授權 gh，環境已開放主機存取。主機 `/home/anr2/.local/bin/gh auth status` 成功，帳號為 `wicanr2`。repo view 與分頁 Issues API 確認 repository 為 private，六筆 Issues 均存在且開啟；#1 尚無留言，不重複建立。
- 路由沿用 dosgolem、平台規格優先、工作清單與文件職責入口。更新 CONTEXT 與 WORKLIST 的目前狀態，保留前輪存取受阻的歷史。
- 本機初始化 `main` 分支，逐項核對準備提交的 28 份自製工具與文件。`git check-ignore` 確認 EXE、BIN 與執行收據被排除，原版資料不提交。
- 依 #1 擴充既有輔助工具的 `--toc-queries`，保存後續音軌查詢前後的原始 bytes。只新增有界觀測，不改原版記憶體、暫存器、配置或播放請求；執行結果另追加。
- 首批工具與文件提交 `39d34df` 已推送 `main`，遠端再次核對仍為 private；#1 已登記開工留言。主機 gh 沒有搬入容器，也沒有搬運憑證。
- 輔助工具首輪刪錯斷點，斷言停止；查固定版本的 `push_front` 後修正並核對 BPLIST，以同一 image 乾淨重跑通過。實際 DOS/4GW 重用相同 MCB 配置；前兩軌 MSF 起點保留，控制欄位仍被狀態覆蓋。
- dosgolem 正常入口重跑 150,000,000 步，33 筆原樣滑鼠事件前綴全部套用。102 筆診斷零丟棄，31 次音軌查詢全為 `8108h`。IDA 9.4 三份窄匯出與完整記憶體連起錯誤 TOC、原版取值、起點表、長度相減及第 10 軌零長度 PLAY。收據與雜湊已追加 RESEARCH-LOG，其他七次 PLAY 尚未逐次驗證。
- IDA 同時確認檔案偏移 `0x19395` 是網站版本略過 TOC 末軌檢查的分支。追加勘誤，保留先前未知結論的歷史與原始定位；沒有切換 `.BAK` 或修改 EXE。
- 四套件回歸通過，既有正式戰場收據雜湊未變。下一切片為標準 DOS 配置契約，尚未達 READY；本輪沒有實作正長度 CDDA、桌面、四語或 HD。
- 收尾驗證：輔助 manifest、Python 語法、文件相對連結、索引正對照與 UID/GID 通過。Git 差異檢查排除補丁檔的原始 context 縮排後通過；補丁本身已在四套件重跑時套用驗證。專案無執行中或已停止容器，未建新 image；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 修改。

## 2026-10-12：MCB 配置修正與 CDDA 正常流程

- 依 dosgolem、平台規格優先與文件職責路由開工。讀既有 MCB 規格、DPMI 1.0 及 FreeDOS 固定版本，先記 READY 契約，再實作明確啟用的 32 位 MCB 配置池；既有單向診斷配置器維持預設。
- 新模式包含 MCB 標頭、首次適合配置、釋放重用、相鄰合併、縮放、損壞與描述子耗盡錯誤。原版一段／兩段配置重用 `5C60h`／`5C62h`，前 15 筆長度與 CUE 相鄰軌差一致。第 136,777,435 步首次送出合法第 10 軌 PLAY，無修改 EXE 或軌號。
- 接續 READY 的 CDDA 切片，補 PLAY／STOP／RESUME、Audio Status、依虛擬時間推進的 PCM 串流，以及分塊、自然結束、換曲、非法邊界與真實 BIN 測試。含顯式 PREGAP 的播放保持拒絕；末軌原版異常長度不猜補。
- 四套件回歸與原版正常入口至戰場重跑通過：750,000,000 步、390 筆普通滑鼠事件，沒有未支援 DPMI 或指令。戰場畫面與先前驗收雜湊相同；完整收據 SHA-256 `06971d5bca19eb5103dd7c6b6a6c7bd759c5a2ed3b3b361d21fb79d26d6b2e36`。
- 原版自行選第 10、5、10、5、5、10、2、6 軌，無播放拒絕，輸出 19,818,728 個立體聲樣本框。以事件時間獨立計算長度，逐段比較 BIN 的 79,274,912 bytes，SHA-256 `d81c04498183d8cf64ef3409f50b51aeb0c10047cbe1049f13cf0a4ae9e9320a` 相符。桌面播放與人耳確認尚未完成。
- 來源重新核對通過 ZIP CRC、12 檔雜湊、唯讀與 UID/GID。首次文件檢查未掛 `/original`，對既有 DOSBox 輔助連結作 `stat` 時失敗；補掛唯讀原版並改用 `lstat` 檢查連結本身後，乾淨重跑通過，沒有產品缺陷。
- Docker 一次性容器均已結束並移除，未建新 image。文件連結、索引正對照、擁有權與差異檢查通過；補丁 context 的縮排由實際套用及回歸驗證。外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異。Git 只納入自製工具與文件，原版及 PCM 留在本機。
## 2026-10-12：CDDA 邊界完成，後續優先 HD

- 使用者指定完成 CD 音訊後先執行 HD 化。CONTEXT 與 WORKLIST 已調整順序，四語範圍保留。桌面呈現與人耳確認仍為後續依賴。
- 知識路由命中 dosgolem、平台規格優先與文件職責。核對 GNU libcdio 的 PREGAP 靜音契約及固定 DOSBox Staging v0.82.2 的 EOF 行為後，記錄 READY 契約。實作等長靜音與 lead-out 停止，原始 PLAY 參數完整保留，沒有改原版 EXE 或表格。
- 十五條 AUDIO 軌全段與 BIN 的獨立區段 SHA-256 一致；靜音跨越、暫停續播、分塊、超長長度與非法起點測試通過。末軌的受控極大請求輸出 6,452 磁區後停止。這是裝置測試，不宣稱正常玩家已選到末軌。
- 四套件回歸及正常入口至首場戰場重跑通過。完整收據 `c6b33e27450f09a6668cf77d848a13774c8e2780f7d96126b266da773570e075`；八次正常選曲、19,818,728 個立體聲樣本框與戰場 PNG 都與前次一致。獨立解析 CUE、計算時間並逐段比較全部 PCM 通過，詳見 RESEARCH-LOG。
- 來源 ZIP CRC、12 檔雜湊、唯讀與 UID/GID 再次通過。驗證命令首次漏 Docker 的 stdin 開關而沒有執行本文；補 `-i` 後同一 image 乾淨重跑通過，屬驗證命令問題。
- 收尾的相對連結、索引正對照、擁有權與 Git 差異檢查通過。無本專案殘留容器，未建新 image；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異。原版與 PCM 仍被 Git 排除。

## 2026-10-12：HD 風格確認、唯讀素材觀測與首張肖像試作

- CDDA 邊界修正提交 `2c520b5` 已推送 main，主機 gh 登記驗收並關閉 #2。桌面及人耳確認明列留在 #3，沒有用 PCM 技術驗證代替聽覺確認。
- 路由命中資產考古、IDA、平台規格及文件職責。使用共同決策與 imagegen 技能，先生成同場景 A／B 風格比較，再由使用者選 B 戰國手繪風。沿用原版 DOS 玩法、640×400 邏輯座標、桌面目標與原圖備援。
- 新增 DAT 候選目錄盤點與 IDA 原始交叉參照匯出。201 筆目錄、長度、偏移及讀寫端已查證。沒有把 112 個 OZM 副檔名當作完整美術覆蓋量；壓縮分支可能參考當前 VGA，因此先擷取原版實際解碼緩衝區，不猜獨立解碼器。
- 新增預設關閉的 `-hd-trace`。依 READY 契約只讀 backing RAM、名稱與四個平面，不略過指令、不讀 VGA latch。10,000,000 步開／關與完整 750,000,000 步正常玩家路徑的全部 RAM、畫面、PCM、trace 與輸入相符。完整觀測記錄 19 次載入、17 個來源，零錯誤、丟棄及待返回。四套件通過，正式 CDDA 收據保持原樣。
- 以相同 33 筆滑鼠事件前綴重跑 150,000,000 步，隔離保存 A-0.OZM 原版 RAM 與畫面。新增擷取工具核對四平面雜湊，全部 48,000 像素符合 `(160,50,320,150)`。IDA 的劇情圖呼叫端、四平面複製與目的位置互相符合；定位與雜湊追加於 RESEARCH-LOG。
- 以內建 image_gen 製作第一張 B 風格家康肖像，生成圖、完整提示詞、原始參考與 provenance 索引留在既有 `workplace/hd-prototype/`。沒有新增交付目錄，沒有上傳原版或衍生美術。
- 離線呈現試作通過原圖／HD／缺圖、來源不符、比例／透明度、界限及輸入不變驗證。HD 預覽矩形外零變動；缺圖與原圖 PNG 雜湊完全一致。原生邏輯比例沿用包內 `aspect=false`，未定正式顯示模式或素材包格式。這不是桌面執行中切換完成證據。
- 預覽測試首輪因 Go 預設並行量超過容器程序上限而失敗。保留同一 image、命令及上限，將 GOMAXPROCS 與建置並行量限制為 2 後乾淨重跑通過；屬工具環境問題。
- 下一切片為 #3 桌面呈現與 #5 唯讀繪圖事件、更新範圍及切換契約。其他人物、地形、介面及四語呈現未完成。本輪提交與最後清理結果接於下方。
- 收尾：四套件的完整觀測收據與目前所有整合工具雜湊相符，原 CDDA 完整收據未變；預覽獨立測試、提示詞與美術 provenance、原圖／缺圖逐 byte 核對通過。Python／shell 語法、文件相對連結、索引正對照、UID/GID、root-owned／異常 `.md` 目錄及 Git 差異檢查通過。無本專案殘留容器，未建新 image；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異。遠端再次核對為 private，原版與衍生圖的 Git 排除生效。
- 工具、測試與文件提交 `470d9e5` 已推送 main。[HD #5 進度留言](https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6122684480) 已送出並讀回核對，Issue 保持開啟。僅提交 15 份自製程式與文件，原版、RAM、PCM、擷取原圖及生成美術均未加入 Git。

## 2026-10-13：首張 HD 的桌面執行中切換

- 使用者授權繼續接入桌面與 HD 切換。路由命中 dosgolem、平台規格、正常玩家路徑與文件職責；載入對應入口，沿用 B 戰國手繪風、原版玩法與桌面範圍。沒有另開 remake、瀏覽器或發行工作。
- 檢查既有 GUI image 的實際工具，沿用 Go 1.24.13、Ebitengine 2.9.9、Xvfb、xdotool 與 ffmpeg。依賴鎖定在獨立 modfile，外部 dosgolem 及其 go.mod 唯讀。映像繼承標籤與實際版本不同，以實查及 build-info 為準，未建新 image。
- 先寫 READY 的首張桌面試作契約。原本無頭步進改成可分批執行，桌面 Update 只呼叫相同 CPU.Step；原始輸入仍按成功指令數套用。呈現預設仍為無頭，GUI 只在 desktop 建置標籤及明確旗標啟用。
- 新增 A-0 載入、四平面來源雜湊、原始繪圖五參數與堆疊返回的唯讀觀測。只接受已查證的矩形，再逐像素核對原圖；不符或缺圖即備援。共用既有呈現邏輯，游標最後合成，沒有略過原版指令或改 RAM／VRAM。
- 真實 Ebitengine 視窗的 F6 開／關／恢復、F7 暫停、F9 擷取與 Esc 關窗通過。HD 與缺圖兩次 150,000,000 步完整狀態、原生畫面與 PCM 均符合先前無頭基準；每張視窗擷取逐像素符合呈現圖。縮為 1000×700 後，普通滑鼠點擊仍對應原版座標並引出新遊戲開場，人工目視確認。
- 滑鼠驗證第一次因工具未處理 JSON null 事件清單而停止。保存失敗輸出，修正工具後在同一 image 與命令乾淨重跑通過；產品程式與原版未因這個失敗而修改。總核對保存修正前後工具雜湊與各次收據。
- 步進重構後完整無頭回歸通過 750,000,000 步及 390 筆原樣普通輸入。RAM、trace、戰場畫面、CD 裝置與 PCM 均保持一致；CPU386、machine、dosfile、研究入口四套件及新觀測、備援、游標測試通過。最新收據與範圍見 CONTEXT、RESEARCH-LOG。
- 本輪完成 Linux 容器內第一張肖像的桌面試作。正式素材包、其他人物／地形／介面、四語、一般鍵盤及桌面音訊仍待後續；原版與所有衍生圖僅留本機。最後文件、擁有權、Docker 清理及 GitHub 記錄接於下方。
- 收尾：Python／shell 語法、文件連結與索引正對照、全部新整合工具雜湊、UID/GID、root-owned／異常 `.md` 目錄檢查通過。既有 CDDA 收據未變；三種 GUI 與完整無頭驗證的收據均已獨立核對。Docker 無本專案執行中或已停止容器，未建新 image；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異。儲存庫仍為 private，原版、RAM、PCM、圖像及二進位中間產物的 Git 排除生效。
- 自製工具、依賴鎖定、測試與文件共 19 份提交 `7ca5c9a`，已推送 main。[桌面 #3](https://github.com/wicanr2/tokugawa-dosgolem/issues/3#issuecomment-6138108426) 與 [HD #5](https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6138127948) 的進度留言已送出並讀回核對，兩筆均保持開啟。後續文字另澄清滑鼠驗證使用合成 X11 事件，沒有宣稱實體硬體驗收。

## 2026-10-13：參照《三國演義》的頂端隱藏功能列

- 使用者指定參考 `/home/anr2/cht/softworld_san`。路由命中 GUI 還原、桌面試玩與文件職責；共同決策依既有授權實作，不另啟訪談。參考專案只讀，依三份 windowbar 原始碼及測試查證幾何、啟閉、輸入順序與視窗高度。
- 先記錄 READY 契約，再新增可測的狀態與幾何、桌面轉接及 X11 驗證。功能列提供原版／單張 HD、繼續／暫停、截圖／離開；未實作的四語與桌面音訊留後續。沿用參考的頂端 6 像素啟動、32 高列、24 高下拉選項與 Esc 固定／收起，Shift+Esc 關窗。保留 F6／F7／F9。
- 顯示期間停止原版，增加視窗高度而保持遊戲比例。設定操作核對完整 Proof，原版輸入不接收功能列的點擊。收起當幀及尚未放開的滑鼠手勢繼續隔離，原版恢復前先處理放開，避免讀到殘留按住狀態。
- 新增執行時讀取的完整 Unifont 與原授權全文，來源雜湊固定；沒有使用原版遊戲字模。沿用既有 GUI image、Go 1.24.13 與 Ebitengine 2.9.9，沒有新增工具鏈或依賴版本。
- 第一輪視窗比對因驗證工具假設選完下拉選項後列仍顯示而停止。實際自動收起符合參考；保存失敗輸出，改成先移回頂端再擷取，使用相同 image 及命令重跑通過。這是驗證前提錯誤，未列為產品缺陷。
- 初次 HD、缺圖及普通滑鼠測試通過後，檢查發現原版有可能在收起手勢尚未放開時讀到先前按住狀態。補上步進隔離，並增加未手動暫停時按住叉號收起的真實 X11 驗證。最新三模式乾淨重跑與收尾結果接於下方。
- 最終三模式重跑通過。HD 與缺圖各 150,000,000 步完整原版 RAM、trace、畫面、PCM 符合既有基準；普通 X11 點擊只在收起列後傳回原版，縮放視窗仍開始新遊戲。功能列下方逐像素一致，所有設定 Proof 不變，按住收起手勢期間無 CPU 步進與滑鼠轉送。具體雜湊見 RESEARCH-LOG 與驗收總表。
- 收尾：整合輸入雜湊、Python／shell 語法、文件相對連結、索引正對照、UID/GID 與 root-owned／異常 `.md` 目錄檢查通過。原版與所有執行產物維持 Git 排除；原有 CDDA 及完整無頭收據保留。無本專案執行中或已停止容器，未建新 image。外部 dosgolem 與參考專案全程唯讀，沒有修改使用者的既有差異。
- 字型授權全文保留上游的行末空白與末尾空行，SHA-256 與來源相同；Git 空白檢查只排除這份原文，其餘變更照常檢查。
- 程式、測試、文件與自由授權字型共 16 份提交 `3372a33` 已推送 private main。[桌面 #3 進度留言](https://github.com/wicanr2/tokugawa-dosgolem/issues/3#issuecomment-6139553523) 已送出並讀回核對，Issue 保持開啟。原版、HD 圖、截圖、RAM、PCM 與二進位均未上傳；沒有宣稱四語、桌面音訊或完整 HD 素材包完成。

## 2026-10-13：劇情捲軸 HD

- 使用者確認頂端功能列後，另指定捲軸介面也要 HD，並持續授權開工。沿用 B 戰國手繪風、原版玩法、桌面與缺圖備援；路由命中 GUI 還原、素材考古與文件職責，使用 RE 證據／READY 閘門及 imagegen 技能。
- 首次來源探針因外部 dosgolem HEAD 前進而在版本檢查停止，沒有執行原版或留下輸出。外部四個修正基準檔雜湊仍相同；在 Docker 建立 a9714eb 的獨立 clone，後續全部唯讀使用，沒有回退外部工作樹或放寬檢查。操作入口已同步固定來源。
- 使用 30 筆普通輸入前綴，在 136769993 步擷取 VISUAL.OZM 的完整原版四平面，全部 SHA 符合載入觀測。排除肖像及文字區後，149860 個背景像素與色號一致。新增背景／捲軸來源擷取與生成素材準備工具，沒有猜獨立 OZM 解碼器。
- 內建 imagegen 製作手繪紙張、綠織物捲邊、金色外框；完整生成圖、提示詞、裁切與 provenance 均保存於既有 workplace。只取下方捲軸區，生成上方紅框未接入。原版文字以前景像素保留，沒有生成字、改翻頁或改點選位置。
- 依 READY 契約監看 VISUAL 載入／原版全畫面複製的自然返回；其他全畫面複製即撤銷資格。捲軸文字安全區以外有任何背景變化就回原圖。F6 及功能列同時切換肖像與捲軸，各自缺圖備援；一項有效時即可啟用 HD。
- 四套件與新來源／前景／範圍／生命週期測試通過。真實視窗三種捲軸模式及功能列整合均通過，兩頁原文保留 24324／4036 個呈現前景像素。四次 150M 原版 RAM、trace、畫面與 CD PCM 符合既有基準；原有 CDDA 與完整 750M 收據未變。資料及各次雜湊見 RESEARCH-LOG 與驗收總表。
- 收尾：整合工具與美術 provenance 雜湊、Python／shell 語法、文件連結及索引正對照、UID/GID、root-owned／異常 `.md` 目錄及 Git 差異檢查通過。所有本專案容器已結束，未建新 image。外部 dosgolem 仍保留原先的 `cmd/probe/main.go` 差異；固定 clone 為乾淨 detached HEAD。原版、HD 美術、提示詞、RAM、PCM、截圖及二進位皆被 Git 排除。
- 自製程式、測試與文件共 19 份提交 `4691f6a` 已推送 private main。[HD #5 進度留言](https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6145463978) 已送出並讀回核對，Issue 保持開啟。儲存庫再次確認為 private，沒有上傳原版與衍生圖；目前素材路徑、完整提示詞及使用內建 imagegen 的來源記錄均已保存本機。


## 2026-10-14 全遊戲 HD 盤點與頭像接入

- 承接全遊戲美術 HD、完成後打包三平台的既有授權。未建立 remake、未開始正式封包，也未發布 Release。
- 新增載入返回擷取、107 個 OZM 的獨立解碼、17 個正常來源對照、81 張 KAO 頭像來源與共同繪圖端。保留五個尾端缺 byte 的來源為未知。
- 接入 A-0 至 A-7 與 81 張頭像的唯讀呈現；生成並校正九頁頭像圖集，製作 36 張劇情圖草稿。來源、提示詞、衍生圖與執行產物均只留 workplace。
- 四套件通過。HD 與缺圖各完成 750M 普通玩家路徑，八張劇情及四張頭像的真實視窗原圖／HD／恢復、矩形範圍、地名保留與原版狀態核對通過。收據、限制與雜湊見 RESEARCH-LOG。
- 驗證工具修正：啟動 null 狀態、輸出來源未完整複製、按鍵等待實際回應、確認框遮擋時的擷取時點，以及工作階段中斷。GUI image 的模組快取沿用 /go/pkg/mod。長流程使用 detached Docker，容器內保留期限；未把環境失敗列成產品缺陷。
- CONTEXT、WORKLIST、README 已同步實際範圍。唯一素材覆蓋清單由 hd-art-inventory.py 重生；準備 VISUAL 上方框架 v2，尚未接入。其他介面、地形、單位、動畫、結局、桌面播放與可攜封包入口仍待完成。
- 已結束本輪驗證容器，未建新 image。提交前核對專案擁有權、Git 排除範圍及 Docker 清理狀態。


### 上方框架完成與生成圖片保存

- 53a34d5 已推送 main，並更新 Issue #5：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6161317076。
- VISUAL 紅金框架已完成四種真實視窗驗證及 150M 原版狀態核對。共用視窗九圖塊的來源與 4864 個原生像素已核對；其繪製觀測與呈現接續處理。
- 使用者新增明確授權，要求將生成圖片保存於私人 GitHub repo。建立 assets/hd-b 作可版控的素材來源；既有 workplace 用於本機研究且排除 Git，tools 保存程式，兩者不承擔圖像來源歸檔。
- 保存 139 張 PNG、54 份完整提示詞、頭像裁切紀錄與索引；逐檔 SHA-256、大小及清單核對通過。GitHub repo 再確認為 private。原版資料與擷取原圖未加入來源清單。
- 新的保存工具及入口已掛入 AGENTS、README，CONTEXT 與 WORKLIST 同步新授權及框架狀態。正式全遊戲 HD 與三平台打包仍在進行。
- 提交 `2117da5e28faa41a6300e73342d5cb31713321cc` 已推送 private main。GitHub API 讀回的 196 個素材檔與本機逐檔 Git blob 雜湊、大小及完整清單相同，包含 139 張 PNG、54 份完整提示詞。遠端核對收據位於 `workplace/github-setup/hd-art-remote-verification.json`；素材入口為 https://github.com/wicanr2/tokugawa-dosgolem/tree/main/assets/hd-b 。
- 保存工具的 `--verify-only` 再次通過，原版資料與本機研究輸出維持排除。Docker 無本專案殘留容器，未建新 image。
- 已更新 [HD Issue #5](https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6163315211)，記錄私人素材歸檔、上方框架驗證及尚未完成的範圍，Issue 保持開啟。

## 2026-10-14：共用視窗九圖塊 HD

- 承接使用者的「go」，沿用戰國手繪風、原版 DOS 規則及全美術完成後三平台打包。路由命中資產考古、READY 規格閘門、桌面試玩、README 與文件職責，載入對應入口。沒有新增產品分支或 remake 引擎。
- 先審查 WINDOW.OZM 四平面、九圖塊、原版堆疊參數與保存／還原的 IDA 證據，將契約記為 READY，再接入唯讀觀測與呈現。沿用已存 private repo 的視窗圖，沒有新增生成圖或變更玩家操作。
- 四套件及五種 GUI 驗證通過，每次 150M 普通玩家路徑的 15 個原版欄位、完整 RAM、原生畫面及 PCM 均符合基準。實際視窗核對角色選擇、確認框開關、能力欄、缺圖、頭像共同呈現及只有視窗素材時的頂端列。詳細收據與雜湊見 RESEARCH-LOG，總表由 verify-hd-window.py 重生。
- 頂端列驗證第一次缺輸出子目錄而提前停止，保存原始失敗輸出後修正驗證工具的目錄準備，在同一 image 及命令乾淨重跑通過。未把環境失敗記成產品缺陷。
- 素材覆蓋及 Git 歸檔索引同步視窗的正常路徑驗證狀態；原圖、記憶體、PCM 與 GUI 截圖留在 workplace。全遊戲其他美術、桌面音訊、一般鍵盤與三平台封包仍待完成。收尾與遠端結果接於下方。
- 收尾：來源 ZIP CRC、12 檔雜湊、唯讀及 UID/GID 通過；五次收據的整合來源與目前程式相同。Python 語法、Go 格式、文件連結與索引正對照、素材歸檔完整性、擁有權及 Git 差異檢查通過。文件正對照首輪使用錯誤的連結標題，改用 README 的既有實際標題後核對通過。無本專案殘留容器，未建新 image。
- 自製程式、測試、文件及素材索引共 19 檔提交 `2c400d619d7ef0f87f1785566fcd46c32cb570e9`，已推送 private main 並讀回遠端 commit 核對。[HD Issue #5 進度](https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6164229850) 已登記，Issue 保持開啟。原版及研究輸出未提交；外部 dosgolem 仍只有原先的 `cmd/probe/main.go` 差異，固定副本乾淨。

## 2026-10-14：劇情來源擴充與後續草稿

- 承接使用者的「繼續」，路由命中資產考古、規格閘門及文件職責。沿用既有證據技能與 imagegen，原版 DOS 規則、存檔、手繪風、桌面及全 HD 後三平台打包決定保持有效。
- 審查全部 42 張既有劇情草稿、圖片文字及精確重複來源。IDA 原始指令證實 328 寬來源只顯示左側 320；先記 READY，再產生 53 個來源對照、原圖與字牌矩形。保留未完整解碼來源為未知。
- 內建 imagegen 製作 D-2 裁切構圖 v2、E-1 去星點 v2，兩張加入來源對照。另製作 18 張城郭、軍議與結局草稿，GUNGI 再以 v2 恢復背景金像；修正前版本均保留。圖片、完整提示詞與生成紀錄依既有授權保存於 assets/hd-b。
- 四套件及 53 個來源的資料／呈現測試通過。修正診斷陣列滿額影響 HD 的缺陷，新增滿額後更新與撤銷測試。離線預覽入口同步新增對照表依賴，獨立測試通過。
- HD 與缺圖各完成 750M、390 筆普通輸入，每側 19 次切換；八張劇情及四張頭像的真實視窗、地名、完整 RAM、15 個原版欄位與 PCM 核對通過。45 個已接來源未逐張正常實玩；新 18 張及其 22 個來源尚未接入。總表及雜湊見 RESEARCH-LOG。
- 素材覆蓋與私人歸檔索引區分製作、共同繪圖端接入及正常實玩；正式素材包格式與打包仍待後續。收尾與遠端結果接於下方。

## 2026-10-14 53 來源保存與後續接入

提交 `485deef` 已推送 private main。主機 gh 重核 private，遠端 tree 的 241 個素材檔逐一符合 Git blob 雜湊。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6165978965 。語法、索引正對照、圖片保存與擁有權檢查通過；本專案無殘留容器。使用者再送 go，繼續接入 22 個來源，保留既有 53 來源收據於 catalog/story。

75 來源準備及四套件通過，leprobe 測試 26.525s。新輸出 story-expanded 保留舊 story。HD／缺圖 GUI 各以有界 800s 容器執行，增加軍議圖抽樣；結果未完成前不提高正常路徑驗證數。

同步查下一批地形來源。首次以程式區段差值換算資料字串，五個 EA 無交叉參照；改由 IDA get_fileregion_ea 核對，資料段差值為 0x5800，程式段原有差值 0x4800 不可混用。資源字串均由資料指標表引用，尚未據此推定圖塊幾何。正式 i64 保持唯讀，分析副本在容器 /tmp。

75 來源的 HD／缺圖 GUI 均通過，各 750M、390 筆普通輸入、九張劇情及四張頭像、21 次不改原版狀態的切換。總表 SHA-256 `6f64f6dde7e94f21f1782c5388c073318a8020d4a03b330c8d391baabf229caf`。GUNGI v2 已目視核對實際遊戲內構圖。新增地形擷取工具與索引入口，正常 RAM、來源平面及 IDA 讀取端核對通過，輸出 143 個原版來源格，包含空白。未製作地形 HD，未開始正式打包。

本批最後重核 75 來源、九張正常劇情、兩次 750M GUI 總表及 160 PNG／75 提示詞通過。地形擷取重跑核對 143 格、逐檔 PNG 與索引，補齊 Python 版本；初版收據按雜湊保存，原始 PNG 不覆寫。語法、文件索引正對照、Git whitespace、UID/GID 及 root-owned／誤建 .md 目錄檢查通過。所有 tokugawa 容器已自動刪除，沒有重建 image。遠端再次確認 private，接續提交本批程式、索引及文件。

提交 `c094c55` 已推送私人 main，主機 gh 核對遠端 SHA 一致。Issue #5 維持 OPEN，進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6166351414 。本輪新增 21 張 PNG 與提示詞已在前一提交 `485deef` 保存且核對遠端；本提交更新 75 來源與九張正常驗證的素材狀態。原版與本機研究產物均未加入 Git。最後容器清理及擁有權檢查通過。

## 2026-10-14 戰場地面 HD 接入

新增唯讀地面觀測與界限測試，四套件及正常 750M 基準核對通過。確認早期 MAP_CHIP 與正常戰場 DAC 的差異，使用實際 A-0 調色盤逐點核對 50 個完整地面矩形，沒有猜測亮度偏移。地面呈現契約已 READY。另生成十張單位草稿及八張地面草稿；floor-006／022 誤生水面，正另製 v2，舊稿保留。單位 002／007 曾被目視誤判為弓兵；EXE 原始文字有武士、騎兵、步兵、步槍，與來源索引映射尚未核對，維持 source-index 草稿，不據口頭判讀修改兵種。

地面切片實作與四套件通過。準備格式化命令首次 shell 引號錯誤，未啟動工作負載；改用具名 heredoc 後成功，未改產品行為來處理腳本錯誤。兩次真實視窗驗證各完成 750M／390 普通輸入，總表 SHA-256 `3902facc85f45e23495935031addeb8d76b27fbdd2b0c4cf793637509f585fdd`。50 矩形、六種來源使用 HD，遮擋整格保留原圖；已目視核對此階段的原版／HD 混合畫面，物件與單位尚待接入。

本批保存新增 20 張生成素材及三張總覽、20 份提示詞，含兩張水面誤生的修正前草稿。素材總量 183 PNG／95 提示詞，透明單位 alpha 0..255 原樣保存。PNG、提示詞、生成記錄與索引已核對；GitHub 推送與最後清理接續記錄。

提交 `5462fd4` 已推送私人 main，主機 gh 核對遠端 SHA 相同。遠端 287 個圖片、提示詞與索引檔逐項符合本機 Git blob SHA，沒有 original/ 或 workplace/ 資料。Issue #5 維持 OPEN，進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6167356430 。最後語法、來源綁定、文件索引正對照、Git whitespace、UID/GID、root-owned／誤建 .md 目錄檢查通過；所有 tokugawa 容器均已刪除，沿用原 image，沒有留下持續程序。後續依 WORKLIST 的物件／單位自然繪製切片進行。

## 2026-10-14 透明前景與地面合成

- 命中復古遊戲規格驗收路由，載入 reverse-engineer-retro-game-remake 的 evidence-and-re、spec-gated-workflow、verification-and-handoff；新研究輸出沿用 workplace，依文件職責新增 terrain/objects 並掛入 AGENTS。主機 gh auth status 成功，遠端確認 private，Issue #5 OPEN。
- 有界唯讀診斷完整跑 750M，134 次地面與 11 次透明圖塊返回、15 欄基準、完整 RAM、原生畫面與 PCM 均符合。準備 39 個實際來源遮罩；讀錯既有 group 的單數名稱已修正。證據審查後升 READY，才實作順序、自然返回、上限、遮罩及整格來源核對。
- 四套件通過，leprobe 26.413s。HD／缺圖 GUI 各自然跑完 750M／390 筆普通輸入，各三次開關不改原版。真實視窗六種地面、65 矩形、7067 個原生前景像素通過，缺圖零變動。離線 64 格與桌面 65 格的差異以原始索引及實際游標 AND／XOR 獨立核對。總表 SHA-256 78591ff9cf7a2d4e1682581cd2afe5950b8d514694cd8d3d1dd0e9ba6ddb78a9。
- 使用內建 image_gen.imagegen 製作樹木、石塊、藍／紅城郭四張透明草稿，檢視後保存原始 PNG、alpha、提示詞及生成記錄，未接入遊戲。保存腳本原先假設 1280 尺寸，檢查實際 1254×1254 後改核對尺寸界限，不改生成 bytes。本輪素材總量 188 PNG／99 提示詞。
- 舊 terrain/trace、terrain/runtime 與劇情收據保留原版本；新呈現與原版資料、存檔、規則隔離。地面重疊留下的碎片、HD 物件／單位、其他美術與正式三平台封包仍待後續。下一切片為 objects-000 樹木的透明 HD 呈現。最後素材、文件索引、擁有權、容器與遠端核對接續記錄。

提交 `888be53953cb4f1225650cb941c5f4a6fa7a30e7` 已推送 private main，主機 gh 重核遠端 SHA 一致。297 個圖片、提示詞與索引檔逐項符合遠端 Git blob；original/、workplace/ 未入遠端。Issue #5 保持 OPEN，進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6168468573 。素材核對為 188 PNG／99 提示詞、296187457 indexed bytes。語法、文件索引正對照、完整來源綁定、Git whitespace、UID/GID 與 root-owned／誤建 .md 目錄檢查通過，所有 tokugawa 容器已自動刪除，沿用原 image。

## 2026-10-14 樹叢透明 HD 接入

- 使用者接續 go，沿用已載入的規格驗收與文件職責入口，新增 terrain/trees 並掛入 AGENTS。原版九次樹叢繪製中，五項已被地面重畫，三項完整背景及一項介面邊緣已核對。完整背景重建與實際游標合成符合原版 PNG。
- 放大原圖後修正單棵樹判讀：原版有兩根分開的主要下方樹幹及中間較高的短棕色區段。內建 image_gen.imagegen 製作 v2，原圖／草稿／素材對齊預覽已檢視；v1 保留為誤判草稿。明示裁切與來源範圍、原始 PNG／alpha／完整提示詞及修正記錄保存，沒有覆寫生成原圖。
- READY 後實作完整背景核對、透明合成、原版輪廓還原及其他可見前景保留。四套件通過，leprobe 25.378s。三種 GUI 各自然 750M、390 筆普通輸入、三次原版狀態不變切換；15 欄、完整 RAM、原生畫面、trace 與 PCM 符合基準。三處樹叢 HD、82 原版人物像素及 1394 個舊輪廓背景還原通過，介面邊緣備援；目視核對搭配地面與只有樹叢兩種實際畫面。
- 獨立總表 SHA-256 42e02fd79ed8dbb9393a2f9418adf216875d2df314a7921ec939c55b1fe08455。舊 objects 收據用當次來源快照唯讀重核通過，總表不改。覆蓋清單區分一種物件、三個可見實例與其他草稿，不增加整個圖集完成率。生成美術保存核對為 189 PNG／100 提示詞，298937061 indexed bytes。
- 其餘物件、單位、地面碎片、其他美術與正式三平台打包仍待後續。下一切片為介面邊緣的可驗證部分背景與裁切。最後擁有權、容器、私人遠端與 Issue 核對接續記錄。

提交 `bc44bef6c8361eda09ce8d71d35bbdd46d6dcc93` 已推送私人 main，主機 gh 核對遠端 SHA 一致。300 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 保持 OPEN，進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6169579498 。最後語法、文件索引正對照、當次及歷史收據 SHA、素材索引、UID/GID、root-owned／誤建 .md 目錄與 Git whitespace 檢查通過；所有 tokugawa 容器均已自動刪除，沒有重建 image 或遺留背景程序。

## 2026-10-14 原版繪圖前背景與介面邊緣樹叢

- 沿用規格驗收、逆向與文件職責入口。確認右側樹叢的 480 個像素缺地面來源，287 個仍有原版樹叢；只裁切會留輪廓。靜態 MAPGAMEN 補底仍差 14 像素，沒有猜補。
- 有界唯讀擷取十一個原版圖塊的入口背景、原始四平面及自然返回；獨立核對來源 PNG、索引零遮罩、十五欄與完整 RAM，全部通過，總表 d131450b3925c415736c83584cceb089704207d4a452c4acc9b6929051261a0b。
- READY 後接入桌面背景資格；未知背景、未知覆蓋及不符返回仍備援，介面原版透明處不允許 HD 擴張。四套件通過，研究入口 24.322s。三模式各自然 750M／390 事件、三次切換、完整 RAM、十五欄、原生畫面與 PCM 符合基準；四處樹叢、82 原版人物像素、772 介面像素與 1926 舊輪廓背景還原通過，缺圖零變動。總表 911938c881bada744c90831b3167eb6904f2cfd6cd606e29fd5371dc55e3f46b，舊三處收據保留。
- 放大十種單位原圖與草稿，units-001／003 姿勢及幾何已比較；units-006 v1 刀勢有誤，內建 image_gen.imagegen 製作 v2，原始 alpha、提示詞及來源保存。人物 HD 尚未接入。
- 保存工具及索引核對 190 PNG／101 提示詞，300172326 indexed bytes。遠端仍 private，Issue #5 OPEN。後續接人物與樹叢重疊合成及地面碎片，再補其他全遊戲美術。

提交 `6c2be518016e254f67c2fc9b6bf300cffa442c66` 已推送私人 main，主機 gh 核對 SHA 一致。303 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6170765470 。當次與歷史收據、來源快照、素材索引、語法、文件入口、Git whitespace 及擁有權檢查通過；所有 tokugawa 容器均已自動刪除，沿用原 image。

## 2026-10-14 地面完整順序與重疊人物 HD

- 沿用規格驗收入口，從正常路徑 145 個自然返回重建原版 134 地面與 11 前景。67 可見地面及六前景的完整矩形符合原版，研究公式單列排除原版游標；兩格各 1440 原生像素的地面碎片可替換，四項介面覆寫的模型殘片保持備援。
- READY 後保存有界完整地面 journal，原 65 格觀測不變；共同畫布合成所有合格 HD 層。原版最後可見的前景保有優先權，自己的 HD 只混合一次，透明處能顯示其他 HD 前景與正確背景。units-001／003 的原圖姿勢、來源範圍、裁切及 alpha 已核對並接入。
- 四套件通過，研究入口 27.346s。三模式各自然 750M／390 事件，完整 RAM、十五欄、原生畫面、trace、PCM 與基準相符，三次切換不改原版，真實 X11 與呈現圖完整吻合。正常 67 地面、四樹叢、兩人物，413696 呈現像素與獨立公式相符；重疊 82 原生人物像素、1620 介面像素及兩人物 1584／1574 舊輪廓還原通過，全部缺圖零變動。總表 5af50794c64a7442aee41bc88609a314ea0755de81cd1b8ec96049ff1a4946ce。
- 內建 image_gen.imagegen 製作 MAPGAMEN 紅金戰場介面草稿，來源及既有手繪風格參考、1586×992 原始圖、完整提示詞與 SHA 保存。此草稿尚未接入。私人素材保存核對 191 PNG／102 提示詞，302088815 indexed bytes，units-001／003 索引改為正常路徑已驗證。
- 全遊戲 HD 仍未完成。接續戰場框架、其他物件／單位及未製作地面、美術與動畫；正式三平台打包尚未開始。

提交 `cfe75a136a10ce6c552c11e2b73d459a04cead7f` 已推送私人 main，主機 gh 核對 SHA 一致。306 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6171788132 。當次與歷史收據、來源快照、素材索引、語法、文件入口、Git whitespace、UID/GID 及 root-owned／誤建 .md 目錄檢查通過；所有 tokugawa 容器均已自動刪除，沒有重建 image。

## 2026-10-15 戰場紅金框架 HD

- 沿用規格驗收與文件入口，先核對原版 MAPGAMEN 全畫面複製。657204327 步進入 sub_1C084，657204466 自然返回；256000 索引與來源逐 byte 相同，當時 DAC 解出的 PNG 與索引逐點一致，完整 750M 原版狀態及 RAM 符合基準。來源總表 a1ba0e3ded13622c59f3b4e5fb81ff474cf1fe55175dc1da72646f54dc38f926。
- 手繪框架原圖不改，v1 幾何試作保留。檢視後修正兩側凸耳邊界，v2 skin SHA 3de79e3eebe01aa970e45b8bf377b6e381bec14c9b4d9f569994aaecbe78e435，幾何總表 9476514d407923222ee06346da4869153279795b7a55293a76c7287e71bd33f9。原版天氣圖與 `(464,291)-(608,365)` 完整相符；動態區及徽記採整區資格，文字及未知標記依原版索引差異保留。
- READY 後接入固定十六色與來源遮罩、同索引人物保護、場景影響區及原版繪製前索引底圖。單元測試涵蓋來源、透明／不透明索引、未知 palette／淡出、整區資格、缺圖、原始輸入不變及框架與其他 HD 層的合成。第一個不透明前景測試選到透明索引 0，修正測試樣本後重跑，沒有修改產品規則。
- 首輪 GUI 逐點公式已通過，但腳本誤把正常能力／確認視窗當作裸露徽記，以及切換次數仍按單畫面計數；attempt-01 原樣保留。修正驗證器後同一 binary／image／普通輸入乾淨重跑三模式，各自然 750M／390 事件，完整 RAM、十五欄、原生圖與 PCM 符合基準，五次切換 Proof 相同，六張真實 X11 圖與呈現 PNG 完整相符。
- 框架＋共同場景改變 761107 呈現像素，框架 347932 與場景 413696 像素符合獨立公式；1564 個透明處像素用 HD 框架底圖，56 個未知介面像素保留。只有框架保留原版人物；缺框架時原有 HD 場景保持 412342 像素變動。總表 943c5bb05efbfb009cb282db0c052d8f146379094f3924ed83d7315ae864b66e，共用 binary a3b822fc53d4ed9948465eaf4ca28dd0cb37c7efd688ef645d05204d6e248c3b。
- 193 PNG／104 提示詞與 305012173 indexed bytes 保存核對通過。純生成 skin 及完整提示詞、幾何 metadata、v1 工具文字快照加入素材索引；原版 mask、idx、native 圖與含原版預覽不加入 Git。素材工具保持原有檔案類型白名單，以文字快照保存舊腳本。原始生成圖及各歷史收據不改。
- 全遊戲 HD 尚未完成；下一切片為天氣、MAP_WAKU 等其他戰場圖像與共同繪圖端，其後補剩餘 OZM、地面與非空物件／單位。正式三平台打包尚未開始。

提交 `b6bcfa0010dcb5ff7f5147266d0a63012013c751` 已推送私人 main，主機 gh 核對 SHA 一致。313 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6173478172 。當次及歷史收據、來源快照、素材索引、語法、文件入口、Git whitespace、UID/GID、root-owned／誤建 .md 目錄檢查通過；所有 tokugawa 容器均已自動刪除，沿用原 image，沒有遺留背景程序。

## 2026-10-15 共用 OZM 繪圖與天氣 HD

- 命中規格驗收與文件職責入口；沿用原版整合及戰國手繪風。先從既有覆蓋表擷取 28 個未製作且可解碼的 OZM 參考，五個未完整解碼來源不填補，原圖與總覽維持本機。
- 依 IDA Pro 9.4 的 sub_1C1BC／sub_1C26D 原始定位確認來源 offset、byte columns、rows、目的 pitch 與位址。新診斷核對實際 RAM 四平面，避免把最後載入檔名當作快取來源。四套件及自然 750M／390 事件通過，16 次繪圖均與靜態來源矩形逐索引相符，完整 RAM、十五欄及 PCM 符合基準。來源總表 7abe3e4f463ffb001c5024bf89111b7209c643c818027c1d6ccdd509be9cdc69。
- 內建 image_gen.imagegen 製作 WEATHER1／2；首版把原圖岩峰畫成松樹，依原圖修正 v2，兩版與完整提示詞皆保留。READY 後接入共用不透明繪圖端、來源裁切及 palette／整區資格、全畫面清除與未知交疊撤銷。正式規則、資料及存檔維持原版。
- 固定 GUI image 四套件通過，研究入口 19.915s。晴天及缺圖兩模式各自然 750M／390 事件、三次原版狀態不變切換，完整 RAM、十五欄、原生圖及 CD PCM 符合基準；六張真實 X11 圖與呈現 PNG 完整一致。天氣 42624 呈現像素逐點符合手繪取樣，缺圖同區零變動，天氣區外的框架、67 地面、樹叢與人物 HD 完全相同。總表 743d43ac25ffe6d318daa13f1f307d66c861a2584d8c5ff6656a024c780f593e，共用 binary c8d83595c1f593de60b180e5b1dfaa1e565626cb6f9ecb67579619dbfd333314。陰天為共同繪圖端及來源測試通過，正常路徑未抽測。
- 另製作 OP2 騎兵與 OP8 五武將開場草稿。保留原始 2172×724／2171×724 與提示詞；原版 640×200 幾何及 OP2 未使用藍底區待校準，不加入 production profile。覆蓋表現為 79 個 OZM 已準備、兩個部分完成及兩個新開場草稿，全遊戲 HD 仍未完成，未開始三平台打包。
- 私人素材保存與 verify-only 核對 199 PNG、110 提示詞、319872400 indexed bytes，共 327 個素材檔。語法、文件入口、目前與來源快照雜湊、Git whitespace、UID/GID 及 root-owned／誤建 .md 目錄檢查通過。來源探針、建置、兩個 GUI 及收尾容器均已結束並自動刪除，沿用既有 image。

提交 `af96706f99add0196e96e262accce17d1e34f104` 已推送私人 main，主機 gh 核對 SHA 一致。327 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6174208391 。所有 tokugawa 容器已自動刪除；下一步依目前覆蓋表處理其餘戰場美術、開場草稿及圖集，不重開已驗收的天氣切片。

## 2026-10-15 全部地面來源的 HD 素材

- 上輪為已驗證進展，接續唯一狀態表，命中規格驗收與文件職責入口。本輪補足 MAP_CHIP 尚缺的 96 個地面來源，沒有改原版地形規則、座標、輸入或存檔。
- 以內建 image_gen.imagegen 製作紅褐土路、深藍及淺藍水面材質，各 1536×1024；沿已審查草地畫作及每張原圖自己的美術邊界合成。104 來源共有三組完全相同的索引，因此新增 93 張 384×256 圖；原有八張不改，合計 101 張獨立圖覆蓋全部 104 來源。
- 七頁第一版總覽顯示水面的孤立色點產生細碎綠色小島。保留第一版，第二版以明示四像素連通區規則處理材質紋理，再逐項檢視全部 104 組原圖／HD。這是美術近似，不當作原版地形語意。兩版草稿、原始生成材質、完整提示詞、原始工具 SHA 對應的 v1 文字快照與來源／幾何 metadata 均保留。
- 草稿工具曾誤用輸出父目錄讀取原始索引，及把 21 組道路／河道都當成同形；原始 bytes 查證只有前 12 對相同，其餘保持獨立。保存工具的重複 owner metadata 檢查也曾中止；我在保存完成前啟動的建置因素材尚缺而失敗。失敗 log 保留，修正工具依賴後用同一 image／命令乾淨重跑。覆蓋表另明確區分歷史八項與目前 104 項契約。
- READY 後只更新來源固定 Asset／AssetSHA。104 來源的共同呈現、缺圖、未知遮擋、原始輸入不變及矩形外保護測試通過；四套件通過，研究入口 25.741s。正常 GUI 自然完成 750M／390 事件，十五欄、完整 RAM、原生畫面、trace 及 CD PCM 符合基準，三次切換 Proof 相同。三張真實 X11 與呈現 PNG 完整相符，原圖與 HD 又逐點符合前次天氣版本；收據中全部 104 地面素材 ready。正常首戰仍只抽測六個來源、67 處可見區，其餘 98 來源未逐項正常抽測。
- 唯一總表 7c3c0a8ccb899015ce3080fe91e5c0b56b7cad88811a2f6074114c384f5eca9d；來源對照 06828ec98b7011a925a72031f9623c6e82170474f782cd975f5acb87ff06bbd3；binary 9d4ebacda3e42dc2a36e3681e409c088dd86c069c5cce90892348970031e9d13。素材保存目前為 390 PNG、113 提示詞、373567657 indexed bytes；原版原圖、遮罩與執行收據留本機。
- 全遊戲 HD 仍未完成，接續非空物件／單位、其他 OZM、開場與介面、動畫；全美術完成前不開始三平台打包。各製作、建置與 GUI 容器已結束，沒有重建 image。

提交 `2f5e9a7172d3d88bd0ba4d40d03094bf3db0a4fc` 已推送私人 main，主機 gh 核對 SHA 一致。527 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6174942347 。保存索引、來源快照、Python 語法、文件入口、Git whitespace、UID/GID、root-owned／誤建 .md 目錄檢查通過；所有 tokugawa 容器已自動刪除，沒有殘留背景程序。

## 2026-10-15 完整非空單位圖集 HD

- 上輪為已驗證進展，沿唯一狀態表與規格驗收入口接續剩餘單位。13 個原始索引的十個非空來源及三個空白分開，依原始索引零 alpha 與已核對 SourceBounds 準備；不推定機械兵種映射，不改遊戲規則、資料或存檔。
- 內建 image_gen.imagegen 修正 002／007 的抬腿姿勢 v2，首輪十圖對照另發現 008 單尖槍頭與原版分叉輪廓不同，再作局部 v2。第一次裁切、對照、輸入與腳本原樣保存於 units/attempt-01。最後兩頁十組原圖／HD 已逐項檢視；006 使用既有刀勢 v2，001／003 的素材 SHA、裁切及準備後 PNG 保持相同。
- READY 後擴充固定來源對照為十項。每張實際來源通過來源／HD SHA、原始 alpha、裁切、共同場景、透明輪廓、逐項缺圖、未知覆蓋、原始輸入及矩形外保護測試；三個空白索引沒有 HD。固定 GUI image 四套件通過，研究入口 24.077s，沒有新增原版攔截或正式玩法分支。
- 自然原版入口至首戰 750M／390 事件，完整 RAM、十五欄、原生畫面、trace 及 CD PCM 符合基準，三次切換 Proof 相同。三張真實 X11 與呈現 PNG 完整吻合，原圖、HD 及還原後完整畫面又逐點等於前次地面版本。十項單位載入皆 ready；正常首戰仍只看見 001／003，其餘八項只列來源與共同場景通過。
- 唯一總表 a417f849eec3b4f2715c93da3aea429690451f9e9625810b2e4d7dd86f7d7bd3；來源對照 3a2edcd1567a56ab1d6573356a376ed22ce7238deafab9b7133fe0c6546d2e2e；binary b3d881260858e4058ef2c2d726696b2b4ba87f7e3dc70078b796f203569ec6b2。原始生成圖、全部修正版、兩次透明裁切、純 HD 總覽及完整提示詞保存；原圖對照及執行收據維持本機。
- 素材庫目前 415 PNG、116 提示詞、377450824 indexed bytes。全遊戲 HD 仍未完成，接續其餘 23 非空物件、其他 OZM、介面與動畫；正式三平台打包尚未開始。製作、建置與 GUI 容器均已結束，沿用既有 image。

提交 `87ab034b0bf0d91274c78d47802e8ae1dc7193c5` 已推送私人 main，主機 gh 核對 SHA 一致。563 個素材檔逐項符合遠端 Git blob，original/、workplace/ 未加入遠端。Issue #5 進度留言：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6175780323 。保存索引、來源快照、Python 語法、文件入口、Git whitespace、UID/GID、root-owned／誤建 .md 目錄檢查通過；所有 tokugawa 容器已自動刪除，沒有殘留背景程序。


## 2026-10-15：獨立物件 HD 與城牆草稿保存

- 依規格驗收與文件職責路由接續物件圖集。內建 imagegen 產出 20 張手繪物件並保存完整提示詞；24 項非空物件完成裁切及四頁原圖對照檢視，24、25 空白保留。
- 核准 0、1、2、3、18、19、22、23，共八項。桌面載入及頂端 HD 可用性接上逐物件 profile，保留舊樹叢資產旗標；ObjectHD 計所有物件，TreeHD 只計 0。16 項城牆的屋簷與分層未對齊，留草稿，不接入正式呈現。
- 四套件通過，研究入口 28.641 秒；新版本正常 750M／390 事件、完整 RAM、十五欄及三次切換通過。三張 X11 擷取與呈現 PNG、前次單位版均完整一致。新增七項物件僅共同場景測試，未逐項正常抽測。
- 獨立總表：terrain/objects/expanded/verification-summary.json，SHA-256 cf75a9f0b21a450b781fb8a0f30ebe3dd41612af08e1d56cf76c5d8d33463f73。來源、審查、二進位與 GUI 雜湊由總表串接。
- 保存工具對有限取樣 alpha=254 的分類已修正，沒有改 PNG alpha；重跑保存與 verify-only 通過，460 PNG、136 提示詞、407334935 索引 bytes。原版來源與執行產物維持本機。
- 覆蓋清單重生：104 地面、十個非空單位、八項物件已接入，16 物件草稿、其餘 OZM／動畫及五筆解碼未知仍待完成；全遊戲 HD 未完成，三平台打包未開始。
- 建置及 GUI 容器均正常退出並移除，後續附 GitHub 與擁有權核對結果。

GitHub 收尾：功能與素材提交 db97be7f0d17b9f898c2302fb311769b52cf95bd 已推送 main。主機 gh 再次確認 PRIVATE，遠端 main SHA 一致；immutable tree 的 632 個 assets/hd-b 檔案逐項 blob SHA 與本機一致，tree 未截斷，original/、workplace/ 不在遠端。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6176564528 。最後保存核對為 460 PNG、136 提示詞；完整工作根的 root-owned 檔案及 .md 目錄檢查為空，所有本次輸出抽查 UID/GID 1000:1000，tokugawa 容器為空，未重建或清理其他專案 image。


## 2026-10-15：完成城牆校準及完整物件圖集

- 上一輪屬實際進展。本輪命中並載入規格驗收及文件職責入口，接續 16 項牆件。用已保存的手繪橫牆、直牆、城門及斷牆作分區材質，對齊相位、屋簷與石基；source index-zero 輪廓柔化 0.65 原生像素並限制於原版範圍。三頁對照及圍牆／短牆接縫檢視後，第二版進入 READY，第一版保留。
- 24 非空物件全部接入，6／10、12／13 依已確認相同的來源裁切重用；其餘不直接鏡像。既有八項素材、裁切及準備後 PNG 的 SHA 不變，24、25 空白保留。只改既有 Go profile 資料表，沒有新增原版觀測或 gameplay 寫入。
- 四套件通過，研究入口 26.951 秒。新版本 750M／390 事件、完整 RAM、十五欄及三次切換通過，三張真實 X11 擷取與呈現 PNG 及前版完整相同。正常首戰仍只有 objects-000，離線圍牆不作正常路徑證據。
- 驗收總表：terrain/objects/walls/verification-summary.json，SHA-256 04e41b453715467568b098768f93a13a48662a63cd3548548d124ed43fe47a11。右側 atlas padding 的 6400 像素全零；MAP_CHIP 的 104 地面、24 物件及十個非空單位全數已準備，五個空白格保留。
- 內建 imagegen 另製作 BATTLE01／BATTLE02 戰鬥背景及 OP4 徽記三張草稿，完整提示詞、參考圖雜湊及原始生成圖均保存。未校準黑區及 runtime，保持 draft。
- 保存工具新增紀錄時與舊 remaining-generation.json 變數衝突，拒絕覆寫；分開變數後以同一 Docker 命令乾淨重跑通過，舊檔未改。覆蓋工具亦將新背景草稿排除於已審查劇情計數。
- 保存及 verify-only：538 PNG、139 提示詞、423404392 索引 bytes。覆蓋清單：80 OZM prepared、一項 partial、五項 draft、21 項已解碼未製作與五項解碼未知；81 頭像已準備，PCX 待處理。全遊戲 HD 未完成，三平台打包未開始。
- 本輪建置與 GUI 容器均正常結束且移除，遠端與擁有權核對結果接續記錄。

GitHub 收尾：功能、校準圖與新草稿提交 d8a879e495bc1a59ef134e22a39eff4c9a14d59f 已推送 main。主機 gh 確認 PRIVATE、遠端 main 相同；immutable tree 的 721 個 assets/hd-b 檔案逐項 blob SHA 與本機相符，tree 未截斷且沒有 original/ 或 workplace/。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6177319488 。最後保存核對為 538 PNG、139 提示詞；執行收據的 Go source SHA 與目前工作樹相同，完整工作根沒有 root-owned 檔案或 .md 目錄，新增工具、輸出及圖片抽查皆為 UID/GID 1000:1000，tokugawa 容器已清空，沿用既有 image。

## 2026-10-15：六張靜態圖與正常開場標誌

- 命中規格驗收、文件職責及 README 現況路由。校準 BATTLE01／02、OP2／8、OP4 既有生成圖，另以內建 imagegen 製作 OP1 Family Soft 標誌；六張逐項審查，原始生成圖、裁切、完整提示詞及來源 SHA 保存。
- 共用不透明呈現新增 PaintRect 及 VisibleRects。保留原始呼叫、來源／目的矩形及雜湊，後畫矩形只扣除可見區；上限 64 筆與 256 片段，未知直接 VRAM 變化仍備援。明示空 mouse-file 阻止主機輸入混入無輸入基準，普通滑鼠模式不改。
- 舊二進位與匹配來源快照的原版無輸入 200M 只觀測到 OP1／OP5，沒有把其他五張當作正常開場。首次缺 DISPLAY、再次漏建 startup-probe 及 logo 準備掛載錯誤均保留，分類工具環境／準備錯誤；同 image 修正後成功。
- OP1 首次 12M palette 拒絕，原先只判為淡入；原版七次擷取證實標誌有獨立十六色。22M／26.333333M 穩定結果一致，其他五次為淡入／淡出。依來源精確配置 palette，沒有放寬其他來源。最初兩次 GUI 結果原樣保留。
- 首版與 palette 修正版四套件均通過，最終研究入口 27.337 秒。最終 OP1 GUI 的畫面公式與原版狀態已通過，但驗證器把收據 physical_mouse_events 誤讀成狀態檔 mouse_events；修正後在乾淨子目錄重跑同一命令，正式通過。這是驗證器欄位錯誤，非產品滑鼠缺陷。
- 正常標誌 200M、完整 RAM、十五欄、53352 像素公式、四張 X11 與五次切換通過；首戰 750M／390 事件、三次切換、完整 RAM、十五欄及三張 X11 擷取符合舊基準與前版。六項素材全部 ready；正常新來源只列 OP1，另五項來源與共同呈現測試通過。
- 獨立總表 SHA-256：4990b8c4766bc6b83094d16e7199e2cc2639085e193ca310394a747e9e8e6379；二進位 56b7d5441afdca078b8a1073ecf0db5e6016a724284b23f543c14bf1e639f750。生成素材保存及 verify-only 通過，546 PNG、140 提示詞、428970184 索引 bytes；原版 PNG／idx／RAM／PCM 與執行收據維持本機。
- 覆蓋清單重生為 86 OZM prepared、一項 partial、20 項已解碼待製作及五項解碼未知；81 頭像與完整 MAP_CHIP 已準備。全遊戲 HD 未完成，三平台打包尚未開始。下一批為正常標題與其餘開場、動畫及介面。
- 製作、建置及 GUI 容器已退出並自動刪除，未重建 image。遠端、文件入口及擁有權核對結果接續記錄。

GitHub 收尾：功能、六張校準圖及來源／提示詞提交 ee811e061472a10821f9d854650d357760044a3f 已推送私人 main。主機 gh 確認 PRIVATE，遠端 main SHA 一致；immutable tree 的 739 個 assets/hd-b 檔案逐項 blob SHA 與本機一致，tree 未截斷，沒有 original/ 或 workplace/。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6178913960 。來源 ZIP／12 個解包檔、CRC、UID/GID 與工具輸入核對通過；Python 語法、文件連結、入口正對照、目前 Go SHA／GUI 來源快照及 Git whitespace 通過。完整工作根沒有 root-owned 檔案或 .md 目錄，新增輸出抽查 UID/GID 1000:1000，所有 tokugawa 容器已退出並移除，沿用既有 image。

## 2026-10-15：20 項剩餘美術與動畫圖集

- 上輪屬實際進展。本輪核對乾淨 a785894 與唯一目前狀態表，命中規格驗收、文件職責、README 標準，沿用 imagegen 技能；沒有啟動 subagent。
- 以固定解碼資料核對 20 個已解碼缺項。內建 imagegen 共生成 18 張圖，原始圖與完整提示詞每批即保存，另補齊有序輸入圖、用途及 SHA。五筆解碼未知保持未知，沒有以猜測補圖。
- 八個 ANIME 圖集共 48 格，原始索引驗證 16 格精確鏡像，32 組獨立來源中一個全空格；先逐項證明再鏡像，沒有把近似姿態當成相同。另補 OP5 標題、BATTLE／MAP_WAKU、開場、結局、表情圖與兩個 JIMEN。END11B 的完字以外人物固定重用，空格與原版留白保留。
- 第一版準備後發現 OP9 緩衝區 inclusive／exclusive 邊界漏一個實體像素，第二版修正。核對原始索引再確認 BATTLE 的末個非零列為 246，第三版將美術範圍改為 247；前兩版與對應工具全文快照均保留。這些版本未註冊到執行器。
- 三頁 HD 總覽及 v3 OP9 全圖檢視通過；獨立靜態核對 20 圖、48 動畫格、16 鏡像、一個空格、八格天空、三格交鋒、buffer 與來源 SHA 通過，並選定 v3。美術核對 SHA-256：81b62258274df16fbc1881e642ca64e46ec2527bb991c5a6f16009fb5a7a04e7；來源對照 958c4a55724f7b505a2e2f0e164554f24edf819d2ca747bb0ecfa8df617fa839。
- 保存與 verify-only 通過：797 PNG、158 提示詞、578495190 索引 bytes，含原始生成圖、裁切、三版準備與選定圖；原版擷取、字模與執行收據維持本機。唯一覆蓋表為 106 OZM prepared、一項分層處理、五項解碼未知，新增 20 項尚未註冊到執行器。
- 本批沒有改 Go 檔或執行器載入表，因此不重跑相同的 750M 玩家路徑。現行開場及首戰收據的來源 SHA 保持一致；這只維持既有範圍，不用來驗收新增美術。下一步接入正常 OP5，驗證選單遮擋，再處理動畫與其餘 UI；候選字區／多區域未達 READY 不進正式路徑。
- 全遊戲 HD 未完成，三平台打包尚未開始。容器均已退出並自動刪除；遠端、文件及擁有權核對結果接續記錄。

GitHub 收尾：素材與工具提交 781a34483ff59027079c628a988a1726a26f3b28 已推送 main，主機 gh 確認 PRIVATE 與遠端 SHA 相同。涵蓋 1020 份素材檔的遠端 Git 子樹 SHA 5142b3aa11d84fff4d7dac9bc46a71c7ce9d239a 與本機完全一致；配合保存索引的逐檔 SHA-256 核對，確認所有圖片、提示詞及 metadata 相符。tree 未截斷，沒有 original/ 或 workplace/。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6179739762 。797 PNG／158 提示詞的保存核對、18 組完整有序輸入圖、20 項美術及 32 組鏡像等價動畫來源核對通過；Python 語法、文件連結、入口正對照、148 項既有整合來源 SHA 與快照、UID/GID 及 Git whitespace 均通過。完整工作根沒有 root-owned 檔案或 .md 目錄，所有本批容器已退出並移除，沒有重建 image。新增 20 項仍待執行器註冊與正常玩家路徑驗證，目標維持全遊戲 HD。

## 2026-10-15：20 項繪圖端、正常標題與選單平台缺口

- 命中規格驗收、文件職責及平台規格優先入口，沿用單一目前狀態表，沒有啟動 subagent。20 項 v3 美術註冊共同不透明繪圖端，新增多區域與固定來源前景保留；未知直接 VRAM 寫入仍退回原圖。
- 視窗及頭像保存實際返回步數，後畫且有效的矩形才遮住舊背景，原始參數與來源雜湊不改。全部來源、48 動畫格、八格天空、三格交鋒及四個眼部區域走實際呈現測試，含缺圖、原圖固定前景、來源變更、錯誤區域與遮擋界限。
- 首版四套件通過，研究入口 32.120 秒。正常 OP5／缺圖各自然 200M；571560 呈現像素符合 HD 公式，原選單 67584 像素不被標題覆寫，缺圖僅退回標題。三次切換、完整 RAM、十五欄及真實 X11 通過。首戰 750M／390 事件回歸亦通過，原版狀態與前次完整畫面相同。總表 SHA-256 2c7c4dcf7cc3ff4188701df1b2900f35ac20522a69dd9fb8e354a02ab8eb7e60。
- 本機保存及 verify-only 通過：797 PNG、158 提示詞、578511578 索引 bytes。新增執行器契約及來源索引已保存 assets/hd-b；原版圖、字模、音訊及執行收據維持本機。唯一覆蓋表為 106 OZM prepared、一項 VISUAL 分層、五項尾端未知；新增 20 項已註冊，新增正常來源只列 OP5。
- 來源邊界診斷確認五筆 OZM 的三組 payload 都在第四平面的最後一 byte 前到達 EOF。前三平面及已知前綴 SHA 保存，不補右下角八個像素。RAINEND.PCX 可解為 640×480／256 色，原版使用端未確認，不據此新增繪圖模式。
- 普通歷史簡介在 116451960 步遇到 STD，普通離開在 80014500 步遇到 BIOS AX=0003。按公開平台契約補齊。倒序 MOVSB 的首次測試漏設 DS／ES 描述子，修正驗證環境後乾淨重跑四套件通過，研究入口 39.583 秒。普通探針首次漏建 startup-probe，保留環境錯誤後在 ready 子目錄重跑同一版本及輸入，未算產品缺陷。
- STD 版本標題、缺圖及完整首戰再次通過。普通離開在 80017192 步 dos_exit，BDA 恢復模式 3／80 欄；這不證明文字畫面呈現。歷史簡介越過 STD 後，在 116451971 步遇到 word RCR BX,1，正常載入 OP2 但未完成繪製。平台總表 SHA-256 7d2b3ea2ca43ad3e27b6535444c63d4120234999aec9db4475263bceec51f3ec。
- RCR 的一位暫存器形式依平台手冊達 READY，後續建置與普通路徑結果接續記錄。全遊戲 HD 未完成，三平台打包尚未開始。

RCR 收尾：最終四套件通過，研究入口 38.639 秒；新版本正常標題 200M 與首戰 750M／390 事件再次通過。完整 RAM、十五欄及所有 X11／原圖／HD／還原符合既有基準。缺圖證據保持 STD 版本，來源差異嚴格限 CPU 補丁與 CPU 測試。最終平台總表 SHA-256 bd9a8808a12469a7dffb63f22436ac4bf50c44ab3d32c2318b100f7ae641b216。新的選單驗證器首次誤讀 step 欄位，改為收據的 scheduled_step／actual_step 後重跑通過，未改產品或輸入。

普通歷史簡介自然 600M，經 OP2／OP2_P、OP4、劇情、OP8、OP9、OP-FULL 回到主選單，未遇未支援服務；這些特殊繪圖仍待 HD 接入。正常 OP-FULL 載入的前三平面與已知來源一致，枚舉第四平面唯一未知 byte，只有 0xAA 符合完整原版 SHA。完整索引與證據只存本機，沒有補 DAT 或修改靜態解碼器，也不推廣到其他未觀測狀態。來源證據 SHA-256 d5b43b21ed717cef2a426c2fff9cb46903f916d88651345030b9e05403ee62cf。

補丁新 hunk 的 context 行觸發 Git space-before-tab 誤報，以 .gitattributes 僅排除該補丁的此項規則，其餘 whitespace 檢查保留。全遊戲 HD 未完成；下一步為歷史片段特殊繪圖及 OP-FULL 已知原版來源，三平台打包尚未開始。遠端與最終清理結果接續記錄。

GitHub 收尾：功能、來源契約與工具提交 f998956b63601749c61d1f2567d3f7eef406131c 已推送私人 main，主機 gh 核對遠端 SHA 相同。1023 份 assets/hd-b 檔案的 Git 子樹 SHA ca761eec8e76431b957b91077c4cd6e44adb7c25 與本機一致；配合逐檔 SHA-256 保存核對，確認圖片、提示詞與 metadata 相符。遠端 tree 未截斷，沒有 original/ 或 workplace/。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6180775383 。797 PNG／158 提示詞、原始 ZIP／12 解包檔、CRC、Python 語法、文件連結、入口正對照、228 項目前來源／快照 SHA 及 Git whitespace 均通過。完整工作根沒有 root-owned 檔案或 .md 目錄，新增工具與證據抽查 UID/GID 1000:1000；所有本批容器已退出並移除，沿用既有 image。全遊戲 HD 目標保持進行中，打包未開始。

## 2026-10-15：家康完整開場圖與原版色盤轉場

- 命中復古規格驗收、文件職責、IDA 及平台規格入口，沿用唯一狀態表。沒有啟動 subagent，外部 dosgolem 及正式 IDA 資料庫維持唯讀。
- IDA 9.4 隔離副本與普通 600M 只讀觀測確認完整複製、四個來源指標別名及位移範圍。65 筆自然返回、78 項來源快照與完整 RAM／十五欄通過，OP-FULL 兩次正常來源的尾端皆為 0xAA。未補 DAT 或靜態解碼器。
- 內建 imagegen 製作 OP-FULL B 風格；第一版依共用色盤參考出現綠色背景帶，原生色盤證據確認藍白雲帶後局部生成第二版。兩版原始圖、校準圖、提示詞及有序參考 metadata 均保存。第一版先以 `73812a83cfa22d07bd295618d8d75c8b7d25853b` 推送私人 repo，最終選定第二版。
- 完整複製的首版四套件、標題與首戰通過，但 GUI 只短暫匹配 OP4，未通過 OP-FULL。保留原版本後，以六個原生亮度與兩個循環樣本查清來源專用色盤。已知 VGA DAC 六位元契約直接沿公開規格與固定 dosgolem 轉換，不深挖硬體等待。
- READY 後只讓 OP4／OP-FULL 接受精確 48 通道的亮度／循環族。HD 色彩轉換屬呈現近似；來源、目的、遮擋與未知色盤仍保持拒絕，原版 palette、時序、玩法及存檔未改。新全畫面入口撤銷舊圖層，捲軸需重新核對完整 VISUAL。
- 最終四套件通過，研究入口 31.477 秒。五次 GUI 使用相同 82 項來源與二進位；正常、缺圖、八時點色彩各自然 600M，完整 RAM／十五欄、原圖／HD／還原及 X11 公式通過。效果模式十七次切換，六個淡入及兩個循環時點都驗證。個別缺 OP-FULL 僅退回該圖；標題 200M、首戰 750M／390 事件及完整畫面回歸通過。
- 效果驗證首輪最後暫停跨至 550M 色盤更新邊界，保留失敗收據及七組擷取；以先前已確認的觀測起點在 ready 子目錄重跑通過。只調整擷取時點，沒有調整產品資格。素材 README 改版後的 verify-only 曾早於保存完成執行；同步保存工具文字並等保存結束後，重跑正常保存及核對通過。
- 最終總表 SHA-256 `2ab4e2768c2fc6f59671f4102e47c2ec7cf0dfe3f9f9b5bbb44206fbfd190554`。素材庫保存與 verify-only 為 801 PNG、160 提示詞、587394203 indexed bytes；唯一覆蓋表為 107 OZM prepared、一項 VISUAL 分層及四項缺美術，靜態解碼仍有五項未知。原版來源、擷取、RAM／PCM 及執行收據維持本機。
- 全遊戲 HD 維持進行中，下一步局部／捲動開場與其他動畫路徑；正式三平台打包尚未開始。本批建置、GUI 及準備容器已結束，沿用既有 image。遠端、文件入口及擁有權最後核對結果接續記錄。

GitHub 收尾：功能、第二版開場圖與提示詞提交 `6a2e45d71ea9a0c9555b27ae73777879cd656a84` 已推送 main。主機 gh 確認 PRIVATE、遠端 main 相同；1035 份 assets/hd-b 檔案的完整 Git 子樹 SHA `01a2874f341b1b8aaf8d6604708211efb2391db1` 與本機一致，tree 未截斷，original/、workplace/ 未加入遠端。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6182178946 。保存核對、ZIP／12 解包檔與 CRC、Python 語法、文件連結、工具入口正對照、410 項目前來源／快照 SHA、UID/GID 與 Git whitespace 通過。完整工作根沒有 root-owned 檔案或 .md 目錄，全部 tokugawa 容器已退出並移除，沒有重建 image。

本批後續只讀切片：新增 tools/verify-hd-opening-partial-source.py，以既有普通歷史簡介的匹配快照核對 14 次未位移局部繪圖。三格交鋒、五張歷史劇情及武將／眼部來源與目的逐點一致，矩形外未變；OP9 的差異只在已知暫存區。OP2_P／OP9 原生色盤另有差異，保留 DRAFT，未改 Go 或既有 HD 結論。證據 SHA-256 `e179e94ac069b7b8a255a38cd4cb0afcc1d3c33e64bd1c0b8ea9ecf0862f5245`，來源及產物留本機，工具與入口保存私人 repo。來源、色盤與暫存區審查是下一步，不重開已通過完整複製。此切片容器已正常結束並自動移除，沿用既有 image。

## 2026-10-15：局部開場 HD、五張歷史劇情與缺項草稿

- 上一輪屬實際進展。本輪從乾淨 250e25e 核對目前狀態，命中規格驗收與文件職責入口，沿用逆向、IDA 及 imagegen 技能；沒有啟動 subagent。接入前逐筆重查原始證據，發現 OP9 的六筆來源差異數其實為零，已在研究記錄追加勘誤，正式實作維持完整四平面指紋。
- READY 後接入 sub_11E37 的三格交鋒、sub_11BA6 的五個武將偏移、sub_11C16 的四格眼部與 sub_11DCB 的五張歷史劇情。來源 alias、自然返回、完整來源與目的索引都保留；兩組來源專用色盤只接受精確 48 通道，其他來源不放寬。未知新繪圖仍扣除舊畫面資格，沒有 guest、玩法或存檔寫入。
- 首輪 fixture 的 NRGBA 型別轉換編譯錯誤保留於 runtime。修正測試後在 runtime/ready 重建，四套件通過，研究入口 35.467 秒；84 項來源凍結後沒有改 Go。十四個真實來源 fixture、全部宣告偏移、錯誤參數／alias／來源／目的、缺圖、色盤與遮擋通過。
- 慢速真實視窗已完成正常十四組、缺圖十七組。正常第三張武將的暫停跨過畫格，520100001 步停止；缺圖已完成全部畫面，外層期限於 592333333 步結束，兩者均未產生完整收據。原輸出及當時驗證器全文保留；沒有將這些停止當產品缺陷或完整驗收。
- 再查正常試玩與驗收路由，將畫面抽樣及完整狀態分開。新的 capture 工具重新核對 93 張實際 X11／呈現 PNG、原生索引裁切、HD 公式及 F6 proof；正常 29 次、缺圖 35 次切換皆不改原版。另以同一二進位、同一普通輸入及同一 84 項來源，在真實視窗各完成正常／同三項缺圖的 600M；完整 RAM、十五欄與基準一致。全部十二個新局部返回及五個劇情返回都取得來源資格。
- 標題 200M、首戰 750M／390 事件回歸通過，首戰三張完整畫面與前版相同。正常 HD 武將仍只列兩張 X11 抽樣，其他三張維持來源／共同呈現測試分類。擷取器增加現有 F7 脈衝控制及收尾期限，未重跑全十七格，不聲稱新操作策略已驗收。
- 局部總表 SHA-256 `ba0de3c64fcf7fffb1ee390c3e99d25b240ed874162d7ee3c68a399c0b5d03f3`；畫面總表 `3f735af6968610df41a0fa161f4c4795ec9e8f42677572199676d54071372138`。唯一覆蓋表目前 107 OZM prepared、一項 VISUAL 分層、四項 draft；正常劇情十四張，20 項共用來源中 OP5／OP2_P／OP9 三項有正常抽樣。
- 等待 GUI 時，以既有原版收據核對十三個 OP2／OP8 位移與跨列複製；OP8 起初的 8／7／6／1 個來源外像素明示未知，未接入正式路徑。新證據 SHA-256 `b46d791603cf03c77261106439f703744edb5f67bceecb86bf60a14f4508e319`。兩組穩定色盤與 OP2_P／OP9 相同，下一步審查來源轉換及保護契約。
- 四項 EOF 來源抽出 47992 個已知像素，未知八像素參考區設透明，不補 DAT。內建 imagegen 製作 E-4 營帳，再加入 B-6 前景武將；B-7／C-10 依相同 payload 共用草稿。兩圖、完整提示詞與有序參考 metadata 已先以 02412b9 推送私人 repo。保存及 verify-only 為 803 PNG、162 提示詞、593441022 indexed bytes；原版參考、平面、RAM／PCM 及執行收據仍留本機。
- 全遊戲 HD 仍在進行，下一步捲動、四項部分來源的校準／資格及其他動畫消費端、PCX；三平台打包尚未開始。全部本批建置與 GUI 容器已結束，未重建 image。最後來源、文件、擁有權與遠端核對結果接續記錄。

本批提交 `227d3b97fc56e161b5d2990f57cc2202c03f00b2` 已推送 main。主機 gh 核對 PRIVATE；1041 份 assets/hd-b 檔案的 Git 子樹 `2df9e2b6d4c3134cec8b80d499e6c3c188abdf8f` 與本機一致，tree 未截斷，original/、workplace/ 未加入遠端。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6183521430 。來源完整性、803 PNG／162 提示詞保存、504 項來源快照 SHA、六個相同二進位、文件入口與連結、Python 語法及 whitespace 通過。完整工作根沒有 root-owned 檔案或 .md 目錄，tokugawa 容器為空。

## 2026-10-15：OP2／OP8 捲動開場 HD

- 先完成前批遠端核對及 Issue 留言，94c47f2 保存交接收據。本切片命中規格驗收與文件職責路由，逐筆重查十三筆來源、IDA 9.4 原始入口與兩段迴圈，審查後記錄 READY；沒有新產品分支、硬體考古或 subagent。
- 新增兩個原版自然入口的唯讀資格、連續位移、線性跨列呈現及來源外像素保護。N=0 仍要求完整四平面 SHA；後續要求相同指標與整張已知來源符合 N+1。自然返回核對實際讀取區與完整目的，未知及不合資格處備援。只接受精確共用／兩組穩定／全黑色盤，原版指令、VRAM、DAC 及存檔不改。
- 四套件首次通過，研究入口 49.585 秒。十三筆真實 fixture 與全部 682 個連續繪圖狀態、防護及共同呈現通過；建置後 86 項來源凍結。二進位 5df4a99c1e4a7f46d75ee5ba51dfc4843d34ad84c6e3555be4c99fed7fed31c8。
- 正常／個別缺 OP2、OP8 各五組、共三十張 X11 配對畫面通過。正常 OP8 N=1 的七個來源外像素保留，缺圖 N=2 的六個亦保留；各十一個 HD 切換不改原版。兩模式都自然完成普通歷史 600M，完整 RAM、十五欄與原版相同，700 次資格返回包含新增 682 次。
- 首戰首次完成 750M 後，驗證器因漏準備 LICENSE-toolbar-font.txt 快照停止。保留上層結果並將失敗分類為驗證準備；補齊與早期凍結 SHA 相同的 86 項快照，在 regression/ready 用同一二進位與命令乾淨重跑通過。三張完整視窗與前批相同，沒有重建或改 Go。AGENTS 重生入口補上執行前保存 tools 快照的命令。
- 捲動總表 SHA-256 6d33f51a4b447c21fc4747a5010922a92e06074d27df7158703d78278411415a。來源 ZIP、CRC 與十二個解包檔於 2026-10-15T03:35:53Z 再核對通過；86 項凍結來源、Python 語法及 Go 格式通過。兩個未使用的空輸出目錄已移除。
- 已有素材及來源分類隨覆蓋清單更新；沒有新增或替換生成 PNG。下一步四項部分來源的校準／資格、其他動畫消費端及 PCX。全遊戲 HD 與三平台打包尚未完成。遠端提交、保存核對與清理結果接續記錄。

本批提交 `707a24342d1e679fb0a0575edbb5502f4007db20` 已推送 main。主機 gh 確認仍為 PRIVATE、遠端 main 與本機相同；1041 份 assets/hd-b 檔案的 Git 子樹 `1c19a334ae65fdec5ad74d7541fd775f5f28f9d5` 相同，tree 未截斷，original/、workplace/ 未加入遠端。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6183897037 。保存與 verify-only 為 803 PNG、162 提示詞、593441022 indexed bytes；覆蓋表的正常靜態來源為四項，全遊戲範圍仍未完成。Python 語法、Go 格式、文件連結、工具入口正對照、凍結來源與快照、完整來源 CRC 及 Git whitespace 通過。完整工作根沒有 root-owned 檔案或 .md 目錄，全部 tokugawa 容器已退出並移除，沿用原 image。

## 2026-10-15：四項部分來源故事與 PCX 草稿

- 上一輪完成捲動，屬進展；本輪由乾淨 484d017 與空容器清單接手，命中規格驗收及文件職責路由。核對兩組實際已知平面與既有手繪圖，記錄 DRAFT、視覺審查及 READY，沒有訪談、subagent 或修改外部 dosgolem。
- 四項故事來源沿三個完整平面及第四前綴識別，不固定未知最後 byte。新增 loader 返回、兩個已證實故事入口及自然返回 guard，包含相同內容的新基址與未知值變更拒絕；呈現核對 47992 個已知原生索引，八個未知像素保留原圖。未改 guest、規則、輸入、DAC 或存檔。
- 準備兩張 640×300 校準素材及純 HD 總覽，原始生成 PNG／提示詞不改，原圖對照僅留本機。四來源共用兩張素材，75 個完整劇情加四個部分來源；正常十四張範圍不擴大。
- 四套件首次通過，研究入口 79.627 秒。兩組實際來源、四名稱、256 種 synthetic 未知 byte、兩入口、來源／基址／guard／原生圖、色盤及缺圖測試通過。建置後 90 項來源凍結，兩次 GUI 已先準備完整 tools 授權快照；沒有重現上一輪的快照缺檔。
- 普通歷史 600M 與首戰 750M／390 事件的完整 RAM、十五欄均與原版相同，首戰三張 X11 與前版完全一致。四項素材載入 ready，但這些正常流程沒有遇到四個新故事場景，保留 source/shared-consumer tests 分類。總表 SHA-256 8ef0940819f3280b1763eb7b266514c6a565876c7ab5f6e2511b886af06f2d52，二進位 dae00e3e4fd91e217159078b95447eb1c952e46760f790c8f2a0c458f48fa268。
- 回歸期間用內建 image_gen 製作 RAINEND 出版商標誌草稿，1448×1086，圖像 d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93；提示詞 b7266d44f0c14b019842da63d8756bea3997d75eff29fa0a7c9e76c88f120cbe。保留彩色立體標誌與 rainbow 字樣，底部版權文字另留原版，原 caller／顯示模式仍待查證，未接 production。
- 覆蓋表為 111 OZM 有素材、一項 VISUAL 分層，四部分來源已條件式註冊；PCX 為新草稿。私人保存目前 807 PNG、163 提示詞、596522181 indexed bytes。全遊戲 HD、其他動畫消費端及 PCX 尚未完成，打包未開始。遠端核對及最後清理結果接續記錄。

本批提交 `68a89729e1f403edc7fd28d6904e9493cd0d2a9d` 已推送 main。主機 gh 核對仍為 PRIVATE、遠端 main 相同；1050 份 assets/hd-b 檔案的 Git 子樹 `9de9713108cf662e0b2bb39e46b5ab7aec8c1561` 與本機相同，tree 未截斷，original/、workplace/ 未加入遠端。Issue #5 進度：https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6184379063 。807 PNG／163 提示詞的保存及 verify-only 通過；180 項來源快照與 90 項目前凍結來源相同。原始 ZIP CRC、十二個解包檔於 2026-10-15T04:20:44Z 再核對通過，Python 語法、Go 格式、文件連結、工具入口正對照及 Git whitespace 通過。完整工作根沒有 root-owned 檔案或 .md 目錄，全部 tokugawa 容器已退出並移除，沒有重建 image。

## 2026-10-15：啟動顯示選擇完成，MARK 自然執行及跨頁例外定位

- 使用者確認恢復包內 MARK 標誌、MARK 也要 HD，並指定進入遊戲前選原版／HD，遊戲內頂端功能列仍可動態切換。本次完成啟動選擇與既有功能列整合，MARK 的正式接入仍進行中。
- 路由載入 spec-gated workflow、文件職責、IDA 工具及平台規格優先。X11 首幀擷取問題另載入 retro-game-playtest。沿用既有固定 Docker image、UID/GID 1000，未修改外部 dosgolem 或正式 IDA 資料庫。
- 新桌面選擇預設 ask；滑鼠、Tab／方向鍵與 Enter／Space 操作。確認手勢放開前原版維持 0 步，選擇只改呈現設定。明示 original／hd 旗標供自動驗證；既有驗證器已更新明示 original。
- 同一二進位 56f1a4897e40cbe30c42d47068f9bbfe0761e5084a7ccaf942fc3be85fd6c8ee、93 項凍結來源：原版／HD 選擇各自然 200M，功能列原版／HD／還原的原版 proof 相同；完整首戰 750M／390 事件，RAM 與十五欄符合既有原版基準，三張 X11 與呈現 PNG 相同。四套件通過，cmd/leprobe 65.334 秒。
- 初輪腳本誤把正常游標移動列為失敗，保留兩份失敗輸出與原腳本，乾淨重跑通過。初始黑圖是首幀呈現前擷取，保留原圖並加 X11 像素後置條件；獨立 startup-only 擷取及目視確認通過。沒有把環境或驗證器問題登記成遊戲缺陷。
- 匯總驗收 desktop-prototype/startup-choice/verification-summary.json，SHA-256 173ca02527bafa43cffcb3698b5db435bfe111f8c8bec7246ebf61897297f9de。精確 R2 驗證器由保留前版與本次已記錄修正重建，SHA-256 必須符合當次收據；來源快照與各版本輸出保留。
- MARK.EXE.BAK 自然 LE 入口先停在記憶體 byte SHR，依 Intel 規格擴充 C0 /5 ib。131072 組值／計數／DS、SS 輸入與界限拒絕測試通過。DPMI 初始 data descriptor 的可寫性依公開 DPMI 契約及固定 dosemu2 實作修正；base／limit 邊界保留。兩項修正接在 dosgolem.patch 後，只套研究副本。
- MARK 已自然選取 VESA 0101h 的 640×480、256 色，仍在 5,381,498 步因原版跨頁例外停止。IDA 確認原版 INT 0Dh handler、分頁及 IRET；Open Watcom 官方文件確認 DOS/4GW 正規化中斷／例外堆疊。原版 MARK 尚未繪圖返回、自然退出或 HD 接入，不稱啟動標誌恢復完成。位址、原始雜湊、R 陣列順序勘誤與後續最小入口見 RESEARCH-LOG。
- 另保存動畫 OR 合成與透明繪圖消費端的窄範圍原始證據；其他動畫未因定位而列完成。
- 原版 ZIP／12 檔、唯讀及 UID/GID 於 05:20:40Z 通過；807 PNG、163 提示詞、596522181 bytes 保存核對通過。沒有新增生成圖，原版、PCX 合成預覽及執行收據仍留本機。
- 主機 gh auth status 通過，repo 確認 PRIVATE，Issue #5 為 OPEN。提交、推送與遠端留言核對於本筆後續追加。全遊戲 HD、MARK 及三平台打包尚未完成。

本批程式提交 414278dca9cdbeaf9dce7eac142ec17fb12c61aa 已推送 main，主機 git ls-remote 核對一致。Issue #5 保持 OPEN，進度留言已送出並可回查：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6185165391

PRIVATE 狀態再次確認，遠端與本機 assets/hd-b 子樹皆為 9de9713108cf662e0b2bb39e46b5ab7aec8c1561。沒有原版或 workplace 進入版控。完整工作根沒有 root-owned 檔案或 .md 目錄，正式 DX5.i64 雜湊不變，本批 Docker 容器均已自動移除。

Git 的補丁檔空白檢查曾把 unified diff context 前綴視為來源格式錯誤。更新 .gitattributes 時誤覆蓋原有 dosgolem.patch 規則，已恢復原兩行，只增加 mark-cpu.patch 的格式前綴例外；其餘 whitespace 檢查保留，補丁本身、二進位及 93 項凍結來源未改。這項修正與遠端記錄隨後續文件提交保存。

## 2026-10-15：MARK 自然執行及獨立桌面 HD 通過

- 上一輪完成啟動選擇及首戰回歸，本輪屬實質進展。依平台規格優先、spec-gated workflow 與文件職責續做 MARK。Open Watcom 的 DOS/4GW 例外堆疊契約與既有 VGA 標準時序作平台依據，沒有深挖硬體 driver／ISR 時序。
- 新增結構化段界限回報、DOS AH=25h 向量查詢、normalized frame 交付及平坦 32-bit IRET／CS word CMP。四個原版 VESA 分頁由 MARK 自己的 INT 0Dh handler 完成，原 REP 從剩餘進度續行。
- 補齊原版色盤漸入／漸出的 memory byte ADD／SUB，262144 組值與既有 AL immediate 語意差分、DS／SS、唯讀及界限測試通過。r5 的 SIB 測試把 EBP index 當成 base，保留失敗與分類後訂正；沒有修改正確的位址解析。
- r6 完整索引已符合 PCX，但在未初始化的 VBE 回掃等待停止；VBE BIOS 以 dosgolem 既有標準表建立 0100h／0101h 時序，保持色盤及 pixel geometry，頻率及雙態回掃測試通過。r8 原版入口自然 15989208 步正常退出，完整漸入、停留、漸出及向量／文字模式還原均保存。
- 獨立 PCX 解碼、307200 色號與 runtime 目標色盤均一致。固定 program=game／mark profile 分離原版來源；MARK 拒絕 DX5 位址觀測，獨立執行不偽造 COMMAND.COM 或 EXEC 成功。
- MARK 桌面 HD 使用既存私人生成圖，沒有新增圖片。640×480 原版保持兩倍畫布，頂端列擴高至 1024，y=449..479 版權區保留當下原版像素。來源、色盤與 RAM guard 限定資格；色彩效果屬呈現近似，未知狀態及缺圖回原版。
- 四套件及 HD 資格／版權／備援／色盤／viewport 測試通過，cmd/leprobe 115.021 秒。同一二進位 4d66c2a7ce4493ff77b180a0fe5b411c902e9406a9a697934002ce7afe0acb16、102 項凍結來源：兩次 MARK GUI 的完整 RAM 及十三個原版欄位符合 r8，六張實際 X11 符合呈現 PNG；遊戲 750M／390 事件的完整 RAM、十五欄及三張 X11 通過。
- 總表 desktop-prototype/publisher/verification-summary.json，SHA-256 580a71ffc2fe8e23cda8c6122871b9f9b2f1c4c05f1c07b5c0c08fceb7e63e0b。亮起原圖、上方 HD 採樣及下方版權另以獨立公式核對。原版與 HD 截圖已目視查看。來源快照保存全部 102 項；r8 已保存當時五份獨立研究工具的精確雜湊副本。
- 唯一 art-coverage.json 已重生，RAINEND 列為已準備、獨立 MARK 桌面驗證；完整啟動串接仍 false，其他十七項動畫消費端未列完成。
- 主機 gh 登入及 PRIVATE 狀態已核對，Issue #5 為 OPEN。提交、來源完整性、圖片保存及 Docker 清理核對於收尾追加。全遊戲 HD 與三平台打包仍未完成，下一步為先選顯示模式，再依序執行 MARK 與 DX5 的桌面啟動器。

收尾核對：原始 ZIP CRC、十二個解包檔、唯讀及 UID/GID 於 2026-10-15T07:08:11Z 通過。807 張 PNG、163 份提示詞、596522181 indexed bytes 的保存核對通過，圖片子樹未改。四份 Python 工具語法及 run-probes.sh 的 shell 語法通過。完整工作根沒有 root-owned 檔案或 .md 目錄；正式 DX5.i64 的 SHA-256 仍為 8afa40d3eaeed4851b78ed2ddc80ccbc1c65e14cb2481a09316d42670efce02e。全部本批 tokugawa 容器已退出並移除，沒有重建 image。程式與文件提交及遠端 Issue 進度隨後追加。

程式提交 e42ed9356d44059a9b3ef49c881eddb4ff2932e3 已推送 main，git ls-remote 與 GitHub tree API 核對一致。102 項目前來源與建置快照的 SHA-256 完全相同，十一份新增工具均已登記 AGENTS 入口且為 UID/GID 1000:1000。儲存庫仍為 PRIVATE，遠端 1050 份 assets/hd-b 檔案的子樹為 9de9713108cf662e0b2bb39e46b5ab7aec8c1561，tree 未截斷；沒有 original/ 或 workplace/ 進入版控。Issue #5 的進度已送出，標示獨立 MARK 驗證完成、完整啟動鏈仍待串接：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6186670059

## 2026-10-15：完整桌面啟動鏈實作及驗收

- 前輪為實質進展，完成 MARK 原版執行及獨立 HD。依已確認範圍接續顯示選擇 → MARK → DX5；載入 spec-gated workflow、桌面試玩與文件職責，grilling 核對已有相同決定，沒有重新詢問。
- 固定 Ebitengine 2.9.9 原始碼明示同一程序不得呼叫 RunGame 兩次。READY 契約先寫入 RESEARCH-LOG，再以同一二進位的 mark profile 作子程序；父程序確認原版正常退出後才執行 game profile。兩份 RAM 與虛擬時間獨立保存，沒有模擬 COMMAND.COM 或假造 EXEC 成功。
- 預設在 MARK 前選擇一次；MARK 的最後 hd_enabled 帶進遊戲，遊戲保留既有功能列。關窗停止整個啟動器，錯誤與非正常退出停止；中斷會取消子程序。十一份既有 DX5 局部驗證器明示 publisher-intro=false，完整鏈驗證器使用預設值。
- 凍結 103 項來源，二進位 SHA-256 baafa065d2f8b204dae538371ee9de617b2f53717a14ec5d9c54d3086ca2d859。四套件通過，研究入口 53.770 秒。正常 WM_DELETE_WINDOW 關閉模式選擇畫面已通過，MARK 0 步、無輸入轉送，DX5 未執行。兩種最後模式的完整鏈及首戰回歸仍在驗收，未先宣稱 CONFORMED。

三模式驗收與獨立匯總均已通過，完整啟動鏈契約追加為 CONFORMED。兩條正常流程刻意在 MARK 改成另一種模式，遊戲確實沿用最後設定且沒有再詢問；各自然完成 MARK 15989208 步與遊戲 750M／390 事件。兩次 MARK 完整 RAM／十三欄、兩次遊戲完整 RAM／十五欄及完整 PCM 符合各自基準，十張實際 X11 與呈現相同；MARK 的版權區及功能列切換／還原的原版 proof 均保持。總表 startup-chain/verification-summary.json 的 SHA-256 為 1ad1539a0bca5c4eabd0d2ff0dfef099af235ed74a8359f0afd2c207fa074a72。MARK HD 與首戰 HD 已目視檢視。

覆蓋盤點初輪因舊收據要求目前驗證器的 SHA-256 而失敗，屬工具版本核對問題。新增 publisher-intro=false 改變了十一份局部驗證器的原始碼；三份受影響收據改以已保存的完整快照核對當時雜湊，並列出 verification_tool_sources。沒有修改舊收據或放寬雜湊比對，乾淨重跑通過。唯一覆蓋清單的 RAINEND 現為 game_startup_integrated=true；其他十七項動畫消費端未列完成。

原始 ZIP CRC、十二個檔案、唯讀與 UID/GID 於 2026-10-15T07:37:58Z 通過；807 PNG、163 提示詞及 596522181 indexed bytes 核對通過，圖片子樹未改。文件另載入 README 現況規範，更新唯一狀態表、待辦與穩定入口。103 項編譯輸入保持凍結，完整根沒有 root-owned 或 .md 目錄；正式 DX5.i64 雜湊不變，全部本批容器已移除。下一步依既有證據接剩餘動畫及特殊繪圖消費端；全遊戲 HD、桌面音訊與三平台封包仍未完成。提交與遠端核對隨後追加。

提交 d4a27beb0645cef522ccf9e0bce91ce704b1a32e 已推送 main，git ls-remote 與 GitHub tree API 核對一致；主機 gh 登入有效，repo 仍為 PRIVATE。遠端 assets/hd-b 仍有 1050 份檔案，子樹 9de9713108cf662e0b2bb39e46b5ab7aec8c1561 與本機一致，tree 未截斷，沒有 original/ 或 workplace/ 進入版控。103 項來源及快照、全部 Python 語法、Go 格式、文件連結與新工具入口核對通過。Issue #5 的完整鏈進度已送出，仍保留全遊戲 HD 待辦：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6187209150

## 2026-10-15：首戰普通操作與合成消費端查證

- 依復古遊戲路由載入逆向證據、規格閘門、平台規格優先與 IDA 9.4 工具契約。從正式資料庫的獨立副本取得 OR 全畫面、透明矩形及動畫 caller，保留原始名稱、載入後位址與 bytes。
- 初次交叉核對把 IDA 載入後 bytes 與原始檔案 bytes 混用而拒絕。改用固定 dosgolem LE loader 後，448 列指令全部相符，32 列 relocation 差異另記，不改正式 DX5.i64。
- 新增預設關閉的唯讀合成探針及普通滑鼠接續旗標。既有 390 筆事件全部結束後，才以真實 X11 滑鼠接續；保存來源、繪圖前後索引、自然返回及實際視窗，不注入原版狀態。
- 首批 105 項來源的四套件與 750M 控制組通過。普通選取單位開啟移動／攻擊／特殊選單；選攻擊後在 754004409 步遇到未支援的 D1 E0。原始失敗收據保留在 consumers/battle/。
- 依 Intel SDM 補 D1 暫存器 SHL count-one。r2 測試同時發現舊 C1 count-one 沒有正確設定 OF，另有舊否定測試仍拒絕新形式；保留失敗並訂正標準契約與測試，不迎合錯誤 oracle。
- r3 凍結 106 項來源，四套件通過，cmd/leprobe 147.457 秒。二進位 SHA-256 e963049505486439fb063df22364f2b73fea18e5d36e517242494ab02bd8c32a。原版控制組與相同普通攻擊操作正在重跑；正式 HD 合成仍為 DRAFT，尚未列為接入完成。

本輪沒有新增圖片。主機 gh 登入有效，儲存庫仍為 PRIVATE，Issue #5 保持 OPEN。後續收據、來源核對、清理及提交於本節追加。

r3 的 750M／390 事件控制組通過完整 RAM 與十五欄核對，普通攻擊越過 D1 後於 752026383 步停在 66 39 07。依 Intel CMP 契約補唯讀 word memory；r4 的標準邊界測試未設 DS 描述子而被正確拒絕，補測試條件後保留 r4，以 r5 新快照重跑。

r5 四套件通過，cmd/leprobe 98.320 秒，桌面二進位 SHA-256 e46979be891ad469b6b274e822e5f8975bfa0be9025f310155898d48a9c30089。750M 控制組的完整 RAM／十五欄仍符合基準；同版 MARK 自然 15989208 步正常退出，完整 RAM／十三欄符合 r8。MARK 控制環境初次缺少 startup-probe，第二次缺少收據所需 engine 來源，兩次屬啟動／收據環境錯誤；保存 stderr，補完整輸出環境後以相同命令乾淨重跑通過，沒有修改產品。

r5 普通攻擊於 751041328 步停在 66 81 3D 50 60 03 00 FF FF。依同一 Intel CMP 定義補 81 /7 16 位元立即值形式；r6 新快照凍結 106 項來源，正在建置及驗證。各次 X11 時間序列不同，沒有宣稱首戰後的不同停止步數可逐步同狀態比較。尚未取得實際 OR／透明返回，正式 HD 合成維持 DRAFT。

r6 四套件通過，cmd/leprobe 90.797 秒，桌面二進位 SHA-256 3bd1f710e71ac3e44627c59ec815b811cfec56e23760632d8bc9e01efe81b09d。750M／390 事件控制組完整 RAM／十五欄、MARK 自然 15989208 步完整 RAM／十三欄皆通過；106 項目前來源與快照雜湊相同。桌面普通攻擊仍在接續，這是平台與研究工具的中途保存，不是戰鬥 HD 或全遊戲 HD 完成聲明。

中途提交 d58cf1c7154b3462df0a19166c5d96ce462e4962 已推送 main，git ls-remote 核對一致。repo 仍為 PRIVATE，assets/hd-b 子樹仍為 9de9713108cf662e0b2bb39e46b5ab7aec8c1561。提交前 Python 語法、shell 語法、106 項編譯來源、正式 DX5.i64 及完整根的擁有權核對通過，沒有原版或 workplace 進入版控。

r6 普通攻擊於 752041330 步停在 66 25 FF 7F。集中匯出 0x1F400..0x1F800 的 363 列 IDA 載入後 bytes，由 LE loader 全數核對後，合成單步篩查只回報這個 AND AX 形式。探針初版把 Bus 的 error 契約寫成 bool 而未編譯，依實際介面修正後正常執行；未將合成單步當成正常玩家或語意證據。

r7 補 AND AX,imm16 後，同範圍 363 列篩查零錯誤，四套件通過，cmd/leprobe 96.858 秒。桌面二進位 SHA-256 2521df32aec96f68e8bd8f50bfd931560cd47fe0eedcd3a04c5885061da91c71，106 項來源與額外研究工具快照已保存。普通攻擊、750M 控制組及同版 MARK 正在重跑；正式 HD 合成仍為 DRAFT。

r7 750M 控制組完整 RAM／十五欄與 MARK 自然 15989208 步完整 RAM／十三欄通過。普通 X11 已操作單位選單、攻擊範圍、移動範圍、移動預覽與確認。將附近藍色單位稱為對手的說法缺少陣營證據，已向使用者訂正；選取該單位未觸發交鋒，不將它寫成戰鬥完成。

右鍵拖曳的普通 X11 操作改變了視野，隨後顯示含「此回合終了／武將表／勝利條件表示／戰況報告／打開音樂／關閉音樂／儲存進度／回到標題／遊戲結束」的原版選單；選回合終了後出現確認框。這是實際操作截圖，只描述已看見的行為，不推定一般右鍵捲動規則。

互動工具達到 1500 秒上限時直接 terminate 原版，r7/receipt.json 因而為空。保留全部十七次命令、十八張 X11 擷取與來源快照，不補造原始輸入或完整 GUI 收據。這是研究工具收尾失敗，原版控制組不受影響，沒有交鋒／合成驗收。

修正 explore-battle.py：到期及例外先以普通 Shift+Esc 正常關窗，保存完整原版收據；加入明示 drag 命令、3600 秒預設上限與工具測試標記。cleanup-check 的早期 1M 檢查點尚無原生圖，呈現 PNG 為 alpha=0 黑色，X11 則為不透明黑，嚴格像素比對正確拒絕。例外收尾已正常 desktop_closed 並保存 106 項來源的完整收據；這是工具測試，沒有聲稱玩家路徑通過。改以既有 10M 有效圖形檢查點驗證到期收尾，cleanup-check-r2 保留獨立批次，Go 與遊戲二進位皆不改。

例外及到期正常收尾均已通過 tools/verify-composite-probe.py --harness-close-only，結果保存 r7/cleanup-verification.json。兩次都是原版 desktop_closed、完整 RAM 與收據、106 項來源；10M 有效畫面的 X11 與呈現 PNG 完全相符。harness_only 保持 true，完整玩家／合成驗證拒絕以這些短測充數。

r8 沿用 r7 同一桌面二進位及全部 106 項凍結編譯來源，只有互動研究及驗證 Python 工具修正。原版及研究來源唯讀；已驗收的 r7/control 複製至 r8/control，reference.json 記錄原始相對路徑、收據、RAM 與二進位 SHA，明示沒有新增控制組執行。r7 原控制組及舊驗證工具快照不改，r8 正常操作另行保存，不混成同一測次。

## 2026-10-15：正常存讀檔與回合推進

- r8 正常 X11 操作「儲存進度」及確認後，DAT 工作副本只有 I003.SAV 的 134 bytes 改動，其他 200 個資源完整一致。全部十二個輸入另存唯讀副本與逐檔 manifest，下載原始資料不改，也不另擬存檔格式。
- FILE 3 讀回先因把原生座標再除二而點錯，保存原失敗；改正後到確認，再補正常確認。九筆普通輸入自然 140M，完整原生索引及包含游標的 RGBA 與存檔畫面一致。獨立核對工具加入零差異 assert，乾淨重跑通過。
- r9 從相同原版存檔正常讀回，121599999 步正常關窗。五張 X11 擷取與呈現 PNG 一致，武將表的原版／HD 切換保持原版狀態。桌面速度 1 下極短按下未開選單，真實 X11 保持至少 100000 原版步數後成功；研究工具新增 hold_steps，不改原版滑鼠契約。原版畫面及存檔只留本機。
- 十九筆正常讀檔／回合輸入在 180761953 步遇到 66 AF。依 Intel SCASW 公開契約補單次形式，新增 ES／DS 分離、雙 DF、ECX 不變、唯讀、界限及 CMP 差分測試。r10 四套件通過，cmd/leprobe 58.404 秒；106 項凍結來源與二進位保留。
- 相同十九筆輸入以同版原版探針到 200M 及 600M，沒有未支援指令。600M 畫面由 1／20 推進 2／20；獨立來源／輸入／邊界核對通過，未觀測到交鋒合成。verify-composite-probe.py --native-only 明示 INCOMPLETE，沒有將平台進展列為戰鬥 HD 完成。

r10/native-turns/ 正在以 49 筆普通輸入接續多回合；同版下載基準的 750M 控制組另行重跑，MARK 環境已備妥。沒有新圖片，807 張 PNG 與 163 份提示詞的既有私人保存不改。編譯 Go 輸入保持凍結，結果、清理及遠端保存於本節追加。

r10 下載基準 750M／390 事件的完整 RAM 與十五欄通過，同版 MARK 自然 15989208 步、退出碼 0，完整 RAM／十三欄通過。多回合探針在 1327438966 步遇到 83 75 FC 01，已套用 39／49 筆普通輸入；保留原停止及完整輸出，沒有 OR／透明返回。下一步按 Intel 80386 的 XOR 規格補記憶體形式，不逆向遊戲規則或修改原版。

中途保存前核對全部 106 項編譯來源與當次快照一致，三份變更 Python 語法通過，存讀檔與合成核對工具另以雜湊檔名保存完整來源。完整工作根沒有 root-owned 或 .md 目錄，正式 DX5.i64 雜湊不變，所有本批容器已移除，沒有重建 image。原版及 workplace 維持排除，圖片子樹不改。

提交 b527640b19de6b0302728aaf21ffa2dcd05b98d3 已推送 main，git ls-remote 核對一致。repo 仍為 PRIVATE，Issue #5 保持 OPEN，存讀檔、武將表及 SCASW 的進度留言為：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6191777727

r11 依 READY 契約補 83 /6 32 位元記憶體 XOR，256 種 imm8、四種 DS／SS 定址、標準邊界及拒絕測試通過；四套件通過，cmd/leprobe 57.091 秒。獨立 IDA 9.4 匯出 0x15B00..0x15C00，86 列載入後 bytes 相符，86 個合成單步零錯誤。形式探針新增 --rows 及 8192 列上限，不把合成結果當正常玩法。相同 49 筆普通回合輸入正在重跑，Go 來源保持凍結。

r11 原版全部 49 筆普通輸入已套用，越過記憶體 XOR，新增 WEATHER2.OZM 的正常不透明返回；於 1579516359 步停在 0x22A06 的 C1 6D FC 02，尚無交鋒合成。r12 依公開 SHR 契約補 C1 /5 32 位元記憶體，256 種 count、四種 DS／SS 定址、標準旗標及拒絕測試通過，四套件通過，cmd/leprobe 50.817 秒。0x22A00..0x22B00 的 86 列及合成原語範圍的 448 列載入後 bytes 均相符，形式篩查零錯誤；原版完整多回合正另行重跑。

為縮短後續原版讀檔前綴，r11/save-turn3/ 使用已凍結的舊二進位及來源嘗試正常存檔。初次只掛 source-snapshot 而漏掉素材診斷 inventory，第二次巢狀掛載的目標尚不存在，第三次重用已建立 composite-samples 的失敗目錄；均分類為研究環境失敗，沒有修改產品。修正為專案唯讀掛 /project、舊 source-snapshot/tools 另掛 /project/tools，並使用新 ready/ 輸出，乾淨重跑 1500M／49 事件後，原版停在存檔確認框，DAT 與舊存檔相同，未列存檔成功。ready-r2/ 只將最後普通按下延長至 1M 步，再以新輸出重試；不用 RAM 注入或猜補存檔欄位。

r12 越過記憶體 SHR 後，在 1579516888 步遇到 ADC AL,0，下一條相鄰原版指令為 ADC BX,0。依 Intel 公開契約補兩個形式，新獨立寬整數旗標測試通過；r13 的舊否定測試仍拒絕新 word register 形式，保留失敗，改以未支援的 word memory 驗證拒絕後，以 r14 新來源重跑四套件。

r11/save-turn3/ready-r2/ 成功完成普通存檔確認。新 DAT 為 3bf1827150e9955dd169c0d464b85a11a9ce2ff3e0764a7ecda4cb7a3678e1b5；相對下載基準只有 I003.SAV 的 4399 bytes 改動，其他 200 資源不變。新增 import-original-save.py 核對來源、全部普通輸入及差異後，保存唯讀原版資料副本與逐檔 manifest，沒有原版資料進 Git。

r12/load-turn3/ 以九筆普通標題／FILE 3／確認輸入自然到 140M，完整原生索引 3a6b2843173644bdcc0822e18fe2f7dd874ae04be8e12543f9eaf9629f6bc0de 與存檔時相同。獨立核對發現 217383 個非黑像素的 RGB 亮度不同，對應通道差 4 或 5，游標定義相同；沒有宣稱完整 RGBA 一致，也不深挖或猜補原版色盤時序。工具明示 PASS_NATIVE_INDICES_ONLY，後續 HD 同狀態比較以實際讀回後的原版畫面為基準。初次過度嚴格的 RGBA assert 失敗未當成 CPU 或存檔產品缺陷。

## 2026-10-15：正常交鋒與正式 HD 合成

- r14 ADC 四套件通過，cmd/leprobe 43.395 秒。115 列附近指令的載入後 bytes 相符；114 個合成單步中唯一錯誤為另一函式的 POP DS 在合成狀態讀到未登錄 selector 3，不是缺少 opcode，不調整 CPU 迎合篩查。
- 正常第三回合存檔十九筆讀檔／回合輸入自然到 600M，四次交鋒共十二筆 BATTLE OR／JIMEN 透明返回全部獨立核對通過。正常來源新增 BATTLE02、ANIME01／03／04／08 及 WEATHER2；來源與呈現分開，不冒稱動畫 HD 全部完成。
- r14 750M／390 事件完整 RAM／十五欄及 MARK 自然 15989208 步完整 RAM／十三欄通過。控制組初次漏掉凍結來源外的 player-path.json，另存環境失敗後將固定輸入複製至獨立輸出，乾淨重跑通過。原版交鋒 GUI 初次缺 startup-probe 輸出目錄，保留失敗並在 ready/ 乾淨重跑；三張 X11 與呈現 PNG 相同，212M 正常關窗。
- 依正常來源審查為 READY，再接正式矩形合成。保留 source-zero 背景、十八個 OR 混色原生像素、原版文字及個別缺圖的遮擋資格；只允許 BATTLE／JIMEN 已證實來源。矩形沿用 256／64 上限，來源裁切 SHA 與合成期望 SHA 分開。
- r15 四套件通過，cmd/leprobe 47.666 秒。真實來源及每個像素的獨立所有權測試通過；正常 600M 完整 RAM 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d 與十七個原版欄位，包含畫面、輸入、PCM、原版合成及不透明 trace，均符合 r14 原版。核對入口 r15/native-control/same-state-verification.json。
- r15 正常 GUI 六張 X11 與呈現 PNG 相同，207M 正常關窗，四次 F6 切換原版 proof 不變。200M 因原版色盤不符而整體備援，207M 正常 HD 框架、紙張、兩側頭像、山景與地面已目視。地面有 (128,286)..(208,366) 的原圖備援，對應 201901325 步已關閉視窗中的 KAO 22 舊記錄；200M 與 207M 的完整原生索引 SHA 均為 52a4eef96949e3ae1e5885c54d94572aaec05efda3a552668465c406149181a4，不能將此舊遮擋當成目前原版仍有頭像。

目前缺口是共用視窗的矩形還原尚未同步 graphics／KAO 生命週期，以及動畫還原、個別缺圖和頂端單位縮圖。全遊戲 HD、桌面音訊及三平台封包仍未完成。來源、Python 語法與完整根擁有權核對通過，全部本批容器已移除，沒有新增圖片或重建 image。

提交 4b5b06292948db5d00de3c8f9f72b1db49ceb2d4 已推送 main，git ls-remote 核對一致，repo 仍為 PRIVATE，圖片子樹不改。共用視窗還原契約經舊 CONFORMED 來源、原始矩形複製指令與正常前後完整索引相同審查為 READY，再實作區域內 graphics／KAO 快照；原版四平面保存 buffer 與返回目的均需逐點一致。r16 的測試誤將具體 *image.NRGBA 作介面斷言，屬測試編譯錯誤，保留失敗；改為直接呼叫 SetNRGBA 後以 r17 新來源重跑。

r17 四套件通過，研究入口 45.487 秒。正常 600M 完整 RAM／十七欄符合 r14；三種 GUI 共十四張 X11 與呈現 PNG 一致，九次 F6 的原版狀態不變。獨立 verify-hd-combat.py 核對十八個 OR 混色原生像素、3012 個文字前景像素、個別缺 JIMEN_1 的 82300 呈現像素及區域外 HD 不變，全部通過。207.4M ANIME08 原版騎馬人物與 HD 對照已目視，208.8M 動畫返回後的保存區符合動畫前原生及 HD 背景；下方新視窗屬後續合法繪圖，沒有被錯誤還原。戰鬥合成及共用視窗還原追加為 CONFORMED，頂端單位縮圖仍另列。

以相同十九筆普通輸入自然到 215M 的唯讀地形／前景診斷另存 r17/unit-diagnostic/。頂端兩個 48×48 縮圖正常進入 0x1F9D0、返回 0x1F9BE，來源 offset 28202／28178，目的 (240,60)／(400,60)；現有前景 guard 因返回位置不同而拒絕。原始來源對應既有 units 7／3，無須重畫；呼叫端及透明公式仍須查證後才擴充正式資格。地形事件達到 512 上限而丟棄 408 筆，完整前景診斷 58 筆、零丟棄，因此不以不完整地形序列推論後段繪圖。

提交 3f273da910697ff9e4dc190d18983b68aaeafecc 已推送 main，git ls-remote 核對一致；repo 仍為 PRIVATE。唯一覆蓋清單已更新為七項動畫／介面正常 HD 抽樣，五項靜態來源正常 HD 抽樣。驗證器全文保存於 r17/verify-hd-combat.py，歷史核對維持精確工具雜湊，沒有覆寫舊收據。圖片子樹維持 9de9713108cf662e0b2bb39e46b5ab7aec8c1561，原版及 workplace 未入 Git。Issue #5 進度：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6194745403

## 2026-10-16：交鋒頂端單位縮圖

- IDA 9.4 的獨立副本匯出 244 列 wrapper／48×48 原語，與正常 dosgolem 載入後 RAM bytes 全數相符，27 列原始檔案 relocation 差異另列。206.3M 原版 X11 的兩個完整透明公式通過，7／3 來源分別有 996／923 個非零原生像素；207.4M 下緣已被動畫覆蓋，初次整區公式正確拒絕，不把過渡畫面當穩定樣本。
- 依 READY 擴充已證實的兩個目的矩形，重用十張既有單位素材及紙張背景。r18 四套件通過，研究入口 45.974 秒；600M 完整 RAM／十七欄與 r14 相同。四種 GUI 保存完整收據，原版切換不變；實測發現繪製時來源 RGB 與後續固定來源 PNG 不同，縮圖保持備援，沒有宣稱完成。
- r18 的研究個案準備曾漏建 build-sources.json，屬研究環境錯誤。從已凍結的 106 項來源明確產生 manifest 後乾淨重跑，未改建置來源或二進位；初次錯誤另存 prepare-environment-correction.json。
- 回到 READY 補跨色盤生命週期：縮圖來源及自然返回依索引完整核對，RGB 只在當前已核准色盤下呈現。既有地面／物件的 RGB 契約保持；r19 保存新的 106 項來源與獨立建置，正在重跑四套件與實際視窗。

r19 四套件通過，研究入口 59.603 秒。獨立 --units 驗收通過四模式十七張 X11、十一次切換、兩個縮圖的完整索引及 alpha 公式、9216 呈現像素的個別單位缺圖、82300 呈現像素的個別地面缺圖，以及動畫遮擋／還原。600M 完整 RAM／十七欄符合 r14，原版 PCM 不變；來源 7／3 的實際 HD 已目視。地面缺圖第一個程序以 143 中止且收據為空，原因未知；原狀及 termination.json 保留，ready/ 相同凍結二進位乾淨重跑成功。縮圖切片追加為 CONFORMED，全遊戲 HD 仍未完成。

2026-10-16T01:27:18Z 原始 ZIP CRC、十二個檔案、唯讀及 UID/GID 通過。106 項編譯來源與快照、所有 Python 語法及 Go 格式通過；唯一美術覆蓋清單已重生，新增兩項交鋒縮圖正常來源，與地圖的兩個樣本分開記錄。兩版獨立驗證器保存完整來源及精確 SHA，r17 舊收據未改。完整根無 root-owned 或 .md 目錄，正式 DX5.i64 雜湊不變，本批容器全部退出並移除；沿用既有 image，沒有新增圖片。提交及遠端核對隨後追加。

提交 27ea807db5bb040d715cea896e9b9f66e46f2dbf 已推送 main，git ls-remote 核對一致。repo 仍為 PRIVATE；遠端圖片子樹 9de9713108cf662e0b2bb39e46b5ab7aec8c1561，1050 份檔案，tree 未截斷。Issue #5 保持 OPEN，縮圖接入進度：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6205953190

## 2026-10-16：399 列結局與紙張邊緣

- 核對 EXE 名稱表與 DAT 的不同索引，20 個來源有 23 個名稱引用，重複名稱保留各自索引。初次按 129 筆／每來源一筆假設拒絕，核對正常 JIMEN_0／1 的 129／130 後改成有界 131 範圍，以完整檔名配對，不宣稱原版陣列容量。consumer-audit 保存來源、原始定位及 IDA bytes。
- 原始 END-11／11B 的 640×399 來源由 sub_1C084 實際複製 400 列。依 READY 只資格化前 399 列，第 400 列保留原版；兩個真實來源、自然返回、未知末列、破壞及缺圖測試通過。縮圖外側 768 原生紙張像素已查證，從舊保留區移回既有 BATTLE 紙張圖層。
- r20 四套件通過，研究入口 58.924 秒；四種 GUI 及正常 600M 已完成。獨立驗證在 208.7M 的動畫還原比較拒絕：原版比較區相同，但 HD 背景有差異。查明當時下一個 WINDOW 的 (112,270)..(528,398) 只畫到一半，pending_draw 尚未返回，BATTLE02／JIMEN 因矩形外原生內容不同而正確備援；HD 差異位於比較區 y=221..259 的背景，不是紙張還原錯誤。第一次進度訊息的紙張描述已訂正。
- 保留完整初次擷取及來源，分類為驗證取樣時點。只修改研究 Python 工具：wait_restore 可另以 steady_resource 等待真正還原及已核准背景同時成立，仍保留有界步數、普通 F7 暫停及完整像素 assert。desktop-animation/ready/ 沿用同一二進位及 106 項來源乾淨重跑，沒有改遊戲、放寬守衛或增加逐週期時序研究。

新批次尚在執行時曾提前呼叫驗證器，因 ready/ 的完成收據還不存在而讀到舊批次，重複舊取樣失敗。重新載入試玩驗證入口後，驗證器改為：只要 ready/ 已建立就必須使用它，未完成明確拒絕，不退回上層舊收據。這是驗證依賴順序修正，原版與生產程式不改。

r20 的 ready/ 正常到 208.9M 關窗，獨立驗收通過。四模式十七張 X11、十一次切換、768 原生紙張邊緣像素、兩縮圖 alpha、個別缺單位及地面、文字／OR 混色、動畫遮擋及真正還原均通過。600M 完整 RAM／十七欄符合 r14，CD PCM 不變。END-11／11B 另列真實來源與合成共同消費端測試，正常結局未抽測；沒有把合成狀態列為正常結局證據。

覆蓋清單重生通過，新增兩個 399 列綁定及 768 紙張邊緣像素，正常樣本數不擴張。純呈現總表、早期失敗及各版驗證器全文均保留；106 項來源與快照、Python 語法、Go 格式及完整根擁有權核對通過，正式 DX5.i64 雜湊不變。全部本批容器已移除，沒有新增圖片或重建 image。接續 END-2 及 MAP_WAKU 特殊消費端，全遊戲 HD、桌面音訊及三平台打包仍未完成。

提交 250191c49ceb4a4867a7a65aaf67f1290256369a 已推送 main，git ls-remote 與遠端 tree 核對一致，repo 仍為 PRIVATE，圖片子樹與 1050 份檔案不變。Issue #5 的進度為：
https://github.com/wicanr2/tokugawa-dosgolem/issues/5#issuecomment-6206615456

### 2026-10-16：地圖快取主框架與後畫場景

r21 四套件通過，正常讀檔的實際視窗辨識出 541 個被後畫地形／單位覆寫的非黑來源像素。r22 記錄地面真正返回步數，主框架依先後順序保留後畫矩形，其他 OZM 契約不變。四套件通過，研究入口 102.014 秒，106 項來源凍結。正常 600M 的完整 RAM、十五個原版欄位與原有合成 trace 相同；原有 67 筆 graphics trace 不變，新增 72 筆快取框架唯讀返回。

兩個普通讀檔桌面案例共六張 X11、四次切換，68744 個框架呈現像素與 66424 個保留畫布像素逐點核對，兩案皆保持 67 處 HD 地面。個別缺圖的 68744 個底圖像素另核對固定 MAPGAMEN 索引及 HD 底圖；沒有撤銷其他已合格圖層。r21 漏建 engine、IDA 漏掛 /original，以及 r22 驗證器的 null 清單與底圖假設失敗均保留，分類為工具環境或驗證問題；最終只採完整的 r22 收據。

覆蓋清單更新為正常剩餘來源抽樣八項，MAP_WAKU 限前 88 列。下方圖塊、END-2、未抽測結局、桌面音訊及三平台封包未完成。來源、實際視窗及驗證器全文保存於 consumers/battle/r22/；原始資料、正式 IDA、807 張生成圖及素材索引保持不變，原版截圖與收據留本機。Docker、擁有權及原始 ZIP／十二檔核對見 r22。

本批提交標題為「Integrate cached map frame without covering later HD terrain」，推送與 Issue #5 保存收據為 workplace/github-setup/hd-map-frame-r22-sync.json。

### 2026-10-16：地圖下方八處快取框架

沿 r22 的原版 RAM 與 IDA 窄範圍輸出，核對主框架及下方快取的完整來源。左側三次的平面 1 偏移與靜態圖塊有 455 個索引差異，以實際 reader 公式及固定來源分開驗證，保留原值，不修 guest。READY 後接入四種美術裁切、八個目的與自然返回，黑色緩衝區及後畫場景保留現有畫布；一般／快取 trace 分別限 512／1024，避免排擠原有證據。

r23 四套件通過，研究入口 113.317 秒，106 項來源凍結。普通十九筆原版讀檔／結束回合自然 600M，完整 RAM、十五個原版欄位、原有合成 trace 及一般 67 筆 graphics trace 不變；72 次主框架、576 次下方圖塊全部自然返回且來源相符。

正常與個別缺圖兩案各以九筆普通讀檔到 140.1M，六張實際 X11、四次狀態不變切換通過。八處圖塊新增 18552 個 HD 呈現像素，主框架保持 68744 個，總計 87296 個逐點符合美術；保留畫布 80640 個，其中 58600 個仍為 HD。缺圖的底圖／原生畫面另行逐點核對，兩案 67 處 HD 地面保持。來源工具、獨立驗證器及覆蓋清單保存於 consumers/battle/r23/，未把合成測試當正常原版證據。

原始 ZIP／十二檔、唯讀、擁有權及正式 IDA 雜湊核對通過，沿用既有 image，容器已清理。生成圖及提示詞未改，原版索引、RAM、畫面與音訊只留本機。END-2 位移消費端、未抽測正常結局、桌面音訊與三平台封包仍待完成；不宣稱全遊戲 HD 完成。

本批提交標題為「Integrate eight cached lower map-frame tiles」，推送及 Issue #5 收據為 workplace/github-setup/hd-map-corners-r23-sync.json。


## 2026-10-16：END-2 結局位移的條件式 HD 接入

- 命中逆向／spec 閘門、IDA 9.4、正常玩家路徑與文件職責路由，沿用固定 GUI image、IDA image、dosgolem 副本與既有研究目錄。
- 查證 END-2 的原版四平面左移與 400×150 顯示，新增有界原始指令來源探針。先發現 word RCL 缺口，依 Intel SDM 補 16-bit register count-one 形式；其他未授權形式仍拒絕。未修改外部 dosgolem、EXE、DAT 或正式 IDA 資料庫。
- 初始完整來源與最多 240 次連續位移接入 host HD；來源、指標、caller、自然返回、原生目的及色盤皆須通過，缺圖回原版。既有美術及提示詞沒有新增或改寫。
- 四套件通過，106 項來源凍結；原始指令 241 狀態、四組 CPU 平面與共同呈現／負向測試通過。600M 普通操作的完整 RAM 與十七欄等於 r23，三張實際讀檔視窗及兩次切換相同。詳見 [本輪驗收](workplace/hd-prototype/catalog/remaining/consumers/battle/r24/runtime/verification-summary.json)。
- r24 根保留初次 CPU 缺口，ready/ 保存來源與透明 fixture 錯誤，runtime/ 保存修正後乾淨建置及回歸。準備工具的 git 缺件與反向 patch 操作失敗另在 RESEARCH-LOG 分類，不寫成遊戲缺陷。
- 原始 ZIP／十二檔完整性及 UID/GID 1000:1000 通過，沒有 root-owned 或 .md 目錄。正式 DX5.i64 的 SHA 保持 8afa40d3eaeed4851b78ed2ddc80ccbc1c65e14cb2481a09316d42670efce02e。全部本批容器已移除，沿用既有 image，沒有建立重複工具鏈。
- 正常結局 GUI、其餘動畫抽測、桌面音訊與三平台打包仍待完成，全遊戲 HD 維持未完成。此輪自製程式／測試／文件同步至既有 private repo；提交、遠端素材樹與 Issue 留言結果保存於 [同步收據](workplace/github-setup/hd-ending-r24-sync.json)，留言本文在 [本機草稿](workplace/github-setup/hd-ending-r24.md)。原版來源及研究收據只留本機。

## 2026-10-16：使用者授權的受控結局 HD 驗收

使用者允許調整正常結局驗收。本輪沿逆向、spec 閘門、IDA、試玩與文件職責路由，改以隔離測試入口執行原版結局，所有收據標明受控狀態及非正常通關。正式啟動器、玩法、存檔與下載基準不變。

- 準備工具保存 106 項來源及唯一 EIP 轉移，40M 入口前後完整 RAM 相同。原版結局自行載圖、繪圖、左移、顯示文字及等候；最後送入固定普通左鍵，由原版自然退出，沒有強迫結束或改寫客體記憶體。
- r25 初版保留 32 筆內部等待配額停止；ready/ 改為有界 caller 檢查點。實際視窗暴露 END-11／11B 專用色盤與 END-4..10 左上方目的未支援，分別查清原版色盤、七個 caller、完整四平面及兩個特殊四目的呼叫，再補 READY 與實作。r26 色盤版本四套件通過，r27 合併版本四套件通過，研究入口 101.821 秒；未新增 CPU 指令。
- r27 受控三案在 27 個完整檢查點及最終 RAM 相同，均在 451009120 步退出碼 0。十二個結局來源、241 個左移返回、八格天空、兩案共 132 張真實 X11、88 次狀態不變切換、還原及個別缺 END-2 通過。獨立核對核准素材與完整輸出像素，文字、地名及未知末列保留。結果為 [受控驗收總表](workplace/hd-prototype/catalog/remaining/consumers/battle/r27/controlled/verification-summary.json)。
- r27 控制組首次輸出掛載錯誤，以及 GUI 初次漏複製驗證器輔助來源，皆在啟動前或工具階段停止。修正隔離環境後以同一建置乾淨重跑，不列產品缺陷。早期色盤、目的及驗證器失敗保持原批次，不覆寫為通過。
- 唯一覆蓋清單重生，受控十二項與原有正常抽樣分開。20 項動畫／介面中八項正常、四項受控，下一批為 ANIME01..07 與 OP6。全遊戲 HD、桌面聽覺及三平台封包未完成，正常通關保留證據限制。
- 原始 ZIP／十二檔、唯讀與 UID/GID 1000:1000 核對通過。106 項來源與輔助驗證工具符合凍結副本，正式 IDA 雜湊不變，沒有 root-owned 或 .md 目錄。沿用既有 image，本批容器完成後移除；未建立新的工具鏈或生成美術。807 張 PNG、163 份提示詞及全部 1050 份私人素材不變。

本輪提交與推送自製程式、測試及文件；遠端核對、素材子樹與 Issue #5 結果保存於 workplace/github-setup/hd-controlled-ending-r27-sync.json，留言本文為 workplace/github-setup/hd-controlled-ending-r27.md。原版資料、執行產物與受控畫面只留本機。

## 2026-10-16：剩餘動畫、OP6 與既定 HD 接入驗收完成

使用者取消正常通關驗收。本輪依逆向、spec、實際視窗及文件職責路由，重用 r27 生產繪圖程式，增加隔離的原版動畫／OP6 驗收工具；沒有修改遊戲機制或生產 106 項編譯輸入。

- 查既有 IDA 9.4 原始指令與名稱表，按原版三個欄位選取八張動畫的分支。原版執行載圖、共同矩形繪圖、等待及背景還原；OP6 用明示的隔離 loader 參數執行原版完整繪圖。每筆 EIP、dword 及 stack 變更記入受控 manifest。
- r28 選在 40M 淡入中，色盤亮度上限 142，HD 守衛正確退回原圖。r29 改用原版 80M 穩定色盤；首次桌面受到一筆自動滑鼠位置影響，補空輸入隔離後重跑。之後發現原版標題選單的 WINDOW 仍覆蓋非正常戰場測試。全部失敗與來源保持原批次，不放寬 RAM、色盤或 WINDOW 守衛。
- r30 先由原版重畫 OP5 背景清除選單圖層，再執行動畫。三案在 102894281 步完成，55 個檢查點與最終完整 RAM 及原版欄位相同。45 個原版畫格、八次還原、OP6 全幅、108 張實際 X11、72 次切換、原版／HD／還原與缺 ANIME03／OP6 通過。
- 獨立核對首版漏算原版游標，OP6 的差異只在左上 32×32，共 716 輸出像素。保留首版工具及錯誤，補獨立 AND／XOR／DAC 游標公式後沿同一收據核對全部像素，沒有排除矩形或修改產品。
- 覆蓋清單重生並通過 HD 接入門檻；20 項動畫／介面八項正常、十二項受控，其餘已盤點故事、頭像、地面、單位、物件及介面沿既有共同呈現與抽樣。全美術接入及既定驗收完成，接續三平台封包。正常通關不列門檻，未逐張正常試玩的素材、原版字型／文字及未知像素保留證據限制。
- 原始 ZIP／十二檔、唯讀與 UID/GID 1000:1000 通過，正式 IDA 與 807 張生成 PNG／163 份提示詞不變。沿用既有 image，本批容器完成後移除。Windows／macOS 的固定版本 Go 依賴預查可使用無 CGO 建置；這只是工具鏈預查，不代表平台啟動驗收。

本批自製工具及文件同步至 private repo。提交、遠端素材樹與 Issue #5 結果保存於 workplace/github-setup/hd-art-r30-sync.json；原版及受控執行產物不加入 Git。封包目錄、完整版號與本機權利邊界已讀取對應知識路由，尚未建立正式交付包。

## 2026-10-16：獨立桌面路徑、持續存檔與封包工具

沿既定桌面與三平台打包範圍，將研究路徑轉接到執行檔旁的資源，以及平台使用者資料目錄。重用原有檔案 overlay 保存正常遊戲寫入，原版資料保持唯讀；正式遊玩關閉大型 RAM／PCM 診斷，明示驗證旗標仍可保存收據。CD PCM 加入有界主機播放佇列，不回寫原版狀態。

- r1 保留測試誤把讀取介面當成寫入介面的編譯錯誤。r2 四套件及 Linux 桌面建置通過；r3 補 Windows 分隔符，PCM 競態與 Wine 中文路徑／重開工作副本通過。初次 Wine 未設定 UTF-8 locale，Unicode mkdir 失敗，改隔離環境後同一 exe 通過。
- r4 修正具體暫存檔路徑經 helper 再次轉換的問題，四套件通過，研究入口 217.619 秒。外層 240 秒限制在測試完成後中斷桌面建置，沿同一來源單獨重建成功；未重跑已通過的整套測試。Windows 測試亦通過。
- 新版本 109 項輸入與實際載入資源清單一致；原版 20M 完整 RAM 等於 r27，SHA 49545cf899202fc33c5383a82906fb466ed329857ac8c1a29caa8c66acb61bc6。新增封包、裝置音訊及真正 ZIP 驗證工具，採固定 Go／SDK，含原版資料只進本機 full-local。RRSAL-1.0 依固定範本建立。
- 裝置驗證初次輸入檔位於自訂 state 根之外，工具路徑被轉接而停止；第二次的 20M 尚未走到原版選曲，桌面速度 1 亦在外層配額前未完成。保留兩批工具失敗，不列音訊產品缺陷或已驗證。第三次沿既有正常輸入至 150M，第一筆 PLAY 的既有證據為 136777435，採已有的速度 60 驗證模式，結果待回填。

本節為實作與測試 checkpoint，裝置播放、四個平台／架構 ZIP 及 macOS 真機仍分開記錄。正常通關依使用者決定排除驗收，不將取消登記為通過。

## 2026-10-16：桌面音訊與三平台本機打包完成

- 桌面音訊第三次沿正常輸入至 150M 完成，實際 PulseAudio loopback 非靜音；開／關裝置的完整 RAM、十四欄與原版 PCM 相同。人耳未驗證，加速模式的丟框／補靜音計數與主機近似分類保存於 r4/audio-device/verification-summary.json。
- 首批完成 Linux／Windows，但 macOS 的無 CGO 建置缺 GLFW／Metal 綁定。保留 failed-delivery 及原始錯誤，從相同 109 項來源重建現行版；四套件無變更，沿用逐項相同的 r4 收據。SDK image 缺 Git、Wine image 缺 GUI 驗證套件，補成既有 SDK 的版本化 revision，來源與套件清單保存。
- v.0.1.0-20261016 四包完成，程式來源 d8f4a772f7b5a5736885cc8f312ecf0d57a70f4a。Windows／Linux amd64 與 macOS amd64／arm64 各約 528 MB；逐檔 SHA、版本、原版五檔、架構、macOS 12.0 目標、LICENSE、十份第三方聲明及 UTF-8／BOM 通過。現行入口 dist-all/v.0.1.0-20261016/，完整 SHA 保存 SHA256SUMS.json，含原版資料不加入 Git 或 Release。
- Linux 與 Wine 真正 ZIP 各驗證 MARK 模式選擇、15989208 步自然退出、正常標題 82M、功能列雙向切換、三張 X11 與正常關窗。包內所有來源與原版 bytes 不變；macOS 編譯／格式核對通過，真機、分發簽章、人耳及預設速度的實體桌面效能未驗證。
- 原始 ZIP／十二檔、UID/GID 與唯讀檢查通過。只修復原版控制輸出裡由 dockerd 建立的單一 root-owned 空 mountpoint，未遞迴 chown。807 張生成 PNG／163 份提示詞與私人素材子樹不變，程式、文件與工具鏈來源保存 GitHub。正常通關排除驗收；四語仍由 #4 接續，#6 保留共同與真機驗收限制。

本批最終同步收據為 workplace/github-setup/desktop-package-v010-sync.json，Issue 本文與遠端狀態一同保存；版本總表為 dist-all/v.0.1.0-20261016/smoke/verification-summary.json。本輪收尾檢查專案容器並移除已完成的工作；保留現行專案工具 image 及其他專案既有 image。

## 2026-10-16：三平台完整版重查、推廣影片與故事背景

依使用者要求交付三平台完整版、錄製推廣影片並撰寫 README 故事背景。命中並載入推廣影片、原版素材來源、README、文件職責與 dist-all 路由，沿用既有 GUI image，沒有新增遊戲機制、生成配樂或重新開啟正常通關驗收。

- 四個 v.0.1.0-20261016 ZIP 重新逐檔核對通過，收據在 smoke/archives-verification-promo.json。109 項編譯輸入與 r4 相同，原版來源及四包 SHA 不變；無須重建相同二進位。Linux 與 Wine 的既有實際啟動仍有效，macOS／Windows 真機、人耳與預設速度效能未驗證。
- 從正式 Linux ZIP 的唯讀解包目錄錄製真正 X11。正常 MARK 自然退出後進入 DX5，沿 390 筆普通滑鼠輸入至首戰，F6 與頂端功能列切換的原版 proof 不變，最後完成 760M 預算。全部包內來源不變，錄影與收據保存在 workplace/promo/v.0.1.0-20261016/。
- 首次錄影已正常封存 MKV，但工具誤要求 FFmpeg 接 SIGINT 後只能退出 0，實際為預期的 255。保留原工具及錄影，新增 --finalize-existing 獨立核對 ffprobe、時間表、完整遊戲收據、三次狀態 proof 與逐檔 SHA 後通過，未重錄或修改遊戲。後續錄製工具改用 stdin q 正常封存。
- 影片配樂直接使用原版 CD 第十軌，CUE 對應 BIN file sectors 51365..58176、2352 bytes／sector、44.1 kHz／16-bit／雙聲道。PCM SHA d0aad76dbf41c674928428264e0ae932c84e0ff08c97540982b7c6f5caad74b2；只調音量與首尾淡入淡出。字幕使用 Noto Serif CJK TC，兩份字型 SHA 與授權保存；沒有使用主機 runtime。
- 首版黑畫面判定抓到 MARK 原生半秒漸入。完整幀與來源確認後，只將該剪輯起點分類為原生效果，其他黑畫面仍拒絕；首版檢查與工具保存 edit/verify-attempt01/。首版成片的戰場中點仍在載入徽記，保留至 edit/film-v1/，改用同一錄影已完成的 HD 戰場暫停展示。其他十段沿用已核對 SHA 與參數的片段，最終成片重新完整驗證。
- 最終成片為 72 秒、1920×1080、30 fps／2160 幀，H.264／AAC 雙聲道，大小 9205707 bytes。影片 SHA cc0b5fe0a28bc423d01f54365ecc20288caf997f27dbf7e1c6ac9376c5e7b0db。完整解碼、25 項字幕邊界、音量與靜音、實際內容區黑幀／停格分類通過；平均 -14.0 dB、峰值 -2.2 dB，無一秒以上靜音。Codex 檢視十一段總覽與完整 HD 戰場，另檢視 MARK、字幕及功能列，未宣稱人耳確認。交付入口 dist-all/v.0.1.0-20261016/promo/，內含影片、海報、SRT、總覽、來源權利、驗證及獨立 SHA256SUMS.json，不更動根目錄四包雜湊清單。
- README 補上三方原到關原與大阪之陣的故事背景，章節依原版 DX5.EXE 的 CP950 文字，史實依濱松市博物館、日本國立公文書館及岡崎市觀光協會。原版全文只留本機，檔案偏移、工具與雜湊寫入 RESEARCH-LOG。修正 CONTEXT 中較早結局列及任務段落仍稱全遊戲 HD 未完成的過時描述，歷史研究紀錄保留。
- 原始 ZIP／十二檔、唯讀及 UID/GID 1000:1000 再核對通過，沒有 root-owned 或 .md 目錄。所有本批 Docker 容器完成後移除，沒有重建 image，保留現行工具鏈與既有 IDA image。生成美術 1050 份檔案的素材樹 9de9713108cf662e0b2bb39e46b5ab7aec8c1561 不變。

自製錄製／合成／驗證工具、README 與文件同步至既有 private repo，Issue #6 登記本機交付及驗證範圍，#4／#6 保持開啟。同步結果保存於 workplace/github-setup/promo-v010-sync.json，留言本文為 workplace/github-setup/promo-v010-issue.txt。原版來源、研究輸出、四包與影片均排除版控；未建立或移動 tag／Release。

## 2026-10-17：README 遊戲沿革、家康歷史與故事推廣背景

依使用者要求擴充 README，分為遊戲沿革、德川家康與日本戰國、遊戲故事背景。命中並載入 README 標準及文件職責入口，沿用既有原版章節證據與 GUI Docker image；沒有新增文件目錄、遊戲機制或美術。

- 遊戲沿革補入 MAME PC-98 目錄的日本版記錄：Family Soft、1992 年 12 月 10 日、五張磁片。目錄固定至 commit eb947cba9b909d29d4f9fb2851e2ee69789d0be0，全文 SHA 與擷取欄位保存 workplace/promo/v.0.1.0-20261016/readme-history-catalog.json。繁中光碟版的 1997 年明示為民間《遊戲總表》的記載，沒有用 MARK 版權年份推算發售日。舊 Family Soft 官方頁 DNS 失敗，保留存取結果，改用已讀取的 MAME 目錄並載明來源限制。
- 家康歷史從三河與人質時代、桶狹間後的獨立、織田同盟與濱松，推進到三方原、長篠、秀吉與關東轉封、關原、江戶幕府、大阪冬夏之陣。各段連結岡崎市觀光協會、濱松市及日本國立公文書館。遊戲故事依原版十二章整理為五組年代，保留原版章名，史實背景與遊戲條件分開表述。來源與推論界限追加於 RESEARCH-LOG 的遊戲沿革與歷史背景來源補充，不重寫先前研究紀錄。
- Docker 內檢查 README 標題、十七個本機連結與錨點，十二個章節標頭仍存在原始 EXE，八個代表篇名與章節表相符。MAME 原始 XML、固定 commit 與 metadata 相同，109 項編譯輸入不變；來源、連結與衛生驗證通過，收據保存 workplace/promo/v.0.1.0-20261016/readme-history-verification.json。純文件修改不重建四包、不重跑遊戲驗收。
- 同步先前使用者指示關閉 Issues 的現況。主機 gh 重新確認六筆皆為 CLOSED，修正 CONTEXT 與 WORKLIST 仍稱 #4／#6 開啟的過時欄位；未重開、留言或新建 Issue。四語、真機與人耳限制仍保持未完成或未驗證。
- 檔案與輸出 UID/GID 1000:1000，沒有 root-owned 或 .md 目錄。全部本批 Docker 容器完成後移除，沒有新建 image。生成美術樹 9de9713108cf662e0b2bb39e46b5ab7aec8c1561 不變，來源快照與研究輸出只留本機。

本輪五份文件同步至既有私人 GitHub，提交與遠端核對保存 workplace/github-setup/readme-history-sync.json。四個封包、推廣影片、原版來源與生成美術保持既有版本；未建立或修改 tag／Release。

## 2026-10-17：依使用者授權公開 GitHub 儲存庫

使用者指示「github repo 改成 public 囉」。主機 gh 確認變更前仍為 PRIVATE，依既有授權執行公開；命中並載入 RRSAL 授權邊界與文件職責入口，沿用既有條款，沒有另改授權。

- 公開前在既有 GUI Docker image 核對全部 Git 歷史。1754 個 blob、561346067 bytes 沒有 original/、workplace/、dist-all/ 或原版二進位／音訊等路徑，沒有匹配原始 ZIP／十二個來源檔的完整 SHA-256，也沒有憑證模式或內嵌媒體模式命中。當前 1281 個版控檔案符合原版排除範圍；收據在 workplace/github-setup/public-history-audit.json。這是實際路徑、來源雜湊與模式檢查，不宣稱能證明所有未知敏感資料不存在。
- 主機 gh 核對沒有 GitHub Actions artifacts，也沒有既有 Release。含原版的四包及推廣影片維持本機，不隨可見度修改上傳。
- README、CONTEXT、WORKLIST 與 AGENTS 改用公開儲存庫的現況。HD 素材 README、索引的 classification／rights 與保存工具的對應文案同步更新；原始保存授權日期維持 2026-10-14，rights 記錄 2026-10-17 的公開授權。圖片、完整提示詞、逐檔來源及裁切紀錄不變，索引格式不變。
- tools/archive-hd-art.py --verify-only 通過，807 張 PNG、163 份提示詞及所有素材記錄完全相同，資料總量 596522181 bytes。保存工具 SHA 為 cac72c3489df492161eab94e2d37e2d9e0b72c8c2e7334f0cf141328ef2171d2，索引已同步。收據為 workplace/github-setup/public-art-verification.json；素材樹只因 README 與索引 metadata 改變。
- 先將說明同步至 main，再以主機 gh repo edit 的 public 可見度設定公開，最後重查 GitHub。六筆 Issues 保持關閉，未重開或新建追蹤單；原版、封包、遊戲機制、版本、tag 及 Release 不改。
- 原始資料與研究輸入唯讀，寫入限定素材索引及同步輸出，UID/GID 1000:1000。沒有 root-owned 或 .md 目錄；全部本批 Docker 容器完成後移除，沒有建立新 image。

可見度、提交、遠端與匿名讀取核對保存於 workplace/github-setup/public-repository-sync.json。公開網址為 https://github.com/wicanr2/tokugawa-dosgolem；自製內容依 LICENSE，原版資料仍不在儲存庫中。

## 2026-10-17：正式版 1.0.0 的來源凍結與 README 截圖

使用者指定正式版 1.0.0、清理工作區，並要求先確認最終 HD 已保存 GitHub，後續追加 README 原版／HD 截圖。沿版號、dist-all、文件職責與收尾路由，正式版採 v.1.0.0-20261017，保留既有測試範圍與原版資料邊界。正常通關維持排除，不新增四語或真機已完成聲明。

- 清理前先執行 archive-hd-art.py --verify-only，並以 GitHub 完整 HD 素材樹逐檔核對 Git blob。1050 份檔案、807 張 PNG、163 份提示詞及所有最終修正版完全相同，遠端樹為 cd330c4194005578692a461752a927e61c7c2679。收據為 workplace/github-setup/final-hd-v100-verification.json。
- 使用正常 Linux v.0.1.0 錄影的 731M 首戰，原版／HD／還原三張圖的完整原版 proof 相同。只裁去畫布下方非遊戲區，兩張 1280×800 PNG 像素不改，保存 assets/screenshots/，README 並列展示並載明來源版本。使用者授權的公開例外只限這兩張介紹截圖，完整錄影、其他原圖、音訊及原版檔不入 Git。證據保存在 workplace/github-setup/readme-v100-screenshots.json。
- README 改用正式版號與四包連結；實際封包由 package-desktop.py 的 ldflags 寫入相同完整版本，沿 r4 已通過且逐項相同的 109 項輸入，採乾淨 build-v.1.0.0-20261017 工作目錄重建。預定交付與驗證入口 dist-all/v.1.0.0-20261017/，沿用 GUI 與既有 macOS／Wine 工具鏈。
- 新增 tools/clean-workspace.py，預設只列清單，--apply 僅刪除可重建編譯快取、不含遊戲存檔的 Wine 環境，以及完整 manifest／ZIP 與全部內容相同的封包副本。原版、HD、研究來源、IDA、收據、存檔、截圖、錄影與 ZIP 都保留。來源凍結後先完成建置與驗證，再執行清理。

本節記錄建置前的凍結來源；正式包驗收、清理大小與 tag／Release 結果在後續完成紀錄追加。公開 Release 本文沿用 workplace/github-setup/v100-release-notes.md，最終同步收據為 workplace/github-setup/v100-release-sync.json，兩者在交付時建立。

Linux 正式 ZIP 的 772 檔、MARK 15989208 步自然退出、正常標題 81M、功能列切換、三張 X11 與正常關窗通過。Windows 首次驗證在 Wine 冷啟動的 -version 超過工具 20 秒限制，尚未到遊戲測試；保留 smoke/windows-attempt01/。驗證工具只將第一次 Windows 版本查詢放寬到有界 90 秒，其他 X11 命令仍為 20 秒，再於同一 image／同一 ZIP 的乾淨輸出重跑。原版程式、109 項編譯輸入及封包沒有變更。

Windows 第二次已通過版本查詢，停在 MARK 模式選擇狀態檔寫出後、X11 可見視窗尚未就緒的查詢。重查正常玩家路徑路由，保留 smoke/windows-attempt02/。find_window 改為最多 30 秒觀測唯一可見遊戲視窗，就緒前不送輸入；零個或多個視窗都不猜選。正式程式與封包仍不改，沿同一 Wine image 與 ZIP 乾淨重跑。
