package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshaya-cp/golang_project/internal/config"
	"github.com/akshaya-cp/golang_project/internal/router"
	"github.com/gin-gonic/gin"
)

// Server wraps the HTTP server and Gin engine.
type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	engine *gin.Engine
	http   *http.Server
}

func New(cfg *config.Config, log *slog.Logger) *Server {
	engine := router.New(cfg, log)

	return &Server{
		cfg:    cfg,
		log:    log,
		engine: engine,
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      engine,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Run starts listening and shuts down gracefully on SIGINT/SIGTERM.
func (s *Server) Run() error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info("starting http server", "addr", s.cfg.Addr(), "env", s.cfg.AppEnv)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		s.log.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	s.log.Info("server stopped gracefully")
	return nil
}
