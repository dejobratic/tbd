package database

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

		if metrics.queryDuration == nil {
			t.Error("queryDuration is nil")
		}
	})
}

func TestRecordDatabaseQuery(t *testing.T) {
	t.Run("records query duration with operation label", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		ctx := context.Background()

		metrics.RecordQuery(ctx, "create_order", 0.1)
		metrics.RecordQuery(ctx, "get_order_by_id", 0.05)

		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		found := false
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "db_query_duration_seconds" {
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
			t.Error("db_query_duration_seconds metric not found")
		}
	})
}
