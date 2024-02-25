package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
)

func onGetMessages(store store.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		messages, errMsgs := store.Messages(r.Context(), sid.Int64())
		if errMsgs != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch messages", errAttr(errMsgs))

			return
		}

		responseOK(w, http.StatusOK, messages)
	}
}

func onGetQuitGame(process *processState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !process.gameProcessActive.Load() {
			responseErr(w, http.StatusNotFound, nil)

			return
		}

		slog.Debug("Close game request")

		if errQuit := process.Quit(r.Context()); errQuit != nil {
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

func onGetNames(store store.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		messages, errMsgs := store.UserNames(r.Context(), sid.Int64())
		if errMsgs != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch names", errAttr(errMsgs))

			return
		}

		responseOK(w, http.StatusOK, messages)
	}
}

type CurrentState struct {
	Tags        []string      `json:"tags"`
	GameRunning bool          `json:"game_running"`
	Server      serverState   `json:"server"`
	Players     []PlayerState `json:"players"`
}

func onGetState(state *gameState, process *processState) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		players := state.players.all()
		if players == nil {
			players = []PlayerState{}
		}

		responseOK(w, http.StatusOK, CurrentState{
			Tags:        []string{},
			Server:      state.server,
			Players:     players,
			GameRunning: process.gameProcessActive.Load(),
		})
	}
}

func onGGetLaunchGame(process *processState, settingsMgr *settingsManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if process.gameProcessActive.Load() {
			responseErr(w, http.StatusConflict, "Game process active")
			slog.Warn("Failed to launch game, process active already")

			return
		}

		go process.launchGameAndWait(settingsMgr)

		responseOK(w, http.StatusOK, map[string]string{})
	}
}

type WebUserSettings struct {
	userSettings
	UniqueTags []string `json:"unique_tags"`
}

func onGetSettings(settings *settingsManager, rules *rules.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		wus := WebUserSettings{
			userSettings: settings.Settings(),
			UniqueTags:   rules.UniqueTags(),
		}

		responseOK(w, http.StatusOK, wus)
	}
}

func onPutSettings(settings *settingsManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wus WebUserSettings
		if !bind(w, r, &wus) {
			return
		}

		if errValidate := wus.userSettings.Validate(); errValidate != nil {
			responseErr(w, http.StatusBadRequest, errValidate)
			return
		}

		if errSave := settings.replace(wus.userSettings); errSave != nil {
			responseErr(w, http.StatusInternalServerError, errSave)
			return
		}

		responseOK(w, http.StatusOK, settings.Settings())
	}
}

func onCallVote(state *gameState, connection rconConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		player, errPlayer := state.players.bySteamID(sid, true)
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

		cmd := fmt.Sprintf("callvote kick \"%d %s\"", player.UserID, reason)

		resp, errCallVote := connection.exec(r.Context(), cmd, false)
		if errCallVote != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to call vote", slog.String("steam_id", sid.String()), errAttr(errCallVote))

			return
		}

		slog.Debug(resp, slog.String("cmd", cmd))

		responseOK(w, http.StatusNoContent, nil)
	}
}

type PostNotesOpts struct {
	Note string `json:"note"`
}

func onPostNotes(db store.Querier, state *gameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		var opts PostNotesOpts
		if !bind(w, r, &opts) {
			return
		}

		player, errPlayer := getPlayerOrCreate(r.Context(), db, sid)

		if errPlayer != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to get or create player", errAttr(errPlayer))

			return
		}

		player.Notes = opts.Note

		if errSave := db.PlayerUpdate(r.Context(), player.toUpdateParams()); errSave != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to save player notes", errAttr(errSave))

			return
		}

		state.players.update(player)

		responseOK(w, http.StatusNoContent, nil)
	}
}

type PostMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

type UnmarkResponse struct {
	Remaining int `json:"remaining"`
}

func onDeleteMarkedPlayer(db store.Querier, state *gameState, re *rules.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		remaining, errUnmark := unMark(r.Context(), re, db, state, sid)
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
}

func onMarkPlayerPost(sm *settingsManager, db store.Querier, state *gameState, re *rules.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if errCreateMark := mark(r.Context(), sm, db, state, re, sid, opts.Attrs); errCreateMark != nil {
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
}

func onUpdateWhitelistPlayer(db store.Querier, state *gameState, enable bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := steamIDParam(w, r)
		if !sidOk {
			return
		}

		if errWl := whitelist(r.Context(), db, state, sid, enable); errWl != nil {
			responseErr(w, http.StatusInternalServerError, nil)
			slog.Error("Failed to whitelist steam_id", errAttr(errWl), slog.String("steam_id", sid.String()))

			return
		}

		responseOK(w, http.StatusNoContent, nil)
	}
}
