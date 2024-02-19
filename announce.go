package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/rules"
	"strings"
	"time"
)

type announcer struct {
}

// announceMatch handles announcing after a match is triggered against a player.
func announceMatch(ctx context.Context, player Player, matches []*rules.MatchResult) {
	settings := d.Settings()

	if len(matches) == 0 {
		return
	}

	if time.Since(player.AnnouncedGeneralLast) >= DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if player.Whitelisted {
			msg = "Matched whitelisted player"
		}

		for _, match := range matches {
			slog.Debug(msg, slog.String("match_type", match.MatcherType),
				slog.String("steam_id", player.SteamID.String()), slog.String("name", player.Name), slog.String("origin", match.Origin))
		}

		player.AnnouncedGeneralLast = time.Now()

		d.players.update(player)
	}

	if player.Whitelisted {
		return
	}

	if settings.PartyWarningsEnabled && time.Since(player.AnnouncedPartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := d.sendChat(ctx, ChatDestParty, "(%d) [%s] [%s] %s ", player.UserID, match.Origin, strings.Join(match.Attributes, ","), player.Name); errLog != nil {
				slog.Error("Failed to send party log message", errAttr(errLog))

				return
			}
		}

		player.AnnouncedPartyLast = time.Now()

		d.players.update(player)
	}
}

// sendChat is used to send chat messages to the various chat interfaces in game: say|say_team|say_party.
func sendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
	if !d.ready(ctx) {
		return errInvalidReadyState
	}

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

	return d.execRcon(cmd)
}
