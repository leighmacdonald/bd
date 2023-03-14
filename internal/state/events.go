package state

import (
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net"
	"time"
)

type updateType int

const (
	Kill updateType = iota
	Profile
	Bans
	Status
	Mark
	Message
	Lobby
	Map
	Hostname
	Tags
	Address
	Whitelist
	changeMap
)

type KillData struct {
	SourceName string
	VictimName string
}

type LobbyData struct {
	Team model.Team
}

type StatusData struct {
	PlayerSID steamid.SID64
	Ping      int
	UserID    int64
	Name      string
	Connected time.Duration
}

type UpdateEvent struct {
	Kind   updateType
	Source steamid.SID64
	Data   any
}

type MarkData struct {
	Target steamid.SID64
	Attrs  []string
	Delete bool
}

type WhitelistData struct {
	Target  steamid.SID64
	Enabled bool
}

type MessageData struct {
	Name      string
	CreatedAt time.Time
	Message   string
	TeamOnly  bool
	Dead      bool
}

type HostnameData struct {
	Hostname string
}

type MapData struct {
	Name string
}

type mapChangeEvent struct{}

type TagsData struct {
	Tags []string
}

type AddressData struct {
	Ip   net.IP
	Port uint16
}
