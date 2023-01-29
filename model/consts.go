package model

// Embedded via goreleaser
var (
	// BuildVersion is the current Git tag or snapshot name
	BuildVersion = "dev"
	// BuildCommit is the current git commit SHA
	BuildCommit = "none"
	// BuildDate is the build BuildDate in the RFC3339 format
	BuildDate = "unknown"
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
	EvtLobbyPlayerTeam
)
