package main

import (
	"context"
	"encoding/json"
	"fmt"
	"fyne.io/systray"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
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

const (
	profileAgeLimit = time.Hour * 24
)

func createRulesEngine(sm *settingsManager) *rules.Engine {
	rulesEngine := rules.New()

	if sm.Settings().RunMode != ModeTest { //nolint:nestif
		// Try and load our existing custom players
		if platform.Exists(sm.LocalPlayerListPath()) {
			input, errInput := os.Open(sm.LocalPlayerListPath())
			if errInput != nil {
				slog.Error("Failed to open local player list", errAttr(errInput))
			} else {
				var localPlayersList rules.PlayerListSchema
				if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
					slog.Error("Failed to parse local player list", errAttr(errRead))
				} else {
					count, errPlayerImport := rulesEngine.ImportPlayers(&localPlayersList)
					if errPlayerImport != nil {
						slog.Error("Failed to import local player list", errAttr(errPlayerImport))
					} else {
						slog.Info("Loaded local player list", slog.Int("count", count))
					}
				}

				LogClose(input)
			}
		}

		// Try and load our existing custom rules
		if platform.Exists(sm.LocalRulesListPath()) {
			input, errInput := os.Open(sm.LocalRulesListPath())
			if errInput != nil {
				slog.Error("Failed to open local rules list", errAttr(errInput))
			} else {
				var localRules rules.RuleSchema
				if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
					slog.Error("Failed to parse local rules list", errAttr(errRead))
				} else {
					count, errRulesImport := rulesEngine.ImportRules(&localRules)
					if errRulesImport != nil {
						slog.Error("Failed to import local rules list", errAttr(errRulesImport))
					}

					slog.Debug("Loaded local rules list", slog.Int("count", count))
				}

				LogClose(input)
			}
		}
	}

	return rulesEngine
}

// openApplicationPage launches the http frontend using the platform specific browser launcher function.
func openApplicationPage(plat platform.Platform, addr string) {
	appURL := fmt.Sprintf("http://%s", addr)
	if errOpen := plat.OpenURL(appURL); errOpen != nil {
		slog.Error("Failed to open URL", slog.String("url", appURL), errAttr(errOpen))
	}
}

func run() int {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	versionInfo := Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
	plat := platform.New()
	settingsMgr := newSettingsManager(plat)

	if errSetup := settingsMgr.setup(); errSetup != nil {
		slog.Error("Failed to create settings directories", errAttr(errSetup))

		return 1
	}

	if errSettings := settingsMgr.validateAndLoad(); errSettings != nil {
		slog.Error("Failed to load settings", errAttr(errSettings))

		return 1
	}

	settings := settingsMgr.Settings()

	logCloser := MustCreateLogger(settingsMgr)
	defer logCloser()

	slog.Info("Starting BD",
		slog.String("version", versionInfo.Version),
		slog.String("date", versionInfo.Date),
		slog.String("commit", versionInfo.Commit),
		slog.String("via", versionInfo.BuiltBy))

	db, dbCloser, errDb := store.CreateDb(settingsMgr.DBPath())
	if errDb != nil {
		slog.Error("failed to create database", errAttr(errDb))
		return 1
	}
	defer dbCloser()

	//fsCache, cacheErr := NewCache(settingsMgr.ConfigRoot(), DurationCacheTimeout)
	//if cacheErr != nil {
	//	slog.Error("Failed to setup cache", errAttr(cacheErr))
	//	return 1
	//}

	logChan := make(chan string)

	logReader, errLogReader := newLogReader(filepath.Join(settings.TF2Dir, "console.log"), logChan, true)
	if errLogReader != nil {
		slog.Error("Failed to create logreader", errAttr(errLogReader))
		return 1
	}
	defer logReader.tail.Cleanup()

	go logReader.start(rootCtx)

	rcon := newRconConnection(settings.Rcon.String(), settings.Rcon.Password)

	playerStates := newPlayerState()
	state := newGameState(db, settingsMgr, playerStates, rcon)

	dataSource, errDataSource := newDataSource(settings)
	if errDataSource != nil {
		slog.Error("failed to create data source", errAttr(errDataSource))
		return 1
	}

	updater := newProfileUpdater(db, dataSource, state)
	go updater.start(rootCtx)

	announcer := newAnnounceHandler(settingsMgr, rcon, state)

	kickRequestChan := make(chan kickRequest)

	discordPresence := newDiscordState(state, settingsMgr)
	go discordPresence.start(rootCtx)

	re := createRulesEngine(settingsMgr)
	eventChan := make(chan LogEvent)
	parser := NewLogParser(logChan, eventChan)
	process := newProcessState(plat, rcon)

	httpServer, errWeb := newHTTPServer(rootCtx, settings.HTTPListenAddr, db, state, process, settingsMgr, re)

	if errWeb != nil {
		slog.Error("Failed to initialize http server", errAttr(errWeb))
	}

	go testLogFeeder(logChan)

	tray := NewSystray(
		plat.Icon(),
		func() {
			if errOpen := plat.OpenURL(fmt.Sprintf("http://%s/", settings.HTTPListenAddr)); errOpen != nil {
				slog.Error("Failed to open browser", errAttr(errOpen))
			}
		}, func() {
			go process.LaunchGameAndWait(settings)
		},
	)

	defer systray.Quit()

	shutdown := func() {
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()

		if errShutdown := httpServer.Shutdown(timeout); errShutdown != nil {
			slog.Error("Failed to shutdown cleanly", errAttr(errShutdown))
		}
	}

	go func() {
		time.Sleep(time.Second * 1)

		if settings.RunMode == ModeRelease {
			openApplicationPage()
		}
	}()

	serviceGroup, serviceCtx := errgroup.WithContext(rootCtx)
	serviceGroup.Go(func() error {
		shutdown()

		return nil
	})

	serviceGroup.Go(func() error {
		systray.Run(tray.OnReady(stop), func() {
			shutdown()
		})
		return nil
	})

	serviceGroup.Go(func() error {
		<-serviceCtx.Done()

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
