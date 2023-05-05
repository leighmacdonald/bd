package web

import (
	"context"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
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

func Setup() {
	logger = detector.Logger().Named("api")
	engine := createRouter()
	if errRoutes := setupRoutes(engine); errRoutes != nil {
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

func createRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), ginzap.GinzapWithConfig(logger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/players"},
	}))
	_ = engine.SetTrustedProxies(nil)
	return engine
}

func setupRoutes(engine *gin.Engine) error {
	absStaticPath, errStaticPath := filepath.Abs("./internal/web/dist")
	if errStaticPath != nil {
		return errors.Wrap(errStaticPath, "Failed to setup static paths")
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))
	engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	engine.GET("/players", getPlayers())
	engine.POST("/mark", postMarkPlayer())
	// These should match routes defined in the frontend. This allows us to use the browser
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

func createTestPlayer() store.PlayerCollection {
	var randPlayer = func(userId int64) *store.Player {
		team := store.Blu
		if userId%2 == 0 {
			team = store.Red
		}
		sid := steamid.SID64(76561197960265728 + userId)
		return &store.Player{
			SteamIdString:    sid.String(),
			Name:             util.RandomString(40),
			CreatedOn:        time.Now(),
			UpdatedOn:        time.Now(),
			ProfileUpdatedOn: time.Now(),
			KillsOn:          rand.Intn(20),
			RageQuits:        rand.Intn(10),
			DeathsBy:         rand.Intn(20),
			Notes:            "User notes \ngo here",
			Whitelisted:      false,
			RealName:         "Real Name Goes Here",
			NamePrevious:     "",
			AccountCreatedOn: time.Time{},
			Visibility:       0,
			AvatarHash:       "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb",
			CommunityBanned:  false,
			NumberOfVACBans:  0,
			LastVACBanOn:     nil,
			NumberOfGameBans: 0,
			EconomyBan:       false,
			Team:             team,
			Connected:        float64(rand.Intn(3600)),
			UserId:           userId,
			Ping:             rand.Intn(150),
			Kills:            rand.Intn(50),
			Deaths:           rand.Intn(300),
			Matches:          []*rules.MatchResult{},
		}
	}
	var testPlayers store.PlayerCollection
	for i := int64(0); i < 24; i++ {
		p := randPlayer(i)
		switch i {
		case 1:
			p.NumberOfVACBans = 2
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
