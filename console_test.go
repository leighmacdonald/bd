package main

import (
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestParseEvent(t *testing.T) {
	type tc struct {
		text     string
		match    bool
		expected model.LogEvent
	}
	ts := time.Date(2023, time.February, 24, 23, 37, 19, 0, time.UTC)
	cases := []tc{
		{
			text:     "02/24/2023 - 23:37:19: PopcornBucketGames :  I did tell you vix.",
			match:    true,
			expected: model.LogEvent{Type: model.EvtMsg, Player: "PopcornBucketGames", Message: "I did tell you vix.", Timestamp: ts},
		}, {
			text:     "02/24/2023 - 23:37:19: *DEAD* that's pretty thick-headed :  ty",
			match:    true,
			expected: model.LogEvent{Type: model.EvtMsg, Player: "that's pretty thick-headed", Message: "ty", Timestamp: ts, Dead: true},
		}, {
			text:     "02/24/2023 - 23:37:19: *DEAD*(TEAM) Hassium :  thats the problem vixian",
			match:    true,
			expected: model.LogEvent{Type: model.EvtMsg, Player: "Hassium", Message: "thats the problem vixian", Timestamp: ts, Dead: true, TeamOnly: true},
		}, {
			text:     "02/24/2023 - 23:37:19: ❤ Ashley ❤ killed [TrC] Nosy with spy_cicle.",
			match:    true,
			expected: model.LogEvent{Type: model.EvtKill, Player: "❤ Ashley ❤", Victim: "[TrC] Nosy", Timestamp: ts},
		}, {
			text:     "02/24/2023 - 23:37:19: ❤ Ashley ❤ killed [TrC] Nosy with spy_cicle. (crit)",
			match:    true,
			expected: model.LogEvent{Type: model.EvtKill, Player: "❤ Ashley ❤", Victim: "[TrC] Nosy", Timestamp: ts},
		}, {
			text:     "02/24/2023 - 23:37:19: Hassium connected",
			match:    true,
			expected: model.LogEvent{Type: model.EvtConnect, Player: "Hassium", Timestamp: ts},
		}, {
			text:  "02/24/2023 - 23:37:19: #    672 \"🎄AndreaJingling🎄\" [U:1:238393055] 42:57      62    0 active",
			match: true,
			expected: model.LogEvent{Type: model.EvtStatusId, Timestamp: ts, PlayerPing: 62, UserId: 672, Player: "🎄AndreaJingling🎄",
				PlayerSID: steamid.SID64(76561198198658783), PlayerConnected: time.Duration(2577000000000)},
		}, {
			text:  "02/24/2023 - 23:37:19: #    672 \"some nerd\" [U:1:238393055] 42:57:02    62    0 active",
			match: true,
			expected: model.LogEvent{Type: model.EvtStatusId, Timestamp: ts, PlayerPing: 62, UserId: 672, Player: "some nerd",
				PlayerSID: steamid.SID64(76561198198658783), PlayerConnected: time.Duration(154622000000000)},
		}, {
			text:     "02/24/2023 - 23:37:19: hostname: Uncletopia | Seattle | 1 | All Maps",
			match:    true,
			expected: model.LogEvent{Type: model.EvtHostname, Timestamp: ts, MetaData: "Uncletopia | Seattle | 1 | All Maps"},
		}, {
			text:     "02/24/2023 - 23:37:19: map     : pl_swiftwater_final1 at: 0 x, 0 y, 0 z",
			match:    true,
			expected: model.LogEvent{Type: model.EvtMap, Timestamp: ts, MetaData: "pl_swiftwater_final1"},
		}, {
			text:     "02/24/2023 - 23:37:19: tags    : nocrits,nodmgspread,payload,uncletopia",
			match:    true,
			expected: model.LogEvent{Type: model.EvtTags, Timestamp: ts, MetaData: "nocrits,nodmgspread,payload,uncletopia"},
		}, {
			text:     "02/24/2023 - 23:37:19: udp/ip  : 74.91.117.2:27015",
			match:    true,
			expected: model.LogEvent{Type: model.EvtAddress, Timestamp: ts, MetaData: "74.91.117.2:27015"}},
		{
			// 02/26/2023 - 16:45:43: Disconnect: #TF_Idle_kicked.
			// 02/26/2023 - 16:39:59: Connected to 169.254.174.254:26128
			// 02/26/2023 - 16:32:28: ุ has been idle for too long and has been kicked
			text:     "02/24/2023 - 23:37:19: Disconnect: #TF_Idle_kicked.",
			match:    true,
			expected: model.LogEvent{Type: model.EvtDisconnect, Timestamp: ts, MetaData: "#TF_Idle_kicked."}},
	}
	reader := newLogParser(nil, nil)
	for num, testCase := range cases {
		var event model.LogEvent
		err := reader.parseEvent(testCase.text, &event)
		if testCase.match {
			require.EqualValuesf(t, testCase.expected, event, "Test failed: %d", num)
		} else {
			require.ErrorIs(t, err, errNoMatch)
		}
	}
}
