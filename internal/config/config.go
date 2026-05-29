package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application settings loaded from environment variables.
type Config struct {
	AppEnv   string
	HTTPHost string
	HTTPPort int
	LogLevel string
}

// Load reads configuration from the environment.
// In development, it also loads variables from a .env file if present.
func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	// .env is optional; production typically injects env vars directly.
	if appEnv == "development" {
		_ = godotenv.Load()
	}

	port, err := strconv.Atoi(getEnv("HTTP_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
	}

	return &Config{
		AppEnv:   appEnv,
		HTTPHost: getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort: port,
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}, nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
