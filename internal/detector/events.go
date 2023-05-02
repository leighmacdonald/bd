package detector

import (
	"context"
	"github.com/leighmacdonald/bd/internal/model"
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
	updateMark
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
	team model.Team
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
func (bd *BD) incomingLogEventHandler(ctx context.Context) {
	defer bd.logger.Info("log event handler exited")
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			var update updateStateEvent
			switch evt.Type {
			case model.EvtMap:
				update = updateStateEvent{kind: updateMap, data: mapEvent{mapName: evt.MetaData}}
			case model.EvtHostname:
				update = updateStateEvent{kind: updateHostname, data: hostnameEvent{hostname: evt.MetaData}}
			case model.EvtTags:
				update = updateStateEvent{kind: updateTags, data: tagsEvent{tags: strings.Split(evt.MetaData, ",")}}
			case model.EvtAddress:
				pcs := strings.Split(evt.MetaData, ":")
				portValue, errPort := strconv.ParseUint(pcs[1], 10, 16)
				if errPort != nil {
					bd.logger.Error("Failed to parse port: %v", zap.Error(errPort), zap.String("port", pcs[1]))
					continue
				}
				ip := net.ParseIP(pcs[0])
				if ip == nil {
					bd.logger.Error("Failed to parse ip", zap.String("ip", pcs[0]))
					continue
				}
				update = updateStateEvent{kind: updateAddress, data: addressEvent{ip: ip, port: uint16(portValue)}}
			case model.EvtDisconnect:
				update = updateStateEvent{kind: changeMap, source: evt.PlayerSID, data: mapChangeEvent{}}
			case model.EvtKill:
				update = updateStateEvent{
					kind:   updateKill,
					source: evt.PlayerSID,
					data:   killEvent{victimName: evt.Victim, sourceName: evt.Player},
				}
			case model.EvtMsg:
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
			case model.EvtStatusId:
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
			case model.EvtLobby:
				update = updateStateEvent{kind: updateLobby, source: evt.PlayerSID, data: lobbyEvent{team: evt.Team}}
			}
			bd.gameStateUpdate <- update
		}
	}
}
