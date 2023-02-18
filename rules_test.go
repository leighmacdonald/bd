package main

import (
	"bufio"
	"bytes"
	"github.com/leighmacdonald/bd/model"
	"github.com/stretchr/testify/require"
	"image"
	"image/jpeg"
	"testing"
)

func genTestRules() ruleSchema {
	return ruleSchema{
		baseSchema: baseSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json",
			FileInfo: fileInfo{
				Authors:     []string{"test author"},
				Description: "Test List",
				Title:       "Test description",
				UpdateURL:   "http://localhost",
			},
		},
		Rules: []ruleDefinition{
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "contains test",
				Triggers: ruleTriggers{
					UsernameTextMatch: &ruleTriggerNameMatch{
						CaseSensitive: false,
						Mode:          textMatchModeContains,
						Patterns:      []string{"MYG)T"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "cs equals test",
				Triggers: ruleTriggers{
					ChatMsgTextMatch: &ruleTriggerTextMatch{
						CaseSensitive: true,
						Mode:          textMatchModeEqual,
						Patterns:      []string{"CS Equal String"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "ci equals test",
				Triggers: ruleTriggers{
					ChatMsgTextMatch: &ruleTriggerTextMatch{
						CaseSensitive: false,
						Mode:          textMatchModeEqual,
						Patterns:      []string{"Ci equal String"},
					},
				},
			},
		},
	}

}

func TestTextRules(t *testing.T) {
	const testSteamId = 76561197961279983
	const customListTitle = "Custom List"
	tr := genTestRules()
	re, reErr := newRulesEngine(nil, nil)
	require.NoError(t, reErr)

	require.NoError(t, re.ImportRules(&tr))
	re.registerSteamIdMatcher(newSteamIdMatcher(customListTitle, testSteamId))
	re.registerTextMatcher(newGeneralTextMatcher(customListTitle, textMatchTypeName, textMatchModeContains, false, "test", "blah"))

	rm, eRm := newRegexTextMatcher(customListTitle, textMatchTypeName, `^test`)
	require.NoError(t, eRm)
	re.registerTextMatcher(rm)

	_, badRegex := newRegexTextMatcher(customListTitle, textMatchTypeName, `^t\s\x\t`)
	require.Error(t, badRegex)

	require.EqualValues(t, &ruleMatchResult{origin: customListTitle}, re.matchSteam(testSteamId))
	require.Nil(t, re.matchSteam(testSteamId+100))

	testCases := []struct {
		mt      textMatchType
		text    string
		matched bool
	}{
		{mt: textMatchTypeName, text: "**MYG)T**", matched: true},
		{mt: textMatchTypeName, text: "**myG)T**", matched: true},
		{mt: textMatchTypeMessage, text: "**myG)T**", matched: false},
		{mt: textMatchTypeName, text: "test", matched: true},
		{mt: textMatchTypeMessage, text: "Ci EqUaL String", matched: true},
		{mt: textMatchTypeMessage, text: "CS Equal String", matched: true},
		{mt: textMatchTypeMessage, text: "CS EqUaL StRing", matched: false},
	}

	for num, tc := range testCases {
		switch tc.mt {
		case textMatchTypeName:
			require.Equal(t, tc.matched, re.matchName(tc.text) != nil, "Test %d failed", num)
		case textMatchTypeMessage:
			require.Equal(t, tc.matched, re.matchText(tc.text) != nil, "Test %d failed", num)
		}
	}
	require.NoError(t, re.Mark(MarkOpts{}))
}

func TestAvatarRules(t *testing.T) {
	const listName = "test avatar"
	var buf bytes.Buffer
	testAvatar := image.NewRGBA(image.Rect(0, 0, 50, 50))
	require.NoError(t, jpeg.Encode(bufio.NewWriter(&buf), testAvatar, &jpeg.Options{Quality: 10}))
	re, reErr := newRulesEngine(nil, nil)
	require.NoError(t, reErr)
	re.registerAvatarMatcher(newAvatarMatcher(listName, avatarMatchExact, model.HashBytes(buf.Bytes())))
	result := re.matchAvatar(buf.Bytes())
	require.NotNil(t, result)
	require.Equal(t, listName, result.origin)
}
