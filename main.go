package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// Build info.
	version = "master" //nolint:gochecknoglobals
	commit  = "latest" //nolint:gochecknoglobals
	date    = "n/a"    //nolint:gochecknoglobals
	builtBy = "src"    //nolint:gochecknoglobals
)

func main() {
	versionInfo := detector.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}

	userSettings, errSettings := detector.NewSettings()
	if errSettings != nil {
		panic(fmt.Sprintf("Failed to initialize settings: %v\n", errSettings))
	}

	if errReadSettings := userSettings.ReadDefaultOrCreate(); errReadSettings != nil {
		panic(fmt.Sprintf("Failed to read settings: %v", errReadSettings))
	}

	userSettings.MustValidate()

	switch userSettings.RunMode {
	case detector.ModeProd:
		gin.SetMode(gin.ReleaseMode)
	case detector.ModeTest:
		gin.SetMode(gin.TestMode)
	case detector.ModeDebug:
		gin.SetMode(gin.DebugMode)
	}

	logger := detector.MustCreateLogger(userSettings)

	logger.Info("Starting...")

	dataStore := store.New(userSettings.DBPath(), logger)
	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		logger.Error("Failed to migrate database", zap.Error(errMigrate))

		return
	}

	fsCache, cacheErr := detector.NewCache(logger, userSettings.ConfigRoot(), detector.DurationCacheTimeout)
	if cacheErr != nil {
		logger.Error("Failed to setup cache", zap.Error(cacheErr))

		return
	}

	logChan := make(chan string)

	logReader, errLogReader := detector.NewLogReader(logger, filepath.Join(userSettings.GetTF2Dir(), "console.log"), logChan)
	if errLogReader != nil {
		logger.Error("Failed to create logreader", zap.Error(errLogReader))

		return
	}

	application := detector.New(logger, userSettings, dataStore, versionInfo, fsCache, logReader, logChan)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application.Start(rootCtx)

	systray.Run(application.Systray.OnReady, func() {
		if errShutdown := application.Shutdown(rootCtx); errShutdown != nil {
			logger.Error("Failed to shutdown cleanly")
		}
		logger.Info("Bye")
	})
}
