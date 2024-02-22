package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type Player struct {
	store.Player

	NamePrevious string `json:"name_previous"`

	EconomyBan steamweb.EconBanState `json:"economy_ban"`

	// - Parsed Ephemeral data

	// tf_lobby_debug
	Team Team `json:"team"`

	// status
	// Connected is how long the user has been in the server
	Connected float64 `json:"connected"`

	MapTimeStart time.Time `json:"-"`
	MapTime      float64   `json:"map_time"`

	// In game user id
	UserID int `json:"user_id"`
	Ping   int `json:"ping"`

	// g15_dumpplayer
	Score       int  `json:"score"`
	IsConnected bool `json:"is_connected"`
	Alive       bool `json:"alive"`
	Health      int  `json:"health"`
	Valid       bool `json:"valid"` // What is it?
	Deaths      int  `json:"deaths"`
	Kills       int  `json:"kills"`

	// - Misc

	KPM float64 `json:"kpm"`

	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int `json:"kick_attempt_count"`

	// Tracks the duration between announces to chat
	AnnouncedPartyLast time.Time `json:"-"`

	AnnouncedGeneralLast time.Time `json:"-"`

	OurFriend bool `json:"our_friend"`

	Sourcebans []models.SbBanRecord `json:"sourcebans"`
	// Dirty indicates that state which has database backed fields has been changed and need to be saved
	Dirty bool `json:"-"`

	Matches []*rules.MatchResult `json:"matches"`
}

func (ps Player) SID64() steamid.SID64 {
	return steamid.New(ps.SteamID)
}

func (ps Player) toUpdateParams() store.PlayerUpdateParams {
	return store.PlayerUpdateParams{
		Visibility:       ps.Visibility,
		RealName:         ps.RealName,
		AccountCreatedOn: time.Time{},
		AvatarHash:       ps.AvatarHash,
		CommunityBanned:  ps.CommunityBanned,
		GameBans:         ps.GameBans,
		VacBans:          ps.VacBans,
		LastVacBanOn:     sql.NullTime{},
		KillsOn:          ps.KillsOn,
		DeathsBy:         ps.DeathsBy,
		RageQuits:        ps.RageQuits,
		Notes:            ps.Notes,
		Whitelist:        ps.Whitelist,
		UpdatedOn:        time.Now(),
		ProfileUpdatedOn: ps.ProfileUpdatedOn,
		SteamID:          ps.SteamID,
	}
}

func (ps Player) MatchAttr(tags []string) bool {
	for _, match := range ps.Matches {
		for _, tag := range tags {
			if match.HasAttr(tag) {
				return true
			}
		}
	}

	return false
}

const (
	playerDisconnect = time.Second * 5
	playerExpiration = time.Second * 60
)

func (ps Player) isProfileExpired() bool {
	return time.Since(ps.ProfileUpdatedOn) < profileAgeLimit || ps.Personaname != ""
}

func (ps Player) isDisconnected() bool {
	return time.Since(ps.UpdatedOn) > playerDisconnect
}

func (ps Player) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > playerExpiration
}

const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

func newPlayer(sid64 steamid.SID64, name string) Player {
	curTIme := time.Now()

	return Player{
		Player: store.Player{
			SteamID:          sid64.Int64(),
			Personaname:      name,
			AvatarHash:       defaultAvatarHash,
			AccountCreatedOn: time.Time{},
			Visibility:       int64(steamweb.VisibilityPublic),
			CreatedOn:        curTIme,
			UpdatedOn:        curTIme,
			ProfileUpdatedOn: curTIme.AddDate(-1, 0, 0),
		},
		Matches: rules.MatchResults{},
	}
}

func playerToPlayerUpdateParams(player Player) store.PlayerUpdateParams {
	return store.PlayerUpdateParams{
		SteamID:          player.SteamID,
		Visibility:       player.Visibility,
		RealName:         player.RealName,
		AccountCreatedOn: player.AccountCreatedOn,
		AvatarHash:       player.AvatarHash,
		CommunityBanned:  player.CommunityBanned,
		GameBans:         player.GameBans,
		VacBans:          player.VacBans,
		LastVacBanOn:     player.LastVacBanOn,
		KillsOn:          player.KillsOn,
		DeathsBy:         player.DeathsBy,
		RageQuits:        player.RageQuits,
		Notes:            player.Notes,
		Whitelist:        player.Whitelist,
		UpdatedOn:        player.UpdatedOn,
		ProfileUpdatedOn: player.ProfileUpdatedOn,
		Personaname:      player.Personaname,
	}
}

func playerRowToPlayer(row store.PlayerRow) Player {
	player := store.Player{
		SteamID:          row.SteamID,
		Personaname:      row.Personaname,
		Visibility:       row.Visibility,
		RealName:         row.RealName,
		AccountCreatedOn: row.AccountCreatedOn,
		AvatarHash:       row.AvatarHash,
		CommunityBanned:  row.CommunityBanned,
		GameBans:         row.GameBans,
		VacBans:          row.VacBans,
		LastVacBanOn:     sql.NullTime{},
		KillsOn:          row.KillsOn,
		DeathsBy:         row.DeathsBy,
		RageQuits:        row.RageQuits,
		Notes:            row.Notes,
		Whitelist:        row.Whitelist,
		ProfileUpdatedOn: row.ProfileUpdatedOn,
		CreatedOn:        row.CreatedOn,
		UpdatedOn:        row.UpdatedOn,
	}

	player.LastVacBanOn = row.LastVacBanOn

	return Player{Player: player}
}

// getPlayerOrCreate attempts to fetch a player from the current player states. If it doesn't exist it will be
// inserted into the database and returned. If you only want players actively in the game, use the playerState functions
// instead.
func getPlayerOrCreate(ctx context.Context, db store.Querier, players *playerState, sid64 steamid.SID64) (Player, error) {
	activePlayer, errPlayer := players.bySteamID(sid64.Int64())
	if errPlayer == nil {
		return activePlayer, nil
	}

	playerRow, errGet := db.Player(ctx, sid64.Int64())
	if errGet != nil {
		if !errors.Is(errGet, sql.ErrNoRows) {
			return Player{}, errors.Join(errGet, errGetPlayer)
		}

		// use date in past to trigger update.
		playerRow.ProfileUpdatedOn = time.Now().AddDate(-1, 0, 0)
	}

	player := playerRowToPlayer(playerRow)

	player.MapTimeStart = time.Now()

	defer players.update(player)

	return player, nil
}

type UserNameHistory struct {
	BaseSID
	NameID    int64     `json:"name_id"`
	Name      string    `json:"name"`
	FirstSeen time.Time `json:"first_seen"`
}

type UserNameHistoryCollection []UserNameHistory
