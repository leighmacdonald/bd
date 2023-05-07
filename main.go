package main

import (
	"context"
	"github.com/leighmacdonald/bd/internal/detector"
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
	versionInfo := detector.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
	detector.Setup(versionInfo, false)
	userSettings := detector.Settings()
	if !(userSettings.GetHttpEnabled() || userSettings.GetGuiEnabled()) {
		panic("Must enable at least one of the gui or http packages")
	}
	if userSettings.GetHttpEnabled() {
		web.Setup(detector.Logger(), false)
		go web.Start(ctx)
	}
	if userSettings.GetGuiEnabled() {
		go detector.Start(ctx)
		ui.Setup(ctx, detector.Logger(), versionInfo)
		ui.Start(ctx) // *must* be called from main goroutine
	} else {
		detector.Start(ctx)
	}
}
