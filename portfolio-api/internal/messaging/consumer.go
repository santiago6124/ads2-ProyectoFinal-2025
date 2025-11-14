package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"portfolio-api/internal/models"
	"portfolio-api/internal/repositories"
)

// PortfolioUpdateEvent evento para actualizar portfolio
type PortfolioUpdateEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Quantity  string    `json:"quantity"`
	Price     string    `json:"price"`
	OrderType string    `json:"order_type"` // "buy" o "sell"
	TotalCost string    `json:"total_cost"`
	Fee       string    `json:"fee"`
	Timestamp time.Time `json:"timestamp"`
}

// connectWithRetry intenta conectarse a RabbitMQ con reintentos y backoff exponencial
func connectWithRetry(url string, maxRetries int) (*amqp.Connection, error) {
	for i := 0; i < maxRetries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("✅ Portfolio Consumer successfully connected to RabbitMQ")
			return conn, nil
		}

		if i < maxRetries-1 {
			wait := time.Duration(1<<uint(i)) * time.Second // Backoff: 1s, 2s, 4s, 8s, 16s
			log.Printf("⚠️ Portfolio Consumer failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries", maxRetries)
}

// Consumer consumer para procesar actualizaciones de portfolio
type Consumer struct {
	connection     *amqp.Connection
	channel        *amqp.Channel
	queueName      string
	portfolioRepo  repositories.PortfolioRepository
	logger         *logrus.Logger
}

// NewConsumer crea un nuevo consumer de portfolio
func NewConsumer(
	rabbitmqURL string,
	portfolioRepo repositories.PortfolioRepository,
	logger *logrus.Logger,
) (*Consumer, error) {
	// Usar connectWithRetry para conexión robusta
	conn, err := connectWithRetry(rabbitmqURL, 7) // 7 intentos: ~127 segundos total
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declarar exchange
	exchangeName := "portfolio.events"
	err = ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declarar queue
	queueName := "portfolio.updates"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "portfolio.dlx",
		},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,
		"portfolio.update",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Declarar Dead Letter Exchange (DLX)
	dlxName := "portfolio.dlx"
	err = ch.ExchangeDeclare(
		dlxName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declarar Dead Letter Queue
	_, err = ch.QueueDeclare(
		"portfolio.failed",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	err = ch.QueueBind(
		"portfolio.failed",
		"#",
		dlxName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind DLQ: %w", err)
	}

	log.Printf("✅ Portfolio consumer initialized, listening on queue: %s", queueName)

	return &Consumer{
		connection:    conn,
		channel:       ch,
		queueName:     queueName,
		portfolioRepo: portfolioRepo,
		logger:        logger,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *Consumer) Start(ctx context.Context) error {
	// Set QoS - procesar un mensaje a la vez
	err := c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("🔄 Portfolio worker started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Portfolio worker shutting down...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Procesar mensaje
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("❌ Error processing portfolio update: %v", err)
				// Error recuperable - requeue
				msg.Nack(false, true)
			} else {
				// Ack si todo salió bien
				msg.Ack(false)
			}
		}
	}
}

// processMessage procesa un mensaje de actualización de portfolio
func (c *Consumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	start := time.Now()

	var event PortfolioUpdateEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📊 PORTFOLIO WORKER - Starting processing")
	log.Printf("📦 Order ID: %s", event.OrderID)
	log.Printf("👤 User: %d | Symbol: %s | Type: %s", event.UserID, event.Symbol, event.OrderType)
	log.Printf("📈 Quantity: %s @ Price: %s", event.Quantity, event.Price)

	// Parse decimal values
	quantity, err := decimal.NewFromString(event.Quantity)
	if err != nil {
		return fmt.Errorf("invalid quantity format: %w", err)
	}

	price, err := decimal.NewFromString(event.Price)
	if err != nil {
		return fmt.Errorf("invalid price format: %w", err)
	}

	fee := decimal.Zero
	if event.Fee != "" {
		fee, err = decimal.NewFromString(event.Fee)
		if err != nil {
			return fmt.Errorf("invalid fee format: %w", err)
		}
	}

	// Get or create portfolio
	log.Printf("1️⃣ [Order %s] Getting portfolio for user %d...", event.OrderID, event.UserID)
	portfolio, err := c.portfolioRepo.GetByUserID(ctx, int64(event.UserID))
	if err != nil || portfolio == nil {
		// Create new portfolio
		log.Printf("   Portfolio not found, creating new one...")
		portfolio = &models.Portfolio{
			ID:                   primitive.NewObjectID(),
			UserID:               int64(event.UserID),
			Currency:             "USD",
			TotalValue:           decimal.Zero,
			TotalInvested:        decimal.Zero,
			ProfitLoss:           decimal.Zero,
			ProfitLossPercentage: decimal.Zero,
			Holdings:             []models.Holding{},
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := c.portfolioRepo.Create(ctx, portfolio); err != nil {
			return fmt.Errorf("failed to create portfolio: %w", err)
		}
		log.Printf("✓ [Order %s] New portfolio created", event.OrderID)
	} else {
		log.Printf("✓ [Order %s] Portfolio found with %d holdings", event.OrderID, len(portfolio.Holdings))
	}

	// Update holding
	log.Printf("2️⃣ [Order %s] Updating holding for %s...", event.OrderID, event.Symbol)
	if err := c.updateHolding(portfolio, event.Symbol, quantity, price, fee, event.OrderType, event.OrderID); err != nil {
		return fmt.Errorf("failed to update holding: %w", err)
	}

	// Save updated portfolio
	log.Printf("3️⃣ [Order %s] Saving portfolio to database...", event.OrderID)
	portfolio.UpdatedAt = time.Now()
	portfolio.Metadata.LastOrderProcessed = time.Now()

	if err := c.portfolioRepo.Update(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	elapsed := time.Since(start)
	log.Printf("✅ PORTFOLIO WORKER - Completed in %v", elapsed)
	log.Printf("   Order: %s | User: %d | Symbol: %s", event.OrderID, event.UserID, event.Symbol)
	log.Printf("   Holdings count: %d | Total invested: %s", len(portfolio.Holdings), portfolio.TotalInvested.String())
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

// updateHolding actualiza o crea un holding en el portfolio
func (c *Consumer) updateHolding(portfolio *models.Portfolio, symbol string, quantity, price, fee decimal.Decimal, txType, orderID string) error {
	// Find existing holding
	existingHolding, exists := portfolio.GetHoldingBySymbol(symbol)

	if txType == "buy" {
		if exists {
			// Update existing holding
			log.Printf("   Found existing holding, updating...")

			// Calculate new average buy price using weighted average
			currentInvested := existingHolding.AverageBuyPrice.Mul(existingHolding.Quantity)
			newInvestment := price.Mul(quantity)
			totalInvested := currentInvested.Add(newInvestment)
			newQuantity := existingHolding.Quantity.Add(quantity)

			existingHolding.AverageBuyPrice = totalInvested.Div(newQuantity)
			existingHolding.Quantity = newQuantity
			existingHolding.TotalInvested = totalInvested
			existingHolding.CurrentPrice = price
			existingHolding.CurrentValue = price.Mul(newQuantity)
			existingHolding.ProfitLoss = existingHolding.CurrentValue.Sub(existingHolding.TotalInvested)
			if existingHolding.TotalInvested.GreaterThan(decimal.Zero) {
				existingHolding.ProfitLossPercentage = existingHolding.ProfitLoss.Div(existingHolding.TotalInvested).Mul(decimal.NewFromInt(100))
			}
			existingHolding.LastPurchaseDate = time.Now()
			existingHolding.TransactionsCount++

			// Add cost basis entry
			costBasisEntry := models.CostBasisEntry{
				Date:     time.Now(),
				Quantity: quantity,
				Price:    price,
				OrderID:  orderID,
			}
			existingHolding.CostBasis = append(existingHolding.CostBasis, costBasisEntry)

			log.Printf("   Updated: Qty %.8f @ Avg %.2f | Invested: %.2f",
				newQuantity.InexactFloat64(),
				existingHolding.AverageBuyPrice.InexactFloat64(),
				totalInvested.InexactFloat64())
		} else {
			// Create new holding
			log.Printf("   Creating new holding...")
			totalInvestment := price.Mul(quantity)

			newHolding := models.Holding{
				CryptoID:              symbol,
				Symbol:                symbol,
				Name:                  symbol,
				Quantity:              quantity,
				AverageBuyPrice:       price,
				TotalInvested:         totalInvestment,
				CurrentPrice:          price,
				CurrentValue:          totalInvestment,
				ProfitLoss:            decimal.Zero,
				ProfitLossPercentage:  decimal.Zero,
				PercentageOfPortfolio: decimal.Zero,
				FirstPurchaseDate:     time.Now(),
				LastPurchaseDate:      time.Now(),
				TransactionsCount:     1,
				CostBasis: []models.CostBasisEntry{
					{
						Date:     time.Now(),
						Quantity: quantity,
						Price:    price,
						OrderID:  orderID,
					},
				},
			}

			portfolio.Holdings = append(portfolio.Holdings, newHolding)

			log.Printf("   Created: Qty %.8f @ %.2f | Invested: %.2f",
				quantity.InexactFloat64(),
				price.InexactFloat64(),
				totalInvestment.InexactFloat64())
		}

	} else if txType == "sell" {
		if !exists {
			return fmt.Errorf("cannot sell %s: holding not found", symbol)
		}

		if existingHolding.Quantity.LessThan(quantity) {
			return fmt.Errorf("cannot sell %s: insufficient quantity (have %.8f, trying to sell %.8f)",
				symbol, existingHolding.Quantity.InexactFloat64(), quantity.InexactFloat64())
		}

		log.Printf("   Selling from existing holding...")

		// Calculate reduction in invested amount (proportional to quantity sold)
		investedReduction := existingHolding.TotalInvested.Mul(quantity).Div(existingHolding.Quantity)

		existingHolding.Quantity = existingHolding.Quantity.Sub(quantity)
		existingHolding.TotalInvested = existingHolding.TotalInvested.Sub(investedReduction)
		existingHolding.CurrentPrice = price
		existingHolding.CurrentValue = price.Mul(existingHolding.Quantity)

		if existingHolding.Quantity.GreaterThan(decimal.Zero) {
			existingHolding.ProfitLoss = existingHolding.CurrentValue.Sub(existingHolding.TotalInvested)
			if existingHolding.TotalInvested.GreaterThan(decimal.Zero) {
				existingHolding.ProfitLossPercentage = existingHolding.ProfitLoss.Div(existingHolding.TotalInvested).Mul(decimal.NewFromInt(100))
			}
		} else {
			// Sold all, remove holding
			log.Printf("   Sold all holdings, removing from portfolio")
			portfolio.RemoveHolding(symbol)
		}

		existingHolding.TransactionsCount++

		log.Printf("   Sold: Qty %.8f @ %.2f | Remaining: %.8f",
			quantity.InexactFloat64(),
			price.InexactFloat64(),
			existingHolding.Quantity.InexactFloat64())
	}

	// Recalculate portfolio totals
	c.recalculatePortfolio(portfolio)

	return nil
}

// recalculatePortfolio recalcula los totales del portfolio
func (c *Consumer) recalculatePortfolio(portfolio *models.Portfolio) {
	totalInvested := decimal.Zero
	totalCurrentValue := decimal.Zero

	for i := range portfolio.Holdings {
		holding := &portfolio.Holdings[i]
		totalInvested = totalInvested.Add(holding.TotalInvested)
		totalCurrentValue = totalCurrentValue.Add(holding.CurrentValue)

		// Calculate percentage of portfolio
		if totalCurrentValue.GreaterThan(decimal.Zero) {
			holding.PercentageOfPortfolio = holding.CurrentValue.Div(totalCurrentValue).Mul(decimal.NewFromInt(100))
		}
	}

	portfolio.TotalInvested = totalInvested
	portfolio.TotalValue = totalCurrentValue  // Only crypto value, cash managed by Users API
	portfolio.ProfitLoss = totalCurrentValue.Sub(totalInvested)

	if totalInvested.GreaterThan(decimal.Zero) {
		portfolio.ProfitLossPercentage = portfolio.ProfitLoss.Div(totalInvested).Mul(decimal.NewFromInt(100))
	} else {
		portfolio.ProfitLossPercentage = decimal.Zero
	}

	// Update diversification metrics
	portfolio.Diversification.HoldingsCount = len(portfolio.Holdings)
}

// Stop detiene el consumer
func (c *Consumer) Stop() error {
	log.Printf("🛑 Portfolio consumer stopping...")
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}
