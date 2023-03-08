package rules

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

type MatchResult struct {
	Origin     string // Title of the list that the match was generated against
	Attributes []string
	//Proof       []string
	MatcherType string
}

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
	matchType  avatarMatchType
	origin     string
	hashes     []string
	attributes []string
}

func (m avatarMatcher) Type() avatarMatchType {
	return m.matchType
}

func (m avatarMatcher) Match(hexDigest string) *MatchResult {
	for _, hash := range m.hashes {
		if hash == hexDigest {
			return &MatchResult{Origin: m.origin, MatcherType: string(m.Type()), Attributes: m.attributes}
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
	steamID    steamid.SID64
	origin     string
	attributes []string
	lastSeen   playerLastSeen
}

func (m steamIDMatcher) Match(sid64 steamid.SID64) *MatchResult {
	if sid64 == m.steamID {
		return &MatchResult{Origin: m.origin, MatcherType: "steam_id", Attributes: m.attributes}
	}
	return nil
}

func newSteamIDMatcher(origin string, sid64 steamid.SID64, attributes []string) steamIDMatcher {
	return steamIDMatcher{steamID: sid64, origin: origin, attributes: attributes}
}

type regexTextMatcher struct {
	matcherType textMatchType
	patterns    []*regexp.Regexp
	origin      string
	attributes  []string
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

func newRegexTextMatcher(origin string, matcherType textMatchType, attributes []string, patterns ...string) (regexTextMatcher, error) {
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
		attributes:  attributes,
	}, nil
}

type generalTextMatcher struct {
	matcherType   textMatchType
	mode          textMatchMode
	caseSensitive bool
	patterns      []string
	attributes    []string
	origin        string
}

func (m generalTextMatcher) Match(value string) *MatchResult {
	switch m.mode {
	case textMatchModeStartsWith:
		for _, prefix := range m.patterns {
			if m.caseSensitive {
				if strings.HasPrefix(value, prefix) {
					return &MatchResult{Origin: m.origin, Attributes: m.attributes}
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

func newGeneralTextMatcher(origin string, matcherType textMatchType, matchMode textMatchMode, caseSensitive bool, attributes []string, patterns ...string) TextMatcher {
	return generalTextMatcher{
		origin:        origin,
		matcherType:   matcherType,
		mode:          matchMode,
		caseSensitive: caseSensitive,
		patterns:      patterns,
		attributes:    attributes,
	}
}
