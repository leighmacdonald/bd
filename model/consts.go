package model

import "github.com/leighmacdonald/steamid/v2/steamid"

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

type MarkFunc func(sid64 steamid.SID64, attrs []string) error

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type KickFunc func(userId int64, reason KickReason) error

type QueryNamesFunc func(sid64 steamid.SID64) ([]UserNameHistory, error)

type QueryUserMessagesFunc func(sid64 steamid.SID64) (UserMessageCollection, error)
