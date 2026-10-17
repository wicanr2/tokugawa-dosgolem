# 戰國手繪 HD 素材

使用者於 2026-10-14 授權保存生成圖片，並於 2026-10-17 授權公開本儲存庫。這裡保存生成圖、由生成圖裁切的素材與完整提示詞，作為開發來源。適用條款見 [LICENSE](../../LICENSE)；原版資料、擷取原圖與音樂仍只留本機。

- [圖片與提示詞索引](index.json) 記錄 SHA-256、尺寸、來源位置、提示詞與裁切依據。
- [劇情圖](art/) 及 [家康肖像](A-0-hd-b-v1.png)。
- [81 張頭像總覽](kao/selected/contact-sheet.png)、[九頁頭像圖集](kao/)。
- [捲軸與紅金框架](scroll/)、[共用視窗邊框](window/)。
- [戰場地面、物件與單位草稿](terrain/)，物件／單位 PNG 保留透明背景。
- [地面圖集的材質與衍生圖](terrain/materials/)，保留兩版製作紀錄及完整提示詞。
- [十項非空單位的透明裁切](terrain/units/)，原版三個空白來源保持空白。
- [物件的透明裁切及審查](terrain/objects/)，獨立物件分項接入，城牆接縫草稿另列。
- [城牆校準與完整物件圖集](terrain/objects/walls/)，共用手繪材質對齊屋簷、石基及接縫，原版空白格保留。
- [晴天與陰天天氣圖](art/weather-generation.json)，第二版保留原版岩峰構圖。
- [開場騎兵與武將群像草稿](art/opening-generation.json)，保留原始生成圖，幾何與呈現待驗證。
- [戰鬥背景與開場徽記草稿](art/remaining-generation-v1.json)，保留原版參考定位、完整提示詞及未校準的原始生成圖。
- [校準背景、徽記與開場圖](art/static/)，五項有效美術區及原版緩衝區的幾何紀錄。
- [Family Soft 字標](art/static/logo/)，保留原始品牌字樣與位置，另存生成圖與完整提示詞。
- [動畫、標題與剩餘介面素材](art/remaining/)，20 項來源的逐格裁切、鏡像與留白核對；美術準備與執行器接入分開記錄。
- [OP-FULL 開場肖像](art/opening-full-generation-v2.json)與[完整提示詞](art/OP-FULL-hd-b-v2.prompt.txt)，第二版校正原生色盤下的藍白雲帶；[選定素材](art/opening-full/OP-FULL.OZM-hd-v2.png)與第一版均保存，來源、校準與遊戲內驗證分開記錄。
- [營帳場景](art/E-4-generation-v1.json)與[前景武將變體](art/B-6-generation-v1.json)，由已知來源區域製作，涵蓋 E-4 及三個相同 payload 的 B-6／B-7／C-10；未知八像素與正式來源資格另列，保持草稿。
- [部分來源故事素材](art/partial-story/)，沿用上述兩組生成圖，準備四項來源的校準圖及純 HD 總覽；原版未知八像素保留原圖，來源資格與正常路徑驗證分開記錄。
- [RAINEND 出版商標誌草稿](art/RAINEND-generation-v1.json)與[完整提示詞](art/RAINEND-hd-b-v1.prompt.txt)，保持彩色立體標誌與原構圖；PCX 繪圖端及底部原版文字呈現尚未接入。
- [風格比較](concept/style-comparison-v1.png)。

索引中的狀態區分已驗證、已製作與草稿。圖片存在不代表遊戲內整合完成。正式素材包格式及全遊戲覆蓋以專案 CONTEXT、WORKLIST 與 RESEARCH-LOG 為準。

保存工具為專案 tools/archive-hd-art.py，在固定 GUI Docker image 中執行。原版資料與擷取原圖仍使用本機 original/、workplace/；本目錄沒有遊戲資料包。
