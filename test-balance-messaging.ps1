# ============================================================================
# CryptoSim - Balance Messaging Integration Test
# Tests the RabbitMQ request-response pattern for balance communication
# ============================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "CryptoSim Balance Messaging Test Suite" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$USERS_API = "http://localhost:8001"
$PORTFOLIO_API = "http://localhost:8005"
$RABBITMQ_MGMT = "http://localhost:15672"
$RABBITMQ_USER = "guest"
$RABBITMQ_PASS = "guest"

# Test counters
$global:TestsPassed = 0
$global:TestsFailed = 0

# Helper function for test assertions
function Test-Assert {
    param(
        [string]$TestName,
        [bool]$Condition,
        [string]$Message
    )

    if ($Condition) {
        Write-Host "✅ PASS: $TestName" -ForegroundColor Green
        $global:TestsPassed++
    } else {
        Write-Host "❌ FAIL: $TestName - $Message" -ForegroundColor Red
        $global:TestsFailed++
    }
}

# ============================================================================
# Test 1: Service Health Checks
# ============================================================================
Write-Host "`n📋 Test 1: Service Health Checks" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

function Test-ServiceHealth {
    param([string]$ServiceName, [string]$Url)

    try {
        $response = Invoke-WebRequest -Uri $Url -Method GET -TimeoutSec 5 -UseBasicParsing
        Test-Assert `
            "Service $ServiceName is healthy" `
            ($response.StatusCode -eq 200) `
            "Status code: $($response.StatusCode)"
    } catch {
        Test-Assert `
            "Service $ServiceName is healthy" `
            $false `
            "Service unreachable: $($_.Exception.Message)"
    }
}

Test-ServiceHealth "users-api" "$USERS_API/health"
Test-ServiceHealth "portfolio-api" "$PORTFOLIO_API/health"

# ============================================================================
# Test 2: RabbitMQ Infrastructure
# ============================================================================
Write-Host "`n📋 Test 2: RabbitMQ Infrastructure" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    $base64Auth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("${RABBITMQ_USER}:${RABBITMQ_PASS}"))
    $headers = @{
        Authorization = "Basic $base64Auth"
    }

    # Check exchanges
    $exchanges = Invoke-RestMethod -Uri "$RABBITMQ_MGMT/api/exchanges/%2F" -Headers $headers -Method GET
    $balanceRequestExchange = $exchanges | Where-Object { $_.name -eq "balance.request.exchange" }
    $balanceResponseExchange = $exchanges | Where-Object { $_.name -eq "balance.response.exchange" }

    Test-Assert `
        "Exchange 'balance.request.exchange' exists" `
        ($null -ne $balanceRequestExchange) `
        "Exchange not found"

    Test-Assert `
        "Exchange 'balance.response.exchange' exists" `
        ($null -ne $balanceResponseExchange) `
        "Exchange not found"

    # Check queues
    $queues = Invoke-RestMethod -Uri "$RABBITMQ_MGMT/api/queues/%2F" -Headers $headers -Method GET
    $balanceRequestQueue = $queues | Where-Object { $_.name -eq "balance.request" }
    $balanceResponseQueue = $queues | Where-Object { $_.name -eq "balance.response.portfolio" }

    Test-Assert `
        "Queue 'balance.request' exists" `
        ($null -ne $balanceRequestQueue) `
        "Queue not found"

    Test-Assert `
        "Queue 'balance.response.portfolio' exists" `
        ($null -ne $balanceResponseQueue) `
        "Queue not found"

    # Check consumers
    if ($null -ne $balanceRequestQueue) {
        Test-Assert `
            "Users worker is consuming from 'balance.request'" `
            ($balanceRequestQueue.consumers -gt 0) `
            "No consumers connected"
    }

    if ($null -ne $balanceResponseQueue) {
        Test-Assert `
            "Portfolio API is consuming from 'balance.response.portfolio'" `
            ($balanceResponseQueue.consumers -gt 0) `
            "No consumers connected"
    }

} catch {
    Write-Host "❌ Failed to check RabbitMQ infrastructure: $($_.Exception.Message)" -ForegroundColor Red
    $global:TestsFailed += 4
}

# ============================================================================
# Test 3: User Registration & Login
# ============================================================================
Write-Host "`n📋 Test 3: User Registration & Login" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

$timestamp = Get-Date -Format "yyyyMMddHHmmss"
$testUser = @{
    username = "testuser_$timestamp"
    email = "testuser_$timestamp@example.com"
    password = "TestPassword123!"
    first_name = "Test"
    last_name = "User"
}

try {
    # Register user
    $registerResponse = Invoke-RestMethod `
        -Uri "$USERS_API/api/auth/register" `
        -Method POST `
        -ContentType "application/json" `
        -Body ($testUser | ConvertTo-Json)

    Test-Assert `
        "User registration successful" `
        ($null -ne $registerResponse.id) `
        "No user ID returned"

    $userId = $registerResponse.id
    Write-Host "   Created user ID: $userId" -ForegroundColor Gray

    # Login
    $loginPayload = @{
        email = $testUser.email
        password = $testUser.password
    }

    $loginResponse = Invoke-RestMethod `
        -Uri "$USERS_API/api/auth/login" `
        -Method POST `
        -ContentType "application/json" `
        -Body ($loginPayload | ConvertTo-Json)

    Test-Assert `
        "User login successful" `
        ($null -ne $loginResponse.access_token) `
        "No access token returned"

    $accessToken = $loginResponse.access_token
    Write-Host "   Access token obtained" -ForegroundColor Gray

    # Verify user balance
    $userHeaders = @{
        Authorization = "Bearer $accessToken"
    }

    $userDetails = Invoke-RestMethod `
        -Uri "$USERS_API/api/users/$userId" `
        -Method GET `
        -Headers $userHeaders

    Test-Assert `
        "User has initial balance" `
        ($userDetails.current_balance -eq 100000.0) `
        "Balance: $($userDetails.current_balance)"

    Write-Host "   User balance: $($userDetails.current_balance)" -ForegroundColor Gray

} catch {
    Write-Host "❌ User registration/login failed: $($_.Exception.Message)" -ForegroundColor Red
    $global:TestsFailed += 3
    exit 1
}

# ============================================================================
# Test 4: Portfolio API Balance Request (RabbitMQ)
# ============================================================================
Write-Host "`n📋 Test 4: Portfolio API Balance Request via RabbitMQ" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    # Get portfolio (this triggers balance request via RabbitMQ)
    Write-Host "   Requesting portfolio for user $userId..." -ForegroundColor Gray

    $portfolioResponse = Invoke-RestMethod `
        -Uri "$PORTFOLIO_API/api/portfolios/$userId" `
        -Method GET `
        -Headers $userHeaders `
        -TimeoutSec 10

    Test-Assert `
        "Portfolio retrieved successfully" `
        ($null -ne $portfolioResponse) `
        "No portfolio data returned"

    Test-Assert `
        "Portfolio contains balance data" `
        ($null -ne $portfolioResponse.total_cash) `
        "No balance in portfolio"

    # Verify balance matches
    $portfolioBalance = $portfolioResponse.total_cash
    $expectedBalance = 100000.0

    Test-Assert `
        "Balance matches expected value" `
        ($portfolioBalance -eq $expectedBalance) `
        "Expected: $expectedBalance, Got: $portfolioBalance"

    Write-Host "   Portfolio balance: $portfolioBalance" -ForegroundColor Gray

} catch {
    Write-Host "❌ Portfolio balance request failed: $($_.Exception.Message)" -ForegroundColor Red
    $global:TestsFailed += 3
}

# ============================================================================
# Test 5: Message Flow Verification
# ============================================================================
Write-Host "`n📋 Test 5: Message Flow Verification" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    # Check message rates
    Start-Sleep -Seconds 2  # Wait for messages to process

    $queues = Invoke-RestMethod -Uri "$RABBITMQ_MGMT/api/queues/%2F" -Headers $headers -Method GET
    $balanceRequestQueue = $queues | Where-Object { $_.name -eq "balance.request" }
    $balanceResponseQueue = $queues | Where-Object { $_.name -eq "balance.response.portfolio" }

    if ($null -ne $balanceRequestQueue) {
        Write-Host "   balance.request queue:" -ForegroundColor Gray
        Write-Host "     - Messages ready: $($balanceRequestQueue.messages_ready)" -ForegroundColor Gray
        Write-Host "     - Messages unacked: $($balanceRequestQueue.messages_unacknowledged)" -ForegroundColor Gray
        Write-Host "     - Total messages: $($balanceRequestQueue.messages)" -ForegroundColor Gray

        Test-Assert `
            "No messages stuck in balance.request queue" `
            ($balanceRequestQueue.messages_ready -eq 0) `
            "Messages stuck: $($balanceRequestQueue.messages_ready)"
    }

    if ($null -ne $balanceResponseQueue) {
        Write-Host "   balance.response.portfolio queue:" -ForegroundColor Gray
        Write-Host "     - Messages ready: $($balanceResponseQueue.messages_ready)" -ForegroundColor Gray
        Write-Host "     - Messages unacked: $($balanceResponseQueue.messages_unacknowledged)" -ForegroundColor Gray
        Write-Host "     - Total messages: $($balanceResponseQueue.messages)" -ForegroundColor Gray

        Test-Assert `
            "No messages stuck in balance.response queue" `
            ($balanceResponseQueue.messages_ready -eq 0) `
            "Messages stuck: $($balanceResponseQueue.messages_ready)"
    }

} catch {
    Write-Host "❌ Failed to verify message flow: $($_.Exception.Message)" -ForegroundColor Red
    $global:TestsFailed += 2
}

# ============================================================================
# Test 6: Load Test (Multiple Concurrent Requests)
# ============================================================================
Write-Host "`n📋 Test 6: Load Test (10 concurrent requests)" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    Write-Host "   Sending 10 concurrent portfolio requests..." -ForegroundColor Gray

    $jobs = 1..10 | ForEach-Object {
        Start-Job -ScriptBlock {
            param($Url, $Token)
            $headers = @{ Authorization = "Bearer $Token" }
            try {
                $response = Invoke-RestMethod -Uri $Url -Method GET -Headers $headers -TimeoutSec 10
                return @{ Success = $true; Balance = $response.total_cash }
            } catch {
                return @{ Success = $false; Error = $_.Exception.Message }
            }
        } -ArgumentList "$PORTFOLIO_API/api/portfolios/$userId", $accessToken
    }

    $results = $jobs | Wait-Job | Receive-Job
    $jobs | Remove-Job

    $successCount = ($results | Where-Object { $_.Success -eq $true }).Count
    $failCount = ($results | Where-Object { $_.Success -eq $false }).Count

    Write-Host "   Results: $successCount success, $failCount failed" -ForegroundColor Gray

    Test-Assert `
        "All concurrent requests succeeded" `
        ($successCount -eq 10) `
        "$failCount requests failed"

    # Verify all balances are consistent
    $balances = $results | Where-Object { $_.Success -eq $true } | Select-Object -ExpandProperty Balance
    $uniqueBalances = $balances | Select-Object -Unique

    Test-Assert `
        "All responses have consistent balance" `
        ($uniqueBalances.Count -eq 1) `
        "Inconsistent balances: $($uniqueBalances -join ', ')"

} catch {
    Write-Host "❌ Load test failed: $($_.Exception.Message)" -ForegroundColor Red
    $global:TestsFailed += 2
}

# ============================================================================
# Test 7: Check Service Logs
# ============================================================================
Write-Host "`n📋 Test 7: Service Logs Check" -ForegroundColor Yellow
Write-Host "─────────────────────────────────────────" -ForegroundColor Yellow

try {
    Write-Host "   Checking users-worker logs..." -ForegroundColor Gray
    $workerLogs = docker logs cryptosim-users-worker --tail 50 2>&1

    $hasReceivedRequest = $workerLogs -match "Received balance request"
    $hasSentResponse = $workerLogs -match "Sent balance response"

    Test-Assert `
        "Users worker received balance requests" `
        $hasReceivedRequest `
        "No 'Received balance request' in logs"

    Test-Assert `
        "Users worker sent balance responses" `
        $hasSentResponse `
        "No 'Sent balance response' in logs"

    Write-Host "`n   Checking portfolio-api logs..." -ForegroundColor Gray
    $portfolioLogs = docker logs cryptosim-portfolio-api --tail 50 2>&1

    # Note: Portfolio logs depend on implementation - update when service is integrated

} catch {
    Write-Host "⚠️  Could not check logs (services might not be running)" -ForegroundColor Yellow
}

# ============================================================================
# Test Summary
# ============================================================================
Write-Host "`n" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Total Tests Passed: $global:TestsPassed" -ForegroundColor Green
Write-Host "Total Tests Failed: $global:TestsFailed" -ForegroundColor Red

if ($global:TestsFailed -eq 0) {
    Write-Host "`n✅ ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "The balance messaging system is working correctly." -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n❌ SOME TESTS FAILED!" -ForegroundColor Red
    Write-Host "Please check the errors above and review the logs." -ForegroundColor Red
    exit 1
}
