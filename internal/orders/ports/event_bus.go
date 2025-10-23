package ports

import "context"

// EventBus defines the contract for publishing order lifecycle events.
type EventBus interface {
	PublishOrderCreated(ctx context.Context, orderID string) error
	PublishOrderProcessed(ctx context.Context, orderID string) error
	PublishOrderFailed(ctx context.Context, orderID string, reason string) error
}
