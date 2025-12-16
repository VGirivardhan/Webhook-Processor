package repositories

import (
	"context"
	"time"

	"webhook-processor/internal/domain/entities"
)

// WebhookQueueRepository defines the interface for webhook queue operations
type WebhookQueueRepository interface {
	// Create creates a new webhook queue entry
	Create(ctx context.Context, webhook *entities.WebhookQueue) error

	// Update updates a webhook queue entry
	Update(ctx context.Context, webhook *entities.WebhookQueue) error

	// GetNextWebhookForProcessing atomically gets and locks ONE webhook for a specific retry level
	// Uses PostgreSQL's SELECT FOR UPDATE SKIP LOCKED for optimal concurrency
	GetNextWebhookForProcessing(ctx context.Context, workerID string, retryLevel int) (*entities.WebhookQueue, error)

	// UpdateRetryAttempt updates retry attempt information
	UpdateRetryAttempt(ctx context.Context, webhookID int64, retryLevel int, startedAt time.Time, completedAt *time.Time, durationMs int64, httpStatus int, responseBody, errorMsg string) error

	// MarkCompleted marks a webhook as completed
	MarkCompleted(ctx context.Context, webhookID int64, processingStartedAt time.Time) error

	// MarkFailed marks a webhook as failed
	MarkFailed(ctx context.Context, webhookID int64, errorMsg string) error
}
