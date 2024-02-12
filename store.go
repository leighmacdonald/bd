package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DataStore interface {
	Close() error
	Connect() error
	Init() error
	SaveUserNameHistory(ctx context.Context, hist *UserNameHistory) error
	SaveMessage(ctx context.Context, message *UserMessage) error
	SavePlayer(ctx context.Context, state *Player) error
	SearchPlayers(ctx context.Context, opts SearchOpts) ([]Player, error)
	FetchNames(ctx context.Context, sid64 steamid.SID64) (UserNameHistoryCollection, error)
	FetchMessages(ctx context.Context, sid steamid.SID64) (UserMessageCollection, error)
	GetPlayer(ctx context.Context, steamID steamid.SID64, create bool, player *Player) error
}

type SqliteStore struct {
	sync.RWMutex
	db     *sql.DB
	dsn    string
	logger *slog.Logger
}

func NewStore(dsn string) *SqliteStore {
	return &SqliteStore{dsn: dsn + "?cache=shared&mode=rwc", logger: slog.Default().WithGroup("sqlite")}
}

func (store *SqliteStore) Close() error {
	if store.db == nil {
		return nil
	}

	if errClose := store.db.Close(); errClose != nil {
		return errors.Join(errClose, errCloseDatabase)
	}

	return nil
}

func (store *SqliteStore) Connect() error {
	database, errOpen := sql.Open("sqlite", store.dsn)
	if errOpen != nil {
		return errors.Join(errOpen, errOpenDatabase)
	}

	for _, pragma := range []string{"PRAGMA encoding = 'UTF-8'", "PRAGMA foreign_keys = ON"} {
		_, errPragma := database.Exec(pragma)
		if errPragma != nil {
			return fmt.Errorf("%w: %s", errStorePragma, pragma)
		}
	}

	store.db = database

	return nil
}

func (store *SqliteStore) Init() error {
	if store.db == nil {
		if errConn := store.Connect(); errConn != nil {
			return errConn
		}
	}

	fsDriver, errIofs := iofs.New(migrations, "migrations")
	if errIofs != nil {
		return errors.Join(errIofs, errStoreIOFSOpen)
	}

	sqlDriver, errDriver := sqlite.WithInstance(store.db, &sqlite.Config{})
	if errDriver != nil {
		return errors.Join(errDriver, errStoreDriver)
	}

	migrator, errNewMigrator := migrate.NewWithInstance("iofs", fsDriver, "sqlite", sqlDriver)
	if errNewMigrator != nil {
		return errors.Join(errNewMigrator, errCreateMigration)
	}

	if errMigrate := migrator.Up(); errMigrate != nil {
		return errors.Join(errMigrate, errPerformMigration)
	}

	// Note that we do not call migrator.Close and instead close the fsDriver manually.
	// This is because sqlite will wipe the db when :memory: is used and the connection closes
	// for any reason, which the migrator does when called.
	if errClose := fsDriver.Close(); errClose != nil {
		return errors.Join(errClose, errStoreIOFSClose)
	}

	return nil
}

func (store *SqliteStore) SaveUserNameHistory(ctx context.Context, hist *UserNameHistory) error {
	store.Lock()
	defer store.Unlock()

	query, args, errSQL := sq.
		Insert("player_names").
		Columns("steam_id", "name", "created_on").
		Values(hist.SteamID.Int64(), hist.Name, time.Now()).
		ToSql()
	if errSQL != nil {
		return errors.Join(errSQL, errGenerateQuery)
	}

	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		return errors.Join(errExec, errStoreExec)
	}

	return nil
}

func (store *SqliteStore) SaveMessage(ctx context.Context, message *UserMessage) error {
	store.Lock()
	defer store.Unlock()

	query := sq.
		Insert("player_messages").
		Columns("steam_id", "message", "created_on").
		Values(message.SteamID, message.Message, message.Created).
		Suffix("RETURNING \"message_id\"").
		RunWith(store.db)

	if errExec := query.QueryRowContext(ctx).Scan(&message.MessageID); errExec != nil {
		return errors.Join(errExec, errStoreExec)
	}

	return nil
}

func (store *SqliteStore) insertPlayer(ctx context.Context, state *Player) error {
	query, args, errSQL := sq.
		Insert("player").
		Columns("steam_id", "visibility", "real_name", "account_created_on", "avatar_hash",
			"community_banned", "game_bans", "vac_bans", "last_vac_ban_on", "kills_on", "deaths_by",
			"rage_quits", "notes", "whitelist", "created_on", "updated_on", "profile_updated_on").
		Values(state.SteamID.Int64(), state.Visibility, state.RealName, state.AccountCreatedOn, state.AvatarHash,
			state.CommunityBanned, state.NumberOfGameBans, state.NumberOfVACBans, state.LastVACBanOn, state.KillsOn,
			state.DeathsBy, state.RageQuits, state.Notes, state.Whitelisted, state.CreatedOn,
			state.UpdatedOn, state.ProfileUpdatedOn).
		ToSql()
	if errSQL != nil {
		return errors.Join(errSQL, errGenerateQuery)
	}

	store.Lock()
	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		store.Unlock()

		return errors.Join(errExec, errStoreExec)
	}

	store.Unlock()

	if state.Name != "" {
		name, errName := NewUserNameHistory(state.SteamID, state.Name)
		if errName != nil {
			return errName
		}

		if errSaveName := store.SaveUserNameHistory(ctx, name); errSaveName != nil {
			return errSaveName
		}
	}

	return nil
}

func (store *SqliteStore) updatePlayer(ctx context.Context, state *Player) error {
	query, args, errSQL := sq.
		Update("player").
		Set("visibility", state.Visibility).
		Set("real_name", state.RealName).
		Set("account_created_on", state.AccountCreatedOn).
		Set("avatar_hash", state.AvatarHash).
		Set("community_banned", state.CommunityBanned).
		Set("game_bans", state.NumberOfGameBans).
		Set("vac_bans", state.NumberOfVACBans).
		Set("last_vac_ban_on", state.LastVACBanOn).
		Set("kills_on", state.KillsOn).
		Set("deaths_by", state.DeathsBy).
		Set("rage_quits", state.RageQuits).
		Set("notes", state.Notes).
		Set("whitelist", state.Whitelisted).
		Set("updated_on", state.UpdatedOn).
		Set("profile_updated_on", state.ProfileUpdatedOn).
		Where(sq.Eq{"steam_id": state.SteamID.Int64()}).ToSql()

	if errSQL != nil {
		return errors.Join(errSQL, errGenerateQuery)
	}

	store.Lock()

	_, errExec := store.db.ExecContext(ctx, query, args...)
	if errExec != nil {
		store.Unlock()

		return errors.Join(errExec, errStoreExec)
	}

	store.Unlock()

	var existing Player
	if errExisting := store.GetPlayer(ctx, state.SteamID, false, &existing); errExisting != nil {
		return errExisting
	}

	if existing.Name != state.Name {
		name, errName := NewUserNameHistory(state.SteamID, state.Name)
		if errName != nil {
			return errName
		}

		if errSaveName := store.SaveUserNameHistory(ctx, name); errSaveName != nil {
			return errSaveName
		}
	}

	return nil
}

func (store *SqliteStore) SavePlayer(ctx context.Context, state *Player) error {
	if !state.SteamID.Valid() {
		return errInvalidSid
	}

	return store.updatePlayer(ctx, state)
}

type SearchOpts struct {
	Query string
}

func (store *SqliteStore) SearchPlayers(ctx context.Context, opts SearchOpts) ([]Player, error) {
	store.Lock()
	defer store.Unlock()

	builder := sq.
		Select("p.steam_id", "p.visibility", "p.real_name", "p.account_created_on", "p.avatar_hash",
			"p.community_banned", "p.game_bans", "p.vac_bans", "p.last_vac_ban_on", "p.kills_on", "p.deaths_by",
			"p.rage_quits", "p.notes", "p.whitelist", "p.created_on", "p.updated_on", "p.profile_updated_on", "pn.name").
		From("player p").
		LeftJoin("player_names pn ON p.steam_id = pn.steam_id ").
		OrderBy("p.updated_on DESC").
		Limit(1000)

	sid64, errSid := steamid.StringToSID64(opts.Query)
	if errSid == nil && sid64.Valid() {
		builder = builder.Where(sq.Like{"p.steam_id": sid64})
	} else if opts.Query != "" {
		builder = builder.Where(sq.Like{"pn.name": fmt.Sprintf("%%%s%%", opts.Query)})
	}

	query, args, errSQL := builder.ToSql()
	if errSQL != nil {
		return nil, errors.Join(errSQL, errGenerateQuery)
	}

	rows, rowErr := store.db.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if rowErr != nil {
		return nil, errors.Join(rowErr, errStoreExec)
	}

	defer LogClose(rows)

	var col []Player

	for rows.Next() {
		var (
			prevName *string
			player   Player
			sid      int64
		)

		if errScan := rows.Scan(&sid, &player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		); errScan != nil {
			return nil, errors.Join(errScan, errScanResult)
		}

		player.SteamID = steamid.New(sid)

		if prevName != nil {
			player.Name = *prevName
			player.NamePrevious = *prevName
		}

		col = append(col, player)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), errRows)
	}

	return col, nil
}

func (store *SqliteStore) GetPlayer(ctx context.Context, steamID steamid.SID64, create bool, player *Player) error {
	if !steamID.Valid() {
		return errInvalidSid
	}

	query, args, errSQL := sq.
		Select("p.visibility", "p.real_name", "p.account_created_on", "p.avatar_hash",
			"p.community_banned", "p.game_bans", "p.vac_bans", "p.last_vac_ban_on", "p.kills_on", "p.deaths_by",
			"p.rage_quits", "p.notes", "p.whitelist", "p.created_on", "p.updated_on", "p.profile_updated_on", "pn.name").
		From("player p").
		LeftJoin("player_names pn ON p.steam_id = pn.steam_id ").
		Where(sq.Eq{"p.steam_id": steamID}).
		OrderBy("pn.created_on DESC").
		Limit(1).
		ToSql()
	if errSQL != nil {
		return errors.Join(errSQL, errGenerateQuery)
	}

	var prevName *string

	store.Lock()

	rowErr := store.db.
		QueryRowContext(ctx, query, args...).
		Scan(&player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		)

	store.Unlock()

	player.SteamID = steamID
	player.Matches = []*rules.MatchResult{}

	if rowErr != nil {
		if !errors.Is(rowErr, sql.ErrNoRows) || !create {
			return errors.Join(rowErr, errScanResult)
		}

		if errSave := store.insertPlayer(ctx, player); errSave != nil {
			return errSave
		}
	}

	if prevName != nil {
		player.NamePrevious = *prevName
	}

	return nil
}

func (store *SqliteStore) FetchNames(ctx context.Context, steamID steamid.SID64) (UserNameHistoryCollection, error) {
	store.Lock()
	defer store.Unlock()

	query, args, errSQL := sq.
		Select("name_id", "name", "created_on").
		From("player_names").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSQL != nil {
		return nil, errors.Join(errSQL, errGenerateQuery)
	}

	rows, errQuery := store.db.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errors.Join(errQuery, errStoreExec)
	}

	defer LogClose(rows)

	var hist UserNameHistoryCollection

	for rows.Next() {
		var nameHistory UserNameHistory
		if errScan := rows.Scan(&nameHistory.NameID, &nameHistory.Name, &nameHistory.FirstSeen); errScan != nil {
			return nil, errors.Join(errScan, errScanResult)
		}

		nameHistory.SteamID = steamID

		hist = append(hist, nameHistory)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), errRows)
	}

	return hist, nil
}

func (store *SqliteStore) FetchMessages(ctx context.Context, steamID steamid.SID64) (UserMessageCollection, error) {
	store.Lock()
	defer store.Unlock()

	query, args, errSQL := sq.
		Select("steam_id", "message_id", "message", "created_on").
		From("player_messages").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSQL != nil {
		return nil, errors.Join(errSQL, errGenerateQuery)
	}

	rows, errQuery := store.db.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errors.Join(errQuery, errStoreExec)
	}

	defer LogClose(rows)

	var messages UserMessageCollection

	for rows.Next() {
		var (
			message UserMessage
			sid     int64
		)

		if errScan := rows.Scan(&sid, &message.MessageID, &message.Message, &message.Created); errScan != nil {
			return nil, errors.Join(errScan, errScanResult)
		}

		message.SteamID = steamid.New(sid)
		messages = append(messages, message)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), errRows)
	}

	return messages, nil
}
