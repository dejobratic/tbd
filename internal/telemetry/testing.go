package telemetry

import (
	"context"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type noopTraceExporter struct{}

func (n *noopTraceExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n *noopTraceExporter) Shutdown(_ context.Context) error {
	return nil
}

type noopMetricExporter struct{}

func (n *noopMetricExporter) Temporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (n *noopMetricExporter) Aggregation(_ sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.AggregationDefault{}
}

func (n *noopMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	return nil
}

func (n *noopMetricExporter) ForceFlush(_ context.Context) error {
	return nil
}

func (n *noopMetricExporter) Shutdown(_ context.Context) error {
	return nil
}

func NewNoopTraceExporter() sdktrace.SpanExporter {
	return &noopTraceExporter{}
}

func NewNoopMetricExporter() sdkmetric.Exporter {
	return &noopMetricExporter{}
}
