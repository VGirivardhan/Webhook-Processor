package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"webhook-processor/internal/config"
	"webhook-processor/internal/domain/entities"
	"webhook-processor/internal/domain/services"
)

// webhookServiceImpl implements the WebhookService interface
type webhookServiceImpl struct {
	httpClient *http.Client
}

// NewWebhookService creates a new webhook service
func NewWebhookService(clientConfig config.HTTPClientConfig) services.WebhookService {
	return &webhookServiceImpl{
		httpClient: &http.Client{
			Timeout: clientConfig.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:    clientConfig.MaxIdleConns,
				IdleConnTimeout: clientConfig.IdleConnTimeout,
			},
		},
	}
}

// SendWebhook sends a webhook request and returns the response
func (s *webhookServiceImpl) SendWebhook(ctx context.Context, webhook *entities.WebhookQueue) (*services.WebhookResponse, error) {
	startTime := time.Now().UTC()

	// Use the complete webhook URL directly
	fullURL := webhook.WebhookURL

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return &services.WebhookResponse{
			Error:    err,
			Duration: time.Since(startTime),
		}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "Webhook-Processor/1.0")
	req.Header.Set("Accept", "application/json")

	// Send the request
	resp, err := s.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return &services.WebhookResponse{
			Error:    err,
			Duration: duration,
		}, fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &services.WebhookResponse{
			StatusCode: resp.StatusCode,
			Error:      err,
			Duration:   duration,
		}, fmt.Errorf("failed to read response body: %w", err)
	}

	return &services.WebhookResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Duration:   duration,
	}, nil
}
