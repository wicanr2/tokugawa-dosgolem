package main

type hdGraphicsProfile struct {
	Source                      graphicsSourceProfile
	IndicesSHA, Asset, AssetSHA string
}

var hdGraphicsProfiles = []hdGraphicsProfile{
	{Source: graphicsSourceProfile{Name: "WEATHER1.OZM", Width: 144, Height: 74, Planes: []string{
		"a8d511ceb8b4688656ec7944f210375042bdbec816a2082d9c34cefcd05ca5fb",
		"ae66b075320d6736912f2de9a4cfb4cade333a88c10590702ae961d301eac06b",
		"b222d568eb6c6903028673fc3c04cf10028831180a79b6ee956996d5ba1bd4d0",
		"fe54a1f03c53452798651eed75efc4f28c4d231bb61235991e825c831242d0b8"}},
		IndicesSHA: "ad397c7ae15a08571b43e66fed4fc9fc9236c37909b374fe69ef1e51771066c0", Asset: "WEATHER1-hd-b-v2.png", AssetSHA: "44ea325fa5e95ab3d8b686a9cf895c3347a47d99f081c58c7406257c49bd014b"},
	{Source: graphicsSourceProfile{Name: "WEATHER2.OZM", Width: 144, Height: 74, Planes: []string{
		"b615455619ce5b3fdf942e28a5ee7a927fd2a9f7c77357872f6a213cca66ccb9",
		"13ded0b5fa5fedf1ebc943f3a934820a994718a1f36acb63b081533c35127451",
		"5c07fe2eac5734a4e0cb78d1c2e6d275312147ec243187b3f4593e49a2ec18ad",
		"67ebc7d139ad34f1d7f330110616bc7c27c84cc4ba475ac9502b1803826ac069"}},
		IndicesSHA: "61220f0bcfe5c2c85d09f3a451deeef33c2363109d8de992d99d19951cf84696", Asset: "WEATHER2-hd-b-v2.png", AssetSHA: "b5a024f4ae985a46f6c8e8686ae286a3583394b59e48fa08d821b2f1a146562a"},
}

var hdGraphicsSources = func() []graphicsSourceProfile {
	var result []graphicsSourceProfile
	for _, p := range hdGraphicsProfiles {
		result = append(result, p.Source)
	}
	return result
}()
