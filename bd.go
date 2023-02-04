package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/ui"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"time"
)

type BD struct {
	// TODO
	// - decouple remaining ui bindings
	// - generalized matchers
	// - estimate private steam account ages (find nearby non-private account)
	logChan           chan string
	incomingLogEvents chan model.LogEvent
	serverState       *model.ServerState
	ctx               context.Context
	logReader         *logReader
	logParser         *LogParser
	rules             *RulesEngine
	rconConnection    rconConnection
	settings          *model.Settings
	store             dataStore
	gui               ui.UserInterface
	dryRun            bool
	gameActive        bool
}

func New(ctx context.Context, settings *model.Settings, store dataStore) BD {
	rootApp := BD{
		settings:          settings,
		logChan:           make(chan string),
		incomingLogEvents: make(chan model.LogEvent),
		rules:             newRulesEngine(),
		serverState:       model.NewServerState(),
		store:             store,
		ctx:               ctx,
	}

	rootApp.createLogReader()
	rootApp.createLogParser()

	return rootApp
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
			bd.serverState.Lock()
			for _, ps := range bd.serverState.Players {
				if ps.SteamId == evt.PlayerSID {
					dt := time.Now()
					ps.DisconnectedAt = &dt
				}
			}
			bd.serverState.Unlock()
			go func() {
				time.Sleep(60 * time.Second)
				bd.serverState.Lock()
				var newState []*model.PlayerState
				for _, ps := range bd.serverState.Players {
					if ps.SteamId != evt.PlayerSID {
						newState = append(newState, ps)
					}
				}
				bd.serverState.Players = newState
				bd.serverState.Unlock()
			}()
			bd.gui.OnDisconnect(evt.PlayerSID)
			log.Printf("Player disconnected: %d", evt.PlayerSID.Int64())
		case model.EvtMsg:
			bd.gui.OnUserMessage(model.EvtUserMessage{
				Team:      evt.Team,
				Player:    evt.Player,
				PlayerSID: evt.PlayerSID,
				UserId:    evt.UserId,
				Message:   evt.Message,
			})
		case model.EvtStatusId:
			bd.serverState.Lock()
			newPlayer := true
			ep := bd.serverState.Players
			for _, existingPlayer := range ep {
				if existingPlayer.SteamId == evt.PlayerSID {
					newPlayer = false
					break
				}
			}
			if newPlayer {
				np := model.NewPlayerState(evt.PlayerSID, evt.Player)
				np.UserId = evt.UserId
				go np.Update()
				ep = append(ep, &np)
			}
			bd.serverState.Players = ep
			log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			bd.serverState.Unlock()
		}
	}
}

func (bd *BD) AttachGui(gui ui.UserInterface) {
	fn := func() {
		launchTF2(bd.settings.Rcon.Password(), bd.settings.Rcon.Port(), bd.settings.TF2Root, bd.settings.SteamRoot, bd.settings.GetSteamId())
		bd.gameActive = true
	}
	gui.OnLaunchTF2(fn)
	bd.gui = gui
}

func (bd *BD) uiStateUpdater() {
	updateTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-bd.ctx.Done():
			return
		case <-updateTicker.C:
			if bd.gui == nil {
				return
			}
			bd.serverState.RLock()
			sc := bd.serverState
			bd.serverState.RUnlock()
			bd.gui.OnServerState(sc)
		}
	}

}

func (bd *BD) playerStateUpdater() {
	for range time.NewTicker(time.Second * 10).C {
		if !bd.gameActive {
			continue
		}
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
	//var matched MatchedPlayerList
	for _, ps := range bd.serverState.Players {
		if bd.rules.matchSteam(ps.SteamId) {
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
	go bd.logParser.start(bd.ctx)
	go bd.playerStateUpdater()
	go bd.refreshLists()
	go bd.uiStateUpdater()
	go bd.eventHandler()
	//go ui2.eventSender(bd.ctx, bd, bd.ui)
	<-bd.ctx.Done()
}
