package frontend

import (
	"embed"
	"errors"
	"fmt"
)

//go:embed *.ico *.png
var content embed.FS

var ErrStaticPath = errors.New("failed to setup static paths")

type Name string

const (
	IconWindows Name = "icon.ico"
	IconOther   Name = "icon.png"
)

func Read(name Name) []byte {
	data, errRead := content.ReadFile(string(name))
	if errRead != nil {
		panic(fmt.Sprintf("Cannot load embed asset: %v", errRead))
	}

	return data
}
