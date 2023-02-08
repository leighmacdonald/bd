package main

import (
	"context"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"log"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type logReader struct {
	tail    *tail.Tail
	outChan chan string
}

func (reader *logReader) start(ctx context.Context) {
	for {
		select {
		case msg := <-reader.tail.Lines:
			reader.outChan <- strings.TrimSuffix(msg.Text, "\r")
		case <-ctx.Done():
			if errStop := reader.tail.Stop(); errStop != nil {
				log.Printf("Failed to Close tail: %v\n", errStop)
			}
			return
		}
	}
}

func newLogReader(path string, outChan chan string, echo bool) (*logReader, error) {
	logger := tail.DiscardingLogger
	if echo {
		logger = tail.DefaultLogger
	}
	//goland:noinspection GoBoolExpressions
	tailConfig := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      runtime.GOOS == "windows",
		Logger:    logger,
	}
	//goland:noinspection ALL
	t, errTail := tail.TailFile(path, tailConfig)
	if errTail != nil {
		return nil, errors.Wrap(errTail, "Failed to configure tail")
	}
	reader := logReader{
		tail:    t,
		outChan: outChan,
	}
	return &reader, nil
}

var (
	errNoMatch          = errors.New("no match found")
	errRconDisconnected = errors.New("rcon not connected")
)

// parseTimestamp will convert the source formatted log timestamps into a time.Time value
func parseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse("01/02/2006 - 15:04:05", timestamp)
}

type LogParser struct {
	evtChan       chan model.LogEvent
	ReadChannel   chan string
	rxLobbyPlayer *regexp.Regexp
	rx            []*regexp.Regexp
}

func (l *LogParser) ParseEvent(msg string, outEvent *model.LogEvent) error {
	// the index must match the index of the EventType const values
	for i, rxMatcher := range l.rx {
		if m := rxMatcher.FindStringSubmatch(msg); m != nil {
			t := model.EventType(i)
			outEvent.Type = t
			switch t {
			case model.EvtLobbyPlayerTeam:
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(m[1]))
				if m[2] == "DEFENDERS" {
					outEvent.Team = model.Red
				} else {
					outEvent.Team = model.Blu
				}
			case model.EvtConnect:
				outEvent.Player = m[1]
			case model.EvtDisconnect:
				outEvent.Player = m[1]
			case model.EvtMsg:
				ts, errTs := parseTimestamp(m[1])
				if errTs != nil {
					log.Printf("Failed to parse timestamp for message log: %s", errTs)
					continue
				}
				outEvent.Timestamp = ts
				outEvent.Player = m[2]
				outEvent.Message = m[3]
			case model.EvtStatusId:
				ts, errTs := parseTimestamp(m[1])
				if errTs != nil {
					log.Printf("Failed to parse timestamp for message log: %s", errTs)
					continue
				}
				outEvent.Timestamp = ts
				userId, errUserId := strconv.ParseInt(m[2], 10, 32)
				if errUserId != nil {
					log.Printf("Failed to parse userid: %v", errUserId)
					continue
				}
				ping, errPing := strconv.ParseInt(m[6], 10, 32)
				if errPing != nil {
					log.Printf("Failed to parse ping: %v", errUserId)
					continue
				}
				outEvent.UserId = userId
				outEvent.Player = m[3]
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(m[4]))
				outEvent.PlayerConnected = m[5]
				outEvent.PlayerPing = int(ping)
			case model.EvtKill:
				outEvent.Player = m[1]
				outEvent.Victim = m[2]
			}
			return nil
		}
	}
	return errNoMatch
}

// TODO why keep this?
func (l *LogParser) start(ctx context.Context, gs *model.GameState) {
	for {
		select {
		case msg := <-l.ReadChannel:
			var logEvent model.LogEvent
			if err := l.ParseEvent(msg, &logEvent); err != nil || errors.Is(err, errNoMatch) {
				continue
			}

			if logEvent.Type == model.EvtMsg {
				gs.RLock()
				for _, p := range gs.Players {
					if p.Name == logEvent.Player {
						logEvent.PlayerSID = p.SteamId
						break
					}
				}
				gs.RUnlock()
				if logEvent.PlayerSID == 0 {
					// We don't know the player yet.
					continue
				}
			}

			l.evtChan <- logEvent
		case <-ctx.Done():
			return
		}
	}
}

func NewLogParser(readChannel chan string, evtChan chan model.LogEvent) *LogParser {
	lp := LogParser{
		evtChan:       evtChan,
		ReadChannel:   readChannel,
		rxLobbyPlayer: regexp.MustCompile(`\s+(Member|Pending)\[\d+]\s+(?P<sid>\[.+?]).+?TF_GC_TEAM_(?P<team>(DEFENDERS|INVADERS))`),
		rx: []*regexp.Regexp{
			regexp.MustCompile(`^(.+?)\skilled\s(.+?)\swith\s(.+)(\.|\. \(crit\))$`),
			regexp.MustCompile(`^(?P<dt>\d{2}/\d{2}/\d{4}\s-\s\d{2}:\d{2}:\d{2}):\s(?P<name>.+?)\s:\s{2}(?P<message>.+?)$`),
			regexp.MustCompile(`(?:.+?\.)?(\S+)\sconnected$`),
			regexp.MustCompile(`(^Disconnecting from abandoned match server$|\(Server shutting down\)$)`),
			regexp.MustCompile(`(?P<dt>^[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s#\s{1,6}(?P<id>\d{1,6})\s"(?P<name>.+?)"\s+(?P<sid>\[U:\d:\d{1,10}])\s{1,8}(?P<time>\d{2,3}:\d{2})\s+(?P<ping>\d{1,4})\s{1,8}(?P<loss>\d{1,3})\s(spawning|active)$`)},
	}
	return &lp
}
