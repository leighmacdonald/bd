package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
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

type kickRequest struct {
	steamID steamid.SID64
	reason  KickReason
}

type kickHandler struct {
	settings *settingsManager
	rcon     rconConnection
}

func newKickHandler(settings *settingsManager, rcon rconConnection) kickHandler {
	return kickHandler{
		settings: settings,
		rcon:     rcon,
	}
}

// autoKicker handles making kick votes. It prioritizes manual vote kick requests from the user before trying
// to kick players that match the auto kickable criteria. Auto kick attempts will cycle through the players with the least
// amount of kick attempts.
func (kh kickHandler) autoKicker(ctx context.Context, players *playerState, kickRequestChan chan kickRequest) {
	kickTicker := time.NewTicker(time.Millisecond * 100)

	var kickRequests []kickRequest

	for {
		select {
		case request := <-kickRequestChan:
			kickRequests = append(kickRequests, request)
		case <-kickTicker.C:
			var (
				kickTarget Player
				reason     KickReason
			)

			curSettings := kh.settings.Settings()

			if !curSettings.KickerEnabled {
				continue
			}

			if len(kickRequests) == 0 { //nolint:nestif
				kickable := players.kickable()
				if len(kickable) == 0 {
					continue
				}

				var valid []Player

				for _, player := range kickable {
					if player.MatchAttr(curSettings.KickTags) {
						valid = append(valid, player)
					}
				}

				if len(valid) == 0 {
					continue
				}

				sort.SliceStable(valid, func(i, j int) bool {
					return valid[i].KickAttemptCount < valid[j].KickAttemptCount
				})

				reason = KickReasonCheating
				kickTarget = valid[0]
			} else {
				request := kickRequests[0]

				if len(kickRequests) > 1 {
					kickRequests = kickRequests[1:]
				} else {
					kickRequests = nil
				}

				player, errPlayer := players.bySteamID(request.steamID.Int64())
				if errPlayer != nil {
					slog.Error("Failed to get player to kick", errAttr(errPlayer))

					continue
				}

				reason = request.reason
				kickTarget = player
			}

			kickTarget.KickAttemptCount++

			players.update(kickTarget)

			cmd := fmt.Sprintf("callvote kick \"%d %s\"", kickTarget.UserID, reason)

			resp, errCallVote := kh.rcon.exec(ctx, cmd, false)
			if errCallVote != nil {
				slog.Error("Failed to call vote", slog.String("steam_id", kickTarget.SID64().String()), errAttr(errCallVote))

				return
			}

			slog.Debug(resp, slog.String("cmd", cmd))
		case <-ctx.Done():
			return
		}
	}
}
