package detector

import (
	"net"
	"time"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

const logTimestampFormat = "01/02/2006 - 15:04:05"

// parseTimestamp will convert the source formatted log timestamps into a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, errParse := time.Parse(logTimestampFormat, timestamp)
	if errParse != nil {
		return time.Time{}, errors.Wrap(errParse, "Failed to parse timestamp")
	}

	return parsedTime, nil
}

type LogEvent struct {
	Type            EventType
	Player          string
	PlayerPing      int
	PlayerConnected time.Duration
	Team            store.Team
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

type Server struct {
	ServerName string    `json:"server_name"`
	Addr       net.IP    `json:"-"`
	Port       uint16    `json:"-"`
	CurrentMap string    `json:"current_map"`
	Tags       []string  `json:"-"`
	LastUpdate time.Time `json:"last_update"`
}

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
