package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"webhook-processor/internal/domain/enums"
)

func TestWebhookQueue_CanRetry(t *testing.T) {
	tests := []struct {
		name        string
		retryCount  int
		status      enums.WebhookStatus
		expected    bool
		description string
	}{
		{
			name:        "can retry when pending and under max retries",
			retryCount:  0,
			status:      enums.WebhookStatusPending,
			expected:    true,
			description: "Should allow retry when status is pending and retry count is 0",
		},
		{
			name:        "can retry when processing and under max retries",
			retryCount:  3,
			status:      enums.WebhookStatusProcessing,
			expected:    true,
			description: "Should allow retry when status is processing and retry count is under max",
		},
		{
			name:        "cannot retry when at max retries",
			retryCount:  enums.MaxRetryAttempts,
			status:      enums.WebhookStatusPending,
			expected:    false,
			description: "Should not allow retry when retry count equals max retry attempts",
		},
		{
			name:        "cannot retry when over max retries",
			retryCount:  enums.MaxRetryAttempts + 1,
			status:      enums.WebhookStatusPending,
			expected:    false,
			description: "Should not allow retry when retry count exceeds max retry attempts",
		},
		{
			name:        "cannot retry when completed",
			retryCount:  2,
			status:      enums.WebhookStatusCompleted,
			expected:    false,
			description: "Should not allow retry when status is completed",
		},
		{
			name:        "cannot retry when failed",
			retryCount:  enums.MaxRetryAttempts,
			status:      enums.WebhookStatusFailed,
			expected:    false,
			description: "Should not allow retry when status is failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &WebhookQueue{
				RetryCount: tt.retryCount,
				Status:     tt.status,
			}

			result := webhook.CanRetry()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestWebhookQueue_Creation(t *testing.T) {
	t.Run("should create webhook queue with all required fields", func(t *testing.T) {
		queueID := uuid.New()
		eventID := "test-event-123"
		webhookURL := "https://example.com/webhook?param=value"
		configID := int64(1)
		now := time.Now().UTC()

		webhook := &WebhookQueue{
			ID:          1,
			QueueID:     queueID,
			EventType:   enums.EventTypeCredit,
			EventID:     eventID,
			ConfigID:    configID,
			WebhookURL:  webhookURL,
			Status:      enums.WebhookStatusPending,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Verify all fields are set correctly
		assert.Equal(t, int64(1), webhook.ID)
		assert.Equal(t, queueID, webhook.QueueID)
		assert.Equal(t, enums.EventTypeCredit, webhook.EventType)
		assert.Equal(t, eventID, webhook.EventID)
		assert.Equal(t, configID, webhook.ConfigID)
		assert.Equal(t, webhookURL, webhook.WebhookURL)
		assert.Equal(t, enums.WebhookStatusPending, webhook.Status)
		assert.Equal(t, 0, webhook.RetryCount)
		assert.Equal(t, now, webhook.NextRetryAt)
		assert.Equal(t, now, webhook.CreatedAt)
		assert.Equal(t, now, webhook.UpdatedAt)
	})
}

func TestWebhookQueue_RetryAttemptFields(t *testing.T) {
	t.Run("should handle retry attempt fields correctly", func(t *testing.T) {
		webhook := &WebhookQueue{}
		now := time.Now().UTC()
		duration := int64(1500) // 1.5 seconds
		httpStatus := 200
		responseBody := `{"success": true}`
		errorMsg := ""

		// Set retry 0 fields
		webhook.Retry0StartedAt = &now
		webhook.Retry0CompletedAt = &now
		webhook.Retry0DurationMs = &duration
		webhook.Retry0HTTPStatus = &httpStatus
		webhook.Retry0ResponseBody = &responseBody
		webhook.Retry0Error = &errorMsg

		// Verify retry 0 fields
		require.NotNil(t, webhook.Retry0StartedAt)
		require.NotNil(t, webhook.Retry0CompletedAt)
		require.NotNil(t, webhook.Retry0DurationMs)
		require.NotNil(t, webhook.Retry0HTTPStatus)
		require.NotNil(t, webhook.Retry0ResponseBody)
		require.NotNil(t, webhook.Retry0Error)

		assert.Equal(t, now, *webhook.Retry0StartedAt)
		assert.Equal(t, now, *webhook.Retry0CompletedAt)
		assert.Equal(t, duration, *webhook.Retry0DurationMs)
		assert.Equal(t, httpStatus, *webhook.Retry0HTTPStatus)
		assert.Equal(t, responseBody, *webhook.Retry0ResponseBody)
		assert.Equal(t, errorMsg, *webhook.Retry0Error)
	})

	t.Run("should handle nil retry attempt fields", func(t *testing.T) {
		webhook := &WebhookQueue{}

		// All retry fields should be nil by default
		assert.Nil(t, webhook.Retry0StartedAt)
		assert.Nil(t, webhook.Retry0CompletedAt)
		assert.Nil(t, webhook.Retry0DurationMs)
		assert.Nil(t, webhook.Retry0HTTPStatus)
		assert.Nil(t, webhook.Retry0ResponseBody)
		assert.Nil(t, webhook.Retry0Error)

		assert.Nil(t, webhook.Retry6StartedAt)
		assert.Nil(t, webhook.Retry6CompletedAt)
		assert.Nil(t, webhook.Retry6DurationMs)
		assert.Nil(t, webhook.Retry6HTTPStatus)
		assert.Nil(t, webhook.Retry6ResponseBody)
		assert.Nil(t, webhook.Retry6Error)
	})
}

func TestWebhookQueue_StatusTracking(t *testing.T) {
	t.Run("should track processing lifecycle", func(t *testing.T) {
		webhook := &WebhookQueue{
			Status:         enums.WebhookStatusPending,
			RetryCount:     0,
			LastError:      "",
			LastHTTPStatus: 0,
		}

		// Initially pending
		assert.Equal(t, enums.WebhookStatusPending, webhook.Status)
		assert.True(t, webhook.CanRetry())

		// Move to processing
		webhook.Status = enums.WebhookStatusProcessing
		webhook.ProcessingStartedAt = &time.Time{}
		now := time.Now().UTC()
		webhook.ProcessingStartedAt = &now

		assert.Equal(t, enums.WebhookStatusProcessing, webhook.Status)
		assert.True(t, webhook.CanRetry())
		require.NotNil(t, webhook.ProcessingStartedAt)

		// Complete successfully
		webhook.Status = enums.WebhookStatusCompleted
		webhook.CompletedAt = &now
		webhook.LastHTTPStatus = 200

		assert.Equal(t, enums.WebhookStatusCompleted, webhook.Status)
		assert.False(t, webhook.CanRetry())
		require.NotNil(t, webhook.CompletedAt)
		assert.Equal(t, 200, webhook.LastHTTPStatus)
	})

	t.Run("should track failure lifecycle", func(t *testing.T) {
		webhook := &WebhookQueue{
			Status:         enums.WebhookStatusPending,
			RetryCount:     enums.MaxRetryAttempts,
			LastError:      "connection timeout",
			LastHTTPStatus: 500,
		}

		// At max retries, should not be able to retry
		assert.False(t, webhook.CanRetry())

		// Mark as failed
		webhook.Status = enums.WebhookStatusFailed
		webhook.CompletedAt = &time.Time{}
		now := time.Now().UTC()
		webhook.CompletedAt = &now

		assert.Equal(t, enums.WebhookStatusFailed, webhook.Status)
		assert.False(t, webhook.CanRetry())
		assert.Equal(t, "connection timeout", webhook.LastError)
		assert.Equal(t, 500, webhook.LastHTTPStatus)
	})
}

func TestWebhookQueue_EventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType enums.EventType
		valid     bool
	}{
		{
			name:      "credit event type",
			eventType: enums.EventTypeCredit,
			valid:     true,
		},
		{
			name:      "debit event type",
			eventType: enums.EventTypeDebit,
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &WebhookQueue{
				EventType: tt.eventType,
			}

			assert.Equal(t, tt.eventType, webhook.EventType)
			if tt.valid {
				assert.True(t, webhook.EventType.IsValid())
				assert.NoError(t, webhook.EventType.Validate())
			}
		})
	}
}

// Benchmark tests for performance-critical operations
func BenchmarkWebhookQueue_CanRetry(b *testing.B) {
	webhook := &WebhookQueue{
		RetryCount: 3,
		Status:     enums.WebhookStatusPending,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = webhook.CanRetry()
	}
}

func BenchmarkWebhookQueue_Creation(b *testing.B) {
	queueID := uuid.New()
	now := time.Now().UTC()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &WebhookQueue{
			ID:          int64(i),
			QueueID:     queueID,
			EventType:   enums.EventTypeCredit,
			EventID:     "test-event",
			ConfigID:    1,
			WebhookURL:  "https://example.com/webhook",
			Status:      enums.WebhookStatusPending,
			RetryCount:  0,
			NextRetryAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
}
