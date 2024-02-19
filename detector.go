package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const (
	profileAgeLimit = time.Hour * 24
)

func createRulesEngine(settings UserSettings) *rules.Engine {
	rulesEngine := rules.New()

	if settings.RunMode != ModeTest { //nolint:nestif
		// Try and load our existing custom players
		if platform.Exists(settings.LocalPlayerListPath()) {
			input, errInput := os.Open(settings.LocalPlayerListPath())
			if errInput != nil {
				slog.Error("Failed to open local player list", errAttr(errInput))
			} else {
				var localPlayersList rules.PlayerListSchema
				if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
					slog.Error("Failed to parse local player list", errAttr(errRead))
				} else {
					count, errPlayerImport := rulesEngine.ImportPlayers(&localPlayersList)
					if errPlayerImport != nil {
						slog.Error("Failed to import local player list", errAttr(errPlayerImport))
					} else {
						slog.Info("Loaded local player list", slog.Int("count", count))
					}
				}

				LogClose(input)
			}
		}

		// Try and load our existing custom rules
		if platform.Exists(settings.LocalRulesListPath()) {
			input, errInput := os.Open(settings.LocalRulesListPath())
			if errInput != nil {
				slog.Error("Failed to open local rules list", errAttr(errInput))
			} else {
				var localRules rules.RuleSchema
				if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
					slog.Error("Failed to parse local rules list", errAttr(errRead))
				} else {
					count, errRulesImport := rulesEngine.ImportRules(&localRules)
					if errRulesImport != nil {
						slog.Error("Failed to import local rules list", errAttr(errRulesImport))
					}

					slog.Debug("Loaded local rules list", slog.Int("count", count))
				}

				LogClose(input)
			}
		}
	}

	return rulesEngine
}

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

// callVote handles sending the vote commands to the game client.
func callVote(ctx context.Context, userID int, reason KickReason) error {
	if !d.ready(ctx) {
		return errInvalidReadyState
	}

	return d.execRcon(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
}

// processChecker handles checking and updating the running state of the tf2 process.
func processChecker(ctx context.Context) {
	ticker := time.NewTicker(DurationProcessTimeout)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			existingState := d.gameProcessActive.Load()

			newState, errRunningStatus := d.platform.IsGameRunning()
			if errRunningStatus != nil {
				slog.Error("Failed to get process run status", errAttr(errRunningStatus))

				continue
			}

			if existingState != newState {
				d.gameProcessActive.Store(newState)
				slog.Info("Game process state changed", slog.Bool("is_running", newState))
			}

			// Handle auto closing the app on game close if enabled
			if !d.gameHasStartedOnce.Load() || !d.Settings().AutoCloseOnGameExit {
				continue
			}

			if !newState {
				slog.Info("Auto-closing on game exit", slog.Duration("uptime", time.Since(d.startupTime)))
				os.Exit(0)
			}
		}
	}
}

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

// openApplicationPage launches the http frontend using the platform specific browser launcher function.
func openApplicationPage() {
	appURL := fmt.Sprintf("http://%s", d.settings.HTTPListenAddr)
	if errOpen := d.platform.OpenURL(appURL); errOpen != nil {
		slog.Error("Failed to open URL", slog.String("url", appURL), errAttr(errOpen))
	}
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

	go func() {
		if errWeb := d.Web.startWeb(ctx); errWeb != nil {
			slog.Error("Web start returned error", errAttr(errWeb))
		}
	}()

	if running, errRunning := d.platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce.Load() && d.Settings().AutoLaunchGame {
			go d.LaunchGameAndWait()
		}
	}

	if d.settings.RunMode == ModeRelease {
		d.openApplicationPage()
	}
}

// steamIDStringList transforms a steamid.Collection into a comma separated list of SID64 strings.
func steamIDStringList(collection steamid.Collection) string {
	ids := make([]string, len(collection))
	for index, steamID := range collection {
		ids[index] = steamID.String()
	}

	return strings.Join(ids, ",")
}

// profileUpdater will update the 3rd party data from remote APIs.
// It will wait a short amount of time between updates to attempt to batch send the requests
// as much as possible.
func profileUpdater(ctx context.Context) {
	var (
		queue       steamid.Collection
		update      = make(chan any)
		updateTimer = time.NewTicker(DurationUpdateTimer)
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			go func() { update <- true }()
		case steamID := <-d.profileUpdateQueue:
			queue = append(queue, steamID)
			if len(queue) == 100 {
				go func() { update <- true }()
			}
		case <-update:
			if len(queue) == 0 {
				continue
			}

			updateData := d.fetchProfileUpdates(ctx, queue)
			d.applyRemoteData(updateData)

			for _, player := range d.players.all() {
				localPlayer := player
				if errSave := d.dataStore.SavePlayer(ctx, &localPlayer); errSave != nil {
					if errSave.Error() != "sql: database is closed" {
						slog.Error("Failed to save updated player state",
							slog.String("sid", localPlayer.SteamID.String()), errAttr(errSave))
					}
				}

				d.players.update(localPlayer)
			}

			ourSteamID := d.Settings().SteamID

			for steamID, friends := range updateData.friends {
				for _, friend := range friends {
					if friend.SteamID == ourSteamID {
						if actualPlayer, errPlayer := d.players.bySteamID(steamID); errPlayer == nil {
							actualPlayer.OurFriend = true

							d.players.update(actualPlayer)

							break
						}
					}
				}
			}

			slog.Info("Updated",
				slog.Int("sums", len(updateData.summaries)), slog.Int("bans", len(updateData.bans)),
				slog.Int("sourcebans", len(updateData.sourcebans)), slog.Int("fiends", len(updateData.friends)))

			queue = nil
		}
	}
}

// applyRemoteData updates the current player states with new incoming data.
func applyRemoteData(data updatedRemoteData) {
	players := d.players.all()

	for _, curPlayer := range players {
		player := curPlayer
		for _, sum := range data.summaries {
			if sum.SteamID == player.SteamID {
				player.AvatarHash = sum.AvatarHash
				player.AccountCreatedOn = time.Unix(int64(sum.TimeCreated), 0)
				player.Visibility = sum.CommunityVisibilityState

				break
			}
		}

		for _, ban := range data.bans {
			if ban.SteamID == player.SteamID {
				player.CommunityBanned = ban.CommunityBanned
				player.CommunityBanned = ban.VACBanned
				player.NumberOfGameBans = ban.NumberOfGameBans
				player.NumberOfVACBans = ban.NumberOfVACBans
				player.EconomyBan = ban.EconomyBan

				if ban.VACBanned {
					since := time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
					player.LastVACBanOn = &since
				}

				break
			}
		}

		if sb, ok := data.sourcebans[player.SteamID]; ok {
			player.Sourcebans = sb
		}

		player.UpdatedOn = time.Now()
		player.ProfileUpdatedOn = player.UpdatedOn

		d.players.update(player)
	}
}

type updatedRemoteData struct {
	summaries  []steamweb.PlayerSummary
	bans       []steamweb.PlayerBanState
	sourcebans SourcebansMap
	friends    FriendMap
}

// fetchProfileUpdates handles fetching and updating new player data from the configured DataSource implementation,
// it handles fetching the following data points:
//
// - Valve Profile Summary
// - Valve Game/VAC Bans
// - Valve Friendslist
// - Scraped sourcebans data via bd-api at https://bd-api.roto.lol
//
// If the user does not configure their own steam api key using LocalDataSource, then the
// default bd-api backed APIDataSource will instead be used as a proxy for fetching the results.
func fetchProfileUpdates(ctx context.Context, queued steamid.Collection) updatedRemoteData {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	var (
		updated   updatedRemoteData
		waitGroup = &sync.WaitGroup{}
	)

	d.dataSourceMu.RLock()
	defer d.dataSourceMu.RUnlock()

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newSummaries, errSum := d.dataSource.Summaries(localCtx, c)
		if errSum == nil {
			updated.summaries = newSummaries
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newBans, errSum := d.dataSource.Bans(localCtx, c)
		if errSum == nil {
			updated.bans = newBans
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newSourcebans, errSum := d.dataSource.Sourcebans(localCtx, c)
		if errSum == nil {
			updated.sourcebans = newSourcebans
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newFriends, errSum := d.dataSource.Friends(localCtx, c)
		if errSum == nil {
			updated.friends = newFriends
		}
	}(queued)

	waitGroup.Wait()

	return updated
}

// discordStateUpdater handles updating the discord presence data with the current game state. It uses the
// discord local IPC socket.
func discordStateUpdater(ctx context.Context, presenceEnabled bool) {
	const discordAppID = "1076716221162082364"

	timer := time.NewTicker(time.Second * 10)
	isRunning := false

	for {
		select {
		case <-timer.C:
			if !presenceEnabled {
				if isRunning {
					// Logout of existing connection on settings change
					if errLogout := d.discordPresence.Logout(); errLogout != nil {
						slog.Error("Failed to logout of discord client", errAttr(errLogout))
					}

					isRunning = false
				}

				continue
			}

			if !isRunning {
				if errLogin := d.discordPresence.Login(discordAppID); errLogin != nil {
					slog.Debug("Failed to login to discord", errAttr(errLogin))

					continue
				}

				isRunning = true
			}

			if isRunning {
				d.serverMu.RLock()

				if errUpdate := discordUpdateActivity(d.discordPresence, len(d.players.all()),
					d.server, d.gameProcessActive.Load(), d.startupTime); errUpdate != nil {
					slog.Error("Failed to update discord activity", errAttr(errUpdate))

					isRunning = false
				}

				d.serverMu.RUnlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

type kickRequest struct {
	steamID steamid.SID64
	reason  KickReason
}

// autoKicker handles making kick votes. It prioritizes manual vote kick requests from the user before trying
// to kick players that match the auto kickable criteria. Auto kick attempts will cycle through the players with the least
// amount of kick attempts.
func autoKicker(ctx context.Context, players *playerState, kickRequestChan chan kickRequest) {
	kickTicker := time.NewTicker(time.Millisecond * 100)

	var kickRequests []kickRequest

	for {
		select {
		case request := <-kickRequestChan:
			kickRequests = append(kickRequests, request)
		case <-kickTicker.C:
			var (
				kickedPlayer Player
				reason       KickReason
			)

			curSettings := d.Settings()

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
				kickedPlayer = valid[0]
			} else {
				request := kickRequests[0]

				if len(kickRequests) > 1 {
					kickRequests = kickRequests[1:]
				} else {
					kickRequests = nil
				}

				player, errPlayer := players.bySteamID(request.steamID)
				if errPlayer != nil {
					slog.Error("Failed to get player to kick", errAttr(errPlayer))

					continue
				}

				reason = request.reason
				kickedPlayer = player
			}

			kickedPlayer.KickAttemptCount++

			players.update(kickedPlayer)

			if errVote := d.callVote(ctx, kickedPlayer.UserID, reason); errVote != nil {
				slog.Error("Failed to callvote", errAttr(errVote))
			}
		case <-ctx.Done():
			return
		}
	}
}
