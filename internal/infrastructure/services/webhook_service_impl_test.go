package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"webhook-processor/internal/config"
	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/enums"
)

func TestWebhookServiceImpl_SendWebhook(t *testing.T) {
	t.Run("should send webhook successfully", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/webhook", r.URL.Path)
			assert.Equal(t, "value", r.URL.Query().Get("param"))
			assert.Equal(t, "Webhook-Processor/1.0", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			// Send successful response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "message": "webhook received"}`))
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook?param=value",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()
		startTime := time.Now()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, `{"success": true, "message": "webhook received"}`, response.Body)
		assert.True(t, response.Duration > 0)
		assert.True(t, response.Duration < time.Second) // Should be fast for local test
		assert.Nil(t, response.Error)

		// Verify timing
		assert.True(t, time.Since(startTime) >= response.Duration)
	})

	t.Run("should handle HTTP error responses", func(t *testing.T) {
		// Create test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.NoError(t, err) // HTTP errors are not Go errors
		require.NotNil(t, response)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, `{"error": "internal server error"}`, response.Body)
		assert.True(t, response.Duration > 0)
		assert.Nil(t, response.Error)
	})

	t.Run("should handle network errors", func(t *testing.T) {
		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Millisecond * 1, // Very short timeout
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook with invalid URL that will timeout
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: "http://192.0.2.1:12345/webhook", // Non-routable IP for timeout
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, response)
		assert.Equal(t, 0, response.StatusCode) // No status code for network errors
		assert.Empty(t, response.Body)
		assert.True(t, response.Duration > 0)
		assert.NotNil(t, response.Error)
	})

	t.Run("should handle invalid URLs", func(t *testing.T) {
		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook with invalid URL
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: "://invalid-url", // Invalid URL
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, response)
		assert.Equal(t, 0, response.StatusCode)
		assert.Empty(t, response.Body)
		assert.True(t, response.Duration >= 0) // Duration can be 0 for immediate failures
		assert.NotNil(t, response.Error)
		assert.Contains(t, err.Error(), "failed to create HTTP request")
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		// Create test server with delay
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Millisecond * 100) // Delay to allow cancellation
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.Error(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Duration > 0)
		assert.NotNil(t, response.Error)
	})

	t.Run("should handle different HTTP methods correctly", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify it's always GET
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"method": "` + r.Method + `"}`))
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, response.Body, `"method": "GET"`)
	})

	t.Run("should set correct headers", func(t *testing.T) {
		// Create test server that checks headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check required headers
			userAgent := r.Header.Get("User-Agent")
			accept := r.Header.Get("Accept")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user_agent": "` + userAgent + `", "accept": "` + accept + `"}`))
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, response.Body, `"user_agent": "Webhook-Processor/1.0"`)
		assert.Contains(t, response.Body, `"accept": "application/json"`)
	})

	t.Run("should handle large response bodies", func(t *testing.T) {
		// Create large response body
		largeBody := make([]byte, 1024*1024) // 1MB
		for i := range largeBody {
			largeBody[i] = 'A'
		}

		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(largeBody)
		}))
		defer server.Close()

		// Create service
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 30,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert
		assert.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, len(largeBody), len(response.Body))
		assert.True(t, response.Duration > 0)
	})

	t.Run("should handle response body read error", func(t *testing.T) {
		// Create test server that closes connection abruptly after headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Write headers first
			w.WriteHeader(http.StatusOK)

			// Force flush headers
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// Close the connection abruptly without sending body
			// This will cause io.ReadAll to fail when trying to read the response body
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					conn.Close() // Abruptly close the connection
				}
			}
		}))
		defer server.Close()

		// Create service with short timeout to avoid hanging
		clientConfig := config.HTTPClientConfig{
			Timeout:         time.Second * 5,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 90,
		}
		service := NewWebhookService(clientConfig)

		// Create webhook
		webhook := &entities.WebhookQueue{
			ID:         1,
			QueueID:    uuid.New(),
			EventType:  enums.EventTypeCredit,
			EventID:    "test-event-123",
			ConfigID:   1,
			WebhookURL: server.URL + "/webhook",
			Status:     enums.WebhookStatusProcessing,
		}

		ctx := context.Background()

		// Execute
		response, err := service.SendWebhook(ctx, webhook)

		// Assert - This should trigger the io.ReadAll error path
		assert.Error(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusOK, response.StatusCode) // Headers were received
		assert.Empty(t, response.Body)                      // Body should be empty due to read error
		assert.True(t, response.Duration > 0)
		assert.NotNil(t, response.Error)
		assert.Contains(t, err.Error(), "failed to read response body")
	})
}

func TestWebhookServiceImpl_URLParsing(t *testing.T) {
	tests := []struct {
		name        string
		webhookURL  string
		expectError bool
		description string
	}{
		{
			name:        "simple URL",
			webhookURL:  "https://example.com/webhook",
			expectError: false,
			description: "Simple HTTPS URL should work",
		},
		{
			name:        "URL with query parameters",
			webhookURL:  "https://example.com/webhook?param1=value1&param2=value2",
			expectError: false,
			description: "URL with query parameters should work",
		},
		{
			name:        "URL with port",
			webhookURL:  "https://example.com:8443/webhook",
			expectError: false,
			description: "URL with custom port should work",
		},
		{
			name:        "HTTP URL",
			webhookURL:  "http://example.com/webhook",
			expectError: false,
			description: "HTTP URL should work",
		},
		{
			name:        "URL with path segments",
			webhookURL:  "https://api.example.com/v1/webhooks/receive",
			expectError: false,
			description: "URL with multiple path segments should work",
		},
		{
			name:        "URL with special characters in query",
			webhookURL:  "https://example.com/webhook?data=%7B%22test%22%3A%22value%22%7D",
			expectError: false,
			description: "URL with encoded special characters should work",
		},
		{
			name:        "invalid URL scheme",
			webhookURL:  "ftp://example.com/webhook",
			expectError: false, // HTTP client will handle this, but request creation should succeed
			description: "Invalid scheme should not fail request creation",
		},
		{
			name:        "malformed URL",
			webhookURL:  "://invalid-url",
			expectError: true,
			description: "Malformed URL should fail",
		},
		{
			name:        "empty URL",
			webhookURL:  "",
			expectError: true,
			description: "Empty URL should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service
			clientConfig := config.HTTPClientConfig{
				Timeout:         time.Second * 1,
				MaxIdleConns:    10,
				IdleConnTimeout: time.Second * 90,
			}
			service := NewWebhookService(clientConfig)

			// Create webhook
			webhook := &entities.WebhookQueue{
				ID:         1,
				QueueID:    uuid.New(),
				EventType:  enums.EventTypeCredit,
				EventID:    "test-event-123",
				ConfigID:   1,
				WebhookURL: tt.webhookURL,
				Status:     enums.WebhookStatusProcessing,
			}

			ctx := context.Background()

			// Execute
			response, err := service.SendWebhook(ctx, webhook)

			// Assert
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				// Note: We might get network errors for valid URLs that don't exist,
				// but we shouldn't get URL parsing errors
				if err != nil {
					// If there's an error, it should be a network error, not a parsing error
					assert.NotContains(t, err.Error(), "failed to create HTTP request", tt.description)
				}
			}

			require.NotNil(t, response, "Response should never be nil")
			assert.True(t, response.Duration >= 0, "Duration should be non-negative")
		})
	}
}

// Benchmark tests
func BenchmarkWebhookServiceImpl_SendWebhook(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create service
	clientConfig := config.HTTPClientConfig{
		Timeout:         time.Second * 30,
		MaxIdleConns:    10,
		IdleConnTimeout: time.Second * 90,
	}
	service := NewWebhookService(clientConfig)

	// Create webhook
	webhook := &entities.WebhookQueue{
		ID:         1,
		QueueID:    uuid.New(),
		EventType:  enums.EventTypeCredit,
		EventID:    "test-event-123",
		ConfigID:   1,
		WebhookURL: server.URL + "/webhook",
		Status:     enums.WebhookStatusProcessing,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SendWebhook(ctx, webhook)
	}
}

func BenchmarkWebhookServiceImpl_SendWebhook_LargeResponse(b *testing.B) {
	// Create large response
	largeBody := make([]byte, 1024*10) // 10KB
	for i := range largeBody {
		largeBody[i] = 'A'
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	}))
	defer server.Close()

	// Create service
	clientConfig := config.HTTPClientConfig{
		Timeout:         time.Second * 30,
		MaxIdleConns:    10,
		IdleConnTimeout: time.Second * 90,
	}
	service := NewWebhookService(clientConfig)

	// Create webhook
	webhook := &entities.WebhookQueue{
		ID:         1,
		QueueID:    uuid.New(),
		EventType:  enums.EventTypeCredit,
		EventID:    "test-event-123",
		ConfigID:   1,
		WebhookURL: server.URL + "/webhook",
		Status:     enums.WebhookStatusProcessing,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SendWebhook(ctx, webhook)
	}
}
