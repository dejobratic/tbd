package domain_test

import (
	"testing"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
)

func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name    string
		order   domain.Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: domain.Order{
				ID:            "test-id",
				CustomerEmail: "user@example.com",
				AmountCents:   1000,
				Status:        domain.StatusPending,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing email",
			order: domain.Order{
				ID:          "test-id",
				AmountCents: 1000,
				Status:      domain.StatusPending,
			},
			wantErr: true,
		},
		{
			name: "whitespace only email",
			order: domain.Order{
				ID:            "test-id",
				CustomerEmail: "   ",
				AmountCents:   1000,
				Status:        domain.StatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			order: domain.Order{
				ID:            "test-id",
				CustomerEmail: "notanemail",
				AmountCents:   1000,
				Status:        domain.StatusPending,
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			order: domain.Order{
				ID:            "test-id",
				CustomerEmail: "user@example.com",
				AmountCents:   0,
				Status:        domain.StatusPending,
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			order: domain.Order{
				ID:            "test-id",
				CustomerEmail: "user@example.com",
				AmountCents:   -100,
				Status:        domain.StatusPending,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Order.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderIsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status domain.OrderStatus
		want   bool
	}{
		{"completed is terminal", domain.StatusCompleted, true},
		{"failed is terminal", domain.StatusFailed, true},
		{"canceled is terminal", domain.StatusCanceled, true},
		{"pending is not terminal", domain.StatusPending, false},
		{"processing is not terminal", domain.StatusProcessing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := domain.Order{Status: tt.status}
			if got := order.IsTerminal(); got != tt.want {
				t.Errorf("Order.IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}
