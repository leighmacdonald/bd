package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd/frontend"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type Web struct {
	*http.Server
}

func newWebServer(detector *Detector) (*Web, error) {
	mux, errRoutes := createMux(detector)
	if errRoutes != nil {
		return nil, errRoutes
	}

	httpServer := &http.Server{
		Addr:         detector.Settings().HTTPListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Web{
		Server: httpServer,
	}, nil
}

func (w *Web) startWeb(ctx context.Context) error {
	w.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	if errServe := w.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		return errors.Join(errServe, errHTTPListen)
	}

	return nil
}

func bind(w http.ResponseWriter, r *http.Request, receiver any) bool {
	if errDecode := json.NewDecoder(r.Body).Decode(receiver); errDecode != nil {
		responseErr(w, http.StatusBadRequest, nil)

		slog.Error("Received malformed request", errAttr(errDecode))

		return false
	}

	return true
}

func responseErr(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Could not encode error payload", errAttr(err))
		}
	}
}

func responseOK(w http.ResponseWriter, status int, data any) {
	if data == nil {
		data = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", errAttr(err))
	}
}

// createMux configures the routes. If the `release` tag is enabled, serves files from the embedded assets
// in the binary.
func createMux(detector *Detector) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/state", getState(detector))
	mux.HandleFunc("/messages/:steam_id", getMessages(detector))
	mux.HandleFunc("/names/:steam_id", getNames(detector))
	mux.HandleFunc("/mark/:steam_id", markPlayer(detector))
	mux.HandleFunc("/settings", settings(detector))
	mux.HandleFunc("/launch", getLaunchGame(detector))
	mux.HandleFunc("/quit", getQuitGame(detector))
	mux.HandleFunc("/whitelist/:steam_id", whitelist(detector))
	mux.HandleFunc("/notes/:steam_id", postNotes(detector))
	mux.HandleFunc("/callvote/:steam_id/:reason", callVote(detector))

	defaultHandler, errStatic := frontend.AddRoutes(mux)
	if errStatic != nil {
		return nil, errors.Join(errStatic, errHTTPRoutes)
	}

	mux.Handle("/", defaultHandler)

	return mux, nil
}

func ensureMethod(w http.ResponseWriter, r *http.Request, allowedMethods ...string) bool {
	valid := false

	for _, method := range allowedMethods {
		if method == r.Method {
			valid = true

			break
		}
	}

	if !valid {
		if _, err := io.WriteString(w, "Unsupported method"); err != nil {
			slog.Error("Failed to write error message", errAttr(err))
		}

		w.WriteHeader(http.StatusMethodNotAllowed)

		return false
	}

	return true
}

func steamIDParam(w http.ResponseWriter, r *http.Request) (steamid.SID64, bool) {
	sidValue := r.PathValue("steam_id")
	steamID := steamid.New(sidValue)

	if !steamID.Valid() {
		responseErr(w, http.StatusBadRequest, nil)
		slog.Error("Failed to parse steam id param", slog.String("steam_id", sidValue))

		return "", false
	}

	return steamID, true
}

func (w *Web) Stop(ctx context.Context) error {
	if w.Server == nil {
		return nil
	}

	timeout, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	if errShutdown := w.Server.Shutdown(timeout); errShutdown != nil {
		return errors.Join(errShutdown, errHTTPShutdown)
	}

	return nil
}
