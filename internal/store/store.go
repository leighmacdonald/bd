package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DataStore interface {
	Close() error
	Connect() error
	Init() error
	SaveName(ctx context.Context, steamID steamid.SID64, name string) error
	SaveMessage(ctx context.Context, message *model.UserMessage) error
	SavePlayer(ctx context.Context, state *model.Player) error
	SearchPlayers(ctx context.Context, opts model.SearchOpts) (model.PlayerCollection, error)
	FetchNames(ctx context.Context, sid64 steamid.SID64) (model.UserNameHistoryCollection, error)
	FetchMessages(ctx context.Context, sid steamid.SID64) (model.UserMessageCollection, error)
	LoadOrCreatePlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error
	GetPlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error
}

type SqliteStore struct {
	db     *sql.DB
	dsn    string
	logger *zap.Logger
}

func New(dsn string, logger *zap.Logger) *SqliteStore {
	return &SqliteStore{dsn: dsn, logger: logger}
}

func (store *SqliteStore) Close() error {
	if store.db == nil {
		return nil
	}
	if errClose := store.db.Close(); errClose != nil {
		return errors.Wrapf(errClose, "Failed to Close database\n")
	}
	return nil
}

func (store *SqliteStore) Connect() error {
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

func (store *SqliteStore) Init() error {
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

func (store *SqliteStore) SaveName(ctx context.Context, steamID steamid.SID64, name string) error {
	query, args, err := sq.
		Insert("player_names").
		Columns("steam_id", "name", "created_on").
		Values(steamID, name, time.Now()).
		ToSql()
	if err != nil {
		return err
	}
	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		return errors.Wrap(errExec, "Failed to save name")
	}
	return nil
}

func (store *SqliteStore) SaveMessage(ctx context.Context, message *model.UserMessage) error {
	query := sq.
		Insert("player_messages").
		Columns("steam_id", "message", "created_on").
		Values(message.PlayerSID, message.Message, time.Now()).
		Suffix("RETURNING \"message_id\"").
		RunWith(store.db)
	if errExec := query.QueryRowContext(ctx).Scan(&message.MessageId); errExec != nil {
		return errors.Wrap(errExec, "Failed to save message")
	}
	return nil
}

func (store *SqliteStore) insertPlayer(ctx context.Context, state *model.Player) error {
	query, args, errSql := sq.
		Insert("player").
		Columns("steam_id", "visibility", "real_name", "account_created_on", "avatar_hash",
			"community_banned", "game_bans", "vac_bans", "last_vac_ban_on", "kills_on", "deaths_by",
			"rage_quits", "notes", "whitelist", "created_on", "updated_on", "profile_updated_on").
		Values(state.SteamId.Int64(), state.Visibility, state.RealName, state.AccountCreatedOn, state.AvatarHash,
			state.CommunityBanned, state.NumberOfGameBans, state.NumberOfVACBans, state.LastVACBanOn, state.KillsOn,
			state.DeathsBy, state.RageQuits, state.Notes, state.Whitelisted, state.CreatedOn,
			state.UpdatedOn, state.ProfileUpdatedOn).
		ToSql()
	if errSql != nil {
		return errSql
	}
	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		return errors.Wrap(errExec, "Could not save player state")
	}
	state.Dangling = false
	return nil
}

func (store *SqliteStore) updatePlayer(ctx context.Context, state *model.Player) error {
	state.UpdatedOn = time.Now()
	query, args, errSql := sq.
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
		Where(sq.Eq{"steam_id": state.SteamId}).ToSql()
	if errSql != nil {
		return errSql
	}
	_, errExec := store.db.ExecContext(ctx, query, args...)
	if errExec != nil {
		return errors.Wrap(errExec, "Could not update player state")
	}
	return nil
}

func (store *SqliteStore) SavePlayer(ctx context.Context, state *model.Player) error {
	if !state.SteamId.Valid() {
		return errors.New("Invalid steam id")
	}
	if state.Dangling {
		return store.insertPlayer(ctx, state)
	}
	return store.updatePlayer(ctx, state)
}

func (store *SqliteStore) SearchPlayers(ctx context.Context, opts model.SearchOpts) (model.PlayerCollection, error) {
	qb := sq.
		Select("p.steam_id", "p.visibility", "p.real_name", "p.account_created_on", "p.avatar_hash",
			"p.community_banned", "p.game_bans", "p.vac_bans", "p.last_vac_ban_on", "p.kills_on", "p.deaths_by",
			"p.rage_quits", "p.notes", "p.whitelist", "p.created_on", "p.updated_on", "p.profile_updated_on", "pn.name").
		From("player p").
		LeftJoin("player_names pn ON p.steam_id = pn.steam_id ").
		OrderBy("p.updated_on DESC").
		Limit(1000)

	sid64, errSid := steamid.StringToSID64(opts.Query)
	if errSid == nil && sid64.Valid() {
		qb = qb.Where(sq.Like{"p.steam_id": sid64})
	} else if opts.Query != "" {
		qb = qb.Where(sq.Like{"pn.name": fmt.Sprintf("%%%s%%", opts.Query)})
	}
	query, args, errSql := qb.ToSql()
	if errSql != nil {
		return nil, errSql
	}
	rows, rowErr := store.db.QueryContext(ctx, query, args...)
	if rowErr != nil {
		return nil, rowErr
	}
	defer util.LogClose(store.logger, rows)
	var col model.PlayerCollection
	for rows.Next() {
		var prevName *string
		var player model.Player
		if errScan := rows.Scan(&player.SteamId, &player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		); errScan != nil {
			return nil, errScan
		}
		if prevName != nil {
			player.Name = *prevName
			player.NamePrevious = *prevName
		}
		col = append(col, &player)

	}
	return col, nil
}

func (store *SqliteStore) GetPlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error {
	query, args, errSql := sq.
		Select("p.visibility", "p.real_name", "p.account_created_on", "p.avatar_hash",
			"p.community_banned", "p.game_bans", "p.vac_bans", "p.last_vac_ban_on", "p.kills_on", "p.deaths_by",
			"p.rage_quits", "p.notes", "p.whitelist", "p.created_on", "p.updated_on", "p.profile_updated_on", "pn.name").
		From("player p").
		LeftJoin("player_names pn ON p.steam_id = pn.steam_id ").
		Where(sq.Eq{"p.steam_id": steamID}).
		OrderBy("pn.created_on DESC").
		Limit(1).
		ToSql()
	if errSql != nil {
		return errSql
	}
	var prevName *string
	rowErr := store.db.
		QueryRowContext(ctx, query, args...).
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
	}
	player.SteamId = steamID
	player.Dangling = false
	if prevName != nil {
		player.NamePrevious = *prevName
	}
	return nil
}

func (store *SqliteStore) LoadOrCreatePlayer(ctx context.Context, steamID steamid.SID64, player *model.Player) error {
	query, args, errSql := sq.
		Select("p.visibility", "p.real_name", "p.account_created_on", "p.avatar_hash",
			"p.community_banned", "p.game_bans", "p.vac_bans", "p.last_vac_ban_on", "p.kills_on", "p.deaths_by",
			"p.rage_quits", "p.notes", "p.whitelist", "p.created_on", "p.updated_on", "p.profile_updated_on", "pn.name").
		From("player p").
		LeftJoin("player_names pn ON p.steam_id = pn.steam_id").
		Where(sq.Eq{"p.steam_id": steamID}).
		OrderBy("pn.created_on DESC").
		Limit(1).
		ToSql()
	if errSql != nil {
		return errSql
	}
	var prevName *string
	rowErr := store.db.
		QueryRowContext(ctx, query, args...).
		Scan(&player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		)
	player.SteamId = steamID
	if rowErr != nil {
		if rowErr != sql.ErrNoRows {
			return rowErr
		}
		player.Dangling = true
		return store.SavePlayer(ctx, player)
	}
	player.Dangling = false
	if prevName != nil {
		player.NamePrevious = *prevName
	}
	return nil
}

func (store *SqliteStore) FetchNames(ctx context.Context, steamID steamid.SID64) (model.UserNameHistoryCollection, error) {
	query, args, errSql := sq.
		Select("name_id", "name", "created_on").
		From("player_names").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSql != nil {
		return nil, errSql
	}
	rows, errQuery := store.db.QueryContext(ctx, query, args...)
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer util.LogClose(store.logger, rows)
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

func (store *SqliteStore) FetchMessages(ctx context.Context, steamID steamid.SID64) (model.UserMessageCollection, error) {
	query, args, errSql := sq.
		Select("message_id", "message", "created_on").
		From("player_messages").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSql != nil {
		return nil, errSql
	}
	rows, errQuery := store.db.QueryContext(ctx, query, args...)
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer util.LogClose(store.logger, rows)
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
