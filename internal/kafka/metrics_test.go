package kafka

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestInitializeMetrics(t *testing.T) {
	t.Run("initializes all metric instruments successfully", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		if metrics == nil {
			t.Fatal("NewMetrics() returned nil")
		}

		if metrics.producerLatency == nil {
			t.Error("producerLatency is nil")
		}
	})
}

func TestRecordKafkaPublish(t *testing.T) {
	t.Run("records publish latency with topic and status labels", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		ctx := context.Background()

		metrics.RecordPublish(ctx, "order.created", 0.2, true)
		metrics.RecordPublish(ctx, "order.failed", 0.3, false)

		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		found := false
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "kafka_producer_latency_seconds" {
					found = true
					histogram, ok := m.Data.(metricdata.Histogram[float64])
					if !ok {
						t.Fatal("Expected Histogram[float64] data type")
					}
					if len(histogram.DataPoints) != 2 {
						t.Errorf("Expected 2 data points, got %d", len(histogram.DataPoints))
					}
				}
			}
		}

		if !found {
			t.Error("kafka_producer_latency_seconds metric not found")
		}
	})
}
