package rules_test

import (
	"bufio"
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func genTestRules() rules.RuleSchema {
	return rules.RuleSchema{
		BaseSchema: rules.BaseSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json",
			FileInfo: rules.FileInfo{
				Authors:     []string{"test author"},
				Description: "Test List",
				Title:       "Test description",
				UpdateURL:   "http://localhost",
			},
		},
		Rules: []rules.RuleDefinition{
			{
				Actions: rules.RuleActions{
					Mark: []string{"cheater"},
				},
				Description: "contains test ci",
				Triggers: rules.RuleTriggers{
					UsernameTextMatch: &rules.RuleTriggerNameMatch{
						CaseSensitive: false,
						Mode:          rules.TextMatchModeContains,
						Patterns:      []string{"test_contains_value_ci"},
					},
				},
			},
			{
				Actions: rules.RuleActions{
					Mark: []string{"cheater"},
				},
				Description: "contains test cs",
				Triggers: rules.RuleTriggers{
					UsernameTextMatch: &rules.RuleTriggerNameMatch{
						CaseSensitive: true,
						Mode:          rules.TextMatchModeContains,
						Patterns:      []string{"test_contains_value_CS"},
					},
				},
			},
			{
				Actions: rules.RuleActions{
					Mark: []string{"cheater"},
				},
				Description: "regex name match test",
				Triggers: rules.RuleTriggers{
					UsernameTextMatch: &rules.RuleTriggerNameMatch{
						Mode:     rules.TextMatchModeRegex,
						Patterns: []string{"name_regex_test$"},
					},
				},
			},
			{
				Actions: rules.RuleActions{
					Mark: []string{"cheater"},
				},
				Description: "equality test cs",
				Triggers: rules.RuleTriggers{
					ChatMsgTextMatch: &rules.RuleTriggerTextMatch{
						CaseSensitive: true,
						Mode:          rules.TextMatchModeEqual,
						Patterns:      []string{"test_equal_value_CS"},
					},
				},
			},
			{
				Actions: rules.RuleActions{
					Mark: []string{"cheater"},
				},
				Description: "equality test ci",
				Triggers: rules.RuleTriggers{
					ChatMsgTextMatch: &rules.RuleTriggerTextMatch{
						CaseSensitive: false,
						Mode:          rules.TextMatchModeEqual,
						Patterns:      []string{"test_equal_value_CI"},
					},
				},
			},
		},
	}
}

const customListTitle = "Custom List"

func TestSteamRules(t *testing.T) {
	engine := rules.New()
	testSteamID := steamid.New(76561197961279983)
	list := engine.UserPlayerList()
	list.RegisterSteamIDMatcher(rules.NewSteamIDMatcher(customListTitle, testSteamID, []string{"test_attr"}))
	steamMatch := engine.MatchSteam(testSteamID)
	require.NotNil(t, steamMatch, "Failed to match steamid")
	require.Equal(t, customListTitle, steamMatch[0].Origin)
	require.Nil(t, engine.MatchSteam(steamid.New(testSteamID.Int64()+1)), "Matched invalid steamid")
}

func TestTextRules(t *testing.T) {
	engine := rules.New()
	tr := genTestRules()
	_, errImport := engine.ImportRules(&tr)
	require.NoError(t, errImport)

	testAttrs := []string{"test_attr"}

	list := engine.UserRuleList()
	list.RegisterTextMatcher(rules.NewGeneralTextMatcher(customListTitle, rules.TextMatchTypeName, rules.TextMatchModeContains, false, testAttrs, "test", "blah"))

	rm, eRm := rules.NewRegexTextMatcher(customListTitle, rules.TextMatchTypeMessage, testAttrs, `^test.+?`)
	require.NoError(t, eRm)
	list.RegisterTextMatcher(rm)

	_, badRegex := rules.NewRegexTextMatcher(customListTitle, rules.TextMatchTypeName, testAttrs, `^t\s\x\t`)
	require.Error(t, badRegex)

	testCases := []struct {
		mt      rules.TextMatchType
		text    string
		matched bool
	}{
		{mt: rules.TextMatchTypeName, text: "** test_Contains_value_cI **", matched: true},
		{mt: rules.TextMatchTypeName, text: "** test_contains_value_CS **", matched: true},
		{mt: rules.TextMatchTypeName, text: "blah_name_regex_test", matched: true},
		{mt: rules.TextMatchTypeName, text: "Uncle Dane", matched: false},
		{mt: rules.TextMatchTypeMessage, text: "test_equal_value_CS", matched: true},
		{mt: rules.TextMatchTypeMessage, text: "test_Equal_value_cI", matched: true},
		{mt: rules.TextMatchTypeMessage, text: "test_regex", matched: true},
		{mt: rules.TextMatchTypeMessage, text: "A sample ok message", matched: false},
	}

	for num, testCase := range testCases {
		switch testCase.mt {
		case rules.TextMatchTypeAny:
			// Not Implemented
			continue
		case rules.TextMatchTypeName:
			require.Equal(t, testCase.matched, engine.MatchName(testCase.text) != nil, "Test %d failed", num)
		case rules.TextMatchTypeMessage:
			require.Equal(t, testCase.matched, engine.MatchMessage(testCase.text) != nil, "Test %d failed", num)
		}
	}

	require.Error(t, engine.Mark(rules.MarkOpts{}))
}

func TestAvatarRules(t *testing.T) {
	const listName = "test avatar"

	var (
		engine     = rules.New()
		buf        bytes.Buffer
		testAvatar = image.NewRGBA(image.Rect(0, 0, 50, 50))
	)

	require.NoError(t, jpeg.Encode(bufio.NewWriter(&buf), testAvatar, &jpeg.Options{Quality: 10}))

	list := engine.UserRuleList()
	list.RegisterAvatarMatcher(rules.NewAvatarMatcher(listName, rules.AvatarMatchExact, rules.HashBytes(buf.Bytes())))

	result := engine.MatchAvatar(buf.Bytes())
	require.NotNil(t, result)
	require.Equal(t, listName, result[0].Origin)
}
