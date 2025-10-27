package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// testConfig returns a valid Config for testing purposes.
func testConfig() Config {
	return Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		SampleRate:     1.0,
	}
}

// setupTelemetryWithTracing initializes telemetry with tracing enabled and returns cleanup function.
func setupTelemetryWithTracing(t *testing.T) (*Telemetry, func()) {
	t.Helper()

	ctx := context.Background()
	cfg := testConfig()
	cfg.EnableTracing = true
	cfg.EnableMetrics = false

	tel, err := Initialize(ctx, cfg, WithTraceExporter(NewNoopTraceExporter()))
	if err != nil {
		t.Fatalf("failed to initialize telemetry: %v", err)
	}

	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	}

	return tel, cleanup
}

// setupTelemetryWithMetrics initializes telemetry with metrics enabled and returns cleanup function.
func setupTelemetryWithMetrics(t *testing.T) (*Telemetry, func()) {
	t.Helper()

	ctx := context.Background()
	cfg := testConfig()
	cfg.EnableTracing = false
	cfg.EnableMetrics = true

	tel, err := Initialize(ctx, cfg, WithMetricExporter(NewNoopMetricExporter()))
	if err != nil {
		t.Fatalf("failed to initialize telemetry: %v", err)
	}

	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	}

	return tel, cleanup
}

// setupTelemetryWithBoth initializes telemetry with both tracing and metrics enabled.
func setupTelemetryWithBoth(t *testing.T) (*Telemetry, func()) {
	t.Helper()

	ctx := context.Background()
	cfg := testConfig()
	cfg.EnableTracing = true
	cfg.EnableMetrics = true

	tel, err := Initialize(ctx, cfg,
		WithTraceExporter(NewNoopTraceExporter()),
		WithMetricExporter(NewNoopMetricExporter()),
	)
	if err != nil {
		t.Fatalf("failed to initialize telemetry: %v", err)
	}

	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	}

	return tel, cleanup
}

// setupTracerProvider sets up an in-memory tracer provider for testing and returns cleanup function.
func setupTracerProvider(t *testing.T) (*tracetest.InMemoryExporter, func()) {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)

	cleanup := func() {
		otel.SetTracerProvider(nil)
	}

	return exp, cleanup
}
