package detector

import (
	"context"
	"net"
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/assets"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Web struct {
	*http.Server
	Engine *gin.Engine
}

func NewWeb(detector *Detector) (*Web, error) {
	engine := createRouter(detector.log, detector.Settings().RunMode)
	if errRoutes := setupRoutes(engine, detector); errRoutes != nil {
		return nil, errRoutes
	}

	httpServer := &http.Server{
		Addr:         detector.Settings().HTTPListenAddr,
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

func bind(ctx *gin.Context, receiver any, log *zap.Logger) bool {
	if errBind := ctx.BindJSON(&receiver); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})

		log.Error("Received malformed request", zap.Error(errBind))

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

func createRouter(logger *zap.Logger, mode RunModes) *gin.Engine {
	switch mode {
	case ModeRelease:
		gin.SetMode(gin.ReleaseMode)
	case ModeTest:
		gin.SetMode(gin.TestMode)
	case ModeDebug:
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/state"},
	}))

	engine.Use(ginzap.RecoveryWithZap(logger, true))

	_ = engine.SetTrustedProxies(nil)

	return engine
}

// setupRoutes configures the routes. If the `release` tag is enabled, serves files from the embedded assets
// in the binary.
func setupRoutes(engine *gin.Engine, detector *Detector) error {
	if errStatic := assets.StaticRoutes(engine, detector.Settings().RunMode == ModeTest); errStatic != nil {
		return errors.Wrap(errStatic, "Failed to setup static routes")
	}

	engine.GET("/state", getState(detector))
	engine.GET("/messages/:steam_id", getMessages(detector))
	engine.GET("/names/:steam_id", getNames(detector))
	engine.POST("/mark/:steam_id", postMarkPlayer(detector))
	engine.DELETE("/mark/:steam_id", deleteMarkedPlayer(detector))
	engine.GET("/settings", getSettings(detector))
	engine.GET("/launch", getLaunch(detector))
	engine.PUT("/settings", putSettings(detector))
	engine.POST("/whitelist/:steam_id", updateWhitelistPlayer(detector, true))
	engine.DELETE("/whitelist/:steam_id", updateWhitelistPlayer(detector, false))
	engine.POST("/notes/:steam_id", postNotes(detector))
	engine.POST("/callvote/:steam_id/:reason", callVote(detector))

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

func steamIDParam(ctx *gin.Context, log *zap.Logger) (steamid.SID64, bool) {
	steamID := steamid.New(ctx.Param("steam_id"))
	if !steamID.Valid() {
		responseErr(ctx, http.StatusBadRequest, nil)
		log.Error("Failed to parse steam id param", zap.String("steam_id", ctx.Param("steam_id")))

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
