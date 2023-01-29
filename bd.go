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

	logChan        chan string
	serverState    *model.ServerState
	serverStateMu  *sync.RWMutex
	ctx            context.Context
	logReader      *logReader
	logParser      *LogParser
	matchers       []Matcher
	playerLists    playerListCollection
	playerListsMu  *sync.RWMutex
	ruleLists      ruleListCollection
	ruleListsMu    *sync.RWMutex
	rconConfig     rconConfigProvider
	rconConnection rconConnection
	gui            ui.UserInterface
	eventChan      chan model.Event
	dryRun         bool
}

func New(ctx context.Context, rconConfig rconConfigProvider) BD {
	rootApp := BD{
		logChan:       make(chan string),
		playerListsMu: &sync.RWMutex{},
		serverStateMu: &sync.RWMutex{},
		ruleListsMu:   &sync.RWMutex{},
		rconConfig:    rconConfig,
		serverState: &model.ServerState{
			Players: map[steamid.SID64]*model.PlayerState{},
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
	incomingLogEvents := make(chan model.LogEvent)

	rootApp.logParser = NewLogParser(rootApp.logChan, incomingLogEvents)
	go func() {
		for {
			evt := <-incomingLogEvents
			switch evt.Type {
			case model.EvtDisconnect:
				delete(rootApp.serverState.Players, evt.PlayerSID)
				log.Printf("Removed disconnected player state: %d", evt.PlayerSID.Int64())
			case model.EvtMsg:
				rootApp.eventChan <- model.Event{Name: model.EvtMsg, Value: model.EvtUserMessage{
					Team:      evt.Team,
					Player:    evt.Player,
					PlayerSID: evt.PlayerSID,
					UserId:    evt.UserId,
					Message:   evt.Message,
				}}
			case model.EvtStatusId:
				log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			}
		}
	}()

	return rootApp
}

func (bd *BD) AttachGui(gui ui.UserInterface) {
	bd.gui = gui
}

func (bd *BD) playerStateUpdater() {
	for range time.NewTicker(time.Second * 10).C {
		updatePlayerState(bd.ctx, bd.rconConfig.String(), bd.rconConfig.Password(), bd.serverState)
		for _, v := range bd.serverState.Players {
			log.Printf("%d, %d, %s, %v, %s\n", v.UserId, v.SteamId, v.Name, v.Team, v.ConnectedTime)
		}
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
	for steamId, ps := range bd.serverState.Players {
		for _, matcher := range bd.matchers {
			if !matcher.FindMatch(steamId, &matched) {
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

func eventSender(ctx context.Context, bd *BD, ui ui.UserInterface) {
	for {
		select {
		case evt := <-bd.eventChan:
			switch evt.Name {
			case model.EvtMsg:
				value := evt.Value.(model.EvtUserMessage)
				ui.OnUserMessage(value)
			}
		case <-ctx.Done():
			return
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

func (bd *BD) callVote(userId int) error {
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
	//go ui2.eventSender(bd.ctx, bd, bd.ui)
	<-bd.ctx.Done()
}
