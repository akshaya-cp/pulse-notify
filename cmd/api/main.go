package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/akshaya-cp/golang_project/internal/auth"
	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/database"
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

	tokenMgr := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, tokenMgr)

	deps := router.Deps{
		Config:   cfg,
		Log:      appLog,
		DB:       db,
		TokenMgr: tokenMgr,
		Health:   handler.NewHealthHandler(db),
		Auth:     handler.NewAuthHandler(authSvc),
	}

	srv := server.New(cfg, appLog, deps)
	if err := srv.Run(); err != nil {
		appLog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
