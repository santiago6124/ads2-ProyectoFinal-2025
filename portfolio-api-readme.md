# 💼 Portfolio API - Microservicio de Gestión de Portafolios

## 📋 Descripción

El microservicio **Portfolio API** es responsable del cálculo, análisis y gestión de los portafolios de inversión de los usuarios en CryptoSim. Proporciona métricas de rendimiento en tiempo real, análisis histórico, diversificación del portafolio y consume eventos de RabbitMQ para mantener sincronizados los holdings con las órdenes ejecutadas.

## 🎯 Responsabilidades

- **Gestión de Holdings**: Tracking de todas las posiciones de criptomonedas
- **Cálculo de Rendimiento**: P&L (Profit & Loss) en tiempo real y histórico
- **Análisis de Portafolio**: Diversificación, riesgo y métricas avanzadas
- **Snapshots Históricos**: Registro temporal del valor del portafolio
- **Sincronización Automática**: Consumer de RabbitMQ para órdenes ejecutadas
- **Métricas de Performance**: ROI, Sharpe Ratio, volatilidad del portafolio
- **Rebalanceo Sugerido**: Recomendaciones de optimización
- **Comparación con Mercado**: Benchmark contra índices de mercado

## 🏗️ Arquitectura

### Estructura del Proyecto
```
portfolio-api/
├── cmd/
│   └── main.go                        # Punto de entrada
├── internal/
│   ├── controllers/                   # Controladores HTTP
│   │   ├── portfolio_controller.go
│   │   ├── performance_controller.go
│   │   ├── holdings_controller.go
│   │   └── analytics_controller.go
│   ├── services/                      # Lógica de negocio
│   │   ├── portfolio_service.go
│   │   ├── calculation_service.go     # Motor de cálculos
│   │   ├── performance_service.go
│   │   ├── snapshot_service.go
│   │   ├── rebalancing_service.go
│   │   └── benchmark_service.go
│   ├── repositories/                  # Acceso a datos
│   │   ├── portfolio_repository.go
│   │   ├── snapshot_repository.go
│   │   └── mongodb_repository.go
│   ├── models/                        # Modelos de dominio
│   │   ├── portfolio.go
│   │   ├── holding.go
│   │   ├── performance.go
│   │   ├── snapshot.go
│   │   └── metrics.go
│   ├── dto/                           # Data Transfer Objects
│   │   ├── portfolio_response.go
│   │   ├── performance_dto.go
│   │   └── rebalancing_dto.go
│   ├── messaging/                     # RabbitMQ
│   │   ├── order_consumer.go
│   │   ├── event_processor.go
│   │   └── portfolio_updater.go
│   ├── calculator/                    # Motor de cálculos
│   │   ├── pnl_calculator.go          # Profit & Loss
│   │   ├── roi_calculator.go          # Return on Investment
│   │   ├── risk_calculator.go         # Risk metrics
│   │   ├── diversification.go
│   │   └── weighted_average.go
│   ├── analyzer/                      # Análisis avanzado
│   │   ├── portfolio_analyzer.go
│   │   ├── correlation_matrix.go
│   │   ├── sharpe_ratio.go
│   │   └── volatility.go
│   ├── clients/                       # Clientes internos
│   │   ├── market_client.go
│   │   ├── orders_client.go
│   │   └── users_client.go
│   ├── scheduler/                     # Tareas programadas
│   │   ├── snapshot_scheduler.go
│   │   ├── metrics_updater.go
│   │   └── cleanup_job.go
│   ├── middleware/                    # Middlewares
│   │   ├── auth_middleware.go
│   │   ├── cache_middleware.go
│   │   └── logging_middleware.go
│   └── config/                        # Configuración
│       └── config.go
├── pkg/
│   ├── utils/                         # Utilidades
│   │   ├── decimal.go
│   │   ├── percentage.go
│   │   └── response.go
│   ├── cache/                         # Cache
│   │   └── portfolio_cache.go
│   └── errors/                        # Manejo de errores
│       └── portfolio_errors.go
├── tests/                             # Tests
│   ├── unit/
│   │   ├── calculator_test.go
│   │   └── analyzer_test.go
│   ├── integration/
│   │   └── portfolio_flow_test.go
│   └── mocks/
│       └── repository_mock.go
├── scripts/                           # Scripts de utilidad
│   ├── recalculate_all.sh
│   └── migrate_data.sh
├── docs/                              # Documentación
│   ├── swagger.yaml
│   ├── metrics_guide.md
│   └── calculation_formulas.md
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Modelo de Datos

### Colección: portfolios (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  "total_value": NumberDecimal("125000.00"),
  "total_invested": NumberDecimal("100000.00"),
  "total_cash": NumberDecimal("25000.00"),
  "profit_loss": NumberDecimal("25000.00"),
  "profit_loss_percentage": NumberDecimal("25.00"),
  "currency": "USD",
  
  "holdings": [
    {
      "crypto_id": "bitcoin",
      "symbol": "BTC",
      "name": "Bitcoin",
      "quantity": NumberDecimal("0.5"),
      "average_buy_price": NumberDecimal("40000.00"),
      "total_invested": NumberDecimal("20000.00"),
      "current_price": NumberDecimal("45000.00"),
      "current_value": NumberDecimal("22500.00"),
      "profit_loss": NumberDecimal("2500.00"),
      "profit_loss_percentage": NumberDecimal("12.5"),
      "percentage_of_portfolio": NumberDecimal("18.0"),
      "first_purchase_date": ISODate("2024-01-01T10:00:00Z"),
      "last_purchase_date": ISODate("2024-01-10T15:30:00Z"),
      "transactions_count": 5
    },
    {
      "crypto_id": "ethereum",
      "symbol": "ETH",
      "name": "Ethereum",
      "quantity": NumberDecimal("10"),
      "average_buy_price": NumberDecimal("2500.00"),
      "total_invested": NumberDecimal("25000.00"),
      "current_price": NumberDecimal("3000.00"),
      "current_value": NumberDecimal("30000.00"),
      "profit_loss": NumberDecimal("5000.00"),
      "profit_loss_percentage": NumberDecimal("20.0"),
      "percentage_of_portfolio": NumberDecimal("24.0"),
      "first_purchase_date": ISODate("2024-01-05T12:00:00Z"),
      "last_purchase_date": ISODate("2024-01-12T09:15:00Z"),
      "transactions_count": 8
    }
  ],
  
  "performance": {
    "daily_change": NumberDecimal("1250.50"),
    "daily_change_percentage": NumberDecimal("1.01"),
    "weekly_change": NumberDecimal("5500.00"),
    "weekly_change_percentage": NumberDecimal("4.59"),
    "monthly_change": NumberDecimal("12000.00"),
    "monthly_change_percentage": NumberDecimal("10.61"),
    "yearly_change": NumberDecimal("25000.00"),
    "yearly_change_percentage": NumberDecimal("25.00"),
    "all_time_high": NumberDecimal("130000.00"),
    "all_time_high_date": ISODate("2024-01-14T16:00:00Z"),
    "all_time_low": NumberDecimal("95000.00"),
    "all_time_low_date": ISODate("2023-12-20T09:00:00Z"),
    "best_performing_asset": "ETH",
    "worst_performing_asset": "MATIC",
    "roi": NumberDecimal("25.00"),
    "annualized_return": NumberDecimal("30.50")
  },
  
  "risk_metrics": {
    "volatility_24h": NumberDecimal("0.042"),
    "volatility_7d": NumberDecimal("0.068"),
    "volatility_30d": NumberDecimal("0.125"),
    "sharpe_ratio": NumberDecimal("1.85"),
    "sortino_ratio": NumberDecimal("2.10"),
    "max_drawdown": NumberDecimal("-15.5"),
    "max_drawdown_date": ISODate("2024-01-08T14:00:00Z"),
    "beta": NumberDecimal("1.2"),
    "alpha": NumberDecimal("0.05"),
    "value_at_risk_95": NumberDecimal("-5000.00"),
    "conditional_var_95": NumberDecimal("-7500.00")
  },
  
  "diversification": {
    "holdings_count": 5,
    "concentration_index": NumberDecimal("0.35"),
    "herfindahl_index": NumberDecimal("0.2156"),
    "categories": {
      "Layer1": NumberDecimal("42.0"),
      "DeFi": NumberDecimal("35.0"),
      "Gaming": NumberDecimal("15.0"),
      "Other": NumberDecimal("8.0")
    },
    "largest_position_percentage": NumberDecimal("24.0"),
    "top_3_concentration": NumberDecimal("66.0")
  },
  
  "metadata": {
    "last_calculated": ISODate("2024-01-15T10:30:00Z"),
    "last_order_processed": ISODate("2024-01-15T10:25:00Z"),
    "calculation_version": "2.1.0",
    "needs_recalculation": false
  },
  
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_at": ISODate("2024-01-15T10:30:00Z")
}
```

### Colección: portfolio_snapshots (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439012"),
  "portfolio_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  "timestamp": ISODate("2024-01-15T00:00:00Z"),
  "interval": "daily", // "hourly" | "daily" | "weekly" | "monthly"
  
  "value": {
    "total": NumberDecimal("125000.00"),
    "invested": NumberDecimal("100000.00"),
    "profit_loss": NumberDecimal("25000.00"),
    "profit_loss_percentage": NumberDecimal("25.00")
  },
  
  "holdings_snapshot": [
    {
      "symbol": "BTC",
      "quantity": NumberDecimal("0.5"),
      "value": NumberDecimal("22500.00"),
      "price": NumberDecimal("45000.00"),
      "percentage": NumberDecimal("18.0")
    },
    {
      "symbol": "ETH",
      "quantity": NumberDecimal("10"),
      "value": NumberDecimal("30000.00"),
      "price": NumberDecimal("3000.00"),
      "percentage": NumberDecimal("24.0")
    }
  ],
  
  "metrics": {
    "volatility": NumberDecimal("0.068"),
    "sharpe_ratio": NumberDecimal("1.85"),
    "diversification_index": NumberDecimal("0.35")
  },
  
  "market_comparison": {
    "btc_performance": NumberDecimal("22.5"),
    "market_avg_performance": NumberDecimal("18.3"),
    "outperformance": NumberDecimal("4.2")
  }
}
```

### Índices MongoDB
```javascript
// Índices para optimización
db.portfolios.createIndex({ "user_id": 1 }, { unique: true })
db.portfolios.createIndex({ "updated_at": -1 })
db.portfolios.createIndex({ "metadata.needs_recalculation": 1 })

db.portfolio_snapshots.createIndex({ "user_id": 1, "timestamp": -1 })
db.portfolio_snapshots.createIndex({ "portfolio_id": 1, "interval": 1, "timestamp": -1 })
db.portfolio_snapshots.createIndex({ "timestamp": -1 }, { expireAfterSeconds: 7776000 }) // 90 días
```

## 🔌 API Endpoints

### Portafolio Principal

#### GET `/api/portfolio/:userId`
Obtiene el portafolio completo del usuario.

**Headers:**
```
Authorization: Bearer [token]
```

**Query Parameters:**
- `include_metrics`: Incluir métricas de riesgo (true/false) - default: true
- `currency`: Moneda de conversión (USD/EUR/BTC) - default: USD

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user_id": 123,
    "summary": {
      "total_value": 125000.00,
      "total_invested": 100000.00,
      "total_cash": 25000.00,
      "profit_loss": 25000.00,
      "profit_loss_percentage": 25.00,
      "currency": "USD"
    },
    "holdings": [
      {
        "symbol": "BTC",
        "name": "Bitcoin",
        "quantity": 0.5,
        "average_buy_price": 40000.00,
        "current_price": 45000.00,
        "current_value": 22500.00,
        "profit_loss": 2500.00,
        "profit_loss_percentage": 12.5,
        "percentage_of_portfolio": 18.0,
        "24h_change": 2.5
      },
      {
        "symbol": "ETH",
        "name": "Ethereum",
        "quantity": 10,
        "average_buy_price": 2500.00,
        "current_price": 3000.00,
        "current_value": 30000.00,
        "profit_loss": 5000.00,
        "profit_loss_percentage": 20.0,
        "percentage_of_portfolio": 24.0,
        "24h_change": 3.2
      }
    ],
    "allocation": {
      "crypto": 100000.00,
      "cash": 25000.00,
      "crypto_percentage": 80.0,
      "cash_percentage": 20.0
    },
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

#### GET `/api/portfolio/:userId/holdings`
Lista detallada de holdings actuales.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "holdings": [
      {
        "crypto_id": "bitcoin",
        "symbol": "BTC",
        "name": "Bitcoin",
        "quantity": 0.5,
        "average_buy_price": 40000.00,
        "total_invested": 20000.00,
        "current_price": 45000.00,
        "current_value": 22500.00,
        "profit_loss": 2500.00,
        "profit_loss_percentage": 12.5,
        "percentage_of_portfolio": 18.0,
        "transactions": {
          "total": 5,
          "buys": 5,
          "sells": 0,
          "first_purchase": "2024-01-01T10:00:00Z",
          "last_activity": "2024-01-10T15:30:00Z"
        },
        "cost_basis": [
          {
            "date": "2024-01-01T10:00:00Z",
            "quantity": 0.2,
            "price": 38000.00
          },
          {
            "date": "2024-01-10T15:30:00Z",
            "quantity": 0.3,
            "price": 41333.33
          }
        ]
      }
    ],
    "total_holdings": 5,
    "total_value": 100000.00
  }
}
```

### Performance y Métricas

#### GET `/api/portfolio/:userId/performance`
Obtiene métricas de rendimiento del portafolio.

**Query Parameters:**
- `period`: Período de análisis (24h, 7d, 30d, 1y, all) - default: 30d
- `compare_to`: Comparar con benchmark (market, btc, eth) - opcional

**Response (200):**
```json
{
  "success": true,
  "data": {
    "period": "30d",
    "performance": {
      "absolute_return": 12000.00,
      "percentage_return": 10.61,
      "annualized_return": 127.32,
      "time_weighted_return": 10.45,
      "money_weighted_return": 10.89
    },
    "comparison": {
      "portfolio_return": 10.61,
      "benchmark_return": 8.50,
      "alpha": 2.11,
      "tracking_error": 0.045,
      "information_ratio": 0.47
    },
    "risk_metrics": {
      "volatility": 0.125,
      "annualized_volatility": 0.433,
      "sharpe_ratio": 1.85,
      "sortino_ratio": 2.10,
      "calmar_ratio": 2.45,
      "max_drawdown": -15.5,
      "recovery_time_days": 8,
      "downside_deviation": 0.089,
      "upside_capture": 1.15,
      "downside_capture": 0.85
    },
    "best_performers": [
      {
        "symbol": "SOL",
        "return_percentage": 45.6,
        "contribution_to_portfolio": 4.56
      },
      {
        "symbol": "ETH",
        "return_percentage": 20.0,
        "contribution_to_portfolio": 4.80
      }
    ],
    "worst_performers": [
      {
        "symbol": "MATIC",
        "return_percentage": -12.3,
        "contribution_to_portfolio": -0.98
      }
    ]
  }
}
```

#### GET `/api/portfolio/:userId/history`
Histórico de valor del portafolio.

**Query Parameters:**
- `from`: Fecha inicial (YYYY-MM-DD)
- `to`: Fecha final (YYYY-MM-DD)
- `interval`: Intervalo de datos (hourly, daily, weekly) - default: daily

**Response (200):**
```json
{
  "success": true,
  "data": {
    "interval": "daily",
    "history": [
      {
        "date": "2024-01-01",
        "total_value": 100000.00,
        "profit_loss": 0,
        "daily_change": 0,
        "daily_change_percentage": 0
      },
      {
        "date": "2024-01-02",
        "total_value": 102500.00,
        "profit_loss": 2500.00,
        "daily_change": 2500.00,
        "daily_change_percentage": 2.5
      }
    ],
    "summary": {
      "start_value": 100000.00,
      "end_value": 125000.00,
      "total_change": 25000.00,
      "total_change_percentage": 25.00,
      "best_day": {
        "date": "2024-01-10",
        "value": 130000.00,
        "change": 8000.00
      },
      "worst_day": {
        "date": "2024-01-08",
        "value": 95000.00,
        "change": -10000.00
      }
    }
  }
}
```

### Análisis y Diversificación

#### GET `/api/portfolio/:userId/analysis`
Análisis completo del portafolio.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "diversification": {
      "score": 7.5,
      "holdings_count": 5,
      "concentration_index": 0.35,
      "herfindahl_index": 0.2156,
      "effective_holdings": 4.64,
      "recommendations": [
        {
          "type": "high_concentration",
          "message": "ETH representa 24% del portafolio. Considere diversificar.",
          "severity": "medium"
        }
      ]
    },
    "correlation_matrix": {
      "BTC_ETH": 0.85,
      "BTC_SOL": 0.72,
      "ETH_SOL": 0.78
    },
    "risk_assessment": {
      "risk_level": "moderate",
      "risk_score": 6.2,
      "var_95": -5000.00,
      "cvar_95": -7500.00,
      "stress_test": {
        "market_crash_20": -25000.00,
        "btc_crash_50": -11250.00
      }
    },
    "optimization_suggestions": [
      {
        "action": "rebalance",
        "from": "ETH",
        "to": "Stablecoins",
        "amount_percentage": 5,
        "expected_risk_reduction": 0.8,
        "expected_return_impact": -0.2
      }
    ]
  }
}
```

#### POST `/api/portfolio/:userId/snapshot`
Crea un snapshot manual del portafolio.

**Headers:**
```
Authorization: Bearer [token]
```

**Request Body:**
```json
{
  "note": "Antes de rebalanceo mensual",
  "tags": ["rebalancing", "monthly"]
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Snapshot creado exitosamente",
  "data": {
    "snapshot_id": "507f1f77bcf86cd799439013",
    "timestamp": "2024-01-15T10:30:00Z",
    "total_value": 125000.00,
    "note": "Antes de rebalanceo mensual"
  }
}
```

### Rebalanceo

#### GET `/api/portfolio/:userId/rebalancing`
Obtiene sugerencias de rebalanceo.

**Query Parameters:**
- `strategy`: Estrategia de rebalanceo (equal_weight, market_cap, custom) - default: equal_weight
- `threshold`: Umbral de desviación para rebalanceo (%) - default: 5

**Response (200):**
```json
{
  "success": true,
  "data": {
    "current_allocation": {
      "BTC": 18.0,
      "ETH": 24.0,
      "SOL": 20.0,
      "MATIC": 18.0,
      "ADA": 20.0
    },
    "target_allocation": {
      "BTC": 20.0,
      "ETH": 20.0,
      "SOL": 20.0,
      "MATIC": 20.0,
      "ADA": 20.0
    },
    "rebalancing_actions": [
      {
        "action": "sell",
        "symbol": "ETH",
        "quantity": 1.33,
        "value": 4000.00,
        "reason": "Overweight by 4%"
      },
      {
        "action": "buy",
        "symbol": "BTC",
        "quantity": 0.044,
        "value": 2000.00,
        "reason": "Underweight by 2%"
      }
    ],
    "estimated_cost": 25.00,
    "expected_improvement": {
      "risk_reduction": 0.015,
      "diversification_increase": 0.08
    }
  }
}
```

## 📊 Motor de Cálculos

### PnL Calculator
```go
// pnl_calculator.go
package calculator

import (
    "math"
    "github.com/shopspring/decimal"
)

type PnLCalculator struct {
    marketClient *clients.MarketClient
}

type PnLResult struct {
    TotalValue           decimal.Decimal
    TotalInvested        decimal.Decimal
    RealizedPnL          decimal.Decimal
    UnrealizedPnL        decimal.Decimal
    TotalPnL             decimal.Decimal
    PnLPercentage        decimal.Decimal
    DailyPnL             decimal.Decimal
    DailyPnLPercentage   decimal.Decimal
}

func (calc *PnLCalculator) Calculate(portfolio *models.Portfolio) (*PnLResult, error) {
    result := &PnLResult{
        TotalInvested: decimal.Zero,
        TotalValue:    decimal.Zero,
    }
    
    // Calculate for each holding
    for _, holding := range portfolio.Holdings {
        currentPrice, err := calc.marketClient.GetPrice(holding.Symbol)
        if err != nil {
            return nil, err
        }
        
        // Current value
        holdingValue := holding.Quantity.Mul(currentPrice)
        result.TotalValue = result.TotalValue.Add(holdingValue)
        
        // Total invested
        invested := holding.Quantity.Mul(holding.AverageBuyPrice)
        result.TotalInvested = result.TotalInvested.Add(invested)
        
        // Unrealized PnL for this holding
        holdingPnL := holdingValue.Sub(invested)
        result.UnrealizedPnL = result.UnrealizedPnL.Add(holdingPnL)
    }
    
    // Add cash to total value
    result.TotalValue = result.TotalValue.Add(portfolio.TotalCash)
    
    // Calculate total PnL
    result.TotalPnL = result.RealizedPnL.Add(result.UnrealizedPnL)
    
    // Calculate percentage
    if result.TotalInvested.IsPositive() {
        result.PnLPercentage = result.TotalPnL.Div(result.TotalInvested).Mul(decimal.NewFromInt(100))
    }
    
    // Calculate daily changes
    result.DailyPnL = calc.calculateDailyChange(portfolio)
    if portfolio.YesterdayValue.IsPositive() {
        result.DailyPnLPercentage = result.DailyPnL.Div(portfolio.YesterdayValue).Mul(decimal.NewFromInt(100))
    }
    
    return result, nil
}

func (calc *PnLCalculator) CalculateCostBasis(transactions []models.Transaction) decimal.Decimal {
    // FIFO (First In, First Out) calculation
    var queue []models.Transaction
    totalCost := decimal.Zero
    
    for _, tx := range transactions {
        if tx.Type == "buy" {
            queue = append(queue, tx)
        } else if tx.Type == "sell" {
            remaining := tx.Quantity
            
            for len(queue) > 0 && remaining.IsPositive() {
                if queue[0].Quantity.LessThanOrEqual(remaining) {
                    remaining = remaining.Sub(queue[0].Quantity)
                    queue = queue[1:]
                } else {
                    queue[0].Quantity = queue[0].Quantity.Sub(remaining)
                    remaining = decimal.Zero
                }
            }
        }
    }
    
    // Calculate average cost of remaining holdings
    totalQuantity := decimal.Zero
    for _, tx := range queue {
        totalCost = totalCost.Add(tx.Quantity.Mul(tx.Price))
        totalQuantity = totalQuantity.Add(tx.Quantity)
    }
    
    if totalQuantity.IsPositive() {
        return totalCost.Div(totalQuantity)
    }
    
    return decimal.Zero
}
```

### Risk Calculator
```go
// risk_calculator.go
package calculator

type RiskCalculator struct {
    historicalData *repositories.HistoricalRepository
}

func (rc *RiskCalculator) CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
    if len(returns) == 0 {
        return 0
    }
    
    // Calculate average return
    avgReturn := mean(returns)
    
    // Calculate standard deviation
    stdDev := standardDeviation(returns)
    
    if stdDev == 0 {
        return 0
    }
    
    // Sharpe Ratio = (Portfolio Return - Risk Free Rate) / Standard Deviation
    return (avgReturn - riskFreeRate) / stdDev
}

func (rc *RiskCalculator) CalculateValueAtRisk(portfolio *models.Portfolio, confidence float64) decimal.Decimal {
    // Get historical returns
    returns := rc.getHistoricalReturns(portfolio, 30) // Last 30 days
    
    // Sort returns
    sort.Float64s(returns)
    
    // Calculate VaR at given confidence level
    index := int(math.Floor((1 - confidence/100) * float64(len(returns))))
    
    if index < len(returns) {
        varValue := returns[index]
        return decimal.NewFromFloat(varValue).Mul(portfolio.TotalValue)
    }
    
    return decimal.Zero
}

func (rc *RiskCalculator) CalculateMaxDrawdown(values []float64) (float64, int) {
    if len(values) == 0 {
        return 0, 0
    }
    
    maxDrawdown := 0.0
    maxDrawdownIndex := 0
    peak := values[0]
    
    for i, value := range values {
        if value > peak {
            peak = value
        }
        
        drawdown := (peak - value) / peak
        if drawdown > maxDrawdown {
            maxDrawdown = drawdown
            maxDrawdownIndex = i
        }
    }
    
    return maxDrawdown * 100, maxDrawdownIndex //