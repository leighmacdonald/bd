package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRules(t *testing.T) {
	rulesA := ruleListCollection{TF2BDRules{
		Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json",
		FileInfo: FileInfo{
			Authors:     []string{"gabe"},
			Description: "test rule set",
			Title:       "rules_a",
			UpdateURL:   "",
		},
		Rules: []Rules{
			{
				Actions: Actions{
					Mark: []string{"test_mark"},
				},
				Description: "",
				Triggers: Triggers{
					Mode: modeTrigMatchAny,
					UsernameTextMatch: &usernameTextMatch{
						CaseSensitive: false,
						Mode:          modeEqual,
						Patterns: []string{
							"test player",
						},
					},
				},
			},
		},
	}}
	p1 := playerState{
		name:             "test player",
		steamId:          76561197961279983,
		team:             red,
		userId:           100,
		connectedTime:    0,
		kickAttemptCount: 0,
	}

	testCases := []struct {
		ps      playerState
		matched bool
	}{
		{ps: p1, matched: true},
	}

	for _, tc := range testCases {
		var m MatchedPlayerList
		require.Equal(t, tc.matched, rulesA.FindMatch(tc.ps, &m))
	}
}
