package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

type ServerState struct {
	Server     string
	CurrentMap string
	Players    []PlayerState
}

type PlayerState struct {
	Name             string
	SteamId          steamid.SID64
	ConnectedAt      time.Time
	Team             Team
	UserId           int64
	ConnectedTime    time.Duration
	KickAttemptCount int
}
