# Run Webhook Processor with environment file
# This script loads .env file and runs the processor

Write-Host "Starting Webhook Processor with environment file..." -ForegroundColor Green

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
Write-Host "WORKER_COUNT: $env:WORKER_COUNT" -ForegroundColor Cyan

Write-Host "`nStarting webhook processor..." -ForegroundColor Green
go run ./cmd/webhook-processor/main.go
