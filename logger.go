package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dotse/slug"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

// tailLogAdapter implements a tail.logger interface using log/slog.
type tailLogAdapter struct {
	echo bool
}

func (t tailLogAdapter) Fatal(v ...interface{}) {
	if !t.echo {
		return
	}

	// Not actually fatal.
	slog.Error("Fatal error", slog.Any("value", v))
}

func (t tailLogAdapter) Fatalf(format string, v ...interface{}) {
	if !t.echo {
		return
	}

	// Not actually fatal.
	slog.Error("Fatal error", slog.Any("value", fmt.Sprintf(format, v...)))
}

func (t tailLogAdapter) Fatalln(v ...interface{}) {
	if !t.echo {
		return
	}
	// Not actually fatal.
	slog.Error("Fatal error", slog.Any("value", v))
}

func (t tailLogAdapter) Panic(v ...interface{}) {
	if !t.echo {
		return
	}

	panic(v)
}

func (t tailLogAdapter) Panicf(format string, v ...interface{}) {
	if !t.echo {
		return
	}

	panic(fmt.Sprintf(format, v...))
}

func (t tailLogAdapter) Panicln(v ...interface{}) {
	if !t.echo {
		return
	}

	panic(fmt.Sprintf("%v\n", v))
}

func (t tailLogAdapter) Print(v ...interface{}) {
	if !t.echo {
		return
	}

	slog.Info(fmt.Sprintf("%v", v))
}

func (t tailLogAdapter) Printf(format string, v ...interface{}) {
	if !t.echo {
		return
	}

	slog.Info(fmt.Sprintf(format, v...))
}

func (t tailLogAdapter) Println(v ...interface{}) {
	if !t.echo {
		return
	}

	slog.Info(fmt.Sprintf("%v\n", v))
}

func MustCreateLogger(sm *settingsManager) func() {
	var (
		logHandler slog.Handler
		level      slog.Level
	)

	switch sm.Settings().LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}

	closer := func() {}

	opts := slug.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: level,
		},
	}

	settings := sm.Settings()

	if settings.DebugLogEnabled {
		logFile, errLogFile := os.Create(sm.LogFilePath())
		if errLogFile != nil {
			panic(fmt.Sprintf("Failed to open logfile: %v", errLogFile))
		}

		closer = func() {
			if errClose := logFile.Close(); errClose != nil {
				panic(fmt.Sprintf("Failed to close log file: %v", errClose))
			}
		}

		logHandler = slug.NewHandler(opts, logFile)
	} else {
		logHandler = slug.NewHandler(opts, os.Stdout)
	}

	slog.SetDefault(slog.New(logHandler))

	return closer
}

func errAttr(err error) slog.Attr {
	return slog.Any("error", err)
}

func sidAttr(steamID steamid.SID64) slog.Attr {
	return slog.String("sid", steamID.String())
}
