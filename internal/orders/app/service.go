package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

// Service bundles use cases for handling orders via the API.
type Service struct {
	repo      ports.OrderRepository
	events    ports.EventBus
	idemStore ports.IdempotencyStore
}

// NewService wires required dependencies.
func NewService(repo ports.OrderRepository, events ports.EventBus, idem ports.IdempotencyStore) *Service {
	return &Service{repo: repo, events: events, idemStore: idem}
}

// CreateOrderInput captures payload for creating an order.
type CreateOrderInput struct {
	CustomerEmail string `json:"customer_email"`
	AmountCents   int64  `json:"amount_cents"`
}

// CreateOrder orchestrates order creation and event emission.
func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	orderID, err := generateOrderID()
	if err != nil {
		return nil, err
	}

	order := domain.Order{
		ID:            orderID,
		CustomerEmail: input.CustomerEmail,
		AmountCents:   input.AmountCents,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	// For skeleton implementation we log errors but do not fail the request.
	if err := s.events.PublishOrderCreated(ctx, order.ID); err != nil {
		return &order, fmt.Errorf("order saved but failed to publish event: %w", err)
	}

	return &order, nil
}

// GetOrder retrieves an order by ID.
func (s *Service) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return s.repo.GetByID(ctx, id)
}

// ListOrders returns orders using a filter.
func (s *Service) ListOrders(ctx context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	return s.repo.List(ctx, filter)
}

// CancelOrder attempts to cancel a pending order.
func (s *Service) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.StatusPending {
		return nil, fmt.Errorf("cannot cancel order in status %s", order.Status)
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.StatusCanceled); err != nil {
		return nil, err
	}

	order.Status = domain.StatusCanceled
	order.UpdatedAt = time.Now().UTC()

	return order, nil
}

// SaveIdempotentResponse writes response details for a key.
func (s *Service) SaveIdempotentResponse(ctx context.Context, key string, response ports.StoredResponse) error {
	return s.idemStore.Save(ctx, key, response)
}

// GetIdempotentResponse retrieves previously stored response data.
func (s *Service) GetIdempotentResponse(ctx context.Context, key string) (*ports.StoredResponse, error) {
	return s.idemStore.Get(ctx, key)
}

func generateOrderID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate order id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
