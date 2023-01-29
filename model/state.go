package model

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

type ServerState struct {
	Server     string
	CurrentMap string
	Players    map[steamid.SID64]*PlayerState
}

type PlayerState struct {
	Name             string
	SteamId          steamid.SID64
	ConnectedAt      int
	Team             Team
	UserId           int
	ConnectedTime    time.Duration
	KickAttemptCount int
}
