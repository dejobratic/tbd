package queries_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dejobratic/tbd/internal/orders/app/queries"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

type inMemoryRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		orders: make(map[string]domain.Order),
	}
}

func (r *inMemoryRepository) Create(ctx context.Context, order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *inMemoryRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, exists := r.orders[id]
	if !exists {
		return nil, ports.ErrNotFound
	}
	return &order, nil
}

func (r *inMemoryRepository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	orders := make([]domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *inMemoryRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	order, exists := r.orders[id]
	if !exists {
		return ports.ErrNotFound
	}
	order.Status = status
	order.UpdatedAt = time.Now().UTC()
	r.orders[id] = order
	return nil
}

func TestGetOrder(t *testing.T) {
	t.Run("returns order by ID", func(t *testing.T) {
		repo := newInMemoryRepository()
		handler := queries.NewGetOrderQueryHandler(repo)
		ctx := context.Background()

		expectedOrder := domain.Order{
			ID:            "test-order-123",
			CustomerEmail: "test@example.com",
			AmountCents:   1999,
			Status:        domain.StatusPending,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		if err := repo.Create(ctx, expectedOrder); err != nil {
			t.Fatalf("failed to create test order: %v", err)
		}

		query := queries.GetOrderQuery{OrderID: "test-order-123"}
		result, err := handler.Handle(ctx, query)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected order to be returned, got nil")
		}

		if result.ID != expectedOrder.ID {
			t.Errorf("expected ID %s, got %s", expectedOrder.ID, result.ID)
		}

		if result.CustomerEmail != expectedOrder.CustomerEmail {
			t.Errorf("expected email %s, got %s", expectedOrder.CustomerEmail, result.CustomerEmail)
		}

		if result.AmountCents != expectedOrder.AmountCents {
			t.Errorf("expected amount %d, got %d", expectedOrder.AmountCents, result.AmountCents)
		}

		if result.Status != expectedOrder.Status {
			t.Errorf("expected status %s, got %s", expectedOrder.Status, result.Status)
		}
	})

	t.Run("returns not found error for nonexistent order", func(t *testing.T) {
		repo := newInMemoryRepository()
		handler := queries.NewGetOrderQueryHandler(repo)
		ctx := context.Background()

		query := queries.GetOrderQuery{OrderID: "nonexistent-order"}
		result, err := handler.Handle(ctx, query)

		if !errors.Is(err, ports.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}

		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("returns validation error when order ID is empty", func(t *testing.T) {
		repo := newInMemoryRepository()
		handler := queries.NewGetOrderQueryHandler(repo)
		ctx := context.Background()

		query := queries.GetOrderQuery{OrderID: ""}
		result, err := handler.Handle(ctx, query)

		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		if err.Error() != "order_id is required" {
			t.Errorf("expected 'order_id is required' error, got %v", err)
		}

		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("returns validation error when order ID is whitespace", func(t *testing.T) {
		repo := newInMemoryRepository()
		handler := queries.NewGetOrderQueryHandler(repo)
		ctx := context.Background()

		query := queries.GetOrderQuery{OrderID: "   "}
		result, err := handler.Handle(ctx, query)

		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		if err.Error() != "order_id is required" {
			t.Errorf("expected 'order_id is required' error, got %v", err)
		}

		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("retrieves correct order from multiple orders", func(t *testing.T) {
		repo := newInMemoryRepository()
		handler := queries.NewGetOrderQueryHandler(repo)
		ctx := context.Background()

		orders := []domain.Order{
			{
				ID:            "order-1",
				CustomerEmail: "user1@example.com",
				AmountCents:   1000,
				Status:        domain.StatusPending,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
			{
				ID:            "order-2",
				CustomerEmail: "user2@example.com",
				AmountCents:   2000,
				Status:        domain.StatusCompleted,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
			{
				ID:            "order-3",
				CustomerEmail: "user3@example.com",
				AmountCents:   3000,
				Status:        domain.StatusCanceled,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		}

		for _, order := range orders {
			if err := repo.Create(ctx, order); err != nil {
				t.Fatalf("failed to create order %s: %v", order.ID, err)
			}
		}

		for _, expectedOrder := range orders {
			query := queries.GetOrderQuery{OrderID: expectedOrder.ID}
			result, err := handler.Handle(ctx, query)

			if err != nil {
				t.Errorf("failed to get order %s: %v", expectedOrder.ID, err)
				continue
			}

			if result.ID != expectedOrder.ID {
				t.Errorf("expected ID %s, got %s", expectedOrder.ID, result.ID)
			}

			if result.Status != expectedOrder.Status {
				t.Errorf("expected status %s, got %s", expectedOrder.Status, result.Status)
			}
		}
	})
}

func TestGetOrderQueryValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   queries.GetOrderQuery
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid order ID",
			query:   queries.GetOrderQuery{OrderID: "order-123"},
			wantErr: false,
		},
		{
			name:    "empty order ID",
			query:   queries.GetOrderQuery{OrderID: ""},
			wantErr: true,
			errMsg:  "order_id is required",
		},
		{
			name:    "whitespace order ID",
			query:   queries.GetOrderQuery{OrderID: "  \t  "},
			wantErr: true,
			errMsg:  "order_id is required",
		},
		{
			name:    "valid UUID order ID",
			query:   queries.GetOrderQuery{OrderID: "550e8400-e29b-41d4-a716-446655440000"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
