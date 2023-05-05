package main

import (
	"context"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/ui"
	"github.com/leighmacdonald/bd/internal/web"
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
	if !(detector.Settings().GetHttpEnabled() || detector.Settings().GetGuiEnabled()) {
		panic("Must enable at least one of the gui or http packages")
	}
	if detector.Settings().GetHttpEnabled() {
		web.Setup()
		go web.Start(ctx)
	}
	if detector.Settings().GetGuiEnabled() {
		go detector.Start(ctx)
		ui.Setup(ctx, versionInfo)
		ui.Start(ctx) // *must* be called from main goroutine
	} else {
		detector.Start(ctx)
	}
}
