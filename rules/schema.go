package rules

import (
	"encoding/json"
	"io"
)

type ruleTriggerMode string

//const (
// modeTrigMatchAny ruleTriggerMode = "match_any"
// modeTrigMatchAll ruleTriggerMode = "match_all"
//)

const (
	LocalRuleName   = "local"
	LocalRuleAuthor = "local"
	urlPlayerSchema = "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json"
	urlRuleSchema   = "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json"
)

type textMatchMode string

const (
	textMatchModeContains   textMatchMode = "contains"
	textMatchModeRegex      textMatchMode = "regex"
	textMatchModeEqual      textMatchMode = "equal"
	textMatchModeStartsWith textMatchMode = "starts_with"
	textMatchModeEndsWith   textMatchMode = "ends_with"
	textMatchModeWord       textMatchMode = "word" // not really needed?
)

type baseSchema struct {
	Schema   string   `json:"$schema" yaml:"schema"`
	FileInfo fileInfo `json:"file_info" yaml:"file_info"`
}

type fileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

func NewPlayerListSchema(players ...playerDefinition) PlayerListSchema {
	if players == nil {
		// Prevents json encoder outputting `null` value instead of empty array `[]`
		players = []playerDefinition{}
	}
	return PlayerListSchema{
		baseSchema: baseSchema{
			Schema: urlPlayerSchema,
			FileInfo: fileInfo{
				Authors:     []string{LocalRuleAuthor},
				Description: "local player list",
				Title:       LocalRuleName,
				UpdateURL:   "",
			},
		},
		Players: players,
	}
}
func NewRuleSchema(rules ...ruleDefinition) RuleSchema {
	if rules == nil {
		rules = []ruleDefinition{}
	}
	return RuleSchema{
		baseSchema: baseSchema{
			Schema: urlRuleSchema,
			FileInfo: fileInfo{
				Authors:     []string{LocalRuleAuthor},
				Description: "local",
				Title:       LocalRuleName,
				UpdateURL:   "",
			},
		},
		Rules: rules,
	}
}

type RuleSchema struct {
	baseSchema
	Rules []ruleDefinition `json:"rules" yaml:"rules"`
}

type ruleTriggerNameMatch struct {
	CaseSensitive bool          `json:"case_sensitive" yaml:"case_sensitive"`
	Mode          textMatchMode `json:"mode" yaml:"mode"`
	Patterns      []string      `json:"patterns" yaml:"patterns"`
}

type ruleTriggerAvatarMatch struct {
	AvatarHash string `json:"avatar_hash"`
}

type ruleTriggerTextMatch struct {
	CaseSensitive bool          `json:"case_sensitive"`
	Mode          textMatchMode `json:"mode"`
	Patterns      []string      `json:"patterns"`
}

type ruleTriggers struct {
	AvatarMatch       []ruleTriggerAvatarMatch `json:"avatar_match" yaml:"avatar_match"`
	Mode              ruleTriggerMode          `json:"mode" yaml:"mode"`
	UsernameTextMatch *ruleTriggerNameMatch    `json:"username_text_match" yaml:"username_text_match"`
	ChatMsgTextMatch  *ruleTriggerTextMatch    `json:"chatmsg_text_match" yaml:"chat_msg_text_match"`
}

type ruleActions struct {
	TransientMark []string                 `json:"transient_mark"`
	AvatarMatch   []ruleTriggerAvatarMatch `json:"avatar_match"` // ?
	Mark          []string                 `json:"mark"`
}

type ruleDefinition struct {
	Actions     ruleActions  `json:"actions,omitempty"`
	Description string       `json:"description"`
	Triggers    ruleTriggers `json:"triggers,omitempty"`
}

type PlayerListSchema struct {
	baseSchema
	Players []playerDefinition `json:"players"`
}

type playerLastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}

type playerDefinition struct {
	Attributes []string       `json:"Attributes"`
	LastSeen   playerLastSeen `json:"last_seen,omitempty"`
	SteamID    interface{}    `json:"steamid"`
	Proof      []string       `json:"proof,omitempty"`
	Origin     string         `json:"origin,omitempty"` // TODO add to schema?
}

func ParsePlayerSchema(reader io.Reader, schema *PlayerListSchema) error {
	if errUnmarshal := json.NewDecoder(reader).Decode(schema); errUnmarshal != nil {
		return errUnmarshal
	}
	// Filter out people w/o cheater tags
	var cheatersOnly []playerDefinition
	for _, p := range schema.Players {
		isCheater := false
		for _, attr := range p.Attributes {
			if attr == "cheater" {
				isCheater = true
				break
			}
		}
		if !isCheater {
			continue
		}
		cheatersOnly = append(cheatersOnly, p)
	}
	schema.Players = cheatersOnly
	return nil
}

func ParseRulesList(reader io.Reader, schema *RuleSchema) error {
	if errUnmarshal := json.NewDecoder(reader).Decode(schema); errUnmarshal != nil {
		return errUnmarshal
	}
	return nil
}
