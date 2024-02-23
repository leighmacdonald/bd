package rules

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

type MatchResult struct {
	Origin     string   `json:"origin"` // Title of the list that the match was generated against
	Attributes []string `json:"attributes"`
	// Proof       []string
	MatcherType string `json:"matcher_type"`
}

func (mr MatchResult) HasAttr(attr string) bool {
	for _, resultAttr := range mr.Attributes {
		if strings.EqualFold(resultAttr, attr) {
			return true
		}
	}

	return false
}

type MatchResults []MatchResult

type TextMatchType string

const (
	TextMatchTypeAny     TextMatchType = "any"
	TextMatchTypeName    TextMatchType = "name"
	TextMatchTypeMessage TextMatchType = "message"
)

type AvatarMatchType string

const (
	// 1:1 match of avatar
	AvatarMatchExact AvatarMatchType = "hash_full"
	// Reduced matcher
	// avatarMatchReduced AvatarMatchType = "hash_reduced".
)

// AvatarMatcherI provides an interface to match avatars using custom methods.
type AvatarMatcherI interface {
	Match(hexDigest string) (MatchResult, bool)
	Type() AvatarMatchType
}

type AvatarMatcher struct {
	matchType  AvatarMatchType
	origin     string
	hashes     []string
	attributes []string
}

func (m AvatarMatcher) Type() AvatarMatchType {
	return m.matchType
}

func (m AvatarMatcher) Match(hexDigest string) (MatchResult, bool) {
	for _, hash := range m.hashes {
		if hash == hexDigest {
			return MatchResult{Origin: m.origin, MatcherType: string(m.Type()), Attributes: m.attributes}, true
		}
	}

	return MatchResult{}, false
}

func NewAvatarMatcher(origin string, avatarMatchType AvatarMatchType, hashes ...string) AvatarMatcher {
	return AvatarMatcher{
		origin:    origin,
		matchType: avatarMatchType,
		hashes:    hashes,
	}
}

// TextMatcher provides an interface to build text based matchers for names or in game messages.
type TextMatcher interface {
	// Match performs a text based match
	Match(text string) (MatchResult, bool)
	Type() TextMatchType
}

// SteamIDMatcherI provides a basic interface to match steam ids.
type SteamIDMatcherI interface {
	Match(sid64 steamid.SID64) (MatchResult, bool)
	HasOneOfAttr(attrs ...string) bool
	LastSeen() time.Time
	SteamID() steamid.SID64
}

type SteamIDMatcher struct {
	steamID    steamid.SID64
	origin     string
	attributes []string
	lastSeen   PlayerLastSeen
}

func (m SteamIDMatcher) SteamID() steamid.SID64 {
	return m.SteamID()
}

func (m SteamIDMatcher) LastSeen() time.Time {
	return time.Unix(m.lastSeen.Time, 0)
}

func (m SteamIDMatcher) HasOneOfAttr(attrs ...string) bool {
	for _, attr := range attrs {
		if slices.ContainsFunc(m.attributes, func(s string) bool {
			return strings.EqualFold(attr, s)
		}) {
			return true
		}
	}

	return false
}

func (m SteamIDMatcher) Match(sid64 steamid.SID64) (MatchResult, bool) {
	if sid64 == m.steamID {
		return MatchResult{Origin: m.origin, MatcherType: "steam_id", Attributes: m.attributes}, true
	}

	return MatchResult{}, false
}

func NewSteamIDMatcher(origin string, sid64 steamid.SID64, attributes []string) SteamIDMatcher {
	return SteamIDMatcher{steamID: sid64, origin: origin, attributes: attributes}
}

type RegexTextMatcher struct {
	matcherType TextMatchType
	patterns    []*regexp.Regexp
	origin      string
	attributes  []string
}

func (m RegexTextMatcher) Match(value string) *MatchResult {
	for _, re := range m.patterns {
		if re.MatchString(value) {
			return &MatchResult{Origin: m.origin, MatcherType: string(m.Type())}
		}
	}

	return nil
}

func (m RegexTextMatcher) Type() TextMatchType {
	return m.matcherType
}

func NewRegexTextMatcher(origin string, matcherType TextMatchType, attributes []string, patterns ...string) (RegexTextMatcher, error) {
	compiled := make([]*regexp.Regexp, len(patterns))

	for index, inputPattern := range patterns {
		compiledRx, compErr := regexp.Compile(inputPattern)
		if compErr != nil {
			return RegexTextMatcher{}, errors.Join(compErr, fmt.Errorf("%w: %s", ErrInvalidRegex, inputPattern))
		}

		compiled[index] = compiledRx
	}

	return RegexTextMatcher{
		origin:      origin,
		matcherType: matcherType,
		patterns:    compiled,
		attributes:  attributes,
	}, nil
}

type GeneralTextMatcher struct {
	matcherType   TextMatchType
	mode          TextMatchMode
	caseSensitive bool
	patterns      []string
	attributes    []string
	origin        string
}

func (m GeneralTextMatcher) Match(value string) (MatchResult, bool) { //nolint:gocognit,cyclop
	switch m.mode {
	case TextMatchModeRegex:
		// Not implemented
	case TextMatchModeStartsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasPrefix(value, prefix) {
					return MatchResult{Origin: m.origin, Attributes: m.attributes}, true
				}
			} else {
				if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
					return MatchResult{Origin: m.origin}, true
				}
			}
		}
	case TextMatchModeEndsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasSuffix(value, prefix) {
					return MatchResult{Origin: m.origin}, true
				}
			} else {
				if strings.HasSuffix(strings.ToLower(value), strings.ToLower(prefix)) {
					return MatchResult{Origin: m.origin, MatcherType: string(m.Type())}, true
				}
			}
		}
	case TextMatchModeEqual:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if value == prefix {
					return MatchResult{Origin: m.origin}, true
				}
			} else {
				if strings.EqualFold(value, prefix) {
					return MatchResult{Origin: m.origin}, true
				}
			}
		}
	case TextMatchModeContains:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.Contains(value, prefix) {
					return MatchResult{Origin: m.origin, MatcherType: string(m.Type())}, true
				}
			} else {
				if strings.Contains(strings.ToLower(value), strings.ToLower(prefix)) {
					return MatchResult{Origin: m.origin}, true
				}
			}
		}
	case TextMatchModeWord:
		if !m.caseSensitive {
			value = strings.ToLower(value)
		}

		for _, word := range strings.Split(value, " ") {
			for _, pattern := range m.patterns {
				if m.caseSensitive {
					if pattern == word {
						return MatchResult{Origin: m.origin}, true
					}
				} else {
					if strings.EqualFold(strings.ToLower(pattern), word) {
						return MatchResult{Origin: m.origin}, true
					}
				}
			}
		}
	}

	return MatchResult{}, false
}

func (m GeneralTextMatcher) Type() TextMatchType {
	return m.matcherType
}

func NewGeneralTextMatcher(origin string, matcherType TextMatchType, matchMode TextMatchMode, caseSensitive bool, attributes []string, patterns ...string) GeneralTextMatcher {
	return GeneralTextMatcher{
		origin:        origin,
		matcherType:   matcherType,
		mode:          matchMode,
		caseSensitive: caseSensitive,
		patterns:      patterns,
		attributes:    attributes,
	}
}
