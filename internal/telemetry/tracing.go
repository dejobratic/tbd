package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/dejobratic/tbd/internal/telemetry"

func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.SetAttributes(attrs...)
}

func AddSpanEvent(span trace.Span, eventName string, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

func RecordSpanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func SetSpanSuccess(span trace.Span) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, "")
}

func TraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

func SpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}
