package main

import (
	"context"
	gerrors "errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
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

	logFilePath := ""
	if userSettings.GetDebugLogEnabled() {
		logFilePath = userSettings.LogFilePath()
	}

	logger, errLogger := detector.NewLogger(logFilePath)
	if errLogger != nil {
		logger.Error("Failed to create logger", "err", errLogger)
	}

	dataStore := store.New(userSettings.DBPath(), logger)
	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		logger.Error("Failed to migrate database", "err", errMigrate)

		return
	}

	fsCache, cacheErr := detector.NewCache(logger, userSettings.ConfigRoot(), detector.DurationCacheTimeout)
	if cacheErr != nil {
		logger.Error("Failed to setup cache", "err", cacheErr)

		return
	}

	logChan := make(chan string)

	logReader, errLogReader := detector.NewLogReader(logger, filepath.Join(userSettings.GetTF2Dir(), "console.log"), logChan)
	if errLogReader != nil {
		logger.Error("Failed to create logreader", "err", errLogReader)

		return
	}

	application := detector.New(logger, userSettings, dataStore, versionInfo, fsCache, logReader, logChan)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	execGroup, grpCtx := errgroup.WithContext(rootCtx)
	execGroup.Go(func() error {
		application.Start(rootCtx)

		return nil
	})

	application.Start(grpCtx)

	execGroup.Go(func() error {
		<-grpCtx.Done()
		var err error

		return gerrors.Join(err, application.Shutdown())
	})

	if errExit := execGroup.Wait(); errExit != nil {
		logger.Error(errExit.Error())
	}
}
