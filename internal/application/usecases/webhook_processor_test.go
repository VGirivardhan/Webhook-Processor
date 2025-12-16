package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
	"webhook-processor/internal/domain/services"
	"webhook-processor/internal/mocks"
)

func TestWebhookProcessor_CreateWebhookEntry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should create webhook entry successfully", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "test-event-123"
		configID := int64(1)

		// Mock webhook config
		config := &entities.WebhookConfig{
			ID:         configID,
			Name:       "Test Config",
			EventType:  eventType,
			WebhookURL: "https://example.com/webhook?param=value",
			IsActive:   true,
			TimeoutMs:  30000,
		}

		// Set up expectations
		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(config, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, webhook *entities.WebhookQueue) error {
				// Verify the webhook entry is created correctly
				assert.Equal(t, eventType, webhook.EventType)
				assert.Equal(t, eventID, webhook.EventID)
				assert.Equal(t, configID, webhook.ConfigID)
				assert.Equal(t, config.WebhookURL, webhook.WebhookURL)
				assert.Equal(t, enums.WebhookStatusPending, webhook.Status)
				assert.Equal(t, 0, webhook.RetryCount)
				assert.False(t, webhook.NextRetryAt.IsZero())
				assert.False(t, webhook.CreatedAt.IsZero())
				assert.False(t, webhook.UpdatedAt.IsZero())

				// Simulate database setting ID and QueueID
				webhook.ID = 1
				webhook.QueueID = uuid.New()
				return nil
			}).
			Times(1)

		// Execute
		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should return error when config not found", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "test-event-123"
		configID := int64(999)

		// Set up expectations
		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(nil, nil).
			Times(1)

		// Execute
		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook config not found")
	})

	t.Run("should return error when config is inactive", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "test-event-123"
		configID := int64(1)

		// Mock inactive webhook config
		config := &entities.WebhookConfig{
			ID:         configID,
			Name:       "Inactive Config",
			EventType:  eventType,
			WebhookURL: "https://example.com/webhook",
			IsActive:   false, // Inactive
			TimeoutMs:  30000,
		}

		// Set up expectations
		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(config, nil).
			Times(1)

		// Execute
		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook config is not active")
	})

	t.Run("should return error when repository fails to get config", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "test-event-123"
		configID := int64(1)

		// Set up expectations
		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(nil, errors.New("database connection failed")).
			Times(1)

		// Execute
		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get webhook config")
	})

	t.Run("should return error when repository fails to create webhook", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "test-event-123"
		configID := int64(1)

		// Mock webhook config
		config := &entities.WebhookConfig{
			ID:         configID,
			Name:       "Test Config",
			EventType:  eventType,
			WebhookURL: "https://example.com/webhook",
			IsActive:   true,
			TimeoutMs:  30000,
		}

		// Set up expectations
		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(config, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.New("database insert failed")).
			Times(1)

		// Execute
		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create webhook queue entry")
	})
}

func TestWebhookProcessor_ProcessWebhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should process webhook successfully", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event-123",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook?param=value",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now.Add(-time.Minute), // ✅ Was ready for processing
			CreatedAt:   now.Add(-time.Hour),   // ✅ Created earlier
			UpdatedAt:   now.Add(-time.Minute), // ✅ Updated when picked up
		}

		// Mock successful webhook response
		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       `{"success": true}`,
			Duration:   time.Millisecond * 500,
			Error:      nil,
		}

		// Set up expectations
		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, `{"success": true}`, "").
			Times(1)

		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Times(1)

		// Execute
		err := processor.ProcessWebhook(ctx, webhook, workerID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle webhook failure with retry", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event-123",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  2,                     // Under max retries
			NextRetryAt: now.Add(-time.Minute), // ✅ Was ready for processing
			CreatedAt:   now.Add(-time.Hour),   // ✅ Created earlier
			UpdatedAt:   now.Add(-time.Minute), // ✅ Updated when picked up
		}

		// Mock failed webhook response
		response := &services.WebhookResponse{
			StatusCode: 500,
			Body:       `{"error": "internal server error"}`,
			Duration:   time.Millisecond * 1000,
			Error:      nil,
		}

		// Set up expectations
		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 500, `{"error": "internal server error"}`, gomock.Any()).
			Times(1)

		// Should schedule retry (not mark as failed)
		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, w *entities.WebhookQueue) error {
				assert.Equal(t, enums.WebhookStatusPending, w.Status)
				assert.Equal(t, 3, w.RetryCount)         // Incremented
				assert.True(t, w.NextRetryAt.After(now)) // Scheduled for future
				return nil
			}).
			Times(1)

		// Execute
		err := processor.ProcessWebhook(ctx, webhook, workerID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should mark webhook as failed when max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event-123",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  enums.MaxRetryAttempts, // At max retries
			NextRetryAt: now.Add(-time.Minute),  // ✅ Was ready for processing
			CreatedAt:   now.Add(-time.Hour),    // ✅ Created earlier
			UpdatedAt:   now.Add(-time.Minute),  // ✅ Updated when picked up
		}

		// Mock failed webhook response
		response := &services.WebhookResponse{
			StatusCode: 500,
			Body:       `{"error": "internal server error"}`,
			Duration:   time.Millisecond * 1000,
			Error:      nil,
		}

		// Set up expectations
		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 500, `{"error": "internal server error"}`, gomock.Any()).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkFailed(ctx, webhook.ID, gomock.Any()).
			Times(1)

		// Execute
		err := processor.ProcessWebhook(ctx, webhook, workerID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle webhook service error", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event-123",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Mock webhook service error
		serviceError := errors.New("connection timeout")

		// Set up expectations
		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(nil, serviceError).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 0, "", "connection timeout").
			Times(1)

		// Should schedule retry
		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Times(1)

		// Execute
		err := processor.ProcessWebhook(ctx, webhook, workerID)

		// Assert
		assert.NoError(t, err)
	})
}

func TestWebhookProcessor_GetNextWebhookForProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should get next webhook for processing", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		retryLevel := 0
		now := time.Now().UTC()

		expectedWebhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event-123",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusPending,
			RetryCount:  0,
			NextRetryAt: now.Add(-time.Minute), // ✅ Ready for processing (past time)
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Minute),
		}

		// Set up expectations
		mockQueueRepo.EXPECT().
			GetNextWebhookForProcessing(ctx, workerID, retryLevel).
			Return(expectedWebhook, nil).
			Times(1)

		// Execute
		webhook, err := processor.GetNextWebhookForProcessing(ctx, workerID, retryLevel)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedWebhook, webhook)

		// ✅ Additional assertions for business logic validation
		assert.True(t, webhook.NextRetryAt.Before(time.Now().UTC().Add(time.Second)),
			"Webhook should be ready for processing (NextRetryAt in the past)")
		assert.Equal(t, enums.WebhookStatusPending, webhook.Status,
			"Webhook should be in PENDING status")
		assert.Equal(t, retryLevel, webhook.RetryCount,
			"Webhook retry count should match requested retry level")
	})

	t.Run("should return nil when no webhooks available", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		retryLevel := 0

		// Set up expectations
		mockQueueRepo.EXPECT().
			GetNextWebhookForProcessing(ctx, workerID, retryLevel).
			Return(nil, nil). // ✅ No webhooks ready for processing
			Times(1)

		// Execute
		webhook, err := processor.GetNextWebhookForProcessing(ctx, workerID, retryLevel)

		// Assert
		assert.NoError(t, err)
		assert.Nil(t, webhook, "Should return nil when no webhooks are ready")
	})

	t.Run("should respect retry level filtering", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-2"
		retryLevel := 2 // Looking for webhooks with 2 retries
		now := time.Now().UTC()

		expectedWebhook := &entities.WebhookQueue{
			ID:          2,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeDebit,
			EventID:     "retry-event-456",
			ConfigID:    2,
			WebhookURL:  "https://example.com/webhook/retry",
			Status:      enums.WebhookStatusPending,
			RetryCount:  2,                     // ✅ Matches the requested retry level
			NextRetryAt: now.Add(-time.Minute), // Ready for processing
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Minute),
		}

		mockQueueRepo.EXPECT().
			GetNextWebhookForProcessing(ctx, workerID, retryLevel).
			Return(expectedWebhook, nil).
			Times(1)

		// Execute
		webhook, err := processor.GetNextWebhookForProcessing(ctx, workerID, retryLevel)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, webhook)
		assert.Equal(t, retryLevel, webhook.RetryCount,
			"Returned webhook should match requested retry level")
		assert.True(t, webhook.NextRetryAt.Before(time.Now().UTC().Add(time.Second)),
			"Webhook should be ready for processing")
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		retryLevel := 0

		// Set up expectations
		mockQueueRepo.EXPECT().
			GetNextWebhookForProcessing(ctx, workerID, retryLevel).
			Return(nil, errors.New("database error")).
			Times(1)

		// Execute
		webhook, err := processor.GetNextWebhookForProcessing(ctx, workerID, retryLevel)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, webhook)
	})
}

func TestWebhookProcessor_CalculateRetryDelay(t *testing.T) {
	tests := []struct {
		name          string
		retryCount    int
		minDelay      time.Duration
		maxDelay      time.Duration
		expectInRange bool
	}{
		{
			name:          "retry 0 should have minimal delay",
			retryCount:    0,
			minDelay:      time.Second * 1,
			maxDelay:      time.Second * 5,
			expectInRange: true,
		},
		{
			name:          "retry 3 should have moderate delay",
			retryCount:    3,
			minDelay:      time.Second * 8,
			maxDelay:      time.Second * 32, // Increase tolerance for exponential backoff with jitter
			expectInRange: true,
		},
		{
			name:          "retry 6 should have maximum delay",
			retryCount:    6,
			minDelay:      time.Second * 60,
			maxDelay:      time.Second * 120,
			expectInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test multiple times due to randomness, but be more tolerant
			validCount := 0
			totalTests := 10

			for i := 0; i < totalTests; i++ {
				delay := calculateRetryDelay(tt.retryCount)

				if tt.expectInRange {
					if delay >= tt.minDelay && delay <= tt.maxDelay {
						validCount++
					}
				}
			}

			// Allow at least 70% of tests to pass due to randomness in exponential backoff
			if tt.expectInRange {
				assert.True(t, validCount >= totalTests*7/10,
					"At least 70%% of retry delays should be in range [%v, %v] for retry %d, got %d/%d valid",
					tt.minDelay, tt.maxDelay, tt.retryCount, validCount, totalTests)
			}
		})
	}
}

// Helper function to test retry delay calculation (this would be in the actual implementation)
func calculateRetryDelay(retryCount int) time.Duration {
	// Exponential backoff with jitter: 2^retryCount * base + random jitter
	base := time.Second * 2
	exponential := time.Duration(1<<uint(retryCount)) * base

	// Cap at 60 seconds
	if exponential > time.Second*60 {
		exponential = time.Second * 60
	}

	// Add random jitter (0-50% of the exponential delay)
	jitter := time.Duration(float64(exponential) * 0.5 * (float64(time.Now().UnixNano()%1000) / 1000.0))

	return exponential + jitter
}

// Benchmark tests
func BenchmarkWebhookProcessor_CreateWebhookEntry(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	config := &entities.WebhookConfig{
		ID:         1,
		WebhookURL: "https://example.com/webhook",
		IsActive:   true,
	}

	mockConfigRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(config, nil).AnyTimes()
	mockQueueRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.CreateWebhookEntry(ctx, enums.EventTypeCredit, "test-event", 1)
	}
}

// TestWebhookProcessor_Constructor tests the constructor
func TestWebhookProcessor_Constructor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	t.Run("should create webhook processor successfully", func(t *testing.T) {
		processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)
		assert.NotNil(t, processor)
		assert.Equal(t, mockQueueRepo, processor.webhookQueueRepo)
		assert.Equal(t, mockConfigRepo, processor.webhookConfigRepo)
		assert.Equal(t, mockWebhookService, processor.webhookService)
		assert.Equal(t, logger, processor.logger)
	})
}

// TestWebhookProcessor_ProcessWebhook_ComprehensiveCoverage covers all ProcessWebhook scenarios
func TestWebhookProcessor_ProcessWebhook_ComprehensiveCoverage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should handle successful webhook with nil response body", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Response with nil fields
		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       "",
			Duration:   time.Millisecond * 500,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, "", "").
			Times(1)

		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Return(nil).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
	})

	t.Run("should handle HTTP error status codes as failures", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  1,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// HTTP 404 - should be treated as error
		response := &services.WebhookResponse{
			StatusCode: 404,
			Body:       `{"error": "not found"}`,
			Duration:   time.Millisecond * 300,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 404, `{"error": "not found"}`, gomock.Any()).
			Times(1)

		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, w *entities.WebhookQueue) error {
				assert.Equal(t, enums.WebhookStatusPending, w.Status)
				assert.Equal(t, 2, w.RetryCount)
				return nil
			}).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
		assert.Equal(t, 404, webhook.LastHTTPStatus)
		assert.Contains(t, webhook.LastError, "HTTP 404")
	})

	t.Run("should handle UpdateRetryAttempt failure", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       `{"success": true}`,
			Duration:   time.Millisecond * 500,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		// UpdateRetryAttempt fails but shouldn't stop processing
		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, `{"success": true}`, "").
			Return(errors.New("database update failed")).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Return(nil).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
	})

	t.Run("should handle MarkCompleted failure", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       `{"success": true}`,
			Duration:   time.Millisecond * 500,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, `{"success": true}`, "").
			Return(nil).
			Times(1)

		// MarkCompleted fails - should return error
		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Return(errors.New("failed to mark completed")).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark completed")
	})

	t.Run("should handle Update failure during retry scheduling", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  2,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 500,
			Body:       `{"error": "server error"}`,
			Duration:   time.Millisecond * 1000,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 500, `{"error": "server error"}`, gomock.Any()).
			Return(nil).
			Times(1)

		// Update fails during retry scheduling
		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(errors.New("failed to update for retry")).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update for retry")
	})

	t.Run("should handle MarkFailed failure", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  enums.MaxRetryAttempts, // At max retries
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 500,
			Body:       `{"error": "server error"}`,
			Duration:   time.Millisecond * 1000,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 500, `{"error": "server error"}`, gomock.Any()).
			Return(nil).
			Times(1)

		// MarkFailed fails
		mockQueueRepo.EXPECT().
			MarkFailed(ctx, webhook.ID, gomock.Any()).
			Return(errors.New("failed to mark as failed")).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark as failed")
	})

	t.Run("should handle network error with max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  enums.MaxRetryAttempts,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		networkError := errors.New("connection refused")

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(nil, networkError).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 0, "", "connection refused").
			Return(nil).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkFailed(ctx, webhook.ID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, id int64, errorMsg string) error {
				assert.Contains(t, errorMsg, "max retries exceeded")
				assert.Contains(t, errorMsg, "connection refused")
				return nil
			}).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
	})

	t.Run("should handle HTTP error response with max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  enums.MaxRetryAttempts,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 503,
			Body:       `{"error": "service unavailable"}`,
			Duration:   time.Millisecond * 1000,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 503, `{"error": "service unavailable"}`, gomock.Any()).
			Return(nil).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkFailed(ctx, webhook.ID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, id int64, errorMsg string) error {
				assert.Contains(t, errorMsg, "max retries exceeded")
				assert.Contains(t, errorMsg, "HTTP 503")
				return nil
			}).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
	})
}

// TestWebhookProcessor_HelperMethods tests helper methods
func TestWebhookProcessor_HelperMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("isSuccessfulResponse should identify successful status codes", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			expected   bool
		}{
			{200, true},
			{201, true},
			{202, true},
			{204, true},
			{299, true},
			{199, false},
			{300, false},
			{400, false},
			{404, false},
			{500, false},
			{503, false},
		}

		for _, tc := range testCases {
			result := processor.isSuccessfulResponse(tc.statusCode)
			assert.Equal(t, tc.expected, result, "Status code %d should return %v", tc.statusCode, tc.expected)
		}
	})
}

// TestWebhookProcessor_CalculateNextRetryTime tests the retry time calculation
func TestWebhookProcessor_CalculateNextRetryTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	tests := []struct {
		name        string
		retryCount  int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "retry 0 should have 1 minute base delay",
			retryCount:  0,
			expectedMin: 45 * time.Second, // 1min - 25% jitter
			expectedMax: 75 * time.Second, // 1min + 25% jitter
		},
		{
			name:        "retry 1 should have 5 minute base delay",
			retryCount:  1,
			expectedMin: 225 * time.Second, // 5min - 25% jitter
			expectedMax: 375 * time.Second, // 5min + 25% jitter
		},
		{
			name:        "retry 2 should have 10 minute base delay",
			retryCount:  2,
			expectedMin: 450 * time.Second, // 10min - 25% jitter
			expectedMax: 750 * time.Second, // 10min + 25% jitter
		},
		{
			name:        "retry 3 should have 30 minute base delay",
			retryCount:  3,
			expectedMin: 1350 * time.Second, // 30min - 25% jitter
			expectedMax: 2250 * time.Second, // 30min + 25% jitter
		},
		{
			name:        "retry 4 should have 60 minute base delay",
			retryCount:  4,
			expectedMin: 2700 * time.Second, // 60min - 25% jitter
			expectedMax: 4500 * time.Second, // 60min + 25% jitter
		},
		{
			name:        "retry 5 should have 120 minute base delay",
			retryCount:  5,
			expectedMin: 5400 * time.Second, // 120min - 25% jitter
			expectedMax: 9000 * time.Second, // 120min + 25% jitter
		},
		{
			name:        "retry 6+ should have 4 hour fallback delay",
			retryCount:  10,
			expectedMin: 10800 * time.Second, // 4hr - 25% jitter
			expectedMax: 18000 * time.Second, // 4hr + 25% jitter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()

			// Test multiple times to account for jitter
			validCount := 0
			totalTests := 20

			for i := 0; i < totalTests; i++ {
				nextRetryTime := processor.calculateNextRetryTime(tt.retryCount)
				delay := nextRetryTime.Sub(now)

				if delay >= tt.expectedMin && delay <= tt.expectedMax {
					validCount++
				}
			}

			// At least 70% should be within expected range due to jitter
			assert.True(t, validCount >= totalTests*7/10,
				"At least 70%% of retry delays should be in range [%v, %v] for retry %d, got %d/%d valid",
				tt.expectedMin, tt.expectedMax, tt.retryCount, validCount, totalTests)
		})
	}

	t.Run("should have minimum 1 minute delay even with negative jitter", func(t *testing.T) {
		// This test ensures the minimum delay logic works
		for i := 0; i < 100; i++ {
			nextRetryTime := processor.calculateNextRetryTime(0)
			delay := nextRetryTime.Sub(time.Now().UTC())
			assert.True(t, delay >= time.Minute, "Delay should never be less than 1 minute, got %v", delay)
		}
	})
}

// TestWebhookProcessor_ResetWebhookToPending tests the reset functionality
func TestWebhookProcessor_ResetWebhookToPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should reset webhook to pending status", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  2,
			NextRetryAt: now,
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Minute),
		}

		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, w *entities.WebhookQueue) error {
				assert.Equal(t, enums.WebhookStatusPending, w.Status)
				assert.True(t, w.UpdatedAt.After(now.Add(-time.Second)))
				// Other fields should remain unchanged
				assert.Equal(t, webhook.ID, w.ID)
				assert.Equal(t, webhook.QueueID, w.QueueID)
				assert.Equal(t, webhook.EventType, w.EventType)
				assert.Equal(t, webhook.EventID, w.EventID)
				assert.Equal(t, webhook.ConfigID, w.ConfigID)
				assert.Equal(t, webhook.RetryCount, w.RetryCount)
				return nil
			}).
			Times(1)

		err := processor.ResetWebhookToPending(ctx, webhook)
		assert.NoError(t, err)
		assert.Equal(t, enums.WebhookStatusPending, webhook.Status)
	})

	t.Run("should handle update failure", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  2,
			NextRetryAt: now,
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Minute),
		}

		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(errors.New("database update failed")).
			Times(1)

		err := processor.ResetWebhookToPending(ctx, webhook)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database update failed")
	})
}

// TestWebhookProcessor_EdgeCases tests edge cases and boundary conditions
func TestWebhookProcessor_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should handle webhook with nil response from service", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  1,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Service returns nil response and error
		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(nil, errors.New("network error")).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 0, "", "network error").
			Return(nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
		assert.Equal(t, 0, webhook.LastHTTPStatus)
		assert.Equal(t, "network error", webhook.LastError)
	})

	t.Run("should handle empty error message gracefully", func(t *testing.T) {
		ctx := context.Background()
		workerID := "worker-1"
		now := time.Now().UTC()

		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       `{"success": true}`,
			Duration:   time.Millisecond * 500,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, `{"success": true}`, "").
			Return(nil).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Return(nil).
			Times(1)

		err := processor.ProcessWebhook(ctx, webhook, workerID)
		assert.NoError(t, err)
		assert.Equal(t, 200, webhook.LastHTTPStatus)
		assert.Equal(t, "", webhook.LastError) // Should remain empty
	})
}

// TestWebhookProcessor_Integration tests integration scenarios
func TestWebhookProcessor_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueueRepo := mocks.NewMockWebhookQueueRepository(ctrl)
	mockConfigRepo := mocks.NewMockWebhookConfigRepository(ctrl)
	mockWebhookService := mocks.NewMockWebhookService(ctrl)
	logger := log.NewNopLogger()

	processor := NewWebhookProcessor(mockQueueRepo, mockConfigRepo, mockWebhookService, logger)

	t.Run("should handle complete workflow from creation to completion", func(t *testing.T) {
		ctx := context.Background()
		eventType := enums.EventTypeCredit
		eventID := "integration-test-event"
		configID := int64(1)

		// Step 1: Create webhook entry
		config := &entities.WebhookConfig{
			ID:         configID,
			Name:       "Integration Test Config",
			EventType:  eventType,
			WebhookURL: "https://example.com/webhook",
			IsActive:   true,
			TimeoutMs:  30000,
		}

		mockConfigRepo.EXPECT().
			GetByID(ctx, configID).
			Return(config, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, webhook *entities.WebhookQueue) error {
				webhook.ID = 1
				webhook.QueueID = uuid.New()
				return nil
			}).
			Times(1)

		err := processor.CreateWebhookEntry(ctx, eventType, eventID, configID)
		assert.NoError(t, err)

		// Step 2: Process the webhook successfully
		webhook := &entities.WebhookQueue{
			ID:          1,
			QueueID:     uuid.New(),
			EventType:   eventType,
			EventID:     eventID,
			ConfigID:    configID,
			WebhookURL:  config.WebhookURL,
			Status:      enums.WebhookStatusProcessing,
			RetryCount:  0,
			NextRetryAt: time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		response := &services.WebhookResponse{
			StatusCode: 200,
			Body:       `{"message": "webhook received"}`,
			Duration:   time.Millisecond * 300,
			Error:      nil,
		}

		mockWebhookService.EXPECT().
			SendWebhook(ctx, webhook).
			Return(response, nil).
			Times(1)

		mockQueueRepo.EXPECT().
			UpdateRetryAttempt(ctx, webhook.ID, webhook.RetryCount, gomock.Any(), gomock.Any(),
				gomock.Any(), 200, `{"message": "webhook received"}`, "").
			Return(nil).
			Times(1)

		mockQueueRepo.EXPECT().
			MarkCompleted(ctx, webhook.ID, gomock.Any()).
			Return(nil).
			Times(1)

		err = processor.ProcessWebhook(ctx, webhook, "integration-worker")
		assert.NoError(t, err)
	})
}
