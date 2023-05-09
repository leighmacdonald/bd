package main

import (
	"context"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/ui"
	"github.com/leighmacdonald/bd/internal/web"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
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

	userSettings, errSettings := detector.NewSettings()
	if errSettings != nil {
		fmt.Printf("Failed to initialize settings: %v\n", errSettings)
		os.Exit(1)
	}
	if errReadSettings := userSettings.ReadDefaultOrCreate(); errReadSettings != nil {
		fmt.Printf("Failed to read settings: %v", errReadSettings)
	}

	userSettings.MustValidate()

	dbPath := userSettings.DBPath()
	logFilePath := ""
	if userSettings.GetDebugLogEnabled() {
		logFilePath = userSettings.LogFilePath()
	}
	rootLogger := detector.MustCreateLogger(logFilePath)

	dataStore := store.New(dbPath, detector.Logger())
	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		rootLogger.Panic("Failed to migrate database", zap.Error(errMigrate))
	}

	detector.Init(versionInfo, userSettings, rootLogger, dataStore, false)

	if userSettings.GetHttpEnabled() {
		web.Init(detector.Logger(), false)
		go web.Start(ctx)
	}
	if userSettings.GetGuiEnabled() {
		go detector.Start(ctx)
		ui.Init(ctx, detector.Logger(), versionInfo)
		ui.Start(ctx) // *must* be called from main goroutine
	} else {
		detector.Start(ctx)
	}
}
