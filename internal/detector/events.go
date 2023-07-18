package detector

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
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

func (ut updateType) String() string {
	switch ut {
	case updateKill:
		return "kill"
	case updateProfile:
		return "profile"
	case updateBans:
		return "bans"
	case updateStatus:
		return "status"
	case updateMessage:
		return "message"
	case updateLobby:
		return "lobby"
	case updateMap:
		return "map_name"
	case updateHostname:
		return "hostname"
	case updateTags:
		return "tags"
	case updateAddress:
		return "address"
	case changeMap:
		return "change_map"
	case updateMark:
		return "mark"
	case updateWhitelist:
		return "whitelist"
	case updateTeam:
		return "team"
	case updateKickAttempts:
		return "kicks"
	case updateNotes:
		return "notes"
	case playerTimeout:
		return "timeout"
	case updateTestPlayer:
		return "test_player"
	default:
		return "unknown"
	}
}

type killEvent struct {
	sourceName string
	victimName string
}

type lobbyEvent struct {
	team store.Team
}

type statusEvent struct {
	ping      int
	userID    int
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

type noteEvent struct {
	body string
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

func newStatusUpdate(sid steamid.SID64, ping int, userID int, name string, connected time.Duration) updateStateEvent {
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

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-d.stateUpdates:
			log.Debug("Game state update input received", zap.String("kind", update.kind.String()))

			if update.kind == updateStatus && !update.source.Valid() {
				continue
			}

			switch update.kind {
			case updateMessage:
				evt, ok := update.data.(messageEvent)
				if !ok {
					continue
				}

				d.onUpdateMessage(ctx, log, evt)
			case updateKill:
				evt, ok := update.data.(killEvent)
				if !ok {
					continue
				}

				d.onKill(evt)
			case updateBans:
				evt, ok := update.data.(steamweb.PlayerBanState)
				if !ok {
					continue
				}

				d.onBans(evt)
			case updateKickAttempts:
				d.onKickAttempt(update.source)
			case updateStatus:
				evt, ok := update.data.(statusEvent)
				if !ok {
					continue
				}

				d.onStatus(ctx, update.source, evt)
			case updateTags:
				evt, ok := update.data.(tagsEvent)
				if !ok {
					continue
				}

				d.onTags(evt)
			case updateHostname:
				evt, ok := update.data.(hostnameEvent)
				if !ok {
					continue
				}

				d.onHostname(evt)
			case updateMap:
				evt, ok := update.data.(mapEvent)
				if !ok {
					continue
				}

				d.onMapName(evt)
			case changeMap:
				d.onMapChange()
			}
		}
	}
}

func (d *Detector) onUpdateMessage(ctx context.Context, log *zap.Logger, evt messageEvent) {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	namedPlayer, srdOk := d.players.ByName(evt.name)
	if !srdOk {
		return
	}

	if errUm := d.AddUserMessage(ctx, namedPlayer, evt.message, evt.dead, evt.teamOnly); errUm != nil {
		log.Error("Failed to handle user message", zap.Error(errUm))
	}
}

func (d *Detector) onKill(evt killEvent) {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	ourSid := d.Settings().SteamID

	src, srcOk := d.players.ByName(evt.sourceName)
	if !srcOk {
		return
	}

	target, targetOk := d.players.ByName(evt.sourceName)
	if !targetOk {
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
}

func (d *Detector) onBans(evt steamweb.PlayerBanState) {
	player, exists := d.GetPlayer(evt.SteamID)
	if !exists {
		return
	}

	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	player.NumberOfVACBans = evt.NumberOfVACBans
	player.NumberOfGameBans = evt.NumberOfGameBans
	player.CommunityBanned = evt.CommunityBanned
	player.EconomyBan = evt.EconomyBan

	if evt.DaysSinceLastBan > 0 {
		subTime := time.Now().AddDate(0, 0, -evt.DaysSinceLastBan)
		player.LastVACBanOn = &subTime
	}
}

func (d *Detector) onKickAttempt(steamID steamid.SID64) {
	player, exists := d.GetPlayer(steamID)
	if !exists {
		return
	}

	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	player.KickAttemptCount++
}

func (d *Detector) onStatus(ctx context.Context, steamID steamid.SID64, evt statusEvent) {
	player, errPlayer := d.GetPlayerOrCreate(ctx, steamID)
	if errPlayer != nil {
		d.log.Error("Failed to get or create player", zap.Error(errPlayer))

		return
	}

	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	player.Ping = evt.ping
	player.UserID = evt.userID
	player.Name = evt.name
	player.Connected = evt.connected.Seconds()
	player.UpdatedOn = time.Now()

	d.log.Debug("Player status updated",
		zap.String("sid", steamID.String()),
		zap.Int("tags", evt.ping),
		zap.Int("uid", evt.userID),
		zap.String("name", evt.name),
		zap.Int("connected", int(evt.connected.Seconds())))
}

func (d *Detector) onTags(evt tagsEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.Tags = evt.tags
	d.server.LastUpdate = time.Now()

	d.log.Debug("Tags updated", zap.Strings("tags", evt.tags))
}

func (d *Detector) onHostname(evt hostnameEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.ServerName = evt.hostname
	d.server.LastUpdate = time.Now()

	d.log.Debug("Hostname changed", zap.String("hostname", evt.hostname))
}

func (d *Detector) onMapName(evt mapEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.CurrentMap = evt.mapName

	d.log.Debug("Map changed", zap.String("map", evt.mapName))
}

func (d *Detector) onMapChange() {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	for _, p := range d.players {
		p.Kills = 0
		p.Deaths = 0
	}

	d.server.CurrentMap = ""
	d.server.ServerName = ""
}
