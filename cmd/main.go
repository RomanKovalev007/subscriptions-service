package main

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

	"github.com/RomanKovalev007/subscriptions-service/internal/config"
	"github.com/RomanKovalev007/subscriptions-service/internal/repository"
	"github.com/RomanKovalev007/subscriptions-service/internal/service"
	"github.com/RomanKovalev007/subscriptions-service/internal/transport"
	"github.com/RomanKovalev007/subscriptions-service/migrations"
	"github.com/RomanKovalev007/subscriptions-service/pkg/postgres"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("error load config: %s\n", err)
		os.Exit(1)
	}

	logger := buildLogger(cfg.Server.LogLevel)

	ctx := context.Background()

	pool, err := postgres.New(ctx, cfg.Postgres.DSN(), postgres.PoolConfig{
		MaxConns:        cfg.Postgres.MaxConns,
		MinConns:        cfg.Postgres.MinConns,
		MaxConnLifetime: cfg.Postgres.MaxConnLifetimeDuration(),
		MaxConnIdleTime: cfg.Postgres.MaxConnIdleTimeDuration(),
	})
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(cfg.Postgres.DSN(), logger); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	repo := repository.NewSubscriptionRepository(pool)
	svc := service.NewSubscriptionService(repo, logger)
	h := transport.New(svc, logger)

	addr := cfg.Server.Host + ":" + cfg.Server.Port

	srv := &http.Server{
		Addr:         addr,
		Handler:      h.Router(),
		ReadTimeout:  cfg.Server.ReadTimeoutDuration(),
		WriteTimeout: cfg.Server.WriteTimeoutDuration(),
		IdleTimeout:  cfg.Server.IdleTimeoutDuration(),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func runMigrations(dsn string, logger *slog.Logger) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	logger.Info("migrations applied")
	return nil
}

func buildLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}