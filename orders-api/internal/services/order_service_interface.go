package services

import (
	"context"
	"orders-api/internal/dto"
	"orders-api/internal/models"
)

// OrderService interface que define las operaciones de órdenes
type OrderService interface {
	CreateOrder(ctx context.Context, req *dto.CreateOrderRequest, userID int) (*models.Order, error)
	GetOrder(ctx context.Context, orderID string, userID int) (*models.Order, error)
	ListUserOrders(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, *dto.OrdersSummary, error)
	CancelOrder(ctx context.Context, orderID string, userID int, reason string) error
}
