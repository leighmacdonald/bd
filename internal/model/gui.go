package model

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type UserInterface interface {
	Refresh()
	Start(ctx context.Context)
	Quit()
	UpdateServerState(state Server)
	UpdatePlayerState(collection PlayerCollection)
	AddUserMessage(message UserMessage)
	UpdateAttributes([]string)
	SetAvatar(sid64 steamid.SID64, avatar []byte)
}
