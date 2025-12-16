package workers

import (
	"fmt"
	"sync"

	"github.com/go-kit/log"

	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/config"
	"webhook-processor/internal/infrastructure/metrics"
)

// WorkerPool manages a pool of specialized webhook workers
type WorkerPool struct {
	workers   []*WebhookWorker
	processor *usecases.WebhookProcessor
	logger    log.Logger
	config    config.WorkerPoolConfig
	running   bool
	mu        sync.RWMutex
	metrics   *metrics.WebhookMetrics
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	processor *usecases.WebhookProcessor,
	logger log.Logger,
	config config.WorkerPoolConfig,
	metrics *metrics.WebhookMetrics,
) *WorkerPool {
	return &WorkerPool{
		processor: processor,
		logger:    logger,
		config:    config,
		workers:   make([]*WebhookWorker, 0, len(config.Workers)),
		metrics:   metrics,
	}
}

// Start starts all workers in the pool
func (wp *WorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return fmt.Errorf("worker pool is already running")
	}

	wp.logger.Log("level", "info", "msg", "starting worker pool",
		"worker_count", len(wp.config.Workers))

	// Create and start workers for each retry level
	for _, workerConfig := range wp.config.Workers {
		worker := NewWebhookWorker(
			workerConfig.RetryLevel,
			wp.processor,
			wp.logger,
			workerConfig.PollInterval,
			wp.metrics,
		)

		if err := worker.Start(); err != nil {
			// Stop any workers that were already started
			wp.stopWorkers()
			return fmt.Errorf("failed to start worker for level %d: %w",
				workerConfig.RetryLevel, err)
		}

		wp.workers = append(wp.workers, worker)

		wp.logger.Log("level", "info", "msg", "worker started",
			"retry_level", workerConfig.RetryLevel,
			"poll_interval", workerConfig.PollInterval,
			"description", workerConfig.Description)
	}

	wp.running = true
	wp.logger.Log("level", "info", "msg", "worker pool started successfully",
		"total_workers", len(wp.workers))

	return nil
}

// Stop stops all workers in the pool
func (wp *WorkerPool) Stop() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return fmt.Errorf("worker pool is not running")
	}

	wp.logger.Log("level", "info", "msg", "stopping worker pool")

	wp.stopWorkers()
	wp.running = false

	wp.logger.Log("level", "info", "msg", "worker pool stopped")

	return nil
}

// stopWorkers stops all workers
func (wp *WorkerPool) stopWorkers() {
	var wg sync.WaitGroup

	for _, worker := range wp.workers {
		wg.Add(1)
		go func(w *WebhookWorker) {
			defer wg.Done()
			if err := w.Stop(); err != nil {
				wp.logger.Log("level", "error", "msg", "failed to stop worker",
					"worker_id", w.GetID(), "retry_level", w.GetRetryLevel(), "error", err)
			}
		}(worker)
	}

	wg.Wait()
	wp.workers = wp.workers[:0] // Clear the slice
}
