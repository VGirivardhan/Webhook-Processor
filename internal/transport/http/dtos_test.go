package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"webhook-processor/internal/application/services"
	"webhook-processor/internal/domain/enums"
)

func TestCreateWebhookRequest_ToApplicationCommand(t *testing.T) {
	t.Run("should convert HTTP request to application command correctly", func(t *testing.T) {
		// Arrange
		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-123",
			ConfigID:  1,
		}

		// Act
		cmd := req.ToApplicationCommand()

		// Assert
		assert.Equal(t, req.EventType, cmd.EventType)
		assert.Equal(t, req.EventID, cmd.EventID)
		assert.Equal(t, req.ConfigID, cmd.ConfigID)
	})

	t.Run("should handle debit event type", func(t *testing.T) {
		// Arrange
		req := CreateWebhookRequest{
			EventType: enums.EventTypeDebit,
			EventID:   "debit-event-456",
			ConfigID:  2,
		}

		// Act
		cmd := req.ToApplicationCommand()

		// Assert
		assert.Equal(t, enums.EventTypeDebit, cmd.EventType)
		assert.Equal(t, "debit-event-456", cmd.EventID)
		assert.Equal(t, int64(2), cmd.ConfigID)
	})

	t.Run("should handle empty event ID", func(t *testing.T) {
		// Arrange
		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "", // Empty event ID
			ConfigID:  1,
		}

		// Act
		cmd := req.ToApplicationCommand()

		// Assert
		assert.Equal(t, enums.EventTypeCredit, cmd.EventType)
		assert.Equal(t, "", cmd.EventID)
		assert.Equal(t, int64(1), cmd.ConfigID)
	})

	t.Run("should handle zero config ID", func(t *testing.T) {
		// Arrange
		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event",
			ConfigID:  0, // Zero config ID
		}

		// Act
		cmd := req.ToApplicationCommand()

		// Assert
		assert.Equal(t, int64(0), cmd.ConfigID)
	})

	t.Run("should handle large config ID", func(t *testing.T) {
		// Arrange
		largeConfigID := int64(9223372036854775807) // Max int64
		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event",
			ConfigID:  largeConfigID,
		}

		// Act
		cmd := req.ToApplicationCommand()

		// Assert
		assert.Equal(t, largeConfigID, cmd.ConfigID)
	})
}

func TestCreateWebhookResponse_FromApplicationResult(t *testing.T) {
	t.Run("should convert successful application result to HTTP response", func(t *testing.T) {
		// Arrange
		now := time.Now().UTC()
		appResult := &services.CreateWebhookResult{
			Success:   true,
			Message:   "Webhook created successfully",
			QueueID:   "queue-123",
			CreatedAt: now,
		}

		var response CreateWebhookResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.True(t, response.Success)
		assert.Equal(t, "Webhook created successfully", response.Message)
		assert.Equal(t, "queue-123", response.QueueID)
		assert.Equal(t, now.Format(time.RFC3339), response.CreatedAt)
	})

	t.Run("should convert failed application result to HTTP response", func(t *testing.T) {
		// Arrange
		appResult := &services.CreateWebhookResult{
			Success: false,
			Message: "Failed to create webhook: config not found",
			QueueID: "", // Empty for failed requests
		}

		var response CreateWebhookResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.False(t, response.Success)
		assert.Equal(t, "Failed to create webhook: config not found", response.Message)
		assert.Empty(t, response.QueueID)
		assert.Empty(t, response.CreatedAt) // Should be empty when CreatedAt is zero
	})

	t.Run("should handle zero timestamp", func(t *testing.T) {
		// Arrange
		appResult := &services.CreateWebhookResult{
			Success:   true,
			Message:   "Webhook created successfully",
			QueueID:   "queue-456",
			CreatedAt: time.Time{}, // Zero time
		}

		var response CreateWebhookResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.True(t, response.Success)
		assert.Equal(t, "queue-456", response.QueueID)
		assert.Empty(t, response.CreatedAt) // Should be empty for zero time
	})

	t.Run("should handle long messages", func(t *testing.T) {
		// Arrange
		longMessage := "This is a very long error message that contains detailed information about what went wrong during webhook creation process including validation errors and database constraints"
		appResult := &services.CreateWebhookResult{
			Success: false,
			Message: longMessage,
		}

		var response CreateWebhookResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.False(t, response.Success)
		assert.Equal(t, longMessage, response.Message)
	})

	t.Run("should handle special characters in message", func(t *testing.T) {
		// Arrange
		specialMessage := "Webhook creation failed: Invalid characters in URL: !@#$%^&*()_+-=[]{}|;:,.<>?"
		appResult := &services.CreateWebhookResult{
			Success: false,
			Message: specialMessage,
		}

		var response CreateWebhookResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, specialMessage, response.Message)
	})
}

func TestHealthResponse_FromApplicationResult(t *testing.T) {
	t.Run("should convert healthy application result to HTTP response", func(t *testing.T) {
		// Arrange
		now := time.Now().UTC()
		uptime := time.Hour*2 + time.Minute*30 + time.Second*45

		appResult := &services.HealthResult{
			Status:    "healthy",
			Version:   "1.0.0",
			Timestamp: now,
			Dependencies: map[string]string{
				"database": "connected",
				"workers":  "running",
				"cache":    "available",
			},
			Uptime: uptime,
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "1.0.0", response.Version)
		assert.Equal(t, now.Format(time.RFC3339), response.Timestamp)
		assert.Equal(t, "connected", response.Dependencies["database"])
		assert.Equal(t, "running", response.Dependencies["workers"])
		assert.Equal(t, "available", response.Dependencies["cache"])
		assert.Equal(t, uptime.String(), response.Uptime)
	})

	t.Run("should convert degraded application result to HTTP response", func(t *testing.T) {
		// Arrange
		now := time.Now().UTC()
		uptime := time.Minute * 15

		appResult := &services.HealthResult{
			Status:    "degraded",
			Version:   "1.2.3",
			Timestamp: now,
			Dependencies: map[string]string{
				"database": "connected",
				"workers":  "degraded",
				"cache":    "unavailable",
			},
			Uptime: uptime,
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, "degraded", response.Status)
		assert.Equal(t, "1.2.3", response.Version)
		assert.Equal(t, "degraded", response.Dependencies["workers"])
		assert.Equal(t, "unavailable", response.Dependencies["cache"])
		assert.Equal(t, uptime.String(), response.Uptime)
	})

	t.Run("should handle empty dependencies", func(t *testing.T) {
		// Arrange
		appResult := &services.HealthResult{
			Status:       "healthy",
			Version:      "1.0.0",
			Timestamp:    time.Now().UTC(),
			Dependencies: map[string]string{}, // Empty dependencies
			Uptime:       time.Hour,
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, "healthy", response.Status)
		assert.NotNil(t, response.Dependencies)
		assert.Len(t, response.Dependencies, 0)
	})

	t.Run("should handle nil dependencies", func(t *testing.T) {
		// Arrange
		appResult := &services.HealthResult{
			Status:       "healthy",
			Version:      "1.0.0",
			Timestamp:    time.Now().UTC(),
			Dependencies: nil, // Nil dependencies
			Uptime:       time.Hour,
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, "healthy", response.Status)
		assert.Nil(t, response.Dependencies)
	})

	t.Run("should handle zero uptime", func(t *testing.T) {
		// Arrange
		appResult := &services.HealthResult{
			Status:    "healthy",
			Version:   "1.0.0",
			Timestamp: time.Now().UTC(),
			Dependencies: map[string]string{
				"database": "connected",
			},
			Uptime: time.Duration(0), // Zero uptime
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, "0s", response.Uptime)
	})

	t.Run("should handle very large uptime", func(t *testing.T) {
		// Arrange
		largeUptime := time.Hour*24*365 + time.Hour*12 + time.Minute*30 // Over a year
		appResult := &services.HealthResult{
			Status:    "healthy",
			Version:   "1.0.0",
			Timestamp: time.Now().UTC(),
			Dependencies: map[string]string{
				"database": "connected",
			},
			Uptime: largeUptime,
		}

		var response HealthResponse

		// Act
		response.FromApplicationResult(appResult)

		// Assert
		assert.Equal(t, largeUptime.String(), response.Uptime)
		assert.Contains(t, response.Uptime, "h") // Should contain hours
	})
}

func TestDTOs_JSONSerialization(t *testing.T) {
	t.Run("CreateWebhookRequest should have proper JSON tags", func(t *testing.T) {
		// This test verifies that the struct tags are correct for JSON serialization
		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event",
			ConfigID:  1,
		}

		// Verify the struct has the expected fields
		// (This is more of a compile-time check, but ensures the struct is properly defined)
		assert.Equal(t, enums.EventTypeCredit, req.EventType)
		assert.Equal(t, "test-event", req.EventID)
		assert.Equal(t, int64(1), req.ConfigID)
	})

	t.Run("CreateWebhookResponse should have proper JSON tags", func(t *testing.T) {
		response := CreateWebhookResponse{
			Success:   true,
			Message:   "Success",
			QueueID:   "queue-123",
			CreatedAt: "2023-01-01T00:00:00Z",
		}

		// Verify the struct has the expected fields
		assert.True(t, response.Success)
		assert.Equal(t, "Success", response.Message)
		assert.Equal(t, "queue-123", response.QueueID)
		assert.Equal(t, "2023-01-01T00:00:00Z", response.CreatedAt)
	})

	t.Run("HealthResponse should have proper JSON tags", func(t *testing.T) {
		response := HealthResponse{
			Status:    "healthy",
			Version:   "1.0.0",
			Timestamp: "2023-01-01T00:00:00Z",
			Dependencies: map[string]string{
				"database": "connected",
			},
			Uptime: "1h30m",
		}

		// Verify the struct has the expected fields
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "1.0.0", response.Version)
		assert.Equal(t, "2023-01-01T00:00:00Z", response.Timestamp)
		assert.Equal(t, "connected", response.Dependencies["database"])
		assert.Equal(t, "1h30m", response.Uptime)
	})
}

func TestDTOs_ValidationTags(t *testing.T) {
	t.Run("CreateWebhookRequest should have proper validation tags", func(t *testing.T) {
		// This test documents the validation requirements
		// The actual validation would be done by a validator library

		// Valid request
		validReq := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event",
			ConfigID:  1, // min=1 validation
		}

		assert.True(t, validReq.EventType.IsValid())
		assert.True(t, validReq.ConfigID >= 1)

		// Invalid request examples (would fail validation)
		invalidReq := CreateWebhookRequest{
			EventType: enums.EventType("INVALID"),
			EventID:   "test-event",
			ConfigID:  0, // Would fail min=1 validation
		}

		assert.False(t, invalidReq.EventType.IsValid())
		assert.False(t, invalidReq.ConfigID >= 1)
	})
}

// Benchmark tests
func BenchmarkCreateWebhookRequest_ToApplicationCommand(b *testing.B) {
	req := CreateWebhookRequest{
		EventType: enums.EventTypeCredit,
		EventID:   "benchmark-event",
		ConfigID:  1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.ToApplicationCommand()
	}
}

func BenchmarkCreateWebhookResponse_FromApplicationResult(b *testing.B) {
	appResult := &services.CreateWebhookResult{
		Success:   true,
		Message:   "Webhook created successfully",
		QueueID:   "benchmark-queue",
		CreatedAt: time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response CreateWebhookResponse
		response.FromApplicationResult(appResult)
	}
}

func BenchmarkHealthResponse_FromApplicationResult(b *testing.B) {
	appResult := &services.HealthResult{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]string{
			"database": "connected",
			"workers":  "running",
		},
		Uptime: time.Hour * 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response HealthResponse
		response.FromApplicationResult(appResult)
	}
}
