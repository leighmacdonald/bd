package rules

import (
	"encoding/json"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	errDuplicateSteamID = errors.New("duplicate steam id")
)

const (
	exportIndentSize = 4
)

type textMatchType string

const (
	textMatchTypeAny     textMatchType = "any"
	textMatchTypeName    textMatchType = "name"
	textMatchTypeMessage textMatchType = "message"
)

type avatarMatchType string

const (
	// 1:1 match of avatar
	avatarMatchExact avatarMatchType = "hash_full"
	// Reduced matcher
	//avatarMatchReduced avatarMatchType = "hash_reduced"
)

// AvatarMatcher provides an interface to match avatars using custom methods
type AvatarMatcher interface {
	Match(hexDigest string) *MatchResult
	Type() avatarMatchType
}

type avatarMatcher struct {
	matchType avatarMatchType
	origin    string
	hashes    []string
}

func (m avatarMatcher) Type() avatarMatchType {
	return m.matchType
}

func (m avatarMatcher) Match(hexDigest string) *MatchResult {
	for _, hash := range m.hashes {
		if hash == hexDigest {
			return &MatchResult{Origin: m.origin, MatcherType: string(m.Type())}
		}
	}
	return nil
}

func newAvatarMatcher(origin string, avatarMatchType avatarMatchType, hashes ...string) avatarMatcher {
	return avatarMatcher{
		origin:    origin,
		matchType: avatarMatchType,
		hashes:    hashes,
	}
}

// TextMatcher provides an interface to build text based matchers for names or in game messages
type TextMatcher interface {
	// Match performs a text based match
	Match(text string) *MatchResult
	Type() textMatchType
}

// SteamIDMatcher provides a basic interface to match steam ids.
type SteamIDMatcher interface {
	Match(sid64 steamid.SID64) *MatchResult
}

type steamIDMatcher struct {
	steamID steamid.SID64
	origin  string
}

func (m steamIDMatcher) Match(sid64 steamid.SID64) *MatchResult {
	if sid64 == m.steamID {
		return &MatchResult{Origin: m.origin, MatcherType: "steam_id"}
	}
	return nil
}

func newSteamIDMatcher(origin string, sid64 steamid.SID64) steamIDMatcher {
	return steamIDMatcher{steamID: sid64, origin: origin}
}

type regexTextMatcher struct {
	matcherType textMatchType
	patterns    []*regexp.Regexp
	origin      string
}

func (m regexTextMatcher) Match(value string) *MatchResult {
	for _, re := range m.patterns {
		if re.MatchString(value) {
			return &MatchResult{Origin: m.origin, MatcherType: string(m.Type())}
		}
	}
	return nil
}

func (m regexTextMatcher) Type() textMatchType {
	return m.matcherType
}

func newRegexTextMatcher(origin string, matcherType textMatchType, patterns ...string) (regexTextMatcher, error) {
	var compiled []*regexp.Regexp
	for _, inputPattern := range patterns {
		c, compErr := regexp.Compile(inputPattern)
		if compErr != nil {
			return regexTextMatcher{}, errors.Wrapf(compErr, "Invalid regex pattern: %s\n", inputPattern)
		}
		compiled = append(compiled, c)
	}
	return regexTextMatcher{
		origin:      origin,
		matcherType: matcherType,
		patterns:    compiled,
	}, nil
}

type generalTextMatcher struct {
	matcherType   textMatchType
	mode          textMatchMode
	caseSensitive bool
	patterns      []string
	origin        string
}

func (m generalTextMatcher) Match(value string) *MatchResult {
	switch m.mode {
	case textMatchModeStartsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasPrefix(value, prefix) {
					return &MatchResult{Origin: m.origin}
				}
			} else {
				if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
					return &MatchResult{Origin: m.origin}
				}
			}
		}
	case textMatchModeEndsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasSuffix(value, prefix) {
					return &MatchResult{Origin: m.origin}
				}
			} else {
				if strings.HasSuffix(strings.ToLower(value), strings.ToLower(prefix)) {
					return &MatchResult{Origin: m.origin, MatcherType: string(m.Type())}
				}
			}
		}
	case textMatchModeEqual:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if value == prefix {
					return &MatchResult{Origin: m.origin}
				}
			} else {
				if strings.EqualFold(value, prefix) {
					return &MatchResult{Origin: m.origin}
				}
			}
		}
	case textMatchModeContains:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.Contains(value, prefix) {
					return &MatchResult{Origin: m.origin, MatcherType: string(m.Type())}
				}
			} else {
				if strings.Contains(strings.ToLower(value), strings.ToLower(prefix)) {
					return &MatchResult{Origin: m.origin}
				}
			}
		}
	case textMatchModeWord:
		if !m.caseSensitive {
			value = strings.ToLower(value)
		}
		for _, iw := range strings.Split(value, " ") {
			for _, p := range m.patterns {
				if m.caseSensitive {
					if p == iw {
						return &MatchResult{Origin: m.origin}
					}
				} else {
					if strings.EqualFold(strings.ToLower(p), iw) {
						return &MatchResult{Origin: m.origin}
					}
				}
			}
		}
	}
	return nil
}

func (m generalTextMatcher) Type() textMatchType {
	return m.matcherType
}

func newGeneralTextMatcher(origin string, matcherType textMatchType, matchMode textMatchMode, caseSensitive bool, patterns ...string) TextMatcher {
	return generalTextMatcher{
		origin:        origin,
		matcherType:   matcherType,
		mode:          matchMode,
		caseSensitive: caseSensitive,
		patterns:      patterns,
	}
}

func NewEngine(localRules *RuleSchema, localPlayers *PlayerListSchema) (*Engine, error) {
	re := Engine{
		RWMutex:        &sync.RWMutex{},
		matchersSteam:  nil,
		matchersText:   nil,
		matchersAvatar: nil,
	}
	if localRules != nil {
		if errImport := re.ImportRules(localRules); errImport != nil {
			log.Printf("Failed to load local rules: %v\n", errImport)
			return nil, errImport
		}
	} else {
		ls := NewRuleSchema()
		re.rulesLists = append(re.rulesLists, &ls)
	}
	if localPlayers != nil {
		if errImport := re.ImportPlayers(localPlayers); errImport != nil {
			log.Printf("Failed to load local players: %v\n", errImport)
			return nil, errImport
		}
	} else {
		ls := NewPlayerListSchema()
		re.playerLists = append(re.playerLists, &ls)
	}
	return &re, nil
}

type MatchResult struct {
	Origin string // Title of the list that the match was generated against
	//Attributes  []string
	//Proof       []string
	MatcherType string
}

type MarkOpts struct {
	SteamID    steamid.SID64
	Attributes []string
	Proof      []string
	Name       string
}

type Engine struct {
	*sync.RWMutex
	matchersSteam  []SteamIDMatcher
	matchersText   []TextMatcher
	matchersAvatar []AvatarMatcher
	rulesLists     []*RuleSchema
	playerLists    []*PlayerListSchema
	knownTags      []string
}

func (e *Engine) Mark(opts MarkOpts) error {
	e.Lock()
	defer e.Unlock()
	for _, knownPlayer := range e.playerLists[0].Players {
		if knownPlayer.SteamID == opts.SteamID {
			return errDuplicateSteamID
		}
	}
	e.playerLists[0].Players = append(e.playerLists[0].Players, playerDefinition{
		Attributes: opts.Attributes,
		LastSeen: playerLastSeen{
			Time:       int(time.Now().Unix()),
			PlayerName: opts.Name,
		},
		SteamID: opts.SteamID,
		Proof:   opts.Proof,
	})
	return nil
}

// UniqueTags returns a list of the unique known tags across all player lists
func (e *Engine) UniqueTags() []string {
	e.RLock()
	defer e.RUnlock()
	return e.knownTags
}

func newJSONPrettyEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetIndent("", strings.Repeat(" ", exportIndentSize))
	return enc
}

// ExportPlayers writes the json encoded player list matching the listName provided to the io.Writer
func (e *Engine) ExportPlayers(listName string, w io.Writer) error {
	e.RLock()
	defer e.RUnlock()
	for _, pl := range e.playerLists {
		if listName == pl.FileInfo.Title {
			return newJSONPrettyEncoder(w).Encode(pl)
		}
	}
	return errors.Errorf("Unknown player list: %s", listName)
}

// ExportRules writes the json encoded rules list matching the listName provided to the io.Writer
func (e *Engine) ExportRules(listName string, w io.Writer) error {
	e.RLock()
	defer e.RUnlock()
	for _, pl := range e.rulesLists {
		if listName == pl.FileInfo.Title {
			return newJSONPrettyEncoder(w).Encode(pl)
		}
	}
	return errors.Errorf("Unknown rule list: %s", listName)
}

// ImportRules loads the provided ruleset for use
func (e *Engine) ImportRules(list *RuleSchema) error {
	for _, rule := range list.Rules {
		if rule.Triggers.UsernameTextMatch != nil {
			e.registerTextMatcher(newGeneralTextMatcher(
				list.FileInfo.Title,
				textMatchTypeName,
				rule.Triggers.UsernameTextMatch.Mode,
				rule.Triggers.UsernameTextMatch.CaseSensitive,
				rule.Triggers.UsernameTextMatch.Patterns...))
		}

		if rule.Triggers.ChatMsgTextMatch != nil {
			e.registerTextMatcher(newGeneralTextMatcher(
				list.FileInfo.Title,
				textMatchTypeMessage,
				rule.Triggers.ChatMsgTextMatch.Mode,
				rule.Triggers.ChatMsgTextMatch.CaseSensitive,
				rule.Triggers.ChatMsgTextMatch.Patterns...))
		}

		if len(rule.Triggers.AvatarMatch) > 0 {
			var hashes []string
			for _, h := range rule.Triggers.AvatarMatch {
				if len(h.AvatarHash) != 40 {
					continue
				}
				hashes = append(hashes, h.AvatarHash)
			}
			e.registerAvatarMatcher(newAvatarMatcher(
				list.FileInfo.Title,
				avatarMatchExact,
				hashes...))
		}
	}
	e.rulesLists = append(e.rulesLists, list)
	return nil
}

// ImportPlayers loads the provided player list for matching
func (e *Engine) ImportPlayers(list *PlayerListSchema) error {
	var playerAttrs []string
	for _, player := range list.Players {
		var steamID steamid.SID64
		// Some entries can be raw number types in addition to strings...
		switch value := player.SteamID.(type) {
		case float64:
			steamID = steamid.SID64(int64(value))
		case string:
			sid64, errSid := steamid.StringToSID64(player.SteamID.(string))
			if errSid != nil {
				log.Printf("Failed to import steamid: %v\n", errSid)
				continue
			}
			steamID = sid64
		}
		if !steamID.Valid() {
			log.Printf("tried to import invalid steamdid: %v", player.SteamID)
			continue
		}
		e.registerSteamIDMatcher(newSteamIDMatcher(list.FileInfo.Title, steamID))
		playerAttrs = append(playerAttrs, player.Attributes...)
	}
	e.Lock()
	for _, newTag := range playerAttrs {
		found := false
		for _, known := range e.knownTags {
			if strings.EqualFold(newTag, known) {
				found = true
				break
			}
		}
		if !found {
			e.knownTags = append(e.knownTags, newTag)
		}
	}
	e.playerLists = append(e.playerLists, list)
	e.Unlock()
	return nil
}

func (e *Engine) registerSteamIDMatcher(matcher SteamIDMatcher) {
	e.Lock()
	e.matchersSteam = append(e.matchersSteam, matcher)
	e.Unlock()
}

func (e *Engine) registerAvatarMatcher(matcher AvatarMatcher) {
	e.Lock()
	e.matchersAvatar = append(e.matchersAvatar, matcher)
	e.Unlock()
}

func (e *Engine) registerTextMatcher(matcher TextMatcher) {
	e.Lock()
	e.matchersText = append(e.matchersText, matcher)
	e.Unlock()
}

func (e *Engine) matchTextType(text string, matchType textMatchType) *MatchResult {
	for _, matcher := range e.matchersText {
		if matcher.Type() != textMatchTypeAny && matcher.Type() != matchType {
			continue
		}
		match := matcher.Match(text)
		if match != nil {
			return match
		}
	}
	return nil
}

func (e *Engine) MatchSteam(steamID steamid.SID64) *MatchResult {
	for _, sm := range e.matchersSteam {
		match := sm.Match(steamID)
		if match != nil {
			return match
		}
	}
	return nil
}

func (e *Engine) MatchName(name string) *MatchResult {
	return e.matchTextType(name, textMatchTypeName)
}

func (e *Engine) MatchMessage(text string) *MatchResult {
	return e.matchTextType(text, textMatchTypeMessage)
}

//func (e *Engine) matchAny(text string) *MatchResult {
//	return e.matchTextType(text, textMatchTypeAny)
//}

func (e *Engine) matchAvatar(avatar []byte) *MatchResult {
	if avatar == nil {
		return nil
	}
	hexDigest := model.HashBytes(avatar)
	for _, matcher := range e.matchersAvatar {
		match := matcher.Match(hexDigest)
		if match != nil {
			return match
		}
	}
	return nil
}
