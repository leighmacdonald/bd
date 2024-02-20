package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dotse/slug"
)

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
