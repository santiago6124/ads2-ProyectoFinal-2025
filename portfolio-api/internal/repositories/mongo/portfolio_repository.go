package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"portfolio-api/internal/models"
	"portfolio-api/internal/repositories"
)

// MongoPortfolioRepository implements PortfolioRepository using MongoDB
type MongoPortfolioRepository struct {
	collection *mongo.Collection
}

// NewPortfolioRepository creates a new MongoDB portfolio repository
func NewPortfolioRepository(db *mongo.Database) repositories.PortfolioRepository {
	return &MongoPortfolioRepository{
		collection: db.Collection("portfolios"),
	}
}

// Create creates a new portfolio
func (r *MongoPortfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	if portfolio.ID.IsZero() {
		portfolio.ID = primitive.NewObjectID()
	}
	portfolio.CreatedAt = time.Now()
	portfolio.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, portfolio)
	if err != nil {
		// Handle duplicate key error gracefully (race condition in concurrent requests)
		if mongo.IsDuplicateKeyError(err) {
			return nil // Portfolio already exists, treat as success
		}
		return fmt.Errorf("failed to create portfolio: %w", err)
	}

	return nil
}

// GetByID retrieves a portfolio by its ID
func (r *MongoPortfolioRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Portfolio, error) {
	var portfolio models.Portfolio
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("portfolio not found")
		}
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return &portfolio, nil
}

// GetByUserID retrieves a portfolio by user ID
func (r *MongoPortfolioRepository) GetByUserID(ctx context.Context, userID int64) (*models.Portfolio, error) {
	// Fetch raw BSON document
	var rawDoc bson.M
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&rawDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("portfolio not found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Convert BSON document to Portfolio model
	portfolio, err := r.bsonToPortfolio(rawDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to convert BSON to portfolio: %w", err)
	}

	return portfolio, nil
}

// bsonToPortfolio converts a BSON document to Portfolio model
func (r *MongoPortfolioRepository) bsonToPortfolio(doc bson.M) (*models.Portfolio, error) {
	portfolio := &models.Portfolio{}

	// Parse basic fields
	if v, ok := doc["_id"]; ok {
		if oid, ok := v.(primitive.ObjectID); ok {
			portfolio.ID = oid
		}
	}

	if v, ok := doc["user_id"]; ok {
		switch val := v.(type) {
		case int64:
			portfolio.UserID = val
		case int32:
			portfolio.UserID = int64(val)
		case int:
			portfolio.UserID = int64(val)
		}
	}

	if v, ok := doc["currency"]; ok {
		if str, ok := v.(string); ok {
			portfolio.Currency = str
		}
	}

	// Parse decimal fields from strings
	portfolio.TotalValue = parseDecimalField(doc, "total_value")
	portfolio.TotalInvested = parseDecimalField(doc, "total_invested")
	portfolio.TotalCash = parseDecimalField(doc, "total_cash")
	portfolio.ProfitLoss = parseDecimalField(doc, "profit_loss")
	portfolio.ProfitLossPercentage = parseDecimalField(doc, "profit_loss_percentage")

	// Parse timestamps
	if v, ok := doc["created_at"]; ok {
		if t, ok := v.(primitive.DateTime); ok {
			portfolio.CreatedAt = t.Time()
		}
	}

	if v, ok := doc["updated_at"]; ok {
		if t, ok := v.(primitive.DateTime); ok {
			portfolio.UpdatedAt = t.Time()
		}
	}

	// Parse holdings array
	if v, ok := doc["holdings"]; ok {
		if arr, ok := v.(primitive.A); ok {
			for _, item := range arr {
				if holdingDoc, ok := item.(bson.M); ok {
					holding := models.Holding{}

					if v, ok := holdingDoc["symbol"]; ok {
						if str, ok := v.(string); ok {
							holding.Symbol = str
						}
					}

					if v, ok := holdingDoc["name"]; ok {
						if str, ok := v.(string); ok {
							holding.Name = str
						}
					}

					holding.Quantity = parseDecimalField(holdingDoc, "quantity")
					holding.AverageBuyPrice = parseDecimalField(holdingDoc, "average_buy_price")
					holding.TotalInvested = parseDecimalField(holdingDoc, "total_invested")
					holding.CurrentPrice = parseDecimalField(holdingDoc, "current_price")
					holding.CurrentValue = parseDecimalField(holdingDoc, "current_value")
					holding.ProfitLoss = parseDecimalField(holdingDoc, "profit_loss")
					holding.ProfitLossPercentage = parseDecimalField(holdingDoc, "profit_loss_percentage")
					holding.PercentageOfPortfolio = parseDecimalField(holdingDoc, "percentage_of_portfolio")

					if v, ok := holdingDoc["first_purchase_date"]; ok {
						if t, ok := v.(primitive.DateTime); ok {
							holding.FirstPurchaseDate = t.Time()
						}
					}

					if v, ok := holdingDoc["last_purchase_date"]; ok {
						if t, ok := v.(primitive.DateTime); ok {
							holding.LastPurchaseDate = t.Time()
						}
					}

					if v, ok := holdingDoc["transactions_count"]; ok {
						switch val := v.(type) {
						case int64:
							holding.TransactionsCount = int(val)
						case int32:
							holding.TransactionsCount = int(val)
						case int:
							holding.TransactionsCount = val
						}
					}

					portfolio.Holdings = append(portfolio.Holdings, holding)
				}
			}
		}
	}

	return portfolio, nil
}

// parseDecimalField parses a decimal field from BSON document
func parseDecimalField(doc bson.M, fieldName string) decimal.Decimal {
	if v, ok := doc[fieldName]; ok {
		if str, ok := v.(string); ok {
			if d, err := decimal.NewFromString(str); err == nil {
				return d
			}
		}
	}
	return decimal.Zero
}

// Update updates an existing portfolio
func (r *MongoPortfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
	portfolio.UpdatedAt = time.Now()

	filter := bson.M{"_id": portfolio.ID}
	update := bson.M{"$set": portfolio}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}

// Delete deletes a portfolio by ID
func (r *MongoPortfolioRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}

// DeleteByUserID deletes a portfolio by user ID
func (r *MongoPortfolioRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

// List retrieves portfolios with pagination
func (r *MongoPortfolioRepository) List(ctx context.Context, limit, offset int) ([]*models.Portfolio, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetNeedingRecalculation retrieves portfolios that need recalculation
func (r *MongoPortfolioRepository) GetNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error) {
	filter := bson.M{"metadata.needs_recalculation": true}
	opts := options.Find().SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios needing recalculation: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetByUserIDs retrieves portfolios for multiple users
func (r *MongoPortfolioRepository) GetByUserIDs(ctx context.Context, userIDs []int64) ([]*models.Portfolio, error) {
	filter := bson.M{"user_id": bson.M{"$in": userIDs}}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetTopPerformers retrieves top performing portfolios
func (r *MongoPortfolioRepository) GetTopPerformers(ctx context.Context, limit int, period string) ([]*models.Portfolio, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "profit_loss_percentage", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performers: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetPortfolioStats retrieves portfolio statistics
func (r *MongoPortfolioRepository) GetPortfolioStats(ctx context.Context) (*repositories.PortfolioStats, error) {
	// Simplified implementation - would use aggregation pipeline in production
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count portfolios: %w", err)
	}

	return &repositories.PortfolioStats{
		TotalPortfolios: count,
	}, nil
}

// UpdateMetadata updates only the metadata field
func (r *MongoPortfolioRepository) UpdateMetadata(ctx context.Context, userID int64, metadata map[string]interface{}) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"metadata":   metadata,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

// BulkUpdate updates multiple portfolios
func (r *MongoPortfolioRepository) BulkUpdate(ctx context.Context, portfolios []*models.Portfolio) error {
	if len(portfolios) == 0 {
		return nil
	}

	var writes []mongo.WriteModel
	for _, portfolio := range portfolios {
		portfolio.UpdatedAt = time.Now()
		filter := bson.M{"_id": portfolio.ID}
		update := bson.M{"$set": portfolio}
		writes = append(writes, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update))
	}

	_, err := r.collection.BulkWrite(ctx, writes)
	if err != nil {
		return fmt.Errorf("failed to bulk update portfolios: %w", err)
	}

	return nil
}

// UpdateHoldingsFromOrder updates portfolio holdings based on an order execution
func (r *MongoPortfolioRepository) UpdateHoldingsFromOrder(ctx context.Context, userID int64, symbol string, quantity, price float64, orderType string) error {
	fmt.Printf("🔍 DEBUG: UpdateHoldingsFromOrder called - userID=%d, symbol=%s, qty=%f, price=%f, type=%s\n", userID, symbol, quantity, price, orderType)

	// Get existing portfolio or create new one
	portfolio, err := r.GetByUserID(ctx, userID)
	if err != nil {
		fmt.Printf("🔍 DEBUG: No existing portfolio found, creating new one for user %d\n", userID)
		// Portfolio doesn't exist, create a new one
		portfolio = &models.Portfolio{
			UserID:        userID,
			TotalValue:    decimal.Zero,
			TotalInvested: decimal.Zero,
			TotalCash:     decimal.Zero,
			ProfitLoss:    decimal.Zero,
			ProfitLossPercentage: decimal.Zero,
			Currency:      "USD",
			Holdings:      []models.Holding{},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
	} else {
		fmt.Printf("🔍 DEBUG: Found existing portfolio with %d holdings\n", len(portfolio.Holdings))
	}

	quantityDec := decimal.NewFromFloat(quantity)
	priceDec := decimal.NewFromFloat(price)
	now := time.Now()

	// Find existing holding
	holdingIndex := -1
	for i, h := range portfolio.Holdings {
		if h.Symbol == symbol {
			holdingIndex = i
			break
		}
	}

	if orderType == "buy" {
		// Buy operation: add or update holding
		investmentAmount := quantityDec.Mul(priceDec)
		fmt.Printf("🔍 DEBUG: Buy operation - investment amount: %s\n", investmentAmount.String())

		if holdingIndex >= 0 {
			fmt.Printf("🔍 DEBUG: Updating existing holding at index %d\n", holdingIndex)
			// Update existing holding
			holding := &portfolio.Holdings[holdingIndex]
			oldQuantity := holding.Quantity
			oldInvested := holding.TotalInvested

			// Calculate new average price
			newQuantity := oldQuantity.Add(quantityDec)
			newInvested := oldInvested.Add(investmentAmount)
			newAvgPrice := decimal.Zero
			if newQuantity.GreaterThan(decimal.Zero) {
				newAvgPrice = newInvested.Div(newQuantity)
			}

			holding.Quantity = newQuantity
			holding.AverageBuyPrice = newAvgPrice
			holding.TotalInvested = newInvested
			holding.CurrentPrice = priceDec
			holding.CurrentValue = newQuantity.Mul(priceDec)
			holding.ProfitLoss = holding.CurrentValue.Sub(newInvested)
			if newInvested.GreaterThan(decimal.Zero) {
				holding.ProfitLossPercentage = holding.ProfitLoss.Div(newInvested).Mul(decimal.NewFromInt(100))
			}
			holding.LastPurchaseDate = now
			holding.TransactionsCount++
			fmt.Printf("🔍 DEBUG: Updated holding - qty: %s, avgPrice: %s, invested: %s\n",
				newQuantity.String(), newAvgPrice.String(), newInvested.String())
		} else {
			fmt.Printf("🔍 DEBUG: Creating new holding for %s\n", symbol)
			// Create new holding
			newHolding := models.Holding{
				Symbol:                symbol,
				Name:                  symbol,
				Quantity:              quantityDec,
				AverageBuyPrice:       priceDec,
				TotalInvested:         investmentAmount,
				CurrentPrice:          priceDec,
				CurrentValue:          investmentAmount,
				ProfitLoss:            decimal.Zero,
				ProfitLossPercentage:  decimal.Zero,
				PercentageOfPortfolio: decimal.Zero,
				FirstPurchaseDate:     now,
				LastPurchaseDate:      now,
				TransactionsCount:     1,
			}
			portfolio.Holdings = append(portfolio.Holdings, newHolding)
			fmt.Printf("🔍 DEBUG: New holding created - qty: %s, price: %s, value: %s\n",
				quantityDec.String(), priceDec.String(), investmentAmount.String())
		}

		// Update portfolio total invested
		portfolio.TotalInvested = portfolio.TotalInvested.Add(investmentAmount)
		fmt.Printf("🔍 DEBUG: Portfolio total invested: %s\n", portfolio.TotalInvested.String())

	} else if orderType == "sell" {
		// Sell operation: reduce or remove holding
		if holdingIndex < 0 {
			return fmt.Errorf("cannot sell %s: holding not found in portfolio", symbol)
		}

		holding := &portfolio.Holdings[holdingIndex]
		if holding.Quantity.LessThan(quantityDec) {
			return fmt.Errorf("cannot sell %f %s: insufficient quantity (have %s)", quantity, symbol, holding.Quantity.String())
		}

		sellCostBasis := quantityDec.Mul(holding.AverageBuyPrice)

		// Reduce quantity
		holding.Quantity = holding.Quantity.Sub(quantityDec)
		holding.TotalInvested = holding.TotalInvested.Sub(sellCostBasis)

		// Update current values
		holding.CurrentPrice = priceDec
		holding.CurrentValue = holding.Quantity.Mul(priceDec)
		holding.ProfitLoss = holding.CurrentValue.Sub(holding.TotalInvested)
		if holding.TotalInvested.GreaterThan(decimal.Zero) {
			holding.ProfitLossPercentage = holding.ProfitLoss.Div(holding.TotalInvested).Mul(decimal.NewFromInt(100))
		} else {
			holding.ProfitLossPercentage = decimal.Zero
		}
		holding.TransactionsCount++

		// Remove holding if quantity is zero
		if holding.Quantity.IsZero() {
			portfolio.Holdings = append(portfolio.Holdings[:holdingIndex], portfolio.Holdings[holdingIndex+1:]...)
		}

		// Update portfolio total invested
		portfolio.TotalInvested = portfolio.TotalInvested.Sub(sellCostBasis)
	}

	// Recalculate portfolio totals
	totalHoldingsValue := decimal.Zero
	for i := range portfolio.Holdings {
		totalHoldingsValue = totalHoldingsValue.Add(portfolio.Holdings[i].CurrentValue)
	}

	portfolio.TotalValue = totalHoldingsValue.Add(portfolio.TotalCash)
	portfolio.ProfitLoss = portfolio.TotalValue.Sub(portfolio.TotalInvested).Sub(portfolio.TotalCash)
	if portfolio.TotalInvested.GreaterThan(decimal.Zero) {
		portfolio.ProfitLossPercentage = portfolio.ProfitLoss.Div(portfolio.TotalInvested).Mul(decimal.NewFromInt(100))
	}

	fmt.Printf("🔍 DEBUG: Final portfolio - totalValue: %s, totalInvested: %s, holdings count: %d\n",
		portfolio.TotalValue.String(), portfolio.TotalInvested.String(), len(portfolio.Holdings))

	// Update timestamp
	portfolio.UpdatedAt = now

	// Convert decimal fields to strings for BSON storage
	holdingsBSON := make([]bson.M, len(portfolio.Holdings))
	for i, h := range portfolio.Holdings {
		holdingsBSON[i] = bson.M{
			"crypto_id":                h.CryptoID,
			"symbol":                   h.Symbol,
			"name":                     h.Name,
			"quantity":                 h.Quantity.String(),
			"average_buy_price":        h.AverageBuyPrice.String(),
			"total_invested":           h.TotalInvested.String(),
			"current_price":            h.CurrentPrice.String(),
			"current_value":            h.CurrentValue.String(),
			"profit_loss":              h.ProfitLoss.String(),
			"profit_loss_percentage":   h.ProfitLossPercentage.String(),
			"percentage_of_portfolio":  h.PercentageOfPortfolio.String(),
			"first_purchase_date":      h.FirstPurchaseDate,
			"last_purchase_date":       h.LastPurchaseDate,
			"transactions_count":       h.TransactionsCount,
		}
	}

	// Upsert portfolio with string-based decimal values
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"user_id":                 portfolio.UserID,
			"total_value":             portfolio.TotalValue.String(),
			"total_invested":          portfolio.TotalInvested.String(),
			"total_cash":              portfolio.TotalCash.String(),
			"profit_loss":             portfolio.ProfitLoss.String(),
			"profit_loss_percentage":  portfolio.ProfitLossPercentage.String(),
			"currency":                portfolio.Currency,
			"holdings":                holdingsBSON,
			"updated_at":              portfolio.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"created_at": portfolio.CreatedAt,
		},
	}
	opts := options.Update().SetUpsert(true)

	fmt.Printf("🔍 DEBUG: Executing MongoDB upsert for user %d...\n", userID)
	result, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		fmt.Printf("🔍 DEBUG: MongoDB upsert FAILED: %v\n", err)
		return fmt.Errorf("failed to update portfolio holdings: %w", err)
	}

	fmt.Printf("🔍 DEBUG: MongoDB upsert result - matched: %d, modified: %d, upserted: %d\n",
		result.MatchedCount, result.ModifiedCount, result.UpsertedCount)

	return nil
}
