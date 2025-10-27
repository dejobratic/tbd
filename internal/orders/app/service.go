package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dejobratic/tbd/internal/orders/app/commands"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/metrics"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

// Service bundles use cases for handling orders via the API.
type Service struct {
	repo               ports.OrderRepository
	events             ports.EventBus
	idemStore          ports.IdempotencyStore
	createOrderHandler commands.CommandHandler
}

// NewService wires required dependencies.
func NewService(
	repo ports.OrderRepository,
	events ports.EventBus,
	idem ports.IdempotencyStore,
	logger *slog.Logger,
	metrics *metrics.Metrics,
) *Service {
	coreHandler := commands.NewCreateOrderCommandHandler(repo, events)
	observableHandler := commands.NewObservableCommandHandler(coreHandler, logger, metrics)

	return &Service{
		repo:               repo,
		events:             events,
		idemStore:          idem,
		createOrderHandler: observableHandler,
	}
}

// CreateOrderInput captures payload for creating an order.
type CreateOrderInput struct {
	CustomerEmail string `json:"customer_email"`
	AmountCents   int64  `json:"amount_cents"`
}

// CreateOrder orchestrates order creation and event emission.
func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	cmd := commands.CreateOrderCommand{
		CustomerEmail: input.CustomerEmail,
		AmountCents:   input.AmountCents,
	}
	return s.createOrderHandler.Handle(ctx, cmd)
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
