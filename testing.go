package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type mkPlayerFunc func(ctx context.Context, sid64 steamid.SID64) (Player, error)

// CreateTestPlayers will generate fake player data for testing purposes.
// nolint:gosec
func CreateTestPlayers(playerState *playerState, fn mkPlayerFunc, count int) {
	idIdx := 0
	knownIDs := steamid.Collection{
		"76561197998365611", "76561197977133523", "76561198065825165", "76561198004429398", "76561198182505218",
		"76561197989961569", "76561198183927541", "76561198005026984", "76561197997861796", "76561198377596915",
		"76561198336028289", "76561198066637626", "76561198818013048", "76561198196411029", "76561198079544034",
		"76561198008337801", "76561198042902038", "76561198013287458", "76561198038487121", "76561198046766708",
		"76561197963310062", "76561198017314810", "76561197967842214", "76561197984047970", "76561198020124821",
		"76561198010868782", "76561198022397372", "76561198016314731", "76561198087124802", "76561198024022137",
		"76561198015577906", "76561197997861796",
	}

	randPlayer := func(userId int) Player {
		team := Blu
		if userId%2 == 0 {
			team = Red
		}

		player, errPlayer := fn(context.Background(), knownIDs[idIdx])
		if errPlayer != nil {
			panic(errPlayer)
		}

		if player.Personaname == "" {
			player.Personaname = fmt.Sprintf("%d - %d", userId, player.SteamID)
		}

		player.Visibility = int64(steamweb.VisibilityPublic)
		player.KillsOn = int64(rand.Intn(20))
		player.RageQuits = int64(rand.Intn(10))
		player.DeathsBy = int64(rand.Intn(20))
		player.Team = team
		player.Connected = float64(rand.Intn(3600))
		player.UserID = userId
		player.Ping = rand.Intn(150)
		player.Kills = rand.Intn(50)
		player.Deaths = rand.Intn(300)

		idIdx++

		return player
	}

	var testPlayers []Player

	for i := 0; i < count; i++ {
		player := randPlayer(i)

		switch i {
		case 1:
			player.VacBans = 2
			player.Notes = "User notes \ngo here"
			last := time.Now().AddDate(-1, 0, 0)
			player.LastVacBanOn.Time = last
		case 4:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			})

		case 7:
			player.Team = Spec
		}

		testPlayers = append(testPlayers, player)
	}

	playerState.replace(testPlayers)
}

func testLogFeeder(ingest *logIngest) {
	if testLogPath, isTest := os.LookupEnv("TEST_CONSOLE_LOG"); isTest {
		logPath := "testdata/console.log"
		if testLogPath != "" {
			logPath = testLogPath
		}

		body, errRead := os.ReadFile(logPath)
		if errRead != nil {
			slog.Error("Failed to load TEST_CONSOLE_LOG", slog.String("path", logPath), errAttr(errRead))

			return
		}

		lines := strings.Split(string(body), "\n")
		curLine := 0
		lineCount := len(lines)

		go func() {
			// Delay the incoming data a bit so its more realistic
			updateTicker := time.NewTicker(time.Millisecond * 10)

			for {
				<-updateTicker.C

				ingest.external <- lines[curLine]

				curLine++

				// Wrap back around once we are done
				if curLine >= lineCount {
					curLine = 0
				}
			}
		}()
	}
}
