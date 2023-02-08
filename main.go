package main

import (
	"context"
	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/model"
	_ "github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	settings := model.NewSettings()
	gameState := model.NewGameState()
	rulesEngine := newRulesEngine()
	if errReadSettings := settings.ReadDefault(); errReadSettings != nil {
		log.Println(errReadSettings)
	}
	if settings.ApiKey != "" {
		if errApiKey := steamweb.SetKey(settings.ApiKey); errApiKey != nil {
			log.Printf("Failed to set steam api key: %v\n", errApiKey)
		}
	}
	store := newSqliteStore(settings.DBPath())
	if errMigrate := store.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		log.Printf("Failed to migrate database: %v\n", errMigrate)
		os.Exit(1)
	}
	defer store.Close()
	bd := New(ctx, &settings, store, &gameState, &rulesEngine)
	gui := ui.New(ctx, &settings, &gameState)
	bd.AttachGui(gui)
	go bd.start()
	gui.Start()
}
