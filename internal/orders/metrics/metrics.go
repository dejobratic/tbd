package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	ordersCreatedTotal    metric.Int64Counter
	orderCreationDuration metric.Float64Histogram
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.ordersCreatedTotal, err = meter.Int64Counter(
		"orders_created_total",
		metric.WithDescription("Total number of orders created"),
		metric.WithUnit("{order}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create orders_created_total counter: %w", err)
	}

	m.orderCreationDuration, err = meter.Float64Histogram(
		"order_creation_duration_seconds",
		metric.WithDescription("Duration of order creation operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create order_creation_duration histogram: %w", err)
	}

	return m, nil
}

func (m *Metrics) RecordOrderCreated(ctx context.Context, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	m.ordersCreatedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *Metrics) RecordOrderCreationDuration(ctx context.Context, durationSeconds float64) {
	m.orderCreationDuration.Record(ctx, durationSeconds)
}
