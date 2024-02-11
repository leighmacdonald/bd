package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/bd/addons"
	"github.com/leighmacdonald/bd/discord/client"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const (
	profileAgeLimit = time.Hour * 24
)

type Detector struct {
	players *playerState
	// logChan            chan string
	eventChan          chan LogEvent
	gameProcessActive  atomic.Bool
	startupTime        time.Time
	server             *Server
	serverMu           *sync.RWMutex
	reader             *LogReader
	parser             *LogParser
	rconConn           *rcon.RemoteConsole
	rconMu             *sync.RWMutex
	settings           UserSettings
	settingsMu         *sync.RWMutex
	discordPresence    *client.Client
	rules              *rules.Engine
	Web                *Web
	dataStore          DataStore
	profileUpdateQueue chan steamid.SID64
	stateUpdates       chan updateStateEvent
	cache              Cache
	Systray            *Systray
	platform           platform.Platform
	gameHasStartedOnce atomic.Bool
	dataSource         DataSource
	dataSourceMu       sync.RWMutex
	g15                Parser
	kickRequestChan    chan kickRequest
}

// NewDetector allocates and configures the root detector application.
func NewDetector(settings UserSettings, database DataStore, versionInfo Version, cache Cache,
	reader *LogReader, logChan chan string, dataSource DataSource,
) *Detector {
	plat := platform.New()
	isRunning, _ := plat.IsGameRunning()

	slog.Info("bd starting", slog.String("version", versionInfo.Version))

	if settings.APIKey != "" {
		if errAPIKey := steamweb.SetKey(settings.APIKey); errAPIKey != nil {
			slog.Error("Failed to set steam api key", errAttr(errAPIKey))
		}
	}

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

	eventChan := make(chan LogEvent)
	parser := NewLogParser(logChan, eventChan)

	application := &Detector{
		players:            newPlayerState(),
		eventChan:          eventChan,
		startupTime:        time.Now(),
		server:             &Server{},
		serverMu:           &sync.RWMutex{},
		settingsMu:         &sync.RWMutex{},
		reader:             reader,
		parser:             parser,
		rconConn:           nil,
		rconMu:             &sync.RWMutex{},
		settings:           settings,
		dataStore:          database,
		stateUpdates:       make(chan updateStateEvent),
		cache:              cache,
		discordPresence:    client.New(),
		rules:              rulesEngine,
		platform:           plat,
		profileUpdateQueue: make(chan steamid.SID64),
		dataSource:         dataSource,
		g15:                NewG15Parser(),
		kickRequestChan:    make(chan kickRequest),
	}

	application.gameProcessActive.Store(isRunning)
	application.gameHasStartedOnce.Store(isRunning)

	tray := NewSystray(
		plat.Icon(),
		func() {
			if errOpen := plat.OpenURL(fmt.Sprintf("http://%s/", settings.HTTPListenAddr)); errOpen != nil {
				slog.Error("Failed to open browser", errAttr(errOpen))
			}
		}, func() {
			go application.LaunchGameAndWait()
		},
	)

	application.Systray = tray

	web, errWeb := NewWeb(application)
	if errWeb != nil {
		panic(errWeb)
	}

	application.Web = web

	return application
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

// Settings returns a copy of the current user settings.
func (d *Detector) Settings() UserSettings {
	d.settingsMu.RLock()
	defer d.settingsMu.RUnlock()

	return d.settings
}

// SaveSettings first validates, then writes the current settings to disk.
func (d *Detector) SaveSettings(settings UserSettings) error {
	if errValidate := settings.Validate(); errValidate != nil {
		return errValidate
	}

	if errSave := settings.Save(); errSave != nil {
		return errSave
	}

	d.settingsMu.Lock()
	defer d.settingsMu.Unlock()

	d.dataSourceMu.Lock()
	defer d.dataSourceMu.Unlock()

	if settings.BdAPIEnabled {
		ds, errDs := NewAPIDataSource(settings.BdAPIAddress)
		if errDs != nil {
			return errors.Join(errDs, errDataSourceAPI)
		}

		d.dataSource = ds
	} else {
		ds, errDs := NewLocalDataSource(settings.APIKey)
		if errDs != nil {
			return errors.Join(errDs, errDataSourceLocal)
		}

		d.dataSource = ds
	}

	d.settings = settings

	return nil
}

func (d *Detector) Rules() *rules.Engine {
	return d.rules
}

const (
	maxVoiceBans   = 200
	voiceBansPerms = 0o755
)

// exportVoiceBans will write the most recent 200 bans to the `voice_ban.dt`. This must be done while the game is not
// currently running.
func (d *Detector) exportVoiceBans() error {
	bannedIds := d.rules.FindNewestEntries(maxVoiceBans, d.Settings().KickTags)
	if len(bannedIds) == 0 {
		return nil
	}

	vbPath := filepath.Join(d.Settings().TF2Dir, "voice_ban.dt")

	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, voiceBansPerms)
	if errOpen != nil {
		return errors.Join(errOpen, errVoiceBanOpen)
	}

	if errWrite := VoiceBanWrite(vbFile, bannedIds); errWrite != nil {
		return errors.Join(errWrite, errVoiceBanWrite)
	}

	slog.Info("Generated voice_ban.dt successfully")

	return nil
}

// LaunchGameAndWait is the main entry point to launching the game. It will install the included addon, write the
// voice bans out if enabled and execute the platform specific launcher command, blocking until exit.
func (d *Detector) LaunchGameAndWait() {
	defer func() {
		d.gameProcessActive.Store(false)
		d.rconMu.Lock()
		d.rconConn = nil
		d.rconMu.Unlock()
	}()

	settings := d.Settings()

	if errInstall := addons.Install(settings.TF2Dir); errInstall != nil {
		slog.Error("Error trying to install addon", errAttr(errInstall))
	}

	if settings.VoiceBansEnabled {
		if errVB := d.exportVoiceBans(); errVB != nil {
			slog.Error("Failed to export voiceban list", errAttr(errVB))
		}
	}

	args, errArgs := getLaunchArgs(
		settings.Rcon.Password,
		settings.Rcon.Port,
		settings.SteamDir,
		settings.SteamID)

	if errArgs != nil {
		slog.Error("Failed to get TF2 launch args", errAttr(errArgs))

		return
	}

	d.gameHasStartedOnce.Store(true)

	if errLaunch := d.platform.LaunchTF2(settings.TF2Dir, args); errLaunch != nil {
		slog.Error("Failed to launch game", errAttr(errLaunch))
	}
}

func (d *Detector) updateState(updates ...updateStateEvent) {
	for _, update := range updates {
		d.stateUpdates <- update
	}
}

// unMark will unmark & remove a player from your local list. This *will not* unmark players from any
// other list sources. If you want to not kick someone on a 3rd party list, you can instead whitelist the player.
func (d *Detector) unMark(ctx context.Context, sid64 steamid.SID64) (int, error) {
	player, errPlayer := d.GetPlayerOrCreate(ctx, sid64)
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
func (d *Detector) mark(ctx context.Context, sid64 steamid.SID64, attrs []string) error {
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
func (d *Detector) whitelist(ctx context.Context, sid64 steamid.SID64, enabled bool) error {
	player, errPlayer := d.GetPlayerOrCreate(ctx, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	player.Whitelisted = enabled
	player.Dirty = true

	if errSave := d.dataStore.SavePlayer(ctx, &player); errSave != nil {
		return errors.Join(errSave, errSavePlayer)
	}

	d.players.update(player)

	if enabled {
		d.rules.WhitelistAdd(sid64)
	} else {
		d.rules.WhitelistRemove(sid64)
	}

	go d.updateState(newWhitelistEvent(player.SteamID, enabled))

	slog.Info("Update player whitelist status successfully",
		slog.String("steam_id", player.SteamID.String()), slog.Bool("enabled", enabled))

	return nil
}

// updatePlayerState fetches the current game state over rcon using both the `status` and `g15_dumpplayer` command
// output. The results are then parsed and applied to the current player and server states.
func (d *Detector) updatePlayerState(ctx context.Context) error {
	if !d.ready(ctx) {
		return errInvalidReadyState
	}

	// Sent to client, response via log output
	_, errStatus := d.rconMulti("status")
	if errStatus != nil {
		return errors.Join(errStatus, errRCONStatus)
	}

	dumpPlayer, errDumpPlayer := d.rconMulti("g15_dumpplayer")
	if errDumpPlayer != nil {
		return errors.Join(errDumpPlayer, errRCONG15)
	}

	var dump DumpPlayer
	if errG15 := d.g15.Parse(bytes.NewBufferString(dumpPlayer), &dump); errG15 != nil {
		return errors.Join(errG15, errG15Parse)
	}

	for index, sid := range dump.SteamID {
		if index == 0 || index > 32 || !sid.Valid() {
			// Actual data always starts at 1
			continue
		}

		player, errPlayer := d.players.bySteamID(sid)
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

		d.players.update(player)
	}

	return nil
}

// statusUpdater is the background worker handling updating the game state.
func (d *Detector) statusUpdater(ctx context.Context) {
	defer slog.Debug("status updater exited")

	statusTimer := time.NewTicker(DurationStatusUpdateTimer)

	for {
		select {
		case <-statusTimer.C:
			if errUpdate := d.updatePlayerState(ctx); errUpdate != nil {
				slog.Debug("Failed to query state", errAttr(errUpdate))

				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetPlayerOrCreate attempts to fetch a player from the current player states. If it doesn't exist it will be
// inserted into the database and returned. If you only want players actively in the game, use the playerState functions
// instead.
func (d *Detector) GetPlayerOrCreate(ctx context.Context, sid64 steamid.SID64) (Player, error) {
	player, errPlayer := d.players.bySteamID(sid64)
	if errPlayer == nil {
		return player, nil
	}

	player = NewPlayer(sid64, "")

	if errGet := d.dataStore.GetPlayer(ctx, sid64, true, &player); errGet != nil {
		if !errors.Is(errGet, sql.ErrNoRows) {
			return player, errors.Join(errGet, errGetPlayer)
		}

		player.ProfileUpdatedOn = time.Now().AddDate(-1, 0, 0)
	}

	player.MapTimeStart = time.Now()

	defer d.players.update(player)

	if time.Since(player.ProfileUpdatedOn) < profileAgeLimit && player.Name != "" {
		return player, nil
	}

	go func() { d.profileUpdateQueue <- sid64 }()

	return player, nil
}

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

func (d *Detector) checkHandler(ctx context.Context) {
	defer slog.Debug("checkHandler exited")

	checkTimer := time.NewTicker(DurationCheckTimer)

	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTimer.C:
			player, errPlayer := d.players.bySteamID(d.Settings().SteamID)
			if errPlayer != nil {
				// We have not connected yet.
				continue
			}

			d.checkPlayerStates(ctx, player.Team)
		}
	}
}

// cleanupHandler is used to track of players and their expiration times. It will remove and reset expired players
// and server from the current known state once they have been disconnected for the timeout periods.
func (d *Detector) cleanupHandler(ctx context.Context) {
	const disconnectMsg = "Disconnected"

	defer slog.Debug("cleanupHandler exited")

	deleteTimer := time.NewTicker(time.Second * time.Duration(d.Settings().PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			settings := d.Settings()

			slog.Debug("Delete update input received", slog.String("state", "start"))
			d.serverMu.Lock()
			if time.Since(d.server.LastUpdate) > time.Second*time.Duration(settings.PlayerDisconnectTimeout) {
				name := d.server.ServerName
				if !strings.HasPrefix(name, disconnectMsg) {
					name = fmt.Sprintf("%s %s", disconnectMsg, name)
				}

				d.server = &Server{ServerName: name}
			}

			d.serverMu.Unlock()

			for _, player := range d.players.all() {
				if player.IsDisconnected() {
					player.IsConnected = false
					d.players.update(player)
				}

				if player.IsExpired() {
					d.players.remove(player.SteamID)
					slog.Debug("Flushing expired player", slog.Int64("steam_id", player.SteamID.Int64()))
				}
			}

			slog.Debug("Delete update input received", slog.String("state", "end"))

			deleteTimer.Reset(time.Second * time.Duration(settings.PlayerExpiredTimeout))
		}
	}
}

// addUserName will add an entry into the players username history table and check the username
// against the rules sets.
func (d *Detector) addUserName(ctx context.Context, player *Player) error {
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
func (d *Detector) addUserMessage(ctx context.Context, player *Player, message string, dead bool, teamOnly bool) error {
	userMessage, errMessage := NewUserMessage(player.SteamID, message, dead, teamOnly)
	if errMessage != nil {
		return errors.Join(errMessage, errCreateMessage)
	}

	if errSave := d.dataStore.SaveMessage(ctx, userMessage); errSave != nil {
		return errors.Join(errSave, errSaveMessage)
	}

	return nil
}

// refreshLists updates the 3rd party player lists using their update url.
func (d *Detector) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.settings.Lists)
	for _, list := range playerLists {
		boundList := list

		count, errImport := d.rules.ImportPlayers(&boundList)
		if errImport != nil {
			slog.Error("Failed to import player list", slog.String("name", boundList.FileInfo.Title), errAttr(errImport))
		} else {
			slog.Info("Imported player list", slog.String("name", boundList.FileInfo.Title), slog.Int("count", count))
		}
	}

	for _, list := range ruleLists {
		boundList := list

		count, errImport := d.rules.ImportRules(&boundList)
		if errImport != nil {
			slog.Error("Failed to import rules list (%s): %v\n", slog.String("name", boundList.FileInfo.Title), errAttr(errImport))
		} else {
			slog.Info("Imported rules list", slog.String("name", boundList.FileInfo.Title), slog.Int("count", count))
		}
	}
}

// checkPlayerStates will run a check against the current player state for matches.
func (d *Detector) checkPlayerStates(ctx context.Context, validTeam Team) {
	currentPlayers := d.players.all()
	for _, curPlayer := range currentPlayers {
		player := curPlayer

		if player.IsDisconnected() || len(player.Matches) > 0 {
			continue
		}

		if matchSteam := d.rules.MatchSteam(player.SteamID); matchSteam != nil { //nolint:nestif
			player.Matches = matchSteam

			if validTeam == player.Team {
				d.announceMatch(ctx, player, matchSteam)
				d.players.update(player)
			}
		} else if player.Name != "" {
			if matchName := d.rules.MatchName(player.Name); matchName != nil && validTeam == player.Team {
				player.Matches = matchName

				if validTeam == player.Team {
					d.announceMatch(ctx, player, matchName)
					d.players.update(player)
				}
			}
		}

		if player.Dirty {
			if errSave := d.dataStore.SavePlayer(ctx, &player); errSave != nil {
				slog.Error("Failed to save dirty player state", errAttr(errSave))

				continue
			}

			player.Dirty = false
		}
	}
}

// announceMatch handles announcing after a match is triggered against a player.
func (d *Detector) announceMatch(ctx context.Context, player Player, matches []*rules.MatchResult) {
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

// ensureRcon makes sure that the rcon client is connected.
func (d *Detector) ensureRcon(ctx context.Context) error {
	d.rconMu.RLock()
	if d.rconConn != nil {
		d.rconMu.RUnlock()

		return nil
	}
	d.rconMu.RUnlock()

	settings := d.Settings()

	conn, errConn := rcon.Dial(ctx, settings.Rcon.String(), settings.Rcon.Password, DurationRCONRequestTimeout)
	if errConn != nil {
		return errors.Join(errConn, fmt.Errorf("%v: %s", errRCONConnect, settings.Rcon.String()))
	}

	d.rconMu.Lock()
	d.rconConn = conn
	d.rconMu.Unlock()

	return nil
}

func (d *Detector) ready(ctx context.Context) bool {
	if !d.gameProcessActive.Load() {
		return false
	}

	if errRcon := d.ensureRcon(ctx); errRcon != nil {
		slog.Debug("RCON is not ready yet", errAttr(errRcon))

		return false
	}

	return true
}

// sendChat is used to send chat messages to the various chat interfaces in game: say|say_team|say_party.
func (d *Detector) sendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
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
		return fmt.Errorf("%v: %s", errInvalidChatType, destination)
	}

	return d.execRcon(cmd)
}

func (d *Detector) quitGame() error {
	if !d.gameProcessActive.Load() {
		return errNotMarked
	}

	return d.execRcon("quit")
}

func (d *Detector) execRcon(cmd string) error {
	d.rconMu.Lock()
	defer d.rconMu.Unlock()

	_, errExec := d.rconConn.Exec(cmd)
	if errExec != nil {
		return fmt.Errorf("%v: %s", errRCONExec, cmd)
	}

	return nil
}

// callVote handles sending the vote commands to the game client.
func (d *Detector) callVote(ctx context.Context, userID int, reason KickReason) error {
	if !d.ready(ctx) {
		return errInvalidReadyState
	}

	return d.execRcon(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
}

// processChecker handles checking and updating the running state of the tf2 process.
func (d *Detector) processChecker(ctx context.Context) {
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
func (d *Detector) Shutdown(ctx context.Context) error {
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
func (d *Detector) openApplicationPage() {
	appURL := fmt.Sprintf("http://%s", d.settings.HTTPListenAddr)
	if errOpen := d.platform.OpenURL(appURL); errOpen != nil {
		slog.Error("Failed to open URL", slog.String("url", appURL), errAttr(errOpen))
	}
}

// Start handles starting up all the background services, starting the http service, opening the URL and launching the
// game if configured.
func (d *Detector) Start(ctx context.Context) {
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

	d.openApplicationPage()
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
func (d *Detector) profileUpdater(ctx context.Context) {
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
func (d *Detector) applyRemoteData(data updatedRemoteData) {
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
func (d *Detector) fetchProfileUpdates(ctx context.Context, queued steamid.Collection) updatedRemoteData {
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

// rconMulti is used for rcon responses that exceed the size of a single rcon packet (g15_dumpplayer).
func (d *Detector) rconMulti(cmd string) (string, error) {
	d.rconMu.Lock()
	defer d.rconMu.Unlock()

	cmdID, errWrite := d.rconConn.Write(cmd)
	if errWrite != nil {
		return "", errors.Join(errWrite, errRCONExec)
	}

	var response string

	for {
		resp, respID, errRead := d.rconConn.Read()
		if errRead != nil {
			return "", errors.Join(errRead, errRCONRead)
		}

		if cmdID == respID {
			s := len(resp)
			response += resp

			if s < 4000 {
				break
			}
		}
	}

	return response, nil
}

// CreateTestPlayers will generate fake player data for testing purposes.
// nolint:gosec
func CreateTestPlayers(detector *Detector, count int) {
	idIdx := 0
	knownIds := steamid.Collection{
		"76561197998365611", "76561197977133523", "76561198065825165", "76561198004429398", "76561198182505218",
		"76561197989961569", "76561198183927541", "76561198005026984", "76561197997861796", "76561198377596915",
		"76561198336028289", "76561198066637626", "76561198818013048", "76561198196411029", "76561198079544034",
		"76561198008337801", "76561198042902038", "76561198013287458", "76561198038487121", "76561198046766708",
		"76561197963310062", "76561198017314810", "76561197967842214", "76561197984047970", "76561198020124821",
		"76561198010868782", "76561198022397372", "76561198016314731", "76561198087124802", "76561198024022137",
		"76561198015577906", "76561197997861796",
	}

	randPlayer := func(userId int) Player {
		team := Blu
		if userId%2 == 0 {
			team = Red
		}

		player, errPlayer := detector.GetPlayerOrCreate(context.Background(), knownIds[idIdx])
		if errPlayer != nil {
			panic(errPlayer)
		}

		if player.Name == "" {
			player.Name = fmt.Sprintf("%d - %s", userId, player.SteamID.String())
		}

		player.Visibility = steamweb.VisibilityPublic
		player.KillsOn = rand.Intn(20)
		player.RageQuits = rand.Intn(10)
		player.DeathsBy = rand.Intn(20)
		player.Team = team
		player.Connected = float64(rand.Intn(3600))
		player.UserID = userId
		player.Ping = rand.Intn(150)
		player.Kills = rand.Intn(50)
		player.Deaths = rand.Intn(300)

		idIdx++

		return player
	}

	var testPlayers []Player

	for i := 0; i < count; i++ {
		player := randPlayer(i)

		switch i {
		case 1:
			player.NumberOfVACBans = 2
			player.Notes = "User notes \ngo here"
			last := time.Now().AddDate(-1, 0, 0)
			player.LastVACBanOn = &last
		case 4:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			})

		case 7:
			player.Team = Spec
		}

		testPlayers = append(testPlayers, player)
	}

	detector.players.replace(testPlayers)
}

func (d *Detector) stateUpdater(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-d.stateUpdates:
			slog.Debug("Game state update input received", slog.String("kind", update.kind.String()))

			if update.kind == updateStatus && !update.source.Valid() {
				continue
			}

			switch update.kind { //nolint:exhaustive
			case updateKill:
				evt, ok := update.data.(killEvent)
				if !ok {
					continue
				}

				d.onKill(evt)
			case updateBans:
				evt, ok := update.data.(steamweb.PlayerBanState)
				if !ok {
					continue
				}

				d.onBans(evt)
			case updateKickAttempts:
				d.onKickAttempt(update.source)
			case updateStatus:
				evt, ok := update.data.(statusEvent)
				if !ok {
					continue
				}

				d.onStatus(ctx, update.source, evt)
			case updateTags:
				evt, ok := update.data.(tagsEvent)
				if !ok {
					continue
				}

				d.onTags(evt)
			case updateHostname:
				evt, ok := update.data.(hostnameEvent)
				if !ok {
					continue
				}

				d.onHostname(evt)
			case updateMap:
				evt, ok := update.data.(mapEvent)
				if !ok {
					continue
				}

				d.onMapName(evt)
			case changeMap:
				d.onMapChange()
			default:
				slog.Debug("unhandled state update case")
			}
		}
	}
}

func (d *Detector) onUpdateMessage(ctx context.Context, evt messageEvent) {
	player, errPlayer := d.players.byName(evt.name)
	if errPlayer != nil {
		return
	}

	if errUm := d.addUserMessage(ctx, &player, evt.message, evt.dead, evt.teamOnly); errUm != nil {
		slog.Error("Failed to handle user message", errAttr(errUm))
	}

	d.players.update(player)
}

func (d *Detector) onKill(evt killEvent) {
	ourSid := d.Settings().SteamID

	src, srcErr := d.players.byName(evt.sourceName)
	if srcErr != nil {
		return
	}

	target, targetErr := d.players.byName(evt.sourceName)
	if targetErr != nil {
		return
	}

	src.Kills++
	target.Deaths++

	if target.SteamID == ourSid {
		src.DeathsBy++
	}

	if src.SteamID == ourSid {
		target.KillsOn++
	}

	d.players.update(src)
	d.players.update(target)
}

func (d *Detector) onBans(evt steamweb.PlayerBanState) {
	player, errPlayer := d.players.bySteamID(evt.SteamID)
	if errPlayer != nil {
		return
	}

	player.NumberOfVACBans = evt.NumberOfVACBans
	player.NumberOfGameBans = evt.NumberOfGameBans
	player.CommunityBanned = evt.CommunityBanned
	player.EconomyBan = evt.EconomyBan

	if evt.DaysSinceLastBan > 0 {
		subTime := time.Now().AddDate(0, 0, -evt.DaysSinceLastBan)
		player.LastVACBanOn = &subTime
	}

	d.players.update(player)
}

func (d *Detector) onKickAttempt(steamID steamid.SID64) {
	player, errPlayer := d.players.bySteamID(steamID)
	if errPlayer != nil {
		return
	}

	player.KickAttemptCount++

	d.players.update(player)
}

func (d *Detector) onStatus(ctx context.Context, steamID steamid.SID64, evt statusEvent) {
	player, errPlayer := d.GetPlayerOrCreate(ctx, steamID)
	if errPlayer != nil {
		slog.Error("Failed to get or create player", errAttr(errPlayer))

		return
	}

	player.Ping = evt.ping
	player.Connected = evt.connected.Seconds()
	player.UpdatedOn = time.Now()
	player.UserID = evt.userID

	if player.Name != evt.name {
		player.Name = evt.name
		if errAddName := d.addUserName(ctx, &player); errAddName != nil {
			slog.Error("Could not save new user name", errAttr(errAddName))
		}
	}

	d.players.update(player)

	slog.Debug("Player status updated",
		slog.String("sid", steamID.String()),
		slog.Int("tags", evt.ping),
		slog.Int("uid", evt.userID),
		slog.String("name", evt.name),
		slog.Int("connected", int(evt.connected.Seconds())))
}

func (d *Detector) onTags(evt tagsEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.Tags = evt.tags
	d.server.LastUpdate = time.Now()

	slog.Debug("Tags updated", slog.String("tags", strings.Join(evt.tags, ",")))
}

func (d *Detector) onHostname(evt hostnameEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.ServerName = evt.hostname
	d.server.LastUpdate = time.Now()

	slog.Debug("Hostname changed", slog.String("hostname", evt.hostname))
}

func (d *Detector) onMapName(evt mapEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.CurrentMap = evt.mapName

	slog.Debug("Map changed", slog.String("map", evt.mapName))
}

func (d *Detector) onMapChange() {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	for _, curPlayer := range d.players.all() {
		player := curPlayer

		player.Kills = 0
		player.Deaths = 0
		player.MapTimeStart = time.Now()
		player.MapTime = 0

		d.players.update(player)
	}

	d.server.CurrentMap = ""
	d.server.ServerName = ""
}

// incomingLogEventHandler handles mapping incoming LogEvent payloads into the more generalized
// updateStateEvent used for all state updates.
func (d *Detector) incomingLogEventHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-d.eventChan:
			switch evt.Type { //nolint:exhaustive
			case EvtMap:
				// update = updateStateEvent{kind: updateMap, data: mapEvent{mapName: evt.MetaData}}
			case EvtHostname:
				d.onHostname(hostnameEvent{hostname: evt.MetaData})
			case EvtTags:
				d.onTags(tagsEvent{tags: strings.Split(evt.MetaData, ",")})
			case EvtAddress:
				pcs := strings.Split(evt.MetaData, ":")

				_, errPort := strconv.ParseUint(pcs[1], 10, 16)
				if errPort != nil {
					slog.Error("Failed to parse port: %v", errAttr(errPort), slog.String("port", pcs[1]))

					continue
				}

				parsedIP := net.ParseIP(pcs[0])
				if parsedIP == nil {
					slog.Error("Failed to parse ip", slog.String("ip", pcs[0]))

					continue
				}
			case EvtStatusID:
				d.onStatus(ctx, evt.PlayerSID, statusEvent{
					ping:      evt.PlayerPing,
					userID:    evt.UserID,
					name:      evt.Player,
					connected: evt.PlayerConnected,
				})
			case EvtDisconnect:
				d.onMapChange()
			case EvtKill:
				d.onKill(killEvent{victimName: evt.Victim, sourceName: evt.Player})
			case EvtMsg:
				d.onUpdateMessage(ctx, messageEvent{
					steamID:   evt.PlayerSID,
					name:      evt.Player,
					createdAt: evt.Timestamp,
					message:   evt.Message,
					teamOnly:  evt.TeamOnly,
					dead:      evt.Dead,
				})
			case EvtConnect:
			case EvtLobby:
			}
		}
	}
}

// discordStateUpdater handles updating the discord presence data with the current game state. It uses the
// discord local IPC socket.
func (d *Detector) discordStateUpdater(ctx context.Context) {
	const discordAppID = "1076716221162082364"

	timer := time.NewTicker(time.Second * 10)
	isRunning := false

	for {
		select {
		case <-timer.C:
			if !d.Settings().DiscordPresenceEnabled {
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
func (d *Detector) autoKicker(ctx context.Context, kickRequestChan chan kickRequest) {
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

			settings := d.Settings()

			if !settings.KickerEnabled {
				continue
			}

			if len(kickRequests) == 0 { //nolint:nestif
				kickable := d.players.kickable()
				if len(kickable) == 0 {
					continue
				}

				var valid []Player

				for _, player := range kickable {
					if player.MatchAttr(settings.KickTags) {
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

				player, errPlayer := d.players.bySteamID(request.steamID)
				if errPlayer != nil {
					slog.Error("Failed to get player to kick", errAttr(errPlayer))

					continue
				}

				reason = request.reason
				kickedPlayer = player
			}

			kickedPlayer.KickAttemptCount++

			d.players.update(kickedPlayer)

			if errVote := d.callVote(ctx, kickedPlayer.UserID, reason); errVote != nil {
				slog.Error("Failed to callvote", errAttr(errVote))
			}
		case <-ctx.Done():
			return
		}
	}
}
