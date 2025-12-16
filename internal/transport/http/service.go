package http

import (
	"context"

	"webhook-processor/internal/application/services"
)

// Service defines the interface for HTTP transport operations
type Service interface {
	// CreateWebhook handles webhook creation requests
	CreateWebhook(ctx context.Context, req CreateWebhookRequest) (CreateWebhookResponse, error)

	// GetHealth handles health check requests
	GetHealth(ctx context.Context) (HealthResponse, error)
}

// service implements the Service interface
type service struct {
	appService services.WebhookApplicationService
}

// NewService creates a new HTTP transport service
func NewService(appService services.WebhookApplicationService) Service {
	return &service{
		appService: appService,
	}
}

// CreateWebhook handles HTTP webhook creation requests
func (s *service) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (CreateWebhookResponse, error) {
	// Convert HTTP request to application command
	cmd := req.ToApplicationCommand()

	// Call application service
	result, err := s.appService.CreateWebhook(ctx, cmd)
	if err != nil {
		return CreateWebhookResponse{
			Success: false,
			Message: "Failed to create webhook: " + err.Error(),
		}, err
	}

	// Convert application result to HTTP response
	var response CreateWebhookResponse
	response.FromApplicationResult(result)

	return response, nil
}

// GetHealth handles HTTP health check requests
func (s *service) GetHealth(ctx context.Context) (HealthResponse, error) {
	// Call application service
	result, err := s.appService.GetHealth(ctx)
	if err != nil {
		return HealthResponse{}, err
	}

	// Convert application result to HTTP response
	var response HealthResponse
	response.FromApplicationResult(result)

	return response, nil
}
