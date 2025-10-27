package database

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	queryDuration metric.Float64Histogram
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.queryDuration, err = meter.Float64Histogram(
		"db_query_duration_seconds",
		metric.WithDescription("Database query duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create db_query_duration histogram: %w", err)
	}

	return m, nil
}

func (m *Metrics) RecordQuery(ctx context.Context, operation string, durationSeconds float64) {
	m.queryDuration.Record(ctx, durationSeconds, metric.WithAttributes(
		attribute.String("operation", operation),
	))
}
