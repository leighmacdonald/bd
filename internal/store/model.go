package store

import (
	"context"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v2/steamid"
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

type BaseSID struct {
	SteamId       steamid.SID64 `json:"-"`
	SteamIdString string        `json:"steam_id"`
}

type UserMessage struct {
	BaseSID
	MessageId int64     `json:"message_id"`
	Team      Team      `json:"team"`
	UserId    int64     `json:"user_id"`
	Message   string    `json:"message"`
	Created   time.Time `json:"created"`
	Dead      bool      `json:"dead"`
	TeamOnly  bool      `json:"team_only"`
}

func NewUserMessage(sid64 steamid.SID64, message string, dead bool, teamOnly bool) (*UserMessage, error) {
	if !sid64.Valid() {
		return nil, ErrInvalidSid
	}
	if message == "" {
		return nil, ErrEmptyValue
	}
	return &UserMessage{
		BaseSID:  BaseSID{SteamId: sid64, SteamIdString: sid64.String()},
		Message:  message,
		Created:  time.Now(),
		Dead:     dead,
		TeamOnly: teamOnly,
	}, nil
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
