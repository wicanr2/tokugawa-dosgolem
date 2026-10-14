package main

import (
	"image"
	"image/color"
)

const battleFrameAsset = "battle-frame-hd-b-v2.png"
const battleFrameAssetSHA = "3de79e3eebe01aa970e45b8bf377b6e381bec14c9b4d9f569994aaecbe78e435"
const battleFrameMaskSHA = "b0fc8f0075d0259e59f665f44527487d93472c05cf64201104a4cc385d065eaf"

var battleFrameCrestRect = image.Rect(100, 70, 360, 320)
var battleFrameWeatherRect = image.Rect(464, 291, 608, 365)
var battleFrameMinimapRect = image.Rect(434, 16, 632, 214)

var battleFrameNativePalette = [16]color.NRGBA{
	{0, 0, 0, 255}, {32, 32, 32, 255}, {190, 69, 44, 255}, {81, 20, 20, 255},
	{251, 215, 20, 255}, {44, 81, 0, 255}, {20, 20, 227, 255}, {227, 227, 56, 255},
	{0, 0, 0, 255}, {81, 81, 81, 255}, {227, 190, 93, 255}, {215, 0, 0, 255},
	{251, 81, 81, 255}, {44, 215, 44, 255}, {32, 227, 251, 255}, {251, 251, 251, 255},
}
