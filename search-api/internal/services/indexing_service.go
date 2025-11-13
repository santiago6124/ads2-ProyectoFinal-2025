package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"search-api/internal/clients"
	"search-api/internal/models"
	"search-api/internal/repositories"
)

// IndexingService handles order indexing operations
type IndexingService struct {
	ordersClient *clients.OrdersClient
	solrRepo     repositories.SearchRepository
	logger       *logrus.Logger
}

// LegacyOrderEvent represents minimal order data passed via messaging events
type LegacyOrderEvent struct {
	EventType    string
	OrderID      string
	UserID       int
	Type         string
	Status       string
	CryptoSymbol string
	Quantity     string
	Price        string
	TotalAmount  string
	Fee          string
	Timestamp    string
	ErrorMessage string
}

// NewIndexingService creates a new indexing service
func NewIndexingService(
	ordersClient *clients.OrdersClient,
	solrRepo repositories.SearchRepository,
	logger *logrus.Logger,
) *IndexingService {
	return &IndexingService{
		ordersClient: ordersClient,
		solrRepo:     solrRepo,
		logger:       logger,
	}
}

// SyncOrderFromEvent synchronizes an order from a RabbitMQ event
// It invokes orders-api to get the complete order data and then indexes it
func (s *IndexingService) SyncOrderFromEvent(ctx context.Context, orderID string, eventType string, legacy *LegacyOrderEvent) error {
	s.logger.WithFields(logrus.Fields{
		"order_id":   orderID,
		"event_type": eventType,
	}).Info("Syncing order from event")

	// For delete events, remove from index
	if eventType == "orders.cancelled" || eventType == "orders.failed" {
		// Check if we should delete - only delete if explicitly cancelled
		// For failed orders, we might want to keep them indexed for search
		if eventType == "orders.cancelled" {
			return s.DeleteOrder(ctx, orderID)
		}
		// For failed orders, update the status but keep indexed
	}

	// For create/update/execute events, fetch complete order from orders-api
	orderResp, err := s.ordersClient.GetOrderByID(ctx, orderID)
	var order *models.Order
	if err != nil {
		if legacy == nil {
			return fmt.Errorf("failed to fetch order from orders-api: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"order_id":   orderID,
			"event_type": eventType,
			"error":      err,
		}).Warn("Falling back to event payload for indexing")

		order = s.orderFromLegacyEvent(legacy, eventType)
	} else {
		order = s.convertToOrderModel(orderResp)
	}

	// Index the order in SolR
	if err := s.IndexOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to index order: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"order_id":   orderID,
		"event_type": eventType,
	}).Info("Order synced successfully")

	return nil
}

// IndexOrder indexes an order in SolR
func (s *IndexingService) IndexOrder(ctx context.Context, order *models.Order) error {
	// Build searchable text from order fields
	searchText := s.buildSearchText(order)
	order.SearchText = searchText

	// Convert to SolR document format
	solrDoc := s.orderToSolrDoc(order)

	// Index in SolR
	s.logger.WithFields(logrus.Fields{
		"order_id": order.ID,
		"doc":      solrDoc,
	}).Info("Prepared Solr document for indexing")

	if err := s.solrRepo.IndexOrder(ctx, solrDoc); err != nil {
		return fmt.Errorf("failed to index order in SolR: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"order_id": order.ID,
		"status":   order.Status,
	}).Debug("Order indexed successfully")

	return nil
}

// DeleteOrder removes an order from the search index
func (s *IndexingService) DeleteOrder(ctx context.Context, orderID string) error {
	if err := s.solrRepo.DeleteOrderByID(ctx, orderID); err != nil {
		return fmt.Errorf("failed to delete order from SolR: %w", err)
	}

	s.logger.WithField("order_id", orderID).Debug("Order deleted from index")
	return nil
}

// convertToOrderModel converts OrderResponse from orders-api to Order model
func (s *IndexingService) convertToOrderModel(resp *clients.OrderResponse) *models.Order {
	order := &models.Order{
		ID:           resp.ID,
		UserID:       resp.UserID,
		Type:         resp.Type,
		Status:       resp.Status,
		OrderKind:    resp.OrderKind,
		CryptoSymbol: resp.CryptoSymbol,
		CryptoName:   resp.CryptoName,
		Quantity:     resp.Quantity,
		Price:        resp.OrderPrice,
		TotalAmount:  resp.TotalAmount,
		Fee:          resp.Fee,
		CreatedAt:    resp.CreatedAt,
		UpdatedAt:    resp.UpdatedAt,
	}

	if resp.ExecutedAt != nil {
		order.ExecutedAt = resp.ExecutedAt
	}

	if resp.CancelledAt != nil {
		order.CancelledAt = resp.CancelledAt
	}

	return order
}

// orderFromLegacyEvent builds a minimal order model from legacy event data
func (s *IndexingService) orderFromLegacyEvent(evt *LegacyOrderEvent, eventType string) *models.Order {
	timestamp := time.Now()
	if evt.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, evt.Timestamp); err == nil {
			timestamp = parsed
		}
	}

	order := &models.Order{
		ID:           evt.OrderID,
		UserID:       evt.UserID,
		Type:         evt.Type,
		Status:       evt.Status,
		OrderKind:    "",
		CryptoSymbol: evt.CryptoSymbol,
		CryptoName:   strings.ToUpper(evt.CryptoSymbol),
		Quantity:     evt.Quantity,
		Price:        evt.Price,
		TotalAmount:  evt.TotalAmount,
		Fee:          evt.Fee,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
		ErrorMessage: evt.ErrorMessage,
	}

	if strings.HasSuffix(eventType, "orders.executed") {
		order.ExecutedAt = &timestamp
	}
	if strings.HasSuffix(eventType, "orders.cancelled") {
		order.CancelledAt = &timestamp
	}

	return order
}

// buildSearchText builds a searchable text field from order data
func (s *IndexingService) buildSearchText(order *models.Order) string {
	parts := []string{
		order.ID,
		order.CryptoSymbol,
		order.CryptoName,
		order.Type,
		order.Status,
		order.OrderKind,
		order.Quantity,
		order.Price,
		order.TotalAmount,
	}

	return strings.Join(parts, " ")
}

// orderToSolrDoc converts Order model to SolR document format
func (s *IndexingService) orderToSolrDoc(order *models.Order) map[string]interface{} {
	totalAmountValue, err := strconv.ParseFloat(order.TotalAmount, 64)
	if err != nil {
		totalAmountValue = 0
	}

	quantityValue, err := strconv.ParseFloat(order.Quantity, 64)
	if err != nil {
		quantityValue = 0
	}

	priceValue, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		priceValue = 0
	}

	feeValue, err := strconv.ParseFloat(order.Fee, 64)
	if err != nil {
		feeValue = 0
	}

	doc := map[string]interface{}{
		"id":                     order.ID,
		"user_id":                order.UserID,
		"type":                   order.Type,
		"status":                 order.Status,
		"order_kind":             order.OrderKind,
		"crypto_symbol":          order.CryptoSymbol,
		"crypto_name":            order.CryptoName,
		"quantity":               quantityValue,
		"quantity_display_s":     order.Quantity,
		"price":                  priceValue,
		"price_display_s":        order.Price,
		"total_amount_display":   totalAmountValue,
		"total_amount_display_s": order.TotalAmount,
		"total_amount_value":     totalAmountValue,
		"fee":                    feeValue,
		"fee_display_s":          order.Fee,
		"created_at":             order.CreatedAt.Format(time.RFC3339),
		"updated_at":             order.UpdatedAt.Format(time.RFC3339),
		"search_text":            order.SearchText,
	}

	if order.ExecutedAt != nil {
		doc["executed_at"] = order.ExecutedAt.Format(time.RFC3339)
	}

	if order.CancelledAt != nil {
		doc["cancelled_at"] = order.CancelledAt.Format(time.RFC3339)
	}

	if order.ErrorMessage != "" {
		doc["error_message"] = order.ErrorMessage
	}

	return doc
}
