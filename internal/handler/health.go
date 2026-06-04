package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler serves liveness/readiness style checks.
type HealthHandler struct {
	startedAt time.Time
	db        *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		startedAt: time.Now(),
		db:        db,
	}
}

// Check returns service health including a database ping when configured.
func (h *HealthHandler) Check(c *gin.Context) {
	status := "ok"
	dbStatus := "up"
	httpStatus := http.StatusOK

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

	c.JSON(httpStatus, gin.H{
		"status":    status,
		"service":   "pulse-notify-api",
		"database":  dbStatus,
		"uptime":    time.Since(h.startedAt).Round(time.Second).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
