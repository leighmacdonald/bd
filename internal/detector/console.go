package detector

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

type LogReader struct {
	tail    *tail.Tail
	outChan chan string
	logger  *slog.Logger
}

func (reader *LogReader) start(ctx context.Context) {
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
				reader.logger.Error("Failed to stop tailing console.log cleanly", "err", errStop)
			}

			return
		}
	}
}

func newLogReader(logger *slog.Logger, path string, outChan chan string, echo bool) (*LogReader, error) {
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
	tailFile, errTail := tail.TailFile(path, tailConfig)
	if errTail != nil {
		return nil, errors.Wrap(errTail, "Failed to configure tail")
	}

	logReader := LogReader{
		tail:    tailFile,
		outChan: outChan,
		logger:  logger.WithGroup("logreader"),
	}

	return &logReader, nil
}

var ErrNoMatch = errors.New("no match found")

type LogParser struct {
	evtChan     chan LogEvent
	ReadChannel chan string
	rx          []*regexp.Regexp
	logger      *slog.Logger
}

const (
	teamPrefix     = "(TEAM) "
	deadPrefix     = "*DEAD* "
	deadTeamPrefix = "*DEAD*(TEAM) "
)

func (parser *LogParser) Parse(msg string, outEvent *LogEvent) error {
	// the index must match the index of the EventType const values
	for i, rxMatcher := range parser.rx {
		if match := rxMatcher.FindStringSubmatch(msg); match != nil { //nolint:nestif
			outEvent.Type = EventType(i)
			if outEvent.Type != EvtLobby {
				if errTS := outEvent.ApplyTimestamp(match[1]); errTS != nil {
					parser.logger.Error("Failed to parse timestamp", "err", errTS)
				}
			}

			switch outEvent.Type {
			case EvtConnect:
				outEvent.Player = match[2]
			case EvtDisconnect:
				outEvent.MetaData = match[2]
			case EvtMsg:
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
			case EvtStatusID:
				userID, errUserID := strconv.ParseInt(match[2], 10, 32)
				if errUserID != nil {
					parser.logger.Error("Failed to parse status userid", "err", errUserID)

					continue
				}

				ping, errPing := strconv.ParseInt(match[7], 10, 32)
				if errPing != nil {
					parser.logger.Error("Failed to parse status ping", "err", errPing)

					continue
				}

				dur, durErr := parseConnected(match[5])
				if durErr != nil {
					parser.logger.Error("Failed to parse status duration", "err", durErr)

					continue
				}

				outEvent.UserID = userID
				outEvent.Player = match[3]
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(match[4]))
				outEvent.PlayerConnected = dur
				outEvent.PlayerPing = int(ping)
			case EvtKill:
				outEvent.Player = match[2]
				outEvent.Victim = match[3]
			case EvtHostname:
				outEvent.MetaData = match[2]
			case EvtMap:
				outEvent.MetaData = match[2]
			case EvtTags:
				outEvent.MetaData = match[2]
			case EvtAddress:
				outEvent.MetaData = match[2]
			case EvtLobby:
				outEvent.PlayerSID = steamid.SID3ToSID64(steamid.SID3(match[2]))
				if match[3] == "INVADERS" {
					outEvent.Team = store.Blu
				} else {
					outEvent.Team = store.Red
				}
			}

			return nil
		}
	}

	return ErrNoMatch
}

func parseConnected(d string) (time.Duration, error) {
	var (
		pcs      = strings.Split(d, ":")
		dur      time.Duration
		parseErr error
	)

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

	if parseErr != nil {
		return 0, errors.Wrap(parseErr, "Failed to parse connected duration")
	}

	return dur, nil
}

// TODO why keep this?
func (parser *LogParser) start(ctx context.Context) {
	for {
		select {
		case msg := <-parser.ReadChannel:
			var logEvent LogEvent
			if err := parser.Parse(msg, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
				continue
			}

			parser.evtChan <- logEvent
			// select {
			// case parser.evtChan <- logEvent:
			// default:
			// 	parser.logger.Debug("Event channel full")
			// }
		case <-ctx.Done():
			return
		}
	}
}

func NewLogParser(logger *slog.Logger, readChannel chan string, evtChan chan LogEvent) *LogParser {
	return &LogParser{
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
			regexp.MustCompile(`^\s{2}(Member|Pending)\[\d+]\s+(?P<sid>\[.+?]).+?TF_GC_TEAM_(?P<team>(DEFENDERS|INVADERS))\s{2}type\s=\sMATCH_PLAYER$`),
		},
	}
}
