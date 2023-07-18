package store

import (
	"time"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type Player struct {
	// - Permanent storage backed
	BaseSID

	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string `json:"name"`

	// CreatedOn is the first time we have seen the player
	CreatedOn time.Time `json:"created_on"`

	// UpdatedOn is the last time we have received a status update from rcon
	// This is used to calculate when we consider the player disconnected and also when
	// they are expired and should be removed from the player pool entirely.
	UpdatedOn        time.Time `json:"updated_on"`
	ProfileUpdatedOn time.Time `json:"profile_updated_on"`

	// The users kill count vs this player
	KillsOn   int `json:"kills_on"`
	RageQuits int `json:"rage_quits"`
	DeathsBy  int `json:"deaths_by"`

	Notes       string `json:"notes"`
	Whitelisted bool   `json:"whitelisted"`

	// PlayerSummary
	RealName         string    `json:"real_name"`
	NamePrevious     string    `json:"name_previous"`
	AccountCreatedOn time.Time `json:"account_created_on"`

	// ProfileVisibility represents whether the profile is visible or not, and if it is visible, why you are allowed to see it.
	// Note that because this WebAPI does not use authentication, there are only two possible values returned:
	// 1 - the profile is not visible to you (Private, Friends Only, etc),
	// 3 - the profile is "Public", and the data is visible.
	// Mike Blaszczak's post on Steam forums says, "The community visibility state this API returns is different
	// from the privacy state. It's the effective visibility state from the account making the request to the account
	// being viewed given the requesting account's relationship to the viewed account.".
	Visibility steamweb.VisibilityState `json:"visibility"`
	AvatarHash string                   `json:"avatar_hash"`

	// PlayerBanState
	CommunityBanned  bool                  `json:"community_banned"`
	NumberOfVACBans  int                   `json:"number_of_vac_bans"`
	LastVACBanOn     *time.Time            `json:"last_vac_ban_on"`
	NumberOfGameBans int                   `json:"number_of_game_bans"`
	EconomyBan       steamweb.EconBanState `json:"economy_ban"`

	// - Parsed Ephemeral data

	// tf_lobby_debug
	Team Team `json:"team"`

	// status
	// Connected is how long the user has been in the server
	Connected float64 `json:"connected"`
	// In game user id
	UserID int `json:"user_id"`
	Ping   int `json:"ping"`

	// g15_dumpplayer
	Score       int  `json:"score"`
	IsConnected bool `json:"is_connected"` // probably not needed
	Alive       bool `json:"alive"`
	Health      int  `json:"health"`
	Valid       bool `json:"valid"` // What is it?
	Deaths      int  `json:"deaths"`
	Kills       int  `json:"kills"`

	// - Misc

	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int `json:"kick_attempt_count"`

	// Tracks the duration between announces to chat
	AnnouncedPartyLast time.Time `json:"-"`

	AnnouncedGeneralLast time.Time `json:"-"`

	OurFriend bool `json:"our_friend"`

	Sourcebans []models.SbBanRecord
	// Dirty indicates that state which has database backed fields has been changed and need to be saved
	Dirty bool `json:"-"`

	Matches []*rules.MatchResult `json:"matches"`
}

func (ps *Player) IsDisconnected() bool {
	return time.Since(ps.UpdatedOn) > time.Second*6
}

func (ps *Player) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > time.Second*20
}

func (ps *Player) Touch() {
	ps.UpdatedOn = time.Now()
	ps.Dirty = true
}

const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

type PlayerCollection []*Player

func (players *PlayerCollection) ByName(name string) (*Player, bool) {
	for _, player := range *players {
		if player.Name == name {
			return player, true
		}
	}

	return nil, false
}

func (players *PlayerCollection) Player(sid64 steamid.SID64) *Player {
	for _, player := range *players {
		if player.SteamID == sid64 {
			return player
		}
	}

	return nil
}

func NewPlayer(sid64 steamid.SID64, name string) *Player {
	curTIme := time.Now()

	return &Player{
		BaseSID:          BaseSID{sid64},
		Name:             name,
		Matches:          []*rules.MatchResult{},
		AvatarHash:       defaultAvatarHash,
		AccountCreatedOn: time.Time{},
		Visibility:       steamweb.VisibilityPublic,
		CreatedOn:        curTIme,
		UpdatedOn:        curTIme,
		ProfileUpdatedOn: curTIme.AddDate(-1, 0, 0),
	}
}

type UserNameHistory struct {
	BaseSID
	NameID    int64     `json:"name_id"`
	Name      string    `json:"name"`
	FirstSeen time.Time `json:"first_seen"`
}

type UserNameHistoryCollection []UserNameHistory

func NewUserNameHistory(steamID steamid.SID64, name string) (*UserNameHistory, error) {
	if name == "" {
		return nil, ErrEmptyValue
	}

	return &UserNameHistory{
		BaseSID:   BaseSID{SteamID: steamID},
		Name:      name,
		FirstSeen: time.Now(),
	}, nil
}
