package rules

import "github.com/leighmacdonald/steamid/v3/steamid"

type RuleTriggerMode string

// const (
// modeTrigMatchAny RuleTriggerMode = "match_any"
// modeTrigMatchAll RuleTriggerMode = "match_all"
// )

const (
	LocalRuleName   = "local"
	LocalRuleAuthor = "local"
	urlPlayerSchema = "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json"
	urlRuleSchema   = "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/rules.schema.json"
)

type TextMatchMode string

const (
	TextMatchModeContains   TextMatchMode = "contains"
	TextMatchModeRegex      TextMatchMode = "regex"
	TextMatchModeEqual      TextMatchMode = "equal"
	TextMatchModeStartsWith TextMatchMode = "starts_with"
	TextMatchModeEndsWith   TextMatchMode = "ends_with"
	TextMatchModeWord       TextMatchMode = "word" // not really needed?
)

type BaseSchema struct {
	Schema   string   `json:"$schema" yaml:"schema"` //nolint:tagliatelle
	FileInfo FileInfo `json:"file_info" yaml:"file_info"`
}

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

func NewPlayerListSchema(players ...PlayerDefinition) *PlayerListSchema {
	if players == nil {
		// Prevents json encoder outputting `null` value instead of empty array `[]`
		players = []PlayerDefinition{}
	}

	return &PlayerListSchema{
		BaseSchema: BaseSchema{
			Schema: urlPlayerSchema,
			FileInfo: FileInfo{
				Authors:     []string{LocalRuleAuthor},
				Description: "local player list",
				Title:       LocalRuleName,
				UpdateURL:   "",
			},
		},
		Players: players,
	}
}

func NewRuleSchema(rules ...RuleDefinition) *RuleSchema {
	if rules == nil {
		rules = []RuleDefinition{}
	}

	return &RuleSchema{
		BaseSchema: BaseSchema{
			Schema: urlRuleSchema,
			FileInfo: FileInfo{
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
	BaseSchema
	Rules          []RuleDefinition       `json:"rules" yaml:"rules"`
	MatchersText   []TextMatchHandler     `json:"-" yaml:"-"`
	MatchersAvatar []AvatarMatcherHandler `json:"-" yaml:"-"`
}

type RuleTriggerNameMatch struct {
	CaseSensitive bool          `json:"case_sensitive" yaml:"case_sensitive"`
	Mode          TextMatchMode `json:"mode" yaml:"mode"`
	Patterns      []string      `json:"patterns" yaml:"patterns"`
	Attributes    []string      `json:"attributes" yaml:"attributes"` // New
}

type RuleTriggerAvatarMatch struct {
	AvatarHash string `json:"avatar_hash"`
}

type RuleTriggerTextMatch struct {
	CaseSensitive bool          `json:"case_sensitive"`
	Mode          TextMatchMode `json:"mode"`
	Patterns      []string      `json:"patterns"`
	Attributes    []string      `json:"attributes" yaml:"attributes"` // New
}

type RuleTriggers struct {
	AvatarMatch       []RuleTriggerAvatarMatch `json:"avatar_match" yaml:"avatar_match"`
	Mode              RuleTriggerMode          `json:"mode" yaml:"mode"`
	UsernameTextMatch *RuleTriggerNameMatch    `json:"username_text_match" yaml:"username_text_match"` //nolint:tagliatelle
	ChatMsgTextMatch  *RuleTriggerTextMatch    `json:"chatmsg_text_match" yaml:"chat_msg_text_match"`  //nolint:tagliatelle
}

type RuleActions struct {
	TransientMark []string                 `json:"transient_mark"`
	AvatarMatch   []RuleTriggerAvatarMatch `json:"avatar_match"` // ?
	Mark          []string                 `json:"mark"`
}

type RuleDefinition struct {
	Actions     RuleActions  `json:"actions,omitempty"`
	Description string       `json:"description"`
	Triggers    RuleTriggers `json:"triggers,omitempty"`
}

type PlayerListSchema struct {
	BaseSchema
	Players       []PlayerDefinition      `json:"players"`
	matchersSteam []SteamIDMatcherHandler `yaml:"-"`
}

type PlayerLastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int64  `json:"time,omitempty"`
}

type PlayerDefinition struct {
	Attributes []string       `json:"attributes"`
	LastSeen   PlayerLastSeen `json:"last_seen,omitempty"`
	SteamID    steamid.SID64  `json:"steamid"` //nolint:tagliatelle
	Proof      []string       `json:"proof,omitempty"`
	Origin     string         `json:"origin,omitempty"`
}
