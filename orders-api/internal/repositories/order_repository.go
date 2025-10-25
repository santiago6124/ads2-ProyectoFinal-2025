package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/pkg/database"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id string) (*models.Order, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*models.Order, error)
	Update(ctx context.Context, order *models.Order) error
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, error)
	// ListAll y GetAdminStatistics eliminados en sistema simplificado (funciones admin no necesarias)
	GetOrdersSummary(ctx context.Context, userID int) (*dto.OrdersSummary, error)
	UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error
	GetPendingOrders(ctx context.Context, limit int) ([]models.Order, error)
	GetOrdersByStatus(ctx context.Context, status models.OrderStatus, limit int) ([]models.Order, error)
	BulkUpdateStatus(ctx context.Context, orderIDs []string, status models.OrderStatus) error
}

type orderRepository struct {
	db         *database.Database
	collection *mongo.Collection
}

func NewOrderRepository(db *database.Database) OrderRepository {
	return &orderRepository{
		db:         db,
		collection: db.GetCollection("orders"),
	}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	if order.ID.IsZero() {
		order.ID = primitive.NewObjectID()
	}

	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	if order.OrderNumber == "" {
		order.OrderNumber = models.NewOrderNumber()
	}

	_, err := r.collection.InsertOne(ctx, order)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("order with number %s already exists", order.OrderNumber)
		}
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	var order models.Order
	filter := bson.M{"_id": objectID}

	err = r.collection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	var order models.Order
	filter := bson.M{"order_number": orderNumber}

	err := r.collection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) Update(ctx context.Context, order *models.Order) error {
	objectID, err := primitive.ObjectIDFromHex(order.ID.Hex())
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	order.UpdatedAt = time.Now()

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": order}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *orderRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, error) {
	mongoFilter := bson.M{"user_id": userID}
	r.applyFilters(mongoFilter, filter)

	return r.executeQuery(ctx, mongoFilter, filter)
}

// ListAll comentado - función admin no necesaria en sistema simplificado
/*
func (r *orderRepository) ListAll(ctx context.Context, filter *dto.AdminOrderFilterRequest) ([]models.Order, int64, error) {
	mongoFilter := bson.M{}

	if filter.UserID != nil {
		mongoFilter["user_id"] = *filter.UserID
	}

	r.applyFilters(mongoFilter, &filter.OrderFilterRequest)

	return r.executeQuery(ctx, mongoFilter, &filter.OrderFilterRequest)
}
*/

func (r *orderRepository) applyFilters(mongoFilter bson.M, filter *dto.OrderFilterRequest) {
	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}

	if filter.CryptoSymbol != nil {
		mongoFilter["crypto_symbol"] = bson.M{"$regex": *filter.CryptoSymbol, "$options": "i"}
	}

	if filter.Type != nil {
		mongoFilter["type"] = *filter.Type
	}

	// Filtros de fecha From/To eliminados en sistema simplificado
	// Se puede agregar después si se necesita
}

func (r *orderRepository) executeQuery(ctx context.Context, mongoFilter bson.M, filter *dto.OrderFilterRequest) ([]models.Order, int64, error) {
	total, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(filter.GetOffset()))
	findOptions.SetLimit(int64(filter.Limit))

	// Sort por created_at descendente por defecto (más recientes primero)
	findOptions.SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find orders: %w", err)
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("failed to decode orders: %w", err)
	}

	return orders, total, nil
}

func (r *orderRepository) GetOrdersSummary(ctx context.Context, userID int) (*dto.OrdersSummary, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"user_id": userID}},
		{"$group": bson.M{
			"_id":               nil,
			"total_invested":    bson.M{"$sum": "$total_amount"},
			"total_orders":      bson.M{"$sum": 1},
			"successful_orders": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "executed"}}, 1, 0}}},
			"failed_orders":     bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "failed"}}, 1, 0}}},
			"total_fees":        bson.M{"$sum": "$fee"},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders summary: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode summary: %w", err)
	}

	if len(results) == 0 {
		return &dto.OrdersSummary{}, nil
	}

	result := results[0]
	summary := &dto.OrdersSummary{
		TotalVolume:     parseDecimalFromBSON(result["total_invested"]),
		TotalOrders:     parseInt64FromBSON(result["total_orders"]),
		ExecutedOrders:  parseInt64FromBSON(result["successful_orders"]),
		FailedOrders:    parseInt64FromBSON(result["failed_orders"]),
		PendingOrders:   0, // Se puede calcular si se necesita
		CancelledOrders: 0, // Se puede calcular si se necesita
	}

	return summary, nil
}

// GetAdminStatistics comentado - función admin no necesaria en sistema simplificado
/*
func (r *orderRepository) GetAdminStatistics(ctx context.Context) (*dto.AdminStatistics, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	pipeline := []bson.M{
		{"$facet": bson.M{
			"overall": []bson.M{
				{"$group": bson.M{
					"_id":           nil,
					"total_orders":  bson.M{"$sum": 1},
					"total_volume":  bson.M{"$sum": "$total_amount"},
					"total_fees":    bson.M{"$sum": "$fee"},
					"avg_order":     bson.M{"$avg": "$total_amount"},
				}},
			},
			"today": []bson.M{
				{"$match": bson.M{"created_at": bson.M{"$gte": today}}},
				{"$group": bson.M{
					"_id":           nil,
					"orders_today":  bson.M{"$sum": 1},
					"volume_today":  bson.M{"$sum": "$total_amount"},
				}},
			},
			"top_cryptos": []bson.M{
				{"$group": bson.M{
					"_id":           "$crypto_symbol",
					"total_orders":  bson.M{"$sum": 1},
					"total_volume":  bson.M{"$sum": "$total_amount"},
				}},
				{"$sort": bson.M{"total_volume": -1}},
				{"$limit": 10},
			},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin statistics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode statistics: %w", err)
	}

	if len(results) == 0 {
		return &dto.AdminStatistics{}, nil
	}

	result := results[0]
	stats := &dto.AdminStatistics{}

	if overall, ok := result["overall"].([]interface{}); ok && len(overall) > 0 {
		if overallData, ok := overall[0].(bson.M); ok {
			stats.TotalOrders = parseInt64FromBSON(overallData["total_orders"])
			stats.TotalVolume = parseDecimalFromBSON(overallData["total_volume"])
			stats.TotalFeesCollected = parseDecimalFromBSON(overallData["total_fees"])
			stats.AverageOrderSize = parseDecimalFromBSON(overallData["avg_order"])
		}
	}

	if today, ok := result["today"].([]interface{}); ok && len(today) > 0 {
		if todayData, ok := today[0].(bson.M); ok {
			stats.OrdersToday = parseInt64FromBSON(todayData["orders_today"])
			stats.VolumeToday = parseDecimalFromBSON(todayData["volume_today"])
		}
	}

	if topCryptos, ok := result["top_cryptos"].([]interface{}); ok {
		stats.TopCryptocurrencies = make([]dto.CryptoStats, len(topCryptos))
		for i, crypto := range topCryptos {
			if cryptoData, ok := crypto.(bson.M); ok {
				stats.TopCryptocurrencies[i] = dto.CryptoStats{
					Symbol:      parseStringFromBSON(cryptoData["_id"]),
					TotalOrders: parseInt64FromBSON(cryptoData["total_orders"]),
					TotalVolume: parseDecimalFromBSON(cryptoData["total_volume"]),
				}
			}
		}
	}

	return stats, nil
}
*/

func (r *orderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == models.OrderStatusExecuted {
		update["$set"].(bson.M)["executed_at"] = time.Now()
	} else if status == models.OrderStatusCancelled {
		update["$set"].(bson.M)["cancelled_at"] = time.Now()
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *orderRepository) GetPendingOrders(ctx context.Context, limit int) ([]models.Order, error) {
	filter := bson.M{"status": models.OrderStatusPending}
	findOptions := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{"created_at", 1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending orders: %w", err)
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("failed to decode orders: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) GetOrdersByStatus(ctx context.Context, status models.OrderStatus, limit int) ([]models.Order, error) {
	filter := bson.M{"status": status}
	findOptions := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders: %w", err)
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("failed to decode orders: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) BulkUpdateStatus(ctx context.Context, orderIDs []string, status models.OrderStatus) error {
	var objectIDs []primitive.ObjectID
	for _, id := range orderIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid order ID %s: %w", id, err)
		}
		objectIDs = append(objectIDs, objectID)
	}

	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == models.OrderStatusCancelled {
		update["$set"].(bson.M)["cancelled_at"] = time.Now()
	}

	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to bulk update orders: %w", err)
	}

	if result.MatchedCount != int64(len(orderIDs)) {
		return fmt.Errorf("only %d out of %d orders were updated", result.MatchedCount, len(orderIDs))
	}

	return nil
}

// Helper functions for parsing BSON data
func parseDecimalFromBSON(value interface{}) decimal.Decimal {
	switch v := value.(type) {
	case float64:
		return decimal.NewFromFloat(v)
	case int32:
		return decimal.NewFromInt32(v)
	case int64:
		return decimal.NewFromInt(v)
	case primitive.Decimal128:
		d, _ := primitive.ParseDecimal128(v.String())
		return decimal.RequireFromString(d.String())
	default:
		return decimal.Zero
	}
}

func parseInt64FromBSON(value interface{}) int64 {
	switch v := value.(type) {
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func parseStringFromBSON(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}