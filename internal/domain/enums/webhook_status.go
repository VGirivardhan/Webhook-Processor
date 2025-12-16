package enums

// WebhookStatus represents the processing status of a webhook
type WebhookStatus string

const (
	// WebhookStatusPending indicates the webhook is waiting to be processed
	WebhookStatusPending WebhookStatus = "PENDING"

	// WebhookStatusProcessing indicates the webhook is currently being processed
	WebhookStatusProcessing WebhookStatus = "PROCESSING"

	// WebhookStatusCompleted indicates the webhook was successfully processed
	WebhookStatusCompleted WebhookStatus = "COMPLETED"

	// WebhookStatusFailed indicates the webhook failed after all retry attempts
	WebhookStatusFailed WebhookStatus = "FAILED"
)

// MaxRetryAttempts defines the maximum number of retry attempts
// This is fixed by the database schema (retry_0 through retry_6 = 7 total attempts)
const MaxRetryAttempts = 6

// IsCompleted checks if the status is completed
func (s WebhookStatus) IsCompleted() bool {
	return s == WebhookStatusCompleted
}
