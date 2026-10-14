# 德川家康 × dosgolem

將 DOS 原版《德川家康》接到 dosgolem 執行，保留原版程式與資料。此專案不製作 remake。

GitHub 儲存庫：[wicanr2/tokugawa-dosgolem](https://github.com/wicanr2/tokugawa-dosgolem)。

已下載並核對 jsdos。dosgolem 能從原版入口，經主選單、角色建立、敘事與出陣確認，顯示戰場地形、人物單位及小地圖，並持續接收滑鼠查詢；普通滑鼠能推進對話並選取劇情選項。單位操作、戰鬥與存讀檔仍待驗證。

CDDA 裝置已支援原版 PLAY／STOP／RESUME、本機 BIN 的 PCM 串流、PREGAP 靜音與映像結尾停止。十五條音軌及正常入口至首場戰場的選曲、換曲與 PCM 已驗證，無頭收據保存事件、樣本數與音訊雜湊；桌面輸出及人耳確認待驗。後續優先製作可切換的 HD 素材包，現況見 CONTEXT。

HD 採戰國手繪風。Linux 桌面試作已接入 75 個劇情來源的共同繪圖路徑，正常流程抽測九張。捲軸、紅金框架、共用視窗及 81 張頭像亦已接入，正常流程抽測四張頭像及角色選擇、確認、能力視窗。來源測試與正常玩家路徑的驗證範圍見 CONTEXT。

戰場已接入八項地面素材、樹叢 v2、兩個單位來源及紅金框架的 HD。首戰抽測六種地面、67 處可見區、四處樹叢及兩個人物；框架、重疊、透明底圖與原版可見人物優先權已驗證。文字、動態地圖、天氣及未知標記保留原版。未知覆蓋或個別缺圖回原圖，其餘物件、單位與其他美術仍在製作。

F6 切換原版／HD，個別缺圖回原圖，文字及地名保留原版。滑鼠移到頂端可展開設定列，Esc 固定／收起，提供顯示、暫停與截圖。全遊戲美術仍在製作，完成後打包 Windows、Linux、macOS。正式素材包、四語、桌面音訊與一般鍵盤輸入仍待後續。建置與驗證入口見 AGENTS。

- [工作規則與重跑命令](AGENTS.md)
- [HD 圖像與完整提示詞](assets/hd-b/README.md)
- [目前狀態](CONTEXT.md)
- [接入待辦與 CD／四語／HD 規劃](WORKLIST.md)
- [啟動證據與平台修正契約](RESEARCH-LOG.md)
- [工作歷程](WORKLOG.md)

執行依賴 `/home/anr2/cht/dosgolem`。來源檔只存本機 `original/`，研究產物存 `workplace/`，兩者均排除版控。下載來源與逐檔雜湊保存在 `original/source.json`。

使用固定 Go Docker image，依 AGENTS 的完整命令執行 `tools/run-probes.sh -steps 750000000 -mouse-file /project/tools/player-path.json`。腳本核對 dosgolem 基準、在研究副本套用 [平台修正](tools/dosgolem.patch)，跑回歸測試後讓原版 LE 自然執行。操作時間表、原生畫面與停止原因會寫入收據；完整驗證範圍見 CONTEXT。
