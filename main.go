package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	_ "modernc.org/sqlite"
)

var (
	// Build info embedded at build time.
	version = "master" //nolint:gochecknoglobals
	commit  = "latest" //nolint:gochecknoglobals
	date    = "n/a"    //nolint:gochecknoglobals
)

func createRulesEngine(settings userSettings) *rules.Engine {
	rulesEngine := rules.New()

	if settings.RunMode != ModeTest { //nolint:nestif
		// Try and load our existing custom players
		if platform.Exists(settings.LocalPlayerListPath()) {
			input, errInput := os.Open(settings.LocalPlayerListPath())
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
		if platform.Exists(settings.LocalRulesListPath()) {
			input, errInput := os.Open(settings.LocalRulesListPath())
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
func openApplicationPage(plat platform.Platform, appURL string) {
	if errOpen := plat.OpenURL(appURL); errOpen != nil {
		slog.Error("Failed to open URL", slog.String("url", appURL), errAttr(errOpen))
	}
}

func isErrorAddressAlreadyInUse(err error) bool {
	var errOpError *net.OpError
	ok := errors.As(err, &errOpError)
	if !ok {
		return false
	}

	var errSyscallError *os.SyscallError
	ok = errors.As(errOpError.Err, &errSyscallError)
	if !ok {
		return false
	}

	var errErrno syscall.Errno
	ok = errors.As(errSyscallError.Err, &errErrno)
	if !ok {
		return false
	}

	if errors.Is(errErrno, syscall.EADDRINUSE) {
		return true
	}

	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}

	return false
}

var errConfigRootCreate = errors.New("failed to create config root")

func ensureConfigPath() (string, error) {
	configPath := configdir.LocalConfig("bd")

	if errMakePath := os.MkdirAll(configPath, 0o700); errMakePath != nil {
		return "", errors.Join(errMakePath, errConfigRootCreate)
	}

	return configPath, nil
}

func run() int {
	versionInfo := Version{Version: version, Commit: commit, Date: date}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	plat := platform.New()

	slog.Info("Starting", slog.String("ver", versionInfo.Version),
		slog.String("date", versionInfo.Date), slog.String("commit", versionInfo.Commit))

	configRoot, errConfigPath := ensureConfigPath()
	if errConfigPath != nil {
		slog.Error("Failed to setup config root", errAttr(errConfigPath))
		return 1
	}

	db, dbCloser, errDB := store.CreateDB(path.Join(configRoot, "bd.sqlite?cache=shared"))
	if errDB != nil {
		slog.Error("failed to create database", errAttr(errDB))
		return 1
	}
	defer dbCloser()

	settingsMgr := newSettingsManager(configRoot, db, plat)

	settings, errSettings := settingsMgr.settings(ctx)
	if errSettings != nil {
		slog.Error("Failed to read settings from database", errAttr(errSettings))
		return 1
	}

	logCloser := MustCreateLogger(settings, configRoot)
	defer logCloser()

	rcon := newRconConnection(settings.Rcon.String(), settings.Rcon.Password)
	state := newGameState(db, settingsMgr, newPlayerStates(), rcon, db)
	parser := newLogParser()
	broadcaster := newEventBroadcaster()

	var logSrc backgroundService

	ingest, errLogReader := newLogIngest(filepath.Join(settings.Tf2Dir, "console.log"), parser, true, broadcaster)
	if errLogReader != nil {
		slog.Error("Failed to create log startEventEmitter", errAttr(errLogReader))
		return 1
	}

	go testLogFeeder(ctx, ingest)

	logSrc = ingest

	chat := newChatRecorder(db, broadcaster)

	broadcaster.registerConsumer(state.eventChan, EvtAny)

	dataSource, errDataSource := newDataSource(settings)
	if errDataSource != nil {
		slog.Error("failed to create data source", errAttr(errDataSource))
		return 1
	}

	re := createRulesEngine(settings)

	cache, cacheErr := NewCache(configRoot, DurationCacheTimeout)
	if cacheErr != nil {
		slog.Error("Failed to set up cache", errAttr(cacheErr))
		return 1
	}

	lm := newListManager(cache, re, settingsMgr)
	updater := newPlayerDataLoader(db, dataSource, settingsMgr, re, state.profileUpdateQueue, state.playerDataChan)
	discordPresence := newDiscordState(state, settingsMgr)
	processHandler := newProcessState(plat, rcon, settingsMgr)
	statusHandler := newStatusUpdater(rcon, processHandler, state, time.Second*2)
	bigBrotherHandler := newOverwatch(settingsMgr, rcon, state)

	mux, errRoutes := createHandlers(ctx, db, state, processHandler, settingsMgr, re, rcon)
	if errRoutes != nil {
		slog.Error("failed to create http handlers", errAttr(errRoutes))

		return 1
	}

	httpServer := newHTTPServer(ctx, settings.HttpListenAddr, mux)

	// Start all the background workers
	for _, svc := range []backgroundService{discordPresence, chat, logSrc, updater, statusHandler, &bigBrotherHandler, processHandler, state} {
		go svc.start(ctx)
	}

	go func() {
		if err := lm.start(ctx); err != nil {
			slog.Error("Failed to start list manager", errAttr(err))
		}
	}()

	go func() {
		time.Sleep(time.Second * 3)

		if settings.AutoLaunchGame {
			go processHandler.launchGame(settings)
		}

		if settings.RunMode == ModeRelease {
			openApplicationPage(plat, settings.AppURL())
		}
	}()

	go func() {
		if errServe := httpServer.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
			if isErrorAddressAlreadyInUse(errServe) {
				// Exit early on bind error.
				slog.Error("Listen address already in use", errAttr(errServe))
				stop()

				return
			}

			slog.Error("Unhandled error trying to shutdown http service", errAttr(errServe))
		}
	}()

	if settings.SystrayEnabled {
		slog.Debug("Using systray")

		tray := newAppSystray(plat, settingsMgr, processHandler)

		systray.Run(tray.OnReady(ctx), func() {
			stop()
		})
	}

	<-ctx.Done()

	timeout, cancelHTTP := context.WithTimeout(context.Background(), time.Second*15)
	defer cancelHTTP()

	if errShutdown := httpServer.Shutdown(timeout); errShutdown != nil {
		slog.Error("Failed to shutdown cleanly", errAttr(errShutdown))
	} else {
		slog.Debug("HTTP Service shutdown successfully")
	}

	return 0
}

func main() {
	os.Exit(run())
}
