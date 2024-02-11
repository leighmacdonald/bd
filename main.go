package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/golang-migrate/migrate/v4"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"
)

var (
	// Build info embedded by goreleaser.
	version = "master" //nolint:gochecknoglobals
	commit  = "latest" //nolint:gochecknoglobals
	date    = "n/a"    //nolint:gochecknoglobals
	builtBy = "src"    //nolint:gochecknoglobals
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	versionInfo := Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}

	userSettings, errSettings := NewSettings()
	if errSettings != nil {
		panic(fmt.Sprintf("Failed to initialize settings: %v\n", errSettings))
	}

	if errReadSettings := userSettings.ReadDefaultOrCreate(); errReadSettings != nil {
		panic(fmt.Sprintf("Failed to read settings: %v", errReadSettings))
	}

	if errValidate := userSettings.Validate(); errValidate != nil {
		panic(fmt.Sprintf("Failed to validate settings: %v", errValidate))
	}

	logCloser := MustCreateLogger(userSettings)
	defer logCloser()

	logger := slog.Default()

	slog.Info("Starting BD",
		slog.String("version", versionInfo.Version),
		slog.String("date", versionInfo.Date),
		slog.String("commit", versionInfo.Commit),
		slog.String("via", versionInfo.BuiltBy))

	dataStore := NewStore(userSettings.DBPath())
	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		slog.Error("Failed to migrate database", errAttr(errMigrate))

		return
	}

	fsCache, cacheErr := NewCache(userSettings.ConfigRoot(), DurationCacheTimeout)
	if cacheErr != nil {
		slog.Error("Failed to setup cache", errAttr(cacheErr))

		return
	}

	logChan := make(chan string)

	logReader, errLogReader := newLogReader(filepath.Join(userSettings.TF2Dir, "console.log"), logChan, true)
	if errLogReader != nil {
		logger.Error("Failed to create logreader", errAttr(errLogReader))

		return
	}

	var (
		dataSource DataSource
		errDS      error
	)

	if userSettings.BdAPIEnabled {
		dataSource, errDS = NewAPIDataSource(userSettings.BdAPIAddress)
	} else {
		dataSource, errDS = NewLocalDataSource(userSettings.APIKey)
	}

	if errDS != nil {
		logger.Error("Failed to initialize data source", errAttr(errDS))
		os.Exit(1)
	}

	application := NewDetector(userSettings, dataStore, versionInfo, fsCache, logReader, logChan, dataSource)

	testLogPath, isTest := os.LookupEnv("TEST_CONSOLE_LOG")

	if isTest {
		body, errRead := os.ReadFile(testLogPath)
		if errRead != nil {
			logger.Error("Failed to load TEST_CONSOLE_LOG", slog.String("path", testLogPath), errAttr(errRead))
			os.Exit(2)
		}

		lines := strings.Split(string(body), "\n")
		curLine := 0
		lineCount := len(lines)

		go func() {
			updateTicker := time.NewTicker(time.Millisecond * 10)

			for {
				<-updateTicker.C
				logChan <- strings.Trim(lines[curLine], "\r")
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

		if errShutdown := application.Shutdown(context.Background()); errShutdown != nil { //nolint:contextcheck
			logger.Error("Failed to gracefully shutdown", errAttr(errShutdown))
		}

		systray.Quit()

		return nil
	})

	if err := serviceGroup.Wait(); err != nil {
		logger.Error("Sad Goodbye", errAttr(err))

		return
	}

	logger.Info("Happy Goodbye")
}
