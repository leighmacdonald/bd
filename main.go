package main

import (
	"context"
	"fmt"
	"fyne.io/systray"
	"github.com/leighmacdonald/bd/discord/client"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"golang.org/x/sync/errgroup"
	"log/slog"
	_ "modernc.org/sqlite"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	// Build info embedded by goreleaser.
	version = "master" //nolint:gochecknoglobals
	commit  = "latest" //nolint:gochecknoglobals
	date    = "n/a"    //nolint:gochecknoglobals
	builtBy = "src"    //nolint:gochecknoglobals
)

func testLogFeeder(logChan chan string) {
	if testLogPath, isTest := os.LookupEnv("TEST_CONSOLE_LOG"); isTest {
		body, errRead := os.ReadFile(testLogPath)
		if errRead != nil {
			slog.Error("Failed to load TEST_CONSOLE_LOG", slog.String("path", testLogPath), errAttr(errRead))

			return
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
}

func run() int {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupTime := time.Now()

	versionInfo := Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
	plat := platform.New()

	userSettings, errSettings := loadAndValidateSettings()
	if errSettings != nil {
		slog.Error("Failed to load settings", errAttr(errSettings))

		return 1
	}

	logCloser := MustCreateLogger(userSettings)
	defer logCloser()

	slog.Info("Starting BD",
		slog.String("version", versionInfo.Version),
		slog.String("date", versionInfo.Date),
		slog.String("commit", versionInfo.Commit),
		slog.String("via", versionInfo.BuiltBy))

	db, dbCloser, errDb := store.CreateDb(userSettings.DBPath())
	if errDb != nil {
		slog.Error("failed to create database", errAttr(errDb))
		return 1
	}
	defer dbCloser()

	fsCache, cacheErr := NewCache(userSettings.ConfigRoot(), DurationCacheTimeout)
	if cacheErr != nil {
		slog.Error("Failed to setup cache", errAttr(cacheErr))
		return 1
	}

	logChan := make(chan string)

	logReader, errLogReader := newLogReader(filepath.Join(userSettings.TF2Dir, "console.log"), logChan, true)
	if errLogReader != nil {
		slog.Error("Failed to create logreader", errAttr(errLogReader))
		return 1
	}

	dataSource, errDataSource := NewDataSource(userSettings)
	if errDataSource != nil {
		slog.Error("failed to create data source", errAttr(errDataSource))
		return 1
	}

	profileUpdateQueue := make(chan steamid.SID64)
	kickRequestChan := make(chan kickRequest)
	g15 := newG15Parser()
	discordPresence := client.New()

	playerStates := newPlayerState()
	state := newStateHandler(userSettings, playerStates)

	rules := createRulesEngine(userSettings)
	eventChan := make(chan LogEvent)
	parser := NewLogParser(logChan, eventChan)
	web, errWeb := newWebServer(application)

	if errWeb != nil {
		panic(errWeb)
	}

	go testLogFeeder(logChan)

	tray := NewSystray(
		plat.Icon(),
		func() {
			if errOpen := plat.OpenURL(fmt.Sprintf("http://%s/", settings.HTTPListenAddr)); errOpen != nil {
				slog.Error("Failed to open browser", errAttr(errOpen))
			}
		}, func() {
			go application.LaunchGameAndWait()
		},
	)

	serviceGroup, serviceCtx := errgroup.WithContext(rootCtx)
	serviceGroup.Go(func() error {
		application.Start(serviceCtx)
		return nil
	})

	serviceGroup.Go(func() error {
		systray.Run(tray.OnReady(stop), func() {
			if errShutdown := application.Shutdown(context.Background()); errShutdown != nil {
				slog.Error("Failed to shutdown cleanly")
			}
		})
		return nil
	})

	serviceGroup.Go(func() error {
		<-serviceCtx.Done()

		if errShutdown := application.Shutdown(context.Background()); errShutdown != nil { //nolint:contextcheck
			slog.Error("Failed to gracefully shutdown", errAttr(errShutdown))
		}

		systray.Quit()

		return nil
	})

	if err := serviceGroup.Wait(); err != nil {
		slog.Error("Sad Goodbye", errAttr(err))

		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
