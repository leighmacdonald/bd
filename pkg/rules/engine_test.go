package rules

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/require"
	"image"
	"image/jpeg"
	"testing"
)

func genTestRules() RuleSchema {
	return RuleSchema{
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
				Description: "contains test ci",
				Triggers: ruleTriggers{
					UsernameTextMatch: &ruleTriggerNameMatch{
						CaseSensitive: false,
						Mode:          textMatchModeContains,
						Patterns:      []string{"test_contains_value_ci"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "contains test cs",
				Triggers: ruleTriggers{
					UsernameTextMatch: &ruleTriggerNameMatch{
						CaseSensitive: true,
						Mode:          textMatchModeContains,
						Patterns:      []string{"test_contains_value_CS"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "regex name match test",
				Triggers: ruleTriggers{
					UsernameTextMatch: &ruleTriggerNameMatch{
						Mode:     textMatchModeRegex,
						Patterns: []string{"name_regex_test$"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "equality test cs",
				Triggers: ruleTriggers{
					ChatMsgTextMatch: &ruleTriggerTextMatch{
						CaseSensitive: true,
						Mode:          textMatchModeEqual,
						Patterns:      []string{"test_equal_value_CS"},
					},
				},
			},
			{
				Actions: ruleActions{
					Mark: []string{"cheater"},
				},
				Description: "equality test ci",
				Triggers: ruleTriggers{
					ChatMsgTextMatch: &ruleTriggerTextMatch{
						CaseSensitive: false,
						Mode:          textMatchModeEqual,
						Patterns:      []string{"test_equal_value_CI"},
					},
				},
			},
		},
	}

}

const customListTitle = "Custom List"

func TestSteamRules(t *testing.T) {
	const testSteamID = 76561197961279983
	registerSteamIDMatcher(newSteamIDMatcher(customListTitle, testSteamID, []string{"test_attr"}))
	steamMatch := MatchSteam(testSteamID)
	require.NotNil(t, steamMatch, "Failed to match steamid")
	require.Equal(t, customListTitle, steamMatch.Origin)
	require.Nil(t, MatchSteam(testSteamID+1), "Matched invalid steamid")
}

func TestTextRules(t *testing.T) {
	tr := genTestRules()
	_, errImport := ImportRules(&tr)
	require.NoError(t, errImport)
	testAttrs := []string{"test_attr"}
	registerTextMatcher(newGeneralTextMatcher(customListTitle, textMatchTypeName, textMatchModeContains, false, testAttrs, "test", "blah"))

	rm, eRm := newRegexTextMatcher(customListTitle, textMatchTypeMessage, testAttrs, `^test.+?`)
	require.NoError(t, eRm)
	registerTextMatcher(rm)

	_, badRegex := newRegexTextMatcher(customListTitle, textMatchTypeName, testAttrs, `^t\s\x\t`)
	require.Error(t, badRegex)

	testCases := []struct {
		mt      textMatchType
		text    string
		matched bool
	}{
		{mt: textMatchTypeName, text: "** test_Contains_value_cI **", matched: true},
		{mt: textMatchTypeName, text: "** test_contains_value_CS **", matched: true},
		{mt: textMatchTypeName, text: "blah_name_regex_test", matched: true},
		{mt: textMatchTypeName, text: "Uncle Dane", matched: false},
		{mt: textMatchTypeMessage, text: "test_equal_value_CS", matched: true},
		{mt: textMatchTypeMessage, text: "test_Equal_value_cI", matched: true},
		{mt: textMatchTypeMessage, text: "test_regex", matched: true},
		{mt: textMatchTypeMessage, text: "A sample ok message", matched: false},
	}

	for num, tc := range testCases {
		switch tc.mt {
		case textMatchTypeName:
			require.Equal(t, tc.matched, MatchName(tc.text) != nil, "Test %d failed", num)
		case textMatchTypeMessage:
			require.Equal(t, tc.matched, MatchMessage(tc.text) != nil, "Test %d failed", num)
		}
	}
	require.Error(t, Mark(MarkOpts{}))
}

func TestAvatarRules(t *testing.T) {
	const listName = "test avatar"
	var buf bytes.Buffer
	testAvatar := image.NewRGBA(image.Rect(0, 0, 50, 50))
	require.NoError(t, jpeg.Encode(bufio.NewWriter(&buf), testAvatar, &jpeg.Options{Quality: 10}))

	registerAvatarMatcher(newAvatarMatcher(listName, avatarMatchExact, HashBytes(buf.Bytes())))
	result := matchAvatar(buf.Bytes())
	require.NotNil(t, result)
	require.Equal(t, listName, result.Origin)
}
