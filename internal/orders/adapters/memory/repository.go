package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

// Repository provides an in-memory store useful for local development and tests.
type Repository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

// NewRepository constructs a new in-memory repository.
func NewRepository() *Repository {
	return &Repository{orders: make(map[string]domain.Order)}
}

// Create stores a new order instance.
func (r *Repository) Create(_ context.Context, order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

// GetByID fetches a single order by identifier.
func (r *Repository) GetByID(_ context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return nil, ports.ErrNotFound
	}
	copy := order
	return &copy, nil
}

// List returns orders respecting the provided filter. Pagination is 1-based.
func (r *Repository) List(_ context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Order
	for _, order := range r.orders {
		if filter.Status != nil && order.Status != *filter.Status {
			continue
		}
		result = append(result, order)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	if start >= len(result) {
		return []domain.Order{}, nil
	}

	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}

	slice := make([]domain.Order, end-start)
	copy(slice, result[start:end])

	return slice, nil
}

// UpdateStatus sets the status and updatedAt timestamp for an order.
func (r *Repository) UpdateStatus(_ context.Context, id string, status domain.OrderStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[id]
	if !ok {
		return ports.ErrNotFound
	}

	order.Status = status
	order.UpdatedAt = time.Now().UTC()
	r.orders[id] = order
	return nil
}
