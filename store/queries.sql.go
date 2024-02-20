// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: queries.sql

package store

import (
	"context"
	"database/sql"
	"time"
)

const messageSave = `-- name: MessageSave :exec
INSERT INTO player_messages (message_id, steam_id, message, team, dead, created_on)
VALUES (?, ?, ?, ?, ?, ?)
`

type MessageSaveParams struct {
	MessageID int64     `json:"message_id"`
	SteamID   int64     `json:"steam_id"`
	Message   string    `json:"message"`
	Team      bool      `json:"team"`
	Dead      bool      `json:"dead"`
	CreatedOn time.Time `json:"created_on"`
}

func (q *Queries) MessageSave(ctx context.Context, arg MessageSaveParams) error {
	_, err := q.exec(ctx, q.messageSaveStmt, messageSave,
		arg.MessageID,
		arg.SteamID,
		arg.Message,
		arg.Team,
		arg.Dead,
		arg.CreatedOn,
	)
	return err
}

const messages = `-- name: Messages :many
SELECT message_id, steam_id, message, team, dead, created_on
FROM player_messages
WHERE steam_id = ?1
`

func (q *Queries) Messages(ctx context.Context, steamID int64) ([]PlayerMessage, error) {
	rows, err := q.query(ctx, q.messagesStmt, messages, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlayerMessage
	for rows.Next() {
		var i PlayerMessage
		if err := rows.Scan(
			&i.MessageID,
			&i.SteamID,
			&i.Message,
			&i.Team,
			&i.Dead,
			&i.CreatedOn,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const player = `-- name: Player :one
SELECT p.steam_id,
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
       p.personaname
FROM player p
WHERE p.steam_id = ?1
`

type PlayerRow struct {
	SteamID          int64        `json:"steam_id"`
	Visibility       int64        `json:"visibility"`
	RealName         string       `json:"real_name"`
	AccountCreatedOn time.Time    `json:"account_created_on"`
	AvatarHash       string       `json:"avatar_hash"`
	CommunityBanned  bool         `json:"community_banned"`
	GameBans         int64        `json:"game_bans"`
	VacBans          int64        `json:"vac_bans"`
	LastVacBanOn     sql.NullTime `json:"last_vac_ban_on"`
	KillsOn          int64        `json:"kills_on"`
	DeathsBy         int64        `json:"deaths_by"`
	RageQuits        int64        `json:"rage_quits"`
	Notes            string       `json:"notes"`
	Whitelist        bool         `json:"whitelist"`
	CreatedOn        time.Time    `json:"created_on"`
	UpdatedOn        time.Time    `json:"updated_on"`
	ProfileUpdatedOn time.Time    `json:"profile_updated_on"`
	Personaname      string       `json:"personaname"`
}

func (q *Queries) Player(ctx context.Context, steamID int64) (PlayerRow, error) {
	row := q.queryRow(ctx, q.playerStmt, player, steamID)
	var i PlayerRow
	err := row.Scan(
		&i.SteamID,
		&i.Visibility,
		&i.RealName,
		&i.AccountCreatedOn,
		&i.AvatarHash,
		&i.CommunityBanned,
		&i.GameBans,
		&i.VacBans,
		&i.LastVacBanOn,
		&i.KillsOn,
		&i.DeathsBy,
		&i.RageQuits,
		&i.Notes,
		&i.Whitelist,
		&i.CreatedOn,
		&i.UpdatedOn,
		&i.ProfileUpdatedOn,
		&i.Personaname,
	)
	return i, err
}

const playerInsert = `-- name: PlayerInsert :one
INSERT INTO player (steam_id, personaname, visibility, real_name, account_created_on,
                    avatar_hash, community_banned, game_bans, vac_bans, last_vac_ban_on,
                    kills_on, deaths_by, rage_quits, notes, whitelist, profile_updated_on,
                    created_on, updated_on)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING steam_id, personaname, visibility, real_name, account_created_on, avatar_hash, community_banned, game_bans, vac_bans, last_vac_ban_on, kills_on, deaths_by, rage_quits, notes, whitelist, profile_updated_on, created_on, updated_on
`

type PlayerInsertParams struct {
	SteamID          int64        `json:"steam_id"`
	Personaname      string       `json:"personaname"`
	Visibility       int64        `json:"visibility"`
	RealName         string       `json:"real_name"`
	AccountCreatedOn time.Time    `json:"account_created_on"`
	AvatarHash       string       `json:"avatar_hash"`
	CommunityBanned  bool         `json:"community_banned"`
	GameBans         int64        `json:"game_bans"`
	VacBans          int64        `json:"vac_bans"`
	LastVacBanOn     sql.NullTime `json:"last_vac_ban_on"`
	KillsOn          int64        `json:"kills_on"`
	DeathsBy         int64        `json:"deaths_by"`
	RageQuits        int64        `json:"rage_quits"`
	Notes            string       `json:"notes"`
	Whitelist        bool         `json:"whitelist"`
	ProfileUpdatedOn time.Time    `json:"profile_updated_on"`
	CreatedOn        time.Time    `json:"created_on"`
	UpdatedOn        time.Time    `json:"updated_on"`
}

func (q *Queries) PlayerInsert(ctx context.Context, arg PlayerInsertParams) (Player, error) {
	row := q.queryRow(ctx, q.playerInsertStmt, playerInsert,
		arg.SteamID,
		arg.Personaname,
		arg.Visibility,
		arg.RealName,
		arg.AccountCreatedOn,
		arg.AvatarHash,
		arg.CommunityBanned,
		arg.GameBans,
		arg.VacBans,
		arg.LastVacBanOn,
		arg.KillsOn,
		arg.DeathsBy,
		arg.RageQuits,
		arg.Notes,
		arg.Whitelist,
		arg.ProfileUpdatedOn,
		arg.CreatedOn,
		arg.UpdatedOn,
	)
	var i Player
	err := row.Scan(
		&i.SteamID,
		&i.Personaname,
		&i.Visibility,
		&i.RealName,
		&i.AccountCreatedOn,
		&i.AvatarHash,
		&i.CommunityBanned,
		&i.GameBans,
		&i.VacBans,
		&i.LastVacBanOn,
		&i.KillsOn,
		&i.DeathsBy,
		&i.RageQuits,
		&i.Notes,
		&i.Whitelist,
		&i.ProfileUpdatedOn,
		&i.CreatedOn,
		&i.UpdatedOn,
	)
	return i, err
}

const playerSearch = `-- name: PlayerSearch :many
SELECT p.steam_id,
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
       p.profile_updated_on,
       p.created_on,
       p.updated_on,
       p.personaname
FROM player p
WHERE (?1 = 0 OR p.steam_id = ?1)
  AND (?2 IS '' OR p.personaname LIKE ?2)
ORDER BY p.updated_on DESC
LIMIT 1000
`

type PlayerSearchParams struct {
	SteamID interface{} `json:"steam_id"`
	Name    interface{} `json:"name"`
}

type PlayerSearchRow struct {
	SteamID          int64        `json:"steam_id"`
	Visibility       int64        `json:"visibility"`
	RealName         string       `json:"real_name"`
	AccountCreatedOn time.Time    `json:"account_created_on"`
	AvatarHash       string       `json:"avatar_hash"`
	CommunityBanned  bool         `json:"community_banned"`
	GameBans         int64        `json:"game_bans"`
	VacBans          int64        `json:"vac_bans"`
	LastVacBanOn     sql.NullTime `json:"last_vac_ban_on"`
	KillsOn          int64        `json:"kills_on"`
	DeathsBy         int64        `json:"deaths_by"`
	RageQuits        int64        `json:"rage_quits"`
	Notes            string       `json:"notes"`
	Whitelist        bool         `json:"whitelist"`
	ProfileUpdatedOn time.Time    `json:"profile_updated_on"`
	CreatedOn        time.Time    `json:"created_on"`
	UpdatedOn        time.Time    `json:"updated_on"`
	Personaname      string       `json:"personaname"`
}

func (q *Queries) PlayerSearch(ctx context.Context, arg PlayerSearchParams) ([]PlayerSearchRow, error) {
	rows, err := q.query(ctx, q.playerSearchStmt, playerSearch, arg.SteamID, arg.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlayerSearchRow
	for rows.Next() {
		var i PlayerSearchRow
		if err := rows.Scan(
			&i.SteamID,
			&i.Visibility,
			&i.RealName,
			&i.AccountCreatedOn,
			&i.AvatarHash,
			&i.CommunityBanned,
			&i.GameBans,
			&i.VacBans,
			&i.LastVacBanOn,
			&i.KillsOn,
			&i.DeathsBy,
			&i.RageQuits,
			&i.Notes,
			&i.Whitelist,
			&i.ProfileUpdatedOn,
			&i.CreatedOn,
			&i.UpdatedOn,
			&i.Personaname,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const playerUpdate = `-- name: PlayerUpdate :exec
UPDATE player
SET visibility         = ?1,
    real_name          = ?2,
    account_created_on = ?3,
    avatar_hash        = ?4,
    community_banned   = ?5,
    game_bans          = ?6,
    vac_bans           = ?7,
    last_vac_ban_on    = ?8,
    kills_on           = ?9,
    deaths_by          = ?10,
    rage_quits         = ?11,
    notes              = ?12,
    whitelist          = ?13,
    updated_on         = ?14,
    profile_updated_on = ?15,
    personaname        = ?16
WHERE steam_id = ?17
`

type PlayerUpdateParams struct {
	Visibility       int64        `json:"visibility"`
	RealName         string       `json:"real_name"`
	AccountCreatedOn time.Time    `json:"account_created_on"`
	AvatarHash       string       `json:"avatar_hash"`
	CommunityBanned  bool         `json:"community_banned"`
	GameBans         int64        `json:"game_bans"`
	VacBans          int64        `json:"vac_bans"`
	LastVacBanOn     sql.NullTime `json:"last_vac_ban_on"`
	KillsOn          int64        `json:"kills_on"`
	DeathsBy         int64        `json:"deaths_by"`
	RageQuits        int64        `json:"rage_quits"`
	Notes            string       `json:"notes"`
	Whitelist        bool         `json:"whitelist"`
	UpdatedOn        time.Time    `json:"updated_on"`
	ProfileUpdatedOn time.Time    `json:"profile_updated_on"`
	Personaname      string       `json:"personaname"`
	SteamID          int64        `json:"steam_id"`
}

func (q *Queries) PlayerUpdate(ctx context.Context, arg PlayerUpdateParams) error {
	_, err := q.exec(ctx, q.playerUpdateStmt, playerUpdate,
		arg.Visibility,
		arg.RealName,
		arg.AccountCreatedOn,
		arg.AvatarHash,
		arg.CommunityBanned,
		arg.GameBans,
		arg.VacBans,
		arg.LastVacBanOn,
		arg.KillsOn,
		arg.DeathsBy,
		arg.RageQuits,
		arg.Notes,
		arg.Whitelist,
		arg.UpdatedOn,
		arg.ProfileUpdatedOn,
		arg.Personaname,
		arg.SteamID,
	)
	return err
}

const userNameSave = `-- name: UserNameSave :exec
INSERT INTO player_names (name_id, steam_id, name, created_on)
VALUES (?, ?, ?, ?)
`

type UserNameSaveParams struct {
	NameID    int64     `json:"name_id"`
	SteamID   int64     `json:"steam_id"`
	Name      string    `json:"name"`
	CreatedOn time.Time `json:"created_on"`
}

func (q *Queries) UserNameSave(ctx context.Context, arg UserNameSaveParams) error {
	_, err := q.exec(ctx, q.userNameSaveStmt, userNameSave,
		arg.NameID,
		arg.SteamID,
		arg.Name,
		arg.CreatedOn,
	)
	return err
}

const userNames = `-- name: UserNames :many
SELECT name_id, steam_id, name, created_on
FROM player_names
WHERE steam_id = ?1
`

func (q *Queries) UserNames(ctx context.Context, steamID int64) ([]PlayerName, error) {
	rows, err := q.query(ctx, q.userNamesStmt, userNames, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlayerName
	for rows.Next() {
		var i PlayerName
		if err := rows.Scan(
			&i.NameID,
			&i.SteamID,
			&i.Name,
			&i.CreatedOn,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
