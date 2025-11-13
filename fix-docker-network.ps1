# ============================================================================
# Docker Network Troubleshooting Script
# Fixes common Docker Hub connectivity issues
# ============================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Docker Network Troubleshooting" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Restart Docker Desktop
Write-Host "📋 Step 1: Restarting Docker Desktop..." -ForegroundColor Yellow
Write-Host "Please close Docker Desktop manually and wait 10 seconds..." -ForegroundColor Gray
Write-Host "Press any key when ready to continue..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

# Step 2: Clean Docker Build Cache
Write-Host "`n📋 Step 2: Cleaning Docker build cache..." -ForegroundColor Yellow
docker builder prune -af
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Build cache cleaned" -ForegroundColor Green
} else {
    Write-Host "⚠️  Failed to clean cache" -ForegroundColor Yellow
}

# Step 3: Test Docker Hub connectivity
Write-Host "`n📋 Step 3: Testing Docker Hub connectivity..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "https://registry-1.docker.io/v2/" -Method GET -TimeoutSec 10 -UseBasicParsing
    Write-Host "✅ Can reach Docker Hub (HTTP $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "❌ Cannot reach Docker Hub: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "⚠️  Check your internet connection or firewall settings" -ForegroundColor Yellow
}

# Step 4: Pull base images manually
Write-Host "`n📋 Step 4: Pulling base images manually..." -ForegroundColor Yellow

$images = @(
    "golang:1.21-alpine",
    "alpine:3.19"
)

foreach ($image in $images) {
    Write-Host "  Pulling $image..." -ForegroundColor Cyan
    docker pull $image
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✅ $image pulled successfully" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Failed to pull $image" -ForegroundColor Red
    }
}

# Step 5: Configure Docker daemon (optional)
Write-Host "`n📋 Step 5: Docker daemon configuration..." -ForegroundColor Yellow
Write-Host "Current Docker info:" -ForegroundColor Gray
docker info | Select-String -Pattern "Registry"

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Troubleshooting Complete" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "If issues persist, try:" -ForegroundColor Yellow
Write-Host "  1. Restart your computer" -ForegroundColor Gray
Write-Host "  2. Check Windows Firewall settings" -ForegroundColor Gray
Write-Host "  3. Try using a VPN or different network" -ForegroundColor Gray
Write-Host "  4. Check Docker Desktop settings → Resources → Network" -ForegroundColor Gray
Write-Host ""
