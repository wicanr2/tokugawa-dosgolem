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
