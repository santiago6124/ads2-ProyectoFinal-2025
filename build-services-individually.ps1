# ============================================================================
# Build Services Individually
# Builds each service one at a time to avoid network timeout issues
# ============================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Building Services Individually" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$services = @(
    "users-api",
    "users-worker",
    "orders-api",
    "search-api",
    "market-data-api",
    "portfolio-api",
    "frontend"
)

$successCount = 0
$failCount = 0

foreach ($service in $services) {
    Write-Host "`n📦 Building $service..." -ForegroundColor Yellow
    Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

    # Build with increased timeout
    $env:COMPOSE_HTTP_TIMEOUT = "300"
    docker-compose build --no-cache $service 2>&1

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ $service built successfully" -ForegroundColor Green
        $successCount++
    } else {
        Write-Host "❌ $service build failed" -ForegroundColor Red
        $failCount++

        # Ask if user wants to continue
        Write-Host "`nContinue with next service? (Y/N)" -ForegroundColor Yellow
        $response = Read-Host
        if ($response -ne "Y" -and $response -ne "y") {
            Write-Host "Stopping build process..." -ForegroundColor Red
            break
        }
    }
}

# Summary
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Build Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $failCount" -ForegroundColor Red

if ($failCount -eq 0) {
    Write-Host "`n✅ All services built successfully!" -ForegroundColor Green
    Write-Host "Now you can run: docker-compose up -d" -ForegroundColor Gray
} else {
    Write-Host "`n⚠️  Some services failed to build" -ForegroundColor Yellow
    Write-Host "Check the errors above and try again" -ForegroundColor Gray
}
