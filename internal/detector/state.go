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
	mu            sync.RWMutex
}

func newPlayerState() *playerState {
	return &playerState{}
}

func (state *playerState) byName(name string) (store.Player, error) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.Name == name {
			return knownPlayer, nil
		}
	}

	return store.Player{}, errPlayerNotFound
}

func (state *playerState) bySteamID(sid64 steamid.SID64) (store.Player, error) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64 {
			return knownPlayer, nil
		}
	}

	return store.Player{}, errPlayerNotFound
}

func (state *playerState) update(updated store.Player) {
	state.mu.Lock()
	defer state.mu.Unlock()

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
	state.mu.RLock()
	defer state.mu.RUnlock()

	return state.activePlayers
}

func (state *playerState) replace(replacementPlayers []store.Player) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.activePlayers = replacementPlayers
}

func (state *playerState) remove(sid64 steamid.SID64) {
	state.mu.Lock()
	defer state.mu.Unlock()

	var valid []store.Player //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID != sid64 {
			continue
		}

		valid = append(valid, player)
	}

	state.activePlayers = valid
}
