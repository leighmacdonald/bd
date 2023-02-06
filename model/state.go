package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"sync"
	"time"
)

type GameState struct {
	*sync.RWMutex
	Server   *ServerState
	Players  PlayersStateCollection
	Messages []UserMessage
}

type ServerState struct {
	ServerName string
	CurrentMap string
}

func NewGameState() *GameState {
	return &GameState{
		RWMutex: &sync.RWMutex{},
		Server: &ServerState{
			ServerName: "n/a",
			CurrentMap: "n/a",
		},
		Players: nil,
	}
}

type PlayersStateCollection []*PlayerState

type PlayerState struct {
	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string

	// SteamId is the 64bit steamid of the user
	SteamId steamid.SID64

	// First time we see the player
	ConnectedAt time.Time
	Connected   string

	// We got their disconnect message. This is used to calculate when to remove
	// the player from the slice as there is a grace period on disconnect before dropping
	DisconnectedAt *time.Time

	Team Team
	// In game user id
	UserId int64

	// The users kill count vs this player
	KillsOn int

	// The users death count vs this player
	DeathsBy int
	Ping     int
	// Incremented on each kick attempt. Used to cycle through and not attempt the same bot
	KickAttemptCount int

	BanState *steamweb.PlayerBanState
	Summary  *steamweb.PlayerSummary

	// CreatedOn is the first time we have seen the player
	CreatedOn time.Time

	// UpdatedOn is the last time we have interacted with the player
	UpdatedOn time.Time

	// Dangling tracks if the player has been inserted into db yet
	Dangling bool
}

func NewPlayerState(sid64 steamid.SID64, name string) PlayerState {
	t0 := time.Now()
	return PlayerState{
		Name:             name,
		SteamId:          sid64,
		ConnectedAt:      t0,
		Team:             0,
		UserId:           0,
		KillsOn:          0,
		DeathsBy:         0,
		KickAttemptCount: 0,
		BanState:         nil,
		Summary:          nil,
		CreatedOn:        t0,
		UpdatedOn:        t0,
		Dangling:         true,
	}
}
