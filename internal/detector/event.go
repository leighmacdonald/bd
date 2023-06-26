package detector

import (
	"net"
	"time"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

const logTimestampFormat = "01/02/2006 - 15:04:05"

// parseTimestamp will convert the source formatted log timestamps into a time.Time value
func parseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse(logTimestampFormat, timestamp)
}

type LogEvent struct {
	Type            EventType
	Player          string
	PlayerPing      int
	PlayerConnected time.Duration
	Team            store.Team
	UserId          int64
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
	ts, errTs := parseTimestamp(tsString)
	if errTs != nil {
		return errTs
	}
	e.Timestamp = ts
	return nil
}

type Event struct {
	Name  EventType
	Value any
}

type Server struct {
	ServerName string
	Addr       net.IP
	Port       uint16
	CurrentMap string
	Tags       []string
	LastUpdate time.Time
}
