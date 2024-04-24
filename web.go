package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// newHTTPServer configures the HTTP server using the address and handler provided. Note that we also
// replace the default context with our own parent context to allow better shutdown semantics.
func newHTTPServer(ctx context.Context, listenAddr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

// bind is a helper function to ensure the json request body binds to the provided receiver. The receiver should
// be a pointer to the matching struct. If it fails a error is returned to the http client and no further processing
// is required in the route handler beyond this call.
func bind(w http.ResponseWriter, r *http.Request, receiver any) bool {
	if errDecode := json.NewDecoder(r.Body).Decode(receiver); errDecode != nil {
		responseErr(w, http.StatusBadRequest, nil)

		slog.Error("Received malformed request",
			errAttr(errDecode), slog.String("path", r.RequestURI), slog.String("method", r.Method))

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

// responseOK provides a helper for successful responses. All responses currently return a list, so to
// make life a little easier we ensure we return a empty list instead of a null value so we do not have to deal
// with 2 separate types on the frontend.
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
func createHandlers(store store.Querier, state *gameState, process *processState, settings *settingsManager, re *rules.Engine, rcon rconConnection) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/state", onGetState(state, process))
	mux.HandleFunc("GET /api/messages/{steam_id}", onGetMessages(store))
	mux.HandleFunc("GET /api/names/{steam_id}", onGetNames(store))
	mux.HandleFunc("POST /api/mark/{steam_id}", onMarkPlayerPost(settings, store, state, re))
	mux.HandleFunc("DELETE /api/mark/{steam_id}", onDeleteMarkedPlayer(store, state, re))
	mux.HandleFunc("GET /api/settings", onGetSettings(settings, re))
	mux.HandleFunc("PUT /api/settings", onPutSettings(settings))
	mux.HandleFunc("GET /api/launch", onGGetLaunchGame(process, settings))
	mux.HandleFunc("GET /api/quit", onGetQuitGame(process))
	mux.HandleFunc("POST /api/whitelist/{steam_id}", onUpdateWhitelistPlayer(store, state, true))
	mux.HandleFunc("DELETE /api/whitelist/{steam_id}", onUpdateWhitelistPlayer(store, state, false))
	mux.HandleFunc("POST /api/notes/{steam_id}", onPostNotes(store, state))
	mux.HandleFunc("POST /api/callvote/{steam_id}/{reason}", onCallVote(state, rcon))

	if settings.Settings().RunMode == ModeTest {
		// Don't rely on assets when testing api endpoints
		return mux, nil
	}

	if errStatic := AddRoutes(mux, "./frontend/dist"); errStatic != nil {
		return nil, errors.Join(errStatic, errHTTPRoutes)
	}

	return mux, nil
}

// steamIDParam provides a helper function for pulling out and validating a steam_id value from
// a route parameter. Expects the parameter to always be called `steam_id`.
func steamIDParam(w http.ResponseWriter, r *http.Request) (steamid.SteamID, bool) {
	sidValue := r.PathValue("steam_id")
	steamID := steamid.New(sidValue)

	if !steamID.Valid() {
		responseErr(w, http.StatusBadRequest, nil)
		slog.Error("Failed to parse steam id param", slog.String("steam_id", sidValue))

		return steamID, false
	}

	return steamID, true
}
