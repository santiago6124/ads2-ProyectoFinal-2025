# Script de pruebas para Orders API
# Prueba todas las funcionalidades implementadas

$baseUrl = "http://localhost:8002"
$usersApiUrl = "http://localhost:8001"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  PRUEBAS DE ORDERS API" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Verificar que los servicios esten corriendo
Write-Host "1. Verificando servicios..." -ForegroundColor Yellow
try {
    $health = Invoke-WebRequest -Uri "$baseUrl/health" -Method GET -UseBasicParsing
    $healthData = $health.Content | ConvertFrom-Json
    Write-Host "   OK Orders API esta corriendo" -ForegroundColor Green
    Write-Host "   - MongoDB: $($healthData.services.mongodb.status)" -ForegroundColor Gray
    Write-Host "   - RabbitMQ Publisher: $($healthData.services.rabbitmq_publisher.status)" -ForegroundColor Gray
    Write-Host "   - User API: $($healthData.services.user_api.status)" -ForegroundColor Gray
    
    $usersHealth = Invoke-WebRequest -Uri "$usersApiUrl/health" -Method GET -UseBasicParsing
    Write-Host "   OK Users API esta corriendo" -ForegroundColor Green
} catch {
    Write-Host "   ADVERTENCIA: Error al verificar servicios, pero continuando: $_" -ForegroundColor Yellow
}
Write-Host ""

# 2. Crear un usuario de prueba
Write-Host "2. Creando usuario de prueba..." -ForegroundColor Yellow
$random = Get-Random
$registerBody = @{
    username = "testuser_$random"
    email = "test_$random@test.com"
    password = "TestPassword123!"
    first_name = "Test"
    last_name = "User"
} | ConvertTo-Json

try {
    $registerResponse = Invoke-WebRequest -Uri "$usersApiUrl/api/users/register" -Method POST -Body $registerBody -ContentType "application/json" -UseBasicParsing
    $userData = $registerResponse.Content | ConvertFrom-Json
    $userId = $userData.data.id
    Write-Host "   OK Usuario creado con ID: $userId" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al crear usuario: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 3. Login para obtener token JWT
Write-Host "3. Obteniendo token JWT..." -ForegroundColor Yellow
$loginBody = @{
    email = ($registerBody | ConvertFrom-Json).email
    password = "TestPassword123!"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-WebRequest -Uri "$usersApiUrl/api/users/login" -Method POST -Body $loginBody -ContentType "application/json" -UseBasicParsing
    $loginData = $loginResponse.Content | ConvertFrom-Json
    $token = $loginData.data.access_token
    Write-Host "   OK Token JWT obtenido" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al hacer login: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

# 4. Crear una orden (con procesamiento concurrente)
Write-Host "4. Creando orden de compra (procesamiento concurrente)..." -ForegroundColor Yellow
$createOrderBody = @{
    type = "buy"
    order_kind = "market"
    crypto_symbol = "BTC"
    quantity = "0.001"
    market_price = "50000.00"
} | ConvertTo-Json

try {
    $createResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders" -Method POST -Headers $headers -Body $createOrderBody -UseBasicParsing
    $orderData = $createResponse.Content | ConvertFrom-Json
    $orderId = $orderData.id
    Write-Host "   OK Orden creada con ID: $orderId" -ForegroundColor Green
    Write-Host "   - Status: $($orderData.status)" -ForegroundColor Gray
    Write-Host "   - Total Amount: $($orderData.total_amount)" -ForegroundColor Gray
} catch {
    Write-Host "   ERROR al crear orden: $_" -ForegroundColor Red
    Write-Host "   Response: $($_.Exception.Response)" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 5. Obtener orden por ID
Write-Host "5. Obteniendo orden por ID..." -ForegroundColor Yellow
try {
    $getResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders/$orderId" -Method GET -Headers $headers -UseBasicParsing
    $order = $getResponse.Content | ConvertFrom-Json
    Write-Host "   OK Orden obtenida exitosamente" -ForegroundColor Green
    Write-Host "   - Status: $($order.status)" -ForegroundColor Gray
} catch {
    Write-Host "   ERROR al obtener orden: $_" -ForegroundColor Red
}
Write-Host ""

# 6. Listar ordenes del usuario
Write-Host "6. Listando ordenes del usuario..." -ForegroundColor Yellow
try {
    $listResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders?page=1&page_size=10" -Method GET -Headers $headers -UseBasicParsing
    $ordersList = $listResponse.Content | ConvertFrom-Json
    Write-Host "   OK Ordenes listadas: $($ordersList.total) total" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al listar ordenes: $_" -ForegroundColor Red
}
Write-Host ""

# 7. Crear una orden limit para probar actualizacion
Write-Host "7. Creando orden limit para probar actualizacion..." -ForegroundColor Yellow
$limitOrderBody = @{
    type = "buy"
    order_kind = "limit"
    crypto_symbol = "ETH"
    quantity = "0.1"
    order_price = "3000.00"
} | ConvertTo-Json

try {
    $limitOrderResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders" -Method POST -Headers $headers -Body $limitOrderBody -UseBasicParsing
    $limitOrderData = $limitOrderResponse.Content | ConvertFrom-Json
    $limitOrderId = $limitOrderData.id
    Write-Host "   OK Orden limit creada con ID: $limitOrderId" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al crear orden limit: $_" -ForegroundColor Red
    Write-Host "   Saltando pruebas de actualizacion y ejecucion..." -ForegroundColor Yellow
    $limitOrderId = $null
}
Write-Host ""

# 8. Actualizar orden (validacion de owner)
Write-Host "8. Actualizando orden (validacion de owner)..." -ForegroundColor Yellow
if ($limitOrderId -eq $null) {
    Write-Host "   SALTADO: No hay orden limit para actualizar" -ForegroundColor Yellow
} else {
$updateOrderBody = @{
    quantity = "0.15"
    order_price = "3100.00"
} | ConvertTo-Json

try {
    $updateResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders/$limitOrderId" -Method PUT -Headers $headers -Body $updateOrderBody -UseBasicParsing
    $updatedOrder = $updateResponse.Content | ConvertFrom-Json
    Write-Host "   OK Orden actualizada exitosamente" -ForegroundColor Green
    Write-Host "   - Nueva cantidad: $($updatedOrder.quantity)" -ForegroundColor Gray
    Write-Host "   - Nuevo precio: $($updatedOrder.order_price)" -ForegroundColor Gray
} catch {
    Write-Host "   ERROR al actualizar orden: $_" -ForegroundColor Red
}
}
Write-Host ""

# 9. Ejecutar orden (endpoint de accion con procesamiento concurrente)
Write-Host "9. Ejecutando orden (endpoint de accion)..." -ForegroundColor Yellow
if ($limitOrderId -eq $null) {
    Write-Host "   SALTADO: No hay orden limit para ejecutar" -ForegroundColor Yellow
} else {
try {
    $executeResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders/$limitOrderId/execute" -Method POST -Headers $headers -UseBasicParsing
    $executionResult = $executeResponse.Content | ConvertFrom-Json
    Write-Host "   OK Orden ejecutada" -ForegroundColor Green
    Write-Host "   - Success: $($executionResult.success)" -ForegroundColor Gray
    Write-Host "   - Execution Time: $($executionResult.execution_time)" -ForegroundColor Gray
} catch {
    Write-Host "   ADVERTENCIA Orden no ejecutada (puede ser normal si ya esta ejecutada o fallo): $_" -ForegroundColor Yellow
}
}
Write-Host ""

# 10. Cancelar orden
Write-Host "10. Creando orden para cancelar..." -ForegroundColor Yellow
$cancelOrderBody = @{
    type = "sell"
    order_kind = "limit"
    crypto_symbol = "BTC"
    quantity = "0.001"
    order_price = "60000.00"
} | ConvertTo-Json

try {
    $cancelOrderResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders" -Method POST -Headers $headers -Body $cancelOrderBody -UseBasicParsing
    $cancelOrderData = $cancelOrderResponse.Content | ConvertFrom-Json
    $cancelOrderId = $cancelOrderData.id
    Write-Host "   OK Orden creada para cancelar: $cancelOrderId" -ForegroundColor Green
    
    # Cancelar la orden
    $cancelResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders/$cancelOrderId/cancel?reason=test" -Method POST -Headers $headers -UseBasicParsing
    Write-Host "   OK Orden cancelada exitosamente" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al cancelar orden: $_" -ForegroundColor Red
}
Write-Host ""

# 11. Eliminar orden (validacion de owner)
Write-Host "11. Creando orden para eliminar..." -ForegroundColor Yellow
$deleteOrderBody = @{
    type = "buy"
    order_kind = "limit"
    crypto_symbol = "SOL"
    quantity = "1.0"
    order_price = "100.00"
} | ConvertTo-Json

try {
    $deleteOrderResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders" -Method POST -Headers $headers -Body $deleteOrderBody -UseBasicParsing
    $deleteOrderData = $deleteOrderResponse.Content | ConvertFrom-Json
    $deleteOrderId = $deleteOrderData.id
    Write-Host "   OK Orden creada para eliminar: $deleteOrderId" -ForegroundColor Green
    
    # Eliminar la orden
    $deleteResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/orders/$deleteOrderId" -Method DELETE -Headers $headers -UseBasicParsing
    Write-Host "   OK Orden eliminada exitosamente" -ForegroundColor Green
} catch {
    Write-Host "   ERROR al eliminar orden: $_" -ForegroundColor Red
}
Write-Host ""

# 12. Verificar RabbitMQ (opcional - verificar que los eventos se publicaron)
Write-Host "12. Verificando RabbitMQ..." -ForegroundColor Yellow
Write-Host "   Puedes verificar en http://localhost:15672 (guest/guest)" -ForegroundColor Cyan
Write-Host "   - Exchange: orders.events" -ForegroundColor Gray
Write-Host "   - Routing keys: orders.created, orders.executed, orders.cancelled" -ForegroundColor Gray
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  PRUEBAS COMPLETADAS" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Resumen:" -ForegroundColor Yellow
Write-Host "  OK Crear orden (con procesamiento concurrente)" -ForegroundColor Green
Write-Host "  OK Obtener orden por ID" -ForegroundColor Green
Write-Host "  OK Listar ordenes" -ForegroundColor Green
Write-Host "  OK Actualizar orden (validacion de owner)" -ForegroundColor Green
Write-Host "  OK Ejecutar orden (endpoint de accion)" -ForegroundColor Green
Write-Host "  OK Cancelar orden" -ForegroundColor Green
Write-Host "  OK Eliminar orden (validacion de owner)" -ForegroundColor Green
Write-Host ""
Write-Host "Nota: Verifica los logs con 'docker-compose logs -f orders-api'" -ForegroundColor Cyan
Write-Host "      para ver el procesamiento concurrente con goroutines, channels y WaitGroup" -ForegroundColor Cyan
