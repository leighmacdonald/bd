package rules

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"sort"
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

func New(localRules *RuleSchema, localPlayers *PlayerListSchema) (*Engine, error) {
	re := Engine{
		RWMutex:        &sync.RWMutex{},
		matchersSteam:  nil,
		matchersText:   nil,
		matchersAvatar: nil,
	}
	if localRules != nil {
		if _, errImport := re.ImportRules(localRules); errImport != nil {
			return nil, errors.Wrap(errImport, "Failed to load local rules")
		}
	} else {
		ls := NewRuleSchema()
		re.rulesLists = append(re.rulesLists, &ls)
	}
	if localPlayers != nil {
		_, errImport := re.ImportPlayers(localPlayers)
		if errImport != nil {
			return nil, errors.Wrap(errImport, "Failed to load local players")
		}
	} else {
		ls := NewPlayerListSchema()
		re.playerLists = append(re.playerLists, &ls)
	}
	return &re, nil
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

type MarkOpts struct {
	SteamID    steamid.SID64
	Attributes []string
	Proof      []string
	Name       string
}

func (e *Engine) FindNewestEntries(max int, validAttrs []string) steamid.Collection {
	e.RLock()
	defer e.RUnlock()
	var matchers []steamIDMatcher
	for _, m := range e.matchersSteam {
		sm := m.(steamIDMatcher)
		valid := false
		for _, tag := range sm.attributes {
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
		matchers = append(matchers, sm)
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

func (e *Engine) Unmark(steamID steamid.SID64) bool {
	e.Lock()
	defer e.Unlock()
	if len(e.playerLists) == 0 {
		return false
	}
	found := false
	var players []playerDefinition
	for _, knownPlayer := range e.playerLists[0].Players {
		strId := steamID.String()
		if knownPlayer.SteamID == strId {
			found = true
			continue
		}
		players = append(players, knownPlayer)
	}
	e.playerLists[0].Players = players
	// Remove the matcher from memory
	var validMatchers []SteamIDMatcher
	for _, matcher := range e.matchersSteam {
		if match := matcher.Match(steamID); match == nil {
			validMatchers = append(validMatchers, matcher)
		}
	}
	e.matchersSteam = validMatchers
	return found
}

func (e *Engine) Mark(opts MarkOpts) error {
	if len(opts.Attributes) == 0 {
		return errors.New("Invalid attribute count")
	}
	e.Lock()
	updatedAttributes := false
	for idx, knownPlayer := range e.playerLists[0].Players {
		knownSid64, errSid64 := steamid.StringToSID64(knownPlayer.SteamID)
		if errSid64 != nil {
			continue
		}
		if knownSid64 == opts.SteamID {
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
				e.Unlock()
				return errDuplicateSteamID
			}
			e.playerLists[0].Players[idx].Attributes = append(e.playerLists[0].Players[idx].Attributes, newAttr...)
			updatedAttributes = true
		}
	}
	if !updatedAttributes {
		e.playerLists[0].Players = append(e.playerLists[0].Players, playerDefinition{
			Attributes: opts.Attributes,
			LastSeen: playerLastSeen{
				Time:       int(time.Now().Unix()),
				PlayerName: opts.Name,
			},
			SteamID: opts.SteamID.String(),
			Proof:   opts.Proof,
		})
	}
	e.Unlock()
	if !updatedAttributes {
		e.registerSteamIDMatcher(newSteamIDMatcher(LocalRuleName, opts.SteamID, opts.Attributes))
	}
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
func (e *Engine) ImportRules(list *RuleSchema) (int, error) {
	count := 0
	for _, rule := range list.Rules {
		if rule.Triggers.UsernameTextMatch != nil {
			attrs := rule.Triggers.UsernameTextMatch.Attributes
			if len(attrs) == 0 {
				attrs = append(attrs, "trigger_name")
			}
			e.registerTextMatcher(newGeneralTextMatcher(
				list.FileInfo.Title,
				textMatchTypeName,
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
			e.registerTextMatcher(newGeneralTextMatcher(
				list.FileInfo.Title,
				textMatchTypeMessage,
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
			e.registerAvatarMatcher(newAvatarMatcher(
				list.FileInfo.Title,
				avatarMatchExact,
				hashes...))
			count++
		}
	}
	e.rulesLists = append(e.rulesLists, list)
	return count, nil
}

// ImportPlayers loads the provided player list for matching
func (e *Engine) ImportPlayers(list *PlayerListSchema) (int, error) {
	var playerAttrs []string
	var count int
	for _, player := range list.Players {
		steamID, errSid := steamid.StringToSID64(player.SteamID)
		if errSid != nil {
			return 0, errors.Wrap(errSid, "Failed to parse steamid")
		}
		if !steamID.Valid() {
			return 0, errors.Errorf("Received malformed steamid: %v", steamID)
		}
		e.registerSteamIDMatcher(newSteamIDMatcher(list.FileInfo.Title, steamID, player.Attributes))
		playerAttrs = append(playerAttrs, player.Attributes...)
		count++
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
	return count, nil
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
	hexDigest := HashBytes(avatar)
	for _, matcher := range e.matchersAvatar {
		match := matcher.Match(hexDigest)
		if match != nil {
			return match
		}
	}
	return nil
}

func HashBytes(b []byte) string {
	hash := sha1.New()
	hash.Write(b)
	return hex.EncodeToString(hash.Sum(nil))
}
