package main

import (
	"context"
	"database/sql"
	"embed"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	_ "modernc.org/sqlite"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

var (
	// ErrNoResult is returned on successful queries which return no rows
	ErrNoResult = errors.New("No results found")
)

type dataStore interface {
	Close()
	Connect() error
	Init() error
	SaveName(ctx context.Context, source steamid.SID64, name string) error
	SaveMessage(ctx context.Context, source steamid.SID64, message string) error
	SavePlayer(ctx context.Context, state *model.PlayerState) error
	LoadOrCreatePlayer(ctx context.Context, steamId steamid.SID64, player *model.PlayerState) error
}

type sqliteStore struct {
	db  *sql.DB
	dsn string
}

func newSqliteStore(dsn string) *sqliteStore {
	return &sqliteStore{dsn: dsn}
}

func (store *sqliteStore) Close() {
	if store.db == nil {
		return
	}
	if errClose := store.db.Close(); errClose != nil {
		log.Printf("Failed to Close database: %v\n", errClose)
	}
}

func (store *sqliteStore) Connect() error {
	database, errOpen := sql.Open("sqlite", store.dsn)
	if errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open database")
	}
	_, errFKSupport := database.Exec("PRAGMA foreign_keys = ON")
	if errFKSupport != nil {
		return errors.Wrap(errFKSupport, "Failed to enable foreign key support")
	}
	store.db = database
	return nil
}

func (store *sqliteStore) Init() error {
	if store.db == nil {
		if errConn := store.Connect(); errConn != nil {
			return errConn
		}
	}
	fsDriver, errIofs := iofs.New(migrations, "migrations")
	if errIofs != nil {
		return errors.Wrap(errIofs, "failed to create iofs")
	}
	sqlDriver, errDriver := sqlite.WithInstance(store.db, &sqlite.Config{})
	if errDriver != nil {
		return errDriver
	}
	migrator, errNewMigrator := migrate.NewWithInstance("iofs", fsDriver, "sqlite", sqlDriver)
	if errNewMigrator != nil {
		return errors.Wrap(errNewMigrator, "Failed to create migrator")
	}
	if errMigrate := migrator.Up(); errMigrate != nil {
		return errors.Wrap(errMigrate, "Failed to migrate database")
	}
	errSource, errDatabase := migrator.Close()
	if errSource != nil {
		return errors.Wrap(errSource, "Failed to Close source driver")
	}
	if errDatabase != nil {
		return errors.Wrap(errDatabase, "Failed to Close database driver")
	}
	return nil
}

func (store *sqliteStore) SaveName(ctx context.Context, source steamid.SID64, name string) error {
	const query = `INSERT INTO player_names (steam_id, name) VALUES (?, ?)`
	if _, errExec := store.db.ExecContext(ctx, query, source.Int64(), name); errExec != nil {
		return errors.Wrap(errExec, "Failed to save name")
	}
	return nil
}

func (store *sqliteStore) SaveMessage(ctx context.Context, steamId steamid.SID64, message string) error {
	const query = `INSERT INTO player_messages (steam_id, message) VALUES (?, ?)`
	if message == "" {
		return nil
	}
	if _, errExec := store.db.ExecContext(ctx, query, steamId.Int64(), message); errExec != nil {
		return errors.Wrap(errExec, "Failed to save name")
	}
	return nil
}

func (store *sqliteStore) SavePlayer(ctx context.Context, state *model.PlayerState) error {
	// TODO sqlite replace is a bit odd, but maybe worth investigation eventually
	const insertQuery = `INSERT INTO player (steam_id, kills_on, deaths_by, created_on, updated_on) VALUES (?, ?, ?, ?, ?)`
	const updateQuery = `UPDATE player SET kills_on = ?, deaths_by = ?, updated_on = ? where steam_id = ?`
	if state.Dangling {
		_, errExec := store.db.ExecContext(ctx, insertQuery, state.SteamId, state.KillsOn, state.DeathsBy, state.CreatedOn, state.UpdatedOn)
		if errExec != nil {
			return errors.Wrap(errExec, "Could not save player state")
		}
		state.Dangling = false
	} else {
		state.UpdatedOn = time.Now()
		_, errExec := store.db.ExecContext(ctx, updateQuery, state.KillsOn, state.DeathsBy, state.UpdatedOn, state.SteamId)
		if errExec != nil {
			return errors.Wrap(errExec, "Could not update player state")
		}
	}
	if errSaveName := store.SaveName(ctx, state.SteamId, state.Name); errSaveName != nil {
		log.Printf("Failed to save name")
	}
	return nil
}

func (store *sqliteStore) LoadOrCreatePlayer(ctx context.Context, steamId steamid.SID64, player *model.PlayerState) error {
	const query = `SELECT kills_on, deaths_by, created_on, updated_on FROM player where steam_id = ?`
	rowErr := dbErr(store.db.
		QueryRow(query, steamId).
		Scan(&player.KillsOn, &player.DeathsBy, &player.CreatedOn, &player.UpdatedOn))
	if rowErr != nil {
		if rowErr != ErrNoResult {
			return rowErr
		}
		player.Dangling = true
		return store.SavePlayer(ctx, player)
	}
	return nil
}

func dbErr(rootError error) error {
	if rootError == nil {
		return rootError
	}
	err := rootError.Error()
	if err == "no rows in result set" {
		return ErrNoResult
	}
	return rootError
}
