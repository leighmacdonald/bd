package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/rules"
)

type bigBrother struct {
	state    *gameState
	rcon     rconConnection
	settings *settingsManager
}

func newBigBrother(settings *settingsManager, rcon rconConnection, state *gameState) bigBrother {
	return bigBrother{settings: settings, rcon: rcon, state: state}
}

func (a bigBrother) start(ctx context.Context) {
	timer := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-timer.C:
			a.update()
		case <-ctx.Done():
			return
		}
	}
}

// announceMatch handles announcing after a match is triggered against a player.
func (a bigBrother) announceMatch(ctx context.Context, player PlayerState, matches []rules.MatchResult) {
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
func (a bigBrother) sendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
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

func (a bigBrother) watch(ctx context.Context) {
	updateTimer := time.NewTicker(time.Second)

	for {
		select {
		case <-updateTimer.C:
		}
	}
}

func (a bigBrother) update() {
}

func (a bigBrother) findCandidates() []PlayerState {
	a.state.players.RLock()
	defer a.state.players.RUnlock()

	return nil
}
