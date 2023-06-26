package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"golang.org/x/exp/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
)

//go:embed migrations/*.sql
var migrations embed.FS

var (
	ErrInvalidSid = errors.New("Invalid steamid")
	ErrEmptyValue = errors.New("value cannot be empty")
)

type DataStore interface {
	Close() error
	Connect() error
	Init() error
	SaveUserNameHistory(ctx context.Context, hist *UserNameHistory) error
	SaveMessage(ctx context.Context, message *UserMessage) error
	SavePlayer(ctx context.Context, state *Player) error
	SearchPlayers(ctx context.Context, opts SearchOpts) (PlayerCollection, error)
	FetchNames(ctx context.Context, sid64 steamid.SID64) (UserNameHistoryCollection, error)
	FetchMessages(ctx context.Context, sid steamid.SID64) (UserMessageCollection, error)
	GetPlayer(ctx context.Context, steamID steamid.SID64, create bool, player *Player) error
}

type SqliteStore struct {
	db     *sql.DB
	dsn    string
	logger *slog.Logger
}

func New(dsn string, logger *slog.Logger) *SqliteStore {
	return &SqliteStore{dsn: dsn, logger: logger.WithGroup("sqlite")}
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
	// Note that we do not call migrator.Close and instead close the fsDriver manually.
	// This is because sqlite will wipe the db when :memory: is used and the connection closes
	// for any reason, which the migrator does when called.
	if errClose := fsDriver.Close(); errClose != nil {
		return errors.Wrap(errClose, "Failed to close fs driver")
	}
	return nil
}

func (store *SqliteStore) SaveUserNameHistory(ctx context.Context, hist *UserNameHistory) error {
	query, args, err := sq.
		Insert("player_names").
		Columns("steam_id", "name", "created_on").
		Values(hist.SteamID, hist.Name, time.Now()).
		ToSql()
	if err != nil {
		return err
	}
	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		return errors.Wrap(errExec, "Failed to save name")
	}
	return nil
}

func (store *SqliteStore) SaveMessage(ctx context.Context, message *UserMessage) error {
	query := sq.
		Insert("player_messages").
		Columns("steam_id", "message", "created_on").
		Values(message.SteamID, message.Message, message.Created).
		Suffix("RETURNING \"message_id\"").
		RunWith(store.db)
	if errExec := query.QueryRowContext(ctx).Scan(&message.MessageID); errExec != nil {
		return errors.Wrap(errExec, "Failed to save message")
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
		return errSQL
	}
	if _, errExec := store.db.ExecContext(ctx, query, args...); errExec != nil {
		return errors.Wrap(errExec, "Could not save player state")
	}
	if state.Name != "" {
		name, errName := NewUserNameHistory(state.SteamID, state.Name)
		if errName != nil {
			return errName
		}
		if errSaveName := store.SaveUserNameHistory(ctx, name); errSaveName != nil {
			return errors.Wrap(errSaveName, "Could not save user name history")
		}
	}
	return nil
}

func (store *SqliteStore) updatePlayer(ctx context.Context, state *Player) error {
	state.UpdatedOn = time.Now()
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
		Where(sq.Eq{"steam_id": state.SteamID}).ToSql()
	if errSQL != nil {
		return errSQL
	}
	_, errExec := store.db.ExecContext(ctx, query, args...)
	if errExec != nil {
		return errors.Wrap(errExec, "Could not update player state")
	}
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
			return errors.Wrap(errSaveName, "Could not save user name history")
		}
	}
	return nil
}

func (store *SqliteStore) SavePlayer(ctx context.Context, state *Player) error {
	if !state.SteamID.Valid() {
		return errors.New("Invalid steam id")
	}
	return store.updatePlayer(ctx, state)
}

type SearchOpts struct {
	Query string
}

func (store *SqliteStore) SearchPlayers(ctx context.Context, opts SearchOpts) (PlayerCollection, error) {
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
	query, args, errSQL := qb.ToSql()
	if errSQL != nil {
		return nil, errSQL
	}
	rows, rowErr := store.db.QueryContext(ctx, query, args...)
	if rowErr != nil {
		return nil, rowErr
	}
	defer util.LogClose(store.logger, rows)
	var col PlayerCollection
	for rows.Next() {
		var prevName *string
		var player Player
		if errScan := rows.Scan(&player.SteamID, &player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
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
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return col, nil
}

func (store *SqliteStore) GetPlayer(ctx context.Context, steamID steamid.SID64, create bool, player *Player) error {
	if !steamID.Valid() {
		return errors.New("Invalid steam id")
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
		return errSQL
	}
	var prevName *string
	rowErr := store.db.
		QueryRowContext(ctx, query, args...).
		Scan(&player.Visibility, &player.RealName, &player.AccountCreatedOn, &player.AvatarHash,
			&player.CommunityBanned, &player.NumberOfGameBans, &player.NumberOfVACBans,
			&player.LastVACBanOn, &player.KillsOn, &player.DeathsBy, &player.RageQuits, &player.Notes,
			&player.Whitelisted, &player.CreatedOn, &player.UpdatedOn, &player.ProfileUpdatedOn, &prevName,
		)
	player.SteamID = steamID
	player.SteamIDString = steamID.String()
	player.Matches = []*rules.MatchResult{}
	if rowErr != nil {
		if !errors.Is(rowErr, sql.ErrNoRows) || !create {
			return rowErr
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
	query, args, errSQL := sq.
		Select("name_id", "name", "created_on").
		From("player_names").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSQL != nil {
		return nil, errSQL
	}
	rows, errQuery := store.db.QueryContext(ctx, query, args...)
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer util.LogClose(store.logger, rows)
	var hist UserNameHistoryCollection
	for rows.Next() {
		var h UserNameHistory
		if errScan := rows.Scan(&h.NameID, &h.Name, &h.FirstSeen); errScan != nil {
			return nil, errScan
		}
		hist = append(hist, h)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return hist, nil
}

func (store *SqliteStore) FetchMessages(ctx context.Context, steamID steamid.SID64) (UserMessageCollection, error) {
	query, args, errSQL := sq.
		Select("steam_id", "message_id", "message", "created_on").
		From("player_messages").
		Where(sq.Eq{"steam_id": steamID}).
		ToSql()
	if errSQL != nil {
		return nil, errSQL
	}
	rows, errQuery := store.db.QueryContext(ctx, query, args...)
	if errQuery != nil {
		if errors.Is(errQuery, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errQuery
	}
	defer util.LogClose(store.logger, rows)
	var messages UserMessageCollection
	for rows.Next() {
		var m UserMessage
		if errScan := rows.Scan(&m.SteamID, &m.MessageID, &m.Message, &m.Created); errScan != nil {
			return nil, errScan
		}
		m.SteamIDString = m.SteamID.String()
		messages = append(messages, m)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return messages, nil
}
