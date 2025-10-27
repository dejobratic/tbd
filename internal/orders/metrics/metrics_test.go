package metrics

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

		if metrics.ordersCreatedTotal == nil {
			t.Error("ordersCreatedTotal is nil")
		}

		if metrics.orderCreationDuration == nil {
			t.Error("orderCreationDuration is nil")
		}
	})
}

func TestRecordOrderCreated(t *testing.T) {
	t.Run("records order creation count with success status", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		ctx := context.Background()

		metrics.RecordOrderCreated(ctx, true)
		metrics.RecordOrderCreated(ctx, false)

		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		found := false
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "orders_created_total" {
					found = true
					sum, ok := m.Data.(metricdata.Sum[int64])
					if !ok {
						t.Fatal("Expected Sum[int64] data type")
					}
					if len(sum.DataPoints) != 2 {
						t.Errorf("Expected 2 data points, got %d", len(sum.DataPoints))
					}
				}
			}
		}

		if !found {
			t.Error("orders_created_total metric not found")
		}
	})
}

func TestRecordOrderCreationDuration(t *testing.T) {
	t.Run("records order creation duration", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		ctx := context.Background()

		metrics.RecordOrderCreationDuration(ctx, 1.5)
		metrics.RecordOrderCreationDuration(ctx, 2.3)

		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		found := false
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "order_creation_duration_seconds" {
					found = true
					histogram, ok := m.Data.(metricdata.Histogram[float64])
					if !ok {
						t.Fatal("Expected Histogram[float64] data type")
					}
					if len(histogram.DataPoints) != 1 {
						t.Errorf("Expected 1 data point, got %d", len(histogram.DataPoints))
					}
					if histogram.DataPoints[0].Count != 2 {
						t.Errorf("Expected count=2, got %d", histogram.DataPoints[0].Count)
					}
				}
			}
		}

		if !found {
			t.Error("order_creation_duration_seconds metric not found")
		}
	})
}
