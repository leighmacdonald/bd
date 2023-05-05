package store

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"strings"
	"time"
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

type UserMessage struct {
	MessageId int64
	Team      Team
	Player    string
	PlayerSID steamid.SID64
	UserId    int64
	Message   string
	Created   time.Time
	Dead      bool
	TeamOnly  bool
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
