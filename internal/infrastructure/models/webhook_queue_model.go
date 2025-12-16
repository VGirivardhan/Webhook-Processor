package models

import (
	"time"

	"webhook-processor/internal/domain/enums"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookQueueModel represents the GORM model for webhook_queue table
type WebhookQueueModel struct {
	ID      int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	QueueID uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();uniqueIndex" json:"queue_id"`

	// Event information
	EventType enums.EventType `gorm:"type:event_type;not null" json:"event_type"`
	EventID   string          `gorm:"type:varchar(255);not null" json:"event_id"`

	// Webhook details
	ConfigID   int64  `gorm:"not null" json:"config_id"`
	WebhookURL string `gorm:"type:text;not null" json:"webhook_url"`

	// Processing status
	Status enums.WebhookStatus `gorm:"type:webhook_status;not null;default:'PENDING'" json:"status"`

	// Retry tracking
	RetryCount  int       `gorm:"not null;default:0" json:"retry_count"`
	NextRetryAt time.Time `gorm:"not null;default:NOW()" json:"next_retry_at"`

	// Individual retry attempt columns
	Retry0StartedAt    *time.Time `gorm:"column:retry_0_started_at" json:"retry_0_started_at"`
	Retry0CompletedAt  *time.Time `gorm:"column:retry_0_completed_at" json:"retry_0_completed_at"`
	Retry0DurationMs   *int64     `gorm:"column:retry_0_duration_ms" json:"retry_0_duration_ms"`
	Retry0HTTPStatus   *int       `gorm:"column:retry_0_http_status" json:"retry_0_http_status"`
	Retry0ResponseBody *string    `gorm:"column:retry_0_response_body;type:text" json:"retry_0_response_body"`
	Retry0Error        *string    `gorm:"column:retry_0_error;type:text" json:"retry_0_error"`

	Retry1StartedAt    *time.Time `gorm:"column:retry_1_started_at" json:"retry_1_started_at"`
	Retry1CompletedAt  *time.Time `gorm:"column:retry_1_completed_at" json:"retry_1_completed_at"`
	Retry1DurationMs   *int64     `gorm:"column:retry_1_duration_ms" json:"retry_1_duration_ms"`
	Retry1HTTPStatus   *int       `gorm:"column:retry_1_http_status" json:"retry_1_http_status"`
	Retry1ResponseBody *string    `gorm:"column:retry_1_response_body;type:text" json:"retry_1_response_body"`
	Retry1Error        *string    `gorm:"column:retry_1_error;type:text" json:"retry_1_error"`

	Retry2StartedAt    *time.Time `gorm:"column:retry_2_started_at" json:"retry_2_started_at"`
	Retry2CompletedAt  *time.Time `gorm:"column:retry_2_completed_at" json:"retry_2_completed_at"`
	Retry2DurationMs   *int64     `gorm:"column:retry_2_duration_ms" json:"retry_2_duration_ms"`
	Retry2HTTPStatus   *int       `gorm:"column:retry_2_http_status" json:"retry_2_http_status"`
	Retry2ResponseBody *string    `gorm:"column:retry_2_response_body;type:text" json:"retry_2_response_body"`
	Retry2Error        *string    `gorm:"column:retry_2_error;type:text" json:"retry_2_error"`

	Retry3StartedAt    *time.Time `gorm:"column:retry_3_started_at" json:"retry_3_started_at"`
	Retry3CompletedAt  *time.Time `gorm:"column:retry_3_completed_at" json:"retry_3_completed_at"`
	Retry3DurationMs   *int64     `gorm:"column:retry_3_duration_ms" json:"retry_3_duration_ms"`
	Retry3HTTPStatus   *int       `gorm:"column:retry_3_http_status" json:"retry_3_http_status"`
	Retry3ResponseBody *string    `gorm:"column:retry_3_response_body;type:text" json:"retry_3_response_body"`
	Retry3Error        *string    `gorm:"column:retry_3_error;type:text" json:"retry_3_error"`

	Retry4StartedAt    *time.Time `gorm:"column:retry_4_started_at" json:"retry_4_started_at"`
	Retry4CompletedAt  *time.Time `gorm:"column:retry_4_completed_at" json:"retry_4_completed_at"`
	Retry4DurationMs   *int64     `gorm:"column:retry_4_duration_ms" json:"retry_4_duration_ms"`
	Retry4HTTPStatus   *int       `gorm:"column:retry_4_http_status" json:"retry_4_http_status"`
	Retry4ResponseBody *string    `gorm:"column:retry_4_response_body;type:text" json:"retry_4_response_body"`
	Retry4Error        *string    `gorm:"column:retry_4_error;type:text" json:"retry_4_error"`

	Retry5StartedAt    *time.Time `gorm:"column:retry_5_started_at" json:"retry_5_started_at"`
	Retry5CompletedAt  *time.Time `gorm:"column:retry_5_completed_at" json:"retry_5_completed_at"`
	Retry5DurationMs   *int64     `gorm:"column:retry_5_duration_ms" json:"retry_5_duration_ms"`
	Retry5HTTPStatus   *int       `gorm:"column:retry_5_http_status" json:"retry_5_http_status"`
	Retry5ResponseBody *string    `gorm:"column:retry_5_response_body;type:text" json:"retry_5_response_body"`
	Retry5Error        *string    `gorm:"column:retry_5_error;type:text" json:"retry_5_error"`

	Retry6StartedAt    *time.Time `gorm:"column:retry_6_started_at" json:"retry_6_started_at"`
	Retry6CompletedAt  *time.Time `gorm:"column:retry_6_completed_at" json:"retry_6_completed_at"`
	Retry6DurationMs   *int64     `gorm:"column:retry_6_duration_ms" json:"retry_6_duration_ms"`
	Retry6HTTPStatus   *int       `gorm:"column:retry_6_http_status" json:"retry_6_http_status"`
	Retry6ResponseBody *string    `gorm:"column:retry_6_response_body;type:text" json:"retry_6_response_body"`
	Retry6Error        *string    `gorm:"column:retry_6_error;type:text" json:"retry_6_error"`

	// General tracking
	LastError      string `gorm:"type:text" json:"last_error"`
	LastHTTPStatus int    `json:"last_http_status"`

	// Timestamps
	CreatedAt           time.Time  `gorm:"default:NOW()" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"default:NOW()" json:"updated_at"`
	ProcessingStartedAt *time.Time `json:"processing_started_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	DeletedAt           *time.Time `gorm:"index" json:"deleted_at"`
}

// TableName returns the table name for GORM
func (WebhookQueueModel) TableName() string {
	return "webhook_queue"
}

// BeforeCreate is a GORM hook that runs before creating a record
func (w *WebhookQueueModel) BeforeCreate(tx *gorm.DB) error {
	if w.QueueID == uuid.Nil {
		w.QueueID = uuid.New()
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (w *WebhookQueueModel) BeforeUpdate(tx *gorm.DB) error {
	w.UpdatedAt = time.Now().UTC()
	return nil
}
