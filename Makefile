# Webhook Processor Makefile

.PHONY: help build test test-coverage clean deps mocks lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the binaries"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  mocks         - Generate mocks for testing"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"

# Build targets
build:
	@echo "Building webhook-api..."
	go build -o bin/webhook-api ./cmd/webhook-api
	@echo "Building webhook-processor..."
	go build -o bin/webhook-processor ./cmd/webhook-processor

# Test targets
test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-domain:
	@echo "Running domain layer tests..."
	go test -v ./internal/domain/...

test-application:
	@echo "Running application layer tests..."
	go test -v ./internal/application/...

test-infrastructure:
	@echo "Running infrastructure layer tests..."
	go test -v ./internal/infrastructure/...

test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/domain/... ./internal/application/... ./internal/infrastructure/...

test-benchmark:
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem ./...

test-race:
	@echo "Running tests with race detection..."
	go test -race ./...

# Mock generation
mocks:
	@echo "Generating mocks..."
	@mkdir -p internal/mocks
	mockgen -source internal/domain/repositories/webhook_config_repository.go -destination internal/mocks/mock_webhook_config_repository.go -package mocks
	mockgen -source internal/domain/repositories/webhook_queue_repository.go -destination internal/mocks/mock_webhook_queue_repository.go -package mocks
	mockgen -source internal/domain/services/webhook_service.go -destination internal/mocks/mock_webhook_service.go -package mocks
	@echo "Mocks generated successfully!"

# Mock generation (Windows compatible - requires mockgen in PATH)
mocks-win:
	@echo "Generating mocks (Windows)..."
	if not exist "internal\\mocks" mkdir "internal\\mocks"
	mockgen -source internal\\domain\\repositories\\webhook_config_repository.go -destination internal\\mocks\\mock_webhook_config_repository.go -package mocks
	mockgen -source internal\\domain\\repositories\\webhook_queue_repository.go -destination internal\\mocks\\mock_webhook_queue_repository.go -package mocks
	mockgen -source internal\\domain\\services\\webhook_service.go -destination internal\\mocks\\mock_webhook_service.go -package mocks
	@echo "Mocks generated successfully!"

# Linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Utility targets
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development targets
dev-setup: deps mocks
	@echo "Development environment setup complete!"

.DEFAULT_GOAL := help
