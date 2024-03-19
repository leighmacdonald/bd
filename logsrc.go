package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
)

type eventBroadcaster struct {
	// Events are broadcast to any registered consumers
	eventConsumer map[EventType][]chan LogEvent
	mu            sync.RWMutex
}

func newEventBroadcaster() *eventBroadcaster {
	return &eventBroadcaster{eventConsumer: make(map[EventType][]chan LogEvent)}
}

func (e *eventBroadcaster) broadcast(logEvent LogEvent) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Emit all events to these consumers
	for _, consumer := range e.eventConsumer[EvtAny] {
		consumer <- logEvent
	}

	// Emit specifically requested events to consumers
	for _, consumer := range e.eventConsumer[logEvent.Type] {
		consumer <- logEvent
	}
}

func (e *eventBroadcaster) registerConsumer(consumer chan LogEvent, eventTypes ...EventType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(eventTypes) == 0 {
		eventTypes = append(eventTypes, EvtAny)
	}

	for _, evtType := range eventTypes {
		_, found := e.eventConsumer[evtType]
		if !found {
			e.eventConsumer[evtType] = []chan LogEvent{}
		}

		e.eventConsumer[evtType] = append(e.eventConsumer[evtType], consumer)
	}
}

type logIngest struct {
	tail   *tail.Tail
	logger *slog.Logger
	parser Parser
	// Use mostly for testing, allowing simple feeding of an existing console.log file
	external    chan string
	broadcaster *eventBroadcaster
}

func newLogIngest(path string, parser Parser, echo bool, broadcaster *eventBroadcaster) (*logIngest, error) {
	//goland:noinspection GoBoolExpressions
	tailConfig := tail.Config{
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      runtime.GOOS == "windows",
		Logger:    tailLogAdapter{echo: echo},
	}

	tailFile, errTail := tail.TailFile(path, tailConfig)
	if errTail != nil {
		return nil, errors.Join(errTail, errLogTailCreate)
	}

	return &logIngest{
		tail:        tailFile,
		logger:      slog.Default().WithGroup("logReader"),
		parser:      parser,
		broadcaster: broadcaster,
		external:    make(chan string),
	}, nil
}

func (li *logIngest) lineEmitter(ctx context.Context, incoming chan string) {
	for {
		select {
		case msg := <-li.tail.Lines:
			if msg == nil {
				// Happens on linux only?
				continue
			}

			line := strings.TrimSuffix(msg.Text, "\r")
			if line == "" {
				continue
			}

			incoming <- line
		case externalLine := <-li.external:
			line := strings.TrimSuffix(externalLine, "\r")
			if line == "" {
				continue
			}
			incoming <- line
		case <-ctx.Done():
			return
		}
	}
}

// start begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (li *logIngest) start(ctx context.Context) {
	defer li.tail.Cleanup()
	incomingLogLines := make(chan string)

	go li.lineEmitter(ctx, incomingLogLines)

	for {
		select {
		case line := <-incomingLogLines:
			var logEvent LogEvent
			if err := li.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
				// slog.Debug("could not match line", slog.String("line", line))
				continue
			}

			li.broadcaster.broadcast(logEvent)
		case <-ctx.Done():
			if errStop := li.tail.Stop(); errStop != nil {
				li.logger.Error("Failed to stop tailing console.log cleanly", errAttr(errStop))
			}

			return
		}
	}
}

type srcdsPacket byte

const (
	// Normal log messages (unsupported).
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret.
	s2aLogString2 srcdsPacket = 0x53
)

type udpListener struct {
	udpAddr     *net.UDPAddr
	broadcaster *eventBroadcaster
	parser      *logParser
}

func newUDPListener(logAddr string, parser *logParser, broadcaster *eventBroadcaster) (*udpListener, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Join(errResolveUDP, errResolveAddr)
	}

	return &udpListener{
		udpAddr:     udpAddr,
		broadcaster: broadcaster,
		parser:      parser,
	}, nil
}

// Start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (l *udpListener) start(ctx context.Context) {
	type newMsg struct {
		source int64
		body   string
	}

	connection, errListenUDP := net.ListenUDP("udp4", l.udpAddr)
	if errListenUDP != nil {
		slog.Error("Failed to start log listener", errAttr(errListenUDP))

		return
	}

	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			slog.Error("Failed to close connection cleanly", errAttr(errConnClose))
		}
	}()

	slog.Info("Starting log reader",
		slog.String("listen_addr", fmt.Sprintf("%s/udp", l.udpAddr.String())))

	var (
		count          = uint64(0)
		insecureCount  = uint64(0)
		errCount       = uint64(0)
		msgIngressChan = make(chan newMsg)
	)

	go func() {
		// Close the listener on context cancellation
		<-ctx.Done()
		if errClose := connection.Close(); errClose != nil {
			slog.Error("failed to close udp connection cleanly", errAttr(errClose))
		}
	}()

	startTime := time.Now()
	buffer := make([]byte, 1024)

	for {
		// Reuse memory
		clear(buffer)

		readLen, _, errReadUDP := connection.ReadFromUDP(buffer)
		if errReadUDP != nil {
			if errors.Is(errReadUDP, net.ErrClosed) {
				return
			}

			slog.Warn("UDP log read error", errAttr(errReadUDP))

			continue
		}

		switch srcdsPacket(buffer[4]) {
		case s2aLogString:
			if insecureCount%10000 == 0 {
				slog.Error("Using unsupported log packet type 0x52",
					slog.Int64("count", int64(insecureCount+1)))
			}

			insecureCount++
			errCount++
		case s2aLogString2:
			line := string(buffer)

			idx := strings.Index(line, "L ")
			if idx == -1 {
				slog.Warn("Received malformed log message: Failed to find marker")

				errCount++

				continue
			}

			secret, errConv := strconv.ParseInt(line[5:idx], 10, 32)
			if errConv != nil {
				slog.Error("Received malformed log message: Failed to parse secret",
					errAttr(errConv))

				errCount++

				continue
			}

			msgIngressChan <- newMsg{source: secret, body: line[idx : readLen-2]}

			count++

			if count%10000 == 0 {
				rate := float64(count) / time.Since(startTime).Seconds()

				slog.Debug("UDP SRCDS Logger Packets",
					slog.Uint64("count", count),
					slog.Float64("messages/sec", rate),
					slog.Uint64("errors", errCount))

				startTime = time.Now()
			}
		}
	}
}
