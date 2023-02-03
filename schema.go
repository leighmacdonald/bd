package main

import (
	"encoding/json"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type TF2BDRules struct {
	Schema   string   `json:"$schema"`
	FileInfo FileInfo `json:"file_info"`
	Rules    []Rules  `json:"rules"`
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

type usernameTextMatch struct {
	CaseSensitive bool          `json:"case_sensitive"`
	Mode          textMatchMode `json:"mode"`
	// TODO precompile regex patterns
	Patterns []string `json:"patterns"`
}

type AvatarMatch struct {
	AvatarHash string `json:"avatar_hash"`
}

type Triggers struct {
	AvatarMatch       []AvatarMatch      `json:"avatar_match"`
	Mode              triggerMode        `json:"mode"`
	UsernameTextMatch *usernameTextMatch `json:"username_text_match"`
	ChatMsgTextMatch  *ChatMsgTextMatch  `json:"chatmsg_text_match"`
}

type Actions struct {
	TransientMark []string      `json:"transient_mark"`
	AvatarMatch   []AvatarMatch `json:"avatar_match"`
	Mark          []string      `json:"mark"`
}

type ChatMsgTextMatch struct {
	CaseSensitive bool          `json:"case_sensitive"`
	Mode          textMatchMode `json:"mode"`
	Patterns      []string      `json:"patterns"`
}

type Rules struct {
	Actions     Actions  `json:"actions,omitempty"`
	Description string   `json:"description"`
	Triggers    Triggers `json:"triggers,omitempty"`
}

func parseRulesList(data []byte, schema *TF2BDRules) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	return nil
}

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

type LastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}

type Players struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen,omitempty"`
	SteamId    any      `json:"steamid"`
	Proof      []string `json:"proof,omitempty"`
}

type TF2BDPlayerList struct {
	Schema   string    `json:"$schema"`
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
}

type playerListCollection []TF2BDPlayerList

type MatchedPlayerList struct {
	list   TF2BDPlayerList
	player Players
}

type Matcher interface {
	FindMatch(steamId steamid.SID64, match *MatchedPlayerList) bool
}

func (c playerListCollection) FindMatch(steamId steamid.SID64, match *MatchedPlayerList) bool {
	for _, list := range c {
		for _, p := range list.Players {
			if p.SteamId == steamId {
				*match = MatchedPlayerList{
					list:   list,
					player: p,
				}
				return true
			}
		}
	}
	return false
}

func parseTF2BD(data []byte, schema *TF2BDPlayerList) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	// Filter out people w/o cheater tags
	var cheatersOnly []Players
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
