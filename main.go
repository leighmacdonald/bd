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
	// Build info
	version = "master"
	commit  = "latest"
	date    = "n/a"
	builtBy = "src"
)

func main() {
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

	bd := detector.New(logger, userSettings, dataStore, versionInfo, fsCache, logReader, logChan)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	execGroup, grpCtx := errgroup.WithContext(rootCtx)
	execGroup.Go(func() error {
		bd.Start(rootCtx)
		return nil
	})

	bd.Start(grpCtx)

	execGroup.Go(func() error {
		<-grpCtx.Done()
		var err error
		return gerrors.Join(err, bd.Shutdown())
	})

	if errExit := execGroup.Wait(); errExit != nil {
		fmt.Println(errExit.Error())
	}
}
