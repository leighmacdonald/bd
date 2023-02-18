package model

import (
	"crypto/sha1"
	_ "embed"
	"encoding/hex"
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"time"
)

type ServerState struct {
	ServerName string
	CurrentMap string
}

type PlayerState struct {
	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string

	// PlayerSummary
	RealName         string
	NamePrevious     string
	AccountCreatedOn time.Time
	Visibility       int
	Avatar           fyne.Resource
	AvatarHash       string

	// PlayerBanState
	CommunityBanned  bool
	NumberOfVACBans  int
	DaysSinceLastBan int
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

	Kills  int
	Deaths int

	// The users kill count vs this player
	KillsOn   int
	RageQuits int
	// The users death count vs this player
	DeathsBy int
	Ping     int
	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int

	// CreatedOn is the first time we have seen the player
	CreatedOn time.Time

	// UpdatedOn is the last time we have interacted with the player
	UpdatedOn time.Time

	Dangling bool
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
		ps.Avatar = res
		ps.AvatarHash = HashBytes(buf)
	}
}

func NewPlayerState(sid64 steamid.SID64, name string) PlayerState {
	t0 := time.Now()
	return PlayerState{
		Name:             name,
		RealName:         "",
		NamePrevious:     "",
		AccountCreatedOn: time.Time{},
		Avatar:           nil,
		AvatarHash:       "",
		CommunityBanned:  false,
		NumberOfVACBans:  0,
		DaysSinceLastBan: 0,
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

func init() {
	//avatar, errDecode := jpeg.Decode(bytes.NewReader(defaultAvatarJpeg))
	//if errDecode != nil {
	//	log.Panic("Failed to decode default profile avatar")
	//}
	//DefaultAvatarImage = avatar
}

func HashBytes(b []byte) string {
	hash := sha1.New()
	hash.Write(b)
	return hex.EncodeToString(hash.Sum(nil))
}
