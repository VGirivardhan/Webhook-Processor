package entities

import (
	"time"

	"webhook-processor/internal/domain/enums"

	"github.com/google/uuid"
)

// WebhookQueue represents a webhook processing queue entry
type WebhookQueue struct {
	ID      int64     `json:"id"`
	QueueID uuid.UUID `json:"queue_id"`

	// Event information
	EventType enums.EventType `json:"event_type"` // EventTypeCredit or EventTypeDebit
	EventID   string          `json:"event_id"`   // Original transaction/event ID

	// Webhook details
	ConfigID   int64  `json:"config_id"`
	WebhookURL string `json:"webhook_url"`

	// Processing status
	Status enums.WebhookStatus `json:"status"` // WebhookStatusPending, WebhookStatusProcessing, etc.

	// Retry tracking
	RetryCount  int       `json:"retry_count"`
	NextRetryAt time.Time `json:"next_retry_at"`

	// Individual retry attempt tracking (retry_0 through retry_6)
	Retry0StartedAt    *time.Time `json:"retry_0_started_at,omitempty"`
	Retry0CompletedAt  *time.Time `json:"retry_0_completed_at,omitempty"`
	Retry0DurationMs   *int64     `json:"retry_0_duration_ms,omitempty"`
	Retry0HTTPStatus   *int       `json:"retry_0_http_status,omitempty"`
	Retry0ResponseBody *string    `json:"retry_0_response_body,omitempty"`
	Retry0Error        *string    `json:"retry_0_error,omitempty"`

	Retry1StartedAt    *time.Time `json:"retry_1_started_at,omitempty"`
	Retry1CompletedAt  *time.Time `json:"retry_1_completed_at,omitempty"`
	Retry1DurationMs   *int64     `json:"retry_1_duration_ms,omitempty"`
	Retry1HTTPStatus   *int       `json:"retry_1_http_status,omitempty"`
	Retry1ResponseBody *string    `json:"retry_1_response_body,omitempty"`
	Retry1Error        *string    `json:"retry_1_error,omitempty"`

	Retry2StartedAt    *time.Time `json:"retry_2_started_at,omitempty"`
	Retry2CompletedAt  *time.Time `json:"retry_2_completed_at,omitempty"`
	Retry2DurationMs   *int64     `json:"retry_2_duration_ms,omitempty"`
	Retry2HTTPStatus   *int       `json:"retry_2_http_status,omitempty"`
	Retry2ResponseBody *string    `json:"retry_2_response_body,omitempty"`
	Retry2Error        *string    `json:"retry_2_error,omitempty"`

	Retry3StartedAt    *time.Time `json:"retry_3_started_at,omitempty"`
	Retry3CompletedAt  *time.Time `json:"retry_3_completed_at,omitempty"`
	Retry3DurationMs   *int64     `json:"retry_3_duration_ms,omitempty"`
	Retry3HTTPStatus   *int       `json:"retry_3_http_status,omitempty"`
	Retry3ResponseBody *string    `json:"retry_3_response_body,omitempty"`
	Retry3Error        *string    `json:"retry_3_error,omitempty"`

	Retry4StartedAt    *time.Time `json:"retry_4_started_at,omitempty"`
	Retry4CompletedAt  *time.Time `json:"retry_4_completed_at,omitempty"`
	Retry4DurationMs   *int64     `json:"retry_4_duration_ms,omitempty"`
	Retry4HTTPStatus   *int       `json:"retry_4_http_status,omitempty"`
	Retry4ResponseBody *string    `json:"retry_4_response_body,omitempty"`
	Retry4Error        *string    `json:"retry_4_error,omitempty"`

	Retry5StartedAt    *time.Time `json:"retry_5_started_at,omitempty"`
	Retry5CompletedAt  *time.Time `json:"retry_5_completed_at,omitempty"`
	Retry5DurationMs   *int64     `json:"retry_5_duration_ms,omitempty"`
	Retry5HTTPStatus   *int       `json:"retry_5_http_status,omitempty"`
	Retry5ResponseBody *string    `json:"retry_5_response_body,omitempty"`
	Retry5Error        *string    `json:"retry_5_error,omitempty"`

	Retry6StartedAt    *time.Time `json:"retry_6_started_at,omitempty"`
	Retry6CompletedAt  *time.Time `json:"retry_6_completed_at,omitempty"`
	Retry6DurationMs   *int64     `json:"retry_6_duration_ms,omitempty"`
	Retry6HTTPStatus   *int       `json:"retry_6_http_status,omitempty"`
	Retry6ResponseBody *string    `json:"retry_6_response_body,omitempty"`
	Retry6Error        *string    `json:"retry_6_error,omitempty"`

	// General tracking
	LastError      string `json:"last_error"`
	LastHTTPStatus int    `json:"last_http_status"`

	// Timestamps
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ProcessingStartedAt *time.Time `json:"processing_started_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	DeletedAt           *time.Time `json:"deleted_at"`
}

// CanRetry checks if the webhook can be retried
func (w *WebhookQueue) CanRetry() bool {
	return w.RetryCount < enums.MaxRetryAttempts && !w.Status.IsCompleted()
}
