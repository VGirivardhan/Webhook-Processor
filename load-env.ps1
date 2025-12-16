# PowerShell script to load environment variables from .env file
# This script reads .env file and sets environment variables for the current session

param(
    [string]$EnvFile = ".env"
)

if (-not (Test-Path $EnvFile)) {
    Write-Host "Environment file '$EnvFile' not found. Creating from template..." -ForegroundColor Yellow
    if (Test-Path "environment.env") {
        Copy-Item "environment.env" $EnvFile
        Write-Host "Created '$EnvFile' from 'environment.env' template." -ForegroundColor Green
        Write-Host "Please edit '$EnvFile' with your specific values." -ForegroundColor Yellow
    } else {
        Write-Host "Template file 'environment.env' not found!" -ForegroundColor Red
        exit 1
    }
}

Write-Host "Loading environment variables from '$EnvFile'..." -ForegroundColor Green

# Read the .env file and set environment variables
Get-Content $EnvFile | ForEach-Object {
    # Skip empty lines and comments
    if ($_ -match '^\s*$' -or $_ -match '^\s*#') {
        return
    }
    
    # Parse KEY=VALUE format
    if ($_ -match '^([^=]+)=(.*)$') {
        $key = $matches[1].Trim()
        $value = $matches[2].Trim()
        
        # Remove quotes if present
        $value = $value -replace '^"(.*)"$', '$1'
        $value = $value -replace "^'(.*)'$", '$1'
        
        # Set environment variable
        [Environment]::SetEnvironmentVariable($key, $value, "Process")
        Write-Host "  $key = $value" -ForegroundColor Cyan
    }
}

Write-Host "Environment variables loaded successfully!" -ForegroundColor Green
