package main

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type steamIdMatchType string

const (
	steamIdMatchSID64 steamIdMatchType = "steam64"
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
	avatarMatchReduced avatarMatchType = "hash_reduced"
)

type AvatarMatcher interface {
	Match(avatar []byte) bool
	Type() avatarMatchType
}

type TextMatcher interface {
	Match(text string) bool
	Type() textMatchType
}

type SteamIdMatcher interface {
	Match(sid64 steamid.SID64) bool
	Type() steamIdMatchType
}
type generalTextMatcher struct {
	matcherType textMatchType
}

func (m generalTextMatcher) Match(text string) bool {
	return false
}
func (m generalTextMatcher) Type() textMatchType {
	return m.matcherType
}

func newTextMatcher(matcherType textMatchType) TextMatcher {
	return generalTextMatcher{
		matcherType: matcherType,
	}
}

type RulesEngine struct {
	matchersSteam  []SteamIdMatcher
	matchersText   []TextMatcher
	matchersAvatar []AvatarMatcher
}

func (e *RulesEngine) registerSteamIdMatcher(matcher SteamIdMatcher) {
	e.matchersSteam = append(e.matchersSteam, matcher)
}

func (e *RulesEngine) registerAvatarMatcher(matcher AvatarMatcher) {
	e.matchersAvatar = append(e.matchersAvatar, matcher)
}

func (e *RulesEngine) registerTextMatcher(matcher TextMatcher) {
	e.matchersText = append(e.matchersText, matcher)
}

func (e *RulesEngine) matchTextType(text string, matchType textMatchType) bool {
	for _, matcher := range e.matchersText {
		if !(matcher.Type() != textMatchTypeAny || matcher.Type() != matchType) || !matcher.Match(text) {
			continue
		}
		// TODO do something with match
		return true
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
	sha := hashBytes(avatar)
	return sha == "acd123"
}
