package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"time"
)

// statusUpdater is responsible for periodically sending `status` and `g15_dumpplayer` command to the
// game client.
type statusUpdater struct {
	rcon       rconConnection
	process    *processState
	state      *gameState
	updateRate time.Duration
	g15        g15Parser
}

func newStatusUpdater(rcon rconConnection, process *processState, state *gameState, updateRate time.Duration) statusUpdater {
	return statusUpdater{
		rcon:       rcon,
		process:    process,
		state:      state,
		updateRate: updateRate,
		g15:        newG15Parser(),
	}
}

func (s statusUpdater) start(ctx context.Context) {
	timer := time.NewTicker(s.updateRate)

	for {
		select {
		case <-timer.C:
			if !s.process.gameProcessActive.Load() {
				// Don't do anything until the game is open
				continue
			}

			if err := s.updatePlayerState(ctx); err != nil {
				slog.Error("failed to update player state", errAttr(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

// updatePlayerState fetches the current game state over rcon using both the `status` and `g15_dumpplayer` command
// output. The results are then parsed and applied to the current player and server states.
func (s statusUpdater) updatePlayerState(ctx context.Context) error {
	// Sent to client, response via log output
	_, errStatus := s.rcon.exec(ctx, "status", true)

	if errStatus != nil {
		return errors.Join(errStatus, errRCONStatus)
	}

	dumpPlayer, errDumpPlayer := s.rcon.exec(ctx, "g15_dumpplayer", true)
	if errDumpPlayer != nil {
		return errors.Join(errDumpPlayer, errRCONG15)
	}

	var dump DumpPlayer
	if errG15 := s.g15.Parse(bytes.NewBufferString(dumpPlayer), &dump); errG15 != nil {
		return errors.Join(errG15, errG15Parse)
	}

	for index, sid := range dump.SteamID {
		if index == 0 || index > 32 || !sid.Valid() {
			// Actual data always starts at 1
			continue
		}

		player, errPlayer := s.state.players.bySteamID(sid, true)
		if errPlayer != nil {
			// status command is what we use to add players to the active game.
			continue
		}

		player.MapTime = time.Since(player.MapTimeStart).Seconds()

		if player.Kills > 0 {
			player.KPM = float64(player.Kills) / (player.MapTime / 60)
		}

		player.Ping = dump.Ping[index]
		player.Score = dump.Score[index]
		player.Deaths = dump.Deaths[index]
		player.IsConnected = dump.Connected[index]
		player.Team = Team(dump.Team[index])
		player.Alive = dump.Alive[index]
		player.Health = dump.Health[index]
		player.Valid = dump.Valid[index]
		player.UserID = dump.UserID[index]
		player.UpdatedOn = time.Now()

		s.state.players.update(player)
	}

	return nil
}
