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
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type mkPlayerFunc func(ctx context.Context, sid64 steamid.SteamID) (PlayerState, error)

// CreateTestPlayers will generate fake player data for testing purposes.
// nolint:gosec
func CreateTestPlayers(playerState *playerStates, fn mkPlayerFunc, count int) {
	idIdx := 0
	knownIDs := steamid.Collection{
		steamid.New("76561197998365611"), steamid.New("76561197977133523"), steamid.New("76561198065825165"),
		steamid.New("76561198004429398"), steamid.New("76561198182505218"), steamid.New("76561197989961569"),
		steamid.New("76561198183927541"), steamid.New("76561198005026984"), steamid.New("76561197997861796"),
		steamid.New("76561198377596915"), steamid.New("76561198336028289"), steamid.New("76561198066637626"),
		steamid.New("76561198818013048"), steamid.New("76561198196411029"), steamid.New("76561198079544034"),
		steamid.New("76561198008337801"), steamid.New("76561198042902038"), steamid.New("76561198013287458"),
		steamid.New("76561198038487121"), steamid.New("76561198046766708"), steamid.New("76561197963310062"),
		steamid.New("76561198017314810"), steamid.New("76561197967842214"), steamid.New("76561197984047970"),
		steamid.New("76561198020124821"), steamid.New("76561198010868782"), steamid.New("76561198022397372"),
		steamid.New("76561198016314731"), steamid.New("76561198087124802"), steamid.New("76561198024022137"),
		steamid.New("76561198015577906"), steamid.New("76561197997861796"),
	}

	randPlayer := func(userId int) PlayerState {
		team := Blu
		if userId%2 == 0 {
			team = Red
		}

		player, errPlayer := fn(context.Background(), knownIDs[idIdx])
		if errPlayer != nil {
			panic(errPlayer)
		}

		if player.Personaname == "" {
			player.Personaname = fmt.Sprintf("%d - %s", userId, player.SteamID.String())
		}

		player.Visibility = int64(steamweb.VisibilityPublic)
		player.KillsOn = int64(rand.Intn(20))
		player.RageQuits = int64(rand.Intn(10))
		player.DeathsBy = int64(rand.Intn(20))
		player.Team = team
		player.Connected = time.Duration(rand.Intn(3600) * 1000000)
		player.UserID = userId
		player.Ping = rand.Intn(150)
		player.Kills = rand.Intn(50)
		player.Deaths = rand.Intn(300)

		idIdx++

		return player
	}

	var testPlayers []PlayerState

	for i := 0; i < count; i++ {
		player := randPlayer(i)

		switch i {
		case 1:
			player.VacBans = 2
			player.Notes = "User notes \ngo here"
			last := time.Now().AddDate(-1, 0, 0).Unix()
			player.LastVacBanOn = last
		case 4:
			player.Matches = append(player.Matches, rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			player.Matches = append(player.Matches, rules.MatchResult{
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

func testLogFeeder(ctx context.Context, ingest *logIngest) {
	testLogPath, found := os.LookupEnv("TEST_CONSOLE_LOG")
	if !found {
		return
	}

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

	// Delay the incoming data a bit so its more realistic
	updateTicker := time.NewTicker(time.Millisecond * 100)

	for {
		select {
		case <-updateTicker.C:
			ingest.external <- lines[curLine]
			curLine++

			// Wrap back around once we are done
			if curLine >= lineCount {
				curLine = 0
			}
		case <-ctx.Done():
			return
		}
	}
}
