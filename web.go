package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd/frontend"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

func newHTTPServer(ctx context.Context, listenAddr string, store store.Querier, state *gameState, process *processState,
	settingsMgr *settingsManager, re *rules.Engine) (*http.Server, error) {
	mux, errRoutes := createHandlers(store, state, process, settingsMgr, re)
	if errRoutes != nil {
		return nil, errRoutes
	}

	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return httpServer, nil
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

// createHandlers configures the routes. If the `release` tag is enabled, serves files from the embedded assets
// in the binary.
func createHandlers(store store.Querier, state *gameState, process *processState, settings *settingsManager, re *rules.Engine) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /state", onGetState(state, process))
	mux.HandleFunc("GET /messages/{steam_id}", onGetMessages(store))
	mux.HandleFunc("GET /names/{steam_id}", onGetNames(store))
	mux.HandleFunc("POST /mark/{steam_id}", onMarkPlayerPost(store, state, re))
	mux.HandleFunc("DELETE /mark/{steam_id}", onDeleteMarkedPlayer(store, state, re))
	mux.HandleFunc("GET /settings", onGetSettings(settings, re))
	mux.HandleFunc("PUT /settings", onPutSettings(settings))
	mux.HandleFunc("GET /launch", onGGetLaunchGame(process, settings))
	mux.HandleFunc("GET /quit", onGetQuitGame(process))
	mux.HandleFunc("POST /whitelist/{steam_id}", onUpdateWhitelistPlayer(store, state.players, re, true))
	mux.HandleFunc("DELETE /whitelist/{steam_id}", onUpdateWhitelistPlayer(store, state.players, re, false))
	mux.HandleFunc("POST /notes/{steam_id}", onPostNotes(store, state))
	mux.HandleFunc("POST /callvote/{steam_id}/{reason}", onCallVote(state))

	if settings.Settings().RunMode == ModeTest {
		// Don't rely on assets when testing api endpoints
		return mux, nil
	}

	if errStatic := frontend.AddRoutes(mux, "./frontend/dist"); errStatic != nil {
		return nil, errors.Join(errStatic, errHTTPRoutes)
	}

	return mux, nil
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
