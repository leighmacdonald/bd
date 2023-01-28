package main

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type rconConfig struct {
	address  string
	password string
	port     uint16
}

func (cfg rconConfig) Addr() string {
	return fmt.Sprintf("%s:%d", cfg.address, cfg.port)
}

type BD struct {
	// TODO
	// - decouple remaining ui bindings
	// - generalized matchers
	messages       binding.StringList
	logChan        chan string
	serverState    *serverState
	serverStateMu  *sync.RWMutex
	ctx            context.Context
	logReader      *logReader
	logParser      *LogParser
	matchers       []Matcher
	playerLists    playerListCollection
	playerListsMu  *sync.RWMutex
	ruleLists      ruleListCollection
	ruleListsMu    *sync.RWMutex
	rconConfig     rconConfig
	rconConnection rconConnection
	ui             *Ui
	eventChan      chan event
	dryRun         bool
}

func New() BD {
	rootApp := BD{
		logChan:       make(chan string),
		messages:      binding.NewStringList(),
		playerListsMu: &sync.RWMutex{},
		serverStateMu: &sync.RWMutex{},
		ruleListsMu:   &sync.RWMutex{},
		rconConfig:    rconConfig{address: "127.0.0.1", password: golib.RandomString(10), port: randPort()},
		serverState: &serverState{
			players: map[steamid.SID64]*playerState{},
		},
		ctx:    context.Background(),
		dryRun: true,
	}
	tf2Path, tf2PathErr := getTF2Folder()
	if tf2PathErr != nil {
		panic(tf2PathErr)
	}
	consoleLogPath := filepath.Join(tf2Path, "console.log")
	reader, errLogReader := newLogReader(consoleLogPath, rootApp.logChan, true)
	if errLogReader != nil {
		panic(errLogReader)
	}
	rootApp.logReader = reader
	incomingLogEvents := make(chan LogEvent)

	rootApp.logParser = NewLogParser(rootApp.logChan, incomingLogEvents)
	go func() {
		for {
			evt := <-incomingLogEvents
			switch evt.Type {
			case EvtDisconnect:
				delete(rootApp.serverState.players, evt.PlayerSID)
				log.Printf("Removed disconnected player state: %d", evt.PlayerSID.Int64())
			case EvtMsg:
				rootApp.eventChan <- event{EvtMsg, evtUserMessageData{
					team:      evt.Team,
					player:    evt.Player,
					playerSID: evt.PlayerSID,
					userId:    evt.UserId,
					message:   evt.Message,
				}}
			case EvtStatusId:
				log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			}
		}
	}()

	rootApp.ui = newUi(rootApp.serverState)

	return rootApp
}
func (bd *BD) playerStateUpdater() {
	for range time.NewTicker(time.Second * 10).C {
		updatePlayerState(bd.ctx, bd.rconConfig.Addr(), bd.rconConfig.password, bd.serverState)
		for _, v := range bd.serverState.players {
			log.Printf("%d, %d, %s, %v, %s\n", v.userId, v.steamId, v.name, v.team, v.connectedTime)
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
	for steamId, ps := range bd.serverState.players {
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
				if errVote := bd.callVote(ps.userId); errVote != nil {
					log.Printf("Error calling vote: %v", errVote)
				}
				ps.kickAttemptCount++
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
	bd.ui.RootWindow.Show()

	go eventSender(bd.ctx, bd, bd.ui)

	bd.ui.Application.Run()
}
