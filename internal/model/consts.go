package model

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

const (
	DurationStatusUpdateTimer    = time.Second * 2
	DurationDisconnected         = DurationStatusUpdateTimer * 3
	DurationPlayerExpired        = DurationStatusUpdateTimer * 10
	DurationCheckTimer           = time.Second * 3
	DurationUpdateTimer          = time.Second * 1
	DurationAnnounceMatchTimeout = time.Minute * 5
	DurationCacheTimeout         = time.Hour * 12
	DurationWebRequestTimeout    = time.Second * 5
	DurationRCONRequestTimeout   = time.Second
	DurationProcessTimeout       = time.Second * 3
)

type Team int

const (
	Red Team = iota
	Blu
)

type EventType int

const (
	EvtKill EventType = iota
	EvtMsg
	EvtConnect
	EvtDisconnect
	EvtStatusId
	EvtHostname
	EvtMap
	EvtTags
	EvtAddress
	EvtLobby
)

type SteamIDFunc func(sid64 steamid.SID64)

type SteamIDErrFunc func(sid64 steamid.SID64) error

type GetPlayer func(sid64 steamid.SID64) *Player

type GetPlayerOffline func(ctx context.Context, sid64 steamid.SID64, player *Player) error

type SearchOpts struct {
	Query string
}

type SavePlayer func(ctx context.Context, state *Player) error

type SearchPlayers func(ctx context.Context, opts SearchOpts) (PlayerCollection, error)

type MarkFunc func(sid64 steamid.SID64, attrs []string) error

type NoteFunc func(sid64 steamid.SID64, note string) error

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type KickFunc func(userId int64, reason KickReason) error

type ChatDest string

const (
	ChatDestAll   ChatDest = "all"
	ChatDestTeam  ChatDest = "team"
	ChatDestParty ChatDest = "party"
)

type ChatFunc func(destination ChatDest, format string, args ...any) error

type LaunchFunc func()

type QueryNamesFunc func(ctx context.Context, sid64 steamid.SID64) (UserNameHistoryCollection, error)

type QueryUserMessagesFunc func(ctx context.Context, sid64 steamid.SID64) (UserMessageCollection, error)

type Version struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}
