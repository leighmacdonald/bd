package main

import (
	"context"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/ui"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// Build info
	version = "master"
	commit  = "latest"
	date    = "n/a"
	builtBy = "src"
)

func main() {
	ctx := context.Background()
	versionInfo := model.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
	detector.Setup(versionInfo)
	go detector.Start(ctx)
	ui.Setup(ctx, versionInfo)
	ui.Start(ctx)
}
