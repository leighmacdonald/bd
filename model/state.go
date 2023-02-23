package model

import (
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"net"
	"sync"
	"time"
)

type MarkFunc func(sid64 steamid.SID64, attrs []string) error

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type KickFunc func(userId int64, reason KickReason) error

type ServerState struct {
	ServerName string
	Addr       net.IP
	Port       uint16
	CurrentMap string
	Tags       []string
	LastUpdate time.Time
}

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

type PlayerState struct {
	*sync.RWMutex
	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string

	// PlayerSummary
	RealName         string
	NamePrevious     string
	AccountCreatedOn time.Time

	Visibility ProfileVisibility
	Avatar     fyne.Resource // TODO store somewhere else so we dont couple ui item to the model
	AvatarHash string

	// PlayerBanState
	CommunityBanned  bool
	NumberOfVACBans  int
	LastVACBanOn     *time.Time
	NumberOfGameBans int
	EconomyBan       bool

	// SteamId is the 64bit steamid of the user
	SteamId steamid.SID64

	// First time we see the player
	ConnectedAt time.Time
	Connected   string

	Team Team
	// In game user id
	UserId int64

	Kills  int64
	Deaths int64

	// The users kill count vs this player
	KillsOn   int64
	RageQuits int64
	// The users death count vs this player
	DeathsBy int64
	Ping     int
	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int

	ProfileUpdatedOn time.Time

	// CreatedOn is the first time we have seen the player
	CreatedOn time.Time

	// UpdatedOn is the last time we have interacted with the player
	UpdatedOn time.Time

	// Tracks the duration between announces
	AnnouncedLast time.Time

	Dangling bool

	OurFriend bool

	Dirty bool
}

func (ps *PlayerState) GetSteamID() steamid.SID64 {
	ps.RLock()
	defer ps.RUnlock()
	return ps.SteamId
}

func (ps *PlayerState) GetName() string {
	ps.RLock()
	defer ps.RUnlock()
	return ps.Name
}

func (ps *PlayerState) GetAvatarHash() string {
	ps.RLock()
	defer ps.RUnlock()
	return ps.AvatarHash
}

func (ps *PlayerState) Touch() {
	ps.Lock()
	defer ps.Unlock()
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

func (ps *PlayerState) AvatarUrl() string {
	ps.RLock()
	defer ps.RUnlock()
	avatarHash := defaultAvatarHash
	if ps.AvatarHash != "" {
		avatarHash = ps.AvatarHash
	}
	return fmt.Sprintf("%s/%s/%s_full.jpg", baseAvatarUrl, firstN(avatarHash, 2), avatarHash)
}

func (ps *PlayerState) SetAvatar(hash string, buf []byte) {
	res := fyne.NewStaticResource(fmt.Sprintf("%s.jpg", hash), buf)
	if res == nil {
		log.Printf("Failed to load avatar\n")
		return
	} else {
		ps.Lock()
		ps.Avatar = res
		ps.AvatarHash = rules.HashBytes(buf)
		ps.Unlock()
	}
}

func NewPlayerState(sid64 steamid.SID64, name string) *PlayerState {
	t0 := time.Now()
	return &PlayerState{
		RWMutex:          &sync.RWMutex{},
		Name:             name,
		RealName:         "",
		NamePrevious:     "",
		AccountCreatedOn: time.Time{},
		Avatar:           nil,
		AvatarHash:       "",
		CommunityBanned:  false,
		Visibility:       ProfileVisibilityPublic,
		NumberOfVACBans:  0,
		LastVACBanOn:     nil,
		NumberOfGameBans: 0,
		EconomyBan:       false,
		SteamId:          sid64,
		ConnectedAt:      t0,
		Connected:        "",
		Team:             0,
		UserId:           0,
		KillsOn:          0,
		DeathsBy:         0,
		Ping:             0,
		KickAttemptCount: 0,
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
