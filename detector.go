package main

import (
	"context"
	"errors"
	"os"
	"time"
)

// // BD is the main application container
// type BD struct {
//	// TODO
//	// - estimate private steam account ages (find nearby non-private account)
//	// - "unmark" players, overriding any lists that may match
//	// - track rage quits
//	// - install vote fail mod
//	// - wipe map session stats k/d
//	// - track k/d over entire session?
//	// - track history of interactions with players
//	// - colourise messages that trigger
//	// - track stopwatch time-ish via 02/28/2023 - 23:40:21: Teams have been switched.
// }

// func getPlayerByName(name string) *store.Player {
//	playersMu.RLock()
//	defer playersMu.RUnlock()
//	for _, player := range players {
//		if player.Name == name {
//			return player
//		}
//	}
//	return nil
// }

//func checkHandler(ctx context.Context) {
//	defer slog.Debug("checkHandler exited")
//
//	checkTimer := time.NewTicker(DurationCheckTimer)
//
//	for {
//		select {
//		case <-ctx.Done():
//			return
//		case <-checkTimer.C:
//			player, errPlayer := d.players.bySteamID(d.Settings().SteamID)
//			if errPlayer != nil {
//				// We have not connected yet.
//				continue
//			}
//
//			d.checkPlayerStates(ctx, player.Team)
//		}
//	}
//}

// Shutdown closes any open rcon connection and will flush any player list to disk.
func Shutdown(ctx context.Context) error {
	if d.reader != nil && d.reader.tail != nil {
		d.reader.tail.Cleanup()
	}

	var err error

	d.rconMu.Lock()

	if d.rconConn != nil {
		LogClose(d.rconConn)
	}

	d.rconMu.Unlock()

	if errCloseDB := d.dataStore.Close(); errCloseDB != nil {
		err = errors.Join(errCloseDB, errCloseDatabase)
	}

	lCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if errWeb := d.Web.Shutdown(lCtx); errWeb != nil {
		err = errors.Join(errWeb, errCloseWeb)
	}

	return err
}

// Start handles starting up all the background services, starting the http service, opening the URL and launching the
// game if configured.
func Start(ctx context.Context) {
	go d.reader.start(ctx)
	go d.parser.start(ctx)
	go d.refreshLists(ctx)
	go d.incomingLogEventHandler(ctx)
	go d.stateUpdater(ctx)
	go d.cleanupHandler(ctx)
	go d.checkHandler(ctx)
	go d.statusUpdater(ctx)
	go d.processChecker(ctx)
	go d.discordStateUpdater(ctx)
	go d.profileUpdater(ctx)
	go d.autoKicker(ctx, d.kickRequestChan)

	if _, found := os.LookupEnv("TEST_PLAYERS"); found {
		go func() {
			generateTimer := time.NewTicker(time.Second * 5)

			for {
				select {
				case <-generateTimer.C:
					CreateTestPlayers(d, 24) //nolint:contextcheck
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	if running, errRunning := d.platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce.Load() && d.Settings().AutoLaunchGame {
			go d.LaunchGameAndWait()
		}
	}

}
