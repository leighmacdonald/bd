package main

import (
	"context"
	gerrors "errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/web"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	execGroup, grpCtx := errgroup.WithContext(rootCtx)
	execGroup.Go(func() error {
		detector.Start(rootCtx)
		return nil
	})
	if userSettings.GetHTTPEnabled() {
		execGroup.Go(func() error {
			web.Init(detector.Logger(), false)
			return web.Start(grpCtx, userSettings.HTTPListenAddr)
		})
		execGroup.Go(func() error {
			<-grpCtx.Done()
			var err error
			return gerrors.Join(err, web.Stop(), detector.Shutdown())
		})
	}

	if errExit := execGroup.Wait(); errExit != nil {
		fmt.Println(errExit.Error())
	}
}
