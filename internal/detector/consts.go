package detector

import (
	"time"
)

const (
	DurationStatusUpdateTimer = time.Second * 2

	DurationCheckTimer           = time.Second * 3
	DurationUpdateTimer          = time.Second * 1
	DurationAnnounceMatchTimeout = time.Minute * 5
	DurationCacheTimeout         = time.Hour * 12
	DurationWebRequestTimeout    = time.Second * 5
	DurationRCONRequestTimeout   = time.Second * 2
	DurationProcessTimeout       = time.Second * 3
)

type EventType int

const (
	EvtKill EventType = iota
	EvtMsg
	EvtConnect
	EvtDisconnect
	EvtStatusID
	EvtHostname
	EvtMap
	EvtTags
	EvtAddress
	EvtLobby
)

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type ChatDest string

const (
	ChatDestAll   ChatDest = "all"
	ChatDestTeam  ChatDest = "team"
	ChatDestParty ChatDest = "party"
)

type Version struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}
