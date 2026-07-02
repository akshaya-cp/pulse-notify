package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateRefreshToken returns a cryptographically random, URL-safe opaque
// token. Refresh tokens are opaque (not JWTs) so they can be revoked by simply
// deleting them from Redis on logout or rotation.
func GenerateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
