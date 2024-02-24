package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var errPlayerNotFound = errors.New("player not found")

type serverState struct {
	ServerName string    `json:"server_name"`
	Addr       net.IP    `json:"-"`
	Port       uint16    `json:"-"`
	CurrentMap string    `json:"current_map"`
	Tags       []string  `json:"-"`
	LastUpdate time.Time `json:"last_update"`
}

type playerStates struct {
	activePlayers []PlayerState
	sync.RWMutex
}

func newPlayerStates() *playerStates {
	return &playerStates{}
}

func (state *playerStates) current() []PlayerState {
	state.RLock()
	defer state.RUnlock()

	return state.activePlayers
}

func (state *playerStates) byName(name string) (PlayerState, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.Personaname == name {
			return knownPlayer, nil
		}
	}

	return PlayerState{}, errPlayerNotFound
}

func (state *playerStates) bySteamID(sid64 steamid.SID64) (PlayerState, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64 {
			return knownPlayer, nil
		}
	}

	return PlayerState{}, errPlayerNotFound
}

func (state *playerStates) update(updated PlayerState) {
	state.Lock()
	defer state.Unlock()

	var valid []PlayerState //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID == updated.SteamID {
			continue
		}

		valid = append(valid, player)
	}

	valid = append(valid, updated)

	state.activePlayers = valid
}

func (state *playerStates) all() []PlayerState {
	state.Lock()
	defer state.Unlock()

	return state.activePlayers
}

func (state *playerStates) kickable() []PlayerState {
	state.Lock()
	defer state.Unlock()

	var kickable []PlayerState

	for _, player := range state.activePlayers {
		if len(player.Matches) > 0 && !player.Whitelist {
			kickable = append(kickable, player)
		}
	}

	return kickable
}

func (state *playerStates) replace(replacementPlayers []PlayerState) {
	state.Lock()
	defer state.Unlock()

	state.activePlayers = replacementPlayers
}

func (state *playerStates) remove(sid64 steamid.SID64) {
	state.Lock()
	defer state.Unlock()

	var valid []PlayerState //nolint:prealloc

	for _, player := range state.activePlayers {
		if player.SteamID != sid64 {
			continue
		}

		valid = append(valid, player)
	}

	state.activePlayers = valid
}

// checkPlayerStates will run a check against the current player state for matches.
func (state *playerStates) checkPlayerState(ctx context.Context, re *rules.Engine, player PlayerState, validTeam Team, announcer bigBrother) {
	if player.isDisconnected() || len(player.Matches) > 0 {
		return
	}

	if matchSteam := re.MatchSteam(player.SID64()); matchSteam != nil {
		player.Matches = matchSteam

		if validTeam == player.Team {
			announcer.announceMatch(ctx, player, matchSteam)
			// state.update(*player)
		}
	} else if player.Personaname != "" {
		if matchName := re.MatchName(player.Personaname); matchName != nil && validTeam == player.Team {
			player.Matches = matchName

			if validTeam == player.Team {
				announcer.announceMatch(ctx, player, matchName)
				// state.update(*player)
			}
		}
	}
}

type gameState struct {
	mu         *sync.RWMutex
	updateChan chan updateStateEvent
	settings   *settingsManager
	players    *playerStates
	server     serverState
	store      store.Querier
	rcon       rconConnection
}

func newGameState(store store.Querier, settings *settingsManager, playerState *playerStates, rcon rconConnection) *gameState {
	return &gameState{
		mu:         &sync.RWMutex{},
		store:      store,
		settings:   settings,
		players:    playerState,
		rcon:       rcon,
		server:     serverState{},
		updateChan: make(chan updateStateEvent),
	}
}

func (s *gameState) start(ctx context.Context) {
	for {
		select {

		case update := <-s.updateChan:
			slog.Debug("Game state update input received", slog.String("kind", update.kind.String()))

			if update.kind == updateStatus && !update.source.Valid() {
				continue
			}

			switch update.kind { //nolint:exhaustive
			case updateKill:
				evt, ok := update.data.(killEvent)
				if !ok {
					continue
				}

				s.onKill(evt)
			case updateKickAttempts:
				s.onKickAttempt(update.source)
			case updateStatus:
				evt, ok := update.data.(statusEvent)
				if !ok {
					continue
				}

				s.onStatus(ctx, update.source, evt)
			case updateTags:
				evt, ok := update.data.(tagsEvent)
				if !ok {
					continue
				}

				s.onTags(evt)
			case updateHostname:
				evt, ok := update.data.(hostnameEvent)
				if !ok {
					continue
				}

				s.onHostname(evt)
			case updateMap:
				evt, ok := update.data.(mapEvent)
				if !ok {
					continue
				}

				s.onMapName(evt)
			case changeMap:
				s.onMapChange()
			default:
				slog.Debug("unhandled state update case")
			}
		case <-ctx.Done():
			return
		}
	}
}

// cleanupHandler is used to track of players and their expiration times. It will remove and reset expired players
// and server from the current known state once they have been disconnected for the timeout periods.
func (s *gameState) startCleanupHandler(ctx context.Context, settingsMgr *settingsManager) {
	const disconnectMsg = "Disconnected"

	deleteTimer := time.NewTicker(time.Second * time.Duration(settingsMgr.Settings().PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			settings := settingsMgr.Settings()

			slog.Debug("Delete update input received", slog.String("state", "start"))

			s.mu.Lock()
			if time.Since(s.server.LastUpdate) > time.Second*time.Duration(settings.PlayerDisconnectTimeout) {
				name := s.server.ServerName
				if !strings.HasPrefix(name, disconnectMsg) {
					name = fmt.Sprintf("%s %s", disconnectMsg, name)
				}

				s.server = serverState{ServerName: name}
			}

			s.mu.Unlock()

			for _, player := range s.players.all() {
				if player.isDisconnected() {
					player.IsConnected = false
					s.players.update(player)
				}

				if player.IsExpired() {
					s.players.remove(player.SID64())
					slog.Debug("Flushing expired player", slog.String("steam_id", player.SID64().String()))
				}
			}

			slog.Debug("Delete update input received", slog.String("state", "end"))

			deleteTimer.Reset(time.Second * time.Duration(settings.PlayerExpiredTimeout))
		}
	}
}

func (s *gameState) CurrentServerState() serverState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.server
}

func (s *gameState) onKill(evt killEvent) {
	ourSid := s.settings.Settings().SteamID

	src, srcErr := s.players.byName(evt.sourceName)
	if srcErr != nil {
		return
	}

	target, targetErr := s.players.byName(evt.sourceName)
	if targetErr != nil {
		return
	}

	src.Kills++
	target.Deaths++

	if target.SteamID == ourSid {
		src.DeathsBy++
	}

	if src.SteamID == ourSid {
		target.KillsOn++
	}

	s.players.update(src)
	s.players.update(target)
}

func (s *gameState) onKickAttempt(steamID steamid.SID64) {
	player, errPlayer := s.players.bySteamID(steamID)
	if errPlayer != nil {
		return
	}

	player.KickAttemptCount++

	s.players.update(player)
}

func (s *gameState) onStatus(ctx context.Context, steamID steamid.SID64, evt statusEvent) {
	player, errPlayer := getPlayerOrCreate(ctx, s.store, s.players, steamID)
	if errPlayer != nil {
		slog.Error("Failed to get or create player", errAttr(errPlayer))

		return
	}

	player.Ping = evt.ping
	player.Connected = evt.connected.Seconds()
	player.UpdatedOn = time.Now()
	player.UserID = evt.userID

	if player.Personaname != evt.name {
		player.Personaname = evt.name
		errAddName := s.store.UserNameSave(ctx, store.UserNameSaveParams{
			SteamID:   player.SteamID.Int64(),
			Name:      player.Personaname,
			CreatedOn: time.Now(),
		})
		if errAddName != nil {
			var sqliteErr *sqlite.Error
			if errors.As(errAddName, &sqliteErr) {
				if sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY {
					slog.Error("Could not save new user name", errAttr(errAddName))
				}
			} else {
				slog.Error("Could not save new user name", errAttr(errAddName))
			}
		}
	}

	s.players.update(player)

	slog.Debug("Player status updated",
		slog.String("sid", steamID.String()),
		slog.Int("tags", evt.ping),
		slog.Int("uid", evt.userID),
		slog.String("name", evt.name),
		slog.Int("connected", int(evt.connected.Seconds())))
}

func (s *gameState) onTags(evt tagsEvent) {
	s.server.Tags = evt.tags
	s.server.LastUpdate = time.Now()

	slog.Debug("Tags updated", slog.String("tags", strings.Join(evt.tags, ",")))
}

func (s *gameState) onHostname(evt hostnameEvent) {
	s.server.ServerName = evt.hostname
	s.server.LastUpdate = time.Now()

	slog.Debug("Hostname changed", slog.String("hostname", evt.hostname))
}

func (s *gameState) onMapName(evt mapEvent) {
	s.server.CurrentMap = evt.mapName

	slog.Debug("Map changed", slog.String("map", evt.mapName))
}

func (s *gameState) onMapChange() {
	for _, curPlayer := range s.players.all() {
		player := curPlayer

		player.Kills = 0
		player.Deaths = 0
		player.MapTimeStart = time.Now()
		player.MapTime = 0

		s.players.update(player)
	}

	s.server.CurrentMap = ""
	s.server.ServerName = ""
}
