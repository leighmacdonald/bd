package main

import (
	"github.com/leighmacdonald/bd/model"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMatcher(t *testing.T) {
	re := RulesEngine{}
	re.registerTextMatcher(newTextMatcher(textMatchTypeName))

}

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
						Mode:          textMatchModeEqual,
						Patterns: []string{
							"test player",
						},
					},
				},
			},
		},
	}}
	p1 := model.NewPlayerState(76561197961279983, "test player")
	p1.Team = model.Red
	p1.UserId = 100
	p1.KickAttemptCount = 0

	testCases := []struct {
		ps      model.PlayerState
		matched bool
	}{
		{ps: p1, matched: true},
	}

	for _, tc := range testCases {
		var m MatchedPlayerList
		require.Equal(t, tc.matched, rulesA.FindMatch(tc.ps, &m))
	}
}
