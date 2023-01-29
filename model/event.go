package model

import "github.com/leighmacdonald/steamid/v2/steamid"

type LogEvent struct {
	Type      EventType
	Player    string
	UserId    int64
	PlayerSID steamid.SID64
	Victim    string
	VictimSID steamid.SID64
	Message   string
	Team      Team
}

type Event struct {
	Name  EventType
	Value any
}

type EvtUserMessage struct {
	Team      Team
	Player    string
	PlayerSID steamid.SID64
	UserId    int64
	Message   string
}
