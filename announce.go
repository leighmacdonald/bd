package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/rules"
)

type announceHandler struct {
	state    *gameState
	rcon     rconConnection
	settings *settingsManager
}

func newAnnounceHandler(settings *settingsManager, rcon rconConnection, state *gameState) announceHandler {
	return announceHandler{settings: settings, rcon: rcon, state: state}
}

// announceMatch handles announcing after a match is triggered against a player.
func (a announceHandler) announceMatch(ctx context.Context, player Player, matches []*rules.MatchResult) {
	settings := a.settings.Settings()

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
				slog.String("sid", player.SID64().String()),
				slog.String("name", player.Personaname),
				slog.String("origin", match.Origin))
		}

		player.AnnouncedGeneralLast = time.Now()

		a.state.players.update(player)
	}

	if player.Whitelist {
		return
	}

	if settings.PartyWarningsEnabled && time.Since(player.AnnouncedPartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := a.sendChat(ctx, ChatDestParty, "(%d) [%s] [%s] %s ", player.UserID, match.Origin, strings.Join(match.Attributes, ","), player.Personaname); errLog != nil {
				slog.Error("Failed to send party log message", errAttr(errLog))

				return
			}
		}

		player.AnnouncedPartyLast = time.Now()

		a.state.players.update(player)
	}
}

// sendChat is used to send chat messages to the various chat interfaces in game: say|say_team|say_party.
func (a announceHandler) sendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
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

	resp, errExec := a.rcon.exec(ctx, cmd, false)
	if errExec != nil {
		return errExec
	}

	slog.Debug(resp, slog.String("cmd", cmd))

	return nil
}
