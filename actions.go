package main

import (
	"context"
	"errors"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"os"
)

// unMark will unmark & remove a player from your local list. This *will not* unmark players from any
// other list sources. If you want to not kick someone on a 3rd party list, you can instead whitelist the player.
func unMark(ctx context.Context, store store.Querier, sid64 steamid.SID64) (int, error) {
	player, errPlayer := getPlayerOrCreate(ctx, store, sid64)
	if errPlayer != nil {
		return 0, errPlayer
	}

	if !d.rules.Unmark(sid64) {
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

	go d.updateState(newMarkEvent(sid64, nil, false))

	return len(valid), nil
}

// mark will add a new entry in your local player list.
func mark(ctx context.Context, sid64 steamid.SID64, attrs []string) error {
	player, errPlayer := d.GetPlayerOrCreate(ctx, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	name := player.Name
	if name == "" {
		name = player.NamePrevious
	}

	if errMark := d.rules.Mark(rules.MarkOpts{
		SteamID:    sid64,
		Attributes: attrs,
		Name:       name,
		Proof:      []string{},
	}); errMark != nil {
		return errors.Join(errMark, errMark)
	}

	settings := d.Settings()

	outputFile, errOf := os.OpenFile(settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Join(errOf, errPlayerListOpen)
	}

	defer LogClose(outputFile)

	if errExport := d.rules.ExportPlayers(rules.LocalRuleName, outputFile); errExport != nil {
		slog.Error("Failed to save updated player list", errAttr(errExport))
	}

	go d.updateState(newMarkEvent(sid64, attrs, true))

	return nil
}

// whitelist prevents a player marked in 3rd party lists from being flagged for kicking.
func whitelist(ctx context.Context, db store.Querier, state *playerState, rules *rules.Engine, sid64 steamid.SID64, enabled bool) error {
	player, errPlayer := getPlayerOrCreate(ctx, db, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	player.Whitelisted = enabled
	player.Dirty = true

	if errSave := db.PlayerUpdate(ctx, &player); errSave != nil {
		return errors.Join(errSave, errSavePlayer)
	}

	state.update(player)

	if enabled {
		rules.WhitelistAdd(sid64)
	} else {
		rules.WhitelistRemove(sid64)
	}

	go d.updateState(newWhitelistEvent(player.SteamID, enabled))

	slog.Info("Update player whitelist status successfully",
		slog.String("steam_id", player.SteamID.String()), slog.Bool("enabled", enabled))

	return nil
}

// addUserName will add an entry into the players username history table and check the username
// against the rules sets.
func addUserName(ctx context.Context, player *Player) error {
	unh, errMessage := NewUserNameHistory(player.SteamID, player.Name)
	if errMessage != nil {
		return errors.Join(errMessage, errGetNames)
	}

	if errSave := d.dataStore.SaveUserNameHistory(ctx, unh); errSave != nil {
		return errors.Join(errSave, errSaveNames)
	}

	return nil
}

// addUserMessage will add an entry into the players message history table and check the message
// against the rules sets.
func addUserMessage(ctx context.Context, player *Player, message string, dead bool, teamOnly bool) error {
	userMessage, errMessage := NewUserMessage(player.SteamID, message, dead, teamOnly)
	if errMessage != nil {
		return errors.Join(errMessage, errCreateMessage)
	}

	if errSave := d.dataStore.SaveMessage(ctx, userMessage); errSave != nil {
		return errors.Join(errSave, errSaveMessage)
	}

	return nil
}
