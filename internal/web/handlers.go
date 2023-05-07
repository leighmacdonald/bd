package web

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net/http"
	"os"
)

func getPlayers() gin.HandlerFunc {
	testPlayers := createTestPlayer()
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

func getSettings() gin.HandlerFunc {
	type webUserSettings struct {
		*detector.UserSettings
		UniqueTags []string `json:"unique_tags"`
	}
	return func(ctx *gin.Context) {
		wus := webUserSettings{
			UserSettings: detector.Settings(),
			UniqueTags:   rules.UniqueTags(),
		}
		responseOK(ctx, http.StatusOK, wus)
	}
}

func postMarkPlayer() gin.HandlerFunc {
	type postOpts struct {
		SteamID steamid.SID64 `json:"steamID"`
		Attrs   []string      `json:"attrs"`
	}
	return func(ctx *gin.Context) {
		var po postOpts
		if !bind(ctx, &po) {
			return
		}
	}
}
