package queries

import (
	"context"
	"errors"
	"strings"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

// GetOrderQuery represents a request to retrieve an order by its ID.
type GetOrderQuery struct {
	OrderID string
}

// GetOrderQueryHandler executes GetOrderQuery and returns the order if found.
type GetOrderQueryHandler struct {
	repo ports.OrderRepository
}

// NewGetOrderQueryHandler constructs a GetOrderQueryHandler.
func NewGetOrderQueryHandler(repo ports.OrderRepository) *GetOrderQueryHandler {
	return &GetOrderQueryHandler{repo: repo}
}

// Handle executes the query and retrieves the order.
func (h *GetOrderQueryHandler) Handle(ctx context.Context, query GetOrderQuery) (*domain.Order, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	order, err := h.repo.GetByID(ctx, query.OrderID)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// Validate ensures the query has valid parameters.
func (q GetOrderQuery) Validate() error {
	if strings.TrimSpace(q.OrderID) == "" {
		return errors.New("order_id is required")
	}
	return nil
}
