package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"mira-tp4/internal/core"
	"mira-tp4/internal/enrichment"
	apihttp "mira-tp4/internal/http"
	"mira-tp4/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	pool, err := postgres.NewPool(ctx, cfg.databaseURL)
	if err != nil {
		logger.Error("database_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		logger.Error("migrations_failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations_applied")

	repo := postgres.NewRepository(pool)

	dispatcher := enrichment.NewDispatcher(cfg.enrichmentQueueSize, logger)
	worker := enrichment.NewPool(repo, enrichment.NewNaiveEnricher(), logger, cfg.enrichmentTimeout)
	workersWG := worker.Start(dispatcher.Jobs(), cfg.enrichmentWorkers)
	logger.Info("enrichment_pool_started", "workers", cfg.enrichmentWorkers, "queue_size", cfg.enrichmentQueueSize)

	service := core.NewService(repo, dispatcher)
	router := apihttp.NewRouter(service, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.port,
		Handler: router,
	}

	go func() {
		logger.Info("server_starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("server_shutting_down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server_shutdown_error", "error", err)
	}

	// The HTTP server has stopped accepting requests, so no more jobs will
	// be enqueued: safe to close the dispatcher and let workers drain
	// whatever is still buffered, bounded by the same shutdown window.
	dispatcher.Close()
	drained := make(chan struct{})
	go func() {
		workersWG.Wait()
		close(drained)
	}()
	select {
	case <-drained:
		logger.Info("enrichment_pool_drained")
	case <-shutdownCtx.Done():
		logger.Warn("enrichment_pool_drain_timeout")
	}
}

type config struct {
	port                string
	databaseURL         string
	enrichmentWorkers   int
	enrichmentQueueSize int
	enrichmentTimeout   time.Duration
}

func loadConfig() config {
	return config{
		port:                envOr("PORT", "8080"),
		databaseURL:         envOr("DATABASE_URL", "postgres://mira:mira@localhost:5432/mira?sslmode=disable"),
		enrichmentWorkers:   envIntOr("ENRICHMENT_WORKERS", 4),
		enrichmentQueueSize: envIntOr("ENRICHMENT_QUEUE_SIZE", 100),
		enrichmentTimeout:   envDurationOr("ENRICHMENT_TIMEOUT", 10*time.Second),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
