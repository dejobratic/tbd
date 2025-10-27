package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	ErrInvalidConfig         = errors.New("invalid telemetry configuration")
	ErrMissingServiceName    = errors.New("service name is required")
	ErrMissingServiceVersion = errors.New("service version is required")
	ErrInvalidSampleRate     = errors.New("sample rate must be between 0.0 and 1.0")
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	EnableTracing  bool
	EnableMetrics  bool
	SampleRate     float64
}

type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	traceExporter  sdktrace.SpanExporter
	metricExporter sdkmetric.Exporter
}

type Option func(*telemetryOptions)

type telemetryOptions struct {
	traceExporter  sdktrace.SpanExporter
	metricExporter sdkmetric.Exporter
}

func WithTraceExporter(exporter sdktrace.SpanExporter) Option {
	return func(opts *telemetryOptions) {
		opts.traceExporter = exporter
	}
}

func WithMetricExporter(exporter sdkmetric.Exporter) Option {
	return func(opts *telemetryOptions) {
		opts.metricExporter = exporter
	}
}

func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, ErrMissingServiceName)
	}

	if c.ServiceVersion == "" {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, ErrMissingServiceVersion)
	}

	if c.SampleRate < 0.0 || c.SampleRate > 1.0 {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, ErrInvalidSampleRate)
	}

	return nil
}

func Initialize(ctx context.Context, cfg Config, opts ...Option) (*Telemetry, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	options := &telemetryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	res, err := createResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tel := &Telemetry{}

	if cfg.EnableTracing {
		tp, exp, err := initializeTracing(ctx, res, cfg, options.traceExporter)
		if err != nil {
			return nil, fmt.Errorf("initialize tracing: %w", err)
		}
		otel.SetTracerProvider(tp)
		tel.tracerProvider = tp
		tel.traceExporter = exp
	}

	if cfg.EnableMetrics {
		mp, exp, err := initializeMetrics(ctx, res, cfg, options.metricExporter)
		if err != nil {
			if tel.traceExporter != nil {
				_ = tel.traceExporter.Shutdown(ctx)
			}
			return nil, fmt.Errorf("initialize metrics: %w", err)
		}
		otel.SetMeterProvider(mp)
		tel.meterProvider = mp
		tel.metricExporter = exp
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tel, nil
}

func createResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
}

func initializeTracing(ctx context.Context, res *resource.Resource, cfg Config, providedExporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, sdktrace.SpanExporter, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if providedExporter != nil {
		exporter = providedExporter
	} else {
		// NOTE: Using WithInsecure() for plaintext gRPC connection.
		// This is intentional for this learning/demo project to work with the local
		// Docker Compose OTLP collector which doesn't have TLS configured.
		// In production, you would either:
		// 1. Remove WithInsecure() to use TLS with system certificates
		// 2. Use WithTLSCredentials() for custom TLS configuration
		// 3. Run behind a service mesh (Istio/Linkerd) that handles TLS at the sidecar level
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("create trace exporter: %w", err)
		}
	}

	sampler := createSampler(cfg.SampleRate)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)

	return tp, exporter, nil
}

func initializeMetrics(ctx context.Context, res *resource.Resource, cfg Config, providedExporter sdkmetric.Exporter) (*sdkmetric.MeterProvider, sdkmetric.Exporter, error) {
	var exporter sdkmetric.Exporter
	var err error

	if providedExporter != nil {
		exporter = providedExporter
	} else {
		// NOTE: Using WithInsecure() for plaintext gRPC connection.
		// See comment in initializeTracing() for rationale and production alternatives.
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("create metric exporter: %w", err)
		}
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)

	return mp, exporter, nil
}

func createSampler(sampleRate float64) sdktrace.Sampler {
	if sampleRate <= 0.0 {
		return sdktrace.NeverSample()
	}

	if sampleRate >= 1.0 {
		return sdktrace.AlwaysSample()
	}

	return sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(sampleRate),
	)
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error

	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown tracer provider: %w", err))
		}
	}

	if t.traceExporter != nil {
		if err := t.traceExporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown trace exporter: %w", err))
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown meter provider: %w", err))
		}
	}

	if t.metricExporter != nil {
		if err := t.metricExporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown metric exporter: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (t *Telemetry) TracerProvider() *sdktrace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) MeterProvider() *sdkmetric.MeterProvider {
	return t.meterProvider
}
