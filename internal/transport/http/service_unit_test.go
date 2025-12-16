package http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"webhook-processor/internal/application/services"
	"webhook-processor/internal/domain/enums"
)

// Simple mock for unit testing
type unitTestMockWebhookApplicationService struct {
	createWebhookResult *services.CreateWebhookResult
	createWebhookError  error
	healthResult        *services.HealthResult
	healthError         error
}

func (m *unitTestMockWebhookApplicationService) CreateWebhook(ctx context.Context, cmd services.CreateWebhookCommand) (*services.CreateWebhookResult, error) {
	if m.createWebhookResult != nil {
		return m.createWebhookResult, m.createWebhookError
	}
	return &services.CreateWebhookResult{
		Success:   true,
		Message:   "Webhook created successfully",
		QueueID:   "test-queue",
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (m *unitTestMockWebhookApplicationService) GetHealth(ctx context.Context) (*services.HealthResult, error) {
	if m.healthError != nil {
		return m.healthResult, m.healthError
	}
	if m.healthResult != nil {
		return m.healthResult, nil
	}
	return &services.HealthResult{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]string{
			"database": "connected",
		},
		Uptime: time.Hour,
	}, nil
}

func TestHTTPService_CreateWebhook_Unit(t *testing.T) {
	t.Run("should create webhook successfully", func(t *testing.T) {
		// Arrange
		mockAppService := &unitTestMockWebhookApplicationService{
			createWebhookResult: &services.CreateWebhookResult{
				Success:   true,
				Message:   "Webhook created successfully",
				QueueID:   "queue-123",
				CreatedAt: time.Now().UTC(),
			},
		}

		httpService := NewService(mockAppService)
		ctx := context.Background()

		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-123",
			ConfigID:  1,
		}

		// Act
		response, err := httpService.CreateWebhook(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Webhook created successfully", response.Message)
		assert.Equal(t, "queue-123", response.QueueID)
		assert.NotEmpty(t, response.CreatedAt)
	})

	t.Run("should handle application service error", func(t *testing.T) {
		// Arrange
		mockAppService := &unitTestMockWebhookApplicationService{
			createWebhookResult: &services.CreateWebhookResult{
				Success: false,
				Message: "Failed to create webhook: config not found",
			},
			createWebhookError: errors.New("config not found"),
		}

		httpService := NewService(mockAppService)
		ctx := context.Background()

		req := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "test-event-456",
			ConfigID:  999,
		}

		// Act
		response, err := httpService.CreateWebhook(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.False(t, response.Success)
		assert.Contains(t, response.Message, "Failed to create webhook")
		assert.Empty(t, response.QueueID)
	})

	t.Run("should handle different event types", func(t *testing.T) {
		eventTypes := []enums.EventType{
			enums.EventTypeCredit,
			enums.EventTypeDebit,
		}

		for _, eventType := range eventTypes {
			mockAppService := &unitTestMockWebhookApplicationService{
				createWebhookResult: &services.CreateWebhookResult{
					Success:   true,
					Message:   "Webhook created successfully",
					QueueID:   "queue-" + string(eventType),
					CreatedAt: time.Now().UTC(),
				},
			}

			httpService := NewService(mockAppService)
			ctx := context.Background()

			req := CreateWebhookRequest{
				EventType: eventType,
				EventID:   "test-event",
				ConfigID:  1,
			}

			// Act
			response, err := httpService.CreateWebhook(ctx, req)

			// Assert
			assert.NoError(t, err)
			assert.True(t, response.Success)
			assert.Equal(t, "queue-"+string(eventType), response.QueueID)
		}
	})
}

func TestHTTPService_GetHealth_Unit(t *testing.T) {
	t.Run("should get health status successfully", func(t *testing.T) {
		// Arrange
		now := time.Now().UTC()
		uptime := time.Hour * 2

		mockAppService := &unitTestMockWebhookApplicationService{
			healthResult: &services.HealthResult{
				Status:    "healthy",
				Version:   "1.0.0",
				Timestamp: now,
				Dependencies: map[string]string{
					"database": "connected",
					"workers":  "running",
				},
				Uptime: uptime,
			},
		}

		httpService := NewService(mockAppService)
		ctx := context.Background()

		// Act
		response, err := httpService.GetHealth(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "1.0.0", response.Version)
		assert.Equal(t, now.Format(time.RFC3339), response.Timestamp)
		assert.Equal(t, "connected", response.Dependencies["database"])
		assert.Equal(t, "running", response.Dependencies["workers"])
		assert.Equal(t, uptime.String(), response.Uptime)
	})

	t.Run("should handle application service error", func(t *testing.T) {
		// Arrange
		mockAppService := &unitTestMockWebhookApplicationService{
			healthResult: nil,
			healthError:  errors.New("database connection failed"),
		}

		httpService := NewService(mockAppService)
		ctx := context.Background()

		// Act
		response, err := httpService.GetHealth(ctx)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, HealthResponse{}, response)
	})
}

func TestNewService_Unit(t *testing.T) {
	t.Run("should create service successfully", func(t *testing.T) {
		mockAppService := &unitTestMockWebhookApplicationService{}

		// Act
		service := NewService(mockAppService)

		// Assert
		assert.NotNil(t, service)

		// Verify it implements the Service interface
		var _ Service = service
	})
}
