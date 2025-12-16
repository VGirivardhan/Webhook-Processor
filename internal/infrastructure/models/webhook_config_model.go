package models

import (
	"time"

	"webhook-processor/internal/domain/enums"

	"gorm.io/gorm"
)

// WebhookConfigModel represents the GORM model for webhook_configs table
type WebhookConfigModel struct {
	ID         int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string          `gorm:"type:varchar(255);not null" json:"name"`
	EventType  enums.EventType `gorm:"type:event_type;not null" json:"event_type"`
	WebhookURL string          `gorm:"type:text;not null" json:"webhook_url"`
	IsActive   bool            `gorm:"default:true" json:"is_active"`
	TimeoutMs  int             `gorm:"default:30000" json:"timeout_ms"`
	CreatedAt  time.Time       `gorm:"default:NOW()" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"default:NOW()" json:"updated_at"`
	DeletedAt  *time.Time      `gorm:"index" json:"deleted_at"`
}

// TableName returns the table name for GORM
func (WebhookConfigModel) TableName() string {
	return "webhook_configs"
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (w *WebhookConfigModel) BeforeUpdate(tx *gorm.DB) error {
	w.UpdatedAt = time.Now().UTC()
	return nil
}
