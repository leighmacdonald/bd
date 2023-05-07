package detector

import (
	"context"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"net"
	"strconv"
	"strings"
	"time"
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
	updateWhitelist
	changeMap
)

type killEvent struct {
	sourceName string
	victimName string
}

type lobbyEvent struct {
	team store.Team
}

type statusEvent struct {
	playerSID steamid.SID64
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

type updateMarkEvent struct {
	target steamid.SID64
	attrs  []string
	delete bool
}

type updateWhitelistEvent struct {
	target  steamid.SID64
	enabled bool
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
func incomingLogEventHandler(ctx context.Context) {
	defer rootLogger.Info("log event handler exited")
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
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
					rootLogger.Error("Failed to parse port: %v", zap.Error(errPort), zap.String("port", pcs[1]))
					continue
				}
				ip := net.ParseIP(pcs[0])
				if ip == nil {
					rootLogger.Error("Failed to parse ip", zap.String("ip", pcs[0]))
					continue
				}
				update = updateStateEvent{kind: updateAddress, data: addressEvent{ip: ip, port: uint16(portValue)}}
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
			case EvtStatusId:
				update = updateStateEvent{
					kind:   updateStatus,
					source: evt.PlayerSID,
					data: statusEvent{
						playerSID: evt.PlayerSID,
						ping:      evt.PlayerPing,
						userID:    evt.UserId,
						name:      evt.Player,
						connected: evt.PlayerConnected,
					},
				}
			case EvtLobby:
				update = updateStateEvent{kind: updateLobby, source: evt.PlayerSID, data: lobbyEvent{team: evt.Team}}
			}
			gameStateUpdate <- update
		}
	}
}
