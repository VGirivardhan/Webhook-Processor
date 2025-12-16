package repositories

import (
	"context"

	"webhook-processor/internal/domain/entities"
)

// WebhookConfigRepository defines the interface for webhook config operations
type WebhookConfigRepository interface {
	// GetByID retrieves a webhook config by ID (ONLY method actually used)
	GetByID(ctx context.Context, id int64) (*entities.WebhookConfig, error)
}
