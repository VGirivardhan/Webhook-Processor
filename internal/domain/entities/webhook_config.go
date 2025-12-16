package entities

import (
	"time"

	"webhook-processor/internal/domain/enums"
)

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	EventType  enums.EventType `json:"event_type"` // EventTypeCredit or EventTypeDebit
	WebhookURL string          `json:"webhook_url"`
	IsActive   bool            `json:"is_active"`
	TimeoutMs  int             `json:"timeout_ms"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
