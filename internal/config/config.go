package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application settings loaded from environment variables.
type Config struct {
	AppEnv   string
	HTTPHost string
	HTTPPort int
	LogLevel string

	DatabaseURL string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	RedisURL string

	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string

	WorkerConcurrency int
	WorkerMaxRetries  int

	RateLimitRequests int
	RateLimitWindow   time.Duration
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

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
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

	workerConcurrency, err := strconv.Atoi(getEnv("WORKER_CONCURRENCY", "4"))
	if err != nil || workerConcurrency < 1 {
		return nil, fmt.Errorf("invalid WORKER_CONCURRENCY: must be a positive integer")
	}

	workerMaxRetries, err := strconv.Atoi(getEnv("WORKER_MAX_RETRIES", "3"))
	if err != nil || workerMaxRetries < 0 {
		return nil, fmt.Errorf("invalid WORKER_MAX_RETRIES: must be a non-negative integer")
	}

	rateLimitRequests, err := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	if err != nil || rateLimitRequests < 0 {
		return nil, fmt.Errorf("invalid RATE_LIMIT_REQUESTS: must be a non-negative integer")
	}

	rateLimitWindow, err := time.ParseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_WINDOW: %w", err)
	}

	return &Config{
		AppEnv:   appEnv,
		HTTPHost: getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort: port,
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DatabaseURL: databaseURL,

		JWTSecret:     jwtSecret,
		JWTAccessTTL:  accessTTL,
		JWTRefreshTTL: refreshTTL,

		RedisURL: getEnv("REDIS_URL", "redis://127.0.0.1:6379/0"),

		KafkaBrokers: splitAndTrim(getEnv("KAFKA_BROKERS", "127.0.0.1:9092")),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "notifications"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "pulse-notify-workers"),

		WorkerConcurrency: workerConcurrency,
		WorkerMaxRetries:  workerMaxRetries,

		RateLimitRequests: rateLimitRequests,
		RateLimitWindow:   rateLimitWindow,
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

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
