package ports

import (
	"context"
	"errors"

	"github.com/dejobratic/tbd/internal/orders/domain"
)

// OrderRepository exposes persistence operations required by the application layer.
type OrderRepository interface {
	Create(ctx context.Context, order domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Order, error)
	UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error
}

// ListFilter narrows list queries by status and pagination.
type ListFilter struct {
	Status   *domain.OrderStatus
	Page     int
	PageSize int
}

var (
	// ErrNotFound is returned when the requested order does not exist.
	ErrNotFound = errors.New("order not found")
)
