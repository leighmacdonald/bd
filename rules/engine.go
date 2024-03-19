package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

var (
	ErrParseSteamID      = errors.New("failed to parse steam id")
	ErrDuplicateSteamID  = errors.New("duplicate steam id")
	ErrEncodePlayers     = errors.New("failed to encode player list")
	ErrUnknownPlayerList = errors.New("unknown player list")
	ErrEncodeRules       = errors.New("failed to encode rules")
	ErrUnknownRuleList   = errors.New("unknown rules list")
	ErrInvalidRegex      = errors.New("invalid regex pattern")
	ErrInvalidAttributes = errors.New("invalid attribute count")
)

type Engine struct {
	rulesLists  []*RuleSchema
	playerLists []*PlayerListSchema
	knownTags   []string
	sync.RWMutex
}

func New() *Engine {
	return &Engine{
		rulesLists:  []*RuleSchema{NewRuleSchema()},
		playerLists: []*PlayerListSchema{NewPlayerListSchema()},
		knownTags:   []string{},
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

// FindNewestEntries will scan all loaded lists and return the most recent matches as determined by the last seen attr.
// This is mostly only useful for exporting voice bans since there is a limited amount you can export and using the most
// recent seems like the most sensible option.
func (e *Engine) FindNewestEntries(max int, validAttrs []string) steamid.Collection {
	e.RLock()
	defer e.RUnlock()

	var matchers []SteamIDMatcherHandler

	for _, list := range e.playerLists {
		for _, m := range list.matchersSteam {
			if m.HasOneOfAttr(validAttrs...) {
				matchers = append(matchers, m)
			}
		}
	}

	sort.Slice(matchers, func(i, j int) bool {
		return matchers[i].LastSeen().After(matchers[j].LastSeen())
	})

	var valid steamid.Collection

	for i, s := range matchers {
		if i == max {
			break
		}

		valid = append(valid, s.SteamID())
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

// Unmark a player from the local player list.
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

	for idx := range list.Players {
		if list.Players[idx].SteamID == steamID {
			found = true

			continue
		}

		players = append(players, list.Players[idx])
	}

	list.Players = players

	var (
		userList      = e.UserPlayerList()
		validMatchers []SteamIDMatcherHandler
	)

	for _, matcher := range userList.matchersSteam {
		if _, matchFound := matcher.Match(steamID); matchFound {
			validMatchers = append(validMatchers, matcher)
		}
	}

	userList.matchersSteam = validMatchers

	return found
}

// Mark a player on the local player list.
func (e *Engine) Mark(opts MarkOpts) error {
	if len(opts.Attributes) == 0 {
		return ErrInvalidAttributes
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
				Time:       time.Now().Unix(),
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
				return errors.Join(errEncode, ErrEncodePlayers)
			}

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrUnknownPlayerList, listName)
}

// ExportRules writes the json encoded rules list matching the listName provided to the io.Writer.
func (e *Engine) ExportRules(listName string, writer io.Writer) error {
	e.RLock()
	defer e.RUnlock()

	for _, pl := range e.rulesLists {
		if listName == pl.FileInfo.Title {
			if errEncode := newJSONPrettyEncoder(writer).Encode(pl); errEncode != nil {
				return errors.Join(errEncode, ErrEncodeRules)
			}

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrUnknownRuleList, listName)
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

			list.RegisterAvatarMatcher(NewAvatarMatcher(list.FileInfo.Title, AvatarMatchExact, hashes...))

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
			return 0, errors.Join(steamid.ErrInvalidSID, ErrParseSteamID)
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

func (pls *PlayerListSchema) RegisterSteamIDMatcher(matcher SteamIDMatcherHandler) {
	pls.matchersSteam = append(pls.matchersSteam, matcher)
}

func (rs *RuleSchema) RegisterAvatarMatcher(matcher AvatarMatcherHandler) {
	rs.MatchersAvatar = append(rs.MatchersAvatar, matcher)
}

func (rs *RuleSchema) RegisterTextMatcher(matcher TextMatchHandler) {
	rs.MatchersText = append(rs.MatchersText, matcher)
}

func (rs *RuleSchema) matchTextType(text string, matchType TextMatchType) (MatchResult, bool) {
	for _, matcher := range rs.MatchersText {
		if matcher.Type() != TextMatchTypeAny && matcher.Type() != matchType {
			continue
		}

		if match, found := matcher.Match(text); found {
			return match, true
		}
	}

	return MatchResult{}, false
}

func (e *Engine) MatchSteam(steamID steamid.SID64) MatchResults {
	e.RLock()
	defer e.RUnlock()

	var matches MatchResults

	for _, list := range e.playerLists {
		for _, sm := range list.matchersSteam {
			if match, found := sm.Match(steamID); found {
				matches = append(matches, match)

				break
			}
		}
	}

	return matches
}

func (e *Engine) MatchName(name string) []MatchResult {
	var results MatchResults

	for _, list := range e.rulesLists {
		if match, found := list.matchTextType(name, TextMatchTypeName); found {
			results = append(results, match)

			continue
		}
	}

	return results
}

func (e *Engine) MatchMessage(text string) []MatchResult {
	var results MatchResults

	for _, list := range e.rulesLists {
		if match, found := list.matchTextType(text, TextMatchTypeMessage); found {
			results = append(results, match)

			continue
		}
	}

	return results
}

// func (e *Engine) matchAny(text string) *MatchResult {
//	   return e.matchTextType(text, TextMatchTypeAny)
// }

func (e *Engine) MatchAvatar(avatar []byte) []MatchResult {
	if avatar == nil {
		return nil
	}

	var (
		hexDigest = HashBytes(avatar)
		matches   []MatchResult
	)

	for _, list := range e.rulesLists {
		for _, matcher := range list.MatchersAvatar {
			if match, found := matcher.Match(hexDigest); found {
				matches = append(matches, match)

				break
			}
		}
	}

	return matches
}

const (
	maxVoiceBans   = 200
	voiceBansPerms = 0o755
)

// ExportVoiceBans will write the most recent 200 bans to the `voice_ban.dt`. This must be done while the game is not
// currently running.
func (e *Engine) ExportVoiceBans(tf2Dir string, kickTags []string) error {
	bannedIDs := e.FindNewestEntries(maxVoiceBans, kickTags)
	if len(bannedIDs) == 0 {
		return nil
	}

	vbPath := filepath.Join(tf2Dir, "voice_ban.dt")

	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, voiceBansPerms)
	if errOpen != nil {
		return errors.Join(errOpen, ErrVoiceBanOpen)
	}

	if errWrite := VoiceBanWrite(vbFile, bannedIDs); errWrite != nil {
		return errors.Join(errWrite, ErrVoiceBanWrite)
	}

	slog.Info("Generated voice_ban.dt successfully", slog.String("path", vbPath))

	return nil
}

func HashBytes(b []byte) string {
	hash := sha256.New()
	hash.Write(b)

	return hex.EncodeToString(hash.Sum(nil))
}
