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

func getMessages(d *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		messages, errMsgs := d.dataStore.FetchMessages(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, messages)
	}
}

func getNames(d *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		messages, errMsgs := d.dataStore.FetchNames(ctx, sid)
		if errMsgs != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, messages)
	}
}

func getPlayers(d *Detector) gin.HandlerFunc {
	_, isTest := os.LookupEnv("TEST")
	testPlayers := createTestPlayers(d, 24)
	if isTest {
		go func() {
			t := time.NewTicker(time.Second * 5)
			for {
				<-t.C
				for _, p := range testPlayers {
					p.UpdatedOn = time.Now()
					p.Connected += 5
					p.Ping = rand.Intn(110)
					p.Kills = rand.Intn(50)
					p.Deaths = rand.Intn(30)
				}
			}
		}()
	}
	return func(ctx *gin.Context) {
		if isTest {
			for _, plr := range d.Players() {
				plr.UpdatedOn = time.Now()
			}
			players := d.Players()
			responseOK(ctx, http.StatusOK, players)
			return
		}
		players := d.Players()
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

func getSettings(d *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wus := webUserSettings{
			UserSettings: d.settings,
			UniqueTags:   rules.UniqueTags(),
		}
		responseOK(ctx, http.StatusOK, wus)
	}
}

func postSettings(d *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var wus webUserSettings
		if !bind(ctx, &wus) {
			return
		}
		wus.RWMutex = &sync.RWMutex{}
		// TODO Proper validation
		d.settings = wus.UserSettings
		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postNotesOpts struct {
	Note string `json:"note"`
}

func postNotes(d *Detector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		var opts postNotesOpts
		if !bind(ctx, &opts) {
			return
		}
		player, errPlayer := d.GetPlayerOrCreate(ctx, sid, false)
		if errPlayer != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		player.Notes = opts.Note
		if errSave := d.dataStore.SavePlayer(ctx, player); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)
	}
}

type postMarkPlayerOpts struct {
	Attrs []string `json:"attrs"`
}

func postMarkPlayer(d *Detector) gin.HandlerFunc {
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
		if errMark := d.Mark(ctx, sid, opts.Attrs); errMark != nil {
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

func updateWhitelistPlayer(d *Detector, enable bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, sidOk := steamIdParam(ctx)
		if !sidOk {
			return
		}
		if errWl := d.Whitelist(ctx, sid, enable); errWl != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)
	}
}
