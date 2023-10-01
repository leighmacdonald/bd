package detector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	gerrors "errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/bd/pkg/discord/client"
	"github.com/leighmacdonald/bd/pkg/g15"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/bd/pkg/voiceban"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var ErrInvalidReadyState = errors.New("Invalid ready state")

const (
	profileAgeLimit = time.Hour * 24
)

type Detector struct {
	log     *zap.Logger
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
	tr                 *tr.Translator
	Web                *Web
	dataStore          store.DataStore
	profileUpdateQueue chan steamid.SID64
	stateUpdates       chan updateStateEvent
	cache              Cache
	Systray            *Systray
	platform           platform.Platform
	gameHasStartedOnce atomic.Bool
	dataSource         DataSource
	g15                g15.Parser
}

func New(logger *zap.Logger, settings UserSettings, database store.DataStore, versionInfo Version, cache Cache,
	reader *LogReader, logChan chan string, dataSource DataSource,
) *Detector {
	plat := platform.New()
	isRunning, _ := plat.IsGameRunning()

	translator, errTrans := tr.NewTranslator()
	if errTrans != nil {
		panic(errTrans)
	}

	logger.Info("bd starting", zap.String("version", versionInfo.Version))

	// tr, errTranslator := tr.NewTranslator()
	// if errTranslator != nil {
	// 	rootLogger.Error("Failed to load translations", "err", errTranslator)
	// }

	if settings.APIKey != "" {
		if errAPIKey := steamweb.SetKey(settings.APIKey); errAPIKey != nil {
			logger.Error("Failed to set steam api key", zap.Error(errAPIKey))
		}
	}

	rulesEngine := rules.New()

	if settings.RunMode != ModeTest { //nolint:nestif
		// Try and load our existing custom players/rules
		if util.Exists(settings.LocalPlayerListPath()) {
			input, errInput := os.Open(settings.LocalPlayerListPath())
			if errInput != nil {
				logger.Error("Failed to open local player list", zap.Error(errInput))
			} else {
				var localPlayersList rules.PlayerListSchema
				if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
					logger.Error("Failed to parse local player list", zap.Error(errRead))
				} else {
					count, errPlayerImport := rulesEngine.ImportPlayers(&localPlayersList)
					if errPlayerImport != nil {
						logger.Error("Failed to import local player list", zap.Error(errPlayerImport))
					} else {
						logger.Info("Loaded local player list", zap.Int("count", count))
					}
				}
				util.LogClose(logger, input)
			}
		}

		if util.Exists(settings.LocalRulesListPath()) {
			input, errInput := os.Open(settings.LocalRulesListPath())
			if errInput != nil {
				logger.Error("Failed to open local rules list", zap.Error(errInput))
			} else {
				var localRules rules.RuleSchema
				if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
					logger.Error("Failed to parse local rules list", zap.Error(errRead))
				} else {
					count, errRulesImport := rulesEngine.ImportRules(&localRules)
					if errRulesImport != nil {
						logger.Error("Failed to import local rules list", zap.Error(errRulesImport))
					}
					logger.Debug("Loaded local rules list", zap.Int("count", count))
				}
				util.LogClose(logger, input)
			}
		}
	}

	eventChan := make(chan LogEvent)
	parser := NewLogParser(logger, logChan, eventChan)

	application := &Detector{
		log:                logger,
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
		tr:                 translator,
		platform:           plat,
		profileUpdateQueue: make(chan steamid.SID64),
		dataSource:         dataSource,
		g15:                g15.New(),
	}

	application.gameProcessActive.Store(isRunning)
	application.gameHasStartedOnce.Store(isRunning)

	tray := NewSystray(
		logger,
		plat.Icon(),
		func() {
			if errOpen := plat.OpenURL(fmt.Sprintf("http://%s/", settings.HTTPListenAddr)); errOpen != nil {
				logger.Error("Failed to open browser", zap.Error(errOpen))
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
//
// }

func (d *Detector) Settings() UserSettings {
	d.settingsMu.RLock()
	defer d.settingsMu.RUnlock()

	return d.settings
}

func (d *Detector) SaveSettings(settings UserSettings) error {
	if errValidate := settings.Validate(); errValidate != nil {
		return errValidate
	}

	if errSave := settings.Save(); errSave != nil {
		return errSave
	}

	d.settingsMu.Lock()
	defer d.settingsMu.Unlock()

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

func (d *Detector) exportVoiceBans() error {
	bannedIds := d.rules.FindNewestEntries(maxVoiceBans, d.Settings().KickTags)
	if len(bannedIds) == 0 {
		return nil
	}

	vbPath := filepath.Join(d.Settings().TF2Dir, "voice_ban.dt")

	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, voiceBansPerms)
	if errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open voicebans file")
	}

	if errWrite := voiceban.Write(vbFile, bannedIds); errWrite != nil {
		return errors.Wrap(errWrite, "Failed to write voicebans file")
	}

	d.log.Info("Generated voice_ban.dt successfully")

	return nil
}

func (d *Detector) LaunchGameAndWait() {
	defer func() {
		d.gameProcessActive.Store(false)
		d.rconMu.Lock()
		d.rconConn = nil
		d.rconMu.Unlock()
	}()

	settings := d.Settings()

	if errInstall := addons.Install(settings.TF2Dir); errInstall != nil {
		d.log.Error("Error trying to install addon", zap.Error(errInstall))
	}

	if settings.VoiceBansEnabled {
		if errVB := d.exportVoiceBans(); errVB != nil {
			d.log.Error("Failed to export voiceban list", zap.Error(errVB))
		}
	}

	args, errArgs := getLaunchArgs(
		settings.Rcon.Password,
		settings.Rcon.Port,
		settings.SteamDir,
		settings.SteamID)

	if errArgs != nil {
		d.log.Error("Failed to get TF2 launch args", zap.Error(errArgs))

		return
	}

	d.gameHasStartedOnce.Store(true)

	if errLaunch := d.platform.LaunchTF2(settings.TF2Dir, args); errLaunch != nil {
		d.log.Error("Failed to launch game", zap.Error(errLaunch))
	}
}

func (d *Detector) updateState(updates ...updateStateEvent) {
	for _, update := range updates {
		d.stateUpdates <- update
	}
}

var errNotMarked = errors.New("Mark does not exist")

func (d *Detector) UnMark(ctx context.Context, sid64 steamid.SID64) (int, error) {
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

func (d *Detector) Mark(ctx context.Context, sid64 steamid.SID64, attrs []string) error {
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
		return errors.Wrap(errMark, "Failed to add mark")
	}

	settings := d.Settings()

	outputFile, errOf := os.OpenFile(settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}

	defer util.LogClose(d.log, outputFile)

	if errExport := d.rules.ExportPlayers(rules.LocalRuleName, outputFile); errExport != nil {
		d.log.Error("Failed to save updated player list", zap.Error(errExport))
	}

	go d.updateState(newMarkEvent(sid64, attrs, true))

	return nil
}

func (d *Detector) Whitelist(ctx context.Context, sid64 steamid.SID64, enabled bool) error {
	player, errPlayer := d.GetPlayerOrCreate(ctx, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	player.Whitelisted = enabled
	player.Touch()

	if errSave := d.dataStore.SavePlayer(ctx, &player); errSave != nil {
		return errors.Wrap(errSave, "Failed to save player")
	}

	d.players.update(player)

	if enabled {
		d.rules.WhitelistAdd(sid64)
	} else {
		d.rules.WhitelistRemove(sid64)
	}

	go d.updateState(newWhitelistEvent(player.SteamID, enabled))

	d.log.Info("Update player whitelist status successfully",
		zap.String("steam_id", player.SteamID.String()), zap.Bool("enabled", enabled))

	return nil
}

func (d *Detector) updatePlayerState(ctx context.Context) error {
	if !d.ready(ctx) {
		return ErrInvalidReadyState
	}

	// Sent to client, response via log output
	_, errStatus := d.rconMulti("status")
	if errStatus != nil {
		return errors.Wrap(errStatus, "Failed to get status results")
	}

	dumpPlayer, errDumpPlayer := d.rconMulti("g15_dumpplayer")
	if errDumpPlayer != nil {
		return errors.Wrap(errDumpPlayer, "Failed to get g15_dumpplayer results")
	}

	var dump g15.DumpPlayer
	if errG15 := d.g15.Parse(bytes.NewBufferString(dumpPlayer), &dump); errG15 != nil {
		return errors.Wrap(errG15, "Failed to parse g15_dumpplayer results")
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
		player.Team = store.Team(dump.Team[index])
		player.Alive = dump.Alive[index]
		player.Health = dump.Health[index]
		player.Valid = dump.Valid[index]
		player.UserID = dump.UserID[index]
		player.UpdatedOn = time.Now()

		d.players.update(player)
	}

	return nil
}

func (d *Detector) statusUpdater(ctx context.Context) {
	defer d.log.Debug("status updater exited")

	statusTimer := time.NewTicker(DurationStatusUpdateTimer)

	for {
		select {
		case <-statusTimer.C:
			if errUpdate := d.updatePlayerState(ctx); errUpdate != nil {
				d.log.Debug("Failed to query state", zap.Error(errUpdate))

				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Detector) GetPlayerOrCreate(ctx context.Context, sid64 steamid.SID64) (store.Player, error) {
	player, errPlayer := d.players.bySteamID(sid64)
	if errPlayer == nil {
		return player, nil
	}

	player = store.NewPlayer(sid64, "")

	if errGet := d.dataStore.GetPlayer(ctx, sid64, true, &player); errGet != nil {
		if !errors.Is(errGet, sql.ErrNoRows) {
			return player, errors.Wrap(errGet, "Failed to fetch player record")
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
	defer d.log.Debug("checkHandler exited")

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

func (d *Detector) cleanupHandler(ctx context.Context) {
	const disconnectMsg = "Disconnected"

	log := d.log.Named("cleanupHandler")
	defer log.Debug("cleanupHandler exited")

	deleteTimer := time.NewTicker(time.Second * time.Duration(d.Settings().PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			settings := d.Settings()

			log.Debug("Delete update input received", zap.String("state", "start"))
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
					log.Debug("Flushing expired player", zap.Int64("steam_id", player.SteamID.Int64()))
				}
			}

			log.Debug("Delete update input received", zap.String("state", "end"))

			deleteTimer.Reset(time.Second * time.Duration(settings.PlayerExpiredTimeout))
		}
	}
}

func (d *Detector) AddUserName(ctx context.Context, player *store.Player, name string) error {
	unh, errMessage := store.NewUserNameHistory(player.SteamID, name)
	if errMessage != nil {
		return errors.Wrap(errMessage, "Failed to load messages")
	}

	if errSave := d.dataStore.SaveUserNameHistory(ctx, unh); errSave != nil {
		return errors.Wrap(errSave, "Failed to save username history")
	}

	if match := d.rules.MatchName(unh.Name); match != nil {
		d.triggerMatch(ctx, player, match)
	}

	return nil
}

func (d *Detector) AddUserMessage(ctx context.Context, player *store.Player, message string, dead bool, teamOnly bool) error {
	userMessage, errMessage := store.NewUserMessage(player.SteamID, message, dead, teamOnly)
	if errMessage != nil {
		return errors.Wrap(errMessage, "Failed to create new message")
	}

	if errSave := d.dataStore.SaveMessage(ctx, userMessage); errSave != nil {
		return errors.Wrap(errSave, "Failed to save user message")
	}

	if match := d.rules.MatchMessage(userMessage.Message); match != nil {
		d.triggerMatch(ctx, player, match)
	}

	return nil
}

func (d *Detector) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.log, d.settings.Lists)
	for _, list := range playerLists {
		boundList := list

		count, errImport := d.rules.ImportPlayers(&boundList)
		if errImport != nil {
			d.log.Error("Failed to import player list", zap.String("name", boundList.FileInfo.Title), zap.Error(errImport))
		} else {
			d.log.Info("Imported player list", zap.String("name", boundList.FileInfo.Title), zap.Int("count", count))
		}
	}

	for _, list := range ruleLists {
		boundList := list

		count, errImport := d.rules.ImportRules(&boundList)
		if errImport != nil {
			d.log.Error("Failed to import rules list (%s): %v\n", zap.String("name", boundList.FileInfo.Title), zap.Error(errImport))
		} else {
			d.log.Info("Imported rules list", zap.String("name", boundList.FileInfo.Title), zap.Int("count", count))
		}
	}
}

func (d *Detector) checkPlayerStates(ctx context.Context, validTeam store.Team) {
	for _, curPlayer := range d.players.all() {
		player := curPlayer

		if player.IsDisconnected() {
			continue
		}

		if matchSteam := d.rules.MatchSteam(player.SteamID); matchSteam != nil { //nolint:nestif
			player.Matches = matchSteam

			if validTeam == player.Team {
				d.triggerMatch(ctx, &player, matchSteam)
			}
		} else if player.Name != "" {
			if matchName := d.rules.MatchName(player.Name); matchName != nil && validTeam == player.Team {
				player.Matches = matchName

				if validTeam == player.Team {
					d.triggerMatch(ctx, &player, matchSteam)
				}
			}
		}

		if player.Dirty {
			if errSave := d.dataStore.SavePlayer(ctx, &player); errSave != nil {
				d.log.Error("Failed to save dirty player state", zap.Error(errSave))

				continue
			}

			player.Dirty = false

			d.players.update(player)
		}
	}
}

func (d *Detector) triggerMatch(ctx context.Context, player *store.Player, matches []*rules.MatchResult) {
	announceGeneralLast := player.AnnouncedGeneralLast
	announcePartyLast := player.AnnouncedPartyLast

	if time.Since(announceGeneralLast) >= DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if player.Whitelisted {
			msg = "Matched whitelisted player"
		}

		for _, match := range matches {
			d.log.Info(msg, zap.String("match_type", match.MatcherType),
				zap.String("steam_id", player.SteamID.String()), zap.String("name", player.Name), zap.String("origin", match.Origin))
		}

		player.AnnouncedGeneralLast = time.Now()
	}

	if player.Whitelisted {
		return
	}

	if d.settings.PartyWarningsEnabled && time.Since(announcePartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := d.SendChat(ctx, ChatDestParty, "(%d) [%s] [%s] %s ", player.UserID, match.Origin, strings.Join(match.Attributes, ","), player.Name); errLog != nil {
				d.log.Error("Failed to send party log message", zap.Error(errLog))

				return
			}
		}

		player.AnnouncedPartyLast = time.Now()
	}

	if d.settings.KickerEnabled { //nolint:nestif
		kickTag := false

		for _, match := range matches {
			for _, tag := range match.Attributes {
				for _, allowedTag := range d.settings.KickTags {
					if strings.EqualFold(tag, allowedTag) {
						kickTag = true

						break
					}
				}
			}
		}

		if kickTag {
			if errVote := d.CallVote(ctx, player.UserID, KickReasonCheating); errVote != nil {
				d.log.Error("Error calling vote", zap.Error(errVote))
			}
		} else {
			d.log.Info("Skipping kick on matched player, no acceptable tag found")
		}
	}

	go d.updateState(newKickAttemptEvent(player.SteamID))
}

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
		return errors.Wrapf(errConn, "failed to connect to client: %v", errConn)
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
		d.log.Debug("RCON is not ready yet", zap.Error(errRcon))

		return false
	}

	return true
}

func (d *Detector) SendChat(ctx context.Context, destination ChatDest, format string, args ...any) error {
	if !d.ready(ctx) {
		return ErrInvalidReadyState
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
		return errors.Errorf("Invalid destination: %s", destination)
	}

	d.rconMu.Lock()

	_, errExec := d.rconConn.Exec(cmd)
	if errExec != nil {
		d.rconMu.Unlock()

		return errors.Wrap(errExec, "Failed to send rcon chat message")
	}

	d.rconMu.Unlock()

	return nil
}

func (d *Detector) CallVote(ctx context.Context, userID int, reason KickReason) error {
	if !d.ready(ctx) {
		return ErrInvalidReadyState
	}

	d.rconMu.Lock()

	_, errExec := d.rconConn.Exec(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
	if errExec != nil {
		d.rconMu.Unlock()

		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}

	d.rconMu.Unlock()

	return nil
}

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
				d.log.Error("Failed to get process run status", zap.Error(errRunningStatus))

				continue
			}

			if existingState != newState {
				d.gameProcessActive.Store(newState)
				d.log.Info("Game process state changed", zap.Bool("is_running", newState))
			}

			// Handle auto closing the app on game close if enabled
			if !d.gameHasStartedOnce.Load() || !d.Settings().AutoCloseOnGameExit {
				continue
			}

			if !newState {
				d.log.Info("Auto-closing on game exit", zap.Duration("uptime", time.Since(d.startupTime)))
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
		util.LogClose(d.log, d.rconConn)
	}

	d.rconMu.Unlock()

	if errCloseDB := d.dataStore.Close(); errCloseDB != nil {
		err = gerrors.Join(errCloseDB)
	}

	if d.Settings().DebugLogEnabled {
		err = gerrors.Join(d.log.Sync())
	}

	lCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if errWeb := d.Web.Shutdown(lCtx); errWeb != nil {
		err = gerrors.Join(errWeb)
	}

	return err
}

func (d *Detector) openApplicationPage() {
	appURL := fmt.Sprintf("http://%s", d.settings.HTTPListenAddr)
	if errOpen := d.platform.OpenURL(appURL); errOpen != nil {
		d.log.Error("Failed to open URL", zap.String("url", appURL), zap.Error(errOpen))
	}
}

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
			d.log.Error("Web start returned error", zap.Error(errWeb))
		}
	}()

	if running, errRunning := d.platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce.Load() && d.Settings().AutoLaunchGame {
			go d.LaunchGameAndWait()
		}
	}

	d.openApplicationPage()
}

func SteamIDStringList(collection steamid.Collection) string {
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
					d.log.Error("Failed to save updated player state", zap.String("sid", localPlayer.SteamID.String()), zap.Error(errSave))
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

			d.log.Info("Updated",
				zap.Int("sums", len(updateData.summaries)), zap.Int("bans", len(updateData.bans)),
				zap.Int("sourcebans", len(updateData.sourcebans)), zap.Int("fiends", len(updateData.friends)))

			queue = nil
		}
	}
}

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

func (d *Detector) fetchProfileUpdates(ctx context.Context, queued steamid.Collection) updatedRemoteData {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	var (
		updated   updatedRemoteData
		waitGroup = &sync.WaitGroup{}
	)

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

func (d *Detector) rconMulti(cmd string) (string, error) {
	d.rconMu.Lock()
	defer d.rconMu.Unlock()

	cmdID, errWrite := d.rconConn.Write(cmd)
	if errWrite != nil {
		return "", errors.Wrap(errWrite, "Failed to send rcon command")
	}

	var response string

	for {
		resp, respID, errRead := d.rconConn.Read()
		if errRead != nil {
			return "", errors.Wrap(errRead, "Failed to read rcon response")
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

	randPlayer := func(userId int) store.Player {
		team := store.Blu
		if userId%2 == 0 {
			team = store.Red
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

	var testPlayers []store.Player

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
			player.Team = store.Spec
		}

		testPlayers = append(testPlayers, player)
	}

	detector.players.replace(testPlayers)
}

func (d *Detector) stateUpdater(ctx context.Context) {
	log := d.log.Named("stateUpdater")

	defer log.Debug("stateUpdater exited")

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-d.stateUpdates:
			log.Debug("Game state update input received", zap.String("kind", update.kind.String()))

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
			}
		}
	}
}

func (d *Detector) onUpdateMessage(ctx context.Context, log *zap.Logger, evt messageEvent) {
	player, errPlayer := d.players.byName(evt.name)
	if errPlayer != nil {
		return
	}

	if errUm := d.AddUserMessage(ctx, &player, evt.message, evt.dead, evt.teamOnly); errUm != nil {
		log.Error("Failed to handle user message", zap.Error(errUm))
	}
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
		d.log.Error("Failed to get or create player", zap.Error(errPlayer))

		return
	}

	player.Ping = evt.ping
	player.UserID = evt.userID
	player.Name = evt.name
	player.Connected = evt.connected.Seconds()
	player.UpdatedOn = time.Now()

	d.players.update(player)

	d.log.Debug("Player status updated",
		zap.String("sid", steamID.String()),
		zap.Int("tags", evt.ping),
		zap.Int("uid", evt.userID),
		zap.String("name", evt.name),
		zap.Int("connected", int(evt.connected.Seconds())))
}

func (d *Detector) onTags(evt tagsEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.Tags = evt.tags
	d.server.LastUpdate = time.Now()

	d.log.Debug("Tags updated", zap.Strings("tags", evt.tags))
}

func (d *Detector) onHostname(evt hostnameEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.ServerName = evt.hostname
	d.server.LastUpdate = time.Now()

	d.log.Debug("Hostname changed", zap.String("hostname", evt.hostname))
}

func (d *Detector) onMapName(evt mapEvent) {
	d.serverMu.Lock()
	defer d.serverMu.Unlock()

	d.server.CurrentMap = evt.mapName

	d.log.Debug("Map changed", zap.String("map", evt.mapName))
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
	log := d.log.Named("LogEventHandler")
	defer log.Info("log event handler exited")

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
					log.Error("Failed to parse port: %v", zap.Error(errPort), zap.String("port", pcs[1]))

					continue
				}

				parsedIP := net.ParseIP(pcs[0])
				if parsedIP == nil {
					log.Error("Failed to parse ip", zap.String("ip", pcs[0]))

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
				d.onUpdateMessage(ctx, log, messageEvent{
					steamID:   evt.PlayerSID,
					name:      evt.Player,
					createdAt: evt.Timestamp,
					message:   evt.Message,
					teamOnly:  evt.TeamOnly,
					dead:      evt.Dead,
				})
			}
		}
	}
}

func (d *Detector) discordStateUpdater(ctx context.Context) {
	const discordAppID = "1076716221162082364"

	log := d.log.Named("discord")
	defer log.Debug("discordStateUpdater exited")

	timer := time.NewTicker(time.Second * 10)
	isRunning := false

	for {
		select {
		case <-timer.C:
			if !d.Settings().DiscordPresenceEnabled {
				if isRunning {
					// Logout of existing connection on settings change
					if errLogout := d.discordPresence.Logout(); errLogout != nil {
						log.Error("Failed to logout of discord client", zap.Error(errLogout))
					}

					isRunning = false
				}

				continue
			}

			if !isRunning {
				if errLogin := d.discordPresence.Login(discordAppID); errLogin != nil {
					log.Debug("Failed to login to discord", zap.Error(errLogin))

					continue
				}

				isRunning = true
			}

			if isRunning {
				d.serverMu.RLock()

				if errUpdate := discordUpdateActivity(d.discordPresence, len(d.players.all()),
					d.server, d.gameProcessActive.Load(), d.startupTime); errUpdate != nil {
					log.Error("Failed to update discord activity", zap.Error(errUpdate))

					isRunning = false
				}

				d.serverMu.RUnlock()
			}
		case <-ctx.Done():
			return
		}
	}
}
