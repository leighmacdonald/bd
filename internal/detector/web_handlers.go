package detector

import (
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

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

func getPlayers(detector *Detector) gin.HandlerFunc {
	_, isTest := os.LookupEnv("TEST")
	testPlayers := createTestPlayers(detector, 24)

	if isTest {
		go func() {
			updateTicker := time.NewTicker(time.Second * 5)

			for {
				<-updateTicker.C

				for _, p := range testPlayers {
					p.UpdatedOn = time.Now()
					p.Connected += 5
					p.Ping = rand.Intn(110)  //nolint:gosec
					p.Kills = rand.Intn(50)  //nolint:gosec
					p.Deaths = rand.Intn(30) //nolint:gosec
				}
			}
		}()
	}

	return func(ctx *gin.Context) {
		if isTest {
			for _, plr := range detector.Players() {
				plr.UpdatedOn = time.Now()
			}

			players := detector.Players()
			responseOK(ctx, http.StatusOK, players)

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
	*UserSettings
	UniqueTags []string `json:"unique_tags"`
}

func getSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wus := webUserSettings{
			UserSettings: detector.settings,
			UniqueTags:   detector.rules.UniqueTags(),
		}

		responseOK(ctx, http.StatusOK, wus)
	}
}

func postSettings(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var wus webUserSettings
		if !bind(ctx, &wus) {
			return
		}

		wus.RWMutex = &sync.RWMutex{}
		// TODO Proper validation
		detector.settings = wus.UserSettings

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postNotesOpts struct {
	Note string `json:"note"`
}

func postNotes(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
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
		if errSave := detector.dataStore.SavePlayer(ctx, player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

func postMarkPlayer(detector *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIDParam(ctx)
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
