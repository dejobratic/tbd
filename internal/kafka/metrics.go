package kafka

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	producerLatency metric.Float64Histogram
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.producerLatency, err = meter.Float64Histogram(
		"kafka_producer_latency_seconds",
		metric.WithDescription("Kafka producer latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka_producer_latency histogram: %w", err)
	}

	return m, nil
}

func (m *Metrics) RecordPublish(ctx context.Context, topic string, durationSeconds float64, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	m.producerLatency.Record(ctx, durationSeconds, metric.WithAttributes(
		attribute.String("topic", topic),
		attribute.String("status", status),
	))
}
