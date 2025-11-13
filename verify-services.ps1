# ============================================================================
# CryptoSim - Service Verification Script
# Quick check of all services and their status
# ============================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "CryptoSim Service Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Service endpoints mapped from docker-compose.yml
$services = @{
    "Users API" = @{
        Url = "http://localhost:8001/health"
        Port = 8001
    }
    "Orders API" = @{
        Url = "http://localhost:8002/health"
        Port = 8002
    }
    "Search API" = @{
        Url = "http://localhost:8003/api/v1/health"
        Port = 8003
    }
    "Market Data API" = @{
        Url = "http://localhost:8004/health"
        Port = 8004
    }
    "Portfolio API" = @{
        Url = "http://localhost:8005/health"
        Port = 8005
    }
    "Frontend" = @{
        Url = "http://localhost:3000"
        Port = 3000
    }
    "RabbitMQ Management" = @{
        Url = "http://localhost:15672"
        Port = 15672
    }
    "Solr" = @{
        Url = "http://localhost:8983/solr"
        Port = 8983
    }
}

# Docker containers
$containers = @(
    "cryptosim-users-api",
    "cryptosim-users-worker",
    "cryptosim-orders-api",
    "cryptosim-search-api",
    "cryptosim-market-data-api",
    "cryptosim-portfolio-api",
    "cryptosim-frontend",
    "cryptosim-users-mysql",
    "cryptosim-orders-mongo",
    "cryptosim-portfolio-mongo",
    "cryptosim-redis",
    "cryptosim-rabbitmq",
    "cryptosim-solr",
    "cryptosim-memcached"
)

# ============================================================================
# Check Docker Containers
# ============================================================================
Write-Host "📦 Docker Containers Status" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

foreach ($container in $containers) {
    try {
        $status = docker ps --filter "name=$container" --format "{{.Status}}"
        if ($status) {
            if ($status -match "Up") {
                Write-Host "  ✅ $container" -ForegroundColor Green
                Write-Host "     $status" -ForegroundColor Gray
            } else {
                Write-Host "  ⚠️  $container" -ForegroundColor Yellow
                Write-Host "     $status" -ForegroundColor Gray
            }
        } else {
            Write-Host "  ❌ $container - NOT RUNNING" -ForegroundColor Red
        }
    } catch {
        Write-Host "  ❌ $container - ERROR: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# ============================================================================
# Check Service Health Endpoints
# ============================================================================
Write-Host "`n🌐 Service Health Endpoints" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

foreach ($serviceName in $services.Keys | Sort-Object) {
    $service = $services[$serviceName]
    Write-Host "  Testing $serviceName (port $($service.Port))..." -ForegroundColor Cyan

    try {
        $response = Invoke-WebRequest -Uri $service.Url -Method GET -TimeoutSec 3 -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Write-Host "    ✅ HEALTHY (HTTP $($response.StatusCode))" -ForegroundColor Green
        } else {
            Write-Host "    ⚠️  RESPONDING (HTTP $($response.StatusCode))" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "    ❌ UNREACHABLE - $($_.Exception.Message)" -ForegroundColor Red
    }
}

# ============================================================================
# Check RabbitMQ Queues
# ============================================================================
Write-Host "`n🐰 RabbitMQ Queues" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    $base64Auth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("guest:guest"))
    $headers = @{
        Authorization = "Basic $base64Auth"
    }

    $queues = Invoke-RestMethod -Uri "http://localhost:15672/api/queues/%2F" -Headers $headers -Method GET

    # Balance messaging queues
    $balanceQueues = @("balance.request", "balance.response.portfolio")

    foreach ($queueName in $balanceQueues) {
        $queue = $queues | Where-Object { $_.name -eq $queueName }
        if ($null -ne $queue) {
            Write-Host "  ✅ Queue: $queueName" -ForegroundColor Green
            Write-Host "     Messages: $($queue.messages) | Consumers: $($queue.consumers)" -ForegroundColor Gray

            if ($queue.consumers -eq 0) {
                Write-Host "     ⚠️  WARNING: No consumers connected!" -ForegroundColor Yellow
            }
            if ($queue.messages_ready -gt 0) {
                Write-Host "     ⚠️  WARNING: $($queue.messages_ready) messages waiting!" -ForegroundColor Yellow
            }
        } else {
            Write-Host "  ❌ Queue: $queueName - NOT FOUND" -ForegroundColor Red
        }
    }

    # Check exchanges
    $exchanges = Invoke-RestMethod -Uri "http://localhost:15672/api/exchanges/%2F" -Headers $headers -Method GET
    $balanceExchanges = @("balance.request.exchange", "balance.response.exchange")

    Write-Host ""
    foreach ($exchangeName in $balanceExchanges) {
        $exchange = $exchanges | Where-Object { $_.name -eq $exchangeName }
        if ($null -ne $exchange) {
            Write-Host "  ✅ Exchange: $exchangeName (type: $($exchange.type))" -ForegroundColor Green
        } else {
            Write-Host "  ❌ Exchange: $exchangeName - NOT FOUND" -ForegroundColor Red
        }
    }

} catch {
    Write-Host "  ❌ Could not connect to RabbitMQ Management API" -ForegroundColor Red
    Write-Host "     $($_.Exception.Message)" -ForegroundColor Gray
}

# ============================================================================
# Check Database Connections
# ============================================================================
Write-Host "`n💾 Database Connections" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

# MySQL
try {
    docker exec cryptosim-users-mysql mysqladmin ping -h localhost -u root -prootpassword 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✅ MySQL (users-mysql)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ MySQL (users-mysql) - Not responding" -ForegroundColor Red
    }
} catch {
    Write-Host "  ❌ MySQL (users-mysql) - $($_.Exception.Message)" -ForegroundColor Red
}

# MongoDB - Orders
try {
    $result = docker exec cryptosim-orders-mongo mongosh --eval "db.adminCommand('ping')" --quiet 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✅ MongoDB (orders-mongo)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ MongoDB (orders-mongo) - Not responding" -ForegroundColor Red
    }
} catch {
    Write-Host "  ❌ MongoDB (orders-mongo) - $($_.Exception.Message)" -ForegroundColor Red
}

# MongoDB - Portfolio
try {
    $result = docker exec cryptosim-portfolio-mongo mongosh --eval "db.adminCommand('ping')" --quiet 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✅ MongoDB (portfolio-mongo)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ MongoDB (portfolio-mongo) - Not responding" -ForegroundColor Red
    }
} catch {
    Write-Host "  ❌ MongoDB (portfolio-mongo) - $($_.Exception.Message)" -ForegroundColor Red
}

# Redis
try {
    $result = docker exec cryptosim-redis redis-cli ping 2>&1
    if ($result -match "PONG") {
        Write-Host "  ✅ Redis" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Redis - Not responding" -ForegroundColor Red
    }
} catch {
    Write-Host "  ❌ Redis - $($_.Exception.Message)" -ForegroundColor Red
}

# ============================================================================
# Summary
# ============================================================================
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Verification Complete" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "  1. Check RabbitMQ Management UI: http://localhost:15672 (guest/guest)" -ForegroundColor Gray
Write-Host "  2. Run integration tests: .\test-balance-messaging.ps1" -ForegroundColor Gray
Write-Host "  3. View service logs: docker-compose logs -f [service-name]" -ForegroundColor Gray
Write-Host ""
