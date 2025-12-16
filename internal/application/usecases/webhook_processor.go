package usecases

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-kit/log"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/domain/repositories"
	"webhook-processor/internal/domain/services"
)

// WebhookProcessor handles webhook processing logic
type WebhookProcessor struct {
	webhookQueueRepo  repositories.WebhookQueueRepository
	webhookConfigRepo repositories.WebhookConfigRepository
	webhookService    services.WebhookService
	logger            log.Logger
}

// NewWebhookProcessor creates a new webhook processor
func NewWebhookProcessor(
	webhookQueueRepo repositories.WebhookQueueRepository,
	webhookConfigRepo repositories.WebhookConfigRepository,
	webhookService services.WebhookService,
	logger log.Logger,
) *WebhookProcessor {
	return &WebhookProcessor{
		webhookQueueRepo:  webhookQueueRepo,
		webhookConfigRepo: webhookConfigRepo,
		webhookService:    webhookService,
		logger:            logger,
	}
}

// CreateWebhookEntry creates a new webhook queue entry for processing
func (wp *WebhookProcessor) CreateWebhookEntry(ctx context.Context, eventType enums.EventType, eventID string, configID int64) error {
	// Get webhook config
	config, err := wp.webhookConfigRepo.GetByID(ctx, configID)
	if err != nil {
		return fmt.Errorf("failed to get webhook config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("webhook config not found: %d", configID)
	}

	if !config.IsActive {
		return fmt.Errorf("webhook config is not active: %d", configID)
	}

	// Create webhook queue entry
	webhook := &entities.WebhookQueue{
		EventType:   eventType,
		EventID:     eventID,
		ConfigID:    configID,
		WebhookURL:  config.WebhookURL,
		Status:      enums.WebhookStatusPending,
		RetryCount:  0,
		NextRetryAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := wp.webhookQueueRepo.Create(ctx, webhook); err != nil {
		return fmt.Errorf("failed to create webhook queue entry: %w", err)
	}

	wp.logger.Log("level", "info", "msg", "webhook entry created",
		"queue_id", webhook.QueueID, "event_type", eventType, "event_id", eventID)

	return nil
}

// ProcessWebhook processes a single webhook
func (wp *WebhookProcessor) ProcessWebhook(ctx context.Context, webhook *entities.WebhookQueue, workerID string) error {
	wp.logger.Log("level", "info", "msg", "processing webhook",
		"queue_id", webhook.QueueID, "worker_id", workerID, "retry_count", webhook.RetryCount)

	// Record attempt start
	attemptStartTime := time.Now().UTC()

	wp.logger.Log("level", "debug", "msg", "recording retry attempt",
		"queue_id", webhook.QueueID, "retry_level", webhook.RetryCount,
		"retry_count", webhook.RetryCount, "started_at", attemptStartTime)

	// Send webhook
	response, err := wp.webhookService.SendWebhook(ctx, webhook)
	attemptEndTime := time.Now().UTC()
	durationMs := attemptEndTime.Sub(attemptStartTime).Milliseconds()

	var httpStatus int
	var responseBody string
	if response != nil {
		httpStatus = response.StatusCode
		responseBody = response.Body
	}

	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	} else if response != nil && !wp.isSuccessfulResponse(response.StatusCode) {
		// HTTP request succeeded but got non-2xx status code - treat as error
		errorMsg = fmt.Sprintf("HTTP %d: %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	// Update retry attempt in database
	if updateErr := wp.webhookQueueRepo.UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, attemptStartTime, &attemptEndTime, durationMs, httpStatus, responseBody, errorMsg); updateErr != nil {
		wp.logger.Log("level", "error", "msg", "failed to update retry attempt",
			"queue_id", webhook.QueueID, "error", updateErr)
	}

	// Update webhook's last status for tracking
	webhook.LastHTTPStatus = httpStatus
	if errorMsg != "" {
		webhook.LastError = errorMsg
	}

	// Check if webhook was successful
	if err == nil && response != nil && wp.isSuccessfulResponse(response.StatusCode) {
		// Mark as completed with the start time of this successful attempt
		if err := wp.webhookQueueRepo.MarkCompleted(ctx, webhook.ID, attemptStartTime); err != nil {
			wp.logger.Log("level", "error", "msg", "failed to mark webhook as completed",
				"queue_id", webhook.QueueID, "error", err)
			return err
		}

		wp.logger.Log("level", "info", "msg", "webhook completed successfully",
			"queue_id", webhook.QueueID, "status_code", response.StatusCode, "retry_count", webhook.RetryCount)

		return nil
	}

	// Check if we should retry
	if webhook.CanRetry() {
		nextRetryAt := wp.calculateNextRetryTime(webhook.RetryCount)

		// Update webhook for next retry - preserve all existing fields
		webhook.RetryCount = webhook.RetryCount + 1
		webhook.NextRetryAt = nextRetryAt
		webhook.Status = enums.WebhookStatusPending
		webhook.UpdatedAt = time.Now().UTC()

		if err := wp.webhookQueueRepo.Update(ctx, webhook); err != nil {
			wp.logger.Log("level", "error", "msg", "failed to update webhook for retry",
				"queue_id", webhook.QueueID, "error", err)
			return err
		}

		wp.logger.Log("level", "info", "msg", "webhook scheduled for retry",
			"queue_id", webhook.QueueID, "retry_count", webhook.RetryCount, "next_retry_at", nextRetryAt)

		return nil
	}

	// Mark as permanently failed
	finalErrorMsg := "max retries exceeded"
	if err != nil {
		finalErrorMsg = fmt.Sprintf("max retries exceeded: %s", err.Error())
	} else if response != nil {
		finalErrorMsg = fmt.Sprintf("max retries exceeded: HTTP %d", response.StatusCode)
	}

	if err := wp.webhookQueueRepo.MarkFailed(ctx, webhook.ID, finalErrorMsg); err != nil {
		wp.logger.Log("level", "error", "msg", "failed to mark webhook as failed",
			"queue_id", webhook.QueueID, "error", err)
		return err
	}

	wp.logger.Log("level", "error", "msg", "webhook permanently failed",
		"queue_id", webhook.QueueID, "error", finalErrorMsg)

	return nil
}

// isSuccessfulResponse checks if the HTTP status code indicates success
func (wp *WebhookProcessor) isSuccessfulResponse(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// calculateNextRetryTime calculates the next retry time with simplified progression: 1min, 5min, 10min, 30min
func (wp *WebhookProcessor) calculateNextRetryTime(retryCount int) time.Time {
	var baseDelay time.Duration

	// Simplified retry progression aligned with worker polling intervals
	switch retryCount {
	case 0: // Next retry will be level 1
		baseDelay = 1 * time.Minute // 1 minute delay
	case 1: // Next retry will be level 2
		baseDelay = 5 * time.Minute // 5 minute delay
	case 2: // Next retry will be level 3
		baseDelay = 10 * time.Minute // 10 minute delay
	case 3: // Next retry will be level 4
		baseDelay = 30 * time.Minute // 30 minute delay
	case 4: // Next retry will be level 5
		baseDelay = 60 * time.Minute // 1 hour delay
	case 5: // Next retry will be level 6 (final)
		baseDelay = 120 * time.Minute // 2 hour delay
	default: // Fallback for any edge cases
		baseDelay = 4 * time.Hour
	}

	// Add random jitter (±25% of the base delay) to prevent thundering herd
	jitterRange := float64(baseDelay) * 0.25
	jitter := time.Duration(rand.Float64()*jitterRange*2 - jitterRange)

	finalDelay := baseDelay + jitter
	if finalDelay < time.Minute {
		finalDelay = time.Minute // Minimum 1 minute delay
	}

	return time.Now().UTC().Add(finalDelay)
}

// GetNextWebhookForProcessing atomically gets and locks ONE webhook for a specific retry level
func (wp *WebhookProcessor) GetNextWebhookForProcessing(ctx context.Context, workerID string, retryLevel int) (*entities.WebhookQueue, error) {
	return wp.webhookQueueRepo.GetNextWebhookForProcessing(ctx, workerID, retryLevel)
}

// ResetWebhookToPending resets a webhook back to pending status (for atomic processing)
func (wp *WebhookProcessor) ResetWebhookToPending(ctx context.Context, webhook *entities.WebhookQueue) error {
	// Update only the necessary fields while preserving all other data
	webhook.Status = enums.WebhookStatusPending
	webhook.UpdatedAt = time.Now().UTC()

	return wp.webhookQueueRepo.Update(ctx, webhook)
}
