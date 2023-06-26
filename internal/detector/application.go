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

	"github.com/leighmacdonald/steamweb/v2"

	"golang.org/x/exp/slog"

	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/bd/pkg/voiceban"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
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

	dataStore store.DataStore
	// triggerUpdate     chan any
	gameStateUpdate chan updateStateEvent
	cache           FsCache

	gameHasStartedOnce bool
}

func New(logger *slog.Logger, settings *UserSettings, db store.DataStore, versionInfo Version, cache FsCache, reader *LogReader, logChan chan string) *Detector {
	isRunning, _ := platform.IsGameRunning()
	newSettings, errSettings := NewSettings()
	if errSettings != nil {
		panic(errSettings)
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
	if settings.RunMode != ModeTest {
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
					count, errPlayerImport := rules.ImportPlayers(&localPlayersList)
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
					count, errRulesImport := rules.ImportRules(&localRules)
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

	return &Detector{
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
		dataStore:          db,
		gameStateUpdate:    make(chan updateStateEvent, 50),
		cache:              cache,
		gameHasStartedOnce: isRunning,
	}
}

func NewLogger(logFile string) (*slog.Logger, error) {
	var w slog.Handler
	if logFile != "" {
		if util.Exists(logFile) {
			if err := os.Remove(logFile); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}
		of, errOf := os.Create(logFile)
		if errOf != nil {
			return nil, errors.Wrap(errOf, "Failed to open log file")
		}
		w = slog.NewTextHandler(of)
	} else {
		w = slog.NewTextHandler(os.Stderr)
	}
	logger := slog.New(w)
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
	req, reqErr := http.NewRequestWithContext(localCtx, "GET", store.AvatarURL(hash), nil)
	if reqErr != nil {
		return nil, errors.Wrap(reqErr, "Failed to create avatar download request")
	}
	resp, respErr := httpClient.Do(req)
	if respErr != nil {
		return nil, errors.Wrapf(respErr, "Failed to download avatar: %s", hash)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Invalid response code downloading avatar: %d", resp.StatusCode)
	}
	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, errors.Wrap(bodyErr, "Failed to read avatar response body")
	}
	defer util.LogClose(d.log, resp.Body)

	if errSet := d.cache.Set(TypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
		return nil, errors.Wrap(errSet, "failed to set cached value")
	}

	return body, nil
}

func NewLogReader(logger *slog.Logger, logPath string, logChan chan string) (*LogReader, error) {
	return newLogReader(logger, logPath, logChan, true)
}

func (d *Detector) exportVoiceBans() error {
	bannedIds := rules.FindNewestEntries(200, d.settings.GetKickTags())
	if len(bannedIds) == 0 {
		return nil
	}
	vbPath := filepath.Join(d.settings.GetTF2Dir(), "voice_ban.dt")
	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, 0o755)
	if errOpen != nil {
		return errOpen
	}
	if errWrite := voiceban.Write(vbFile, bannedIds); errWrite != nil {
		return errWrite
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

	if errLaunch := platform.LaunchTF2(d.settings.GetTF2Dir(), args); errLaunch != nil {
		d.log.Error("Failed to launch game", "err", errLaunch)
	}
}

// Players creates and returns a copy of the current player states
func (d *Detector) Players() []store.Player {
	var p []store.Player
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	for _, plr := range d.players {
		p = append(p, *plr)
	}
	return p
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
	if !rules.Unmark(sid64) {
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
	if errMark := rules.Mark(rules.MarkOpts{
		SteamID:    sid64,
		Attributes: attrs,
		Name:       name,
	}); errMark != nil {
		return errors.Wrap(errMark, "Failed to add mark")
	}

	of, errOf := os.OpenFile(d.settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}
	defer util.LogClose(d.log, of)
	if errExport := rules.ExportPlayers(rules.LocalRuleName, of); errExport != nil {
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
		return errSave
	}
	if enabled {
		rules.WhitelistAdd(sid64)
	} else {
		rules.WhitelistRemove(sid64)
	}
	d.log.Info("Update player whitelist status successfully",
		"steam_id", player.SteamID.Int64(), "enabled", enabled)
	return nil
}

func (d *Detector) updatePlayerState() (string, error) {
	if !d.ready() {
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
			lobbyStatus, errUpdate := d.updatePlayerState()
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
	mu := sync.RWMutex{}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		bans, errBans := steamweb.GetPlayerBans(ctx, steamid.Collection{sid64})
		if errBans != nil || len(bans) == 0 {
			d.log.Error("Failed to fetch player bans", "err", errBans)
		} else {
			mu.Lock()
			defer mu.Unlock()
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
		defer wg.Done()
		summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{sid64})
		if errSummaries != nil || len(summaries) == 0 {
			d.log.Error("Failed to fetch player summary", "err", errSummaries)
		} else {
			mu.Lock()
			defer mu.Unlock()
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
	wg.Wait()

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

//func getPlayerByName(name string) *store.Player {
//	playersMu.RLock()
//	defer playersMu.RUnlock()
//	for _, player := range players {
//		if player.Name == name {
//			return player
//		}
//	}
//	return nil
//}

func (d *Detector) checkHandler(ctx context.Context) {
	defer d.log.Debug("checkHandler exited")
	checkTimer := time.NewTicker(DurationCheckTimer)
	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTimer.C:
			p := d.GetPlayer(d.settings.GetSteamID())
			if p == nil {
				// We have not connected yet.
				continue
			}
			d.checkPlayerStates(ctx, p.Team)
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
			var valid store.PlayerCollection
			expired := 0
			for _, ps := range d.players {
				if ps.IsExpired() {
					if errSave := d.dataStore.SavePlayer(ctx, ps); errSave != nil {
						log.Error("Failed to save expired player state", "err", errSave)
					}
					expired++
				} else {
					valid = append(valid, ps)
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
		var sourcePlayer *store.Player
		var errSource error
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
			evt := update.data.(messageEvent)
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
			d.onUpdateBans(update.source, update.data.(steamweb.PlayerBanState))
		case updateStatus:
			if errUpdate := d.onUpdateStatus(ctx, update.source, update.data.(statusEvent)); errUpdate != nil {
				log.Error("updateStatus error", "err", errUpdate)
			}
		case updateLobby:
			d.onUpdateLobby(update.source, update.data.(lobbyEvent))
		case updateTags:
			d.onUpdateTags(update.data.(tagsEvent))
		case updateHostname:
			d.onUpdateHostname(update.data.(hostnameEvent))
		case updateMap:
			d.onUpdateMap(update.data.(mapEvent))
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
	return 0
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
		return errMessage
	}
	if errSave := d.dataStore.SaveUserNameHistory(ctx, unh); errSave != nil {
		return errSave
	}
	if match := rules.MatchName(unh.Name); match != nil {
		d.triggerMatch(player, match)
	}
	return nil
}

func (d *Detector) AddUserMessage(ctx context.Context, player *store.Player, message string, dead bool, teamOnly bool) error {
	um, errMessage := store.NewUserMessage(player.SteamID, message, dead, teamOnly)
	if errMessage != nil {
		return errMessage
	}
	if errSave := d.dataStore.SaveMessage(ctx, um); errSave != nil {
		return errSave
	}
	if match := rules.MatchMessage(um.Message); match != nil {
		d.triggerMatch(player, match)
	}
	return nil
}

func (d *Detector) onUpdateKill(kill killEvent) {
	source := d.nameToSid(d.players, kill.sourceName)
	target := d.nameToSid(d.players, kill.victimName)
	if !source.Valid() || !target.Valid() {
		return
	}
	ourSid := d.settings.GetSteamID()
	sourcePlayer := d.GetPlayer(source)
	targetPlayer := d.GetPlayer(target)
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
	player.UserId = update.userID
	player.Name = update.name
	player.Connected = update.connected.Seconds()
	player.UpdatedOn = time.Now()
	d.playersMu.Unlock()
	return nil
}

func (d *Detector) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.log, d.settings.GetLists())
	for _, list := range playerLists {
		count, errImport := rules.ImportPlayers(&list)
		if errImport != nil {
			d.log.Error("Failed to import player list", "name", list.FileInfo.Title, "err", errImport)
		} else {
			d.log.Info("Imported player list", "name", list.FileInfo.Title, "count", count)
		}
	}
	for _, list := range ruleLists {
		count, errImport := rules.ImportRules(&list)
		if errImport != nil {
			d.log.Error("Failed to import rules list (%s): %v\n", "name", list.FileInfo.Title, "err", errImport)
		} else {
			d.log.Info("Imported rules list", "name", list.FileInfo.Title, "count", count)
		}
	}
}

func (d *Detector) checkPlayerStates(ctx context.Context, validTeam store.Team) {
	for _, ps := range d.players {
		if ps.IsDisconnected() {
			continue
		}

		if matchSteam := rules.MatchSteam(ps.GetSteamID()); matchSteam != nil {
			ps.Matches = append(ps.Matches, matchSteam...)
			if validTeam == ps.Team {
				d.triggerMatch(ps, matchSteam)
			}
		} else if ps.Name != "" {
			if matchName := rules.MatchName(ps.GetName()); matchName != nil && validTeam == ps.Team {
				ps.Matches = append(ps.Matches, matchSteam...)
				if validTeam == ps.Team {
					d.triggerMatch(ps, matchSteam)
				}
			}
		}
		if ps.Dirty {
			if errSave := d.dataStore.SavePlayer(ctx, ps); errSave != nil {
				d.log.Error("Failed to save dirty player state", "err", errSave)
				continue
			}
			ps.Dirty = false
		}
	}
}

func (d *Detector) triggerMatch(ps *store.Player, matches []*rules.MatchResult) {
	announceGeneralLast := ps.AnnouncedGeneralLast
	announcePartyLast := ps.AnnouncedPartyLast
	if time.Since(announceGeneralLast) >= DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if ps.Whitelisted {
			msg = "Matched whitelisted player"
		}
		for _, match := range matches {
			d.log.Info(msg, "match_type", match.MatcherType,
				"steam_id", ps.SteamID.Int64(), "name", ps.Name, "origin", match.Origin)
		}
		ps.AnnouncedGeneralLast = time.Now()
	}
	if ps.Whitelisted {
		return
	}
	if d.settings.GetPartyWarningsEnabled() && time.Since(announcePartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := d.SendChat(ChatDestParty, "(%d) [%s] [%s] %s ", ps.UserId, match.Origin, strings.Join(match.Attributes, ","), ps.Name); errLog != nil {
				d.log.Error("Failed to send party log message", "err", errLog)
				return
			}
		}
		ps.AnnouncedPartyLast = time.Now()
	}
	if d.settings.GetKickerEnabled() {
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
			if errVote := d.CallVote(ps.UserId, KickReasonCheating); errVote != nil {
				d.log.Error("Error calling vote", "err", errVote)
			}
		} else {
			d.log.Info("Skipping kick, no acceptable tag found")
		}
	}
	ps.KickAttemptCount++
}

func (d *Detector) ensureRcon() error {
	if d.rconConn != nil {
		return nil
	}
	rconConfig := d.settings.GetRcon()
	conn, errConn := rcon.Dial(context.TODO(), rconConfig.String(), rconConfig.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}
	d.rconConn = conn
	return nil
}

func (d *Detector) ready() bool {
	if !d.gameProcessActive {
		return false
	}
	if errRcon := d.ensureRcon(); errRcon != nil {
		d.log.Debug("RCON is not ready yet", "err", errRcon)
		return false
	}
	return true
}

func (d *Detector) SendChat(destination ChatDest, format string, args ...any) error {
	if !d.ready() {
		return ErrInvalidReadyState
	}
	cmd := ""
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

func (d *Detector) CallVote(userID int64, reason KickReason) error {
	if !d.ready() {
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
			newState, errRunningStatus := platform.IsGameRunning()
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

// Shutdown closes any open rcon connection and will flush any player list to disk
func (d *Detector) Shutdown() error {
	if d.reader != nil && d.reader.tail != nil {
		d.reader.tail.Cleanup()
	}
	var err error
	if d.rconConn != nil {
		util.LogClose(d.log, d.rconConn)
	}
	if errCloseDb := d.dataStore.Close(); errCloseDb != nil {
		err = gerrors.Join(errCloseDb)
	}

	// if d.settings.GetDebugLogEnabled() {
	//     err = gerrors.Join(d.log.Sync())
	// }

	// TODO Stop web stuff

	return err
}

func (d *Detector) Start(ctx context.Context) {
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
	if running, errRunning := platform.IsGameRunning(); errRunning == nil && !running {
		if !d.gameHasStartedOnce && d.settings.GetAutoLaunchGame() {
			go d.LaunchGameAndWait()
		}
	}
}
