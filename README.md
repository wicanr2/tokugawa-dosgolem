# 德川家康 × dosgolem

讓 DOS 原版《德川家康》在現代桌面執行，沿用原版程式、玩法與資料，加入戰國手繪 HD、CD 音樂播放及可隨時切換的顯示介面。

## 遊戲故事背景

元龜三年冬，武田信玄的大軍進逼遠江。與織田信長結盟的德川家康，必須面對兵力強大的武田軍。三方原之戰成為他一生的重要挫敗，遊戲也從他必須保全自身的危局開始。這段歷史背景可參考[濱松市博物館的家康與三方原之戰介紹](https://www.city.hamamatsu.shizuoka.jp/hamahaku/02tenji/theme/ieyasudennsyoutohamamatsu.html)。

遊戲從第一章「麒麟兒奮起」展開，讓玩家帶領德川軍，在軍議中作出選擇，再到戰場調度武將與部隊。原版十二章的時間由元龜三年推進到元和元年，經過武田勢力、與豐臣方的角力，走向「命運的關原」、「疾風！真田六連錢」與大阪之陣。故事裡的每次出陣，都連著家康如何保全自身、應對強敵與爭取主導權的抉擇。章名、年代與首章背景依[原版章節文字證據](RESEARCH-LOG.md#故事背景來源)整理。

史實中，豐臣秀吉死後，家康與石田三成等人的對立逐漸擴大。1600 年關原之戰後，家康取得優勢，並於 1603 年建立江戶幕府；1615 年大阪夏之陣則使德川與豐臣之間的爭鬥走向終局。歷史補述依據[日本國立公文書館](https://www.archives.go.jp/exhibition/digital/ieyasu/contents3_01/)與[岡崎市觀光協會的家康年表](https://okazaki-kanko.jp/okazaki-park/feature/history/about)。遊戲中的戰況、選項與勝敗仍由原版程式判定。

## 桌面版內容

- 啟動時選擇原版或 HD，出版商 MARK 標誌也支援 HD，最後設定沿用到遊戲。
- 戰國手繪 HD 涵蓋已盤點的人物、場景、戰場、捲軸與介面；缺圖或未知顯示狀態回原版。
- 播放原版光碟音樂，存檔寫入獨立使用者目錄，保留原版資料。
- 滑鼠移到視窗頂端可展開功能列，隨時切換原版／HD、暫停與截圖。

遊戲操作以滑鼠為主。啟動選擇也可使用 Tab／Enter。

| 操作 | 功能 |
|---|---|
| 滑鼠移到頂端、Esc | 展開或收起功能列 |
| F6 | 切換原版／HD |
| F7 | 暫停／繼續 |
| F9 | 截圖 |

目前原版入口、選單、劇情、首戰、代表性交鋒及存讀檔已有抽樣驗證；HD 另有來源、繪圖、切換與缺圖驗證。CD 桌面輸出已做技術擷取，人耳及實體桌面的預設速度效能尚未確認。四語切換尚未實作。完整範圍與限制以 [CONTEXT.md](CONTEXT.md) 為準，正常通關不列本次驗收門檻。

## 三平台完整版

目前本機版本為 `v.0.1.0-20261016`。解壓後啟動 Windows／Linux 執行檔，或開啟 macOS 的 `Tokugawa.app`；詳細操作與存檔位置見包內 README.txt。

| 平台 | 架構 | 完整版 | 已驗證範圍 |
|---|---|---|---|
| Windows | amd64 | [ZIP](dist-all/v.0.1.0-20261016/full-local/tokugawa-dosgolem-v.0.1.0-20261016-windows-amd64.zip) | Wine 啟動、MARK、正常標題、功能列切換與關窗 |
| Linux | amd64 | [ZIP](dist-all/v.0.1.0-20261016/full-local/tokugawa-dosgolem-v.0.1.0-20261016-linux-amd64.zip) | X11 啟動、MARK、正常標題、功能列切換與關窗 |
| macOS | Intel | [ZIP](dist-all/v.0.1.0-20261016/full-local/tokugawa-dosgolem-v.0.1.0-20261016-macos-amd64.zip) | 編譯與格式核對，待真機試跑 |
| macOS | Apple Silicon | [ZIP](dist-all/v.0.1.0-20261016/full-local/tokugawa-dosgolem-v.0.1.0-20261016-macos-arm64.zip) | 編譯與格式核對，待真機試跑 |

[封包 SHA-256](dist-all/v.0.1.0-20261016/SHA256SUMS.json)與[驗證收據](dist-all/v.0.1.0-20261016/smoke/verification-summary.json)保存在同一版本目錄。完整版含原版遊戲與解碼參考，只留本機，不加入 Git 或 GitHub Release。

## 推廣影片

[觀看 72 秒、1080p 推廣影片](dist-all/v.0.1.0-20261016/promo/tokugawa-dosgolem-v.0.1.0-20261016-promo.mp4)。畫面錄自正式 Linux 封包的正常啟動、劇情與首戰，經剪輯與速度調整，展示原版／HD 與頂端功能列切換；配樂取自原版 CD。影片與素材只留本機，來源及檢查收據保存在同一 promo 目錄。

## 文件與素材

- [工作規則、建置與驗證入口](AGENTS.md)
- [目前狀態](CONTEXT.md)
- [後續工作與四語規劃](WORKLIST.md)
- [逆向與來源證據](RESEARCH-LOG.md)
- [HD 圖像與完整提示詞](assets/hd-b/README.md)
- [工作歷程](WORKLOG.md)

程式與生成素材保存於私人儲存庫 [wicanr2/tokugawa-dosgolem](https://github.com/wicanr2/tokugawa-dosgolem)。原始下載只存本機 original/，研究與錄製素材存 workplace/，交付存 dist-all/；三者均排除版控。開發與研究使用 dosgolem 固定副本，桌面封包可獨立執行。

## 授權

自製程式與文件採 source-available [RRSAL-1.0](LICENSE)。非商業使用免費，商業用途另洽；原版資料、美術與音樂不在授權範圍，第三方元件沿用各自條款。本專案與原版權利人沒有隸屬或官方授權關係。
