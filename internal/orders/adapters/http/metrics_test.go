package http

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

		if metrics.requestDuration == nil {
			t.Error("requestDuration is nil")
		}

		if metrics.requestsTotal == nil {
			t.Error("requestsTotal is nil")
		}
	})
}

func TestRecordHTTPRequest(t *testing.T) {
	t.Run("records request count and duration with method, path, and status labels", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		meter := mp.Meter("test")

		metrics, err := NewMetrics(meter)
		if err != nil {
			t.Fatalf("NewMetrics() failed: %v", err)
		}

		ctx := context.Background()

		metrics.RecordRequest(ctx, "GET", "/orders", 200, 0.5)
		metrics.RecordRequest(ctx, "POST", "/orders", 201, 0.7)

		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		foundCounter := false
		foundHistogram := false

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "http_requests_total" {
					foundCounter = true
					sum, ok := m.Data.(metricdata.Sum[int64])
					if !ok {
						t.Fatal("Expected Sum[int64] data type")
					}
					if len(sum.DataPoints) != 2 {
						t.Errorf("Expected 2 data points, got %d", len(sum.DataPoints))
					}
				}
				if m.Name == "http_request_duration_seconds" {
					foundHistogram = true
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

		if !foundCounter {
			t.Error("http_requests_total metric not found")
		}
		if !foundHistogram {
			t.Error("http_request_duration_seconds metric not found")
		}
	})
}
