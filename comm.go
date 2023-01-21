package main

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"regexp"
	"strings"
	"time"
)

type team int

const (
	red team = iota
	blu
)

type player struct {
	name          string
	steamId       steamid.SID64
	connectedAt   int
	team          team
	userId        int
	connectedTime time.Duration
}

var (
	rx *regexp.Regexp
)

func parseLobbyPlayers(body string) []player {
	var players []player
	for _, line := range strings.Split(body, "\n") {
		match := rx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		p := player{
			name:        "",
			steamId:     steamid.SID3ToSID64(steamid.SID3(match[3])),
			connectedAt: 0,
		}
		if match[4] == "TF_GC_TEAM_INVADERS" {
			p.team = blu
		} else {
			p.team = red
		}
		players = append(players, p)
	}
	return players
}

func init() {
	rx = regexp.MustCompile(`^\s+(Pending|Member)\[(\d+)]\s+(\S+)\s+team\s=\s(TF_GC_TEAM_INVADERS|TF_GC_TEAM_DEFENDERS).+?$`)
}
