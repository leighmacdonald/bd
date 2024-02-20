package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamweb/v2"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
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
		if knownPlayer.Personaname == name {
			return knownPlayer, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (state *playerState) bySteamID(sid64 steamid.SID64) (Player, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64.Int64() {
			return knownPlayer, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (state *playerState) update(updated Player) {
	state.Lock()
	defer state.Unlock()

	valid := make([]Player, len(state.activePlayers))

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
		if len(player.Matches) > 0 && !player.Whitelist {
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
		if player.SteamID != sid64.Int64() {
			continue
		}

		valid = append(valid, player)
	}

	state.activePlayers = valid
}

// checkPlayerStates will run a check against the current player state for matches.
func (state *playerState) checkPlayerState(ctx context.Context, re *rules.Engine, player *Player, validTeam Team) {
	if player.isDisconnected() || len(player.Matches) > 0 {
		return
	}

	if matchSteam := re.MatchSteam(player.SID64()); matchSteam != nil { //nolint:nestif
		player.Matches = matchSteam

		if validTeam == player.Team {
			d.announceMatch(ctx, player, matchSteam)
			d.players.update(player)
		}
	} else if player.Personaname != "" {
		if matchName := d.rules.MatchName(player.Name); matchName != nil && validTeam == player.Team {
			player.Matches = matchName

			if validTeam == player.Team {
				d.announceMatch(ctx, player, matchName)
				d.players.update(player)
			}
		}
	}

	if player.Dirty {
		if errSave := d.dataStore.SavePlayer(ctx, &player); errSave != nil {
			slog.Error("Failed to save dirty player state", errAttr(errSave))

			continue
		}

		player.Dirty = false
	}

}

// cleanupHandler is used to track of players and their expiration times. It will remove and reset expired players
// and server from the current known state once they have been disconnected for the timeout periods.
func (state *playerState) cleanupHandler(ctx context.Context) {
	const disconnectMsg = "Disconnected"

	defer slog.Debug("cleanupHandler exited")

	deleteTimer := time.NewTicker(time.Second * time.Duration(d.Settings().PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			settings := d.Settings()

			slog.Debug("Delete update input received", slog.String("state", "start"))
			d.serverMu.Lock()
			if time.Since(d.server.LastUpdate) > time.Second*time.Duration(settings.PlayerDisconnectTimeout) {
				name := d.server.ServerName
				if !strings.HasPrefix(name, disconnectMsg) {
					name = fmt.Sprintf("%s %s", disconnectMsg, name)
				}

				d.server = &Server{ServerName: name}
			}

			d.serverMu.Unlock()

			for _, player := range d.players.all() {
				if player.IsDisconnected() {
					player.IsConnected = false
					d.players.update(player)
				}

				if player.IsExpired() {
					d.players.remove(player.SteamID)
					slog.Debug("Flushing expired player", slog.Int64("steam_id", player.SteamID.Int64()))
				}
			}

			slog.Debug("Delete update input received", slog.String("state", "end"))

			deleteTimer.Reset(time.Second * time.Duration(settings.PlayerExpiredTimeout))
		}
	}
}

type gameState struct {
	updateChan chan updateStateEvent
	settings   userSettings
	players    *playerState
	server     serverState
	store      store.Querier
	rcon       rconConnection
	g15        g15Parser
}

func newGameState(store store.Querier, settings userSettings, playerState *playerState, rcon rconConnection) *gameState {
	return &gameState{
		store:      store,
		settings:   settings,
		players:    playerState,
		rcon:       rcon,
		server:     serverState{},
		updateChan: make(chan updateStateEvent),
	}
}

func (s *gameState) start(ctx context.Context) {
	statusTimer := time.NewTicker(DurationStatusUpdateTimer)
	for {
		select {
		case <-statusTimer.C:
			if errUpdate := s.updatePlayerState(ctx); errUpdate != nil {
				slog.Debug("Failed to query state", errAttr(errUpdate))

				continue
			}
		case <-ctx.Done():
			return
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
			case updateBans:
				evt, ok := update.data.(steamweb.PlayerBanState)
				if !ok {
					continue
				}

				s.onBans(evt)
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
		}
	}
}

func (s *gameState) onKill(evt killEvent) {
	ourSid := s.settings.SteamID

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

	if target.SteamID == ourSid.Int64() {
		src.DeathsBy++
	}

	if src.SteamID == ourSid.Int64() {
		target.KillsOn++
	}

	s.players.update(src)
	s.players.update(target)
}

func (s *gameState) onBans(evt steamweb.PlayerBanState) {
	player, errPlayer := s.players.bySteamID(evt.SteamID)
	if errPlayer != nil {
		return
	}

	player.VacBans = int64(evt.NumberOfVACBans)
	player.GameBans = int64(evt.NumberOfGameBans)
	player.CommunityBanned = evt.CommunityBanned
	player.EconomyBan = evt.EconomyBan

	if evt.DaysSinceLastBan > 0 {
		subTime := time.Now().AddDate(0, 0, -evt.DaysSinceLastBan)
		player.LastVacBanOn.Scan(subTime)
	}

	s.players.update(player)
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
		if errAddName := s.store.UserNameSave(ctx, store.UserNameSaveParams{
			SteamID:   player.SteamID,
			Name:      player.Personaname,
			CreatedOn: time.Now(),
		}); errAddName != nil {
			slog.Error("Could not save new user name", errAttr(errAddName))
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

func (s *gameState) onUpdateMessage(ctx context.Context, evt messageEvent) {
	player, errPlayer := s.players.byName(evt.name)
	if errPlayer != nil {
		return
	}

	if errUm := s.store.MessageSave(ctx, store.MessageSaveParams{
		SteamID:   player.SteamID,
		Message:   evt.message,
		CreatedOn: evt.createdAt,
		Team:      evt.teamOnly,
	}); errUm != nil {
		slog.Error("Failed to handle user message", errAttr(errUm))
	}

	s.players.update(player)
}

// updatePlayerState fetches the current game state over rcon using both the `status` and `g15_dumpplayer` command
// output. The results are then parsed and applied to the current player and server states.
func (s *gameState) updatePlayerState(ctx context.Context) error {
	// Sent to client, response via log output
	_, errStatus := s.rcon.exec(ctx, "status", true)
	if errStatus != nil {
		return errors.Join(errStatus, errRCONStatus)
	}

	dumpPlayer, errDumpPlayer := s.rcon.exec(ctx, "g15_dumpplayer", true)
	if errDumpPlayer != nil {
		return errors.Join(errDumpPlayer, errRCONG15)
	}

	var dump DumpPlayer
	if errG15 := s.g15.Parse(bytes.NewBufferString(dumpPlayer), &dump); errG15 != nil {
		return errors.Join(errG15, errG15Parse)
	}

	for index, sid := range dump.SteamID {
		if index == 0 || index > 32 || !sid.Valid() {
			// Actual data always starts at 1
			continue
		}

		player, errPlayer := s.players.bySteamID(sid)
		if errPlayer != nil {
			// status command is what we use to add players to the active game.
			continue
		}

		player.MapTime = time.Since(player.MapTimeStart).Seconds()

		if player.Kills > 0 {
			player.KPM = float64(player.Kills) / (player.MapTime / 60)
		}

		player.Ping = dump.Ping[index]
		player.Score = dump.Score[index]
		player.Deaths = dump.Deaths[index]
		player.IsConnected = dump.Connected[index]
		player.Team = Team(dump.Team[index])
		player.Alive = dump.Alive[index]
		player.Health = dump.Health[index]
		player.Valid = dump.Valid[index]
		player.UserID = dump.UserID[index]
		player.UpdatedOn = time.Now()

		s.players.update(player)
	}

	return nil
}
