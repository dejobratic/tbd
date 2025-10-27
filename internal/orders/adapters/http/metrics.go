package http

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	requestDuration metric.Float64Histogram
	requestsTotal   metric.Int64Counter
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.requestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create http_request_duration histogram: %w", err)
	}

	m.requestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create http_requests_total counter: %w", err)
	}

	return m, nil
}

func (m *Metrics) RecordRequest(ctx context.Context, method, path string, statusCode int, durationSeconds float64) {
	m.requestsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", statusCode),
	))
	m.requestDuration.Record(ctx, durationSeconds, metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("path", path),
	))
}
