package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"fyne.io/systray"
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
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	versionInfo := detector.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}

	userSettings, errSettings := detector.NewSettings()
	if errSettings != nil {
		panic(fmt.Sprintf("Failed to initialize settings: %v\n", errSettings))
	}

	if errReadSettings := userSettings.ReadDefaultOrCreate(); errReadSettings != nil {
		panic(fmt.Sprintf("Failed to read settings: %v", errReadSettings))
	}

	userSettings.MustValidate()

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

	testLogPath, isTest := os.LookupEnv("TEST_CONSOLE_LOG")

	if isTest {
		body, errRead := os.ReadFile(testLogPath)
		if errRead != nil {
			logger.Fatal("Failed to load TEST_CONSOLE_LOG", zap.String("path", testLogPath), zap.Error(errRead))
		}

		lines := strings.Split(string(body), "\n")
		curLine := 0
		lineCount := len(lines)

		go func() {
			updateTicker := time.NewTicker(time.Millisecond * 100)

			for {
				<-updateTicker.C
				logChan <- lines[curLine]
				curLine++

				if curLine >= lineCount {
					curLine = 0
				}
			}
		}()
	}

	serviceGroup, serviceCtx := errgroup.WithContext(rootCtx)
	serviceGroup.Go(func() error {
		application.Start(serviceCtx)

		return nil
	})

	serviceGroup.Go(func() error {
		systray.Run(application.Systray.OnReady(stop), func() {
			if errShutdown := application.Shutdown(context.Background()); errShutdown != nil {
				logger.Error("Failed to shutdown cleanly")
			}
		})

		return nil
	})

	serviceGroup.Go(func() error {
		<-serviceCtx.Done()
		if errShutdown := application.Shutdown(context.Background()); errShutdown != nil {
			logger.Error("Failed to gracefully shutdown", zap.Error(errShutdown))
		}
		systray.Quit()
		return nil
	})

	if err := serviceGroup.Wait(); err != nil {
		logger.Error("Sad Goodbye", zap.Error(err))
		return
	}
	logger.Info("Happy Goodbye")
}
