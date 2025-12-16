package services

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"webhook-processor/internal/application/usecases"
	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/mocks"
)

func TestWebhookApplicationService_CreateWebhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := usecases.NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
	service := NewWebhookApplicationService(processor)

	t.Run("should create webhook successfully", func(t *testing.T) {
		ctx := context.Background()
		cmd := CreateWebhookCommand{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-123",
			ConfigID:  1,
		}

		// Mock the processor to succeed
		mockConfigRepo.EXPECT().
			GetByID(ctx, cmd.ConfigID).
			Return(&entities.WebhookConfig{
				ID:         cmd.ConfigID,
				EventType:  cmd.EventType,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
			}, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(nil).
			Times(1)

		// Execute
		result, err := service.CreateWebhook(ctx, cmd)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, "Webhook created successfully", result.Message)
		assert.False(t, result.CreatedAt.IsZero())
		assert.Empty(t, result.QueueID) // QueueID is not returned by current implementation
	})

	t.Run("should return error for invalid event type", func(t *testing.T) {
		ctx := context.Background()
		cmd := CreateWebhookCommand{
			EventType: enums.EventType("INVALID"),
			EventID:   "test-event-123",
			ConfigID:  1,
		}

		// Execute
		result, err := service.CreateWebhook(ctx, cmd)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Invalid event type")
		assert.True(t, result.CreatedAt.IsZero())
		assert.Empty(t, result.QueueID)
	})

	t.Run("should return error when processor fails", func(t *testing.T) {
		ctx := context.Background()
		cmd := CreateWebhookCommand{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-123",
			ConfigID:  999, // Non-existent config
		}

		// Mock the processor to fail
		mockConfigRepo.EXPECT().
			GetByID(ctx, cmd.ConfigID).
			Return(nil, nil). // Config not found
			Times(1)

		// Execute
		result, err := service.CreateWebhook(ctx, cmd)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Failed to create webhook")
		assert.True(t, result.CreatedAt.IsZero())
		assert.Empty(t, result.QueueID)
	})

	t.Run("should handle empty event ID", func(t *testing.T) {
		ctx := context.Background()
		cmd := CreateWebhookCommand{
			EventType: enums.EventTypeCredit,
			EventID:   "", // Empty event ID
			ConfigID:  1,
		}

		// Mock the processor to succeed (empty event ID should be allowed)
		mockConfigRepo.EXPECT().
			GetByID(ctx, cmd.ConfigID).
			Return(&entities.WebhookConfig{
				ID:         cmd.ConfigID,
				EventType:  cmd.EventType,
				WebhookURL: "https://example.com/webhook",
				IsActive:   true,
			}, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, webhook *entities.WebhookQueue) error {
				assert.Equal(t, "", webhook.EventID)
				return nil
			}).
			Times(1)

		// Execute
		result, err := service.CreateWebhook(ctx, cmd)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("should handle zero config ID", func(t *testing.T) {
		ctx := context.Background()
		cmd := CreateWebhookCommand{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-123",
			ConfigID:  0, // Zero config ID
		}

		// Mock the processor to return error for zero config ID
		mockConfigRepo.EXPECT().
			GetByID(ctx, int64(0)).
			Return(nil, nil). // No config found
			Times(1)

		// Execute
		result, err := service.CreateWebhook(ctx, cmd)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "webhook config not found")
	})
}

func TestWebhookApplicationService_GetHealth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := usecases.NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
	service := NewWebhookApplicationService(processor)

	t.Run("should return health status", func(t *testing.T) {
		ctx := context.Background()
		startTime := time.Now()

		// Execute
		result, err := service.GetHealth(ctx)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "healthy", result.Status)
		assert.Equal(t, "1.0.0", result.Version)
		assert.False(t, result.Timestamp.IsZero())
		assert.True(t, result.Timestamp.After(startTime.Add(-time.Second)) || result.Timestamp.Equal(startTime))

		// Check dependencies
		require.NotNil(t, result.Dependencies)
		assert.Equal(t, "connected", result.Dependencies["database"])
		assert.Equal(t, "running", result.Dependencies["workers"])

		// Check uptime (be more lenient with timing)
		assert.True(t, result.Uptime >= 0)            // Allow zero uptime for very fast tests
		assert.True(t, result.Uptime < time.Minute*5) // Be more lenient with timing
	})

	t.Run("should have consistent uptime", func(t *testing.T) {
		ctx := context.Background()

		// Get health twice with a small delay
		result1, err1 := service.GetHealth(ctx)
		time.Sleep(time.Millisecond * 10)
		result2, err2 := service.GetHealth(ctx)

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		require.NotNil(t, result1)
		require.NotNil(t, result2)

		// Second uptime should be greater than first
		assert.True(t, result2.Uptime > result1.Uptime,
			"Second uptime (%v) should be greater than first (%v)", result2.Uptime, result1.Uptime)
	})
}

func TestCreateWebhookCommand_Validation(t *testing.T) {
	tests := []struct {
		name        string
		cmd         CreateWebhookCommand
		expectValid bool
		description string
	}{
		{
			name: "valid credit command",
			cmd: CreateWebhookCommand{
				EventType: enums.EventTypeCredit,
				EventID:   "test-event-123",
				ConfigID:  1,
			},
			expectValid: true,
			description: "Valid credit command should pass validation",
		},
		{
			name: "valid debit command",
			cmd: CreateWebhookCommand{
				EventType: enums.EventTypeDebit,
				EventID:   "test-event-456",
				ConfigID:  2,
			},
			expectValid: true,
			description: "Valid debit command should pass validation",
		},
		{
			name: "invalid event type",
			cmd: CreateWebhookCommand{
				EventType: enums.EventType("INVALID"),
				EventID:   "test-event-123",
				ConfigID:  1,
			},
			expectValid: false,
			description: "Invalid event type should fail validation",
		},
		{
			name: "empty event type",
			cmd: CreateWebhookCommand{
				EventType: enums.EventType(""),
				EventID:   "test-event-123",
				ConfigID:  1,
			},
			expectValid: false,
			description: "Empty event type should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.EventType.Validate()

			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

func TestCreateWebhookResult_Structure(t *testing.T) {
	t.Run("should create result with all fields", func(t *testing.T) {
		now := time.Now().UTC()
		queueID := "test-queue-id"

		result := &CreateWebhookResult{
			Success:   true,
			Message:   "Webhook created successfully",
			QueueID:   queueID,
			CreatedAt: now,
		}

		assert.True(t, result.Success)
		assert.Equal(t, "Webhook created successfully", result.Message)
		assert.Equal(t, queueID, result.QueueID)
		assert.Equal(t, now, result.CreatedAt)
	})

	t.Run("should create error result", func(t *testing.T) {
		result := &CreateWebhookResult{
			Success: false,
			Message: "Failed to create webhook: config not found",
		}

		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Failed to create webhook")
		assert.Empty(t, result.QueueID)
		assert.True(t, result.CreatedAt.IsZero())
	})
}

func TestHealthResult_Structure(t *testing.T) {
	t.Run("should create health result with all fields", func(t *testing.T) {
		now := time.Now().UTC()
		uptime := time.Hour * 2

		result := &HealthResult{
			Status:    "healthy",
			Version:   "1.0.0",
			Timestamp: now,
			Dependencies: map[string]string{
				"database": "connected",
				"workers":  "running",
			},
			Uptime: uptime,
		}

		assert.Equal(t, "healthy", result.Status)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Equal(t, now, result.Timestamp)
		assert.Equal(t, "connected", result.Dependencies["database"])
		assert.Equal(t, "running", result.Dependencies["workers"])
		assert.Equal(t, uptime, result.Uptime)
	})
}

// Integration test with real processor (but mocked dependencies)
func TestWebhookApplicationService_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := usecases.NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
	service := NewWebhookApplicationService(processor)

	t.Run("should handle complete webhook creation flow", func(t *testing.T) {
		ctx := context.Background()

		// Test multiple webhook creations
		commands := []CreateWebhookCommand{
			{
				EventType: enums.EventTypeCredit,
				EventID:   "credit-event-1",
				ConfigID:  1,
			},
			{
				EventType: enums.EventTypeDebit,
				EventID:   "debit-event-1",
				ConfigID:  2,
			},
		}

		for i, cmd := range commands {
			// Mock successful creation for each command
			mockConfigRepo.EXPECT().
				GetByID(ctx, cmd.ConfigID).
				Return(&entities.WebhookConfig{
					ID:         cmd.ConfigID,
					EventType:  cmd.EventType,
					WebhookURL: "https://example.com/webhook",
					IsActive:   true,
				}, nil).
				Times(1)

			mockQueueRepo.EXPECT().
				Create(ctx, gomock.Any()).
				Return(nil).
				Times(1)

			// Execute
			result, err := service.CreateWebhook(ctx, cmd)

			// Assert
			assert.NoError(t, err, "Command %d should succeed", i)
			assert.True(t, result.Success, "Command %d should be successful", i)
		}

		// Test health check
		health, err := service.GetHealth(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
	})
}

// Benchmark tests
func BenchmarkWebhookApplicationService_CreateWebhook(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := usecases.NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
	service := NewWebhookApplicationService(processor)

	config := &entities.WebhookConfig{
		ID:         1,
		WebhookURL: "https://example.com/webhook",
		IsActive:   true,
	}

	mockConfigRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(config, nil).AnyTimes()
	mockQueueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	cmd := CreateWebhookCommand{
		EventType: enums.EventTypeCredit,
		EventID:   "test-event",
		ConfigID:  1,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CreateWebhook(ctx, cmd)
	}
}

func BenchmarkWebhookApplicationService_GetHealth(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := usecases.NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
	service := NewWebhookApplicationService(processor)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetHealth(ctx)
	}
}
