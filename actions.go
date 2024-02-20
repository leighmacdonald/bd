package main

import (
	"context"
	"errors"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"log/slog"
	"os"
	"time"
)

// unMark will unmark & remove a player from your local list. This *will not* unmark players from any
// other list sources. If you want to not kick someone on a 3rd party list, you can instead whitelist the player.
func unMark(ctx context.Context, re *rules.Engine, db store.Querier, state *gameState, sid64 steamid.SID64) (int, error) {
	player, errPlayer := getPlayerOrCreate(ctx, db, state.players, sid64)
	if errPlayer != nil {
		return 0, errPlayer
	}

	if !re.Unmark(sid64) {
		return 0, errNotMarked
	}

	var valid []*rules.MatchResult //nolint:prealloc

	for _, m := range player.Matches {
		if m.Origin == "local" {
			continue
		}

		valid = append(valid, m)
	}

	player.Matches = valid

	state.updateChan <- newMarkEvent(sid64, nil, false)

	return len(valid), nil
}

// mark will add a new entry in your local player list.
func mark(ctx context.Context, sm *settingsManager, db store.Querier, state *gameState, re *rules.Engine, sid64 steamid.SID64, attrs []string) error {
	player, errPlayer := getPlayerOrCreate(ctx, db, state.players, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	name := player.Personaname
	if name == "" {
		name = player.NamePrevious
	}

	if errMark := re.Mark(rules.MarkOpts{
		SteamID:    sid64,
		Attributes: attrs,
		Name:       name,
		Proof:      []string{},
	}); errMark != nil {
		return errors.Join(errMark, errMark)
	}

	outputFile, errOf := os.OpenFile(sm.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Join(errOf, errPlayerListOpen)
	}

	defer LogClose(outputFile)

	if errExport := re.ExportPlayers(rules.LocalRuleName, outputFile); errExport != nil {
		slog.Error("Failed to save updated player list", errAttr(errExport))
	}

	state.updateChan <- newMarkEvent(sid64, attrs, true)

	return nil
}

// whitelist prevents a player marked in 3rd party lists from being flagged for kicking.
func whitelist(ctx context.Context, db store.Querier, state *gameState, rules *rules.Engine, sid64 steamid.SID64, enabled bool) error {
	player, errPlayer := getPlayerOrCreate(ctx, db, state.players, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	player.Whitelist = enabled
	player.Dirty = true

	if errSave := db.PlayerUpdate(ctx, playerToPlayerUpdateParams(player)); errSave != nil {
		return errors.Join(errSave, errSavePlayer)
	}

	state.players.update(player)

	if enabled {
		rules.WhitelistAdd(sid64)
	} else {
		rules.WhitelistRemove(sid64)
	}

	state.updateChan <- newWhitelistEvent(player.SID64(), enabled)

	slog.Info("Update player whitelist status successfully",
		slog.Int64("steam_id", player.SteamID), slog.Bool("enabled", enabled))

	return nil
}

// addUserName will add an entry into the players username history table and check the username
// against the rules sets.
func addUserName(ctx context.Context, db store.Querier, player Player) error {
	if errSave := db.UserNameSave(ctx, store.UserNameSaveParams{
		SteamID:   player.SteamID,
		Name:      player.Personaname,
		CreatedOn: time.Now(),
	}); errSave != nil {
		return errors.Join(errSave, errSaveNames)
	}

	return nil
}

// addUserMessage will add an entry into the players message history table and check the message
// against the rules sets.
func addUserMessage(ctx context.Context, db store.Querier, player Player, message string, dead bool, teamOnly bool) error {
	return db.MessageSave(ctx, store.MessageSaveParams{
		SteamID:   player.SteamID,
		Message:   message,
		Team:      teamOnly,
		Dead:      dead,
		CreatedOn: time.Time{},
	})
}
