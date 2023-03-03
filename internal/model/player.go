package model

import (
	"fmt"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"sync"
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
	sync.RWMutex
	// - Permanent storage backed

	// SteamId is the 64bit steamid of the user
	SteamId steamid.SID64

	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string

	// CreatedOn is the first time we have seen the player
	CreatedOn time.Time

	// UpdatedOn is the last time we have received a status update from rcon
	// This is used to calculate when we consider the player disconnected and also when
	// they are expired and should be removed from the player pool entirely.
	UpdatedOn        time.Time
	ProfileUpdatedOn time.Time

	// The users kill count vs this player
	KillsOn   int
	RageQuits int
	DeathsBy  int

	Notes       string
	Whitelisted bool

	// PlayerSummary
	RealName         string
	NamePrevious     string
	AccountCreatedOn time.Time

	Visibility ProfileVisibility
	AvatarHash string

	// PlayerBanState
	CommunityBanned  bool
	NumberOfVACBans  int
	LastVACBanOn     *time.Time
	NumberOfGameBans int
	EconomyBan       bool

	// - Parsed Ephemeral data

	// tf_lobby_debug
	Team Team

	// status
	// Connected is how long the user has been in the server
	Connected time.Duration
	// In game user id
	UserId int64
	Ping   int

	// Parsed stats from logs
	Kills  int
	Deaths int

	// - Misc

	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int

	// Tracks the duration between announces to chat
	AnnouncedLast time.Time

	// Dangling will be true when the user is new and doesn't have a physical entry in the database yet.
	Dangling bool

	OurFriend bool

	// Dirty indicates that state which has database backed fields has been changed and need to be saved
	Dirty bool

	Match *rules.MatchResult
}

func (ps *Player) IsMatched() bool {
	return ps.Match != nil
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
	return time.Since(ps.UpdatedOn) > DurationDisconnected
}

func (ps *Player) IsExpired() bool {
	return time.Since(ps.UpdatedOn) > DurationPlayerExpired
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
