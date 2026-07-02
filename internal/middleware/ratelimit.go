package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/gin-gonic/gin"
)

// RateLimit applies a Redis-backed fixed-window rate limit keyed by client IP.
// Using Redis keeps the limit consistent across multiple API instances, which
// is essential for a horizontally scaled deployment.
func RateLimit(c *cache.Client, log *slog.Logger, limit int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// A limit of zero (or missing cache) disables rate limiting.
		if c == nil || limit <= 0 {
			ctx.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s", ctx.ClientIP())
		count, err := c.IncrWithWindow(ctx.Request.Context(), key, window)
		if err != nil {
			// Fail open: a Redis blip should not take down the API.
			log.Warn("rate limiter unavailable, allowing request", "error", err)
			ctx.Next()
			return
		}

		ctx.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		ctx.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if int(count) > limit {
			ctx.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, slow down",
			})
			return
		}

		ctx.Next()
	}
}
