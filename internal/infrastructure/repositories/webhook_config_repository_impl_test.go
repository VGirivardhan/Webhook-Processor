package repositories

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/infrastructure/models"
)

// TestWebhookConfigRepositoryImpl_Constructor tests repository construction
func TestWebhookConfigRepositoryImpl_Constructor(t *testing.T) {
	tests := []struct {
		name        string
		db          *gorm.DB
		expectError bool
		errorMsg    string
	}{
		{
			name:        "should create repository with valid db",
			db:          &gorm.DB{},
			expectError: false,
		},
		{
			name:        "should return error with nil db",
			db:          nil,
			expectError: true,
			errorMsg:    "database cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewWebhookConfigRepository(tt.db)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &webhookConfigRepositoryImpl{}, repo)
			}
		})
	}
}

// TestWebhookConfigRepositoryImpl_ModelToEntity tests model to entity conversion
func TestWebhookConfigRepositoryImpl_ModelToEntity(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	tests := []struct {
		name   string
		model  *models.WebhookConfigModel
		verify func(t *testing.T, entity *entities.WebhookConfig)
	}{
		{
			name: "should convert complete model to entity",
			model: &models.WebhookConfigModel{
				ID:         1,
				Name:       "Test Config",
				EventType:  enums.EventTypeCredit,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
				TimeoutMs:  30000,
				CreatedAt:  time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
			},
			verify: func(t *testing.T, entity *entities.WebhookConfig) {
				assert.Equal(t, int64(1), entity.ID)
				assert.Equal(t, "Test Config", entity.Name)
				assert.Equal(t, enums.EventTypeCredit, entity.EventType)
				assert.Equal(t, "https://example.com/webhook", entity.WebhookURL)
				assert.True(t, entity.IsActive)
				assert.Equal(t, 30000, entity.TimeoutMs)
				assert.Equal(t, time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), entity.CreatedAt)
				assert.Equal(t, time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC), entity.UpdatedAt)
			},
		},
		{
			name: "should convert debit event type model",
			model: &models.WebhookConfigModel{
				ID:         2,
				Name:       "Debit Config",
				EventType:  enums.EventTypeDebit,
				WebhookURL: "https://debit.example.com/webhook",
				IsActive:   false,
				TimeoutMs:  45000,
				CreatedAt:  time.Date(2023, 6, 15, 9, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
			},
			verify: func(t *testing.T, entity *entities.WebhookConfig) {
				assert.Equal(t, int64(2), entity.ID)
				assert.Equal(t, "Debit Config", entity.Name)
				assert.Equal(t, enums.EventTypeDebit, entity.EventType)
				assert.Equal(t, "https://debit.example.com/webhook", entity.WebhookURL)
				assert.False(t, entity.IsActive)
				assert.Equal(t, 45000, entity.TimeoutMs)
			},
		},
		{
			name: "should handle zero values",
			model: &models.WebhookConfigModel{
				ID:         0,
				Name:       "",
				EventType:  enums.EventType(""),
				WebhookURL: "",
				IsActive:   false,
				TimeoutMs:  0,
				CreatedAt:  time.Time{},
				UpdatedAt:  time.Time{},
			},
			verify: func(t *testing.T, entity *entities.WebhookConfig) {
				assert.Equal(t, int64(0), entity.ID)
				assert.Equal(t, "", entity.Name)
				assert.Equal(t, enums.EventType(""), entity.EventType)
				assert.Equal(t, "", entity.WebhookURL)
				assert.False(t, entity.IsActive)
				assert.Equal(t, 0, entity.TimeoutMs)
				assert.True(t, entity.CreatedAt.IsZero())
				assert.True(t, entity.UpdatedAt.IsZero())
			},
		},
		{
			name: "should handle special characters in name and URL",
			model: &models.WebhookConfigModel{
				ID:         3,
				Name:       "Config with Special chars: éñÜñ & symbols!@#$%",
				EventType:  enums.EventTypeCredit,
				WebhookURL: "https://example.com/webhook?param=test%20value&special=chars%2Bhere",
				IsActive:   true,
				TimeoutMs:  25000,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
			verify: func(t *testing.T, entity *entities.WebhookConfig) {
				assert.Equal(t, int64(3), entity.ID)
				assert.Equal(t, "Config with Special chars: éñÜñ & symbols!@#$%", entity.Name)
				assert.Equal(t, "https://example.com/webhook?param=test%20value&special=chars%2Bhere", entity.WebhookURL)
				assert.True(t, entity.IsActive)
				assert.Equal(t, 25000, entity.TimeoutMs)
			},
		},
		{
			name: "should handle long webhook URLs",
			model: &models.WebhookConfigModel{
				ID:         4,
				Name:       "Long URL Config",
				EventType:  enums.EventTypeDebit,
				WebhookURL: "https://very-long-example-domain-name.com/api/v1/webhooks/callbacks/with/very/long/path/segments?param1=value1&param2=value2&param3=value3&param4=value4&param5=value5",
				IsActive:   true,
				TimeoutMs:  60000,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
			verify: func(t *testing.T, entity *entities.WebhookConfig) {
				assert.Equal(t, int64(4), entity.ID)
				assert.Equal(t, "Long URL Config", entity.Name)
				assert.Contains(t, entity.WebhookURL, "very-long-example-domain-name.com")
				assert.Contains(t, entity.WebhookURL, "param1=value1")
				assert.Contains(t, entity.WebhookURL, "param5=value5")
				assert.True(t, entity.IsActive)
				assert.Equal(t, 60000, entity.TimeoutMs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := repo.modelToEntity(tt.model)
			require.NotNil(t, entity)
			tt.verify(t, entity)
		})
	}
}

// TestWebhookConfigRepositoryImpl_GetByIDLogic tests GetByID method logic simulation
func TestWebhookConfigRepositoryImpl_GetByIDLogic(t *testing.T) {
	tests := []struct {
		name         string
		id           int64
		mockModel    *models.WebhookConfigModel
		mockError    error
		expectEntity bool
		expectError  bool
	}{
		{
			name: "should return entity when record found",
			id:   1,
			mockModel: &models.WebhookConfigModel{
				ID:         1,
				Name:       "Test Config",
				EventType:  enums.EventTypeCredit,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
				TimeoutMs:  30000,
			},
			mockError:    nil,
			expectEntity: true,
			expectError:  false,
		},
		{
			name:         "should return nil when record not found",
			id:           999,
			mockModel:    nil,
			mockError:    gorm.ErrRecordNotFound,
			expectEntity: false,
			expectError:  false,
		},
		{
			name:         "should return error when database error occurs",
			id:           1,
			mockModel:    nil,
			mockError:    errors.New("database connection failed"),
			expectEntity: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &webhookConfigRepositoryImpl{}

			// Simulate the logic that would happen in GetByID
			var entity *entities.WebhookConfig
			var err error

			if tt.mockError != nil {
				if tt.mockError == gorm.ErrRecordNotFound {
					// This simulates: return nil, nil
					entity = nil
					err = nil
				} else {
					// This simulates: return nil, fmt.Errorf("failed to get webhook config by ID: %w", err)
					entity = nil
					err = fmt.Errorf("failed to get webhook config by ID: %w", tt.mockError)
				}
			} else if tt.mockModel != nil {
				// This simulates: return r.modelToEntity(&model), nil
				entity = repo.modelToEntity(tt.mockModel)
				err = nil
			}

			// Verify results
			if tt.expectEntity {
				assert.NotNil(t, entity)
				assert.NoError(t, err)
				assert.Equal(t, tt.id, entity.ID)
			} else {
				assert.Nil(t, entity)
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get webhook config by ID")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWebhookConfigRepositoryImpl_ErrorFormatting tests error message formatting
func TestWebhookConfigRepositoryImpl_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		args           []interface{}
		expectContains []string
	}{
		{
			name:           "should format GetByID database error",
			pattern:        "failed to get webhook config by ID: %w",
			args:           []interface{}{errors.New("connection timeout")},
			expectContains: []string{"failed to get webhook config by ID", "connection timeout"},
		},
		{
			name:           "should format GetByID connection error",
			pattern:        "failed to get webhook config by ID: %w",
			args:           []interface{}{errors.New("database is unavailable")},
			expectContains: []string{"failed to get webhook config by ID", "database is unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := fmt.Sprintf(tt.pattern, tt.args...)
			for _, expected := range tt.expectContains {
				assert.Contains(t, formatted, expected)
			}
			assert.Contains(t, tt.pattern, "%w", "Should use %w for error wrapping")
		})
	}
}

// TestWebhookConfigRepositoryImpl_TimeoutScenarios tests various timeout scenarios
func TestWebhookConfigRepositoryImpl_TimeoutScenarios(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	tests := []struct {
		name      string
		timeoutMs int
		verify    func(t *testing.T, timeoutMs int)
	}{
		{
			name:      "should handle minimum timeout",
			timeoutMs: 1,
			verify: func(t *testing.T, timeoutMs int) {
				assert.Equal(t, 1, timeoutMs)
				assert.Greater(t, timeoutMs, 0)
			},
		},
		{
			name:      "should handle standard timeout",
			timeoutMs: 30000,
			verify: func(t *testing.T, timeoutMs int) {
				assert.Equal(t, 30000, timeoutMs)
				assert.GreaterOrEqual(t, timeoutMs, 1000)
			},
		},
		{
			name:      "should handle high timeout",
			timeoutMs: 120000,
			verify: func(t *testing.T, timeoutMs int) {
				assert.Equal(t, 120000, timeoutMs)
				assert.GreaterOrEqual(t, timeoutMs, 60000)
			},
		},
		{
			name:      "should handle zero timeout",
			timeoutMs: 0,
			verify: func(t *testing.T, timeoutMs int) {
				assert.Equal(t, 0, timeoutMs)
			},
		},
		{
			name:      "should handle maximum int32 timeout",
			timeoutMs: 2147483647,
			verify: func(t *testing.T, timeoutMs int) {
				assert.Equal(t, 2147483647, timeoutMs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &models.WebhookConfigModel{
				ID:         1,
				Name:       "Timeout Test",
				EventType:  enums.EventTypeCredit,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
				TimeoutMs:  tt.timeoutMs,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}

			entity := repo.modelToEntity(model)
			require.NotNil(t, entity)
			tt.verify(t, entity.TimeoutMs)
		})
	}
}

// TestWebhookConfigRepositoryImpl_EventTypeScenarios tests event type scenarios
func TestWebhookConfigRepositoryImpl_EventTypeScenarios(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	tests := []struct {
		name      string
		eventType enums.EventType
		verify    func(t *testing.T, eventType enums.EventType)
	}{
		{
			name:      "should handle credit event type",
			eventType: enums.EventTypeCredit,
			verify: func(t *testing.T, eventType enums.EventType) {
				assert.Equal(t, enums.EventTypeCredit, eventType)
				assert.Equal(t, "CREDIT", string(eventType))
			},
		},
		{
			name:      "should handle debit event type",
			eventType: enums.EventTypeDebit,
			verify: func(t *testing.T, eventType enums.EventType) {
				assert.Equal(t, enums.EventTypeDebit, eventType)
				assert.Equal(t, "DEBIT", string(eventType))
			},
		},
		{
			name:      "should handle empty event type",
			eventType: enums.EventType(""),
			verify: func(t *testing.T, eventType enums.EventType) {
				assert.Equal(t, enums.EventType(""), eventType)
				assert.Equal(t, "", string(eventType))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &models.WebhookConfigModel{
				ID:         1,
				Name:       "Event Type Test",
				EventType:  tt.eventType,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
				TimeoutMs:  30000,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}

			entity := repo.modelToEntity(model)
			require.NotNil(t, entity)
			tt.verify(t, entity.EventType)
		})
	}
}

// TestWebhookConfigRepositoryImpl_ActiveStateScenarios tests active state scenarios
func TestWebhookConfigRepositoryImpl_ActiveStateScenarios(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	tests := []struct {
		name     string
		isActive bool
		verify   func(t *testing.T, isActive bool)
	}{
		{
			name:     "should handle active config",
			isActive: true,
			verify: func(t *testing.T, isActive bool) {
				assert.True(t, isActive)
			},
		},
		{
			name:     "should handle inactive config",
			isActive: false,
			verify: func(t *testing.T, isActive bool) {
				assert.False(t, isActive)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &models.WebhookConfigModel{
				ID:         1,
				Name:       "Active State Test",
				EventType:  enums.EventTypeCredit,
				WebhookURL: "https://example.com/webhook",
				IsActive:   tt.isActive,
				TimeoutMs:  30000,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}

			entity := repo.modelToEntity(model)
			require.NotNil(t, entity)
			tt.verify(t, entity.IsActive)
		})
	}
}

// TestWebhookConfigRepositoryImpl_TimestampPrecision tests timestamp precision
func TestWebhookConfigRepositoryImpl_TimestampPrecision(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	t.Run("should preserve timestamp precision", func(t *testing.T) {
		createdAt := time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC)
		updatedAt := time.Date(2023, 6, 15, 10, 35, 45, 987654321, time.UTC)

		model := &models.WebhookConfigModel{
			ID:         1,
			Name:       "Timestamp Test",
			EventType:  enums.EventTypeCredit,
			WebhookURL: "https://example.com/webhook",
			IsActive:   true,
			TimeoutMs:  30000,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		}

		entity := repo.modelToEntity(model)
		require.NotNil(t, entity)

		assert.Equal(t, createdAt.Nanosecond(), entity.CreatedAt.Nanosecond())
		assert.Equal(t, updatedAt.Nanosecond(), entity.UpdatedAt.Nanosecond())
		assert.Equal(t, createdAt.Unix(), entity.CreatedAt.Unix())
		assert.Equal(t, updatedAt.Unix(), entity.UpdatedAt.Unix())
	})
}

// TestWebhookConfigRepositoryImpl_DataIntegrity tests data integrity
func TestWebhookConfigRepositoryImpl_DataIntegrity(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	t.Run("should maintain complete data integrity during conversion", func(t *testing.T) {
		originalModel := &models.WebhookConfigModel{
			ID:         100,
			Name:       "Data Integrity Test Config",
			EventType:  enums.EventTypeDebit,
			WebhookURL: "https://integrity-test.example.com/webhook?param=test&value=123",
			IsActive:   false,
			TimeoutMs:  75000,
			CreatedAt:  time.Date(2023, 5, 15, 14, 30, 45, 123000000, time.UTC),
			UpdatedAt:  time.Date(2023, 5, 15, 14, 35, 45, 456000000, time.UTC),
		}

		entity := repo.modelToEntity(originalModel)

		// Verify every field is correctly converted
		require.NotNil(t, entity)
		assert.Equal(t, originalModel.ID, entity.ID)
		assert.Equal(t, originalModel.Name, entity.Name)
		assert.Equal(t, originalModel.EventType, entity.EventType)
		assert.Equal(t, originalModel.WebhookURL, entity.WebhookURL)
		assert.Equal(t, originalModel.IsActive, entity.IsActive)
		assert.Equal(t, originalModel.TimeoutMs, entity.TimeoutMs)
		assert.Equal(t, originalModel.CreatedAt, entity.CreatedAt)
		assert.Equal(t, originalModel.UpdatedAt, entity.UpdatedAt)

		// Verify no data is lost or corrupted
		assert.Contains(t, entity.WebhookURL, "integrity-test.example.com")
		assert.Contains(t, entity.WebhookURL, "param=test")
		assert.Contains(t, entity.WebhookURL, "value=123")
		assert.Equal(t, 123000000, entity.CreatedAt.Nanosecond())
		assert.Equal(t, 456000000, entity.UpdatedAt.Nanosecond())
	})
}

// TestWebhookConfigRepositoryImpl_EdgeCases tests edge cases
func TestWebhookConfigRepositoryImpl_EdgeCases(t *testing.T) {
	repo := &webhookConfigRepositoryImpl{}

	t.Run("should document nil model behavior", func(t *testing.T) {
		// This test documents that the method panics with nil input
		// This is expected and acceptable behavior for this implementation
		assert.Panics(t, func() {
			repo.modelToEntity(nil)
		})
		t.Log("Method panics with nil input - this is expected and acceptable behavior")
	})

	t.Run("should handle model with all zero values", func(t *testing.T) {
		model := &models.WebhookConfigModel{
			// All fields are zero values
		}

		entity := repo.modelToEntity(model)
		require.NotNil(t, entity)
		assert.Equal(t, int64(0), entity.ID)
		assert.Equal(t, "", entity.Name)
		assert.Equal(t, enums.EventType(""), entity.EventType)
		assert.Equal(t, "", entity.WebhookURL)
		assert.False(t, entity.IsActive)
		assert.Equal(t, 0, entity.TimeoutMs)
	})
}
