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

type MarkFunc func(sid64 steamid.SID64, attrs []string) error

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type KickFunc func(ctx context.Context, userId int64, reason KickReason) error

type LaunchFunc func()

type QueryNamesFunc func(ctx context.Context, sid64 steamid.SID64) (UserNameHistoryCollection, error)

type QueryUserMessagesFunc func(ctx context.Context, sid64 steamid.SID64) (UserMessageCollection, error)
