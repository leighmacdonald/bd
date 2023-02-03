package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"log"
	"sync"
	"time"
)

type ServerState struct {
	*sync.RWMutex
	Server     string
	CurrentMap string
	Players    []*PlayerState
}

func NewServerState() *ServerState {
	return &ServerState{
		RWMutex:    &sync.RWMutex{},
		Server:     "n/a",
		CurrentMap: "n/a",
		Players:    []*PlayerState{},
	}
}

type PlayerState struct {
	// Name is the current in-game name of the player. This can be different from their name via steam api when
	// using changer/stealers
	Name string

	// SteamId is the 64bit steamid of the user
	SteamId steamid.SID64

	// First time we see the player
	ConnectedAt time.Time

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

func (ps *PlayerState) Update() {
	wg := &sync.WaitGroup{}
	var (
		banState steamweb.PlayerBanState
		summary  steamweb.PlayerSummary
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		bans, errBans := steamweb.GetPlayerBans(steamid.Collection{ps.SteamId})
		if errBans != nil {
			log.Printf("Failed to fetch bans: %v", errBans)
			return
		}
		if len(bans) != 1 {
			return
		}
		banState = bans[0]
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		summaries, errBans := steamweb.PlayerSummaries(steamid.Collection{ps.SteamId})
		if errBans != nil {
			log.Printf("Failed to fetch summaries: %v", errBans)
			return
		}
		if len(summaries) != 1 {
			return
		}
		summary = summaries[0]
	}()
	wg.Wait()
	ps.BanState = &banState
	ps.Summary = &summary
}
