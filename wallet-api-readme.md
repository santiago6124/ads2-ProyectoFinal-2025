# 💰 Wallet API - Microservicio de Gestión de Billetera Virtual

## 📋 Descripción

El microservicio **Wallet API** es el componente crítico que gestiona las billeteras virtuales y el saldo de los usuarios en CryptoSim. Implementa transacciones ACID para garantizar la integridad financiera, maneja bloqueos de fondos durante las órdenes y mantiene un historial completo de todas las transacciones monetarias.

## 🎯 Responsabilidades

- **Gestión de Saldos**: Control de balance disponible y bloqueado
- **Transacciones ACID**: Garantía de consistencia en operaciones monetarias
- **Bloqueo de Fondos**: Reserva de fondos durante ejecución de órdenes
- **Historial de Transacciones**: Registro auditable de todos los movimientos
- **Validación de Fondos**: Verificación de saldo antes de operaciones
- **Conciliación**: Reconciliación automática de saldos
- **Multi-moneda**: Soporte para múltiples monedas (futuro)
- **Límites y Restricciones**: Control de límites de retiro y operación

## 🏗️ Arquitectura

### Estructura del Proyecto
```
wallet-api/
├── cmd/
│   └── main.go                        # Punto de entrada
├── internal/
│   ├── controllers/                   # Controladores HTTP
│   │   ├── wallet_controller.go
│   │   ├── transaction_controller.go
│   │   ├── balance_controller.go
│   │   └── admin_controller.go
│   ├── services/                      # Lógica de negocio
│   │   ├── wallet_service.go
│   │   ├── transaction_service.go
│   │   ├── balance_service.go
│   │   ├── locking_service.go        # Gestión de bloqueos
│   │   ├── reconciliation_service.go # Conciliación
│   │   └── validation_service.go
│   ├── repositories/                  # Acceso a datos
│   │   ├── wallet_repository.go
│   │   ├── transaction_repository.go
│   │   └── mongodb_repository.go
│   ├── models/                        # Modelos de dominio
│   │   ├── wallet.go
│   │   ├── transaction.go
│   │   ├── balance.go
│   │   ├── lock.go
│   │   └── limits.go
│   ├── dto/                           # Data Transfer Objects
│   │   ├── wallet_response.go
│   │   ├── transaction_dto.go
│   │   ├── deposit_request.go
│   │   └── withdrawal_request.go
│   ├── transaction/                   # Motor transaccional
│   │   ├── manager.go                 # Transaction manager
│   │   ├── saga.go                    # Saga pattern
│   │   ├── rollback.go
│   │   └── idempotency.go
│   ├── locking/                       # Sistema de bloqueos
│   │   ├── lock_manager.go
│   │   ├── distributed_lock.go
│   │   └── timeout_handler.go
│   ├── audit/                         # Auditoría
│   │   ├── transaction_logger.go
│   │   ├── audit_trail.go
│   │   └── compliance.go
│   ├── validators/                    # Validaciones
│   │   ├── amount_validator.go
│   │   ├── limits_validator.go
│   │   └── fraud_detector.go
│   ├── clients/                       # Clientes internos
│   │   ├── users_client.go
│   │   └── orders_client.go
│   ├── middleware/                    # Middlewares
│   │   ├── auth_middleware.go
│   │   ├── idempotency_middleware.go
│   │   ├── rate_limit_middleware.go
│   │   └── logging_middleware.go
│   └── config/                        # Configuración
│       └── config.go
├── pkg/
│   ├── utils/                         # Utilidades
│   │   ├── decimal.go
│   │   ├── uuid.go
│   │   └── response.go
│   ├── errors/                        # Manejo de errores
│   │   └── wallet_errors.go
│   └── security/                      # Seguridad
│       ├── encryption.go
│       └── hash.go
├── tests/                             # Tests
│   ├── unit/
│   │   ├── wallet_service_test.go
│   │   └── transaction_test.go
│   ├── integration/
│   │   └── transaction_flow_test.go
│   └── stress/
│       └── concurrent_transactions_test.go
├── scripts/                           # Scripts de utilidad
│   ├── reconcile_balances.sh
│   ├── audit_report.sh
│   └── migrate_data.sh
├── docs/                              # Documentación
│   ├── swagger.yaml
│   ├── transaction_flow.md
│   └── security_measures.md
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Modelo de Datos

### Colección: wallets (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  "wallet_number": "WAL-2024-000123",
  "status": "active", // "active" | "suspended" | "closed"
  
  "balance": {
    "available": NumberDecimal("75000.00"),
    "locked": NumberDecimal("25000.00"),
    "total": NumberDecimal("100000.00"),
    "currency": "USD"
  },
  
  "limits": {
    "daily_withdrawal": NumberDecimal("10000.00"),
    "daily_deposit": NumberDecimal("50000.00"),
    "single_transaction": NumberDecimal("25000.00"),
    "monthly_volume": NumberDecimal("500000.00")
  },
  
  "usage_today": {
    "withdrawn": NumberDecimal("2000.00"),
    "deposited": NumberDecimal("5000.00"),
    "transactions_count": 8,
    "last_transaction": ISODate("2024-01-15T10:25:00Z")
  },
  
  "locks": [
    {
      "lock_id": "LOCK-2024-000456",
      "order_id": "ORD-2024-000789",
      "amount": NumberDecimal("25000.00"),
      "locked_at": ISODate("2024-01-15T10:20:00Z"),
      "expires_at": ISODate("2024-01-15T10:50:00Z"),
      "status": "active", // "active" | "released" | "executed" | "expired"
      "reason": "order_execution"
    }
  ],
  
  "verification": {
    "last_reconciled": ISODate("2024-01-15T00:00:00Z"),
    "balance_hash": "3b4c5d6e7f8a9b0c1d2e3f4g5h6i7j8k",
    "transaction_count": 1547,
    "checksum": "a1b2c3d4e5f6"
  },
  
  "metadata": {
    "initial_balance": NumberDecimal("100000.00"),
    "total_deposits": NumberDecimal("150000.00"),
    "total_withdrawals": NumberDecimal("50000.00"),
    "total_fees_paid": NumberDecimal("500.00"),
    "account_age_days": 15
  },
  
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_at": ISODate("2024-01-15T10:25:00Z"),
  "last_activity": ISODate("2024-01-15T10:25:00Z")
}
```

### Colección: wallet_transactions (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439012"),
  "transaction_id": "TXN-2024-000012345",
  "wallet_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  "idempotency_key": "ord-exec-789-attempt-1",
  
  "type": "order_execute", // "deposit" | "withdrawal" | "order_lock" | "order_release" | "order_execute" | "fee" | "refund" | "adjustment"
  "status": "completed", // "pending" | "processing" | "completed" | "failed" | "reversed"
  
  "amount": {
    "value": NumberDecimal("-22525.00"),
    "currency": "USD",
    "fee": NumberDecimal("22.50"),
    "net": NumberDecimal("-22547.50")
  },
  
  "balance": {
    "before": NumberDecimal("100000.00"),
    "after": NumberDecimal("77452.50"),
    "available_before": NumberDecimal("75000.00"),
    "available_after": NumberDecimal("77452.50"),
    "locked_before": NumberDecimal("25000.00"),
    "locked_after": NumberDecimal("0.00")
  },
  
  "reference": {
    "type": "order",
    "id": "ORD-2024-000789",
    "description": "Buy 0.5 BTC at $45,000",
    "metadata": {
      "crypto_symbol": "BTC",
      "quantity": 0.5,
      "price": 45000.00
    }
  },
  
  "processing": {
    "initiated_at": ISODate("2024-01-15T10:25:00Z"),
    "completed_at": ISODate("2024-01-15T10:25:01Z"),
    "processing_time_ms": 1000,
    "attempts": 1,
    "errors": []
  },
  
  "audit": {
    "ip_address": "192.168.1.100",
    "user_agent": "CryptoSim/1.0",
    "session_id": "sess_abc123",
    "api_version": "v1"
  },
  
  "reversal": {
    "is_reversed": false,
    "reversed_by": null,
    "reversal_transaction_id": null,
    "reversal_reason": null
  },
  
  "created_at": ISODate("2024-01-15T10:25:00Z"),
  "updated_at": ISODate("2024-01-15T10:25:01Z")
}
```

### Colección: transaction_locks (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439013"),
  "lock_id": "LOCK-2024-000456",
  "wallet_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  
  "amount": NumberDecimal("25000.00"),
  "reason": "order_execution",
  "reference_id": "ORD-2024-000789",
  
  "status": "active", // "active" | "released" | "executed" | "expired" | "cancelled"
  
  "timeline": {
    "created_at": ISODate("2024-01-15T10:20:00Z"),
    "expires_at": ISODate("2024-01-15T10:50:00Z"),
    "released_at": null,
    "executed_at": null
  },
  
  "metadata": {
    "order_type": "buy",
    "crypto_symbol": "BTC",
    "estimated_price": 45000.00
  }
}
```

### Índices MongoDB
```javascript
// Índices para optimización
db.wallets.createIndex({ "user_id": 1 }, { unique: true })
db.wallets.createIndex({ "wallet_number": 1 }, { unique: true })
db.wallets.createIndex({ "status": 1 })
db.wallets.createIndex({ "updated_at": -1 })

db.wallet_transactions.createIndex({ "wallet_id": 1, "created_at": -1 })
db.wallet_transactions.createIndex({ "transaction_id": 1 }, { unique: true })
db.wallet_transactions.createIndex({ "idempotency_key": 1 }, { unique: true })
db.wallet_transactions.createIndex({ "user_id": 1, "created_at": -1 })
db.wallet_transactions.createIndex({ "reference.type": 1, "reference.id": 1 })
db.wallet_transactions.createIndex({ "type": 1, "status": 1 })

db.transaction_locks.createIndex({ "lock_id": 1 }, { unique: true })
db.transaction_locks.createIndex({ "wallet_id": 1, "status": 1 })
db.transaction_locks.createIndex({ "reference_id": 1 })
db.transaction_locks.createIndex({ "timeline.expires_at": 1 }, { expireAfterSeconds: 0 })
```

## 🔌 API Endpoints

### Gestión de Billetera

#### GET `/api/wallet/:userId`
Obtiene la información completa de la billetera del usuario.

**Headers:**
```
Authorization: Bearer [token]
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "wallet_number": "WAL-2024-000123",
    "user_id": 123,
    "status": "active",
    "balance": {
      "available": 75000.00,
      "locked": 25000.00,
      "total": 100000.00,
      "currency": "USD"
    },
    "limits": {
      "daily_withdrawal": 10000.00,
      "daily_deposit": 50000.00,
      "single_transaction": 25000.00,
      "monthly_volume": 500000.00
    },
    "usage": {
      "today": {
        "withdrawn": 2000.00,
        "deposited": 5000.00,
        "remaining_withdrawal": 8000.00,
        "remaining_deposit": 45000.00
      },
      "this_month": {
        "total_volume": 125000.00,
        "remaining_volume": 375000.00
      }
    },
    "active_locks": 1,
    "last_activity": "2024-01-15T10:25:00Z"
  }
}
```

#### GET `/api/wallet/:userId/balance`
Obtiene solo el saldo disponible (endpoint optimizado).

**Headers:**
```
Authorization: Bearer [token]
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "available": 75000.00,
    "locked": 25000.00,
    "total": 100000.00,
    "currency": "USD",
    "as_of": "2024-01-15T10:30:00Z"
  }
}
```

### Transacciones

#### GET `/api/wallet/:userId/transactions`
Lista el historial de transacciones.

**Query Parameters:**
- `type`: Tipo de transacción (deposit, withdrawal, order_lock, etc.)
- `status`: Estado (completed, failed, reversed)
- `from`: Fecha inicial (YYYY-MM-DD)
- `to`: Fecha final (YYYY-MM-DD)
- `page`: Página (default: 1)
- `limit`: Límite por página (default: 20, max: 100)
- `sort`: Ordenamiento (created_at, -created_at, amount, -amount)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "transaction_id": "TXN-2024-000012345",
        "type": "order_execute",
        "status": "completed",
        "amount": -22547.50,
        "description": "Buy 0.5 BTC at $45,000",
        "balance_after": 77452.50,
        "created_at": "2024-01-15T10:25:00Z"
      },
      {
        "transaction_id": "TXN-2024-000012344",
        "type": "deposit",
        "status": "completed",
        "amount": 5000.00,
        "description": "Virtual deposit",
        "balance_after": 100000.00,
        "created_at": "2024-01-15T09:00:00Z"
      }
    ],
    "pagination": {
      "total": 1547,
      "page": 1,
      "limit": 20,
      "total_pages": 78
    },
    "summary": {
      "total_deposits": 150000.00,
      "total_withdrawals": 50000.00,
      "total_fees": 500.00,
      "net_change": 100000.00
    }
  }
}
```

#### GET `/api/wallet/:userId/transaction/:transactionId`
Obtiene detalles de una transacción específica.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "transaction_id": "TXN-2024-000012345",
    "type": "order_execute",
    "status": "completed",
    "amount": {
      "value": -22525.00,
      "fee": 22.50,
      "net": -22547.50,
      "currency": "USD"
    },
    "balance": {
      "before": 100000.00,
      "after": 77452.50,
      "available_before": 75000.00,
      "available_after": 77452.50
    },
    "reference": {
      "type": "order",
      "id": "ORD-2024-000789",
      "description": "Buy 0.5 BTC at $45,000",
      "crypto_symbol": "BTC",
      "quantity": 0.5,
      "execution_price": 45000.00
    },
    "timeline": {
      "initiated": "2024-01-15T10:25:00Z",
      "completed": "2024-01-15T10:25:01Z",
      "processing_time_ms": 1000
    },
    "reversible": false
  }
}
```

### Operaciones de Fondos

#### POST `/api/wallet/:userId/deposit`
Deposita fondos virtuales (solo admin).

**Headers:**
```
Authorization: Bearer [admin-token]
Content-Type: application/json
```

**Request Body:**
```json
{
  "amount": 10000.00,
  "reason": "Monthly bonus",
  "reference": "BONUS-2024-01"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Depósito procesado exitosamente",
  "data": {
    "transaction_id": "TXN-2024-000012346",
    "amount": 10000.00,
    "new_balance": 87452.50,
    "timestamp": "2024-01-15T10:35:00Z"
  }
}
```

#### POST `/api/wallet/:userId/withdraw`
Retira fondos virtuales.

**Headers:**
```
Authorization: Bearer [token]
X-Idempotency-Key: unique-key-123
```

**Request Body:**
```json
{
  "amount": 5000.00,
  "reason": "Profit taking",
  "destination": "external_wallet"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Retiro procesado exitosamente",
  "data": {
    "transaction_id": "TXN-2024-000012347",
    "amount": 5000.00,
    "fee": 5.00,
    "net_amount": 4995.00,
    "new_balance": 82457.50,
    "timestamp": "2024-01-15T10:40:00Z"
  }
}
```

### Bloqueo de Fondos

#### POST `/api/wallet/:userId/lock`
Bloquea fondos para una orden (interno).

**Headers:**
```
X-Internal-Service: orders-api
X-API-Key: internal-secret-key
```

**Request Body:**
```json
{
  "amount": 25000.00,
  "order_id": "ORD-2024-000790",
  "duration_seconds": 1800,
  "metadata": {
    "order_type": "buy",
    "crypto_symbol": "ETH",
    "estimated_quantity": 8.33
  }
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "lock_id": "LOCK-2024-000457",
    "amount_locked": 25000.00,
    "expires_at": "2024-01-15T11:10:00Z",
    "available_balance": 57457.50
  }
}
```

#### POST `/api/wallet/:userId/release/:lockId`
Libera fondos bloqueados (interno).

**Headers:**
```
X-Internal-Service: orders-api
X-API-Key: internal-secret-key
```

**Response (200):**
```json
{
  "success": true,
  "message": "Fondos liberados exitosamente",
  "data": {
    "lock_id": "LOCK-2024-000457",
    "amount_released": 25000.00,
    "new_available_balance": 82457.50
  }
}
```

#### POST `/api/wallet/:userId/execute/:lockId`
Ejecuta una transacción con fondos bloqueados (interno).

**Headers:**
```
X-Internal-Service: orders-api
X-API-Key: internal-secret-key
```

**Request Body:**
```json
{
  "final_amount": 24975.00,
  "fee": 25.00,
  "order_details": {
    "crypto_symbol": "ETH",
    "quantity": 8.325,
    "execution_price": 3000.00
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "transaction_id": "TXN-2024-000012348",
    "lock_id": "LOCK-2024-000457",
    "amount_debited": 25000.00,
    "new_balance": 57457.50,
    "lock_released": true
  }
}
```

### Administración

#### POST `/api/wallet/admin/reconcile`
Ejecuta reconciliación de saldos (admin only).

**Headers:**
```
Authorization: Bearer [admin-token]
```

**Request Body:**
```json
{
  "user_ids": [123, 124, 125],
  "full_scan": false
}
```

**Response (202):**
```json
{
  "success": true,
  "message": "Reconciliación iniciada",
  "data": {
    "job_id": "RECON-2024-000089",
    "users_to_process": 3,
    "estimated_time": "30 seconds"
  }
}
```

#### GET `/api/wallet/admin/audit/:userId`
Obtiene reporte de auditoría (admin only).

**Query Parameters:**
- `from`: Fecha inicial
- `to`: Fecha final
- `include_details`: Incluir detalles completos (true/false)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user_id": 123,
    "period": {
      "from": "2024-01-01",
      "to": "2024-01-15"
    },
    "summary": {
      "opening_balance": 100000.00,
      "closing_balance": 57457.50,
      "total_deposits": 15000.00,
      "total_withdrawals": 5000.00,
      "total_orders": 52542.50,
      "total_fees": 525.43,
      "transaction_count": 1547
    },
    "suspicious_activities": [],
    "balance_verification": {
      "calculated_balance": 57457.50,
      "recorded_balance": 57457.50,
      "match": true
    }
  }
}
```

## 🔐 Sistema Transaccional

### Transaction Manager
```go
// transaction_manager.go
package transaction

import (
    "context"
    "fmt"
    "github.com/shopspring/decimal"
    "go.mongodb.org/mongo-driver/mongo"
)

type TransactionManager struct {
    db           *mongo.Database
    lockManager  *locking.LockManager
    idempotency  *IdempotencyManager
}

type Transaction struct {
    ID              string
    IdempotencyKey  string
    Type            string
    Amount          decimal.Decimal
    WalletID        string
    UserID          int
}

func (tm *TransactionManager) ExecuteTransaction(ctx context.Context, tx *Transaction) error {
    // Check idempotency
    if result := tm.idempotency.Check(tx.IdempotencyKey); result != nil {
        return nil // Transaction already processed
    }
    
    // Start MongoDB session
    session, err := tm.db.Client().StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)
    
    // Execute in transaction
    err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
        if err := session.StartTransaction(); err != nil {
            return err
        }
        
        // Get wallet with lock
        wallet, err := tm.getWalletWithLock(sc, tx.WalletID)
        if err != nil {
            return err
        }
        
        // Validate transaction
        if err := tm.validateTransaction(wallet, tx); err != nil {
            session.AbortTransaction(sc)
            return err
        }
        
        // Update balance
        newBalance := tm.calculateNewBalance(wallet, tx)
        
        // Create transaction record
        txRecord := tm.createTransactionRecord(tx, wallet, newBalance)
        
        // Update wallet
        if err := tm.updateWallet(sc, wallet.ID, newBalance); err != nil {
            session.AbortTransaction(sc)
            return err
        }
        
        // Insert transaction record
        if err := tm.insertTransaction(sc, txRecord); err != nil {
            session.AbortTransaction(sc)
            return err
        }
        
        // Commit transaction
        if err := session.CommitTransaction(sc); err != nil {
            return err
        }
        
        // Mark idempotency key as processed
        tm.idempotency.Mark(tx.IdempotencyKey, txRecord.ID)
        
        return nil
    })
    
    return err
}

func (tm *TransactionManager) validateTransaction(wallet *Wallet, tx *Transaction) error {
    // Check wallet status
    if wallet.Status != "active" {
        return ErrWalletNotActive
    }
    
    // Check balance for debits
    if tx.Amount.IsNegative() {
        required := tx.Amount.Abs()
        if wallet.Balance.Available.LessThan(required) {
            return ErrInsufficientFunds
        }
    }
    
    // Check limits
    if err := tm.checkLimits(wallet, tx); err != nil {
        return err
    }
    
    return nil
}

func (tm *TransactionManager) checkLimits(wallet *Wallet, tx *Transaction) error {
    amount := tx.Amount.Abs()
    
    // Single transaction limit
    if amount.GreaterThan(wallet.Limits.SingleTransaction) {
        return ErrExceedsTransactionLimit
    }
    
    // Daily limits
    if tx.Type == "withdrawal" {
        dailyUsed := wallet.UsageToday.Withdrawn.Add(amount)
        if dailyUsed.GreaterThan(wallet.Limits.DailyWithdrawal) {
            return ErrExceedsDailyWithdrawalLimit
        }
    }
    
    if tx.Type == "deposit" {
        dailyUsed := wallet.UsageToday.Deposited.Add(amount)
        if dailyUsed.GreaterThan(wallet.Limits.DailyDeposit) {
            return ErrExceedsDailyDepositLimit
        }
    }
    
    return nil
}
```

### Lock Manager
```go
// lock_manager.go
package locking

import (
    "context"
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type LockManager struct {
    repo            *repositories.LockRepository
    timeoutHandler  *TimeoutHandler
}

type FundsLock struct {
    LockID      string
    WalletID    string
    Amount      decimal.Decimal
    OrderID     string
    ExpiresAt   time.Time
    Status      string
}

func (lm *LockManager) LockFunds(ctx context.Context, walletID string, amount decimal.Decimal, orderID string, duration time.Duration) (*FundsLock, error) {
    lock := &FundsLock{
        LockID:    fmt.Sprintf("LOCK-%s", uuid.New().String()),
        WalletID:  walletID,
        Amount:    amount,
        OrderID:   orderID,
        ExpiresAt: time.Now().Add(duration),
        Status:    "active",
    }
    
    // Atomic operation to lock funds
    err := lm.repo.WithTransaction(ctx, func(tx *mongo.SessionContext) error {
        // Get wallet
        wallet, err := lm.repo.GetWallet(*tx, walletID)
        if err != nil {
            return err
        }
        
        // Check available balance
        if wallet.Balance.Available.LessThan(amount) {
            return ErrInsufficientFunds
        }
        
        // Update wallet balance
        wallet.Balance.Available = wallet.Balance.Available.Sub(amount)
        wallet.Balance.Locked = wallet.Balance.Locked.Add(amount)
        
        // Save lock
        if err := lm.repo.CreateLock(*tx, lock); err != nil {
            return err
        }
        
        // Update wallet
        if err := lm.repo.UpdateWallet(*tx, wallet); err != nil {
            return err
        }
        
        return nil
    })
    
    if err != nil {
        return nil, err
    }
    
    // Schedule timeout handler
    lm.timeoutHandler.Schedule(lock.LockID, lock.ExpiresAt)
    
    return lock, nil
}

func