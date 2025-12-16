package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"

	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/infrastructure/metrics"
)

// WebhookWorker represents a specialized webhook processing worker
type WebhookWorker struct {
	id           string
	retryLevel   int
	processor    *usecases.WebhookProcessor
	logger       log.Logger
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
	metrics      *metrics.WebhookMetrics
}

// NewWebhookWorker creates a new specialized webhook worker
func NewWebhookWorker(
	retryLevel int,
	processor *usecases.WebhookProcessor,
	logger log.Logger,
	pollInterval time.Duration,
	metrics *metrics.WebhookMetrics,
) *WebhookWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebhookWorker{
		id:           fmt.Sprintf("retry-%d-%s", retryLevel, uuid.New().String()[:8]),
		retryLevel:   retryLevel,
		processor:    processor,
		logger:       logger,
		pollInterval: pollInterval,
		ctx:          ctx,
		cancel:       cancel,
		metrics:      metrics,
	}
}

// Start starts the webhook worker
func (w *WebhookWorker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("worker %s is already running", w.id)
	}

	w.running = true

	w.logger.Log("level", "info", "msg", "starting worker",
		"worker_id", w.id, "retry_level", w.retryLevel, "poll_interval", w.pollInterval)

	w.wg.Add(1)
	go w.processLoop()

	return nil
}

// Stop stops the webhook worker
func (w *WebhookWorker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return fmt.Errorf("worker %s is not running", w.id)
	}

	w.logger.Log("level", "info", "msg", "stopping worker",
		"worker_id", w.id, "retry_level", w.retryLevel)

	w.cancel()
	w.wg.Wait()
	w.running = false

	w.logger.Log("level", "info", "msg", "worker stopped",
		"worker_id", w.id, "retry_level", w.retryLevel)

	return nil
}

// GetID returns the worker ID
func (w *WebhookWorker) GetID() string {
	return w.id
}

// GetRetryLevel returns the retry level this worker handles
func (w *WebhookWorker) GetRetryLevel() int {
	return w.retryLevel
}

// processLoop is the main processing loop - processes ONE webhook at a time
func (w *WebhookWorker) processLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Log("level", "info", "msg", "process loop stopped",
				"worker_id", w.id, "retry_level", w.retryLevel)
			return
		case <-ticker.C:
			w.processNextWebhook()
		}
	}
}

// processNextWebhook atomically gets and processes the next webhook for this worker's retry level
func (w *WebhookWorker) processNextWebhook() {
	// Start measuring complete worker busy time
	startTime := time.Now().UTC()
	var finalStatusCode int

	defer func() {
		// Only record metrics if we actually processed a webhook (finalStatusCode != 0)
		if finalStatusCode != 0 {
			workerDuration := time.Since(startTime)
			w.metrics.RecordWorkerProcessing(finalStatusCode, w.retryLevel, workerDuration)
		}
	}()

	// Get webhook specific to this retry level
	webhook, err := w.processor.GetNextWebhookForProcessing(w.ctx, w.id, w.retryLevel)
	if err != nil {
		w.logger.Log("level", "error", "msg", "failed to get next webhook",
			"worker_id", w.id, "retry_level", w.retryLevel, "error", err)
		return
	}

	if webhook == nil {
		// No work available for this retry level - this is normal
		return
	}

	// Process the webhook (already locked atomically by SELECT FOR UPDATE)
	if err := w.processor.ProcessWebhook(w.ctx, webhook, w.id); err != nil {
		w.logger.Log("level", "error", "msg", "failed to process webhook",
			"worker_id", w.id, "retry_level", w.retryLevel, "queue_id", webhook.QueueID, "error", err)

		// Reset to pending status on error
		if resetErr := w.processor.ResetWebhookToPending(w.ctx, webhook); resetErr != nil {
			w.logger.Log("level", "error", "msg", "failed to reset webhook to pending",
				"worker_id", w.id, "retry_level", w.retryLevel, "queue_id", webhook.QueueID, "error", resetErr)
		}

		// Use the last known status code from the webhook, or 500 for processing errors
		if webhook.LastHTTPStatus != 0 {
			finalStatusCode = webhook.LastHTTPStatus
		} else {
			finalStatusCode = 500 // Processing error
		}
	} else {
		// Success - use the final status code from the webhook
		finalStatusCode = webhook.LastHTTPStatus
	}
}
