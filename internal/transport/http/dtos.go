package http

import (
	"time"

	"webhook-processor/internal/application/services"
	"webhook-processor/internal/domain/enums"
)

// HTTP Transport DTOs - These are specific to the HTTP transport layer
// They handle JSON marshaling/unmarshaling and HTTP-specific concerns

// CreateWebhookRequest represents an HTTP request to create a webhook
type CreateWebhookRequest struct {
	EventType enums.EventType `json:"event_type" validate:"required"`
	EventID   string          `json:"event_id"`
	ConfigID  int64           `json:"config_id" validate:"required,min=1"`
}

// CreateWebhookResponse represents an HTTP response after creating a webhook
type CreateWebhookResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	QueueID   string `json:"queue_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"` // ISO 8601 string for HTTP
}

// HealthResponse represents HTTP response for service health status
type HealthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Timestamp    string            `json:"timestamp"` // ISO 8601 string for HTTP
	Dependencies map[string]string `json:"dependencies"`
	Uptime       string            `json:"uptime"` // Duration string for HTTP
}

// Conversion functions between HTTP DTOs and Application DTOs

// ToApplicationCommand converts HTTP request to application command
func (r CreateWebhookRequest) ToApplicationCommand() services.CreateWebhookCommand {
	return services.CreateWebhookCommand{
		EventType: r.EventType,
		EventID:   r.EventID,
		ConfigID:  r.ConfigID,
	}
}

// FromApplicationResult converts application result to HTTP response
func (r *CreateWebhookResponse) FromApplicationResult(result *services.CreateWebhookResult) {
	r.Success = result.Success
	r.Message = result.Message
	r.QueueID = result.QueueID
	if !result.CreatedAt.IsZero() {
		r.CreatedAt = result.CreatedAt.Format(time.RFC3339)
	}
}

// FromApplicationResult converts application health result to HTTP response
func (r *HealthResponse) FromApplicationResult(result *services.HealthResult) {
	r.Status = result.Status
	r.Version = result.Version
	r.Timestamp = result.Timestamp.Format(time.RFC3339)
	r.Dependencies = result.Dependencies
	r.Uptime = result.Uptime.String()
}
