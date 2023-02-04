package main

import (
	"encoding/json"
)

type ruleSchema struct {
	Schema   string         `json:"$schema" yaml:"schema"`
	FileInfo schemaFileInfo `json:"file_info" yaml:"file_info"`
	Rules    []schemaRules  `json:"rules" yaml:"rules"`
}

type triggerMode string

const (
	modeTrigMatchAny triggerMode = "match_any"
	modeTrigMatchAll triggerMode = "match_all"
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

type triggerNameMatch struct {
	CaseSensitive bool          `json:"case_sensitive" yaml:"case_sensitive"`
	Mode          textMatchMode `json:"mode" yaml:"mode"`
	// TODO precompile regex patterns
	Patterns []string `json:"patterns" yaml:"patterns"`
}

type TF2BDAvatarMatch struct {
	AvatarHash string `json:"avatar_hash"`
}

type TF2BDTriggers struct {
	AvatarMatch       []TF2BDAvatarMatch `json:"avatar_match" yaml:"avatar_match"`
	Mode              triggerMode        `json:"mode" yaml:"mode"`
	UsernameTextMatch *triggerNameMatch  `json:"username_text_match" yaml:"username_text_match"`
	ChatMsgTextMatch  *TriggerTextMatch  `json:"chatmsg_text_match" yaml:"chat_msg_text_match"`
}

type TF2BDActions struct {
	TransientMark []string           `json:"transient_mark"`
	AvatarMatch   []TF2BDAvatarMatch `json:"avatar_match"` // ?
	Mark          []string           `json:"mark"`
}

type TriggerTextMatch struct {
	CaseSensitive bool          `json:"case_sensitive"`
	Mode          textMatchMode `json:"mode"`
	Patterns      []string      `json:"patterns"`
}

type schemaRules struct {
	Actions     TF2BDActions  `json:"actions,omitempty"`
	Description string        `json:"description"`
	Triggers    TF2BDTriggers `json:"triggers,omitempty"`
}

func parseRulesList(data []byte, schema *ruleSchema) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	return nil
}

type schemaFileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

type schemaLastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}

type schemaPlayer struct {
	Attributes []string       `json:"attributes"`
	LastSeen   schemaLastSeen `json:"last_seen,omitempty"`
	SteamId    string         `json:"steamid"`
	Proof      []string       `json:"proof,omitempty"`
}

type schemaPlayerList struct {
	Schema   string         `json:"$schema"`
	FileInfo schemaFileInfo `json:"file_info"`
	Players  []schemaPlayer `json:"players"`
}

func parsePlayerSchema(data []byte, schema *schemaPlayerList) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	// Filter out people w/o cheater tags
	var cheatersOnly []schemaPlayer
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

func parseTF2BDRules(data []byte, schema *ruleSchema) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	// TODO Filter out / adjust anything?
	return nil
}
