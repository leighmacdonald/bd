package main

import (
	"context"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/rules"
	_ "github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/steamweb"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	settings := model.NewSettings()
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
			if errRead := rules.ParsePlayerSchema(input, &localPlayersList); errRead != nil {
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
			if errRead := rules.ParseRulesList(input, &localRules); errRead != nil {
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
	bd := New(&settings, store, engine)
	defer bd.Shutdown()
	gui := ui.New(&settings)
	bd.AttachGui(ctx, gui)
	go bd.start(ctx)
	gui.Start()
}
