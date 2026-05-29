package main

import (
	"log"
	"os"

	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/logger"
	"github.com/akshaya-cp/golang_project/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	appLog := logger.New(cfg.AppEnv, cfg.LogLevel)

	srv := server.New(cfg, appLog)
	if err := srv.Run(); err != nil {
		appLog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
