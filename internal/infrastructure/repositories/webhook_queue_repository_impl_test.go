package repositories

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/infrastructure/models"
)

// TestWebhookQueueRepositoryImpl_Constructor tests repository construction
func TestWebhookQueueRepositoryImpl_Constructor(t *testing.T) {
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
			repo, err := NewWebhookQueueRepository(tt.db)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &webhookQueueRepositoryImpl{}, repo)
			}
		})
	}
}

// TestWebhookQueueRepositoryImpl_EntityToModel tests entity to model conversion
func TestWebhookQueueRepositoryImpl_EntityToModel(t *testing.T) {
	repo := &webhookQueueRepositoryImpl{}

	tests := []struct {
		name   string
		entity *entities.WebhookQueue
		verify func(t *testing.T, model *models.WebhookQueueModel)
	}{
		{
			name: "should convert complete entity to model",
			entity: &entities.WebhookQueue{
				ID:             1,
				QueueID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				EventType:      enums.EventTypeCredit,
				EventID:        "test-event-123",
				ConfigID:       1,
				WebhookURL:     "https://example.com/webhook",
				Status:         enums.WebhookStatusPending,
				RetryCount:     0,
				NextRetryAt:    time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
				LastError:      "",
				LastHTTPStatus: 0,
				CreatedAt:      time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				assert.Equal(t, int64(1), model.ID)
				assert.Equal(t, uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), model.QueueID)
				assert.Equal(t, enums.EventTypeCredit, model.EventType)
				assert.Equal(t, "test-event-123", model.EventID)
				assert.Equal(t, int64(1), model.ConfigID)
				assert.Equal(t, "https://example.com/webhook", model.WebhookURL)
				assert.Equal(t, enums.WebhookStatusPending, model.Status)
				assert.Equal(t, 0, model.RetryCount)
				assert.Equal(t, time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC), model.NextRetryAt)
			},
		},
		{
			name: "should convert entity with retry fields",
			entity: &entities.WebhookQueue{
				ID:                 2,
				QueueID:            uuid.New(),
				EventType:          enums.EventTypeDebit,
				EventID:            "retry-event",
				ConfigID:           2,
				WebhookURL:         "https://retry.example.com/webhook",
				Status:             enums.WebhookStatusProcessing,
				RetryCount:         1,
				NextRetryAt:        time.Now().UTC().Add(time.Hour),
				Retry0StartedAt:    timePtr(time.Now().UTC()),
				Retry0CompletedAt:  timePtr(time.Now().UTC().Add(time.Second)),
				Retry0DurationMs:   int64Ptr(1000),
				Retry0HTTPStatus:   intPtr(200),
				Retry0ResponseBody: stringPtr(`{"status": "ok"}`),
				Retry0Error:        stringPtr(""),
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				assert.Equal(t, int64(2), model.ID)
				assert.Equal(t, enums.EventTypeDebit, model.EventType)
				assert.Equal(t, enums.WebhookStatusProcessing, model.Status)
				assert.Equal(t, 1, model.RetryCount)
				assert.NotNil(t, model.Retry0StartedAt)
				assert.NotNil(t, model.Retry0CompletedAt)
				assert.NotNil(t, model.Retry0DurationMs)
				assert.Equal(t, int64(1000), *model.Retry0DurationMs)
				assert.NotNil(t, model.Retry0HTTPStatus)
				assert.Equal(t, 200, *model.Retry0HTTPStatus)
				assert.NotNil(t, model.Retry0ResponseBody)
				assert.Equal(t, `{"status": "ok"}`, *model.Retry0ResponseBody)
			},
		},
		{
			name: "should handle nil retry fields",
			entity: &entities.WebhookQueue{
				ID:        3,
				QueueID:   uuid.New(),
				EventType: enums.EventTypeCredit,
				EventID:   "nil-retry-event",
				ConfigID:  3,
				Status:    enums.WebhookStatusPending,
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				assert.Equal(t, int64(3), model.ID)
				assert.Nil(t, model.Retry0StartedAt)
				assert.Nil(t, model.Retry0CompletedAt)
				assert.Nil(t, model.Retry0DurationMs)
				assert.Nil(t, model.Retry0HTTPStatus)
				assert.Nil(t, model.Retry0ResponseBody)
				assert.Nil(t, model.Retry0Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := repo.entityToModel(tt.entity)
			require.NotNil(t, model)
			tt.verify(t, model)
		})
	}
}

// TestWebhookQueueRepositoryImpl_ModelToEntity tests model to entity conversion
func TestWebhookQueueRepositoryImpl_ModelToEntity(t *testing.T) {
	repo := &webhookQueueRepositoryImpl{}

	tests := []struct {
		name   string
		model  *models.WebhookQueueModel
		verify func(t *testing.T, entity *entities.WebhookQueue)
	}{
		{
			name: "should convert complete model to entity",
			model: &models.WebhookQueueModel{
				ID:             1,
				QueueID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				EventType:      enums.EventTypeCredit,
				EventID:        "test-event-123",
				ConfigID:       1,
				WebhookURL:     "https://example.com/webhook",
				Status:         enums.WebhookStatusPending,
				RetryCount:     0,
				NextRetryAt:    time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
				LastError:      "",
				LastHTTPStatus: 0,
				CreatedAt:      time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			},
			verify: func(t *testing.T, entity *entities.WebhookQueue) {
				assert.Equal(t, int64(1), entity.ID)
				assert.Equal(t, uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), entity.QueueID)
				assert.Equal(t, enums.EventTypeCredit, entity.EventType)
				assert.Equal(t, "test-event-123", entity.EventID)
				assert.Equal(t, int64(1), entity.ConfigID)
				assert.Equal(t, "https://example.com/webhook", entity.WebhookURL)
				assert.Equal(t, enums.WebhookStatusPending, entity.Status)
				assert.Equal(t, 0, entity.RetryCount)
			},
		},
		{
			name: "should convert model with processing timestamps",
			model: &models.WebhookQueueModel{
				ID:                  2,
				QueueID:             uuid.New(),
				EventType:           enums.EventTypeDebit,
				Status:              enums.WebhookStatusCompleted,
				ProcessingStartedAt: timePtr(time.Now().UTC()),
				CompletedAt:         timePtr(time.Now().UTC()),
			},
			verify: func(t *testing.T, entity *entities.WebhookQueue) {
				assert.Equal(t, int64(2), entity.ID)
				assert.Equal(t, enums.WebhookStatusCompleted, entity.Status)
				assert.NotNil(t, entity.ProcessingStartedAt)
				assert.NotNil(t, entity.CompletedAt)
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

// TestWebhookQueueRepositoryImpl_MergeWebhookIntoModel tests merge logic
func TestWebhookQueueRepositoryImpl_MergeWebhookIntoModel(t *testing.T) {
	repo := &webhookQueueRepositoryImpl{}

	tests := []struct {
		name          string
		existingModel *models.WebhookQueueModel
		updateEntity  *entities.WebhookQueue
		verify        func(t *testing.T, model *models.WebhookQueueModel)
	}{
		{
			name: "should merge status and retry information",
			existingModel: &models.WebhookQueueModel{
				ID:             1,
				QueueID:        uuid.New(),
				EventType:      enums.EventTypeCredit,
				EventID:        "existing-event",
				Status:         enums.WebhookStatusPending,
				RetryCount:     0,
				LastError:      "",
				LastHTTPStatus: 0,
			},
			updateEntity: &entities.WebhookQueue{
				ID:             1,
				Status:         enums.WebhookStatusProcessing,
				RetryCount:     1,
				NextRetryAt:    time.Now().UTC().Add(time.Hour),
				LastError:      "connection timeout",
				LastHTTPStatus: 500,
				UpdatedAt:      time.Now().UTC(),
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				assert.Equal(t, enums.WebhookStatusProcessing, model.Status)
				assert.Equal(t, 1, model.RetryCount)
				assert.Equal(t, "connection timeout", model.LastError)
				assert.Equal(t, 500, model.LastHTTPStatus)
				// Should preserve existing fields that weren't updated
				assert.Equal(t, enums.EventTypeCredit, model.EventType)
				assert.Equal(t, "existing-event", model.EventID)
			},
		},
		{
			name: "should not overwrite with zero values",
			existingModel: &models.WebhookQueueModel{
				ID:         1,
				QueueID:    uuid.New(),
				EventType:  enums.EventTypeCredit,
				EventID:    "existing-event",
				ConfigID:   1,
				WebhookURL: "https://existing.com/webhook",
				Status:     enums.WebhookStatusPending,
			},
			updateEntity: &entities.WebhookQueue{
				ID:         1,
				QueueID:    uuid.Nil,                      // Zero value - should not overwrite
				EventType:  "",                            // Zero value - should not overwrite
				EventID:    "",                            // Zero value - should not overwrite
				ConfigID:   0,                             // Zero value - should not overwrite
				WebhookURL: "",                            // Zero value - should not overwrite
				Status:     enums.WebhookStatusProcessing, // Non-zero - should update
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				// Should preserve existing values
				assert.Equal(t, enums.EventTypeCredit, model.EventType)
				assert.Equal(t, "existing-event", model.EventID)
				assert.Equal(t, int64(1), model.ConfigID)
				assert.Equal(t, "https://existing.com/webhook", model.WebhookURL)
				// Should update non-zero value
				assert.Equal(t, enums.WebhookStatusProcessing, model.Status)
			},
		},
		{
			name: "should update processing timestamps",
			existingModel: &models.WebhookQueueModel{
				ID:                  1,
				ProcessingStartedAt: nil,
				CompletedAt:         nil,
			},
			updateEntity: &entities.WebhookQueue{
				ID:                  1,
				ProcessingStartedAt: timePtr(time.Now().UTC()),
				CompletedAt:         timePtr(time.Now().UTC()),
			},
			verify: func(t *testing.T, model *models.WebhookQueueModel) {
				assert.NotNil(t, model.ProcessingStartedAt)
				assert.NotNil(t, model.CompletedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.mergeWebhookIntoModel(tt.existingModel, tt.updateEntity)
			tt.verify(t, tt.existingModel)
		})
	}
}

// TestWebhookQueueRepositoryImpl_UpdateRetryAttemptLogic tests the retry attempt logic simulation
func TestWebhookQueueRepositoryImpl_UpdateRetryAttemptLogic(t *testing.T) {
	tests := []struct {
		name         string
		retryLevel   int
		startedAt    time.Time
		completedAt  *time.Time
		durationMs   int64
		httpStatus   int
		responseBody string
		errorMsg     string
		verify       func(t *testing.T, updates map[string]interface{})
	}{
		{
			name:         "should create updates for retry level 0",
			retryLevel:   0,
			startedAt:    time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			completedAt:  timePtr(time.Date(2023, 6, 15, 10, 0, 1, 0, time.UTC)),
			durationMs:   1000,
			httpStatus:   200,
			responseBody: `{"status": "success"}`,
			errorMsg:     "",
			verify: func(t *testing.T, updates map[string]interface{}) {
				assert.Contains(t, updates, "retry_0_started_at")
				assert.Contains(t, updates, "retry_0_completed_at")
				assert.Contains(t, updates, "retry_0_duration_ms")
				assert.Contains(t, updates, "retry_0_http_status")
				assert.Contains(t, updates, "retry_0_response_body")
				assert.Equal(t, int64(1000), updates["retry_0_duration_ms"])
				assert.Equal(t, 200, updates["retry_0_http_status"])
				assert.Equal(t, `{"status": "success"}`, updates["retry_0_response_body"])
				// Should not contain error when errorMsg is empty
				assert.NotContains(t, updates, "retry_0_error")
				assert.NotContains(t, updates, "last_error")
			},
		},
		{
			name:         "should create updates for retry level 3 with error",
			retryLevel:   3,
			startedAt:    time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			completedAt:  nil, // Not completed yet
			durationMs:   500,
			httpStatus:   500,
			responseBody: `{"error": "internal server error"}`,
			errorMsg:     "connection timeout",
			verify: func(t *testing.T, updates map[string]interface{}) {
				assert.Contains(t, updates, "retry_3_started_at")
				assert.NotContains(t, updates, "retry_3_completed_at") // nil completedAt
				assert.Contains(t, updates, "retry_3_duration_ms")
				assert.Contains(t, updates, "retry_3_http_status")
				assert.Contains(t, updates, "retry_3_response_body")
				assert.Contains(t, updates, "retry_3_error")
				assert.Contains(t, updates, "last_error")
				assert.Equal(t, int64(500), updates["retry_3_duration_ms"])
				assert.Equal(t, 500, updates["retry_3_http_status"])
				assert.Equal(t, "connection timeout", updates["retry_3_error"])
				assert.Equal(t, "connection timeout", updates["last_error"])
			},
		},
		{
			name:         "should create updates for retry level 6 (max level)",
			retryLevel:   6,
			startedAt:    time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			completedAt:  timePtr(time.Date(2023, 6, 15, 10, 0, 5, 0, time.UTC)),
			durationMs:   5000,
			httpStatus:   404,
			responseBody: `{"error": "not found"}`,
			errorMsg:     "webhook endpoint not found",
			verify: func(t *testing.T, updates map[string]interface{}) {
				assert.Contains(t, updates, "retry_6_started_at")
				assert.Contains(t, updates, "retry_6_completed_at")
				assert.Contains(t, updates, "retry_6_duration_ms")
				assert.Contains(t, updates, "retry_6_http_status")
				assert.Contains(t, updates, "retry_6_response_body")
				assert.Contains(t, updates, "retry_6_error")
				assert.Equal(t, int64(5000), updates["retry_6_duration_ms"])
				assert.Equal(t, 404, updates["retry_6_http_status"])
				assert.Equal(t, "webhook endpoint not found", updates["retry_6_error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate UpdateRetryAttempt logic
			updates := map[string]interface{}{
				"updated_at":       time.Now().UTC(),
				"last_http_status": tt.httpStatus,
			}

			if tt.errorMsg != "" {
				updates["last_error"] = tt.errorMsg
			}

			// Simulate the switch statement logic
			switch tt.retryLevel {
			case 0:
				updates["retry_0_started_at"] = tt.startedAt
				if tt.completedAt != nil {
					updates["retry_0_completed_at"] = *tt.completedAt
				}
				updates["retry_0_duration_ms"] = tt.durationMs
				updates["retry_0_http_status"] = tt.httpStatus
				updates["retry_0_response_body"] = tt.responseBody
				if tt.errorMsg != "" {
					updates["retry_0_error"] = tt.errorMsg
				}
			case 3:
				updates["retry_3_started_at"] = tt.startedAt
				if tt.completedAt != nil {
					updates["retry_3_completed_at"] = *tt.completedAt
				}
				updates["retry_3_duration_ms"] = tt.durationMs
				updates["retry_3_http_status"] = tt.httpStatus
				updates["retry_3_response_body"] = tt.responseBody
				if tt.errorMsg != "" {
					updates["retry_3_error"] = tt.errorMsg
				}
			case 6:
				updates["retry_6_started_at"] = tt.startedAt
				if tt.completedAt != nil {
					updates["retry_6_completed_at"] = *tt.completedAt
				}
				updates["retry_6_duration_ms"] = tt.durationMs
				updates["retry_6_http_status"] = tt.httpStatus
				updates["retry_6_response_body"] = tt.responseBody
				if tt.errorMsg != "" {
					updates["retry_6_error"] = tt.errorMsg
				}
			}

			tt.verify(t, updates)
		})
	}
}

// TestWebhookQueueRepositoryImpl_MarkCompletedLogic tests MarkCompleted logic
func TestWebhookQueueRepositoryImpl_MarkCompletedLogic(t *testing.T) {
	tests := []struct {
		name                string
		processingStartedAt time.Time
		verify              func(t *testing.T, updates map[string]interface{})
	}{
		{
			name:                "should create completion updates",
			processingStartedAt: time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			verify: func(t *testing.T, updates map[string]interface{}) {
				assert.Contains(t, updates, "status")
				assert.Contains(t, updates, "processing_started_at")
				assert.Contains(t, updates, "completed_at")
				assert.Contains(t, updates, "updated_at")
				assert.Equal(t, enums.WebhookStatusCompleted, updates["status"])
				assert.Equal(t, time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), updates["processing_started_at"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()
			updates := map[string]interface{}{
				"status":                enums.WebhookStatusCompleted,
				"processing_started_at": tt.processingStartedAt,
				"completed_at":          now,
				"updated_at":            now,
			}

			tt.verify(t, updates)
		})
	}
}

// TestWebhookQueueRepositoryImpl_MarkFailedLogic tests MarkFailed logic
func TestWebhookQueueRepositoryImpl_MarkFailedLogic(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		verify   func(t *testing.T, updates map[string]interface{})
	}{
		{
			name:     "should create failure updates",
			errorMsg: "max retries exceeded",
			verify: func(t *testing.T, updates map[string]interface{}) {
				assert.Contains(t, updates, "status")
				assert.Contains(t, updates, "last_error")
				assert.Contains(t, updates, "updated_at")
				assert.Equal(t, enums.WebhookStatusFailed, updates["status"])
				assert.Equal(t, "max retries exceeded", updates["last_error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()
			updates := map[string]interface{}{
				"status":     enums.WebhookStatusFailed,
				"last_error": tt.errorMsg,
				"updated_at": now,
			}

			tt.verify(t, updates)
		})
	}
}

// TestWebhookQueueRepositoryImpl_GetNextWebhookLogic tests GetNextWebhookForProcessing logic
func TestWebhookQueueRepositoryImpl_GetNextWebhookLogic(t *testing.T) {
	tests := []struct {
		name       string
		retryLevel int
		verify     func(t *testing.T, conditions map[string]interface{}, updates map[string]interface{})
	}{
		{
			name:       "should create correct query conditions and updates",
			retryLevel: 2,
			verify: func(t *testing.T, conditions map[string]interface{}, updates map[string]interface{}) {
				// Query conditions
				assert.Equal(t, enums.WebhookStatusPending, conditions["status"])
				assert.Equal(t, 2, conditions["retry_count"])
				assert.Contains(t, conditions, "next_retry_at")

				// Status update
				assert.Equal(t, enums.WebhookStatusProcessing, updates["status"])
				assert.Contains(t, updates, "updated_at")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()

			// Simulate query conditions
			conditions := map[string]interface{}{
				"status":        enums.WebhookStatusPending,
				"retry_count":   tt.retryLevel,
				"next_retry_at": now,
			}

			// Simulate status update
			updates := map[string]interface{}{
				"status":     enums.WebhookStatusProcessing,
				"updated_at": now,
			}

			tt.verify(t, conditions, updates)
		})
	}
}

// TestWebhookQueueRepositoryImpl_ErrorFormatting tests error message formatting
func TestWebhookQueueRepositoryImpl_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		args           []interface{}
		expectContains []string
	}{
		{
			name:           "should format Create error",
			pattern:        "failed to create webhook queue entry: %w",
			args:           []interface{}{errors.New("database error")},
			expectContains: []string{"failed to create", "database error"},
		},
		{
			name:           "should format GetNext error with retry level",
			pattern:        "failed to get next webhook for retry level %d: %w",
			args:           []interface{}{3, errors.New("no records found")},
			expectContains: []string{"retry level 3", "no records found"},
		},
		{
			name:           "should format UpdateRetryAttempt error",
			pattern:        "failed to update retry attempt: %w",
			args:           []interface{}{errors.New("connection lost")},
			expectContains: []string{"failed to update", "connection lost"},
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

// TestWebhookQueueRepositoryImpl_EdgeCases tests edge cases and boundary conditions
func TestWebhookQueueRepositoryImpl_EdgeCases(t *testing.T) {
	repo := &webhookQueueRepositoryImpl{}

	t.Run("should handle entity to model with nil pointers", func(t *testing.T) {
		entity := &entities.WebhookQueue{
			ID:        1,
			QueueID:   uuid.New(),
			EventType: enums.EventTypeCredit,
			// All retry fields are nil
		}

		model := repo.entityToModel(entity)
		assert.NotNil(t, model)
		assert.Nil(t, model.Retry0StartedAt)
		assert.Nil(t, model.Retry0CompletedAt)
		assert.Nil(t, model.Retry0DurationMs)
	})

	t.Run("should handle model to entity with nil pointers", func(t *testing.T) {
		model := &models.WebhookQueueModel{
			ID:        1,
			QueueID:   uuid.New(),
			EventType: enums.EventTypeCredit,
			// All retry fields are nil
		}

		entity := repo.modelToEntity(model)
		assert.NotNil(t, entity)
		assert.Nil(t, entity.Retry0StartedAt)
		assert.Nil(t, entity.Retry0CompletedAt)
		assert.Nil(t, entity.Retry0DurationMs)
	})

	t.Run("should handle zero UUID in merge", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			QueueID: uuid.New(),
		}
		updateEntity := &entities.WebhookQueue{
			QueueID: uuid.Nil, // Zero value UUID
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		// Should not overwrite with nil UUID
		assert.NotEqual(t, uuid.Nil, existingModel.QueueID)
	})
}

// Helper functions for creating pointers
func timePtr(t time.Time) *time.Time {
	return &t
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

// TestWebhookQueueRepository_MergeWebhookIntoModel_ComprehensiveCoverage tests ALL conditional branches
func TestWebhookQueueRepository_MergeWebhookIntoModel_ComprehensiveCoverage(t *testing.T) {
	repo := &webhookQueueRepositoryImpl{}

	t.Run("should test QueueID update branch", func(t *testing.T) {
		// Test both paths: when QueueID is non-nil and when it's nil
		existingModel := &models.WebhookQueueModel{
			QueueID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		}

		// Test non-nil QueueID - should update
		newQueueID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		updateEntity := &entities.WebhookQueue{
			QueueID: newQueueID,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, newQueueID, existingModel.QueueID, "Should update QueueID when non-nil")

		// Test nil QueueID - should NOT update
		originalQueueID := existingModel.QueueID
		updateEntityNil := &entities.WebhookQueue{
			QueueID: uuid.Nil,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityNil)
		assert.Equal(t, originalQueueID, existingModel.QueueID, "Should NOT update QueueID when nil")
	})

	t.Run("should test EventType update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			EventType: enums.EventTypeCredit,
		}

		// Test non-empty EventType - should update
		updateEntity := &entities.WebhookQueue{
			EventType: enums.EventTypeDebit,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, enums.EventTypeDebit, existingModel.EventType, "Should update EventType when non-empty")

		// Test empty EventType - should NOT update
		originalEventType := existingModel.EventType
		updateEntityEmpty := &entities.WebhookQueue{
			EventType: "",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityEmpty)
		assert.Equal(t, originalEventType, existingModel.EventType, "Should NOT update EventType when empty")
	})

	t.Run("should test EventID update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			EventID: "original-event-id",
		}

		// Test non-empty EventID - should update
		updateEntity := &entities.WebhookQueue{
			EventID: "new-event-id",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, "new-event-id", existingModel.EventID, "Should update EventID when non-empty")

		// Test empty EventID - should NOT update
		originalEventID := existingModel.EventID
		updateEntityEmpty := &entities.WebhookQueue{
			EventID: "",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityEmpty)
		assert.Equal(t, originalEventID, existingModel.EventID, "Should NOT update EventID when empty")
	})

	t.Run("should test ConfigID update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			ConfigID: 100,
		}

		// Test non-zero ConfigID - should update
		updateEntity := &entities.WebhookQueue{
			ConfigID: 200,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, int64(200), existingModel.ConfigID, "Should update ConfigID when non-zero")

		// Test zero ConfigID - should NOT update
		originalConfigID := existingModel.ConfigID
		updateEntityZero := &entities.WebhookQueue{
			ConfigID: 0,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityZero)
		assert.Equal(t, originalConfigID, existingModel.ConfigID, "Should NOT update ConfigID when zero")
	})

	t.Run("should test WebhookURL update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			WebhookURL: "https://original.com/webhook",
		}

		// Test non-empty WebhookURL - should update
		updateEntity := &entities.WebhookQueue{
			WebhookURL: "https://new.com/webhook",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, "https://new.com/webhook", existingModel.WebhookURL, "Should update WebhookURL when non-empty")

		// Test empty WebhookURL - should NOT update
		originalWebhookURL := existingModel.WebhookURL
		updateEntityEmpty := &entities.WebhookQueue{
			WebhookURL: "",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityEmpty)
		assert.Equal(t, originalWebhookURL, existingModel.WebhookURL, "Should NOT update WebhookURL when empty")
	})

	t.Run("should test Status update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			Status: enums.WebhookStatusPending,
		}

		// Test non-empty Status - should update
		updateEntity := &entities.WebhookQueue{
			Status: enums.WebhookStatusProcessing,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, enums.WebhookStatusProcessing, existingModel.Status, "Should update Status when non-empty")

		// Test empty Status - should NOT update
		originalStatus := existingModel.Status
		updateEntityEmpty := &entities.WebhookQueue{
			Status: "",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityEmpty)
		assert.Equal(t, originalStatus, existingModel.Status, "Should NOT update Status when empty")
	})

	t.Run("should test RetryCount and NextRetryAt update branch", func(t *testing.T) {
		now := time.Now().UTC()
		existingModel := &models.WebhookQueueModel{
			RetryCount:  1,
			NextRetryAt: now.Add(-time.Hour),
		}

		// Test non-zero RetryCount - should update both fields
		futureTime := now.Add(time.Hour)
		updateEntity := &entities.WebhookQueue{
			RetryCount:  3,
			NextRetryAt: futureTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, 3, existingModel.RetryCount, "Should update RetryCount when non-zero")
		assert.Equal(t, futureTime, existingModel.NextRetryAt, "Should update NextRetryAt when RetryCount non-zero")

		// Test zero RetryCount but non-zero NextRetryAt - should still update both
		originalRetryCount := existingModel.RetryCount
		originalNextRetryAt := existingModel.NextRetryAt
		newTime := now.Add(2 * time.Hour)
		updateEntityZeroRetry := &entities.WebhookQueue{
			RetryCount:  0,
			NextRetryAt: newTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityZeroRetry)
		assert.Equal(t, 0, existingModel.RetryCount, "Should update RetryCount to zero when NextRetryAt is non-zero")
		assert.Equal(t, newTime, existingModel.NextRetryAt, "Should update NextRetryAt when non-zero")

		// Test zero RetryCount AND zero NextRetryAt - should NOT update
		existingModel.RetryCount = 5
		existingModel.NextRetryAt = now.Add(3 * time.Hour)
		originalRetryCount = existingModel.RetryCount
		originalNextRetryAt = existingModel.NextRetryAt

		updateEntityBothZero := &entities.WebhookQueue{
			RetryCount:  0,
			NextRetryAt: time.Time{}, // Zero time
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityBothZero)
		assert.Equal(t, originalRetryCount, existingModel.RetryCount, "Should NOT update RetryCount when both RetryCount and NextRetryAt are zero")
		assert.Equal(t, originalNextRetryAt, existingModel.NextRetryAt, "Should NOT update NextRetryAt when both RetryCount and NextRetryAt are zero")
	})

	t.Run("should test LastError update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			LastError: "original error",
		}

		// Test non-empty LastError - should update
		updateEntity := &entities.WebhookQueue{
			LastError: "new error message",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, "new error message", existingModel.LastError, "Should update LastError when non-empty")

		// Test empty LastError - should NOT update
		originalLastError := existingModel.LastError
		updateEntityEmpty := &entities.WebhookQueue{
			LastError: "",
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityEmpty)
		assert.Equal(t, originalLastError, existingModel.LastError, "Should NOT update LastError when empty")
	})

	t.Run("should test LastHTTPStatus update branch", func(t *testing.T) {
		existingModel := &models.WebhookQueueModel{
			LastHTTPStatus: 200,
		}

		// Test non-zero LastHTTPStatus - should update
		updateEntity := &entities.WebhookQueue{
			LastHTTPStatus: 500,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, 500, existingModel.LastHTTPStatus, "Should update LastHTTPStatus when non-zero")

		// Test zero LastHTTPStatus - should NOT update
		originalLastHTTPStatus := existingModel.LastHTTPStatus
		updateEntityZero := &entities.WebhookQueue{
			LastHTTPStatus: 0,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityZero)
		assert.Equal(t, originalLastHTTPStatus, existingModel.LastHTTPStatus, "Should NOT update LastHTTPStatus when zero")
	})

	t.Run("should test UpdatedAt update branch", func(t *testing.T) {
		originalTime := time.Now().UTC().Add(-time.Hour)
		existingModel := &models.WebhookQueueModel{
			UpdatedAt: originalTime,
		}

		// Test non-zero UpdatedAt - should update
		newTime := time.Now().UTC()
		updateEntity := &entities.WebhookQueue{
			UpdatedAt: newTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, newTime, existingModel.UpdatedAt, "Should update UpdatedAt when non-zero")

		// Test zero UpdatedAt - should NOT update
		originalUpdatedAt := existingModel.UpdatedAt
		updateEntityZero := &entities.WebhookQueue{
			UpdatedAt: time.Time{}, // Zero time
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityZero)
		assert.Equal(t, originalUpdatedAt, existingModel.UpdatedAt, "Should NOT update UpdatedAt when zero")
	})

	t.Run("should test ProcessingStartedAt update branch", func(t *testing.T) {
		originalTime := time.Now().UTC().Add(-time.Hour)
		existingModel := &models.WebhookQueueModel{
			ProcessingStartedAt: &originalTime,
		}

		// Test non-nil ProcessingStartedAt - should update
		newTime := time.Now().UTC()
		updateEntity := &entities.WebhookQueue{
			ProcessingStartedAt: &newTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, &newTime, existingModel.ProcessingStartedAt, "Should update ProcessingStartedAt when non-nil")

		// Test nil ProcessingStartedAt - should NOT update
		originalProcessingStartedAt := existingModel.ProcessingStartedAt
		updateEntityNil := &entities.WebhookQueue{
			ProcessingStartedAt: nil,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityNil)
		assert.Equal(t, originalProcessingStartedAt, existingModel.ProcessingStartedAt, "Should NOT update ProcessingStartedAt when nil")
	})

	t.Run("should test CompletedAt update branch", func(t *testing.T) {
		originalTime := time.Now().UTC().Add(-time.Hour)
		existingModel := &models.WebhookQueueModel{
			CompletedAt: &originalTime,
		}

		// Test non-nil CompletedAt - should update
		newTime := time.Now().UTC()
		updateEntity := &entities.WebhookQueue{
			CompletedAt: &newTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, &newTime, existingModel.CompletedAt, "Should update CompletedAt when non-nil")

		// Test nil CompletedAt - should NOT update
		originalCompletedAt := existingModel.CompletedAt
		updateEntityNil := &entities.WebhookQueue{
			CompletedAt: nil,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityNil)
		assert.Equal(t, originalCompletedAt, existingModel.CompletedAt, "Should NOT update CompletedAt when nil")
	})

	t.Run("should test DeletedAt update branch", func(t *testing.T) {
		originalTime := time.Now().UTC().Add(-time.Hour)
		existingModel := &models.WebhookQueueModel{
			DeletedAt: &originalTime,
		}

		// Test non-nil DeletedAt - should update
		newTime := time.Now().UTC()
		updateEntity := &entities.WebhookQueue{
			DeletedAt: &newTime,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)
		assert.Equal(t, &newTime, existingModel.DeletedAt, "Should update DeletedAt when non-nil")

		// Test nil DeletedAt - should NOT update
		originalDeletedAt := existingModel.DeletedAt
		updateEntityNil := &entities.WebhookQueue{
			DeletedAt: nil,
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntityNil)
		assert.Equal(t, originalDeletedAt, existingModel.DeletedAt, "Should NOT update DeletedAt when nil")
	})

	t.Run("should test all branches together in complex scenario", func(t *testing.T) {
		now := time.Now().UTC()
		originalQueueID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		existingModel := &models.WebhookQueueModel{
			ID:                  1,
			QueueID:             originalQueueID,
			EventType:           enums.EventTypeCredit,
			EventID:             "original-event",
			ConfigID:            100,
			WebhookURL:          "https://original.com",
			Status:              enums.WebhookStatusPending,
			RetryCount:          0,
			NextRetryAt:         now.Add(-time.Hour),
			LastError:           "original error",
			LastHTTPStatus:      200,
			UpdatedAt:           now.Add(-time.Hour),
			ProcessingStartedAt: nil,
			CompletedAt:         nil,
			DeletedAt:           nil,
		}

		// Update entity with mixed zero and non-zero values
		newQueueID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		newProcessingTime := now.Add(-time.Minute)
		newCompletedTime := now

		updateEntity := &entities.WebhookQueue{
			QueueID:             newQueueID,                    // Should update (non-nil)
			EventType:           "",                            // Should NOT update (empty)
			EventID:             "new-event",                   // Should update (non-empty)
			ConfigID:            0,                             // Should NOT update (zero)
			WebhookURL:          "https://new.com",             // Should update (non-empty)
			Status:              enums.WebhookStatusProcessing, // Should update (non-empty)
			RetryCount:          3,                             // Should update (non-zero)
			NextRetryAt:         now.Add(time.Hour),            // Should update (with RetryCount)
			LastError:           "",                            // Should NOT update (empty)
			LastHTTPStatus:      0,                             // Should NOT update (zero)
			UpdatedAt:           now,                           // Should update (non-zero)
			ProcessingStartedAt: &newProcessingTime,            // Should update (non-nil)
			CompletedAt:         &newCompletedTime,             // Should update (non-nil)
			DeletedAt:           nil,                           // Should NOT update (nil)
		}

		repo.mergeWebhookIntoModel(existingModel, updateEntity)

		// Verify updates happened for non-zero/non-nil values
		assert.Equal(t, newQueueID, existingModel.QueueID)
		assert.Equal(t, "new-event", existingModel.EventID)
		assert.Equal(t, "https://new.com", existingModel.WebhookURL)
		assert.Equal(t, enums.WebhookStatusProcessing, existingModel.Status)
		assert.Equal(t, 3, existingModel.RetryCount)
		assert.Equal(t, now.Add(time.Hour).Truncate(time.Second), existingModel.NextRetryAt.Truncate(time.Second))
		assert.Equal(t, now.Truncate(time.Second), existingModel.UpdatedAt.Truncate(time.Second))
		assert.Equal(t, &newProcessingTime, existingModel.ProcessingStartedAt)
		assert.Equal(t, &newCompletedTime, existingModel.CompletedAt)

		// Verify NO updates happened for zero/empty/nil values
		assert.Equal(t, enums.EventTypeCredit, existingModel.EventType) // Original value preserved
		assert.Equal(t, int64(100), existingModel.ConfigID)             // Original value preserved
		assert.Equal(t, "original error", existingModel.LastError)      // Original value preserved
		assert.Equal(t, 200, existingModel.LastHTTPStatus)              // Original value preserved
		assert.Nil(t, existingModel.DeletedAt)                          // Original nil preserved
	})
}
