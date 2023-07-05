package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var ErrDuplicateSteamID = errors.New("duplicate steam id")

type Engine struct {
	rulesLists  []*RuleSchema
	playerLists []*PlayerListSchema
	knownTags   []string
	whitelist   steamid.Collection
	sync.RWMutex
}

func New() *Engine {
	return &Engine{
		rulesLists:  []*RuleSchema{NewRuleSchema()},
		playerLists: []*PlayerListSchema{NewPlayerListSchema()},
		knownTags:   []string{},
		whitelist:   steamid.Collection{},
		RWMutex:     sync.RWMutex{},
	}
}

const (
	exportIndentSize = 4
)

type MarkOpts struct {
	SteamID    steamid.SID64
	Attributes []string
	Proof      []string
	Name       string
}

func (e *Engine) Whitelisted(sid64 steamid.SID64) bool {
	e.RLock()
	defer e.RUnlock()

	for _, entry := range e.whitelist {
		if entry == sid64 {
			return true
		}
	}

	return false
}

func (e *Engine) WhitelistAdd(sid64 steamid.SID64) bool {
	if e.Whitelisted(sid64) {
		return false
	}

	e.Lock()
	defer e.Unlock()

	e.whitelist = append(e.whitelist, sid64)

	return true
}

func (e *Engine) WhitelistRemove(sid64 steamid.SID64) bool {
	e.Lock()
	defer e.Unlock()

	var (
		removed = false
		newWl   steamid.Collection
	)

	for _, whitelistEntry := range e.whitelist {
		if sid64 == whitelistEntry {
			removed = true

			continue
		}

		newWl = append(newWl, whitelistEntry)
	}

	e.whitelist = newWl

	return removed
}

func (e *Engine) FindNewestEntries(max int, validAttrs []string) steamid.Collection {
	e.RLock()
	defer e.RUnlock()

	var matchers []SteamIDMatcher

	for _, list := range e.playerLists {
		for _, m := range list.matchersSteam {
			steamMatcher, ok := m.(SteamIDMatcher)
			if !ok {
				continue
			}

			valid := false

			for _, tag := range steamMatcher.attributes {
				for _, okTags := range validAttrs {
					if strings.EqualFold(tag, okTags) {
						valid = true

						break
					}
				}
			}

			if !valid {
				continue
			}

			matchers = append(matchers, steamMatcher)
		}
	}

	sort.Slice(matchers, func(i, j int) bool {
		return matchers[i].lastSeen.Time > matchers[j].lastSeen.Time
	})

	var valid steamid.Collection

	for i, s := range matchers {
		if i == max {
			break
		}

		valid = append(valid, s.steamID)
	}

	return valid
}

func (e *Engine) UserPlayerList() *PlayerListSchema {
	for _, list := range e.playerLists {
		if list.FileInfo.Title == LocalRuleName {
			return list
		}
	}

	panic("User player list schema doesn't exist")
}

func (e *Engine) UserRuleList() *RuleSchema {
	for _, list := range e.rulesLists {
		if list.FileInfo.Title == LocalRuleName {
			return list
		}
	}

	panic("User rules schema doesn't exist")
}

func (e *Engine) Unmark(steamID steamid.SID64) bool {
	e.Lock()
	defer e.Unlock()

	if len(e.playerLists) == 0 {
		return false
	}

	var ( //nolint:prealloc
		found   = false
		list    = e.UserPlayerList()
		players []PlayerDefinition
	)

	for _, knownPlayer := range list.Players {
		strID := steamID
		if knownPlayer.SteamID == strID {
			found = true

			continue
		}

		players = append(players, knownPlayer)
	}

	list.Players = players

	var (
		userList      = e.UserPlayerList()
		validMatchers []SteamIDMatcherI
	)

	for _, matcher := range userList.matchersSteam {
		if match := matcher.Match(steamID); match == nil {
			validMatchers = append(validMatchers, matcher)
		}
	}

	userList.matchersSteam = validMatchers

	return found
}

func (e *Engine) Mark(opts MarkOpts) error {
	if len(opts.Attributes) == 0 {
		return errors.New("Invalid attribute count")
	}

	e.Lock()
	defer e.Unlock()

	var (
		updatedAttributes = false
		userList          = e.UserPlayerList()
	)

	for idx, knownPlayer := range userList.Players {
		if !knownPlayer.SteamID.Valid() {
			continue
		}

		if knownPlayer.SteamID == opts.SteamID {
			var newAttr []string

			for _, updatedAttr := range opts.Attributes {
				isNew := true

				for _, existingAttr := range knownPlayer.Attributes {
					if strings.EqualFold(updatedAttr, existingAttr) {
						isNew = false

						break
					}
				}

				if isNew {
					newAttr = append(newAttr, updatedAttr)
				}
			}

			if len(newAttr) == 0 {
				return ErrDuplicateSteamID
			}

			userList.Players[idx].Attributes = append(userList.Players[idx].Attributes, newAttr...)
			updatedAttributes = true
		}
	}

	if !updatedAttributes {
		userList.Players = append(userList.Players, PlayerDefinition{
			Attributes: opts.Attributes,
			LastSeen: PlayerLastSeen{
				Time:       int(time.Now().Unix()),
				PlayerName: opts.Name,
			},
			SteamID: opts.SteamID,
			Proof:   opts.Proof,
		})
	}

	if !updatedAttributes {
		userList.RegisterSteamIDMatcher(NewSteamIDMatcher(LocalRuleName, opts.SteamID, opts.Attributes))
	}

	return nil
}

// UniqueTags returns a list of the unique known tags across all player lists.
func (e *Engine) UniqueTags() []string {
	e.RLock()
	defer e.RUnlock()

	if e.knownTags == nil {
		return []string{}
	}

	return e.knownTags
}

func newJSONPrettyEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetIndent("", strings.Repeat(" ", exportIndentSize))

	return enc
}

// ExportPlayers writes the json encoded player list matching the listName provided to the io.Writer.
func (e *Engine) ExportPlayers(listName string, writer io.Writer) error {
	e.RLock()
	defer e.RUnlock()

	for _, pl := range e.playerLists {
		if listName == pl.FileInfo.Title {
			if errEncode := newJSONPrettyEncoder(writer).Encode(pl); errEncode != nil {
				return errors.Wrap(errEncode, "Failed to encode player list")
			}

			return nil
		}
	}

	return errors.Errorf("Unknown player list: %s", listName)
}

// ExportRules writes the json encoded rules list matching the listName provided to the io.Writer.
func (e *Engine) ExportRules(listName string, writer io.Writer) error {
	e.RLock()
	defer e.RUnlock()

	for _, pl := range e.rulesLists {
		if listName == pl.FileInfo.Title {
			if errEncode := newJSONPrettyEncoder(writer).Encode(pl); errEncode != nil {
				return errors.Wrap(errEncode, "Failed to encode rules")
			}

			return nil
		}
	}

	return errors.Errorf("Unknown rule list: %s", listName)
}

// ImportRules loads the provided ruleset for use.
func (e *Engine) ImportRules(list *RuleSchema) (int, error) {
	count := 0

	for _, rule := range list.Rules {
		if rule.Triggers.UsernameTextMatch != nil {
			attrs := rule.Triggers.UsernameTextMatch.Attributes
			if len(attrs) == 0 {
				attrs = append(attrs, "trigger_name")
			}

			list.RegisterTextMatcher(NewGeneralTextMatcher(
				list.FileInfo.Title,
				TextMatchTypeName,
				rule.Triggers.UsernameTextMatch.Mode,
				rule.Triggers.UsernameTextMatch.CaseSensitive,
				attrs,
				rule.Triggers.UsernameTextMatch.Patterns...))
			count++
		}

		if rule.Triggers.ChatMsgTextMatch != nil {
			attrs := rule.Triggers.ChatMsgTextMatch.Attributes
			if len(attrs) == 0 {
				attrs = append(attrs, "trigger_msg")
			}

			list.RegisterTextMatcher(NewGeneralTextMatcher(
				list.FileInfo.Title,
				TextMatchTypeMessage,
				rule.Triggers.ChatMsgTextMatch.Mode,
				rule.Triggers.ChatMsgTextMatch.CaseSensitive,
				attrs,
				rule.Triggers.ChatMsgTextMatch.Patterns...))
			count++
		}

		if len(rule.Triggers.AvatarMatch) > 0 {
			var hashes []string

			for _, h := range rule.Triggers.AvatarMatch {
				if len(h.AvatarHash) != 40 {
					continue
				}

				hashes = append(hashes, h.AvatarHash)
			}

			list.RegisterAvatarMatcher(NewAvatarMatcher(
				list.FileInfo.Title,
				AvatarMatchExact,
				hashes...))
			count++
		}
	}

	e.rulesLists = append(e.rulesLists, list)

	return count, nil
}

// ImportPlayers loads the provided player list for matching.
func (e *Engine) ImportPlayers(list *PlayerListSchema) (int, error) {
	var (
		playerAttrs []string
		count       int
	)

	for _, player := range list.Players {
		if !player.SteamID.Valid() {
			return 0, errors.Wrap(steamid.ErrInvalidSID, "Failed to parse steamid")
		}

		list.RegisterSteamIDMatcher(NewSteamIDMatcher(list.FileInfo.Title, player.SteamID, player.Attributes))

		playerAttrs = append(playerAttrs, player.Attributes...)
		count++
	}

	e.Lock()
	defer e.Unlock()

	var newLists []*PlayerListSchema

	for _, lst := range e.playerLists {
		if lst.FileInfo.Title != list.FileInfo.Title {
			newLists = append(newLists, lst)
		}
	}

	newLists = append(newLists, list)

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

	e.playerLists = newLists

	return count, nil
}

func (pls *PlayerListSchema) RegisterSteamIDMatcher(matcher SteamIDMatcherI) {
	pls.matchersSteam = append(pls.matchersSteam, matcher)
}

func (rs *RuleSchema) RegisterAvatarMatcher(matcher AvatarMatcherI) {
	rs.MatchersAvatar = append(rs.MatchersAvatar, matcher)
}

func (rs *RuleSchema) RegisterTextMatcher(matcher TextMatcher) {
	rs.MatchersText = append(rs.MatchersText, matcher)
}

func (rs *RuleSchema) matchTextType(text string, matchType TextMatchType) *MatchResult {
	for _, matcher := range rs.MatchersText {
		if matcher.Type() != TextMatchTypeAny && matcher.Type() != matchType {
			continue
		}

		match := matcher.Match(text)
		if match != nil {
			return match
		}
	}

	return nil
}

func (e *Engine) MatchSteam(steamID steamid.SID64) []*MatchResult {
	if e.Whitelisted(steamID) {
		return nil
	}

	e.RLock()
	defer e.RUnlock()

	var matches []*MatchResult

	for _, list := range e.playerLists {
		for _, sm := range list.matchersSteam {
			match := sm.Match(steamID)
			if match != nil {
				matches = append(matches, match)

				break
			}
		}
	}

	return matches
}

func (e *Engine) MatchName(name string) []*MatchResult {
	var found []*MatchResult

	for _, list := range e.rulesLists {
		match := list.matchTextType(name, TextMatchTypeName)
		if match != nil {
			found = append(found, match)

			continue
		}
	}

	return found
}

func (e *Engine) MatchMessage(text string) []*MatchResult {
	var found []*MatchResult

	for _, list := range e.rulesLists {
		match := list.matchTextType(text, TextMatchTypeMessage)
		if match != nil {
			found = append(found, match)

			continue
		}
	}

	return found
}

// func (e *Engine) matchAny(text string) *MatchResult {
//	   return e.matchTextType(text, TextMatchTypeAny)
// }

func (e *Engine) MatchAvatar(avatar []byte) []*MatchResult {
	if avatar == nil {
		return nil
	}

	var (
		hexDigest = HashBytes(avatar)
		matches   []*MatchResult
	)

	for _, list := range e.rulesLists {
		for _, matcher := range list.MatchersAvatar {
			match := matcher.Match(hexDigest)
			if match != nil {
				matches = append(matches, match)

				break
			}
		}
	}

	return matches
}

func HashBytes(b []byte) string {
	hash := sha256.New()
	hash.Write(b)

	return hex.EncodeToString(hash.Sum(nil))
}
