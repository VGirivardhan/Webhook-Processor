package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/domain/repositories"
	"webhook-processor/internal/infrastructure/models"
)

// webhookQueueRepositoryImpl implements the WebhookQueueRepository interface
type webhookQueueRepositoryImpl struct {
	db *gorm.DB
}

// NewWebhookQueueRepository creates a new webhook queue repository
func NewWebhookQueueRepository(db *gorm.DB) (repositories.WebhookQueueRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	return &webhookQueueRepositoryImpl{db: db}, nil
}

// Create creates a new webhook queue entry
func (r *webhookQueueRepositoryImpl) Create(ctx context.Context, webhook *entities.WebhookQueue) error {
	model := r.entityToModel(webhook)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create webhook queue entry: %w", err)
	}
	webhook.ID = model.ID
	webhook.QueueID = model.QueueID
	return nil
}

// Update updates a webhook queue entry with intelligent field merging
func (r *webhookQueueRepositoryImpl) Update(ctx context.Context, webhook *entities.WebhookQueue) error {
	var currentModel models.WebhookQueueModel
	if err := r.db.WithContext(ctx).Where("id = ?", webhook.ID).First(&currentModel).Error; err != nil {
		return fmt.Errorf("failed to get current webhook state: %w", err)
	}

	r.mergeWebhookIntoModel(&currentModel, webhook)

	if err := r.db.WithContext(ctx).Save(&currentModel).Error; err != nil {
		return fmt.Errorf("failed to update webhook queue entry: %w", err)
	}
	return nil
}

// GetNextWebhookForProcessing atomically gets and locks ONE webhook for a specific retry level
// Uses PostgreSQL's SELECT FOR UPDATE SKIP LOCKED for optimal concurrency
func (r *webhookQueueRepositoryImpl) GetNextWebhookForProcessing(ctx context.Context, workerID string, retryLevel int) (*entities.WebhookQueue, error) {
	var model models.WebhookQueueModel

	// Start transaction for atomic operation
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer tx.Rollback()

	// Atomically select and lock ONE webhook for the specific retry level using GORM's clause.Locking
	now := time.Now().UTC()

	err := tx.
		Where("status = ? AND retry_count = ? AND next_retry_at <= ?",
			enums.WebhookStatusPending, retryLevel, now).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Order("next_retry_at ASC").
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Commit() // No work available for this retry level
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get next webhook for retry level %d: %w", retryLevel, err)
	}

	// Update the selected webhook to PROCESSING status atomically
	if err := tx.Model(&model).
		Updates(map[string]interface{}{
			"status":     enums.WebhookStatusProcessing,
			"updated_at": now,
		}).Error; err != nil {
		return nil, fmt.Errorf("failed to update webhook status for retry level %d: %w", retryLevel, err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction for retry level %d: %w", retryLevel, err)
	}

	// Update model in memory and convert to entity
	model.Status = enums.WebhookStatusProcessing
	model.UpdatedAt = now

	return r.modelToEntity(&model), nil
}

// UpdateRetryAttempt updates retry attempt information
func (r *webhookQueueRepositoryImpl) UpdateRetryAttempt(ctx context.Context, webhookID int64, retryLevel int, startedAt time.Time, completedAt *time.Time, durationMs int64, httpStatus int, responseBody, errorMsg string) error {
	updates := map[string]interface{}{
		"updated_at":       time.Now().UTC(),
		"last_http_status": httpStatus,
	}

	if errorMsg != "" {
		updates["last_error"] = errorMsg
	}

	// Update specific retry attempt columns based on retry level
	switch retryLevel {
	case 0:
		updates["retry_0_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_0_completed_at"] = *completedAt
		}
		updates["retry_0_duration_ms"] = durationMs
		updates["retry_0_http_status"] = httpStatus
		updates["retry_0_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_0_error"] = errorMsg
		}
	case 1:
		updates["retry_1_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_1_completed_at"] = *completedAt
		}
		updates["retry_1_duration_ms"] = durationMs
		updates["retry_1_http_status"] = httpStatus
		updates["retry_1_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_1_error"] = errorMsg
		}
	case 2:
		updates["retry_2_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_2_completed_at"] = *completedAt
		}
		updates["retry_2_duration_ms"] = durationMs
		updates["retry_2_http_status"] = httpStatus
		updates["retry_2_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_2_error"] = errorMsg
		}
	case 3:
		updates["retry_3_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_3_completed_at"] = *completedAt
		}
		updates["retry_3_duration_ms"] = durationMs
		updates["retry_3_http_status"] = httpStatus
		updates["retry_3_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_3_error"] = errorMsg
		}
	case 4:
		updates["retry_4_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_4_completed_at"] = *completedAt
		}
		updates["retry_4_duration_ms"] = durationMs
		updates["retry_4_http_status"] = httpStatus
		updates["retry_4_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_4_error"] = errorMsg
		}
	case 5:
		updates["retry_5_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_5_completed_at"] = *completedAt
		}
		updates["retry_5_duration_ms"] = durationMs
		updates["retry_5_http_status"] = httpStatus
		updates["retry_5_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_5_error"] = errorMsg
		}
	case 6:
		updates["retry_6_started_at"] = startedAt
		if completedAt != nil {
			updates["retry_6_completed_at"] = *completedAt
		}
		updates["retry_6_duration_ms"] = durationMs
		updates["retry_6_http_status"] = httpStatus
		updates["retry_6_response_body"] = responseBody
		if errorMsg != "" {
			updates["retry_6_error"] = errorMsg
		}
	}

	if err := r.db.WithContext(ctx).
		Model(&models.WebhookQueueModel{}).
		Where("id = ?", webhookID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update retry attempt: %w", err)
	}

	return nil
}

// MarkCompleted marks a webhook as completed
func (r *webhookQueueRepositoryImpl) MarkCompleted(ctx context.Context, webhookID int64, processingStartedAt time.Time) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&models.WebhookQueueModel{}).
		Where("id = ?", webhookID).
		Updates(map[string]interface{}{
			"status":                enums.WebhookStatusCompleted,
			"processing_started_at": processingStartedAt,
			"completed_at":          now,
			"updated_at":            now,
		}).Error; err != nil {
		return fmt.Errorf("failed to mark webhook as completed: %w", err)
	}
	return nil
}

// MarkFailed marks a webhook as failed
func (r *webhookQueueRepositoryImpl) MarkFailed(ctx context.Context, webhookID int64, errorMsg string) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&models.WebhookQueueModel{}).
		Where("id = ?", webhookID).
		Updates(map[string]interface{}{
			"status":     enums.WebhookStatusFailed,
			"last_error": errorMsg,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to mark webhook as failed: %w", err)
	}
	return nil
}

func (r *webhookQueueRepositoryImpl) mergeWebhookIntoModel(model *models.WebhookQueueModel, update *entities.WebhookQueue) {
	// Core fields - update if non-zero/non-empty in update entity
	if update.QueueID != uuid.Nil {
		model.QueueID = update.QueueID
	}

	if update.EventType != "" {
		model.EventType = update.EventType
	}

	if update.EventID != "" {
		model.EventID = update.EventID
	}

	if update.ConfigID != 0 {
		model.ConfigID = update.ConfigID
	}

	if update.WebhookURL != "" {
		model.WebhookURL = update.WebhookURL
	}

	if update.Status != "" {
		model.Status = update.Status
	}

	if update.RetryCount != 0 || !update.NextRetryAt.IsZero() {
		model.RetryCount = update.RetryCount
		model.NextRetryAt = update.NextRetryAt
	}

	if update.LastError != "" {
		model.LastError = update.LastError
	}

	if update.LastHTTPStatus != 0 {
		model.LastHTTPStatus = update.LastHTTPStatus
	}

	// Timestamp fields - update if non-zero in update entity
	if !update.UpdatedAt.IsZero() {
		model.UpdatedAt = update.UpdatedAt
	}

	if update.ProcessingStartedAt != nil {
		model.ProcessingStartedAt = update.ProcessingStartedAt
	}

	if update.CompletedAt != nil {
		model.CompletedAt = update.CompletedAt
	}

	if update.DeletedAt != nil {
		model.DeletedAt = update.DeletedAt
	}

}

// entityToModel converts domain entity to GORM model
// Now simple since structures match!
func (r *webhookQueueRepositoryImpl) entityToModel(webhook *entities.WebhookQueue) *models.WebhookQueueModel {
	return &models.WebhookQueueModel{
		ID:                  webhook.ID,
		QueueID:             webhook.QueueID,
		EventType:           webhook.EventType,
		EventID:             webhook.EventID,
		ConfigID:            webhook.ConfigID,
		WebhookURL:          webhook.WebhookURL,
		Status:              webhook.Status,
		RetryCount:          webhook.RetryCount,
		NextRetryAt:         webhook.NextRetryAt,
		LastError:           webhook.LastError,
		LastHTTPStatus:      webhook.LastHTTPStatus,
		CreatedAt:           webhook.CreatedAt,
		UpdatedAt:           webhook.UpdatedAt,
		ProcessingStartedAt: webhook.ProcessingStartedAt,
		CompletedAt:         webhook.CompletedAt,
		DeletedAt:           webhook.DeletedAt,

		// Direct mapping of retry attempt fields
		Retry0StartedAt:    webhook.Retry0StartedAt,
		Retry0CompletedAt:  webhook.Retry0CompletedAt,
		Retry0DurationMs:   webhook.Retry0DurationMs,
		Retry0HTTPStatus:   webhook.Retry0HTTPStatus,
		Retry0ResponseBody: webhook.Retry0ResponseBody,
		Retry0Error:        webhook.Retry0Error,

		Retry1StartedAt:    webhook.Retry1StartedAt,
		Retry1CompletedAt:  webhook.Retry1CompletedAt,
		Retry1DurationMs:   webhook.Retry1DurationMs,
		Retry1HTTPStatus:   webhook.Retry1HTTPStatus,
		Retry1ResponseBody: webhook.Retry1ResponseBody,
		Retry1Error:        webhook.Retry1Error,

		Retry2StartedAt:    webhook.Retry2StartedAt,
		Retry2CompletedAt:  webhook.Retry2CompletedAt,
		Retry2DurationMs:   webhook.Retry2DurationMs,
		Retry2HTTPStatus:   webhook.Retry2HTTPStatus,
		Retry2ResponseBody: webhook.Retry2ResponseBody,
		Retry2Error:        webhook.Retry2Error,

		Retry3StartedAt:    webhook.Retry3StartedAt,
		Retry3CompletedAt:  webhook.Retry3CompletedAt,
		Retry3DurationMs:   webhook.Retry3DurationMs,
		Retry3HTTPStatus:   webhook.Retry3HTTPStatus,
		Retry3ResponseBody: webhook.Retry3ResponseBody,
		Retry3Error:        webhook.Retry3Error,

		Retry4StartedAt:    webhook.Retry4StartedAt,
		Retry4CompletedAt:  webhook.Retry4CompletedAt,
		Retry4DurationMs:   webhook.Retry4DurationMs,
		Retry4HTTPStatus:   webhook.Retry4HTTPStatus,
		Retry4ResponseBody: webhook.Retry4ResponseBody,
		Retry4Error:        webhook.Retry4Error,

		Retry5StartedAt:    webhook.Retry5StartedAt,
		Retry5CompletedAt:  webhook.Retry5CompletedAt,
		Retry5DurationMs:   webhook.Retry5DurationMs,
		Retry5HTTPStatus:   webhook.Retry5HTTPStatus,
		Retry5ResponseBody: webhook.Retry5ResponseBody,
		Retry5Error:        webhook.Retry5Error,

		Retry6StartedAt:    webhook.Retry6StartedAt,
		Retry6CompletedAt:  webhook.Retry6CompletedAt,
		Retry6DurationMs:   webhook.Retry6DurationMs,
		Retry6HTTPStatus:   webhook.Retry6HTTPStatus,
		Retry6ResponseBody: webhook.Retry6ResponseBody,
		Retry6Error:        webhook.Retry6Error,
	}
}

// modelToEntity converts GORM model to domain entity
// Now simple since structures match!
func (r *webhookQueueRepositoryImpl) modelToEntity(model *models.WebhookQueueModel) *entities.WebhookQueue {
	return &entities.WebhookQueue{
		ID:                  model.ID,
		QueueID:             model.QueueID,
		EventType:           model.EventType,
		EventID:             model.EventID,
		ConfigID:            model.ConfigID,
		WebhookURL:          model.WebhookURL,
		Status:              model.Status,
		RetryCount:          model.RetryCount,
		NextRetryAt:         model.NextRetryAt,
		LastError:           model.LastError,
		LastHTTPStatus:      model.LastHTTPStatus,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		ProcessingStartedAt: model.ProcessingStartedAt,
		CompletedAt:         model.CompletedAt,
		DeletedAt:           model.DeletedAt,

		// Direct mapping of retry attempt fields
		Retry0StartedAt:    model.Retry0StartedAt,
		Retry0CompletedAt:  model.Retry0CompletedAt,
		Retry0DurationMs:   model.Retry0DurationMs,
		Retry0HTTPStatus:   model.Retry0HTTPStatus,
		Retry0ResponseBody: model.Retry0ResponseBody,
		Retry0Error:        model.Retry0Error,

		Retry1StartedAt:    model.Retry1StartedAt,
		Retry1CompletedAt:  model.Retry1CompletedAt,
		Retry1DurationMs:   model.Retry1DurationMs,
		Retry1HTTPStatus:   model.Retry1HTTPStatus,
		Retry1ResponseBody: model.Retry1ResponseBody,
		Retry1Error:        model.Retry1Error,

		Retry2StartedAt:    model.Retry2StartedAt,
		Retry2CompletedAt:  model.Retry2CompletedAt,
		Retry2DurationMs:   model.Retry2DurationMs,
		Retry2HTTPStatus:   model.Retry2HTTPStatus,
		Retry2ResponseBody: model.Retry2ResponseBody,
		Retry2Error:        model.Retry2Error,

		Retry3StartedAt:    model.Retry3StartedAt,
		Retry3CompletedAt:  model.Retry3CompletedAt,
		Retry3DurationMs:   model.Retry3DurationMs,
		Retry3HTTPStatus:   model.Retry3HTTPStatus,
		Retry3ResponseBody: model.Retry3ResponseBody,
		Retry3Error:        model.Retry3Error,

		Retry4StartedAt:    model.Retry4StartedAt,
		Retry4CompletedAt:  model.Retry4CompletedAt,
		Retry4DurationMs:   model.Retry4DurationMs,
		Retry4HTTPStatus:   model.Retry4HTTPStatus,
		Retry4ResponseBody: model.Retry4ResponseBody,
		Retry4Error:        model.Retry4Error,

		Retry5StartedAt:    model.Retry5StartedAt,
		Retry5CompletedAt:  model.Retry5CompletedAt,
		Retry5DurationMs:   model.Retry5DurationMs,
		Retry5HTTPStatus:   model.Retry5HTTPStatus,
		Retry5ResponseBody: model.Retry5ResponseBody,
		Retry5Error:        model.Retry5Error,

		Retry6StartedAt:    model.Retry6StartedAt,
		Retry6CompletedAt:  model.Retry6CompletedAt,
		Retry6DurationMs:   model.Retry6DurationMs,
		Retry6HTTPStatus:   model.Retry6HTTPStatus,
		Retry6ResponseBody: model.Retry6ResponseBody,
		Retry6Error:        model.Retry6Error,
	}
}
