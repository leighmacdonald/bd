package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v4/steamid"
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

// bySteamID returns a player currently being tracked in the game state. If connectedOnly is true, then
// players who have timed out already are ignored.
func (state *playerStates) bySteamID(sid64 steamid.SteamID) (PlayerState, error) {
	state.RLock()
	defer state.RUnlock()

	for _, knownPlayer := range state.activePlayers {
		if knownPlayer.SteamID == sid64 {
			return knownPlayer, nil
		}
	}

	return PlayerState{}, errPlayerNotFound
}

// update replaces the current players state with the new state provided.
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
	state.RLock()
	defer state.RUnlock()

	return state.activePlayers
}

func (state *playerStates) replace(replacementPlayers []PlayerState) {
	state.Lock()
	defer state.Unlock()

	state.activePlayers = replacementPlayers
}

func (state *playerStates) remove(sid64 steamid.SteamID) {
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
func (state *playerStates) checkPlayerState(ctx context.Context, re *rules.Engine, player PlayerState, validTeam Team, announcer overwatch) {
	if !player.IsConnected || len(player.Matches) > 0 {
		return
	}

	if matchSteam := re.MatchSteam(player.SteamID); matchSteam != nil {
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
	mu                 *sync.RWMutex
	playerDataChan     chan playerDataUpdate
	profileUpdateQueue chan steamid.SteamID
	eventChan          chan LogEvent
	settings           configManager
	players            *playerStates
	db                 store.Querier
	server             serverState
	store              store.Querier
	rcon               rconConnection
}

func newGameState(store store.Querier, settings configManager, playerState *playerStates, rcon rconConnection,
	db store.Querier,
) *gameState {
	return &gameState{
		mu:                 &sync.RWMutex{},
		store:              store,
		settings:           settings,
		players:            playerState,
		rcon:               rcon,
		db:                 db,
		server:             serverState{},
		playerDataChan:     make(chan playerDataUpdate),
		eventChan:          make(chan LogEvent),
		profileUpdateQueue: make(chan steamid.SteamID),
	}
}

func (s *gameState) start(ctx context.Context) {
	for {
		select {
		case playerData := <-s.playerDataChan:
			settings, errSettings := s.settings.settings(ctx)
			if errSettings != nil {
				slog.Error("Failed to read settings", errAttr(errSettings))
				continue
			}
			s.applyRemoteData(ctx, playerData, settings)
		case evt := <-s.eventChan:
			slog.Debug("received event", slog.Int("type", int(evt.Type)))
			switch evt.Type { //nolint:exhaustive
			case EvtMap:
				s.onMapName(mapEvent{mapName: evt.MetaData})
			case EvtHostname:
				s.onHostname(hostnameEvent{hostname: evt.MetaData})
			case EvtTags:
				s.onTags(tagsEvent{tags: strings.Split(evt.MetaData, ",")})
			case EvtAddress:
				pcs := strings.Split(evt.MetaData, ":")

				_, errPort := strconv.ParseUint(pcs[1], 10, 16)
				if errPort != nil {
					slog.Error("Failed to parse port: %v", errAttr(errPort), slog.String("port", pcs[1]))

					continue
				}

				parsedIP := net.ParseIP(pcs[0])
				if parsedIP == nil {
					slog.Error("Failed to parse ip", slog.String("ip", pcs[0]))

					continue
				}
			case EvtStatusID:
				s.onStatus(ctx, evt.PlayerSID, statusEvent{
					ping:      evt.PlayerPing,
					userID:    evt.UserID,
					name:      evt.Player,
					connected: evt.PlayerConnected,
				})
			case EvtDisconnect:
				s.onMapChange()
			case EvtKill:
				settings, errSettings := s.settings.settings(ctx)
				if errSettings != nil {
					slog.Error("Failed to read settings", errAttr(errSettings))
					continue
				}
				s.onKill(killEvent{victimName: evt.Victim, sourceName: evt.Player}, settings)
			case EvtMsg:
			case EvtConnect:
			case EvtLobby:
			case EvtAny:
			}
		case <-ctx.Done():
			return
		}
	}
}

// cleanupHandler is used to track of players and their expiration times. It will remove and reset expired players
// and server from the current known state once they have been disconnected for the timeout periods.
func (s *gameState) startCleanupHandler(ctx context.Context, settingsMgr *configManager) {
	const disconnectMsg = "Disconnected"

	settings, errSettings := settingsMgr.settings(ctx)
	if errSettings != nil {
		slog.Error("Failed to read settings", errAttr(errSettings))
		return
	}

	deleteTimer := time.NewTicker(time.Second * time.Duration(settings.PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			settings, errSettings = settingsMgr.settings(ctx)
			if errSettings != nil {
				slog.Error("Failed to read settings", errAttr(errSettings))
				return
			}

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
				if player.IsConnected {
					player.IsConnected = false
					s.players.update(player)
				}

				if player.IsExpired() {
					s.players.remove(player.SteamID)
					slog.Debug("Flushing expired player", slog.String("steam_id", player.SteamID.String()))
				}
			}

			slog.Debug("Delete update input received", slog.String("state", "end"))

			deleteTimer.Reset(time.Second * time.Duration(settings.PlayerExpiredTimeout))
		}
	}
}

// applyRemoteData updates the current player states with new incoming remote data sources.
func (s *gameState) applyRemoteData(ctx context.Context, data playerDataUpdate, settings userSettings) {
	player, errPlayer := s.players.bySteamID(data.steamID)
	if errPlayer != nil {
		return
	}

	// Summary
	player.AvatarHash = data.summary.AvatarHash
	player.AccountCreatedOn = time.Unix(int64(data.summary.TimeCreated), 0)
	player.Visibility = int64(data.summary.CommunityVisibilityState)

	// Bans
	player.CommunityBanned = data.bans.CommunityBanned
	player.GameBans = int64(data.bans.NumberOfGameBans)
	player.VacBans = int64(data.bans.NumberOfVACBans)
	player.EconomyBan = data.bans.EconomyBan
	if player.VacBans > 0 && data.bans.DaysSinceLastBan > 0 {
		player.LastVacBanOn = time.Now().AddDate(0, 0, -data.bans.DaysSinceLastBan).Unix()
	}

	// Sourcebans
	player.Sourcebans = data.sourcebans

	// Friends
	player.Friends = data.friends
	for _, friend := range data.friends {
		if friend.SteamID == settings.SteamID {
			player.OurFriend = true
			break
		}
	}

	// meta
	player.UpdatedOn = time.Now()
	player.ProfileUpdatedOn = player.UpdatedOn

	if errSave := s.db.PlayerUpdate(ctx, player.toUpdateParams()); errSave != nil {
		if errSave.Error() != "sql: database is closed" {
			slog.Error("Failed to save updated player data",
				slog.String("sid", player.SteamID.String()), errAttr(errSave))
		}
		return
	}
	s.players.update(player)
}

func (s *gameState) CurrentServerState() serverState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.server
}

func (s *gameState) onKill(evt killEvent, settings userSettings) {
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

	if target.SteamID == settings.SteamID {
		src.DeathsBy++
	}

	if src.SteamID == settings.SteamID {
		target.KillsOn++
	}

	s.players.update(src)
	s.players.update(target)
}

func (s *gameState) getPlayerOrCreate(ctx context.Context, steamID steamid.SteamID) (PlayerState, error) {
	player, errPlayer := s.players.bySteamID(steamID)
	if errPlayer != nil {
		if !errors.Is(errPlayer, errPlayerNotFound) {
			return PlayerState{}, errPlayer
		}

		createdPlayer, errCreate := loadPlayerOrCreate(ctx, s.store, steamID)
		if errCreate != nil {
			return PlayerState{}, errCreate
		}

		player = createdPlayer
	}

	return player, nil
}

func (s *gameState) onStatus(ctx context.Context, steamID steamid.SteamID, evt statusEvent) {
	player, errPlayer := s.getPlayerOrCreate(ctx, steamID)
	if errPlayer != nil {
		return
	}

	player.MapTimeStart = time.Now()
	player.Ping = evt.ping
	player.Connected = evt.connected
	player.UpdatedOn = time.Now()
	player.IsConnected = true
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

	// Trigger update of external data if it's been long enough, or the player is new to us.
	if time.Since(player.ProfileUpdatedOn) > time.Hour*24 {
		// TODO save friends and sourcebans data locally
		slog.Debug("Updating user data", sidAttr(steamID))
		s.profileUpdateQueue <- steamID
	}

	slog.Debug("Player status updated",
		slog.String("sid", steamID.String()),
		slog.Int("tags", evt.ping),
		slog.Int("uid", evt.userID),
		slog.String("name", evt.name),
		slog.Int("connected", int(evt.connected.Seconds())))
}

func (s *gameState) onTags(evt tagsEvent) {
	s.mu.Lock()
	s.server.Tags = evt.tags
	s.server.LastUpdate = time.Now()
	s.mu.Unlock()

	slog.Debug("Tags updated", slog.String("tags", strings.Join(evt.tags, ",")))
}

func (s *gameState) onHostname(evt hostnameEvent) {
	s.mu.Lock()
	s.server.ServerName = evt.hostname
	s.server.LastUpdate = time.Now()
	s.mu.Unlock()

	slog.Debug("Hostname changed", slog.String("hostname", evt.hostname))
}

func (s *gameState) onMapName(evt mapEvent) {
	s.mu.Lock()
	s.server.CurrentMap = evt.mapName
	s.mu.Unlock()

	slog.Debug("Map changed", slog.String("map", evt.mapName))
}

func (s *gameState) onMapChange() {
	players := s.players.all()
	for _, curPlayer := range players {
		player := curPlayer
		player.IsConnected = true
		player.Kills = 0
		player.Deaths = 0
		player.MapTimeStart = time.Now()
		player.MapTime = 0

		s.players.update(player)
	}
	s.mu.Lock()
	s.server.CurrentMap = ""
	s.server.ServerName = ""
	s.mu.Unlock()
}
