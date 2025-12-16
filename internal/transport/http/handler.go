package http

import (
	"context"
	"encoding/json"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHTTPHandler creates a new HTTP handler with all routes
func NewHTTPHandler(svc Service, logger log.Logger) http.Handler {
	endpoints := MakeEndpoints(svc, logger)

	// Create HTTP handlers using Go-Kit transport
	createWebhookHandler := httptransport.NewServer(
		endpoints.CreateWebhookEndpoint,
		decodeCreateWebhookRequest,
		encodeResponse,
		httptransport.ServerBefore(httptransport.PopulateRequestContext),
	)

	getHealthHandler := httptransport.NewServer(
		endpoints.GetHealthEndpoint,
		decodeGetHealthRequest,
		encodeResponse,
		httptransport.ServerBefore(httptransport.PopulateRequestContext),
	)

	router := mux.NewRouter()

	// Register routes
	router.Handle("/webhooks", createWebhookHandler).Methods("POST")
	router.Handle("/health", getHealthHandler).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Add HTTP middleware
	router.Use(loggingMiddleware(logger))
	router.Use(corsMiddleware)
	router.Use(recoveryMiddleware(logger))

	return router
}

// Request decoders

// decodeCreateWebhookRequest decodes the create webhook request
func decodeCreateWebhookRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

// decodeGetHealthRequest decodes the health check request (no body)
func decodeGetHealthRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

// Response encoder

// encodeResponse encodes the response as JSON
func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}
