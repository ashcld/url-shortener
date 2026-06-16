package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashcloud/url-shortener/internal/config"
	httphandler "github.com/ashcloud/url-shortener/internal/handler/http"
	"github.com/ashcloud/url-shortener/internal/kafka"
	"github.com/ashcloud/url-shortener/internal/service"
	pgstore "github.com/ashcloud/url-shortener/internal/storage/postgres"
	redisstore "github.com/ashcloud/url-shortener/internal/storage/redis"
	"github.com/ashcloud/url-shortener/pkg/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.App.Env)

	if err := run(cfg, log); err != nil {
		log.Error("application error", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// PostgreSQL
	log.Info("connecting to postgres")
	pool, err := pgstore.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	log.Info("running migrations")
	if err = pgstore.RunMigrations(cfg.Postgres.DSN(), "migrations"); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	log.Info("migrations done")

	// Redis
	log.Info("connecting to redis")
	redisClient, err := redisstore.New(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer redisClient.Close()

	// Kafka producer
	log.Info("initializing kafka producer")
	clickProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicClicks)
	defer func() {
		if err := clickProducer.Close(); err != nil {
			log.Warn("kafka producer close error", "err", err)
		}
	}()

	// Storage layer
	urlRepo := pgstore.NewURLRepo(pool)
	urlCache := redisstore.NewURLCache(redisClient, cfg.Redis.CacheTTL)

	// Service layer
	urlSvc := service.NewURLService(urlRepo, urlCache, clickProducer, log, cfg.Short.CodeLength)

	// HTTP
	urlHandler := httphandler.NewURLHandler(urlSvc, cfg.App.BaseURL)
	handler := httphandler.NewHandler(urlHandler, log)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port),
		Handler:      handler.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("http server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("server stopped")
	return nil
}
