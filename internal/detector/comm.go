package detector

import (
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"regexp"
	"strings"
)

var (
	lobbyPlayerRx *regexp.Regexp
)

type rconConnection interface {
	Exec(command string) (string, error)
	Close() error
}

func parseLobbyPlayers(body string) []*store.Player {
	var lobbyPlayers []*store.Player
	for _, line := range strings.Split(body, "\n") {
		match := lobbyPlayerRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		ps := store.NewPlayer(steamid.SID3ToSID64(steamid.SID3(match[3])), "")
		if match[4] == "TF_GC_TEAM_INVADERS" {
			ps.Team = store.Blu
		} else {
			ps.Team = store.Red
		}
		lobbyPlayers = append(lobbyPlayers, ps)
	}
	return lobbyPlayers
}

func init() {
	lobbyPlayerRx = regexp.MustCompile(`^\s+(Pending|Member)\[(\d+)]\s+(\S+)\s+team\s=\s(TF_GC_TEAM_INVADERS|TF_GC_TEAM_DEFENDERS).+?$`)
}
