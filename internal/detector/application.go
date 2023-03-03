package detector

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hugolgst/rich-go/client"
	"github.com/leighmacdonald/bd/internal/addons"
	"github.com/leighmacdonald/bd/internal/cache"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BD is the main application container
type BD struct {
	// TODO
	// - estimate private steam account ages (find nearby non-private account)
	// - "unmark" players, overriding any lists that may match
	// - track rage quits
	// - auto generate voice_ban.dt
	// - install vote fail mod
	// - wipe map session stats k/d
	// - track k/d over entire session?
	// - track history of interactions with players
	// - colourise messages that trigger
	// - auto launch tf2 upon open
	// - track stopwatch time-ish via 02/28/2023 - 23:40:21: Teams have been switched.
	// - Save custom notes on users
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
	gameStateUpdate    chan updateGameStateEvent
	cache              cache.FsCache
	startupTime        time.Time
	gameHasStartedOnce bool
	richPresenceActive bool
}

// New allocates a new bot detector application instance
func New(settings *model.Settings, store store.DataStore, rules *rules.Engine, cache cache.FsCache) BD {
	logChan := make(chan string)
	eventChan := make(chan model.LogEvent)
	rootApp := BD{
		store:              store,
		rules:              rules,
		settings:           settings,
		logChan:            logChan,
		incomingLogEvents:  eventChan,
		serverMu:           &sync.RWMutex{},
		players:            model.PlayerCollection{},
		playersMu:          &sync.RWMutex{},
		triggerUpdate:      make(chan any),
		gameStateUpdate:    make(chan updateGameStateEvent, 50),
		cache:              cache,
		logParser:          newLogParser(logChan, eventChan),
		startupTime:        time.Now(),
		gameHasStartedOnce: platform.IsGameRunning(),
	}

	rootApp.createLogReader()

	rootApp.reload()

	return rootApp
}

func (bd *BD) Settings() *model.Settings {
	return bd.settings
}

func (bd *BD) reload() {
	if bd.settings.DiscordPresenceEnabled {
		if errLogin := bd.discordLogin(); errLogin != nil {
			log.Printf("Failed to login for discord rich presence\n")
		}
	} else {
		client.Logout()
	}
}

const discordAppID = "1076716221162082364"

func (bd *BD) discordLogin() error {
	if !bd.richPresenceActive {
		if errLogin := client.Login(discordAppID); errLogin != nil {
			return errors.Wrap(errLogin, "Failed to login to discord api\n")
		}
		bd.richPresenceActive = true
	}
	return nil
}

func (bd *BD) discordLogout() {
	if bd.richPresenceActive {
		client.Logout()
		bd.richPresenceActive = false
	}
}

func (bd *BD) discordUpdateActivity(cnt int) {
	if !bd.settings.DiscordPresenceEnabled {
		return
	}
	bd.serverMu.RLock()
	defer bd.serverMu.RUnlock()
	if time.Since(bd.server.LastUpdate) > model.DurationDisconnected {
		bd.discordLogout()
		return
	}
	if bd.server.CurrentMap != "" {
		if errLogin := bd.discordLogin(); errLogin != nil {
			return
		}
		name := ""
		buttons := []*client.Button{
			{
				Label: "GitHub",
				Url:   "https://github.com/leighmacdonald/bd",
			},
		}
		if !bd.server.Addr.IsLinkLocalUnicast() /*SDR*/ && !bd.server.Addr.IsPrivate() {
			buttons = append(buttons, &client.Button{
				Label: "Connect",
				Url:   fmt.Sprintf("steam://connect/%s:%d", bd.server.Addr.String(), bd.server.Port),
			})
		}
		currentMap := discordAssetNameMap(bd.server.CurrentMap)
		if errSetActivity := client.SetActivity(client.Activity{
			State:      "In-Game",
			Details:    bd.server.ServerName,
			LargeImage: fmt.Sprintf("map_%s", currentMap),
			LargeText:  currentMap,
			SmallImage: name,
			SmallText:  bd.server.CurrentMap,
			Party: &client.Party{
				ID:         "-1",
				Players:    cnt,
				MaxPlayers: 24,
			},
			Timestamps: &client.Timestamps{
				Start: &bd.startupTime,
			},
			Buttons: buttons,
		}); errSetActivity != nil {
			log.Printf("Failed to set discord activity: %v\n", errSetActivity)
		}
	}
}

func fetchAvatar(ctx context.Context, c cache.Cache, hash string) ([]byte, error) {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := c.Get(cache.TypeAvatar, hash, buf)
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
	defer util.LogClose(resp.Body)

	if errSet := c.Set(cache.TypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
		return nil, errors.Wrap(errSet, "failed to set cached value")
	}

	return body, nil
}

func (bd *BD) createLogReader() {
	consoleLogPath := filepath.Join(bd.settings.TF2Dir, "console.log")
	reader, errLogReader := newLogReader(consoleLogPath, bd.logChan, true)
	if errLogReader != nil {
		panic(errLogReader)
	}
	bd.logReader = reader
}

func (bd *BD) eventHandler() {
	for {
		evt := <-bd.incomingLogEvents
		switch evt.Type {
		case model.EvtMap:
			bd.gameStateUpdate <- updateGameStateEvent{kind: updateMap, data: mapEvent{mapName: evt.MetaData}}
		case model.EvtHostname:
			bd.gameStateUpdate <- updateGameStateEvent{kind: updateHostname, data: hostnameEvent{hostname: evt.MetaData}}
		case model.EvtTags:
			bd.gameStateUpdate <- updateGameStateEvent{kind: updateTags, data: tagsEvent{tags: strings.Split(evt.MetaData, ",")}}
		case model.EvtAddress:
			pcs := strings.Split(evt.MetaData, ":")
			portValue, errPort := strconv.ParseUint(pcs[1], 10, 16)
			if errPort != nil {
				log.Printf("Failed to parse port: %v", errPort)
				continue
			}
			ip := net.ParseIP(pcs[0])
			if ip == nil {
				log.Printf("Failed to parse ip: %v", pcs[0])
				continue
			}
			bd.gameStateUpdate <- updateGameStateEvent{kind: updateAddress, data: addressEvent{ip: ip, port: uint16(portValue)}}
		case model.EvtDisconnect:
			// We don't really care about this, handled later via UpdatedOn timeout so that there is a
			// lag between actually removing the player from the player table.
			log.Printf("Player disconnected: %d", evt.PlayerSID.Int64())
		case model.EvtKill:
			bd.gameStateUpdate <- updateGameStateEvent{
				kind:   updateKill,
				source: evt.PlayerSID,
				data:   killEvent{victimName: evt.Victim, sourceName: evt.Player},
			}
		case model.EvtMsg:
			bd.gameStateUpdate <- updateGameStateEvent{
				kind:   updateMessage,
				source: evt.PlayerSID,
				data: messageEvent{
					name:      evt.Player,
					createdAt: evt.Timestamp,
					message:   evt.Message,
					teamOnly:  evt.TeamOnly,
					dead:      evt.Dead,
				},
			}
		case model.EvtStatusId:
			bd.gameStateUpdate <- updateGameStateEvent{
				kind:   updateStatus,
				source: evt.PlayerSID,
				data: statusEvent{
					playerSID: evt.PlayerSID,
					ping:      evt.PlayerPing,
					userID:    evt.UserId,
					name:      evt.Player,
					connected: evt.PlayerConnected,
				},
			}
		case model.EvtLobby:
			bd.gameStateUpdate <- updateGameStateEvent{kind: updateLobby, source: evt.PlayerSID, data: lobbyEvent{team: evt.Team}}
		}
	}
}

func (bd *BD) LaunchGameAndWait() {
	if errInstall := addons.Install(bd.settings.TF2Dir); errInstall != nil {
		log.Printf("Error trying to install addons: %v", errInstall)
	}
	args, errArgs := getLaunchArgs(
		bd.settings.Rcon.Password(),
		bd.settings.Rcon.Port(),
		bd.settings.SteamDir,
		bd.settings.GetSteamId())
	if errArgs != nil {
		log.Println(errArgs)
		return
	}
	bd.gameHasStartedOnce = true
	if errLaunch := platform.LaunchTF2(bd.settings.TF2Dir, args); errLaunch != nil {
		log.Printf("Failed to launch game: %v\n", errLaunch)
	}
}

func (bd *BD) OnMark(sid64 steamid.SID64, attrs []string) error {
	bd.gameStateUpdate <- updateGameStateEvent{
		kind:   updateMark,
		source: bd.settings.GetSteamId(),
		data: updateMarkEvent{
			target: sid64,
			attrs:  attrs,
		},
	}
	return nil
}

func (bd *BD) OnWhitelist(sid64 steamid.SID64) error {
	bd.gameStateUpdate <- updateGameStateEvent{
		kind:   updateWhitelist,
		source: bd.settings.GetSteamId(),
		data: updateWhitelistEvent{
			target: sid64,
		},
	}
	return nil
}

type updateType int

const (
	updateKill updateType = iota
	updateProfile
	updateBans
	updateStatus
	updateMark
	updateMessage
	updateLobby
	updateMap
	updateHostname
	updateTags
	updateAddress
	updateWhitelist
)

type killEvent struct {
	sourceName string
	victimName string
}

type lobbyEvent struct {
	team model.Team
}

type statusEvent struct {
	playerSID steamid.SID64
	ping      int
	userID    int64
	name      string
	connected time.Duration
}

type updateGameStateEvent struct {
	kind   updateType
	source steamid.SID64
	data   any
}

type updateMarkEvent struct {
	target steamid.SID64
	attrs  []string
}

type updateWhitelistEvent struct {
	target steamid.SID64
}

type messageEvent struct {
	name      string
	createdAt time.Time
	message   string
	teamOnly  bool
	dead      bool
}

type hostnameEvent struct {
	hostname string
}

type mapEvent struct {
	mapName string
}
type tagsEvent struct {
	tags []string
}

type addressEvent struct {
	ip   net.IP
	port uint16
}

func fetchSteamWebUpdates(updates steamid.Collection) ([]updateGameStateEvent, error) {
	var results []updateGameStateEvent
	summaries, errSummaries := steamweb.PlayerSummaries(updates)
	if errSummaries != nil {
		return nil, errors.Wrap(errSummaries, "Failed to fetch summaries: %v\n")
	}
	for _, sum := range summaries {
		sid, errSid := steamid.SID64FromString(sum.Steamid)
		if errSid != nil {
			log.Printf("Invalid sid from api?: %v\n", errSid)
			continue
		}
		results = append(results, updateGameStateEvent{
			kind:   updateProfile,
			source: sid,
			data:   sum,
		})
	}
	log.Printf("Fetched %d summaries", len(summaries))
	bans, errBans := steamweb.GetPlayerBans(updates)
	if errBans != nil {
		return nil, errors.Wrap(errBans, "Failed to fetch bans: %v\n")
	}
	for _, ban := range bans {
		sid, errSid := steamid.SID64FromString(ban.SteamID)
		if errSid != nil {
			return nil, errors.Wrap(errSummaries, "Invalid sid from api?: %v\n")
		}
		results = append(results, updateGameStateEvent{
			kind:   updateBans,
			source: sid,
			data:   ban,
		})
	}
	log.Printf("Fetched %d bans", len(bans))
	return results, nil
}

func (bd *BD) statusUpdater(ctx context.Context) {
	statusTimer := time.NewTicker(model.DurationStatusUpdateTimer)
	for {
		select {
		case <-statusTimer.C:
			lobbyStatus, errUpdate := updatePlayerState(ctx, bd.settings.Rcon.String(), bd.settings.Rcon.Password())
			if errUpdate != nil {
				log.Printf("Failed to query state: %v\n", errUpdate)
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

// gameStateTracker handle processing incoming updateGameStateEvent events and applying them to the
// current known player states stored locally in the players map.
func (bd *BD) gameStateTracker(ctx context.Context) {
	var queuedUpdates steamid.Collection
	queueUpdate := false

	queueAvatars := make(chan steamid.SID64, 32)
	deleteTimer := time.NewTicker(model.DurationPlayerExpired)
	checkTimer := time.NewTicker(model.DurationCheckTimer)
	updateTimer := time.NewTicker(model.DurationUpdateTimer)

	updateUI := func() {
		bd.playersMu.Lock()
		sort.Slice(bd.players, func(i, j int) bool {
			return strings.ToLower(bd.players[i].Name) < strings.ToLower(bd.players[j].Name)
		})
		bd.playersMu.Unlock()
		bd.gui.UpdatePlayerState(bd.players)
		bd.gui.Refresh()
		queueUpdate = false
	}

	for {
		select {
		case <-updateTimer.C:
			if queueUpdate {
				// TODO not necessary?
				updateUI()
			}
			if len(queuedUpdates) == 0 || bd.settings.ApiKey == "" {
				continue
			}
			if len(queuedUpdates) > 100 {
				var trimmed steamid.Collection
				for i := len(queuedUpdates) - 1; len(trimmed) < 100; i-- {
					trimmed = append(trimmed, queuedUpdates[i])
				}
				queuedUpdates = trimmed
			}
			log.Printf("Updating %d profiles\n", len(queuedUpdates))
			results, errUpdates := fetchSteamWebUpdates(queuedUpdates)
			if errUpdates != nil {
				continue
			}
			for _, r := range results {
				select {
				case bd.gameStateUpdate <- r:
				default:
					log.Printf("Game update channel full\n")
				}

			}
			queuedUpdates = nil
		case <-checkTimer.C:
			p := bd.GetPlayer(bd.settings.GetSteamId())
			if p == nil {
				// We have not connected yet.
				continue
			}
			bd.checkPlayerStates(ctx, p.Team)
			queueUpdate = true
		case <-deleteTimer.C:
			bd.playersMu.Lock()
			var valid []*model.Player
			expired := 0
			for _, ps := range bd.players {
				if ps.IsExpired() {
					if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
						log.Printf("Failed to save expired player state: %v\n", errSave)
					}
					expired++
				} else {
					valid = append(valid, ps)
				}
			}
			bd.players = valid
			bd.playersMu.Unlock()
			if expired > 0 {
				log.Printf("Players expired: %d\n", expired)
			}
			queueUpdate = true
			bd.discordUpdateActivity(len(valid))
			bd.gui.UpdatePlayerState(bd.players)
		case sid64 := <-queueAvatars:
			p := bd.GetPlayer(sid64)
			if p == nil || p.AvatarHash == "" {
				continue
			}
			avatar, errDownload := fetchAvatar(ctx, bd.cache, p.AvatarHash)
			if errDownload != nil {
				log.Printf("Failed to download avatar [%s]: %v\n", p.AvatarHash, errDownload)
				continue
			}
			bd.gui.SetAvatar(sid64, avatar)
			queueUpdate = true
		case update := <-bd.gameStateUpdate:
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
					log.Printf("Failed to handle user message: %v", errUm)
					continue
				}
			case updateKill:
				bd.onUpdateKill(update.data.(killEvent))
			case updateBans:
				bd.onUpdateBans(update.source, update.data.(steamweb.PlayerBanState))
			case updateProfile:
				bd.onUpdateProfile(update.source, update.data.(steamweb.PlayerSummary))
				queueAvatars <- update.source
			case updateStatus:
				if errUpdate := bd.onUpdateStatus(ctx, bd.store, update.source, update.data.(statusEvent), &queuedUpdates); errUpdate != nil {
					log.Printf("updateStatus error: %v\n", errUpdate)
				}
			case updateMark:
				d := update.data.(updateMarkEvent)
				if errUpdate := bd.onUpdateMark(d); errUpdate != nil {
					log.Printf("updateMark error: %v\n", errUpdate)
				}
			case updateWhitelist:
				if errUpdate := bd.onUpdateWhitelist(update.data.(updateWhitelistEvent)); errUpdate != nil {
					log.Printf("updateWhitelist error: %v\n", errUpdate)
				}
			case updateLobby:
				bd.onUpdateLobby(update.source, update.data.(lobbyEvent))
			case updateTags:
				bd.onUpdateTags(update.data.(tagsEvent))
			case updateHostname:
				bd.onUpdateHostname(update.data.(hostnameEvent))
			case updateMap:
				bd.onUpdateMap(update.data.(mapEvent))
			}
			queueUpdate = true
		}
	}
}

func (bd *BD) onUpdateTags(event tagsEvent) {
	bd.serverMu.Lock()
	bd.server.Tags = event.tags
	bd.serverMu.Unlock()
	bd.serverMu.RLock()
	bd.gui.UpdateServerState(bd.server)
	bd.serverMu.RUnlock()
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
		log.Printf("Error trying to store user messge log: %v\n", errSaveMsg)
	}
	if match := bd.rules.MatchMessage(um.Message); match != nil {
		bd.triggerMatch(ctx, player, match)
	}
	bd.gui.AddUserMessage(um)
	bd.gui.Refresh()
	return nil
}

func (bd *BD) onUpdateKill(kill killEvent) {
	source := bd.nameToSid(bd.players, kill.sourceName)
	target := bd.nameToSid(bd.players, kill.victimName)
	if !source.Valid() || !target.Valid() {
		return
	}
	sourcePlayer := bd.GetPlayer(source)
	bd.playersMu.Lock()
	sourcePlayer.Kills++
	sourcePlayer.Touch()
	bd.playersMu.Unlock()

	targetPlayer := bd.GetPlayer(source)
	bd.playersMu.Lock()
	targetPlayer.Kills++
	targetPlayer.Touch()
	bd.playersMu.Unlock()

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
		return errors.New("Unknown player, cannot whitelist")
	}
	bd.playersMu.Lock()
	player.Whitelisted = true
	player.Touch()
	bd.playersMu.Unlock()
	log.Printf("whitelisted player: %d", player.SteamId)
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
	if errMark := bd.rules.Mark(rules.MarkOpts{
		SteamID:    status.target,
		Attributes: status.attrs,
		Name:       name,
	}); errMark != nil {
		return errors.Wrap(errMark, "Failed to add mark")
	}
	of, errOf := os.OpenFile(bd.settings.LocalPlayerListPath(), os.O_RDWR|os.O_CREATE, 0666)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open player list for updating")
	}
	if errExport := bd.rules.ExportPlayers(rules.LocalRuleName, of); errExport != nil {
		log.Printf("Failed to export player list: %v\n", errExport)
	}
	util.LogClose(of)
	return nil
}

// AttachGui connects the backend functions to the frontend gui
// TODO Use channels for communicating instead
func (bd *BD) AttachGui(gui model.UserInterface) {
	gui.UpdateAttributes(bd.rules.UniqueTags())
	bd.gui = gui
}

func (bd *BD) refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, bd.settings.Lists)
	for _, list := range playerLists {
		if errImport := bd.rules.ImportPlayers(&list); errImport != nil {
			log.Printf("Failed to import player list (%s): %v\n", list.FileInfo.Title, errImport)
		}
	}
	for _, list := range ruleLists {
		if errImport := bd.rules.ImportRules(&list); errImport != nil {
			log.Printf("Failed to import rules list (%s): %v\n", list.FileInfo.Title, errImport)
		}
	}
	// TODO move
	bd.gui.UpdateAttributes(bd.rules.UniqueTags())
}

func (bd *BD) checkPlayerStates(ctx context.Context, validTeam model.Team) {
	for _, ps := range bd.players {
		if ps.IsDisconnected() {
			continue
		}
		if matchSteam := bd.rules.MatchSteam(ps.GetSteamID()); matchSteam != nil {
			ps.Match = matchSteam
			if validTeam == ps.Team {
				bd.triggerMatch(ctx, ps, matchSteam)
			}
		} else if ps.Name != "" {
			if matchName := bd.rules.MatchName(ps.GetName()); matchName != nil && validTeam == ps.Team {
				ps.Match = matchSteam
				if validTeam == ps.Team {
					bd.triggerMatch(ctx, ps, matchSteam)
				}
			}
		}
		if ps.Dirty {
			if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
				log.Printf("Failed to save player state: %v\n", errSave)
				continue
			}
			ps.Dirty = false
		}
	}
	bd.gui.UpdatePlayerState(bd.players)
}

func (bd *BD) triggerMatch(ctx context.Context, ps *model.Player, match *rules.MatchResult) {
	if ps.Whitelisted {
		log.Printf("Matched (%s):  %d %s %s [whitelisted]", match.MatcherType, ps.SteamId, ps.Name, match.Origin)
		return
	} else {
		log.Printf("Matched (%s):  %d %s %s", match.MatcherType, ps.SteamId, ps.Name, match.Origin)
	}
	if bd.settings.PartyWarningsEnabled && time.Since(ps.AnnouncedLast) >= model.DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		if errLog := bd.partyLog(ctx, "Bot: (%d) [%s] %s ", ps.UserId, match.Origin, ps.Name); errLog != nil {
			log.Printf("Failed to send party log message: %s\n", errLog)
			return
		}
		bd.playersMu.Lock()
		ps.AnnouncedLast = time.Now()
		bd.playersMu.Unlock()
	}
	if bd.settings.KickerEnabled {
		kickTag := false
		for _, tag := range match.Attributes {
			for _, allowedTag := range bd.settings.KickTags {
				if strings.EqualFold(tag, allowedTag) {
					kickTag = true
					break
				}
			}
		}
		if kickTag {
			if errVote := bd.CallVote(ctx, ps.UserId, model.KickReasonCheating); errVote != nil {
				log.Printf("Error calling vote: %v\n", errVote)
			}
		} else {
			log.Printf("Skipping kick, no acceptable tag found")
		}
	}
	bd.playersMu.Lock()
	ps.KickAttemptCount++
	bd.playersMu.Unlock()
}

func (bd *BD) connectRcon(ctx context.Context) error {
	if bd.rconConnection != nil {
		util.LogClose(bd.rconConnection)
	}
	conn, errConn := rcon.Dial(ctx, bd.settings.Rcon.String(), bd.settings.Rcon.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}
	bd.rconConnection = conn
	return nil
}

func (bd *BD) partyLog(ctx context.Context, fmtStr string, args ...any) error {
	if errConn := bd.connectRcon(ctx); errConn != nil {
		return errConn
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("say_party %s", fmt.Sprintf(fmtStr, args...)))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon say_party")
	}
	return nil
}

func (bd *BD) CallVote(ctx context.Context, userID int64, reason model.KickReason) error {
	if errConn := bd.connectRcon(ctx); errConn != nil {
		return errConn
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
			if !bd.gameHasStartedOnce || !bd.settings.AutoCloseOnGameExit {
				continue
			}
			if !platform.IsGameRunning() {
				log.Printf("Auto-closing on game exit\n")
				bd.gui.Quit()
			}
		}
	}
}

// Shutdown closes any open rcon connection and will flush any player list to disk
func (bd *BD) Shutdown() {
	if bd.rconConnection != nil {
		util.LogClose(bd.rconConnection)
	}
	if bd.settings.DiscordPresenceEnabled {
		client.Logout()
	}
	util.LogClose(bd.store)
	log.Printf("Goodbye\n")
}

func (bd *BD) Start(ctx context.Context) {
	go bd.logReader.start(ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(ctx)
	go bd.refreshLists(ctx)
	go bd.eventHandler()
	go bd.gameStateTracker(ctx)
	go bd.statusUpdater(ctx)
	go bd.processChecker(ctx)
	if !bd.gameHasStartedOnce && bd.settings.AutoLaunchGame && !platform.IsGameRunning() {
		go bd.LaunchGameAndWait()
	}
	<-ctx.Done()
}
