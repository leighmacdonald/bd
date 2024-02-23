package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"github.com/nxadm/tail"
)

type logIngest struct {
	tail *tail.Tail
	mu   sync.RWMutex
	// Events are broadcast to any registered consumers
	eventConsumer map[EventType][]chan LogEvent
	logger        *slog.Logger
	parser        Parser
	// Use mostly for testing, allowing simple feeding of an existing console.log file
	external chan string
}

func newLogIngest(path string, parser Parser, echo bool) (*logIngest, error) {
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
		tail:          tailFile,
		logger:        slog.Default().WithGroup("logReader"),
		parser:        parser,
		eventConsumer: make(map[EventType][]chan LogEvent),
		external:      make(chan string),
	}, nil
}

func (li *logIngest) registerConsumer(consumer chan LogEvent, eventTypes ...EventType) {
	li.mu.Lock()
	defer li.mu.Unlock()

	if len(eventTypes) == 0 {
		eventTypes = append(eventTypes, EvtAny)
	}

	for _, evtType := range eventTypes {
		_, found := li.eventConsumer[evtType]
		if !found {
			li.eventConsumer[evtType] = []chan LogEvent{}
		}

		li.eventConsumer[evtType] = append(li.eventConsumer[evtType], consumer)
	}
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

// startEventEmitter begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (li *logIngest) startEventEmitter(ctx context.Context) {
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

			li.mu.RLock()
			// Emit all events to these consumers
			for _, consumer := range li.eventConsumer[EvtAny] {
				consumer <- logEvent
			}

			// Emit specifically requested events to consumers
			for _, consumer := range li.eventConsumer[logEvent.Type] {
				consumer <- logEvent
			}

			li.mu.RUnlock()
		case <-ctx.Done():
			if errStop := li.tail.Stop(); errStop != nil {
				li.logger.Error("Failed to stop tailing console.log cleanly", errAttr(errStop))
			}

			return
		}
	}
}
