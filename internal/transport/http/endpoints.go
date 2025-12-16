package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
)

// Endpoints holds all the service endpoints
type Endpoints struct {
	CreateWebhookEndpoint endpoint.Endpoint
	GetHealthEndpoint     endpoint.Endpoint
}

// MakeEndpoints creates all service endpoints (middleware applied at HTTP level)
func MakeEndpoints(svc Service, logger log.Logger) Endpoints {
	return Endpoints{
		CreateWebhookEndpoint: makeCreateWebhookEndpoint(svc),
		GetHealthEndpoint:     makeGetHealthEndpoint(svc),
	}
}

// makeCreateWebhookEndpoint creates the create webhook endpoint
func makeCreateWebhookEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateWebhookRequest)
		response, err := svc.CreateWebhook(ctx, req)
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

// makeGetHealthEndpoint creates the health check endpoint
func makeGetHealthEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		response, err := svc.GetHealth(ctx)
		if err != nil {
			return response, err
		}
		return response, nil
	}
}
