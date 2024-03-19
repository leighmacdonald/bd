package platform

import (
	"embed"
	"fmt"
)

//go:embed *.ico *.png
var content embed.FS

type iconName string

const (
	IconWindows iconName = "icon.ico"
	IconOther   iconName = "icon.png"
)

func readIcon(name iconName) []byte {
	data, errRead := content.ReadFile(string(name))
	if errRead != nil {
		panic(fmt.Sprintf("Cannot load embed asset: %v", errRead))
	}

	return data
}
