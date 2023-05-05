package store

import (
	"fmt"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

// ProfileVisibility represents whether the profile is visible or not, and if it is visible, why you are allowed to see it.
// Note that because this WebAPI does not use authentication, there are only two possible values returned:
// 1 - the profile is not visible to you (Private, Friends Only, etc),
// 3 - the profile is "Public", and the data is visible.
// Mike Blaszczak's post on Steam forums says, "The community visibility state this API returns is different
// than the privacy state. It's the effective visibility state from the account making the request to the account
// being viewed given the requesting account's relationship to the viewed account."
type ProfileVisibility int

const (
	ProfileVisibilityPrivate ProfileVisibility = iota + 1
	ProfileVisibilityFriendsOnly
	ProfileVisibilityPublic
)

type Player struct {
	// - Permanent storage backed

	// SteamId is the 64bit steamid of the user
	SteamId       steamid.SID64 `json:"-"`
	SteamIdString string        `json:"steam_id"`
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

	Visibility ProfileVisibility `json:"visibility"`
	AvatarHash string            `json:"avatar_hash"`

	// PlayerBanState
	CommunityBanned  bool       `json:"community_banned"`
	NumberOfVACBans  int        `json:"number_of_vac_bans"`
	LastVACBanOn     *time.Time `json:"last_vac_ban_on"`
	NumberOfGameBans int        `json:"number_of_game_bans"`
	EconomyBan       bool       `json:"economy_ban"`

	// - Parsed Ephemeral data

	// tf_lobby_debug
	Team Team `json:"team"`

	// status
	// Connected is how long the user has been in the server
	Connected float64 `json:"connected"`
	// In game user id
	UserId int64 `json:"user_id"`
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

	// Dangling will be true when the user is new and doesn't have a physical entry in the database yet.
	Dangling bool `json:"-"`

	OurFriend bool `json:"our_friend"`

	// Dirty indicates that state which has database backed fields has been changed and need to be saved
	Dirty bool `json:"-"`

	Matches []*rules.MatchResult `json:"matches"`
}

func (ps *Player) IsMatched() bool {
	return len(ps.Matches) > 0
}

func (ps *Player) GetSteamID() steamid.SID64 {
	return ps.SteamId
}

func (ps *Player) GetName() string {
	return ps.Name
}

func (ps *Player) GetAvatarHash() string {
	return ps.AvatarHash
}

func (ps *Player) IsDisconnected() bool {
	return time.Since(ps.UpdatedOn) > time.Second*6
}

func (ps *Player) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > time.Second*20
}

func (ps *Player) Touch() {
	ps.Dirty = true
}

func firstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

const defaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

// API returns non https urls, this will resolve them over https
const baseAvatarUrl = "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/avatars"

func AvatarUrl(hash string) string {
	avatarHash := defaultAvatarHash
	if hash != "" {
		avatarHash = hash
	}
	return fmt.Sprintf("%s/%s/%s_full.jpg", baseAvatarUrl, firstN(avatarHash, 2), avatarHash)
}

type PlayerCollection []*Player

func (players PlayerCollection) AsAny() []any {
	bl := make([]any, len(players))
	for i, r := range players {
		bl[i] = r
	}
	return bl
}

func NewPlayer(sid64 steamid.SID64, name string) *Player {
	t0 := time.Now()
	return &Player{
		Name:             name,
		AccountCreatedOn: time.Time{},
		Visibility:       ProfileVisibilityPublic,
		SteamId:          sid64,
		SteamIdString:    sid64.String(),
		CreatedOn:        t0,
		UpdatedOn:        t0,
		Dangling:         true,
	}
}

type UserNameHistory struct {
	NameId    int64
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
