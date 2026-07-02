package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/database"
	"github.com/akshaya-cp/golang_project/internal/events"
	"github.com/akshaya-cp/golang_project/internal/logger"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/akshaya-cp/golang_project/internal/worker"
)

// The worker is a separate process from the API. It consumes notification
// events from Kafka and processes them concurrently, which is what makes the
// platform "distributed": producers (API) and consumers (workers) scale
// independently behind the message broker.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	appLog := logger.New(cfg.AppEnv, cfg.LogLevel)

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(connectCtx, cfg.DatabaseURL)
	if err != nil {
		appLog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(connectCtx, db); err != nil {
		appLog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	consumer := events.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	notifRepo := repository.NewNotificationRepository(db)
	notifier := worker.NewSimulatedNotifier(0.25)
	pool := worker.NewPool(consumer, notifRepo, notifier, appLog, cfg.WorkerConcurrency, cfg.WorkerMaxRetries)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLog.Info("notification worker starting",
		"brokers", cfg.KafkaBrokers,
		"topic", cfg.KafkaTopic,
		"group", cfg.KafkaGroupID,
	)

	if err := pool.Run(ctx); err != nil {
		appLog.Error("worker pool exited with error", "error", err)
		os.Exit(1)
	}

	appLog.Info("notification worker stopped gracefully")
}
