package web

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"sync"
)

func getMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		messages, errMsgs := detector.Store().FetchMessages(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, messages)
	}
}

func getNames() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		messages, errMsgs := detector.Store().FetchNames(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, messages)
	}
}

func getPlayers() gin.HandlerFunc {
	testPlayers := createTestPlayers(24)
	return func(ctx *gin.Context) {
		if _, isTest := os.LookupEnv("TEST"); isTest {
			responseOK(ctx, http.StatusOK, testPlayers)
			return
		}
		players := detector.Players()
		var p []store.Player
		if players != nil {
			p = players
		}
		responseOK(ctx, http.StatusOK, p)
	}
}

type webUserSettings struct {
	*detector.UserSettings
	UniqueTags []string `json:"unique_tags"`
}

func getSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wus := webUserSettings{
			UserSettings: detector.Settings(),
			UniqueTags:   rules.UniqueTags(),
		}
		responseOK(ctx, http.StatusOK, wus)
	}
}

func postSettings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var wus webUserSettings
		if !bind(ctx, &wus) {
			return
		}
		wus.RWMutex = &sync.RWMutex{}
		detector.SetSettings(wus.UserSettings)
		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postNotesOpts struct {
	Note string `json:"note"`
}

func postNotes() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		var opts postNotesOpts
		if !bind(ctx, &opts) {
			return
		}
		player, errPlayer := detector.GetPlayerOrCreate(ctx, sid, false)
		if errPlayer != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		player.Notes = opts.Note
		if errSave := detector.Store().SavePlayer(ctx, player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

func postMarkPlayer() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		var opts postMarkPlayerOpts
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

func updateWhitelistPlayer(enable bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
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
