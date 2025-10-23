package kafka

import (
	"context"
	"log/slog"
)

// NoopEventBus logs events without sending them to Kafka. Useful for local dev before wiring Kafka.
type NoopEventBus struct{}

// NewNoopEventBus returns a new no-op event publisher.
func NewNoopEventBus() *NoopEventBus {
	return &NoopEventBus{}
}

func (n *NoopEventBus) PublishOrderCreated(_ context.Context, orderID string) error {
	slog.Debug("event::order_created", "order_id", orderID)
	return nil
}

func (n *NoopEventBus) PublishOrderProcessed(_ context.Context, orderID string) error {
	slog.Debug("event::order_processed", "order_id", orderID)
	return nil
}

func (n *NoopEventBus) PublishOrderFailed(_ context.Context, orderID string, reason string) error {
	slog.Debug("event::order_failed", "order_id", orderID, "reason", reason)
	return nil
}
