package detector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	gerrors "errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/bd/pkg/discord/client"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/bd/pkg/voiceban"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ErrInvalidReadyState = errors.New("Invalid ready state")

const (
	profileAgeLimit = time.Hour * 24
)

type Detector struct {
	log                *zap.Logger
	players            store.PlayerCollection
	playersMu          *sync.RWMutex
	logChan            chan string
	eventChan          chan LogEvent
	gameProcessActive  atomic.Bool
	startupTime        time.Time
	server             *Server
	serverMu           *sync.RWMutex
	reader             *LogReader
	parser             *LogParser
	rconConn           rconConnection
	settings           *UserSettings
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
}

func New(logger *zap.Logger, settings *UserSettings, database store.DataStore, versionInfo Version, cache Cache,
	reader *LogReader, logChan chan string, dataSource DataSource,
) *Detector {
	plat := platform.New()
	isRunning, _ := plat.IsGameRunning()

	newSettings, errSettings := NewSettings()
	if errSettings != nil {
		panic(errSettings)
	}

	translator, errTrans := tr.NewTranslator()
	if errTrans != nil {
		panic(errTrans)
	}

	logger.Info("bd starting", zap.String("version", versionInfo.Version))

	// tr, errTranslator := tr.NewTranslator()
	// if errTranslator != nil {
	// 	rootLogger.Error("Failed to load translations", "err", errTranslator)
	// }

	if settings.GetAPIKey() != "" {
		if errAPIKey := steamweb.SetKey(settings.GetAPIKey()); errAPIKey != nil {
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
		players:            nil,
		playersMu:          &sync.RWMutex{},
		logChan:            logChan,
		eventChan:          eventChan,
		startupTime:        time.Now(),
		server:             &Server{},
		serverMu:           &sync.RWMutex{},
		reader:             reader,
		parser:             parser,
		rconConn:           nil,
		settings:           newSettings,
		dataStore:          database,
		stateUpdates:       make(chan updateStateEvent),
		cache:              cache,
		discordPresence:    client.New(),
		rules:              rulesEngine,
		tr:                 translator,
		platform:           plat,
		profileUpdateQueue: make(chan steamid.SID64),
		dataSource:         dataSource,
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

func MustCreateLogger(conf *UserSettings) *zap.Logger {
	var loggingConfig zap.Config

	switch conf.RunMode {
	case ModeProd:
		loggingConfig = zap.NewProductionConfig()
		loggingConfig.DisableCaller = true
	case ModeDebug:
		loggingConfig = zap.NewDevelopmentConfig()
		loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	case ModeTest:
		return zap.NewNop()
	default:
		panic(fmt.Sprintf("Unknown run mode: %s", conf.RunMode))
	}

	if conf.DebugLogEnabled {
		if util.Exists(conf.LogFilePath()) {
			if err := os.Remove(conf.LogFilePath()); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}

		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, conf.LogFilePath())
	}

	level, errLevel := zap.ParseAtomicLevel(conf.LogLevel)
	if errLevel != nil {
		panic(fmt.Sprintf("Failed to parse log level: %v", errLevel))
	}

	loggingConfig.Level.SetLevel(level.Level())

	l, errLogger := loggingConfig.Build()
	if errLogger != nil {
		panic("Failed to create log config")
	}

	return l.Named("bd")
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

func (d *Detector) Settings() *UserSettings {
	return d.settings
}

func (d *Detector) Rules() *rules.Engine {
	return d.rules
}

func (d *Detector) fetchAvatar(ctx context.Context, hash string) ([]byte, error) {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := d.cache.Get(TypeAvatar, hash, buf)

	if errCache == nil {
		return buf.Bytes(), nil
	}

	if errCache != nil && !errors.Is(errCache, ErrCacheExpired) {
		return nil, errors.Wrap(errCache, "unexpected cache error")
	}

	localCtx, cancel := context.WithTimeout(ctx, DurationWebRequestTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(localCtx, http.MethodGet, store.AvatarURL(hash), nil)
	if reqErr != nil {
		return nil, errors.Wrap(reqErr, "Failed to create avatar download request")
	}

	resp, respErr := httpClient.Do(req) //nolint:bodyclose
	if respErr != nil {
		return nil, errors.Wrapf(respErr, "Failed to download avatar: %s", hash)
	}

	defer util.LogClose(d.log, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Invalid response code downloading avatar: %d", resp.StatusCode)
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, errors.Wrap(bodyErr, "Failed to read avatar response body")
	}

	if errSet := d.cache.Set(TypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
		return nil, errors.Wrap(errSet, "failed to set cached value")
	}

	return body, nil
}

func NewLogReader(logger *zap.Logger, logPath string, logChan chan string) (*LogReader, error) {
	return newLogReader(logger, logPath, logChan, true)
}

const (
	maxVoiceBans   = 200
	voiceBansPerms = 0o755
)

func (d *Detector) exportVoiceBans() error {
	bannedIds := d.rules.FindNewestEntries(maxVoiceBans, d.settings.GetKickTags())
	if len(bannedIds) == 0 {
		return nil
	}

	vbPath := filepath.Join(d.settings.GetTF2Dir(), "voice_ban.dt")

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
		d.rconConn = nil
	}()

	if errInstall := addons.Install(d.settings.GetTF2Dir()); errInstall != nil {
		d.log.Error("Error trying to install addon", zap.Error(errInstall))
	}

	if d.settings.GetVoiceBansEnabled() {
		if errVB := d.exportVoiceBans(); errVB != nil {
			d.log.Error("Failed to export voiceban list", zap.Error(errVB))
		}
	}

	rconConfig := d.settings.GetRcon()
	args, errArgs := getLaunchArgs(
		rconConfig.Password(),
		rconConfig.Port(),
		d.settings.GetSteamDir(),
		d.settings.GetSteamID())

	if errArgs != nil {
		d.log.Error("Failed to get TF2 launch args", zap.Error(errArgs))

		return
	}

	d.gameHasStartedOnce.Store(true)

	if errLaunch := d.platform.LaunchTF2(d.settings.GetTF2Dir(), args); errLaunch != nil {
		d.log.Error("Failed to launch game", zap.Error(errLaunch))
	}
}

func (d *Detector) updateState(updates ...updateStateEvent) {
	for _, update := range updates {
		d.stateUpdates <- update
	}
}

func (d *Detector) UnMark(ctx context.Context, sid64 steamid.SID64) error {
	_, errPlayer := d.GetPlayerOrCreate(ctx, sid64)
	if errPlayer != nil {
		return errPlayer
	}

	if !d.rules.Unmark(sid64) {
		return errors.New("Mark does not exist")
	}

	// Remove existing mark data
	player, exists := d.GetPlayer(sid64)
	if !exists {
		return nil
	}

	var valid []*rules.MatchResult //nolint:prealloc

	d.playersMu.Lock()

	for _, m := range player.Matches {
		if m.Origin == "local" {
			continue
		}

		valid = append(valid, m)
	}

	player.Matches = valid

	d.playersMu.Unlock()

	d.updateState(newMarkEvent(sid64, nil, false))

	return nil
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

	outputFile, errOf := os.OpenFile(d.settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}

	defer util.LogClose(d.log, outputFile)

	if errExport := d.rules.ExportPlayers(rules.LocalRuleName, outputFile); errExport != nil {
		d.log.Error("Failed to save updated player list", zap.Error(errExport))
	}

	d.updateState(newMarkEvent(sid64, attrs, true))

	return nil
}

func (d *Detector) Whitelist(ctx context.Context, sid64 steamid.SID64, enabled bool) error {
	player, playerErr := d.GetPlayerOrCreate(ctx, sid64)
	if playerErr != nil {
		return playerErr
	}

	d.playersMu.Lock()

	player.Whitelisted = enabled
	player.Touch()

	if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
		return errors.Wrap(errSave, "Failed to save player")
	}
	d.playersMu.Unlock()

	if enabled {
		d.rules.WhitelistAdd(sid64)
	} else {
		d.rules.WhitelistRemove(sid64)
	}

	d.updateState(newWhitelistEvent(player.SteamID, enabled))

	d.log.Info("Update player whitelist status successfully",
		zap.String("steam_id", player.SteamID.String()), zap.Bool("enabled", enabled))

	return nil
}

func (d *Detector) updatePlayerState(ctx context.Context) (string, error) {
	if !d.ready(ctx) {
		return "", ErrInvalidReadyState
	}

	// Sent to client, response via log output
	_, errStatus := d.rconConn.Exec("status")
	if errStatus != nil {
		return "", errors.Wrap(errStatus, "Failed to get status results")
	}
	// TODO g15_dumpplayer
	// Sent to client, response via direct rcon response
	lobbyStatus, errDebug := d.rconConn.Exec("tf_lobby_debug")
	if errDebug != nil {
		return "", errors.Wrap(errDebug, "Failed to get debug results")
	}

	return lobbyStatus, nil
}

func (d *Detector) statusUpdater(ctx context.Context) {
	defer d.log.Debug("status updater exited")

	statusTimer := time.NewTicker(DurationStatusUpdateTimer)

	for {
		select {
		case <-statusTimer.C:
			lobbyStatus, errUpdate := d.updatePlayerState(ctx)
			if errUpdate != nil {
				d.log.Debug("Failed to query state", zap.Error(errUpdate))

				continue
			}

			for _, line := range strings.Split(lobbyStatus, "\n") {
				d.parser.ReadChannel <- line
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Detector) GetPlayerOrCreate(ctx context.Context, sid64 steamid.SID64) (*store.Player, error) {
	player, exists := d.GetPlayer(sid64)

	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	if !exists {
		if errGet := d.dataStore.GetPlayer(ctx, sid64, true, player); errGet != nil {
			if !errors.Is(errGet, sql.ErrNoRows) {
				return player, errors.Wrap(errGet, "Failed to fetch player record")
			}

			player.ProfileUpdatedOn = time.Now().AddDate(-1, 0, 0)
		}

		d.players = append(d.players, player)
	}

	if time.Since(player.ProfileUpdatedOn) < profileAgeLimit && player.Name != "" {
		return player, nil
	}

	go func() { d.profileUpdateQueue <- sid64 }()

	return player, nil
}

func (d *Detector) GetPlayer(sid64 steamid.SID64) (*store.Player, bool) {
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	for _, player := range d.players {
		if player.SteamID == sid64 {
			return player, true
		}
	}

	return store.NewPlayer(sid64, ""), false
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
			player, found := d.GetPlayer(d.settings.GetSteamID())
			if !found {
				// We have not connected yet.
				continue
			}

			d.checkPlayerStates(ctx, player.Team)
		}
	}
}

func (d *Detector) cleanupHandler(ctx context.Context) {
	log := d.log.Named("cleanupHandler")
	defer log.Debug("cleanupHandler exited")

	deleteTimer := time.NewTicker(time.Second * time.Duration(d.settings.PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			log.Debug("Delete update input received", zap.String("state", "start"))
			d.serverMu.Lock()
			if time.Since(d.server.LastUpdate) > time.Second*time.Duration(d.settings.PlayerDisconnectTimeout) {
				d.server = &Server{}
			}
			d.serverMu.Unlock()

			var expired steamid.Collection

			d.playersMu.RLock()

			for _, currentState := range d.players {
				player := currentState
				if player.IsExpired() {
					expired = append(expired, player.SteamID)
				}
			}

			d.playersMu.RUnlock()

			for _, exp := range expired {
				d.updateState(newPlayerTimeoutEvent(exp))
			}

			if len(expired) > 0 {
				log.Debug("Flushing expired players", zap.Int("count", len(expired)))
			}

			log.Debug("Delete update input received", zap.String("state", "end"))
		}
	}
}

func (d *Detector) performAvatarDownload(ctx context.Context, hash string) {
	_, errDownload := d.fetchAvatar(ctx, hash)
	if errDownload != nil {
		d.log.Error("Failed to download avatar", zap.String("hash", hash), zap.Error(errDownload))

		return
	}
}

func (d *Detector) nameToSid(players store.PlayerCollection, name string) steamid.SID64 {
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	for _, player := range players {
		if name == player.Name {
			return player.SteamID
		}
	}

	return ""
}

func (d *Detector) onUpdateLobby(steamID steamid.SID64, evt lobbyEvent) {
	d.updateState(newTeamEvent(steamID, evt.team))
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

func (d *Detector) onUpdateKill(kill killEvent) {
	var (
		source = d.nameToSid(d.players, kill.sourceName)
		target = d.nameToSid(d.players, kill.victimName)
		ourSid = d.settings.GetSteamID()
	)

	if !source.Valid() || !target.Valid() {
		return
	}

	sourcePlayer, inGameSource := d.GetPlayer(source)
	targetPlayer, inGameTarget := d.GetPlayer(target)

	if !inGameSource || !inGameTarget {
		return
	}

	sourcePlayer.Kills++
	targetPlayer.Deaths++

	if targetPlayer.SteamID == ourSid {
		sourcePlayer.DeathsBy++
	}

	if sourcePlayer.SteamID == ourSid {
		targetPlayer.KillsOn++
	}

	sourcePlayer.Touch()
	targetPlayer.Touch()
}

func (d *Detector) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.log, d.settings.GetLists())
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
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	for _, lPlayer := range d.players {
		player := lPlayer
		if player.IsDisconnected() {
			continue
		}

		if matchSteam := d.rules.MatchSteam(player.SteamID); matchSteam != nil { //nolint:nestif
			player.Matches = append(player.Matches, matchSteam...)
			if validTeam == player.Team {
				d.triggerMatch(ctx, player, matchSteam)
			}
		} else if player.Name != "" {
			if matchName := d.rules.MatchName(player.Name); matchName != nil && validTeam == player.Team {
				player.Matches = append(player.Matches, matchSteam...)
				if validTeam == player.Team {
					d.triggerMatch(ctx, player, matchSteam)
				}
			}
		}

		if player.Dirty {
			if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
				d.log.Error("Failed to save dirty player state", zap.Error(errSave))

				continue
			}

			player.Dirty = false
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

	if d.settings.GetPartyWarningsEnabled() && time.Since(announcePartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := d.SendChat(ctx, ChatDestParty, "(%d) [%s] [%s] %s ", player.UserID, match.Origin, strings.Join(match.Attributes, ","), player.Name); errLog != nil {
				d.log.Error("Failed to send party log message", zap.Error(errLog))

				return
			}
		}

		player.AnnouncedPartyLast = time.Now()
	}

	if d.settings.GetKickerEnabled() { //nolint:nestif
		kickTag := false

		for _, match := range matches {
			for _, tag := range match.Attributes {
				for _, allowedTag := range d.settings.GetKickTags() {
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
			d.log.Info("Skipping kick, no acceptable tag found")
		}
	}

	d.updateState(newKickAttemptEvent(player.SteamID))
}

func (d *Detector) ensureRcon(ctx context.Context) error {
	if d.rconConn != nil {
		return nil
	}

	rconConfig := d.settings.GetRcon()

	conn, errConn := rcon.Dial(ctx, rconConfig.String(), rconConfig.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}

	d.rconConn = conn

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

	_, errExec := d.rconConn.Exec(cmd)
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon chat message")
	}

	return nil
}

func (d *Detector) CallVote(ctx context.Context, userID int, reason KickReason) error {
	if !d.ready(ctx) {
		return ErrInvalidReadyState
	}

	_, errExec := d.rconConn.Exec(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}

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
			if !d.gameHasStartedOnce.Load() || !d.settings.GetAutoCloseOnGameExit() {
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

	if d.rconConn != nil {
		util.LogClose(d.log, d.rconConn)
	}

	if errCloseDB := d.dataStore.Close(); errCloseDB != nil {
		err = gerrors.Join(errCloseDB)
	}

	if d.settings.GetDebugLogEnabled() {
		err = gerrors.Join(d.log.Sync())
	}

	lCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if errWeb := d.Web.Shutdown(lCtx); errWeb != nil {
		err = gerrors.Join(errWeb)
	}

	return err
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

	go func() {
		if errWeb := d.Web.startWeb(ctx); errWeb != nil {
			d.log.Error("Web start returned error")
		}
	}()

	if running, errRunning := d.platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce.Load() && d.settings.GetAutoLaunchGame() {
			go d.LaunchGameAndWait()
		}
	}
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
		updateTimer = time.NewTicker(time.Second)
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
			updatedPlayers := d.applyRemoteData(updateData)

			for _, player := range updatedPlayers {
				d.playersMu.RLock()

				if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
					d.log.Error("Failed to save updated player state", zap.String("sid", player.SteamID.String()), zap.Error(errSave))
				}

				d.playersMu.RUnlock()
			}

			d.log.Info("Updated",
				zap.Int("sums", len(updateData.summaries)), zap.Int("bans", len(updateData.bans)),
				zap.Int("sourcebans", len(updateData.sourcebans)), zap.Int("fiends", len(updateData.friends)))

			queue = nil
		}
	}
}

func (d *Detector) applyRemoteData(data updatedRemoteData) []*store.Player {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	var updatedPlayers []*store.Player //nolint:prealloc

	for _, player := range d.players {
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
		updatedPlayers = append(updatedPlayers, player)
	}

	return updatedPlayers
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

// nolint:gosec
func CreateTestPlayers(detector *Detector, count int) store.PlayerCollection {
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

	randPlayer := func(userId int) *store.Player {
		team := store.Blu
		if userId%2 == 0 {
			team = store.Red
		}

		player, errP := detector.GetPlayerOrCreate(context.TODO(), knownIds[idIdx])
		if errP != nil {
			panic(errP)
		}

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

	var testPlayers store.PlayerCollection

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
	detector.playersMu.Lock()
	detector.players = testPlayers
	detector.playersMu.Unlock()

	return testPlayers
}
