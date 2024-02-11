package main

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/rules"
	"github.com/pkg/errors"
	"log/slog"
	"net/http"
)

func getMessages(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchMessages(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch messages", errAttr(errMsgs))

			return
		}

		responseOK(ctx, http.StatusOK, messages)
	}
}

func getQuitGame(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !detector.gameProcessActive.Load() {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		slog.Info("Close game request")

		if errQuit := detector.quitGame(); errQuit != nil {
			if errors.Is(errQuit, errGameStopped) {
				responseOK(ctx, http.StatusOK, nil)

				return
			}

			slog.Error("Failed to close game", errAttr(errQuit))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, nil)
	}
}

func getNames(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchNames(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to fetch names", errAttr(errMsgs))

			return
		}

		responseOK(ctx, http.StatusOK, messages)
	}
}

type CurrentState struct {
	Tags        []string `json:"tags"`
	GameRunning bool     `json:"game_running"`
	Server      *Server  `json:"server"`
	Players     []Player `json:"players"`
}

func getState(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		detector.serverMu.RLock()
		defer detector.serverMu.RUnlock()

		players := detector.players.all()
		if players == nil {
			players = []Player{}
		}

		responseOK(ctx, http.StatusOK, CurrentState{
			Tags:        []string{},
			Server:      detector.server,
			Players:     players,
			GameRunning: detector.gameProcessActive.Load(),
		})
	}
}

func getLaunchGame(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if detector.gameProcessActive.Load() {
			responseErr(ctx, http.StatusConflict, "Game process active")
			slog.Warn("Failed to launch game, process active already")

			return
		}

		go detector.LaunchGameAndWait()

		responseOK(ctx, http.StatusNoContent, gin.H{})
	}
}

type WebUserSettings struct {
	UserSettings
	UniqueTags []string `json:"unique_tags"`
}

func getSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wus := WebUserSettings{
			UserSettings: detector.Settings(),
			UniqueTags:   detector.rules.UniqueTags(),
		}

		responseOK(ctx, http.StatusOK, wus)
	}
}

func putSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var wus WebUserSettings
		if !bind(ctx, &wus) {
			return
		}

		if errSave := detector.SaveSettings(wus.UserSettings); errSave != nil {
			responseErr(ctx, http.StatusBadRequest, errSave.Error())
			slog.Error("Failed to save settings", errAttr(errSave))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

func callVote(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		player, errPlayer := detector.players.bySteamID(sid)
		if errPlayer != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			slog.Error("Failed to get player state", errAttr(errPlayer), slog.String("steam_id", sid.String()))

			return
		}

		if player.UserID <= 0 {
			responseErr(ctx, http.StatusNotFound, nil)
			slog.Error("Failed to get player user id", slog.String("steam_id", sid.String()))

			return
		}

		reason := KickReason(ctx.Param("reason"))

		if errVote := detector.callVote(ctx, player.UserID, reason); errVote != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to call vote", slog.String("steam_id", sid.String()), errAttr(errVote))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type PostNotesOpts struct {
	Note string `json:"note"`
}

func postNotes(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		var opts PostNotesOpts
		if !bind(ctx, &opts) {
			return
		}

		player, errPlayer := detector.GetPlayerOrCreate(ctx, sid)

		if errPlayer != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to get or create player", errAttr(errPlayer))

			return
		}

		player.Notes = opts.Note

		player.Dirty = true

		if errSave := detector.dataStore.SavePlayer(ctx, &player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to save player notes", errAttr(errSave))

			return
		}

		detector.players.update(player)

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type PostMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

type UnmarkResponse struct {
	Remaining int `json:"remaining"`
}

func deleteMarkedPlayer(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		remaining, errUnmark := detector.unMark(ctx, sid)
		if errUnmark != nil {
			if errors.Is(errUnmark, errNotMarked) {
				responseOK(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, "Failed to unmark player")

			slog.Error("Failed to unmark player", errAttr(errUnmark))
		}

		ctx.JSON(http.StatusOK, UnmarkResponse{Remaining: remaining})
	}
}

func postMarkPlayer(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		var opts PostMarkPlayerOpts
		if !bind(ctx, &opts) {
			return
		}

		if len(opts.Attrs) == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			slog.Error("Received no mark attributes")

			return
		}

		if errMark := detector.mark(ctx, sid, opts.Attrs); errMark != nil {
			if errors.Is(errMark, rules.ErrDuplicateSteamID) {
				responseErr(ctx, http.StatusConflict, nil)
				slog.Warn("Tried to mark duplicate steam id", slog.String("steam_id", sid.String()))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to mark steam id", errAttr(errMark))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

func updateWhitelistPlayer(detector *Detector, enable bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
		if !sidOk {
			return
		}

		if errWl := detector.whitelist(ctx, sid, enable); errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to whitelist steam_id", errAttr(errWl), slog.String("steam_id", sid.String()))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}
