package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type BD struct {
	// TODO
	// - decouple remaining ui bindings
	// - generalized matchers

	logChan           chan string
	incomingLogEvents chan model.LogEvent
	serverState       model.ServerState
	serverStateMu     *sync.RWMutex
	ctx               context.Context
	logReader         *logReader
	logParser         *LogParser
	matchers          []Matcher
	playerLists       playerListCollection
	playerListsMu     *sync.RWMutex
	ruleLists         ruleListCollection
	ruleListsMu       *sync.RWMutex
	rconConfig        rconConfigProvider
	rconConnection    rconConnection
	gui               ui.UserInterface
	dryRun            bool
}

func New(ctx context.Context, rconConfig rconConfigProvider) BD {
	rootApp := BD{
		logChan:           make(chan string),
		incomingLogEvents: make(chan model.LogEvent),
		playerListsMu:     &sync.RWMutex{},
		serverStateMu:     &sync.RWMutex{},
		ruleListsMu:       &sync.RWMutex{},
		rconConfig:        rconConfig,
		serverState: model.ServerState{
			Players: []model.PlayerState{},
		},
		ctx:    ctx,
		dryRun: true,
	}
	tf2Path, _ := getTF2Folder()
	//if tf2PathErr != nil {
	//	panic(tf2PathErr)
	//}
	consoleLogPath := filepath.Join(tf2Path, "console.log")
	reader, errLogReader := newLogReader(consoleLogPath, rootApp.logChan, true)
	if errLogReader != nil {
		panic(errLogReader)
	}
	rootApp.logReader = reader

	rootApp.logParser = NewLogParser(rootApp.logChan, rootApp.incomingLogEvents)

	return rootApp
}

func (bd *BD) eventHandler() {
	for {
		evt := <-bd.incomingLogEvents
		switch evt.Type {
		case model.EvtDisconnect:
			go func(lc context.Context, sid steamid.SID64) {
				// wait for timeout until player removed from list
				to, cancel := context.WithTimeout(lc, time.Second*120)
				defer cancel()
				<-to.Done()
				var newPlayers []model.PlayerState
				bd.serverStateMu.Lock()
				for _, pl := range bd.serverState.Players {
					if pl.SteamId != sid {
						newPlayers = append(newPlayers, pl)
					}
				}
				bd.serverState.Players = newPlayers
				bd.serverStateMu.Unlock()
			}(bd.ctx, evt.PlayerSID)
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
			bd.serverStateMu.Lock()
			newPlayer := true
			ep := bd.serverState.Players
			for _, existingPlayer := range ep {
				if existingPlayer.SteamId == evt.PlayerSID {
					newPlayer = false
					break
				}
			}
			if newPlayer {
				ep = append(ep, model.PlayerState{
					Name:        evt.Player,
					SteamId:     evt.PlayerSID,
					ConnectedAt: time.Now(),
					Team:        0,
					UserId:      evt.UserId,
				})
			}
			bd.serverState.Players = ep
			log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			bd.serverStateMu.Unlock()
		}
	}
}

func (bd *BD) AttachGui(gui ui.UserInterface) {
	fn := func() {
		launchTF2(bd.rconConfig.Password(), bd.rconConfig.Port())
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
			bd.serverStateMu.RLock()
			sc := bd.serverState
			bd.serverStateMu.RUnlock()
			bd.gui.OnServerState(sc)
		}
	}

}

func (bd *BD) playerStateUpdater() {
	for range time.NewTicker(time.Second * 10).C {
		updatePlayerState(bd.ctx, bd.rconConfig.String(), bd.rconConfig.Password())
		bd.checkPlayerStates()
	}
}

func (bd *BD) listUpdater() {
	var update = func() {
		initList := downloadPlayerLists(bd.ctx)
		bd.playerListsMu.Lock()
		defer bd.playerListsMu.Unlock()
		bd.playerLists = initList
	}
	// Ensure ran once at launch
	update()
	tick := time.NewTicker(1 * time.Hour)
	for range tick.C {
		update()
	}
}

func (bd *BD) checkPlayerStates() {
	var matched MatchedPlayerList
	for _, ps := range bd.serverState.Players {
		for _, matcher := range bd.matchers {
			if !matcher.FindMatch(ps.SteamId, &matched) {
				continue
			}
			if bd.dryRun {
				if errPL := bd.partyLog("(DRY) Matched player: %s %s %s",
					matched.player.SteamId,
					strings.Join(matched.player.Attributes, ","),
					matched.list.FileInfo.Description,
				); errPL != nil {
					log.Println(errPL)
					continue
				}
			} else {
				if errVote := bd.callVote(ps.UserId); errVote != nil {
					log.Printf("Error calling vote: %v", errVote)
				}
				ps.KickAttemptCount++
			}
			// Only try to vote once per iteration
			break

		}
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
	go bd.listUpdater()
	go bd.uiStateUpdater()
	go bd.eventHandler()
	//go ui2.eventSender(bd.ctx, bd, bd.ui)
	<-bd.ctx.Done()
}
