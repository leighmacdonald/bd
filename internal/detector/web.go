package detector

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Web struct {
	*http.Server
	Engine *gin.Engine
}

func NewWeb(detector *Detector) (*Web, error) {
	engine := createRouter(detector.log)
	if errRoutes := setupRoutes(engine, detector); errRoutes != nil {
		return nil, errRoutes
	}

	httpServer := &http.Server{
		Addr:         detector.settings.HTTPListenAddr,
		Handler:      engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Web{
		Server: httpServer,
		Engine: engine,
	}, nil
}

func (w *Web) startWeb(ctx context.Context) error {
	w.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	if errServe := w.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		return errors.Wrap(errServe, "HTTP server returned error")
	}

	return nil
}

func bind(ctx *gin.Context, receiver any) bool {
	if errBind := ctx.BindJSON(&receiver); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})

		return false
	}

	return true
}

func responseErr(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}

func responseOK(ctx *gin.Context, status int, data any) {
	if data == nil {
		data = []string{}
	}

	ctx.JSON(status, data)
}

func createRouter(logger *zap.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		// SkipPaths:  []string{"/players"},
	}))

	engine.Use(ginzap.RecoveryWithZap(logger, true))

	_ = engine.SetTrustedProxies(nil)

	return engine
}

func setupRoutes(engine *gin.Engine, detector *Detector) error {
	if detector.settings.RunMode != gin.TestMode {
		absStaticPath, errStaticPath := filepath.Abs("./internal/detector/dist")
		if errStaticPath != nil {
			return errors.Wrap(errStaticPath, "Failed to setup static paths")
		}

		engine.StaticFS("/dist", http.Dir(absStaticPath))
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	engine.GET("/players", getPlayers(detector))
	engine.GET("/messages/:steam_id", getMessages(detector))
	engine.GET("/names/:steam_id", getNames(detector))
	engine.POST("/mark/:steam_id", postMarkPlayer(detector))
	engine.GET("/settings", getSettings(detector))
	engine.POST("/settings", postSettings(detector))
	engine.POST("/whitelist/:steam_id", updateWhitelistPlayer(detector, true))
	engine.DELETE("/whitelist/:steam_id", updateWhitelistPlayer(detector, false))
	engine.POST("/notes/:steam_id", postNotes(detector))

	// These should match any routes defined in the frontend. This allows us to use the browser
	// based routing
	jsRoutes := []string{"/"}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName: "bd",
			})
		})
	}

	return nil
}

type jsConfig struct {
	SiteName string `json:"site_name"`
}

// nolint:gosec
func CreateTestPlayers(detector *Detector, count int) store.PlayerCollection {
	idIdx := 0
	knownIds := steamid.Collection{
		"76561197998365611", "76561197977133523", "76561198065825165", "76561198004429398", "76561198182505218",
		"76561197989961569", "76561198183927541", "76561198005026984", "76561197997861796", "76561198377596915",
		"76561198336028289", "76561198066637626", "76561198818013048", "76561198196411029", "76561198079544034",
		"76561198008337801", "76561198042902038", "76561198013287458", "76561198038487121", "76561198046766708",
		"76561197963310062", "76561198017314810", "76561197967842214", "76561197984047970", "76561198020124821",
		"76561198010868782", "76561198022397372", "76561198016314731", "76561198087124802", "76561198024022137",
		"76561198015577906", "76561197997861796",
	}

	randPlayer := func(userId int64) *store.Player {
		team := store.Blu
		if userId%2 == 0 {
			team = store.Red
		}

		player, errP := detector.GetPlayerOrCreate(context.TODO(), knownIds[idIdx], true)
		if errP != nil {
			panic(errP)
		}

		player.KillsOn = rand.Intn(20)
		player.RageQuits = rand.Intn(10)
		player.DeathsBy = rand.Intn(20)
		player.Team = team
		player.Connected = float64(rand.Intn(3600))
		player.UserID = userId
		player.Ping = rand.Intn(150)
		player.Kills = rand.Intn(50)
		player.Deaths = rand.Intn(300)
		idIdx++

		return player
	}

	var testPlayers store.PlayerCollection

	for i := 0; i < count; i++ {
		player := randPlayer(int64(i))

		switch i {
		case 1:
			player.NumberOfVACBans = 2
			player.Notes = "User notes \ngo here"
			last := time.Now().AddDate(-1, 0, 0)
			player.LastVACBanOn = &last
		case 4:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			player.Matches = append(player.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			})

		case 7:
			player.Team = store.Spec
		}

		testPlayers = append(testPlayers, player)
	}

	return testPlayers
}

func steamIDParam(ctx *gin.Context) (steamid.SID64, bool) {
	steamID := steamid.New(ctx.Param("steam_id"))
	if !steamID.Valid() {
		responseErr(ctx, http.StatusBadRequest, nil)

		return "", false
	}

	return steamID, true
}

func (w *Web) Stop(ctx context.Context) error {
	if w.Server == nil {
		return nil
	}

	timeout, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	if errShutdown := w.Server.Shutdown(timeout); errShutdown != nil {
		return errors.Wrap(errShutdown, "Failed to shutdown http service")
	}

	return nil
}
