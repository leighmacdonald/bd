package detector

import (
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
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
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:8900"}
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = false
	corsConfig.AllowCredentials = false

	router.Use(cors.New(corsConfig))
	router.Use(ErrorHandler(logger))
	router.Use(gin.Recovery())

	router.StaticFS("/dist", http.Dir(absStaticPath))
	router.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	router.GET("/players", getPlayers(bd))
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

func getPlayers(bd *BD) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bd.playersMu.RLock()
		defer bd.playersMu.RUnlock()
		responseOK(ctx, http.StatusOK, bd.players)
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

func (api *Api) bind(ctx *gin.Context, recv any) bool {
	if errBind := ctx.BindJSON(&recv); errBind != nil {
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
