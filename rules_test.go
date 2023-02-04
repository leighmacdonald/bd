package main

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/require"
	"image"
	"image/jpeg"
	"io"
	"strings"
	"testing"
)

func genTestRules() io.Reader {
	return strings.NewReader(`{
  "$schema": "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json",
  "file_info": {
    "authors": [
      "pazer"
    ],
    "description": "Official rules list for TF2 Bot Detector.",
    "title": "Official rules list",
    "update_url": "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/rules.official.json"
  },
  "rules": [
    {
      "actions": {
        "mark": [
          "cheater"
        ]
      },
      "description": "(catbot) mygot",
      "triggers": {
        "username_text_match": {
          "case_sensitive": false,
          "mode": "contains",
          "patterns": [
            "MYG)T"
          ]
        }
      }
    },
    {
      "actions": {
        "mark": [
          "cheater"
        ]
      },
      "description": "misc discord advertisement bot",
      "triggers": {
        "username_text_match": {
          "case_sensitive": false,
          "mode": "contains",
          "patterns": [
            "discord.gg/ngXUzkRh7C"
          ]
        }
      }
    }
  ]
}`)
}

func TestTextRules(t *testing.T) {
	const testSteamId = 76561197961279983
	re := newRulesEngine()
	require.NoError(t, re.ImportRules(genTestRules()))
	re.registerSteamIdMatcher(newSteamIdMatcher(testSteamId))
	re.registerTextMatcher(newGeneralTextMatcher(textMatchTypeName, textMatchModeContains, false, "test", "blah"))
	rm, eRm := newRegexTextMatcher(textMatchTypeName, `^test`)
	require.NoError(t, eRm)
	re.registerTextMatcher(rm)

	_, badRegex := newRegexTextMatcher(textMatchTypeName, `^t\s\x\t`)
	require.Error(t, badRegex)

	require.True(t, re.matchSteam(testSteamId))

	testCases := []struct {
		mt      textMatchType
		text    string
		matched bool
	}{
		{mt: textMatchTypeName, text: "**MYG)T**", matched: true},
		{mt: textMatchTypeName, text: "**myG)T**", matched: true},
		{mt: textMatchTypeMessage, text: "**myG)T**", matched: false},
		{mt: textMatchTypeName, text: "test", matched: true},
	}

	for _, tc := range testCases {
		switch tc.mt {
		case textMatchTypeName:
			require.Equal(t, tc.matched, re.matchName(tc.text))
		case textMatchTypeMessage:
			require.Equal(t, tc.matched, re.matchText(tc.text))
		}
	}

}

func TestAvatarRules(t *testing.T) {
	re := newRulesEngine()
	var buf bytes.Buffer
	imgBuf := bufio.NewWriter(&buf)
	testAvatar := image.NewRGBA(image.Rect(0, 0, 50, 50))
	require.NoError(t, jpeg.Encode(imgBuf, testAvatar, &jpeg.Options{Quality: 10}))
	re.registerAvatarMatcher(newAvatarMatcher(avatarMatchExact, hashBytes(buf.Bytes())))
	require.True(t, re.matchAvatar(buf.Bytes()))
}
