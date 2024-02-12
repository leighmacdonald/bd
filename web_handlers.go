package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/bd/rules"
)

func getMessages(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchMessages(r.Context(), sid)
		if errMsgs != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch messages", errAttr(errMsgs))

			return
		}

		responseOK(w, http.StatusOK, messages)
	}
}

func getQuitGame(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		if !detector.gameProcessActive.Load() {
			responseErr(w, http.StatusNotFound, nil)

			return
		}

		slog.Info("Close game request")

		if errQuit := detector.quitGame(); errQuit != nil {
			if errors.Is(errQuit, errGameStopped) {
				responseOK(w, http.StatusOK, nil)

				return
			}

			slog.Error("Failed to close game", errAttr(errQuit))
			responseErr(w, http.StatusInternalServerError, nil)

			return
		}

		responseOK(w, http.StatusOK, nil)
	}
}

func getNames(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchNames(r.Context(), sid)
		if errMsgs != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch names", errAttr(errMsgs))

			return
		}

		responseOK(w, http.StatusOK, messages)
	}
}

type CurrentState struct {
	Tags        []string `json:"tags"`
	GameRunning bool     `json:"game_running"`
	Server      *Server  `json:"server"`
	Players     []Player `json:"players"`
}

func getState(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		detector.serverMu.RLock()
		defer detector.serverMu.RUnlock()

		players := detector.players.all()
		if players == nil {
			players = []Player{}
		}

		responseOK(w, http.StatusOK, CurrentState{
			Tags:        []string{},
			Server:      detector.server,
			Players:     players,
			GameRunning: detector.gameProcessActive.Load(),
		})
	}
}

func getLaunchGame(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		if detector.gameProcessActive.Load() {
			responseErr(w, http.StatusConflict, "Game process active")
			slog.Warn("Failed to launch game, process active already")

			return
		}

		go detector.LaunchGameAndWait()

		responseOK(w, http.StatusNoContent, map[string]string{})
	}
}

type WebUserSettings struct {
	UserSettings
	UniqueTags []string `json:"unique_tags"`
}

func settings(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getSettings(detector, w)
		case http.MethodPut:
			putSettings(detector, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func getSettings(detector *Detector, w http.ResponseWriter) {
	wus := WebUserSettings{
		UserSettings: detector.Settings(),
		UniqueTags:   detector.rules.UniqueTags(),
	}

	responseOK(w, http.StatusOK, wus)
}

func putSettings(detector *Detector, w http.ResponseWriter, r *http.Request) {
	var wus WebUserSettings
	if !bind(w, r, &wus) {
		return
	}

	if errSave := detector.SaveSettings(wus.UserSettings); errSave != nil {
		responseErr(w, http.StatusBadRequest, errSave.Error())
		slog.Error("Failed to save settings", errAttr(errSave))

		return
	}

	responseOK(w, http.StatusNoContent, nil)
}

func callVote(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodPost) {
			return
		}

		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		player, errPlayer := detector.players.bySteamID(sid)
		if errPlayer != nil {
			responseErr(w, http.StatusNotFound, nil)
			slog.Error("Failed to get player state", errAttr(errPlayer), slog.String("steam_id", sid.String()))

			return
		}

		if player.UserID <= 0 {
			responseErr(w, http.StatusNotFound, nil)
			slog.Error("Failed to get player user id", slog.String("steam_id", sid.String()))

			return
		}

		reason := KickReason(r.PathValue("reason"))

		if errVote := detector.callVote(r.Context(), player.UserID, reason); errVote != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to call vote", slog.String("steam_id", sid.String()), errAttr(errVote))

			return
		}

		responseOK(w, http.StatusNoContent, nil)
	}
}

type PostNotesOpts struct {
	Note string `json:"note"`
}

func postNotes(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodPost) {
			return
		}

		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		var opts PostNotesOpts
		if !bind(w, r, &opts) {
			return
		}

		player, errPlayer := detector.GetPlayerOrCreate(r.Context(), sid)

		if errPlayer != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to get or create player", errAttr(errPlayer))

			return
		}

		player.Notes = opts.Note

		player.Dirty = true

		if errSave := detector.dataStore.SavePlayer(r.Context(), &player); errSave != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to save player notes", errAttr(errSave))

			return
		}

		detector.players.update(player)

		responseOK(w, http.StatusNoContent, nil)
	}
}

type PostMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

type UnmarkResponse struct {
	Remaining int `json:"remaining"`
}

func deleteMarkedPlayer(detector *Detector, w http.ResponseWriter, r *http.Request) {
	sid, sidOk := steamIDParam(w, r)
	if !sidOk {
		return
	}

	remaining, errUnmark := detector.unMark(r.Context(), sid)
	if errUnmark != nil {
		if errors.Is(errUnmark, errNotMarked) {
			responseOK(w, http.StatusNotFound, nil)

			return
		}

		responseErr(w, http.StatusInternalServerError, "Failed to unmark player")

		slog.Error("Failed to unmark player", errAttr(errUnmark))
	}

	responseOK(w, http.StatusOK, UnmarkResponse{Remaining: remaining})
}

func markPlayerPost(detector *Detector, w http.ResponseWriter, r *http.Request) {
	sid, sidOk := steamIDParam(w, r)
	if !sidOk {
		return
	}

	var opts PostMarkPlayerOpts
	if !bind(w, r, &opts) {
		return
	}

	if len(opts.Attrs) == 0 {
		responseErr(w, http.StatusBadRequest, nil)
		slog.Error("Received no mark attributes")

		return
	}

	if errCreateMark := detector.mark(r.Context(), sid, opts.Attrs); errCreateMark != nil {
		if errors.Is(errCreateMark, rules.ErrDuplicateSteamID) {
			responseErr(w, http.StatusConflict, nil)
			slog.Warn("Tried to mark duplicate steam id", slog.String("steam_id", sid.String()))

			return
		}

		responseErr(w, http.StatusInternalServerError, nil)
		slog.Error("Failed to mark steam id", errAttr(errCreateMark))

		return
	}

	responseOK(w, http.StatusNoContent, nil)
}

func markPlayer(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			markPlayerPost(detector, w, r)
		case http.MethodDelete:
			deleteMarkedPlayer(detector, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func whitelist(detector *Detector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			updateWhitelistPlayer(w, r, detector, true)
		case http.MethodDelete:
			updateWhitelistPlayer(w, r, detector, false)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func updateWhitelistPlayer(w http.ResponseWriter, r *http.Request, detector *Detector, enable bool) {
	sid, sidOk := steamIDParam(w, r)
	if !sidOk {
		return
	}

	if errWl := detector.whitelist(r.Context(), sid, enable); errWl != nil {
		responseErr(w, http.StatusInternalServerError, nil)
		slog.Error("Failed to whitelist steam_id", errAttr(errWl), slog.String("steam_id", sid.String()))

		return
	}

	responseOK(w, http.StatusNoContent, nil)
}
