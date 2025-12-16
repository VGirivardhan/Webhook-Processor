package services

import (
	"context"
	"time"

	"webhook-processor/internal/domain/entities"
)

// WebhookService defines the interface for webhook processing operations
type WebhookService interface {
	// SendWebhook sends a webhook request and returns the response
	SendWebhook(ctx context.Context, webhook *entities.WebhookQueue) (*WebhookResponse, error)
}

// WebhookResponse represents the response from a webhook call
type WebhookResponse struct {
	StatusCode int           `json:"status_code"`
	Body       string        `json:"body"`
	Duration   time.Duration `json:"duration"`
	Error      error         `json:"error"`
}
