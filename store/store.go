package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

var (
	ErrOpenDatabase     = errors.New("failed to open database")
	ErrCloseDatabase    = errors.New("failed to cleanly close database")
	ErrStorePragma      = errors.New("failed to enable pragma")
	ErrStoreIOFSOpen    = errors.New("failed to create migration iofs")
	ErrStoreIOFSClose   = errors.New("failed to close migration iofs")
	ErrStoreDriver      = errors.New("failed to create db driver")
	ErrCreateMigration  = errors.New("failed to create migrator")
	ErrPerformMigration = errors.New("failed to migrate database")
)

func CreateDB(dbPath string) (*Queries, func(), error) {
	dbConn, errDB := Connect(dbPath)
	if errDB != nil {
		return nil, nil, errDB
	}

	closer := func() {
		Close(dbConn)
	}

	if errMigrate := Migrate(dbConn); errMigrate != nil {
		return nil, nil, errMigrate
	}

	return New(dbConn), closer, nil
}

func Connect(dsn string) (*sql.DB, error) {
	dsn += "?cache=shared&mode=rwc"
	database, errOpen := sql.Open("sqlite", dsn)
	if errOpen != nil {
		return nil, errors.Join(errOpen, ErrOpenDatabase)
	}

	pragmas := []string{
		"PRAGMA journal_mode = 'WAL'", // WAL does not work if mapped to a network drive
		"PRAGMA encoding = 'UTF-8'",
		"PRAGMA foreign_keys = ON",
		"PRAGMA cache_size = -4096", // Double the cache size from 2mb (negative is correct)
	}

	for _, pragma := range pragmas {
		_, errPragma := database.Exec(pragma)
		if errPragma != nil {
			return nil, fmt.Errorf("%w: %s", ErrStorePragma, pragma)
		}
	}

	return database, nil
}

func Close(db *sql.DB) {
	if db == nil {
		return
	}

	if errClose := db.Close(); errClose != nil {
		slog.Error("Failed to close database", slog.String("error", errClose.Error()))
	}
}

func Migrate(db *sql.DB) error {
	fsDriver, errIofs := iofs.New(migrations, "migrations")
	if errIofs != nil {
		return errors.Join(errIofs, ErrStoreIOFSOpen)
	}

	sqlDriver, errDriver := sqlite.WithInstance(db, &sqlite.Config{})
	if errDriver != nil {
		return errors.Join(errDriver, ErrStoreDriver)
	}

	migrator, errNewMigrator := migrate.NewWithInstance("iofs", fsDriver, "sqlite", sqlDriver)
	if errNewMigrator != nil {
		return errors.Join(errNewMigrator, ErrCreateMigration)
	}

	if errMigrate := migrator.Up(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		return errors.Join(errMigrate, ErrPerformMigration)
	}

	// We do not call migrator.Close and instead close the fsDriver manually.
	// This is because sqlite will wipe the db when :memory: is used and the connection closes
	// for any reason, which the migrator does when called.
	if errClose := fsDriver.Close(); errClose != nil {
		return errors.Join(errClose, ErrStoreIOFSClose)
	}

	return nil
}
