package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler serves liveness/readiness style checks for load balancers and orchestrators.
type HealthHandler struct {
	startedAt time.Time
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{startedAt: time.Now()}
}

// Check returns basic service health. DB/Kafka checks come in later phases.
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "pulse-notify-api",
		"uptime":    time.Since(h.startedAt).Round(time.Second).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
