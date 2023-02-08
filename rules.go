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
	Match(hexDigest string) *ruleMatchResult
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

func (m avatarMatcher) Match(hexDigest string) *ruleMatchResult {
	for _, hash := range m.hashes {
		if hash == hexDigest {
			return &ruleMatchResult{origin: m.origin}
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

type TextMatcher interface {
	// Match performs a text based match
	// TODO Return a struct containing match & list source metadata
	Match(text string) *ruleMatchResult
	Type() textMatchType
}

type SteamIdMatcher interface {
	Match(sid64 steamid.SID64) *ruleMatchResult
}

type steamIdMatcher struct {
	steamId steamid.SID64
	origin  string
}

func (m steamIdMatcher) Match(sid64 steamid.SID64) *ruleMatchResult {
	if sid64 == m.steamId {
		return &ruleMatchResult{origin: m.origin}
	}
	return nil
}

func newSteamIdMatcher(origin string, sid64 steamid.SID64) steamIdMatcher {
	return steamIdMatcher{steamId: sid64, origin: origin}
}

type regexTextMatcher struct {
	matcherType textMatchType
	patterns    []*regexp.Regexp
	origin      string
}

func (m regexTextMatcher) Match(value string) *ruleMatchResult {
	for _, re := range m.patterns {
		if re.MatchString(value) {
			return &ruleMatchResult{origin: m.origin}
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

func (m generalTextMatcher) Match(value string) *ruleMatchResult {
	switch m.mode {
	case textMatchModeStartsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasPrefix(value, prefix) {
					return &ruleMatchResult{origin: m.origin}
				}
			} else {
				if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
					return &ruleMatchResult{origin: m.origin}
				}
			}
		}
		return nil
	case textMatchModeEndsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasSuffix(value, prefix) {
					return &ruleMatchResult{origin: m.origin}
				}
			} else {
				if strings.HasSuffix(strings.ToLower(value), strings.ToLower(prefix)) {
					return &ruleMatchResult{origin: m.origin}
				}
			}
		}
		return nil
	case textMatchModeEqual:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if value == prefix {
					return &ruleMatchResult{origin: m.origin}
				}
			} else {
				if strings.EqualFold(value, prefix) {
					return &ruleMatchResult{origin: m.origin}
				}
			}
		}
		return nil
	case textMatchModeContains:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.Contains(value, prefix) {
					return &ruleMatchResult{origin: m.origin}
				}
			} else {
				if strings.Contains(strings.ToLower(value), strings.ToLower(prefix)) {
					return &ruleMatchResult{origin: m.origin}
				}
			}
		}
		return nil
	case textMatchModeWord:
		if !m.caseSensitive {
			value = strings.ToLower(value)
		}
		for _, iw := range strings.Split(value, " ") {
			for _, p := range m.patterns {
				if m.caseSensitive {
					if p == iw {
						return &ruleMatchResult{origin: m.origin}
					}
				} else {
					if strings.EqualFold(strings.ToLower(p), iw) {
						return &ruleMatchResult{origin: m.origin}
					}
				}
			}
		}
		return nil
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

func newRulesEngine() RulesEngine {
	return RulesEngine{
		RWMutex:        &sync.RWMutex{},
		matchersSteam:  nil,
		matchersText:   nil,
		matchersAvatar: nil,
	}
}

type ruleMatchResult struct {
	origin string
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
			log.Printf("tried to import invalid steamdid: %v", player.SteamId)
			continue
		}
		e.registerSteamIdMatcher(newSteamIdMatcher(list.FileInfo.Title, steamId))
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

func (e *RulesEngine) matchTextType(text string, matchType textMatchType) *ruleMatchResult {
	for _, matcher := range e.matchersText {
		if matcher.Type() != textMatchTypeAny && matcher.Type() != matchType {
			continue
		}
		return matcher.Match(text)
	}
	return nil
}

func (e *RulesEngine) matchSteam(steamId steamid.SID64) *ruleMatchResult {
	for _, sm := range e.matchersSteam {
		return sm.Match(steamId)
	}
	return nil
}

func (e *RulesEngine) matchName(name string) *ruleMatchResult {
	return e.matchTextType(name, textMatchTypeName)
}

func (e *RulesEngine) matchText(text string) *ruleMatchResult {
	return e.matchTextType(text, textMatchTypeMessage)
}

func (e *RulesEngine) matchAny(text string) *ruleMatchResult {
	return e.matchTextType(text, textMatchTypeAny)
}

func hashBytes(b []byte) string {
	hash := sha1.New()
	hash.Write(b)
	return hex.EncodeToString(hash.Sum(nil))
}

func (e *RulesEngine) matchAvatar(avatar []byte) *ruleMatchResult {
	if avatar == nil {
		return nil
	}
	hexDigest := hashBytes(avatar)
	for _, matcher := range e.matchersAvatar {
		m := matcher.Match(hexDigest)
		if m != nil {
			return m
		}
	}
	return nil
}
