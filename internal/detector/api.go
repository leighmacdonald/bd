package detector

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type jsConfig struct {
	SiteName string `json:"siteName"`
}

func NewApi(bd *BD) *Api {
	logger := bd.logger.Named("api")

	absStaticPath, errStaticPath := filepath.Abs("./internal/detector/dist")
	if errStaticPath != nil {
		logger.Fatal("Invalid static path", zap.Error(errStaticPath))
	}

	router := gin.New()

	router.Use(ErrorHandler(logger))
	router.Use(gin.Recovery())

	router.StaticFS("/dist", http.Dir(absStaticPath))
	router.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	router.GET("/players", getPlayers())
	// These should match routes defined in the frontend. This allows us to use the browser
	// based routing
	jsRoutes := []string{"/"}
	for _, rt := range jsRoutes {
		router.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName: "bd",
			})
		})
	}
	httpServer := &http.Server{
		Addr:         "localhost:8900",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	api := Api{
		bd:         bd,
		logger:     logger,
		httpServer: httpServer,
	}
	return &api
}

func createTestPlayer() model.PlayerCollection {
	var randPlayer = func(userId int64) *model.Player {
		team := model.Blu
		if userId%2 == 0 {
			team = model.Red
		}
		sid := steamid.SID64(76561197960265728 + userId)
		return &model.Player{
			SteamIdString:    sid.String(),
			Name:             golib.RandomString(40),
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
			Match:            nil,
		}
	}
	var testPlayers model.PlayerCollection
	for i := int64(0); i < 24; i++ {
		p := randPlayer(i)
		switch i {
		case 1:
			p.NumberOfVACBans = 2
			last := time.Now().AddDate(-1, 0, 0)
			p.LastVACBanOn = &last
		case 4:
			p.Match = &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"cheater"},
				MatcherType: "string",
			}
		case 6:
			p.Match = &rules.MatchResult{
				Origin:      "Test Rules List",
				Attributes:  []string{"other"},
				MatcherType: "string",
			}

		case 7:
			p.Team = model.Spec
		}
		testPlayers = append(testPlayers, p)
	}
	return testPlayers
}

func getPlayers() gin.HandlerFunc {
	testPlayers := createTestPlayer()
	return func(ctx *gin.Context) {
		if _, isTest := os.LookupEnv("TEST"); isTest {
			responseOK(ctx, http.StatusOK, testPlayers)
			return
		}
		playersMu.RLock()
		defer playersMu.RUnlock()
		p := model.PlayerCollection{}
		if players != nil {
			p = players
		}
		responseOK(ctx, http.StatusOK, p)
	}
}

func responseErr(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}

func responseOK(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}

type Api struct {
	bd         *BD
	logger     *zap.Logger
	httpServer *http.Server
}

func (api *Api) ListenAndServe(ctx context.Context) error {
	api.logger.Info("Service status changed", zap.String("state", "ready"))
	defer api.logger.Info("Service status changed", zap.String("state", "stopped"))
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if errShutdown := api.httpServer.Shutdown(shutdownCtx); errShutdown != nil {
			api.logger.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()
	return api.httpServer.ListenAndServe()
}

func (api *Api) bind(ctx *gin.Context, recveiver any) bool {
	if errBind := ctx.BindJSON(&recveiver); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		api.logger.Error("Invalid request", zap.Error(errBind))
		return false
	}
	return true
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}
