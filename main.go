package main

import (
	"context"
	"encoding/json"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/steamweb"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"log"
	"os"
)

var (
	// Build info
	version string = "master"
	commit  string = "latest"
	date    string = "n/a"
	builtBy string = "src"
)

func main() {
	ctx := context.Background()
	settings, errSettings := model.NewSettings()
	if errSettings != nil {
		log.Panicf("Failed to initialize settings: %v", errSettings)
	}
	localRules := rules.NewRuleSchema()
	localPlayersList := rules.NewPlayerListSchema()
	if errReadSettings := settings.ReadDefaultOrCreate(); errReadSettings != nil {
		log.Println(errReadSettings)
	}
	// Try and load our existing custom players/rules
	if exists(settings.LocalPlayerListPath()) {
		input, errInput := os.Open(settings.LocalPlayerListPath())
		if errInput != nil {
			log.Printf("Failed to open local player list\n")
		} else {
			if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
				log.Printf("Failed to parse local player list: %v\n", errRead)
			}
			logClose(input)
		}
	}
	if exists(settings.LocalRulesListPath()) {
		input, errInput := os.Open(settings.LocalRulesListPath())
		if errInput != nil {
			log.Printf("Failed to open local rules list\n")
		} else {
			if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
				log.Printf("Failed to parse local rules list: %v\n", errRead)
			}
			logClose(input)
		}
	}
	engine, ruleEngineErr := rules.NewEngine(&localRules, &localPlayersList)
	if ruleEngineErr != nil {
		log.Panicf("Failed to setup rules engine: %v\n", ruleEngineErr)
	}
	if settings.ApiKey != "" {
		if errAPIKey := steamweb.SetKey(settings.ApiKey); errAPIKey != nil {
			log.Printf("Failed to set steam api key: %v\n", errAPIKey)
		}
	}
	store := newSqliteStore(settings.DBPath())
	if errMigrate := store.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		log.Printf("Failed to migrate database: %v\n", errMigrate)
		os.Exit(1)
	}
	defer logClose(store)
	cache := newFsCache(settings.ConfigRoot(), model.DurationCacheTimeout)
	bd := New(settings, store, engine, cache)
	defer bd.Shutdown()
	gui := ui.New(ctx, settings, bd.onMark, store.FetchNames, store.FetchMessages, bd.launchGameAndWait, bd.callVote, bd.onWhitelist)
	bd.AttachGui(gui)
	go bd.start(ctx)
	gui.Start()
}
