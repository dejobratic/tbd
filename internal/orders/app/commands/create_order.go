package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

type CreateOrderCommand struct {
	CustomerEmail string
	AmountCents   int64
}

func (c CreateOrderCommand) Validate() error {
	if strings.TrimSpace(c.CustomerEmail) == "" {
		return errors.New("customer_email is required")
	}
	if !strings.Contains(c.CustomerEmail, "@") {
		return errors.New("customer_email must be valid")
	}
	if c.AmountCents <= 0 {
		return errors.New("amount_cents must be positive")
	}
	return nil
}

type CommandHandler interface {
	Handle(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error)
}

type CreateOrderCommandHandler struct {
	repo   ports.OrderRepository
	events ports.EventBus
}

func NewCreateOrderCommandHandler(
	repo ports.OrderRepository,
	events ports.EventBus,
) *CreateOrderCommandHandler {
	return &CreateOrderCommandHandler{
		repo:   repo,
		events: events,
	}
}

func (h *CreateOrderCommandHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	orderID, err := generateOrderID()
	if err != nil {
		return nil, err
	}

	order := domain.Order{
		ID:            orderID,
		CustomerEmail: cmd.CustomerEmail,
		AmountCents:   cmd.AmountCents,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := h.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	if err := h.events.PublishOrderCreated(ctx, order.ID); err != nil {
		return &order, fmt.Errorf("order saved but failed to publish event: %w", err)
	}

	return &order, nil
}

func generateOrderID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate order id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
