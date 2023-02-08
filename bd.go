package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/ui"
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
	// - decouple remaining ui bindings
	// - generalized matchers
	// - estimate private steam account ages (find nearby non-private account)
	logChan            chan string
	incomingLogEvents  chan model.LogEvent
	gameState          *model.GameState
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
}

func New(ctx context.Context, settings *model.Settings, store dataStore, gameState *model.GameState, rules *RulesEngine) BD {
	rootApp := BD{
		ctx:                ctx,
		store:              store,
		gameState:          gameState,
		rules:              rules,
		settings:           settings,
		logChan:            make(chan string),
		incomingLogEvents:  make(chan model.LogEvent),
		profileUpdateQueue: make(chan steamid.SID64),
	}

	rootApp.createLogReader()
	rootApp.createLogParser()

	return rootApp
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
			if len(queuedUpdates) == 0 {
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
			bd.gameState.Lock()
			for _, player := range bd.gameState.Players {
				for _, summary := range summaries {
					if summary.Steamid == player.SteamId.String() {
						player.Summary = &summary
						break
					}
				}
				for _, ban := range bans {
					if ban.SteamID == player.SteamId.String() {
						player.BanState = &ban
						break
					}
				}
			}
			type avatarUpdate struct {
				url string
				sid steamid.SID64
			}
			var avatarUpdates []avatarUpdate
			for _, p := range bd.gameState.Players {
				if p.Summary == nil {
					continue
				}
				avatarUpdates = append(avatarUpdates, avatarUpdate{
					url: p.Summary.AvatarFull,
					sid: p.SteamId,
				})
			}
			bd.gameState.Unlock()
			log.Printf("Updated %d profiles\n", len(queuedUpdates))
			client := &http.Client{}
			wg := &sync.WaitGroup{}
			var errorCount int32 = 0
			for _, update := range avatarUpdates {
				wg.Add(1)
				go func(u avatarUpdate) {
					defer wg.Done()
					localCtx, cancel := context.WithTimeout(bd.ctx, time.Second*10)
					defer cancel()
					req, reqErr := http.NewRequestWithContext(localCtx, "GET", u.url, nil)
					if reqErr != nil {
						log.Printf("Failed to create avatar download request: %v", reqErr)
						atomic.AddInt32(&errorCount, 1)
						return
					}
					resp, respErr := client.Do(req)
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
					defer func() {
						if errClose := resp.Body.Close(); errClose != nil {
							log.Printf("Failed to close response body: %v", errClose)
						}
					}()
					bd.gameState.Lock()
					for _, player := range bd.gameState.Players {
						player.Avatar = body
						player.AvatarHash = hashBytes(body)
						break
					}
					bd.gameState.Unlock()
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

func (bd *BD) createLogParser() {
	bd.logParser = NewLogParser(bd.logChan, bd.incomingLogEvents)
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
			bd.gameState.Lock()
			bd.gameState.Messages = append(bd.gameState.Messages, model.UserMessage{
				Team:      evt.Team,
				Player:    evt.Player,
				PlayerSID: evt.PlayerSID,
				UserId:    evt.UserId,
				Message:   evt.Message,
				Created:   time.Now(),
			})
			bd.gameState.Unlock()
			bd.gui.Refresh()
			newPs := model.NewPlayerState(evt.PlayerSID, evt.Player)
			if errCreate := bd.store.LoadOrCreatePlayer(bd.ctx, evt.PlayerSID, &newPs); errCreate != nil {
				log.Printf("Error trying to load/create player: %v\n", errCreate)
				bd.gameState.Unlock()
				continue
			}
			if errSaveMsg := bd.store.SaveMessage(bd.ctx, newPs.SteamId, evt.Message); errSaveMsg != nil {
				log.Printf("Error trying to store user messge log: %v\n", errSaveMsg)
			}
		case model.EvtStatusId:
			var ps *model.PlayerState
			bd.gameState.Lock()
			ep := bd.gameState.Players
			for _, existingPlayer := range ep {
				if existingPlayer.SteamId == evt.PlayerSID {
					ps = existingPlayer
					break
				}
			}
			isNew := ps == nil
			if isNew {
				newPs := model.NewPlayerState(evt.PlayerSID, evt.Player)
				if errCreate := bd.store.LoadOrCreatePlayer(bd.ctx, evt.PlayerSID, &newPs); errCreate != nil {
					log.Printf("Error trying to load/create player: %v\n", errCreate)
					bd.gameState.Unlock()
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
				ep = append(ep, ps)
			}
			ps.UpdatedOn = time.Now()
			ps.Ping = evt.PlayerPing
			ps.Connected = evt.PlayerConnected
			log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			if isNew {
				bd.gameState.Players = append(bd.gameState.Players, ps)
			}
			bd.gameState.Unlock()
			bd.profileUpdateQueue <- evt.PlayerSID
			bd.gui.Refresh()
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
		log.Printf("Failed to launch TF2: %v", errStart)
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

func (bd *BD) AttachGui(gui ui.UserInterface) {
	gui.OnLaunchTF2(func() {
		go bd.launchGameAndWait()
	})
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
		if errImport := bd.rules.ImportPlayers(list); errImport != nil {
			log.Printf("Failed to import player list (%s): %v\n", list.FileInfo.Title, errImport)
		}
	}
	for _, list := range ruleLists {
		if errImport := bd.rules.ImportRules(list); errImport != nil {
			log.Printf("Failed to import rules list (%s): %v\n", list.FileInfo.Title, errImport)
		}
	}
}

func (bd *BD) checkPlayerStates() {
	t0 := time.Now()
	bd.gameState.Lock()
	var valid []*model.PlayerState
	for _, ps := range bd.gameState.Players {
		if t0.Sub(ps.UpdatedOn) > time.Second*60 {
			log.Printf("Player expired: %s %s", ps.SteamId.String(), ps.Name)
			continue
		}
		valid = append(valid, ps)
	}

	for _, ps := range valid {
		if t0.Sub(ps.UpdatedOn) > time.Second*60 {

		}
		match := bd.rules.matchSteam(ps.SteamId)
		if match != nil {
			log.Println("Matched player...")
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
	bd.gameState.Players = valid
	bd.gameState.Unlock()
	if bd.gui != nil {
		go bd.gui.Refresh()
	}
}

func (bd *BD) partyLog(fmtStr string, args ...any) error {
	if bd.rconConnection == nil {
		return errRconDisconnected
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("say_party %s", fmt.Sprintf(fmtStr, args...)))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon say_party")
	}
	return nil
}

func (bd *BD) callVote(userId int64) error {
	if bd.rconConnection == nil {
		return errRconDisconnected
	}
	_, errExec := bd.rconConnection.Exec(fmt.Sprintf("callvote kick %d", userId))
	if errExec != nil {
		return errors.Wrap(errExec, "Failed to send rcon callvote")
	}
	return nil
}

func (bd *BD) start() {
	go bd.logReader.start(bd.ctx)
	defer bd.logReader.tail.Cleanup()
	go bd.logParser.start(bd.ctx, bd.gameState)
	go bd.playerStateUpdater()
	go bd.refreshLists()
	go bd.eventHandler()
	go bd.profileUpdater(time.Second * 10)
	<-bd.ctx.Done()
}
