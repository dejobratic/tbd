package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dejobratic/tbd/internal/orders/app/commands"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

type mockRepository struct {
	createFn func(ctx context.Context, order domain.Order) error
}

func (m *mockRepository) Create(ctx context.Context, order domain.Order) error {
	if m.createFn != nil {
		return m.createFn(ctx, order)
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	return nil, nil
}

func (m *mockRepository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	return nil, nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	return nil
}

type mockEventBus struct {
	publishOrderCreatedFn func(ctx context.Context, orderID string) error
}

func (m *mockEventBus) PublishOrderCreated(ctx context.Context, orderID string) error {
	if m.publishOrderCreatedFn != nil {
		return m.publishOrderCreatedFn(ctx, orderID)
	}
	return nil
}

func (m *mockEventBus) PublishOrderProcessed(ctx context.Context, orderID string) error {
	return nil
}

func (m *mockEventBus) PublishOrderFailed(ctx context.Context, orderID string, reason string) error {
	return nil
}

func TestCreateOrder(t *testing.T) {
	t.Run("creates pending order with valid input", func(t *testing.T) {
		repo := &mockRepository{}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "test@example.com",
			AmountCents:   1000,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if order == nil {
			t.Fatal("expected order to be returned, got nil")
		}

		if order.CustomerEmail != cmd.CustomerEmail {
			t.Errorf("expected customer email %s, got %s", cmd.CustomerEmail, order.CustomerEmail)
		}

		if order.AmountCents != cmd.AmountCents {
			t.Errorf("expected amount %d, got %d", cmd.AmountCents, order.AmountCents)
		}

		if order.Status != domain.StatusPending {
			t.Errorf("expected status %s, got %s", domain.StatusPending, order.Status)
		}

		if order.ID == "" {
			t.Error("expected order ID to be generated")
		}
	})

	t.Run("returns validation error when email is empty", func(t *testing.T) {
		repo := &mockRepository{}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "",
			AmountCents:   1000,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "customer_email is required" {
			t.Errorf("expected error %q, got %q", "customer_email is required", err.Error())
		}

		if order != nil {
			t.Errorf("expected nil order, got %+v", order)
		}
	})

	t.Run("returns validation error when email is invalid", func(t *testing.T) {
		repo := &mockRepository{}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "invalid-email",
			AmountCents:   1000,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "customer_email must be valid" {
			t.Errorf("expected error %q, got %q", "customer_email must be valid", err.Error())
		}

		if order != nil {
			t.Errorf("expected nil order, got %+v", order)
		}
	})

	t.Run("returns validation error when amount is zero", func(t *testing.T) {
		repo := &mockRepository{}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "test@example.com",
			AmountCents:   0,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "amount_cents must be positive" {
			t.Errorf("expected error %q, got %q", "amount_cents must be positive", err.Error())
		}

		if order != nil {
			t.Errorf("expected nil order, got %+v", order)
		}
	})

	t.Run("returns validation error when amount is negative", func(t *testing.T) {
		repo := &mockRepository{}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "test@example.com",
			AmountCents:   -100,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "amount_cents must be positive" {
			t.Errorf("expected error %q, got %q", "amount_cents must be positive", err.Error())
		}

		if order != nil {
			t.Errorf("expected nil order, got %+v", order)
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repoErr := errors.New("database connection failed")
		repo := &mockRepository{
			createFn: func(ctx context.Context, order domain.Order) error {
				return repoErr
			},
		}
		events := &mockEventBus{}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "test@example.com",
			AmountCents:   1000,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, repoErr) {
			t.Errorf("expected error to wrap repository error, got: %v", err)
		}

		if order != nil {
			t.Errorf("expected nil order, got %+v", order)
		}
	})

	t.Run("returns order even when event publishing fails", func(t *testing.T) {
		eventErr := errors.New("kafka unavailable")
		repo := &mockRepository{}
		events := &mockEventBus{
			publishOrderCreatedFn: func(ctx context.Context, orderID string) error {
				return eventErr
			},
		}
		handler := commands.NewCreateOrderCommandHandler(repo, events)

		cmd := commands.CreateOrderCommand{
			CustomerEmail: "test@example.com",
			AmountCents:   1000,
		}

		order, err := handler.Handle(context.Background(), cmd)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order == nil {
			t.Fatal("expected order to be returned even on event bus error")
		}

		if order.CustomerEmail != cmd.CustomerEmail {
			t.Errorf("expected customer email %s, got %s", cmd.CustomerEmail, order.CustomerEmail)
		}
	})
}
