package router

import (
	"log/slog"
	"time"

	"github.com/akshaya-cp/golang_project/internal/auth"
	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/handler"
	"github.com/akshaya-cp/golang_project/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps groups handlers and shared dependencies for route registration.
type Deps struct {
	Config       *config.Config
	Log          *slog.Logger
	DB           *pgxpool.Pool
	Cache        *cache.Client
	TokenMgr     *auth.TokenManager
	Health       *handler.HealthHandler
	Auth         *handler.AuthHandler
	Notification *handler.NotificationHandler
}

// New wires HTTP routes and middleware for the API server.
func New(deps Deps) *gin.Engine {
	if !deps.Config.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(deps.Log))
	r.Use(middleware.RateLimit(deps.Cache, deps.Log, deps.Config.RateLimitRequests, deps.Config.RateLimitWindow))

	r.GET("/health", deps.Health.Check)

	jwtAuth := middleware.JWT(deps.TokenMgr)

	v1 := r.Group("/api/v1")
	{
		authRoutes := v1.Group("/auth")
		authRoutes.POST("/signup", deps.Auth.Signup)
		authRoutes.POST("/login", deps.Auth.Login)
		authRoutes.POST("/refresh", deps.Auth.Refresh)
		authRoutes.POST("/logout", deps.Auth.Logout)

		protected := v1.Group("")
		protected.Use(jwtAuth)
		{
			protected.GET("/me", deps.Auth.Me)

			protected.POST("/notifications", deps.Notification.Create)
			protected.GET("/notifications", deps.Notification.List)
			protected.GET("/notifications/:id", deps.Notification.Get)
		}

		admin := v1.Group("/admin")
		admin.Use(jwtAuth, middleware.RequireRole("admin"))
		{
			admin.GET("/notifications", deps.Notification.ListAll)
		}
	}

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

var timeNow = func() time.Time { return time.Now() }

func timeSinceMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
