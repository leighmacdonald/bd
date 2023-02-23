package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hugolgst/rich-go/client"
	"github.com/leighmacdonald/bd/model"
	rules2 "github.com/leighmacdonald/bd/pkg/rules"
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	logChan            chan string
	incomingLogEvents  chan model.LogEvent
	server             model.ServerState
	serverMu           *sync.RWMutex
	players            []*model.PlayerState
	messages           []*model.UserMessage
	logReader          *logReader
	logParser          *logParser
	rules              *rules2.Engine
	rconConnection     rconConnection
	settings           *model.Settings
	store              dataStore
	gui                ui.UserInterface
	profileUpdateQueue chan steamid.SID64
	triggerUpdate      chan any
	cache              localCache
	startupTime        time.Time
	richPresenceActive bool
}

// New allocates a new bot detector application instance
func New(settings *model.Settings, store dataStore, rules *rules2.Engine) BD {
	logChan := make(chan string)
	eventChan := make(chan model.LogEvent)
	rootApp := BD{
		store:              store,
		rules:              rules,
		settings:           settings,
		logChan:            logChan,
		incomingLogEvents:  eventChan,
		serverMu:           &sync.RWMutex{},
		profileUpdateQueue: make(chan steamid.SID64),
		triggerUpdate:      make(chan any),
		cache:              newFsCache(settings.ConfigRoot(), time.Hour*12),
		logParser:          newLogParser(logChan, eventChan),
		startupTime:        time.Now(),
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

func (bd *BD) discordUpdateActivity() {
	if !bd.settings.DiscordPresenceEnabled {
		return
	}
	bd.serverMu.RLock()
	defer bd.serverMu.RUnlock()
	if time.Since(bd.server.LastUpdate) > time.Second*30 {
		bd.discordLogout()
		return
	}
	if bd.server.CurrentMap != "" {
		if errLogin := bd.discordLogin(); errLogin != nil {
			return
		}
		cnt := 0
		name := ""
		ownSID := bd.settings.GetSteamId()
		for _, player := range bd.players {
			if player.SteamId == ownSID {
				name = player.Name
			}
			// TODO remove this once we track disconnected players better
			if time.Since(player.UpdatedOn) < time.Second*30 {
				cnt++
			}
		}
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

func (bd *BD) uiStateUpdater(ctx context.Context) {
	updateTicker := time.NewTicker(time.Second)
	discordStateUpdateTicker := time.NewTicker(time.Second * 10)
	updateQueued := false
	for {
		select {
		case <-bd.triggerUpdate:
			updateQueued = true
		case <-discordStateUpdateTicker.C:
			bd.discordUpdateActivity()
		case <-ctx.Done():
			return
		case <-updateTicker.C:
			if !updateQueued {
				continue
			}
			var pStates []model.PlayerState
			for _, player := range bd.players {
				pStates = append(pStates, *player)
			}
			bd.gui.UpdatePlayerState(pStates)
			bd.gui.Refresh()
			updateQueued = false
		}
	}
}

type avatarUpdate struct {
	urlLocation string
	hash        string
	sid         steamid.SID64
}

// profileUpdater handles fetching 3rd party data of players
// MAYBE priority queue for new players in an already established game?
func (bd *BD) profileUpdater(ctx context.Context, interval time.Duration) {
	var queuedUpdates steamid.Collection
	ticker := time.NewTicker(interval)
	for {
		select {
		case queuedSid := <-bd.profileUpdateQueue:
			existsAlready := false
			for _, sid := range queuedUpdates {
				if sid == queuedSid {
					existsAlready = true
					break
				}
			}
			if !existsAlready {
				queuedUpdates = append(queuedUpdates, queuedSid)
			}
		case <-ticker.C:
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
			summaries, errSummaries := steamweb.PlayerSummaries(queuedUpdates)
			if errSummaries != nil {
				log.Printf("Failed to fetch summaries: %v\n", errSummaries)
				continue
			}
			bans, errBans := steamweb.GetPlayerBans(queuedUpdates)
			if errBans != nil {
				log.Printf("Failed to fetch bans: %v\n", errBans)
				continue
			}
			for _, player := range bd.players {
				for _, summary := range summaries {
					if summary.Steamid == player.SteamId.String() {
						player.Visibility = model.ProfileVisibility(summary.CommunityVisibilityState)
						player.AvatarHash = summary.AvatarHash
						player.AccountCreatedOn = time.Unix(int64(summary.TimeCreated), 0)
						player.RealName = summary.RealName
						break
					}
				}
				for _, ban := range bans {
					if ban.SteamID == player.SteamId.String() {
						player.NumberOfVACBans = ban.NumberOfVACBans
						player.NumberOfGameBans = ban.NumberOfGameBans
						player.CommunityBanned = ban.CommunityBanned
						if ban.DaysSinceLastBan > 0 {
							t0 := time.Now()
							subTime := t0.AddDate(0, 0, -ban.DaysSinceLastBan)
							player.LastVACBanOn = &subTime
						}
						player.EconomyBan = ban.EconomyBan != "none"
						break
					}
				}
				player.ProfileUpdatedOn = time.Now()
				if errSave := bd.store.SavePlayer(ctx, player); errSave != nil {
					log.Printf("Failed to save player state: %v", errSave)
				}
			}

			var avatarUpdates []avatarUpdate
			for _, p := range bd.players {
				if p.AvatarHash == "" {
					continue
				}
				avatarUpdates = append(avatarUpdates, avatarUpdate{
					urlLocation: p.AvatarUrl(),
					hash:        p.AvatarHash,
					sid:         p.SteamId,
				})
			}

			wg := &sync.WaitGroup{}
			var errorCount int32 = 0
			for _, update := range avatarUpdates {
				wg.Add(1)
				if errDownload := bd.updateAvatar(ctx, update); errDownload != nil {
					atomic.AddInt32(&errorCount, 1)
				}
				wg.Done()
			}
			log.Printf("Updated %d avatars. [%d failed]", len(avatarUpdates), errorCount)
			queuedUpdates = nil
		}
	}
}

func (bd *BD) updateAvatar(ctx context.Context, u avatarUpdate) error {
	httpClient := &http.Client{}
	buf := bytes.NewBuffer(nil)
	errCache := bd.cache.Get(cacheTypeAvatar, u.hash, buf)
	if errCache == nil {
		for _, player := range bd.players {
			if player.SteamId == u.sid {
				player.SetAvatar(u.hash, buf.Bytes())
				return nil
			}
		}
	}
	if errCache != nil && !errors.Is(errCache, errCacheExpired) {
		return errors.Wrap(errCache, "unexpected cache error: %v")
	}
	localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	req, reqErr := http.NewRequestWithContext(localCtx, "GET", u.urlLocation, nil)
	if reqErr != nil {
		return errors.Wrap(reqErr, "Failed to create avatar download request")
	}
	resp, respErr := httpClient.Do(req)
	if respErr != nil {
		return errors.Wrap(respErr, "Failed to download avatar")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Invalid response code downloading avatar: %d", resp.StatusCode)
	}
	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return errors.Wrap(bodyErr, "Failed to read avatar response body")
	}
	defer logClose(resp.Body)

	if errSet := bd.cache.Set(cacheTypeAvatar, u.hash, bytes.NewReader(body)); errSet != nil {
		return errors.Wrap(errSet, "failed to set cached value")
	}
	for _, player := range bd.players {
		if player.SteamId == u.sid {
			player.SetAvatar(u.hash, body)
			break
		}
	}
	return nil
}

func (bd *BD) createLogReader() {
	consoleLogPath := filepath.Join(bd.settings.TF2Dir, "console.log")
	reader, errLogReader := newLogReader(consoleLogPath, bd.logChan, true)
	if errLogReader != nil {
		panic(errLogReader)
	}
	bd.logReader = reader
}

func (bd *BD) eventHandler(ctx context.Context) {
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
			for _, p := range bd.players {
				if p.Name == evt.Player {
					atomic.AddInt64(&p.Kills, 1)
					if bd.settings.GetSteamId() == evt.PlayerSID {
						atomic.AddInt64(&p.KillsOn, 1)
					}
					p.Touch()
				} else if p.Name == evt.Victim {
					atomic.AddInt64(&p.Deaths, 1)
					if bd.settings.GetSteamId() == evt.VictimSID {
						atomic.AddInt64(&p.DeathsBy, 1)
					}
					p.Touch()
				}
			}
		case model.EvtMsg:
			var ps *model.PlayerState
			for _, p := range bd.players {
				if p.Name == evt.Player {
					evt.PlayerSID = p.SteamId
					evt.UserId = p.UserId
					ps = p
					break
				}
			}
			if evt.PlayerSID == 0 {
				// We don't know the player yet.
				continue
			}
			um := &model.UserMessage{
				Team:      evt.Team,
				Player:    evt.Player,
				PlayerSID: evt.PlayerSID,
				UserId:    evt.UserId,
				Message:   evt.Message,
				Created:   time.Now(),
			}
			bd.messages = append(bd.messages, um)
			if ps == nil {
				ps = model.NewPlayerState(evt.PlayerSID, evt.Player)
				if errCreate := bd.store.LoadOrCreatePlayer(ctx, evt.PlayerSID, ps); errCreate != nil {
					log.Printf("Error trying to load/create player: %v\n", errCreate)
					continue
				}
			}

			if errSaveMsg := bd.store.SaveMessage(ctx, um); errSaveMsg != nil {
				log.Printf("Error trying to store user messge log: %v\n", errSaveMsg)
			}

			bd.gui.AddUserMessage(*um)

			if match := bd.rules.MatchMessage(um.Message); match != nil {
				bd.triggerMatch(ctx, ps, match)
			}

		case model.EvtStatusId:
			ps := model.NewPlayerState(evt.PlayerSID, evt.Player)
			ep := bd.players
			isNew := true
			for _, existingPlayer := range ep {
				if existingPlayer.SteamId == evt.PlayerSID {
					ps = existingPlayer
					isNew = false
					break
				}
			}
			if isNew {
				if errCreate := bd.store.LoadOrCreatePlayer(ctx, evt.PlayerSID, ps); errCreate != nil {
					log.Printf("Error trying to load/create player: %v\n", errCreate)
					continue
				}
				if evt.Player != "" && evt.Player != ps.NamePrevious {
					if errSaveName := bd.store.SaveName(ctx, evt.PlayerSID, evt.Player); errSaveName != nil {
						log.Printf("Failed to save name")
						continue
					}
				}
				ps.UserId = evt.UserId
			}
			ps.UpdatedOn = time.Now()
			ps.Ping = evt.PlayerPing
			ps.Connected = evt.PlayerConnected
			if isNew {
				ep = append(ep, ps)
			}
			bd.players = ep
			if isNew || time.Since(ps.ProfileUpdatedOn) > time.Hour*6 {
				bd.profileUpdateQueue <- evt.PlayerSID
			}
			bd.triggerUpdate <- true
		}
	}
}

func (bd *BD) launchGameAndWait() {
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
	name := ""
	for _, player := range bd.players {
		if player.SteamId == sid64 {
			name = player.Name
			break
		}
	}
	if errMark := bd.rules.Mark(rules2.MarkOpts{
		SteamID:    sid64,
		Attributes: attrs,
		Name:       name,
	}); errMark != nil {
		return errMark
	}
	of, errOf := os.OpenFile(bd.settings.LocalPlayerListPath(), os.O_RDWR, 0666)
	if errOf != nil {
		return errors.Wrapf(errOf, "Failed to open player list for updating")
	}
	defer logClose(of)
	if errExport := bd.rules.ExportPlayers(rules2.LocalRuleName, of); errExport != nil {
		return errors.Wrapf(errExport, "Failed to export player list")
	}
	return nil
}

// AttachGui connects the backend functions to the frontend gui
// TODO Use channels for communicating instead
func (bd *BD) AttachGui(ctx context.Context, gui ui.UserInterface) {
	gui.SetOnLaunchTF2(func() {
		go bd.launchGameAndWait()
	})
	gui.SetOnMark(bd.onMark)
	gui.SetOnKick(func(userId int64, reason model.KickReason) error {
		return bd.callVote(ctx, userId, reason)
	})
	gui.SetFetchMessageHistory(func(sid64 steamid.SID64) ([]model.UserMessage, error) {
		return bd.store.FetchMessages(ctx, sid64)
	})
	gui.SetFetchNameHistory(func(sid64 steamid.SID64) ([]model.UserNameHistory, error) {
		return bd.store.FetchNames(ctx, sid64)
	})
	gui.UpdateAttributes(bd.rules.UniqueTags())
	bd.gui = gui
}

func (bd *BD) playerStateUpdater(ctx context.Context) {
	for range time.NewTicker(time.Second * 10).C {
		//if bd.gameProcess == nil {
		//	continue
		//}
		updatePlayerState(ctx, bd.settings.Rcon.String(), bd.settings.Rcon.Password())
		bd.checkPlayerStates(ctx)
	}
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
	bd.gui.UpdateAttributes(bd.rules.UniqueTags())
}

func (bd *BD) checkPlayerStates(ctx context.Context) {
	var valid []*model.PlayerState
	for _, ps := range bd.players {
		if time.Since(ps.UpdatedOn) > time.Second*15 {
			log.Printf("Player expired: %s %s", ps.SteamId.String(), ps.Name)
			if errSave := bd.store.SavePlayer(ctx, ps); errSave != nil {
				log.Printf("Failed to save expired player state: %v\n", errSave)
			}
			continue
		}
		valid = append(valid, ps)
	}

	for _, ps := range valid {
		if matchSteam := bd.rules.MatchSteam(ps.GetSteamID()); matchSteam != nil {
			bd.triggerMatch(ctx, ps, matchSteam)
		} else if ps.Name != "" {
			if matchName := bd.rules.MatchName(ps.GetName()); matchName != nil {
				bd.triggerMatch(ctx, ps, matchName)
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
	var plState []model.PlayerState
	for _, player := range valid {
		plState = append(plState, *player)
	}
	bd.players = valid
	bd.gui.UpdatePlayerState(plState)

}

const announceMatchTimeout = time.Minute * 5

func (bd *BD) triggerMatch(ctx context.Context, ps *model.PlayerState, match *rules2.MatchResult) {
	log.Printf("Matched (%s):  %d %s %s", match.MatcherType, ps.SteamId, ps.Name, match.Origin)
	if time.Since(ps.AnnouncedLast) >= announceMatchTimeout {
		// Don't spam friends, but eventually remind them if they manage to forget long enough
		if errLog := bd.partyLog(ctx, "Bot: (%d) [%s] %s ", ps.UserId, match.Origin, ps.Name); errLog != nil {
			log.Printf("Failed to send party log message: %s\n", errLog)
			return
		}
		ps.AnnouncedLast = time.Now()
	}
	if errVote := bd.callVote(ctx, ps.UserId, model.KickReasonCheating); errVote != nil {
		log.Printf("Error calling vote: %v\n", errVote)
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
	// Ensure we save on exit
	playerListFile, playerListFileErr := os.Create(bd.settings.LocalPlayerListPath())
	if playerListFileErr != nil {
		log.Panicf("Failed to open player list for writing: %v\n", playerListFileErr)
	}
	if errWrite := bd.rules.ExportPlayers(rules2.LocalRuleName, playerListFile); errWrite != nil {
		log.Panicf("Failed to export player list: %v\n", playerListFileErr)
	}

	rulesFile, rulesFileErr := os.Create(bd.settings.LocalRulesListPath())
	if rulesFileErr != nil {
		log.Panicf("Failed to open player list for writing: %v\n", rulesFileErr)
	}
	if errWrite := bd.rules.ExportRules(rules2.LocalRuleName, rulesFile); errWrite != nil {
		log.Panicf("Failed to export rules list: %v\n", rulesFileErr)
	}
	logClose(bd.store)
}

const profileUpdateRate = time.Second * 10

func (bd *BD) start(ctx context.Context) {
	go bd.logReader.start(ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(ctx)
	go bd.playerStateUpdater(ctx)
	go bd.refreshLists(ctx)
	go bd.eventHandler(ctx)
	go bd.profileUpdater(ctx, profileUpdateRate)
	go bd.uiStateUpdater(ctx)
	<-ctx.Done()
}
