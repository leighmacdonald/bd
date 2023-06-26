package detector

import (
	"context"
	"time"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

const (
	DurationStatusUpdateTimer = time.Second * 2

	DurationCheckTimer           = time.Second * 3
	DurationUpdateTimer          = time.Second * 1
	DurationAnnounceMatchTimeout = time.Minute * 5
	DurationCacheTimeout         = time.Hour * 12
	DurationWebRequestTimeout    = time.Second * 5
	DurationRCONRequestTimeout   = time.Second
	DurationProcessTimeout       = time.Second * 3
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

type SteamIDFn func(sid64 steamid.SID64)

type SteamIDErrFn func(sid64 steamid.SID64) error

type GetPlayerFm func(sid64 steamid.SID64) *store.Player

type GetPlayerOffline func(ctx context.Context, sid64 steamid.SID64, player *store.Player) error

type SavePlayer func(ctx context.Context, state *store.Player) error

type SearchPlayers func(ctx context.Context, opts store.SearchOpts) (store.PlayerCollection, error)

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

type Version struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}
