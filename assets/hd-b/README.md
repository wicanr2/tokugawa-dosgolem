# 戰國手繪 HD 素材

使用者於 2026-10-14 授權將目前生成的圖片保存於本私人儲存庫。這裡保存生成圖、由生成圖裁切的素材與完整提示詞，作為開發來源。公開發行與素材散布權另行確認。

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
- [風格比較](concept/style-comparison-v1.png)。

索引中的狀態區分已驗證、已製作與草稿。圖片存在不代表遊戲內整合完成。正式素材包格式及全遊戲覆蓋以專案 CONTEXT、WORKLIST 與 RESEARCH-LOG 為準。

保存工具為專案 tools/archive-hd-art.py，在固定 GUI Docker image 中執行。原版資料與擷取原圖仍使用本機 original/、workplace/；本目錄沒有遊戲資料包。
