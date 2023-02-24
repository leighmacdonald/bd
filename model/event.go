package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

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
	//Team            Team
	Timestamp time.Time
	MetaData  string
	Dead      bool
	TeamOnly  bool
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
