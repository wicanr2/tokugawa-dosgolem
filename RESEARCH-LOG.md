# 德川家康 dosgolem 接入證據

## 2026-10-11：LE 入口與 DOS/4G 啟動契約

本作接入 dosgolem，執行原版程式。使用者已明確排除 remake。工作入口與文件索引見 [AGENTS.md](AGENTS.md)。

| 輸入與工具 | 固定值 |
|---|---|
| `DX5.EXE` SHA-256 | `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a` |
| dosgolem HEAD | `a9714ebdab2ad6b529f81225472680f2b11f2842` |
| Go | `go1.24.13 linux/amd64` |
| IDA Pro | 9.4，使用既有 `ida-pro-9.4-idapython:py312-v1` image |
| IDA 位址空間 | LE loader 線性 EA；每筆另存檔案偏移 |
| dosgolem 位址空間 | relocation 後的平坦線性記憶體；selector 是工具標籤 |

已證實：原始有界探針從 dosgolem 線性位址 `0x18DB4` 執行 11 步，在 `0x18E51` 的 `CD 21` 停止。當時 `AH=30h`、`EBX=50484152h`。收據：[entry-probe/receipt.json](workplace/entry-probe/receipt.json)。探針：[le-entry-probe.go](tools/le-entry-probe.go)。沒有執行遊戲畫面。

已證實：IDA 的線性 EA `0x18E51` 對應檔案偏移 `0x14651`。後續 EA `0x18EF0` 對應檔案偏移 `0x146F0`，是 `AX=FF00h`、`DX=0078h` 的 `INT 21h`。EA `0x18F0C` 是取得資料段基址的 `INT 31h AX=0006h`。原始名稱、bytes、函式邊界與交叉參照保留在 [IDA 匯出](workplace/ida/intake.json) 與 `workplace/ida/DX5.i64`，產生工具為 [ida-entry.py](tools/ida-entry.py)。

強推論：此段是 Watcom C/C++32 啟動碼。依據包含入口的原始文字、DOS/4G 偵測、PSP 環境 selector 讀取及環境中的程式路徑複製。公開參照為 [Open Watcom cstrt386.asm](https://github.com/open-watcom/open-watcom-v2/blob/b36230f83e0ad7c40fc682f2d2c10bc034efcf23/bld/clib/startup/a/cstrt386.asm)，固定 commit `b36230f83e0ad7c40fc682f2d2c10bc034efcf23`。原始碼本機快照為 `workplace/watcom-cstrt386.asm`，SHA-256 `75e8775834027651747c7e3d4c76ecdfc324ecdaec1c2a37631d0bb1262233f7`。此參照不證明編譯器小版號，也不證明整個 runtime 相同。

### 有界啟動實驗契約

狀態：READY，僅適用於隔離研究探針，未進正式執行入口。

- 使用原始 LE 與 relocation，讓程式自然執行，不套 FD2 的函式位址或攔截點。
- 依公開啟動契約，入口 `ES` 指向 PSP，`PSP:2Ch` 指向環境，`PSP:80h` 為空命令列。環境採 DOS 的雙 NUL、字串數量及 `C:\DX5.EXE` 結構。
- 主機選定 DOS 6.22 與平坦資料段。DOS/4G 查詢只回報相容主機存在。這是明示的測試環境，不聲稱重現實際 extender 的暫存器全文。
- PSP 與環境置於 LE objects 前已確認為空的區域。使用描述子表，禁止 FD2 專用的 `SegmentRead*` 閉包。
- DPMI 使用 dosgolem 現有通用服務。DOS 只轉接已列明的通用服務，FD2 啟動狀態機呼叫次數必須為零。
- 未支援的指令或服務立即停止，保存 bytes、暫存器與中斷收據。取得第一畫面之前不宣稱可玩。

探針：[startup-probe.go](tools/startup-probe.go)。實驗輸出寫入 `workplace/startup-probe/`。其結果與下一個阻塞點寫入本檔的後續紀錄。

### CPU 切片：16 位元 OR 暫存器

狀態：READY。已證實的需求 bytes 為 `66 09 CA`，dosgolem 線性位址 `0x18F12`，IDA 線性 EA 同為 `0x18F12`，檔案偏移由 IDA 匯出逐項記錄。探針自然執行 35 步後停在此處。

契約：`66 09 /r` 且 ModRM.mod=3，將兩個暫存器低 16 位作 OR，目的高 16 位不變；來源不變；CF、OF 清零，SF、ZF、PF 由 16 位結果決定，其他已定義旗標保留。AF 為架構未定義旗標，不新增對它的語意聲明。沿用 CPU 的 `setLogicFlags16`。其他 prefix 與未支援的記憶體形式仍拒絕，拒絕不發布結果與旗標。

實作差異保存於 [dosgolem.patch](tools/dosgolem.patch)，測試為 [cpu386-integration_test.go](tools/cpu386-integration_test.go)。只在 `workplace/engine/` 研究副本套用，再以 Go overlay 編譯，不改外部 dosgolem 工作樹。

### CPU 切片：ES 絕對位址的段暫存器 word 寫入

狀態：READY。OR16 修正後自然執行 43 步，在 dosgolem 線性位址 `0x18F4E` 的 `66 26 8C 1D 89 90 01 00` 停止。這是將 DS 的 16 位 selector 寫入 `ES:[00019089h]`，為啟動碼保存資料段。IDA 線性 EA `0x18F4E` 的原始資料流與公開 Watcom 的 `__saved_DS` 用法相符。

契約沿用 dosgolem `docs/spec/014-dos4gw-flat-descriptors.md` 的已確認 ES 絕對 word 寫入；補上同一形式帶 `66` prefix 的解碼。只寫兩個 bytes，旗標不變，完整檢查描述子與寫入權限。不能把此指令當成跳過啟動碼的理由。

### CPU 切片：MOV immediate 的冗餘 DS prefix

狀態：READY。ES word 寫入修正後執行 48 步，停在 dosgolem 線性位址 `0x18F6B` 的 `3E BA F0 B5 05 00`。此處是 `MOV EDX,0005B5F0h`，沒有記憶體運算元，DS prefix 不改變結果。

契約：辨認 `3E`，只對 `B8` 至 `BF` 的暫存器 immediate MOV 忽略此冗餘 prefix，保留原有 16／32 位運算元寬度與旗標不變語意。其他帶 DS prefix 的尚未確認形式繼續拒絕。重複 segment prefix 仍拒絕。這是 CPU 解碼能力，沒有遊戲位址條件。

### CPU 切片：SUB byte 暫存器

狀態：READY。DS prefix 修正後執行 63 步，停在 dosgolem 線性位址 `0x18F96` 的 `28 C0`，即 `SUB AL,AL`。契約為 `28 /r` 的 ModRM.mod=3 暫存器形式，支援八個 byte 暫存器，包括 AH 至 BH，正確保留其餘位元。旗標沿用已驗證的 `sub8`，不影響非算術旗標。其他 prefix 與記憶體形式仍拒絕。驗收涵蓋借位、正負溢位、零值與高 byte 寫入。

### CPU 切片：ES 絕對位址的段暫存器 word 讀取

狀態：READY。SUB8 修正後執行 70 步，停在 dosgolem 線性位址 `0x18F9E` 的 `26 66 8E 1D 11 45 03 00`。語意是 `MOV DS,ES:[00034511h]`，載入先前由 PSP 取得的環境 selector。

契約：既有 `8E /r` ES 絕對 word 讀取允許冗餘 `66` prefix。只讀兩個 bytes，檢查來源 descriptor 與完整界限，且必須通過既有 `canLoadSegment` 才發布目的段。拒絕不能改變目的段與旗標。未確認的其他 override 形式仍拒絕。

### CPU 切片：ES byte 比較與載入

狀態：READY。環境 selector 載入修正後自然執行 348 步，在 dosgolem 線性位址 `0x2CF3B` 的 `26 3A 10` 停止，即 `CMP DL,ES:[EAX]`。同一條原始碼路徑的鄰接 bytes 包含 `26 8A 08`，即 `MOV CL,ES:[EAX]`。

契約：`3A /r` 與 `8A /r` 的記憶體形式接受 ES override。沿用 32 位有效位址解碼，來源採 ES descriptor，byte 寬度不變。CMP 只改算術旗標，MOV 不改旗標；來源越界時不發布結果。byte 暫存器保留其他位元。REP 與不相容的運算元 prefix 仍拒絕。

### CPU 切片：PUSH／POP GS

狀態：READY。ES byte 修正後執行 360 步，停在 dosgolem 線性位址 `0x190DB` 的 `0F A8`，即 `PUSH GS`。契約沿用已實作的 FS 保存／還原，為 `0F A8`／`0F A9` 提供 32 位堆疊槽。selector 為 16 位；旗標不變，SS 描述子、ESP 上下界與 POP 的 selector 載入條件都必須檢查。拒絕不改變 ESP 與 GS。未支援的 prefix 仍拒絕。

### CPU 切片：SBB r/m32,r32 的暫存器形式

狀態：READY。GS 保存修正後執行 482 步，在成功呼叫 DPMI `AX=0501h` 配置記憶體之後，停在 dosgolem 線性位址 `0x2583E` 的 `19 C0`，即 `SBB EAX,EAX`。契約沿用既有 `1B /r` 的 32 位暫存器帶借位減法，`19 /r` 交換 ModRM 的目的與來源解讀。完整保留原有借位、AF、OF 修正與其他旗標語意。只接受暫存器形式，測試包含右值加借位超過 32 位的情形。

### CPU 切片：XCHG 暫存器

狀態：READY。SBB 修正後執行 615 步，停在 dosgolem 線性位址 `0x209FF` 的 `87 FA`，即 `XCHG EDX,EDI`。這是原生記憶體配置流程，沒有攔截 allocator。

契約：`87 /r` 的暫存器形式交換 32 位值；帶 `66` 時只交換低 16 位。旗標不變，同一暫存器與自身交換合法。既有記憶體形式維持原契約。測試涵蓋所有暫存器配對與 16／32 位寬度。

### CPU 切片：CMC

狀態：READY。XCHG 修正後執行 699 步，停在 dosgolem 線性位址 `0x208AC` 的 `F5`，即 `CMC`。契約只反轉 CF，其他旗標與暫存器不變；沿用既有 CLC／STC 的 prefix 拒絕條件。獨立測試遍歷低 12 位旗標的全部組合。

### CPU 切片：AND r/m32,r32 的暫存器形式

狀態：READY。CMC 修正後執行 701 步，停在 dosgolem 線性位址 `0x208AF` 的 `21 C8`，即 `AND EAX,ECX`。契約沿用 OR／XOR 的 32 位邏輯旗標，僅接受 ModRM.mod=3。來源不變，目的取得位元 AND；CF、OF 清零，SF、ZF、PF 依結果，其他已定義旗標不變。AF 不作語意聲明。

### CPU 切片：OR dword 記憶體與 sign-extended immediate8

狀態：READY。AND 修正後執行 742 步，停在 dosgolem 線性位址 `0x2092A` 的 `83 0E 01`，即 `OR dword ptr [ESI],1`。契約為 `83 /1` 的 32 位記憶體形式，沿用有效位址與段選擇規則，immediate8 先符號擴展為 32 位。完整讀寫四個 bytes；寫入成功後才更新邏輯旗標。越界或唯讀時不發布記憶體與旗標。16 位與其他 prefix 不擴張。

### DOS 取得時間與唯讀檔案接入

狀態：READY，研究主機環境。OR immediate 修正後執行 985 步，停在 dosgolem 線性位址 `0x2C070` 的 `INT 21h AH=2Ch`。鄰接原始碼重複讀取秒數，因此不能固定傳回同一時間。

契約：DOS `AH=2Ch` 回傳 CH 時、CL 分、DH 秒、DL 百分秒。起點固定午夜，使用 dosgolem `internal/machine/le_bios_clock.go` 的既有一指令一微秒近似，依成功指令數推進，不依查詢次數推進。此為 `hardware-spec approximation`，不宣稱實機速度相同。後續 BIOS／PIT 與 DOS 時間須共用此虛擬時間。

原版工作目錄的唯讀檔案透過 dosgolem `OpenDirectoryReadOnlyFiles` 接入，沿用既有大小寫處理、路徑邊界與 DOS 錯誤碼。光碟仍未掛入，不能把 C: 檔案服務等同 CD 支援。

已證實：兩百萬指令預算執行到原版的兩段秒數等待流程，沒有未支援指令，停止原因是預算。尾端位於 dosgolem 線性位址 `0x2C088`，原版仍在等待第二次秒數變化。依一微秒／指令的既定時鐘，下一次變化須在兩百萬步之後才能被查詢，因此探針預算增加到三百萬步，時鐘比例不變。

### CPU 切片：16 位元 SBB 暫存器

狀態：READY。三百萬步探針自然通過兩次秒數等待，執行 2,000,858 步後在 dosgolem 線性位址 `0x2A23D` 的 `66 19 C0` 停止，即 `SBB AX,AX`。先前的 DPMI 版本與區域鎖定查詢均已由通用服務處理。

契約：`66 19 /r` 與反向編碼 `66 1B /r` 的暫存器形式使用 16 位帶借位減法，目的高 16 位保留，來源不變。借位以 17 位計算，SF、ZF、PF、AF、OF 依 16 位結果與原始運算元決定；非算術旗標保留。未確認的記憶體與其他 prefix 形式仍拒絕。

### CPU 切片：TEST word 記憶體與 immediate16

狀態：READY。SBB16 修正後執行 2,008,147 步，停在 dosgolem 線性位址 `0x25870` 的 `66 F7 05 DA B1 05 00 F0 FF`，即 `TEST word ptr [0005B1DAh],FFF0h`。

契約：`66 F7 /0` 的記憶體形式只讀兩個 bytes，與 immediate16 作 AND，僅更新邏輯旗標。沿用既有暫存器形式及有效位址解碼，讀取不需寫入權限，越界不發布旗標。測試驗證讀取寬度、唯讀、正負結果與界限。

### CPU 切片：SAR dword 記憶體一位

狀態：READY。TEST word 修正後執行 2,008,205 步，停在 dosgolem 線性位址 `0x1025E` 的 `D1 7D F0`，即 `SAR dword ptr [EBP-10h],1`。此時原版已離開先前的 C runtime 初始化區域，尚未驗證遊戲畫面。

契約：`D1 /7` 記憶體形式依 32 位有號值算術右移一位；CF 取原 bit0，OF 清零，SF／ZF／PF 由結果決定，非算術旗標保留，AF 不作語意聲明。沿用有效位址選段，EBP 使用 SS。完整寫入成功後才發布旗標，失敗保持原值與旗標。

### CPU 切片：段暫存器寫入 ESP stack slot

狀態：READY。SAR 修正後執行 2,015,352 步，停在 dosgolem 線性位址 `0x26467` 的 `8C 5C 24 04`，即 `MOV word ptr [ESP+4],DS`。

契約：既有 `8C /r` 的 disp8 記憶體形式擴及 SIB 以 ESP 編碼的有效位址，依既有解碼器選 SS 或 DS，寫入兩個 bytes，旗標不變。保留已確認的 EBP stack local 行為，界限與唯讀拒絕同前。

### CPU 切片：從 ESP stack slot 載入段暫存器

狀態：READY。stack selector 保存修正後執行 2,015,596 步，停在 dosgolem 線性位址 `0x2628B` 的 `8E 44 24 14`，即 `MOV ES,word ptr [ESP+14h]`。契約沿用先前的 `8E /r` EBP stack local 載入，擴及 ESP 編碼的 SIB 有效位址。只讀兩個 bytes，經過來源界限與 `canLoadSegment` 驗證才更新目的段；旗標與 ESP 不變。

### CPU 切片：CS LEA 與間接段載入

狀態：READY。stack selector 載入修正後執行 2,016,477 步，停在 dosgolem 線性位址 `0x2AF1D` 的 `2E 8D 86 C0 AF 02 00`。LEA 只計算有效位址，不讀記憶體，也不加 CS base，因此冗餘 CS prefix 對結果沒有影響。只擴張已存在的 32 位 LEA，旗標不變，其他未支援 prefix 保持拒絕。

同一段原始 bytes 在後續位置包含 `8E 03`，即 `MOV ES,word ptr [EBX]`。段載入的簡單 base-register 間接形式沿用 word 讀取與 `canLoadSegment`，沒有寫入權限要求；來源越界或 selector 未登錄時不發布目的段與旗標。先不擴張尚未驗證的其他 SIB 或 disp32 形式。

### MSCDEX 與實際 CUE/BIN

狀態：READY，研究主機的掛載與查詢子集。原版自然執行 2,016,489 步，在 dosgolem 線性位址 `0x2B04D` 的 `INT 2Fh AX=1500h` 停止。

路由命中平台規格優先，已載入 `~/.codex/knowledge-base/local/retro-platform-spec-first.md`。MSCDEX 契約採 [Microsoft 的 MSCDEX 2.1 規格存檔](https://www.phatcode.net/articles.php?id=220)，全文快照為 `workplace/mscdex21.txt`，SHA-256 `4d19556cb36cd6b767030350c22ab37d4914d4e1f04898978d178aa16685e31d`。

已證實：BIN 的 MODE1/2352 第 16 sector 含有效 ISO 主描述子，2048-byte 邏輯區塊，volume 共 2,309 sectors，root 在 LBA 26，root 只有 `DX5` 目錄。CUE 共 16 軌。收據為 `workplace/disc-inventory.json`，原始 BIN 雜湊同 CONTEXT。

契約：先真實開啟並驗證 CUE、BIN 雜湊、軌道界限與 ISO 主描述子，才讓 `1500h` 回傳一個已掛載的 D:，BX=1、CX=3。`150Ch` 回報選定的相容主機版本 2.10，`150Dh` 透過呼叫端 ES:EBX 寫入真實掛載的 drive list。未支援的 request 不假成功。此切片不宣稱光碟檔案讀取或 CD 音訊已完成。

程式：[cdrom.go](tools/cdrom.go)。測試：[cdrom_test.go](tools/cdrom_test.go)，原版檔案缺失時跳過來源測試，不造替代原版。

### CPU 切片：POP dword 至 base+disp8 記憶體

狀態：READY。真實掛載的 `1500h` 查詢成功後，執行 2,016,501 步，停在 dosgolem 線性位址 `0x2AF04` 的 `8F 47 14`，即 `POP dword ptr [EDI+14h]`。

契約：`8F /0` 的 base+disp8、非 SIB 記憶體形式，從 SS:ESP 讀四個 bytes，寫入有效位址指定的目的段，再增加 ESP 四位元組。旗標不變；來源、目的或 ESP 溢位失敗時不發布 ESP 與目的值。不擴張涉及 ESP/SIB 的有效位址或 16 位形式。

### CPU 切片：段暫存器寫入簡單間接位址

狀態：READY。POP 修正後執行 2,016,509 步，停在 dosgolem 線性位址 `0x2AF14` 的 `66 8C 03`，即 `MOV word ptr [EBX],ES`。契約沿用已確認的段暫存器 word 寫入，擴及 ModRM.mod=0 的簡單 base register，排除絕對位址與 SIB 編碼。支援帶與不帶冗餘 `66`，寬度都為兩個 bytes；旗標不變，拒絕不發布目的 bytes。

### CPU 切片：ENTER 的零層巢狀 stack frame

狀態：READY。間接段保存修正後執行 2,016,628 步，停在 dosgolem 線性位址 `0x2B682` 的 `C8 00 00 00`，即 `ENTER 0,0`。

契約：32 位 `ENTER imm16,imm8`，巢狀層數低五位為零。把舊 EBP 壓入 SS stack，EBP 指向該槽，ESP 再減去 imm16 指定的本地大小。旗標不變，先驗證完整 descriptor 邊界、下溢與映射再發布；非零巢狀層數與其他 prefix 保持拒絕。

### CPU 切片：間接 byte 比較與 dword ROR

狀態：READY。ENTER 修正後執行 2,016,763 步，停在 dosgolem 線性位址 `0x2B444` 的 `38 10`，即 `CMP byte ptr [EAX],DL`。鄰接原始 bytes 包含 `C1 CA 08`，即 `ROR EDX,8`。

契約：`38 /r` 擴及簡單 base register 間接 byte 讀取，排除 SIB 與絕對位址編碼；CMP 不寫記憶體與暫存器，越界不發布旗標。`C1 /1` 暫存器形式旋轉 32 位值，count 取低五位，零時保留狀態，CF 取結果 bit31，count=1 時 OF 為結果最高兩位 XOR。其他旗標保留，count>1 的未定義 OF 選擇保留舊值，不視為硬體保證。

### DPMI 切片：實模式主機服務

狀態：READY，研究平台子集。原版執行 2,016,870 步，在 dosgolem 線性位址 `0x2B6F8` 呼叫 `INT 31h AX=0300h`，BL=2Fh、BH=0、CX=0。50-byte 呼叫封包的 AX=1510h，CX=3。既有 DPMI 只執行已登錄的 guest handler，未轉接頂層主機服務。

契約依 [DPMI 0.9 第 11 節](https://www.phatcode.net/res/262/files/dpmi09.html)。先驗證呼叫封包可讀寫，再嘗試既有 `RealModeInterrupt` 回呼。回呼若處理該服務，完整回填 EDI、ESI、EBP、EBX、EDX、ECX、EAX、ES、DS 與 FLAGS，保留 CS:IP／SS:SP 與保護模式暫存器。未知服務回到既有 guest handler 路徑。只接受原有 CX=0、BH=0、FS=GS=0 子集，不改變遊戲指令。

MSCDEX `1510h` 依 Microsoft 規格轉送真實光碟 request。首次 request 為 IOCTL INPUT，控制碼 10、長度 7，要求軌道範圍與 lead-out。TOC 位址必須由 CUE 的檔案索引、虛擬 PREGAP 與 Red Book 的 150-frame 原點推導，不能用 ISO volume 長度代替整張光碟。

TOC 公式交叉查證 [DOSBox Staging v0.82.2 的 AddTrack 與 GetAudioTracks](https://raw.githubusercontent.com/dosbox-staging/dosbox-staging/v0.82.2/src/dos/cdrom_image.cpp)。本作第二軌 file sector=2686，加 CUE 的 150-sector PREGAP 後 disc LBA=2836；再加 Red Book 原點，MSF=00:39:61。lead-out LBA=98158，與 ISO volume=2309 分開記錄。

首次查詢通過後，第二次控制碼 11 的 track 欄位實際為 26，超過已驗證的 16 軌。依 driver 契約回傳 Done+Error、Sector Not Found，不能改寫成有效軌號。追蹤顯示原版把 AX=0100h 的段號直接用作 flat pointer，造成相鄰查詢 buffer 重疊；目前只記錄現象，不改原版 bytes 或 DPMI 的段號回傳契約。

### 輸入切片：Mouse 查詢轉接

狀態：READY。光碟請求得到正確錯誤碼後，原版繼續至 2,032,313 步，在 dosgolem 線性位址 `0x11969` 呼叫 `INT 33h AX=0000h`。

沿用 dosgolem `internal/dos/bios.go` 的 Microsoft Mouse 狀態機與 `internal/dos/mouse_contract_test.go`，以獨立服務用 Machine 保存滑鼠狀態，不執行第二份遊戲。32 位呼叫只轉接已驗證的 reset、位置、範圍與 mickey 查詢，低 16 位回填、高 16 位及保護模式 FLAGS 保留。其他服務保持拒絕，尚未宣稱圖形游標或事件回呼完成。

### DOS 切片：DBCS 表

狀態：READY。滑鼠初始化後，在 dosgolem 線性位址 `0x27BD9` 透過 DPMI 呼叫實模式 `INT 21h AX=6300h`。測試主機沿用來源封裝的預設英文 DOS 環境，code page=437。依 [DOSBox Staging v0.82.2 的 AH=63h](https://raw.githubusercontent.com/dosbox-staging/dosbox-staging/v0.82.2/src/dos/dos.cpp) 回傳 DS:SI 指向零終止的空 DBCS 表、AL=0、CF=0。表置於已核對空白的 bootstrap 區 `0x8200`。這不改變遊戲自己的中文字模或文字解碼。

### CPU 切片：ES MOVZX byte 至 word

狀態：READY。DBCS 查詢後執行 2,032,508 步，在 dosgolem 線性位址 `0x1A165` 遇到 `66 26 0F B6 00`。沿用 ES 的 byte 來源讀取與有效位址解碼，帶 `66` 時將無號 byte 擴展至目的低 16 位，保留高 16 位與旗標。越界不發布結果。

### CPU 切片：ES MOV word 間接讀取

狀態：READY。MOVZX 修正後，在 dosgolem 線性位址 `0x1A190` 遇到 `66 26 8B 00`。既有 ES word 讀取從絕對位址擴至已驗證的 32 位有效位址解碼。來源只讀兩個 bytes，目的高 16 位與旗標保留，唯讀來源合法，越界不發布結果。

### BIOS 切片：目前顯示模式

狀態：READY。在 dosgolem 線性位址 `0x1A1AC` 呼叫 `INT 10h AH=0Fh`。沿用服務用 Machine 的初始 BDA 與既有 DOS BIOS 查詢，回傳目前 mode、columns 與 page，保留保護模式高 16 位暫存器。尚不宣稱畫面輸出完成。

輔助基準：`tools/aux-dosbox.sh` 在可寫工作副本中執行，已看到原版標題畫面 `workplace/dosbox-aux/25s.png`。初次 DAT 開啟失敗源於複製後仍為 0444，修正工作副本的 user-write 權限後消失。此圖只證明來源可在 DOSBox 啟動，不是 dosgolem 收據。

BIOS 查詢擴張 READY：原版接著在 `0x1A986` 呼叫 AX=1A00h。轉接既有 VGA 組合碼查詢，以及同組已實作的 AH=12h、BL=10h EGA 資訊查詢。只開放這些不帶遠端輸出 buffer 的子功能，不把其他 BIOS 呼叫默認成功。

### CPU 切片：XCHG accumulator 短編碼

狀態：READY。BIOS 查詢後執行 2,032,602 步，在 dosgolem 線性位址 `0x27C7F` 遇到 `91`，即 XCHG EAX,ECX。`91h` 至 `97h` 依編碼交換 accumulator 與指定暫存器，帶 `66` 只交換低 16 位，旗標不變。沿用 `87 /r` 的交換契約，無記憶體副作用。

### CPU 切片：CMPSB 與 repeat

狀態：READY。下一個停止點為 dosgolem 線性位址 `0x27C86` 的 `F3 A6`。`A6` 比較 DS:ESI 與 ES:EDI 的 byte，沿用 SUB8 旗標，按 DF 增減兩個指標。F3 在相等時繼續，F2 在不等時繼續，ECX 每次遞減；初始 ECX=0 不讀取且不改旗標。單次 Step 的 repeat 上限一百萬，超過先拒絕，未支援 prefix 仍拒絕。

### CPU 切片：SBB sign-extended imm8

狀態：READY。CMPSB 後執行 2,032,609 步，在 dosgolem 線性位址 `0x27C8C` 遇到 `83 D9 FF`，即 `SBB ECX,-1`。只擴張 `83 /3` 的 32 位暫存器形式，立即值先符號擴展，再連同輸入 CF 相減。CF 用 64 位算術判斷借位，避免 `FFFFFFFFh+1` 溢位；AF、OF 依原操作數與結果計算，SF、ZF、PF 沿用 SUB，其他旗標保留。不擴張記憶體或帶 prefix 的形式。

### CPU 切片：TEST word 暫存器

狀態：READY。SBB 修正後執行 2,032,675 步，停在 dosgolem 線性位址 `0x27D4C` 的 `66 85 C9`。擴張既有 `85 /r` 的 register-register 形式，帶 `66` 時只對低 16 位 AND 並更新邏輯旗標，不寫兩個操作數。拒絕未支援的段與 repeat prefix。

### BIOS 切片：未安裝的 Video7 擴充

狀態：READY。下一次停止在 dosgolem 線性位址 `0x27E7F` 的 `INT 10h AX=6F00h`，BX=0。這是 Video7 的安裝查詢；原版隨後比較 BX 與 `5637h`。測試主機只提供標準 VGA，沒有 Video7。沿用 dosgolem `internal/dos/bios.go` 未支援 BIOS 功能的記錄與保留暫存器契約，BX 保持零，不回傳安裝簽章。只開放這個無副作用查詢，不把其他 vendor 功能視為成功。Video7 查詢名稱交叉查證 [原廠 V7 VGA Technical Reference Manual，1988-06](https://www.minuszerodegrees.net/manuals/Video%20Seven/Video%20Seven%20-%20V7%20VGA%20chip%20-%20Technical%20reference%20manual%20-%20JUN88.pdf)。

同類查詢 READY：下一次在 `0x27EA4` 呼叫 `INT 10h AX=5F00h`，其後要求 AL=5Fh 才選用該卡。測試主機沒有這個擴充，沿用同一未安裝契約，AX 保持 5F00h，AL 不回傳成功簽章。功能號見 [CHIPS 69030 VGA BIOS OEM Reference Guide](https://www.versalogic.com/wp-content/themes/vsl-new/assets/resources/support/pdf/69030BG.pdf)。

### VGA 切片：標準暫存器與未安裝的廠商埠

狀態：READY。原版自然執行至 `0x27ED9`，讀取 3D5h、索引 30h，以高半 byte 偵測 S3。既有 LE 載入器沒有 PortIn。轉接 dosgolem 已有的 Machine/VGA 埠狀態，並在入口保留完整索引，避免 `VGA.Out` 的索引遮罩將擴充索引錯誤映射到標準暫存器。只實作標準 sequencer 00h–04h、CRTC 00h–18h、既有 GC/AC/DAC 與回掃埠；未安裝的 vendor 索引及 Tseng 3CDh 讀 FFh、忽略寫入，沿用 Machine 未接埠的 FFh 契約。未知的其他輸入仍停止。此切片尚無 VRAM 接線與畫面完成聲明。

### CPU 切片：word OUT

狀態：READY。執行 2,032,766 步停在 dosgolem 線性位址 `0x27F67` 的 `66 EF`，DX=3C4h、AX=1206h。`OUT DX,AX` 是一個 word I/O，交給新增的 `PortOut16` 平台回呼，不默認拆成任意兩個 byte 埠。VGA 的已知 indexed register pair 才按低 byte 索引、高 byte 資料處理。其他埠、32 位 OUT 與未支援 prefix 保持拒絕，暫存器與旗標不變。

### CPU 切片：LES m16:32

狀態：READY。執行 2,032,824 步，停在 dosgolem 線性位址 `0x28004` 的 `C4 5D FA`，即 `LES EBX,[EBP-6]`。沿用 32 位有效位址解碼讀取六個 bytes，低四個為目的 register 的 offset，後兩個為 ES selector。先確認來源完整可讀與 selector 可載入，再同時更新兩個目的。越界、無效 selector、register source 或未支援 prefix 都拒絕；旗標不變。

### VBE 切片：分頁式 256 色顯示

狀態：READY，研究平台子集。原版在 `0x27B45` 透過 DPMI 0300h 呼叫實模式 INT 10h AX=4F00h。契約採 [VESA VBE 2.0 rev1.1](https://www.phatcode.net/res/221/files/vbe20.pdf) 第 4 節與附錄 2 中相容 1.x 的格式。測試主機選擇 VBE 1.2 相容子集，分頁 VRAM 1 MiB，Window A 位於 A0000h、64 KiB 視窗及粒度，6-bit DAC。只列出已實作的 100h（640×400×8）與 101h（640×480×8）。4F00h 的 OEM 與 mode list 指標位於呼叫端 256-byte 輸出區內，並使用實模式 far pointer。

實作 4F00h／01h／02h／03h／05h，清除與保留 VRAM 的 bit15、分頁界限、ModeInfoBlock 與窗口存取一併測試。WinFuncPtr 為 NULL，依規格走 4F05h。未實作的 VBE 功能回傳失敗狀態 014Fh，不宣稱完整 VBE 相容。未知非 VBE BIOS 服務仍停止。CPU 對 A0000h–AFFFFh 的讀寫只在 VBE 模式生效時轉送真實分頁 VRAM，其他記憶體仍走 LE bus。尚未宣稱原版圖像已出現。

### CPU 切片：IMUL word

狀態：READY。VBE 控制器查詢通過後執行 2,033,032 步，停在 dosgolem 線性位址 `0x1A0C5` 的 `66 0F AF 15 90 8B 03 00`。擴張雙操作數 IMUL 的 16 位形式，以 signed word 相乘，保留目的高 16 位。結果無法以 signed word 表示時設 CF、OF，否則清除；其他未定義算術旗標選擇保留，不當作硬體保證。來源越界不發布結果與旗標。

### CPU 切片：CMP word 暫存器

狀態：READY。執行 2,033,036 步，停在 dosgolem 線性位址 `0x1A0D4` 的 `66 39 D0`。擴張既有 register-register CMP，只以低 16 位相減更新旗標，原操作數保持不變。段與 repeat prefix 保持拒絕。

### CPU 切片：ES MOV word 寫入

狀態：READY。執行 2,033,045 步，停在 dosgolem 線性位址 `0x1A0F0` 的 `66 26 89 1A`，即將 BX 寫入 ES:[EDX]。擴張既有 word store 的 ES override，沿用有效位址解碼與兩個 bytes 的 descriptor 寫入檢查。來源暫存器、旗標與相鄰 bytes 保持不變；唯讀或界限不足時拒絕。

### CPU 切片：ES MOVSX word

狀態：READY。執行 2,033,136 步，停在 dosgolem 線性位址 `0x1E117` 的 `26 0F BF 32`。讀 ES:[EDX] 的 signed word 並符號擴展至 ESI，沿用有效位址與 word 讀取界限，不改旗標；越界不發布目的。只擴張這一種來源寬度與 32 位目的。

### CPU 切片：CMP word 記憶體來源

狀態：READY。執行 2,033,153 步，停在 dosgolem 線性位址 `0x1E16D` 的 `66 3B 55 20`。沿用 `3B /r` 的有效位址解碼，帶 `66` 時讀取 word，以目的 register 的低 16 位相減更新旗標，不寫原操作數。越界先拒絕，不發布旗標。

### BIOS 切片：文字游標位置

狀態：READY。執行 2,045,526 步停在 dosgolem 線性位址 `0x298DA` 的 INT 10h AH=02h，BH=0、DX=0。依標準 BIOS 資料區契約，將 DL/DH 寫入 BDA 0040:0050 加 page×2 的位置，保留暫存器與旗標。對應 AH=03h 由同一位置與 0040:0060 shape 回傳，僅接受八個頁面。顯示查詢前同步原版寫入的 BDA 到服務用 Machine，避免讀到初始化時的舊副本。AH=02h 只有位置狀態，尚未宣稱畫出文字游標。BDA 佈局沿用 dosgolem `internal/machine/bda.go`。

同組 shape READY：下一次 `0x28215` 呼叫 AH=01h，CH/CL 指定文字游標的起訖掃描線與停用旗標。保存 CX 至 BDA 0040:0060，供 AH=03h 查詢；未宣稱光柵游標繪製完成。

### CPU 切片：OR byte register destination

狀態：READY。執行 2,046,942 步，停在 dosgolem 線性位址 `0x28882` 的 `08 E0`，即 `OR AL,AH`。只擴張 `08 /r` 的 register-register 形式，先讀取兩個低／高 byte 操作數，再寫入目的 byte 並更新邏輯旗標，保留 register 其他 bits。記憶體形式與 prefix 保持拒絕。

### DOS 檔案切片：原版 DAT 的讀寫開啟

狀態：READY。原版自然執行至 DOS exit=1，主控台顯示 `Cannot open DAT for GAME!`。已證實的呼叫為 `INT 21h AX=3D02h`，dosgolem 線性位址 `0x2888D` 為中斷後位置，DS:EDX 指向 `DX5.DAT`，回傳 AX=5。根因是研究入口只提供 `DirectoryReadOnlyFiles`，沒有 `WriteFileProvider`，與 DOSBox 輔助基準先前的工作副本權限問題相符。

沿用 dosgolem `internal/machine/overlay_files.go` 的 `DirectoryOverlayFiles`。每次探針在 `workplace/startup-probe/state-*` 建立自己的輸出區；第一次讀寫開啟才複製 DAT，後續讀取優先用副本。原始資料仍唯讀掛載。收據記錄副本路徑與前後 FLAGS，未增加新存檔格式、未宣稱存讀檔完成。

### CPU 切片：AL 的 ADD／SUB 立即值

狀態：READY。DAT 已成功以 handle=5 開啟並讀取 header 與 4,221 bytes。下一個停止點為 dosgolem 線性位址 `0x28D90` 的 `2C 61`，同段原始 bytes 包含 `04 41`。兩者為 AL 的 SUB／ADD imm8，沿用 byte 算術旗標，保留 EAX 的其他 24 bits。不擴張 prefix 或其他寬度。

### DOS 切片：查詢選項字元

狀態：READY。執行 2,056,074 步，在 dosgolem 線性位址 `0x29B45` 呼叫 INT 21h AX=3700h。沿用 dosgolem `internal/dos/dosinfo.go` 的 `switchChar` 查詢契約與預設 `/`，回 AL=0、DL='/'、CF=0，保留其他位元。只開放 query，不更改選項字元或擴張其他子功能。

### 鍵盤切片：DOS 狀態與 BIOS 緩衝

狀態：READY。執行 2,058,558 步，在 dosgolem 線性位址 `0x2BBC9` 呼叫 INT 21h AH=0Bh。使用 dosgolem `internal/machine/le_bios_keyboard.go` 已有的 BDA 環狀緩衝與 Enqueue／ReadEnhanced，DOS 狀態查詢回 AL=00h／FFh、CF=0，不消耗按鍵。對應 BIOS 16h 的讀取、peek 與 shift 查詢共用同一緩衝，不另造按鍵來源。空的阻塞式讀取記為等待輸入，不假成功。只接普通按鍵子集，不宣稱 IRQ1、自訂鍵盤 ISR 或所有 enhanced key 的相容性。

### DOS 切片：EXEC 找不到檔案

狀態：READY。接著在 dosgolem 線性位址 `0x2BBDB` 呼叫 INT 21h AX=4B00h，DS:EDX 指向 `.\COMMAND.COM`。記憶體中的 command tail 為 `/c MARK.EXE > nul`。下載包與目前寫入副本都沒有 COMMAND.COM，未曾執行子程式。

沿用 dosgolem `internal/dos/exec.go` 的檔案查找失敗契約，只對根目錄相對檔名或 `.\` 前綴，實際呼叫唯讀檔案提供者。確定得到 `fs.ErrNotExist` 才回 AX=2、CF=1。檔案存在、其他錯誤與未支援路徑仍停止，不假裝 EXEC 或 MARK 成功，不跳過遊戲呼叫。子程式載入與 COMMAND shell 尚未實作。

### CPU 切片：TEST dword 記憶體

狀態：READY。原版收到 EXEC 缺檔錯誤後繼續，在 dosgolem 線性位址 `0x26E1B` 遇到 `85 14 85 9C 77 03 00`，即 TEST [EAX*4+3779Ch],EDX。擴張 32 位 TEST 的記憶體來源，沿用既有有效位址解碼，讀四個 bytes 後更新 AND 邏輯旗標，不寫操作數。來源越界不發布旗標；word 記憶體形式仍未開放。

### CPU 切片：ES／FS AND byte immediate

狀態：READY。下一次停止在 dosgolem 線性位址 `0x26E69` 的 `26 80 20 CF`。同段原始 bytes 含 `64 80 20 CF`；前面的 PUSH ES／POP FS 已把同一 selector 放入 FS。只讓 `80 /4` 記憶體形式接受 ES／FS override，沿用 byte AND 與 descriptor 寫入檢查。加入 FS prefix 解碼，其他未列 FS 指令仍拒絕。越界或唯讀時不發布資料與旗標。

### CPU 切片：ES／FS OR byte immediate

狀態：READY。執行 2,059,717 步停在 dosgolem 線性位址 `0x26E7D` 的 `26 80 08 20`。同一分支的原始 bytes 含 `64 80 0A 30`。讓既有 `80 /1` 記憶體形式接受 ES／FS override，沿用 byte OR 與寫入界限檢查。FS 只接受已列出的記憶體 AND／OR；其他 group 與 register 形式仍拒絕。保留來源 register、相鄰 byte 與非算術旗標。

### VGA 切片：模式 12h 與平面記憶體

狀態：READY。執行 2,059,725 步，在 dosgolem 線性位址 `0x26E9A` 呼叫 INT 10h AX=0012h。沿用 dosgolem `internal/machine/bda.go` 的 SetVideoMode、`vga.go` 的四個平面與 latch，以及 `internal/dos/bios.go` AH=00h 的平台契約。同步實際 BDA，切換為 640×480、16 色，清除平面與重設 VGA 暫存器，停止 VBE 分頁映射。LE bus 的 A0000h–AFFFFh 轉接同一台服務 Machine 的 Read8／Write8。只開放已遇到的模式 12h；未知模式與保留畫面 bit7 尚不接受。測試模式切換、BDA、平面選擇、latch 與原 LE 記憶體未被覆蓋。

### CPU 切片：OR word 記憶體目的

狀態：READY。執行 2,060,333 步停在 dosgolem 線性位址 `0x1A337`，bytes 為 `66 09 05 C4 8B 03 00`。擴張既有 OR word 目的的記憶體形式，以有效位址解碼讀取 word，再與來源 register 低 16 位 OR。寫入成功後才發布邏輯旗標；唯讀或界限不足不寫資料、不改旗標。未支援的段與 repeat prefix 仍拒絕。

### CPU 切片：JNS rel8

狀態：READY。執行 2,060,421 步停在 dosgolem 線性位址 `0x2AD63` 的 `79 D1`。實作 JNS：SF=0 才將 signed rel8 加到下一指令位址，SF=1 繼續下一指令；旗標與 register 不變。不接受未支援 prefix。測試跳與不跳，以及正、負位移。

### CPU 切片：BSR、NEG byte 與 ROL CL

狀態：READY。下一次停在 dosgolem 線性位址 `0x2AD3B` 的 `0F BD C8`。同段已讀取的 bytes 包含 `F6 D9` 與 `D3 C0`。這次只補 register 形式：BSR dword 回傳最高有效 bit 的索引，零來源設 ZF 並選擇保留未定義的目的；NEG byte 以 0 減來源並更新算術旗標；ROL dword 的 count 取 CL&31，零次不變，非零只改 CF，次數一才定義 OF，其他未定義 OF 選擇保留。其餘 register 與旗標按指令契約保留，prefix 與記憶體形式仍拒絕。零來源 BSR 與多次 ROL 的保留選擇不當作硬體保證。

### CPU 切片：ADC dword register

狀態：READY。執行 2,060,443 步停在 dosgolem 線性位址 `0x2AC3C` 的 `13 C9`，同段 bytes 含 `83 D1 00`。只補 13h 的 register 來源與 83h /2 register 的符號擴展 imm8。加上輸入 CF，輸出 CF 用 64 位總和計算，AF、OF 以原操作數與結果計算，避免先合併立即值與 CF 造成溢位。記憶體、word 與 prefix 仍拒絕。

既有 `TestByteTESTRegister` 把 NEG DL 當成未支援指令。這次能力擴張後，將其拒絕案例改為仍未支援的 NOT DL，保留該負向測試；NEG 另有零、借位與 signed overflow 向量。原測試檔以固定 SHA-256 核對後才套用 overlay，未修改外部 dosgolem 工作樹。

### CPU 切片：ROL dword immediate

狀態：READY。執行 2,060,448 步停在 dosgolem 線性位址 `0x2AC49` 的 `C1 C0 08`。擴張既有 ROR immediate 的 register 轉向 group 0，語意沿用先前 ROL CL，只有 count 來源改為 imm8。零次與 count=32 不改旗標，其餘保留限制相同。

### CPU 切片：JNO rel8

狀態：READY。執行 2,060,453 步停在 dosgolem 線性位址 `0x2AC56` 的 `71 08`。沿用 rel8 分支框架，只在 OF=0 時跳轉，其他狀態不變。

### 原生畫面收據

狀態：READY。每次有界探針停止時，由同次執行的 VGA planes 或 VBE VRAM，加上原版透過埠設定的 DAC 產生 PNG。VGA 尺寸與色號映射沿用 dosgolem VGA.Pixels／DACIndex；VBE 依已實作的模式尺寸。以 SHA-256 命名並記錄寬高、顏色數、非黑像素數與 RGBA 雜湊。畫面存在或非黑不代表標題、主選單或可玩性通過；必須實際目視與輸入驗證，不使用 DOSBox 圖片代替。

### CPU 切片：RCR dword,1

狀態：READY。執行 2,060,461 步停在 dosgolem 線性位址 `0x2AC6D` 的 `D1 DA`，同段含 `D1 D8`。擴張 D1h /3 的 register 形式，舊 CF 移入 bit31，舊 bit0 移入 CF，OF 為結果最高兩位 XOR。只改 CF／OF，其他 flags 與 register 保留。未開放記憶體、word 或其他 count。

### CPU 切片：JS 與 ADC accumulator

狀態：READY。執行 2,060,472 步停在 dosgolem 線性位址 `0x2AC80` 的 `78 04`，同段 bytes 含 `15 00 00 00 00`。JS rel8 在 SF=1 時跳轉，其餘不變。15h 讀 imm32 並沿用已測 ADC32，目的為 EAX。未開放 word 與 prefix。

### BIOS 切片：單筆調色盤

狀態：READY。執行 2,060,811 步在 dosgolem 線性位址 `0x1078F` 呼叫 INT 10h AX=1010h，BX=16、CX=DX=0。緊接原始 bytes 含 AX=1001h 的設定。沿用 dosgolem `internal/dos/bios.go` 的 scalar palette 契約，開放 1010h 單筆 DAC、1001h overscan，以及同組 1000h 單筆 attribute palette。全部只傳 register，不借用 shadow Machine 的記憶體指標；高 16 位與 PM flags 保留。需要 ES 指標的 block 功能仍拒絕，待獨立 LE 轉接。

### CPU 切片：LFS 有效位址

狀態：READY。執行 2,061,083 步停在 dosgolem 線性位址 `0x279A7` 的 `0F B4 75 F6`，來源為 SS:[EBP-10] 的 m16:32。將既有 LFS 的絕對 DS 位址形式擴為有效位址解碼，仍先完整讀取 offset／selector 並驗證 selector，再發布目的 register 與 FS。不變更 flags；來源越界或 selector 未登錄時目的保持不變。測試採非零 SS base 與不完整 selector 邊界。

### CPU 切片：ES MOV byte 寫入

狀態：READY。執行 2,061,098 步停在 dosgolem 線性位址 `0x279CA` 的 `26 88 4C 06 02`，目的為 ES:[ESI+EAX+2]。在既有 byte store 的有效位址解碼後改選 ES descriptor。只寫一個 byte，不改來源 register、相鄰 byte 與 flags；唯讀或越界拒絕。段覆寫的 register 目的與未支援 prefix 保持拒絕。

### BIOS 切片：區塊 DAC

狀態：READY。執行 2,061,512 步，在 dosgolem 線性位址 `0x27B45` 透過 DPMI 0300h 呼叫實模式 INT 10h AX=1012h，BX=0、CX=16、ES:DX=5CEF:0000。沿用 dosgolem `internal/dos/bios.go` 的 1012h 契約：每個色號三個 6-bit 分量，來源由真實 guest 記憶體讀取。先驗證整段與 palette range，全部讀取成功才更新 DAC，不發布部分色盤。實模式使用 ES×16 的 descriptor，保護模式使用真實 ES selector；保留 register 與 flags。未實作的 palette 功能不回成功。

### CPU 切片：byte shift CL

狀態：READY。執行 2,061,735 步停在 dosgolem 線性位址 `0x1F8E5` 的 `D2 E4`，即 SHL AH,CL。沿用既有 D0h／C0h 的 byte register SHL／SHR，D2h 只將 count 來源改為 CL&31。零次保持不變；一次定義 OF，多次保留未定義 OF。旗標、低／高 byte 與其他 bits 的界限沿用原實作。count≥8 時 CF 未定義，不以測試選定值宣稱硬體保證。未開放記憶體或其他 group。

### BIOS 切片：讀取 attribute palette

狀態：READY。執行 2,064,696 步，在 dosgolem 線性位址 `0x1BBE4` 呼叫 INT 10h AX=1009h。沿用既有 BIOS 契約，將 16 個 attribute palette 值與 overscan 共 17 bytes 寫到真實 ES:EDX。先檢查整段目的 descriptor 的寫入界限；失敗時不發布部分資料。register 與 flags 不變，不使用 shadow Machine 的記憶體副本。

### BIOS 切片：寫入 attribute palette

狀態：READY。執行 2,064,750 步，在 dosgolem 線性位址 `0x1BC03` 呼叫 AX=1002h。原版先用 1009h 取得 17 bytes，再將前 16 個值填為 16–31。1002h 從相同的真實 ES:EDX 讀取 17 bytes，全數讀取後才設定 16 個 attribute palette 與 overscan，每筆遮罩為 6 bits。來源不完整不發布任何 palette 變化。契約沿用 `internal/dos/bios.go`。

### CPU 切片：CS 間接 CALL

狀態：READY。執行 2,070,766 步停在 dosgolem 線性位址 `0x2AE05` 的 `2E FF 13`。原版已從 DAT 讀取 928 bytes，接著經 CS:[EBX] 的表格呼叫處理常式。擴張既有 FFh /2 memory CALL 的 CS override，先讀取目標，再把下一指令 offset 寫入 SS 堆疊，成功才更新 ESP／EIP。來源越界或堆疊唯讀時不轉移。未開放 CS register CALL、far CALL 或其他 prefix。

### 滑鼠切片：隱藏游標

狀態：READY。執行 2,087,547 步在 dosgolem 線性位址 `0x11929` 呼叫 INT 33h AX=0002h。既有 BIOS 的 1／2 是為其他遊戲保留的空操作，不能直接據此宣稱本作游標完成。研究入口明確保存顯示計數，reset=-1，hide 遞減；目前只接受已經隱藏時的 hide，保持 register 與 flags。show 與可見游標的背景還原尚未實作，仍拒絕。收據記錄計數，不宣稱已畫出滑鼠游標。

### 滑鼠切片：原版遮罩與畫面游標層

狀態：READY。執行 2,088,411 步在 dosgolem 線性位址 `0x11953` 呼叫 AX=0009h，ES:EDX 指向原版 64-byte mask，後續 bytes 呼叫 AX=0001h。契約依 [FreeDOS mouse 專案的 INT 33h 參考](https://github.com/FDOS/mouse/blob/master/int33.lst) 0001h／0002h／0009h：前 16 words 為 screen AND mask，後 16 words 為 cursor XOR mask，每列最高 bit 在左。BX／CX 的 signed hotspot 限 -16..16。完整讀取後才發布形狀；show 將負的顯示計數加一至零，hide 遞減。未定義形狀的 show 仍拒絕，未臆造原廠預設箭頭。

游標先接到擷取畫面的獨立色號層，以原版 mask、目前滑鼠座標與實際 VGA／VBE bit depth 合成。每次從原始 framebuffer 重新合成，hide 或移動不留下舊游標。原始 VRAM 不改寫，收據分開保存原始色號雜湊與游標合成後的畫面；這是顯示層近似，尚未實作 TSR 對 VRAM 的背景儲存與還原，不宣稱 driver 讀回 parity。這份契約擴張前一節的隱藏子集。

參考快照：`workplace/fdos-int33.lst`，來源 `https://raw.githubusercontent.com/FDOS/mouse/master/int33.lst`，28,546 bytes，SHA-256 `9000e025b44351dbdd8b18887d03cb0edf9278c8a556fa9e24d2fe0600cf5411`。2026-10-11 下載，後續查證以這份固定輸入為準。資料表將第二段標為 10h offset，但 16 words 實際長 20h bytes；實作採 32 words、共 64 bytes，兩段各 32 bytes。

游標子集後，原版自然執行達三百萬步，僅因預設步數預算停止，沒有新增未支援服務。當時正在處理從 DAT 讀取的 87,502-byte 區塊。下一次提高有界探針的步數至一千萬，不略過解壓或等待指令。

### VGA 切片：由虛擬時間推進回掃

狀態：READY。一千萬步收據的末段重複停留在 `0x1BB9F–0x1BBA9`。原版第二個 JZ 回到第一個 IN。既有按「埠讀取次數 bit4」翻轉的模型，清除期間每輪連讀兩次，翻轉邊界固定落在第一個讀取，因而無法在第二次讀取見到上升沿。這是平台行為模型與輪詢步幅的固定相位問題，不能增加步數或改翻轉常數掩蓋。

沿用 dosgolem `docs/spec/193-crtc-timing.md` 的 READY 模型。啟用服務 Machine.CRTCTiming，將既有一微秒／成功指令的虛擬時間，依 Machine.StepsPerSecond() 換算成服務 Machine.Steps。CRTC、像素時鐘與回掃狀態仍由既有 VGA 模型計算。每道原版指令照常執行，不跳出 busy-wait、不看遊戲位址。測試標準模式的回掃起訖，以及同樣的雙讀取等待指令片段在有界時間內自行離開。記錄為 hardware-spec approximation，不宣稱週期精確。

驗證：四個平台套件通過。修正後一千萬步的原生畫面顯示 FamilySoft 標誌，已目視核對，檔案為 `workplace/startup-probe/frame-6b35961c2e8122ca5b1093174d85615287e6db6f86b831ac0ad9503763e852b6.png`，640×400，五種 RGB 顏色。PNG SHA-256 同檔名，原始色號 SHA-256 `653b5308aa72b2ebb5f32d4a64f54a2044f0935ad9c08593bcf32b6580b779a8`。這是 dosgolem 原版啟動畫面，尚非主選單或可玩驗收。下一次以三千萬步觀察後續開場，不略過原版指令。

三千萬步的原生畫面仍為 FamilySoft 標誌，已目視核對，PNG SHA-256 `83e0fccfb26a95907212939beb1d2a68b5a56e20d5348dfcae5653033f59c773`。原始色號雜湊與前次相同，DAC 顏色隨原版更新。停止原因只有 `instruction_budget`。末段位於 dosgolem 線性 `0x2C0C8` 的 DOS AH=2Ch 等待迴圈；原版校準值位於線性 `0x5B1E4`，當時為 199,999。堆疊參數為 100，與每毫秒約 200 次、每次五道指令的近似校準一致。下一次保留相同時間模型，僅延長有界預算至一億步觀察後續，不修改原版等待與色盤。

一億步自行顯示《德川家康》標題與主選單，已目視核對：重新開始、載入進度、歷史簡介、離開遊戲。PNG SHA-256 `7df7211c989bca30c3e82077011a96e7ef7142c9029c515aeb5dda5e461697e7`，640×400；原始色號 SHA-256 `c09843bfb23bf6d199b37f18643ae04c81f75647676510c8c6a0783b7b593df4`。沒有新增未支援服務，末段輪詢 INT 33h AX=0003h。這份收據證明原版自然啟動至選單，尚未驗證新遊戲。

### 輸入切片：虛擬時間滑鼠事件

狀態：READY。從實際選單畫面定義下一步普通玩家輸入。在成功指令邊界，依明示的步數時間表呼叫 dosgolem 既有 `internal/dos/bios.go` 的 MoveMouse、PressMouse、ReleaseMouse。事件只帶絕對像素座標與按鍵狀態，不帶遊戲位址，不改遊戲狀態或 EIP。限定 1,024 筆、時間嚴格遞增、必須落在執行預算內，座標及三個按鍵界限先完整驗證。實際座標仍由原版 INT 33h 07h／08h 設定的範圍夾取。

事件收據記錄預定時間、實際時間、要求座標、套用後座標與按鍵狀態。第一條路徑在主選單以左鍵點選畫面上的「重新開始」；按下與放開分開發送。沿正常入口重跑，不以直接呼叫選單處理常式代替。即使畫面推進，仍需目視核對後續畫面，不能由按鍵送出就宣稱玩家路徑完成。

### CPU 切片：XOR byte register immediate

狀態：READY。滑鼠移到「重新開始」後，原版在 80,000,349 步的 dosgolem 線性 `0x1B7D3` 遇到 `80 F2 FF`。補 80h /6 的 byte register immediate，以既有 reg8／setReg8 支援低、高 byte，結果更新 SF、ZF、PF，清除 CF、OF，未定義 AF 沿用現有邏輯旗標選擇。其他 bits、register 與非算術旗標保留。未支援 prefix 仍拒絕，缺少 imm8 不發布目的與旗標。這個切片只擴張 register XOR，不改選單流程。

驗證：四個平台套件通過。事件於 80,000,000 步移到 (300,172)，80,100,000 步按下左鍵，80,200,000 步放開；全部依排定時間套用。原版自行移除原主選單並顯示「自選武將」，PNG SHA-256 `ef2e1676432d01c2ca0e04957b938ed75f07cab95d1b6cff1997f3d60cb03a1c`，已目視核對。沒有跳過原版指令，這份收據證明正常滑鼠路徑能推進到新遊戲設定入口，尚未證明遊戲內場景。下一步依實際畫面點選 (130,43) 的「自選武將」。

後續在 90,000,000／90,100,000／90,200,000 步移動、按下、放開於 (130,43)，原始色號雜湊沒有變化。這不能證明已選入武將。為確認該文字是可點選項目、提示或未完成的畫面，使用既有 DOSBox 輔助腳本的 `--input` 分支：先記錄視窗幾何，依序送 Return 與相同座標的普通滑鼠點選，分段擷取。輔助圖片只用來縮小平台問題，正式收據仍須 dosgolem 重生。

DOSBox 0.74-3 輔助視窗實測為 640×342，標題上方裁掉部分圖像。Return 可顯示主選單；普通 X 滑鼠座標未能建立與遊戲座標的一對一映射，因此這次點選不構成同狀態比較。下一個窄查證使用既有 IDA 9.4 `.i64` 的一次性副本，只匯出 IDA 線性 `0x1B600–0x1BB46` 的原始指令、bytes 與交叉參照，確認目前輸入輪詢與矩形範圍，不修改資料庫名稱或遊戲規則。

窄查證更正：這個畫面是姓名編輯入口。IDA 原函式 `sub_10BD8` 將 Big5「自選武將」複製到可寫字串，再呼叫 `sub_1B7FB`。該輪詢常式右鍵 bit1 或 Esc 回傳 -1；呼叫端在 IDA 線性 `0x10D65` 轉入 `sub_1B312` 的確認流程。六個可選矩形位於字串的六個全形字位置，x 為 12、14、16、18、20、22 欄，y=36、寬二欄、高16。前次 (130,43) 的 x/8=16 正好落在不含邊界的矩形線上，沒有命中；不能將此解讀為平台輸入失效。

證據：IDA 9.4 匯出 `workplace/ida/range-1b600-1bb46.json`、`range-10900-10e00.json`，輸入 DX5.EXE SHA-256 維持既定值。這兩份使用 IDA LE loader 線性位址，與 dosgolem relocated 線性位址逐項對照相符。原函式名只用於導覽。下一條普通玩家路徑保留預設姓名，在 90,100,000 步按下右鍵、90,200,000 步放開，觀察原版確認畫面。

右鍵帶出確認對話框，PNG SHA-256 `efa0d55a6b0d25eb3017a775fcf03a0c8b9feafbca8ac0d066bc5a8f020323a1`，已目視核對。依實際畫面與 IDA `sub_1B312` 的矩形，於 95,100,000 步按下 (352,172) 的 Yes，95,200,000 步放開。原版顯示四個角色肖像，PNG SHA-256 `fd57c8d1887b4398f89a37b715d83799876cd7af5eabac30b06f14be4e456f84`，已目視核對；沒有新增未支援服務。這是角色建立中的肖像選擇，尚非遊戲內場景。

狀態：READY。下一步點第一個肖像 (132,124)。將普通輸入時間表保存為 `tools/player-path.json`，支援 `-mouse-file` 讀取同一陣列格式並記錄原始輸入 SHA-256；同時保留 inline 方式，但兩者不可同時指定。探針成功指令上限從一億延長至一億五千萬，容器的 180 秒逾時、64 MiB guest 配置上限與其他資源上限不變。這只延長有界觀察，不調整虛擬時間或原版等待。

第一個肖像已於 105,100,000／105,200,000 步按下與放開，原版再次顯示確認對話框。PNG SHA-256 `cf65bacf066b70bedd78f160d839ff7ed51709f13169f436690e03baa07ab6e6`，已目視核對；原始色號雜湊與先前姓名確認的 `852047ef996ba5fa4aed2e363be49281d9d3a47501424734ce50768b56da2af0` 相同。四個平台套件仍通過。下一步依同一對話框在 110,100,000／110,200,000 步點選 Yes。

肖像確認後，原版顯示四組能力配置。PNG SHA-256 `00e91cb83113b0f05f91f7a878a5257969732525638318d140592f676f39c322`，已目視核對，沒有新增未支援服務。下一步以普通左鍵點第一組的 (132,250)，再觀察後續確認。增加操作邊界收據：滑鼠原本與新事件都未按鍵時，在事件套用前由同一次原生 framebuffer 擷取畫面，記入 `input_boundary_frames`。這只讀取畫面並輸出 PNG，不改 guest VRAM、EIP、時鐘或輸入；可回查每個選擇前的實際畫面。

觀察更正：第一組配置之後進入能力編輯，並非立即確認。120,000,000 步之前的畫面右側顯示 100，PNG SHA-256 `030b78e6eafb93d1e4a164960939e38b34111151222561e2b5d1ba94c469d866`。下一次左鍵操作後，統率力 70 變 71、右側數字 100 變 99；結尾 PNG SHA-256 `9af314377d287538b6a97159a89c25c37c04db1a56f3655f440c4ea660ae1dd0`，兩圖均已目視核對。這證明原版能力編輯可接收操作，不據此推導完整分配規則。下一步探索普通右鍵結束編輯，保留操作前畫面，再依實際確認對話框繼續。

右鍵已正常結束能力編輯。130,000,000 步前顯示確認，PNG SHA-256 `a22c1f80e263eaa9b60e60bf5471b9c40fc452cb394e48ee3386bbf49d02253e`；點 Yes 後仍出現相同文字，PNG SHA-256 `523eb7ba9eceb1191ed9c97c0e7508eec15c4c1359ea37b20ac59e2bc29a1265`，均已目視核對。窄查證已證實是原版連續兩次確認：IDA 線性 `0x1165A` 與 `0x11676` 各呼叫 `sub_1B312`，前者 Yes 才進後者；後者 Yes 才返回角色建立常式。當時 dosgolem stack 的確認返回位址為 `0x1167B`，已到第二次，並非第一次點擊失效。原始證據為 `workplace/ida/range-11600-11800.json`；同次亦保存 `range-10e00-11600.json` 供能力編輯控制流回查。下一步在 135,100,000／135,200,000 步再點一次 Yes，完成這段正常建立流程。

### MSCDEX 切片：零長度播放的錯誤契約

狀態：READY。角色建立完成後，在 136,777,435 步的 dosgolem 線性 `0x2B6F8` 經 DPMI 0300h 呼叫 INT 2Fh 1510h。真實 22-byte request 位於實模式線性 `0x5CF2`，command=84h、addressing mode=1、起點 DWORD=81080300h、sector count=0。Microsoft MSCDEX 2.1 規格明定 Red Book DWORD 的最高 byte 未使用，其餘 bytes 為二進位分／秒／格；此起點換算 LBA=36,075，不能只因最高 byte=81h 就稱為非法位址。更早的 TOC buffer 重疊問題仍是獨立限制，不因這次命令而視為解決。

平台契約採 Microsoft 規格與 [DOSBox Staging v0.82.2](https://raw.githubusercontent.com/dosbox-staging/dosbox-staging/v0.82.2/src/dos/cdrom_image.cpp) 的 `PlayAudioSector`。後者對 len=0 明確拒絕並停止播放；[MSCDEX 轉接](https://raw.githubusercontent.com/dosbox-staging/dosbox-staging/v0.82.2/src/dos/dos_mscdex.cpp) 由 `lastResult=false` 產生 Error=8000h。這次只實作相同的零長度失敗回傳，寫回 request status=8000h，記錄拒絕原因，不宣稱已播放音樂。正長度播放仍保持未支援即停止，不以成功或假音訊繞過。

先驗證完整封包長度、來源可讀與目的可寫，再發布狀態；失敗不部分寫入。只接受 mode 0／1 的 22-byte PLAY AUDIO，其他未列命令仍拒絕。測試零長度狀態、封包外 bytes 保留、唯讀與短封包拒絕，以及正長度不假處理。

固定參考快照：`workplace/dosbox-staging-v0.82.2-cdrom_image.cpp`，47,358 bytes，SHA-256 `0c6b238c3576a1a3dd8725a6e8d38d90264ae4eaa3f736c8cec8adad6a6215e0`；`workplace/dosbox-staging-v0.82.2-dos_mscdex.cpp`，46,413 bytes，SHA-256 `d4224c790d3f2654cb89ee4e4febcafd8ca1bab830a56d1703d990abcd349d7f`。來源均為上述固定 tag 的公開原始碼，不含遊戲資料。

驗證：四個平台套件通過。零長度播放如實回報失敗後，原版自然載入新遊戲序章，人物圖與繁體中文敘事已目視確認。PNG SHA-256 `244cd00dc6bc7c7a62a98baef6c2089429f9ce3bfaf80423f52d4ec083bad823`，原始色號 SHA-256 `d4391b220ef70744860bf8f76c3fd625df735b300efa778b77071deb476c83cc`，640×400。下一步用 145,100,000／145,200,000 步的普通左鍵推進敘事，保存操作前畫面。光碟拒絕記錄只保留最後 64 筆，另存總數，避免重複請求造成無界主機記憶體與輸出成長。

普通左鍵已推進至「戰國時代最後的勝利者 德川家康」敘事頁，PNG SHA-256 `f8b23041cc0ab4866e40dd4d14eb92489b8c3896d2595ea5cba11f1338f0db62`，已目視核對；操作前畫面與先前序章 PNG 完全同雜湊。這是原版玩家輸入推進，沒有跳過遊戲指令。下一次保持同一輸入與時鐘模型，追加每五百萬步一次的普通左鍵，觀察序章後續及正式場景。探針成功指令上限延長至兩億；容器 180 秒與 guest 64 MiB 上限維持原值，未提高記憶體或跳過等待。

兩億步仍在序章對話，最新 PNG SHA-256 `b81925dc7235dc80deda7097166bf8566252f4d999a11c843b871f7cb8974a39`，已目視確認人物、對話框與中文文字均正常。每五百萬步的點擊均產生不同敘事頁，沒有新增未支援服務。下一次延長有界觀察至五億步，仍用普通左鍵推進，逐次保存操作前畫面；容器期限改為 420 秒，guest 64 MiB 與虛擬時間模型不變。這是同一序章的延長觀察，不新增遊戲規則或修改原版等待。

### MSCDEX 切片：閒置光碟的 STOP AUDIO

狀態：READY。五億步預算的重跑在 375,607,237 步遇到新的 INT 2Fh 1510h。DPMI 封包指定實模式 ES:BX=05CF:0004，線性 `0x5CF4` 的 request 為 `0D 00 85 00 00 01 00 00 40 01 00 00 03`。這是 13-byte STOP AUDIO，不是正長度播放。當時只有一次零長度播放失敗，沒有音訊正在播放。來源 EXE、dosgolem HEAD、IDA 與工具版本沿用本文件固定輸入。

Microsoft MSCDEX 2.1 規格的 STOP AUDIO 段落與 [DOSBox Staging v0.82.2 實作](https://raw.githubusercontent.com/dosbox-staging/dosbox-staging/v0.82.2/src/dos/cdrom_image.cpp) 支持閒置時接受停止。本子集只在目前不支援正長度播放的前提下，接受 subunit 0、13-byte、command 85h，驗證整份封包與目的可寫後將 status 設為 0100h，記錄閒置停止次數。其他 bytes、暫存器與旗標不變。這次成功表示停止一個閒置裝置，不表示播放過音樂。未來加入播放狀態時，STOP 必須改接實際播放狀態，不能沿用無條件閒置假設。

測試涵蓋重複閒置停止、狀態字以外的封包保存、非零 subunit、錯誤 drive、封包長度與唯讀／短 descriptor 的拒絕。正長度播放維持未支援即停止。最新畫面 PNG SHA-256 `be795e15cfa70ddfa62b7341051281b683dfe683be42a31540361fa520ec3425` 已目視核對，仍是序章對話版面。

驗證：四個平台套件通過，五億條指令自然執行完畢，沒有未支援停止；閒置停止計數為 2。末幀 PNG SHA-256 `9defc0559b9af797b90c7f8fb8075b8a2242978317e8f72a58ecbd5f5015a648`，原始色號 SHA-256 `7783492132ada56b8eb3813329985c1eed4147d413ca3d9b817ff9b4307e75e9`，已目視核對為序章對話。不能將預算結束稱為正式遊戲場景通過。為判定是否有普通輸入可離開序章，窄匯出當前輸入輪詢的 IDA 線性 `0x1BB40–0x1BBE0`；不改遊戲判斷。另存 `range-1de00-1e200.json`，其 `sub_1DE81` 是這個輪詢呼叫的既有更新常式，未據此附加遊戲語意。

已證實：當前輪詢在 IDA 線性 `0x1BB77` 只測試 INT 33h BX 的左鍵 bit0；否則呼叫鍵盤檢查與更新後繼續等候。右鍵不會直接離開這個輪詢。保留普通左鍵路徑，追加到七億五千萬步，容器期限 600 秒，guest 與時鐘上限不變。逐頁擷取仍由 dosgolem 原生 VRAM 產生。這次只讓目前可正常推進的敘事自然走完，不新增跳過序章的攔截。

驗證：七億五千萬步只因預算結束，四個平台套件通過。605,000,000 步之前已出現兩項劇情選擇，PNG SHA-256 `5b91fad6020208d3ab3015def1e5aec3f00d811a0e8e3e27b1f763419a650ba0`、原始色號 SHA-256 `b8d25f05737286f162ecd2a4a0ef5cb7b11bff859a3c1e56cd8fff85bf4914d7`，已目視核對。先前 (320,320) 在選項框外，重複點擊不會推進，屬測試路徑需選取選項，不是新平台缺口。下一次只把 605,000,000／605,100,000／605,200,000 步的普通滑鼠座標改成 (320,203)，左鍵選第一項「出城迎擊信玄」，其他條件不變。後續畫面再逐項核對，不據此宣稱戰鬥或存檔通過。

驗證：同一自然入口與普通輸入已推進至出陣選擇。605,100,000 步左鍵確實推進劇情，640,000,000 步前原版顯示新的回應，PNG SHA-256 `398cd2580e062c45f3225dc97c3ccbadbbdfd37faec7f3f6834af3195dfced48`，已目視核對。660,000,000 步前出現武將清單與「麒麟兒奮起／遠江國 三方原」，PNG SHA-256 `8a0c4b289842d7687b471f9e79ed564033f996f7d38daf5951dd0d5b7aeb8f04`，原始色號 SHA-256 `165e2088cf4ca7f3f11099bf4044d278233722796addac143d621c2f138560fd`，已目視核對。最新七億五千萬步完整收據與目前 player-path.json 的 SHA-256 相符，四個平台套件通過，未支援 DPMI 計數為空；原版與外部 dosgolem 保持唯讀。

下一步只用普通輸入點選首列「自選武將」(130,60)，再觀察普通右鍵是否結束選擇。時間表設定 660,100,000 步左鍵、665,100,000 步右鍵，各在後續 100,000 步放開；670,100,000 步保留既有確認對話框的 Yes 座標 (352,172)。這是可丟棄的玩家操作探索，不先宣稱選定兵力、結束出陣或進入戰場。

驗證：點選首列後，665,000,000 步前顯示自選武將的肖像與能力值，PNG SHA-256 `b44142f325c980e6e591138da6e9c44ca4ea278f54b616eacc9486064f6daf4c`，已目視確認。普通右鍵未關閉該能力頁；670,100,000 步的普通左鍵帶出上方確認框，675,000,000 步前 PNG SHA-256 `0cffe2538003e54f965cbe429f5515aab5607aeba2a04c109fa1da61ad86be08`，已目視確認。後續原座標在確認框外，因此停在同一畫面。末幀 SHA-256 `92c5cd82bf95334170bc00767245e60892f9b97c25272e758b70312466edc9b7`；四套件通過，七億五千萬步完整結束，輸入雜湊相符，沒有未支援 DPMI。

依畫面修正下一條普通輸入：675,100,000 步點上方 Yes (310,73)，680,100,000 步在清單位置 (130,60) 按右鍵探索結束選擇。685,100,000／690,100,000 步分別保留中央與上方確認座標。這些後續輸入的效果尚未確認，以同次邊界畫面為準；沒有新增遊戲位址或流程攔截。

驗證：上方 Yes 已正常接受，680,000,000 步前首列武將呈選取狀態，PNG SHA-256 `73ac9c66f92dbb9b0697ec48fd4ecd94193f9a2cbb0065ee832c49653474dd72`，已目視核對。右鍵已進入「那就出擊」，並帶出位於畫面下方的最後確認。末幀 PNG SHA-256 `dceb2f9097decb053a4cf46c3d3955ad7c84037c9209472d5b25ae509e47fb1b`，原始色號 SHA-256 `88e2ccdd55beb47b28e71411f64fcbb188ae2dcf4eab47d2b36e13be4437870d`，已目視核對。前次預留的中央與上方座標都在這個確認框外，因此仍等待；沒有新平台錯誤。七億五千萬步完整收據、輸入雜湊與四套件測試均通過。

下一次只把 695,000,000／695,100,000／695,200,000 步的座標改為這個下方 Yes 的 (352,273)。其餘保持已驗證的原版入口、角色建立、敘事、劇情選項與出陣選擇路徑。三個確認框的位置不同，後續普通輸入依實際畫面定位，不以單一座標宣稱所有確認已完成。

勘誤：上文對 `73ac9c66…` 的「首列武將呈選取狀態」描述過度。直接開啟該圖後可見它是兵種與兵力清單，包含騎士隊、騎兵隊、步槍隊、步兵隊與各列 1000。已證實的是武將確認後畫面推進到這個清單，右鍵後再進入出擊確認；尚未證實各列兵種與實際出陣武將的對應。這個更正不改操作時間表，後續以實際戰場與單位畫面驗證。

### CPU 切片：83h /6 register XOR sign-extended imm8

- 狀態：READY。
- 已證實：695,100,000 步在 (352,273) 按左鍵後，原版通過最後出擊確認，開始載入下一個場景。695,114,195 步的 dosgolem 線性 `0x200DC` 遇到 `83 F0 01`；EAX=0。來源仍為固定 SHA-256 的 DX5.EXE，位址基準為 dosgolem LE relocation 後線性位址，工具版本沿用前文。
- 平台契約：83h /6 的 imm8 先符號擴展至運算元寬度再 XOR。僅擴張 register 形式，支援 32-bit 與 66h 的 16-bit；後者保留暫存器高 16 bits。結果更新 SF、ZF、PF，清除 CF、OF。AF 未定義，沿用既有 setLogicFlags 的清除選擇。其他暫存器與非算術旗標保留。
- 發布條件：完整取到 imm8 才改目的與旗標。其他 prefix、memory 形式及重複 prefix 維持拒絕。
- 驗證：八個暫存器與兩種寬度，獨立列出正／負立即數、零、符號與奇偶結果；驗證高位保存、非目的暫存器與非算術旗標。短指令、未支援 memory 與 prefix 不得部分發布。

- 測試結果：新增切片與 CPU386、machine、dosfile、整合入口四套件通過。修正保存於本專案 dosgolem.patch，外部 dosgolem 不變。下一次仍由原始 LE 入口與相同玩家時間表重跑，局部指令測試不代替戰場載入驗證。

自然重跑已通過上述 XOR，於 695,114,229 步停在 dosgolem 線性 `0x1FBDB` 的 `D0 C2`。此時尚未畫出戰場，不把載入途中畫面列為戰場驗證。

### CPU 切片：戰場載入附近的 register 邏輯指令

- 狀態：READY。
- 證據：`workplace/ida/range-1fb80-1fe00.json`，IDA Pro 9.4，DX5.EXE SHA-256 `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a`。IDA LE loader 線性 `0x1FBDB` 對應檔案偏移 `0x1B3DB`，bytes `D0 C2`，原始指令 `rol dl,1`。與本次 dosgolem 停止位址逐項對照相符。
- 同份窄匯出的相鄰原始程式還包含兩項缺少的形式：IDA 線性 `0x1FC6C`／檔案偏移 `0x1B46C` 的 `66 81 F2 FF FF`，以及 IDA 線性 `0x1FD0F`／檔案偏移 `0x1B50F` 的 `66 23 C2`。已證實原始 bytes 存在；尚未證實本條玩家路徑會執行全部相鄰形式。不附加未查證的遊戲欄位語意。
- 平台契約依 Intel 80386 Programmer's Reference Manual 的 [ROL](https://pdos.csail.mit.edu/6.828/2018/readings/i386/RCL.htm)、[XOR](https://pdos.csail.mit.edu/6.828/2018/readings/i386/XOR.htm) 與 [AND](https://pdos.csail.mit.edu/6.828/2018/readings/i386/AND.htm)。D0h /0 僅新增 register byte 旋轉一次，包含高 byte；CF 取原 bit7，OF 取結果 bit7 XOR CF，其他旗標保留。66h 81h /6 新增 register XOR imm16；66h 23h 新增 r16 與 r16 的 AND，目的由 ModRM reg 欄指定，保留高 16 bits。兩項邏輯指令沿用既有 SF、ZF、PF／CF、OF 契約與未定義 AF 選擇。
- 完整取指後才發布結果；memory、其他未支援寬度與 prefix 仍拒絕。測試獨立列出結果與旗標，涵蓋八個暫存器、高位與非目的值保存、截短指令及拒絕形式。已支援的 66h LODSW、STOSW、OR r16,r16 不重寫。

測試結果：新增三項切片與四個平台套件通過，包含 AND 來源與目的相同的八種暫存器形式。補丁已保存。下一次同入口重跑驗證原版是否自然完成戰場載入。

自然重跑通過 D0h ROL，於 695,114,255 步停在 dosgolem 線性 `0x20129` 的 `23 C0`，即 `AND EAX,EAX`。新匯出 `workplace/ida/range-20000-20500.json` 使用相同輸入雜湊與 IDA 9.4，IDA 線性 `0x20129` 對應檔案偏移 `0x1B929`，原始 bytes 相符。狀態：READY。依同一 Intel AND 契約，將 23h register 形式擴張至 32-bit，完整更新目的與邏輯旗標；相同來源與目的也照常更新旗標。memory 與 prefix 拒絕條件不放寬。新增八乘八暫存器組合的獨立零、符號、奇偶與相同來源測試，再重跑原版。

32-bit AND 切片與四個平台套件通過，保存補丁後再從原入口執行。這次不改輸入時間表與原版狀態。

### CPU 切片：繪圖路徑的 word ADD memory 與 ROL CL

- 狀態：READY。
- 自然路徑已通過 32-bit AND，於 695,114,398 步的 dosgolem 線性 `0x2D8D3` 遇到 `66 03 05 6E 4D 03 00`。IDA 9.4 的 `workplace/ida/range-2d880-2dc00.json` 記錄相同原始指令，IDA 線性 `0x2D8D3`、檔案偏移 `0x290D3`，輸入雜湊沿用固定 DX5.EXE。
- 依 [Intel ADD](https://pdos.csail.mit.edu/6.828/2018/readings/i386/ADD.htm) 擴張 66h 03h 的 memory 來源。沿用 decodeAddress32 選 DS／SS 與 readSegment16 段界限，完整讀取後才呼叫 add16，目的高 16 bits、來源記憶體與其他暫存器不變。獨立測試絕對位址、EBP／SS、SIB、進位、溢位、AF、符號、零與奇偶；截短位址與跨段界限不發布目的或旗標。
- 同一窄匯出在 IDA 線性 `0x2D8B2`／檔案偏移 `0x290B2` 保留 `66 D3 C0`，原始 `ROL AX,CL`。已證實 bytes 存在，未宣稱目前路徑已執行。依前述 Intel ROL 契約補 register word 形式，count 先遮罩為五 bits。count=0 完全保留；非零 CF 取最後移出 bit；只有 count=1 定義 OF，其餘沿用保留舊 OF 的明示選擇。其他旗標與高 16 bits 保留，memory／prefix 仍拒絕。測試 count=0、1、15、16、17、31、32、33 與高位保存。

測試結果：兩項切片與四套件通過。來源記憶體、段界限、SIB、旗標及 CX 同時作目的與計數來源都有獨立驗證；補丁已保存，再沿相同正常入口重跑。

### CPU 切片：byte ADD memory

狀態：READY。原版自然通過 word ADD memory，於 695,114,712 步停在 dosgolem 線性 `0x2CDE3` 的 `02 45 FC`，即 `ADD AL,[EBP-4]`。IDA 9.4 窄匯出 `workplace/ida/range-2cda0-2ce80.json`，IDA 線性 `0x2CDE3`／檔案偏移 `0x285E3`，輸入仍為固定 DX5.EXE。依前述 Intel ADD 契約只擴張 02h 的 byte memory 來源，以 decodeAddress32 與 readSegment8 驗證來源，完整取到 byte 才改目的及旗標。22h AND memory 維持拒絕。測試八個低／高 byte、DS 絕對來源、EBP／SS 負偏移、SIB、旗標、高位、記憶體保存與失敗不發布。

測試結果：byte ADD memory 與四個平台套件通過，補丁保存後沿原入口重跑；未改輸入檔與時鐘。

### CPU 切片：原版生成程式的 XCHG／ROR／ADC

- 狀態：READY。
- 自然路徑已通過 byte ADD，於 695,114,813 步的 dosgolem 線性 `0x5C384` 遇到 `26 86 0F`，即 `XCHG CL,ES:[EDI]`。這是原版執行時生成的程式，沒有對應的原始 EXE 檔案偏移。當次 `workplace/startup-memory.bin` SHA-256 為 `5c8e890ec6f06ec406eb050b5746ef07f99ebf13d852d660148c2ba8459cbf76`；工具為既定 dosgolem HEAD 加本專案補丁，證據位址均為 dosgolem relocation 後線性位址。
- 同一記憶體在 `0x5C39C` 有 `D0 CD`，即 `ROR CH,1`，`0x5C39E` 有 `83 D7 00`，即 `ADC EDI,0`。這兩項已證實生成 bytes 存在，尚未宣稱已執行。起於 `0x5C384` 的原始 32 bytes：`26860f5a4a7c1883c75081ee000000007fde81c604000000d0cd83d700ebd1c3`。
- 依 [Intel XCHG](https://pdos.csail.mit.edu/6.828/2018/readings/i386/XCHG.htm) 新增 byte memory 交換，支援正常 DS／SS 定址與實際 ES override。來源與目的完整存取成功才發布 register，所有旗標保持。單執行緒決定性執行器沒有並行 bus master，不另宣稱實機 BUS LOCK 時序。
- 依前述 Intel rotate 契約新增 D0h /1 register ROR 一次，CF 取原 bit0，OF 取結果 bit7 XOR bit6，其他旗標保持。
- 依 [Intel 80386 手冊第 260 頁 ADC](https://www.read.seas.harvard.edu/~kohler/class/aosref/i386.pdf#page=260) 新增 83h /2 register 32-bit immediate，imm8 符號擴展後與原 CF 一起加，沿用已測 adc32。完整取到立即數才發布結果，其他寬度、memory 與 prefix 不擴張。
- 測試低／高 byte、DS／SS／ES、不可寫與越界失敗、交換前後記憶體與旗標；ROR 的明示 CF／OF 結果；ADC 的兩種初始 CF、正負立即數、零、溢位與高位結果。這些修正不改原版生成的 bytes。

測試結果：三項切片與四個平台套件通過，補丁保存後由同一原始入口重跑。

### CPU 切片：word register NOT

狀態：READY。原版已自然通過生成程式的 XCHG／ROR／ADC，顯示戰場外框與右側小地圖，PNG SHA-256 `c8ff990b6c6c8c12e00d7dd67d3e2ce18332dc9d83f476a99d7cd996874c8bf0`、原始色號 SHA-256 `4f1e1deb2c6632a5f155e99e36f5b3b4fec2c6e719890d13f83ce3fd85a44702`，已目視核對；中央主戰場尚未畫完。711,986,165 步於 dosgolem 線性 `0x1FA42` 遇到 `66 F7 D3`，即 `NOT BX`。IDA 9.4 匯出 `workplace/ida/range-1f9d0-1fba4.json` 記錄相同 bytes，IDA 線性 `0x1FA42`／檔案偏移 `0x1B242`，輸入為固定 DX5.EXE。

依 [Intel NOT](https://pdos.csail.mit.edu/6.828/2018/readings/i386/NOT.htm) 僅補 66h F7h /2 register 形式，反轉低 16 bits，高 16 bits 與全部旗標保留。memory、其他 prefix 不擴張。測試八個暫存器、零／全一／交錯／符號邊界值、兩組原始旗標、截短與拒絕形式。先通過平台回歸，再確認主戰場完成繪製。

測試結果：word NOT 與四個平台套件通過，保存補丁後由相同原入口重跑。

自然重跑通過 NOT，已開始畫出中央主戰場第一個地形圖塊；PNG SHA-256 `dc94c7bfc5c622ce46fff3a3508f0ceff9e9c0c353db9f35974f2c370c857fc9`，已目視確認，仍未畫完。712,034,451 步停在 dosgolem 線性 `0x1FEAB` 的 `66 83 C7 06`，即 `ADD DI,6`。狀態：READY。沿用 Intel ADD 契約，新增 66h 83h /0 register 形式，imm8 符號擴展至 16 bits，再由 add16 更新算術旗標，保留目的高 16 bits。完整取指後才發布；其他 prefix 與 memory 不擴張。測試八個暫存器、正負立即數、進位與溢位，以及失敗不發布。

IDA 9.4 的 `workplace/ida/range-1fe00-20000.json` 確認 `0x1FEAB` 對應檔案偏移 `0x1B6AB`。同一窄匯出還有 IDA 線性 `0x1FFDB`／檔案偏移 `0x1B7DB` 的 `66 81 EF 40 01`，即 `SUB DI,140h`。狀態：READY。依 [Intel SUB](https://pdos.csail.mit.edu/6.828/2018/readings/i386/SUB.htm) 補 66h 81h /5 register imm16，沿用 sub16 算術旗標與高位保存契約；完整取到 imm16 才發布。與 word ADD imm8 一起做獨立邊界、旗標與拒絕測試，其他 memory／prefix 不擴張。

測試結果：word ADD imm8、SUB imm16 與四個平台套件通過，補丁保存後由相同原入口重跑。

### CPU 切片：word register ADC

狀態：READY。原版已自然通過兩項 word 立即數形式，中央戰場地形完整繪製，草地、樹木、河流、右側小地圖、場景文字與下方 `0／20 下午` 可見。PNG SHA-256 `38b8c902bcd6ce0ddd5d8f2e2a4695915ba1462a0babc585cb451206746f2194`，原始色號 SHA-256 `308658bb05227be3959c95fc7176a93634429a11d1b974512271a5147e96b334`，已目視確認；尚未據此宣稱單位操作或戰鬥完成。

712,449,312 步的 dosgolem 線性 `0x245E0` 遇到 `66 13 C3`，即 `ADC AX,BX`。沿用 Intel ADC 規格，只補 66h 13h register 形式。以完整 17-bit 中間和計算 CF，依 16-bit 結果更新 OF、SF、ZF、AF、PF；目的高 16 bits 與其他暫存器保持。測試所有來源／目的組合、相同暫存器、兩種原 CF、進位、正負溢位、AF、零與奇偶。memory／其他 prefix 不擴張，不推導遊戲欄位或亂數內部語意。

IDA 9.4 的 `workplace/ida/range-245a0-24650.json` 確認上述原始 bytes，IDA 線性 `0x245E0`／檔案偏移 `0x1FDE0`，輸入為固定 DX5.EXE。相鄰 ADD AX,imm16 與 XCHG word memory 已支援，不重寫。新增 ADC 與四個平台套件通過，補丁保存後沿相同原始入口重跑。

### CPU 切片：SS word CMP memory immediate

狀態：READY。原版已通過 ADC，於 712,449,722 步的 dosgolem 線性 `0x27035` 遇到 `66 36 83 3D 9F 4D 03 00 00`。這是 `CMP word SS:[34D9Fh],0`，不能忽略 SS override 或改讀 DS。擴張既有 83h /7 word memory 比較，明確允許 SS override；先驗證完整定址、立即數與 SS 的兩個來源 bytes，再用 sub16 更新旗標。全部暫存器與記憶體不變。測試 DS／SS 使用不同 base 與資料、絕對位址／EBP／SIB、符號擴展、旗標、跨段界限與拒絕形式，其他 SS 指令不擴張。

來源為 [Intel 80386 手冊第 287 頁 CMP](https://www.read.seas.harvard.edu/~kohler/class/aosref/i386.pdf#page=287)。IDA 9.4 匯出 `workplace/ida/range-27000-27090.json` 確認相同 bytes，IDA 線性 `0x27035`／檔案偏移 `0x22835`，輸入為固定 DX5.EXE。新增 SS 比較與四套件通過，補丁保存後由相同入口重跑。

### CPU 切片：byte register NOT

狀態：READY。原版自然通過 SS 比較，於 712,449,736 步的 dosgolem 線性 `0x270D8` 遇到 `F6 D0`，即 `NOT AL`。IDA 9.4 匯出 `workplace/ida/range-270ce-27160.json`，IDA 線性 `0x270D8`／檔案偏移 `0x228D8`，bytes 相符，輸入為固定 DX5.EXE。依前述 Intel NOT 契約新增 F6h /2 register byte，包含高 byte；只反轉目的八 bits，所有旗標、其他 bits 與暫存器保持。測試八種低／高 byte 的零、全一、交錯與符號邊界值，兩組旗標與截短／memory／prefix 拒絕；不擴張 memory 形式。

回歸首輪的新 NOT 向量通過，舊 `TestByteTESTRegister` 仍將 NOT DL 當作未支援而失敗。將該拒絕樣本改為仍未支援的 F6h /5 IMUL DL，保留拒絕檢查；NOT 的正反向與狀態保存由新增向量驗證。這是能力矩陣擴張後的過期測試，不刪除拒絕測試或擴張未遇到的 IMUL。

修正過期樣本後，byte NOT 與四個平台套件乾淨重跑通過。補丁已保存，再由相同原版入口與普通輸入重跑。

### CPU 切片：dword AND memory sign-extended imm8

狀態：READY。原版自然通過 byte NOT，於 712,500,240 步的 dosgolem 線性 `0x20523` 遇到 `83 65 0C FE`，即 `AND dword [EBP+0Ch],-2`。依前述 Intel AND 契約擴張 83h /4 的 32-bit memory 形式，立即數先符號擴展，沿用 decodeAddress32 的 DS／SS 定址與 read／writeSegment32。完整讀取與寫入成功後才更新邏輯旗標；其他暫存器與非算術旗標保持。測試 DS／SS／SIB、負立即數與明示結果、來源可讀但不可寫、跨四-byte 界限、截短與未支援 word／prefix 形式，失敗不得部分發布。

IDA Pro 9.4 匯出 `workplace/ida/range-20500-20640.json` 確認相同 bytes，IDA LE loader 線性 `0x20523` 對應檔案偏移 `0x1BD23`，輸入為固定 DX5.EXE。只記錄定址與運算，不附加未證實的遊戲欄位語意。新增切片與四個平台套件通過；補丁已保存，沿相同入口與普通輸入重跑。

### CPU 切片：dword CMP memory register

狀態：READY。原版自然通過 AND memory，於 712,967,136 步的 dosgolem 線性 `0x16D08` 遇到 `39 05 67 43 03 00`，即 `CMP dword [34367h],EAX`。IDA Pro 9.4 匯出 `workplace/ida/range-16cc0-16d40.json` 使用固定 DX5.EXE，原始指令的 IDA LE loader 線性位址為 `0x16D08`，檔案偏移為 `0x12508`。依前述 Intel CMP 契約擴張 39h 的 32-bit memory 目的比較，完整解碼與讀取 DS／SS 四個 bytes 後，以 sub32 更新旗標，不寫回相減結果。所有暫存器與記憶體保持。word memory 與其他 prefix 不擴張。測試八個來源暫存器、DS／SS／SIB、進位、溢位、AF、零與奇偶，以及截短與跨段界限時不發布旗標。

停止時的原生畫面已繪出兩個人物單位，底部顯示 `1／20 下午`。PNG SHA-256 `b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846`，原始色號 SHA-256 `eac8871b05cc4a25ded2be4fb5f25baf0ef0f51f291b68fd11412d4f207cfb82`，已目視核對。這證明單位已顯示，尚未證明玩家可以選取、移動或交戰。新增 CMP 切片與四個平台套件通過；保存補丁後再由原入口重跑。

### CPU 切片：dword register TEST immediate

狀態：READY。原版自然通過 CMP memory，於 712,968,057 步的 dosgolem 線性 `0x2029C` 遇到 `F7 C3 02 00 00 00`，即 `TEST EBX,2`。既有 IDA Pro 9.4 匯出 `workplace/ida/range-20000-20500.json` 記錄相同 bytes，IDA LE loader 線性 `0x2029C`／檔案偏移 `0x1BA9C`；相鄰的 INT 33h AX=3 為普通滑鼠位置與按鍵查詢。來源為固定 DX5.EXE，不改原始名稱。

依 [Intel TEST](https://pdos.csail.mit.edu/6.828/2018/readings/i386/TEST.htm) 補 F7h /0 dword register immediate。完整取到 imm32 後，只以暫存器與立即數的 AND 更新 SF、ZF、PF，清除 CF、OF；未定義 AF 沿用既有邏輯旗標 helper 的清除選擇。全部暫存器、記憶體與其他旗標保持。既有 word／memory 形式不改，其他 prefix 與 /1 仍拒絕。測試八個暫存器、零、符號、高位與低 byte 奇偶，以及截短立即數與拒絕形式時不發布狀態。

測試結果：新增 TEST 切片與四個平台套件通過。補丁已保存，再沿同一原版入口與普通滑鼠時間表重跑。

### 本輪整合啟動驗收

- 已證實：固定 DX5.EXE 由原始 LE 入口自然執行，經主選單、角色建立、敘事、劇情選項、武將確認與出擊確認，抵達戰場。750,000,000 條指令完整結束，停止原因為 `instruction_budget`；不是未支援指令或服務。
- 已證實：390 筆普通滑鼠事件均依時間表套用。715,000,000 至 740,000,000 步的六個邊界畫面與末幀均為 PNG SHA-256 `b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846`，與前述已目視確認的地形、人物單位、小地圖及 `1／20 下午` 相符。末段 trace 持續走原版 INT 33h AX=3 查詢與等待迴圈，沒有跳過指令。
- 已證實：收據 SHA-256 `307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4`，保存於 `workplace/startup-probe/receipts/307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4.json`。輸入時間表 SHA-256 `f621bd1f66e0d4bb492570317ec82b898ec95d3a1f4139a6ee5c451011b30395`，全部工具、補丁與原始輸入雜湊相符。工具版本與 dosgolem HEAD 沿用 CONTEXT，`fd2_startup_calls=0`、`native_runtime_instructions_skipped=false`，未支援 DPMI 計數為空。
- 已證實：CPU386、machine、dosfile、整合入口四套件通過；Python 與 shell 語法、文件索引、相對連結及擁有權檢查通過。原版與外部 dosgolem 工作樹保持唯讀，外部既有 `cmd/probe/main.go` 差異不變。
- 限制：本輪只確認原版啟動至戰場、普通輸入與穩定查詢。戰場地面點擊沒有產生已確認的單位選取、移動或交戰結果；存讀檔未驗證。CD 正長度音訊尚未支援，滑鼠游標仍為已標示的顯示層近似。不得據此宣稱全遊戲相容或完整可玩。

### CD 只讀診斷切片

狀態：READY。此切片只增加有界觀測，不變更 DPMI／MSCDEX 行為。輸入仍為本文件固定 SHA-256 的 DX5.EXE、CUE/BIN 與 dosgolem HEAD。診斷輸出放在 `workplace/cd-trace/`，正式啟動驗收收據保留在原目錄。

- `tools/startup-probe.go` 新增預設關閉的 `-cd-trace`。只在 DPMI 0100h 與模擬 INT 2Fh 的 0300h 邊界，記錄前後暫存器、selector、配置帳本、請求封包、實模式轉送緩衝區及最近 64 條原始指令。
- 每筆記憶體讀取最多 128 bytes，最多保存 256 次呼叫，另計溢出數。輸出位址標為 dosgolem relocated flat linear；IDA 匯出另列 IDA LE loader linear EA 與檔案偏移。
- 觀測只讀 backing memory，不呼叫裝置讀取，不改記憶體、暫存器、旗標、輸入或成功步數，不安裝取代指令的 StepHook。
- 以同原始輸入、同預算、同普通輸入比較觀測開／關的完整 guest 記憶體雜湊、原始色號、停止位置、成功步數與中斷計數；一致才接受觀測工具。後續修正仍須獨立的平台 READY 契約。

驗證結果：CONFORMED，範圍限本次只讀診斷。Go 1.24.13、既定 dosgolem HEAD 加平台補丁，兩次均從固定 DX5.EXE 原入口執行 3,000,000 步，沒有滑鼠事件。只差診斷欄位與隨機命名的可寫副本目錄；其餘收據欄位完全一致，包括停止原因 `instruction_budget`、dosgolem 線性 `0x2AE59`、暫存器 trace、中斷計數與原生畫面。完整記憶體 SHA-256 兩次均為 `22b236d9853cc9696f93bb757d68456d5370a3172f8b8ff797fcdb52a432ad74`，原始色號 SHA-256 均為 `bfc2b1c83db55a1b9465ae59af51b344d18ae6f3f04ada059f4a967bea130b71`。此時畫面仍為黑色，不是可玩性驗收。

| 診斷輸出 | SHA-256 |
|---|---|
| `workplace/cd-trace/trace-on.json` | `49746df9ddeca87c72f0224e7428aa48168f3d5671096f1e34c07b466cc24a98` |
| `workplace/cd-trace/trace-off.json` | `8d50544c1b4e062cb2d6dfc3969b63c6956d270f0f9d4838b6e9722ee6ea10ee` |
| `tools/startup-probe.go` | `6f6c8adf3aa4a38ff59d7934b56d0850ae6282443ddfbb1083901ca737f5b049` |
| `workplace/cd-trace/range-2b000-2b900.json` | `4df62d0f61583649f85a0080f17296c7efc3147a39635f97fd5586baff92b7df` |
| `workplace/cd-trace/range-29080-29580.json` | `8290100bb2ad1458cbe496aa6756247b8e4493d0c2493094f4e5b96efa6ec73e` |

兩次平台回歸均通過 CPU386、machine、dosfile、研究入口四個套件。開啟時保存 100 次診斷呼叫，丟棄零次。原先 750,000,000 步的正式啟動收據沒有被這次短程輸出取代。

### CD 配置與 TOC 回寫資料流

輸入沿用上述固定 DX5.EXE。靜態證據使用 IDA Pro 9.4 的 LE loader，保留原始 `sub_` 名稱、bytes、交叉參照與檔案偏移。動態證據為 Go 1.24.13 的 dosgolem relocation 後平面線性位址；表中逐列明示工具位址與檔案偏移，不把相同數字當成相同位址空間。

| 推論等級 | 原始定位 | 已觀測行為與出處 |
|---|---|---|
| 已證實 | IDA `sub_2B682`，IDA 線性 `0x2B697`／檔案偏移 `0x26E97` | 執行 `INT 31h AX=0100h`，成功後在 IDA `0x2B69E` 將 DX 存到呼叫者提供的位置，直接保留 EAX 返回；完整短函式未將 AX 乘 16。證據 `range-2b000-2b900.json` |
| 已證實 | IDA `sub_2925B`，IDA 線性 `0x29279`／檔案偏移 `0x24A79` | 呼叫 `sub_2B682` 配置請求，將回傳 EAX 當作平面 memcpy 指標，再於 IDA `0x292F4` 右移四位產生實模式 ES，原值低四位放 BX；IDA `0x2930F` 呼叫原始實模式中斷包裝。證據 `range-29080-29580.json` |
| 已證實 | IDA `sub_2933F`，IDA 線性 `0x2935D`／檔案偏移 `0x24B5D` | 同樣將回傳 EAX 當作查詢緩衝區的 memcpy 指標，並於 IDA `0x293B6` 將其原始 DWORD 放入請求的傳輸指標欄位。欄位實際解讀依 MSCDEX 封包，不能只補一處位移便宣稱修復。證據同上 |
| 已證實 | dosgolem 線性 `0x2B699` 的 0100h 返回，成功步數 2,016,636 與 2,016,728 | BX 分別為 1 與 2 paragraphs，AX 分別返回 `0x5C5F` 與 `0x5C60`，selector 低字分別為 `0x0100`、`0x0108`。帳本的實際線性 base 分別為 `0x5C5F0`、`0x5C600`；配置區保持零值。證據 `trace-on.json` |
| 已證實 | dosgolem 線性 `0x2B6FA` 的 0300h 邊界，成功步數 2,016,870 | 真正傳給 INT 2Fh 1510h 的請求在 `0x5C60`，IOCTL 查詢緩衝在 `0x5C5F`，長度 7。回寫前查詢 bytes 為 `0a1a0003000054`，已含請求標頭；TOC 回寫後請求開頭由 `1a0003000054` 變成 `01103a000100`。兩者重疊已被前後快照直接確認 |
| 已證實 | dosgolem 線性 `0x2B6FA`，成功步數 2,017,319 | 下一次查詢 11 的緩衝在 `0x5C62`，請求在 `0x5C63`，待查軌號已被請求長度 `0x1A` 覆蓋。裝置回報 `0x8108`，不假裝查詢成功。證據同上 |

[DPMI 0.9 規格第 9.1 節](https://www.phatcode.net/res/262/files/dpmi09.html)定義 0100h 的 AX 為實模式段號、DX 為 selector，BX 以 16-byte paragraph 計數。[Open Watcom 官方手冊的 DOS/4GW 章節](https://openwatcom.org/ftp/manuals/current/fpguide.pdf)列出該服務，並說明標準零基底平面模式中近指標等於線性位址。這些平台契約支持上述地址換算，不證明網站包內 DOS4GW 的所有實際行為。

目前結論限於本次 dosgolem 執行：原版使用回傳段號作平面指標，兩份緩衝區互相覆蓋。**與後續零長度 PLAY 的完整因果鏈仍為未知。**下一步使用實際 DOS/4GW 的輔助執行確認配置與 TOC 行為，再決定所需相容性處理。尚未改動 DPMI 回傳、配置間距、原版 EXE、軌號或播放長度。

### DOSBox-X 輔助配置查證

沿用既有 image，沒有另建工具鏈。執行器為 DOSBox-X 2026.06.02 SDL1，實際 commit `d75a57c72a74a6d87250e4b2556e70ea02995e51`；image digest 與重跑方式見 AGENTS 的 Docker 邊界。包內 `DOS4GW.EXE` 大小 265,396 bytes、SHA-256 `b8265123ac8a189637448618409ef3ecd2e9f3e1a47062c685a02240f688dec1`。其餘遊戲輸入沿用固定雜湊。輔助腳本 [aux-cd-trace.py](tools/aux-cd-trace.py) 只設中斷斷點、單步跨過中斷與讀取記憶體，沒有改寫 guest 資料或跳過原版指令；音訊停用，不能據此宣稱已聽到音樂。

| 推論等級 | DOSBox-X 原始定位 | 輔助觀測 |
|---|---|---|
| 已證實 | 保護模式 `CS=0160h, EIP=001E3697h`，0100h 呼叫前；返回為 `EIP=001E3699h` | 第一筆 BX=1，返回 AX=`07F1h`、DX=`01C0h`；第二筆 BX=2，返回 AX=`07F3h`、DX=`01C8h`。這些是 DOSBox-X 的執行期值，不是 IDA 位址或 dosgolem 位址 |
| 已證實 | DOSBox-X `DOS MCBS` 原始輸出 | MCB 段號 `07F0h` 管理 16 bytes，下一個 MCB `07F2h` 管理 32 bytes；資料分別起於 `07F1:0000` 與 `07F3:0000`。實際配置資料前各有一個 paragraph 的 MCB |
| 已證實 | 保護模式 `CS=0160h, EIP=001E36F8h` 的 0300h 呼叫，原始 ES=`0168h`、EDI=`00213480h` | 實模式封包指向 `007F:0003`，實模式線性位置 `0x7F3`，並非配置資料的 `0x7F30`。請求內查詢指標是 `0000:07F1`，線性位置 `0x7F1`。兩份誤用指標相差 2 bytes，仍然重疊 |
| 已證實 | `workplace/cd-trace/aux-debug/query-at-segment-before.bin`、`request-at-segment-before.bin` | 回寫前查詢開頭為 `0ab51a00030000`，請求開頭為 `1a00030000d4`。後者的 `1a` 位於前者第三個 byte，直接確認重疊 |
| 已證實 | 同目錄的 `query-at-segment-after.bin`、`request-at-segment-after.bin` | TOC 回寫後查詢開頭為 `0a01103a320001`，請求開頭為 `103a320001d4`。這是原始輔助 dump，沒有登錄成 dosgolem 原版收據 |

這份輔助證據排除了「只有 dosgolem 才將 AX 視為段號」的解釋。它也顯示兩個執行器的 DOS 配置模型有可見差異：dosgolem 的單調配置沒有 MCB，而此次實際 DOS/4GW 透過 DOS 配到含 MCB 的區塊，段號差為 2。相差 1 或 2 會改變被覆蓋的欄位；尚未沿全部軌道查詢至 PLAY 取長度，因此不能宣稱補 MCB 即可修復 CD 音樂。

下一個查證入口為正常釋放／重配、音軌查詢與第一筆 PLAY 的資料流。若需要補 DOS 記憶體配置，先建立包含 MCB、回收、合併、錯誤與邊界的通用契約，不以額外空隙遮住錯誤。正式原版仍保持下載 bytes，不改 DPMI 段號契約或硬編碼音軌。

終端輸入的首次失敗屬驗證工具問題。依固定版本 `debug.cpp`，命令解析使用 LF，Escape 另走 Alt 快捷鍵處理；修正 PTY 模式、命令換行與按鍵輸入後，以同一 image 乾淨重跑。該來源快照為 `workplace/cd-trace/dosbox-x-debug.cpp`，來源 [DOSBox-X 固定 commit](https://raw.githubusercontent.com/joncampbell123/dosbox-x/d75a57c72a74a6d87250e4b2556e70ea02995e51/src/debug/debug.cpp)，SHA-256 `d1b3bc73fdc58ce12fec696be76bd8fa35e049237e8e1b167d797d37264fec27`。沒有把自動化失敗列為產品缺陷。

正式輔助腳本乾淨重跑退出 0，暫存器及七份 64-byte 記憶體擷取與首次成功觀測一致。`registers.json` SHA-256 為 `17302b9c2c60c9466e5f35a1d9578481d1930100df96db2b1135731745ec034d`。腳本 SHA-256 為 `bdf9d2dc6ec2c79752eb0955eb31c6b2d3d421e56853a4221aad8771fb2ce5ef`；同目錄 `manifest.json` SHA-256 為 `681c1b5ef8267be7ef5b11f8082b14ed7e818902a3f1c9d7d7a1cbc85f8a327d`，逐筆保存原始輸入、執行器 binary、記錄檔、終端與記憶體擷取的雜湊，離線逐項核對通過。

### 後續音訊與呈現規劃的事實盤點

- 已證實：既有完整收據 `307f83bf8ffcf8c9d85a61710443b02c29db94145401c74b05189b98bdabefb4` 的 CD 裝置共有 16 軌，其中 15 軌為 AUDIO；八次 PLAY 的 sector count 均為零，閒置 STOP 共七次。TOC buffer 重疊與零長度 PLAY 是否為同一根因，仍為未知；不能只補 PCM 輸出便宣稱原版正常選曲已修復。
- 已證實：以 Python 3.13.7 在既定唯讀 Docker 掛載中，使用 `bytes.find(text.encode("big5"))` 找到以下 bytes。檔案偏移均為十進位，不是 IDA 或 dosgolem 線性位址。

| 輸入與 SHA-256 | 搜尋文字 | 檔案偏移 |
|---|---|---|
| DX5.EXE，`2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a` | 出城迎擊信玄 | 181244 |
| 同一 DX5.EXE | 麒麟兒奮起 | 176472 |
| 同一 DX5.EXE | 家康 | 175075 |
| DX5.DAT，`de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37` | 麒麟兒奮起 | 3153478 |
| 同一 DX5.DAT | 家康 | 4332 |

- 限制：找到指定 Big5 bytes 只證明這些片段存在，不證明整份字串表、控制碼、字模、所有繪字呼叫或完整翻譯入口已解出。「重新開始」的指定 Big5 bytes 在兩檔均未找到，也不能據此宣稱所有選單都是圖片。
- 已證實：cpu386 已有 IntHook 與可取代指令步進的 StepHook；既有 `oracle/call.go` 的 Call 使用 16-bit CPU 與 far 呼叫約定。不能把其他遊戲的 OnCall／Call 能力直接當成本作 LE 的文字與 HD 事件擷取已完成。觀測功能須另有不略過原版指令的契約與測試。
- 已證實：Ebitengine 的[官方能力表](https://ebitengine.org/en/documents/features.html)列出桌面圖形、滑鼠與音訊支援。這只是桌面試作候選的依據，不證明本作相容性或任何打包平台已驗收。瀏覽器已由使用者明確排除。

### 2026-10-12：TOC 至第一筆零長度 PLAY 的因果鏈

路由再次核對 dosgolem、IDA 與平台規格優先入口。輸入沿用本文件固定雜湊的 DX5.EXE、DOS4GW.EXE 與 CUE/BIN。工具為 Go 1.24.13、CONTEXT 的 dosgolem HEAD 與 IDA Pro 9.4。DOSBox-X 沿用前節固定版本與 image，只作輔助證據。沒有切換 `.BAK` 或修改原版。

`--toc-queries 2` 額外擷取後續兩次查詢。首次工具刪錯斷點，斷言拒絕輸出；固定版本 debugger 使用 `push_front` 新增斷點，0100h 在此時的索引為 1。依既有原始碼修正索引並核對 BPLIST 後，以同一 image、命令與原版輸入乾淨重跑，退出 0，manifest 為 `completed=true`，逐檔雜湊通過。這是驗證工具修正。

| 推論等級 | DOSBox-X 輔助定位 | 原始觀測 |
|---|---|---|
| 已證實 | `aux-toc/queries.json`；兩次 `CS=0160h, EIP=001E36F8h` 的 0300h | 兩次都重用請求線性 `0x7F3`、傳輸線性 `0x7F1`；MCB dump 仍為 `07F0h`、`07F2h`。配置不會像目前 dosgolem 一路向後 |
| 已證實 | `toc-00-query-before.bin`／`after.bin` | 第 1 軌：`0b011a00030000` → `0b010002000001`；MSF 起點四 bytes 為 `00020000`，換算 LBA 0 |
| 已證實 | `toc-01-query-before.bin`／`after.bin` | 第 2 軌：`0b021a00030000` → `0b023d27000001`；MSF 起點四 bytes 為 `3d270000`，換算 LBA 2836，符合 CUE 幾何 |
| 已證實 | query 與 `request-after.bin` | 請求狀態 `0100h` 覆蓋查詢最後兩 bytes，控制欄位變成 `01h`。只確認起點保留，不能宣稱完整 TOC 回寫正確 |

dosgolem 以 `tools/player-path.json` 的原樣事件前綴重跑 150,000,000 步，共 33 筆普通滑鼠事件。只保留步數小於預算的事件，沒有改座標或時間。診斷保存 102 次呼叫、丟棄零次；無未支援 DPMI、無 FD2 攔截，停止原因為 `instruction_budget`。收據與完整 backing RAM 另存為 `first-play.json`、`first-play-memory.bin`，沒有取代正式戰場驗收收據。

| 推論等級 | 原始定位與位址空間 | 因果證據 |
|---|---|---|
| 已證實 | dosgolem 成功步數 2,017,319 至 2,032,004，0300h 前後快照 | 31 次音軌查詢的軌號皆被請求長度 `1Ah` 覆蓋，全部回報 `8108h`。查詢前綴由 `0b1a0003000000` 變成 `0b1a0003088100`，起點四 bytes 成為 `00030881` |
| 已證實 | IDA 線性 `0x29607–0x2960D`，檔案偏移起於 `0x24E07` | 原始 `sub_295D1` 直接把查詢 bytes 2 至 5 複製給呼叫者，沒有依裝置錯誤狀態拒絕這份起點 |
| 已證實 | IDA 線性 `0x1DBDC`／`0x1DBF7`，檔案偏移 `0x193DC`／`0x193F7` | 原版對當前軌及下一軌各查一次，解釋 16 軌共 31 次查詢；尾軌改用 lead-out |
| 已證實 | IDA `sub_1DA75`，線性 `0x1DA75–0x1DAC7`／檔案偏移 `0x19275–0x192C7` | 公式為 `minute*4500 + second*75 + frame - 150`。同一錯誤起點得到同一值；最高 byte 不參與公式 |
| 已證實 | IDA 線性 `0x1DC03–0x1DC1B`，檔案偏移起於 `0x19403` | 原版換算兩個起點後以 `sub ebx,eax` 取得長度。原始表格寫入端為 `dword_3ACD4`、`dword_3AE64`，位於 IDA `0x1DC48`／`0x1DC57`，檔案偏移 `0x19448`／`0x19457` |
| 已證實 | 150,000,000 步的 dosgolem backing RAM，線性 `0x3ACD4`、`0x3AE64` | 軌 1 至 16 的起點均為 `81080300h`；軌 1 至 15 長度均為 0，軌 16 為 `FFFF844Dh`。尾軌的損壞 lead-out 尚未修正 |
| 已證實 | IDA 線性 `0x1DCF3`／`0x1DCFF`／`0x1DD05`，檔案偏移 `0x194F3`／`0x194FF`／`0x19505` | 原版直接從兩張表取長度及起點，呼叫原始 `sub_29716` 建立 PLAY 封包 |
| 已證實 | dosgolem 成功步數 136,777,435，請求線性 `0x5CF2` | 第一筆 PLAY 為模式 1、起點 `81080300h`、長度 0，回報 `8000h`。完整記憶體的原始 `dword_358AC` 位於線性 `0x358AC`，值為 10；第 10 軌的表格值逐項與封包一致 |

結論：本次第一筆零長度 PLAY 已連回失敗 TOC 的回寫、原版無條件取值、起點表、長度相減與 PLAY 封包。其餘七次 PLAY 尚未逐次補完整資料流；沒有宣稱全部選曲已查清或音訊已修復。

勘誤：前節將檔案偏移 `0x19395` 的 `EB`／`74` 用途列為未知。本次 IDA 確認前一指令在 IDA 線性 `0x1DB91` 比較 TOC 末軌是否等於 16。網站 EXE 的 `EB 18` 無條件跳至 `0x1DBAF`，略過「Please put Game CD」失敗出口；`.BAK` 的 `74 18` 是相同目標的條件跳躍。這只證實此處靜態控制流，不證明 `.BAK` 的版本來源或其他差異，兩份雜湊不變。

下一個切片尚未達 READY：依 [FreeDOS ke2046 的 MCB 定義](https://raw.githubusercontent.com/FDOS/kernel/ke2046/hdr/mcb.h)與[配置器](https://raw.githubusercontent.com/FDOS/kernel/ke2046/kernel/memmgr.c)，建立包含 paragraph 標頭、first-fit、釋放重用、合併、resize、錯誤與界限的通用 DOS 記憶體契約，並核對 dosgolem 現有配置測試。這些是公開平台語意，不另開遊戲 RE。尚未改配置器、DPMI 回傳、原版 EXE、音軌或播放長度，不能宣稱只加 MCB 即可完成 CD 音樂。

| 收據或工具 | SHA-256 |
|---|---|
| `tools/aux-cd-trace.py` | `74c0b21d706f064abfb3e174c17f558cbe985bbb570198952656f40ae6f35070` |
| `workplace/cd-trace/aux-toc/manifest.json` | `86912374bad33dbc12fdd976b5641f07534f1cbd2dbd122c40b0c741136a26eb` |
| `workplace/cd-trace/aux-toc/queries.json` | `343e2c0579cfdbbe99a636ee5e02fd974d364be91df75ed800296ba399b07793` |
| `workplace/cd-trace/first-play-input.json` | `cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736` |
| `workplace/cd-trace/first-play.json` | `398fa89669bcf3e64ef3f207bd677e9931be01d1cfb808211f9c83b58ff12159` |
| `workplace/cd-trace/first-play-memory.bin` | `3862c326fa4679e92df16b3d8fb7f194f9f19ac73792b5abc1fb94cac06189d1` |
| `workplace/cd-trace/range-29400-29a00.json` | `d94c4d8aeff7b8d4aa34eab50b67b41c317a99d2098802bcf1373339e414386f` |
| `workplace/cd-trace/range-1db50-1dd30.json` | `d15b3a916e2b2d075d3678dfff6ad7a121a86498bad33af68461a9a5be0a5bb7` |
| `workplace/cd-trace/range-1da75-1dac8.json` | `d22e3aae67bd11de7cf70a1d05569e5f741ebb88655266ff9a9b06335e14953c` |
## 2026-10-12：DOS MCB 配置模式契約

狀態：DRAFT → 證據審查 → READY。本節規範通用平台子集，不改原版 EXE 或 MSCDEX 的指標契約。

- 來源：[DPMI 1.0](https://docs.pcjs.org/specs/dpmi/1991_03_12-DPMI_Spec_v10.pdf) 第 72–74 頁、[FreeDOS ke2046 MCB 定義](https://github.com/FDOS/kernel/blob/ke2046/hdr/mcb.h)、[FreeDOS ke2046 記憶體管理](https://github.com/FDOS/kernel/blob/ke2046/kernel/memmgr.c)，以及 dosgolem `docs/spec/012-mcb-chain.md`。規格沒有遊戲位址。
- `EnableDOSMCBMemory(ownerPSP)` 在掛接機器後、首次 DOS 配置前啟用。既有診斷用單向配置維持預設。新模式只管理映像尾端對齊後至 `0xA0000` 的連續區間，不冒充完整 DOS PSP／系統 MCB 鏈。
- 每個 MCB 占 16 bytes。資料區起點為 MCB 段加一；標頭含 M／Z、owner PSP、資料段數、保留零與空白名稱。首次適合配置、分割、釋放重用及相鄰自由區合併都須實作。原始映像與低位記憶體不作配置用途。
- 配置與縮放拒絕零段，回 `8021h`。32 位客戶端每塊只配置一個描述子，基底為實模式段乘 16，界限為資料 bytes 減一。這個模式不支援 16 位客戶端、UMB 或配置策略切換。
- `0100h` 空間不足回 `0008h`，BX 為最大自由資料段數；描述子耗盡回 `8011h` 且不消耗區塊。`0101h/0102h` 未登錄選擇子回 `8022h`。MCB 標頭與主機帳本不一致回 `0007h`，不繼續改寫。
- 縮放保持資料起點，只吃相鄰自由區。採 FreeDOS 的 DOS resize 行為：成長不足時先擴至可用上限，再回 `0008h` 與最大段數；主機帳本及描述子界限同步更新。這是明示的平台相容性選擇，不宣稱 DPMI 規格規定失敗後記憶體必須改變。
- 實模式預設堆疊也從同一配置池取得；線性配置仍位於 1 MiB 以上。釋放同步撤銷描述子及清除引用它的資料段暫存器。保留其他既有 DPMI 簡化，例如未模型化 RPL／TI。
- 證據審查：MCB 尺寸、重用與原版 TOC 重疊的因果已具備；本節範圍可由公開平台契約實作。驗收包含標頭／界限、重用、合併、縮放失敗、損壞與描述子耗盡，並以同一原版入口重跑 TOC 和第一個 PLAY。正長度 PLAY 的音訊能力另依 Issue #2，不以接受空請求假裝播放成功。

驗證：上述子集達 CONFORMED。四套件回歸通過。`workplace/cd-trace/mcb-toc.json` SHA-256 `a604c71736540a30df797e973e21d0361fdc978fc0a5dbdafe64b8dc8424c581` 為 3,000,000 步收據；MCB 讓一段／兩段配置固定重用 `5C60h`／`5C62h`。原版 1–15 筆長度為 `2836,6967,3667,3771,7595,7064,5974,6999,6642,6811,7595,6807,5509,6375,7094`，與掛載 CUE 的相鄰軌起點差一致。末軌長度仍因原版 lead-out 緩衝區重疊而無效，不宣稱全部 TOC 正確。

同一普通滑鼠前綴在第 136,777,435 步送出第 10 軌 PLAY：MSF `41 1C 0B 00`，LBA 51,515，長度 6,811。正長度服務尚未實作，因此如實停在 `INT 31h/0300h`。收據 `workplace/cd-trace/mcb-first-play.json` SHA-256 `07a8cba6c5224566230ea2de6ccaf57b3d82c6b13c79adf4163daa6c39774883`，完整記憶體 `mcb-first-play-memory.bin` SHA-256 `5e4b23af96a94a9d6716a41e9623a502dae472d75d2b75f04d85551a6c4137fe`。這證明第一筆零長度的原因已修正；完整玩家路徑待 CDDA 接入後再驗證。

## CDDA 裝置與 PCM 切片契約（2026-10-12）

知識路由命中通用平台規格，沿用 `local/retro-platform-spec-first.md`。狀態：DRAFT → 證據審查 → READY。來源為本地固定 `workplace/mscdex21.txt` 的 PLAY／STOP／RESUME／Audio Status 與固定 [DOSBox Staging v0.82.2 BinaryFile](https://github.com/dosbox-staging/dosbox-staging/blob/v0.82.2/src/dos/cdrom_image.cpp)；來源雜湊已記於既有光碟研究節。

- MSCDEX `84h` 接受 HSG／MSF 起點及磁區數；`85h` 暫停播放；`88h` 只從暫停續播。第二次閒置 STOP 清除續播位置，自然結束清除播放／暫停狀態。新 PLAY 替換目前曲目，非法範圍回錯誤，不延伸成整軌。
- 成功請求狀態為 `0100h`，播放中附 busy `0200h`。保留零長度 PLAY 的既有 `8000h` 相容性錯誤；無效地址與資料軌回 `8108h`，無可續播狀態回 `810Ch`。未知服務仍停止。
- 新增 IOCTL INPUT `15` 的 11-byte Audio Status，包含暫停旗標及最後 PLAY／下次 RESUME 的 MSF 起訖。既有 TOC 回寫與狀態覆蓋順序不變。
- 裝置以虛擬微秒推進，44,100 Hz、16-bit little-endian、雙聲道，每磁區 588 個立體聲樣本框。時間換算為 `floor(播放微秒 × 44100 / 1000000)`，保留分數餘量；不以主機音訊時鐘驅動原版。標為 `hardware-spec approximation`。
- 只接受已掛載 CUE 的 AUDIO 範圍；資料軌、lead-out 越界與算術溢位拒絕。CUE 顯式 PREGAP 沒有 BIN bytes，這個子集拒絕包含該空隙的 PLAY，不把資料當音訊或猜補靜音。跨相鄰 AUDIO 軌可讀，依各軌 file/disc sector 轉址。PREGAP 播放支援保留為未實作能力。
- PCM 串流逐塊讀取，不載入整個 BIN。無頭輸出只串接實際播放的樣本，事件記錄虛擬時間、輸出樣本位置、起訖、停止與換曲，供還原靜止間隔；樣本數、非零樣本數、峰值與 SHA-256 進收據。輸出留本機。
- 證據審查：第一筆合法 PLAY 已定位，公開介面及 BIN 位元組序已有來源；不需繼續逆向音訊 driver。測試覆蓋換曲、STOP／RESUME、自然結束、分塊獨立性、邊界、跨軌、PREGAP 拒絕及真實 BIN 抽樣。人耳確認留待桌面驗收，不由非零 PCM 取代。

### 實作與正常玩家路徑驗證

本切片的裝置狀態與 PCM 子集達 CONFORMED，時間仍屬 `hardware-spec approximation`。原版 EXE、DAT 與 BIN、外部 dosgolem 工作樹維持原樣。四套件回歸通過，包含描述子耗盡、MCB 損壞、分塊獨立性、STOP／RESUME、自然結束與真實 BIN 位元組抽樣。

| 收據與輸入 | 結果 |
|---|---|
| 固定 EXE／BIN／dosgolem／Go | 沿用本文的固定雜湊、HEAD 與 Go 1.24.13；工具逐檔 SHA-256 在收據 `integration_input_sha256`，已與本輪檔案核對 |
| 首筆播放收據 | `workplace/cd-trace/cdda-first-play.json` SHA-256 `96cf81ae4caba97ce3a74cd213fcef6e7559c624dbbaf3208725b19a13985068`；150,000,000 步，583,115 個立體聲樣本框 |
| 正常玩家路徑 | `tools/player-path.json` SHA-256 `f621bd1f66e0d4bb492570317ec82b898ec95d3a1f4139a6ee5c451011b30395`；390 筆普通滑鼠事件全部套用 |
| 正式完整收據 | `workplace/startup-probe/latest.json` SHA-256 `06971d5bca19eb5103dd7c6b6a6c7bd759c5a2ed3b3b361d21fb79d26d6b2e36`；750,000,000 步，只因預算結束；`fd2_startup_calls=0`、DPMI 未支援為空 |
| 原生畫面 | PNG SHA-256 `b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846`，與先前已目視確認的戰場畫面相同；原始色號 SHA-256 `eac8871b05cc4a25ded2be4fb5f25baf0ef0f51f291b68fd11412d4f207cfb82` |
| 完整記憶體 | `workplace/startup-memory.bin` SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`；包含新平台記憶體狀態，不要求與單向配置器逐位元相同 |
| PCM | 44.1 kHz、s16le、雙聲道；19,818,728 個立體聲樣本框，79,274,912 bytes；SHA-256 `d81c04498183d8cf64ef3409f50b51aeb0c10047cbe1049f13cf0a4ae9e9320a` |
| 音訊統計 | 39,066,917 個非零聲道樣本，絕對峰值 32,768；8 筆 PLAY、零拒絕，事件零丟棄。只證明技術輸出，不宣稱聽覺已確認 |

八筆 PLAY 均來自原版執行，不注入曲目。以每筆播放到下一次停止、完成或預算結束的虛擬時間獨立計算樣本數，再逐段讀取 BIN，比較全部 79,274,912 bytes；樣本數與 SHA-256 一致。

| 虛擬微秒 | 原版軌號 | 起點 LBA | 請求磁區數 |
|---:|---:|---:|---:|
| 136777435 | 10 | 51515 | 6811 |
| 375607684 | 5 | 17241 | 7595 |
| 430614148 | 10 | 51515 | 6811 |
| 535614121 | 5 | 17241 | 7595 |
| 595614159 | 5 | 17241 | 7595 |
| 605110706 | 10 | 51515 | 6811 |
| 657220321 | 2 | 2836 | 6967 |
| 712967868 | 6 | 24836 | 7064 |

完成範圍：MCB 平台錯誤與零長度 PLAY 的因果修正、正常入口至首場戰場的原版選曲／換曲、裝置狀態及無頭 PCM。限制：顯式 PREGAP 播放未支援；末軌長度受原版 lead-out 重疊破壞，尚未走到該玩家路徑。兩者接續 Issue #2 查公開裝置契約，不猜補長度；桌面與人耳確認接續 Issue #3。單位操作、戰鬥與存讀檔維持未驗證。
## 2026-10-12：CDDA 靜音間隙與映像結尾契約

知識路由命中 `local/retro-platform-spec-first.md`。狀態：DRAFT → 證據審查 → READY。本節補充既有 CDDA 子集，不改原版 EXE、TOC 表格或 PLAY 封包。

- [GNU libcdio 的 track pre-gaps 文件](https://libcdio.github.io/libcdio.html)第 6.3 節明定 CUE `PREGAP` 在音軌前插入靜音；`INDEX 00` 才可能指向有來源音訊的間隙。本子集仍不支援 `INDEX 00`，顯式 AUDIO PREGAP 依宣告長度輸出零值 PCM，並消耗相同虛擬播放時間。
- 固定 [DOSBox Staging v0.82.2](https://github.com/dosbox-staging/dosbox-staging/blob/v0.82.2/src/dos/cdrom_image.cpp) 的 `PlayAudioSector` 先驗證起點；`BinaryFile::decode` 依實際可讀 bytes 解碼，`CDAudioCallBack` 在 EOF 嘗試接續，起點到 lead-out 時停止。因此有效音訊起點搭配超過映像末端的長度，不必拒絕整個請求。這是成熟模擬器的相容性依據，不宣稱實體光碟機皆採同一錯誤處理。
- `workplace/dosbox-staging-v0.82.2-cdrom_image.cpp` SHA-256 為 `0c6b238c3576a1a3dd8725a6e8d38d90264ae4eaa3f736c8cec8adad6a6215e0`。本節採 CUE 的靜音長度，與該版本 DOSBox Staging 會略過起點 PREGAP 的策略有明示差異。
- PLAY 使用 64 位計算請求終點。起點在資料軌或 lead-out 以後仍回 `8108h`；零長度維持既有 `8000h`。有效音訊範圍的請求終點超過 lead-out 時，以 lead-out 作實際終點。收據同時保存原始模式、起點、磁區數、請求終點、實際終點與是否裁止，不猜補遊戲的末軌長度。
- 靜音、檔案音訊及相鄰音軌可連續播放。暫停與續播保留分數樣本時間；自然結束回到閒置。狀態查詢回實際播放終點。時間換算仍標為 `hardware-spec approximation`。
- 驗收涵蓋起點位於 PREGAP、跨入／跨出 PREGAP、暫停續播、分塊獨立性、極大長度、非法起點及全部十五軌的 BIN 位元組核對，再由正常入口重生八次選曲與戰場收據。受控裝置測試不冒充正常玩家已走到末軌。

驗證結果：本節子集達 CONFORMED。Go 1.24.13 與既定 dosgolem HEAD 的四套件回歸通過。`workplace/startup-probe/cdda-boundary-test.log` 保存十五條 AUDIO 軌的磁區數、樣本數與 PCM SHA-256，逐軌與 BIN 的獨立區段讀取相符。末軌接受受控 `FFFFFFFFh` 長度，實際輸出 6,452 個磁區、3,793,776 個立體聲樣本框，並在 lead-out 自然停止。PREGAP 的零值 PCM、跨兩段靜音、暫停續播及分塊獨立性均通過。這些是裝置測試，沒有注入原版遊戲的選曲狀態。

正式正常玩家路徑重跑 750,000,000 步、390 筆普通滑鼠事件，零 FD2 攔截、零未支援 DPMI、零音訊拒絕及零事件丟棄。收據 SHA-256 為 `c6b33e27450f09a6668cf77d848a13774c8e2780f7d96126b266da773570e075`，保存於 `workplace/startup-probe/latest.json` 及同雜湊的 receipts 檔。第 10、5、10、5、5、10、2、6 軌的選曲保持一致。重新獨立解析 CUE，以事件間虛擬微秒計算樣本數，再逐段讀 BIN 比較全部 79,274,912 bytes，通過。PCM 仍為 `d81c04498183d8cf64ef3409f50b51aeb0c10047cbe1049f13cf0a4ae9e9320a`，戰場 PNG 仍為 `b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846`。

`workplace/startup-probe/cdda-verification.json` SHA-256 `1398449174fabc42480fa667eca98e38603a56cf775198c09d823c3f4f8cfa72` 記錄逐段曲目、時間、樣本數、完整收據與測試雜湊。邊界測試 log SHA-256 `c54294c99d323b33b8b14548ddccea41b4251bf7549c5a3c3978de23394614b2`；四套件 log SHA-256 `11d01dd523da1734c75118690c840b2059d801eb83a6ce5281e05575a0affb4b`。原版 lead-out 表格損壞仍保留；裝置依映像末端停止，不冒稱 TOC 已修復。桌面播放與人耳確認繼續留在 Issue #3，`listening_verified=false`。
## 2026-10-12：HD 首個素材入口與唯讀觀測契約

路由命中資產考古與 IDA，已載入 `reverse-engineer-retro-game-remake` 的 `references/evidence-and-re.md`、`use-ida-pro-9-4` 及 `/home/anr2/ida_94_official/knowledge-base/ida-94-tools.md`。沿用已驗證的 IDA 9.4 image 與 UID，在正式 DB 的一次性副本查詢，原版及正式 DB 唯讀。輸入沿用本文件固定 DX5.EXE、DX5.DAT 雜湊。

| 推論等級 | 原始定位 | 證據 |
|---|---|---|
| 已證實 | DAT 檔案偏移 0、8、12 | `Ken Lin\0`、DWORD 201、201 筆各 21 bytes 的記錄；名稱欄 13 bytes，DWORD 長度在記錄 +13，DWORD 檔案起點在 +17。第一個 payload 位於 4233，全部連續至檔案結尾 3,908,861。`asset-inventory.json` 逐項保存原始記錄及雜湊 |
| 已證實 | IDA `sub_1D510`，線性 `0x1D56B–0x1D66F`／檔案偏移 `0x18D6B–0x18E6F` | 讀 12-byte 標頭，比較 `Ken Lin`，從標頭 +8 取數量，乘 `15h` 配置及讀取目錄。`asset-xrefs.json` 保留 bytes 與原名 |
| 已證實 | IDA `sub_1D6B0`，線性 `0x1D714–0x1D778`／檔案偏移 `0x18F14–0x18F78` | 以 21-byte stride 比對名稱；記錄 +17 轉成資源起點，+13 轉成長度。`sub_1D800` 於線性 `0x1D831–0x1D855`／檔案偏移 `0x19031–0x19055` 以起點加資源內偏移執行 lseek/read。證據 `range-1d6af-1db25.json` |
| 已證實 | IDA `sub_1BF44`，線性 `0x1BF49–0x1BF68`／檔案偏移 `0x17749–0x17768` | 唯一參數為資源名稱指標；以 80-byte pitch、零偏移與四個緩衝區呼叫 `sub_28689`。證據 `range-1bf38-1bf73.json` |
| 已證實 | IDA `sub_28689`，線性 `0x286AE–0x28760`／檔案偏移 `0x23EAE–0x23F60` | 透過 `sub_1496E` 讀資源，取前兩個 WORD 至 `dword_5B1B8`、`dword_5B1B4`，再分別呼叫四次 `sub_2ADC8`。四個目的緩衝區來源保留 `dword_38C00`、`dword_38BF8`、`dword_38C04`、`dword_38BFC` 原名 |
| 強推論 | IDA `sub_2ADC8`、`sub_2AEB2`，線性 `0x2ADC8–0x2AEE0`／檔案偏移 `0x265C8–0x266E0` | 兩個 WORD 用作欄數及列數，寫入以 pitch 跳列。壓縮碼包含讀取 VGA 當前內容的分支；不能只憑靜態檔案大小宣稱所有 OZM 可獨立解碼。尚未實作正式 OZM 解碼器或 HD 圖層 |

唯讀觀測子集狀態：DRAFT → 證據審查 → READY。只適用固定 EXE 與現有零基底 LE 研究探針，不作通用繪圖 API 或正式素材包格式。

- 預設關閉的 `-hd-trace` 在原版 `sub_1BF44` 的入口及相同堆疊返回處，從 backing RAM 複製有界參數。入口名稱限制 64 bytes；事件上限 256，待返回深度上限 16。超界只記觀測錯誤，不改 guest 狀態。
- 返回時保存原始兩個 WORD 值、四個目的指標，以及各緩衝區按 80-byte pitch 擷取的 SHA-256。只有 1–80 欄、1–400 列、完整 RAM 界限內且不重疊 VGA aperture `0xA0000–0xBFFFF` 的資料才計算，不讀 VGA 埠、不修改暫存器、VRAM、記憶體或指令步進。
- 入口與返回均附成功指令數及原始線性定位。資源名稱與 DAT 雜湊對照另由研究盤點完成。載入事件不能當作目的畫面矩形、遮擋、素材已替換或 HD 已可用的證據。
- 驗收為記憶體不變、錯誤界限與事件上限測試，再以相同原始輸入比較觀測開／關的停止點、暫存器 trace、全部 RAM、原生畫面與音訊。美術風格仍待使用者確認，這個觀測切片不依賴風格。

10,000,000 步的開／關驗證通過。兩側全部 RAM SHA-256 均為 `98bc27be6addc922a0485de03826cf766738a27fc13b9146d39e01136ed61a68`，停止點、trace、中斷計數、DPMI 呼叫、畫面與 PCM 一致。開啟時記錄 WINDOW.OZM、MAP_WAKU.OZM、MAP_CHIP.OZM、OP1.OZM 的四組入口與返回；原始尺寸分別為 18×16、48×120、80×400、80×400，全部符合 DAT payload 前兩個 WORD。四個目的緩衝區均位於 backing RAM，沒有觀測錯誤、事件丟棄或待返回項。收據 `trace-off.json` SHA-256 `a79085f5c53651664c14ee6dd2522665504fa368bfe05f844de84b8c62cb1691`、`trace-on.json` SHA-256 `4a2e606177888a5b5c4b4bdf186e38df02f981459c322dc14ecb2bbd32a5c9fb`；`invariance.json` SHA-256 `f4602a57fb2eaa54a36d165b1b7b20dac91b546b68e71f93a164cc21dc54eb82`。均在 `workplace/hd-prototype/`。

HD 風格使用內建 image_gen 產生一次可丟棄比較圖，原始參考是已驗收戰場 PNG `b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846`。本機 `style-comparison-v1.png` 為 1586×992，SHA-256 `906216c9e8bf00cb303b2e8ee956d5626b106cf3fc8a7e9f7eeafd120a8ceab1`；完整提示詞 `style-comparison-v1.prompt.txt` SHA-256 `20d3b4bec3e1e8d977eaa8b40c13a382eac1404beb78de30fc0498e4c680134a`。A 為原作漫畫風精修，B 為戰國手繪風，尚待使用者選擇。生成圖是 `layout-only` 風格示意，面板幾何、字形及人物細節不當作同狀態、正式規格或素材已接入的證據。圖像含原版參考的衍生內容，只留本機。

盤點 `asset-inventory.json` SHA-256 `947be34a477ab73056d1c480c05f2f712340a7c5c4b1dadbc4e6e2126f2be3b7`；IDA 錨點 `asset-xrefs.json` SHA-256 `678dd0d763ec4ccf6a8eb107032ab06feae9ac5694dd967f30155085c2a27d9a`；目錄查找 `range-1d6af-1db25.json` SHA-256 `4cd4ab0bb26ea2e289565568dc87a2e0366aeacbc44af4de041057c54b048ead`；載入包裝 `range-1bf38-1bf73.json` SHA-256 `27a9fede105d65874d2dc48f4a831c83db11b59dccc0c719d9c39379c997e068`。其餘窄範圍、提示詞與概念圖雜湊由 `prototype.json` 索引。盤點收據在靜態證據審查前產生，當時的 `strong inference` 標記保留，目錄欄位的升級依本節讀寫端證據。

完整原版入口至戰場重跑 750,000,000 步、390 筆普通滑鼠事件通過，唯讀觀測子集達 CONFORMED。記錄 19 次載入、17 個不同來源，零錯誤、零丟棄、零待返回。每次尺寸均符合來源標頭，四個原版解碼緩衝區均成功保存雜湊。與 CDDA 完成收據比較，全部 RAM、停止點、trace、中斷、普通輸入、畫面與 CD 裝置及 PCM 一致。RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`；完整觀測收據 SHA-256 `31f8b5b7e96d9007047feff195624ae0dd806692a1a9e62d019964e8bda8201e`。`workplace/hd-prototype/normal-path-verification.json` SHA-256 `a4c8ab544ca6419621ba990eb2fd1a6724c3d5d6a7e11ace556d3fd891c8115f` 保存逐項來源、時間、尺寸與平面雜湊。正式 CDDA 收據未被診斷輸出取代。

使用者在完整觀測驗證後選定 B 戰國手繪風。採用此方向製作人物、地形與介面，原版玩法及邏輯座標不變。這項決定只定美術風格，未定正式素材包格式、顯示比例或繪圖擷取 API。下一切片為單項原版素材擷取、手繪試作及實際繪圖使用端的定位。

### 第一張劇情素材的擷取契約

路由沿用資產考古、IDA 與文件職責。擷取工具子集狀態：DRAFT → 證據審查 → READY。只處理本機試作，不加入正式呈現路徑或定義 HD 素材包格式。

- 使用相同正常玩家輸入前綴執行 150,000,000 步。輸出另存 `workplace/hd-prototype/asset-capture/`，保留完整戰場收據。最後載入必須為 A-0.OZM、40-byte 欄寬、150 列、80-byte pitch；四個平面在停止時仍須與載入返回的 SHA-256 相符。
- 依 IDA `sub_1C084` 的四次 VGA plane 選擇與複製順序，將四個緩衝區合成 320×150 色號。以原生畫面候選矩形 `(160,50,320,150)` 逐點建立色號與 RGB 對照。任何色號對應多個顏色即拒絕輸出，不猜調色盤或補缺色。
- 驗收保存 RAM、原生 PNG、輸入、工具與 DAT 資源的 SHA-256、四平面 SHA-256、色號與候選矩形。PNG 的每個像素須與候選矩形一致。這只確認本次畫面上的素材位置，實際繪圖呼叫參數與遮擋仍另查證。

擷取子集達 CONFORMED。正常前綴 33 筆滑鼠事件、150,000,000 步，最後載入 A-0.OZM。全部 48,000 個像素匹配 `(160,50,320,150)`。來源 payload 檔案偏移 566,431、長度 15,505、SHA-256 `9c4a2d83ccf6be585b09d10a761f3d05920805887aeea509885c9126afee0a81`。收據 SHA-256 `2a72776a2cabb6a2769287cc0902821003d3e3e664a4b6ab294e0460de64c4e7`；RAM `0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f`。擷取 PNG `0b7a25670a568f2e8e61ecfdf4d7ea264dac13caf67dea7ddf9218b370f1239c`，`A-0-extraction.json` 為 `df4958895a97ae730cce3a824682867a8fa4964c0fd6eb547abcf85c040ee6c0`，均在 `workplace/hd-prototype/asset-capture/`。未出現的色號 0、13 保留「未觀測」，不補猜 RGB。

補充繪圖使用端的已證實靜態證據：IDA 原始 `sub_18002` 在線性 `0x18018–0x18038`、檔案偏移 `0x13818–0x13838` 將劇情圖編號加 20，交給 `sub_1BF73` 載入，接著以來源偏移 0、寬 40 bytes、高 150、目的 pitch 80、目的線性 `0xA0FB4` 呼叫 `sub_1C1BC`。目的減 VGA aperture 起點 `0xA0000` 等於 `50*80+20`，對應像素 `(160,50)`。`sub_1C1BC` 依序選四平面，再由原始 `sub_1C26D` 按列 `rep movsb` 複製，這條路徑沒有透明遮罩。`range-17fb0-180a0.json`、`range-1bf73-1c550.json` 保存 bytes、原名、檔案偏移與交叉參照；本次像素驗證與靜態矩形相符。其他素材的遮擋與繪圖路徑仍未知。

單張離線呈現試作契約：DRAFT → 證據審查 → READY。只用於檢查第一張 HD 素材能否留在原版已證實矩形，不定正式素材包或桌面架構。輸入原生 640×400 PNG、320×150 原素材及可選 HD PNG；必須先核對整個原素材與畫面矩形一致。HD 關閉、缺圖、來源不符、透明或比例明顯不符時保留原圖。輸出暫用邏輯座標等比放大，原版外框、文字及矩形外像素必須保持一致。驗收包含缺圖與錯誤來源備援、開關、輸入像素不變、界限與實際擷取畫面。這是離線 preview，不能宣稱桌面執行中切換已完成。

第一張 B 風格素材以內建 image_gen 參考 `A-0-original.png` 及比較圖的 B 區域生成，完整提示詞存 `workplace/hd-prototype/A-0-hd-b-v1.prompt.txt`。生成 PNG 為 1832×858，SHA-256 `503bbecf951fe5f8a8c30e797304433c62657d1906f461018440a9207f11a225`；提示詞 SHA-256 `3ac001b76201e555117ff650b524c9754f64d4bbb5a92b8e67ed7f1798534fac`。原圖內容與裁切作參考，生成細節是美術試作，不能當原版行為證據。圖像及提示詞留本機，由 `prototype.json` 索引，未加入 Git。

離線呈現子集達 CONFORMED。原生畫面 640×400 暫以兩倍邏輯座標輸出 1280×800，沿用包內 `dosbox.conf` 的 `aspect=false`，沒有另定 CRT 或桌面顯示模式。生成素材的比例因整數尺寸相差約 0.09%，僅在試作映射至已證實矩形，正式包尚未定規格。HD 預覽共 192,000 個像素變動，全部位於兩倍肖像矩形，矩形外零變動。原圖及缺圖輸出的 PNG SHA-256 同為 `4517ea850104f4b7e6b6a5589ad5f8ff746cbada976ce1f89ebc3858dde203bd`；HD 預覽為 `86dfb08019d186852a69d43996dbae326de8d6512f35cf7eb9288a33c06947a7`。手動目視確認只替換肖像，外框、徽章與下方原文保持原版。

預覽測試涵蓋關閉、缺圖、錯誤來源、比例不符、透明圖、畫面／倍率界限、輸入不變及完整矩形外核對。`preview-test.log` SHA-256 `2ea3c38d89344e1cc130a9a223a49c4d702b257a692123c6b4e2be2502e5de03`。IDA 指標交叉參照 `range-38bf0-38c10.json` SHA-256 `2d8c0b4fde99935634248353b28c96d4c1d988e4931b304f30534807891661e6`；複製端 `range-1bf73-1c550.json` 為 `d50027225833948b080dce1207902e09caea3e09b3d7b01490d4775cc3e6b1f8`；劇情圖呼叫端 `range-17fb0-180a0.json` 為 `4b78a751a88fc1c60c38fce399b8ddcbfe0ec2f35f92df03c283b4b0e76466f2`。桌面呈現、執行中切換、游標／更新範圍及正式素材包仍待後續 READY 契約。

## 2026-10-13：首張 HD 桌面試作契約

路由命中 dosgolem、平台規格、正常玩家路徑與文件職責。狀態：DRAFT → 證據審查 → READY，只授權可撤回的 Linux 桌面試作；不定正式素材包格式、跨平台交付或全遊戲相容性。使用者已確認 B 風格及原版玩法、桌面與可切換 HD 範圍。

- 沿用 dosgolem 固定 HEAD、原版 EXE／DAT 與原本的成功指令時間。既有步進迴圈只改成可分批呼叫，無頭仍連續跑；桌面 Update 分批執行相同 CPU.Step。GUI 繪製、切換與擷取不得略過原版指令或回寫 RAM／VRAM。正常輸入時間表仍在精確成功指令數套用。
- 固定既有 GUI image `sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4`，實查 Go 1.24.13、Ebitengine 2.9.9、Xvfb、xdotool、ffmpeg 與 gcc 已存在；不由 image tag／繼承標籤推定版本。依 [Ebitengine 視窗與邏輯畫面契約](https://ebitengine.org/en/documents/cheatsheet.html) 及容器內固定版本原始碼試作。新增依賴只寫輸出目錄的 modfile，不改外部 go.mod，不用主機 runtime 或憑證。
- 第一張只支援固定 A-0.OZM。觀測沿用已證實載入入口／返回與四平面 SHA-256，再核對原始 `sub_1C1BC` 的五個參數及相同堆疊返回。只接受來源偏移 0、寬 40 bytes、高 150、pitch 80、目的 `0xA0FB4`；新載入先撤銷資格。識別只讀 backing RAM，不能只靠整張截圖判斷。
- 畫面呈現另逐像素核對 320×150 原圖與原生矩形，遇到更新、遮擋或調色盤差異立即回原圖。HD 與關閉／缺圖備援共用既有離線呈現邏輯。原圖擷取 PNG 固定 SHA-256 `0b7a25670a568f2e8e61ecfdf4d7ea264dac13caf67dea7ddf9218b370f1239c`，生成 HD 僅從本機指定檔案讀取。
- 邏輯座標保留 640×400，試作畫布為 1280×800。Ebitengine 的 CursorPosition 先轉回畫布座標，再除以二傳給原版普通 MoveMouse／PressMouse／ReleaseMouse。離開視窗或失焦須釋放按鍵。使用輸入時間表時停用桌面滑鼠，避免驗收條件漂移；另測普通滑鼠路徑。
- 原生畫面與 HD 分開保存。游標沿用原版遮罩，最後才合成到呈現畫布；透明遮罩不覆寫 HD，XOR 色取原生底圖結果，仍標為既有 display-layer approximation，不宣稱 TSR VRAM 背景讀回已實作。
- F6 在同一原版狀態切換原圖／HD，F7 僅暫停試作步進；每次切換記錄步數、原版 RAM／CPU／原生色號證據及呈現 PNG。視窗狀態放標題，不把研究資訊畫進遊戲 HUD。關窗後保存完整收據，不改原版檔案。
- 驗收包括觀測的錯誤參數／堆疊、輸入不變、來源／缺圖備援、真實 Xvfb 視窗與鍵盤事件、同輸入的原版 RAM／CPU／畫面／PCM 核對，以及普通滑鼠縮放。音訊仍保存原版 PCM，本切片先驗證畫面與切換，不把桌面音訊或人耳確認稱為完成。

### 實作與驗證結果

本節的首張 Linux 桌面試作子集達 CONFORMED。只適用既定 EXE 的 A-0.OZM，未定正式素材包。原始程式、資料與外部 dosgolem 都唯讀；主機沒有啟動 GUI 或專案程式。沿用既有 GUI image，沒有建新 image。Go 1.24.13 與 Ebitengine 2.9.9、五個傳遞依賴由 `tools/desktop.mod`／`desktop.sum` 鎖定；建置使用輸出目錄的 modfile 與 Go overlay，外部 go.mod 不改。

| 已證實的觀測 | 原始定位與結果 |
|---|---|
| A-0 原版繪圖 | dosgolem 成功步數 137,089,680、線性 `0x1C1BC`，堆疊 `0x5C558`，五個參數 `[0,40,150,80,0xA0FB4]`。第 137,096,468 步自然返回線性 `0x1803D`，與原始堆疊加四相符，這時才開啟替換資格。沒有跳過繪圖指令 |
| 執行中切換 | 真實 X11 F6 在第 140,333,333 步開啟 HD，當時未暫停。第 141,666,666 步暫停後切回原版，再切回 HD。三次切換的 RAM、CPU、原生色號、調色盤及游標證據前後一致 |
| 真實視窗 | Xvfb 內實際 Ebitengine 視窗為 1280×800，xdotool 送真實鍵盤事件，ffmpeg 從 X11 擷取。HD、原圖、恢復 HD 三張都逐像素符合呈現 PNG，恢復 HD 與前次 HD 的 PNG 雜湊相同 |
| 缺圖備援 | 同樣的普通輸入與三次切換，HD 路徑不存在時回 `fallback_missing_asset`。原圖、開啟及恢復後的呈現 PNG 完全相同，實際視窗逐像素核對也通過 |
| 兩個 150,000,000 步的 GUI 路徑 | 全部 33 筆普通輸入照原步數套用。HD 與缺圖兩側的完整 RAM 均為 `0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f`，與先前原生無頭擷取相同；trace、原生畫面、DPMI、輸入邊界、CD 裝置與 PCM 均一致 |
| 普通滑鼠 | 無輸入時間表，真實視窗縮為 1000×700，含上下留黑。xdotool 的普通 X11 事件在視窗座標 `(469,306)` 對應原版 `(300,171)`，按下與放開分別記在第 85,000,000／85,333,333 步。從主選單進入顯示「時間是」的新遊戲開場，已人工目視確認。只驗證合成 X11 事件的新遊戲點擊，未測實體滑鼠硬體或完整角色建立 |
| 完整無頭回歸 | 步進重構後從原版入口重跑 750,000,000 步、390 筆普通滑鼠事件。全部 RAM、trace、原生戰場畫面、輸入與 CD PCM 同原完整基準，RAM 仍為 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`；四套件通過 |

滑鼠測試初次遇到 Go 空 slice 序列化為 null，驗證工具直接取長度而停止。修正工具接受空事件清單後，以同一 image、命令與資源限制乾淨重跑通過。未修改產品程式或遊戲判斷。初次 stdout／stderr 與修正前的驗證工具存於 `workplace/desktop-prototype/`，歷史版本的工具 SHA-256 與每次執行對應由下表總核對索引保存。縮放滑鼠測試的 1000×700 視窗與 1280×800 呈現圖尺寸不同，驗證座標與玩家動作，不宣稱兩者可直接逐像素相等。

| 收據／產物 | SHA-256 |
|---|---|
| `workplace/desktop-prototype/verification-summary.json` | `00826f14baa6b1f9c1e2707b77019efb2d7242bed6f5d0fa42bd9a0945017dcf` |
| `headless/startup-probe/latest.json` | `1eb3fe503bac5ecfb0261e0f94cf762a765895d5d4fb9b4a0e1e712ee210ac7c` |
| `desktop-hd.json` | `baac5b2b5923783f397eaeb3f4809a06ef1830b73a88121d327b0cfa3a46b97f` |
| `desktop-missing.json` | `a405e06ad8b13c54f7ba784b265c7e4b7b03a7f312a62c192199c43614e87729` |
| `desktop-mouse.json` | `292f10edd68193081741c14514fef60d9cc7bc798302b40aac766543a03636e5` |
| `gui-hd-enabled.png`，真實視窗 | `2d845709d0182e910855765564d9dd7dd7bcf3d32829d7bb263192636ba31c84` |
| `gui-mouse-after-click.png`，新遊戲開場 | `8d777a0d7e37b1ffb4dc3b4fd35eec4b73f2a574924e4505f60c1235ef24d6bd` |
| `tokugawa-desktop-prototype`，Linux amd64 中間產物 | `a36b9a83f0d10c3f5ed34f1a6bdb1fb819a091a3721c14288e6435e1610a53e0` |

原生畫面與強化畫面分開保存，正式 CDDA 及先前 HD 載入收據沒有被覆寫。所有圖像、RAM、PCM 與可執行中間產物留本機；只提交自製程式、依賴鎖定與文件。桌面音訊、一般鍵盤轉送、其餘素材與四語、其他 OS 及正式發行仍未驗收。


## 2026-10-13 頂端隱藏功能列契約

狀態：READY。使用者指定參考 `/home/anr2/cht/softworld_san` 的頂端隱藏列。路由命中 `local/retro-gui-restoration.md`，新檔職責依 `local/project-document-responsibilities.md`，參考專案只讀。

- 已證實：參考 `internal/ui/windowbar.go` SHA-256 `3ad9a9a6d779339c2f106215f5bec42f9f6e7aa2810162c7e318ba87dbdedf93` 與 `cmd/san1/windowbar.go` SHA-256 `67bfc46bbfc0cb742bd3a827ad62392381507d5b1c32ae4ba5f2c4cefa99c93f`。640 邏輯寬度、32 高功能列、24 高選單列，頂端 6 像素啟動。移開自動收起；Esc 固定／關閉，關閉時游標仍在頂端須先移開才重開。
- 已證實：參考 main.go 在原版輸入與轉場前處理功能列；顯示期間不推進遊戲。視窗增加功能列高度，遊戲維持原尺寸，下移顯示。
- 本作契約：沿用上述幾何與顯示方式。三個欄位為顯示、執行、工具，接現有原版／單張 HD、繼續／暫停、截圖／離開。保留 F6／F7／F9；Esc 改為固定／收起，Shift+Esc 離開。叉號只收起列。未實作的四語及音訊不放假開關。
- 原版隔離：顯示與收起的當幀、顯示期間、按住功能列按鍵直到放開，均不轉送滑鼠。顯示期間不呼叫 CPU 步進。HD／暫停／截圖操作前後核對原版 Proof；遊戲座標仍為 640×400。
- 視窗與繪製：固定呈現 2 倍，功能列顯示時邏輯畫布 1280×864，收起時 1280×800。視窗高度按目前顯示比例增減 32，保留最近鄰縮放。
- 字型：執行時讀取參考專案的完整 Unifont HEX，不改字模。來源 SHA-256 `20e8b505f602488697979eefc69857f7f6106bceab702f5ac559f4f84e0e7494`，原授權全文 SHA-256 `1e74cb82bf476843e97c2596297b04219b1a7e51f7238944a8c031cb9401fa87`，分別保存於 tools/toolbar-font.hex.gz 與 tools/LICENSE-toolbar-font.txt。功能列只取用所需字元；缺字拒絕啟動，避免空白文字。
- 驗收：Docker 單元測試檢查啟閉、選單命中、視窗幾何與字元涵蓋；Xvfb 普通 X11 事件檢查頂端顯示、移開收起、Esc 固定、滑鼠及鍵盤選項、點擊不滲入遊戲、縮放後遊戲操作。既有無頭／GUI 收據保留，不將輔助視窗測試稱為全遊戲相容。

### 隱藏功能列驗收

狀態：CONFORMED，範圍限 Linux 容器桌面試作。Go 1.24.13、Ebitengine 2.9.9、既有 GUI image；原版及外部 dosgolem 唯讀。四套件與功能列狀態、邊界、尺寸、字型涵蓋測試通過。

- 真實 X11 事件通過頂端展開、移開收起、Esc 固定與防止立即重開、滑鼠選單及 Tab／方向鍵／Enter、工具截圖、叉號收起、Shift+Esc 關窗。
- 未手動暫停時，按住叉號收起並移進遊戲區，原版成功指令數及滑鼠事件數保持不變；放開後才解除隔離。HD／暫停／截圖選項操作前後的完整 Proof 相同。
- 1280×800 視窗展開為 1280×864，1000×700 展開為 1000×750，收起恢復。功能列下方全部像素符合相同呈現圖，沒有擠壓遊戲。普通滑鼠模式不帶輸入時間表，功能列點擊未傳入原版；唯一左鍵對應原版約 (300,171)，新遊戲開場已目視確認。
- HD 與缺圖各跑至 150,000,000 步；完整 RAM、trace、原生畫面、CD PCM、輸入及邊界畫面符合先前無頭基準。缺圖選項停用，不假切換。完整 RAM SHA-256 `0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f`。

| 模式 | 原生收據 SHA-256 | GUI 核對 SHA-256 |
|---|---|---|
| hd | `02a9333ac63c97a64f746df04c1b84c2cc626892e0960e47f24e29bd89b334b6` | `a65e46738a4a9c0cccd7acf0006b9390e31ec9aa49520c80191d64a480a98313` |
| mouse | `f8ecf3c9dbb3ce40b80d742f72b8c38f619d541bef687444a4392c7cd913cc08` | `7773f69b7ad0a065866f299bea7c2181e7bc54408c0c6be2adb53ef5c23dce7d` |
| missing | `3d89dd63a8c3b2712397db47f0465fba4bd650c92e026c7578137d69891a8da4` | `6f3f86466e207340d6277f576bbb3fc42cf62dcbec5f35947c9abedadac74f7e` |

驗收總表：`workplace/desktop-prototype/toolbar/verification-summary.json`，SHA-256 `1e329edf8c1ca1c05b4584e08fa45142c9e06967797b98dfb7c9a3670a738ff1`。內含實際二進位、驗證工具、參考原始碼、字型及截圖雜湊。整合輸入逐項與目前檔案核對通過。原有完整 CDDA 及 750,000,000 步無頭收據保留；本輪沒有重新宣稱後續戰鬥、存讀檔、四語、桌面音訊或其他平台完成。


## 2026-10-13 劇情捲軸 HD 契約

使用者指定捲軸介面也要 HD，沿用 B 戰國手繪風與可切換原版的既有決定。路由命中 GUI 還原、素材考古及文件職責；使用 reverse-engineer-retro-game-remake 的證據及 READY 閘門與 imagegen 技能，不建立 remake 引擎。

已證實的來源：VISUAL.OZM 在 DAT 檔案偏移 2692940，長 81152 bytes，payload SHA-256 `d749447e8f9a19a172d7cc51e6bba5db2fc8017db5b4752844ee2ed22d87d4e9`。原版載入後四平面各 32000 bytes；普通輸入前綴 30 筆，固定 136769993 步擷取，停止在下一張 A-0 載入前。完整平面與載入返回的雜湊相同。

IDA 9.4 的 `range-1bf73-1c550.json` 保留原名、bytes 與檔案偏移：`sub_1C084` 在線性 `0x1C084` 起，依 `dword_38C00`、`dword_38BF8`、`dword_38C04`、`dword_38BFC` 的順序，每平面複製 `0x1F40` 個 DWORD 至 VGA `0xA0000`，最後於線性 `0x1C10A` 返回。這是全畫面複製，與肖像的 `sub_1C1BC` 分開觀測。

來源擷取比對固定原生 150M 畫面，在排除肖像 `(160,50,320,150)` 與文字安全區 `(80,268,510,114)` 後，149860 個背景像素與色號一致。捲軸取 `(0,250,640,150)`；本次 1009 個前景差異像素全部在文字安全區，界限 `(209,305)-(479,319)`。背景 PNG `scroll-original.png` SHA-256 `e8abcce3af8bc8bd4243ad27aecc0f8b07122813af23dd79d958650de04f6690`，完整底圖 `VISUAL-original.png` 為 `cd8d249121f87e9a86fa5d4aa7e51609a68ae3252864be46dcb9142cb3f1542b`。原圖、來源輸入與擷取收據都在 `workplace/hd-prototype/scroll/`。

狀態：READY，範圍限劇情捲軸的可撤回桌面試作。

- 維持 640×400 原版邏輯與原版文字、按鍵、選項位置。重繪紙張、捲邊及外框，文字與其他前景保留原生像素，不生成文字，不改翻頁或存檔。
- 原版 VISUAL 載入與四平面 SHA 通過後，監看 `sub_1C084` 的自然返回與 ESP+4，這時才開啟背景資格。其他全畫面複製立即撤銷；肖像載入不撤銷已畫好的背景。不得改 RAM／VRAM 或跳過指令。
- 呈現前核對捲軸 640×150 原背景。文字安全區外有任何變動就回原圖，以涵蓋動畫、遮擋與調色盤差異；安全區內與原底圖不同的像素逐點作前景，精確保留原色及位置。不得把動態前景當作已解析的文字語意。
- F6 與原有功能列一起切換肖像及捲軸。素材分項缺圖回原圖；只要一項 HD 素材有效即可啟用，記錄各項模式。原版模式完全保持既有輸出；游標仍在最後合成。
- 生成素材只作美術，不能反推正式幾何。保存完整提示詞及內建工具產物，原版及衍生圖都只留本機。原版上方紅框不列為本切片完成；四語與字型重繪另行實作。
- 驗收：來源／平面與背景來源、矩形外不變、前景精確保留、缺圖／透明／比例錯誤備援、不同原版文字頁、真實視窗 F6 與功能列操作，以及 150M 原版 RAM、trace、畫面與 PCM 相同。其他捲軸型別須另有來源證據才接入。

### 劇情捲軸實作與驗收

狀態：CONFORMED，限已查證的 VISUAL 劇情捲軸與 Linux 桌面試作。

- 原版觀測：`{'kind': 'scroll_draw_enter', 'linear_eip': 114820, 'logical_rect': [0, 250, 640, 150], 'return_linear': 98866, 'stack_linear': 378244, 'step': 136770000}`。
- 原版觀測：`{'eligible': True, 'kind': 'scroll_draw_return', 'linear_eip': 98866, 'step': 136770139}`。

內建 image_gen.imagegen 以完整 VISUAL 底圖及既有 B 肖像作參考生成 1586×992 畫布；只取像素 (0,620,1586,372) 為捲軸。上方生成紅框未接入。素材 scroll-hd-b-v1.png SHA-256 `b08c787b1431c2e890bf0dd33779a0a9b1551ebe531ec829450566404be5b488`；完整提示詞 SHA-256 `28b81238fdd2fcadc88a6aefad0b565ac1ad9b0e232b36e7136f6f19632b2a08`。來源與裁切由 `workplace/hd-prototype/scroll/scroll-art.json` 保存，全部原圖及衍生圖只留本機。

四套件、新背景自然返回、其他全畫面繪圖撤銷、矩形外與前景保留、來源不符及透明／比例／缺圖備援測試通過。真實 X11 視窗的兩頁原文，原圖／HD／恢復後逐像素符合呈現 PNG。第一頁精確保留 24324 個呈現前景像素，第二頁 4036 個；文字不是由 OCR 或生成圖重建。紙張與外框改變像素分別為 359676、379964，全部在捲軸矩形內。

| 模式 | 原生收據 SHA-256 | 核對 SHA-256 |
|---|---|---|
| scroll | `aac286718f3485b160a65779e2cdc9ebeabfe7b9cf7452befc36d85fd5368459` | `741ea13e84d574dd3e282a3c980db1104f27ea7d7ce3c68a847bba6da4e224f0` |
| scroll-missing | `05c7b3c58bb83aadb09b664f0f1e038725f03dde8b8048c8df7475b0474bb2ea` | `bd5a552509d050bec14a345586ab1f42c3a04b22bef72a70abb5034c73ea6462` |
| scroll-only | `5714fdcf2de3adc79089b15f61a941b6a533463bdbf4b2f988e5dd3a06dae49e` | `eed372bd920913c90765431c03f346513c74f524884075d502a9ff0e7e37dcad` |
| windowbar-hd | `a64e6783f757c1e4bd73f5a60b7a9b3673696c4477ea367c8a06ad2db11b0dfa` | `c3d5c93c52d224d799ff1e42d5fc2530ab50d41bdb5f7658e94ac72d892a305e` |

各模式均自然執行至 150,000,000 步；全部原版 RAM、trace、原生畫面、普通輸入與 CD PCM 符合原基準，RAM SHA-256 `0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f`。缺捲軸圖時只回原捲軸，肖像仍 HD；只有捲軸圖時仍可 HD，肖像回原圖。頂端列能切換兩項素材，顯示與收起手勢的原版隔離、視窗高度及同狀態核對保持通過。

驗收總表 `workplace/hd-prototype/scroll/verification-summary.json` SHA-256 `00bedb656362f6ff567015dbbec89450adbc8ed636bda3cb49508d5bd5c8986f`，記錄目前二進位、工具、素材、來源、截圖及各模式雜湊。外部 dosgolem HEAD 已前進，固定 commit 的獨立副本唯讀掛載，沒有放寬 HEAD／來源檢查或修改外部工作樹。既有完整 CDDA 及 750M 收據保持原樣。本輪不擴張為全部介面、完整 HD 包、字型重繪、四語或音訊完成。


## HD 素材覆蓋盤點與載入返回擷取

日期：2026-10-13。沿用已驗收的原版載入觀測契約，新增診斷輸出，不進正式素材包格式。狀態：READY。來源固定為本文件前述 EXE、DAT 與 dosgolem a9714eb；原始定位仍為 IDA 9.4 線性 0x1BF44 入口與各自堆疊返回、四個平面指標 0x38C00／0x38BF8／0x38C04／0x38BFC。

只在 -hd-export 明確開啟時，把載入返回已核對 SHA-256 的四平面合成索引圖；同時讀取 VGA 的唯讀 DAC 與色號轉接表。目的平面須在一般 backing RAM，尺寸受 80 byte 欄寬、400 列及 80 byte pitch 約束。輸出獨立寫至 workplace/hd-prototype/catalog/hd-assets/，保存名稱、返回事件、索引及 PNG 雜湊。載入時 DAC 不直接當作顯示時調色盤證據；實際呈現另與原生畫面逐點核對。相同來源重載不得覆寫不相同的索引或 PNG。開關對照須保持完整 RAM、原生畫面、CPU 停止點、輸入與 PCM 一致。這個切片不解釋未觀測的來源、透明度或繪圖層序。


### 劇情圖擴充契約

狀態：READY。沿用已證實的 sub_18002 劇情圖載入與 sub_1C1BC 五參數、堆疊返回及像素核對，只將來源限定從 A-0 擴充為正常路徑已擷取的 A-0 至 A-7。各來源須有自己的四平面 SHA-256、320×150 原圖及 PNG SHA-256。新載入撤銷上一張資格，原版繪圖返回後才啟用；原生矩形全部像素仍須匹配該來源。個別圖缺失、檔案指紋不符、幾何或不透明度錯誤，只回原圖，不借用上一張 HD 圖。A-0 既有旗標與備援保持相容；新來源暫用本機目錄，不定正式素材包 schema。

使用者已確認全遊戲美術，再打包 Windows、Linux、macOS。本切片只擴充已遇到的劇情圖，不代表全遊戲美術完成。


### KAO.GRP 頭像與呈現契約

狀態：來源格式已證實，桌面觀測／呈現子集 READY。IDA 9.4 線性 sub_1C6E8 的 0x1C70B／0x1C710 複製 index×0xC80 的 3200 bytes，來源指標 dword_39888。0x1C72B–0x1C749 將 byte-column+x、80×y 轉成 VGA 目的；0x1C74C–0x1C7BA 依四平面呼叫 sub_1C68D。後者在線性 0x1C6AA、0x1C6BB、0x1C6CE、0x1C6D2 以 80 列、每列 10 bytes、來源 pitch 10／目的 pitch 80 複製，故每張 80×80、每平面 800 bytes。原名、bytes、檔案偏移與 xref 保存在 catalog 的兩份 range 匯出與 art-consumers 匯出。

DAT 的 KAO.GRP 為 259200 bytes，共 81 張。正常 750M backing RAM 的來源 buffer 完整符合 DAT。最後 scratch 符合 index 76；角色建立的 105M、115M、120M、125M 原生邊界畫面均在 (96,86,80,80) 匹配完整 6400 像素。原生對照只證實第 76 張；其餘 80 張由共同已證實格式擷取，不能說已逐張在遊戲出現。收據與雜湊集中在 workplace/hd-prototype/catalog/kao/kao-extraction.json 及 native-boundary-matches.json。

呈現只觀測原版 sub_1C6E8 的三個參數與堆疊返回。index 限 0..80，目的矩形必須完整落在 640×400。所選原始 3200 bytes 須符合各自指紋，原版完成繪製後才登記矩形；相同目的的新繪製撤銷上一筆，原版全畫面複製清空頭像圖層。呈現時再核對該矩形全部原圖像素，遇到遮擋、調色盤差異、缺圖或非法圖像立即保留原版。可同時保存有界的多個目的矩形；不改 guest RAM、VRAM、CPU 或輸入。GUI 驗收使用既有正常前綴、第 76 張頭像與相同 150M 基準；圖像切換與缺圖兩側比較完整 RAM、原生畫面、PCM 及普通輸入。


## 2026-10-14 全美術盤點、八張劇情圖與 KAO 頭像驗收

本輪延續全遊戲美術 HD 與 Windows、Linux、macOS 本機打包決定。玩法與存檔仍由原版 DOS 負責。以下完成聲明只涵蓋實際驗證切片。

### 載入返回擷取與 OZM 解碼

狀態：載入返回診斷 CONFORMED。750M 普通玩家路徑擷取 17 個來源，與舊無頭基準的 57 個共用欄位及完整 RAM 一致。RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`；收據 `catalog/startup-probe/latest.json` SHA-256 `13d5602ab77bef5b2aa55843f851381503bbb13a06d57ba78eb0381c7c59b98e`。共用欄位清單及刻意排除的工具／輸出路徑欄位保存在 `catalog/export-verification.json`。

`tools/hd-ozm-probe.py` 依 IDA 9.4 的 sub_28689 與 sub_2ADC8 原始 bytes 解碼 107 個 OZM；17 個正常路徑來源的四平面與索引指紋全部吻合。B-6、B-7、C-10、E-4、OP-FULL 的最後一個平面均少一 byte。該 byte 可能來自原版讀取 buffer 的剩餘內容，語意為未知；未猜值補齊。13 個 VGA 檔包含 Big5 劇情與控制碼，不能只按副檔名算成美術圖集。解碼結果及輸入雜湊集中在 `catalog/decoded/inventory.json`。

### 八張劇情圖及頭像呈現

狀態：A-0 至 A-7 與正常角色建立的 KAO 76..79 呈現切片 CONFORMED，限 Linux 容器桌面。KAO 其餘索引的格式與共同繪圖端已證實，尚未逐張在正常玩家路徑抽測。

九張手繪圖集由內建 image_gen.imagegen 生成，各頁原圖、完整提示詞與生成產物均留本機。實際格線不完全等距，`tools/hd-kao-prepare.py` 依檢視後的切線裁成 81 張獨立圖，再補黑底保持比例。每張記錄原始索引、來源及 HD SHA-256、圖集、裁切與 padding；收據在 `catalog/kao/selected/manifest.json`，總覽為同目錄 contact-sheet.png。生成圖不作遊戲座標來源。

四個 Go 套件通過，包括 81 個原版來源指紋、錯誤索引、錯誤堆疊、原版繪圖返回、全畫面重繪撤銷、原始 bytes 改變及一個前景像素遮擋時的備援。桌面只能在原版返回及來源像素匹配後覆蓋；原版游標最後合成。A-6、A-7 的圖片內地名保留原版字形。

| 模式 | 原生收據 SHA-256 | 真實視窗核對 SHA-256 |
|---|---|---|
| HD | `873b16e7d3e608304d947d3e320f626053197eb674f7b2ee0b3050848bc8cfd8` | `2fb1cf28b56af9d9f33aae3a564b4d33e07c4f07f7feaea4e3946a66ec4086ce` |
| 缺圖 | `90d2c77d9efe3b04ca120f9d4bb7f55746128191ac0c46a38e905113c1df796e` | `14d374cf36e8cbcdb6519a22d4abad0a8c0ca851eefb4d7816ad5456e757e7af` |

兩側各自然執行 750M、390 筆普通滑鼠輸入，19 次切換均不改原版 Proof。完整 RAM、原生畫面、trace、中斷、DPMI、PCM 及輸入邊界畫面符合既有基準；整合來源雜湊也與建置輸入相符。真實 X11 畫面逐像素等於呈現 PNG。四張頭像在 103M 的替換矩形分別為 `(96,86,80,80)`、`(208,86,80,80)`、`(320,86,80,80)`、`(432,86,80,80)`，改變呈現像素為 25091、24982、23562、24758；矩形之外完全不變。收據與驗證器快照分別在 `catalog/kao-desktop/`、`catalog/kao-desktop-missing/`。

勘誤：最初在 105.1M 普通點擊之後抽樣，確認框已遮住頭像，來源不符備援正確生效。改在點擊前的正常畫面抽樣，不放寬像素檢查。啟動時 null 狀態、未複製 engine 來源、過短固定按鍵及外層工作階段中斷皆屬驗證工具／環境問題；已修正並乾淨重跑。長流程改用有界的 detached Docker，未改原版指令或虛擬時間。

### 素材覆蓋與未完成項

`tools/hd-art-inventory.py` 重生唯一的 `catalog/art-coverage.json`。目前有 36 張劇情手繪草稿，解碼索引完全相同的來源共用後對應 45 個 OZM；另有 81 張頭像及已接入的捲軸。製作完成、共同繪圖端已接入、正常路徑抽測分開記錄。地形、單位、其他介面、動畫、結局、出版商圖仍待製作及接入。圖集內部還需細分，不計全遊戲完成百分比。

未接入的草稿仍須審查構圖、烘製文字與尺寸。例：E-1 草稿出現新增星點，D-2 原圖寬 328，需在接入前處理。不得把檔案存在當成正式素材完成。

### VISUAL 上方框架契約

狀態：READY。沿用已證實的 VISUAL 四平面、sub_1C084 自然返回及背景資格，不新增 guest 攔截或寫入。原版上方矩形為 `(0,0,640,250)`；中央 `(160,50,320,150)` 始終保留既有劇情圖層。

- 原圖取已核對的 VISUAL-original.png 上方 640×250，PNG SHA-256 `7686cb21cbf195893ac0f75e495fa304446fd6d1e4f41f1d78300890ffdb9116`。
- 原版背景資格成立且中央以外全部原生像素與來源匹配時，才替換框架。任何新前景、調色盤差異或未知來源均回原版；不蓋中央圖、不改捲軸文字、輸入及原版記憶體。
- 使用已生成的同一 VISUAL 手繪底圖，按原版矩形分區縮放。v1 裁切線未對齊紙張，保留為未使用草稿；v2 對齊原版中央矩形。準備工具為 tools/hd-frame-prepare.py，素材及來源格線在 workplace/hd-prototype/scroll/frame/frame-art-v2.json；HD PNG SHA-256 `e38ddf2ba19ae73e06f1906d06467e7a6f8571ef9e6fe261a447aa203098c209`。
- 框架與其他素材各自缺圖備援，共用 F6 與頂端列。驗證來源、中央及矩形外不變、遮擋、幾何／透明度錯誤、真實視窗原版／HD／恢復、缺圖，以及相同 150M 普通輸入下的完整原版狀態。


### 2026-10-14 上方框架驗收與視窗來源

VISUAL 上方裝飾框架狀態：CONFORMED，限 Linux 容器桌面。四套件、中央圖層及下方捲軸不變、原圖前景遮擋與透明圖備援測試通過。

- 完整 HD、缺框架圖、只有框架 HD 三種真實視窗通過。正常替換 447881 個呈現像素；只有框架時，中央與下方共 576000 個呈現像素全部不變。缺框架圖時只有框架回原圖，其他 HD 圖層繼續呈現。
- 只有框架素材時，頂端列仍能切換 HD，視窗幾何、操作隔離及原版狀態核對通過。
- 四次 150M 的完整 RAM、原生畫面、普通輸入、trace、DPMI 與 CD PCM 符合既有基準。總表 `workplace/hd-prototype/scroll/frame/verification-summary.json` SHA-256 `8ed91a4a01b41782ba4d85e7f12ef9435e28ac6504f5af422e602c89e098fbc0`，記錄二進位、驗證器快照、收據、素材及原版輸入核對。

共用視窗來源已證實，尚未接入桌面呈現。IDA 9.4 原始輸出在 `catalog/range-1af00-1b165.json` 與 `range-1b165-1b300.json`。sub_1AFB0 初始化 WINDOW.OZM，原始全域 dword_35153 指向四個各 288 bytes 的平面。sub_1B00C 呼叫 sub_1B0E1，EAX 選 0..8 圖塊，EBX 為 VGA byte offset。sub_1B134 每列 movsw，來源 pitch 18、目的 pitch 80，共 16 列。sub_1B165 保存原版畫面，sub_1B245 還原，保留原名與原始定位。

原版 backing RAM 的 1152 bytes SHA-256 `26d936f2a599e4c6a552de7a07a890e8ccbec3d51750e22383c7f2fde8ce9e50`，四平面分別符合正常載入指紋。105M 正常角色建立畫面中，標題視窗 (80,20,128,48) 的 18 個邊框圖塊與一個空白內頁圖塊逐點匹配，共核對 4864 個原生像素，涵蓋全部九個圖塊的 2304 個不同來源像素。來源收據為 `catalog/window/window-extraction.json`，工具 `tools/hd-window-extract.py`。實際繪製呼叫與自然返回仍待桌面觀測，不從靜態圖塊存在宣稱整合完成。

### 2026-10-14 生成圖片的 private repo 保存授權

使用者明確指定「目前這些圖片都很棒 記得保存在 github repo 當中喔」。依新指示，生成 HD 圖及由生成圖裁切的素材、總覽與提示詞可保存於 assets/hd-b。這項授權取代先前對生成圖僅限本機的專案規則；先前研究紀錄保留當時邊界。

保存工具採明確來源清單，拒絕覆寫不同內容的圖片；逐檔記錄大小、SHA-256、尺寸、提示詞與裁切依據，核對索引完整性。目前包含 139 張 PNG、54 份提示詞，原始圖檔總量 182357874 bytes。索引是開發素材來源保存，不是正式素材包 schema。原版 EXE、DAT、光碟、音軌、擷取原圖、PCM 與執行收據維持本機。新目錄已掛入 AGENTS 與 README。

## 2026-10-14 共用視窗 HD 契約

狀態：DRAFT → 證據審查 → READY，範圍限原版 WINDOW.OZM 的九圖塊視窗呈現。輸入沿用固定 EXE SHA-256 `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a`、DAT 與 dosgolem a9714eb。IDA Pro 9.4 的 LE loader 線性 EA 與 dosgolem 本作線性位址一致；檔案偏移仍分欄保存。證據入口為前節兩份 range 匯出及 `catalog/window/window-extraction.json`。

已證實：`sub_1B00C` 的四個堆疊參數依序是 byte-column x、像素 y、圖塊欄數、圖塊列數。IDA 線性 `0x1B016..0x1B024` 計算 `x + 80*y`；`0x1B03E..0x1B04C`、`0x1B05F..0x1B0A0` 的迴圈按欄列減二拼九圖塊，每塊 16×16。寬高不滿三塊不列為支援範圍。`sub_1B165` 於 `0x1B181` 增加 `dword_35157`，以增加前的值選擇 `unk_34ED3` 的 16-byte 記錄；`sub_1B245` 於 `0x1B24F` 先減少同一計數再還原該記錄。原始 bytes 與 xref 保留於 range 匯出。

已證實：原始四平面共 1152 bytes，SHA-256 `26d936f2a599e4c6a552de7a07a890e8ccbec3d51750e22383c7f2fde8ce9e50`。144×16 原圖 PNG SHA-256 `8a4c3c581d76b131644f69733b61783fafde14541263cfde1d3dfe594d66978b`，全部九個圖塊已與正常角色建立的原生畫面核對。頭像框實際位於 y=68，頭像 y=86..165，故下緣圖塊也可能含前景。HD 圖為既有 B 風格 1254×1254 九宮格，SHA-256 `ae2dbf50f7d00cc6d6617bf50911aa05ef461cb1f937f1cf3249b0d3838527f5`，已保存於 `assets/hd-b/window/`。

- 觀測只讀 `sub_1B00C` 的堆疊、原始全域及 backing RAM。來源指紋通過，且矩形完整位於 640×400、欄數 3..40、列數 3..25 時才登記待返回。自然返回位址與 ESP+4 同時符合後才啟用，返回時再次核對來源。不得跳過指令或寫 guest。
- 畫面保留有界的 64 個視窗。相同矩形重畫替換舊記錄；全畫面 `sub_1C084` 撤銷目前視窗。`sub_1B165` 自然返回並確認計數增加一後保存呈現狀態；`sub_1B245` 入口暫停視窗資格，返回及計數減一符合後還原對應狀態。保存層數上限 64，未知／損壞來源及未追蹤還原均回原版，不推測遺失狀態。
- 原圖檔案 SHA 與尺寸須符合，HD 九宮格須不透明、正方形、邊長可被三整除，尺寸受 8192 上限約束。預先切成九個圖塊，再依原版矩形重複拼接，不由生成圖決定遊戲座標。
- 呈現前以原生畫面核對四個完整角塊，任何遮擋或調色盤不符就保留原版視窗。其餘位置與預期來源圖塊不同的像素，包含文字、頭像及邊框前景，均明確覆回原生像素；其他背景才使用 HD。視窗按繪製順序處理，頭像及游標在其後合成。備援時恢復該矩形原生像素，避免先畫的 HD 圖層覆住新前景。
- 原版模式、F6、頂端列、滑鼠位置與存檔格式不變。只有視窗素材時也能啟用 HD；缺圖只停用這一項。此切片不解析文字、不變更資料格式、不改存檔。未觀測的其他介面不外推完成。
- 驗收包含原始來源與堆疊返回、保存／還原生命週期、來源損壞與非法矩形、前景及矩形外不變、缺圖及透明度、真實視窗切換、確認框出現及關閉。正常 150M 路徑比較完整 RAM、原生畫面、trace、輸入與 CD PCM；原版收據由 dosgolem 重生。圖像保存沿用使用者對 private repo 的明確授權，原始資料及執行輸出保持本機。

### 共用視窗實作與驗收

狀態：CONFORMED，限 WINDOW.OZM 九圖塊繪圖端與 Linux 容器桌面中已抽測的角色選擇、確認及能力視窗。未外推其他平台、其他圖集或全遊戲 HD 完成。

- `tools/hd-window.go` 只讀來源、堆疊、保存層數及自然返回。四套件通過，涵蓋原始 buffer 指紋、錯誤堆疊、非法矩形、保存／還原層數、全畫面撤銷、返回前來源損壞、角塊遮擋與輸入不變。呈現測試逐點核對九圖塊重複拼接、矩形外不變及邊框上的前景。
- 角色選擇五個視窗的正常繪製參數與先前原生像素證據一致。只有視窗 HD 時改變 123407 個呈現背景像素，保留 97120 個文字／頭像前景及 798720 個矩形外像素。搭配頭像 HD 時四張人物繼續替換；缺視窗圖時邊框完全回原圖，四張頭像仍為 HD。
- 普通輸入開啟 `(200,150)-(504,198)` 確認框，逐點保留 2524 個前景像素；確認後的六個視窗保留 66080 個前景像素。原圖／HD／恢復的原版 Proof 相同，恢復後 PNG 與先前 HD 相同，真實 X11 擷取逐像素符合呈現 PNG。已目視核對文字、框架及人物。
- 只有視窗素材時，頂端列能啟用 HD；展開、固定、收起、鍵盤操作、暫停、截圖、按住收起手勢隔離及縮放比例通過。驗證工具首次未建立輸出子目錄而提前停止，保存失敗輸出後補上工具的目錄準備，以同一 image 及命令重跑通過，沒有修改產品或原版來繞過失敗。
- 五次均自然執行 150M、33 筆普通滑鼠事件。15 個原版欄位及全部 RAM 與既有基準一致；每次記錄 26 次保存返回、24 次還原返回，無待返回觀測。RAM SHA-256 `0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f`。舊 750M 完整收據保留，本輪未重跑全程，不將本次 150M 驗證稱為新的全程驗收。

總表：`workplace/hd-prototype/catalog/window/verification-summary.json`，SHA-256 `555a70005c3f15d4f9ddc72fe7f17146da2f6d385f17a6c9a236f2579348d94f`。由 `tools/verify-hd-window.py` 核對各次原版欄位、完整 RAM、建置輸入、二進位、驗證器快照與截圖；共同二進位 SHA-256 `13f7ae1da7eb8297e84e5f6b97e65dd9feb688c00ebbab794311ac8b95f75a46`。

| 模式 | 原版收據 SHA-256 | GUI 核對 SHA-256 |
|---|---|---|
| 只有視窗 | `d8c1e9dcf694775b08288bd7de29888fd3e688e907f7d9e9f44dc796a06710ce` | `6902f3767d2857e51bcff1ef40837426a35a8f3c76d8c7a9f673bc4297802075` |
| 視窗加頭像 | `324e3912d4f57b74a50ae5ca2474df64dfabcbf182bce183b6e3512e9d436d0e` | `f71fe3f38f334efb6202bd48e99448812a75e8c5168c9cbb529d3fad58dd6178` |
| 缺視窗圖 | `2a34758bcc413ee907a96d6a7ff61f71f1f88844a9f951ac8e5f461753ffba67` | `f0db1503a33df5dba202ccbd3b94c9929cdee8c244eb450ce400292f3ecafe13` |
| 確認及能力視窗 | `6dba8a45ae9f747e88760b003d6f5bb0933992555e12934ce8bcc79ffe06c70b` | `ff6d29038d64be6ee0ef683c186ea9a2c12c6d9291d396c3a021f2ef5a12469a` |
| 只有視窗的頂端列 | `518e8d7c26573c2e04ad35b06c358bf4d41a8d5295e8e0c1e16118f0a93f53c5` | `4e76a3f4f888a090e6a1c516e43ee8b0ce7104320eb471ca69e262b4c4555c3b` |

## 2026-10-14 劇情來源批次擴充

狀態：DRAFT → 證據審查 → READY。本切片沿用原版劇情呈現，擴充已製作且經構圖審查的素材；不新增玩法、存檔格式或正式素材包格式。輸入 EXE、DAT、dosgolem 與 IDA 9.4 版本沿用前述固定雜湊。靜態解碼證據仍為 `range-28660-28860.json`、`range-2adc8-2af40.json`，17 個正常來源已核對四平面與索引。

已證實：`workplace/hd-prototype/range-17fb0-180a0.json` 的 IDA 線性 `0x18018..0x18038` 以選定資源載入，再固定呼叫 `sub_1C1BC(0,40,150,80,0xA0FB4)`，返回於 `0x1803D`。`range-1bf73-1c550.json` 的線性 `0x1C27E..0x1C28C` 每列複製 40 bytes，來源 pitch 固定 80，目的 pitch 由參數給出。故 328×150 來源在此路徑只顯示左側 320×150，右側八個像素不進目的矩形。這是原始指令證據；未將未走到的劇情稱為正常路徑驗收。

- 來源準備工具重新核對固定 DAT 的 payload、獨立解碼四平面、索引與尺寸，再產生本機原圖及編譯期對照表。來源限定已審查的 42 張劇情及其完整索引相同的資源；40 或 41 byte 欄寬、150 列，其他尺寸與五個未完整解碼來源繼續拒絕。精確共用須完整四平面及索引一致，不只比較 RGB。
- 保留原版載入、自然返回及逐像素資格核對。每個來源有自己的名稱、欄寬、四平面指紋及 320×150 可見原圖；41 欄來源只在顯示副本裁掉最右八個像素，完整來源指紋仍包含 41 欄。原版繪圖參數仍固定，不修改原圖、RAM、VRAM、CPU 或輸入。
- A-0 至 A-7 的既有來源與像素檢查保持有效。未在正常玩家路徑走到的來源，只能記為「來源與共同繪圖端已接入」；不得增加正常路徑驗證數。驗證範圍分開記錄。
- A-6、A-7、B-1、B-2、C-5、C-8、E-2 的圖片內文字在經目視核對的原始矩形逐點保留，其精確重複資源沿用相同矩形。這些是原生字形／圖牌保留，不是已完成的文字語意解析。其餘已審查草稿沒有圖片內文字；來源及 HD 對照在 `catalog/story-review-1.png` 至 `story-review-3.png`。
- D-2 使用對齊可見 320×150 裁切的 v2；E-1 使用移除草稿新增星點的 v2。修正前 v1 保留為草稿，修正版須檢視構圖、不透明度、比例及雜湊後才加入對照表。生成圖與完整提示詞依既有授權保存 private repo；原版參考與研究對照留本機。
- 每個已接入來源測試原始資料／平面、模擬原版載入與自然返回、錯誤欄寬／指紋、缺圖、遮擋、矩形外及字牌保留；這些是來源與呈現測試，不冒充完整劇情實玩。另以既有正常入口至首戰的 750M 路徑抽測八張劇情圖，核對完整 RAM、trace、原生畫面、普通輸入與 PCM，確認新增對照表不改原版狀態。
- 程式審查確認 `hdLive` 目前依診斷陣列長度取得載入返回；診斷達 256 筆後不再追加，會使後續 HD 資格失效。來源為 `tools/hd-live.go` 的 `previous := len(h.Trace.Events)` 與 `tools/hd-trace.go` 的有界 `event`。改用獨立的最後返回事件及序號驅動呈現；診斷上限、丟棄數及原版執行維持原樣。增加滿額後仍可更新與來源不符撤銷的回歸測試。這是呈現記錄的缺陷，不擴張逆向範圍。

### 53 個來源對照與正常路徑驗收

狀態：來源與共同繪圖端切片 CONFORMED。53 個來源由 42 張審查後素材對應，包含 D-2 的 41 欄完整指紋與左 320 像素顯示裁切。其餘 45 個來源未在正常劇情逐張走到，保留此限制。工具與本機對照表為 `tools/hd-story-prepare.py`、`catalog/story/profiles.json`；編譯期資料為 `tools/hd-portraits.go`，仍非正式素材包格式。

- 四套件與 53 個實際來源的載入、自然返回、錯誤指紋／欄寬、原圖、HD、地名、缺圖與遮擋測試通過。測試使用原始索引重建四平面，未將模擬堆疊當成正常玩家實玩。診斷滿額後兩次返回仍能更新資格與撤銷不符來源，輸入記憶體保持不變。
- HD 與缺圖各自然完成 750M、390 筆普通輸入。15 個原版欄位及完整 RAM、原生畫面、trace、PCM 符合既有基準，每側 19 次切換均不改 Proof。八張劇情與四張頭像的真實 X11 畫面符合呈現 PNG；原有地名仍精確保留。全部 53 個來源的原圖均載入，HD 模式的 53 個素材指紋符合對照表，缺圖模式只保留 A-0 的既有個別入口。
- 完整 RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`。共用二進位 SHA-256 `a14af2222b7bfa1c4b5c5e135adb5e6958a83b22905fcbd1133365d08039e058`。獨立總表 `catalog/story/verification-summary.json` SHA-256 `af2316dba2e2b79b7331b2739738943e94aa47bb174935d7a711ddc6e14f96e7`，由 `tools/verify-hd-story.py` 重核各次輸入、原圖、實際素材、二進位、驗證器快照與截圖。

| 模式 | 原版收據 SHA-256 | 真實視窗核對 SHA-256 |
|---|---|---|
| HD | `624093d9de2554180a073f8f0b3da5fc01cac0742af73781b989b363215233ca` | `4839dbeb7079d692ca9d77709831d3be4161b64d8881503b655bc0c21868ff5a` |
| 缺圖 | `3a80c4bda5df1282005f1ce462872ae393e248cf136d103553fbde2b71d11219` | `fb2788c4a729e556844090b673349cc4b4de222417197bb5423445b060bf96bf` |

### 新增圖與修正版

內建 image_gen.imagegen 完成 D-2 裁切構圖 v2、E-1 去星點 v2，均經檢視後進入本輪來源對照。另製作 18 張城郭、軍議及結局草稿，對應剩餘 22 個可完整解碼的 320／328×150 資源。新草稿僅做構圖與不透明度檢視，未加入 53 個來源對照，不計正常劇情驗證。比較圖為 `catalog/remaining-story-hd-review.png`，原版參考留本機。

GUNGI v1 將背景金像畫成人物，已另製作 v2 恢復金像並保留四位坐姿人物；v1 保留草稿。各版 PNG、完整提示詞及生成記錄保存於 `assets/hd-b/art/`。本機來源為 `catalog/art/revisions.json`、`remaining-generation.json`、`gungi-revision.json`。新增草稿接入時，F-6、H-1、H-2 的原版字牌須保留；E-5、F-3、YOULOSE 使用左側 320 像素裁切。GUNGI 選用 v2。地形、單位、動畫及其他介面仍待後續，不因本批圖存在宣稱全遊戲 HD 完成。

## 2026-10-14 其餘可解碼劇情來源接入契約

狀態：DRAFT → 證據審查 → READY。沿用上一節固定輸入、共同繪圖端與來源核對契約，加入已審查的 18 張生成圖及四個完整平面／索引相同的別名，總計 60 張選定素材、75 個來源。END-8 與 G-4 雖然 RGB 相同，索引不同，維持各自來源及素材。GUNGI 選 v2。E-5、F-3、YOULOSE 保留完整 41 欄指紋，顯示左 320 像素。

新增城郭字牌於 F-6、H-1、H-2 的原圖矩形 `(246,109)-(307,139)` 保留，F-6 的完整重複 G-3 沿用。目視與逐列像素核對顯示金色外框 y=110、137，黑色外緣 y=109..138。審查圖在 catalog/story/castle-label-review.png。不重新繪製或猜測字形。

輸出在既有 catalog 下的 story-expanded，與 story 的 53 來源及 GUI 收據分開。新來源逐項跑既有載入、返回、指紋、裁切、字牌、缺圖及遮擋測試，再跑正常 750M GUI 對照。GUNGI 的正常載入已記錄於 step 570613558..570928304；是否能於正常畫面替換須另核對繪製參數及真實視窗，不以載入來源當成呈現通過。尚未抽樣的來源保留正常路徑未知。

### 75 來源實作與正常路徑驗收

狀態：來源與共同繪圖端切片 CONFORMED。75 個來源由 60 張審查後素材對應，來源準備、原始平面、自然返回、錯誤指紋／欄寬、可見裁切、字牌、缺圖及遮擋測試通過。正常路徑實際抽測 A-0 至 A-7 及 GUNGI，共九張；其餘 66 個來源保留正常路徑未知。

HD 與缺圖各自然完成 750M、390 筆普通滑鼠事件。15 個原版欄位、完整 RAM、原生畫面與 PCM 符合既有基準，每側 21 次切換均不改原版狀態。九張劇情／軍議及四張頭像的真實 X11 擷取符合呈現 PNG。GUNGI 實際抽樣步數 575M，HD 改變 191897 個中央呈現像素，缺圖改變零；矩形外不變，背景金像構圖已目視核對。此模式以旗標停用上方框架及 WINDOW 素材，隔離中央圖像核對；它們既有獨立驗證維持原範圍。

總表 `catalog/story-expanded/verification-summary.json`，SHA-256 `6f64f6dde7e94f21f1782c5388c073318a8020d4a03b330c8d391baabf229caf`。原版 RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`；共用二進位 SHA-256 `0f23dbd91550a996c2c72a52db43b80f7994a072f5fbadb2329d92cb98696d28`。

| 模式 | 原版收據 SHA-256 | 真實視窗核對 SHA-256 |
|---|---|---|
| portraits | `589639eb8b0529d59303d5064db779033e7de4915d12d119960329f8bce1f070` | `b0e7498231c9364f79a239f4cd1a4d1d2d7a1115147245705a7c93638ae36a9e` |
| portraits-missing | `194c113e42dd05933c0bad44dc1ad8ad5b6822ede26136daeb7d527679996fa6` | `57a0eb4a00c56a54fe15fc25d053acd079ac5f3b8df90eaafd8eb865e432b9e6` |

## 2026-10-14 地形圖集的最小細分證據

輸入沿用固定 EXE、DAT 與正常 750M RAM。工具 IDA Pro 9.4，位址為 LE loader 線性 EA，檔案偏移另存於原始匯出。`asset-xrefs-resources.json` 以 get_fileregion_ea 查證 MAP_CHIP 字串位於 IDA EA 0x32381、EXE 檔案偏移 0x2CB81，經 0x3543C 指標表引用。其他資源同理，不套用程式區段的固定差值。

已證實：`art-consumers-35988-3598c-35990-42168-4216c-42170-42174.json` 的 IDA 0x1E962／0x1E974／0x1E986／0x1E998 保存四個 0x8000-byte 配置。正常 RAM 的原始全域 0x42174、0x42168、0x42170、0x4216C 指向的各 32000 bytes 與 MAP_CHIP 獨立解碼及正常載入四平面完全一致。來源全域 0x35988／0x3598C／0x35990 的值為 0／0x5000／0x6E00，來源 pitch=80，對應圖集 y=0／256／352。原名與 operand 保留。

已證實：`range-1fc00-20040.json` 的 IDA 0x1FD42..0x1FD68 與 0x1FDF7..0x1FE25 依索引除以 13、列乘 0xA00、欄乘 6 計算地面來源。0x1FD01 的拷貝迴圈為 32 列，每列三個 word，加 0x4A 後 pitch=80，寬 48 像素。物件及單位的列距 0xF00 見 0x1FE30..0x1FE9D；`range-1f8d2-1fc07.json` 的 sub_1F9D0 迴圈 48 列、三個 word，同樣寬 48。它將四平面 OR 後取反作目的遮罩，再 OR 原始平面，故僅索引零透明。

來源細分為 104 個 48×32 地面格、26 個 48×48 物件格、13 個 48×48 單位格，含空白；這是圖集的完整可見格線，實際遊戲使用索引集合仍未知，未推定所有格都會使用。工具 tools/hd-terrain-extract.py 逐項重建索引、平面與透明遮罩，產生本機原版格、總覽及 terrain/terrain-extraction.json。不是 HD 美術、正式素材包格式或正常操作驗收。正式 HD 接入契約尚未 READY，沒有改 production path。

地形細分收據 SHA-256 `d007176ba26ef50fe77da68af02a7c61002188833c6a5f1571a187c7b218bbc9`，記錄 Python 3.11.2、Pillow 9.4.0、IDA image、工具及各份 IDA 證據雜湊。104 個地面格均非空，26 個物件格中 24 個非空，13 個單位格中 10 個非空。空白格仍保留原始索引，未假造圖像。

## 2026-10-14 戰場地面呈現研究

狀態：DRAFT。先用獨立 hd-terrain-trace 記錄原版 sub_1FC07 的 ESI／EDI、原始旗標、四平面與自然返回，只讀 backing RAM。沿用固定 EXE、DAT、dosgolem 與普通 750M 時間表，核對完整原版狀態後才決定地面 HD 呈現契約。診斷不接管桌面或寫入 guest。

### 地面正常繪製與調色盤

已證實：正常 750M 的唯讀觀測記錄 134 次地面入口與自然返回，沒有丟棄或待返回。8 個來源索引為 0、2、6、13、15、22、62、67，兩個原始遮罩旗標皆零。15 個原版欄位、完整 RAM、原生畫面及 PCM 與既有基準相同。總表 terrain/trace/verification.json SHA-256 `e667d6d7e70bd7d26ac6eaef894da3536a065880c8478ecadc03bba09a6e3914`，探針當時全部輸入另存 source-snapshot。初次驗證腳本誤用 startup-probe/startup-memory.bin 路徑；核對程式實際寫入的根目錄檔名後同容器重跑通過，屬驗證路徑錯誤。

載入 MAP_CHIP 時的 DAC 與最後戰場不同，三個草地色的 RGB 各相差一個 6-bit DAC 階。因此早期原圖零個完整矩形匹配，不用亮度偏移補過。改以正常 A-0 載入時實際記錄的完整調色盤重建來源，再對既有原生戰場逐點核對：65 個未互相重疊矩形中 50 個完整匹配，涵蓋 0、2、6、13、15、22。其餘被物件、人物或介面覆住。收據 terrain/trace/native-floor-palette-verification.json SHA-256 `686a1d08341483bb8cccec8b8bdd6e98b6a2618a093f82203e01a5058926e206`。62／67 曾正常繪製，但末幀已被後續原版繪製覆蓋，不計末幀 HD 驗收。

### 地面 HD 呈現契約

狀態：DRAFT → 證據審查 → READY。玩家範圍限原版地面 sub_1FC07 已自然繪製且目前仍完整可見的 48×32 矩形，使用已審查的 B 風格素材。原始輸入與 IDA 9.4 位址空間沿用前節，存檔、規則、座標及輸入不變，研究對照表不定義正式素材包格式。

- 來源 ESI 必須能以 pitch=80、13 欄、每格 6 bytes×32 列解成 0..103 索引，原始地面基址全域 0x35988 必須為零。EDI 以 pitch=80 解成螢幕 byte-column 與 y，完整 48×32 矩形須在 640×400。四個已證實的 MAP_CHIP buffer 各 32000 bytes 需符合完整指紋，任何界限、指紋或堆疊不符均撤銷資格。
- 只讀 CPU 暫存器、堆疊與 backing RAM。原版遮罩分支兩個旗標同時非零時不替換。自然返回 EA 及 ESP+4 同時符合，且來源重新核對後才登記。相同或重疊矩形的新繪製撤銷舊項；最多 256 項。全畫面 sub_1C084 或地形配置入口 sub_1E93F 撤銷，不猜測其他恢復流程。診斷上限與呈現狀態分開。
- 104 個來源以原始索引和已核對的正常調色盤產生獨立本機 PNG，編譯期保存 SHA；原版初次載入擷取保持不變。HD 只接受已審查來源的固定檔名及 SHA、不透明、比例 3:2、尺寸上限 8192，預先縮放至顯示尺寸。缺圖或錯誤只停用該來源。
- 呈現前整個 48×32 原生矩形必須逐點符合自己的來源 PNG。任何文字、人物、物件、游標或調色盤不符都保留完整原圖矩形，不以色彩近似猜遮罩。所有地面背景先呈現；其他既有介面、頭像與原版游標依原有層序處理。原版／HD 開關不重跑或寫入 guest。
- 驗收：來源與透明度／尺寸／SHA、原始暫存器解碼及自然返回、來源損壞、遮罩旗標、重疊撤銷、全畫面撤銷、缺圖、遮擋、原版輸入不變；正常 750M 的真實視窗開關、完整 RAM 與 15 個原版欄位、原生畫面及 PCM 比對。初版只批准已檢視的 floor-000，其餘新圖經相同比例、材質與構圖審查後逐項加入對照表。單位草稿尚未接入。

八種地面素材已逐項檢視：0、2、6、13、15、22、62、67。6／22 選用去除誤生水面的 v2，其他用 v1；八種皆不透明且比例 3:2，構圖對照保存在 terrain/floor-review.png，修正版另有完整提示詞與來源記錄。104 來源與八項已審查 HD 的編譯期對照由 hd-terrain-prepare.py 重生。十種單位仍為未接入草稿。

### 地面實作與正常首戰驗收

狀態：CONFORMED，限上述地面切片。四套件通過，研究入口 20.161s，包含原始四平面、自然返回、堆疊與幾何界限、重疊撤銷、完整畫面撤銷、來源損壞、遮罩分支、104 原圖及八項素材的 SHA、遮擋與矩形外不變。原版記憶體未被觀測或呈現修改。

HD 與缺圖各以 390 筆普通滑鼠輸入自然完成 750M。15 個原版欄位與完整 RAM、原生畫面、trace、CD PCM 符合基準，每側三次切換 Proof 相同。首戰真實 X11 擷取符合呈現 PNG，六種地面 0、2、6、13、15、22 的 50 個完整矩形使用 HD，矩形外及遮擋維持原圖；缺圖改變零個像素。62、67 在當前末幀已被後續繪製覆蓋，只有來源／共同繪圖端測試，不增加正常可見驗證數。原有九張劇情及四張頭像 GUI 收據保留為先前版本，本輪未重跑該模式。

總表 terrain/runtime/verification-summary.json SHA-256 `3902facc85f45e23495935031addeb8d76b27fbdd2b0c4cf793637509f585fdd`。共用二進位 SHA-256 `fdeb1b1daf508b65eb6044ddd30ea7b2b11bd227c8708c007e4bd0efbf8ad60d`，完整 RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`。

| 模式 | 原版收據 SHA-256 | 真實視窗核對 SHA-256 | HD 矩形／變動像素 |
|---|---|---|---|
| terrain | `bd6dfa65f721284491c196cf4b9ac300006c70a8c912a78b33c56c721038de8d` | `bafcc64c500c9cd20c886f32fb08e15460c3e76090616ebd4e31c500556970b4` | 50／307200 |
| terrain-missing | `5fe8a193645affb2404a73554a767da7f05a42fb1726e8d8f8c84c2957a03461` | `42013b1179c422dd2441befe0e6920372c73551d5e07fa3446c878e81b7ca22a` | 0／0 |

圖像由內建 image_gen.imagegen 生成，選定素材與修正前草稿、透明單位草稿、完整提示詞及生成記錄存 assets/hd-b/terrain。原始參考、正常擷取與執行收據留本機。單位 002／007 的騎兵／弓兵口頭判讀未被來源索引映射證實，僅記為未核對草稿；原版 EXE 檔案偏移 0x2A8D6..0x2A8EB 明示武士、騎兵、步兵、步槍，不能單憑像素為來源命名。未修改單位規則或將未核對草稿接入。

## 2026-10-14 地面上的透明前景

狀態：DRAFT。延續地面已驗收切片，查證原版 sub_1F9D0 的物件／單位 ESI、EDI、旗標及自然返回。只讀診斷保存於 terrain/objects/trace，當時原始碼另存 source-snapshot，先核對完整 750M 原版狀態。既有 terrain/runtime 不覆寫。

IDA 9.4 的既有 range-1f8d2-1fc07.json 記錄 LE 線性 EA 0x1FA09..0x1FA42 的四平面 OR／NOT 遮罩及 0x1FA5F..0x1FB7F 的目的 AND、來源 OR；普通分支 EDX=0xFFFFFFFF，只有索引零透明。0x1F9F2 加入 VGA aperture 基址，0x1F9F8 的 48 列與每列三個 word、pitch=80 不變。range-1fc00-20040.json 的 0x1FE2A 將 EDI 上移 20 列，再依 0x3598C／0x35990 來源呼叫同一函式，返回分別為 0x1FE69／0x1FEA2。輸入 EXE SHA-256 仍為 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a；原始 file_offset、bytes 與名稱均保留於匯出。

本輪目標是保留已證實的原版前景像素，同時替換其下的地面。未知疊圖、調色盤不符、未追蹤的呼叫或記錄界限維持完整原圖備援。單位草稿的兵種映射與 HD 素材接入另列，不能靠顏色差猜測透明遮罩。

### 正常繪製與前景保留契約

狀態：DRAFT → 證據審查 → READY，限正常地面上的原版透明前景。唯讀 750M 記錄 134 次地面、九次 objects 索引 0、兩次 units 索引 1／3 的自然繪製，沒有丟棄或待返回。15 個原版欄位、完整 RAM、原生畫面及 PCM 符合基準。以同一正常 DAC、原始索引零遮罩及實際先後順序合成，65 個現行地面矩形中 64 個逐點符合原生畫面，包含 14 格有前景、6202 個前景像素，其中 746 個與自己的地面同色。剩下一格維持備援，不猜補。收據 terrain/objects/trace/verification.json；來源準備 tools/hd-terrain-object-prepare.py。

- 39 個物件／單位原版來源保留各自索引及完整 SHA，以原始索引零產生 alpha=0，其餘 alpha=255，使用已核對正常 DAC。空白格保持透明，不補圖。來源 PNG 保存在 terrain/objects/sources，指紋編入 hd-terrain-sprite-profiles.go；這是研究對照，不改正式素材包格式。
- sub_1F9D0 入口只讀 ESI／EDI／ESP，完整四平面及三個原始來源基址核對。objects 來源 y=256／304，units y=352，每列 13 格、48×48、pitch=80，目的完整矩形需在 640×400。只接受已證實普通遮罩分支及對應返回 0x1FE69／0x1FEA2；未知呼叫、幾何、來源或堆疊撤銷資格。自然返回 EA 與 ESP+4 同時符合、來源重新核對後登記。
- 每次地面及前景自然返回獲得共同遞增順序。地面仍撤銷重疊舊地面。保留前景順序，不因地面部分重畫就刪掉整個前景；每個地面只合成比自己晚的相交前景。地面最多 256 項、前景最多 512 項，滿額撤銷並回原圖。全畫面 0x1C084／地形配置 0x1E93F 一起撤銷兩類；計數及診斷上限分開。
- 呈現前以自己的原圖與後續前景 alpha 合成預期原生 48×32，整格逐點符合才換 HD 地面。前景 alpha 非零的像素以實際原生色逐點放回，包括與地面同色的像素。未知疊圖、游標、文字、調色盤、缺少前景來源或任何尺寸不符整格保留。未改 RAM、VRAM、規則、座標、輸入或存檔。
- 驗收包括 39 個實際來源的 SHA／alpha、自然返回、錯誤來源／旗標／呼叫／幾何、重疊順序、同色前景、透明洞、後畫地面的順序篩選、全畫面及上限撤銷、未知遮擋與缺圖；正常 HD／缺圖各 750M 的真實視窗、三次開關、完整 RAM 與 15 欄基準比對。前景原版像素另逐點核對，不把地面變動範圍當成前景通過證據。

準備工具初次驗證讀錯既有 group 單數名稱 object／unit，修正索引查詢後同 image 重跑通過；這是工具欄位錯誤。單位兵種映射及單位 HD 草稿仍未接入。

### 透明前景實作與正常首戰驗收

狀態：CONFORMED，限保留原版透明前景的地面切片。39 個來源的 SHA、原始索引零 alpha、空白格與非零黑色像素、普通繪製、堆疊返回、未知呼叫、順序、透明洞、同色前景、上限與全畫面撤銷均通過。四套件通過，研究入口 26.413s。呈現遇到尚未返回的原版繪製會回原圖，不在部分完成的前景上替換地面。

HD／缺圖各完成 750M、390 筆普通輸入；15 個原版欄位、完整 RAM、原生畫面、trace 與 CD PCM 符合基準，每側三次切換 Proof 一致。真實 X11 抽樣在 742M／743M，六種地面 65 個完整矩形使用 HD，370940 個呈現像素改變；7067 個原生前景像素對應的 28268 個呈現像素逐點保留。缺圖改變零。39 個前景來源皆載入核對，未接入 HD 物件／單位。目視可見前景周圍的整格原版草地已替換，部分原版地面被後續重疊繪製而撤銷整格資格的碎片仍保留；不外推整個戰場或全遊戲 HD。

離線圖比桌面少一格的原因已核對：原始 frame PNG 含 16×16 游標，桌面 snapshotNative 保留原始畫面與游標分層，最後由 compositeDesktopCursor 合成。唯一差異格為 `(304,312)-(352,344)`，游標原點 `(320,320)`。verify-terrain-evidence.py 從原始圖集索引、完整順序及實際 AND／XOR 遮罩獨立重建，加入游標後全部 65 格精確符合原版 PNG；不是放寬像素匹配。離線原版的 64 格／6202 前景證據保持不變。

總表 terrain/objects/verification-summary.json SHA-256 `78591ff9cf7a2d4e1682581cd2afe5950b8d514694cd8d3d1dd0e9ba6ddb78a9`。共用二進位 SHA-256 `c5900ffa8b21d5fda795283fb7f05639acaebd76e0c92e180edef2345b43d304`，完整 RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`。原版順序研究總表 SHA-256 `07f377f943cf8da160399db0d6aba9451f9ef0e07ffd6e07a31f61f204439dd8`，39 個來源對照 SHA-256 `fe58b4f1ad96c67dea74f5170f40fa54f5426fa9deaeb67bbbf14075e082f78b`。

| 模式 | 原版收據 SHA-256 | 真實視窗核對 SHA-256 |
|---|---|---|
| terrain | `e107b1dfab8fd3787c4c25656dc3f43cab8873682992505f188f9e6f1945432a` | `69591b7b827c8a14135dff8da6d3808688e58bdb92d6a49d402f50f23d8e8300` |
| terrain-missing | `14423a02f68c6a9cb5efae04fd1ed4b2a485d4fd49fce996f04131d05770707d` | `2138e3b7e79062ef80c5955d08cec35bd360876a33ea869198b4dba53163db02` |

### 物件手繪草稿

內建 image_gen.imagegen 完成 object-000 樹木、001 三組石塊、002 藍色城郭、003 紅色城郭，輸出皆為 1254×1254 的透明 PNG，保留原始 alpha。圖片、完整提示詞與生成記錄保存於 assets/hd-b/terrain；未作正式輪廓／基點審查或接入。首戰 objects 索引 0 可作下一個垂直切片，其他物件尚未正常抽樣。首次保存腳本誤設固定 1280 尺寸，檢查實際輸出後改核對方形尺寸界限並保存原始 bytes，沒有縮放覆寫生成圖。

## 2026-10-14 首戰樹木的透明 HD 研究

狀態：DRAFT。原始 sub_1F9D0 的讀取端、索引零透明、尺寸與自然返回沿用前節已證實證據。正常九次 objects-000 記錄中，五次已被後畫地面完全覆蓋；四棵仍可見。三棵的完整 48×48 下方有已核對地面，右側一棵只有 1824／2304 像素的完整地面來源，跨入介面區域，維持整棵原圖備援。原版規則、存檔、輸入與繪圖照常執行。

新切片只研究三棵完整背景的透明 HD 合成：以目前各地面來源與全部後續原版前景重建預期畫面，完整矩形逐點符合才還原原版樹木下方背景，按原始順序合成新樹及其他前景。未知背景、介面或重疊已替換物件保持原圖，避免猜測樹葉下的像素。生成樹木 alpha>=8 的實際範圍為 `(25,131)-(1235,1075)`；可丟棄預覽使用外擴三像素的明示裁切 `(22,128)-(1238,1078)`，對齊原版非透明範圍 `(0,0)-(47,47)`，生成 PNG 本身不改。

### 構圖勘誤與樹叢 v2

放大原版後，底部有兩根分開的主要樹幹，中間較高處另有短棕色區段；不能把這個圖塊視為單棵樹。先前 v1 的單樹構圖不接入。使用內建 image_gen.imagegen 以放大的原版構圖及 v1 筆觸參考重製 v2，保留兩個主要底部基點、後方棕色區段及連接樹冠，不推定畫外樹木數。原圖與 v2 已放大檢視，v1 及原有幾何試作保留。

選定 v2 SHA-256 f6885ec6d1d557973fdb304fca537e095eec4c65e37439340ef2c39da2d94cd2。1254×1254，alpha>=8 範圍 `(10,0)-(1244,1242)`；明示裁切 `(7,0)-(1247,1245)`，外部 alpha 最大 1，生成原始 PNG 不改。裁切後按原版 `(0,0)-(47,47)` 放入 48×48 格，來源最後一欄／列留透明。完整背景研究與 v2 幾何收據 tree-source-verification.json SHA-256 5d22bc734176ddac1541387162ea429d1911625cde09c0eb0c983f09b1e1f721；v1 收據另按其 SHA 保存。

### 樹叢透明 HD 呈現契約

狀態：DRAFT → 證據審查 → READY。只批准 objects 索引 0 的 v2，規則、存檔、座標、來源索引及原版繪製皆不變。資料與工具位址空間沿用原始 DX5.EXE／DAT／IDA 9.4 與前節自然繪製證據；不新增猜測性行為或正式素材包 schema。

- 每個候選 48×48 矩形的每個像素必須有目前有效地面來源。以各地面順序及其後的全部原版前景，重建原版色及最後所有者；完整矩形符合未合成游標的 native snapshot，且候選仍有原始可見像素才啟用。五個已被重畫的舊項不替換，背景不足的介面邊緣項維持原圖。
- 先以已驗證地面 HD 重建下方背景；未啟用該地面 HD 時用自己的原版來源。按原始順序合成其他前景與候選 v2；像素只受比自己的地面晚的前景影響。新圖透明處由已證實的下方圖像補回，不留下原版樹葉或樹幹輪廓。
- 原版最後可見且不屬於候選的前景像素，再以原生色保留，包含早畫但仍可見的人物，防止新樹冠擴及原圖透明洞而蓋住人物。原版游標仍最後合成。若與本幀已完成 HD 物件矩形相交，後項不替換，避免重建時把前項退回原圖；未抽樣重疊場景不外推支援。
- 原圖 48×48、索引零 alpha、原始 SHA 及 v2 完整 SHA 固定。PNG 須為方形、48..8192 邊長且有透明與可見像素，明示裁切必須在畫布內，裁切外 alpha 不能大於 7，原始非透明範圍必須為 `(0,0)-(47,47)`。只在啟動時準備 2 倍素材，保留裁切內原始 alpha，呈現採整數 alpha 混合。
- 缺圖、來源／幾何不符、未知覆蓋、繪製尚未返回、背景不足或原版／HD 開關關閉都不改輸入。只有樹叢素材可用時，頂端列仍能啟用 HD。三種 GUI 驗收：地面與樹叢一起 HD、只有樹叢、缺樹叢及地面素材；各自然 750M、390 普通輸入，四套件、完整 RAM、15 欄基準、原生畫面與 PCM、三次切換、真實 X11、人物及矩形外像素核對。正常可見 HD 樣本限三個完整樹叢。

### 樹叢 v2 正常路徑驗收

狀態：CONFORMED，限上述 objects-000 三處完整背景。四套件通過，研究入口 25.378s；真實來源 SHA、v2 裁切與原始 alpha、背景還原、透明洞、早畫仍可見及晚畫人物、未知覆蓋、背景不足、已覆寫來源與缺圖備援均通過。原生輸入及圖像未被呈現改寫。

地面＋樹叢、只有樹叢、缺圖各自然跑完 750M、390 筆普通輸入，15 個原版欄位、完整 RAM、原生畫面、trace、CD PCM 符合基準，各三次切換 Proof 相同。三個完整樹叢真實 X11 呈現符合 PNG，82 個仍可見原版人物像素逐點保留。只有樹叢時改變 18929 個輸出像素，其中 1394 個原有輪廓像素以已證實的下方圖像還原；缺圖改變零。搭配 65 個 HD 地面時改變 387920 像素。矩形外不變，右側介面邊緣樹叢維持原圖。未外推其他物件、單位 HD、移動／戰鬥或完整戰場 HD。

總表 terrain/trees/verification-summary.json SHA-256 `42e02fd79ed8dbb9393a2f9418adf216875d2df314a7921ec939c55b1fe08455`，共用二進位 SHA-256 `e8cfac7ea3f32dd876922d0bf554287b09d4d93d93a956d8f49498b1145dcbd7`。完整 RAM SHA-256 仍為 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`。由 tools/verify-hd-trees.py 獨立核對三個來源快照、圖片、二進位、原版狀態與來源幾何。

| 模式 | 原版收據 SHA-256 | 真實視窗核對 SHA-256 |
|---|---|---|
| trees | `ea2b3d3ca28631e9d7fb05faeacd8d51810eedd451ccf87299e2928b4f1bbbfa` | `61ca8312d38e5bf465386783d5a9a2a6f14f26db27cc6eaf628bf70d20c8fa64` |
| trees-only | `453a86556e0ba90b676a2924bdebb359687c9dd225097160ad2d5bba7126a09b` | `56b68cac3603b87ea8a5501d05641c160915aab9e3523baa20e37bbf949bc4ea` |
| trees-missing | `aac83bbffbc7ea612962766be7ad38d9931a804e1375648109945db6265cbf8e` | `6a4d02feed2823a05b9373a7d546fcabe4902ddd9d8386d85114584c31ad93c0` |

先前透明前景切片的收據與總表保留不變。其驗證器新增唯讀來源快照與 verify-only 選項，重核當次來源而不誤用目前樹叢版本的程式 SHA；舊原版遮罩、游標及 65 格驗證再次通過，未改寫歷史總表。

## 2026-10-14 介面邊緣與原版繪製前背景

狀態：DRAFT。右側 objects-000 的 `(376,196)-(424,244)` 有 480 個像素缺少保留地面來源，其中 287 個仍是原版樹叢。單純依地面範圍裁切會留下舊輪廓，不能採用。MAPGAMEN.OZM 原始索引 SHA-256 99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c 與正常載入一致，但補入該底圖後仍有 14 像素不符；目前 RAM 的通用 loader buffer 已被後續來源重用。底圖不能直接當成當前原版完整背景。這不推翻前三處完整地面驗收。

改以最小診斷在原版 sub_1F9D0 真正執行前擷取 native snapshot 的原生背景，返回後以原始四平面、DAC 與索引零遮罩逐點重建，核對 48×48 結果。已證實的入口、返回、來源界限與普通遮罩契約沿用前節 IDA 9.4 的 LE 線性 EA／原始 bytes。診斷只讀，不注入或修改原版，不先接入正式呈現。輸出與正常 750M 基準核對保存在 terrain/backgrounds。

### 原版背景證據審查與呈現契約

狀態：READY。四套件通過，研究入口 20.244s。診斷沿普通輸入自然完成 750M／390 事件，十一項入口、來源及自然返回全部逐點吻合原版索引零透明合成。獨立 Pillow 核對擷取來源與既有 48×48 原圖、三張圖片的 PNG／RGBA SHA、十五欄基準、完整 RAM 及當次工具來源快照；總表 `background-verification.json` SHA-256 `d131450b3925c415736c83584cceb089704207d4a452c4acc9b6929051261a0b`。包含介面邊緣樹叢、units-001、units-003。此項已證實原版普通繪圖的背景與結果，尚不宣稱人物 HD 或 GUI 完成。

- 桌面在同一入口擷取背景，只有自然返回且原始來源、DAC 與完整返回圖逐點符合才保存替換資格。快照只讀 VGA 狀態，不帶游標，不更動原版記憶體、寄存器、繪圖或規則。背景像素只留在程序記憶體；收據保存 SHA 與擷取步數。
- 已有完整地面來源的像素沿用既有精確重建。缺地面來源的像素改用這次繪圖前的原版背景，再依候選及後續前景順序重建。完整 48×48 必須符合當前原生圖；未知後續覆蓋仍回原圖。被後續地面覆蓋的原版人物或樹叢不復活。
- 缺地面來源處的新 HD 圖只在候選原版非透明範圍內合成，原版透明處保留原始介面及裝飾。新圖透明處還原擷取背景。已知原版其他可見前景仍原樣保留；缺圖、未返回、來源不符、背景未驗證及重疊已替換物件保持原有備援。
- 驗收沿用完整玩家路徑、真實 X11、三次切換、完整 RAM、十五欄基準與矩形外保護，另逐點驗證右側 480 個非地面像素的背景還原與介面保留。先前三處完整背景收據保持不變。

### 原版背景接入正常視窗驗收

狀態：CONFORMED，限 objects-000 四處正常場景。四套件通過，研究入口 24.322s。三種真實視窗各自然完成 750M／390 筆普通輸入，完整 RAM、十五欄、原生畫面與 PCM 符合基準，三次切換的原版 Proof 相同。十一個桌面背景的來源 SHA、前後 RGBA SHA 及入口／返回步數與獨立診斷逐項相符。

介面邊緣的 480 個非地面像素使用實際原版背景；772 個原版透明處呈現像素保持介面裝飾，82 個原版可見人物像素保持不變。只有樹叢 HD 時改變 25310 像素，1926 個原有輪廓呈現像素還原背景。地面＋樹叢時改變 393768 像素，缺圖為零；各模式矩形外皆不變。四處樹叢實際 PNG 與 X11 相同。人物與樹叢重疊的 HD 多物件合成及地面碎片仍待後續，不因本項通過而宣稱全遊戲完成。

總表 `terrain/backgrounds/verification-summary.json` SHA-256 `911938c881bada744c90831b3167eb6904f2cfd6cd606e29fd5371dc55e3f46b`；共用二進位 `fb53e0e66a5eba650bf5b648b0828e60b2d86265b50148eabffe545a7610a333`。三模式原版收據／GUI 核對 SHA：

| 模式 | 原版收據 | GUI 核對 |
|---|---|---|
| trees | `935d021d05bfa803df9dc68bb8a23e98d485c48465a0ef14464047aacf21806f` | `dadc59d0743de35ac40074300941b7a4b520ab3f328b2ab07fff14e4f763f542` |
| trees-only | `595729046b0e6470c9874cd5d4a633c6d2dd218f8ae3bbd323726dcecc7fa541` | `649bd4212d21bdb3e813714e584b48f949fe1ad74e82ef264c2b1956b6b4854e` |
| trees-missing | `14c2b71b88114902d4353e5c612f3a2fa31a7466c538b269c691a5b5412f55a0` | `2a9a77f356e167c32c4c6cdc8cfe552c98857e92c5e3bb827637924a4957bf6e` |

另以內建 image_gen.imagegen 修正 units-006 草稿的刀勢，v2 把銀色刀刃放回原圖左下方，保留面向右方的跨步構圖。原始生成 PNG SHA-256 `901d37d63d0847131787fc4f1e880a0e29d1fb6e82b0203fc1d033cb2d282cd7`，完整提示詞與來源記錄保存於 `assets/hd-b/terrain/unit-revisions.json`；尚未接入人物呈現，v1 保留為已取代草稿。

## 2026-10-14 地面可見碎片與重疊人物

狀態：DRAFT。目前保留的 65 個完整地面矩形會丟掉部分重疊後仍可見的舊格，畫面因而留下原圖碎片。首戰共有 134 次地面、11 次前景自然返回，已保存普通順序、來源與幾何。下一步從原始返回事件重建完整順序，逐項核對目前原生畫面的可見區域，再審查共同呈現。units-001 與 units-003 的原版背景已證實；其手繪草稿已比較原始姿勢與方向，生成圖裁切外 alpha 最大 1。這兩個索引不推定兵種名稱。

研究繼續沿用原始 EXE／DAT 雜湊及 IDA 9.4 的 sub_1FC07、sub_1F9D0 原名、線性 EA、bytes 與 caller 證據。此項只擴充來源順序與桌面呈現，原版繪製與狀態不改。未知介面覆蓋仍備援，不以來源原色碰巧相同就略過完整矩形核對。

### 完整順序審查與共同呈現契約

狀態：READY。`source-scene-verification.json` SHA-256 `d2d45e01db7c7d952b252a98d0c3e394f99feae4208fd0f24654f43d0e4385b2`。原始 145 個自然返回重建 134 地面、11 前景；67 個可見地面及六個前景的完整矩形符合原生畫面，原版游標區單列排除，後續 runtime 仍須核對不含游標的完整 native snapshot。可見地面包含原有 65 格，以及 order 78 `(304,56)-(352,88)`、order 86 `(280,88)-(328,120)` 各剩 1440 原生像素的碎片。其餘四項模型地面殘片已被未追蹤介面覆寫，矩形不符，維持備援。原圖背景及六個來源已證實，HD 外觀與合成尚待 GUI。

- 地面自然返回另保存有界的完整順序，現有 65 格觀測不變。完整順序按實際繪製先後建立每個像素最後的地面與前景來源；用後續地面覆蓋較早來源，已消失物件不復活。
- 候選地面與前景都要完整矩形逐點符合當前 native snapshot，且仍有原版可見像素。未知覆蓋、未返回、來源與幾何不符保持原圖。缺地面處只使用已驗證繪圖前背景，新 HD 前景不能覆蓋原版透明處的介面。
- 所有合格圖層在同一畫布依序合成，避免逐個物件重建時把前一個 HD 退回原圖。原版最後可見的前景仍保有優先權；該來源有合格 HD 時用自己的 HD，否則保留原版像素。原版可見來源的 HD 透明處顯示其他合格圖層與背景，不能留下自己的舊輪廓。原版游標仍最後合成。
- 只新增 units-001／003 兩項：v1 PNG SHA 分別 `a8c37f61ef22f645a58cb824758ab34620e535a2588f0fce5e71490cf3dd3a72`、`6c9a0ef3d9a3db01bec673fb37d7f8fca4a2f2a08cf6a14d81ecd101ec867654`；原版範圍 `(2,0)-(47,48)`、`(0,0)-(41,48)`；生成圖裁切 `(265,39)-(1098,1191)`、`(117,90)-(984,1159)`。保留裁切內原始 alpha，裁切外 alpha 不能大於 7。來源索引、方向與基點不改，不推定兵種名稱。
- 測試需覆蓋地面碎片、後畫地面、早畫仍可見人物、原版同色前景、多項 HD 重疊、透明處還原、介面、缺圖及讀取不變。GUI 三模式為地面＋樹叢＋兩人物、只有前景、全部缺圖，各普通 750M／390 事件與基準核對。正式包格式與原版規則不在此變更。

### 完整地面與重疊 HD 人物驗收

狀態：CONFORMED，限六種地面、objects-000、units-001／003 的首戰正常場景。完整順序保存 134 個地面與 11 個前景，原有 65 格觀測仍保留；共同畫布啟用 67 個地面可見區，共 102720 原生背景像素、四樹叢及兩人物。原版最後可見人物的 HD 延後一次合成，避免後畫新樹冠遮住原先仍可見的人物，也避免半透明像素重複混合。其他不合格來源維持原版。

四套件通過，研究入口 27.346s。地面＋前景、只有前景、全部缺圖三種真實視窗各自然完成 750M／390 事件；完整 RAM、十五欄、原生畫面、原始 trace、CD PCM 符合基準。三次切換 Proof 相同，所有真實 X11 圖與呈現 PNG 完整符合。獨立 Pillow 依來源、完整順序、裁切與整數 alpha 公式核對畫面；其公式核對排除 `(320,320)-(336,336)` 原版游標區，不把排除區稱為公式已驗證。游標沿既有呈現路徑，原版及真實視窗核對不排除。

- 地面＋前景：改變 412342 個呈現像素，413696 個受影響且不含游標的像素與獨立公式逐點相符。兩人物舊輪廓分別 1584／1574 像素還原背景；重疊區原版可見人物 82 原生像素使用自己的 HD 優先權；1620 介面呈現像素保持原樣。
- 只有前景：改變 33965，獨立核對 52736 像素；兩人物舊輪廓 1584／1570 像素還原原版地面與其他 HD 前景。重疊 82 原生像素及介面 1620 像素通過。
- 全部缺圖：改變零；不回寫原版、不復活已覆寫來源。各模式資格範圍外不變。地面 partial journal、後畫地面、早畫可見人物、同色前景、半透明一次混合、其他 HD 層背景、介面、缺圖及原始輸入不變均有針對性測試。

總表 `terrain/scene/verification-summary.json` SHA-256 `5af50794c64a7442aee41bc88609a314ea0755de81cd1b8ec96049ff1a4946ce`；共用二進位 `45c19293bbe1ebfc69adafd8acf3ef1a6a9f0860e6b1a522294def349fb91345`。目前桌面預設啟用共同合成；舊切片 GUI 可用 `-hd-terrain-scene=false` 隔離，過去收據與總表不改寫。

| 模式 | 原版收據 | GUI 核對 |
|---|---|---|
| scene | `a0f09917680225e046987e76bcb72286c9ee4ecbd166a9b852b5e5fec48e6e43` | `19c14b759cc178be9fe3079fc4bbdef702f0f5938d64e2cacf286e9a1ff3fccc` |
| scene-only | `cc3f8f927a95fbbf092639d67b9b58be4d0c8db054af899ea421a6d50c3b9c1d` | `d363c4d2f2aa35db78ebb7860656c4a6c2a6c5b31b3c9c978d92d27ab8f6b512` |
| scene-missing | `de1c49ef04206628bd4a094e07bca4bf0ebfa1763d7f8e15bad5e87446aeed19` | `003fa2496c05007ac86002d0b1315add6293d3d32ace284ec72545328ec0432e` |

另以內建 image_gen.imagegen 製作 MAPGAMEN 戰場紅金框架草稿，原始 1586×992 PNG SHA-256 `0a8fd2e1ead2bc8c5a83bb2d4da0cb9f8e81e7fa5bdc3c7d4b7fcaaa360af93e`。原版布局為主要參考，既有 VISUAL 框架只作筆觸參考。三葉葵、紅漆柱、金框、黑色開口及空白標籤區已檢視，幾何校準、動態區域保護與正式呈現尚未完成。來源、完整提示詞及草稿保存於 `assets/hd-b/art/ui-generation.json`。

## 2026-10-14 戰場框架來源與全畫面繪製

狀態：DRAFT。沿 MAPGAMEN 已核對載入來源，先觀測既有 sub_1C084 全畫面繪製入口與自然返回。此入口在 VISUAL 已有原始 bytes 與正常路徑證據，但不能單憑同一入口宣稱 MAPGAMEN 已繪製。只有當前載入四平面符合既有 MAPGAMEN SHA、入口／返回堆疊正確且返回原生 640×400 索引 SHA 為 `99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c` 才記錄資格；另保存當時 DAC 與原生圖片，獨立核對完整索引及色彩。未知來源及其他全畫面複製撤銷資格。

保持原始 EXE／DAT 與 IDA 9.4 線性 EA 位址空間及現有函式名。診斷只讀、不替換圖像、不推測不同 palette 的 RGB 差；來源載入時 PNG 只代表載入時 palette，正式畫面另查證。後續框架須保護戰場、動態地圖、天氣、文字及原版介面覆蓋；手繪三葉葵不能殘留到已被戰場覆蓋的位置。

### 2026-10-15 原版框架來源核對

來源狀態：已證實。四套件通過，研究入口 27.427s。MAPGAMEN 載入返回為 657204319 步；原版 657204327 步進入 sub_1C084，在 657204466 步自然返回 EA `0x12C2A`，堆疊正好加四。256000 個原生索引與來源逐 byte 相同，SHA 為 99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c。當時 DAC 解出的 PNG 與載入擷取相同，SHA 為 998fd37004058cb794e04ea58eca72e56a87eebe349dbaafe56fc1b19bd0b24b，獨立逐像素色彩核對通過。

完整 750M／390 事件、十五欄、原生畫面、PCM 及完整 RAM 符合基準。唯一來源總表 `battle-frame/source-verification.json` SHA-256 a1ba0e3ded13622c59f3b4e5fb81ff474cf1fe55175dc1da72646f54dc38f926。第一個測試編譯因 fixture 的 image.Image 不符合 nativeSnapshot 的 *image.NRGBA 而失敗，已修正測試輸入並在同一 image／命令乾淨重跑；不是產品缺陷，失敗 log 保存在 source-probe/platform-test-before-fixture-fix.log。

### 框架幾何審查與呈現契約

狀態：READY。手繪版原始 1586×992 不改，v1 幾何草稿與來源保留。v2 把兩側凸耳的原版 x=64 邊界及 y=88/120/152/184/216/248/280/312 分區對齊，消除錯用黑色開口的採樣。三葉葵獨立對齊原版 `(111,77)-(352,313)`，只在整個 `(100,70)-(360,320)` 索引仍等於原版框架時顯示。v2 PNG SHA-256 3de79e3eebe01aa970e45b8bf377b6e381bec14c9b4d9f569994aaecbe78e435；幾何總表 9476514d407923222ee06346da4869153279795b7a55293a76c7287e71bd33f9。兩次可丟棄預覽已檢視，預覽不是正式 GUI 收據。

- 原版框架索引與當次全畫面複製資格固定；原生 palette 必須等於已核對的十六色，未知 palette 或淡出期間回原圖。框架遮罩使用來源非黑色與不碰邊、最多 256 像素的封閉黑孔，原版大開口及徽記保持獨立。遮罩不是原版繪圖的推測性改寫。
- 原版 WEATHER1 的 144×74 圖與正常畫面 `(464,291)-(608,365)` 完整逐點相同，為已證實的天氣區域。該區只在整個 native 索引仍是框架空白底圖時換 HD 紙張；天氣出現後整區保留。動態地圖 `(434,16)-(632,214)` 排除，字樣及原版額外標記依索引差異保留。
- 不讓框架蓋過原版同色人物或 HD 圖層。共同畫布記錄合格圖層的實際影響區及原版已核對的可見前景；框架後畫時排除這些像素。沒有合格場景模型時，已開始戰場繪圖的框架保持備援。
- 圖塊繪製前背景另保存 48×48 原生索引。缺地面處只有該背景索引與框架來源索引吻合才採用 HD 框架底圖，否則保留原版背景；既有原版前景優先權與一次 alpha 混合不變。此來源鏈須由正常 GUI 核對，不能用純預覽取代。
- 三種 GUI 為框架＋共同場景、只有框架、缺框架但共同場景仍可用；各抽測載入後徽記及首戰，保護動態地圖、天氣、文字、可見人物、未知標記、矩形外及原版狀態。只有框架時原版人物仍保留，缺框架時既有 HD 地面／人物不退回原圖。

2026-10-15 GUI 階段勘誤：正常 658M 後已疊上武將能力／確認視窗，徽記不是完整裸露的來源矩形。首輪逐點畫面公式已通過，最後的「徽記必須 HD」假設錯誤；應驗證遮擋時整區備援，未遮擋的正向條件以原版全畫面複製收據及單元測試為準，不稱為正常 GUI 已看見。驗證器也修正兩個畫面合計五次切換的計數。原版指令、輸入與呈現邏輯不因這兩項驗證器問題而調整，首輪輸出保留於 attempt-01 目錄。

### 2026-10-15 框架 v2 正常 GUI 驗收

狀態：CONFORMED，限正常能力／確認視窗與首戰的 MAPGAMEN 框架。四套件通過，研究入口 19.820s。三模式各自然 750M／390 事件，完整 RAM、十五欄、原生畫面、trace、PCM 符合基準；五次切換 Proof 相同，六張真實 X11 與呈現 PNG 完整符合。原生索引、背景索引與 SHA、實際 palette、固定幾何、場景影響區及原版可見前景均有獨立核對。

- 能力／確認視窗：有框架時改變 331928 呈現像素，83226 原生框架像素使用 HD；1022976 呈現像素與公式相符，排除當時 `(130,60)-(146,76)` 游標公式區。原版能力、確認、頭像與額外標記保持原樣，被視窗遮住的徽記整區備援。空白天氣底圖符合整區資格而使用 HD 紙張。
- 首戰共同 HD：改變 761107 呈現像素。框架 347932、場景 413696 像素符合獨立公式；67 地面、四樹叢、兩人物及重疊區 82 原生人物像素通過。1564 呈現像素使用原版繪製前索引確認的 HD 框架底圖，56 個仍未知的介面像素保留原樣。兩人物輪廓 1620／1574 像素還原背景。動態地圖及 144×74 天氣整區不改，文字與原版額外標記按索引差異保留。
- 只有框架：改變 348729 呈現像素，349496 與框架公式相符；原版地面、樹叢及人物維持原圖，包含同索引前景保護。缺框架：原有共同 HD 場景保持 412342 像素變動，框架及空白天氣底圖回原圖。首戰公式區排除 `(320,320)-(336,336)` 游標，真實視窗核對不排除。

總表 `battle-frame/verification-summary.json` SHA-256 `943c5bb05efbfb009cb282db0c052d8f146379094f3924ed83d7315ae864b66e`；共用二進位 `a3b822fc53d4ed9948465eaf4ca28dd0cb37c7efd688ef645d05204d6e248c3b`。MAP_WAKU、天氣美術、其餘來源與全遊戲 HD 仍待後續；不把本項稱為所有戰場 UI 已 HD。

| 模式 | 原版收據 | GUI 核對 |
|---|---|---|
| battle-frame | `461c7c397e3716852e0cfe2edd5f9834031eb5f52a78f616984a0ac69ac27f81` | `1155d1fce3630840b96769b53eb87ce92963e1398471668a5d138bd4a6d2b8ae` |
| battle-frame-only | `7696d99997f6092a0404a48492e752e30021692516abdbc70b745f93edf20b87` | `10b38e719bac490e7429640bd9b590ba055a0397e5ba28e5d7b95d508ebfa11e` |
| battle-frame-missing | `492850b01230528c0ace7b7647d4852bd6a5d37ff113c6e044a6e4a38c658276` | `d3d2c417882e5e200406ec88a61e17598b92ea020e918fe2115d87e7bcb45274` |

## 2026-10-15 其餘 OZM 與共同不透明繪圖

狀態：DRAFT。全遊戲 HD 尚有 28 個已解碼 OZM 未製作，另有五個未支援解碼、地面／物件／單位圖集及 PCX。依當前覆蓋表重生其餘來源總覽，沒有填補不完整資料。先製作 WEATHER1／2，並查證共同繪圖，避免為每張圖重建不同的繪圖端。

原始證據 `workplace/hd-prototype/range-1bf73-1c550.json`，IDA Pro 9.4，輸入 DX5.EXE SHA-256 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a，位址空間為 IDA LE 線性 EA。sub_1C1BC 在 `0x1C1CA` 讀取原始 `[ebp+arg_0]` 作來源 offset，四次呼叫 sub_1C26D 並使用 globals `0x38C00/0x38BF8/0x38C04/0x38BFC`。sub_1C26D 的 `0x1C27E` 讀寬度、`0x1C281` 執行 `rep movsb`，每列來源固定加 `0x50`，目的加傳入 pitch。由原始 push 順序可確認外層五參數為來源 offset、byte columns、rows、目的 pitch、目的位址。這是已證實的不透明 byte-copy 契約；不同來源在正常路徑的參數與畫面仍需診斷核對。

診斷在入口重新讀取實際來源 dimensions、指標與四平面 SHA，不把最後載入檔名當成當前 RAM 所有者。只接受已解碼的精確來源、80-byte pitch、VGA 範圍內的完整矩形；自然返回後核對同一來源及完整目標索引。來源不明、來源／目的越界、不同 pitch 或未返回均記錄限制，不擴張推論。原版資料、規則、輸入及存檔不改。

### 共同繪圖與天氣呈現審查

狀態：READY。2026-10-15 四套件通過，完整 750M／390 事件、十五欄及完整 RAM 符合原版基準。16 次自然返回均與各自靜態解碼來源逐 byte 相符，沒有未知來源或漏記。涵蓋 OP1、OP5、VISUAL、九種劇情、MAPGAMEN 及 WEATHER1；總表 `graphics/source-verification.json` SHA-256 `7abe3e4f463ffb001c5024bf89111b7209c643c818027c1d6ccdd509be9cdc69`。WEATHER1 在 712445805 步進入 sub_1C1BC，712449249 步自然返回 EA `0x15BAF`，參數 `[0,18,74,80,0xA5B2A]`，來源完整 144×74 對應目標 `(464,291)-(608,365)`；索引 SHA-256 `ad397c7ae15a08571b43e66fed4fc9fc9236c37909b374fe69ef1e51771066c0`。

天氣沿共同不透明繪圖契約接入：入口核對實際 RAM 四平面、尺寸、五參數，只有原版自然返回且目標索引等於來源才保存顯示資格。全畫面複製清除舊資格，後續共用矩形複製撤銷交疊圖。正式呈現再核對完整目標索引及已確認十六色 palette；任一不符、淡出、遮擋或缺圖均整區備援。HD 不寫 guest，F6 與頂端功能列沿現有開關。來源矩形裁切沿同一契約，未採樣來源只記為共同繪圖端已接入。

晴天及陰天 v2 已檢視，保留原圖裸岩尖峰及雲層，不採用把岩峰畫成松樹的 v1。v1、v2、完整提示詞及生成來源保留於私人素材目錄。晴天 v2 SHA-256 `44ea325fa5e95ab3d8b686a9cf895c3347a47d99f081c58c7406257c49bd014b`；陰天 v2 `b5a024f4ae985a46f6c8e8686ae286a3583394b59e48fa08d821b2f1a146562a`。兩幅皆為 1750×899 不透明圖，顯示時依既有取樣公式縮至 288×148；陰天仍須正常玩家路徑抽樣。

### 晴天天氣正常 GUI 驗收

狀態：CONFORMED，限共同不透明繪圖的 WEATHER1 首戰呈現與逐項缺圖。固定 GUI image 的四套件通過，研究入口 19.915s。來源與 HD 指紋、非零來源 offset 裁切、完整矩形與外側、palette／淡出、未驗證返回、全畫面清除、未知交疊撤銷及原始記憶體不變均有測試。WEATHER2 使用自己四平面與索引，並通過同一裁切／備援測試，未稱正常陰天畫面已驗收。

兩次原版入口至首戰皆自然 750M／390 事件，十五欄及完整 RAM SHA-256 `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9` 符合既有基準；三次切換 Proof 相同，六張真實 X11 畫面與呈現 PNG 完整一致。晴天 42624 呈現像素逐點符合 v2 取樣公式，缺圖同區零變動。兩模式的原版擷取圖相同，所有天氣矩形外的 HD 畫面亦逐點一致，框架、67 地面、樹叢與人物保留。此畫面的游標在天氣區外，天氣公式未排除像素。

總表 `graphics/verification-summary.json` SHA-256 `743d43ac25ffe6d318daa13f1f307d66c861a2584d8c5ff6656a024c780f593e`；二進位 `c8d83595c1f593de60b180e5b1dfaa1e565626cb6f9ecb67579619dbfd333314`；兩項來源對照表 `c37e63154005630193dc12cf5e1520124fc0c9553278a979e34c3af3e3e999e1`。

| 模式 | 原版收據 | GUI 核對 |
|---|---|---|
| weather | `3d7da52e846a1d16705dd1f9b88b787d3e24866de88869b2e4f2cf3109c94d4f` | `4dfaaab2b960ce4b1ae7c946fb8ae1fc4c265a5174977eaee9af04529cddf4f2` |
| weather-missing | `520af7ad657470f784b7a249a7e4c747e024c1061d4e5a145a010837fa3be37f` | `b01732303772401ce986d863a690f9726ebb7f2ac6453ce0a65f19b5edab906b` |

另以內建 image_gen.imagegen 製作 OP2 騎兵與 OP8 五武將開場草稿，分別 2172×724／2171×724；來源、原始輸出與完整提示詞保存。生成比例與原版 640×200 不同，OP2 藍底未使用區也須幾何校準，兩圖尚未接入正式呈現。它們只增加草稿覆蓋，不列為已準備或正常路徑通過。

## 2026-10-15 地面全圖集材質製作

狀態：DRAFT。MAP_CHIP 原版的 104 張 48×32 地面來源、四平面及繪圖幾何沿既有已證實契約，不修改原版規則、地形、移動或存檔。此次只擴充已接入的逐項 PNG，正式素材包格式不變。現有八項及其來源／GUI 收據保留，不因補圖替換。

原版索引色分布可重生：草地來源主要使用 5／7／13，暗處為 0／1／8；土路來源含 2／3／10，水面來源含 6／9／14。以原始索引建立每張的材質分區，手繪材質補足細節。這是 HD 美術幾何製作，不把顏色類別稱為原版地形規則或兵種語意。遮罩及低頻明暗由原始 bytes 重生，邊界柔化及取樣公式完整保存；同來源索引才能共用衍生圖，不因外觀近似合併。

製作採既有已審查草地繪畫，加 imagegen 產生的紅褐土路、深藍水面及淺藍水面。先輸出草稿、逐項總覽與來源雜湊，檢視原圖的道路、河道方向、明暗及空白邊界後才加入已審查 profile。源 PNG、原始索引及遮罩留本機；純生成材質、衍生 HD 圖及提示詞可保存私人 repo。現有八項正常首戰畫面維持相同，新增來源須通過逐項圖形及共同繪圖端測試，不以檔案數宣稱全遊戲完成。

草稿工具勘誤：輸出掛載是 `/output`，原版索引必須從 `/project/workplace/hd-prototype/catalog/decoded/` 讀取；先前以輸出父目錄定位造成缺檔，未修改任何原版。另一項工具假設把全部 21 個道路／水面來源當成同形配對；原始 bytes 只證明 38..49 與 59..70 的 12 對完全同形。其餘九對差異 408..1204 像素，各自保留自己的遮罩，不合併或套用相鄰圖的形狀。修正的是核對假設，材質合成始終讀取每張自己的索引。

第一版七頁逐項總覽已檢視。直接把水面內的孤立草地色點當成草地會產生細碎綠色小島；例如純水面 103 只有十二個黃色紋理點，不是十二塊陸地。第二版保留第一版所有輸出，以四鄰域連通區大小不超過四個原生像素的孤立色點作材質內紋理，清除它們造成的材質小島，再依各張自己的遮罩柔化邊緣。這是明示的 HD 美術近似，不稱原版地形規則已證實；連到大塊草地的岸線保留。第一版不加入正式 profile。

### 完整地面素材的呈現審查

狀態：READY。第二版七頁的 104 組原圖／HD 已逐項檢視，包含 38 種草地、21 種紅褐道路、45 種深淺水面，保留原版方向、主要邊界及明暗。新材質為三幅 1536×1024 不透明 imagegen 繪畫；93 張新衍生圖為 384×256，三組原始索引完全相同的來源共用圖像。既有八張及其 SHA 不改，96 個缺圖來源補齊；原版原始四平面與 104 來源 PNG 不改。`materials/review-v2.json` 鎖定 manifest 及七頁逐項總覽。

正式呈現沿既有 sub_1FC07、原始 ESI／EDI、四平面 SHA、自然返回、來源與遮擋比對、前景優先權及逐項缺圖契約。只更新固定來源對應的 Asset／AssetSHA，不新增遊戲狀態、規則或存檔欄位。驗收為 104 來源的完整共同呈現／缺圖／未知覆蓋／矩形外不變測試，再跑一次正常入口至首戰，核對全部 RAM、十五欄及完整原圖／HD 畫面與前次天氣版本一致。正常首戰仍只證明六個可見地面來源，其他 98 個不冒稱已在正常流程看見。

### 全地面素材與共同呈現驗收

狀態：CONFORMED，限 104 地面來源的素材、共同呈現與既有正常首戰。104 來源均通過自己的來源 PNG／HD SHA、不透明比例、完整呈現、逐項缺圖、未知覆蓋及矩形外不變測試。三組共用來源為 33→10、34→11、35→12，原始索引 SHA 完全相同。原有八項素材 SHA 保留；其餘 96 個來源使用 93 張新 HD，合計 101 張獨立圖。

固定 GUI image 四套件通過，研究入口 25.741s。新版本自然執行 750M／390 事件，十五欄及完整 RAM `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9` 符合基準；三次切換 Proof 相同，三張真實 X11 與呈現 PNG 完整相符。原圖、HD 及還原後的完整畫面均逐點等於前次已驗收天氣版本，包含 67 地面可見區、人物、樹叢、框架及天氣。啟動收據的全部 104 地面素材為 ready，並逐項符合當次編譯來源與 HD 指紋。正常首戰仍只有六個可見來源，不把其餘 98 個列為正常玩家路徑已抽測。

唯一總表 `terrain/materials/verification-summary.json` SHA-256 `7c3c0a8ccb899015ce3080fe91e5c0b56b7cad88811a2f6074114c384f5eca9d`；來源對照 `06828ec98b7011a925a72031f9623c6e82170474f782cd975f5acb87ff06bbd3`；原版收據 `a29f4c07e42c7aa3004b59ce84d83603b0165cf6812a02424c72e3323712d770`；GUI 核對 `79ad1e3a86267bfe225edb39bd03daa0d9f4e9c2b6eb1b95f0ea8c89fc65ead5`；binary `9d4ebacda3e42dc2a36e3681e409c088dd86c069c5cce90892348970031e9d13`。第一版腳本已以與其原 manifest 完全相同的 SHA 保存 `prepare-v1-source.txt`，歷史草稿可回查。

保存工具最初未排除共用圖的重複索引，停在同檔不同 metadata 檢查；當時先啟動的建置因未保存完的新素材而失敗。失敗 log 保存在 `runtime/startup-probe/platform-test-before-archive-fix.log`，修正保存工具只保存原始 owner 後同一 image／建置命令乾淨重跑通過。覆蓋工具的舊八項 count／工具 SHA 契約也改為明確區分歷史八項與新 104 項，沒有修改原版或放寬來源指紋。

## 2026-10-15 完整單位圖集 HD

狀態：DRAFT。原版 MAP_CHIP 的 13 個單位來源、來源 y=352、48×48、四平面及 sub_1F9D0／返回 EA `0x1FEA2` 的只讀觀測沿既有已證實契約。十個非空來源 0..9 與三個空白 10..12 分開；空白不補圖。已有 001／003 的 HD 及正常首戰證據保留。本輪只補其餘八項已確認來源的圖像，不改兵種、數值、行動、規則或存檔。

來源 alpha 只由索引零判斷，非零黑色保持不透明。各原圖的非透明 bounds 與生成圖 alpha 大於 7 的 bounds 分別保存；生成圖的裁切增加三像素餘量，使用同一共同呈現。001／003 的既有裁切恰為此公式，SHA 與幾何不變。002／007 的 v1 姿勢與原版緊湊的抬腿輪廓不同，先以原圖及原手繪圖修正 v2，再逐項檢視十個非空來源。索引與視覺姿勢是美術對照，不把騎乘判讀或生成圖標籤稱為原版兵種映射已證實。

第一次十圖對照中，008 的單尖槍頭仍與原版下方分叉的銀色輪廓不同。保留 `units/attempt-01/` 的當次裁切、對照、腳本與輸入，再對原手繪圖只修正槍頭，保留紅甲、黃色外衣、面向與持杆姿勢。001／003 的兩項既有素材及裁切仍不改，機械兵種映射仍列未知。

### 十項非空單位的呈現審查

狀態：READY。目前兩頁十組原圖／HD 已逐項檢視；002／007 使用抬腿姿勢 v2，008 使用分叉槍頭 v2，006 使用已保存的下左刀刃 v2，其餘沿 v1。全部原始生成圖為 1254×1254，真正透明背景。來源索引零 alpha、SourceBounds、原圖 SHA、生成圖 SHA、alpha 大於 7 的裁切加三像素餘量逐項保存。001／003 的 SHA、裁切及準備後圖像不改，10..12 的三張原版空白保持空白。

只擴充 `hdTerrainUnitProfiles` 為十項，沿既有來源四平面、自然返回、已觀測背景、共同場景、可見前景優先權及一次 alpha 混合契約。原版資料、玩法及存檔不改。驗收要求每項實際來源進共同場景，核對自身 alpha、缺圖仍保留原版前景、未知覆蓋不替換、矩形外及原始輸入不變；正常入口至首戰再核對 RAM、十五欄及原圖／HD 與前次地面版本相同。正常首戰仍只顯示 001／003，其餘八項只稱來源與共同呈現已驗證，不宣稱正常流程逐項已看見。

### 十項單位的來源與共同場景驗收

狀態：CONFORMED，限十項非空單位來源、共同呈現與既有正常首戰。原始索引 y=352..399 的逐項 SHA、索引零 alpha、三項空白、SourceBounds、生成圖 SHA、裁切、準備後透明範圍與缺圖均已核對。十張實際來源逐項通過共同場景、完整背景、未知覆蓋、原版前景保留、原始輸入不變及矩形外保護測試。001／003 的原始素材 SHA、裁切及準備後 PNG 不改。

固定 GUI image 四套件通過，研究入口 24.077s。自然原版入口至首戰 750M／390 事件，十五欄及完整 RAM `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9` 符合基準；三次切換 Proof 相同，三張真實 X11 與呈現 PNG 完整相符。原圖、HD 及還原後的完整畫面均逐點等於前次地面全圖集版本。全部十項單位載入為 ready，13 個來源的三項空白仍沒有 HD；正常首戰顯示的來源仍為 001／003，其他八項與機械兵種映射不外推。

唯一總表 `terrain/units/verification-summary.json` SHA-256 `a417f849eec3b4f2715c93da3aea429690451f9e9625810b2e4d7dd86f7d7bd3`；來源對照 `3a2edcd1567a56ab1d6573356a376ed22ce7238deafab9b7133fe0c6546d2e2e`；原版收據 `938e7ae1c30d4a301db41cba7c32d6fd98c8fb538ca9e06865d9d131ef91a5b8`；GUI 核對 `1f52c999f24650ab78990ee65cfaf8513624e455e2713849b7c39f095edd070a`；binary `b3d881260858e4058ef2c2d726696b2b4ba87f7e3dc70078b796f203569ec6b2`。第一版裁切與工具保存在 attempt-01，生成圖及提示詞不覆寫。


## 2026-10-15：物件擴充的來源與 READY 契約

本切片沿用已驗證的 MAP_CHIP 前景來源、自然返回及共同畫布契約。原始輸入 DX5.DAT 的 SHA-256 為 de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37；MAP_CHIP 索引 SHA-256 為 0e4e6f63e71a39309e21888724ed0e4da29e8f58cfb49b55095340132f4b010b。逐項 PNG、索引與生成圖雜湊見 workplace/hd-prototype/catalog/terrain/objects/expanded/draft-profiles.json。工具為固定 GUI Docker 的 Go 1.24.13、Pillow 9.4.0；素材由內建 imagegen 產生。

- 已證實：26 格原生物件來源位於索引圖 y=256、304，13 欄、每格 48×48；只有 palette index 0 透明，24、25 全空。RGB 黑色不作透明判斷。
- 已證實：沿用原始 sub_1F9D0 的物件自然返回 0x1FE69 及既有來源平面／參數 guard，位址基準為 IDA 9.4 的遊戲程式位址。不新增原版執行攔截或 guest 寫入。
- READY：0、1、2、3、18、19、22、23 進入共同物件呈現。0 保持既有資產 SHA、SourceBounds 與 Crop，原有樹叢旗標優先。每項依來源指紋、已審查 PNG 指紋、原生範圍、alpha 裁切建立 typed profile；來源、遮擋、未知覆蓋或缺圖失敗時保留原圖。
- READY：ObjectHD 計數涵蓋所有物件，TreeHD 只計 index 0；單位計數維持獨立。多物件依自然繪製順序及原生可見 owner 合成。原版狀態、存檔及玩法沒有新的資料寫入路徑。
- DRAFT：4..17、20、21 的生成圖屋簷高度與分層不一致，尚未核准接縫；只保存圖片、提示詞、來源與裁切草稿，不加入 runtime profile。原版用途不依圖像猜補。
- 驗收：全部 26 格索引及 alpha；已核准物件的共同場景、透明邊界、缺圖、未知覆蓋及矩形外保護；多物件重疊依既有優先權公式。正常 750M／390 輸入核對完整 RAM、十五欄、原圖／HD 切換及前版真實視窗。正常首戰只有 objects-000，其餘來源不冒稱正常路徑已逐項驗收。

美術是已選 B 戰國手繪風的呈現近似；來源與繪圖端的已證實定位不等於美術逐像素忠實。全遊戲 HD 與三平台打包仍未完成。


### 本切片 CONFORMED 與後續停止線

核准八項物件的共同場景、缺圖、透明範圍、未知覆蓋及多物件優先權測試通過，四套件通過，研究入口 28.641 秒。完整正常流程 750M／390 筆普通輸入、完整 RAM、十五欄、三次原圖／HD 切換及三張真實 X11 擷取均通過；原圖與 HD 擷取完全等於前次單位版本。獨立驗收總表 SHA-256：cf75a9f0b21a450b781fb8a0f30ebe3dd41612af08e1d56cf76c5d8d33463f73；profiles SHA-256：a5450148ad4f4a394135ad7fb27f4ff9dd266c22a9eb98f4149e170b793fd637；二進位 SHA-256：70eca5d1c293c750af48e7d451cc794a25c5da0b9c0ce8861011a8db3e0cb226。收據、工具與建置來源見該總表，正常路徑僅 objects-000，不擴張至新增七項物件。

保存工具第一次拒絕裁切預覽，原因是有限取樣的 alpha 最大值為 254，未取到原始生成圖少數 alpha=255 像素。原圖仍以 0..255 核對，裁切預覽改用透明且可見的取樣範圍契約，PNG alpha 不改。此為保存驗證工具的分類錯誤，非遊戲或素材缺陷。修正後同一 Docker 命令保存及 verify-only 均通過，共 460 PNG、136 提示詞。

已證實的城牆重用事實：在既有 48×48 來源 RGBA 中裁去全透明外圍後，6 與 10 均為 48×30，像素完全相同；12 與 13 均為 19×48，像素完全相同。這只確認來源重用，尚未核准 HD 接縫。5／9 的 alpha 相同但 RGBA 不同；4／7、8／11、14／15、16／17、20／21 的 alpha 與水平鏡像也不相同，不直接鏡像補圖。來源為同一 MAP_CHIP 索引及 objects/sources，Pillow 9.4.0 的 getbbox／crop／逐 byte 比較，原始定位沿用本節來源表。16 項城牆維持 DRAFT，後續只校準必要的可見接縫與輪廓，不擴大遊戲 RE。


## 2026-10-15：完整物件圖集的城牆幾何 READY

沿用前節 MAP_CHIP 來源與共同物件繪圖端。16 項牆件為 4..17、20、21。三頁逐項原圖／HD 及完整圍牆、短牆接縫試作已檢視，審查見 objects/walls/draft-v2/review.json。原版索引圖 SHA-256 仍為 0e4e6f63e71a39309e21888724ed0e4da29e8f58cfb49b55095340132f4b010b；輸入生成圖、提示詞、實際裁切座標與準備工具快照均由該版 draft-manifest.json 串接。Pillow 9.4.0，固定 GUI Docker；幾何位址是來源圖格 48×48 的像素座標，不是 executable 位址。

- 已證實的來源不變：index 0 透明，24、25 空白；6／10 與 12／13 裁去透明外圍後 RGBA 相同。新圖不複製原版 RGB。
- READY 的呈現近似：以已生成的橫牆 6、直牆 12、城門 5、斷牆 20／21 作分區材質。橫向構件共用相位及屋簷、灰白牆面、石基的高度；直向構件共用縱向相位。接縫端點取共同 RGB，向內一原生像素漸變。原版透明輪廓以 0.65 原生像素柔化並限制在原版 SourceBounds，消除陰影棋盤格；此為 HD 美術近似，來源 RGBA 仍原樣保留。
- READY 的資料擴充：24 個非空物件進入既有 typed profile，共用同一觀測與呈現器。10 重用校準 6，13 重用校準 12，各自的原版來源指紋與 SourceBounds 保留；其他項目不以鏡像推定。既有八項資產、裁切及準備後 PNG 的 SHA 必須不變。
- READY 的備援與驗收：沿用已完成的來源、自然返回、未知覆蓋、缺圖及原版可見 owner 契約；全部 24 項走共同場景測試，五個原版空白 atlas 格維持空白。重建新版並抽測正常首戰的完整 RAM、十五欄及切換，擷取須等於前版。正常路徑只有 objects-000，離線圍牆不充當正常玩家路徑證據。

只有已審查的第二版進入資料擴充，第一版保留為幾何試作。原版玩法與存檔不變，正式素材包格式沒有新增契約。


### 完整物件圖集 CONFORMED

全部 24 非空物件通過共同場景、來源、alpha、缺圖、未知覆蓋及矩形外保護測試，四套件通過，研究入口 26.951 秒。完整 750M／390 普通輸入、完整 RAM、十五欄、三次切換及三張 X11 擷取通過；原圖及 HD 擷取等於前次八物件版。驗收總表 SHA-256：04e41b453715467568b098768f93a13a48662a63cd3548548d124ed43fe47a11；profiles：9edc4c59b15c38b40ebd745d306a7e11570892c8b149178312076b2bb6bb3426；二進位：32596d729107fc8993b822b13849581af1fa0869cd758557709768cf54522a4e。

原版圖集右側 16×400 像素全為 index 0，沒有未分類圖像。104 地面、24 非空物件、十個非空單位均已準備，兩物件與三單位空白保留，因此 MAP_CHIP 美術覆蓋改列 prepared；正常路徑範圍仍為六種地面、物件 0 及單位 1／3。完整遊戲美術仍未完成。

另由內建 imagegen 產出 BATTLE01／BATTLE02／OP4 三張不透明草稿，實際 1586×992，參考圖與完整提示詞雜湊見 catalog/art/remaining-generation-v1.json。岩峰與草地、灰天版本及三葉徽記已目視檢查；黑色緩衝區、幾何與繪圖端未驗，不進 production profile。


## 2026-10-15：五項靜態美術及不透明可見片段 READY

沿用已驗證的原版 sub_1C1BC 與 sub_1C084 不透明複製契約；位址為 IDA 9.4 的遊戲程式位址。來源四平面、尺寸、原始五參數、自然返回 EIP／ESP+4 及當下目的索引雜湊都保持原有核對。原始 DX5.DAT SHA-256：de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37。五資源的 payload、索引、四平面與生成圖 SHA 見 graphics/static/draft-manifest.json，Pillow 9.4.0 的幾何審查及工具快照一併保存。

- 已證實：BATTLE01／02 的 index 0 共 148302 像素，全部連通上邊界，為黑色留白；美術區為 [0,220,640,400]。OP2 的有效圖為 [0,0,480,150]，其餘像素全 index 6。OP4 以排除黑色 index 0／8 的範圍 [197,79,438,315] 校準徽記；不把黑色 index 8 誤算為畫布全滿。OP8 使用完整 640×200。
- READY：新增五項 source-specific profile，保留原始索引與四平面指紋。PaintRect 只限制 host 的繪圖區域，不改 guest 複製。既有天氣 profile 的空 PaintRect 代表完整來源，行為與原圖不變。缺圖、錯誤來源、palette、矩形或可見片段不符時保留原圖。
- READY：後續已知不透明複製的目的矩形，從較早已驗證圖像的可見區域扣除。保留原始 SourceRect、DestRect、原始參數與驗證雜湊，另存 VisibleRects；未知來源的合法不透明複製同樣作遮擋，避免同索引前景被覆寫。完整複製及非法幾何保持撤銷；最多 64 筆圖像、合計 256 片段，超限保守撤銷。
- READY：每次呈現先核對原始來源雜湊，再對所有可見且在 PaintRect 內的片段核對原生目的索引，全部通過才寫 host 畫布。未知直接 VRAM 更新仍觸發片段備援，不把像素相同當成已觀測繪圖的替代證據。
- 驗收：五個實際來源的全圖與非零來源裁切、留白保護、缺圖、palette、未知覆蓋、片段保護及矩形外不變；部分遮擋與同色未知不透明矩形不被背景覆蓋。原版正常首戰的完整 RAM、十五欄、切換與前版擷取維持一致。新來源正常流程是否被使用，另由無輸入開場原版查證，不拿離線測試冒充。

五張美術已逐張檢視，審查見 graphics/static/review.json。這是手繪呈現近似及既有不透明契約的可見區域擴充，沒有 guest 寫入、玩法或存檔格式變更。原版無輸入查證首次受 Ebitengine import 時要求 DISPLAY 阻擋，分類為工具環境；保留失敗記錄，沿用同 image、二進位與來源快照，在有界 Xvfb 內重跑。


### OP1 與空輸入時間表的 READY 補充

舊二進位與匹配的來源快照在有界 Xvfb 內自然跑完無輸入 200M，僅 OP1／OP5 取得不透明來源與自然返回收據；原始結果見 graphics/static/normal-intro/ready/receipt.json。五項新資源不因此冒稱正常開場已顯示。追加 OP1 的 Family Soft 字標，保留拼字、F／i 的菱形、y 下緣與 S 輪廓，位置以排除黑色 0／8 的原始範圍 [202,157,436,214] 校準；實際權威矩形見 logo/draft-manifest.json。

READY：OP1 走同一個來源指紋、可見片段與 PaintRect 契約。指定 mouse-file 即代表使用該時間表，即使內容為空陣列也不混入主機滑鼠；未指定 mouse-file 且無排程輸入時，既有普通滑鼠模式不變。空時間表不向 guest 注入事件，正常無輸入 200M 的完整狀態與舊原版收據核對；OP1 顯示期間以真實視窗核對原圖／HD 切換。這不是 direct-entry，也不跳過原版程式。

### OP1 獨立調色盤的證據與 READY 補充

首次 12M 擷取得到 `fallback_palette`。原先只歸因於淡入，等待到原版離開 OP1 仍不合戰場調色盤；保留兩次結果於 `graphics/static/desktop-logo/` 及其 `ready/`。兩者沒有修改 guest。問題是 OP1 使用獨立調色盤，不能拿戰場完整十六色當作標誌的完成條件。

`graphics/static/palette-probe/` 使用同一已測二進位及來源快照，從原版入口自然跑 50M，沒有滑鼠時間表事件。七次 F7／F9 只暫停與擷取 host；6M、10M、14M、18.333333M 為淡入，22M 及 26.333333M 的十六色完全相同，30.333333M 已淡出。兩次穩定擷取的完整索引 SHA 為 `653b5308aa72b2ebb5f32d4a64f54a2044f0935ad9c08593bcf32b6580b779a8`；`palette-captures.json` SHA 為 `cbe61f72fc33ce5a3e2ed5873ac5d06fa9fdba96d9688c478ce579c1f1f73936`。輸入仍為本節固定 EXE／DAT，Go 1.24.13、Ebitengine 2.9.9，Pillow 9.4.0；擷取不新增工具位址空間。

READY：僅 OP1 的已核對來源使用這份獨立完整調色盤，其他來源維持戰場十六色。每個 DAC 映射後的 RGB 必須精確符合；淡入、淡出、缺色及未知調色盤保持原圖。來源四平面、原始幾何、自然返回、可見區及 PaintRect 條件照常成立，沒有把 palette 檢查放寬。測試使用七次實際原版擷取，只有兩次穩定狀態可呈現 HD，且不能把此調色盤用於其他來源。新版本重跑 200M 開場及 750M／390 事件首戰核對完整原版狀態。

### 六圖與可見片段的 CONFORMED 範圍

六項來源已準備並註冊。BATTLE01／02 的 148302 原生 index-zero 留白、OP2 未使用的藍色緩衝區、OP4／OP1 黑色外圍及所有 PaintRect 均由獨立工具核對。原始索引、四平面與裁切各自保留，沒有用生成圖反推原版規則。

四套件通過，研究入口 27.337 秒；八項實際來源的非零來源裁切、缺圖、palette、未知覆蓋、可見片段、整區原子備援與矩形外保護通過。七次原版 palette 擷取直接作測試輸入，只有兩次穩定 OP1 可套 HD；其他來源不能套該 palette。

OP1 正常無輸入開場 200M、四張 X11 擷取及五次切換通過。HD 53352 呈現像素符合逐點取樣，36393 像素有變更，範圍外不變；完整 RAM 與十五欄等於舊原版無輸入基準。新版本首戰自然 750M／390 事件、三次切換、完整 RAM、十五欄及三張 X11 擷取等於前次城牆版。收據中的來源 SHA 與匹配快照、目前 Go 檔一致。

獨立總表 SHA-256：4990b8c4766bc6b83094d16e7199e2cc2639085e193ca310394a747e9e8e6379。輸入、幾何、調色盤原版證據、建置與 GUI 雜湊由總表串接。CONFORMED 限 OP1 正常開場及既有首戰回歸；另外五項只聲明實際來源與共同呈現測試，不冒稱正常玩家路徑已抽測。原版文字、規則與存檔沒有變更；動畫、其餘 OZM、PCX、其他平台及正式封包未完成。

## 2026-10-15：剩餘 20 項已解碼 OZM 的美術準備

本節只涵蓋素材準備，不變更 Go 執行器、原版輸入、規則、存檔或繪圖端。DAT SHA-256 為 de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37；20 項的原始檔案偏移、payload、四平面及索引 SHA 來自固定 decoded/inventory.json，逐項另存 remaining/sources.json。沒有建立新的反組譯位址空間。工具為固定 GUI image 中 Python 3.11.2、Pillow 9.4.0；來源與工具全文快照、完整提示詞及依序輸入圖角色／SHA 一併保存。

已證實的靜態資料：ANIME01..08 每張 624×300，分成 3×2 的 208×150 格。原始 48 格有 16 格完整水平鏡像，ANIME02 全六格鏡像 ANIME01，ANIME08 全六格鏡像 ANIME07，ANIME06 前四格鏡像 ANIME05；其他格不因外觀相近而鏡像。ANIME04 第六格完全為零。END-2B 左上 512×336 是 2×4 的 256×84 格；OP2_P 為三個 128×119 格；OP9 另有四條 72×27 眼部來源。END-11／11B 只有 [89,37,262,210] 範圍內原始索引不同；選定圖在徽記以外強制重用 END-11 的人物素材，避免編輯造成漂移。

美術準備 READY 契約：18 張生成圖覆蓋 20 個來源，鏡像重用只接受完整原始索引逐格相同的證據。輸出為兩倍原生尺寸、opaque PNG；原始空格保持全黑。END-2 下方、END-2B 紅色緩衝區、OP9 藍色緩衝區及 JIMEN 的各自零索引留白分別保護。BATTLE 實際末列為 246，PaintRect 的 exclusive edge 為 247，不以目測少算兩列。原版字模、文字與執行收據不烘進生成素材。OP5 保留原版 Roman 字樣 IEYASU TOKUGUWA；這是原版拼字，未自動修正。

美術準備 CONFORMED 範圍：20 張選定圖、48 動畫格、16 精確鏡像、一個空格、八個天空格、三個交鋒格及 buffer／留白核對通過；三頁總覽與原始生成圖逐項檢視。核對總表 SHA-256：81b62258274df16fbc1881e642ca64e46ec2527bb991c5a6f16009fb5a7a04e7；選定來源 SHA-256：958c4a55724f7b505a2e2f0e164554f24edf819d2ca747bb0ecfa8df617fa839。第一版 OP9 的緩衝區最後一個實體像素修正於第二版，BATTLE 的最後兩列範圍修正於第三版；舊版與工具快照保留。

繪圖端整合仍為 DRAFT。profiles.json 的 paint_rects、preserve_regions 只是靜態來源中可見區與字區候選，沒有證明動態文字或 actor 的所有寫入端。它們沒有進入 production path。新來源註冊、調色盤、選單遮擋、執行中切換與正常動畫／戰鬥需另經 READY 審查及驗證；本節不把美術準備收據冒充遊戲正常路徑。五個 unsupported OZM、PCX 與全遊戲 HD 完成仍未通過。

## 2026-10-15：20 項不透明來源、固定前景及已觀測遮擋 READY

- 輸入沿前節固定 EXE／DAT、remaining/profiles.json SHA-256 958c4a55724f7b505a2e2f0e164554f24edf819d2ca747bb0ecfa8df617fa839，以及 v3 美術核對。原版畫面與原始索引逐項檢視，20 項的尺寸、四平面及有效區由同一來源資料核對。工具為固定 GUI Docker 的 Go 1.24.13、Python 3.11.2 及 Pillow 9.4.0。
- 已證實：共用不透明入口維持 IDA 9.4 遊戲程式位址 sub_1C1BC／sub_1C084 的原始參數及自然返回契約。既有正常 750M 診斷中，本批新來源僅 OP5 出現；不把其他來源的註冊宣稱為正常玩家路徑驗收。
- 已證實：正常無輸入開場的 OP5 複製返回於 32182644 步；來源已核對的主選單 WINDOW 於 46797968 步自然返回，矩形為 [232,150,408,246]。原始收據為 graphics/static/normal-intro/ready/receipt.json，視窗事件見 graphics/static/desktop-logo/palette/ready/desktop-logo.json。WINDOW 的原始 1152 bytes SHA 及 KAO 逐張來源資格沿既有已驗證觀測，不新增 guest 攔截。
- READY：註冊 20 項 source-specific profile。單一 PaintRect 與多個 PaintRects 互斥；多區域均須非空、在來源內且不重疊。OP9 的有效三區避開藍色緩衝區。來源、幾何、自然返回與完整調色盤資格不放寬，來源尚未在正常流程抽測時明示限制。
- READY：固定前景以來源矩形及十六位索引集合定義。只有已核對原圖內的固定字／緩衝區符合條件時，host 呈現複製當下原生像素，其他像素取 HD。原版來源及目前目的索引仍須完整相符；動態寫字、actor 或其他未知 VRAM 變化仍整區備援，不因落在字區而忽略差異。本契約不實作任何新的動態文字辨識。
- READY：已驗證 WINDOW／KAO 的自然返回記錄 completed_step。較晚返回的視窗或頭像矩形從較早不透明圖的本次可見區扣除；較早者不阻止後畫圖像。保留原始繪圖資料及歷史可見區，僅在 host 呈現副本切割，最多 256 片段。原版視窗保存／還原沿原有來源資格及快照，遮擋備援不依賴 HD 前景檔是否存在。
- 驗收：28 項實際來源的指紋、裁切、缺圖、有效區與固定前景；48 個動畫格、八個天空格、三個交鋒格、四條眼部來源及空格逐項走共同呈現；新舊及同索引遮擋、返回前後、保存／還原與超限保護。正常標題及缺圖各自然 200M，完整 RAM、十五欄及切換符合既有原版基準；正常 750M／390 事件首戰回歸維持。動態戰鬥、未出現來源及五項未知解碼、PCX、正式素材包不外推。
# 2026-10-15：普通選單的 CPU 與 BIOS 缺口，READY

輸入沿用網站 EXE，SHA-256 `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a`。
執行器基準為 dosgolem `a9714ebdab2ad6b529f81225472680f2b11f2842` 加本專案平台修正。
以下位址均為 dosgolem CPU 的線性指令位址，不是檔案偏移。

- 已證實：「歷史簡介」普通滑鼠按下／放開後，第 116451960 步在 `0x11B24` 的 `FD` 停止。收據為 `remaining/source-boundary/menu-history/receipt.json`。
- 已證實：「離開遊戲」第 80014500 步在 `0x26E9A` 的 `CD 10` 停止，EAX 為 3。收據為同目錄的 `menu-exit/receipt.json`。
- 平台前提：STD 只將 DF 設為 1，後續字串操作遞減索引。來源為 [Oracle x86 指令手冊](https://docs.oracle.com/cd/E19620-01/805-4693/instructionset-18/index.html) 及 [Intel 指令手冊](https://cdrdv2-public.intel.com/774492/325383-sdm-vol-2abcd.pdf)。
- 平台前提：BIOS 模式 3 是 80×25 文字模式，頁面位於 `0xB8000`。來源為 [DOSBox Staging v0.82.2 模式表](https://github.com/dosbox-staging/dosbox-staging/blob/v0.82.2/src/ints/int10_modes.cpp)。

審查後的 READY 範圍：支援無 prefix 的 STD，僅更新 DF，保留其他旗標、暫存器與記憶體；不接受尚未驗證的 prefix。使用真正的倒序重疊 MOVSB 驗證效果。BIOS 僅新增 AX=0003 的清畫面文字模式恢復，更新 BDA 的模式、80 欄、25 列、頁面及游標，清除八個文字頁為空白與屬性 07。重置本專案 VBE／埠索引狀態，保留鍵盤 BDA、其他 RAM 與 CPU 狀態。不擴充其他文字 BIOS 服務，不將文字畫面稱為 HD。

修正在 `tools/dosgolem.patch` 與 `tools/video.go`，不修改外部 dosgolem。新收據放既有驗證目錄的 `platform/`，保留原停止版本。驗證包括平台回歸、原普通選單重跑，以及 HD 標題與首戰的原狀態比較。遇到下一項未支援服務時繼續明確停止，不補成功回傳。


## 2026-10-15：20 項來源接入與 OP5 正常路徑，CONFORMED 限定範圍

READY 的來源、美術及多區域／固定前景契約見本檔本批前節。新增 20 項來源均使用各自原始索引雜湊，完整／裁切、48 格動畫、八格天空、三格交鋒及四個眼部區域走實際呈現端測試。原版固定字形以來源索引保留；未知動態寫入不放寬指紋。後畫視窗與頭像以真正返回步數扣除可見片段，保留原始呼叫及矩形。

- 首版四套件通過，研究入口 32.120 秒。無輸入正常 OP5 與缺圖各自然 200M；F6 三次切換，完整 RAM、十五欄與原無輸入基準一致，四張真實 X11 擷取各自符合呈現 PNG。
- OP5 實際替換 142890 原生像素，對應 571560 呈現像素；348797 呈現像素改變。原版主選單矩形 [232,150,408,246] 的 67584 呈現像素排除於標題覆蓋，視窗保持 HD，文字保留。缺標題圖只使標題退回原圖，視窗不變。
- 完整首戰 750M／390 事件、完整 RAM、十五欄、三次切換與三張真實 X11 通過，既有天氣及其他圖層不變。
- 首版總表 remaining/runtime-verification-summary.json SHA-256 2c7c4dcf7cc3ff4188701df1b2900f35ac20522a69dd9fb8e354a02ab8eb7e60。收據及來源快照保留，之後平台版本另放 platform，不覆寫此版。
- 本批新增正常來源只列 OP5。其餘 19 項通過來源與共同呈現測試，未逐項正常抽測。未知文字寫入及其他 palette 仍備援，不宣稱全遊戲 HD 完成。

## 2026-10-15：五個 OZM 尾端及 PCX 的來源邊界

只讀原始 DAT，SHA-256 de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37。診斷工具 tools/hd-boundary-probe.py，Python 3.11.2／Pillow 9.4.0。輸出 remaining/source-boundary/facts.json。以下均為 DAT 檔案偏移與 payload 相對位置，非 IDA 或 guest 線性位址。

| 來源 | DAT 檔案偏移 | payload bytes | 停止位置 |
|---|---:|---:|---|
| B-6／B-7／C-10.OZM，相同 payload | 1062387 為 B-6 | 17689 | 第四平面已有 5999／6000 bytes，下一個 literal 讀取已達 EOF |
| E-4.OZM | 1568487 | 13953 | 第四平面已有 5999／6000 bytes，下一個 literal 讀取已達 EOF |
| OP-FULL.OZM | 2362614 | 63949 | 第四平面已有 31999／32000 bytes，下一個 literal 讀取已達 EOF |
| RAINEND.PCX | 2830614 | 97864 | Pillow 正常解碼為 640×480、256 色索引圖；原版使用端未確認 |

前三平面及第四平面已知前綴的雜湊保存於 facts.json。五個 OZM 只缺右下角八個像素的最高位元，該位元仍未知；不從鄰近像素、其他版本或黑色假設補入。三組 payload SHA 依序為 5adbdc7565e03b848456328c140bb99ae14f6d1d14c8d62dc811b92efaa29aa9、4b542aa5856d39e6d9d3f454cfb7829229804819471f7ded137106e7c5d7e0ea、2e1766de32820be66ded41f855297fe622edcd5fcef6f68eefd6c35656f78bae。

PCX payload SHA-256 78d0179bac1793eabb170925fd9b24378249c83c49320eaac27c8f70f65c1763。原圖為 Rainbow Multimedia 標誌，僅存本機參考；未由尺寸推定正式畫面模式，也未新增 256 色 HD 路徑。正常選單來源查證先遇到 CPU／BIOS 缺口，證據及 READY 範圍見本檔本批平台節。

## 2026-10-15：歷史簡介的 word RCR，READY

STD 版本的普通歷史簡介在第 116451971 步停止，dosgolem 線性位址 `0x11B47`，原始 bytes `66 D1 DB`，為 `RCR BX,1`。EXE 與執行器基準沿用本批前節。收據為 `remaining/source-boundary/menu-history/platform/ready/receipt.json`。這次已自然載入 OP2.OZM，但尚未完成該畫面繪製，不列為 HD 正常驗收。

再次比對平台路由，沿用「復古平台規格優先」。[Oracle RCR 手冊](https://docs.oracle.com/cd/E19620-01/805-4693/instructionset-117/index.html) 與本批已引用 Intel 指令手冊定義：16 位元原值右移一位，原 CF 進入 bit15，原 bit0 進入 CF；單位旋轉的 OF 為結果 bit15 XOR bit14。其他旗標不變。

READY 僅新增 operand16、D1 /3、暫存器目的、固定一位的形式。保留高 16 位、其餘暫存器及記憶體；未驗證的記憶體、立即數次數與其他 prefix 保持拒絕。以全部八個目的暫存器、CF 兩種輸入、正負／零／單位元與原 OF 兩值驗證結果及不受影響狀態。實作仍只改專案平台補丁，不修改外部 dosgolem。

## 2026-10-15：RCR 平台切片與歷史簡介查證結果

RCR 的 READY 範圍實作於專案補丁，八個目的暫存器與全部列出的 carry／overflow 案例通過。最終四套件通過，研究入口 38.639 秒。正常 OP5 再次自然 200M、完整首戰 750M／390 事件的完整 RAM、十五欄、原圖／HD／還原及 X11 通過。缺圖證據沿用 STD 平台版本；來源核對證實兩版只差 CPU 補丁及 CPU 回歸測試，呈現程式及全部圖片完全相同。

最終平台總表 remaining/runtime/rcr/verification-summary.json SHA-256 bd9a8808a12469a7dffb63f22436ac4bf50c44ab3d32c2318b100f7ae641b216。普通歷史簡介收據 SHA-256 d30733b5b420cee4f1e9ac02cee5a326d4ab57b120232135d0750b47587dbaaa；普通離開的版本及限制保持平台節記錄。

歷史簡介普通滑鼠按下／放開各在 80M／81M，自然執行到 600M 預算，未遇未支援服務。正常載入 OP2、OP2_P、OP4、OP5、VISUAL、6、11、19、28、43、OP8、OP9、OP-FULL，550829475 步再載入 OP5，最後原生畫面目視為主選單。這證明這一段原版普通操作；特殊繪圖尚未被現有共同不透明觀測涵蓋，不列成開場 HD 正常驗收。

## 2026-10-15：OP-FULL 單次正常 RAM 的尾端恢復

以下是原版正常載入 RAM 的證據，不是補入原 DAT 的 byte。輸入 DAT／EXE 與本批前節一致；工具 tools/hd-tail-recover.py，Python 3.11.2。位址基準為 DAT 檔案偏移、平面內 row-major 偏移及載入事件保留的 guest 線性基址，三者分開記錄。

- 原版 OP-FULL 載入返回為第 529234797 步。前三個完整平面的 SHA 與靜態解碼逐一一致。
- 第四平面只有最後一 byte 未知。已知 column-major 前綴轉成 row-major 後，固定其他 31999 bytes，完整枚舉 0..255；只有 170／0xAA 的完整平面 SHA 符合原版收據。
- 四平面全部核對一致，完整 640×400 索引 SHA-256 c9b332bfc54a189abfc4b7caf3c1821738624dedd41f19332ee5e4b44a1716a3。來源證據 remaining/source-boundary/OP-FULL-runtime/receipt.json SHA-256 d5b43b21ed717cef2a426c2fff9cb46903f916d88651345030b9e05403ee62cf。
- 已證實範圍只限該次正常 RAM 值。未證實其他播放歷史的尾端穩定性、原始檔缺失 literal 或 HD 繪圖端。DAT 及靜態解碼器保持不變；覆蓋表仍列五項靜態解碼未知，另列一項 runtime tail observation。
# 2026-10-15：歷史簡介特殊繪圖，只讀診斷 DRAFT

問題：原版普通歷史簡介已回到選單，但現有共同不透明觀測只記到 OP1／OP5，不能驗收其他開場 HD。輸入沿用網站 DX5.EXE SHA-256 `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a`、DAT SHA-256 `de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37`。

- 已證實：IDA 9.4 正式資料庫隔離副本的 `sub_11AEB` 使用原始 `dword_386D4[eax]`，在 `0x11B24..0x11B56` 以 STD、LODSW、SHR、RCR、STOSW 修改四個候選平面的前 16002 bytes。來源與位移語意仍需正常動態對帳。
- 已證實：`sub_11BA6`、`sub_11C16` 等外層選擇四平面，再呼叫 `sub_11B60`。後者將六個原始參數轉成逐列 `sub_1C9CE` 複製；`sub_1C9CE` 由 `0x1C9DB..0x1C9EF` 讀目的、來源及 bytes，經 REP MOVSD／MOVSB 複製。
- 已證實：IDA 的 `0x129DF` 是 `0x386D4` 的直接寫入端。直接引用不充分，初始來源所有權與 alias 仍需追其上下文及正常 RAM。
- 原始定位保存在 `remaining/history/ida/range-11a00-11d00.json`、`range-11aeb-12900.json`、`range-1c9ce-1caa0.json`、`range-12900-12b20.json` 及 `art-consumers-386d4-11aeb.json`。所有位址為 IDA LE loader 線性 EA，檔案偏移逐列獨立保存；工具名不代表語意已證實。

DRAFT 診斷只在明確啟用 `-hd-opening-trace` 時讀取 backing RAM。記錄上述外層與位移入口、原始堆疊返回、參數、`0x386D4` 候選四平面、載入名稱提示及自然返回原生畫面；名稱提示不當來源識別器。每來源提示／入口只抽樣第 1、2、3、8、32、128、256、512、1024 次，上限 128 筆及 32 層 pending。輸出原始平面、索引、原生 PNG、palette 與 SHA，只有本機研究用途。

驗證包括完整記憶體不變、錯誤指標／堆疊界限、抽樣上限與普通歷史簡介的 600M 同輸入對照。只有原版自然返回才記錄完成；不新增繪圖替換、不略過 CPU、不修改素材或將猜測寫入正式來源表。動態證據足以給出來源、幾何、遮擋與失敗契約後，再審查 READY。

# 2026-10-15：歷史簡介完整複製路徑，READY

沿用本批 DRAFT 的固定 EXE／DAT、IDA 9.4 位址空間及 dosgolem 基準。四套件通過，研究入口 31.201 秒。普通歷史簡介的唯讀觀測自然 600M，完整 RAM SHA-256 `c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2`、十五欄與原基準完全一致。65 個抽樣自然返回、零丟棄，78 項建置來源與快照一致。來源總表 `remaining/history/source-verification.json` SHA-256 `9c42fb441cacd903eb4c82c902a9faadbb774e2bc76407eef9009940b22bac5b`。

- 已證實：`sub_129C4` 的 IDA 線性 `0x129DA..0x129FD` 將原始 globals `0x38C00/0x38BF8/0x38C04/0x38BFC` 複製至 `0x386D4..0x386E0`。兩組是指標別名，沒有第二份獨立像素緩衝區。動態四次完整複製的兩組指標逐一相同。
- 已證實：`sub_11F14` 在 `0x11F2A` 固定目的 `0xA0000`；`0x11F48..0x11F6E` 選四平面，每平面由 `dword_386D4[eax]` 複製 `0x7D00` bytes。四個來源 OP4／OP5／VISUAL／OP-FULL 在自然返回時，全部 256000 個目的索引像素與來源相等，來源 bytes 前後不變。
- 已證實：13 個 sub_11AEB 抽樣與逐指令導出的位元位移相符，且該函式不改原生畫面。每平面第一個 word 保留，byte 偏移 2..16001 右移並接前 word 最低位元，其餘 bytes 保留。這不把局部／捲動路徑宣稱為 HD 完成。
- 已證實：OP-FULL 的第四平面尾端 0xAA 在兩次普通執行都相同；完整索引 SHA-256 `c9b332bfc54a189abfc4b7caf3c1821738624dedd41f19332ee5e4b44a1716a3`。只加入明示的 runtime_observed 來源，不把靜態解碼器的五項未知改成已解碼。

READY 範圍：在既有共同不透明觀測增加 `sub_11F14` 完整複製入口。每次入口及返回都確認兩組平面指標仍 alias，來源尺寸固定 640×400、四平面各自精確 SHA、返回 EIP／ESP+4 及完整目的索引一致。其他來源與既有局部矩形契約不放寬。完整複製入口撤銷舊呈現資格；未知來源、alias 改變、來源改變、未返回或目的不符時保持原版。新來源仍接受既有精確 palette 檢查，淡入／淡出退回原圖。

OP-FULL 已依使用者選定 B 戰國手繪風生成。原始圖 SHA-256 `d7cef9b5ac9f237baac15550da0c6c33223116cd8f90032bdb2c80f36b218af7`，完整提示詞及兩張有序參考圖的用途／SHA 保存於 `catalog/art/opening-full-generation-v1.json`。校準只整幅縮為 1280×800，不重畫文字或補原版 bytes。準備工具保留原始圖、校準圖、來源證據與工具全文；原版索引另以 runtime 來源收據保存，不混入靜態解碼完成數。

驗收：新來源與來源不符、alias 改變、缺圖、palette、自然返回、全畫面撤銷及原版記憶體不變測試。普通歷史簡介用同一兩筆滑鼠輸入驗證 OP4／OP-FULL 的真實視窗 HD、原圖及還原，並與同一 600M 原版 RAM／十五欄比較。既有標題與完整首戰回歸；本次不更動原版規則、存檔、字模或公開權利。其他局部／捲動開場與動畫另列未完成。

## 2026-10-15：OP-FULL 專用色盤與開場色彩效果，READY

勘誤：完整複製接入的第一次 GUI 只在 252M 短暫匹配 OP4 共用色盤，之後無法啟用 OP-FULL。這不是完整開場 HD 驗收。原收據與三張 OP4 擷取保留於 history/desktop；該版本標題及首戰回歸通過，不能據此外推新開場。

原版六次亮度取樣及兩次晚期循環取樣皆走同一普通歷史簡介，600M、完整 RAM 與十五欄符合基準。兩份驗證器的確切版本依記錄 SHA 恢復／保存於 palette/verifier-source.txt 及 palette-late/verifier-source.txt，舊收據不改。色彩契約總表 history/palette-effects.json SHA-256 `b3fd4cfa4838b8d4b2e19028626c1df02898ec3c97ef8a3e472ce3d377423431`。

- 已證實：IDA 9.4 的 sub_1BC0E 於 `0x1BC58..0x1BC99` 循環 63 次，在 `0x1BC65..0x1BC74` 對 48 通道遞增，只有新值嚴格小於目標才寫入。零通道保留零，非零目標最後顯示目標減一。六次原生色盤逐通道符合此亮度上限模型。
- 已證實：sub_1BDCB 在 `0x1BDED..0x1BDF9` 先複製目標色盤，再於 `0x1BE15..0x1BE52` 做 21 次循環；通道不等於 21 才遞增。晚期兩次全部 48 通道分別符合第 14、20 次循環，包含超過 DAC 六位元範圍後的遮罩結果。不是普通的等比淡出，不用 Alpha 代替。
- 平台契約：VGA DAC 的六位元輸入依 [DOSBox Staging v0.82.2](https://github.com/dosbox-staging/dosbox-staging/blob/v0.82.2/src/hardware/vga_dac.cpp) 的公開介面處理；dosgolem 固定基準的 internal/machine/machine.go Palette() 使用 `(v<<2)|(v>>4)` 展開為八位元。這是平台前提，不再逆向硬體 driver 或時序。
- 已證實：OP-FULL 的 16 色目標與共用色盤不同。原版雲帶為藍白色；第一版製作參考的共用色盤把它顯示為綠色。內建 imagegen 僅校正第二版雲帶，人物、題字及構圖保持既定 B 風格。原始第二版 SHA-256 `891cddb971b142ffea88841af6b4f771661a203344884ce33af4364af4abb2b2`；第一版與提示詞保留。

READY 僅允許 OP4 及 OP-FULL 的已證實色盤族。每次呈現確認完整 48 通道匹配目標、亮度上限族，或 OP-FULL 的 0..21 循環族；其他色盤保持原版備援。來源平面、自然返回、目的索引與遮擋資格全部維持既有檢查。

HD 色彩轉換明示為 presentation approximation。亮度族將原生上限正規化至新圖 0..255，逐通道取較小值；循環族將新圖 RGB 量化至六位元目標，套用已證實循環及 DAC 遮罩再展開。原版前景直接沿用原生 RGB。只改 host 呈現，不改 guest palette、CPU、等待或存檔。單元測試涵蓋全部亮度級、全部 22 循環級及未知色盤拒絕；GUI 要驗證多個淡入與循環時點，不用短暫匹配代替整段驗收。

## 2026-10-15：完整開場複製與色彩效果，CONFORMED

適用範圍限上述 READY 的固定網站 EXE、DAT、dosgolem 基準及普通歷史簡介兩筆滑鼠輸入。新的工具與收據入口為 tools/verify-hd-opening-summary.py、remaining/history/verification-summary.json，總表 SHA-256 `2ab4e2768c2fc6f59671f4102e47c2ec7cf0dfe3f9f9b5bbb44206fbfd190554`。工具、二進位、來源快照、圖像及各次完整狀態雜湊由總表串接。

- 已證實：sub_11F14 的完整複製可讓 OP4 與 OP-FULL 走既有 host HD 呈現。入口及自然返回核對來源別名、四平面 SHA、完整目的索引；新完整畫面撤銷舊劇情、視窗、頭像、地形與戰場框架資格。VISUAL 亦須重新核對四平面及完整目的，才能啟用既有捲軸；其他局部／捲動端不因此獲准。
- 最終固定 GUI image 四套件通過，研究入口 31.477 秒；實際 OP-FULL／VISUAL fixture、錯誤 alias、來源／目的變更、自然返回、舊圖層、全部亮度級、22 循環級及未知色盤拒絕皆通過。二進位 SHA-256 `34fe9c4d32b94f928269c674bb4659c1ee693f1be1bb2a3a703bf233a351cbd1`，五次 GUI 使用相同 82 項來源快照。
- 普通歷史簡介的正常、缺圖、八時點色彩模式各自然 600M；完整 RAM 皆為 `c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2`，十五欄與原基準逐項相同。正常及缺圖各五次、效果模式十七次 HD 切換的前後原版 proof 完全相同。
- 六個淡入時點實際為 OP4 241M／246M／249M、OP-FULL 534M／540M／544M；全部 48 原生色盤通道與契約一致。兩個循環時點為 547M／549M，分別匹配第 14、20 級。八組 X11 原圖／HD／還原共 24 張實際視窗與呈現 PNG 完全相同，HD 區域公式及矩形外保護通過。這些是八個抽樣時點，不宣稱每個過渡中的未完整色盤都啟用 HD。
- 個別移除第二版 OP-FULL 時，該圖退回原圖且零像素變動，OP4 仍為 HD。正常標題 200M、首戰 750M／390 事件的完整 RAM／十五欄通過；首戰三張完整畫面亦與前個 RCR 版本相同。
- 效果模式第一次最後擷取從 549M 跨至 550M 的未完整色盤，按既定條件退回原版。上層 effects 保留七組成功擷取及失敗驗證器；只調整觀測起點為先前原生證據的 546M／548M，在 effects/ready 乾淨重跑通過。沒有放寬 48 通道資格或更改原版執行。

第二版手繪原圖、校準圖、完整提示詞與第一版都保存於私人素材庫。校準圖 SHA-256 `b460fc80cbcafa150cbeb4be991d5217c1a8bf29c5339f6d768981be4a7dc717`，1280×800；正式來源仍為兩次正常 RAM 一致的 runtime_observed 索引。靜態解碼的五項未知保持原樣。HD 色彩轉換保持呈現近似分類，不宣稱原版像素或硬體時序一致。

本批沒有新增玩法、輸入或存檔分支。CONFORMED 不涵蓋局部／捲動開場、其他動畫、四項缺美術 OZM、PCX、全遊戲 HD 或三平台打包。

## 2026-10-15：未位移局部繪圖的來源證據，DRAFT

沿用上述固定 EXE／DAT、IDA Pro 9.4 的 LE loader 線性 EA 及普通歷史簡介的 600M 原版收據。工具 tools/verify-hd-opening-partial-source.py 在既有 GUI image 的 Python 3.11.2 執行，只讀 probe 的 78 項建置來源快照、候選平面、原生索引與靜態解碼。沒有重跑或改寫原版。新證據 partial/source-verification.json SHA-256 `e179e94ac069b7b8a255a38cd4cb0afcc1d3c33e64bd1c0b8ea9ecf0862f5245`。

| 原始入口與定位 | 已核對來源 | 來源裁切 | 原生目的 |
|---|---|---|---|
| sub_11E37，IDA 線性 EA 0x11E37 | OP2_P.OZM，三次自然返回 | 首參數 0／16／32 bytes，分別為 x=0／128／256；各 128×119 | (248,50)，128×119 |
| sub_11DCB，IDA 線性 EA 0x11DCB | 6／11／19／28／43.OZM，各一次 | (0,0)，320×150 | (160,50)，320×150 |
| sub_11BA6，IDA 線性 EA 0x11BA6 | OP9.OZM，三次自然返回 | 首參數 0／26／52 bytes，分別為 x=0／208／416；各 208×200 | (216,30)，208×200 |
| sub_11C16，IDA 線性 EA 0x11C16 | OP9.OZM，三次自然返回 | 首參數換算成 source row 與 byte column；實際裁切逐筆保存 | (216,98)，72×27 |

原始指令與檔案偏移另見 history/ida/range-11aeb-12900.json，不將上表 EA 與檔案偏移混用。十四次皆確認來源候選指標等於最近載入的四平面基址、原始平面前後不變、目的矩形逐像素等於正確來源裁切，矩形外原生索引完全不變。載入名稱只是查找提示，實際來源另和靜態解碼比對。

- 已證實：三格 OP2_P 及五張劇情的完整來源與靜態解碼相同。
- 已證實：OP9 的有效美術區與靜態解碼相同；不同像素全數位於先前校準清單的 reserved_buffer_rects。不能要求使用暫存區後仍匹配整張來源 SHA，也不能略過有效美術區識別。
- 已證實：OP2_P、OP9 的這些自然返回色盤與共用穩定色盤不同。原始 48 通道保存於逐筆證據；正式接入前須界定精確允許值，不以關掉色盤檢查解決。

這批仍為 DRAFT。下一步審查未位移繪圖的來源資格、固定／變動參數、暫存區、色盤與後畫遮擋，再實作。OP2／OP8 的來源位移另有證據，不能直接套用本表；沒有把十四次原版複製列為新的 HD 驗收。

## 2026-10-15：未位移局部繪圖審查與勘誤，READY

重新逐筆讀取 partial/source-verification.json。十四筆的 reserved_buffer_changed_pixels 都為零，包含六個 OP9 抽樣；來源皆為 exact whole resource。上一節「OP9 不同像素全在暫存區」是空集合的檢查結果，不能據此宣稱本次觀測到暫存區改寫。原始證據不改，正式接入維持完整四平面指紋，不新增放寬來源的遮罩識別。

輸入仍為 DX5.EXE SHA-256 `2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a`、DX5.DAT SHA-256 `de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37`、dosgolem 固定 commit `a9714ebdab2ad6b529f81225472680f2b11f2842`。IDA 9.4 range-11aeb-12900.json 的原始名稱、bytes、檔案偏移與 LE loader 線性 EA 保留。原版普通歷史簡介使用同一 80M／81M 按下及放開，無遊戲狀態注入。

- 已證實：sub_11E37 取首參數作來源 byte 偏移，目的 (248,50)、128×119；原版三次使用 0／16／32。
- 已證實：sub_11BA6 目的 (216,30)、208×200。IDA 線性 EA 0x12860..0x128B8 的五個 push／call 使用 0、0x1A、0x34、0x3E80、0x3E9A。前三筆動態來源及目的已核對，後兩筆依同一指令及已解碼來源接受共同測試，GUI 再抽樣。
- 已證實：sub_11C16 目的 (216,98)、72×27。IDA 線性 EA 0x12804..0x1284E 的四個 push／call 使用 0x3EB4、0x4724、0x4F94、0x5804。前三筆動態核對，第四筆依同一指令與靜態來源接受共同測試，GUI 再抽樣。
- 已證實：sub_11DCB 以來源 (0,0)、320×150 畫至 (160,50)，五張歷史劇情動態核對。可沿已註冊劇情來源與既有呈現，不改素材。
- OP2_P 與 OP9 各有一組在其全部已抽樣返回一致的 16 色值，來源見逐筆 active_palette。只在對應精確來源增加此組允許色盤，原有共用色盤仍接受；任何其他色盤仍退回原圖。兩者不套用 OP4／OP-FULL 的色彩轉換。

READY 契約：三個局部不透明入口轉成既有 graphicsDrawRecord，保存原始來源偏移、固定尺寸、目的、自然返回 EIP／ESP+4 與來源／目的雜湊。僅接受上述已證實來源偏移及對應 OP2_P／OP9 名稱；入口及返回均確認四平面別名，完整來源四平面匹配後才建立資格。後畫固定目的扣除舊圖層，無法識別的輸入仍撤銷被蓋區域並顯示原版，不沿用舊圖。

歷史劇情入口沿 hdLive 的已驗證載入資格；入口及自然返回另確認別名、目前完整來源及目的 320×150 索引一致。錯誤來源、來源／目的改變、未自然返回、錯誤參數或缺圖皆退回原版；完整畫面仍撤銷舊資格。所有處理限 host 畫布，原版指令、VRAM、palette、輸入及存檔不改。

驗收包括十四筆實際來源 fixture、三格交鋒、五張武將、四格眼部、五張歷史劇情、缺圖、來源／alias／目的變更、錯誤偏移、後畫遮擋及完整記憶體不變。普通 GUI 沿相同歷史選單路徑核對原圖／HD／還原與可擷取格，完整 600M RAM／十五欄符合同一原版基準；新版本標題及首戰回歸。GUI 未擷取的格只列共同測試，不稱正常視窗完成。OP2／OP8 的位移、其他動畫、四項缺美術 OZM、PCX 與三平台打包另列。

## 2026-10-15：未位移局部接入，CONFORMED

適用範圍限前節 READY 的四個入口、既有素材與原版普通歷史簡介。來源四平面指紋與自然返回保持原規格，沒有忽略 OP9 暫存區或寫入 guest。最終四套件通過，研究入口 35.467 秒；首輪測試將既有 NRGBA 指標誤作介面轉型，原編譯錯誤留 runtime 上層，修正測試後於 runtime/ready 重建。二進位 SHA-256 `af4de66880fc2e36a859ce8e865873c7cebdd2914bda2c0e0f47a505ac75647d`，84 項來源快照。

- 三格交鋒、五個武將來源偏移、四格眼部及五張歷史劇情皆已接入。十四個實際原版 fixture、原圖／缺圖、來源／目的／alias 變動、錯誤偏移、自然返回、記憶體不變與後畫扣除測試通過，其他既有測試維持通過。
- 正常 X11 已核對十四組：三格交鋒、四格眼部、前兩張武將及五張劇情。缺圖模式十七組包含五張武將；只移除 OP2_P、OP9 與 11 的素材，其他四張劇情仍為 HD。共 93 張原圖／HD／還原實際 X11 與呈現 PNG 完整吻合，來源裁切、HD 取樣公式及配對 proof 通過。正常與缺圖分別 29／35 次 F6 切換保持原版狀態相同。
- 正常擷取第三張武將時從所選畫格跨到下一格，520100001 步停止；缺圖擷取全部完成，但外層期限於 592333333 步終止，沒有完整收據。原始輸出、完整驗證器全文與已完成畫面保留，兩者不充當完整 600M 驗收。
- 按正常玩家路徑與比例抽樣規則，將配對畫面與完整狀態分開補驗，未改 renderer 或放寬來源。新的 capture 核對工具重新逐張驗證既有圖片及 F9 原生索引，不只相信 completed-stages 清單。另以同一二進位、同一輸入與 84 項來源，兩個真實視窗各自然跑完 600M。正常／同三項缺圖的完整 RAM 皆為 `c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2`，十五欄與原版基準逐項一致。
- 兩份完整收據的 graphics.completed_returns 都為 18，原完整複製版本為 6，差十二次正好涵蓋三格交鋒、四格眼部及五張武將；新增五次歷史劇情自然返回的 eligible 均為 true。這證明正常執行的來源資格，不取代未擷取畫格的 HD X11 證明。
- 新版本標題 200M、首戰 750M／390 事件的 RAM、十五欄及 X11 回歸通過，首戰三張完整畫面與前版相同。慢速擷取器另改用現有 F7 脈衝控制短武將段，收尾期限增加；修正後未重跑全十七組，不聲稱已驗證這個操作策略。

唯一總表 partial/verification-summary.json SHA-256 `ba0de3c64fcf7fffb1ee390c3e99d25b240ed874162d7ee3c68a399c0b5d03f3`。配對畫面總表 partial/capture-verification.json SHA-256 `3f735af6968610df41a0fa161f4c4795ec9e8f42677572199676d54071372138`。兩份表將正常 HD 十四組、缺圖十七組、完整狀態及未抽樣的三張正常 HD 武將分開。

CONFORMED 只表示上述入口與已列契約通過。其他三張正常 HD 武將保持來源／共同呈現測試分類；OP2／OP8 捲動、其他動畫消費端、四項部分來源 OZM、PCX、全遊戲 HD 與打包仍未完成。

## 2026-10-15：OP2／OP8 捲動來源，DRAFT

沿用同一固定 EXE／DAT、IDA 9.4 LE loader 線性 EA、Python 3.11.2 及既有普通 600M 只讀探針。新工具 tools/verify-hd-opening-scroll-source.py 核對原始 source snapshot，不重跑或改寫原版。scrolling/source-verification.json SHA-256 `b46d791603cf03c77261106439f703744edb5f67bceecb86bf60a14f4508e319`。

- 已證實：OP2 六筆、OP8 七筆原版繪製的已知 640×200 來源，與先前 sub_11AEB 位元語意的累積轉換一致。第一個 16-bit word 保留；其後線性索引 i 在 N 次位移後來自原始 max(15,i-N)。N 由該來源的原版繪製次數減一，逐筆數值與指令／原始資料核對，沒有用畫面名稱推定。
- 已證實：OP2 固定來源 x=240，240×150，目的 (200,50)；OP8 固定來源 x=440，208×200，目的 (216,30)。原始 helper 使用線性 80-byte stride；OP8 右緣跨到下一列，不能作普通矩形裁切。十三筆目的索引皆等於當次來源，來源平面前後不變。
- 已證實：OP8 N=0／1／2／7 時，最後一列分別有 8／7／6／1 個像素超過正式 200 列來源。這些值在候選緩衝區可讀到，來源美術身份未知，另存座標，不當原圖像素或猜補。較大位移抽樣沒有來源外像素。
- 已證實：每來源抽樣有全黑與一組穩定非黑色盤。OP2 的非黑值與本批 OP2_P 的完整 48 通道相同；OP8 與 OP9 相同。IDA 線性 EA 0x1201A..0x12021 顯示第一次 OP2 繪圖後呼叫 sub_1BBAC(4)；0x126DD..0x126E4 為 OP8 繪圖後呼叫 sub_1BBAC(2)，兩者前面都先呼叫 sub_1BBAC(1)。檔案偏移各自在原匯出中保存。

剩餘 DRAFT 工作為 host 位移呈現、來源外像素保護、這兩組精確色盤及四平面 alias／來源變動的失敗契約。證據已足夠縮小下一個切片，不追硬體週期或其他無關常式。

## 2026-10-15：四項 EOF 來源的已知區域與手繪草稿

工具 tools/hd-prefix-extract.py 在固定 GUI image 讀取未改的 DAT 與既有 decoder。B-6／B-7／C-10 的 payload 完全相同；E-4 是另一組。每組保留前三個完整平面及第四平面 5999 個已知 bytes，轉成逐列前綴，沒有填最後的 byte。每張 320×150 有 47992 個已知像素，未知八像素為 (312,149)..(320,150)，參考 PNG 僅將該區設透明。來源檔與雜湊見 source-boundary/art-prefix/source-prefix.json，色盤標為研究用共用色盤，不稱正常原生擷取。

內建 imagegen 先製作 E-4 營帳，再依 B-6 來源構圖加入左前景紅甲武將。E-4 原始圖 1832×858，SHA-256 `d4568ad6fcf71d5b46438e7717c28afafaaa2175545102e5cc5df1ead84a1ed9`；B-6 原始圖 1832×859，SHA-256 `4e922e9d766114ac5882a95704e1576b950fd95453962a1371c410591a84eeda`。兩圖與有序參考圖用途、SHA、完整提示詞已保存私人 repo。八個未知像素、幾何及正式來源識別仍待審查，保持 draft，不把產圖當成來源已解碼或遊戲內可替換。

## 2026-10-15：OP2／OP8 捲動接入規格，審查後 READY

前節 DRAFT 的十三筆來源證據與 IDA 9.4 原始指令已足以限定此切片。輸入 EXE／DAT、位址空間、執行器版本與普通 80M／81M 滑鼠事件均沿前節，不改 guest。來源證據 SHA-256 為 `b46d791603cf03c77261106439f703744edb5f67bceecb86bf60a14f4508e319`。

- 已證實：IDA LE loader 線性 EA sub_11EA4 將來源 byte 偏移 30、30 columns、150 rows 複製到 0xA0FB9；sub_11C83 使用偏移 55、26 columns、200 rows，目的 0xA097B。80-byte stride 與四平面 alias 沿原始指令。原始檔案偏移另見 range-11aeb-12900.json。
- 已證實：0x12029..0x1213A 的迴圈總計 10+4×25×2+15×2=240 次位移；0x126EC..0x127F8 總計 110+110+220=440 次。兩者初次繪圖為 N=0，後續每次繪圖前位移一次。
- 審查：OP8 的 8 個跨列尾端像素不屬正式 640×200 美術。入口仍讀取原版實際使用的額外一 byte／平面，核對自然返回前後不變與完整目的；只有映射回已證實來源的像素可使用 HD。其他像素保留當下原生圖，不填來源或 DAT。
- 審查：OP2 已校準 PaintRect 為 (0,0)..(480,150)，OP8 為完整來源。位移後逐點轉回原始來源座標，再檢查 PaintRect 與固定前景限制，不用移動後矩形推測素材區。跨列依線性索引處理。

READY 契約：

1. 第一次來源須以既有 640×200 完整四平面 SHA 識別為對應 OP2 或 OP8，並記錄四個 alias。後續僅沿相同指標、對應入口及連續 N+1，逐點核對完整 128000 個已知來源索引等於原始轉換 i<16 ? i : max(15,i-N)。超出 240／440、未知來源、錯誤 alias、改變尺寸或遺失來源資格都退回原版，不搜尋碰巧匹配的位移數。
2. 保存入口四平面實際讀取區雜湊，OP8 含額外一 byte。只在自然返回 EIP 與 ESP+4、來源未變且完整目的等於當次來源複製時取得資格。來源外像素另作未知標記；已知區索引雜湊與完整目的雜湊分開。
3. 後畫固定目的扣除較早圖層，失敗也不復活舊圖。呈現前再次核對可見區已知原生索引；未知、素材保留區與固定前景用原生圖，其他點依原始來源座標及既有縮放取樣。未知遮擋或缺圖維持原有備援。
4. 每次呈現只接受共用原色盤、前節兩組完整穩定色盤或完整全黑色盤。OP2 穩定值等於 OP2_P，OP8 等於 OP9。全黑階段的 HD 亦全黑；不開放其他過渡色盤。只改 host RGB 呈現，原版 DAC 與時間保持原樣。
5. 驗收採十三筆真實來源 fixture、連續轉換／超界、alias／來源／目的／guard 變動、缺圖、未知尾端及矩形外保護。GUI 抽樣早／晚位移，原圖／HD／還原及個別缺圖；同一二進位完成普通歷史 600M 完整 RAM／十五欄，另抽測首戰回歸。沒有逐幀 X11 擷取的位移只列來源及共同呈現測試。

此 READY 沒有改變既定全遊戲 HD 目標或美術風格。其他動畫消費端、四項部分來源與 PCX 仍另列；單一切片完成不能宣稱全遊戲 HD 或三平台封包完成。

## 2026-10-15：OP2／OP8 捲動接入，CONFORMED

適用範圍限前節 READY 的固定 EXE／DAT、dosgolem 基準、兩個自然繪圖入口及既有校準素材。十三筆原版 fixture、OP2 N=0..240 與 OP8 N=0..440 共 682 個連續繪圖狀態、alias／完整來源／目的／guard 變更、自然返回、缺圖、未知尾端及矩形外保護均通過。四套件通過，研究入口 49.585 秒。二進位 SHA-256 `5df4a99c1e4a7f46d75ee5ba51dfc4843d34ad84c6e3555be4c99fed7fed31c8`；建置後 86 項整合來源保持凍結。

- 正常真實 X11 抽樣為 OP2 N=5／127，OP8 N=1／127／255。個別缺 OP2／OP8 素材時為 OP2 N=5／127，OP8 N=2／128／255。各組原圖／HD／還原具有相同原版 proof 與步數，恢復圖 SHA 相同；三十張實際視窗與呈現 PNG 完整相同。兩模式各十一個 F6 切換不改原版狀態。
- 原生目的逐點核對線性來源映射及已知區雜湊。正常 OP8 N=1 的七個來源外像素、缺圖 N=2 的六個均保留原生圖。HD 取樣與完整視窗矩形外保護符合獨立公式；缺圖模式整張視窗零變動。沒有將來源外可讀 bytes 當成原美術。
- 兩次普通歷史皆自然完成 600M；完整 RAM 為 `c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2`，十五欄、原生圖與 PCM 和原版基準逐項相同。graphics.completed_returns 皆為 700，相較已驗收未位移版本 18 新增 682 次，符合兩段繪圖狀態數。
- 首次首戰已完成 750M，但驗證器讀取 source-snapshot/tools/LICENSE-toolbar-font.txt 時缺檔，未產生成功驗收表。這是驗證準備錯誤；上層原始輸出保留。補齊 86 項與早期 frozen manifest 相符的來源快照後，在 regression/ready 以相同二進位及命令乾淨重跑。完整 RAM `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`、十五欄與三張完整視窗皆與前版一致。
- 正常／缺圖的來源快照於執行途中補齊，全部逐項核對早期 frozen-integration-input.json、目前來源及收尾收據，Go 沒有中途修改。已在 AGENTS 的重生入口補上執行前保存完整 tools 快照的命令，避免再次遺漏授權依賴。

唯一總表 scrolling/verification-summary.json SHA-256 `6d33f51a4b447c21fc4747a5010922a92e06074d27df7158703d78278411415a`。正常表為 `c5c19055c0037fb8b44e9b86213a139b5f40bd09d1a06b655ac2753198a3dab0`；缺圖表為 `3852b6df45d0b5c4617fca2d308151caf443d857f48fe16f9bb00246e6df4d99`。總表串接三份完整收據、258 項來源快照、凍結清單、二進位、測試及全部配對圖。RGB 素材及位移後 HD 取樣屬呈現近似，不稱原版像素一致。

本次沒有改 guest、輸入、規則、DAC、時間或存檔。其他位移狀態未逐格 X11 擷取，保持原版來源及共同呈現測試分類。四項部分來源 OZM、其他動畫消費端、PCX、全遊戲 HD、桌面 CD 播放與三平台打包仍未完成。

## 2026-10-15：四項部分來源故事圖，DRAFT

目標為 B-6、B-7、C-10、E-4 的已知美術區與既有故事繪圖消費端。固定 DX5.EXE／DAT、dosgolem commit、IDA 9.4 LE 線性 EA 沿既有接入證據；不改原版或填尾端 byte。原始已知區證據在 remaining/source-boundary/art-prefix/source-prefix.json，來源工具、DAT 與 facts 雜湊均可回查。

- 已證實：三個平面各有 6000 bytes，第四平面有 5999 個逐列已知 bytes。三項 B-6／B-7／C-10 payload 完全相同，E-4 為另一組。每張 47992 個完整已知像素、右下 (312,149)..(320,150) 八個高位未知；低三位已有原始資料，但不得將高位猜作零或推定黑底。
- 已證實：既有 sub_1C1BC 的故事參數為 (0,40,150,80,0xA0FB4)，原生目的 (160,50)..(480,200)；sub_11DCB 使用相同來源／目的，原始證據及正常抽樣見前述故事與未位移切片。只有四項來源實際走這些已核對入口、參數及自然返回時，才能建立呈現資格。四項未在正常玩家路徑抽樣，不能據此保證其他消費端。
- 美術審查：重新檢視兩組原版已知區及既有生成圖。B-6 的左前景紅甲、中央藍甲及右側紅金武將構圖一致；E-4 沒有左前景武將。沿用已選 B 風格與整幅畫布，不重畫或加角色。既有圖片比例可用共同最近鄰校準至 640×300。
- 下一步準備已知索引與幾何，再審查來源前綴、自然返回完整 guard、原生索引、來源外保留、色盤及缺圖契約。草稿未直接接入正式呈現。

## 2026-10-15：部分來源故事圖審查，READY

準備 metadata 為 remaining/partial-story/profiles.json，SHA-256 `ab4288a9b554ce811f3ad1e2b09e448eb5bf66b0b2018072f109b2b6cefbc2ca`。兩組原圖／HD 對照已逐項檢視；整幅圖按既有整數最近鄰流程校準至 640×300，原始生成圖與提示詞不改。原版參考與已知索引只留本機，純 HD 素材另保存私人 repo。

1. 只接受四個已確認名稱及 40 columns／150 rows。從既有 loader 的四個指標讀取 80-byte stride，每平面擷取 6000 bytes；前三平面全 SHA 與第四平面前 5999 bytes 的 SHA 必須匹配。第四平面最後 byte 不固定、不補猜。B-6／B-7／C-10 各自名稱保留，共用已確認相同 payload 的素材。
2. loader 自然返回先建立已知區來源資格。故事入口再次核對來源、固定 (0,40,150,80,0xA0FB4) 與目的，sub_11DCB 另外要求 loader 指標等於原始 alias。入口保存四個基址與完整當次 48000 索引 guard，包含當次實際未知八像素。只有自然 EIP／ESP+4 返回、基址及完整當次來源不變、完整目的等於該來源時才啟用 HD。guard 的未知值屬本次原版觀測，不是固定原圖解碼。
3. 桌面重新核對 47992 個已知原生索引及既有精確故事共用色盤。右下八像素直接使用目前原生 RGB，不解讀為素材色。任何已知來源／原生圖／色盤不符、未返回、錯誤參數、缺圖或幾何不符都維持原圖；原版記憶體、指令、VRAM、輸入、palette 與存檔不改。
4. 四項共用來源的條件式接入不推定各自一定走上述消費端。未出現在正常玩家路徑者保持 source/shared-consumer tests 分類；其他消費端仍屬全遊戲後續範圍。不能用 fixture 或 direct-entry 代替正常玩家路徑驗收。
5. 驗收包括兩組實際已知平面、四名稱、未知末 byte 的 256 種測試輸入、正確／錯誤消費參數、入口及返回的基址／來源／未知 guard／原生目的變動、錯誤色盤、缺圖、已知區 HD 公式與矩形外保護。未知 byte 的測試值明示 synthetic test input，絕不記為原版證據。另由同一建置自然完成普通歷史與首戰回歸，核對完整 RAM／十五欄與既有真實視窗。

READY 的完成範圍為四項美術及已知區的條件式故事消費端。原版最後 byte 仍未知，靜態解碼器狀態不改；正常四項場景、其他動畫及 PCX 不外推為完成。

## 2026-10-15：四項已知區故事素材與條件式接入，CONFORMED

適用範圍限前述 READY 的固定 EXE／DAT、三完整平面／第四前綴、兩個已證實故事消費端及目前八像素保留契約。四項場景沒有新增正常玩家路徑證據；本節不把 source/shared-consumer tests 改稱原版完整場景驗收。

- 兩組實際已知平面、四個資源名稱、兩個入口及 256 種未知末 byte 的測試輸入通過。沒有補 DAT 或選定一個「正確」末 byte。入口保存當次完整來源 guard 及四個基址；自然返回核對完整目的。即使來源被移至內容相同的新基址，或未知值與目的同步改變，仍按規格拒絕。
- 47992 個已知像素的 HD 取樣、八個原生未知像素、矩形外保護、錯誤已知索引、色盤、來源名稱、缺圖及來源副本不可變測試通過。sub_11DCB 的 alias 額外驗證保留；sub_1C1BC 使用既有 loader 指標，不推定其他消費端。
- 固定 GUI image 的四套件通過，研究入口 79.627 秒。二進位 SHA-256 `dae00e3e4fd91e217159078b95447eb1c952e46760f790c8f2a0c458f48fa268`，90 項來源在建置後凍結，兩次執行前已保存完整 tools 及授權快照。
- 普通歷史自然 600M，完整 RAM `c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2`；首戰 750M／390 普通事件，完整 RAM `4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9`。兩次十五欄與原版基準一致，首戰三張完整 X11 與前版相同。四項素材實際桌面載入皆為 ready；歷史的 700 次既有合格繪圖返回保持不變。這些流程沒有載入四項新場景。
- 準備與原圖／HD 視覺審查、四個來源 profile、完整素材 SHA、兩份來源快照及全部回歸收據由 remaining/partial-story/verification-summary.json 串接，總表 SHA-256 `8ef0940819f3280b1763eb7b266514c6a565876c7ab5f6e2511b886af06f2d52`。

素材與來源分類分開：覆蓋表目前 111 個 OZM 已有素材，加一項 VISUAL 分層，包含四項只替換已知區的故事圖。故事 profile 為 75 完整來源加四個部分來源，正常抽樣仍為十四個完整來源。靜態 decoder 的五項尾端未知維持原樣。其他動畫消費端、PCX、全遊戲 HD 與三平台打包不在此 CONFORMED 範圍。

## 2026-10-15：RAINEND 出版商標誌草稿，DRAFT

輸入為 DAT 的 RAINEND.PCX，檔案偏移 2830614、payload SHA-256 `78d0179bac1793eabb170925fd9b24378249c83c49320eaac27c8f70f65c1763`。離線參考 640×480、256 色，SHA-256 `8e5d4ff9a5b677ddeab48a33a9502b28941529625c10a7ca3c6c73dbf3c42be7`。這是解碼圖片，不是原版正常畫面或顯示模式證據。

內建 image_gen 依該參考重畫相同的彩色立體薄板標誌、rainbow 字樣及黑底星點。1448×1086 不透明 PNG 的 SHA-256 `d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93`，完整提示詞 SHA-256 `b7266d44f0c14b019842da63d8756bea3997d75eff29fa0a7c9e76c88f120cbe`。圖像已檢視，品牌字樣與原構圖保留；底部 (0,449)..(640,480) 規劃使用原版文字，生成圖該區的星點不列入呈現。PNG、完整提示詞及有序參考用途已保存私人 repo 素材清單。

目前原始 caller、載入路徑與正常顯示模式未知。沒有把圖片尺寸當成已支援 640×480 顯示，也沒有將草稿接入 production path。下一步從原始 IDA 證據或正常玩家路徑確認 PCX 消費端，再界定 READY；不為此追硬體逐週期細節。

## 2026-10-15：MARK 恢復範圍、啟動選擇與第一個平台阻塞

使用者已選恢復包內 MARK 標誌，再接 HD，並指定進入遊戲前先選原版／HD，頂端功能列也能動態切換。這些選擇已定案，不重新訪談。

### 原版證據

- 輸入 MARK.EXE.BAK，SHA-256 1ea6dc335b42b39542ba8dec35c8ffd103e4b495d30700c4100e60dbcbecaa60，MZ 內嵌 LE 位於檔案偏移 0x2998。版本來源未知，不稱為出廠原版。
- IDA Pro 9.4 的 LE 線性 EA 與檔案偏移各自記錄於 `workplace/hd-prototype/catalog/remaining/consumers/ida/mark/`。sub_101E2 在 EA 0x101F3 開啟 DX5.DAT，在 EA 0x1021A 取 RAINEND.PCX，在 EA 0x10260 呼叫 sub_183BA 繪圖。這些關係已證實。實際顯示模式仍以原版執行為準。
- 恢復前的 DX5 原版嘗試執行 .\\COMMAND.COM，命令尾為 /c MARK.EXE > nul。兩個檔案在來源包內皆不存在。既有缺檔 AX=2／CF=1 契約保留，不能假報 shell 成功。
- 研究工具 `tools/mark-probe-prepare.py` 只改隔離探針的輸入與環境名稱，拒絕全部 DX5 位址觀測與桌面模式。MARK 自然 LE 入口 2,005,974 步停在 dosgolem 線性位址 0x10044，bytes C0 AC 11 80 00 00 00 02，SHR byte ptr [ECX+EDX+80h],2。原版指令沒有略過。收據 `consumers/mark-probe/startup-probe/receipt-3m.json`，fd2_startup_calls=0。未到繪圖，不算已恢復。

### 記憶體 byte SHR 平台契約：READY

公開來源：[Intel SDM Volume 2，SAL/SAR/SHL/SHR](https://cdrdv2-public.intel.com/789581/325383-sdm-vol-2abcd.pdf)。平台語意依規格，不從 MARK 猜測。此次只擴充 C0 /5 ib 的記憶體形式，沿用 decodeAddress32 的 DS／SS 與 SIB 解析。其他未核准形狀維持停止。

計數取低五位；零計數不改資料或旗標。正計數右移並補零；SF／ZF／PF 依結果，CF 為最後移出位元。計數為 1 時 OF 為原值最高位；其他計數 OF 未定義，採既有保留政策。AF 未定義，採既有清零政策。計數不小於 8 的 CF 未定義，採迴圈末位政策。未定義旗標政策不聲稱實機一致。寫入失敗不得先發布新旗標。驗證真實 SIB bytes、SS 基址、截短指令、唯讀／越界與鄰接 byte 不變，再重跑原版 MARK。

### 桌面啟動選擇：READY

這是使用者指定的新桌面介面，無原版玩法語意。畫面只提供原版與 HD 兩個明確選項；滑鼠、左右鍵／Tab 選擇，Enter／Space 確認。預設顯示選擇畫面，不自動進入。選擇期間及確認輸入放開前，DOS 成功指令數、滑鼠事件、完整 RAM／原生畫面保持不變。選擇只設定桌面 enabled，和既有 setHD／功能列共用。缺圖沿用逐項備援。自動驗證可明示 -startup-display original／hd，正式互動預設 ask。不寫入 DOS 存檔。X11 驗證兩種選擇、選擇輸入隔離、正常進入及遊戲內功能列原版／HD／還原。

### MARK 的 DPMI 描述子權限：READY

右移修正後自然執行 5,380,886 步，原版 INT 10h/4F02 選 0101h，已證實 640×480、8-bit。隨後 sub_183BA 的 REP MOVSD 因目的 selector 0100h 不可寫停止。該 selector 由原版 AX=0000 配置，AX=0008 設 limit=0，AX=0007 設 base=000A0000h。實際 width 參數為 640，內部 SHR 2 得 160 dwords。

既有平台把新描述子標為不可寫，這項假設與成熟 DPMI 實作不符。[DPMI 0000 規格](https://www.delorie.com/djgpp/doc/dpmi/api/310000.html)指定 data、present、base／limit=0；[dosemu2 原始實作](https://github.com/dosemu2/dosemu2/blob/master/src/dosext/dpmi/dpmi.c) AllocateDescriptors 的 SetSelector 使用 readonly=0。原始程式另把 limit=0 用作 SVGA 圖形 selector 的平坦界限，需再查明其擴充契約；不能把零界限靜默放寬為 4 GB。先修正已證實的可寫性，重跑確認下一阻塞。正式通用配置保持 base／limit=0，測試不再把不可寫當作防護；零界限的多 byte 存取仍必須拒絕。AX=0009 明確設唯讀後仍拒絕寫入，不回寫外部 dosgolem。

勘誤：前段把中斷暫存器陣列順序讀錯。R 的順序為 EAX、ECX、EDX、EBX。IDA EA 0x180A6 的 B9 01 00 00 00 明確將 ECX 設 1，AX=0008 的實際 limit 為 0x10000，並非零。新描述子的初始 limit 仍為零；原版隨後設為 65536，SVGA 跨頁的保護例外處理見 sub_1801E。下一切片依實際執行判定是否需要例外交付。

### MARK 跨頁例外的後續入口

原版恢復可寫描述子後，自然 5,381,498 步停在 dosgolem 線性 0x183F3 的 REP MOVSD。此前已寫入第一頁 VRAM；當下仍在黑色 DAC 階段，不能把黑圖當成標誌完成。來源平面與繪圖尚未自然返回，HD 接入仍 DRAFT。

IDA 9.4 `consumers/ida/mark/range-17f80-1810b.json` 證實 sub_18086 把 selector 的 base 設 A0000h、limit 設 10000h，並以 DOS AH=25h 安裝 interrupt 0Dh 至 sub_1801E。該 handler 依 ES／DS 選擇分頁，取 EDI／ESI 的高 16 位，呼叫原版分頁函式，再把偏移 AND FFFFh，以 IRET 返回。這是原版的 VESA 跨頁路徑；不放寬 descriptor 以繞過它。

[Open Watcom C/C++ Programmer's Guide，DOS/4GW 中斷及例外](https://open-watcom.github.io/open-watcom-v2-wikidocs/cpguide.html)確認以 INT 21h/25h 安裝的 handler 同時接收 interrupts 和 exceptions，DOS/4GW 主機會轉換兩者堆疊差異。後續只需通用段界限失敗的結構化回報、這個已公開的主機轉接與同權限 IRET，未知 opcode／無註冊 handler 仍停止。規格與實作尚待完成，不以錯誤文字或 MARK 位址硬接跳躍。

公開平台資料已下載至 `consumers/mark-probe/`，入口 `platform-references.json`：
dosemu2 固定 commit d81f0f04804225a0fcedf49cf64b8611f8f01e6d、dpmi.c SHA-256 c98a3b783125ad9bdd86e744076638d9290da31c4840794b5be1ee44c9e2a878；
Open Watcom 文件 SHA-256 d6268a44630bfbcc99aac94ba8dbbe09ec71aabd12c24689b5d7d6795793f37a；
DPMI 0000 文件 SHA-256 10ffc6d4b13495ce140781f4c13612750905fbacbd7e8fa2507dba3593a183ca。
GitHub API 的 master 查詢先回 422；查 repository 的 default_branch 後改取實際 branch 並固定 commit。這是來源下載問題，沒有改工具 image 或退回主機執行。

### 動畫消費端的窄範圍定位

以下為 IDA LE 線性 EA，原始 bytes／檔案偏移分別保存在 `consumers/ida/`。輸入 DX5.EXE SHA-256 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a，IDA Pro 9.4。未把這些定位當成已接 HD。

- 已證實：sub_1BF44 唯一直接 caller 為 sub_1BF73；後者按索引讀取 off_35438。ANIME01.OZM 對應 table EA 35460，BATTLE.OZM 在 3544C／35604，JIMEN0／1 在 3563C／35640；KAO 在 35644。表尾界限未另確認，不推定整張表的完整長度。
- 已證實：sub_23EFE 在 EA 23F48 載入 index 5 的 BATTLE，EA 23F50 呼叫 sub_1C291。此函式四平面逐 dword 以 OR 合成 VRAM，不能沿用不透明複製端。
- 已證實：EA 23F5A／23F93 分別載入 129／130 的 JIMEN0／1，EA 23F86／23FC0 呼叫 sub_1C35B。參數 22 個 word、64 列、來源 offset 0 及目的 offset，來源寬 352。前段指令先 OR 四平面來源，再反相為清除遮罩；完整尾端與步幅仍須讀既有 range-1bf73-1c550.json，不先猜補。
- 強推論：sub_23415 在 EA 23477 動態選取動畫資源；索引運算需另追其資料表，沒有宣稱 48 格正常播放已驗收。
- 已證實：sub_11B60 的四個額外 caller EA 185F2／18628／1865E／18694 位於 sub_185A7。它們是額外消費端，所屬段落仍待核對，不能把該 helper 當成所有 ANIME 的通用繪圖路徑。

正式 DX5.i64 SHA-256 8afa40d3eaeed4851b78ed2ddc80ccbc1c65e14cb2481a09316d42670efce02e 未被寫入。查詢使用既有資料庫的隔離副本，MARK 則建立獨立資料庫。未改名或覆蓋位址。

### 啟動選擇及既有功能列：CONFORMED

此切片僅包含使用者指定的桌面選擇，不包含仍未接入的 MARK。固定 GUI image sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4，Go 1.24.13、Ebitengine 2.9.9。二進位 SHA-256 56f1a4897e40cbe30c42d47068f9bbfe0761e5084a7ccaf942fc3be85fd6c8ee，93 項建置來源及快照保存在 `workplace/desktop-prototype/startup-choice/`。

- 啟動時默認 ask；原版與 HD 兩種選擇各自然完成 200,000,000 步。確認前後完整 RAM、CPU、原生索引、色盤與游標 proof 相同；選擇期間及滑鼠／鍵盤確認手勢保持 0 步，沒有按下傳入 DOS。
- 遊戲中以普通 X11 功能列選原版／HD／還原，各次切換前後 proof 相同，還原畫面相同。兩次完整 RAM 均為 027128acfd96319ad8691b28e7cd26ffcaaadb37fb89e222c5d9bfab25c04e19。普通滑鼠移動在不同暫停邊界送入，故兩次輸入時間戳不同，不稱整條輸入序列完全相同。
- 額外首戰 GUI 自然 750M／390 事件，完整 RAM 4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9，十五個原版欄位與既有無頭基準完全相同。三張真實 X11 與呈現 PNG 相同。四套件通過。
- 第一輪驗證器錯誤要求整段流程沒有滑鼠移動，實際沒有按下外洩；保留失敗收據及腳本，在乾淨 original-r2／hd-r2 用同一二進位重跑通過。
- 第一張選擇截圖發生於首幀呈現前，黑色截圖保留；改由 X11 實際背景色與顏色數確認已呈現。獨立 startup-only 驗證保持 0 步、零滑鼠事件，真實選擇畫面 SHA-256 9ed46da494e302e76023f7f14aa4d8682b9e1305ecffa34b347cb5a8f0fa87ce，已目視確認。一般選擇驗證器也加入相同後置條件。
- 匯總入口 `tools/verify-startup-integration.py`，驗收總表 SHA-256 173ca02527bafa43cffcb3698b5db435bfe111f8c8bec7246ebf61897297f9de。它核對各收據、實際二進位、93 項建置雜湊、來源快照與原版基準。

原版資料完整性於 2026-10-15T05:20:40Z 通過。807 張生成 PNG、163 份提示詞保持通過保存核對，未新增原版圖或執行收據至 Git。MARK、其他動畫消費端及三平台封包尚未完成。

## 2026-10-15：MARK VESA 段界限例外轉接 READY

範圍為既有平坦 32-bit LE 執行器的段界限例外，以及 DOS/4GW 經 DOS AH=25h 登記的 protected-mode handler。輸入 MARK.EXE.BAK SHA-256 1ea6dc335b42b39542ba8dec35c8ffd103e4b495d30700c4100e60dbcbecaa60；來源證據為前節 IDA Pro 9.4 的 LE EA 1801E／18086／183F3 與 r3 自然入口收據。平台契約引用 [Open Watcom DOS/4GW Interrupt and Exception Handling](https://open-watcom.github.io/open-watcom-v2-wikidocs/cpguide.html)，本機固定原文／雜湊見 mark-probe/platform-references.json。IRET 的同權限 EIP、CS、EFLAGS 堆疊與 CMP 語意依 [Intel SDM Volume 2](https://cdrdv2-public.intel.com/789581/325383-sdm-vol-2abcd.pdf)。例外交付／堆疊屬平台 ABI，相對硬體時間仍為 hardware-spec approximation。

已證實的 MARK 路徑：VESA 窗口 selector base A0000h、limit 10000h。REP MOVSD 先完成窗內複製，在下一個 dword 因界限失敗；當次 ESI／EDI／ECX 保留已完成部分。原版 INT 0Dh handler 自行更換 VESA bank、遮掉高位並 IRET，原 REP 應從剩餘部分續行。此處不可重做整筆 REP、跳過 handler、依 MARK 位址硬接跳躍或把 selector 放大。

契約：
1. CPU 每次 Step 清空上一筆 LimitFault。僅已登記 descriptor 的正長度界限違反回報 selector、offset、size、write。唯讀拒絕、未知 descriptor、未知 opcode 或 bus 錯誤維持失敗，不假造界限例外。
2. 只有已啟用的 DOS/4GW 相容轉接與有效 AH=25h 向量 0Dh 才交付。code descriptor 必須 base=0、入口及返回 EIP 均在範圍，stack 須可寫且能完整放入 12 bytes。無 handler 或驗證失敗保持原停止。
3. 在原 SS:ESP 壓入 normalized IRET frame，ESP 起依序 EIP、CS、EFLAGS，沒有原始 CPU error code。EIP 是失敗指令含 prefix 的起點；其他暫存器保留 REP 已完成進度。交付時清 IF／TF，原版 handler 自行執行；交付本身不算成功指令。DOS/4GW 的 stack normalization 由主機提供，不宣稱原 extender 的完整內部 stack 相同。
4. IRET 僅支援平坦、同權限、32-bit 的三 dword frame。先完整讀取與驗證，再更新 EIP／CS／ESP／EFLAGS。VM／NT 與跨特權返回不支援；未知形狀停止。CS 覆寫的 16-bit CMP 依原版 bytes 擴充，來源與旗標按標準語意，不修改來源。
5. 單元驗證包含部分 REP 界限失敗、frame 內容、handler 真正執行並返回、缺向量／錯誤 selector／短 stack／唯讀拒絕／截短 IRET／未知形狀。原版驗證由 MARK 自然 LE 入口重跑，保存每次 fault 與真正原版 bank 呼叫，不用 direct-entry 當正常可玩收據。
6. 原版資料、規則與存檔不改。MARK 的原生繪圖、完整退出及桌面 HD 仍須另驗證，平台切片成功不稱整段恢復完成。

### MARK 色盤漸入的記憶體 ADD byte：READY

r4 自然入口已經過四次真實段界限例外，原版 INT 0Dh handler 自行選取 VESA bank，REP 續傳完成，回到原版色盤漸入。自然 5,384,299 步停在 dosgolem 線性 0x11AC3，bytes 00 84 2A F0 FC FF FF，ADD byte ptr [EDX+EBP-310h],AL。IDA LE EA 相同定位的原始指令已保存在 range-1018b-11b90.json。這只證實色盤程式消費此指令，不要求硬體逐週期調色盤考古。

依 Intel ADD 規格擴充 opcode 00 的 memory destination，以現有 decodeAddress32 選擇 DS／SS、現有 reg8 提供來源、既有 add8 計算旗標。沒有 prefix 的 memory 形式可用；未核准 register／prefix 形式仍拒絕。來源與目標有效才提交 byte 與旗標，寫入拒絕須恢復原旗標。差分測試用 256×256 組值，比較既有已核准 ADD AL,imm8 的旗標，並核對真實 SIB、SS 基址、來源暫存器、鄰近 byte、唯讀及越界拒絕。

勘誤：色盤 ADD 的 SIB 2Ah 使用 EDX 作 base、EBP 作 index，因此預設 DS。首輪測試誤把 EBP index 當成 base 並預期 SS，CPU 實際正確讀取 DS。保留 r5 的失敗測試及分類，測試改以原版 2Ah 核對 DS，另以明示 synthetic SIB 15h 核對 SS；沒有改 CPU 位址解析來迎合錯誤預期。

### VBE BIOS 必須建立標準掃描時序：READY

r6 的四次原版分頁、完整繪圖及來源／目的索引一致已通過。獨立 Pillow 9.4.0 從固定 DAT 的 RAINEND.PCX payload 解碼，97864 bytes、檔案偏移 2830614，307200 個色號 SHA-256 fb43b08b3202d43f0e502528683ea9ecc91a8b43778b680bd5c5c5a69228c5d4，與原版 MARK 的 RAM 及 VBE VRAM 完全相同。

但 r6 自然 25M 仍停在原版 EA 11634–11645 的 3DAh bit3 回掃等待。現有 VBE 4F02 只配置 bank／pixel geometry，沒有初始化 Legacy.VGA 的時序；時序模型使用尚未設定的暫存器，bit3 持續零。不是 PCX 解碼或分頁失敗。

沿用 pinned dosgolem internal/machine/crtc_timing.go 的標準 BIOS 時序表，新增僅設定 VBE 標準掃描時序的方法：0101h 使用既有 640×480 表，0100h 使用既有 640×400 掃描線表，八像素字元與 25.175 MHz。來源為已有 VGA BIOS 模式表及 [Linux v6.12 DRM 的 DMT 標準模式表](https://raw.githubusercontent.com/torvalds/linux/v6.12/drivers/gpu/drm/drm_edid.c)。保持既有 CRTC status approximation，不另反組譯 MARK driver、修改等待條件或按讀取次數硬翻 bit3。VBE bank pixel geometry、色盤與原版指令不由這個時序設定修改。未知模式不設時序。單元測試核對 59.94 Hz／70.09 Hz、bit3 有兩種狀態、保持色盤，再重跑原版自然路徑。單次讀取相位及 retrace width 沿用既有 hardware-spec approximation，不稱逐掃描線或實機一致。

### MARK 色盤漸出的記憶體 SUB byte：READY

r7 的 VBE 時序修正通過，原版 9,647,793 步自然完成色盤漸入，11,721,794 步離開停留迴圈；原生 640×480、256 色標誌已擷取並目視確認。停留虛擬時間為 2,074,001 微秒，沒有注入鍵盤或跳過等待。307200 色號仍和獨立 PCX 解碼一致。原生 PNG SHA-256 b2bc81720d7cbe057333d76a246051e8b6144ffb8ccfeb4ec7e4c923bb57fb4d。

下一條停止為 dosgolem 線性 0x11A15，bytes 28 84 2A F0 FC FF FF，SUB byte ptr [EDX+EBP-310h],AL。依 Intel SUB 的已知契約擴充既有 opcode 28 的 memory 形式；保留原 register 形式。沿用 decodeAddress32／sub8，拒絕寫入時恢復旗標，其他 prefix 仍拒絕。差分測試擴為 ADD／SUB 兩個 opcode、DS／SS 兩種 SIB、256×256 組 byte，參考既有 AL immediate 指令。接續只驗證原版漸出及正常退出，不深入 IRQ0／PIT driver；目前已取得的正常停留證據足以界定玩家流程。

### MARK 原生執行完成及 HD 桌面切片：READY

r8 自然入口於 15,989,208 步以 dos_exit／exit_code 0 結束，fd2_startup_calls=0。四次真實段界限、七次原版 VESA bank 呼叫、兩次 INT 0Dh 向量與 INT 08h 向量登記／還原、黑色漸出與模式 3 還原均已取得收據。CPU、machine 與新增平台／VBE 測試通過。這是獨立 MARK 原生執行完成，遊戲啟動器串接及 HD 尚待下列切片，不稱已恢復桌面完整啟動。

HD 規格：
- 明示 program=game／mark 的固定來源 profile。輸入檔名及 SHA 沿用前節，拒絕任意路徑及未知 profile。MARK 使用包內原始 .BAK，不更名或改原始資料，不宣稱其出廠版本來源。MARK 不安裝 DX5 位址觀測，game 預設及完整基準不變。
- MARK 繪圖資格為已證實的自然入口與返回、五參數、完整來源 hash、當次 RAM guard 及 640×480／256 色 VBE 目的索引共同符合。Source SHA fb43b08b3202d43f0e502528683ea9ecc91a8b43778b680bd5c5c5a69228c5d4。不是只認全畫面雜湊。
- 在原版 sub_11A50 入口只讀 EAX 所指 768-byte 的 6-bit 目標色盤。獨立 PCX 色盤右移二位後 SHA 778051183911d4dd5aebe339a3e2187ecde8c63e425a1314f83504f3b4a63f8a；實際 runtime 值需匹配才可 HD。保留當次目標色盤 RAM guard。
- 生成圖 assets/hd-b/art/RAINEND-hd-b-v1.png，SHA d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93，1448×1086，依現有素材風格用 nearest 採樣為 1280×960。下方原生 y=449..479 的完整版權區採當下原版像素兩倍呈現，圖像產物及版權字模不入 Git。
- 原圖／HD 與頂端列共用 enabled。來源、guard、目的或色盤不符合，或缺圖時完整備援。HD 漸入使用與原版目標色盤相符的 min(target,level) clamp，漸出使用 max(target-level,0)；未知色盤完整備援。這是呈現近似，原版 DAC／虛擬時間／RAM 不改。
- 桌面 layout 依 640×400 遊戲及 640×480 MARK 保持兩倍畫布，功能列增加同樣高度。啟動選擇先顯示，再步進 MARK。選擇及切換前後原版完整 proof 相同，缺圖、文字區及真實 X11 像素驗證後才稱接入完成。
- 本切片先驗證 program=mark 的實際桌面，後續啟動器在選擇後依序執行 MARK 與 DX5。兩程式必須保留各自原生收據，不假報缺少 COMMAND.COM 的 EXEC 成功。正式完整啟動、其他動畫及三平台打包仍未完成。

### MARK 原版與獨立 HD 桌面：CONFORMED

範圍是獨立 MARK 原生入口及桌面 HD；完整啟動器尚未串接。固定 GUI image sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4，Go 1.24.13／Ebitengine 2.9.9。輸入 MARK.EXE.BAK SHA-256 1ea6dc335b42b39542ba8dec35c8ffd103e4b495d30700c4100e60dbcbecaa60，版本來源未知。原版 DAT 與來源 PCX 的雜湊及位址基準沿用前節。

- 原版自然完成 15,989,208 步，退出碼 0，fd2_startup_calls=0。四次真實段界限交付原版 INT 0Dh handler，完成四個 bank 邊界與剩餘 REP 複製，沒有跳過 handler 或放寬 VRAM selector。原版漸入、2,074,001 微秒停留、漸出、向量還原及模式 3 退出通過。虛擬時間及 CRTC 仍屬 hardware-spec approximation。
- opcode 00／28 的 memory ADD／SUB 以 262144 組 byte、DS／SS 差分及拒絕測試通過。SIB 初輪測試預期錯誤保留，已按原始 bytes 訂正，不改位址解析迎合測試。
- 獨立 Pillow 9.4.0 對固定 DAT／PCX 解碼，307200 個色號與正常 MARK RAM、VBE VRAM 相同；PCX 6-bit 目標色盤 SHA 778051183911d4dd5aebe339a3e2187ecde8c63e425a1314f83504f3b4a63f8a 亦與原版 sub_11A50 的 EAX 來源相同。桌面同時核對來源與目標色盤 RAM guard、繪圖自然返回與原生目的。
- HD／缺圖兩次 X11 自然 MARK，各在亮起後用功能列及 F6 切換並恢復。六張實際視窗與呈現 PNG 完全相同，切換的完整原版 proof 相同。兩次完整 RAM 均為 62da54cc0cef42e9a55c64cb1ac958b05b851f19c12247023f6feddd023fa085，與原生 r8 一致；十三個非桌面輸入欄位亦一致。普通游標移動未作相同輸入序列聲明，全部沒有按下外洩。
- 獨立公式核對整張亮起原圖、HD 上方 1280×898 的生成圖 nearest floor 採樣，以及下方 1280×62 的原版版權。原圖／還原相同，缺圖三張完全回原圖。已目視查看原版標誌與真實 HD／頂端列畫面。
- HD 漸入／漸出色彩是呈現近似；未符合已知 palette shape 的狀態保留原圖。沒有修改原版 palette、RAM、時間或存檔。
- 同一二進位 SHA-256 4d66c2a7ce4493ff77b180a0fe5b411c902e9406a9a697934002ce7afe0acb16、102 項凍結來源，四套件通過，cmd/leprobe 115.021 秒。遊戲自然首戰 750M／390 事件、全部 RAM 4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9、十五個原版欄位符合既有基準，三張 X11 與呈現 PNG 相同。

匯總入口 tools/verify-publisher-integration.py，總表 workplace/desktop-prototype/publisher/verification-summary.json，
SHA-256 580a71ffc2fe8e23cda8c6122871b9f9b2f1c4c05f1c07b5c0c08fceb7e63e0b。
source-snapshot/ 保存當次全部 102 項來源；歷史核對明示 --source-root /output/source-snapshot，不以未來工作樹誤判舊收據。

目前獨立操作入口為 tools/run-desktop.sh -program mark -steps 75000000；仍在固定 Docker 執行。game 預設不自動執行 MARK，啟動器後續須先選模式、原生執行 MARK 再進 DX5，沿用最後顯示設定並保存各自收據。既有缺少 COMMAND.COM 的原版 EXEC 錯誤保持原樣。其他動畫消費端、完整啟動串接、全遊戲 HD 及三平台打包仍未完成。

## 2026-10-15：桌面選擇、MARK 與 DX5 串接契約

狀態：READY。使用者已確認恢復 MARK、MARK 也要 HD、進入遊戲前選擇原版／HD，以及頂端功能列可動態切換。本切片完成這條已確認的啟動路徑，不更動原版規則、存檔或素材包格式。

### 已證實前提與停止線

- 原版輸入沿用上節固定 MARK.EXE.BAK、DX5.EXE、DX5.DAT 與 CUE/BIN 的 SHA-256。MARK 獨立原版正常 15989208 步、退出碼 0；DX5 正常首戰 750M／390 事件及完整 RAM 基準均已保存。
- 固定 Ebitengine 2.9.9 的 /go/pkg/mod/github.com/hajimehoshi/ebiten/v2@v2.9.9/run.go 明示同一程序不得呼叫 RunGame 兩次。桌面啟動器先以同一二進位的 mark profile 執行 MARK，正常結束後再啟動目前程序的 game profile；每個程序各開一次遊戲迴圈。
- 這是桌面層恢復順序，兩個 LE 執行環境各自從虛擬時間零開始，分別保存完整收據與 RAM。沒有偽造 COMMAND.COM、DOS EXEC 成功或跨程序 DOS 記憶體延續；不宣稱原版 DOS shell 串接的硬體時鐘一致。
- 原版及 HD 標誌、來源資格與版權區沿用上一節 CONFORMED 契約，不新製圖片。

### 輸入、狀態與錯誤

| 狀態 | 輸入與輸出 |
|---|---|
| 啟動 | 桌面 game profile 預設 publisher-intro=true。先核對 DX5 來源，MARK 由子程序自行核對固定來源與光碟。無頭與明示 mark profile 不重複開啟前置流程 |
| 模式選擇 | 原有 startup-display=ask 在 MARK 視窗顯示一次。滑鼠、Tab／Enter、確認手勢隔離與原版 0 步保持原契約 |
| MARK | 75M 有界原版指令預算，正常自然退出；顯示模式及 hd-publisher-asset 旗標傳入。publisher-speed=0 沿用桌面速度，1..60 只供驗證加速 |
| 進入遊戲 | 只在 MARK program、來源雜湊、dos_exit、exit_code=0、fd2_startup_calls=0 與顯示設定皆有效時繼續。最後 hd_enabled 轉為遊戲的 startup-display，遊戲不再詢問；遊戲腳本輸入不轉送 MARK |
| 關窗 | 在選擇或 MARK 關窗時，以 desktop_closed 的子收據結束整個啟動器；game_started=false，DX5 不執行 |
| 失敗 | MARK 非正常退出、預算耗盡、來源錯誤、子程序失敗或不完整收據均停止，不默默略過。程序中斷會取消 MARK 子程序 |
| 收據 | MARK stdout 保存為帶 SHA-256 的本機收據，stderr 另存；MARK 狀態檔及 RAM 與遊戲分開。父收據附 publisher_prelude，保留子原文及最後模式 |
| 隔離驗證 | 既有只測 DX5 的桌面驗證器明示 publisher-intro=false；完整流程驗證器使用預設值，不以旁路作完成證據 |

### 驗收與垂直鏈

原版 EXE → 固定 program profile → 自然 LE 執行 → 模式選擇／頂端功能列 → MARK 正常退出 → DX5 正常玩家路徑。切換只作用桌面呈現，原版存檔及寫入副本契約不改。

1. 同一凍結二進位驗證兩條正常流程：先選原版再於 MARK 切 HD、先選 HD 再切回原版，證明進入遊戲沿用最後設定。MARK 亮起區與遊戲首戰皆擷取實際 X11、原圖／HD 與原版狀態 proof。
2. MARK 完整 RAM 及十三個原版欄位符合 r8；兩次遊戲 750M／390 普通事件的完整 RAM、十五欄及 PCM 符合既有 headless 基準。切換及還原的 proof 必須相同。
3. 模式選擇畫面正常 WM_DELETE_WINDOW 關閉，驗證 MARK 0 步、沒有選擇或按鈕洩漏、沒有 DX5 的 RAM／收據。
4. 完成四套件、來源快照、素材保存與 Docker 清理核對後，才將本契約標為 CONFORMED。其他十七項動畫消費端、完整遊戲可玩性、音訊人耳與三平台封包維持原待辦。

### 驗收追加：CONFORMED

上節完整桌面啟動鏈的 READY 契約已通過。唯一總表為 workplace/desktop-prototype/startup-chain/verification-summary.json，SHA-256 1ad1539a0bca5c4eabd0d2ff0dfef099af235ed74a8359f0afd2c207fa074a72。驗證工具為 tools/verify-startup-chain.py 與 tools/verify-startup-chain-integration.py，後者可使用 --source-root 指定當次凍結來源。

- 同一二進位 baafa065d2f8b204dae538371ee9de617b2f53717a14ec5d9c54d3086ca2d859、103 項凍結來源，四套件通過，研究入口 53.770 秒。
- 預設完整鏈通過兩次：先選原版、MARK 改 HD、遊戲沿用 HD；先選 HD、MARK 改原版、遊戲沿用原版。模式只詢問一次，MARK 選擇的 before／after 均為 0 步且 guest_unchanged=true；MARK 操作沒有按鈕轉送，遊戲只使用原有 390 筆普通時間表。
- 兩次 MARK 均自然 15989208 步、dos_exit、退出碼 0，完整 RAM 62da54cc0cef42e9a55c64cb1ac958b05b851f19c12247023f6feddd023fa085 與 r8 一致；十三個原版欄位一致。亮起區原版／HD 四張實際 X11 相符，下方版權區保持原版。
- 兩次 DX5 各自然 750M／390 事件，完整 RAM 4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9、十五欄及完整 CD PCM 與既有 headless 基準一致。首戰原版／HD／還原各三張實際 X11 相符，三次 proof 相同。MARK HD 及首戰 HD 已目視檢視。
- 模式選擇畫面以正常 WM_DELETE_WINDOW 關閉，MARK 為 0 步、沒有選擇或原版輸入，父收據 game_started=false，沒有 DX5 RAM。子收據原文、SHA-256、獨立 MARK RAM／狀態檔及遊戲收據均保留。
- 原始 ZIP CRC、十二個檔案及 UID/GID 於 2026-10-15T07:37:58Z 通過；807 PNG、163 提示詞及 596522181 indexed bytes 保存核對通過，沒有新增圖片。全根沒有 root-owned 或 .md 目錄，正式 DX5.i64 雜湊不變，本批容器全部已移除。

覆蓋盤點最初因十一份局部驗證器新增 publisher-intro=false 而拒絕舊工具雜湊，分類為研究台帳的工具版本核對問題。三份舊收據所記錄的驗證器，分別由已存在的 remaining/desktop-title/、graphics/static/desktop-logo/palette/ready/ 與 remaining/partial-story/desktop/source-snapshot/tools/ 取回完整快照；精確 SHA-256 仍嚴格核對。工具新增 verification_tool_sources 列出實際來源，沒有改動舊收據、歷史結果或原版基準。乾淨重跑覆蓋盤點通過，RAINEND 的 game_startup_integrated=true。

完成聲明限於本節的桌面啟動鏈、模式沿用及首戰抽樣。兩個原版 LE 執行環境及虛擬時間獨立，未宣稱 DOS shell EXEC、共享硬體時鐘、全遊戲可通關、剩餘動畫、音訊人耳或其他平台完成。

## 2026-10-15：戰鬥 OR 合成與透明矩形研究

狀態：DRAFT。範圍是原版正常戰場到交鋒畫面的素材使用路徑。先保存來源、繪圖前後索引及普通滑鼠操作；正式 HD 呈現仍待正常路徑證據及 READY 契約，不把研究探針接成正式替換。

### 已證實靜態證據

- 輸入 DX5.EXE SHA-256 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a。固定 IDA 9.4 image 4ac62de83339，從正式 DX5.i64 的獨立 animation-primitives.i64 查詢，不改正式資料庫。
- IDA LE 線性 EA 0x1C291..0x1C35A：四平面各讀取 8000 個 dword，OR 當前 VGA 目的後寫回。索引層的期望為 before | source；來源為 0 時保留背景。
- IDA LE 線性 EA 0x1C35B 起：四個 stack 參數依序為 width_words、rows、source_byte_offset、destination_byte_offset。每個 word 橫跨 16 像素，來源及目的每列 pitch 為 80 bytes。先以四來源平面 OR 的反相遮罩清除目的，再逐平面 OR，因此 source=0 保留背景，其他像素使用 source。
- caller 0x23F48 載入 BATTLE.OZM 後，0x23F50 呼叫 0x1C291。0x23F5A／0x23F93 載入 JIMEN_0／JIMEN_1，0x23F86／0x23FC0 呼叫 0x1C35B，width_words=22、rows=64、source_offset=0；目的偏移由原版計算，保留原欄位與未知語意。
- 0x23477 從原版表格選動畫來源後，0x23507 呼叫既有不透明矩形 0x1C1BC，width_bytes=26、rows=150、來源偏移來自表格。此分支無須推定兵種或以名稱取代來源指紋。
- 原始範圍：consumers/ida/range-1c200-1c700.json，SHA-256 a12561fab5fb2ba94b7d0611c177115d02c536da73746c075d0fee456594beaa；另有 range-23380-23a00.json 及既有 range-23efe-241de.json。全部保留原始名稱、EA、檔案偏移及載入後 bytes。
- 第一次交叉檢查錯把 IDA LE 載入後 bytes 當成檔案 bytes，因 relocation 不同而拒絕。改以固定 dosgolem LE loader 獨立核對，448 列 IDA 載入後指令 bytes 全部相同，32 列檔案差異另記 raw_file_bytes。收據 animation-loaded-byte-verification.json，不混用 EA、原始檔案偏移及 bytes。

### 唯讀研究探針契約

1. hd-composite-trace 預設關閉，只記 0x1C291／0x1C35B 的入口與自然返回，MARK 拒絕啟用。來源沿用完整四平面 SHA-256，不靠最近載入名稱猜測。
2. 原生 640×400 索引在入口及返回分開擷取，依已證實公式核對完整目的矩形。記錄原參數、來源矩形、目的矩形、來源與前後雜湊；未知來源、越界、錯誤回返或來源變動皆拒絕。
3. 索引資料只寫本機 composite-samples，最多 512 筆，檔名帶 SHA-256；不改 guest RAM、暫存器、VGA 狀態、原始資料或存檔。
4. 為研究正常首戰之後的路徑，有界指令上限擴至 2000000000。既有預設及 750M 基準不改；每次 Docker 工作另有 wall-clock 上限。
5. desktop-script-handoff 預設關閉，僅桌面驗證使用。390 筆原有普通滑鼠事件全部套用且末筆按鈕放開後，才交由真實 X11 普通滑鼠接續。保留完整輸入記錄，不注入狀態、座標到 RAM 或程式進入點。
6. 測試包括 RAM 不變、真正返回、來源變動、目的不符、越界及上限。正常路徑先做原版 UI 擷取與研究來源核對，收據不足前不列特殊消費端完成。

### 平台補充：D1 /4 register32，READY

普通 X11 在首戰選取單位，再選「攻擊」，於成功步數 754004409 停在 dosgolem 線性 0x1F43D，原始 bytes D1 E0；EAX=2408。原始 GUI 收據及 105 項凍結來源保存在 consumers/battle/，750M／390 事件的原版控制組已通過完整 RAM 及十五欄比較。這是先前首戰後未執行到的 CPU 形式缺口，不是美術或遊戲規則錯誤。

平台來源：[Intel SDM，SAL/SAR/SHL/SHR](https://cdrdv2-public.intel.com/835781/325462-sdm-vol-1-2abcd-3abcd-4.pdf)，Volume 2B 4-601。D1 /4 的 32 位暫存器形式為 SHL r32,1，CF 取原最高位，OF 為結果符號位 XOR CF；SF／ZF／PF 依結果，AF 未定義。本修正只增加 D1 E0..E7，保留既有 prefix、memory 與其他 group 的拒絕及錯誤界限；不修改 EXE、遊戲資料或原始程式控制流。

實作以 battle-cpu.patch 接在既有平台補丁之後。驗收採 Intel 邊界值、八個暫存器與既有 C1 /4 imm8=1 的獨立差分、其他暫存器及非運算旗標不變、非法 prefix／memory 形式拒絕，四套件與 750M 控制組重跑；再以相同普通操作自然進入攻擊分支。正式 HD 合成仍為 DRAFT。

### 追加勘誤：C1 count-one OF 與舊否定測試

r2 四套件中，CPU 測試拒絕新支援 D1 E0，且 D1／C1 差分在輸入 B177BEDA 得到 OF 不同。這兩個失敗均保留於 consumers/battle/r2/startup-probe/platform-test.log。查證既有 C1 32 位 shift 在 count=1 沒有設定 OF，不能把該錯誤當成 D1 的預期結果。

依 [Intel SDM 的 count-one 旗標契約](https://cdrdv2-public.intel.com/825743/325462-sdm-vol-1-2abcd-3abcd-4.pdf)，SHL 的 OF 為結果最高位 XOR CF，SHR 的 OF 為原最高位，SAR 清零。READY 契約補充修正現有 C1 32 位暫存器 count=1 的 SHL／SHR；count=0 保留所有旗標，其他 count 的既有未定義 OF 政策不改。舊否定測試將 D1 E0 改為仍未支援的 D1 C0，保留原拒絕範圍的覆蓋；新增支援另以正向標準邊界及差分驗證。r3/ 保存這次修正後的新凍結批次，不覆寫 r2。

### 平台補充：CMP word memory，READY

r3 四套件通過，同版 750M／390 事件控制組的完整 RAM 與十五欄符合既有基準。普通 X11 選攻擊已越過 D1 E0，於 752026383 步停在 dosgolem 線性 0x1F718，原始 bytes 66 39 07，暫存器 EAX=1、EDI=58790054。這是 CMP word [EDI],AX 的未支援形式；GUI 輸入的時間不同，停止步數不與首批作同狀態比較。

依 [Intel SDM Volume 2A 的 CMP 定義](https://cdrdv2-public.intel.com/812383/253666-sdm-vol-2a.pdf)，39 /r 比較 r/m16 與 r16，按 destination minus source 設 CF／OF／SF／ZF／AF／PF，不儲存減法結果。READY 範圍為既有 39 /r 的 operand16 記憶體形式，沿用 decodeAddress32、預設 DS／SS 與 readSegment16；暫存器、記憶體及非運算旗標保持，原有 REP、segment override 與 address-size 拒絕不擴張。加入標準旗標邊界、暫存器形式差分、唯讀記憶體與界限測試，再保存 r4 新來源批次重跑。不改原版 EXE、控制流或遊戲規則。

r4 的 4096 組 DS／SS 唯讀來源差分通過，標準邊界測試卻未建立 DS 描述子而被正確拒絕。此失敗屬測試初始條件缺漏，不修改 CPU 或記憶體界限。補齊唯讀 DS 描述子後以 r5 新快照重跑，r4 的原測試及失敗 log 保留。

### 平台補充：CMP word immediate，READY

r5 四套件通過，750M 控制組完整 RAM／十五欄及 MARK 自然 15989208 步的完整 RAM／十三欄皆符合既有基準。普通攻擊操作於 751041328 步停在 dosgolem 線性 0x1F4E8，bytes 66 81 3D 50 60 03 00 FF FF，為 CMP word [0x36050],0xFFFF。保留 r5 的原始操作與停止收據，不把與 r3 不同的 X11 時間序列稱為同狀態比較。

沿用上節 Intel CMP 契約，81 /7 iw 是 r/m16 與 imm16 比較，讀取立即值後設定相同六個運算旗標，來源不寫回。READY 範圍補 81 operand16 group 7 的暫存器及 decodeAddress32 記憶體形式；其他記憶體 group、prefix 與界限維持拒絕。以既有 39 /r 的等價運算、八個暫存器、DS／SS 唯讀 SIB、來源界限及拒絕形式驗證，保存 r6 新批次後重跑普通攻擊。

### 攻擊窄範圍的指令形式篩查及 AND AX：READY

r6 四套件、750M 控制組與 MARK 控制組通過，普通攻擊於 752041330 步停在 dosgolem 線性 0x1F4F9，bytes 66 25 FF 7F。依連續同類缺口的路由檢查，集中篩查已遇到的 IDA LE 線性 0x1F400..0x1F800，沒有擴大到無關 helper 或玩法語意。

IDA 9.4 在正式資料庫的獨立 battle-cpu-primitives.i64 匯出 363 列，range-1f400-1f800.json SHA-256 4e5c50c0bf523de41fbf23d827c439427057de0f4a504ca8852c6e9bc55b0bd0。tools/battle-instruction-probe.go 先由固定 dosgolem LE loader 核對全部載入後 bytes，再以合成 CPU 狀態單步篩查。r6 僅回報 66 25 FF 7F 的形式缺口。這是指令支援篩查，不是正常玩家路徑或語意對拍；不得用它替代普通攻擊驗收。原始範圍、獨立 DB 來源及 battle-instruction-capabilities.json 都留在 consumers/ida/。

依 [Intel SDM Volume 2A，AND，3-78..3-79](https://cdrdv2-public.intel.com/812383/253666-sdm-vol-2a.pdf)，25 iw 為 AND AX,imm16，保留 EAX 高 16 位，CF／OF 清零，SF／ZF／PF 依結果，AF 未定義。READY 範圍只補 operand16 累加器形式，其他暫存器不改，既有 32 位路徑不變；新形式的 REP／REPNZ／segment override 及截斷立即值拒絕。以 81 /4 word register 的等價運算、標準邊界及拒絕測試驗證，r7 再篩查一次同範圍後接續正常攻擊。

r7 同範圍篩查為 363 列載入後 bytes 相符、363 個合成單步及零錯誤；四套件亦通過。收據保存於 consumers/battle/r7/capabilities/battle-instruction-capabilities.json，明示 CPU 來源、工具來源及各自 SHA-256。此結果限於形狀篩查，正常攻擊仍另驗證。

重跑工具時沿用固定 GUI Docker、UID 1000、網路隔離及 Go 快取，唯讀掛專案、原版及固定 dosgolem；將指定批次 capabilities/ 的獨立副本掛 /output。該目錄的 battle-capability-overlay.json 只覆蓋已凍結的 internal 平台來源及 cmd/leprobe/main.go，執行 `go run -overlay /output/battle-capability-overlay.json ./cmd/leprobe -cpu-source /project/workplace/hd-prototype/catalog/remaining/consumers/battle/r7/engine/internal/cpu386/cpu.go`。歷史重跑先核對當次 source-snapshot/tools/battle-instruction-probe.go 的 SHA，再將該檔另行唯讀掛至容器 /project/tools/battle-instruction-probe.go，保持實際編譯與工具自查的路徑一致；不覆寫原有篩查收據。

### 平台補充：單次 SCASW，READY

原版存檔 I003.SAV 經正常標題、FILE 3 與確認讀回，140M／九筆普通輸入的完整原生索引及 RGBA 與存檔畫面一致。普通回合探針沿用此資料，十九筆輸入從原始 LE 入口執行；於 180761953 步停在 dosgolem 線性 0x1F244，原始 bytes 66 AF，AX=0x801E、EDI=58785368、ECX=63。收據保存於 r8/native-turn/。這是單次 SCASW，沒有 REP，不把它解釋成重複掃描。

依 [Intel SDM Volume 2B，SCAS，4-604..4-606](https://cdrdv2-public.intel.com/782151/253667-sdm-vol-2b.pdf)，以 AX 減 ES:EDI 的 16 位元值設定 OF／SF／ZF／AF／PF／CF，然後依 DF 將 EDI 加二或減二；不寫回來源，單次形式不改 ECX。READY 範圍只補 66 AF、32 位元位址及既有 ES 讀取契約。32 位元運算、REP、REPNZ、segment override 與 address-size 形式維持拒絕。以固定 ES、DS／ES 分離、雙 DF、CMP 差分、唯讀與界限、EDI 回繞及拒絕測試驗證後，重跑同一普通回合輸入。這是公開 CPU 規格，沒有重作遊戲 AI 或改 EXE。

### 原版 FILE 3 存讀檔抽樣：已證實

輸入 EXE 沿用本節固定 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a；下載 DAT 為 de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37。r8 在正常首戰後，以真實 X11 的原版「儲存進度」及確認寫入可寫工作副本，原始 original/game/ 保持唯讀。

- 寫後 DAT 為 3de67e0b56826584420d9dbbd70d9cceea94641694610cd89663f0fd979f97e0，大小仍為 3908861 bytes。檔頭 4233 bytes 不變；已盤點的 201 個資源中，只有 I003.SAV 改動 134 bytes，檔案偏移 0x2DD456、長度 25000。其他 200 個資源完整位元組一致，沒有重編圖像或推測存檔欄位。
- save-import/manifest.json 的 SHA-256 為 5f16734dfde5607b58ae0c9abd37003594406f7d53c7a64e8338e7bf0fe8f837，逐檔記錄全部十二個輸入。此輸入明示「原版程式資料加原版存檔」，不冒稱下載基準 DAT。檔案唯讀掛 /original，原版執行仍從真正 LE 入口開始。
- 普通讀回輸入為 80M 點標題「轉入進度」座標 (300,191)、90M 點 FILE 3 (220,176)、100M 點確認 (250,174)，各含移動、按下及放開，間隔 100000 步。load-check/r4/continue-path.json 有九筆，SHA-256 76c111b89409b64331ea15f252bb54a2d20cd264f8746eea5483a27cc04d0fb8。
- 原版自然到 140M，沒有未支援指令或跳過原版 runtime；完整原生索引 SHA-256 eac8871b05cc4a25ded2be4fb5f25baf0ef0f51f291b68fd11412d4f207cfb82，與存檔時相同。獨立 Pillow 9.4.0 比對全部 640×400 RGBA，包含游標，差異為零。收據 8e7d965e2117d8fa57a7a73aa595545694e901aaba0cb0d44484f91e347965ee，完整 RAM 840af476a537363b7009565ae12f875b2917329ae71ed65f458e909fa4740f4e 僅作本次可追溯資料，不稱存讀檔 RAM 相同。
- r9 沿用 r7 桌面二進位及 106 項來源，正常讀檔後開武將表；五張實際 X11 與呈現 PNG 完整一致，原版／HD 切換 guest_unchanged=true。收據 28479c69bf2d791d50c3071575704de35b8a786e3a308de0d7f501efcb11b26c，停止為正常 desktop_closed，成功步數 121599999。軍表文字保留原版，沒有推定單位陣營。

核對入口 tools/verify-original-save.py，固定既有 GUI image 及 Python 3.11.2／Pillow 9.4.0。歷史來源以 --source-root 指向 r8/source-snapshot，另以 --desktop 指向 r9；工具及完整提示以 AGENTS 的穩定路徑為準。本節只證實一個 FILE 3 樣本、畫面還原及武將表 HD 抽樣，不宣稱 SAV 格式已解碼、所有欄位、後續章節或其他平台通過。

原先 load-check/r2 把原生 640×400 圖的座標再除二，點到 (110,62)，沒有選中 FILE 3。這是研究輸入錯誤；r3 改正原生座標後到確認，r4 補正常確認後成功，全部原始收據保留。r9 桌面速度 1 的工具按下只經一次畫面更新，未開武將表；真實 X11 保持按下至少 100000 個原版步數後成功。互動工具新增 hold_steps，預設 100000，記錄實際原版輸入，不改遊戲滑鼠契約或猜測硬體時序。

### SCASW 正常回合追加

r10 四套件通過，cmd/leprobe 58.404 秒，106 項凍結來源，桌面二進位 d61c9d0e2c2cc85ea313744c7c457082e38a95019d50b3ebb44f14c6785ab173。相同十九筆原版讀檔及回合輸入先自然到 200M，再以獨立輸出重跑至 600M，兩次只因預算結束。600M 畫面日期由 1／20 到 2／20，正常來源新增 KOYO.OZM，沒有 OR 或透明矩形返回。

600M 收據 a43d9f2117a43591dbb372acd4d86706ee07d506746b95d109ba5cabf516ee3f，獨立核對 e38a062f3d8e3ec4ee944d911a579b1440098b5744da69e56e8ade2457c0c51b，入口 r10/native-turn-extended/verification-summary.json。verify-composite-probe.py --native-only 明示 native_execution=PASS、result=INCOMPLETE、HD_rendering_integrated=false；全部十九筆輸入及前綴邊界符合 SCASW 原停止批次，來源／資料／完整 RAM 雜湊均保存。這證實一次普通回合已可推進，沒有擴張為交鋒或全遊戲 HD 完成。

### 平台補充：83 /6 dword memory，READY

r10 四套件、下載基準 750M 完整 RAM／十五欄及 MARK 完整 RAM／十三欄通過。原版存檔及 49 筆連續回合輸入在 1327438966 步停止於 dosgolem 線性 0x15B58，原始 bytes 83 75 FC 01，EBP=378252。這是 XOR dword [EBP-4],1；原版已套用 39 筆普通滑鼠，完整原停止保存於 r10/native-turns/receipt.json，不修改後十筆未套用的排程。

依 [Intel 80386 Programmer's Reference Manual 1986，XOR，PDF 第 411 頁](https://read.seas.harvard.edu/~kohler/class/aosref/i386.pdf)，83 /6 ib 先將 imm8 符號擴展到 32 位元，與目的值 XOR 後寫回目的，CF／OF 清零，SF／ZF／PF 依結果，AF 未定義。READY 範圍只補 83 的 32 位元記憶體 group 6，沿用 decodeAddress32 的 DS／SS、來源及目的界限，REP／REPNZ／segment override／operand16 不擴張。失敗前不發布新旗標或寫入；既有暫存器、AND／OR 記憶體路徑保持契約。

驗收採全部 256 種 imm8 與多組 32 位元來源、DS／SS SIB 與負位移、既有暫存器 XOR 的差分、唯讀及跨界拒絕、目的外位元組與其他暫存器不變。完成四套件後以相同 49 筆普通輸入從原始 LE 入口重播；不靠合成單步宣稱正常回合或交鋒通過。

r11 四套件通過，研究入口 57.091 秒。IDA 9.4 窄範圍 0x15B00..0x15C00 的 86 列載入後 bytes 相符、合成單步零錯誤，來源 SHA-256 aa0919e2607d24cec35e44197fe9a2a072300bf0959bb63a7cbe7bd42b2c50a3。相同 49 筆輸入全部套用，原版於 1579516359 步遇到 C1 6D FC 02，保存 r11/native-turns/ 原停止；沒有交鋒 OR／透明返回。

### 平台補充：C1 /5 dword memory，READY

停止位置為 dosgolem 線性 0x22A06，原始 bytes C1 6D FC 02，EBP=378084，為 SHR dword [EBP-4],2。後續四個 stack 欄位也使用相同記憶體右移形式，這是標準 CPU 形式缺口，不推定遊戲欄位語意。

依 [Intel SDM Volume 2B，SAL/SAR/SHL/SHR，4-591..4-594](https://cdrdv2-public.intel.com/782151/253667-sdm-vol-2b.pdf)，C1 /5 以 imm8 的低五位右移 32 位元目的，CF 為最後移出的 bit，SF／ZF／PF 依結果；count=1 的 OF 為來源最高位，count=0 的資料及全部旗標保持，其他 count 的 OF 與非零 count 的 AF 未定義。READY 範圍只補 C1 /5 的 32 位元記憶體形式，沿用 decodeAddress32 與可寫目的的完整四 bytes 事前檢查；不新增 D1 記憶體、其他 group、operand16 或 prefix。

驗收採全部 256 種 count、多組來源與 DS／SS 定址、既有暫存器 SHR 差分、count 0／1／31 邊界、非運算旗標及來源外保護、唯讀／越界拒絕。未定義旗標不當成原版逐週期證據。重播仍只使用普通滑鼠及原版選單；第三回合另由正常「儲存進度」保存，若成功才作後續較短的原版輸入，不把 RAM 注入當讀檔。

r12 四套件通過，cmd/leprobe 50.817 秒，106 項來源保持凍結。IDA 9.4 的 0x22A00..0x22B00 匯出有 86 列，LE 載入後 bytes 全部相符、合成單步零錯誤。已存在的合成原語 0x1C200..0x1C700 共 448 列亦在同版 CPU 篩查零錯誤，原始 IDA JSON 不改，獨立輸出保存 r12/capabilities/composite/。這些僅為 CPU 形式支援診斷，正常多回合與 HD 呈現分開驗證。

r12 原版全部 49 筆普通輸入套用後，越過 SHR；於 1579516888 步停在 dosgolem 線性 0x20A81 的 14 00，AL=11，原始 trace 顯示下一條為 66 83 D3 00。IDA 9.4 的 0x20A00..0x20B00 共 115 列保留原始 EA／bytes，證實這兩個相鄰形式為 ADC AL,0 與 ADC BX,0，不推定整個 helper 的遊戲用途。

### 平台補充：ADC AL 及 word register imm8，READY

依 [Intel SDM Volume 2A，ADC，3-27..3-28](https://cdrdv2-public.intel.com/812383/253666-sdm-vol-2a.pdf)，ADC 計算目的加來源加執行前 CF，更新 OF／SF／ZF／AF／CF／PF。14 ib 為 8 位元 AL 立即值形式，保留 EAX 高 24 位；66 83 /2 register ib 將 imm8 符號擴展到 16 位元，保留目的暫存器高 16 位。READY 範圍只補這兩個形式，記憶體、額外 prefix 與其他 accumulator 形式保持原支援界限。

驗收採 AL 全部 256×256×雙 CF 組合及八個 word 暫存器／全部 imm8／雙 CF；期望旗標以較寬整數、有號範圍及低四位進位獨立計算，驗證高位、其他暫存器及拒絕形式。四套件後以第三回合的原版實際存檔、普通標題及回合輸入重跑，不注入 PC、暫存器或 RAM。

r13 的新運算測試全部通過，但既有 TestIntegrationADCSignExtendedImmediate 仍以 66 83 D7 00 作拒絕樣本，與本次 READY 範圍衝突。保留 r13 失敗，該樣本改為仍未支援的 66 83 17 00 word memory；不改 CPU 或放寬旗標期望。r14 重新凍結來源並乾淨重跑。

### 正常交鋒來源與 HD 合成契約：READY

r14 四套件通過，cmd/leprobe 43.395 秒；同版 750M／390 事件完整 RAM／十五欄及 MARK 自然 15989208 步完整 RAM／十三欄符合基準。第三回合原版存檔的十九筆普通讀檔／回合輸入自然到 600M，無未支援服務或跳過原版指令。收據 r14/native-turn3/receipt.json，SHA-256 91f8abba8873d785ee16ddd2c86fbbbe004743bb50cdfeafd3ceb6c582e65a03。

- 四次 BATTLE.OZM OR 返回、四次 JIMEN_0.OZM 與四次 JIMEN_1.OZM 透明返回，全部十二筆通過完整原生索引的獨立公式核對，沒有拒絕或丟棄事件。verify-composite-probe.py --native-only 明示來源通過、HD_rendering_integrated=false，原版來源證據與正式呈現分開。
- 首次 BATTLE 自然 186809142..186969340 步，來源非零 148320 像素，其中 18 個 OR 混色像素不等於來源色號。JIMEN_0 自然 187281947..187545969，參數 [22,64,0,24160]，目的 (0,302)..(352,366)；JIMEN_1 自然 187859683..188123705，參數 [22,64,0,24196]，目的 (288,302)..(640,366)。完整四平面與索引雜湊沿用既有來源契約。
- 正常不透明來源新增 BATTLE02、ANIME01／03／04／08 及 WEATHER2。動畫仍用既有 208×150 局部消費端，不增加推測動畫路由。
- r14/desktop-original/ready/ 的三張實際 X11 與呈現 PNG 一致，正常 desktop_closed，212M 步。已目視交鋒框架、數值、兩側姓名／等級及動畫階段。KAO.GRP 正常索引 22、0 位於 (40,66)..(120,146)、(520,66)..(600,146)，沿用既有頭像層；後續索引 0 的能力視窗另在 (128,286)..(208,366)。未推定其他武將姓名或兵種映射。

正式呈現的範圍與界限如下。這是既有桌面 HD 圖層的擴充，沒有新遊戲引擎、存檔格式、素材包格式或 guest 寫入。

1. 生產觀測只接受 0x1C291 的 BATTLE 與 0x1C35B 的 JIMEN_0／1，核對完整來源、原參數、真正返回、原生目的及完整公式；未知來源或失敗保持原版。診斷 IDX 寫出仍由顯式探針負責，正常呈現不寫研究檔。
2. 原版來源 0 的位置保留較早 HD 背景；非零位置撤銷較早圖層，無論新素材是否存在。新圖層只替換非零且原版公式結果等於來源的像素，18 個 OR 混色像素保留原版。將遮罩逐列合併為既有 VisibleRects，不改素材格式；矩形及圖層仍受既有 256／64 上限保護，超限完整備援。
3. 分別記錄來源裁切 SHA 與原生合成期望 SHA。呈現必須核對來源裁切，不把含背景的合成結果當成純來源；後續不透明繪圖、視窗、頭像及全畫面還原保持既有遮擋優先權，缺圖不得復活舊圖層。
4. BATTLE 的數值區 (176,52)..(464,148)、左姓名／等級區 (24,170)..(136,216)、右區 (504,170)..(616,216)，依正常畫面允許原版前景差異並逐點保留。其他位置不放寬來源核對。兩個頂端單位縮圖區 (240,56)..(288,112)、(400,56)..(448,112) 先保留整區原版，另列來源消費端待辦，不以背景差分推測其完整輪廓。
5. 單元測試核對真實來源遮罩、非零遮擋、混色保留、來源與返回錯誤、上限、缺圖、文字前景及原圖零變動。正常六億步完整 RAM／原版欄位應與本批原版相同；真實 X11 抽樣交鋒、動畫、原圖／HD／還原及個別缺圖。通過後只將已驗收的戰鬥合成切片標為 CONFORMED，頂端單位縮圖及其他未抽測來源另列。

### 共用視窗的 HD 矩形還原：READY

r15 的正常 GUI 在 201901325 步畫出 KAO 22 的 (128,286)..(208,366) 暫時頭像，207M 原版已還原地面，但 host KAO 記錄未撤銷，graphicsOcclude 因而留下原圖備援。200M 與 207M 的完整原生索引同為 52a4eef96949e3ae1e5885c54d94572aaec05efda3a552668465c406149181a4；四次模式切換的原版 proof 相同。這是呈現生命週期缺口，不修改原版還原或放寬現有來源守衛。

沿用已 CONFORMED 的 WINDOW 來源及保存／還原契約。IDA 9.4 LE 線性 0x1B165 的四個參數為 byte-column x、pixel y、cols、rows；0x1B187..0x1B195 計算 x+80*y，0x1B146..0x1B164 每列複製 cols 個 word，共 rows*16 列，四平面依序保存至 [ebx+0Ch]。unk_34ED3 的原始 16-byte 記錄保留：+0 為 VGA byte offset，+4／+8 為 cols／rows，+0Ch 為保存指標。0x1B245 依原版計數減一後選記錄並四次還原；原始定位見 catalog/range-1af00-1b165.json 與 range-1b165-1b300.json。動畫 caller 0x23426／0x23528 正常使用這一對例程，保存的 (216,100)..(424,260) 包含 208×150 動畫區。

- 在保存入口只讀核對 WINDOW 指紋、堆疊、64 層 host 上限及完整矩形。保存自然返回、計數增加一、原版記錄幾何與四平面 RAM 解碼皆符合入口原生索引後，才保留該區 graphics／KAO 的呈現快照。64 是 host 保護上限，不宣稱原版陣列容量。
- 還原入口先撤銷目前該區的呈現資格。只有相同原版 level／記錄／保存內容、真正返回、計數減一及目的原生索引完全符合保存內容，才恢復保存區域；區域外較新的繪圖保留。未知、損壞、超限或不完整還原留原版，不復活舊圖層。
- graphics 只合併同一原始繪圖記錄在區域內外的可見矩形；不改其原名、來源／目的、堆疊、返回、原參數或來源雜湊。KAO 撤銷與還原區相交的目前記錄，只恢復保存時完整位於該區的已知頭像；部分交疊未驗收時備援。
- 保存的索引及 host 快照不進存檔、素材包或 Git。原始 EXE、RAM、時鐘與輸入不寫入；診斷事件有界。測試及正常 GUI 必須核對區域外新繪圖、錯誤返回、來源／目的破壞、舊頭像撤銷及動畫後的 HD 還原。

### 戰鬥合成與共用視窗還原驗收：CONFORMED

2026-10-15 的 r17 四套件通過，研究入口 45.487 秒，106 項編譯來源保持凍結。獨立驗收入口為 tools/verify-hd-combat.py，完整結果為 workplace/hd-prototype/catalog/remaining/consumers/battle/r17/verification-summary.json。固定 GUI image、Python 3.11.2、Pillow 9.4.0；原版 DX5.EXE 為 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a，第三回合存檔 DAT 為 3bf1827150e9955dd169c0d464b85a11a9ce2ff3e0764a7ecda4cb7a3678e1b5。原版來源及位址基準沿用上述 IDA 9.4 LE 線性定位。

| 驗收 | 結果 |
|---|---|
| 正常十九筆讀檔／結束回合，600M | 完整 RAM 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d 及十七個原版欄位符合 r14，含 PCM、畫面、輸入與兩種繪圖 trace |
| 正常、動畫及個別缺圖三種 GUI | 十四張實際 X11 與呈現 PNG 相同，九次 F6 的原版狀態不變 |
| OR 與文字 | 十八個 OR 混色原生像素、3012 個黑色文字前景原生像素逐點保留 |
| 個別缺 JIMEN_1 | 82300 個呈現像素退回原版，區域外其他 HD 像素不變 |
| 動畫 ANIME08 | 207.4M 的原版／HD／還原一致，來源為騎馬人物；不以 208M 正在複製的另一格冒充穩定畫面 |
| 動畫結束及區域還原 | 208.8M 的原生 (216,100)..(424,260) 及 HD 同區符合動畫前背景，原版真正還原返回後不保留動畫資格；區域外新視窗仍保留 |
| 暫時頭像 | 視窗關閉後不再讓舊 KAO 22 遮住地面；區域快照、來源及目的破壞拒絕測試通過 |

本次 CONFORMED 限 BATTLE／JIMEN 合成、已抽測 ANIME08 與共用視窗還原。頂端兩個單位縮圖仍保留原版，其他動畫及結局未逐項正常 GUI 抽測，全遊戲 HD 未完成。

### 交鋒頂端單位縮圖：RE 與 READY

已證實，原版 EXE 與工具版本沿用上述 r17。正常十九筆輸入自然到 215M 的 r17/unit-diagnostic/receipt.json，在 188175792／188243775 步進入 IDA／dosgolem LE 線性 0x1F9D0，返回 0x1F9BE。來源 offset 28202／28178 對應 MAP_CHIP 的 units 7／3，目的為原生 (240,60)..(288,108) 與 (400,60)..(448,108)。r17 的來源唯讀，不注入 RAM 或 PC。

IDA 9.4 的獨立資料庫匯出 consumers/ida/range-1f940-1fc10.json，共 244 列，與正常 dosgolem 載入後 RAM bytes 全數相同，27 列原始檔案 relocation 差異另列於 top-unit-loaded-byte-verification.json。sub_1F96C 以原參數索引／13 分行、餘數乘六 byte 選來源；0x240C5／0x24188 分別傳入 byte-column 30／50、y=60。保留原始名稱、offset、位址與未知兵種語意。

正常 206.3M X11 原版畫面的兩個矩形全部符合「MAP_CHIP 來源非零取來源，零取 BATTLE 紙張」；兩個來源分別有 996／923 個非零原生像素。獨立結果見 r17/unit-diagnostic/top-unit-source-verification.json。初次選 207.4M 時左圖底部八列已被正常動畫覆寫，故完整矩形公式正確拒絕；改用動畫前原版畫面確認完整透明契約，不放寬遮擋檢查。

READY 範圍限上述兩個目的矩形及返回 0x1F9BE 的 units 0..12，沿用完整四平面指紋、無抖色 flags、真正返回及原版前後透明公式。十個非空來源重用既有 HD 圖，空白來源保持原版。桌面在背景及動畫之後呈現縮圖，逐點核對目前原生目的；動畫、視窗或未知覆蓋不符合完整公式時，該縮圖備援並保留已畫出的其他 HD。只有來源、色盤、原版背景、原版返回與既有 BATTLE 合成資格均符合時，才以既有 HD 紙張重建背景再合成 HD 單位。缺圖只影響該矩形，不改原版、資料格式或玩家操作。驗收包含正常兩圖、個別缺圖、動畫遮擋及結束還原、切換不變與 600M 原版控制組。

### 縮圖跨色盤生命週期的資格勘誤：READY

2026-10-16，r18 四套件通過，研究入口 45.974 秒，正常 600M 完整 RAM／十七欄仍符合 r14；四種 X11 保存完整收據。兩個縮圖的原生背景索引 SHA eccb511940aad109f04f089b1ddb4492c56939003b66e38c5664432bfb5167a0 與 BATTLE 紙張相同，原版透明返回已核對，但 188M 繪製時的來源 RGB SHA 與 206M 呈現時的固定來源 PNG 不同，導致 fallback_background_source。不同色盤階段的 RGB 不可作跨時點來源資格，r18 未列縮圖 HD 完成。

READY 修正只針對此縮圖：原版來源及自然返回新增完整 48×48 索引 SHA，驗證 BeforeIndexed 與來源索引非零覆蓋的公式，再核對目前原生目的索引。固定來源 PNG 的非透明色須能唯一對應已核准的目前色盤；只有索引／原始四平面／目的／原版返回及當前色盤均符合才呈現。保存時 RGB 僅保留診斷及資料完整性，不要求其與後續色盤相同。既有地面／物件的 RGB 契約維持，未放寬未知來源或硬體時序。

### 交鋒頂端單位縮圖驗收：CONFORMED

2026-10-16，r19 四套件通過，研究入口 59.603 秒，106 項編譯來源保持凍結。tools/verify-hd-combat.py --units 的獨立結果保存於 consumers/battle/r19/verification-summary.json。固定工具、原版 EXE、第三回合 DAT 及 LE 位址空間沿用本節前述來源。

- 正常十九筆讀檔／結束回合自然 600M，完整 RAM 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d 及十七個原版欄位符合 r14，原版畫面、輸入與 CD PCM 不變。
- 四模式十七張實際 X11 與呈現相同，十一次模式切換的完整原版 proof 不變。兩個縮圖來源索引、原版背景及原版返回索引吻合，9216＋9216 個呈現像素符合已核准紙張與單位 alpha 的獨立公式。
- 個別缺 units 7，左側 9216 呈現像素保留原版，其他 HD 不變；個別缺 JIMEN_1，82300 呈現像素備援，其他 HD 不變。跨個案比較要求完整原生可見畫面相同，各案切換另核對完整原版 proof，不將不同停止步數稱為完整 RAM 同狀態。
- ANIME08 的 (216,100)..(424,250) 與兩個縮圖下緣相交，兩圖在動畫時明確備援，動畫 HD 保持；真正還原後兩圖恢復 HD。保存區的原生及 HD 與動畫前相同，原版文字及十八個 OR 混色像素仍保留。
- 十個非空來源在兩個目的矩形走共同呈現、獨立 alpha、缺圖、錯誤來源／返回、色盤及未知覆蓋測試。正常 GUI 只抽測 7／3，不推定兵種、武將或其他場景映射。

地面缺圖第一個驗證程序以 143 結束，最後回報 50M，收據為空，原因未知；原狀及 termination.json 保留。相同凍結二進位於 ready/ 乾淨重跑成功，最後獨立驗收只採完整的新收據，不把第一次終止記為產品失敗或成功。全遊戲 HD 與三平台封包仍未完成。

### 剩餘美術消費端與 399 列結局：RE／READY

已證實，20 個來源在原版名稱表有 23 個索引，部分名稱重複。EXE 的資源 ID 與 DAT 目錄索引不同，不能互換；例如 MAP_WAKU 的 EXE ID=0、DAT index=121，END-11 為 103／87，END-11B 為 128／88，END-2 為 82／89，END-2B 為 102／90，OP6 為 121／128。依完整檔名對應，原始表格位址、指標、來源 RAM／收據與 IDA bytes 核對見 consumers/ida/remaining-consumer-audit.json。131 是查至正常 JIMEN_1 ID=130 的有界範圍，不宣稱原版陣列容量。

已證實，IDA 9.4 LE 線性 0x18D48／0x18DA2 分別載入 END-11／11B，再於 0x18D50／0x18DAA 呼叫 sub_1C084。0x1C084..0x1C10A 每平面複製 0x1F40 個 dword，即 32000 bytes、400 列。兩個已核對來源皆為 640×399，最後一列的來源內容未知；現有完整矩形要求 400 列而拒絕。239 列 caller 與 120 列原語證據已與正常 dosgolem RAM bytes 核對，原始檔案 relocation 差異另列，位址空間不混用。原版輸入 EXE、dosgolem 基準與工具版本沿用上述 audit。

READY 範圍只接受 END-11／11B 的精確四平面與 640×399 source，在原版 sub_1C084 的真正返回後核對目的前 399 列，再呈現該已知區。保留原始 raw_args 的 400 列，另以 opaque_full_known_rows 記錄資格範圍；第 400 列原版繼續執行及顯示，不猜補。其他短來源不放寬。真實來源測試須涵蓋兩張圖、真正返回、來源破壞、目前目的破壞、未知末列變化、缺圖、色盤及矩形外保護；正常結局未跑到前只列共用消費端測試，不稱正常 GUI 驗收。

已證實，交鋒縮圖的原生目的為 y=60..107，r19 的 y=56..59／108..111 共 768 個邊緣原生像素與 BATTLE 紙張來源完全相同。READY 允許將舊保留區縮至真正 48×48 目的；外側紙張回到既有 BATTLE 的完整來源及前景 guard，不猜測單位輪廓。以正常 X11 驗證邊緣、縮圖、缺圖、動畫與還原，不改素材或原版。

其他來源仍保持 DRAFT：END-2 經 sub_185A7 逐平面呼叫 sub_11B60，END-2B 已見共同矩形繪圖，MAP_WAKU 先用 sub_1C10B 打包四平面到 RAM，再另行呈現。其完整顯示與遮擋需沿實際 consumer 查證，不能把 RAM 打包當成 VGA 繪圖。來源與初步定位見 range-185a7-189c8.json、range-1e9e0-1ebc0.json；尚未作正常結局或地圖介面完成聲明。

### 399 列結局與紙張邊緣驗收：CONFORMED，按範圍區分

2026-10-16 的 r20 四套件通過，研究入口 58.924 秒，106 項來源及二進位凍結。verify-hd-combat.py --units --paper-margins 的完整結果為 consumers/battle/r20/verification-summary.json，另保留先前純呈現總表與精確驗證器。

- 紙張邊緣有 768 個原生像素、3072 個呈現像素逐點符合核准 HD 紙張。四模式十七張實際 X11、十一次切換、兩單位 alpha、單位與地面個別缺圖、文字／OR 混色及動畫遮擋／真正還原全部通過。
- 正常 600M 完整 RAM 與十七個原版欄位符合 r14，原版畫面、輸入及 CD PCM 不變。
- END-11／11B 的真實 640×399 來源、共同繪圖端及未知末列測試通過；合成狀態明確標為 synthetic，正常結局 GUI 未抽測。只有已知區域的接入契約為 CONFORMED，不稱正常結局或全遊戲美術完成。
- 208.7M 的初次還原取樣正遇到下一個共用視窗半繪狀態，完整原版比較區已還原；已核對 pending_draw 及原版差異只在下方視窗。ready/ 以真正還原事件加上 BATTLE02 資格等待穩定畫面，208.9M 的原版與 HD 比較區皆相同。沒有改原版、放寬像素 assert 或加生產呈現特例。

### MAP_WAKU 主框架快取至 VGA：RE 與 READY

已證實，來源與工具沿用 consumers/ida/waku-frame-source-verification.json：固定 DX5.EXE、IDA 9.4 LE 線性位址及 dosgolem a9714eb；四份窄範圍 bytes 與正常 r19 600M 的完整 RAM 相符，原始檔案 relocation 差異另列。名稱表 ID=0 為 MAP_WAKU，DAT index=121；完整來源為 384×120，索引 SHA 1d0ee5b9b52142beb6813ae52f1db215e5224fae1935a64ea83fe07b97fe9a6b。

0x1EA38 的 sub_1C10B 將前 384×88 打包至 dword_42188 指向的 RAM，四平面 offset 為 0、0x1080、0x2100、0x3180。正常 600M 的全部 33792 索引與原始來源前綴相同。0x1EB97 依序呼叫 sub_1FFF7，該 helper 每列複製 12 個 dword、共 88 列，目的每列跨 80 bytes。第四次的 helper 返回為 0x1EC31，來源 base+0x3180，目的 0xA0005，原生矩形 (40,0)..(424,88)。sub_2062B 先呼叫這段，再呼叫既有地面／單位消費端 sub_1FD92。原名與原參數保留，不推定其他介面名稱。

READY 限主框架前 88 列，不接下方四個 32×32 圖塊或原始 debug 字樣。只有指定 helper／第四平面返回、原始來源與目的參數、普通 RAM 界限、四個精確前綴 SHA、真正 ESP+4 返回、來源不變及完整原生目的索引相同時才給資格。記錄原 helper 的來源／目的與 cache base，幾何參數與原始參數分開。未知來源或失敗撤銷相交資格，不改 guest。

來源 index=8 是黑色預留區，共 15906 像素，後續 sub_1FD92 會在其中呈現場景。主框架呈現只替換非 8 的已核對來源像素；index=8 保留既有 host 畫布，包含已通過的 HD 地面、人物及原版文字，不以原生圖覆蓋它們。非 8 的目前原生內容必須仍等於來源；被其他介面覆寫時備援。色盤、缺圖、後畫遮擋、視窗還原、矩形上限及切換不變沿用既有契約。

驗收先以真實來源及正常 RAM 測來源、四平面順序、錯誤參數／返回、來源／目的破壞、未知色盤、缺圖及黑區既有 HD 保留，再由正常玩家路徑抽測框架。新增只讀 trace 記錄與原有記錄分開核對；完整 RAM、十五個原版欄位及舊 trace 不得改變。未取得正常 HD GUI 前不將此切片列為正常完成。

### MAP_WAKU 後畫地面遮擋：勘誤與 READY 補充

r21 四套件通過，普通九筆讀檔到 140M 的真實視窗中，快取框架已在 118648054..118648596 真正返回並核對完整 33792 索引。後續地面與單位會覆寫原始非黑色框架，共有 541 個非黑來源像素的 RGB 不同，全部位於正常已記錄地面／單位矩形內，沒有區域外差異。這是後畫場景遮擋，不能將全框來源不變當成穩定畫面的資格。原版索引 SHA 3a6b2843173644bdcc0822e18fe2f7dd874ae04be8e12543f9eaf9629f6bc0de 與既有第三回合讀檔相同；RGB 診斷只定位差異，不取代索引來源及真正返回證據。

READY 補充：地面自然返回記錄 EndStep；主框架只對 EndStep 較晚的已核對地面及單位矩形撤銷 host 呈現資格。採完整矩形保守遮擋，不猜透明輪廓，黑區及撤銷區均保留既有 HD 畫布；區域外非黑來源仍逐點核對。其他 OZM 不套地面特例。測試需同時涵蓋早／晚繪製、原生索引碰巧相同、未知矩形及缺圖，正常 GUI 獨立核對剩餘框架與保留的 HD 場景。

r21 無頭程序完成執行後在來源收據階段缺少 /output/engine/internal/cpu386/cpu.go，未產生完整 JSON。這是驗證目錄準備錯誤，r21 保留 stderr，不列原版或產品失敗。r22 先建立完整 engine 工作目錄再乾淨重跑。IDA 第一批漏掛 /original，未產生輸出；相同 image 補掛唯讀原版後重新匯出下方圖塊的原始指令，未改正式資料庫或 guest。

### MAP_WAKU 主框架驗收：CONFORMED，前 88 列

2026-10-16，r22 四套件通過，研究入口 102.014 秒，106 項來源保持凍結。固定 GUI image 中執行 tools/verify-hd-map-frame.py，結果為 consumers/battle/r22/verification-summary.json。

- 普通十九筆讀檔／結束回合自然 600M，完整 RAM SHA 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d、十五個原版欄位及原有合成 trace 相同。原有 67 筆 graphics trace 完全保留，另有 72 筆自然返回的 MAP_WAKU 快取主框架，不將新增唯讀記錄當成原版狀態變化。
- 普通九筆原版讀檔的兩個桌面案例，各有三張實際 X11 與兩次 F6 切換；窗口像素等於呈現檔，切換的完整原版 proof 不變，還原後與原圖相同。
- 75 個後畫地面／單位矩形依真正返回先後保留。68744 個框架呈現像素符合已核准美術；66424 個框內呈現像素保留原有畫布，其中 46087 個仍是 HD 場景。兩案皆保留 67 處 HD 地面，不以框架覆蓋地形、人物或原版文字。
- 個別缺 MAP_WAKU 素材時保留已獨立合格的 MAPGAMEN HD 底圖。68744 個該層像素以固定來源索引與 HD 底圖另行逐點核對；其他區域保持相同，共 68562 個呈現像素不同。這是圖層缺圖備援，沒有撤銷另一個已合格素材。前一版驗證器誤假設所有缺圖像素必須等於原生畫面，失敗版本與理由保留，不改生產程式迎合該假設。
- 第一版驗證器未處理空的 UI 清單，修正 null 清單讀取後沿同一完整收據重查；兩次驗證器失敗皆不列產品缺陷。最終驗證器全文與 SHA 保存在 r22 根目錄。

主框架為 CONFORMED。下方八次 32×32 快取繪製、16×16 字樣、END-2 位移繪圖及未抽測正常結局另列，不宣稱全遊戲 HD 或三平台封包完成。下方 reader 的原始指令另保存 consumers/ida/range-1ec31-1f050.json；其中第一種圖塊的平面 1 來源為 base+0x4080，其他平面為 +0x4200／+0x4300／+0x4380，保留原值，不推定原因或修 guest。

### MAP_WAKU 下方八次快取：RE／READY

已證實，tools/verify-hd-map-cache-source.py 的 consumers/battle/r23/map-cache-source-verification.json 核對 r22 正常 600M 全 RAM SHA 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d、固定 DX5.EXE 與完整 MAP_WAKU 索引。IDA 9.4 LE 線性 0x1EC31..0x1F050 的 332 列 caller 及 0x1FFF7..0x20090 的 60 列原語 bytes 皆等於正常載入 RAM。主框架四平面及下方 0x800 bytes 與靜態來源獨立編碼相同，後者 SHA a1c858f9c927dc56b81277328d84b2ffb00eb280ed91d6162c7bde374b608b9d。

已證實，sub_20020 每平面複製 4 bytes×32 列、VGA pitch=80。八個第四平面真正返回候選為 0x1ECB4／0x1ED37／0x1EDBA／0x1EE3D／0x1EEC0／0x1EF43／0x1EFC6／0x1F049。左、右目的 x=40／392，y=120／184／248／312。靜態美術來源均在 y=88..119，四種 x=0／32／64／96，呼叫次序為 0、0、0、2、1、1、1、3。

左側前三次原版平面偏移為 0x4200／0x4080／0x4300／0x4380。其平面 1 來自主框架平面 3 的 packed offset 0xF00，獨立索引公式為 static_tile 的 bit 1 改用完整來源索引 80×384+i 的 bit 3。實際索引與靜態圖塊有 455 處不同，SHA ef5a5bf4ba41152b313fa25933141483b2082ff820942934a420537b2d43dfd7。其他三種圖塊的四平面順序與靜態索引完全相同。這是已觀測的讀取變體，不推定偏移原因，不改 EXE、cache、VGA 或存檔。

READY 限以上八個 caller／固定目的／四平面及真正 ESP+4 返回。完整主框架及下方快取 SHA 皆必須相符，來源與 base 在返回前不變，實際原生目的須等於實際 reader 索引。記錄原 helper 來源／目的、cache base、靜態美術裁切 SHA、實際 reader SHA 與完整素材指紋，不能將兩種索引混為同一來源。

host 呈現沿核准的對應 32×32 美術裁切，實際 reader 或靜態素材的 index=8 均保留既有畫布，不將素材黑色緩衝區蓋成場景黑塊；其餘像素核對實際 reader，依真正較晚地面／單位、視窗及頭像矩形保守撤銷資格。缺圖沿 r22 已驗證的底圖備援。原版平面差異不影響遊戲狀態或原版顯示。正常八次繪製、缺圖、雙向切換及底圖保留需由實際 X11 驗收，16×16 字樣仍不接入。

有界唯讀 trace 的一般來源仍限 512 筆；快取來源另限 1024 筆，合計最多 1536 筆。分開配額避免新增八次觀測排擠原有一般 trace；不增加未界定的記憶體或觀測。測試涵蓋四種來源、八個目的、錯誤 caller／來源／目的／返回、缺圖、未知色盤、來源或原生目的破壞、黑區及區域外保護、後畫遮擋與配額。完整原版 RAM、十五個原版欄位及既有一般／合成 trace 維持，新增快取記錄單獨核對。

### MAP_WAKU 八處下方圖塊驗收：CONFORMED

2026-10-16，r23 四套件通過，研究入口 113.317 秒，106 項來源凍結。固定 GUI image 中執行 tools/verify-hd-map-frame.py --corners，結果為 consumers/battle/r23/verification-summary.json。驗證器及來源核對器全文、準備前覆蓋清單另保存在該目錄。

- 普通十九筆讀檔／結束回合自然 600M，完整 RAM SHA 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d、十五個原版欄位與原有合成 trace 相同。一般 graphics trace 的 67 筆不變，另有 72 次主框架及 576 次下方圖塊自然返回；八個 caller 各 72 次，來源、原參數、靜態美術裁切與實際 reader 索引分別核對，沒有事件被配額排擠。
- 正常原版讀檔與個別缺 MAP_WAKU 兩案，皆以九筆普通標題／FILE 3／確認輸入到 140.1M。六張實際 X11 與呈現檔完全相符，四次切換的完整原版 proof 不變，還原後原生畫面相同。
- 主框架 68744 個呈現像素保持 r22 結果。八處下方圖塊分別新增 1072／2732／2732／2752／2444／1072／2812／2936 個 HD 呈現像素，共 18552 個；總框架 87296 個逐點符合核准美術。後畫地形／單位及黑色保留區保持既有畫布，共保留 80640 個呈現像素，其中 58600 個仍為 HD。兩案皆保持 67 處 HD 地面。
- 缺圖時 85712 個呈現像素符合獨立資格的 MAPGAMEN HD 底圖，其他該層像素符合原生畫面；其他區域與正常 HD 完全相同。兩案有 87111 個呈現像素不同，黑區及原版平面偏移造成的保留內容不被新圖清除。
- 四種真實來源、八個矩形、兩種黑區、後畫且索引碰巧相同、缺圖、錯誤參數／返回、來源與目前目的破壞、未知色盤、區域外保護及一般／快取獨立配額測試通過。測試中的合成 CPU／目的狀態不當正常執行證據；正常證據由上述自然返回及 X11 提供。

此切片為 CONFORMED。16×16 字樣保留原版，END-2 位移消費端及未抽測正常結局仍未完成；三平台打包保持待辦。既有生成圖與提示詞沒有新增或改寫，原版資料、索引、RAM、音訊及視窗收據只留本機。


### END-2 位移來源與平台缺口：RE → DRAFT → READY

2026-10-16。輸入 DX5.EXE SHA-256 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a，IDA Pro 9.4，位址為 LE loader linear EA；檔案偏移另見 consumers/ida/range-184d0-185a7.json、range-185a7-189c8.json、range-11b60-11ba6.json。正式 DX5.i64 SHA-256 8afa40d3eaeed4851b78ed2ddc80ccbc1c65e14cb2481a09316d42670efce02e 未改。這是來源與局部指令研究，正常結局尚未走到。

已證實 caller 0x187E5 載入 executable ID 82，檔名 END-2.OZM 對應 DAT 89。0x187ED 初次呼叫 sub_185A7，之後以 0x184D0 位移、0x185A7 顯示，迴圈 240 次。wrapper 四次呼叫 sub_11B60，以 150 列、50 bytes、來源及目的 stride 80、目的 A000F 複製四平面，顯示矩形 (120,0)..(520,150)。0x184E6..0x185A2 的本體在四個 loader pointer 各執行 6000 次 word 讀寫，先 byte swap 再 SHL AX,1／RCL BX,1，寫回各前 12000 bytes；每平面另讀尾端後兩 byte。DRAFT 公式為前 96000 索引向左一位，後方不動；連續 n 次時 i<96000 取原來源 min(i+n,96000)。顯示裁切在 n≤240 時只用原版前 150 列內的已知像素。公式須由原始指令探針核對後才授予呈現資格。

首次 r24 來源探針停在 0x18506、bytes 66 D1 D3，CPU 明確拒絕 word RCL；不是遊戲結局驗收失敗。保留 r24 原批次，修正後輸出另放 r24/ready/。平台 READY 僅補 66 D1 /2、mod=3 的 16-bit register RCL by one。依 Intel SDM Volume 2B 的 RCL/RCR 契約，結果為 word 左移一位並帶入舊 CF，CF 取舊 bit15，OF 取新 bit15 XOR 新 CF；SF/ZF/AF/PF 與其他 flags、暫存器高 16 位保留。記憶體、C1 immediate 及未授權 prefix 繼續拒絕。來源：https://www.intel.com/content/dam/www/public/us/en/documents/manuals/64-ia-32-architectures-software-developer-vol-2b-manual.pdf 。差分以獨立 17-bit ring 與全暫存器／旗標驗證，不以遊戲美術推導 CPU 行為。

### END-2 呈現契約：READY

同日 `tools/ending-source-probe.go` 使用修正後 cpu386，核對三份 IDA 匯出的全部 loaded bytes，再在明示合成 RAM 中執行原始本體 57,606,720 條指令。241 個連續狀態皆符合上節獨立公式，11,520,000 次寫入全在四個前 12000-byte 範圍，後方保持不變。收據為 `consumers/battle/r24/ready/ending-source-verification.json`，四組實際 CPU 平面保存 count 0、1、120、240。這證實局部原始指令與來源公式，不證實正常結局或 GUI。

呈現 READY 限固定 END-2 完整來源 SHA b2aa5040cba120939881b9c1ae7bafa1de492aece744bce745df66c3f83b7682、640×400、四個 loader pointer，wrapper 0x185A7 與 caller 返回 0x187F2／0x18822。初始只接受完整四平面指紋；後續須同一四個 base、全來源符合恰好下一次位移，最多 240 次。不搜尋可匹配位移、不因載入名稱而放寬來源。返回時 ESP+4、四平面完整 bytes 與指標不變，實際目的 (120,0)..(520,150) 必須符合當次 400×150 索引。host HD 從既有 END-2 v3 美術裁切 (count,0)..(count+400,150)，保留色盤資格、後畫遮擋、矩形外畫布與缺圖備援。來源、返回、目的、色盤或連續性未知時退回原版；不改 guest、VGA、EXE、DAT、遊戲規則或原版計數器。共同消費端測試與原版 600M 回歸完成後，只記為條件式接入；正常結局 X11 待驗收。


### END-2 條件式接入驗收

2026-10-16。核對入口為 tools/verify-hd-ending.py，結果位於 consumers/battle/r24/runtime/verification-summary.json。原版結局 GUI 尚未走到；本節的通過限原始指令、共同呈現與普通戰場回歸。

- 三份 IDA 9.4 匯出共 447 rows 與 dosgolem loaded bytes 相符；原始函式名、線性 EA、檔案偏移及 operands 保留，沒有改名。
- RCL 的獨立 17-bit ring、八個暫存器、BX 全 65536 值與兩種 CF、高 16 位、其他旗標、拒絕形式測試通過。四套件通過，研究入口 152.238 秒，106 項編譯輸入凍結。
- 原始位移指令的 241 個狀態與共同消費端相符。count 0、1、120、240 的四組實際 CPU 平面，逐點驗證 400×150 HD 裁切；連續性、跳步、上限、來源、指標、幾何、返回 stack、目前目的、色盤、原圖、缺圖、矩形外與後畫視窗皆有測試。合成目的與返回測試不冒稱正常結局。
- 普通十九筆讀檔／回合輸入自然 600M。完整 RAM SHA 28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d，十五個原版欄位與一般／合成兩組 trace 共十七欄完全等於 r23。715 筆 graphics 返回包含 72 主框架、576 下方快取及 67 一般繪圖，沒有加入結局繪圖的正常路徑假證據。
- 九筆普通標題／FILE 3／確認輸入的實際視窗，在 140.3M 正常關閉。三張 X11 與各自呈現 PNG 完全相同，亦等於 r23 的同標籤完整畫面；兩次切換與三次擷取期間原版 proof 不變。
- 唯一覆蓋清單新增 END-2 的 241 個條件式狀態，不增加正常結局數。原有 111 項 OZM 素材、五項靜態解碼未知與八項剩餘正常 HD 來源保持原分類。既有 807 張生成 PNG 與 163 份提示詞不變。

首次來源探針的 word RCL 缺口保留在 r24/source-probe-failure.json。ready/ 保存來源證據與第一版測試失敗，原因是合成原生畫布的未繪區 alpha=0，prepareHDImage 正確拒絕；修正 fixture 為完整不透明並檢查錯誤後，另在 runtime/ 乾淨重建。工具準備另遇 Python image 缺 git 與對 gofmt 後來源反向套 patch 失敗，改沿既有 GUI image 與固定 dosgolem／前三份補丁重建基準；這些是環境與工具操作問題，沒有放寬產品條件或回寫舊證據。

來源與共同消費端的條件式接入完成；正常結局仍未驗收，該正常 GUI 閘門不標 CONFORMED。不開始三平台打包，不把原版資料、平面、索引、畫面、RAM 或音訊加入 Git。

### 結局驗收授權調整與受控入口：DRAFT

2026-10-16，使用者明示「正常結局允許修改驗收，繼續囉」。結局 HD 的呈現閘門可改用受控原版結局，包含實際 X11、動畫、原版／HD／還原及缺圖；完整通關不再是這個呈現切片的前置條件。所有受控收據明示入口與狀態差異，正常通關仍為未知，不將舊收據重新命名成正常結局。

- 輸入 DX5.EXE SHA 2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a；IDA Pro 9.4，位址為 LE loader linear EA。原始定位保留於 consumers/ida/range-10140-101c0.json、range-185a7-189c8.json。
- 已證實 0x10192／0x1019B 比較 dword_342B8 與 10／11，成立時到 0x101A4，bytes E8 FA 84 00 00，呼叫 sub_186A3。該函式依序呼叫 sub_1872D、sub_187C3、sub_189C8、sub_18D26，最後原版 exit(0)。這足以選定結局呼叫端；不由測試狀態推定正式遊戲勝利條件。
- 可撤回 prototype 使用已完成原版初始化的固定指令邊界，記錄全部 RAM／CPU 狀態後，將 EIP 轉至 0x101A4，讓原始 CALL 與原版結局自然執行。若測試需設定 10／11，只寫隔離 RAM 的原始 dword_342B8，記錄前後 bytes；不修改磁碟 EXE、DAT、原版存檔或正式啟動器。
- 測試版由原版來源快照及明示的局部來源轉換產生，獨立編譯，不加入正式啟動路徑。收據以實際測試來源的 SHA 為準，標示 validation_only 與非正常通關；各模式從同一固定入口狀態執行，切換時 RAM／CPU／原生圖保持不變。
- DRAFT prototype 先查明初始化時點、執行所需狀態與原版自然停止。來源、目的、色盤或平台能力缺口仍依原版 bytes 查證，不以放寬 HD guard 讓畫面通過。prototype 足以執行並比對後，再將受控呈現驗收契約標 READY。

### 受控結局呈現驗收：READY

- r25/control/ 從固定 DX5.EXE 與下載基準 DAT 自然執行 40M，停在 OP5 標題。完整 RAM 保存並核對；原生索引 SHA 39a6813ba03cf0517b43d9d04536ff0dbd264f556339959ebca376cb57ff61c8。此固定初始化狀態是受控入口的必要條件。
- 初版受控探針在同一 40M RAM 下只將 CPU EIP 改至 0x101A4，沒有修改任何 RAM bytes、暫存器、flags、EXE、DAT 或存檔。原版 CALL、初始化、END-1、END-2 的 240 次左移、END-2B 與後續圖自然執行到 END-4；在 247550857 步因驗收檢查點達 32 筆停止。停止是測試配額，證據保存 r25/prototype-stop.json 與 native/，不當成原版完成或缺陷。
- READY 使用 r25/ready/ 的第二版凍結來源。只在結局 caller 的延遲入口、END-2 0／120／240 次位移及 END-11B 自然複製返回設置 host 閘門，最多 64 筆。不再擷取文字常式內部逐字等待。閘門不改 guest，解除後從相同原版指令繼續。
- 受控無頭參考與 HD／缺 END-2 兩案從相同完整初始化 RAM 及唯一 EIP 轉移執行。每個檢查點核對步數、CPU、完整 RAM SHA、原生索引與色盤；切換前後維持相同原版 proof，X11 必須等於呈現 PNG，還原須等於原圖。缺圖限 END-2，其他 HD 畫面不變。
- END-2 三個裁切、END-2B 八格及 END-11／11B 的已知區另以核准美術與來源索引逐點比對。未提供 HD 的原生前景與未知末列保留，不放寬來源或色盤資格。其他結局來源依同一原版繪圖與已核准素材抽測，結果按來源記錄。
- 原版自然 exit(0)、完整受控來源快照、控制轉移、各檢查點與無頭／桌面同狀態核對完成後，才標受控結局呈現為 CONFORMED。normal_playthrough_verified 維持 false；此使用者授權的呈現驗收不擴張成正常通關聲明。

### 受控結局尾段色盤與原版輸入等待：RE／READY 補充

- r25/ready/ 的三案在 27 個檢查點的原版 RAM、CPU 與原生畫面相同，END-2 三個位移與 END-2B 八格可切換 HD。END-11／11B 在實際視窗中均為 fallback_palette，暴露來源專用色盤尚未註冊；r20／r24 的合成呈現測試不能證明這段實際色盤已支援。
- 已證實 END-11 的 417099189、417299278、447466383 步與 END-11B 的 450005322 步，無頭及兩個桌面案的全部十六通道相同。來源證據為 r25/ready/ending-palette-source.json；色盤依 DAC 陣列取樣，沒有把 DAC slot 當色號。READY 允許只對固定 END-11／11B 來源註冊這組完整色盤，其他色盤仍備援；不更動美術、原版 DAC 或 fade 常式。
- 已證實最後 sub_1BB48 以 INT 33h、AX=3 輪詢左鍵，另讀鍵盤狀態，未輸入就繼續等待。IDA 9.4 原始定位為 consumers/ida/range-1bb48-1bbac.json；0x1BB77 測試原版回傳 bit 0。r25 無頭 800M 因預算停止；桌面兩案在完成全部末圖擷取後，以普通 X11 左鍵自然退出碼 0。不同牆鐘輸入時點的最終 RAM 不列同狀態比較。
- 後續受控驗收的固定輸入，於 END-11B 最後複製返回後 1M 條原版指令送入普通左鍵狀態，無頭與兩個桌面版本使用相同步數、座標及按鈕。這是已查證的原版輸入契約，不略過等待函式、不強迫 exit、不改 guest RAM。host 閘門期間不推進 DOS，完成末圖擷取後才執行該輸入，讓三案可作完整最終 RAM 對照。
- r25 保留色盤備援與原始手動輸入證據。修正後生產建置、控制初始化與受控三案使用 r26/；正式 startup-probe.go 仍不加入受控入口或驗收用輸入。

### 左上方結局插圖：RE／READY

- 已證實 END-4..10 的完整 loader 四平面符合既有七張故事來源；三個受控執行的目的 `(16,16)..(336,166)` 完整 48000 索引皆等於來源。原生 RGB 亦逐點等於七張既有故事 source PNG，故可重用已核准美術與相對字樣保留區。來源證據為 r26/ending-offset-source.json，不從圖像推定劇情或玩法。
- IDA 9.4 LE 線性原始 caller 見 range-189c8-18db4.json。END-4..8 使用 sub_1C1BC，返回分別為 0x18A07、0x18A6E、0x18AD5、0x18B3C、0x18BA3，raw args 為 `[0,40,150,80,0xA0502]`。END-9／10 使用 sub_1C10B，返回 0x18C19／0x18C8F，前四參數相同，四個目的均為 0xA0502。
- sub_1C10B 會選四個平面並逐次使用四個目的，不能只看第一個目的就當成 VGA 繪圖。range-1c084-1c200.json 中 0x1C10B..0x1C1BC 的 57 列皆符合 r25 原版 RAM，該匯出 SHA 1bb05e29057e0094c190e31d205669ce9aa43ab25656df33ba19d4093651dbe6。本 READY 僅接受上述兩個 caller 及四個相同 VGA 目的，其他 RAM 打包繼續排除。
- READY 擴充既有故事呈現的目的矩形，只允許中央原契約與上述七個固定左上方 caller。來源以實際 loader 檔名、全四平面及普通 RAM 指標核對，不以 A-7 等相同像素別名自動新增其他繪圖資格。自然返回時四個來源 base、完整已知來源及目的索引均須不變／相同。字樣保留區隨相同目的平移，區域外、缺圖與未知狀態保留原版。
- 色盤及左上方接入合併的新建置使用 r27/runtime/；受控來源與三案使用 r27/controlled/，40M 原版控制在 r27/control/。r26/runtime/ 保留已通過四套件的色盤修正版本；r25 兩個已知呈現缺口不重寫為完成。

### 受控結局呈現：CONFORMED

2026-10-16，依本輪使用者授權，只解除結局 HD 的呈現閘門。核對入口為 tools/verify-controlled-ending-summary.py，完整結果為 consumers/battle/r27/controlled/verification-summary.json，SHA-256 a40da3b05150d3f24ac91be336dcbe2bf4005b77a21b06a2db4cbebbda9043d0。正常通關仍為未知，全遊戲 HD 維持未完成。

- 固定 GUI image sha256:baa20f69bccc7dc8ac7a4bbcf93fbcb32019505bf1c31c1312f85aa873bbb9d4，Go 1.24.13、Python 3.11.2、Pillow 9.4.0；dosgolem 基準 a9714ebdab2ad6b529f81225472680f2b11f2842。生產版本四套件通過，研究入口 101.821 秒，106 項編譯輸入保存於 r27/runtime/。本輪沒有新增 CPU 指令形式。
- 40M 普通初始化的完整 RAM SHA a73b974269446406de3b9175b797f8fb1b0b56c105ceab8538c759a7246d6bb4。受控版本僅在該邊界將 EIP 轉至原版 CALL 0x101A4，其他 CPU 值及 RAM 不變。106 項來源中只有測試副本 startup-probe.go 不同，實際 SHA 及局部轉換完整保存在 validation-manifest.json；生產 startup-probe.go 沒有測試入口。
- 無頭原版、桌面 HD 與個別缺 END-2 三案，27 個檢查點的全部 RAM SHA、CPU、原生索引與色盤相同。最後複製返回為 450005322 步，固定普通左鍵輸入在 451005322 步，三案均在 451009120 步由原版退出碼 0。最終完整 RAM、原生畫面、原版 trace、中斷、DPMI、CD PCM 及 graphics trace 相同。原始 EXE、DAT、BIN 與存檔未改。
- 兩個桌面案各有 66 張真正 X11、44 次 F6 切換，共 132 張及 88 次。每張 X11 等於桌面擷取檔；原圖等於獨立原生圖的二倍縮放，還原等於原圖。切換前後的完整原版 proof 相同。HD 以核准素材、實際來源索引、目的、裁切與取樣公式逐點重建，不只比對桌面自身輸出。
- 十二個來源為 END-1、END-2、END-2B、END-4..10、END-11、END-11B。END-2 全部 241 個連續返回通過，實際視窗抽測 0／120／240；END-2B 八個不同裁切全部抽測。七張敘事圖的真正來源、固定 caller、自然返回及左上目的均相符，END-9／10 四個 VGA 目的獨立核對；原版地名及文字保留。END-11／11B 只對已核對原版色盤啟用 HD，未知末列保留。
- 個別缺 END-2 回原圖，其他已合格 HD 繼續呈現。覆蓋清單分開受控十二項與原有正常抽樣，不把受控入口改名成正常玩家路徑；原版畫面、索引、RAM、音訊與收據只留本機。

驗證環境勘誤：r27/control 首次漏掛正確 startup-probe 輸出目錄，修正掛載後以同一建置重跑；兩個 GUI 案初次漏複製驗證器的輔助來源快照，程式尚未啟動即停止，補入相同凍結工具後乾淨重跑。後者原因保存各案 harness-intake-failure.json。兩者均屬工具準備錯誤，沒有放寬原版或 HD 條件。r25 與 r26 的歷史缺口、失敗及來源範圍保持原樣。

### 剩餘動畫及 OP6 受控驗收：DRAFT

2026-10-16，使用者取消正常通關驗收。接續圖像呈現與封包驗證，不新增玩法或強迫勝利。原始輸入、IDA 9.4 LE 位址、工具鏈沿前節固定版本；生產程式仍為 r27 的 106 項來源。

- 已證實 sub_23415 在原版保存矩形後，按 arg_0、dword_3777A 及 dword_376B9／37732 選取 unk_37114／37134 的資源 ID。原版表格與 callsite bytes 保存在 r27/control/startup-memory.bin，指令定位見 consumers/ida/range-23380-23a00.json。0x23507 呼叫共同不透明繪圖，目的為 (216,100)..(424,250)，原版來源偏移依 unk_37174 為 0、26、52、12000、12026、12052。六格後自行還原保存矩形，部分原版分支只播五格。
- 受控 prototype 在原版 40M 初始化後，依上述實際表格選八張動畫的分支，從原始 push／CALL 0x2387E 或 0x2388A 執行，不改動畫、等待、繪圖或還原本體。只在隔離副本設定三個原版測試欄位及 EIP，逐筆記錄前後 bytes 與完整 RAM 雜湊。每次原版返回並清理參數後才開下一個測試。
- OP6 的完整來源已註冊且為 640×400。受控 prototype 沿原始標題的 loader／完整繪圖呼叫，將隔離測試中的資源參數由 OP5 ID=105 改為 OP6 ID=121，保留原版解碼、繪圖與原生目的核對。此案例只證明 OP6 資源可呈現，不推定正式遊戲何時選它。
- 無頭原版與桌面 HD／個別缺圖從相同完整狀態及測試參數執行。驗證原版返回、各動畫格、空格、背景還原、完整 RAM／原生圖、真正 X11、切換不變與缺圖。現有 HD 來源資格、色盤與遮擋不放寬；發現呈現缺口時另補證據及 READY，不能為驗收直接改 guest。

受控初始狀態勘誤：r28 三案完成 45 個原版動畫格、八次還原與 OP6，共 54 個檢查點、102 張 X11；但 40M 標題的原生色盤仍有 142 的亮度上限，與已註冊穩定色盤不同，OP5 與動畫皆正確回退 fallback_palette。r28 不列 HD 通過。保留全部來源、工具及 initial-state-diagnosis.json，另於 r29/control 從原始入口自然執行至 80M，先核對穩定色盤才重新準備受控案例。這是測試初始條件問題，不擴充生產色盤或改原版 fade。

r29/control 的 80M 標題已含原版選單文字，原生索引不再等於未加選單的 OP5 完整資源。準備器改記錄該控制組的實際索引，執行入口另逐項核對十六個穩定色號。無頭來源核對通過 54 個檢查點、45 格及八次原版背景還原，成功步數 101373119。初次桌面案有一筆來自 X11 的自動滑鼠位置，80M 完整 RAM guard 因而拒絕；保留 desktop-hd-input-attempt/ 與 desktop-missing-input-attempt/。新版驗證器以空的 mouse-file 啟用既有輸入隔離，F6／F7／F9 仍由桌面處理；原版輸入維持空排程，沿同一凍結二進位重跑。沒有放寬完整 RAM guard 或改生產輸入契約。

### 剩餘動畫及 OP6 呈現驗收：READY

- r29/source-verification.json 證實原版 45 格的完整目的索引等於各自固定來源裁切；八次原版還原皆等於 80M 控制畫面，OP6 全幅符合完整來源。原版分支畫格數為 6／6／6／5／5／5／6／6，ANIME04 的空白第六格在該分支不播放；空格靜態來源及 HD 準備契約另保持原驗證。
- 本 READY 不要求增加生產呈現。現有來源守衛、共同矩形繪圖、全幅繪圖及視窗保存／還原均重用 r27 實作。桌面需抽測八張動畫各一格及八次還原、OP6 全幅；個別缺 ANIME03／OP6 時只退回對應來源，其他 HD 保持。
- 真正 X11 必須等於呈現檔，原圖須等於獨立原生畫面二倍縮放，還原須等於原圖。HD 依核准素材、來源索引、實際裁切、目的及原版前景遮罩獨立逐點重建。三案全部 54 檢查點、完整最終 RAM 與原版欄位須相同，測試用欄位及 EIP 變更按 manifest 核對。
- 正常通關不列門檻，受控來源不登記為正常玩家路徑。通過後剩餘特殊消費端的接入驗收可收尾；仍保留逐張正常抽樣未完成、原版字型／文字、來源未知像素及平台真機的證據限制。

READY 前置條件補充：r29 的原版標題選單 WINDOW 位於 (232,150)..(408,246)，在受控呼叫動畫時仍是既有活動圖層；動畫覆寫其四角後，WINDOW 正確備援原生矩形，卻使這個非正常戰場的測試畫面包含一塊原版動畫。r29/window-context-diagnosis.json 及 summary-stderr.log 保留該結果，不列完整動畫 HD 通過。r30 在穩定 80M 後先執行原始 0x10B51 的 OP5 loader／full draw，原版完整重畫自然清掉舊 host WINDOW 圖層，才進入動畫。新增完整背景檢查點，合計 55；入口仍不改 RAM，亦不手動刪除 host 圖層或放寬 WINDOW 守衛。後續驗收抽測新增的原圖／HD／還原背景，再依本節驗證動畫與還原。原版通關及正常戰場不由這個測試背景推定。

r30 首次獨立像素核對已通過動畫與背景還原，但 OP6 全幅的左上 32×32 出現 716 個輸出像素差異。已證實差異只在原版游標，見 OP6-difference-diagnosis.json；正式桌面會依原版 AND／XOR mask 重畫游標。驗證器補入獨立的 16×16 原版遮罩、原生索引、DAC 及色盤公式，核對每個游標像素，不排除整個矩形或修改產品。首版工具與失敗保存 summary-attempt-01-stderr.log 及 verify-controlled-art-summary-attempt-01.py，同一收據重查。

### 剩餘圖像與 HD 接入收尾：CONFORMED

2026-10-16，r30/verification-summary.json 為 PASS。三案在 102894281 步完成受控序列，55 個完整狀態及最終 RAM、原生圖、trace、中斷、DPMI、CD PCM 相同。每個案例從 r29/control 的相同 80M 完整 RAM SHA 0b6182a425ab07ed4918c0c989fbaff61ac67fa38ca6c1c92a7a2f225cec18d3 開始；原版重畫背景後才設定明示的三個測試 dword 與 EIP，原版指令及磁碟資料不改。

- ANIME01..08 共 45 個原版播放畫格，來源、目的索引及八次完整背景還原通過。三個五格分支保持原值，靜態空格及未播放畫格沿原有素材測試，不冒稱全部 48 格都有這次動態收據。
- 兩個桌面案各 54 張真正 X11、36 次 F6，共 108 張及 72 次，均保持原版 proof。八張動畫各一格、八次還原、OP5 重畫與 OP6 全幅的完整輸出，逐點符合獨立美術裁切、原版文字遮罩及游標公式。缺 ANIME03／OP6 回原圖，其他 HD 不變。
- 生產 106 項編譯輸入完全等於 r27，沒有新增繪圖特例、CPU 指令或遊戲機制。既有四套件保持 r27 版本範圍；本輪驗證兩個受控 Go 建置、Python 語法與完整原版／X11 證據，不宣稱重新跑過四套件。
- 唯一覆蓋清單確認 111 個 OZM 美術來源、VISUAL 分層、81 頭像、104 地面、十個非空單位及 24 個非空物件已製作與接入；20 項動畫／介面由八項正常及十二項受控抽樣涵蓋。既定 HD 接入驗收完成，可進入三平台封包。

完成範圍限已盤點美術、已知消費端及代表路徑抽樣。原版文字／字型、少量未知來源像素及未知顯示狀態沿原版備援；未逐張正常試玩的素材仍保留其證據等級。依使用者決定不做正常通關驗收，不把取消驗收登記為通關通過。原版資料、原生圖片、RAM、音訊及研究收據只留本機，生成素材維持私人 Git 保存。

### 獨立桌面執行與本機封包：READY

輸入為 85a16a421aaafc276e6b8759491181fe231255aa 的自製程式及固定雜湊原版；本節調整主機端路徑、輸出生命週期與音訊呈現，不修改 DOS 指令、資料、玩法或存檔格式。原有 Docker 探針的預設路徑與有界預算保持。

| 項目 | 契約 |
|---|---|
| 資源與資料路徑 | 正常桌面從執行檔旁的 runtime 與 game 讀取，macOS 改從 Contents/Resources 讀取。來源保持唯讀；使用者資料另放 XDG data、Windows LOCALAPPDATA 或 macOS Application Support，可用明示旗標覆寫。MARK 子程序繼承同一路徑及設定 |
| 存檔 | 重用 dosgolem OpenDirectoryOverlayFiles：讀取優先使用既有工作副本，寫入首次複製原始檔。正常桌面使用固定使用者 saved-data/game；研究預設仍使用一次性隔離目錄。原版 SAV／DAT bytes 不重新編碼 |
| 執行生命週期 | 正常桌面不因研究用 2B 上限自動結束，明示 -steps 仍可作有界驗證。保留原版退出、關窗及 MARK → DX5 鏈；預設關閉 RAM／PCM 大型診斷檔，手動截圖可用 |
| CD 桌面播放 | 沿既有 44.1 kHz／s16le／雙聲道 PCM，加入只接收輸出的有界音訊佇列；使用固定 Ebitengine 2.9.9 audio.Context.NewPlayer 的十六位元契約。音訊消費者不回寫 DOS 狀態、不阻塞原版步進；空佇列補靜音、超限丟最舊完整樣本框並記錄，暫停時停裝置消費。此為主機呈現近似，不宣稱硬體逐週期或人耳驗收 |
| 封包 | Windows amd64、Linux amd64、macOS amd64／arm64；固定 Go 1.24.13 與既有離線模組。使用同一完整版本字串、來源與輸出 SHA、平台／架構、來源權利及存檔位置 manifest。原版與解碼來源只進 dist-all 的 full-local，不進 Git 或 Release |
| 驗收 | 有意義的路徑／PCM 佇列測試、四套件、原版同輸入狀態控制、真正封包啟動及 HD／原圖切換。Linux 音訊用既有 eob-audio-capture:20261009-r3 的 PulseAudio 虛擬裝置擷取；Windows 沿既有 psgolem-wine-verify:noble-r1。macOS 編譯及格式核對與真機啟動分開記錄 |

Ebitengine 十六位元播放器契約直接查固定工具 image 的 /go/pkg/mod/github.com/hajimehoshi/ebiten/v2@v2.9.9/audio/audio.go；它將十六位元 PCM 轉為內部浮點串流。持續存檔契約直接查固定 dosgolem 副本 internal/machine/overlay_files.go 的 OpenRead／OpenWrite，原版讀寫仍經既有 DOS 檔案服務。封包路由已讀 local/retro-remake-dist-all-output.md 與 local/retro-remake-release-versioning.md。
