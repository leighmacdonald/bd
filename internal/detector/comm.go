package detector

import (
	"github.com/leighmacdonald/bd/internal/model"
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

func parseLobbyPlayers(body string) []*model.Player {
	var players []*model.Player
	for _, line := range strings.Split(body, "\n") {
		match := lobbyPlayerRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		ps := model.NewPlayer(steamid.SID3ToSID64(steamid.SID3(match[3])), "")
		if match[4] == "TF_GC_TEAM_INVADERS" {
			ps.Team = model.Blu
		} else {
			ps.Team = model.Red
		}
		players = append(players, ps)
	}
	return players
}

func init() {
	lobbyPlayerRx = regexp.MustCompile(`^\s+(Pending|Member)\[(\d+)]\s+(\S+)\s+team\s=\s(TF_GC_TEAM_INVADERS|TF_GC_TEAM_DEFENDERS).+?$`)
}
