package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/repositories"
	"webhook-processor/internal/infrastructure/models"
)

// webhookConfigRepositoryImpl implements the WebhookConfigRepository interface
type webhookConfigRepositoryImpl struct {
	db *gorm.DB
}

// NewWebhookConfigRepository creates a new webhook config repository
func NewWebhookConfigRepository(db *gorm.DB) (repositories.WebhookConfigRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	return &webhookConfigRepositoryImpl{db: db}, nil
}

// GetByID retrieves a webhook config by ID
func (r *webhookConfigRepositoryImpl) GetByID(ctx context.Context, id int64) (*entities.WebhookConfig, error) {
	var model models.WebhookConfigModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get webhook config by ID: %w", err)
	}
	return r.modelToEntity(&model), nil
}

// modelToEntity converts GORM model to domain entity
func (r *webhookConfigRepositoryImpl) modelToEntity(model *models.WebhookConfigModel) *entities.WebhookConfig {
	return &entities.WebhookConfig{
		ID:         model.ID,
		Name:       model.Name,
		EventType:  model.EventType,
		WebhookURL: model.WebhookURL,
		IsActive:   model.IsActive,
		TimeoutMs:  model.TimeoutMs,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}
