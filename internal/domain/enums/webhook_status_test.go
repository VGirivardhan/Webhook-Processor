package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookStatus_IsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		status   WebhookStatus
		expected bool
	}{
		{
			name:     "completed status should be completed",
			status:   WebhookStatusCompleted,
			expected: true,
		},
		{
			name:     "pending status should not be completed",
			status:   WebhookStatusPending,
			expected: false,
		},
		{
			name:     "processing status should not be completed",
			status:   WebhookStatusProcessing,
			expected: false,
		},
		{
			name:     "failed status should not be completed",
			status:   WebhookStatusFailed,
			expected: false,
		},
		{
			name:     "empty status should not be completed",
			status:   WebhookStatus(""),
			expected: false,
		},
		{
			name:     "invalid status should not be completed",
			status:   WebhookStatus("INVALID"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsCompleted()
			assert.Equal(t, tt.expected, result, "WebhookStatus.IsCompleted() should return %v for %s", tt.expected, tt.status)
		})
	}
}

func TestWebhookStatus_Constants(t *testing.T) {
	t.Run("webhook status constants should have correct values", func(t *testing.T) {
		assert.Equal(t, WebhookStatus("PENDING"), WebhookStatusPending, "WebhookStatusPending should equal 'PENDING'")
		assert.Equal(t, WebhookStatus("PROCESSING"), WebhookStatusProcessing, "WebhookStatusProcessing should equal 'PROCESSING'")
		assert.Equal(t, WebhookStatus("COMPLETED"), WebhookStatusCompleted, "WebhookStatusCompleted should equal 'COMPLETED'")
		assert.Equal(t, WebhookStatus("FAILED"), WebhookStatusFailed, "WebhookStatusFailed should equal 'FAILED'")
	})

	t.Run("only completed status should be completed", func(t *testing.T) {
		assert.False(t, WebhookStatusPending.IsCompleted(), "WebhookStatusPending should not be completed")
		assert.False(t, WebhookStatusProcessing.IsCompleted(), "WebhookStatusProcessing should not be completed")
		assert.True(t, WebhookStatusCompleted.IsCompleted(), "WebhookStatusCompleted should be completed")
		assert.False(t, WebhookStatusFailed.IsCompleted(), "WebhookStatusFailed should not be completed")
	})
}

func TestMaxRetryAttempts(t *testing.T) {
	t.Run("max retry attempts should be 6", func(t *testing.T) {
		assert.Equal(t, 6, MaxRetryAttempts, "MaxRetryAttempts should be 6 (retry_0 through retry_6)")
	})
}

func TestWebhookStatus_StatusTransitions(t *testing.T) {
	t.Run("should represent valid status lifecycle", func(t *testing.T) {
		// Typical successful flow: PENDING -> PROCESSING -> COMPLETED
		statuses := []WebhookStatus{
			WebhookStatusPending,
			WebhookStatusProcessing,
			WebhookStatusCompleted,
		}

		// Only the last status should be completed
		for i, status := range statuses {
			if i == len(statuses)-1 {
				assert.True(t, status.IsCompleted(), "Final status should be completed")
			} else {
				assert.False(t, status.IsCompleted(), "Intermediate status should not be completed")
			}
		}
	})

	t.Run("should represent valid failure lifecycle", func(t *testing.T) {
		// Typical failure flow: PENDING -> PROCESSING -> FAILED
		statuses := []WebhookStatus{
			WebhookStatusPending,
			WebhookStatusProcessing,
			WebhookStatusFailed,
		}

		// None of these should be completed (failed is not completed)
		for _, status := range statuses {
			assert.False(t, status.IsCompleted(), "Status %s should not be completed", status)
		}
	})
}

func TestWebhookStatus_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		status   WebhookStatus
		expected bool
	}{
		{
			name:     "lowercase completed should not be completed",
			status:   WebhookStatus("completed"),
			expected: false,
		},
		{
			name:     "mixed case completed should not be completed",
			status:   WebhookStatus("Completed"),
			expected: false,
		},
		{
			name:     "completed with whitespace should not be completed",
			status:   WebhookStatus(" COMPLETED "),
			expected: false,
		},
		{
			name:     "partial match should not be completed",
			status:   WebhookStatus("COMPLETE"),
			expected: false,
		},
		{
			name:     "numeric status should not be completed",
			status:   WebhookStatus("200"),
			expected: false,
		},
		{
			name:     "special characters should not be completed",
			status:   WebhookStatus("COMPLETED!"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsCompleted()
			assert.Equal(t, tt.expected, result, "WebhookStatus.IsCompleted() should return %v for %s", tt.expected, tt.status)
		})
	}
}

// Benchmark tests for performance
func BenchmarkWebhookStatus_IsCompleted(b *testing.B) {
	status := WebhookStatusCompleted

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsCompleted()
	}
}

func BenchmarkWebhookStatus_IsCompleted_NotCompleted(b *testing.B) {
	status := WebhookStatusPending

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsCompleted()
	}
}

func BenchmarkWebhookStatus_IsCompleted_Invalid(b *testing.B) {
	status := WebhookStatus("INVALID")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsCompleted()
	}
}
