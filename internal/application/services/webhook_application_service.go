package services

import (
	"context"
	"time"

	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/domain/enums"
)

// WebhookApplicationService defines the application service interface for webhook operations
// This layer orchestrates business logic and coordinates between transport and domain layers
type WebhookApplicationService interface {
	// CreateWebhook creates a new webhook entry
	CreateWebhook(ctx context.Context, req CreateWebhookCommand) (*CreateWebhookResult, error)

	// GetHealth returns service health status
	GetHealth(ctx context.Context) (*HealthResult, error)
}

// Commands (Input DTOs)

// CreateWebhookCommand represents a command to create a webhook
type CreateWebhookCommand struct {
	EventType enums.EventType `json:"event_type" validate:"required"`
	EventID   string          `json:"event_id"`
	ConfigID  int64           `json:"config_id" validate:"required,min=1"`
}

// Results (Output DTOs)

// CreateWebhookResult represents the result of creating a webhook
type CreateWebhookResult struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	QueueID   string    `json:"queue_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// HealthResult represents service health status
type HealthResult struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Timestamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
	Uptime       time.Duration     `json:"uptime"`
}

// webhookApplicationServiceImpl implements WebhookApplicationService
type webhookApplicationServiceImpl struct {
	webhookProcessor *usecases.WebhookProcessor
	startTime        time.Time
}

// NewWebhookApplicationService creates a new webhook application service
func NewWebhookApplicationService(webhookProcessor *usecases.WebhookProcessor) WebhookApplicationService {
	return &webhookApplicationServiceImpl{
		webhookProcessor: webhookProcessor,
		startTime:        time.Now().UTC(),
	}
}

// CreateWebhook creates a new webhook entry
func (s *webhookApplicationServiceImpl) CreateWebhook(ctx context.Context, cmd CreateWebhookCommand) (*CreateWebhookResult, error) {
	// Validate command
	if err := cmd.EventType.Validate(); err != nil {
		return &CreateWebhookResult{
			Success: false,
			Message: "Invalid event type: " + err.Error(),
		}, err
	}

	// Call use case
	err := s.webhookProcessor.CreateWebhookEntry(ctx, cmd.EventType, cmd.EventID, cmd.ConfigID)
	if err != nil {
		return &CreateWebhookResult{
			Success: false,
			Message: "Failed to create webhook: " + err.Error(),
		}, err
	}

	return &CreateWebhookResult{
		Success:   true,
		Message:   "Webhook created successfully",
		CreatedAt: time.Now().UTC(),
	}, nil
}

// GetHealth returns service health status
func (s *webhookApplicationServiceImpl) GetHealth(ctx context.Context) (*HealthResult, error) {
	return &HealthResult{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]string{
			"database": "connected",
			"workers":  "running",
		},
		Uptime: time.Since(s.startTime),
	}, nil
}
