# Simple Webhook Load Test
param([int]$Count = 100)

Write-Host "Starting load test with $Count webhooks..." -ForegroundColor Green

$startTime = Get-Date
$results = @()

for ($i = 1; $i -le $Count; $i++) {
    $requestStart = Get-Date
    
    try {
        $body = @{
            event_type = "CREDIT"
            event_id = "load_test_$i"
            config_id = 1
        } | ConvertTo-Json
        
        $response = Invoke-RestMethod -Uri "http://localhost:8080/webhooks" -Method POST -Headers @{"Content-Type"="application/json"} -Body $body
        
        $duration = (Get-Date) - $requestStart
        $results += @{
            Id = $i
            Success = $true
            Duration = $duration.TotalMilliseconds
        }
        
        if ($i % 10 -eq 0) {
            Write-Host "Created $i webhooks..." -ForegroundColor Yellow
        }
    }
    catch {
        $duration = (Get-Date) - $requestStart
        $results += @{
            Id = $i
            Success = $false
            Duration = $duration.TotalMilliseconds
            Error = $_.Exception.Message
        }
        Write-Host "Failed webhook $i : $($_.Exception.Message)" -ForegroundColor Red
    }
}

$totalTime = (Get-Date) - $startTime
$successful = ($results | Where-Object { $_.Success }).Count
$avgTime = ($results | Where-Object { $_.Success } | Measure-Object -Property Duration -Average).Average

Write-Host ""
Write-Host "LOAD TEST COMPLETED" -ForegroundColor Green
Write-Host "Total requests: $Count" -ForegroundColor White
Write-Host "Successful: $successful" -ForegroundColor Green
Write-Host "Failed: $($Count - $successful)" -ForegroundColor Red
Write-Host "Success rate: $([Math]::Round(($successful/$Count)*100, 2))%" -ForegroundColor White
Write-Host "Total time: $([Math]::Round($totalTime.TotalSeconds, 2)) seconds" -ForegroundColor White
Write-Host "Requests/second: $([Math]::Round($Count/$totalTime.TotalSeconds, 2))" -ForegroundColor White
Write-Host "Average response time: $([Math]::Round($avgTime, 2)) ms" -ForegroundColor White

Write-Host ""
Write-Host "Now starting processing monitor..." -ForegroundColor Cyan
