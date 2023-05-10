package web

import (
	"context"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"path/filepath"
	"time"
)

var (
	router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func Init(rootLogger *zap.Logger, testMode bool) {
	logger = rootLogger.Named("api")
	engine := createRouter(testMode)
	if errRoutes := setupRoutes(engine, testMode); errRoutes != nil {
		logger.Panic("Failed to setup routes", zap.Error(errRoutes))
	}
	router = engine
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
	ctx.JSON(status, data)
}

func createRouter(testMode bool) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	if !testMode {
		engine.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
			TimeFormat: time.RFC3339,
			UTC:        true,
			SkipPaths:  []string{"/players"},
		}))
	}
	_ = engine.SetTrustedProxies(nil)
	return engine
}

func setupRoutes(engine *gin.Engine, testMode bool) error {
	if !testMode {
		absStaticPath, errStaticPath := filepath.Abs("./internal/web/dist")
		if errStaticPath != nil {
			return errors.Wrap(errStaticPath, "Failed to setup static paths")
		}
		engine.StaticFS("/dist", http.Dir(absStaticPath))
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}
	engine.GET("/players", getPlayers())
	engine.GET("/messages/:steam_id", getMessages())
	engine.GET("/names/:steam_id", getNames())
	engine.POST("/mark/:steam_id", postMarkPlayer())
	engine.GET("/settings", getSettings())
	engine.POST("/settings", postSettings())
	engine.POST("/whitelist/:steam_id", updateWhitelistPlayer(true))
	engine.DELETE("/whitelist/:steam_id", updateWhitelistPlayer(false))
	engine.POST("/notes/:steam_id", postNotes())

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
	SiteName string `json:"siteName"`
}

func createTestPlayers(count int) store.PlayerCollection {
	idIdx := 0
	knownIds := steamid.Collection{
		76561197998365611, 76561197977133523, 76561198065825165, 76561198004429398, 76561198182505218,
		76561197989961569, 76561198183927541, 76561198005026984, 76561197997861796, 76561198377596915,
		76561198336028289, 76561198066637626, 76561198818013048, 76561198196411029, 76561198079544034,
		76561198008337801, 76561198042902038, 76561198013287458, 76561198038487121, 76561198046766708,
		76561197963310062, 76561198017314810, 76561197967842214, 76561197984047970, 76561198020124821,
		76561198010868782, 76561198022397372, 76561198016314731, 76561198087124802, 76561198024022137,
		76561198015577906, 76561197997861796,
	}
	var randPlayer = func(userId int64) *store.Player {
		team := store.Blu
		if userId%2 == 0 {
			team = store.Red
		}
		p, errP := detector.GetPlayerOrCreate(context.TODO(), knownIds[idIdx], true)
		if errP != nil {
			panic(errP)
		}
		p.KillsOn = rand.Intn(20)
		p.RageQuits = rand.Intn(10)
		p.DeathsBy = rand.Intn(20)
		p.Team = team
		p.Connected = float64(rand.Intn(3600))
		p.UserId = userId
		p.Ping = rand.Intn(150)
		p.Kills = rand.Intn(50)
		p.Deaths = rand.Intn(300)
		idIdx++
		return p
	}
	var testPlayers store.PlayerCollection
	for i := 0; i < count; i++ {
		p := randPlayer(int64(i))
		switch i {
		case 1:
			p.NumberOfVACBans = 2
			p.Notes = "User notes \ngo here"
			last := time.Now().AddDate(-1, 0, 0)
			p.LastVACBanOn = &last
		case 4:
			p.Matches = append(p.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			})
		case 6:
			p.Matches = append(p.Matches, &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			})

		case 7:
			p.Team = store.Spec
		}
		testPlayers = append(testPlayers, p)
	}
	return testPlayers
}

func steamIdParam(ctx *gin.Context) (steamid.SID64, bool) {
	steamId, errSid := steamid.StringToSID64(ctx.Param("steam_id"))
	if errSid != nil {
		responseErr(ctx, http.StatusBadRequest, nil)
		return 0, false
	}
	if !steamId.Valid() {
		responseErr(ctx, http.StatusBadRequest, nil)
		return 0, false
	}
	return steamId, true
}

func Start(ctx context.Context) {
	httpServer = &http.Server{
		Addr:         "localhost:8900",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Info("Service status changed", zap.String("state", "ready"))
	defer logger.Info("Service status changed", zap.String("state", "stopped"))
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil {
			logger.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	if errServe := httpServer.ListenAndServe(); errServe != nil {
		logger.Error("HTTP server returned error", zap.Error(errServe))
	}
}
