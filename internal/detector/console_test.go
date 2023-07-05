package detector_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestParseEvent(t *testing.T) {
	type tc struct {
		text     string
		match    bool
		expected detector.LogEvent
	}

	timeStamp := time.Date(2023, time.February, 24, 23, 37, 19, 0, time.UTC)

	cases := []tc{
		{
			text:     "02/24/2023 - 23:37:19: PopcornBucketGames :  I did tell you vix.",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtMsg, Player: "PopcornBucketGames", Message: "I did tell you vix.", Timestamp: timeStamp},
		},
		{
			text:     "02/24/2023 - 23:37:19: *DEAD* that's pretty thick-headed :  ty",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtMsg, Player: "that's pretty thick-headed", Message: "ty", Timestamp: timeStamp, Dead: true},
		},
		{
			text:     "02/24/2023 - 23:37:19: *DEAD*(TEAM) Hassium :  thats the problem vixian",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtMsg, Player: "Hassium", Message: "thats the problem vixian", Timestamp: timeStamp, Dead: true, TeamOnly: true},
		},
		{
			text:     "02/24/2023 - 23:37:19: ❤ Ashley ❤ killed [TrC] Nosy with spy_cicle.",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtKill, Player: "❤ Ashley ❤", Victim: "[TrC] Nosy", Timestamp: timeStamp},
		},
		{
			text:     "02/24/2023 - 23:37:19: ❤ Ashley ❤ killed [TrC] Nosy with spy_cicle. (crit)",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtKill, Player: "❤ Ashley ❤", Victim: "[TrC] Nosy", Timestamp: timeStamp},
		},
		{
			text:     "02/24/2023 - 23:37:19: Hassium connected",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtConnect, Player: "Hassium", Timestamp: timeStamp},
		},
		{
			text:  "02/24/2023 - 23:37:19: #    672 \"🎄AndreaJingling🎄\" [U:1:238393055] 42:57      62    0 active",
			match: true,
			expected: detector.LogEvent{
				Type: detector.EvtStatusID, Timestamp: timeStamp, PlayerPing: 62, UserID: 672, Player: "🎄AndreaJingling🎄",
				PlayerSID: steamid.New(76561198198658783), PlayerConnected: time.Duration(2577000000000),
			},
		},
		{
			text:  "02/24/2023 - 23:37:19: #    672 \"some nerd\" [U:1:238393055] 42:57:02    62    0 active",
			match: true,
			expected: detector.LogEvent{
				Type: detector.EvtStatusID, Timestamp: timeStamp, PlayerPing: 62, UserID: 672, Player: "some nerd",
				PlayerSID: steamid.New(76561198198658783), PlayerConnected: time.Duration(154622000000000),
			},
		},
		{
			text:     "02/24/2023 - 23:37:19: hostname: Uncletopia | Seattle | 1 | All Maps",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtHostname, Timestamp: timeStamp, MetaData: "Uncletopia | Seattle | 1 | All Maps"},
		},
		{
			text:     "02/24/2023 - 23:37:19: map     : pl_swiftwater_final1 at: 0 x, 0 y, 0 z",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtMap, Timestamp: timeStamp, MetaData: "pl_swiftwater_final1"},
		},
		{
			text:     "02/24/2023 - 23:37:19: tags    : nocrits,nodmgspread,payload,uncletopia",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtTags, Timestamp: timeStamp, MetaData: "nocrits,nodmgspread,payload,uncletopia"},
		},
		{
			text:     "02/24/2023 - 23:37:19: udp/ip  : 74.91.117.2:27015",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtAddress, Timestamp: timeStamp, MetaData: "74.91.117.2:27015"},
		},
		{
			// 02/26/2023 - 16:45:43: Disconnect: #TF_Idle_kicked.
			// 02/26/2023 - 16:39:59: Connected to 169.254.174.254:26128
			// 02/26/2023 - 16:32:28: ุ has been idle for too long and has been kicked
			// 03/09/2023 - 01:08:03: Differing lobby received. Lobby: [A:1:1191368713:22805]/Match79636263/Lobby601530352177650 CurrentlyAssigned: [A:1:1191368713:22805]/Match79636024/Lobby601530352177650 ConnectedToMatchServer: 1 HasLobby: 1 AssignedMatchEnded: 0
			text:     "02/24/2023 - 23:37:19: Differing lobby received. Lobby: [A:1:1191368713:22805]/Match79636263/Lobby601530352177650 CurrentlyAssigned: [A:1:1191368713:22805]/Match79636024/Lobby601530352177650 ConnectedToMatchServer: 1 HasLobby: 1 AssignedMatchEnded: 0",
			match:    true,
			expected: detector.LogEvent{Type: detector.EvtDisconnect, Timestamp: timeStamp, MetaData: "Differing lobby received."},
		},
	}

	reader := detector.NewLogParser(slog.Default(), nil, nil)

	for num, testCase := range cases {
		var (
			event detector.LogEvent
			err   = reader.Parse(testCase.text, &event)
		)

		if testCase.match {
			require.EqualValuesf(t, testCase.expected, event, "Test failed: %d", num)
		} else {
			require.ErrorIs(t, err, detector.ErrNoMatch)
		}
	}
}
