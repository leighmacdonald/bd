package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/dotse/slug"
	"github.com/leighmacdonald/steamid/v4/steamid"
	slogmulti "github.com/samber/slog-multi"
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

func MustCreateLogger(settings userSettings) func() {
	var level slog.Level

	switch settings.LogLevel {
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

	var handlers []slog.Handler

	// Write debug logs to bd.log
	if settings.DebugLogEnabled {
		logFile, errLogFile := os.Create(path.Join(settings.configRoot, "bd.log"))
		if errLogFile != nil {
			panic(fmt.Sprintf("Failed to open logfile: %v", errLogFile))
		}

		closer = func() {
			if errClose := logFile.Close(); errClose != nil {
				panic(fmt.Sprintf("Failed to close log file: %v", errClose))
			}
		}

		handlers = append(handlers, slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: level}))
	}

	// Colourised logs for stdout
	handlers = append(handlers, slug.NewHandler(slug.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: level,
		},
	}, os.Stdout))

	slog.SetDefault(slog.New(slogmulti.Fanout(handlers...)))

	return closer
}

func errAttr(err error) slog.Attr {
	return slog.Any("error", err)
}

func sidAttr(steamID steamid.SteamID) slog.Attr {
	return slog.String("sid", steamID.String())
}
