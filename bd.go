package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type BD struct {
	// TODO
	// - estimate private steam account ages (find nearby non-private account)
	// - "unmark" players, overriding any lists that may match
	// - track rage quits
	logChan            chan string
	incomingLogEvents  chan model.LogEvent
	server             *model.ServerState
	players            []*model.PlayerState
	messages           []*model.UserMessage
	ctx                context.Context
	logReader          *logReader
	logParser          *LogParser
	rules              *RulesEngine
	rconConnection     rconConnection
	settings           *model.Settings
	store              dataStore
	gui                ui.UserInterface
	dryRun             bool
	gameProcess        *os.Process
	profileUpdateQueue chan steamid.SID64
	triggerUpdate      chan any
	lastUpdate         time.Time
	cache              localCache
	markFn             model.MarkFunc
}

func New(ctx context.Context, settings *model.Settings, store dataStore, rules *RulesEngine) BD {
	logChan := make(chan string)
	eventChan := make(chan model.LogEvent)
	rootApp := BD{
		ctx:                ctx,
		store:              store,
		rules:              rules,
		settings:           settings,
		logChan:            logChan,
		incomingLogEvents:  eventChan,
		profileUpdateQueue: make(chan steamid.SID64),
		lastUpdate:         time.Now(),
		triggerUpdate:      make(chan any),
		cache:              newFsCache(settings.ConfigRoot(), time.Hour*12),
		logParser:          newLogParser(logChan, eventChan),
	}

	rootApp.createLogReader()

	return rootApp
}

func (bd *BD) uiStateUpdater() {
	updateTicker := time.NewTicker(time.Second)
	updateQueued := false
	for {
		select {
		case <-bd.triggerUpdate:
			updateQueued = true
		case <-bd.ctx.Done():
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
func (bd *BD) profileUpdater(interval time.Duration) {
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
			existingPlayers := bd.players
			for _, player := range existingPlayers {
				for _, summary := range summaries {
					if summary.Steamid == player.SteamId.String() {
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
						player.DaysSinceLastBan = ban.DaysSinceLastBan
						player.EconomyBan = ban.EconomyBan != "none"
						break
					}
				}
			}

			var avatarUpdates []avatarUpdate
			for _, p := range existingPlayers {
				if p.AvatarHash == "" {
					continue
				}
				avatarUpdates = append(avatarUpdates, avatarUpdate{
					urlLocation: p.AvatarUrl(),
					hash:        p.AvatarHash,
					sid:         p.SteamId,
				})
			}
			log.Printf("Updated %d profiles\n", len(queuedUpdates))
			httpClient := &http.Client{}
			wg := &sync.WaitGroup{}
			var errorCount int32 = 0
			for _, update := range avatarUpdates {
				wg.Add(1)
				go func(u avatarUpdate) {
					defer wg.Done()
					buf := bytes.NewBuffer(nil)
					errCache := bd.cache.Get(cacheTypeAvatar, u.hash, buf)
					if errCache == nil {
						for _, player := range bd.players {
							if player.SteamId == u.sid {
								player.SetAvatar(u.hash, buf.Bytes())
								return
							}
						}
					}
					if errCache != nil && !errors.Is(errCache, ErrCacheExpired) {
						log.Printf("unexpected cache error: %v\n", errCache)
						return
					}
					localCtx, cancel := context.WithTimeout(bd.ctx, time.Second*10)
					defer cancel()
					req, reqErr := http.NewRequestWithContext(localCtx, "GET", u.urlLocation, nil)
					if reqErr != nil {
						log.Printf("Failed to create avatar download request: %v", reqErr)
						atomic.AddInt32(&errorCount, 1)
						return
					}
					resp, respErr := httpClient.Do(req)
					if respErr != nil {
						log.Printf("Failed to download avatar: %v", respErr)
						atomic.AddInt32(&errorCount, 1)
						return
					}
					if resp.StatusCode != http.StatusOK {
						log.Printf("Invalid response code downloading avatar: %d", resp.StatusCode)
						atomic.AddInt32(&errorCount, 1)
						return
					}
					body, bodyErr := io.ReadAll(resp.Body)
					if bodyErr != nil {
						log.Printf("Failed to read avatar body: %v", bodyErr)
						atomic.AddInt32(&errorCount, 1)
						return
					}
					defer logClose(resp.Body)

					if errSet := bd.cache.Set(cacheTypeAvatar, u.hash, bytes.NewReader(body)); errSet != nil {
						log.Printf("failed to set cached value: %v\n", errSet)
					}

					for _, player := range bd.players {
						if player.SteamId == u.sid {
							player.SetAvatar(u.hash, body)
							break
						}
					}
				}(update)
			}
			log.Printf("Downloaded %d avatars. [%d failed]", len(queuedUpdates), errorCount)
			queuedUpdates = nil
		}
	}
}

func (bd *BD) createLogReader() {
	consoleLogPath := filepath.Join(bd.settings.TF2Root, "console.log")
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
		case model.EvtDisconnect:
			// We don't really care about this, handled later via UpdatedOn timeout so that there is a
			// lag between actually removing the player from the player table.
			log.Printf("Player disconnected: %d", evt.PlayerSID.Int64())
		case model.EvtMsg:
			um := &model.UserMessage{
				Team:      evt.Team,
				Player:    evt.Player,
				PlayerSID: evt.PlayerSID,
				UserId:    evt.UserId,
				Message:   evt.Message,
				Created:   time.Now(),
			}
			bd.messages = append(bd.messages, um)
			var ps *model.PlayerState
			isNew := true
			for _, player := range bd.players {
				if player.SteamId == evt.PlayerSID {
					ps = player
					isNew = false
					break
				}
			}
			if isNew {
				newPs := model.NewPlayerState(evt.PlayerSID, evt.Player)
				if errCreate := bd.store.LoadOrCreatePlayer(bd.ctx, evt.PlayerSID, &newPs); errCreate != nil {
					log.Printf("Error trying to load/create player: %v\n", errCreate)
					continue
				}
			}

			if errSaveMsg := bd.store.SaveMessage(bd.ctx, ps.SteamId, evt.Message); errSaveMsg != nil {
				log.Printf("Error trying to store user messge log: %v\n", errSaveMsg)
			}

		case model.EvtStatusId:
			var ps *model.PlayerState
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
				newPs := model.NewPlayerState(evt.PlayerSID, evt.Player)
				if errCreate := bd.store.LoadOrCreatePlayer(bd.ctx, evt.PlayerSID, &newPs); errCreate != nil {
					log.Printf("Error trying to load/create player: %v\n", errCreate)
					continue
				}
				if evt.Player != "" && evt.Player != newPs.NamePrevious {
					if errSaveName := bd.store.SaveName(bd.ctx, evt.PlayerSID, evt.Player); errSaveName != nil {
						log.Printf("Failed to save name")
						continue
					}
				}
				newPs.UserId = evt.UserId
				ps = &newPs
			}
			ps.UpdatedOn = time.Now()
			ps.Ping = evt.PlayerPing
			ps.Connected = evt.PlayerConnected
			//log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			if isNew {
				ep = append(ep, ps)
			}
			bd.players = ep
			if isNew || time.Since(ps.UpdatedOn) > time.Hour {
				bd.profileUpdateQueue <- evt.PlayerSID
			}
			bd.triggerUpdate <- true
		}
	}
}

func (bd *BD) launchGameAndWait() {
	log.Println("Launching tf2...")
	hl2Path := filepath.Join(filepath.Dir(bd.settings.TF2Root), platform.BinaryName)
	args, errArgs := getLaunchArgs(
		bd.settings.Rcon.Password(),
		bd.settings.Rcon.Port(),
		bd.settings.SteamRoot,
		bd.settings.GetSteamId())
	if errArgs != nil {
		log.Println(errArgs)
		return
	}
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		log.Printf("Failed to launch TF2: %v\n", errStart)
		return
	}
	bd.gameProcess = process
	state, errWait := process.Wait()
	if errWait != nil {
		log.Printf("Error waiting for game process: %v\n", errWait)
	} else {
		log.Printf("Game exited: %s\n", state.String())
	}
	bd.gameProcess = nil
}

func (bd *BD) onMark(sid64 steamid.SID64, attrs []string) error {
	name := ""
	for _, player := range bd.players {
		if player.SteamId == sid64 {
			name = player.Name
			break
		}
	}
	if errMark := bd.rules.Mark(MarkOpts{
		steamId:    sid64,
		attributes: attrs,
		name:       name,
	}); errMark != nil {
		return errMark
	}
	of, errOf := os.OpenFile(bd.settings.LocalPlayerListPath(), os.O_RDWR, 0666)
	if errOf != nil {
		return errors.Wrapf(errOf, "Failed to open player list for updating")
	}
	defer logClose(of)
	if errExport := bd.rules.ExportPlayers(localRuleName, of); errExport != nil {
		return errors.Wrapf(errExport, "Failed to export player list")
	}
	return nil
}

func (bd *BD) AttachGui(gui ui.UserInterface) {
	gui.SetOnLaunchTF2(func() {
		go bd.launchGameAndWait()
	})
	gui.SetOnMark(bd.onMark)
	gui.SetOnKick(bd.callVote)
	gui.UpdateAttributes(bd.rules.UniqueTags())
	bd.gui = gui
}

func (bd *BD) playerStateUpdater() {
	for range time.NewTicker(time.Second * 10).C {
		//if bd.gameProcess == nil {
		//	continue
		//}
		updatePlayerState(bd.ctx, bd.settings.Rcon.String(), bd.settings.Rcon.Password())
		bd.checkPlayerStates()
	}
}

func (bd *BD) refreshLists() {
	playerLists, ruleLists := downloadLists(bd.ctx, bd.settings.Lists)
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

func (bd *BD) checkPlayerStates() {
	t0 := time.Now()

	var valid []*model.PlayerState
	for _, ps := range bd.players {
		if t0.Sub(ps.UpdatedOn) > time.Second*300 {
			log.Printf("Player expired: %s %s", ps.SteamId.String(), ps.Name)
			continue
		}
		valid = append(valid, ps)
	}

	for _, ps := range valid {
		match := bd.rules.matchSteam(ps.SteamId)
		if match != nil {
			if errLog := bd.partyLog("Player is a bot: (%d) [%s] %s ", ps.UserId, match.origin, ps.Name); errLog != nil {
				log.Printf("Failed to send party log message: %s\n", errLog)
				continue
			}
			log.Println("Matched player...")
		}
		if ps.Name != "" {
			match = bd.rules.matchName(ps.Name)
			if match != nil {
				log.Println("Matched name...")
			}
		}
		//for _, matcher := range bd.rules {
		//	if !matcher.FindMatch(ps.SteamId, &matched) {
		//		continue
		//	}
		//	if bd.dryRun {
		//		if errPL := bd.partyLog("(DRY) Matched player: %s %s %s",
		//			matched.player.SteamId,
		//			strings.Join(matched.player.Attributes, ","),
		//			matched.list.FileInfo.Description,
		//		); errPL != nil {
		//			log.Println(errPL)
		//			continue
		//		}
		//	} else {
		//		if errVote := bd.callVote(ps.UserId); errVote != nil {
		//			log.Printf("Error calling vote: %v", errVote)
		//		}
		//		ps.KickAttemptCount++
		//	}
		//	// Only try to vote once per iteration
		//	break
		//
		//}
	}
	var plState []model.PlayerState
	for _, player := range valid {
		plState = append(plState, *player)
	}
	bd.players = valid
	bd.gui.UpdatePlayerState(plState)

}

func (bd *BD) connectRcon() error {
	if bd.rconConnection != nil {
		logClose(bd.rconConnection)
	}
	conn, errConn := rcon.Dial(bd.ctx, bd.settings.Rcon.String(), bd.settings.Rcon.Password(), time.Second*5)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to client: %v\n", errConn)
	}
	bd.rconConnection = conn
	return nil
}

func (bd *BD) partyLog(fmtStr string, args ...any) error {
	if errConn := bd.connectRcon(); errConn != nil {
		return errConn
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("say_party %s", fmt.Sprintf(fmtStr, args...)))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon say_party")
	}
	return nil
}

func (bd *BD) callVote(userId int64) error {
	if errConn := bd.connectRcon(); errConn != nil {
		return errConn
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("callvote kick %d", userId))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}
	return nil
}

func (bd *BD) Shutdown() {
	if bd.rconConnection != nil {
		logClose(bd.rconConnection)
	}
	// Ensure we save on exit
	playerListFile, playerListFileErr := os.Create(bd.settings.LocalPlayerListPath())
	if playerListFileErr != nil {
		log.Panicf("Failed to open player list for writing: %v\n", playerListFileErr)
	}
	if errWrite := bd.rules.ExportPlayers(localRuleName, playerListFile); errWrite != nil {
		log.Panicf("Failed to export player list: %v\n", playerListFileErr)
	}

	rulesFile, rulesFileErr := os.Create(bd.settings.LocalRulesListPath())
	if rulesFileErr != nil {
		log.Panicf("Failed to open player list for writing: %v\n", rulesFileErr)
	}
	if errWrite := bd.rules.ExportRules(localRuleName, rulesFile); errWrite != nil {
		log.Panicf("Failed to export rules list: %v\n", rulesFileErr)
	}
	bd.store.Close()
}

func (bd *BD) start() {
	go bd.logReader.start(bd.ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(bd.ctx, bd.players)
	go bd.playerStateUpdater()
	go bd.refreshLists()
	go bd.eventHandler()
	go bd.profileUpdater(time.Second * 10)
	go bd.uiStateUpdater()
	go bd.discordPresence()
	<-bd.ctx.Done()
}
