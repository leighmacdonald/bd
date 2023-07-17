package detector

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
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

			return
		}

		responseOK(ctx, http.StatusOK, messages)
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

			return
		}

		responseOK(ctx, http.StatusOK, messages)
	}
}

func getState(detector *Detector) gin.HandlerFunc {
	type currentState struct {
		Server  *Server         `json:"server"`
		Players []*store.Player `json:"players"`
	}

	return func(ctx *gin.Context) {
		detector.playersMu.RLock()
		defer detector.playersMu.RUnlock()
		detector.serverMu.RLock()
		defer detector.serverMu.RUnlock()

		players := detector.players
		if players == nil {
			players = []*store.Player{}
		}

		responseOK(ctx, http.StatusOK, currentState{Server: detector.server, Players: players})
	}
}

type WebUserSettings struct {
	*UserSettings
	UniqueTags []string `json:"unique_tags"`
}

func getSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wus := WebUserSettings{
			UserSettings: detector.settings,
			UniqueTags:   detector.rules.UniqueTags(),
		}

		responseOK(ctx, http.StatusOK, wus)
	}
}

func postSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var wus WebUserSettings
		if !bind(ctx, &wus) {
			return
		}

		wus.RWMutex = &sync.RWMutex{}
		// TODO Proper validation
		detector.settings = wus.UserSettings

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

			return
		}

		detector.playersMu.Lock()

		player.Notes = opts.Note

		player.Touch()

		if errSave := detector.dataStore.SavePlayer(ctx, player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			detector.playersMu.Unlock()

			return
		}

		detector.playersMu.Unlock()
		detector.updateState(newNoteEvent(sid, opts.Note))
		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type PostMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
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

			return
		}

		if errMark := detector.Mark(ctx, sid, opts.Attrs); errMark != nil {
			if errors.Is(errMark, rules.ErrDuplicateSteamID) {
				responseErr(ctx, http.StatusConflict, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

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

		if errWl := detector.Whitelist(ctx, sid, enable); errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}
