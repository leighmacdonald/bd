package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type kickRequest struct {
	steamID steamid.SteamID
	reason  KickReason
}

// overwatch handles looking through the current player states and finding targets to attempt to perform action against.
// This mainly includes announcing their status to lobby/in-game chat, and trying to kick them.
//
// Players may also be queued for automatic kicks manually by the player when they initiate a kick request from the
// ui/api. These kicks are given first priority.
type overwatch struct {
	state    *gameState
	rcon     rconConnection
	settings *settingsManager
	queued   []kickRequest
}

func newOverwatch(settings *settingsManager, rcon rconConnection, state *gameState) overwatch {
	return overwatch{settings: settings, rcon: rcon, state: state}
}

func (bb overwatch) start(ctx context.Context) {
	timer := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-timer.C:
			bb.update()
		case <-ctx.Done():
			return
		}
	}
}

// nextKickTarget searches for the next eligible target to initiate a vote kick against.
func (bb overwatch) nextKickTarget() (PlayerState, bool) {
	var validTargets []PlayerState

	// Pull names from the manual queue first.
	if len(bb.queued) > 0 {
		player, errNotFound := bb.state.players.bySteamID(bb.queued[0].steamID)
		if errNotFound != nil {
			// They are not in the game anymore.
			if len(bb.queued) > 1 {
				bb.queued = slices.Delete(bb.queued, 0, 1)
			} else {
				bb.queued = bb.queued[1:]
			}
		}

		validTargets = append(validTargets, player)
	}

	for _, player := range bb.state.players.current() {
		if len(player.Matches) > 0 && !player.Whitelist {
			validTargets = append(validTargets, player)
		}
	}

	if len(validTargets) == 0 {
		return PlayerState{}, false
	}

	// Find players we have not tried yet.
	sort.Slice(validTargets, func(i, j int) bool {
		return validTargets[i].KickAttemptCount < validTargets[j].KickAttemptCount
	})

	return validTargets[0], true
}

// announceMatch handles announcing after a match is triggered against a player.
func (bb overwatch) announceMatch(ctx context.Context, player PlayerState, matches []rules.MatchResult) {
	settings := bb.settings.Settings()

	if len(matches) == 0 {
		return
	}

	if time.Since(player.AnnouncedGeneralLast) >= DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if player.Whitelist {
			msg = "Matched whitelisted player"
		}

		for _, match := range matches {
			slog.Debug(msg,
				slog.String("match_type", match.MatcherType),
				slog.String("sid", player.SteamID.String()),
				slog.String("name", player.Personaname),
				slog.String("origin", match.Origin))
		}

		player.AnnouncedGeneralLast = time.Now()

		bb.state.players.update(player)
	}

	if player.Whitelist {
		return
	}

	if settings.PartyWarningsEnabled && time.Since(player.AnnouncedPartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := bb.sendChat(ctx, ChatDestParty, "(%d) [%s] [%s] %s ", player.UserID, match.Origin, strings.Join(match.Attributes, ","), player.Personaname); errLog != nil {
				slog.Error("Failed to send party log message", errAttr(errLog))

				return
			}
		}

		player.AnnouncedPartyLast = time.Now()

		bb.state.players.update(player)
	}
}

// sendChat is used to send chat messages to the various chat interfaces in game: say|say_team|say_party.
func (bb overwatch) sendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
	var cmd string

	switch destination {
	case ChatDestAll:
		cmd = fmt.Sprintf("say %s", fmt.Sprintf(format, args...))
	case ChatDestTeam:
		cmd = fmt.Sprintf("say_team %s", fmt.Sprintf(format, args...))
	case ChatDestParty:
		cmd = fmt.Sprintf("say_party %s", fmt.Sprintf(format, args...))
	default:
		return fmt.Errorf("%w: %s", errInvalidChatType, destination)
	}

	resp, errExec := bb.rcon.exec(ctx, cmd, false)
	if errExec != nil {
		return errExec
	}

	slog.Debug(resp, slog.String("cmd", cmd))

	return nil
}

func (bb overwatch) update() {
}

func (bb overwatch) kick(ctx context.Context, player PlayerState, reason KickReason) {
	player.KickAttemptCount++

	defer bb.state.players.update(player)

	cmd := fmt.Sprintf("callvote kick \"%d %s\"", player.UserID, reason)

	resp, errCallVote := bb.rcon.exec(ctx, cmd, false)
	if errCallVote != nil {
		slog.Error("Failed to call vote", slog.String("steam_id", player.SteamID.String()), errAttr(errCallVote))

		return
	}
	slog.Debug("Kick response", slog.String("resp", resp))
}
