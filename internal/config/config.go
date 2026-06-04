package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application settings loaded from environment variables.
type Config struct {
	AppEnv      string
	HTTPHost    string
	HTTPPort    int
	LogLevel    string
	DatabaseURL string
	JWTSecret   string
	JWTAccessTTL time.Duration
}

// Load reads configuration from the environment.
// In development, it also loads variables from a .env file if present.
func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	if appEnv == "development" {
		_ = godotenv.Load()
	}

	port, err := strconv.Atoi(getEnv("HTTP_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
	}

	jwtTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return &Config{
		AppEnv:       appEnv,
		HTTPHost:     getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:     port,
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		DatabaseURL:  databaseURL,
		JWTSecret:    jwtSecret,
		JWTAccessTTL: jwtTTL,
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
