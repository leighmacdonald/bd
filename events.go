package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

const logTimestampFormat = "01/02/2006 - 15:04:05"

// parseTimestamp will convert the source formatted log timestamps into a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, errParse := time.Parse(logTimestampFormat, timestamp)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, errParseTimestamp)
	}

	return parsedTime, nil
}

type LogEvent struct {
	Type            EventType
	Player          string
	PlayerPing      int
	PlayerConnected time.Duration
	Team            Team
	UserID          int
	PlayerSID       steamid.SID64
	Victim          string
	VictimSID       steamid.SID64
	Message         string
	Timestamp       time.Time
	MetaData        string
	Dead            bool
	TeamOnly        bool
}

func (e *LogEvent) ApplyTimestamp(tsString string) error {
	ts, errTS := parseTimestamp(tsString)
	if errTS != nil {
		return errTS
	}

	e.Timestamp = ts

	return nil
}

type Event struct {
	Name  EventType
	Value any
}

type updateType int

const (
	updateKill updateType = iota
	updateProfile
	updateBans
	updateStatus
	updateLobby
	updateMap
	updateHostname
	updateTags
	changeMap
	updateMark
	updateWhitelist
	updateTeam
	updateKickAttempts
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
	case updateLobby:
		return "lobby"
	case updateMap:
		return "map_name"
	case updateHostname:
		return "hostname"
	case updateTags:
		return "tags"
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

type markEvent struct {
	tags    []string
	addMark bool
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
	steamID   steamid.SID64
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

type tagsEvent struct {
	tags []string
}

type eventHandler struct {
	stateHandler *gameState
	eventChan    chan LogEvent
}

// handles mapping incoming LogEvent payloads into the more generalized
// updateStateEvent used for all state updates.
func (e eventHandler) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-e.eventChan:
			switch evt.Type { //nolint:exhaustive
			case EvtMap:
				// update = updateStateEvent{kind: updateMap, data: mapEvent{mapName: evt.MetaData}}
			case EvtHostname:
				e.stateHandler.onHostname(hostnameEvent{hostname: evt.MetaData})
			case EvtTags:
				e.stateHandler.onTags(tagsEvent{tags: strings.Split(evt.MetaData, ",")})
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
				e.stateHandler.onStatus(ctx, evt.PlayerSID, statusEvent{
					ping:      evt.PlayerPing,
					userID:    evt.UserID,
					name:      evt.Player,
					connected: evt.PlayerConnected,
				})
			case EvtDisconnect:
				e.stateHandler.onMapChange()
			case EvtKill:
				e.stateHandler.onKill(killEvent{victimName: evt.Victim, sourceName: evt.Player})
			case EvtMsg:
				e.stateHandler.onUpdateMessage(ctx, messageEvent{
					steamID:   evt.PlayerSID,
					name:      evt.Player,
					createdAt: evt.Timestamp,
					message:   evt.Message,
					teamOnly:  evt.TeamOnly,
					dead:      evt.Dead,
				})
			case EvtConnect:
			case EvtLobby:
			}
		}
	}
}
