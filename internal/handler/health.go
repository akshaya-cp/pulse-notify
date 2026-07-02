package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler serves liveness/readiness style checks.
type HealthHandler struct {
	startedAt time.Time
	db        *pgxpool.Pool
	cache     *cache.Client
}

func NewHealthHandler(db *pgxpool.Pool, c *cache.Client) *HealthHandler {
	return &HealthHandler{
		startedAt: time.Now(),
		db:        db,
		cache:     c,
	}
}

// Check returns service health including PostgreSQL and Redis pings.
func (h *HealthHandler) Check(c *gin.Context) {
	status := "ok"
	httpStatus := http.StatusOK

	dbStatus := "up"
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.db.Ping(ctx); err != nil {
			status = "degraded"
			dbStatus = "down"
			httpStatus = http.StatusServiceUnavailable
		}
	} else {
		dbStatus = "not_configured"
	}

	redisStatus := "up"
	if h.cache != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.cache.Ping(ctx); err != nil {
			status = "degraded"
			redisStatus = "down"
			httpStatus = http.StatusServiceUnavailable
		}
	} else {
		redisStatus = "not_configured"
	}

	c.JSON(httpStatus, gin.H{
		"status":    status,
		"service":   "pulse-notify-api",
		"database":  dbStatus,
		"redis":     redisStatus,
		"uptime":    time.Since(h.startedAt).Round(time.Second).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
