package main

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"log"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
				log.Printf("Failed to close tail: %v\n", errStop)
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

type EventType int

const (
	EvtKill EventType = iota
	EvtMsg
	EvtConnect
	EvtDisconnect
	EvtStatusId
	EvtLobbyPlayerTeam
)

type LogEvent struct {
	Type      EventType
	Player    string
	UserId    int64
	PlayerSID steamid.SID64
	Victim    string
	VictimSID steamid.SID64
	Message   string
	Team      team
}

type LogParser struct {
	evtChan       chan LogEvent
	ReadChannel   chan string
	rxLobbyPlayer *regexp.Regexp
	rx            []*regexp.Regexp
}

func (l *LogParser) ParseEvent(msg string, outEvent *LogEvent) error {
	// the index must match the index of the EventType const values
	for i, rxMatcher := range l.rx {
		if m := rxMatcher.FindStringSubmatch(msg); m != nil {
			t := EventType(i)
			outEvent.Type = t
			switch t {
			case EvtLobbyPlayerTeam:
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(m[1]))
				if m[2] == "DEFENDER" {
					outEvent.Team = red
				} else {
					outEvent.Team = blu
				}
			case EvtConnect:
				outEvent.Player = m[1]
			case EvtDisconnect:
				outEvent.Player = m[1]
			case EvtMsg:
				outEvent.Player = m[1]
				outEvent.Message = m[2]
			case EvtStatusId:
				userId, errUserId := strconv.ParseInt(m[2], 10, 32)
				if errUserId != nil {
					log.Printf("Failed to parse userid: %v", errUserId)
					continue
				}
				outEvent.UserId = userId
				outEvent.Player = m[3]
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(m[4]))
			case EvtKill:
				outEvent.Player = m[1]
				outEvent.Victim = m[2]
			}
			return nil
		}
	}
	return errNoMatch
}

func (l *LogParser) start(ctx context.Context) {
	for {
		select {
		case msg := <-l.ReadChannel:
			var logEvent LogEvent
			if err := l.ParseEvent(msg, &logEvent); err != nil || errors.Is(err, errNoMatch) {
				continue
			}
			l.evtChan <- logEvent
		case <-ctx.Done():
			return
		}
	}
}

func NewLogParser(readChannel chan string, evtChan chan LogEvent) *LogParser {
	lp := LogParser{
		evtChan:       evtChan,
		ReadChannel:   readChannel,
		rxLobbyPlayer: regexp.MustCompile(`\s+(Member|Pending)\[\d+]\s+(?P<sid>\[.+?]).+?TF_GC_TEAM_(?P<team>(DEFENDERS|INVADERS))`),
		rx: []*regexp.Regexp{
			regexp.MustCompile(`^(.+?)\skilled\s(.+?)\swith\s(.+)(\.|\. \(crit\))$`),
			regexp.MustCompile(`^(.+?)\s:\s\s(.+?)$`),
			regexp.MustCompile(`(?:.+?\.)?(\S+)\sconnected$`),
			regexp.MustCompile(`(^Disconnecting from abandoned match server$|\(Server shutting down\)$)`),
			regexp.MustCompile(`^(.+?)#\s+(?P<id>\d+)\s"(?P<name>.+?)"\s+(?P<sid>\[U:\d:\d+])\s+(?P<time>\d+:\d+)\s+(?P<ping>\d+)\s+(?P<loss>\d+)`)},
	}
	return &lp
}
