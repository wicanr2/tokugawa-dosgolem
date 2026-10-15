# 德川家康目前狀態

更新日期：2026-10-15。規則入口：[AGENTS.md](AGENTS.md)。歷程：[WORKLOG.md](WORKLOG.md)。待辦：[WORKLIST.md](WORKLIST.md)。證據：[RESEARCH-LOG.md](RESEARCH-LOG.md)。

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
| 平台回歸 | 最新四套件通過，研究入口 53.770 秒。103 項凍結來源及同一二進位通過預設完整啟動鏈的原版／HD 與正常關窗；兩次 MARK 完整 RAM／十三欄及兩次首戰 750M／390 事件的完整 RAM／十五欄符合各自原版基準，十張實際 X11 相符 | [platform-test.log](workplace/desktop-prototype/startup-chain/startup-probe/platform-test.log)、[驗收總表](workplace/desktop-prototype/startup-chain/verification-summary.json) |
| 原版時間等待 | DOS 時間與 VGA 回掃共用一微秒／成功指令的虛擬時間；原版自行校準等待迴圈 | 最新收據 `virtual_clock`；[回掃契約](RESEARCH-LOG.md)；未略過指令 |
| 光碟 | AUDIO PREGAP 依 CUE 輸出靜音；有效起點的超長請求在 lead-out 停止，保存原始長度與實際終點。原版末軌表格重疊仍保留，沒有改 EXE 或表格 | [cdrom.go](tools/cdrom.go)、[cdda.go](tools/cdda.go)、[因果與契約](RESEARCH-LOG.md) |
| CDDA 技術驗證 | 完成 PLAY／STOP／RESUME、狀態、自然結束、分塊、跨軌、PREGAP、極大長度與非法起點測試；十五條音軌全段與 BIN 一致。正常流程八次選曲無拒絕，輸出 19,818,728 個立體聲樣本框。桌面與人耳確認未完成 | [CDDA 測試](tools/cdda_test.go)、`workplace/startup-probe/cdda-boundary-test.log`、[獨立核對](workplace/startup-probe/cdda-verification.json)；79,274,912 bytes 逐段與 BIN 及時間公式相符 |
| CD 只讀診斷 | 開關對照通過；同一 3,000,000 步的完整記憶體、畫面、停止位置、中斷計數與原始 trace 一致；未修改原版或 DPMI 回傳契約 | [startup-probe.go](tools/startup-probe.go) 的 `-cd-trace`；`workplace/cd-trace/trace-on.json`、`trace-off.json` |
| 滑鼠與 BIOS | 普通滑鼠已推進選單、角色建立與敘事；已接原版遮罩與畫面游標層，未實作 TSR 的 VRAM 背景讀回 | [video.go](tools/video.go)；`latest.json` 的輸入與邊界畫面 |
| VGA／VESA 顯示 | 原版模式 12h 與 CRTC 自行設定 640×400，開場、標題與主選單已目視確認；VBE 分頁子集另有測試 | [原生標題與選單](workplace/startup-probe/frame-7df7211c989bca30c3e82077011a96e7ef7142c9029c515aeb5dda5e461697e7.png)、[video.go](tools/video.go) |
| DOSBox 輔助基準 | 可顯示原版標題；實際 DOS/4GW 的 MCB 配置可重用，段號相差 2。第 1、2 軌 MSF 起點保留，控制欄位仍遭重疊覆蓋；未作音訊驗證 | [25s.png](workplace/dosbox-aux/25s.png)、[aux-cd-trace.py](tools/aux-cd-trace.py)、`workplace/cd-trace/aux-toc/manifest.json`；詳見 RESEARCH-LOG |
| 正常玩家路徑、存讀檔與音訊 | 原版入口、角色建立、敘事、選項、出陣與首場戰場顯示保持通過，原版選曲／換曲與 PCM 同步通過；單位操作、戰鬥、存讀檔及桌面聽覺未驗證 | [普通輸入時間表](tools/player-path.json)；精確畫面、音訊雜湊與範圍見 RESEARCH-LOG |
| 普通歷史簡介與離開 | 普通點擊後自然 600M，經 OP2、OP2_P、OP4、劇情、OP8、OP9、OP-FULL 回到 OP5 主選單，沒有未支援指令。完整、未位移局部與 OP2／OP8 捲動皆已接 HD，抽樣範圍分別記錄。普通離開在 80017192 步正常 dos_exit，BDA 恢復模式 3／80 欄；不宣稱文字畫面驗收 | [歷史原版收據](workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/rcr/receipt.json)、[捲動驗收](workplace/hd-prototype/catalog/remaining/history/scrolling/verification-summary.json)、[離開收據](workplace/hd-prototype/catalog/remaining/source-boundary/menu-exit/platform/ready/receipt.json) |
| 啟動顯示選擇 | 預設在 MARK 前選原版／HD，只詢問一次。選擇及確認手勢期間原版保持 0 步、沒有按鈕轉送；MARK 內最後設定帶進遊戲。兩條流程均到首戰，MARK 與遊戲功能列的雙向切換及還原保持原版狀態；模式選擇畫面關窗結束整個啟動器 | [桌面啟動器](tools/desktop-publisher.go)、[完整驗收](workplace/desktop-prototype/startup-chain/verification-summary.json) |
| MARK 恢復及 HD | CONFORMED。已接入預設桌面啟動鏈，原版 MARK 自然漸入、停留、漸出並在 15989208 步退出碼 0，隨後啟動 DX5 並沿用最後模式。兩份原版 RAM 與虛擬時間獨立保存；獨立模式的缺圖驗證仍有效 | [啟動器](tools/desktop-publisher.go)、[實際 HD 標誌](workplace/desktop-prototype/startup-chain/hd/mark-last.png)、[完整驗收](workplace/desktop-prototype/startup-chain/verification-summary.json) |
| remake | 排除 | 使用者明確指定原版與 dosgolem 整合 |
| 後續功能範圍 | CDDA 裝置與 PCM 已實作；桌面播放、繁中／簡中／英文／日文與可切換 HD 素材包待後續切片 | 已確認範圍與驗收見 [WORKLIST.md](WORKLIST.md) |
| 後續使用介面 | 桌面版；全遊戲 HD 後打包 Windows、Linux、macOS；瀏覽器排除 | 使用者已確認；目前只驗證 Linux 容器桌面 |
| HD 風格試作 | B 戰國手繪風。75 個完整劇情來源與四個部分來源的已知區已接入既有故事消費端，另有捲軸、紅金框架、共用視窗及 81 張頭像。正常劇情仍抽測十四張、頭像四張；四項部分來源未在正常場景抽測，不把條件式註冊稱為正常驗收 | [既有劇情驗收](workplace/hd-prototype/catalog/story-expanded/verification-summary.json)、[歷史段落](workplace/hd-prototype/catalog/remaining/history/partial/verification-summary.json)、[部分來源驗收](workplace/hd-prototype/catalog/remaining/partial-story/verification-summary.json) |
| HD 素材定位 | DAT 201 筆、112 個 OZM。靜態解碼 107 筆，17 個正常路徑來源的平面與索引一致；五筆尾端少一 byte，保留未知。KAO.GRP 的 81 張頭像索引與繪圖端已確認 | [解碼器](tools/hd-ozm-probe.py)、[頭像擷取](tools/hd-kao-extract.py)、[覆蓋清單](workplace/hd-prototype/catalog/art-coverage.json) |
| OP-FULL 來源尾端與美術 | 兩次普通歷史簡介的完整四平面與索引相同，第四平面尾端皆為 0xAA；保留 runtime_observed 分類，DAT 與靜態解碼器未改。第二版手繪圖校正原生色盤下的藍白雲帶，完整複製端已驗證正常原圖／HD／還原；第一版與提示詞亦保存 | [來源核對](workplace/hd-prototype/catalog/remaining/history/source-verification.json)、[第二版與提示詞](assets/hd-b/art/opening-full-generation-v2.json)、[正常開場 HD](workplace/hd-prototype/catalog/remaining/history/desktop/palette/verification-opening.json) |
| 全遊戲 HD 覆蓋 | 製作中。112 個 OZM 中 111 項已有素材，一項 VISUAL 分層；新增 B-6／B-7／C-10／E-4 的 47992 已知像素素材及條件式故事消費端，每張八個未知像素保留原圖。靜態解碼仍五項未知。20 項既有動畫／介面來源中正常新來源為 OP5／OP2_P／OP9，其他十七項消費端仍待完成；MARK PCX 的完整啟動鏈、模式沿用及動態切換已通過。四項新故事場景未逐項正常抽測，打包未開始 | [覆蓋清單](workplace/hd-prototype/catalog/art-coverage.json)、[部分來源驗收](workplace/hd-prototype/catalog/remaining/partial-story/verification-summary.json)、[圖片與提示詞](assets/hd-b/README.md)；素材準備與正常路徑分開 |
| 戰場地面 HD | 完整 134 次地面返回順序已接入；原有 65 完整格與兩項 1440 像素可見碎片共同呈現，67 處共 102720 原生背景像素。正常六種來源與原有八項 HD 素材不變；全部 104 來源已接入並通過共同呈現、缺圖及遮擋測試，其他 98 個未逐項在正常路徑抽測。被後續地面蓋掉的物件不復活，四項已被未知介面覆寫的模型殘片備援。三模式各自然 750M／390 事件，完整 RAM、十五欄、原生圖與 PCM 符合基準 | [驗收總表](workplace/hd-prototype/catalog/terrain/scene/verification-summary.json)、[真實視窗](workplace/hd-prototype/catalog/terrain/scene/desktop/gui-scene-enabled.png) |
| 全地面 HD 素材 | 全部 104 來源已準備並接入，共 101 張獨立圖。原有八張不改，新增 93 張覆蓋其餘 96 個來源。七頁原圖／HD 逐項檢視及四套件通過；新版本自然 750M／390 事件，完整 RAM、十五欄及原圖／HD 擷取與前次天氣版本相同。正常首戰仍為六種，其他來源不冒稱正常路徑已抽測 | [驗收總表](workplace/hd-prototype/catalog/terrain/materials/verification-summary.json)、[純 HD 總覽](assets/hd-b/terrain/materials/floor-material-gallery-v2.png)、[來源與素材](assets/hd-b/terrain/materials/profiles.json) |
| 戰場前景 HD | 四樹叢與 units-001／003 在同一畫布合成，包含早畫仍可見人物的優先權。三模式各自然 750M／390 事件、三次狀態不變切換；完整 RAM、十五欄、原生圖及 PCM 符合基準。共同呈現 413696 像素與獨立公式相符，重疊區 82 個人物像素正確使用 HD；兩人物舊輪廓 1584／1574 呈現像素還原背景，1620 介面像素保留，全部缺圖零變動。游標公式核對排除 16×16，但真實 X11 與呈現 PNG 完整相符 | [驗收總表](workplace/hd-prototype/catalog/terrain/scene/verification-summary.json)、[真實視窗](workplace/hd-prototype/catalog/terrain/scene/desktop/gui-scene-enabled.png)、[來源及 alpha 契約](RESEARCH-LOG.md) |
| 完整單位 HD | 十個非空來源 0..9 已接入，三個空白 10..12 保留。002／007 姿勢、006 刀勢與 008 分叉槍頭採 v2，001／003 的 SHA、裁切及準備後 PNG 不改。十項實際來源的共同場景、透明邊界、缺圖、未知覆蓋及矩形外保護測試通過；正常 750M／390 事件、完整 RAM、十五欄及原圖／HD 完整擷取等於前次地面版本。正常首戰仍只抽測 001／003，其他八項未逐項正常抽測，不推定兵種映射 | [驗收總表](workplace/hd-prototype/catalog/terrain/units/verification-summary.json)、[純 HD 總覽](assets/hd-b/terrain/units/unit-HD-gallery-v2.png)、[來源與裁切](assets/hd-b/terrain/units/profiles.json) |
| 獨立物件 HD | 0、1、2、3、18、19、22、23 已接入，含既有樹叢、岩石、兩座城樓、布幕、柵欄及兩方向橋面。八項來源的共同場景、透明邊界、缺圖、未知覆蓋、矩形外保護及多物件優先權通過；正常 750M／390 事件、完整 RAM、十五欄及三次原圖／HD 切換通過，真實視窗與前次單位版本相同。正常首戰仍只有 0，其餘七項未逐項正常抽測。完整城牆擴充見下一列，24、25 保留空白 | [驗收總表](workplace/hd-prototype/catalog/terrain/objects/expanded/verification-summary.json)、[物件總覽](assets/hd-b/terrain/objects/object-HD-gallery-v1.png)、[來源及核准索引](assets/hd-b/terrain/objects/profiles.json)；總覽含城牆草稿 |
| 完整物件圖集 HD | 24 個非空物件全部接入，24、25 空白保留。16 項牆件使用共用手繪材質校準屋簷、石基及接縫；6／10、12／13 重用已證實相同的來源形狀，各自來源指紋保留。既有八項圖片、裁切及準備後 PNG 不改。全部物件走共同場景測試，正常 750M／390 事件、完整 RAM、十五欄及三次切換通過，X11 擷取等於前版。正常路徑仍只抽測 0 的四處樹叢；離線圍牆不作玩家路徑證據 | [驗收總表](workplace/hd-prototype/catalog/terrain/objects/walls/verification-summary.json)、[完整物件總覽](assets/hd-b/terrain/objects/walls/object-HD-gallery-v2.png)、[HD 圍牆](assets/hd-b/terrain/objects/walls/draft-v2/wall-HD-enclosure-v2.png)、[來源與裁切](assets/hd-b/terrain/objects/walls/profiles.json) |
| HD 劇情與頭像驗證 | 75 個來源通過資料與共同呈現測試，正常抽測 A-0..A-7、GUNGI 及新增 6／11／19／28／43，共十四張。頭像仍抽測 76..79，地名保留原版。新歷史段落五次返回來源資格皆正確，五組原圖／HD／還原及個別缺 11 備援通過；其餘 61 個劇情與其餘頭像未逐張正常抽測 | [既有收據](workplace/hd-prototype/catalog/story-expanded/desktop/desktop-portraits.json)、[新畫面抽樣](workplace/hd-prototype/catalog/remaining/history/partial/capture-verification.json)、[完整歷史驗收](workplace/hd-prototype/catalog/remaining/history/partial/verification-summary.json) |
| 桌面呈現與普通滑鼠 | Linux 容器內 Ebitengine 2.9.9 試作通過。正常入口與普通時間表走到首張敘事；視窗縮為 1000×700 後，普通 X11 左鍵正確對應原版 `(300,171)` 並開始新遊戲。支援 F6 切換、F7 暫停、F9 擷取；現行功能列與關窗操作見下一列；一般鍵盤、桌面音訊及其他 OS 未驗收 | [建置入口](tools/run-desktop.sh)、[普通滑鼠核對](workplace/desktop-prototype/verification-mouse.json)、[新遊戲開場](workplace/desktop-prototype/gui-mouse-after-click.png)；操作及限制見 AGENTS |
| 劇情捲軸 HD | 完成此介面切片。VISUAL.OZM 原版平面與捲軸底圖已核對，紙張、捲邊及外框以 HD 替換；文字前景逐點保留。兩頁劇情保留 24324／4036 個呈現前景像素。HD、缺捲軸圖及只有捲軸 HD 三模式的 RAM、原生畫面與 CD PCM 均符合同一 150M 基準；頂端功能列整合亦通過。字型重繪及其他介面仍待後續 | [驗收總表](workplace/hd-prototype/scroll/verification-summary.json)、[實際畫面](workplace/hd-prototype/scroll/desktop/gui-scroll-enabled.png)、[第二頁](workplace/hd-prototype/scroll/desktop/gui-scroll-second-enabled.png)、[素材](workplace/hd-prototype/scroll/scroll-hd-b-v1.png)、[完整提示詞](workplace/hd-prototype/scroll/scroll-hd-b-v1.prompt.txt) |
| 劇情上方框架 HD | 完成此切片。完整 HD、缺框架圖、只有框架 HD 三模式及只有框架時的頂端列通過；中央圖與文字維持各自圖層。四次 150M 的原版 RAM、畫面、輸入及 PCM 符合基準 | [驗收總表](workplace/hd-prototype/scroll/frame/verification-summary.json)、[實際畫面](workplace/hd-prototype/scroll/frame/desktop/gui-frame-enabled.png) |
| 共用視窗 HD | 完成已抽測切片。九圖塊金框與紙張沿用原版幾何，文字及邊框前景保留。只有視窗、搭配頭像、缺圖、確認及能力視窗、頂端列五種驗證通過，每次 150M 的完整 RAM、原生畫面、輸入與 PCM 符合基準；其他視窗與平台未逐項驗收 | [驗收總表](workplace/hd-prototype/catalog/window/verification-summary.json)、[角色選擇](workplace/hd-prototype/catalog/window/desktop-window/gui-window-enabled.png)、[確認框](workplace/hd-prototype/catalog/window/desktop-window-modal/gui-window-modal-confirmation-enabled.png) |
| 生成美術的私人保存 | 807 張 PNG、163 份完整提示詞及來源索引已保存，含四項部分來源共用的兩張校準圖、純 HD 總覽與 RAINEND 出版商標誌草稿。原始生成圖不改，未知來源像素不推測；原版圖、索引、字模、音訊及執行收據留本機 | [HD 素材](assets/hd-b/README.md)、[索引](assets/hd-b/index.json)、[保存工具](tools/archive-hd-art.py)；遠端結果見 WORKLOG |
| 戰場框架 HD | v2 在正常能力／確認視窗及首戰通過。三模式各自然 750M／390 事件、五次狀態不變切換，完整 RAM、十五欄、原生圖及 PCM 符合基準，六張真實 X11 圖與呈現 PNG 完整相符。共同 HD 場景的框架 347932、場景 413696 像素符合獨立公式；1564 個人物／樹叢旁透明處呈現像素使用 HD 框架底圖，文字、地圖、天氣及未知標記保留。只有框架時原版人物保留；缺框架仍顯示原有 HD 場景。被視窗遮住的徽記備援，正向徽記限原版複製來源與單元測試，未宣稱正常 GUI 裸露徽記已驗收 | [驗收總表](workplace/hd-prototype/catalog/battle-frame/verification-summary.json)、[真實視窗](workplace/hd-prototype/catalog/battle-frame/desktop/gui-battle-frame-battle-enabled.png)、[來源總表](workplace/hd-prototype/catalog/battle-frame/source-verification.json) |
| 天氣 HD | 晴天／陰天 v2 已接共同不透明 OZM 繪圖端。晴天及缺圖兩模式各自然 750M／390 事件，完整 RAM、十五欄及三次切換通過。42624 呈現像素符合取樣公式，缺圖同區零變動；陰天未在正常路徑抽測。OP2／OP8 的校準及驗證範圍見靜態美術列 | [驗收總表](workplace/hd-prototype/catalog/graphics/verification-summary.json)、[實際畫面](workplace/hd-prototype/catalog/graphics/desktop/gui-weather-enabled.png) |
| 靜態美術與開場標誌 HD | BATTLE01／BATTLE02／OP2／OP4／OP8 及 OP1 共六張已校準，原版緩衝區保留。OP1 正常開場、OP4 徽記及 OP2／OP8 捲動已完成各自正常 X11 抽樣。兩張戰鬥背景只列來源與共同呈現測試，未逐項正常抽測 | [既有標誌驗收](workplace/hd-prototype/catalog/graphics/static/verification-summary.json)、[徽記驗收](workplace/hd-prototype/catalog/remaining/history/verification-summary.json)、[捲動驗收](workplace/hd-prototype/catalog/remaining/history/scrolling/verification-summary.json)、[來源與校準](assets/hd-b/art/static/profiles.json) |
| 開場色彩效果與完整複製 | 新增自然 sub_11F14 全畫面繪製觀測，來源四平面別名及完整目的索引皆核對；未知來源撤銷舊圖層。OP4／OP-FULL 六次亮度與兩次色盤循環的 X11／HD 公式、原圖／HD／還原及各 600M 完整 RAM／十五欄通過；個別缺 OP-FULL 只退回該圖。HD 色彩轉換屬呈現近似，原版 palette、時間及存檔未改。該切片版本標題 200M、首戰 750M 回歸通過；局部及捲動另見下列切片 | [驗收總表](workplace/hd-prototype/catalog/remaining/history/verification-summary.json)、[實際開場 HD](workplace/hd-prototype/catalog/remaining/history/desktop/palette/gui-opening-OP-FULL-enabled.png)、[色彩契約](workplace/hd-prototype/catalog/remaining/history/palette-effects.json) |
| 未位移局部繪圖 HD | 已接入三格交鋒、四格眼部、五張武將及五張歷史劇情。兩次完整歷史皆新增十二次局部返回與五次劇情返回資格，四平面完整指紋未放寬。正常 HD X11 抽樣為三格交鋒、四格眼部、兩張武將與五張劇情，共十四組；缺圖十七組。其他三張武將列來源及共同呈現測試，不稱正常 HD X11 抽樣通過。原先 OP9 暫存區描述已勘誤，十四筆原始抽樣皆未改寫來源 | [驗收總表](workplace/hd-prototype/catalog/remaining/history/partial/verification-summary.json)、[畫面抽樣](workplace/hd-prototype/catalog/remaining/history/partial/capture-verification.json)、[研究與勘誤](RESEARCH-LOG.md) |
| OP2／OP8 捲動 HD | CONFORMED。十三筆原版來源與 682 個連續繪圖狀態測試通過；正常／個別缺圖各五組 X11 原圖／HD／還原。OP8 早期正常七個、缺圖六個來源外像素保留原生圖。兩次完整 600M 都有 700 次合格繪圖返回，新增 682 次；完整 RAM、十五欄與原版基準一致，首戰回歸通過。未逐格 X11 擷取的狀態只列來源／共同呈現測試 | [驗收總表](workplace/hd-prototype/catalog/remaining/history/scrolling/verification-summary.json)、[來源證據](workplace/hd-prototype/catalog/remaining/history/scrolling/source-verification.json)、[READY 與驗收](RESEARCH-LOG.md) |
| 部分來源故事圖 | CONFORMED，限已知區及既有已證實消費端。四項來源已校準，三完整平面／第四前綴與自然返回完整 guard 通過；每張八像素保留原生圖。所有未知末 byte 測試值均為 synthetic input，不當原版證據。四項素材桌面載入通過，既有歷史／首戰回歸通過；這四個正常場景未逐項抽測 | [驗收總表](workplace/hd-prototype/catalog/remaining/partial-story/verification-summary.json)、[準備與幾何](workplace/hd-prototype/catalog/remaining/partial-story/profiles.json)、[選定素材](assets/hd-b/art/partial-story/) |
| PCX 出版商標誌 | CONFORMED。原版 640×480／256 色的來源、目的索引與獨立 PCX 解碼相同，目標色盤及 RAM guard 已核對。獨立模式原圖／HD／還原及缺圖通過；預設完整鏈的雙向切換、模式沿用與首戰通過，下方 y=449..479 版權區保留原版 | [HD 呈現](tools/hd-publisher.go)、[完整鏈驗收](workplace/desktop-prototype/startup-chain/verification-summary.json)、[獨立模式驗收](workplace/desktop-prototype/publisher/verification-summary.json)、[素材及提示詞](assets/hd-b/art/RAINEND-generation-v1.json) |
| 剩餘動畫與介面美術 | 20 項已註冊且通過來源及共同呈現測試；八個 ANIME 圖集含 48 格、16 個精確鏡像及一空格，另有八格結局天空與三格交鋒。正常 OP5 已驗證，本批加入 OP2_P 三格與 OP9 的眼部／武將局部繪圖；另十七項尚未逐項正常抽測，特殊消費端仍須確認 | [既有註冊驗收](workplace/hd-prototype/catalog/remaining/runtime-verification-summary.json)、[局部接入](workplace/hd-prototype/catalog/remaining/history/partial/verification-summary.json)、[選定來源](assets/hd-b/art/remaining/profiles.json) |
| 頂端隱藏功能列 | Linux 容器試作通過。參考 softworld_san：頂端滑鼠展開、移開收起，Esc 固定／收起，Shift+Esc 離開。提供原版／HD 圖像、繼續／暫停、截圖／離開。顯示與收起手勢期間隔離原版步進及輸入；視窗擴高而保持遊戲比例。HD、缺圖兩次 150,000,000 步的完整 RAM、原生畫面與 PCM 符合原基準；普通滑鼠未誤觸，縮放後仍可開始新遊戲 | [驗收總表](workplace/desktop-prototype/toolbar/verification-summary.json)、[功能列畫面](workplace/desktop-prototype/toolbar/windowbar-hd-hd.png)、[原圖選單](workplace/desktop-prototype/toolbar/windowbar-hd-display-menu.png)、[普通滑鼠驗證](workplace/desktop-prototype/toolbar/verification-windowbar-mouse.json)；四語及音訊開關待功能接妥 |
| private GitHub repository 與 Issues | 主機 gh 已確認 [wicanr2/tokugawa-dosgolem](https://github.com/wicanr2/tokugawa-dosgolem) 為 private，六筆 Issues 已建立；#2 技術切片驗收後已關閉，HD 接續 #5，桌面與聽覺驗收留 #3 | [CDDA 完成記錄](https://github.com/wicanr2/tokugawa-dosgolem/issues/2#issuecomment-6121828738)、[DESKTOP #3](https://github.com/wicanr2/tokugawa-dosgolem/issues/3)、[HD #5](https://github.com/wicanr2/tokugawa-dosgolem/issues/5)；Issue 即時狀態直接查 GitHub |
| Git 提交範圍 | 已建立本機 `main` 並推送自製工具、測試、文件及使用者授權的生成美術；原版與執行產物排除 | `.gitignore` 與 Git 實際檔案清單；遠端提交以 GitHub 為準 |
| GitHub 建立工具驗證 | shell 語法、六筆本文擷取與隔離假 API 測試通過；遠端未實測 | `--prepare` 不呼叫 GitHub；查重、相依 URL、登入／列舉失敗、公有同名 repo 與重複 Issues 的停止行為已測。歷程見 WORKLOG |
| Docker 收尾 | 本批建置、三模式完整啟動鏈、覆蓋盤點及來源／美術核對均已結束；沿用既有 image，沒有 root-owned 或 .md 目錄，正式 IDA 資料庫不變，全部 tokugawa 容器已移除 | 本批入口為 desktop-prototype/startup-chain/verification-summary.json；遠端保存見 WORKLOG |

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

目前任務是完成全遊戲 HD。原版 LE 入口至新遊戲戰場的整合啟動已通過四套件及完整收據核對；全遊戲相容、桌面音訊與存讀檔仍按上表分項追蹤，不重新開啟已通過的啟動切片。

主機 `/home/anr2/.local/bin/gh` 已登入 `wicanr2`；private repository、`main` 與六筆 Issues 已建立。Issue 現況直接查 GitHub，先前存取受阻與建立歷程留在 WORKLOG。原版資料仍不得入 Git。

## 後續擴充的已確認決定

- 2026-10-15 使用者確認恢復包內 `MARK.EXE.BAK` 的出版商標誌，並要求 MARK 也可切換 HD。啟動遊戲前先選「原版／HD」，進入後仍可從頂端功能列動態切換。啟動選擇與既有功能列切換已實作並驗證；MARK 的原生入口、顯示、退出、HD 與完整啟動串接均已驗證，遊戲沿用 MARK 最後選擇的模式。原始 `.BAK` 保持不變，其版本來源仍未知。
- 原版 DOS 程式繼續負責玩法、狀態與存檔。本次沒有改成 remake。
- 使用者選擇可切換 HD 素材包，逐項替換人物、場景與介面。原版顯示保留作對照與缺圖備援；單純放大原圖不作為 HD 素材包完成標準。
- 使用者已選 B 戰國手繪風，依同場景比較的自然地形與手繪人物、介面方向製作。生成概念圖不作正式版面或幾何來源，原版邏輯座標仍為 640×400。
- 第一批語言為繁中、簡中、英文與日文。原文與譯文均以 UTF-8 TSV catalog 管理，穩定 key 的規則依 `/home/anr2/.codex/knowledge-base/local/localization-display-semantic-isolation.md`；翻譯只進顯示路徑，不改原版比較、資源名稱或存檔欄位。
- 使用者確認頂端功能列後，另指定捲軸介面也要 HD；劇情捲軸已接入既有 HD 開關，維持原版文字及位置。
- 使用者指定頂端隱藏功能列參考 `/home/anr2/cht/softworld_san`，目前已接入顯示、執行與工具三欄；四語與音訊待功能實作後接入。
- 桌面版為本次目標。使用者已確認全遊戲美術 HD 完成後打包 Windows、Linux、macOS；各平台建置與啟動驗收分開記錄。瀏覽器與 WebAssembly 不列入規劃或後備路線。
- 使用者指定完成 CD 音訊後先執行 HD 化。四語需求保留，實作順序移到 HD 首個場景之後；桌面畫面與音訊試作作為必要依賴一起推進。
- 使用者已授權依 [WORKLIST.md](WORKLIST.md) 的功能範圍開工。MCB 與 CDDA 技術切片已實作，HD 美術風格已選 B。桌面採 Ebitengine 2.9.9，原版邏輯畫面 640×400，既有素材以兩倍尺寸呈現；完整文字擷取與四語仍待後續切片。
