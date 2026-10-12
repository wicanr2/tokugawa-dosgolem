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
