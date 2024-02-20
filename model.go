package main

import (
	"context"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

type QueryNamesFunc func(ctx context.Context, sid64 steamid.SID64) (UserNameHistoryCollection, error)

type QueryUserMessagesFunc func(ctx context.Context, sid64 steamid.SID64) (UserMessageCollection, error)

type Team int

const (
	Spec Team = iota
	Unassigned
	Red
	Blu
)

func (t Team) String() string {
	switch t {
	case Spec:
		return "spectator"
	case Unassigned:
		return "unassigned"
	case Red:
		return "red"
	case Blu:
		return "blu"
	default:
		return "unknown"
	}
}

type BaseSID struct {
	SteamID steamid.SID64 `json:"steam_id"`
}

type UserMessage struct {
	BaseSID
	MessageID int64     `json:"message_id"`
	Team      Team      `json:"team"`
	UserID    int64     `json:"user_id"`
	Message   string    `json:"message"`
	Created   time.Time `json:"created"`
	Dead      bool      `json:"dead"`
	TeamOnly  bool      `json:"team_only"`
}

func (um UserMessage) Formatted() string {
	var msg []string
	if um.TeamOnly {
		msg = append(msg, "(TEAM)")
	}

	if um.Dead {
		msg = append(msg, "(DEAD)")
	}

	msg = append(msg, um.Message)

	return strings.Join(msg, " ")
}

type UserMessageCollection []UserMessage

func (messages UserMessageCollection) AsAny() []any {
	bl := make([]any, len(messages))
	for i, r := range messages {
		bl[i] = r
	}

	return bl
}
