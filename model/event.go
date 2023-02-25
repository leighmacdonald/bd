package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"time"
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
	PlayerConnected string
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

func (e *LogEvent) ApplyTimestamp(tsString string) {
	ts, errTs := parseTimestamp(tsString)
	if errTs != nil {
		log.Printf("Failed to parse timestamp for message log: %s", errTs)
		return
	}
	e.Timestamp = ts
}

type Event struct {
	Name  EventType
	Value any
}

type UserMessage struct {
	MessageId int64
	Team      Team
	Player    string
	PlayerSID steamid.SID64
	UserId    int64
	Message   string
	Created   time.Time
}
