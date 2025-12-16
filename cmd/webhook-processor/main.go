package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/application/workers"
	"webhook-processor/internal/config"
	"webhook-processor/internal/infrastructure/database"
	"webhook-processor/internal/infrastructure/metrics"
	"webhook-processor/internal/infrastructure/repositories"
	"webhook-processor/internal/infrastructure/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger()
	level.Info(logger).Log("msg", "starting webhook processor", "version", "1.0.0")

	// Initialize database
	db, err := database.NewDatabase(cfg)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize database", "error", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "database connection established")

	// Initialize repositories
	webhookQueueRepo, err := repositories.NewWebhookQueueRepository(db)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create webhook queue repository", "error", err)
		os.Exit(1)
	}
	webhookConfigRepo, err := repositories.NewWebhookConfigRepository(db)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create webhook config repository", "error", err)
		os.Exit(1)
	}

	// Initialize metrics
	webhookMetrics := metrics.NewWebhookMetrics()

	// Initialize services
	webhookService := services.NewWebhookService(cfg.HTTPClient)

	// Initialize use cases
	webhookProcessor := usecases.NewWebhookProcessor(
		webhookQueueRepo,
		webhookConfigRepo,
		webhookService,
		logger,
	)

	// Initialize worker pool
	workerPoolConfig := config.GetDefaultWorkerPoolConfig()
	workerPool := workers.NewWorkerPool(webhookProcessor, logger, workerPoolConfig, webhookMetrics)

	// Start worker pool
	if err := workerPool.Start(); err != nil {
		level.Error(logger).Log("msg", "failed to start worker pool", "error", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "worker pool started successfully")

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		level.Info(logger).Log("msg", "starting metrics server", "port", 8081)
		if err := http.ListenAndServe(":8081", nil); err != nil {
			level.Error(logger).Log("msg", "metrics server failed", "error", err)
		}
	}()

	// Setup graceful shutdown

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	level.Info(logger).Log("msg", "shutdown signal received, stopping worker pool")

	// Stop worker pool
	if err := workerPool.Stop(); err != nil {
		level.Error(logger).Log("msg", "failed to stop worker pool", "error", err)
	} else {
		level.Info(logger).Log("msg", "worker pool stopped successfully")
	}

	// Close database connection
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	level.Info(logger).Log("msg", "webhook processor shutdown complete")
}

// setupLogger creates and configures a logger with default settings
func setupLogger() log.Logger {
	// Use text format logger with info level by default
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	logger = level.NewFilter(logger, level.AllowInfo())
	return logger
}
