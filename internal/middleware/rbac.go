package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleFromContext returns the role attached by the JWT middleware.
func RoleFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextRoleKey)
	if !ok {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}

// RequireRole restricts a route to callers holding one of the allowed roles.
// It must run after the JWT middleware so the role is present in the context.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role, ok := RoleFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if _, permitted := allowed[role]; !permitted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}
