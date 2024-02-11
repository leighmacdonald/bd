package main

import (
	"sync"

	"errors"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var errPlayerNotFound = errors.New("player not found")

type playerState struct {
	activePlayers []Player
	sync.RWMutex
}

func newPlayerState() *playerState {
	return &playerState{}
}

func (state *playerState) byName(name string) (Player, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.Name == name {
			return knownPlayer, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (state *playerState) bySteamID(sid64 steamid.SID64) (Player, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64 {
			return knownPlayer, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (state *playerState) update(updated Player) {
	state.Lock()
	defer state.Unlock()

	var valid []Player //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID == updated.SteamID {
			continue
		}

		valid = append(valid, player)
	}

	valid = append(valid, updated)

	state.activePlayers = valid
}

func (state *playerState) all() []Player {
	state.Lock()
	defer state.Unlock()

	return state.activePlayers
}

func (state *playerState) kickable() []Player {
	state.Lock()
	defer state.Unlock()

	var kickable []Player

	for _, player := range state.activePlayers {
		if len(player.Matches) > 0 && !player.Whitelisted {
			kickable = append(kickable, player)
		}
	}

	return kickable
}

func (state *playerState) replace(replacementPlayers []Player) {
	state.Lock()
	defer state.Unlock()

	state.activePlayers = replacementPlayers
}

func (state *playerState) remove(sid64 steamid.SID64) {
	state.Lock()
	defer state.Unlock()

	var valid []Player //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID != sid64 {
			continue
		}

		valid = append(valid, player)
	}

	state.activePlayers = valid
}
