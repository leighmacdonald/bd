package detector

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type logReader struct {
	tail    *tail.Tail
	outChan chan string
	logger  *zap.Logger
}

func (reader *logReader) start(ctx context.Context) {
	for {
		select {
		case msg := <-reader.tail.Lines:
			if msg == nil {
				// Happens on linux only?
				continue
			}
			reader.outChan <- strings.TrimSuffix(msg.Text, "\r")

		case <-ctx.Done():
			if errStop := reader.tail.Stop(); errStop != nil {
				reader.logger.Error("Failed to stop tailing console.log cleanly", zap.Error(errStop))
			}
			return
		}
	}
}

func newLogReader(logger *zap.Logger, path string, outChan chan string, echo bool) (*logReader, error) {
	tailLogger := tail.DiscardingLogger
	if echo {
		tailLogger = tail.DefaultLogger
	}
	//goland:noinspection GoBoolExpressions
	tailConfig := tail.Config{
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      runtime.GOOS == "windows", // TODO Is this still required years later?
		Logger:    tailLogger,
	}
	//goland:noinspection ALL
	t, errTail := tail.TailFile(path, tailConfig)
	if errTail != nil {
		return nil, errors.Wrap(errTail, "Failed to configure tail")
	}
	reader := logReader{
		tail:    t,
		outChan: outChan,
		logger:  logger,
	}
	return &reader, nil
}

var (
	errNoMatch = errors.New("no match found")
)

type logParser struct {
	evtChan     chan model.LogEvent
	ReadChannel chan string
	rx          []*regexp.Regexp
	logger      *zap.Logger
}

const teamPrefix = "(TEAM) "
const deadPrefix = "*DEAD* "
const deadTeamPrefix = "*DEAD*(TEAM) "

func (parser *logParser) parseEvent(msg string, outEvent *model.LogEvent) error {
	// the index must match the index of the EventType const values
	for i, rxMatcher := range parser.rx {
		if match := rxMatcher.FindStringSubmatch(msg); match != nil {
			outEvent.Type = model.EventType(i)
			if outEvent.Type != model.EvtLobby {
				if errTs := outEvent.ApplyTimestamp(match[1]); errTs != nil {
					parser.logger.Error("Failed to parse timestamp", zap.Error(errTs))
				}
			}
			switch outEvent.Type {
			case model.EvtConnect:
				outEvent.Player = match[2]
			case model.EvtDisconnect:
				outEvent.MetaData = match[2]
			case model.EvtMsg:
				name := match[2]
				dead := false
				team := false
				if strings.HasPrefix(name, teamPrefix) {
					name = strings.TrimPrefix(name, teamPrefix)
					team = true
				}
				if strings.HasPrefix(name, deadTeamPrefix) {
					name = strings.TrimPrefix(name, deadTeamPrefix)
					dead = true
					team = true
				} else if strings.HasPrefix(name, deadPrefix) {
					dead = true
					name = strings.TrimPrefix(name, deadPrefix)
				}
				outEvent.TeamOnly = team
				outEvent.Dead = dead
				outEvent.Player = name
				outEvent.Message = match[3]
			case model.EvtStatusId:
				userID, errUserID := strconv.ParseInt(match[2], 10, 32)
				if errUserID != nil {
					parser.logger.Error("Failed to parse status userid", zap.Error(errUserID))
					continue
				}
				ping, errPing := strconv.ParseInt(match[7], 10, 32)
				if errPing != nil {
					parser.logger.Error("Failed to parse status ping", zap.Error(errPing))
					continue
				}
				dur, durErr := parseConnected(match[5])
				if durErr != nil {
					parser.logger.Error("Failed to parse status duration", zap.Error(durErr))
					continue
				}
				outEvent.UserId = userID
				outEvent.Player = match[3]
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(match[4]))
				outEvent.PlayerConnected = dur
				outEvent.PlayerPing = int(ping)
			case model.EvtKill:
				outEvent.Player = match[2]
				outEvent.Victim = match[3]
			case model.EvtHostname:
				outEvent.MetaData = match[2]
			case model.EvtMap:
				outEvent.MetaData = match[2]
			case model.EvtTags:
				outEvent.MetaData = match[2]
			case model.EvtAddress:
				outEvent.MetaData = match[2]
			case model.EvtLobby:
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(match[2]))
				if match[3] == "INVADERS" {
					outEvent.Team = model.Blu
				} else {
					outEvent.Team = model.Red
				}
			}
			return nil
		}
	}
	return errNoMatch
}
func parseConnected(d string) (time.Duration, error) {
	pcs := strings.Split(d, ":")
	var dur time.Duration
	var parseErr error
	switch len(pcs) {
	case 3:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%sh%sm%ss", pcs[0], pcs[1], pcs[2]))
	case 2:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%sm%ss", pcs[0], pcs[1]))
	case 1:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%ss", pcs[0]))
	default:
		dur = 0
	}
	return dur, parseErr
}

// TODO why keep this?
func (parser *logParser) start(ctx context.Context) {
	for {
		select {
		case msg := <-parser.ReadChannel:
			var logEvent model.LogEvent
			if err := parser.parseEvent(msg, &logEvent); err != nil || errors.Is(err, errNoMatch) {
				continue
			}
			parser.evtChan <- logEvent
		case <-ctx.Done():
			return
		}
	}
}

func newLogParser(logger *zap.Logger, readChannel chan string, evtChan chan model.LogEvent) *logParser {
	return &logParser{
		logger:      logger,
		evtChan:     evtChan,
		ReadChannel: readChannel,
		rx: []*regexp.Regexp{
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(.+?)\skilled\s(.+?)\swith\s(.+)(\.|\. \(crit\))$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(?P<name>.+?)\s:\s{2}(?P<message>.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(.+?)\sconnected$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(Connecting to|Differing lobby received.).+?$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s#\s{1,6}(?P<id>\d{1,6})\s"(?P<name>.+?)"\s+(?P<sid>\[U:\d:\d{1,10}])\s{1,8}(?P<time>\d{1,3}:\d{2}(:\d{2})?)\s+(?P<ping>\d{1,4})\s{1,8}(?P<loss>\d{1,3})\s(spawning|active)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\shostname:\s(.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\smap\s{5}:\s(.+?)\sat.+?$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\stags\s{4}:\s(.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\sudp/ip\s{2}:\s(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5})$`),
			regexp.MustCompile(`^\s{2}(Member|Pending)\[\d+]\s+(?P<sid>\[.+?]).+?TF_GC_TEAM_(?P<team>(DEFENDERS|INVADERS))\s{2}type\s=\sMATCH_PLAYER$`)},
	}
}
