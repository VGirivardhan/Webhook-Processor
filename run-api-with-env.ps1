# Run Webhook API with environment file
# This script loads .env file and runs the API

Write-Host "Starting Webhook API with environment file..." -ForegroundColor Green

# Load environment variables from .env file
if (Test-Path ".env") {
    Write-Host "Loading environment variables from .env file..." -ForegroundColor Yellow
    . .\load-env.ps1
} else {
    Write-Host "⚠️ No .env file found. Creating from template..." -ForegroundColor Yellow
    Copy-Item environment.env .env
    Write-Host "✅ Created .env file from template. Please review and modify if needed." -ForegroundColor Green
    . .\load-env.ps1
}

Write-Host "`nEnvironment variables loaded:" -ForegroundColor Yellow
Write-Host "DB_HOST: $env:DB_HOST" -ForegroundColor Cyan
Write-Host "DB_PORT: $env:DB_PORT" -ForegroundColor Cyan
Write-Host "DB_NAME: $env:DB_NAME" -ForegroundColor Cyan
Write-Host "API_PORT: $env:API_PORT" -ForegroundColor Cyan
Write-Host "HTTP_CLIENT_TIMEOUT: $env:HTTP_CLIENT_TIMEOUT" -ForegroundColor Cyan
Write-Host "HTTP_SERVER_READ_TIMEOUT: $env:HTTP_SERVER_READ_TIMEOUT" -ForegroundColor Cyan

Write-Host "`nStarting enhanced API server..." -ForegroundColor Green
go run ./cmd/webhook-api/main.go
