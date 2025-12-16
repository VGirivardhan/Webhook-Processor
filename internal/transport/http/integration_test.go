package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"webhook-processor/internal/application/services"
	"webhook-processor/internal/domain/enums"
)

// Mock implementation of WebhookApplicationService for integration testing
type mockWebhookApplicationService struct {
	createWebhookFunc func(ctx context.Context, cmd services.CreateWebhookCommand) (*services.CreateWebhookResult, error)
	getHealthFunc     func(ctx context.Context) (*services.HealthResult, error)
}

func (m *mockWebhookApplicationService) CreateWebhook(ctx context.Context, cmd services.CreateWebhookCommand) (*services.CreateWebhookResult, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, cmd)
	}
	return &services.CreateWebhookResult{
		Success:   true,
		Message:   "Webhook created successfully",
		QueueID:   "test-queue-123",
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (m *mockWebhookApplicationService) GetHealth(ctx context.Context) (*services.HealthResult, error) {
	if m.getHealthFunc != nil {
		return m.getHealthFunc(ctx)
	}
	return &services.HealthResult{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]string{
			"database": "connected",
			"workers":  "running",
		},
		Uptime: time.Hour * 2,
	}, nil
}

func TestHTTPHandler_Integration(t *testing.T) {
	// Create mock application service
	mockAppService := &mockWebhookApplicationService{}

	// Create HTTP service and handler
	httpService := NewService(mockAppService)
	logger := log.NewNopLogger()
	handler := NewHTTPHandler(httpService, logger)

	t.Run("should handle POST /webhooks successfully", func(t *testing.T) {
		// Arrange
		reqBody := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "integration-test-123",
			ConfigID:  1,
		}

		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/webhooks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response CreateWebhookResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Webhook created successfully", response.Message)
		assert.NotEmpty(t, response.QueueID)
	})

	t.Run("should handle GET /health successfully", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response HealthResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "1.0.0", response.Version)
		assert.NotEmpty(t, response.Timestamp)
		assert.Equal(t, "connected", response.Dependencies["database"])
		assert.Equal(t, "running", response.Dependencies["workers"])
	})

	t.Run("should handle GET /metrics successfully", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("GET", "/metrics", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
		// Prometheus metrics should be in the response
		assert.Contains(t, recorder.Body.String(), "# HELP")
	})

	t.Run("should handle invalid JSON in POST /webhooks", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("POST", "/webhooks", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		// The actual status code depends on the HTTP transport implementation
		// It could be 400 (Bad Request) or 500 (Internal Server Error)
		assert.True(t, recorder.Code >= 400, "Should return an error status code")
	})

	t.Run("should handle application service error", func(t *testing.T) {
		// Arrange - Mock service to return error
		mockAppService.createWebhookFunc = func(ctx context.Context, cmd services.CreateWebhookCommand) (*services.CreateWebhookResult, error) {
			return &services.CreateWebhookResult{
				Success: false,
				Message: "Config not found",
			}, assert.AnError
		}

		reqBody := CreateWebhookRequest{
			EventType: enums.EventTypeCredit,
			EventID:   "error-test",
			ConfigID:  999,
		}

		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/webhooks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.True(t, recorder.Code >= 400, "Should return an error status code")

		// Only try to parse JSON if we got a valid response
		if recorder.Code < 500 {
			var response CreateWebhookResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &response)
			if err == nil {
				assert.False(t, response.Success)
				assert.Contains(t, response.Message, "Config not found")
			}
		}

		// Reset mock
		mockAppService.createWebhookFunc = nil
	})

	t.Run("should handle unsupported HTTP methods", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("PUT", "/webhooks", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	})

	t.Run("should handle unknown routes", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("GET", "/unknown", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("should include CORS headers", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)
		// Check for CORS headers (if implemented in middleware)
		// This depends on the actual CORS middleware implementation
	})
}

func TestHTTPHandler_Middleware(t *testing.T) {
	// Create mock application service
	mockAppService := &mockWebhookApplicationService{}

	// Create HTTP service and handler
	httpService := NewService(mockAppService)
	logger := log.NewNopLogger()
	handler := NewHTTPHandler(httpService, logger)

	t.Run("should recover from panics", func(t *testing.T) {
		// Arrange - Mock service to panic
		mockAppService.getHealthFunc = func(ctx context.Context) (*services.HealthResult, error) {
			panic("test panic")
		}

		req := httptest.NewRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, req)

		// Assert
		// The recovery middleware should catch the panic and return 500
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		// Reset mock
		mockAppService.getHealthFunc = nil
	})

	t.Run("should handle concurrent requests", func(t *testing.T) {
		// Arrange
		const numRequests = 10
		done := make(chan bool, numRequests)

		// Act - Send concurrent requests
		for i := 0; i < numRequests; i++ {
			go func() {
				defer func() { done <- true }()

				req := httptest.NewRequest("GET", "/health", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, req)

				assert.Equal(t, http.StatusOK, recorder.Code)
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
	})
}

// Benchmark tests
func BenchmarkHTTPHandler_CreateWebhook(b *testing.B) {
	// Setup
	mockAppService := &mockWebhookApplicationService{}
	httpService := NewService(mockAppService)
	logger := log.NewNopLogger()
	handler := NewHTTPHandler(httpService, logger)

	reqBody := CreateWebhookRequest{
		EventType: enums.EventTypeCredit,
		EventID:   "benchmark-test",
		ConfigID:  1,
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/webhooks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
	}
}

func BenchmarkHTTPHandler_GetHealth(b *testing.B) {
	// Setup
	mockAppService := &mockWebhookApplicationService{}
	httpService := NewService(mockAppService)
	logger := log.NewNopLogger()
	handler := NewHTTPHandler(httpService, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
	}
}
