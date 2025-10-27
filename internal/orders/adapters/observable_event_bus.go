package adapters

import (
	"context"
	"time"

	"github.com/dejobratic/tbd/internal/kafka"
	"github.com/dejobratic/tbd/internal/orders/ports"
	"github.com/dejobratic/tbd/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type ObservableEventBus struct {
	bus     ports.EventBus
	metrics *kafka.Metrics
}

func NewObservableEventBus(bus ports.EventBus, metrics *kafka.Metrics) *ObservableEventBus {
	return &ObservableEventBus{
		bus:     bus,
		metrics: metrics,
	}
}

func (e *ObservableEventBus) PublishOrderCreated(ctx context.Context, orderID string) error {
	ctx, span := telemetry.StartSpan(ctx, "EventBus.PublishOrderCreated")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", orderID),
		attribute.String("event.type", "order.created"),
		attribute.String("topic", "order.created"),
	)

	start := time.Now()
	err := e.bus.PublishOrderCreated(ctx, orderID)
	duration := time.Since(start).Seconds()

	e.metrics.RecordPublish(ctx, "order.created", duration, err == nil)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}

	telemetry.SetSpanSuccess(span)
	return nil
}

func (e *ObservableEventBus) PublishOrderProcessed(ctx context.Context, orderID string) error {
	ctx, span := telemetry.StartSpan(ctx, "EventBus.PublishOrderProcessed")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", orderID),
		attribute.String("event.type", "order.processed"),
		attribute.String("topic", "order.processed"),
	)

	start := time.Now()
	err := e.bus.PublishOrderProcessed(ctx, orderID)
	duration := time.Since(start).Seconds()

	e.metrics.RecordPublish(ctx, "order.processed", duration, err == nil)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}

	telemetry.SetSpanSuccess(span)
	return nil
}

func (e *ObservableEventBus) PublishOrderFailed(ctx context.Context, orderID string, reason string) error {
	ctx, span := telemetry.StartSpan(ctx, "EventBus.PublishOrderFailed")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", orderID),
		attribute.String("event.type", "order.failed"),
		attribute.String("topic", "order.failed"),
		attribute.String("failure.reason", reason),
	)

	start := time.Now()
	err := e.bus.PublishOrderFailed(ctx, orderID, reason)
	duration := time.Since(start).Seconds()

	e.metrics.RecordPublish(ctx, "order.failed", duration, err == nil)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}

	telemetry.SetSpanSuccess(span)
	return nil
}
