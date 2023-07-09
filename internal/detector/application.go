package detector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	gerrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	"golang.org/x/exp/slog"
)

var ErrInvalidReadyState = errors.New("Invalid ready state")

const (
	profileAgeLimit = time.Hour * 24
)

type Detector struct {
	log               *slog.Logger
	players           store.PlayerCollection
	playersMu         *sync.RWMutex
	logChan           chan string
	eventChan         chan LogEvent
	gameProcessActive bool
	startupTime       time.Time
	server            Server
	serverMu          *sync.RWMutex
	reader            *LogReader
	parser            *LogParser
	rconConn          rconConnection
	settings          *UserSettings
	discordPresence   *client.Client
	rules             *rules.Engine
	tr                *tr.Translator
	Web               *Web
	dataStore         store.DataStore
	// triggerUpdate     chan any
	gameStateUpdate    chan updateStateEvent
	cache              Cache
	systray            *Systray
	platform           platform.Platform
	gameHasStartedOnce bool
}

func New(logger *slog.Logger, settings *UserSettings, database store.DataStore, versionInfo Version, cache Cache, reader *LogReader, logChan chan string) *Detector {
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

	logger.Info("bd starting", "Version", versionInfo.Version)

	// tr, errTranslator := tr.NewTranslator()
	// if errTranslator != nil {
	// 	rootLogger.Error("Failed to load translations", "err", errTranslator)
	// }

	if settings.GetAPIKey() != "" {
		if errAPIKey := steamweb.SetKey(settings.GetAPIKey()); errAPIKey != nil {
			logger.Error("Failed to set steam api key", "err", errAPIKey)
		}
	}

	rulesEngine := rules.New()

	if settings.RunMode != ModeTest { //nolint:nestif
		// Try and load our existing custom players/rules
		if util.Exists(settings.LocalPlayerListPath()) {
			input, errInput := os.Open(settings.LocalPlayerListPath())
			if errInput != nil {
				logger.Error("Failed to open local player list", "err", errInput)
			} else {
				var localPlayersList rules.PlayerListSchema
				if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
					logger.Error("Failed to parse local player list", "err", errRead)
				} else {
					count, errPlayerImport := rulesEngine.ImportPlayers(&localPlayersList)
					if errPlayerImport != nil {
						logger.Error("Failed to import local player list", "err", errPlayerImport)
					} else {
						logger.Info("Loaded local player list", "count", count)
					}
				}
				util.LogClose(logger, input)
			}
		}

		if util.Exists(settings.LocalRulesListPath()) {
			input, errInput := os.Open(settings.LocalRulesListPath())
			if errInput != nil {
				logger.Error("Failed to open local rules list", "err", errInput)
			} else {
				var localRules rules.RuleSchema
				if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
					logger.Error("Failed to parse local rules list", "err", errRead)
				} else {
					count, errRulesImport := rulesEngine.ImportRules(&localRules)
					if errRulesImport != nil {
						logger.Error("Failed to import local rules list", "err", errRulesImport)
					}
					logger.Debug("Loaded local rules list", "count", count)
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
		gameProcessActive:  isRunning,
		startupTime:        time.Now(),
		server:             Server{},
		serverMu:           &sync.RWMutex{},
		reader:             reader,
		parser:             parser,
		rconConn:           nil,
		settings:           newSettings,
		dataStore:          database,
		gameStateUpdate:    make(chan updateStateEvent, 50),
		cache:              cache,
		gameHasStartedOnce: isRunning,
		discordPresence:    client.New(),
		rules:              rulesEngine,
		tr:                 translator,
		systray:            NewSystray(plat.Icon()),
		platform:           plat,
	}

	web, errWeb := NewWeb(application)
	if errWeb != nil {
		panic(errWeb)
	}

	application.Web = web

	return application
}

func NewLogger(logFile string) (*slog.Logger, error) {
	var handler slog.Handler

	if logFile != "" { //nolint:nestif
		if util.Exists(logFile) {
			if err := os.Remove(logFile); err != nil {
				return nil, errors.Wrap(err, "Failed to remove log file")
			}
		}

		of, errOf := os.Create(logFile)
		if errOf != nil {
			return nil, errors.Wrap(errOf, "Failed to open log file")
		}

		handler = slog.NewTextHandler(of, nil)
	} else {
		handler = slog.NewTextHandler(os.Stderr, nil)
	}

	logger := slog.New(handler)

	return logger.WithGroup("bd"), nil
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

func NewLogReader(logger *slog.Logger, logPath string, logChan chan string) (*LogReader, error) {
	return newLogReader(logger, logPath, logChan, true)
}

func (d *Detector) exportVoiceBans() error {
	bannedIds := d.rules.FindNewestEntries(200, d.settings.GetKickTags())
	if len(bannedIds) == 0 {
		return nil
	}

	vbPath := filepath.Join(d.settings.GetTF2Dir(), "voice_ban.dt")

	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, 0o755)
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
		d.gameProcessActive = false
		d.rconConn = nil
	}()

	if errInstall := addons.Install(d.settings.GetTF2Dir()); errInstall != nil {
		d.log.Error("Error trying to install addon", "err", errInstall)
	}

	if d.settings.GetVoiceBansEnabled() {
		if errVB := d.exportVoiceBans(); errVB != nil {
			d.log.Error("Failed to export voiceban list", "err", errVB)
		}
	}

	rconConfig := d.settings.GetRcon()
	args, errArgs := getLaunchArgs(
		rconConfig.Password(),
		rconConfig.Port(),
		d.settings.GetSteamDir(),
		d.settings.GetSteamID())

	if errArgs != nil {
		d.log.Error("Failed to get TF2 launch args", "err", errArgs)

		return
	}

	d.gameHasStartedOnce = true

	if errLaunch := d.platform.LaunchTF2(d.settings.GetTF2Dir(), args); errLaunch != nil {
		d.log.Error("Failed to launch game", "err", errLaunch)
	}
}

// Players creates and returns a copy of the current player states.
func (d *Detector) Players() []store.Player {
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	players := make([]store.Player, len(d.players))
	for index, plr := range d.players {
		players[index] = *plr
	}

	return players
}

func (d *Detector) AddPlayer(p *store.Player) {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	d.players = append(d.players, p)
}

func (d *Detector) UnMark(ctx context.Context, sid64 steamid.SID64) error {
	_, errPlayer := d.GetPlayerOrCreate(ctx, sid64, false)
	if errPlayer != nil {
		return errPlayer
	}

	if !d.rules.Unmark(sid64) {
		return errors.New("Mark does not exist")
	}

	// Remove existing mark data
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	for idx := range d.players {
		if d.players[idx].SteamID == sid64 {
			var valid []*rules.MatchResult

			for _, m := range d.players[idx].Matches {
				if m.Origin == "local" {
					continue
				}

				valid = append(valid, m)
			}

			d.players[idx].Matches = valid

			break
		}
	}

	return nil
}

func (d *Detector) Mark(ctx context.Context, sid64 steamid.SID64, attrs []string) error {
	player, errPlayer := d.GetPlayerOrCreate(ctx, sid64, false)
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
	}); errMark != nil {
		return errors.Wrap(errMark, "Failed to add mark")
	}

	outputFile, errOf := os.OpenFile(d.settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}

	defer util.LogClose(d.log, outputFile)

	if errExport := d.rules.ExportPlayers(rules.LocalRuleName, outputFile); errExport != nil {
		d.log.Error("Failed to save updated player list", "err", errExport)
	}

	return nil
}

func (d *Detector) Whitelist(ctx context.Context, sid64 steamid.SID64, enabled bool) error {
	player, playerErr := d.GetPlayerOrCreate(ctx, sid64, false)
	if playerErr != nil {
		return playerErr
	}

	player.Whitelisted = enabled
	player.Touch()

	if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
		return errors.Wrap(errSave, "Failed to save player")
	}

	if enabled {
		d.rules.WhitelistAdd(sid64)
	} else {
		d.rules.WhitelistRemove(sid64)
	}

	d.log.Info("Update player whitelist status successfully",
		"steam_id", player.SteamID.Int64(), "enabled", enabled)

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
				d.log.Debug("Failed to query state", "err", errUpdate)

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

func (d *Detector) GetPlayerOrCreate(ctx context.Context, sid64 steamid.SID64, active bool) (*store.Player, error) {
	player := d.GetPlayer(sid64)
	if player == nil {
		player = store.NewPlayer(sid64, "")
		if errGet := d.dataStore.GetPlayer(ctx, sid64, true, player); errGet != nil {
			if !errors.Is(errGet, sql.ErrNoRows) {
				return nil, errors.Wrap(errGet, "Failed to fetch player record")
			}

			player.ProfileUpdatedOn = time.Now().AddDate(-1, 0, 0)
		}

		if active {
			d.playersMu.Lock()
			d.players = append(d.players, player)
			d.playersMu.Unlock()
		}
	}

	if time.Since(player.ProfileUpdatedOn) < profileAgeLimit && player.Name != "" {
		return player, nil
	}

	var (
		mutex     = sync.RWMutex{}
		waitGroup = &sync.WaitGroup{}
	)

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()

		bans, errBans := steamweb.GetPlayerBans(ctx, steamid.Collection{sid64})
		if errBans != nil || len(bans) == 0 {
			d.log.Error("Failed to fetch player bans", "err", errBans)
		} else {
			mutex.Lock()
			defer mutex.Unlock()

			ban := bans[0]
			player.NumberOfVACBans = ban.NumberOfVACBans
			player.NumberOfGameBans = ban.NumberOfGameBans
			player.CommunityBanned = ban.CommunityBanned

			if ban.DaysSinceLastBan > 0 {
				subTime := time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
				player.LastVACBanOn = &subTime
			}

			player.EconomyBan = ban.EconomyBan != "none"
			player.ProfileUpdatedOn = time.Now()
		}
	}()

	go func() {
		defer waitGroup.Done()

		summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{sid64})
		if errSummaries != nil || len(summaries) == 0 {
			d.log.Error("Failed to fetch player summary", "err", errSummaries)
		} else {
			mutex.Lock()
			defer mutex.Unlock()

			summary := summaries[0]
			player.Visibility = store.ProfileVisibility(summary.CommunityVisibilityState)
			if player.AvatarHash != summary.Avatar {
				go d.performAvatarDownload(ctx, summary.AvatarHash)
			}

			player.Name = summary.PersonaName
			player.AvatarHash = summary.AvatarHash
			player.AccountCreatedOn = time.Unix(int64(summary.TimeCreated), 0)
			player.RealName = summary.RealName
			player.ProfileUpdatedOn = time.Now()
		}
	}()

	waitGroup.Wait()

	if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
		return nil, errors.Wrap(errSave, "Error trying to save player")
	}

	return player, nil
}

func (d *Detector) GetPlayer(sid64 steamid.SID64) *store.Player {
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	for _, player := range d.players {
		if player.SteamID == sid64 {
			return player
		}
	}

	return nil
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
			player := d.GetPlayer(d.settings.GetSteamID())
			if player == nil {
				// We have not connected yet.
				continue
			}

			d.checkPlayerStates(ctx, player.Team)
		}
	}
}

func (d *Detector) cleanupHandler(ctx context.Context) {
	log := d.log.WithGroup("cleanupHandler")
	defer log.Debug("cleanupHandler exited")

	deleteTimer := time.NewTicker(time.Second * time.Duration(d.settings.PlayerExpiredTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			log.Debug("Delete update input received", "state", "start")
			d.serverMu.Lock()
			if time.Since(d.server.LastUpdate) > time.Second*time.Duration(d.settings.PlayerDisconnectTimeout) {
				d.server = Server{}
			}
			d.serverMu.Unlock()

			var (
				valid   store.PlayerCollection
				expired = 0
			)

			for _, player := range d.players {
				if player.IsExpired() {
					if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
						log.Error("Failed to save expired player state", "err", errSave)
					}
					expired++
				} else {
					valid = append(valid, player)
				}
			}

			d.playersMu.Lock()
			d.players = valid
			d.playersMu.Unlock()

			if expired > 0 {
				log.Debug("Flushing expired players", "count", expired)
			}

			log.Debug("Delete update input received", "state", "end")
		}
	}
}

func (d *Detector) performAvatarDownload(ctx context.Context, hash string) {
	_, errDownload := d.fetchAvatar(ctx, hash)
	if errDownload != nil {
		d.log.Error("Failed to download avatar", "hash", hash, "err", errDownload)

		return
	}
}

func (d *Detector) gameStateUpdater(ctx context.Context) {
	log := d.log.WithGroup("gameStateUpdater")

	defer log.Debug("gameStateUpdater exited")

	for {
		update := <-d.gameStateUpdate

		log.Debug("Game state update input received", "kind", int(update.kind), "state", "start")

		var (
			sourcePlayer *store.Player
			errSource    error
		)

		if update.source.Valid() {
			sourcePlayer, errSource = d.GetPlayerOrCreate(ctx, update.source, true)
			if errSource != nil {
				log.Error("failed to get source player", "err", errSource)

				return
			}

			if sourcePlayer == nil && update.kind != updateStatus {
				// Only register a new user to track once we received a status line
				continue
			}
		}

		switch update.kind {
		case updateMessage:
			evt, ok := update.data.(messageEvent)
			if !ok {
				continue
			}

			if errUm := d.AddUserMessage(ctx, sourcePlayer, evt.message, evt.dead, evt.teamOnly); errUm != nil {
				log.Error("Failed to handle user message", "err", errUm)

				continue
			}
		case updateKill:
			e, ok := update.data.(killEvent)
			if ok {
				d.onUpdateKill(e)
			}
		case updateBans:
			evt, ok := update.data.(steamweb.PlayerBanState)
			if !ok {
				continue
			}

			d.onUpdateBans(update.source, evt)
		case updateStatus:
			evt, ok := update.data.(statusEvent)
			if !ok {
				continue
			}

			if errUpdate := d.onUpdateStatus(ctx, update.source, evt); errUpdate != nil {
				log.Error("updateStatus error", "err", errUpdate)
			}
		case updateLobby:
			evt, ok := update.data.(lobbyEvent)
			if !ok {
				continue
			}

			d.onUpdateLobby(update.source, evt)
		case updateTags:
			evt, ok := update.data.(tagsEvent)
			if !ok {
				continue
			}

			d.onUpdateTags(evt)
		case updateHostname:
			evt, ok := update.data.(hostnameEvent)
			if !ok {
				continue
			}

			d.onUpdateHostname(evt)
		case updateMap:
			evt, ok := update.data.(mapEvent)
			if !ok {
				continue
			}

			d.onUpdateMap(evt)
		case changeMap:
			d.onMapChange()
		}

		log.Debug("Game state update input", "kind", int(update.kind), "state", "end")
	}
}

func (d *Detector) onUpdateTags(event tagsEvent) {
	d.serverMu.Lock()
	d.server.Tags = event.tags
	d.server.LastUpdate = time.Now()
	d.serverMu.Unlock()
}

func (d *Detector) onUpdateMap(event mapEvent) {
	d.serverMu.Lock()
	d.server.CurrentMap = event.mapName
	d.serverMu.Unlock()
}

func (d *Detector) onUpdateHostname(event hostnameEvent) {
	d.serverMu.Lock()
	d.server.ServerName = event.hostname
	d.serverMu.Unlock()
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
	player := d.GetPlayer(steamID)
	if player != nil {
		d.playersMu.Lock()
		player.Team = evt.team
		d.playersMu.Unlock()
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

func (d *Detector) onUpdateKill(kill killEvent) {
	var (
		source = d.nameToSid(d.players, kill.sourceName)
		target = d.nameToSid(d.players, kill.victimName)
		ourSid = d.settings.GetSteamID()
	)

	if !source.Valid() || !target.Valid() {
		return
	}

	var (
		sourcePlayer = d.GetPlayer(source)
		targetPlayer = d.GetPlayer(target)
	)

	d.playersMu.Lock()
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
	d.playersMu.Unlock()
}

func (d *Detector) onMapChange() {
	d.playersMu.Lock()
	for _, player := range d.players {
		player.Kills = 0
		player.Deaths = 0
	}
	d.playersMu.Unlock()
	d.serverMu.Lock()
	d.server.CurrentMap = ""
	d.server.ServerName = ""
	d.serverMu.Unlock()
}

func (d *Detector) onUpdateBans(steamID steamid.SID64, ban steamweb.PlayerBanState) {
	player := d.GetPlayer(steamID)
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	player.NumberOfVACBans = ban.NumberOfVACBans
	player.NumberOfGameBans = ban.NumberOfGameBans
	player.CommunityBanned = ban.CommunityBanned

	if ban.DaysSinceLastBan > 0 {
		subTime := time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
		player.LastVACBanOn = &subTime
	}

	player.EconomyBan = ban.EconomyBan != "none"

	player.Touch()
}

func (d *Detector) onUpdateStatus(ctx context.Context, steamID steamid.SID64, update statusEvent) error {
	player, errPlayer := d.GetPlayerOrCreate(ctx, steamID, true)
	if errPlayer != nil {
		return errPlayer
	}

	d.playersMu.Lock()
	player.Ping = update.ping
	player.UserID = update.userID
	player.Name = update.name
	player.Connected = update.connected.Seconds()
	player.UpdatedOn = time.Now()
	d.playersMu.Unlock()

	return nil
}

func (d *Detector) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.log, d.settings.GetLists())
	for _, list := range playerLists {
		boundList := list

		count, errImport := d.rules.ImportPlayers(&boundList)
		if errImport != nil {
			d.log.Error("Failed to import player list", "name", boundList.FileInfo.Title, "err", errImport)
		} else {
			d.log.Info("Imported player list", "name", boundList.FileInfo.Title, "count", count)
		}
	}

	for _, list := range ruleLists {
		boundList := list

		count, errImport := d.rules.ImportRules(&boundList)
		if errImport != nil {
			d.log.Error("Failed to import rules list (%s): %v\n", "name", boundList.FileInfo.Title, "err", errImport)
		} else {
			d.log.Info("Imported rules list", "name", boundList.FileInfo.Title, "count", count)
		}
	}
}

func (d *Detector) checkPlayerStates(ctx context.Context, validTeam store.Team) {
	for _, player := range d.players {
		if player.IsDisconnected() {
			continue
		}

		if matchSteam := d.rules.MatchSteam(player.GetSteamID()); matchSteam != nil { //nolint:nestif
			player.Matches = append(player.Matches, matchSteam...)
			if validTeam == player.Team {
				d.triggerMatch(ctx, player, matchSteam)
			}
		} else if player.Name != "" {
			if matchName := d.rules.MatchName(player.GetName()); matchName != nil && validTeam == player.Team {
				player.Matches = append(player.Matches, matchSteam...)
				if validTeam == player.Team {
					d.triggerMatch(ctx, player, matchSteam)
				}
			}
		}

		if player.Dirty {
			if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
				d.log.Error("Failed to save dirty player state", "err", errSave)

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
			d.log.Info(msg, "match_type", match.MatcherType,
				"steam_id", player.SteamID.Int64(), "name", player.Name, "origin", match.Origin)
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
				d.log.Error("Failed to send party log message", "err", errLog)

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
				d.log.Error("Error calling vote", "err", errVote)
			}
		} else {
			d.log.Info("Skipping kick, no acceptable tag found")
		}
	}

	player.KickAttemptCount++
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
	if !d.gameProcessActive {
		return false
	}

	if errRcon := d.ensureRcon(ctx); errRcon != nil {
		d.log.Debug("RCON is not ready yet", "err", errRcon)

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

func (d *Detector) CallVote(ctx context.Context, userID int64, reason KickReason) error {
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
			existingState := d.gameProcessActive

			newState, errRunningStatus := d.platform.IsGameRunning()
			if errRunningStatus != nil {
				d.log.Error("Failed to get process run status", "err", errRunningStatus)

				continue
			}

			if existingState != newState {
				d.gameProcessActive = newState
				d.log.Info("Game process state changed", "is_running", newState)
			}

			// Handle auto closing the app on game close if enabled
			if !d.gameHasStartedOnce || !d.settings.GetAutoCloseOnGameExit() {
				continue
			}

			if !newState {
				d.log.Info("Auto-closing on game exit", "uptime", time.Since(d.startupTime))
				os.Exit(0)
			}
		}
	}
}

// Shutdown closes any open rcon connection and will flush any player list to disk.
func (d *Detector) Shutdown() error {
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

	// if d.settings.GetDebugLogEnabled() {
	//     err = gerrors.Join(d.log.Sync())
	// }

	// TODO Stop web stuff

	return err
}

func (d *Detector) Start(ctx context.Context) {
	go d.systray.start()
	go d.reader.start(ctx)
	go d.parser.start(ctx)
	go d.refreshLists(ctx)
	go d.incomingLogEventHandler(ctx)
	go d.gameStateUpdater(ctx)
	go d.cleanupHandler(ctx)
	go d.checkHandler(ctx)
	go d.statusUpdater(ctx)
	go d.processChecker(ctx)
	go d.discordStateUpdater(ctx)

	go func() {
		if errWeb := d.Web.startWeb(ctx); errWeb != nil {
			d.log.Error("Web start returned error")
		}
	}()

	if running, errRunning := d.platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce && d.settings.GetAutoLaunchGame() {
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
