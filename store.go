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
	"io"
	"log"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

type dataStore interface {
	Close() error
	Connect() error
	Init() error
	SaveName(ctx context.Context, steamID steamid.SID64, name string) error
	SaveMessage(ctx context.Context, message *model.UserMessage) error
	SavePlayer(ctx context.Context, state *model.Player) error
	FetchNames(ctx context.Context, sid64 steamid.SID64) (model.UserNameHistoryCollection, error)
	FetchMessages(ctx context.Context, sid steamid.SID64) (model.UserMessageCollection, error)
	LoadOrCreatePlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error
}

type sqliteStore struct {
	db  *sql.DB
	dsn string
}

func newSqliteStore(dsn string) *sqliteStore {
	return &sqliteStore{dsn: dsn}
}

func (store *sqliteStore) Close() error {
	if store.db == nil {
		return nil
	}
	if errClose := store.db.Close(); errClose != nil {
		return errors.Wrapf(errClose, "Failed to Close database\n")
	}
	return nil
}

func (store *sqliteStore) Connect() error {
	database, errOpen := sql.Open("sqlite", store.dsn)
	if errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open database")
	}
	for _, pragma := range []string{"PRAGMA encoding = 'UTF-8'", "PRAGMA foreign_keys = ON"} {
		_, errPragma := database.Exec(pragma)
		if errPragma != nil {
			return errors.Wrapf(errPragma, "Failed to enable pragma: %s", errPragma)
		}
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
	return store.Connect()
}

func (store *sqliteStore) SaveName(ctx context.Context, steamID steamid.SID64, name string) error {
	const query = `INSERT INTO player_names (steam_id, name, created_on) VALUES (?, ?, ?)`
	if _, errExec := store.db.ExecContext(ctx, query, steamID.Int64(), name, time.Now()); errExec != nil {
		return errors.Wrap(errExec, "Failed to save name")
	}
	return nil
}

func (store *sqliteStore) SaveMessage(ctx context.Context, message *model.UserMessage) error {
	const query = `INSERT INTO player_messages (steam_id, message, created_on) VALUES (?, ?, ?) RETURNING message_id`
	if errExec := store.db.QueryRowContext(ctx, query, message.PlayerSID, message.Message, time.Now()).Scan(&message.MessageId); errExec != nil {
		return errors.Wrap(errExec, "Failed to save message")
	}
	return nil
}

func (store *sqliteStore) insertPlayer(ctx context.Context, state *model.Player) error {
	const insertQuery = `
		INSERT INTO player (
                    steam_id, visibility, real_name, account_created_on, avatar_hash, community_banned, game_bans, vac_bans, 
                    last_vac_ban_on, kills_on, deaths_by, rage_quits, notes, whitelist, created_on, updated_on, profile_updated_on) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, errExec := store.db.ExecContext(
		ctx,
		insertQuery,
		state.SteamId.Int64(),
		state.Visibility,
		state.RealName,
		state.AccountCreatedOn,
		state.AvatarHash,
		state.CommunityBanned,
		state.NumberOfGameBans,
		state.NumberOfVACBans,
		state.LastVACBanOn,
		state.KillsOn,
		state.DeathsBy,
		state.RageQuits,
		state.Notes,
		state.Whitelisted,
		state.CreatedOn,
		state.UpdatedOn,
		state.ProfileUpdatedOn,
	); errExec != nil {
		return errors.Wrap(errExec, "Could not save player state")
	}
	state.Dangling = false
	return nil
}

func (store *sqliteStore) updatePlayer(ctx context.Context, state *model.Player) error {
	const updateQuery = `
		UPDATE player 
		SET visibility = ?, 
		    real_name = ?, 
		    account_created_on = ?, 
		    avatar_hash = ?, 
		    community_banned = ?,
		    game_bans = ?,
            vac_bans = ?,
            last_vac_ban_on = ?,
            kills_on = ?, 
            deaths_by = ?, 
            rage_quits = ?, 
            notes = ?,
            whitelist = ?,
            updated_on = ?,
            profile_updated_on = ?
		WHERE steam_id = ?`

	state.UpdatedOn = time.Now()
	_, errExec := store.db.ExecContext(
		ctx,
		updateQuery,
		state.Visibility,
		state.RealName,
		state.AccountCreatedOn,
		state.AvatarHash,
		state.CommunityBanned,
		state.NumberOfGameBans,
		state.NumberOfVACBans,
		state.LastVACBanOn,
		state.KillsOn,
		state.DeathsBy,
		state.RageQuits,
		state.Notes,
		state.Whitelisted,
		state.UpdatedOn,
		state.ProfileUpdatedOn,
		state.SteamId.Int64())
	if errExec != nil {
		return errors.Wrap(errExec, "Could not update player state")
	}
	return nil
}

func (store *sqliteStore) SavePlayer(ctx context.Context, state *model.Player) error {
	if !state.SteamId.Valid() {
		return errors.New("Invalid steam id")
	}
	if state.Dangling {
		return store.insertPlayer(ctx, state)
	}
	return store.updatePlayer(ctx, state)
}

func (store *sqliteStore) LoadOrCreatePlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error {
	const query = `
		SELECT 
		    p.visibility, 
		    p.real_name, 
		    p.account_created_on, 
		    p.avatar_hash, 
		    p.community_banned,
		    p.game_bans,
		    p.vac_bans, 
		    p.last_vac_ban_on,
		    p.kills_on, 
		    p.deaths_by, 
		    p.rage_quits,
		    p.notes,
		    p.whitelist,
		    p.created_on,
		    p.updated_on, 
		    p.profile_updated_on,
			pn.name
		FROM player p
		LEFT JOIN player_names pn ON p.steam_id = pn.steam_id 
		WHERE p.steam_id = ? 
		ORDER BY pn.created_on DESC
		LIMIT 1`

	var prevName *string
	rowErr := store.db.
		QueryRow(query, steamID).
		Scan(&player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		)
	if rowErr != nil {
		if rowErr != sql.ErrNoRows {
			return rowErr
		}
		player.Dangling = true
		player.SteamId = steamID
		return store.SavePlayer(ctx, player)
	}
	player.Dangling = false
	if prevName != nil {
		player.NamePrevious = *prevName
	}
	return nil
}

func logClose(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		log.Printf("Error trying to close: %v\n", errClose)
	}
}

func (store *sqliteStore) FetchNames(ctx context.Context, steamID steamid.SID64) (model.UserNameHistoryCollection, error) {
	const query = `SELECT name_id, name, created_on FROM player_names WHERE steam_id = ?`
	rows, errQuery := store.db.QueryContext(ctx, query, steamID.Int64())
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer logClose(rows)
	var hist model.UserNameHistoryCollection
	for rows.Next() {
		var h model.UserNameHistory
		if errScan := rows.Scan(&h.NameId, &h.Name, &h.FirstSeen); errScan != nil {
			return nil, errScan
		}
		hist = append(hist, h)
	}
	return hist, nil
}
func (store *sqliteStore) FetchMessages(ctx context.Context, steamID steamid.SID64) (model.UserMessageCollection, error) {
	const query = `SELECT message_id, message, created_on FROM player_messages WHERE steam_id = ?`
	rows, errQuery := store.db.QueryContext(ctx, query, steamID.Int64())
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer logClose(rows)
	var messages model.UserMessageCollection
	for rows.Next() {
		var m model.UserMessage
		if errScan := rows.Scan(&m.MessageId, &m.Message, &m.Created); errScan != nil {
			return nil, errScan
		}
		messages = append(messages, m)
	}
	return messages, nil
}
