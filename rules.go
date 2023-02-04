package main

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"regexp"
	"strings"
	"sync"
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

type AvatarMatcher interface {
	Match(hexDigest string) bool
	Type() avatarMatchType
}

type avatarMatcher struct {
	matchType avatarMatchType
	hashes    []string
}

func (m avatarMatcher) Type() avatarMatchType {
	return m.matchType
}

func (m avatarMatcher) Match(hexDigest string) bool {
	for _, hash := range m.hashes {
		if hash == hexDigest {
			return true
		}
	}
	return false
}

func newAvatarMatcher(avatarMatchType avatarMatchType, hashes ...string) avatarMatcher {
	return avatarMatcher{
		matchType: avatarMatchType,
		hashes:    hashes,
	}
}

type TextMatcher interface {
	// Match performs a text based match
	// TODO Return a struct containing match & list source meta data
	Match(text string) bool
	Type() textMatchType
}

type SteamIdMatcher interface {
	Match(sid64 steamid.SID64) bool
}

type steamIdMatcher struct {
	steamId steamid.SID64
}

func (m steamIdMatcher) Match(sid64 steamid.SID64) bool {
	return sid64 == m.steamId
}

func newSteamIdMatcher(sid64 steamid.SID64) steamIdMatcher {
	return steamIdMatcher{steamId: sid64}
}

type regexTextMatcher struct {
	matcherType textMatchType
	patterns    []*regexp.Regexp
}

func (m regexTextMatcher) Match(value string) bool {
	for _, re := range m.patterns {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

func (m regexTextMatcher) Type() textMatchType {
	return m.matcherType
}

func newRegexTextMatcher(matcherType textMatchType, patterns ...string) (regexTextMatcher, error) {
	var compiled []*regexp.Regexp
	for _, inputPattern := range patterns {
		c, compErr := regexp.Compile(inputPattern)
		if compErr != nil {
			return regexTextMatcher{}, errors.Wrapf(compErr, "Invalid regex pattern: %s\n", inputPattern)
		}
		compiled = append(compiled, c)
	}
	return regexTextMatcher{
		matcherType: matcherType,
		patterns:    compiled,
	}, nil
}

type generalTextMatcher struct {
	matcherType   textMatchType
	mode          textMatchMode
	caseSensitive bool
	patterns      []string
}

func (m generalTextMatcher) Match(value string) bool {
	switch m.mode {
	case textMatchModeStartsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasPrefix(value, prefix) {
					return true
				}
			} else {
				if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
					return true
				}
			}
		}
		return false
	case textMatchModeEndsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasSuffix(value, prefix) {
					return true
				}
			} else {
				if strings.HasSuffix(strings.ToLower(value), strings.ToLower(prefix)) {
					return true
				}
			}
		}
		return false
	case textMatchModeEqual:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if value == prefix {
					return true
				}
			} else {
				if strings.EqualFold(value, prefix) {
					return true
				}
			}
		}
		return false
	case textMatchModeContains:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.Contains(value, prefix) {
					return true
				}
			} else {
				if strings.Contains(strings.ToLower(value), strings.ToLower(prefix)) {
					return true
				}
			}
		}
		return false
	case textMatchModeWord:
		if !m.caseSensitive {
			value = strings.ToLower(value)
		}
		for _, iw := range strings.Split(value, " ") {
			for _, p := range m.patterns {
				if m.caseSensitive {
					if p == iw {
						return true
					}
				} else {
					if strings.EqualFold(strings.ToLower(p), iw) {
						return true
					}
				}
			}
		}
		return false
	}
	return false
}

func (m generalTextMatcher) Type() textMatchType {
	return m.matcherType
}

func newGeneralTextMatcher(matcherType textMatchType, matchMode textMatchMode, caseSensitive bool, patterns ...string) TextMatcher {
	return generalTextMatcher{
		matcherType:   matcherType,
		mode:          matchMode,
		caseSensitive: caseSensitive,
		patterns:      patterns,
	}
}

func newRulesEngine() *RulesEngine {
	return &RulesEngine{
		RWMutex:        &sync.RWMutex{},
		matchersSteam:  nil,
		matchersText:   nil,
		matchersAvatar: nil,
	}
}

type RulesEngine struct {
	*sync.RWMutex
	matchersSteam  []SteamIdMatcher
	matchersText   []TextMatcher
	matchersAvatar []AvatarMatcher
}

func (e *RulesEngine) ImportRules(list ruleSchema) error {
	for _, rule := range list.Rules {
		// rule.Actions.Mark
		if rule.Triggers.UsernameTextMatch != nil {
			e.registerTextMatcher(newGeneralTextMatcher(
				textMatchTypeName,
				rule.Triggers.UsernameTextMatch.Mode,
				rule.Triggers.UsernameTextMatch.CaseSensitive,
				rule.Triggers.UsernameTextMatch.Patterns...))
		}

		if rule.Triggers.ChatMsgTextMatch != nil {
			e.registerTextMatcher(newGeneralTextMatcher(
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
			e.registerAvatarMatcher(newAvatarMatcher(avatarMatchExact, hashes...))
		}
	}
	return nil
}

func (e *RulesEngine) ImportPlayers(list playerListSchema) error {
	for _, player := range list.Players {
		var steamId steamid.SID64
		// Some entries can be raw number types in addition to strings...
		switch v := player.SteamId.(type) {
		case float64:
			steamId = steamid.SID64(int64(v))
		case string:
			sid64, errSid := steamid.StringToSID64(player.SteamId.(string))
			if errSid != nil {
				log.Printf("Failed to import steamid: %v\n", errSid)
				continue
			}
			steamId = sid64
		}
		if !steamId.Valid() {
			continue
		}
		e.registerSteamIdMatcher(newSteamIdMatcher(steamId))
	}
	return nil
}

func (e *RulesEngine) registerSteamIdMatcher(matcher SteamIdMatcher) {
	e.Lock()
	e.matchersSteam = append(e.matchersSteam, matcher)
	e.Unlock()
}

func (e *RulesEngine) registerAvatarMatcher(matcher AvatarMatcher) {
	e.Lock()
	e.matchersAvatar = append(e.matchersAvatar, matcher)
	e.Unlock()
}

func (e *RulesEngine) registerTextMatcher(matcher TextMatcher) {
	e.Lock()
	e.matchersText = append(e.matchersText, matcher)
	e.Unlock()
}

func (e *RulesEngine) matchTextType(text string, matchType textMatchType) bool {
	for _, matcher := range e.matchersText {
		if matcher.Type() != textMatchTypeAny && matcher.Type() != matchType {
			continue
		}
		if matcher.Match(text) {
			return true
		}
	}
	return false
}

func (e *RulesEngine) matchSteam(steamId steamid.SID64) bool {
	for _, sm := range e.matchersSteam {
		if sm.Match(steamId) {
			return true
		}
	}
	return false
}

func (e *RulesEngine) matchName(name string) bool {
	return e.matchTextType(name, textMatchTypeName)
}

func (e *RulesEngine) matchText(text string) bool {
	return e.matchTextType(text, textMatchTypeMessage)
}

func (e *RulesEngine) matchAny(text string) bool {
	return e.matchTextType(text, textMatchTypeAny)
}

func hashBytes(b []byte) string {
	hash := sha1.New()
	hash.Write(b)
	return hex.EncodeToString(hash.Sum(nil))
}

func (e *RulesEngine) matchAvatar(avatar []byte) bool {
	if avatar == nil {
		return false
	}
	hexDigest := hashBytes(avatar)
	for _, matcher := range e.matchersAvatar {
		if matcher.Match(hexDigest) {
			return true
		}
	}
	return false
}
