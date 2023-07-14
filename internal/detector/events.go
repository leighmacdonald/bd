package detector

import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamweb/v2"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type updateType int

const (
	updateKill updateType = iota
	updateProfile
	updateBans
	updateStatus
	updateMessage
	updateLobby
	updateMap
	updateHostname
	updateTags
	updateAddress
	changeMap
	updateMark
	updateWhitelist
	updateTeam
	updateKickAttempts
	updateNotes
	playerTimeout
	updateTestPlayer = 1000
)

type killEvent struct {
	sourceName string
	victimName string
}

type lobbyEvent struct {
	team store.Team
}

type statusEvent struct {
	ping      int
	userID    int64
	name      string
	connected time.Duration
}

type updateStateEvent struct {
	kind   updateType
	source steamid.SID64
	data   any
}

func newPlayerTimeoutEvent(sid steamid.SID64) updateStateEvent {
	return updateStateEvent{
		kind:   playerTimeout,
		source: sid,
	}
}

type markEvent struct {
	tags    []string
	addMark bool
}

func newKickAttemptEvent(sid steamid.SID64) updateStateEvent {
	return updateStateEvent{
		kind:   updateKickAttempts,
		source: sid,
	}
}

func newMarkEvent(sid steamid.SID64, tags []string, addMark bool) updateStateEvent {
	return updateStateEvent{
		kind:   updateMark,
		source: sid,
		data: markEvent{
			tags:    tags,
			addMark: addMark, // false to unmark
		},
	}
}

type randomPlayerStateEvent struct {
	updatedOn time.Time
	connected float64
	ping      int
	kills     int
	deaths    int
}

type noteEvent struct {
	body string
}

func newRandomPlayerStateEvent(sid steamid.SID64, connected float64) updateStateEvent {
	return updateStateEvent{
		kind:   updateTestPlayer,
		source: sid,
		data: randomPlayerStateEvent{
			updatedOn: time.Now(),
			connected: connected + 5,
			ping:      rand.Intn(110), //nolint:gosec
			kills:     rand.Intn(50),  //nolint:gosec
			deaths:    rand.Intn(30),  //nolint:gosec
		},
	}
}

func newNoteEvent(sid steamid.SID64, body string) updateStateEvent {
	return updateStateEvent{
		kind:   updateNotes,
		source: sid,
		data: noteEvent{
			body: body,
		},
	}
}

type whitelistEvent struct {
	tags         []string
	addWhitelist bool
}

func newWhitelistEvent(sid steamid.SID64, addWhitelist bool) updateStateEvent {
	return updateStateEvent{
		kind:   updateWhitelist,
		source: sid,
		data: whitelistEvent{
			addWhitelist: addWhitelist, // false to unmark
		},
	}
}

type teamEvent struct {
	team store.Team
}

func newTeamEvent(sid steamid.SID64, team store.Team) updateStateEvent {
	return updateStateEvent{
		kind:   updateTeam,
		source: sid,
		data: teamEvent{
			team: team,
		},
	}
}

type messageEvent struct {
	name      string
	createdAt time.Time
	message   string
	teamOnly  bool
	dead      bool
}

type hostnameEvent struct {
	hostname string
}

type mapEvent struct {
	mapName string
}

type mapChangeEvent struct{}

type tagsEvent struct {
	tags []string
}

type addressEvent struct {
	ip   net.IP
	port uint16
}

// incomingLogEventHandler handles mapping incoming LogEvent payloads into the more generalized
// updateStateEvent used for all state updates.
func (d *Detector) incomingLogEventHandler(ctx context.Context) {
	log := d.log.Named("LogEventHandler")
	defer log.Info("log event handler exited")

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-d.eventChan:
			var update updateStateEvent

			switch evt.Type {
			case EvtMap:
				update = updateStateEvent{kind: updateMap, data: mapEvent{mapName: evt.MetaData}}
			case EvtHostname:
				update = updateStateEvent{kind: updateHostname, data: hostnameEvent{hostname: evt.MetaData}}
			case EvtTags:
				update = updateStateEvent{kind: updateTags, data: tagsEvent{tags: strings.Split(evt.MetaData, ",")}}
			case EvtAddress:
				pcs := strings.Split(evt.MetaData, ":")

				portValue, errPort := strconv.ParseUint(pcs[1], 10, 16)
				if errPort != nil {
					log.Error("Failed to parse port: %v", zap.Error(errPort), zap.String("port", pcs[1]))

					continue
				}

				parsedIP := net.ParseIP(pcs[0])
				if parsedIP == nil {
					log.Error("Failed to parse ip", zap.String("ip", pcs[0]))

					continue
				}

				update = updateStateEvent{kind: updateAddress, data: addressEvent{ip: parsedIP, port: uint16(portValue)}}
			case EvtDisconnect:
				update = updateStateEvent{kind: changeMap, source: evt.PlayerSID, data: mapChangeEvent{}}
			case EvtKill:
				update = updateStateEvent{
					kind:   updateKill,
					source: evt.PlayerSID,
					data:   killEvent{victimName: evt.Victim, sourceName: evt.Player},
				}
			case EvtMsg:
				update = updateStateEvent{
					kind:   updateMessage,
					source: evt.PlayerSID,
					data: messageEvent{
						name:      evt.Player,
						createdAt: evt.Timestamp,
						message:   evt.Message,
						teamOnly:  evt.TeamOnly,
						dead:      evt.Dead,
					},
				}
			case EvtStatusID:
				update = newStatusUpdate(evt.PlayerSID, evt.PlayerPing, evt.UserID, evt.Player, evt.PlayerConnected)
			case EvtLobby:
				update = updateStateEvent{kind: updateLobby, source: evt.PlayerSID, data: lobbyEvent{team: evt.Team}}
			}

			d.stateUpdates <- update
		}
	}
}

func newStatusUpdate(sid steamid.SID64, ping int, userID int64, name string, connected time.Duration) updateStateEvent {
	return updateStateEvent{
		kind:   updateStatus,
		source: sid,
		data: statusEvent{
			ping:      ping,
			userID:    userID,
			name:      name,
			connected: connected,
		},
	}
}

func (d *Detector) stateUpdater(ctx context.Context) {
	log := d.log.Named("stateUpdater")

	defer log.Debug("stateUpdater exited")

	var (
		server  Server
		players = store.PlayerCollection{}
		reset   = make(chan any)
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-reset:
			d.playersMu.Lock()
			d.players = players
			d.playersMu.Unlock()
			d.serverMu.Lock()
			d.server = server
			d.serverMu.Unlock()
		case update := <-d.stateUpdates:
			log.Debug("Game state update input received", zap.Int("kind", int(update.kind)), zap.String("state", "start"))

			if update.kind == updateStatus && !update.source.Valid() {
				continue
			}

			player, inGame := players.Player(update.source)
			if update.kind != updateStatus && !inGame {
				log.Error("Tried to update player not in game", zap.String("sid64", update.source.String()))

				continue
			}

			switch update.kind {
			case updateMessage:
				evt, ok := update.data.(messageEvent)
				if !ok {
					continue
				}

				namedPlayer, srdOk := players.ByName(evt.name)
				if !srdOk {
					continue
				}

				if errUm := d.AddUserMessage(ctx, namedPlayer, evt.message, evt.dead, evt.teamOnly); errUm != nil {
					log.Error("Failed to handle user message", zap.Error(errUm))

					continue
				}
			case updateKill:
				e, ok := update.data.(killEvent)
				if !ok {
					continue
				}

				ourSid := d.settings.GetSteamID()
				src, srdOk := players.ByName(e.sourceName)
				if !srdOk {
					continue
				}

				target, targetOk := players.ByName(e.sourceName)
				if !targetOk {
					continue
				}

				src.Kills++
				target.Deaths++

				if target.SteamID == ourSid {
					src.DeathsBy++
				}

				if src.SteamID == ourSid {
					target.KillsOn++
				}
			case updateBans:
				evt, ok := update.data.(steamweb.PlayerBanState)
				if !ok {
					continue
				}

				player.NumberOfVACBans = evt.NumberOfVACBans
				player.NumberOfGameBans = evt.NumberOfGameBans
				player.CommunityBanned = evt.CommunityBanned

				if evt.DaysSinceLastBan > 0 {
					subTime := time.Now().AddDate(0, 0, -evt.DaysSinceLastBan)
					player.LastVACBanOn = &subTime
				}

				player.EconomyBan = evt.EconomyBan != "none"
			case updateKickAttempts:
				player.KickAttemptCount++
			case updateStatus:
				evt, ok := update.data.(statusEvent)
				if !ok {
					continue
				}

				player.Ping = evt.ping
				player.UserID = evt.userID
				player.Name = evt.name
				player.Connected = evt.connected.Seconds()
				player.UpdatedOn = time.Now()

			case updateLobby:
				evt, ok := update.data.(lobbyEvent)
				if !ok {
					continue
				}
				player.Team = evt.team
			case updateTags:
				evt, ok := update.data.(tagsEvent)
				if !ok {
					continue
				}

				server.Tags = evt.tags
				server.LastUpdate = time.Now()

			case updateHostname:
				evt, ok := update.data.(hostnameEvent)
				if !ok {
					continue
				}

				server.ServerName = evt.hostname
			case updateMap:
				evt, ok := update.data.(mapEvent)
				if !ok {
					continue
				}

				server.CurrentMap = evt.mapName
			case changeMap:
				for _, p := range players {
					p.Kills = 0
					p.Deaths = 0
				}
				server.CurrentMap = ""
				server.ServerName = ""

			}
			log.Debug("Game state update input", zap.Int("kind", int(update.kind)), zap.String("state", "end"))
			reset <- true
		}
	}
}
