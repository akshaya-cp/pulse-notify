package router

import (
	"log/slog"
	"time"

	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/handler"
	"github.com/gin-gonic/gin"
)

// New wires HTTP routes and middleware for the API server.
func New(cfg *config.Config, log *slog.Logger) *gin.Engine {
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(log))

	health := handler.NewHealthHandler()
	r.GET("/health", health.Check)

	return r
}

func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := timeNow()
		c.Next()

		log.Info("http request",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", timeSinceMs(start),
			"client_ip", c.ClientIP(),
		)
	}
}

// Small indirections keep handler tests simple later without pulling in time in every test.
var timeNow = func() time.Time { return time.Now() }

func timeSinceMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
