package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/ui"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"log"
	"os"
	"path/filepath"
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

func New(ctx context.Context, settings *model.Settings, store dataStore) BD {
	rootApp := BD{
		settings:           settings,
		logChan:            make(chan string),
		incomingLogEvents:  make(chan model.LogEvent),
		gameState:          model.NewGameState(),
		rules:              newRulesEngine(),
		profileUpdateQueue: make(chan steamid.SID64),
		store:              store,
		ctx:                ctx,
	}

	rootApp.createLogReader()
	rootApp.createLogParser()

	return rootApp
}

// profileUpdater handles fetching 3rd party data of players
// MAYBE priority queue for new players in a already established game?
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
			bd.gameState.Unlock()
			log.Printf("Updated %d profiles\n", len(queuedUpdates))
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
				newPlayer := model.NewPlayerState(evt.PlayerSID, evt.Player)
				newPlayer.UserId = evt.UserId
				ep = append(ep, &newPlayer)
				ps = &newPlayer
			}
			ps.UpdatedOn = time.Now()
			ps.Ping = evt.PlayerPing
			ps.Connected = evt.PlayerConnected
			log.Printf("[%d] [%d] %s\n", evt.UserId, evt.PlayerSID.Int64(), evt.Player)
			if isNew {
				bd.gameState.Players = append(bd.gameState.Players, ps)
				bd.profileUpdateQueue <- evt.PlayerSID

			}
			bd.gameState.Unlock()
		}
	}
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
	defer bd.gameState.Unlock()
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
	bd.gameState.Players = valid

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
	go bd.eventHandler()
	go bd.profileUpdater(time.Second * 10)
	//go ui2.eventSender(bd.ctx, bd, bd.ui)
	<-bd.ctx.Done()
}
