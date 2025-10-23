package domain

import (
	"errors"
	"strings"
	"time"
)

// OrderStatus captures the lifecycle of an order in the system.
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusProcessing OrderStatus = "processing"
	StatusCompleted  OrderStatus = "completed"
	StatusFailed     OrderStatus = "failed"
	StatusCanceled   OrderStatus = "canceled"
)

// Order represents a purchase request managed by the system.
type Order struct {
	ID            string      `json:"id"`
	CustomerEmail string      `json:"customer_email"`
	AmountCents   int64       `json:"amount_cents"`
	Status        OrderStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// Validate ensures the order adheres to business constraints.
func (o Order) Validate() error {
	if strings.TrimSpace(o.CustomerEmail) == "" {
		return errors.New("customer_email is required")
	}
	if !strings.Contains(o.CustomerEmail, "@") {
		return errors.New("customer_email must be valid")
	}
	if o.AmountCents <= 0 {
		return errors.New("amount_cents must be positive")
	}
	return nil
}

// IsTerminal indicates whether the order is in a terminal state.
func (o Order) IsTerminal() bool {
	switch o.Status {
	case StatusCompleted, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}
