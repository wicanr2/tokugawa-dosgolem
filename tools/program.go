package main

import "fmt"

type programProfile struct {
	File, SHA string
	Publisher bool
}

func selectProgram(name string) (programProfile, error) {
	switch name {
	case "game":
		return programProfile{File: "DX5.EXE", SHA: "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"}, nil
	case "mark":
		return programProfile{File: "MARK.EXE.BAK", SHA: "1ea6dc335b42b39542ba8dec35c8ffd103e4b495d30700c4100e60dbcbecaa60", Publisher: true}, nil
	default:
		return programProfile{}, fmt.Errorf("program must be game or mark")
	}
}
