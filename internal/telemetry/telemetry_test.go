package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Run("returns error when service name is missing", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "",
			ServiceVersion: "1.0.0",
			SampleRate:     1.0,
		}

		err := cfg.Validate()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("expected ErrInvalidConfig, got %v", err)
		}
		if !errors.Is(err, ErrMissingServiceName) {
			t.Errorf("expected ErrMissingServiceName, got %v", err)
		}
	})

	t.Run("returns error when service version is missing", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "",
			SampleRate:     1.0,
		}

		err := cfg.Validate()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("expected ErrInvalidConfig, got %v", err)
		}
		if !errors.Is(err, ErrMissingServiceVersion) {
			t.Errorf("expected ErrMissingServiceVersion, got %v", err)
		}
	})

	t.Run("returns error when sample rate is negative", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			SampleRate:     -0.1,
		}

		err := cfg.Validate()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("expected ErrInvalidConfig, got %v", err)
		}
		if !errors.Is(err, ErrInvalidSampleRate) {
			t.Errorf("expected ErrInvalidSampleRate, got %v", err)
		}
	})

	t.Run("returns error when sample rate is greater than 1.0", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			SampleRate:     1.1,
		}

		err := cfg.Validate()

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("expected ErrInvalidConfig, got %v", err)
		}
		if !errors.Is(err, ErrInvalidSampleRate) {
			t.Errorf("expected ErrInvalidSampleRate, got %v", err)
		}
	})

	t.Run("validates successfully with valid config when telemetry disabled", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			EnableTracing:  false,
			EnableMetrics:  false,
			SampleRate:     0.5,
		}

		err := cfg.Validate()

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("validates successfully with sample rate 0.0", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			SampleRate:     0.0,
		}

		err := cfg.Validate()

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("validates successfully with sample rate 1.0", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			SampleRate:     1.0,
		}

		err := cfg.Validate()

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestInitialize(t *testing.T) {
	t.Run("returns error when config is invalid", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "",
			ServiceVersion: "1.0.0",
		}

		tel, err := Initialize(ctx, cfg)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if tel != nil {
			t.Error("expected nil telemetry, got non-nil")
		}
	})

	t.Run("initializes successfully with tracing enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			EnableTracing:  true,
			EnableMetrics:  false,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithTraceExporter(NewNoopTraceExporter()))

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tel == nil {
			t.Fatal("expected telemetry, got nil")
		}
		if tel.TracerProvider() == nil {
			t.Error("expected tracer provider, got nil")
		}
		if tel.MeterProvider() != nil {
			t.Error("expected nil meter provider, got non-nil")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	})

	t.Run("initializes successfully with metrics enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			EnableTracing:  false,
			EnableMetrics:  true,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithMetricExporter(NewNoopMetricExporter()))

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tel == nil {
			t.Fatal("expected telemetry, got nil")
		}
		if tel.TracerProvider() != nil {
			t.Error("expected nil tracer provider, got non-nil")
		}
		if tel.MeterProvider() == nil {
			t.Error("expected meter provider, got nil")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	})

	t.Run("initializes successfully with both tracing and metrics enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			EnableTracing:  true,
			EnableMetrics:  true,
			SampleRate:     0.5,
		}

		tel, err := Initialize(ctx, cfg,
			WithTraceExporter(NewNoopTraceExporter()),
			WithMetricExporter(NewNoopMetricExporter()),
		)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tel == nil {
			t.Fatal("expected telemetry, got nil")
		}
		if tel.TracerProvider() == nil {
			t.Error("expected tracer provider, got nil")
		}
		if tel.MeterProvider() == nil {
			t.Error("expected meter provider, got nil")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	})

	t.Run("initializes successfully with neither tracing nor metrics enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			EnableTracing:  false,
			EnableMetrics:  false,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tel == nil {
			t.Fatal("expected telemetry, got nil")
		}
		if tel.TracerProvider() != nil {
			t.Error("expected nil tracer provider, got non-nil")
		}
		if tel.MeterProvider() != nil {
			t.Error("expected nil meter provider, got non-nil")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	})
}

func TestCreateSampler(t *testing.T) {
	t.Run("returns sampler when sample rate is 0.0", func(t *testing.T) {
		sampler := createSampler(0.0)

		if sampler == nil {
			t.Error("expected sampler, got nil")
		}
		if sampler.Description() != "AlwaysOffSampler" {
			t.Errorf("expected AlwaysOffSampler, got %s", sampler.Description())
		}
	})

	t.Run("returns sampler when sample rate is negative", func(t *testing.T) {
		sampler := createSampler(-0.1)

		if sampler == nil {
			t.Error("expected sampler, got nil")
		}
		if sampler.Description() != "AlwaysOffSampler" {
			t.Errorf("expected AlwaysOffSampler, got %s", sampler.Description())
		}
	})

	t.Run("returns sampler when sample rate is 1.0", func(t *testing.T) {
		sampler := createSampler(1.0)

		if sampler == nil {
			t.Error("expected sampler, got nil")
		}
		if sampler.Description() != "AlwaysOnSampler" {
			t.Errorf("expected AlwaysOnSampler, got %s", sampler.Description())
		}
	})

	t.Run("returns sampler when sample rate is greater than 1.0", func(t *testing.T) {
		sampler := createSampler(1.5)

		if sampler == nil {
			t.Error("expected sampler, got nil")
		}
		if sampler.Description() != "AlwaysOnSampler" {
			t.Errorf("expected AlwaysOnSampler, got %s", sampler.Description())
		}
	})

	t.Run("returns sampler when sample rate is between 0.0 and 1.0", func(t *testing.T) {
		sampler := createSampler(0.5)

		if sampler == nil {
			t.Error("expected sampler, got nil")
		}
	})
}

func TestShutdown(t *testing.T) {
	t.Run("shuts down successfully when no providers are initialized", func(t *testing.T) {
		tel := &Telemetry{}
		ctx := context.Background()

		err := tel.Shutdown(ctx)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("shuts down successfully when only tracing is enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			EnableTracing:  true,
			EnableMetrics:  false,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithTraceExporter(NewNoopTraceExporter()))
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = tel.Shutdown(shutdownCtx)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("shuts down successfully when only metrics is enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			EnableTracing:  false,
			EnableMetrics:  true,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithMetricExporter(NewNoopMetricExporter()))
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = tel.Shutdown(shutdownCtx)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("shuts down successfully when both are enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			EnableTracing:  true,
			EnableMetrics:  true,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg,
			WithTraceExporter(NewNoopTraceExporter()),
			WithMetricExporter(NewNoopMetricExporter()),
		)
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = tel.Shutdown(shutdownCtx)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestGetterMethods(t *testing.T) {
	t.Run("TracerProvider returns nil when tracing not enabled", func(t *testing.T) {
		tel := &Telemetry{}

		if tel.TracerProvider() != nil {
			t.Error("expected nil, got non-nil")
		}
	})

	t.Run("MeterProvider returns nil when metrics not enabled", func(t *testing.T) {
		tel := &Telemetry{}

		if tel.MeterProvider() != nil {
			t.Error("expected nil, got non-nil")
		}
	})

	t.Run("TracerProvider returns provider when tracing enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			EnableTracing:  true,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithTraceExporter(NewNoopTraceExporter()))
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = tel.Shutdown(shutdownCtx)
		}()

		if tel.TracerProvider() == nil {
			t.Error("expected tracer provider, got nil")
		}
	})

	t.Run("MeterProvider returns provider when metrics enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			EnableMetrics:  true,
			SampleRate:     1.0,
		}

		tel, err := Initialize(ctx, cfg, WithMetricExporter(NewNoopMetricExporter()))
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = tel.Shutdown(shutdownCtx)
		}()

		if tel.MeterProvider() == nil {
			t.Error("expected meter provider, got nil")
		}
	})
}
