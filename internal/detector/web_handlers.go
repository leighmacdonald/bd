package detector

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func getMessages(detector *Detector) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchMessages(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch messages", zap.Error(errMsgs))

			return
		}

		responseOK(ctx, http.StatusOK, messages)
	}
}

func getNames(detector *Detector) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		messages, errMsgs := detector.dataStore.FetchNames(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch names", zap.Error(errMsgs))

			return
		}

		responseOK(ctx, http.StatusOK, messages)
	}
}

type CurrentState struct {
	Tags        []string        `json:"tags"`
	GameRunning bool            `json:"game_running"`
	Server      *Server         `json:"server"`
	Players     []*store.Player `json:"players"`
}

func getState(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		detector.playersMu.RLock()
		defer detector.playersMu.RUnlock()
		detector.serverMu.RLock()
		defer detector.serverMu.RUnlock()

		players := detector.players
		if players == nil {
			players = []*store.Player{}
		}

		responseOK(ctx, http.StatusOK, CurrentState{
			Tags:        []string{},
			Server:      detector.server,
			Players:     players,
			GameRunning: detector.gameProcessActive.Load(),
		})
	}
}

func getLaunch(detector *Detector) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		if detector.gameProcessActive.Load() {
			responseErr(ctx, http.StatusConflict, "Game process active")
			log.Warn("Failed to launch game, process active already")

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
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var wus WebUserSettings
		if !bind(ctx, &wus, log) {
			return
		}

		if errSave := detector.SaveSettings(wus.UserSettings); errSave != nil {
			responseErr(ctx, http.StatusBadRequest, errSave.Error())
			log.Error("Failed to save settings", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type PostNotesOpts struct {
	Note string `json:"note"`
}

func postNotes(detector *Detector) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		var opts PostNotesOpts
		if !bind(ctx, &opts, log) {
			return
		}

		player, errPlayer := detector.GetPlayerOrCreate(ctx, sid)
		if errPlayer != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to get or create player", zap.Error(errPlayer))

			return
		}

		detector.playersMu.Lock()

		player.Notes = opts.Note

		player.Touch()

		if errSave := detector.dataStore.SavePlayer(ctx, player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			detector.playersMu.Unlock()
			log.Error("Failed to save player notes", zap.Error(errSave))

			return
		}

		detector.playersMu.Unlock()

		go detector.updateState(newNoteEvent(sid, opts.Note))

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
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		remaining, errUnmark := detector.UnMark(ctx, sid)
		if errUnmark != nil {
			if errors.Is(errUnmark, errNotMarked) {
				responseOK(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, "Failed to unmark player")

			log.Error("Failed to unmark player", zap.Error(errUnmark))
		}

		ctx.JSON(http.StatusOK, UnmarkResponse{Remaining: remaining})
	}
}

func postMarkPlayer(detector *Detector) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		var opts PostMarkPlayerOpts
		if !bind(ctx, &opts, log) {
			return
		}

		if len(opts.Attrs) == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Received no mark attributes")

			return
		}

		if errMark := detector.Mark(ctx, sid, opts.Attrs); errMark != nil {
			if errors.Is(errMark, rules.ErrDuplicateSteamID) {
				responseErr(ctx, http.StatusConflict, nil)
				log.Warn("Tried to mark duplicate steam id", zap.String("steam_id", sid.String()))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to mark steam id", zap.Error(errMark))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

func updateWhitelistPlayer(detector *Detector, enable bool) gin.HandlerFunc {
	log := detector.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx, log)
		if !sidOk {
			return
		}

		if errWl := detector.Whitelist(ctx, sid, enable); errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to whitelist steam_id", zap.Error(errWl), zap.String("steam_id", sid.String()))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}
