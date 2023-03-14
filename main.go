package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/cache"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/bd/internal/ui"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamweb"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
)

var (
	// Build info
	version string = "master"
	commit  string = "latest"
	date    string = "n/a"
	builtBy string = "src"
)

func mustCreateLogger(logFile string) *zap.Logger {
	loggingConfig := zap.NewProductionConfig()
	//loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if logFile != "" {
		if util.Exists(logFile) {
			os.Remove(logFile)
		}
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, logFile)
	}
	loggingConfig.Level.SetLevel(zap.DebugLevel)
	logger, errLogger := loggingConfig.Build()
	if errLogger != nil {
		fmt.Printf("Failed to create logger: %v\n", errLogger)
		os.Exit(1)
	}

	return logger
}

func main() {
	ctx := context.Background()
	versionInfo := model.Version{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
	settings, errSettings := model.NewSettings()
	if errSettings != nil {
		fmt.Printf("Failed to initialize settings: %v\n", errSettings)
		os.Exit(1)
	}
	if errReadSettings := settings.ReadDefaultOrCreate(); errReadSettings != nil {
		fmt.Printf("Failed to read settings: %v", errReadSettings)
	}
	logFilePath := ""
	if settings.DebugLogEnabled {
		logFilePath = settings.LogFilePath()
	}
	logger := mustCreateLogger(logFilePath)
	defer func() {
		if errSync := logger.Sync(); errSync != nil {
			fmt.Printf("Failed to sync log: %v\n", errSync)
		}
	}()
	if errTranslations := tr.Init(); errTranslations != nil {
		logger.Error("Failed to load translations", zap.Error(errTranslations))
	}
	if settings.GetAPIKey() != "" {
		if errAPIKey := steamweb.SetKey(settings.GetAPIKey()); errAPIKey != nil {
			logger.Error("Failed to set steam api key", zap.Error(errAPIKey))
		}
	}

	localRules := rules.NewRuleSchema()
	localPlayersList := rules.NewPlayerListSchema()

	// Try and load our existing custom players/rules
	if util.Exists(settings.LocalPlayerListPath()) {
		input, errInput := os.Open(settings.LocalPlayerListPath())
		if errInput != nil {
			logger.Error("Failed to open local player list", zap.Error(errInput))
		} else {
			if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
				logger.Error("Failed to parse local player list", zap.Error(errRead))
			} else {
				logger.Debug("Loaded local player list", zap.Int("count", len(localPlayersList.Players)))
			}
			util.LogClose(logger, input)
		}
	}
	if util.Exists(settings.LocalRulesListPath()) {
		input, errInput := os.Open(settings.LocalRulesListPath())
		if errInput != nil {
			logger.Error("Failed to open local rules list", zap.Error(errInput))
		} else {
			if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
				logger.Error("Failed to parse local rules list", zap.Error(errRead))
			} else {
				logger.Debug("Loaded local rules list", zap.Int("count", len(localRules.Rules)))
			}
			util.LogClose(logger, input)
		}
	}
	engine, ruleEngineErr := rules.New(&localRules, &localPlayersList)
	if ruleEngineErr != nil {
		logger.Panic("Failed to setup rules engine", zap.Error(ruleEngineErr))
	}

	dataStore := store.New(settings.DBPath(), logger)
	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		logger.Panic("Failed to migrate database", zap.Error(errMigrate))
	}
	fileSystemCache := cache.New(logger, settings.ConfigRoot(), model.DurationCacheTimeout)

	bd := detector.New(logger, settings, dataStore, engine, fileSystemCache)

	gui := ui.New(ctx, logger, &bd, settings, versionInfo)
	gui.Start(ctx)
}
