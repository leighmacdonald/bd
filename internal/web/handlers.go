package web

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net/http"
	"os"
	"sync"
)

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

type postMarkPlayerOpts struct {
	SteamID string   `json:"steam_id"`
	Attrs   []string `json:"attrs"`
}

func postMarkPlayer() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts postMarkPlayerOpts
		if !bind(ctx, &opts) {
			return
		}
		sid, errSid := steamid.StringToSID64(opts.SteamID)
		if errSid != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if errMark := detector.Mark(sid, opts.Attrs); errMark != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)
	}
}
