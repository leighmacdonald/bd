package web

import (
	"context"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"path/filepath"
	"time"
)

var (
	router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
)

func Setup() {
	logger = detector.Logger().Named("api")
	engine := createRouter()
	if errRoutes := setupRoutes(engine); errRoutes != nil {
		logger.Panic("Failed to setup routes", zap.Error(errRoutes))
	}
	router = engine
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
