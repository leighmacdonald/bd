package model

import "github.com/leighmacdonald/steamid/v2/steamid"

type UserInterface interface {
	Refresh()
	Start()
	SetBuildInfo(version string, commit string, date string, builtBy string)
	UpdateServerState(state Server)
	UpdatePlayerState(collection PlayerCollection)
	AddUserMessage(message UserMessage)
	UpdateAttributes([]string)
	SetAvatar(sid64 steamid.SID64, avatar []byte)
}
