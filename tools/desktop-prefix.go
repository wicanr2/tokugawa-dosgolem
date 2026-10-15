//go:build desktop

package main

import (
	"os"
	"path/filepath"
)

func (g *desktopGame) loadPrefixStories() {
	g.prefixPortraits = map[string]hdPrefixPair{}
	info := map[string]any{}
	for name, profile := range hdPrefixProfiles {
		record := map[string]any{"source_scope": "47992 known pixels; eight native pixels preserved"}
		source, err := os.ReadFile(profile.Source)
		if err != nil {
			record["source_error"] = err.Error()
		} else {
			record["source_sha256"] = prefixStorySHA(source)
		}
		path := filepath.Join(*desktopPortraitDir, profile.Asset)
		var asset desktopPortrait
		if data, e := os.ReadFile(path); e != nil {
			record["hd_error"] = e.Error()
		} else {
			record["hd_sha256"] = prefixStorySHA(data)
			if prefixStorySHA(data) == profile.AssetSHA {
				asset.Asset, e = previewPNG(path)
				if e != nil {
					record["hd_error"] = e.Error()
				}
			} else {
				record["hd_error"] = "asset SHA mismatch"
			}
		}
		pair := newHDPrefixPair(profile, source, asset.Asset)
		record["validation"] = pair.Reason
		info[name] = record
		g.prefixPortraits[name] = pair
	}
	g.assetInfo["story_prefix_portraits"] = info
}
