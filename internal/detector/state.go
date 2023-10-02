package detector

import (
	"sync"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var errPlayerNotFound = errors.New("player not found")

type playerState struct {
	activePlayers []store.Player
	sync.RWMutex
}

func newPlayerState() *playerState {
	return &playerState{}
}

func (state *playerState) byName(name string) (store.Player, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.Name == name {
			return knownPlayer, nil
		}
	}

	return store.Player{}, errPlayerNotFound
}

func (state *playerState) bySteamID(sid64 steamid.SID64) (store.Player, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64 {
			return knownPlayer, nil
		}
	}

	return store.Player{}, errPlayerNotFound
}

func (state *playerState) update(updated store.Player) {
	state.Lock()
	defer state.Unlock()

	var valid []store.Player //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID == updated.SteamID {
			continue
		}

		valid = append(valid, player)
	}

	valid = append(valid, updated)

	state.activePlayers = valid
}

func (state *playerState) all() []store.Player {
	state.Lock()
	defer state.Unlock()

	return state.activePlayers
}

func (state *playerState) kickable() []store.Player {
	state.Lock()
	defer state.Unlock()

	var kickable []store.Player

	for _, player := range state.activePlayers {
		if len(player.Matches) > 0 && !player.Whitelisted {
			kickable = append(kickable, player)
		}
	}

	return kickable
}

func (state *playerState) replace(replacementPlayers []store.Player) {
	state.Lock()
	defer state.Unlock()

	state.activePlayers = replacementPlayers
}

func (state *playerState) remove(sid64 steamid.SID64) {
	state.Lock()
	defer state.Unlock()

	var valid []store.Player //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID != sid64 {
			continue
		}

		valid = append(valid, player)
	}

	state.activePlayers = valid
}
