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

type PlayerState struct {
	SteamID          steamid.SID64 `json:"steam_id"`
	Personaname      string        `json:"personaname"`
	Visibility       int64         `json:"visibility"`
	RealName         string        `json:"real_name"`
	AccountCreatedOn time.Time     `json:"account_created_on"`
	AvatarHash       string        `json:"avatar_hash"`
	CommunityBanned  bool          `json:"community_banned"`
	GameBans         int64         `json:"game_bans"`
	VacBans          int64         `json:"vac_bans"`
	LastVacBanOn     int64         `json:"last_vac_ban_on"`
	KillsOn          int64         `json:"kills_on"`
	DeathsBy         int64         `json:"deaths_by"`
	RageQuits        int64         `json:"rage_quits"`
	Notes            string        `json:"notes"`
	Whitelist        bool          `json:"whitelist"`
	ProfileUpdatedOn time.Time     `json:"profile_updated_on"`
	CreatedOn        time.Time     `json:"created_on"`
	UpdatedOn        time.Time     `json:"updated_on"`

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

	// Misc
	KPM float64 `json:"kpm"`
	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int `json:"kick_attempt_count"`
	// Tracks the duration between announces to chat
	AnnouncedPartyLast   time.Time            `json:"-"`
	AnnouncedGeneralLast time.Time            `json:"-"`
	OurFriend            bool                 `json:"our_friend"`
	Sourcebans           []models.SbBanRecord `json:"sourcebans"`
	Matches              []rules.MatchResult  `json:"matches"`
}

func (ps PlayerState) SID64() steamid.SID64 {
	return steamid.New(ps.SteamID)
}

func (ps PlayerState) MatchAttr(tags []string) bool {
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
	profileAgeLimit  = time.Hour * 24
)

func (ps PlayerState) isProfileExpired() bool {
	return time.Since(ps.ProfileUpdatedOn) < profileAgeLimit || ps.Personaname != ""
}

func (ps PlayerState) isDisconnected() bool {
	return time.Since(ps.UpdatedOn) > playerDisconnect
}

func (ps PlayerState) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > playerExpiration
}

const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

func newPlayer(sid64 steamid.SID64, name string) PlayerState {
	curTIme := time.Now()

	return PlayerState{
		SteamID:          sid64,
		Personaname:      name,
		AvatarHash:       defaultAvatarHash,
		AccountCreatedOn: time.Time{},
		Visibility:       int64(steamweb.VisibilityPublic),
		CreatedOn:        curTIme,
		UpdatedOn:        curTIme,
		ProfileUpdatedOn: curTIme.AddDate(-1, 0, 0),
		Matches:          rules.MatchResults{},
	}
}

func (ps PlayerState) toUpdateParams() store.PlayerUpdateParams {
	return store.PlayerUpdateParams{
		SteamID:          ps.SteamID.Int64(),
		Visibility:       ps.Visibility,
		RealName:         ps.RealName,
		AccountCreatedOn: ps.AccountCreatedOn,
		AvatarHash:       ps.AvatarHash,
		CommunityBanned:  ps.CommunityBanned,
		GameBans:         ps.GameBans,
		VacBans:          ps.VacBans,
		LastVacBanOn: sql.NullTime{
			Time:  time.Unix(ps.LastVacBanOn, 0),
			Valid: ps.LastVacBanOn > 0,
		},
		KillsOn:          ps.KillsOn,
		DeathsBy:         ps.DeathsBy,
		RageQuits:        ps.RageQuits,
		Notes:            ps.Notes,
		Whitelist:        ps.Whitelist,
		UpdatedOn:        ps.UpdatedOn,
		ProfileUpdatedOn: ps.ProfileUpdatedOn,
		Personaname:      ps.Personaname,
	}
}

func playerRowToPlayerState(row store.PlayerRow) PlayerState {
	ps := newPlayer(steamid.New(row.SteamID), row.Personaname)
	ps.Visibility = row.Visibility
	ps.RealName = row.RealName
	ps.AccountCreatedOn = row.AccountCreatedOn
	ps.AvatarHash = row.AvatarHash
	ps.CommunityBanned = row.CommunityBanned
	ps.GameBans = row.GameBans
	ps.VacBans = row.VacBans
	if row.LastVacBanOn.Valid {
		ps.LastVacBanOn = row.LastVacBanOn.Time.Unix()
	}
	ps.KillsOn = row.KillsOn
	ps.DeathsBy = row.DeathsBy
	ps.RageQuits = row.RageQuits
	ps.Notes = row.Notes
	ps.Whitelist = row.Whitelist
	ps.ProfileUpdatedOn = row.ProfileUpdatedOn
	ps.CreatedOn = row.CreatedOn
	ps.UpdatedOn = row.UpdatedOn

	return ps
}

// getPlayerOrCreate attempts to fetch a player from the current player states. If it doesn't exist it will be
// inserted into the database and returned. If you only want players actively in the game, use the playerStates functions
// instead.
func getPlayerOrCreate(ctx context.Context, db store.Querier, players *playerStates, sid64 steamid.SID64) (PlayerState, error) {
	activePlayer, errPlayer := players.bySteamID(sid64)
	if errPlayer == nil {
		return activePlayer, nil
	}

	playerRow, errGet := db.Player(ctx, sid64.Int64())
	if errGet != nil {
		if !errors.Is(errGet, sql.ErrNoRows) {
			return PlayerState{}, errors.Join(errGet, errGetPlayer)
		}

		// use date in past to trigger update.
		playerRow.ProfileUpdatedOn = time.Now().AddDate(-1, 0, 0)
	}

	player := playerRowToPlayerState(playerRow)

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
