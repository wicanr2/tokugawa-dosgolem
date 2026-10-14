package main

import "image"

// 來源索引保留原名，不以推測的兵種或武將命名。
type hdTerrainUnitProfile struct {
	Asset, AssetSHA    string
	SourceBounds, Crop image.Rectangle
}

var hdTerrainUnitProfiles = map[int]hdTerrainUnitProfile{
	1: {"unit-001-hd-b-v1.png", "a8c37f61ef22f645a58cb824758ab34620e535a2588f0fce5e71490cf3dd3a72", image.Rect(2, 0, 47, 48), image.Rect(265, 39, 1098, 1191)},
	3: {"unit-003-hd-b-v1.png", "6c9a0ef3d9a3db01bec673fb37d7f8fca4a2f2a08cf6a14d81ecd101ec867654", image.Rect(0, 0, 41, 48), image.Rect(117, 90, 984, 1159)},
}
