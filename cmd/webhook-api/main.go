package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"webhook-processor/internal/application/services"
	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/config"
	"webhook-processor/internal/infrastructure/database"
	"webhook-processor/internal/infrastructure/repositories"
	infraServices "webhook-processor/internal/infrastructure/services"
	httpTransport "webhook-processor/internal/transport/http"
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
	level.Info(logger).Log("msg", "starting webhook API server")

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

	// Initialize infrastructure services
	webhookInfraService := infraServices.NewWebhookService(cfg.HTTPClient)

	// Initialize use cases
	webhookProcessor := usecases.NewWebhookProcessor(
		webhookQueueRepo,
		webhookConfigRepo,
		webhookInfraService,
		logger,
	)

	// Initialize application services
	appService := services.NewWebhookApplicationService(webhookProcessor)

	// Create HTTP transport service
	httpService := httpTransport.NewService(appService)

	// Create HTTP handler with all routes and middleware
	router := httpTransport.NewHTTPHandler(httpService, log.With(logger, "component", "http"))

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.ReadTimeout,
		WriteTimeout: cfg.HTTPServer.WriteTimeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		level.Info(logger).Log("msg", "starting HTTP server", "port", cfg.HTTPServer.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			level.Error(logger).Log("msg", "HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	level.Info(logger).Log("msg", "shutting down HTTP server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		level.Error(logger).Log("msg", "failed to shutdown HTTP server gracefully", "error", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "HTTP server shutdown complete")
}

// setupLogger creates and configures a logger with default settings
func setupLogger() log.Logger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	return logger
}
