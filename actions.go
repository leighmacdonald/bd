package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// unMark will unmark & remove a player from your local list. This *will not* unmark players from any
// other list sources. If you want to not kick someone on a 3rd party list, you can instead whitelist the player.
func unMark(ctx context.Context, re *rules.Engine, db store.Querier, _ *gameState, sid64 steamid.SteamID) (int, error) {
	player, errPlayer := loadPlayerOrCreate(ctx, db, sid64)
	if errPlayer != nil {
		return 0, errPlayer
	}

	if !re.Unmark(sid64) {
		return 0, errNotMarked
	}

	var valid []rules.MatchResult //nolint:prealloc

	for _, m := range player.Matches {
		if m.Origin == "local" {
			continue
		}

		valid = append(valid, m)
	}

	player.Matches = valid

	return len(valid), nil
}

// mark will add a new entry in your local player list.
func mark(ctx context.Context, sm userSettings, db store.Querier, state *gameState, re *rules.Engine, sid64 steamid.SteamID, attrs []string) error {
	player, errPlayer := state.players.bySteamID(sid64)
	if errPlayer != nil {
		if !errors.Is(errPlayer, errPlayerNotFound) {
			return errPlayer
		}
		created, errCreate := loadPlayerOrCreate(ctx, db, sid64)
		if errCreate != nil {
			return errCreate
		}
		player = created
	}

	if errMark := re.Mark(rules.MarkOpts{
		SteamID:    sid64,
		Attributes: attrs,
		Name:       player.Personaname,
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

	return nil
}

// whitelist prevents a player marked in 3rd party lists from being flagged for kicking.
func whitelist(ctx context.Context, db store.Querier, state *gameState, sid64 steamid.SteamID, enabled bool) error {
	player, errPlayer := loadPlayerOrCreate(ctx, db, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	player.Whitelist = enabled

	if errSave := db.PlayerUpdate(ctx, player.toUpdateParams()); errSave != nil {
		return errors.Join(errSave, errSavePlayer)
	}

	state.players.update(player)

	slog.Info("Update player whitelist status successfully",
		slog.String("steam_id", player.SteamID.String()), slog.Bool("enabled", enabled))

	return nil
}
