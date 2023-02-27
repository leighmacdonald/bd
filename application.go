package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hugolgst/rich-go/client"
	"github.com/leighmacdonald/bd/addons"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/ui"
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

	logChan            chan string
	incomingLogEvents  chan model.LogEvent
	server             model.Server
	serverMu           *sync.RWMutex
	logReader          *logReader
	logParser          *logParser
	rules              *rules.Engine
	rconConnection     rconConnection
	settings           *model.Settings
	store              dataStore
	gui                ui.UserInterface
	triggerUpdate      chan any
	gameStateUpdate    chan updateGameStateEvent
	cache              localCache
	startupTime        time.Time
	richPresenceActive bool
}

// New allocates a new bot detector application instance
func New(settings *model.Settings, store dataStore, rules *rules.Engine) BD {
	logChan := make(chan string)
	eventChan := make(chan model.LogEvent)
	rootApp := BD{
		store:             store,
		rules:             rules,
		settings:          settings,
		logChan:           logChan,
		incomingLogEvents: eventChan,
		serverMu:          &sync.RWMutex{},
		triggerUpdate:     make(chan any),
		gameStateUpdate:   make(chan updateGameStateEvent, 50),
		cache:             newFsCache(settings.ConfigRoot(), model.DurationCacheTimeout),
		logParser:         newLogParser(logChan, eventChan),
		startupTime:       time.Now(),
	}

	rootApp.createLogReader()

	rootApp.reload()

	return rootApp
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

func fetchAvatar(ctx context.Context, cache localCache, hash string) ([]byte, error) {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := cache.Get(cacheTypeAvatar, hash, buf)
	if errCache == nil {
		return buf.Bytes(), nil
	}
	if errCache != nil && !errors.Is(errCache, errCacheExpired) {
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
	defer logClose(resp.Body)

	if errSet := cache.Set(cacheTypeAvatar, hash, bytes.NewReader(body)); errSet != nil {
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
			bd.serverMu.Lock()
			bd.server.LastUpdate = time.Now()
			bd.server.CurrentMap = evt.MetaData
			bd.serverMu.Unlock()
		case model.EvtHostname:
			bd.serverMu.Lock()
			bd.server.LastUpdate = time.Now()
			bd.server.ServerName = evt.MetaData
			bd.serverMu.Unlock()
		case model.EvtTags:
			bd.serverMu.Lock()
			bd.server.Tags = strings.Split(evt.MetaData, ",")
			bd.server.LastUpdate = time.Now()
			bd.serverMu.Unlock()
			// We only bother to call this for the tags event since it should be parsed last for the status output, updating all
			// the other fields at the same time.
			bd.gui.UpdateServerState(bd.server)
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
			bd.serverMu.Lock()
			bd.server.LastUpdate = time.Now()
			bd.server.Addr = ip
			bd.server.Port = uint16(portValue)
			bd.serverMu.Unlock()
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
			bd.gameStateUpdate <- updateGameStateEvent{
				kind:   updateLobby,
				source: evt.PlayerSID,
				data:   lobbyEvent{team: evt.Team},
			}
		}
	}
}

func (bd *BD) launchGameAndWait() {
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
	if errLaunch := platform.LaunchTF2(bd.settings.TF2Dir, args); errLaunch != nil {
		log.Printf("Failed to launch game: %v\n", errLaunch)
	}
}

func (bd *BD) onMark(sid64 steamid.SID64, attrs []string) error {
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

type updateType int

const (
	updateKill updateType = iota
	updateProfile
	updateBans
	updateStatus
	updateMark
	updateMessage
	updateLobby
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

type messageEvent struct {
	name      string
	createdAt time.Time
	message   string
	teamOnly  bool
	dead      bool
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

// gameStateTracker handle processing incoming updateGameStateEvent events and applying them to the
// current known player states stored locally in the players map.
func (bd *BD) gameStateTracker(ctx context.Context) {
	var queuedUpdates steamid.Collection
	queueUpdate := false
	players := map[steamid.SID64]*model.Player{}
	queueAvatars := make(chan steamid.SID64, 32)
	deleteTimer := time.NewTicker(model.DurationPlayerExpired)
	checkTimer := time.NewTicker(model.DurationCheckTimer)
	updateTimer := time.NewTicker(model.DurationUpdateTimer)

	updateUI := func() {
		var p model.PlayerCollection
		for _, pl := range players {
			p = append(p, pl)
		}
		sort.Slice(p, func(i, j int) bool {
			return strings.ToLower(p[i].Name) < strings.ToLower(p[j].Name)
		})
		bd.gui.UpdatePlayerState(p)
		bd.gui.Refresh()
		queueUpdate = false
	}

	for {
		select {
		case <-updateTimer.C:
			if queueUpdate {
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
			p, found := players[bd.settings.GetSteamId()]
			if !found {
				// We have not connected yet.
				continue
			}
			bd.checkPlayerStates(ctx, p.Team, players)
			queueUpdate = true
		case <-deleteTimer.C:
			var valid []*model.Player
			expired := 0
			for steamID, ps := range players {
				if ps.IsExpired() {
					if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
						log.Printf("Failed to save expired player state: %v\n", errSave)
					}
					delete(players, steamID)
					expired++
				}
			}
			if expired > 0 {
				log.Printf("Players expired: %d\n", expired)
			}
			queueUpdate = true
			bd.discordUpdateActivity(len(valid))
		case sid64 := <-queueAvatars:
			p, e := players[sid64]
			if !e || p.AvatarHash == "" {
				continue
			}
			avatar, errDownload := fetchAvatar(ctx, bd.cache, p.AvatarHash)
			if errDownload != nil {
				log.Printf("Failed to download avatar [%s]: %v\n", p.AvatarHash, errDownload)
				continue
			}
			players[sid64].SetAvatar(players[sid64].AvatarHash, avatar)
			queueUpdate = true
		case update := <-bd.gameStateUpdate:
			found := false
			if update.source.Valid() {
				_, found = players[update.source]
				if !found && update.kind != updateStatus {
					// Only register a new user to track once we received a status line
					continue
				}
			}
			switch update.kind {
			case updateMessage:
				um := model.UserMessage{}
				if errUm := onUpdateMessage(ctx, players, update.data.(messageEvent), bd.store, &um); errUm != nil {
					log.Printf("Failed to handle user message: %v", errUm)
					continue
				}
				if match := bd.rules.MatchMessage(um.Message); match != nil {
					bd.triggerMatch(ctx, players[update.source], match)
				}
				bd.gui.AddUserMessage(um)
			case updateKill:
				onUpdateKill(players, update.data.(killEvent))
			case updateBans:
				onUpdateBans(players, update.source, update.data.(steamweb.PlayerBanState))
			case updateProfile:
				onUpdateProfile(players, update.source, update.data.(steamweb.PlayerSummary))
				queueAvatars <- update.source
			case updateStatus:
				if errUpdate := onUpdateStatus(ctx, bd.store, update.source, update.data.(statusEvent), players, found, &queuedUpdates); errUpdate != nil {
					log.Printf("updateStatus error: %v\n", errUpdate)
				}
			case updateMark:
				if errUpdate := bd.onUpdateMark(update.data.(updateMarkEvent), players); errUpdate != nil {
					log.Printf("updateMark error: %v\n", errUpdate)
				}
			case updateLobby:
				onUpdateLobby(players, update.source, update.data.(lobbyEvent))
			}
			queueUpdate = true
		}
	}
}

func nameToSid(players map[steamid.SID64]*model.Player, name string) steamid.SID64 {
	for _, player := range players {
		if name == player.Name {
			return player.SteamId
		}
	}
	return 0
}
func onUpdateLobby(players map[steamid.SID64]*model.Player, steamID steamid.SID64, evt lobbyEvent) {
	players[steamID].Team = evt.team
}

func onUpdateMessage(ctx context.Context, players map[steamid.SID64]*model.Player, msg messageEvent, store dataStore, um *model.UserMessage) error {
	source := nameToSid(players, msg.name)
	if !source.Valid() {
		return errors.New("Invalid steamid")
	}
	um.Player = players[source].Name
	um.Team = players[source].Team
	um.PlayerSID = players[source].SteamId
	um.UserId = players[source].UserId
	um.Message = msg.message
	um.Created = msg.createdAt
	um.Dead = msg.dead
	um.TeamOnly = msg.teamOnly
	if errSaveMsg := store.SaveMessage(ctx, um); errSaveMsg != nil {
		log.Printf("Error trying to store user messge log: %v\n", errSaveMsg)
	}
	return nil
}

func onUpdateKill(players map[steamid.SID64]*model.Player, kill killEvent) {
	source := nameToSid(players, kill.sourceName)
	target := nameToSid(players, kill.victimName)
	if source.Valid() {
		players[source].Kills++
		players[source].Touch()
	}
	if target.Valid() {
		players[target].Deaths++
		players[target].Touch()
	}

}

func onUpdateBans(players map[steamid.SID64]*model.Player, steamID steamid.SID64, ban steamweb.PlayerBanState) {
	players[steamID].NumberOfVACBans = ban.NumberOfVACBans
	players[steamID].NumberOfGameBans = ban.NumberOfGameBans
	players[steamID].CommunityBanned = ban.CommunityBanned
	if ban.DaysSinceLastBan > 0 {
		subTime := time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
		players[steamID].LastVACBanOn = &subTime
	}
	players[steamID].EconomyBan = ban.EconomyBan != "none"
	players[steamID].Touch()
}

func onUpdateProfile(players map[steamid.SID64]*model.Player, steamID steamid.SID64, summary steamweb.PlayerSummary) {
	players[steamID].Visibility = model.ProfileVisibility(summary.CommunityVisibilityState)
	players[steamID].AvatarHash = summary.AvatarHash
	players[steamID].AccountCreatedOn = time.Unix(int64(summary.TimeCreated), 0)
	players[steamID].RealName = summary.RealName
	players[steamID].ProfileUpdatedOn = time.Now()
	players[steamID].Touch()
}

func onUpdateStatus(ctx context.Context, store dataStore, steamID steamid.SID64, update statusEvent, players map[steamid.SID64]*model.Player, found bool, queuedUpdates *steamid.Collection) error {
	if !found {
		ps := model.NewPlayer(steamID, update.name)
		if errCreate := store.LoadOrCreatePlayer(ctx, steamID, ps); errCreate != nil {
			return errors.Wrap(errCreate, "Error trying to load/create player\n")
		}
		if update.name != "" && update.name != ps.NamePrevious {
			if errSaveName := store.SaveName(ctx, steamID, ps.Name); errSaveName != nil {
				return errors.Wrap(errSaveName, "Failed to save name")
			}
		}
		players[steamID] = ps
	}
	players[steamID].Ping = update.ping
	players[steamID].UserId = update.userID
	players[steamID].Name = update.name
	players[steamID].Connected = update.connected
	players[steamID].UpdatedOn = time.Now()
	if time.Since(players[steamID].ProfileUpdatedOn) > model.DurationCacheTimeout {
		*queuedUpdates = append(*queuedUpdates, steamID)
	}
	return nil
}

func (bd *BD) onUpdateMark(status updateMarkEvent, players map[steamid.SID64]*model.Player) error {
	name := ""
	for _, player := range players {
		if player.SteamId == status.target {
			name = player.Name
			continue
		}
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
	logClose(of)
	return nil
}

// AttachGui connects the backend functions to the frontend gui
// TODO Use channels for communicating instead
func (bd *BD) AttachGui(ctx context.Context, gui ui.UserInterface) {
	gui.SetBuildInfo(version, commit, date, builtBy)
	gui.SetOnLaunchTF2(func() {
		go bd.launchGameAndWait()
	})
	gui.SetOnMark(bd.onMark)
	gui.SetOnKick(func(userId int64, reason model.KickReason) error {
		return bd.callVote(ctx, userId, reason)
	})
	gui.SetFetchMessageHistory(func(sid64 steamid.SID64) (model.UserMessageCollection, error) {
		return bd.store.FetchMessages(ctx, sid64)
	})
	gui.SetFetchNameHistory(func(sid64 steamid.SID64) ([]model.UserNameHistory, error) {
		return bd.store.FetchNames(ctx, sid64)
	})
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

func (bd *BD) checkPlayerStates(ctx context.Context, validTeam model.Team, players map[steamid.SID64]*model.Player) {
	for _, ps := range players {
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
	//bd.gui.UpdatePlayerState(players)
}

func (bd *BD) triggerMatch(ctx context.Context, ps *model.Player, match *rules.MatchResult) {
	log.Printf("Matched (%s):  %d %s %s", match.MatcherType, ps.SteamId, ps.Name, match.Origin)
	if bd.settings.PartyWarningsEnabled && time.Since(ps.AnnouncedLast) >= model.DurationAnnounceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		if errLog := bd.partyLog(ctx, "Bot: (%d) [%s] %s ", ps.UserId, match.Origin, ps.Name); errLog != nil {
			log.Printf("Failed to send party log message: %s\n", errLog)
			return
		}
		ps.AnnouncedLast = time.Now()

	}
	if bd.settings.KickerEnabled {
		if errVote := bd.callVote(ctx, ps.UserId, model.KickReasonCheating); errVote != nil {
			log.Printf("Error calling vote: %v\n", errVote)
		}
	}
	ps.KickAttemptCount++
}

func (bd *BD) connectRcon(ctx context.Context) error {
	if bd.rconConnection != nil {
		logClose(bd.rconConnection)
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

func (bd *BD) callVote(ctx context.Context, userID int64, reason model.KickReason) error {
	if errConn := bd.connectRcon(ctx); errConn != nil {
		return errConn
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("callvote kick \"%d %s\"", userID, reason))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}
	return nil
}

// Shutdown closes any open rcon connection and will flush any player list to disk
func (bd *BD) Shutdown() {
	if bd.rconConnection != nil {
		logClose(bd.rconConnection)
	}
	if bd.settings.DiscordPresenceEnabled {
		client.Logout()
	}
	logClose(bd.store)
}

func (bd *BD) start(ctx context.Context) {
	go bd.logReader.start(ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(ctx)
	go bd.refreshLists(ctx)
	go bd.eventHandler()
	go bd.gameStateTracker(ctx)
	go bd.statusUpdater(ctx)
	<-ctx.Done()
}
