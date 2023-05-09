package detector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
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

const (
	profileAgeLimit = time.Hour * 24
)

var (
	players           store.PlayerCollection
	playersMu         *sync.RWMutex
	logChan           chan string
	eventChan         chan LogEvent
	gameProcessActive *atomic.Bool
	startupTime       time.Time
	server            Server
	serverMu          *sync.RWMutex
	reader            *logReader
	parser            *logParser
	rconConn          rconConnection
	settings          *UserSettings

	dataStore store.DataStore
	//triggerUpdate     chan any
	gameStateUpdate chan updateStateEvent
	fsCache         FsCache

	gameHasStartedOnce *atomic.Bool
	rootLogger         *zap.Logger
)

func init() {
	startupTime = time.Now()
	serverMu = &sync.RWMutex{}
	isRunning, _ := platform.IsGameRunning()
	gameProcessActive = &atomic.Bool{}
	gameProcessActive.Store(isRunning)
	gameHasStartedOnce = &atomic.Bool{}
	gameHasStartedOnce.Store(isRunning)
	playersMu = &sync.RWMutex{}
	logChan = make(chan string)
	eventChan = make(chan LogEvent)
	serverMu = &sync.RWMutex{}
	//triggerUpdate = make(chan any)
	gameStateUpdate = make(chan updateStateEvent, 50)
	newSettings, errSettings := NewSettings()
	if errSettings != nil {
		panic(errSettings)
	}
	settings = newSettings
}

func MustCreateLogger(logFile string) *zap.Logger {
	loggingConfig := zap.NewProductionConfig()
	//loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if logFile != "" {
		if util.Exists(logFile) {
			if err := os.Remove(logFile); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, logFile)
	}
	logger, errLogger := loggingConfig.Build()
	if errLogger != nil {
		fmt.Printf("Failed to create logger: %v\n", errLogger)
		os.Exit(1)
	}
	return logger.Named("bd")
}

// Init allocates and configured the core application. Start must should be called after this.
// If testMode is enabled, a temporary database is used and no user settings or lists get used.
func Init(versionInfo Version, s *UserSettings, logger *zap.Logger, ds store.DataStore, testMode bool) {
	settings = s
	dataStore = ds
	rootLogger = logger
	rootLogger.Info("bd starting", zap.String("Version", versionInfo.Version))
	if errTranslations := tr.Init(); errTranslations != nil {
		rootLogger.Error("Failed to load translations", zap.Error(errTranslations))
	}
	if settings.GetAPIKey() != "" {
		if errAPIKey := steamweb.SetKey(settings.GetAPIKey()); errAPIKey != nil {
			rootLogger.Error("Failed to set steam api key", zap.Error(errAPIKey))
		}
	}
	if !testMode {
		// Try and load our existing custom players/rules
		if util.Exists(settings.LocalPlayerListPath()) {
			input, errInput := os.Open(settings.LocalPlayerListPath())
			if errInput != nil {
				rootLogger.Error("Failed to open local player list", zap.Error(errInput))
			} else {
				var localPlayersList rules.PlayerListSchema
				if errRead := json.NewDecoder(input).Decode(&localPlayersList); errRead != nil {
					rootLogger.Error("Failed to parse local player list", zap.Error(errRead))
				} else {
					count, errPlayerImport := rules.ImportPlayers(&localPlayersList)
					if errPlayerImport != nil {
						rootLogger.Error("Failed to import local player list", zap.Error(errPlayerImport))
					} else {
						rootLogger.Info("Loaded local player list", zap.Int("count", count))
					}
				}
				util.LogClose(rootLogger, input)
			}
		}

		if util.Exists(settings.LocalRulesListPath()) {
			input, errInput := os.Open(settings.LocalRulesListPath())
			if errInput != nil {
				rootLogger.Error("Failed to open local rules list", zap.Error(errInput))
			} else {
				var localRules rules.RuleSchema
				if errRead := json.NewDecoder(input).Decode(&localRules); errRead != nil {
					rootLogger.Error("Failed to parse local rules list", zap.Error(errRead))
				} else {
					count, errRulesImport := rules.ImportRules(&localRules)
					if errRulesImport != nil {
						rootLogger.Error("Failed to import local rules list", zap.Error(errRulesImport))
					}
					rootLogger.Debug("Loaded local rules list", zap.Int("count", count))
				}
				util.LogClose(rootLogger, input)
			}
		}
	}

	fsCache = newCache(rootLogger, settings.ConfigRoot(), DurationCacheTimeout)
	parser = newLogParser(rootLogger, logChan, eventChan)
	lr, errLogReader := createLogReader()
	if errLogReader != nil {
		rootLogger.Panic("Failed to create logreader", zap.Error(errLogReader))
	}
	reader = lr
}

//// BD is the main application container
//type BD struct {
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
//}

func fetchAvatar(ctx context.Context, hash string) ([]byte, error) {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := fsCache.Get(TypeAvatar, hash, buf)
	if errCache == nil {
		return buf.Bytes(), nil
	}
	if errCache != nil && !errors.Is(errCache, ErrCacheExpired) {
		return nil, errors.Wrap(errCache, "unexpected cache error")
	}
	localCtx, cancel := context.WithTimeout(ctx, DurationWebRequestTimeout)
	defer cancel()
	req, reqErr := http.NewRequestWithContext(localCtx, "GET", store.AvatarUrl(hash), nil)
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
	defer util.LogClose(rootLogger, resp.Body)

	if errSet := fsCache.Set(TypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
		return nil, errors.Wrap(errSet, "failed to set cached value")
	}

	return body, nil
}

func createLogReader() (*logReader, error) {
	consoleLogPath := filepath.Join(settings.GetTF2Dir(), "console.log")
	return newLogReader(rootLogger, consoleLogPath, logChan, true)
}

func exportVoiceBans() error {
	bannedIds := rules.FindNewestEntries(200, settings.GetKickTags())
	if len(bannedIds) == 0 {
		return nil
	}
	vbPath := filepath.Join(settings.GetTF2Dir(), "voice_ban.dt")
	vbFile, errOpen := os.OpenFile(vbPath, os.O_RDWR|os.O_TRUNC, 0755)
	if errOpen != nil {
		return errOpen
	}
	if errWrite := voiceban.Write(vbFile, bannedIds); errWrite != nil {
		return errWrite
	}
	rootLogger.Info("Generated voice_ban.dt successfully")
	return nil
}

func LaunchGameAndWait() {
	defer func() {
		gameProcessActive.Store(false)
		rconConn = nil
	}()
	if errInstall := addons.Install(settings.GetTF2Dir()); errInstall != nil {
		rootLogger.Error("Error trying to install addon", zap.Error(errInstall))
	}
	if settings.GetVoiceBansEnabled() {
		if errVB := exportVoiceBans(); errVB != nil {
			rootLogger.Error("Failed to export voiceban list", zap.Error(errVB))
		}
	}
	rconConfig := settings.GetRcon()
	args, errArgs := getLaunchArgs(
		rconConfig.Password(),
		rconConfig.Port(),
		settings.GetSteamDir(),
		settings.GetSteamId())
	if errArgs != nil {
		rootLogger.Error("Failed to get TF2 launch args", zap.Error(errArgs))
		return
	}
	gameHasStartedOnce.Store(true)

	if errLaunch := platform.LaunchTF2(rootLogger, settings.GetTF2Dir(), args); errLaunch != nil {
		rootLogger.Error("Failed to launch game", zap.Error(errLaunch))
	}
}

func Store() store.DataStore {
	return dataStore
}

func Settings() *UserSettings {
	return settings
}

func SetSettings(newSettings *UserSettings) {
	settings = newSettings
}

func Logger() *zap.Logger {
	return rootLogger
}

// Players creates and returns a copy of the current player states
func Players() []store.Player {
	var p []store.Player
	playersMu.RLock()
	defer playersMu.RUnlock()
	for _, plr := range players {
		p = append(p, *plr)
	}
	return p
}

func AddPlayer(p *store.Player) {
	playersMu.Lock()
	defer playersMu.Unlock()
	players = append(players, p)
}

func UnMark(ctx context.Context, sid64 steamid.SID64) error {
	_, errPlayer := GetPlayerOrCreate(ctx, sid64, false)
	if errPlayer != nil {
		return errPlayer
	}
	if !rules.Unmark(sid64) {
		return errors.New("Mark does not exist")
	}
	// Remove existing mark data
	playersMu.Lock()
	defer playersMu.Unlock()
	for idx := range players {
		if players[idx].SteamId == sid64 {
			var valid []*rules.MatchResult
			for _, m := range players[idx].Matches {
				if m.Origin == "local" {
					continue
				}
				valid = append(valid, m)
			}
			players[idx].Matches = valid
			break
		}
	}
	return nil
}

func Mark(ctx context.Context, sid64 steamid.SID64, attrs []string) error {
	player, errPlayer := GetPlayerOrCreate(ctx, sid64, false)
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

	of, errOf := os.OpenFile(settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}
	defer util.LogClose(rootLogger, of)
	if errExport := rules.ExportPlayers(rules.LocalRuleName, of); errExport != nil {
		rootLogger.Error("Failed to export player list", zap.Error(errExport))
	}
	return nil
}

func Whitelist(ctx context.Context, sid64 steamid.SID64, enabled bool) error {
	player, playerErr := GetPlayerOrCreate(ctx, sid64, false)
	if playerErr != nil {
		return playerErr
	}
	player.Whitelisted = enabled
	player.Touch()
	if errSave := dataStore.SavePlayer(ctx, player); errSave != nil {
		return errSave
	}
	if enabled {
		rules.WhitelistAdd(sid64)
	} else {
		rules.WhitelistRemove(sid64)
	}
	rootLogger.Info("Update player whitelist status successfully",
		zap.Int64("steam_id", player.SteamId.Int64()), zap.Bool("enabled", enabled))
	return nil
}

func updatePlayerState() (string, error) {
	if !ready() {
		return "", ErrInvalidReadyState
	}
	// Sent to client, response via log output
	_, errStatus := rconConn.Exec("status")
	if errStatus != nil {
		return "", errors.Wrap(errStatus, "Failed to get status results")

	}
	// Sent to client, response via direct rcon response
	lobbyStatus, errDebug := rconConn.Exec("tf_lobby_debug")
	if errDebug != nil {
		return "", errors.Wrap(errDebug, "Failed to get debug results")
	}
	return lobbyStatus, nil
}

func statusUpdater(ctx context.Context) {
	defer rootLogger.Debug("status updater exited")
	statusTimer := time.NewTicker(DurationStatusUpdateTimer)
	for {
		select {
		case <-statusTimer.C:
			lobbyStatus, errUpdate := updatePlayerState()
			if errUpdate != nil {
				rootLogger.Debug("Failed to query state", zap.Error(errUpdate))
				continue
			}
			for _, line := range strings.Split(lobbyStatus, "\n") {
				parser.ReadChannel <- line
			}
		case <-ctx.Done():
			return
		}
	}
}

func GetPlayerOrCreate(ctx context.Context, sid64 steamid.SID64, active bool) (*store.Player, error) {
	player := GetPlayer(sid64)
	if player == nil {
		player = store.NewPlayer(sid64, "")
		if errGet := dataStore.GetPlayer(ctx, sid64, true, player); errGet != nil {
			if !errors.Is(errGet, sql.ErrNoRows) {
				return nil, errors.Wrap(errGet, "Failed to fetch player record")
			}
			player.ProfileUpdatedOn.AddDate(-1, 0, 0)
		}
		if active {
			playersMu.Lock()
			players = append(players, player)
			playersMu.Unlock()
		}
	}
	if time.Since(player.ProfileUpdatedOn) > profileAgeLimit {
		return player, nil
	}
	mu := sync.RWMutex{}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		bans, errBans := steamweb.GetPlayerBans(steamid.Collection{sid64})
		if errBans != nil || len(bans) == 0 {
			rootLogger.Error("Failed to fetch player bans", zap.Error(errBans))
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
		summaries, errSummaries := steamweb.PlayerSummaries(steamid.Collection{sid64})
		if errSummaries != nil || len(summaries) == 0 {
			rootLogger.Error("Failed to fetch player summary", zap.Error(errSummaries))
		} else {
			mu.Lock()
			defer mu.Unlock()
			summary := summaries[0]
			player.Visibility = store.ProfileVisibility(summary.CommunityVisibilityState)
			if player.AvatarHash != summary.Avatar {
				go performAvatarDownload(ctx, summary.AvatarHash)
			}
			player.Name = summary.PersonaName
			player.AvatarHash = summary.AvatarHash
			player.AccountCreatedOn = time.Unix(int64(summary.TimeCreated), 0)
			player.RealName = summary.RealName
			player.ProfileUpdatedOn = time.Now()
		}
	}()
	wg.Wait()

	if errSave := dataStore.SavePlayer(ctx, player); errSave != nil {
		return nil, errors.Wrap(errSave, "Error trying to save player")
	}

	return player, nil
}

func GetPlayer(sid64 steamid.SID64) *store.Player {
	playersMu.RLock()
	defer playersMu.RUnlock()
	for _, player := range players {
		if player.SteamId == sid64 {
			return player
		}
	}
	return nil
}

func getPlayerByName(name string) *store.Player {
	playersMu.RLock()
	defer playersMu.RUnlock()
	for _, player := range players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func checkHandler(ctx context.Context) {
	defer rootLogger.Debug("checkHandler exited")
	checkTimer := time.NewTicker(DurationCheckTimer)
	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTimer.C:
			p := GetPlayer(settings.GetSteamId())
			if p == nil {
				// We have not connected yet.
				continue
			}
			checkPlayerStates(ctx, p.Team)
		}
	}
}

func cleanupHandler(ctx context.Context) {
	defer rootLogger.Debug("cleanupHandler exited")
	deleteTimer := time.NewTicker(time.Second * time.Duration(settings.PlayerExpiredTimeout))
	for {
		select {
		case <-ctx.Done():
			return
		case <-deleteTimer.C:
			rootLogger.Debug("Delete update input received", zap.String("state", "start"))
			serverMu.Lock()
			if time.Since(server.LastUpdate) > time.Second*time.Duration(settings.PlayerDisconnectTimeout) {
				server = Server{}
			}
			serverMu.Unlock()
			var valid store.PlayerCollection
			expired := 0
			for _, ps := range players {
				if ps.IsExpired() {
					if errSave := dataStore.SavePlayer(ctx, ps); errSave != nil {
						rootLogger.Error("Failed to save expired player state", zap.Error(errSave))
					}
					expired++
				} else {
					valid = append(valid, ps)
				}
			}

			playersMu.Lock()
			players = valid
			playersMu.Unlock()
			if expired > 0 {
				rootLogger.Debug("Flushing expired players", zap.Int("count", expired))
			}
			rootLogger.Debug("Delete update input received", zap.String("state", "end"))
		}
	}
}

func performAvatarDownload(ctx context.Context, hash string) {
	_, errDownload := fetchAvatar(ctx, hash)
	if errDownload != nil {
		rootLogger.Error("Failed to download avatar", zap.String("hash", hash), zap.Error(errDownload))
		return
	}
}

func gameStateUpdater(ctx context.Context) {
	defer rootLogger.Debug("gameStateUpdater exited")
	for {
		select {
		case update := <-gameStateUpdate:
			rootLogger.Debug("Game state update input received", zap.Int("kind", int(update.kind)), zap.String("state", "start"))
			var sourcePlayer *store.Player
			var errSource error
			if update.source.Valid() {
				sourcePlayer, errSource = GetPlayerOrCreate(ctx, update.source, true)
				if errSource != nil {
					rootLogger.Error("failed to get source player", zap.Error(errSource))
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
				if errUm := AddUserMessage(ctx, sourcePlayer.SteamId, evt.message, evt.dead, evt.teamOnly); errUm != nil {
					rootLogger.Error("Failed to handle user message", zap.Error(errUm))
					continue
				}
			case updateKill:
				e, ok := update.data.(killEvent)
				if ok {
					onUpdateKill(e)
				}
			case updateBans:
				onUpdateBans(update.source, update.data.(steamweb.PlayerBanState))
			case updateStatus:
				if errUpdate := onUpdateStatus(ctx, update.source, update.data.(statusEvent)); errUpdate != nil {
					rootLogger.Error("updateStatus error", zap.Error(errUpdate))
				}
			case updateLobby:
				onUpdateLobby(update.source, update.data.(lobbyEvent))
			case updateTags:
				onUpdateTags(update.data.(tagsEvent))
			case updateHostname:
				onUpdateHostname(update.data.(hostnameEvent))
			case updateMap:
				onUpdateMap(update.data.(mapEvent))
			case changeMap:
				onMapChange()
			}
			rootLogger.Debug("Game state update input", zap.Int("kind", int(update.kind)), zap.String("state", "end"))
		}
	}
}

func onUpdateTags(event tagsEvent) {
	serverMu.Lock()
	server.Tags = event.tags
	server.LastUpdate = time.Now()
	serverMu.Unlock()
}

func onUpdateMap(event mapEvent) {
	serverMu.Lock()
	server.CurrentMap = event.mapName
	serverMu.Unlock()
}

func onUpdateHostname(event hostnameEvent) {
	serverMu.Lock()
	server.ServerName = event.hostname
	serverMu.Unlock()
}

func nameToSid(players store.PlayerCollection, name string) steamid.SID64 {
	playersMu.RLock()
	defer playersMu.RUnlock()
	for _, player := range players {
		if name == player.Name {
			return player.SteamId
		}
	}
	return 0
}

func onUpdateLobby(steamID steamid.SID64, evt lobbyEvent) {
	player := GetPlayer(steamID)
	if player != nil {
		playersMu.Lock()
		player.Team = evt.team
		playersMu.Unlock()
	}
}

func AddUserMessage(ctx context.Context, sid64 steamid.SID64, message string, dead bool, teamOnly bool) error {
	player, playerErr := GetPlayerOrCreate(ctx, sid64, false)
	if playerErr != nil {
		return playerErr
	}
	um, errMessage := store.NewUserMessage(player.SteamId, message, dead, teamOnly)
	if errMessage != nil {
		return errMessage
	}
	if errSave := dataStore.SaveMessage(ctx, um); errSave != nil {
		return errSave
	}
	if match := rules.MatchMessage(um.Message); match != nil {
		triggerMatch(player, match)
	}
	return nil
}

func onUpdateKill(kill killEvent) {
	source := nameToSid(players, kill.sourceName)
	target := nameToSid(players, kill.victimName)
	if !source.Valid() || !target.Valid() {
		return
	}
	ourSid := settings.GetSteamId()
	sourcePlayer := GetPlayer(source)
	targetPlayer := GetPlayer(target)
	playersMu.Lock()
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
	playersMu.Unlock()
}

func onMapChange() {
	playersMu.Lock()
	for _, player := range players {
		player.Kills = 0
		player.Deaths = 0
	}
	playersMu.Unlock()
	serverMu.Lock()
	server.CurrentMap = ""
	server.ServerName = ""
	serverMu.Unlock()
}

func onUpdateBans(steamID steamid.SID64, ban steamweb.PlayerBanState) {
	player := GetPlayer(steamID)
	playersMu.Lock()
	defer playersMu.Unlock()
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

func onUpdateStatus(ctx context.Context, steamID steamid.SID64, update statusEvent) error {
	player, errPlayer := GetPlayerOrCreate(ctx, steamID, true)
	if errPlayer != nil {
		return errPlayer
	}
	playersMu.Lock()
	player.Ping = update.ping
	player.UserId = update.userID
	player.Name = update.name
	player.Connected = update.connected.Seconds()
	player.UpdatedOn = time.Now()
	playersMu.Unlock()
	return nil
}

func refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, rootLogger, settings.GetLists())
	for _, list := range playerLists {
		count, errImport := rules.ImportPlayers(&list)
		if errImport != nil {
			rootLogger.Error("Failed to import player list", zap.String("name", list.FileInfo.Title), zap.Error(errImport))
		} else {
			rootLogger.Info("Imported player list", zap.String("name", list.FileInfo.Title), zap.Int("count", count))
		}
	}
	for _, list := range ruleLists {
		count, errImport := rules.ImportRules(&list)
		if errImport != nil {
			rootLogger.Error("Failed to import rules list (%s): %v\n", zap.String("name", list.FileInfo.Title), zap.Error(errImport))
		} else {
			rootLogger.Info("Imported rules list", zap.String("name", list.FileInfo.Title), zap.Int("count", count))
		}
	}
}

func checkPlayerStates(ctx context.Context, validTeam store.Team) {
	for _, ps := range players {
		if ps.IsDisconnected() {
			continue
		}

		if matchSteam := rules.MatchSteam(ps.GetSteamID()); matchSteam != nil {
			ps.Matches = append(ps.Matches, matchSteam...)
			if validTeam == ps.Team {
				triggerMatch(ps, matchSteam)
			}
		} else if ps.Name != "" {
			if matchName := rules.MatchName(ps.GetName()); matchName != nil && validTeam == ps.Team {
				ps.Matches = append(ps.Matches, matchSteam...)
				if validTeam == ps.Team {
					triggerMatch(ps, matchSteam)
				}
			}
		}
		if ps.Dirty {
			if errSave := dataStore.SavePlayer(ctx, ps); errSave != nil {
				rootLogger.Error("Failed to save dirty player state", zap.Error(errSave))
				continue
			}
			ps.Dirty = false
		}
	}
}

func triggerMatch(ps *store.Player, matches []*rules.MatchResult) {
	announceGeneralLast := ps.AnnouncedGeneralLast
	announcePartyLast := ps.AnnouncedPartyLast
	if time.Since(announceGeneralLast) >= DurationAnnounceMatchTimeout {
		msg := "Matched player"
		if ps.Whitelisted {
			msg = "Matched whitelisted player"
		}
		for _, match := range matches {
			rootLogger.Info(msg, zap.String("match_type", match.MatcherType),
				zap.Int64("steam_id", ps.SteamId.Int64()), zap.String("name", ps.Name), zap.String("origin", match.Origin))
		}
		ps.AnnouncedGeneralLast = time.Now()
	}
	if ps.Whitelisted {
		return
	}
	if settings.GetPartyWarningsEnabled() && time.Since(announcePartyLast) >= DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		for _, match := range matches {
			if errLog := SendChat(ChatDestParty, "(%d) [%s] [%s] %s ", ps.UserId, match.Origin, strings.Join(match.Attributes, ","), ps.Name); errLog != nil {
				rootLogger.Error("Failed to send party log message", zap.Error(errLog))
				return
			}
		}
		ps.AnnouncedPartyLast = time.Now()
	}
	if settings.GetKickerEnabled() {
		kickTag := false
		for _, match := range matches {
			for _, tag := range match.Attributes {
				for _, allowedTag := range settings.GetKickTags() {
					if strings.EqualFold(tag, allowedTag) {
						kickTag = true
						break
					}
				}
			}
		}
		if kickTag {
			if errVote := CallVote(ps.UserId, KickReasonCheating); errVote != nil {
				rootLogger.Error("Error calling vote", zap.Error(errVote))
			}
		} else {
			rootLogger.Info("Skipping kick, no acceptable tag found")
		}
	}
	ps.KickAttemptCount++
}

func ensureRcon() error {
	if rconConn != nil {
		return nil
	}
	rconConfig := settings.GetRcon()
	conn, errConn := rcon.Dial(context.TODO(), rconConfig.String(), rconConfig.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}
	rconConn = conn
	return nil
}

func ready() bool {
	if !gameProcessActive.Load() {
		return false
	}
	if errRcon := ensureRcon(); errRcon != nil {
		rootLogger.Debug("RCON is not ready yet", zap.Error(errRcon))
		return false
	}
	return true
}

func SendChat(destination ChatDest, format string, args ...any) error {
	if !ready() {
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
	_, errExec := rconConn.Exec(cmd)
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon chat message")
	}
	return nil
}

func CallVote(userID int64, reason KickReason) error {
	if !ready() {
		return ErrInvalidReadyState
	}
	_, errExec := rconConn.Exec(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}
	return nil
}

func processChecker(ctx context.Context) {
	ticker := time.NewTicker(DurationProcessTimeout)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			existingState := gameProcessActive.Load()
			newState, errRunningStatus := platform.IsGameRunning()
			if errRunningStatus != nil {
				rootLogger.Error("Failed to get process run status", zap.Error(errRunningStatus))
				continue
			}
			if existingState != newState {
				gameProcessActive.Store(newState)
				rootLogger.Info("Game process state changed", zap.Bool("is_running", newState))
			}
			// Handle auto closing the app on game close if enabled
			if !gameHasStartedOnce.Load() || !settings.GetAutoCloseOnGameExit() {
				continue
			}
			if !newState {
				rootLogger.Info("Auto-closing on game exit", zap.Duration("uptime", time.Since(startupTime)))
				os.Exit(0)
			}
		}
	}
}

// Shutdown closes any open rcon connection and will flush any player list to disk
func Shutdown() {
	if rconConn != nil {
		util.LogClose(rootLogger, rconConn)
	}
	defer util.LogClose(rootLogger, dataStore)
	rootLogger.Info("Goodbye")
	if settings.GetDebugLogEnabled() {
		if errSync := rootLogger.Sync(); errSync != nil {
			fmt.Printf("Failed to sync log: %v\n", errSync)
		}
	}
}

func Start(ctx context.Context) {
	go reader.start(ctx)
	defer reader.tail.Cleanup()
	go parser.start(ctx)
	go refreshLists(ctx)
	go incomingLogEventHandler(ctx)
	go gameStateUpdater(ctx)
	go cleanupHandler(ctx)
	go checkHandler(ctx)
	go statusUpdater(ctx)
	go processChecker(ctx)
	go discordStateUpdater(ctx)
	if running, errRunning := platform.IsGameRunning(); errRunning == nil && !running {
		if !gameHasStartedOnce.Load() && settings.GetAutoLaunchGame() {
			go LaunchGameAndWait()
		}
	}

	<-ctx.Done()
}
