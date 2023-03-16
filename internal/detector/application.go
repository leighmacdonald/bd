package detector

import (
	"bytes"
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/cache"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/bd/pkg/voiceban"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrInvalidReadyState = errors.New("Invalid ready state")

// BD is the main application container
type BD struct {
	// TODO
	// - estimate private steam account ages (find nearby non-private account)
	// - "unmark" players, overriding any lists that may match
	// - track rage quits
	// - install vote fail mod
	// - wipe map session stats k/d
	// - track k/d over entire session?
	// - track history of interactions with players
	// - colourise messages that trigger
	// - track stopwatch time-ish via 02/28/2023 - 23:40:21: Teams have been switched.
	ctx                context.Context // TODO detach from struct
	logChan            chan string
	incomingLogEvents  chan model.LogEvent
	server             model.Server
	serverMu           *sync.RWMutex
	players            model.PlayerCollection
	playersMu          *sync.RWMutex
	logReader          *logReader
	logParser          *logParser
	rules              *rules.Engine
	rconConnection     rconConnection
	settings           *model.Settings
	store              store.DataStore
	gui                model.UserInterface
	triggerUpdate      chan any
	gameStateUpdate    chan updateStateEvent
	cache              cache.FsCache
	startupTime        time.Time
	gameHasStartedOnce bool
	logger             *zap.Logger
	gameProcessActive  *atomic.Bool
}

// New allocates a new bot detector application instance
func New(ctx context.Context, logger *zap.Logger, settings *model.Settings, store store.DataStore, rules *rules.Engine, cache cache.FsCache) BD {
	logChan := make(chan string)
	eventChan := make(chan model.LogEvent)
	isRunning, _ := platform.IsGameRunning()
	rootApp := BD{
		ctx:                ctx,
		logger:             logger,
		store:              store,
		rules:              rules,
		settings:           settings,
		logChan:            logChan,
		incomingLogEvents:  eventChan,
		serverMu:           &sync.RWMutex{},
		players:            model.PlayerCollection{},
		playersMu:          &sync.RWMutex{},
		triggerUpdate:      make(chan any),
		gameStateUpdate:    make(chan updateStateEvent, 50),
		cache:              cache,
		logParser:          newLogParser(logger, logChan, eventChan),
		startupTime:        time.Now(),
		gameHasStartedOnce: isRunning,
		gameProcessActive:  &atomic.Bool{},
	}

	rootApp.gameProcessActive.Store(isRunning)

	rootApp.createLogReader()

	return rootApp
}

func (bd *BD) Settings() *model.Settings {
	return bd.settings
}

func (bd *BD) Store() store.DataStore {
	return bd.store
}

func (bd *BD) fetchAvatar(ctx context.Context, hash string) ([]byte, error) {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := bd.cache.Get(cache.TypeAvatar, hash, buf)
	if errCache == nil {
		return buf.Bytes(), nil
	}
	if errCache != nil && !errors.Is(errCache, cache.ErrCacheExpired) {
		return nil, errors.Wrap(errCache, "unexpected cache error")
	}
	localCtx, cancel := context.WithTimeout(ctx, model.DurationWebRequestTimeout)
	defer cancel()
	req, reqErr := http.NewRequestWithContext(localCtx, "GET", model.AvatarUrl(hash), nil)
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
	defer util.LogClose(bd.logger, resp.Body)

	if errSet := bd.cache.Set(cache.TypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
		return nil, errors.Wrap(errSet, "failed to set cached value")
	}

	return body, nil
}

func (bd *BD) createLogReader() {
	consoleLogPath := filepath.Join(bd.settings.GetTF2Dir(), "console.log")
	reader, errLogReader := newLogReader(bd.logger, consoleLogPath, bd.logChan, true)
	if errLogReader != nil {
		bd.logger.Panic("Failed to create log reader", zap.Error(errLogReader))
	}
	bd.logReader = reader
}

func (bd *BD) ExportVoiceBans() error {
	bannedIds := bd.rules.FindNewestEntries(200, bd.settings.GetKickTags())
	if len(bannedIds) == 0 {
		return nil
	}
	vbPath := filepath.Join(bd.settings.GetTF2Dir(), "voice_ban.dt")
	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, 0755)
	if errOpen != nil {
		return errOpen
	}
	if errWrite := voiceban.Write(vbFile, bannedIds); errWrite != nil {
		return errWrite
	}
	bd.logger.Info("Generated voice_ban.dt successfully")
	return nil
}

func (bd *BD) LaunchGameAndWait() {
	defer func() {
		bd.gameProcessActive.Store(false)
		bd.rconConnection = nil
	}()
	if errInstall := addons.Install(bd.settings.GetTF2Dir()); errInstall != nil {
		bd.logger.Error("Error trying to install addon", zap.Error(errInstall))
	}
	if bd.settings.GetVoiceBansEnabled() {
		if errVB := bd.ExportVoiceBans(); errVB != nil {
			bd.logger.Error("Failed to export voiceban list", zap.Error(errVB))
		}
	}
	rconConfig := bd.settings.GetRcon()
	args, errArgs := getLaunchArgs(
		bd.logger,
		rconConfig.Password(),
		rconConfig.Port(),
		bd.settings.GetSteamDir(),
		bd.settings.GetSteamId())
	if errArgs != nil {
		bd.logger.Error("Failed to get TF2 launch args", zap.Error(errArgs))
		return
	}
	bd.gameHasStartedOnce = true

	if errLaunch := platform.LaunchTF2(bd.logger, bd.settings.GetTF2Dir(), args); errLaunch != nil {
		bd.logger.Error("Failed to launch game", zap.Error(errLaunch))
	}
}

func (bd *BD) OnUnMark(sid64 steamid.SID64) error {
	bd.gameStateUpdate <- updateStateEvent{
		kind:   updateMark,
		source: bd.settings.GetSteamId(),
		data: updateMarkEvent{
			target: sid64,
			delete: true,
		},
	}
	return nil
}

func (bd *BD) OnMark(sid64 steamid.SID64, attrs []string) error {
	bd.gameStateUpdate <- updateStateEvent{
		kind:   updateMark,
		source: bd.settings.GetSteamId(),
		data: updateMarkEvent{
			target: sid64,
			attrs:  attrs,
		},
	}
	return nil
}

func (bd *BD) OnWhitelist(sid64 steamid.SID64, enabled bool) error {
	bd.gameStateUpdate <- updateStateEvent{
		kind:   updateWhitelist,
		source: bd.settings.GetSteamId(),
		data: updateWhitelistEvent{
			target:  sid64,
			enabled: enabled,
		},
	}
	return nil
}

func fetchSteamWebUpdates(updates steamid.Collection) ([]updateStateEvent, error) {
	var results []updateStateEvent
	summaries, errSummaries := steamweb.PlayerSummaries(updates)
	if errSummaries != nil {
		return nil, errors.Wrap(errSummaries, "Failed to fetch summaries: %v\n")
	}
	for _, sum := range summaries {
		sid, errSid := steamid.SID64FromString(sum.Steamid)
		if errSid != nil {
			return nil, errors.Wrap(errSid, "Invalid sid from steam api?")
		}
		results = append(results, updateStateEvent{
			kind:   updateProfile,
			source: sid,
			data:   sum,
		})
	}
	bans, errBans := steamweb.GetPlayerBans(updates)
	if errBans != nil {
		return nil, errors.Wrap(errBans, "Failed to fetch bans: %v\n")
	}
	for _, ban := range bans {
		sid, errSid := steamid.SID64FromString(ban.SteamID)
		if errSid != nil {
			return nil, errors.Wrap(errSummaries, "Invalid sid from api?: %v\n")
		}
		results = append(results, updateStateEvent{
			kind:   updateBans,
			source: sid,
			data:   ban,
		})
	}
	return results, nil
}

func (bd *BD) updatePlayerState() (string, error) {
	if !bd.ready() {
		return "", ErrInvalidReadyState
	}
	// Sent to client, response via log output
	_, errStatus := bd.rconConnection.Exec("status")
	if errStatus != nil {
		return "", errors.Wrap(errStatus, "Failed to get status results")

	}
	// Sent to client, response via direct rcon response
	lobbyStatus, errDebug := bd.rconConnection.Exec("tf_lobby_debug")
	if errDebug != nil {
		return "", errors.Wrap(errDebug, "Failed to get debug results")
	}
	return lobbyStatus, nil
}

func (bd *BD) statusUpdater(ctx context.Context) {
	defer bd.logger.Debug("status updater exited")
	statusTimer := time.NewTicker(model.DurationStatusUpdateTimer)
	for {
		select {
		case <-statusTimer.C:
			lobbyStatus, errUpdate := bd.updatePlayerState()
			if errUpdate != nil {
				bd.logger.Debug("Failed to query state", zap.Error(errUpdate))
				continue
			}
			for _, line := range strings.Split(lobbyStatus, "\n") {
				bd.logParser.ReadChannel <- line
			}
		case <-ctx.Done():
			return
		}
	}
}

func (bd *BD) GetPlayer(sid64 steamid.SID64) *model.Player {
	bd.playersMu.RLock()
	defer bd.playersMu.RUnlock()
	for _, player := range bd.players {
		if player.SteamId == sid64 {
			return player
		}
	}
	return nil
}

func (bd *BD) getPlayerByName(name string) *model.Player {
	bd.playersMu.RLock()
	defer bd.playersMu.RUnlock()
	for _, player := range bd.players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func (bd *BD) checkHandler(ctx context.Context) {
	defer bd.logger.Debug("checkHandler exited")
	checkTimer := time.NewTicker(model.DurationCheckTimer)
	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTimer.C:
			p := bd.GetPlayer(bd.settings.GetSteamId())
			if p == nil {
				// We have not connected yet.
				continue
			}
			bd.checkPlayerStates(ctx, p.Team)
		}
	}
}

func (bd *BD) cleanupHandler(ctx context.Context) {
	defer bd.logger.Debug("cleanupHandler exited")
	deleteTimer := time.NewTicker(model.DurationPlayerExpired)
	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			bd.logger.Debug("Delete update input received", zap.String("state", "start"))
			bd.serverMu.Lock()
			if time.Since(bd.server.LastUpdate) > model.DurationDisconnected {
				bd.server = model.Server{}
			}
			bd.serverMu.Unlock()
			var valid model.PlayerCollection
			expired := 0
			for _, ps := range bd.players {
				if ps.IsExpired() {
					if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
						bd.logger.Error("Failed to save expired player state", zap.Error(errSave))
					}
					expired++
				} else {
					valid = append(valid, ps)
				}
			}

			bd.playersMu.Lock()
			bd.players = valid
			bd.playersMu.Unlock()
			if expired > 0 {
				bd.logger.Debug("Flushing expired players", zap.Int("count", expired))
			}
			if bd.gui != nil {
				bd.gui.UpdatePlayerState(bd.players)
				bd.gui.UpdateServerState(bd.server)
			}
			bd.logger.Debug("Delete update input received", zap.String("state", "end"))
		}
	}
}

func (bd *BD) performAvatarDownload(ctx context.Context, sid64 steamid.SID64, hash string) {
	avatar, errDownload := bd.fetchAvatar(ctx, hash)
	if errDownload != nil {
		bd.logger.Error("Failed to download avatar", zap.String("hash", hash), zap.Error(errDownload))
		return
	}
	if bd.gui != nil {
		bd.gui.SetAvatar(sid64, avatar)
	}
}

func (bd *BD) gameStateUpdater(ctx context.Context) {
	defer bd.logger.Debug("gameStateUpdater exited")
	var queuedUpdates steamid.Collection
	updateTimer := time.NewTicker(model.DurationUpdateTimer)
	queueAvatars := make(chan steamid.SID64, 32)
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			if len(queuedUpdates) == 0 || bd.settings.GetAPIKey() == "" {
				continue
			}
			if len(queuedUpdates) > 100 {
				var trimmed steamid.Collection
				for i := len(queuedUpdates) - 1; len(trimmed) < 100; i-- {
					trimmed = append(trimmed, queuedUpdates[i])
				}
				queuedUpdates = trimmed
			}
			bd.logger.Info("Updating profiles", zap.Int("count", len(queuedUpdates)))
			results, errUpdates := fetchSteamWebUpdates(queuedUpdates)
			if errUpdates != nil {
				bd.logger.Error("Failed to fetch profiles from steam api", zap.Error(errUpdates))
				continue
			}
			for _, r := range results {
				select {
				case bd.gameStateUpdate <- r:
				default:
					bd.logger.Error("Game update channel full")
				}
			}
			queuedUpdates = nil
		case sid64 := <-queueAvatars:
			bd.logger.Debug("Avatar update input received")
			p := bd.GetPlayer(sid64)
			if p == nil || p.AvatarHash == "" {
				continue
			}
			go bd.performAvatarDownload(ctx, sid64, p.AvatarHash)
		case update := <-bd.gameStateUpdate:
			bd.logger.Debug("Game state update input received", zap.Int("kind", int(update.kind)), zap.String("state", "start"))
			var sourcePlayer *model.Player
			if update.source.Valid() {
				sourcePlayer = bd.GetPlayer(update.source)
				if sourcePlayer == nil && update.kind != updateStatus && update.kind != updateMark {
					// Only register a new user to track once we received a status line
					continue
				}
			}
			switch update.kind {
			case updateMessage:
				if errUm := bd.onUpdateMessage(ctx, update.data.(messageEvent), bd.store); errUm != nil {
					bd.logger.Error("Failed to handle user message", zap.Error(errUm))
					continue
				}
			case updateKill:
				e, ok := update.data.(killEvent)
				if ok {
					bd.onUpdateKill(e)
				}
			case updateBans:
				bd.onUpdateBans(update.source, update.data.(steamweb.PlayerBanState))
			case updateProfile:
				bd.onUpdateProfile(update.source, update.data.(steamweb.PlayerSummary))
				queueAvatars <- update.source
			case updateStatus:
				if errUpdate := bd.onUpdateStatus(ctx, bd.store, update.source, update.data.(statusEvent), &queuedUpdates); errUpdate != nil {
					bd.logger.Error("updateStatus error", zap.Error(errUpdate))
				}
			case updateMark:
				d := update.data.(updateMarkEvent)
				if errUpdate := bd.onUpdateMark(d); errUpdate != nil {
					bd.logger.Error("updateMark error", zap.Error(errUpdate))
				}
			case updateWhitelist:
				if errUpdate := bd.onUpdateWhitelist(update.data.(updateWhitelistEvent)); errUpdate != nil {
					bd.logger.Error("updateWhitelist error", zap.Error(errUpdate))
				}
			case updateLobby:
				bd.onUpdateLobby(update.source, update.data.(lobbyEvent))
			case updateTags:
				bd.onUpdateTags(update.data.(tagsEvent))
			case updateHostname:
				bd.onUpdateHostname(update.data.(hostnameEvent))
			case updateMap:
				bd.onUpdateMap(update.data.(mapEvent))
			case changeMap:
				bd.onMapChange()
			}
			bd.logger.Debug("Game state update input", zap.Int("kind", int(update.kind)), zap.String("state", "end"))
		}
	}
}

func (bd *BD) onUpdateTags(event tagsEvent) {
	bd.serverMu.Lock()
	bd.server.Tags = event.tags
	bd.server.LastUpdate = time.Now()
	bd.serverMu.Unlock()
	if bd.gui != nil {
		bd.serverMu.RLock()
		bd.gui.UpdateServerState(bd.server)
		bd.serverMu.RUnlock()
	}
}

func (bd *BD) onUpdateMap(event mapEvent) {
	bd.serverMu.Lock()
	bd.server.CurrentMap = event.mapName
	bd.serverMu.Unlock()
}

func (bd *BD) onUpdateHostname(event hostnameEvent) {
	bd.serverMu.Lock()
	bd.server.ServerName = event.hostname
	bd.serverMu.Unlock()
}

func (bd *BD) nameToSid(players model.PlayerCollection, name string) steamid.SID64 {
	bd.playersMu.RLock()
	defer bd.playersMu.RUnlock()
	for _, player := range players {
		if name == player.Name {
			return player.SteamId
		}
	}
	return 0
}

func (bd *BD) onUpdateLobby(steamID steamid.SID64, evt lobbyEvent) {
	player := bd.GetPlayer(steamID)
	if player != nil {
		bd.playersMu.Lock()
		player.Team = evt.team
		bd.playersMu.Unlock()
	}
}

func (bd *BD) onUpdateMessage(ctx context.Context, msg messageEvent, store store.DataStore) error {
	player := bd.getPlayerByName(msg.name)
	if player == nil {
		return errors.Errorf("Unknown name: %v", msg.name)
	}

	um := model.UserMessage{}
	bd.playersMu.RLock()
	um.Player = player.Name
	um.Team = player.Team
	um.PlayerSID = player.SteamId
	um.UserId = player.UserId
	bd.playersMu.RUnlock()
	um.Message = msg.message
	um.Created = msg.createdAt
	um.Dead = msg.dead
	um.TeamOnly = msg.teamOnly

	if errSaveMsg := store.SaveMessage(ctx, &um); errSaveMsg != nil {
		bd.logger.Error("Error trying to store user message log", zap.Error(errSaveMsg))
	}
	if match := bd.rules.MatchMessage(um.Message); match != nil {
		bd.triggerMatch(player, match)
	}
	if bd.gui != nil {
		bd.gui.AddUserMessage(um)
		bd.gui.Refresh()
	}
	return nil
}

func (bd *BD) onUpdateKill(kill killEvent) {
	source := bd.nameToSid(bd.players, kill.sourceName)
	target := bd.nameToSid(bd.players, kill.victimName)
	if !source.Valid() || !target.Valid() {
		return
	}
	ourSid := bd.settings.GetSteamId()
	sourcePlayer := bd.GetPlayer(source)
	targetPlayer := bd.GetPlayer(target)
	bd.playersMu.Lock()
	sourcePlayer.Kills++
	targetPlayer.Deaths++
	if targetPlayer.SteamId == ourSid {
		sourcePlayer.DeathsBy++
	}
	if sourcePlayer.SteamId == ourSid {
		targetPlayer.KillsOn++
	}
	sourcePlayer.Touch()
	targetPlayer.Touch()
	bd.playersMu.Unlock()
}

func (bd *BD) onMapChange() {
	bd.playersMu.Lock()
	for _, player := range bd.players {
		player.Kills = 0
		player.Deaths = 0
	}
	bd.playersMu.Unlock()
	bd.serverMu.Lock()
	bd.server.CurrentMap = ""
	bd.server.ServerName = ""
	bd.serverMu.Unlock()
}

func (bd *BD) onUpdateBans(steamID steamid.SID64, ban steamweb.PlayerBanState) {
	player := bd.GetPlayer(steamID)
	bd.playersMu.Lock()
	defer bd.playersMu.Unlock()
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

func (bd *BD) onUpdateProfile(steamID steamid.SID64, summary steamweb.PlayerSummary) {
	player := bd.GetPlayer(steamID)
	bd.playersMu.Lock()
	defer bd.playersMu.Unlock()
	player.Visibility = model.ProfileVisibility(summary.CommunityVisibilityState)
	player.AvatarHash = summary.AvatarHash
	player.AccountCreatedOn = time.Unix(int64(summary.TimeCreated), 0)
	player.RealName = summary.RealName
	player.ProfileUpdatedOn = time.Now()
	player.Touch()
}

func (bd *BD) onUpdateStatus(ctx context.Context, store store.DataStore, steamID steamid.SID64, update statusEvent, queuedUpdates *steamid.Collection) error {
	player := bd.GetPlayer(steamID)
	if player == nil {
		player = model.NewPlayer(steamID, update.name)
		if errCreate := store.LoadOrCreatePlayer(ctx, steamID, player); errCreate != nil {
			return errors.Wrap(errCreate, "Error trying to load/create player\n")
		}
		if update.name != "" && update.name != player.NamePrevious {
			if errSaveName := store.SaveName(ctx, steamID, player.Name); errSaveName != nil {
				return errors.Wrap(errSaveName, "Failed to save name")
			}
		}
		bd.playersMu.Lock()
		bd.players = append(bd.players, player)
		bd.playersMu.Unlock()
	}
	bd.playersMu.Lock()
	player.Ping = update.ping
	player.UserId = update.userID
	player.Name = update.name
	player.Connected = update.connected
	player.UpdatedOn = time.Now()
	if time.Since(player.ProfileUpdatedOn) > model.DurationCacheTimeout {
		*queuedUpdates = append(*queuedUpdates, steamID)
	}
	bd.playersMu.Unlock()
	return nil
}

func (bd *BD) onUpdateWhitelist(event updateWhitelistEvent) error {
	player := bd.GetPlayer(event.target)
	if player == nil {
		return errors.New("Unknown player, cannot update whitelist")
	}
	bd.playersMu.Lock()
	player.Whitelisted = event.enabled
	player.Touch()
	bd.playersMu.Unlock()
	bd.logger.Info("Update player whitelist status successfully",
		zap.Int64("steam_id", player.SteamId.Int64()), zap.Bool("enabled", event.enabled))
	return nil
}

func (bd *BD) onUpdateMark(status updateMarkEvent) error {
	player := bd.GetPlayer(status.target)
	if player == nil {
		player = model.NewPlayer(status.target, "")
		if err := bd.store.GetPlayer(context.Background(), status.target, player); err != nil {
			return err
		}
	}
	name := player.Name
	if name == "" {
		name = player.NamePrevious
	}
	if status.delete {
		bd.rules.Unmark(status.target)
		bd.playersMu.Lock()
		for idx := range bd.players {
			if bd.players[idx].SteamId == status.target {
				bd.players[idx].Match = nil
				break
			}
		}
		bd.playersMu.Unlock()
		bd.gui.UpdatePlayerState(bd.players)
	} else {
		if errMark := bd.rules.Mark(rules.MarkOpts{
			SteamID:    status.target,
			Attributes: status.attrs,
			Name:       name,
		}); errMark != nil {
			return errors.Wrap(errMark, "Failed to add mark")
		}
	}
	of, errOf := os.OpenFile(bd.settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}
	if errExport := bd.rules.ExportPlayers(rules.LocalRuleName, of); errExport != nil {
		bd.logger.Error("Failed to export player list", zap.Error(errExport))
	}
	util.LogClose(bd.logger, of)
	return nil
}

// AttachGui connects the backend functions to the frontend gui
// TODO Use channels for communicating instead
func (bd *BD) AttachGui(gui model.UserInterface) {
	gui.UpdateAttributes(bd.rules.UniqueTags())
	bd.gui = gui
}

func (bd *BD) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, bd.logger, bd.settings.GetLists())
	for _, list := range playerLists {
		count, errImport := bd.rules.ImportPlayers(&list)
		if errImport != nil {
			bd.logger.Error("Failed to import player list", zap.String("name", list.FileInfo.Title), zap.Error(errImport))
		} else {
			bd.logger.Info("Imported player list", zap.String("name", list.FileInfo.Title), zap.Int("count", count))
		}
	}
	for _, list := range ruleLists {
		count, errImport := bd.rules.ImportRules(&list)
		if errImport != nil {
			bd.logger.Error("Failed to import rules list (%s): %v\n", zap.String("name", list.FileInfo.Title), zap.Error(errImport))
		} else {
			bd.logger.Info("Imported rules list", zap.String("name", list.FileInfo.Title), zap.Int("count", count))
		}
	}
	// TODO move
	if bd.gui != nil {
		bd.gui.UpdateAttributes(bd.rules.UniqueTags())
	}
}

func (bd *BD) checkPlayerStates(ctx context.Context, validTeam model.Team) {
	for _, ps := range bd.players {
		if ps.IsDisconnected() {
			continue
		}
		if matchSteam := bd.rules.MatchSteam(ps.GetSteamID()); matchSteam != nil {
			ps.Match = matchSteam
			if validTeam == ps.Team {
				bd.triggerMatch(ps, matchSteam)
			}
		} else if ps.Name != "" {
			if matchName := bd.rules.MatchName(ps.GetName()); matchName != nil && validTeam == ps.Team {
				ps.Match = matchSteam
				if validTeam == ps.Team {
					bd.triggerMatch(ps, matchSteam)
				}
			}
		}
		if ps.Dirty {
			if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
				bd.logger.Error("Failed to save dirty player state", zap.Error(errSave))
				continue
			}
			ps.Dirty = false
		}
	}
	if bd.gui != nil {
		bd.gui.UpdatePlayerState(bd.players)
	}
}

func (bd *BD) triggerMatch(ps *model.Player, match *rules.MatchResult) {
	ps.Lock()
	defer ps.Unlock()
	announceGeneralLast := ps.AnnouncedGeneralLast
	announcePartyLast := ps.AnnouncedPartyLast
	if time.Since(announceGeneralLast) >= model.DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if ps.Whitelisted {
			msg = "Matched whitelisted player"
		}
		bd.logger.Info(msg, zap.String("match_type", match.MatcherType),
			zap.Int64("steam_id", ps.SteamId.Int64()), zap.String("name", ps.Name), zap.String("origin", match.Origin))
		ps.AnnouncedGeneralLast = time.Now()
	}
	if ps.Whitelisted {
		return
	}
	if bd.settings.GetPartyWarningsEnabled() && time.Since(announcePartyLast) >= model.DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		if errLog := bd.SendChat(model.ChatDestParty, "(%d) [%s] [%s] %s ", ps.UserId, match.Origin, strings.Join(match.Attributes, ","), ps.Name); errLog != nil {
			bd.logger.Error("Failed to send party log message", zap.Error(errLog))
			return
		}
		ps.AnnouncedPartyLast = time.Now()
	}
	if bd.settings.GetKickerEnabled() {
		kickTag := false
		for _, tag := range match.Attributes {
			for _, allowedTag := range bd.settings.GetKickTags() {
				if strings.EqualFold(tag, allowedTag) {
					kickTag = true
					break
				}
			}
		}
		if kickTag {
			if errVote := bd.CallVote(ps.UserId, model.KickReasonCheating); errVote != nil {
				bd.logger.Error("Error calling vote", zap.Error(errVote))
			}
		} else {
			bd.logger.Info("Skipping kick, no acceptable tag found")
		}
	}
	ps.KickAttemptCount++
}

func (bd *BD) ensureRcon() error {
	if bd.rconConnection != nil {
		return nil
	}
	rconConfig := bd.settings.GetRcon()
	conn, errConn := rcon.Dial(bd.ctx, rconConfig.String(), rconConfig.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}
	bd.rconConnection = conn
	return nil
}

func (bd *BD) ready() bool {
	if !bd.gameProcessActive.Load() {
		return false
	}
	if errRcon := bd.ensureRcon(); errRcon != nil {
		bd.logger.Debug("RCON is not ready yet", zap.Error(errRcon))
		return false
	}
	return true
}

func (bd *BD) SendChat(destination model.ChatDest, format string, args ...any) error {
	if !bd.ready() {
		return ErrInvalidReadyState
	}
	cmd := ""
	switch destination {
	case model.ChatDestAll:
		cmd = fmt.Sprintf("say %s", fmt.Sprintf(format, args...))
	case model.ChatDestTeam:
		cmd = fmt.Sprintf("say_team %s", fmt.Sprintf(format, args...))
	case model.ChatDestParty:
		cmd = fmt.Sprintf("say_party %s", fmt.Sprintf(format, args...))
	default:
		return errors.Errorf("Invalid destination: %s", destination)
	}
	_, errExec := bd.rconConnection.Exec(cmd)
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon chat message")
	}
	return nil
}

func (bd *BD) CallVote(userID int64, reason model.KickReason) error {
	if !bd.ready() {
		return ErrInvalidReadyState
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}
	return nil
}

func (bd *BD) processChecker(ctx context.Context) {
	ticker := time.NewTicker(model.DurationProcessTimeout)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			existingState := bd.gameProcessActive.Load()
			newState, errRunningStatus := platform.IsGameRunning()
			if errRunningStatus != nil {
				bd.logger.Error("Failed to get process run status", zap.Error(errRunningStatus))
				continue
			}
			if existingState != newState {
				bd.gameProcessActive.Store(newState)
				bd.logger.Info("Game process state changed", zap.Bool("is_running", newState))
			}
			// Handle auto closing the app on game close if enabled
			if !bd.gameHasStartedOnce || !bd.settings.GetAutoCloseOnGameExit() {
				continue
			}
			if !newState {
				bd.logger.Info("Auto-closing on game exit", zap.Duration("uptime", time.Since(bd.startupTime)))
				bd.gui.Quit()
			}
		}
	}
}

// Shutdown closes any open rcon connection and will flush any player list to disk
func (bd *BD) Shutdown() {
	if bd.rconConnection != nil {
		util.LogClose(bd.logger, bd.rconConnection)
	}
	util.LogClose(bd.logger, bd.store)
	bd.logger.Info("Goodbye")
}

func (bd *BD) Start(ctx context.Context) {
	go bd.logReader.start(ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(ctx)
	go bd.refreshLists(ctx)
	go bd.incomingLogEventHandler(ctx)
	go bd.gameStateUpdater(ctx)
	go bd.cleanupHandler(ctx)
	go bd.checkHandler(ctx)
	go bd.statusUpdater(ctx)
	go bd.processChecker(ctx)
	go bd.discordStateUpdater(ctx)
	if running, errRunning := platform.IsGameRunning(); errRunning == nil && !running {
		if !bd.gameHasStartedOnce && bd.settings.GetAutoLaunchGame() {
			go bd.LaunchGameAndWait()
		}
	}
	<-ctx.Done()
}
