package store

import (
	"fmt"
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
	UserID int64 `json:"user_id"`
	Ping   int   `json:"ping"`

	// Parsed stats from logs
	Kills  int `json:"kills"`
	Deaths int `json:"deaths"`

	// - Misc

	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int `json:"kick_attempt_count"`

	// Tracks the duration between announces to chat
	AnnouncedPartyLast time.Time `json:"-"`

	AnnouncedGeneralLast time.Time `json:"-"`

	OurFriend bool `json:"our_friend"`

	InGame     bool
	Sourcebans []models.SbBanRecord
	// Dirty indicates that state which has database backed fields has been changed and need to be saved
	Dirty bool `json:"-"`

	Matches []*rules.MatchResult `json:"matches"`
}

func (ps Player) IsMatched() bool {
	return len(ps.Matches) > 0
}

func (ps Player) GetAvatarHash() string {
	if ps.AvatarHash == "" {
		return defaultAvatarHash
	}

	return ps.AvatarHash
}

func (ps Player) IsDisconnected() bool {
	return time.Since(ps.UpdatedOn) > time.Second*6
}

func (ps Player) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > time.Second*20
}

func (ps Player) Touch() {
	ps.Dirty = true
}

func firstN(text string, count int) string {
	index := 0

	for j := range text {
		if index == count {
			return text[:j]
		}

		index++
	}

	return text
}

const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

// API returns non https urls, this will resolve them over https.
const baseAvatarURL = "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/avatars"

func AvatarURL(hash string) string {
	avatarHash := defaultAvatarHash
	if hash != "" {
		avatarHash = hash
	}

	return fmt.Sprintf("%s/%s/%s_full.jpg", baseAvatarURL, firstN(avatarHash, 2), avatarHash)
}

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

func (players *PlayerCollection) GetOrCreate(sid64 steamid.SID64) *Player {
	if player := players.Player(sid64); player != nil {
		return player
	}

	newPlayer := NewPlayer(sid64, "")

	*players = append(*players, newPlayer)

	return newPlayer
}

func NewPlayer(sid64 steamid.SID64, name string) *Player {
	curTIme := time.Now()

	return &Player{
		BaseSID:          BaseSID{sid64},
		Name:             name,
		Matches:          []*rules.MatchResult{},
		AccountCreatedOn: time.Time{},
		Visibility:       steamweb.VisibilityPublic,
		CreatedOn:        curTIme,
		UpdatedOn:        curTIme,
		ProfileUpdatedOn: curTIme.AddDate(-1, 0, 0),
	}
}

type UserNameHistory struct {
	BaseSID
	NameID    int64
	Name      string
	FirstSeen time.Time
}

type UserNameHistoryCollection []UserNameHistory

func (names UserNameHistoryCollection) AsAny() []any {
	bl := make([]any, len(names))
	for i, r := range names {
		bl[i] = r
	}

	return bl
}

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
