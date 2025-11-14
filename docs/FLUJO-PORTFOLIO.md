# Flujo Técnico: Portfolio y Análisis de Inversiones

## Descripción General

Este documento describe el sistema de gestión de portfolios, el cálculo de 30+ métricas financieras avanzadas, y la actualización automática mediante eventos de RabbitMQ.

## Arquitectura Involucrada

```
┌───────────┐    ┌──────────────┐    ┌────────────────┐
│Orders API │───>│  RabbitMQ    │───>│ Portfolio API  │
│  :8002    │    │orders.events │    │    :8005       │
└───────────┘    └──────────────┘    └────────┬───────┘
                                              │
                   ┌──────────────────────────┼─────────────────┐
                   │                          │                  │
                   ▼                          ▼                  ▼
            ┌────────────┐             ┌──────────┐      ┌────────────┐
            │Market Data │             │ MongoDB  │      │ Users API  │
            │    API     │             │(Portfolio│      │   :8001    │
            │   :8004    │             │   DB)    │      │            │
            └────────────┘             └──────────┘      └────────────┘
                   ▲                                            │
                   └────────────────────────────────────────────┘
                              (Balance Request/Response)
```

---

## 1. OBTENER PORTFOLIO

### Endpoint
```
GET /api/portfolios/:userId
Authorization: Bearer {JWT_TOKEN}
```

### Proceso Paso a Paso

#### 1.1 Cliente solicita portfolio
```bash
curl http://localhost:8005/api/portfolios/123 \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

#### 1.2 Middleware JWT
```go
// portfolio-api/middleware/auth.go
token := extractTokenFromHeader(c.GetHeader("Authorization"))
claims := parseJWT(token)
userIDFromToken := claims["user_id"]  // 123

// Verificar que coincida con el parámetro
userIDParam := c.Param("userId")
if userIDFromToken != userIDParam {
    c.JSON(403, gin.H{"error": "Unauthorized"})
    return
}

c.Set("user_id", userIDFromToken)
c.Next()
```

#### 1.3 Buscar portfolio en MongoDB
```go
// portfolio-api/handlers/portfolio_handler.go
func (h *PortfolioHandler) GetPortfolio(c *gin.Context) {
    userID, _ := strconv.ParseInt(c.Param("userId"), 10, 64)

    portfolio, err := h.portfolioService.GetPortfolio(c.Request.Context(), userID)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            c.JSON(404, gin.H{"error": "Portfolio not found"})
            return
        }
        c.JSON(500, gin.H{"error": "Failed to fetch portfolio"})
        return
    }

    // Verificar si necesita recalcular
    if portfolio.Metadata.NeedsRecalculation {
        portfolio, _ = h.portfolioService.RecalculatePortfolio(c.Request.Context(), userID)
    }

    c.JSON(200, gin.H{
        "status": "success",
        "data": gin.H{
            "portfolio": portfolio,
        },
    })
}
```

#### 1.4 Response del Portfolio

```json
{
  "status": "success",
  "data": {
    "portfolio": {
      "id": "673b5f8a9e1234567890abce",
      "user_id": 123,
      "total_value": "1051.50",
      "total_invested": "1000.00",
      "profit_loss": "51.50",
      "profit_loss_percentage": 5.15,
      "currency": "USD",
      "holdings": [
        {
          "symbol": "BTC",
          "quantity": "0.001",
          "average_buy_price": "50000.00",
          "current_price": "51500.00",
          "current_value": "51.50",
          "profit_loss": "1.50",
          "profit_loss_percentage": 3.0,
          "percentage_of_portfolio": 4.9,
          "first_purchase_date": "2025-11-14T10:30:05Z",
          "last_purchase_date": "2025-11-14T10:30:05Z",
          "transactions_count": 1
        },
        {
          "symbol": "ETH",
          "quantity": "0.5",
          "average_buy_price": "2000.00",
          "current_price": "2100.00",
          "current_value": "1050.00",
          "profit_loss": "50.00",
          "profit_loss_percentage": 5.0,
          "percentage_of_portfolio": 99.86,
          "first_purchase_date": "2025-11-13T15:20:00Z",
          "last_purchase_date": "2025-11-13T15:20:00Z",
          "transactions_count": 1
        }
      ],
      "performance": {
        "daily_change": "15.25",
        "daily_change_percentage": 1.47,
        "weekly_change": "51.50",
        "weekly_change_percentage": 5.15,
        "monthly_change": "51.50",
        "monthly_change_percentage": 5.15,
        "yearly_change": "51.50",
        "yearly_change_percentage": 5.15,
        "all_time_high": "1051.50",
        "all_time_low": "1000.00",
        "roi": 5.15,
        "annualized_return": 62.0,
        "time_weighted_return": 5.15,
        "money_weighted_return": 5.15,
        "best_performing_asset": "ETH",
        "worst_performing_asset": "BTC"
      },
      "risk_metrics": {
        "volatility_24h": 2.5,
        "volatility_7d": 8.3,
        "volatility_30d": 15.7,
        "sharpe_ratio": 1.85,
        "sortino_ratio": 2.34,
        "calmar_ratio": 3.12,
        "max_drawdown": -5.2,
        "max_drawdown_percentage": -5.2,
        "current_drawdown": 0.0,
        "beta": 0.95,
        "alpha": 2.3,
        "value_at_risk_95": -25.50,
        "conditional_var_95": -32.75,
        "downside_deviation": 3.2
      },
      "diversification": {
        "herfindahl_index": 0.995,
        "concentration_index": 0.9986,
        "effective_holdings": 1.005,
        "largest_position_percentage": 99.86,
        "top_3_concentration": 100.0,
        "categories": {
          "Smart Contract Platform": 99.86,
          "Store of Value": 4.9
        }
      },
      "metadata": {
        "last_calculated": "2025-11-14T11:30:00Z",
        "last_order_processed": "673b5f8a9e1234567890abcd",
        "needs_recalculation": false,
        "version": 1
      },
      "created_at": "2025-11-14T10:30:05Z",
      "updated_at": "2025-11-14T11:30:00Z"
    }
  }
}
```

---

## 2. ACTUALIZACIÓN AUTOMÁTICA VÍA RABBITMQ

### Consumer de eventos de órdenes

#### 2.1 Inicializar consumer
```go
// portfolio-api/messaging/consumer.go
func (c *PortfolioConsumer) Start() {
    // Declarar exchange (si no existe)
    c.channel.ExchangeDeclare(
        "orders.events",  // Name
        "topic",          // Type
        true,             // Durable
        false, false, false, nil,
    )

    // Declarar queue dedicada para portfolio
    queue, _ := c.channel.QueueDeclare(
        "portfolio.updates",  // Name
        true,                 // Durable
        false, false, false, nil,
    )

    // Bind SOLO a orders.executed
    c.channel.QueueBind(
        queue.Name,
        "orders.executed",  // Routing key
        "orders.events",
        false, nil,
    )

    // Consumir mensajes
    msgs, _ := c.channel.Consume(
        queue.Name,
        "portfolio-consumer",
        false,  // Manual ack
        false, false, false, nil,
    )

    // Procesar en goroutine
    go func() {
        for msg := range msgs {
            err := c.handleOrderExecuted(msg)
            if err != nil {
                log.Error("Failed to process order", err)
                msg.Nack(false, true)  // Requeue
            } else {
                msg.Ack(false)
            }
        }
    }()
}
```

#### 2.2 Procesar evento de orden ejecutada

```go
func (c *PortfolioConsumer) handleOrderExecuted(msg amqp.Delivery) error {
    var event OrderEvent
    json.Unmarshal(msg.Body, &event)

    log.Info("Processing order event",
        "order_id", event.OrderID,
        "user_id", event.UserID,
        "type", event.Type,
        "crypto", event.CryptoSymbol,
    )

    // Actualizar portfolio del usuario
    return c.portfolioService.ProcessOrderEvent(context.Background(), event)
}
```

#### 2.3 Service Layer - Procesar orden

```go
// portfolio-api/services/portfolio_service.go
func (ps *PortfolioService) ProcessOrderEvent(ctx context.Context, event OrderEvent) error {
    // 1. Buscar o crear portfolio
    portfolio, err := ps.GetOrCreatePortfolio(ctx, event.UserID)
    if err != nil {
        return err
    }

    // 2. Obtener precio actual del mercado
    currentPrice, err := ps.marketDataClient.GetCurrentPrice(event.CryptoSymbol)
    if err != nil {
        return err
    }

    // 3. Actualizar holdings según tipo de orden
    if event.Type == "buy" {
        ps.addToHoldings(portfolio, event, currentPrice)
    } else if event.Type == "sell" {
        ps.removeFromHoldings(portfolio, event, currentPrice)
    }

    // 4. Recalcular todas las métricas
    ps.calculateAllMetrics(portfolio, currentPrice)

    // 5. Actualizar metadata
    portfolio.Metadata.LastCalculated = time.Now()
    portfolio.Metadata.LastOrderProcessed = event.OrderID
    portfolio.Metadata.NeedsRecalculation = false
    portfolio.UpdatedAt = time.Now()

    // 6. Guardar en MongoDB
    filter := bson.M{"user_id": portfolio.UserID}
    update := bson.M{"$set": portfolio}
    _, err = ps.collection.UpdateOne(ctx, filter, update)

    return err
}
```

#### 2.4 Agregar a holdings (BUY)

```go
func (ps *PortfolioService) addToHoldings(portfolio *Portfolio, event OrderEvent, currentPrice decimal.Decimal) {
    quantity := decimal.RequireFromString(event.Quantity)
    price := decimal.RequireFromString(event.Price)
    totalCost := decimal.RequireFromString(event.TotalAmount)

    // Buscar holding existente
    holdingIndex := -1
    for i, h := range portfolio.Holdings {
        if h.Symbol == event.CryptoSymbol {
            holdingIndex = i
            break
        }
    }

    if holdingIndex == -1 {
        // Crear nuevo holding
        holding := Holding{
            Symbol:            event.CryptoSymbol,
            Quantity:          quantity,
            AverageBuyPrice:   price,
            CurrentPrice:      currentPrice,
            FirstPurchaseDate: time.Now(),
            LastPurchaseDate:  time.Now(),
            TransactionsCount: 1,
            CostBasis: []CostBasisEntry{
                {
                    Quantity:  quantity,
                    Price:     price,
                    Date:      time.Now(),
                    TotalCost: totalCost,
                },
            },
        }

        // Calcular valores actuales
        holding.CurrentValue = holding.Quantity.Mul(currentPrice)
        holding.ProfitLoss = holding.CurrentValue.Sub(totalCost)
        holding.ProfitLossPercentage = holding.ProfitLoss.Div(totalCost).Mul(decimal.NewFromInt(100))

        portfolio.Holdings = append(portfolio.Holdings, holding)
    } else {
        // Actualizar holding existente
        holding := &portfolio.Holdings[holdingIndex]

        // Calcular nuevo precio promedio ponderado
        oldCost := holding.Quantity.Mul(holding.AverageBuyPrice)
        newTotalCost := oldCost.Add(totalCost)
        newTotalQuantity := holding.Quantity.Add(quantity)
        newAvgPrice := newTotalCost.Div(newTotalQuantity)

        holding.Quantity = newTotalQuantity
        holding.AverageBuyPrice = newAvgPrice
        holding.CurrentPrice = currentPrice
        holding.LastPurchaseDate = time.Now()
        holding.TransactionsCount++

        // Agregar entrada de cost basis
        holding.CostBasis = append(holding.CostBasis, CostBasisEntry{
            Quantity:  quantity,
            Price:     price,
            Date:      time.Now(),
            TotalCost: totalCost,
        })

        // Recalcular valores actuales
        holding.CurrentValue = holding.Quantity.Mul(currentPrice)
        investedAmount := holding.Quantity.Mul(holding.AverageBuyPrice)
        holding.ProfitLoss = holding.CurrentValue.Sub(investedAmount)
        holding.ProfitLossPercentage = holding.ProfitLoss.Div(investedAmount).Mul(decimal.NewFromInt(100))
    }
}
```

#### 2.5 Remover de holdings (SELL) - FIFO

```go
func (ps *PortfolioService) removeFromHoldings(portfolio *Portfolio, event OrderEvent, currentPrice decimal.Decimal) {
    quantity := decimal.RequireFromString(event.Quantity)

    // Buscar holding
    holdingIndex := -1
    for i, h := range portfolio.Holdings {
        if h.Symbol == event.CryptoSymbol {
            holdingIndex = i
            break
        }
    }

    if holdingIndex == -1 {
        log.Error("Trying to sell crypto not in holdings", "symbol", event.CryptoSymbol)
        return
    }

    holding := &portfolio.Holdings[holdingIndex]

    // Aplicar FIFO (First In First Out) en cost basis
    remainingToSell := quantity
    totalCostBasis := decimal.Zero

    for i := 0; i < len(holding.CostBasis) && remainingToSell.IsPositive(); {
        entry := &holding.CostBasis[i]

        if entry.Quantity.LessThanOrEqual(remainingToSell) {
            // Vender toda esta entrada
            totalCostBasis = totalCostBasis.Add(entry.TotalCost)
            remainingToSell = remainingToSell.Sub(entry.Quantity)

            // Eliminar entrada del array
            holding.CostBasis = append(holding.CostBasis[:i], holding.CostBasis[i+1:]...)
        } else {
            // Vender parcialmente esta entrada
            portionSold := remainingToSell.Div(entry.Quantity)
            costOfPortion := entry.TotalCost.Mul(portionSold)
            totalCostBasis = totalCostBasis.Add(costOfPortion)

            // Actualizar entrada
            entry.Quantity = entry.Quantity.Sub(remainingToSell)
            entry.TotalCost = entry.TotalCost.Sub(costOfPortion)

            remainingToSell = decimal.Zero
            i++
        }
    }

    // Actualizar cantidad total
    holding.Quantity = holding.Quantity.Sub(quantity)
    holding.TransactionsCount++
    holding.CurrentPrice = currentPrice

    // Si vendió todo, eliminar holding
    if holding.Quantity.IsZero() {
        portfolio.Holdings = append(
            portfolio.Holdings[:holdingIndex],
            portfolio.Holdings[holdingIndex+1:]...,
        )
        return
    }

    // Recalcular precio promedio basado en cost basis restante
    if len(holding.CostBasis) > 0 {
        totalCost := decimal.Zero
        totalQty := decimal.Zero
        for _, entry := range holding.CostBasis {
            totalCost = totalCost.Add(entry.TotalCost)
            totalQty = totalQty.Add(entry.Quantity)
        }
        holding.AverageBuyPrice = totalCost.Div(totalQty)
    }

    // Recalcular valores actuales
    holding.CurrentValue = holding.Quantity.Mul(currentPrice)
    investedAmount := holding.Quantity.Mul(holding.AverageBuyPrice)
    holding.ProfitLoss = holding.CurrentValue.Sub(investedAmount)
    holding.ProfitLossPercentage = holding.ProfitLoss.Div(investedAmount).Mul(decimal.NewFromInt(100))
}
```

---

## 3. CÁLCULO DE MÉTRICAS (30+)

### 3.1 Métricas de Performance

```go
func (ps *PortfolioService) calculatePerformanceMetrics(portfolio *Portfolio) Performance {
    perf := Performance{}

    // Obtener histórico de snapshots (guardados por scheduler)
    snapshots, _ := ps.getPortfolioSnapshots(portfolio.UserID)

    // Daily change
    if len(snapshots) > 0 {
        yesterdayValue := snapshots[len(snapshots)-1].TotalValue
        perf.DailyChange = portfolio.TotalValue.Sub(yesterdayValue)
        perf.DailyChangePercentage = perf.DailyChange.Div(yesterdayValue).Mul(decimal.NewFromInt(100))
    }

    // Weekly change (7 días atrás)
    if len(snapshots) >= 7 {
        weekAgoValue := snapshots[len(snapshots)-7].TotalValue
        perf.WeeklyChange = portfolio.TotalValue.Sub(weekAgoValue)
        perf.WeeklyChangePercentage = perf.WeeklyChange.Div(weekAgoValue).Mul(decimal.NewFromInt(100))
    }

    // Monthly change (30 días atrás)
    if len(snapshots) >= 30 {
        monthAgoValue := snapshots[len(snapshots)-30].TotalValue
        perf.MonthlyChange = portfolio.TotalValue.Sub(monthAgoValue)
        perf.MonthlyChangePercentage = perf.MonthlyChange.Div(monthAgoValue).Mul(decimal.NewFromInt(100))
    }

    // All time high/low
    perf.AllTimeHigh = portfolio.TotalValue
    perf.AllTimeLow = portfolio.TotalInvested
    for _, snap := range snapshots {
        if snap.TotalValue.GreaterThan(perf.AllTimeHigh) {
            perf.AllTimeHigh = snap.TotalValue
        }
        if snap.TotalValue.LessThan(perf.AllTimeLow) {
            perf.AllTimeLow = snap.TotalValue
        }
    }

    // ROI (Return on Investment)
    perf.ROI = portfolio.ProfitLoss.Div(portfolio.TotalInvested).Mul(decimal.NewFromInt(100))

    // Annualized Return
    daysInvested := time.Since(portfolio.CreatedAt).Hours() / 24
    if daysInvested > 0 {
        dailyReturn := portfolio.ProfitLoss.Div(portfolio.TotalInvested)
        perf.AnnualizedReturn = dailyReturn.Mul(decimal.NewFromFloat(365.0 / daysInvested)).Mul(decimal.NewFromInt(100))
    }

    // Time Weighted Return (TWR)
    perf.TimeWeightedReturn = ps.calculateTWR(snapshots)

    // Money Weighted Return (MWR / IRR)
    perf.MoneyWeightedReturn = ps.calculateMWR(portfolio)

    // Best/Worst performing asset
    var bestAsset, worstAsset *Holding
    for i := range portfolio.Holdings {
        h := &portfolio.Holdings[i]
        if bestAsset == nil || h.ProfitLossPercentage > bestAsset.ProfitLossPercentage {
            bestAsset = h
        }
        if worstAsset == nil || h.ProfitLossPercentage < worstAsset.ProfitLossPercentage {
            worstAsset = h
        }
    }
    if bestAsset != nil {
        perf.BestPerformingAsset = bestAsset.Symbol
    }
    if worstAsset != nil {
        perf.WorstPerformingAsset = worstAsset.Symbol
    }

    return perf
}
```

### 3.2 Métricas de Riesgo

```go
func (ps *PortfolioService) calculateRiskMetrics(portfolio *Portfolio, snapshots []PortfolioSnapshot) RiskMetrics {
    risk := RiskMetrics{}

    // Volatilidad (desviación estándar de returns)
    returns := []float64{}
    for i := 1; i < len(snapshots); i++ {
        ret, _ := snapshots[i].TotalValue.Sub(snapshots[i-1].TotalValue).Div(snapshots[i-1].TotalValue).Float64()
        returns = append(returns, ret)
    }

    risk.Volatility24h = ps.calculateVolatility(returns[max(0, len(returns)-24):])
    risk.Volatility7d = ps.calculateVolatility(returns[max(0, len(returns)-168):])  // 7*24 horas
    risk.Volatility30d = ps.calculateVolatility(returns[max(0, len(returns)-720):])  // 30*24 horas

    // Sharpe Ratio = (Return - RiskFreeRate) / Volatility
    // Asumimos risk-free rate = 3% anual = 0.03/365 diario
    riskFreeRate := 0.03 / 365
    avgReturn := ps.mean(returns)
    if risk.Volatility24h > 0 {
        risk.SharpeRatio = (avgReturn - riskFreeRate) / risk.Volatility24h
    }

    // Sortino Ratio = (Return - RiskFreeRate) / DownsideDeviation
    downsideReturns := []float64{}
    for _, r := range returns {
        if r < 0 {
            downsideReturns = append(downsideReturns, r)
        }
    }
    risk.DownsideDeviation = ps.calculateVolatility(downsideReturns)
    if risk.DownsideDeviation > 0 {
        risk.SortinoRatio = (avgReturn - riskFreeRate) / risk.DownsideDeviation
    }

    // Maximum Drawdown
    maxDrawdown := 0.0
    peak := snapshots[0].TotalValue
    for _, snap := range snapshots {
        if snap.TotalValue.GreaterThan(peak) {
            peak = snap.TotalValue
        }
        drawdown, _ := snap.TotalValue.Sub(peak).Div(peak).Float64()
        if drawdown < maxDrawdown {
            maxDrawdown = drawdown
        }
    }
    risk.MaxDrawdown = maxDrawdown
    risk.MaxDrawdownPercentage = maxDrawdown * 100

    // Calmar Ratio = AnnualizedReturn / |MaxDrawdown|
    annualizedReturn, _ := portfolio.Performance.AnnualizedReturn.Float64()
    if math.Abs(maxDrawdown) > 0 {
        risk.CalmarRatio = annualizedReturn / math.Abs(maxDrawdown)
    }

    // Current Drawdown
    currentDrawdown, _ := portfolio.TotalValue.Sub(portfolio.Performance.AllTimeHigh).Div(portfolio.Performance.AllTimeHigh).Float64()
    risk.CurrentDrawdown = currentDrawdown

    // Beta (respecto al mercado - BTC como proxy)
    risk.Beta = ps.calculateBeta(snapshots)

    // Alpha (excess return)
    marketReturn := ps.getMarketReturn()  // BTC return
    expectedReturn := riskFreeRate + risk.Beta*(marketReturn-riskFreeRate)
    risk.Alpha = avgReturn - expectedReturn

    // Value at Risk (VaR) 95%
    // Pérdida máxima esperada con 95% de confianza
    sortedReturns := make([]float64, len(returns))
    copy(sortedReturns, returns)
    sort.Float64s(sortedReturns)
    varIndex := int(float64(len(sortedReturns)) * 0.05)
    if varIndex < len(sortedReturns) {
        varReturn := sortedReturns[varIndex]
        risk.ValueAtRisk95, _ = portfolio.TotalValue.Mul(decimal.NewFromFloat(varReturn)).Float64()
    }

    // Conditional VaR (CVaR) - Expected Shortfall
    // Promedio de pérdidas peores que VaR
    if varIndex > 0 {
        worstReturns := sortedReturns[:varIndex]
        avgWorstReturn := ps.mean(worstReturns)
        risk.ConditionalVar95, _ = portfolio.TotalValue.Mul(decimal.NewFromFloat(avgWorstReturn)).Float64()
    }

    return risk
}
```

### 3.3 Métricas de Diversificación

```go
func (ps *PortfolioService) calculateDiversificationMetrics(portfolio *Portfolio) Diversification {
    div := Diversification{}

    // Calcular porcentajes
    totalValue := portfolio.TotalValue
    for i := range portfolio.Holdings {
        h := &portfolio.Holdings[i]
        h.PercentageOfPortfolio, _ = h.CurrentValue.Div(totalValue).Mul(decimal.NewFromInt(100)).Float64()
    }

    // Herfindahl Index (HHI) - Concentración
    // HHI = sum(wi^2) donde wi = weight del asset i
    // HHI = 1 (monopolio), HHI cercano a 0 (muy diversificado)
    hhi := 0.0
    for _, h := range portfolio.Holdings {
        weight := h.PercentageOfPortfolio / 100.0
        hhi += weight * weight
    }
    div.HerfindahlIndex = hhi

    // Concentration Index (similar a HHI)
    div.ConcentrationIndex = hhi

    // Effective Number of Holdings
    // ENH = 1 / HHI
    if hhi > 0 {
        div.EffectiveHoldings = 1.0 / hhi
    }

    // Largest position percentage
    if len(portfolio.Holdings) > 0 {
        largest := 0.0
        for _, h := range portfolio.Holdings {
            if h.PercentageOfPortfolio > largest {
                largest = h.PercentageOfPortfolio
            }
        }
        div.LargestPositionPercentage = largest
    }

    // Top 3 concentration
    if len(portfolio.Holdings) >= 3 {
        percentages := []float64{}
        for _, h := range portfolio.Holdings {
            percentages = append(percentages, h.PercentageOfPortfolio)
        }
        sort.Float64s(percentages)
        top3 := percentages[len(percentages)-1] + percentages[len(percentages)-2] + percentages[len(percentages)-3]
        div.Top3Concentration = top3
    } else {
        div.Top3Concentration = 100.0
    }

    // Categories breakdown (Smart Contract, Store of Value, etc.)
    div.Categories = ps.categorizeHoldings(portfolio.Holdings)

    return div
}
```

---

## 4. SCHEDULER PARA RECALCULACIÓN PERIÓDICA

### CRON Job cada 15 minutos

```go
// portfolio-api/scheduler/scheduler.go
func (s *Scheduler) Start() {
    // Configurar cron: cada 15 minutos
    c := cron.New()
    c.AddFunc("0 */15 * * * *", func() {
        s.recalculateAllPortfolios()
    })
    c.Start()
}

func (s *Scheduler) recalculateAllPortfolios() {
    ctx := context.Background()

    // Obtener todos los portfolios
    cursor, err := s.collection.Find(ctx, bson.M{})
    if err != nil {
        log.Error("Failed to fetch portfolios", err)
        return
    }

    var portfolios []Portfolio
    cursor.All(ctx, &portfolios)

    log.Info("Starting scheduled recalculation", "count", len(portfolios))

    // Procesar en paralelo con goroutines
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10)  // Máximo 10 concurrentes

    for _, p := range portfolios {
        wg.Add(1)
        semaphore <- struct{}{}  // Acquire

        go func(portfolio Portfolio) {
            defer wg.Done()
            defer func() { <-semaphore }()  // Release

            // Obtener precios actuales de todos los holdings
            for i := range portfolio.Holdings {
                h := &portfolio.Holdings[i]
                currentPrice, err := s.marketDataClient.GetCurrentPrice(h.Symbol)
                if err != nil {
                    log.Error("Failed to get price", "symbol", h.Symbol, "error", err)
                    continue
                }
                h.CurrentPrice = currentPrice
            }

            // Recalcular métricas
            s.portfolioService.calculateAllMetrics(&portfolio, decimal.Zero)

            // Guardar snapshot (para análisis histórico)
            snapshot := PortfolioSnapshot{
                UserID:        portfolio.UserID,
                TotalValue:    portfolio.TotalValue,
                TotalInvested: portfolio.TotalInvested,
                ProfitLoss:    portfolio.ProfitLoss,
                Timestamp:     time.Now(),
            }
            s.snapshotsCollection.InsertOne(ctx, snapshot)

            // Actualizar portfolio
            portfolio.Metadata.LastCalculated = time.Now()
            portfolio.Metadata.NeedsRecalculation = false
            portfolio.UpdatedAt = time.Now()

            filter := bson.M{"user_id": portfolio.UserID}
            update := bson.M{"$set": portfolio}
            s.collection.UpdateOne(ctx, filter, update)

        }(p)
    }

    wg.Wait()
    log.Info("Scheduled recalculation completed")
}
```

---

## 5. BALANCE REQUEST/RESPONSE (RABBITMQ)

### 5.1 Portfolio solicita balance del usuario

```go
// portfolio-api/messaging/balance_client.go
func (bc *BalanceClient) RequestUserBalance(userID int64) (*BalanceResponse, error) {
    correlationID := uuid.New().String()

    // Crear request message
    request := BalanceRequest{
        CorrelationID: correlationID,
        UserID:        userID,
        RequestedBy:   "portfolio-api",
        Timestamp:     time.Now(),
    }

    requestJSON, _ := json.Marshal(request)

    // Publicar en exchange
    err := bc.channel.Publish(
        "balance.request.exchange",  // Exchange
        "balance.request",            // Routing key
        false, false,
        amqp.Publishing{
            ContentType:   "application/json",
            Body:          requestJSON,
            CorrelationId: correlationID,
            ReplyTo:       "balance.response.portfolio",  // Queue para respuesta
            Expiration:    "60000",  // 60 segundos TTL
        },
    )

    if err != nil {
        return nil, err
    }

    // Esperar respuesta (con timeout)
    select {
    case response := <-bc.responseChannel:
        if response.CorrelationID == correlationID {
            return &response, nil
        }
    case <-time.After(5 * time.Second):
        return nil, errors.New("Balance request timeout")
    }

    return nil, errors.New("No response received")
}
```

### 5.2 Users Worker procesa request

```go
// users-api/workers/balance_worker.go
func (bw *BalanceWorker) Start() {
    // Declarar exchange
    bw.channel.ExchangeDeclare("balance.request.exchange", "topic", true, false, false, false, nil)

    // Declarar queue
    queue, _ := bw.channel.QueueDeclare("balance.request", true, false, false, false, nil)

    // Bind
    bw.channel.QueueBind(queue.Name, "balance.request", "balance.request.exchange", false, nil)

    // Consumir
    msgs, _ := bw.channel.Consume(queue.Name, "balance-worker", false, false, false, false, nil)

    for msg := range msgs {
        bw.handleBalanceRequest(msg)
    }
}

func (bw *BalanceWorker) handleBalanceRequest(msg amqp.Delivery) {
    var request BalanceRequest
    json.Unmarshal(msg.Body, &request)

    log.Info("Balance request received", "user_id", request.UserID, "correlation_id", request.CorrelationID)

    // Buscar usuario en MySQL
    var user User
    err := bw.db.Where("id = ?", request.UserID).First(&user).Error

    response := BalanceResponse{
        CorrelationID: request.CorrelationID,
        UserID:        request.UserID,
        Timestamp:     time.Now(),
    }

    if err != nil {
        response.Success = false
        response.Error = "User not found"
    } else {
        response.Success = true
        response.Balance = user.InitialBalance.String()
        response.Currency = "USD"
    }

    responseJSON, _ := json.Marshal(response)

    // Publicar respuesta
    bw.channel.Publish(
        "balance.response.exchange",  // Exchange
        msg.ReplyTo,                  // Routing key (balance.response.portfolio)
        false, false,
        amqp.Publishing{
            ContentType:   "application/json",
            Body:          responseJSON,
            CorrelationId: request.CorrelationID,
        },
    )

    msg.Ack(false)
}
```

---

## 6. ESTRUCTURA DE DATOS MONGODB

```json
{
  "_id": ObjectId("673b5f8a9e1234567890abce"),
  "user_id": NumberLong(123),
  "total_value": NumberDecimal("1051.50"),
  "total_invested": NumberDecimal("1000.00"),
  "profit_loss": NumberDecimal("51.50"),
  "profit_loss_percentage": 5.15,
  "currency": "USD",
  "holdings": [
    {
      "symbol": "BTC",
      "quantity": NumberDecimal("0.001"),
      "average_buy_price": NumberDecimal("50000.00"),
      "current_price": NumberDecimal("51500.00"),
      "current_value": NumberDecimal("51.50"),
      "profit_loss": NumberDecimal("1.50"),
      "profit_loss_percentage": 3.0,
      "percentage_of_portfolio": 4.9,
      "first_purchase_date": ISODate("2025-11-14T10:30:05Z"),
      "last_purchase_date": ISODate("2025-11-14T10:30:05Z"),
      "transactions_count": 1,
      "cost_basis": [
        {
          "quantity": NumberDecimal("0.001"),
          "price": NumberDecimal("50000.00"),
          "date": ISODate("2025-11-14T10:30:05Z"),
          "total_cost": NumberDecimal("50.00")
        }
      ]
    }
  ],
  "performance": {
    "daily_change": NumberDecimal("15.25"),
    "daily_change_percentage": 1.47,
    "weekly_change": NumberDecimal("51.50"),
    "weekly_change_percentage": 5.15,
    "monthly_change": NumberDecimal("51.50"),
    "monthly_change_percentage": 5.15,
    "yearly_change": NumberDecimal("51.50"),
    "yearly_change_percentage": 5.15,
    "all_time_high": NumberDecimal("1051.50"),
    "all_time_low": NumberDecimal("1000.00"),
    "roi": 5.15,
    "annualized_return": 62.0,
    "time_weighted_return": 5.15,
    "money_weighted_return": 5.15,
    "best_performing_asset": "ETH",
    "worst_performing_asset": "BTC"
  },
  "risk_metrics": {
    "volatility_24h": 2.5,
    "volatility_7d": 8.3,
    "volatility_30d": 15.7,
    "sharpe_ratio": 1.85,
    "sortino_ratio": 2.34,
    "calmar_ratio": 3.12,
    "max_drawdown": -5.2,
    "max_drawdown_percentage": -5.2,
    "current_drawdown": 0.0,
    "beta": 0.95,
    "alpha": 2.3,
    "value_at_risk_95": -25.50,
    "conditional_var_95": -32.75,
    "downside_deviation": 3.2
  },
  "diversification": {
    "herfindahl_index": 0.995,
    "concentration_index": 0.9986,
    "effective_holdings": 1.005,
    "largest_position_percentage": 99.86,
    "top_3_concentration": 100.0,
    "categories": {
      "Smart Contract Platform": 99.86,
      "Store of Value": 4.9
    }
  },
  "metadata": {
    "last_calculated": ISODate("2025-11-14T11:30:00Z"),
    "last_order_processed": "673b5f8a9e1234567890abcd",
    "needs_recalculation": false,
    "version": 1
  },
  "created_at": ISODate("2025-11-14T10:30:05Z"),
  "updated_at": ISODate("2025-11-14T11:30:00Z")
}
```

---

## 7. DIAGRAMA DE SECUENCIA

```
Orders API   RabbitMQ   Portfolio   Market Data   MongoDB   Users API
    │           │          │             │           │          │
    │─Publish───>│          │             │           │          │
    │ (executed) │          │             │           │          │
    │           │─Deliver──>│             │           │          │
    │           │          │─GET price───>│           │          │
    │           │          │<─BTC: 51k────│           │          │
    │           │          │─Load portfolio──────────>│          │
    │           │          │<─Current data────────────│          │
    │           │          │─Update holdings          │          │
    │           │          │─Calculate 30+ metrics    │          │
    │           │          │─Save portfolio──────────>│          │
    │           │          │<─Saved OK────────────────│          │
    │           │<─ACK─────│             │           │          │
    │           │          │             │           │          │
    │           │          │─Balance Req──────────────────────>│
    │           │          │ (RabbitMQ)               │          │
    │           │          │<─Balance Resp────────────────────│
    │           │          │  ($99,949.95)            │          │
```

---

## 8. EJEMPLOS DE USO

### Ver portfolio
```bash
curl http://localhost:8005/api/portfolios/123 \
  -H "Authorization: Bearer $TOKEN"
```

### Performance específica
```bash
curl http://localhost:8005/api/portfolios/123/performance \
  -H "Authorization: Bearer $TOKEN"
```

### Holdings detallados
```bash
curl http://localhost:8005/api/portfolios/123/holdings \
  -H "Authorization: Bearer $TOKEN"
```

### Histórico (snapshots)
```bash
curl http://localhost:8005/api/portfolios/123/history?days=30 \
  -H "Authorization: Bearer $TOKEN"
```

---

## Resumen

1. **Consumer RabbitMQ**: Escucha orders.executed → Actualiza holdings
2. **FIFO**: Venta usa First In First Out para cost basis
3. **30+ métricas**: Performance, riesgo, diversificación
4. **Scheduler CRON**: Recalcula cada 15 minutos
5. **Balance async**: Request/Response via RabbitMQ con Users API
6. **Snapshots**: Histórico para análisis temporal
7. **Cálculos avanzados**: Sharpe, Sortino, VaR, Beta, Alpha
8. **Goroutines**: Procesamiento paralelo en scheduler
