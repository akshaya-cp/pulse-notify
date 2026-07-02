package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/akshaya-cp/golang_project/internal/auth"
	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/database"
	"github.com/akshaya-cp/golang_project/internal/events"
	"github.com/akshaya-cp/golang_project/internal/handler"
	"github.com/akshaya-cp/golang_project/internal/logger"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/akshaya-cp/golang_project/internal/router"
	"github.com/akshaya-cp/golang_project/internal/server"
	"github.com/akshaya-cp/golang_project/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	appLog := logger.New(cfg.AppEnv, cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		appLog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	if err := database.Migrate(ctx, db); err != nil {
		appLog.Error("database migration failed", "error", err)
		db.Close()
		os.Exit(1)
	}
	appLog.Info("database connected and migrated")

	redisClient, err := cache.Connect(ctx, cfg.RedisURL)
	if err != nil {
		appLog.Error("redis connection failed", "error", err)
		db.Close()
		os.Exit(1)
	}
	appLog.Info("redis connected")

	producer := events.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	appLog.Info("kafka producer ready", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopic)

	tokenMgr := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	userRepo := repository.NewUserRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	authSvc := service.NewAuthService(userRepo, tokenMgr, redisClient, cfg.JWTRefreshTTL)
	notifSvc := service.NewNotificationService(notifRepo, producer, redisClient, appLog)

	deps := router.Deps{
		Config:       cfg,
		Log:          appLog,
		DB:           db,
		Cache:        redisClient,
		TokenMgr:     tokenMgr,
		Health:       handler.NewHealthHandler(db, redisClient),
		Auth:         handler.NewAuthHandler(authSvc),
		Notification: handler.NewNotificationHandler(notifSvc),
	}

	srv := server.New(cfg, appLog, deps)

	err = srv.Run()

	// Release resources the server did not own.
	if cerr := producer.Close(); cerr != nil {
		appLog.Warn("kafka producer close failed", "error", cerr)
	}
	if cerr := redisClient.Close(); cerr != nil {
		appLog.Warn("redis close failed", "error", cerr)
	}

	if err != nil {
		appLog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
