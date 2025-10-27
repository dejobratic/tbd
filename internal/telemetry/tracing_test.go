package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func TestStartSpan(t *testing.T) {
	t.Run("creates span with correct name", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")
		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		if spans[0].Name != "test-operation" {
			t.Errorf("expected span name 'test-operation', got %s", spans[0].Name)
		}
	})

	t.Run("returns context with span", func(t *testing.T) {
		_, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		newCtx, span := StartSpan(ctx, "test-operation")
		defer span.End()

		if newCtx == ctx {
			t.Error("expected new context, got same context")
		}

		spanCtx := span.SpanContext()
		if !spanCtx.IsValid() {
			t.Error("expected valid span context")
		}
	})

	t.Run("creates nested spans correctly", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		ctx1, span1 := StartSpan(ctx, "parent-operation")
		ctx2, span2 := StartSpan(ctx1, "child-operation")
		span2.End()
		span1.End()

		spans := exp.GetSpans()
		if len(spans) != 2 {
			t.Fatalf("expected 2 spans, got %d", len(spans))
		}

		childSpan := spans[0]
		parentSpan := spans[1]

		if childSpan.Parent.SpanID() != parentSpan.SpanContext.SpanID() {
			t.Error("expected child span to have parent span ID")
		}

		if ctx2 == ctx1 || ctx1 == ctx {
			t.Error("expected different contexts for nested spans")
		}
	})
}

func TestAddSpanAttributes(t *testing.T) {
	t.Run("adds attributes to span", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		AddSpanAttributes(span,
			attribute.String("key1", "value1"),
			attribute.Int("key2", 42),
			attribute.Bool("key3", true),
		)

		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		attrs := spans[0].Attributes
		expectedAttrs := map[string]interface{}{
			"key1": "value1",
			"key2": int64(42),
			"key3": true,
		}

		for key, expectedValue := range expectedAttrs {
			found := false
			for _, attr := range attrs {
				if string(attr.Key) == key {
					found = true
					if attr.Value.AsInterface() != expectedValue {
						t.Errorf("expected %s to be %v, got %v", key, expectedValue, attr.Value.AsInterface())
					}
					break
				}
			}
			if !found {
				t.Errorf("expected attribute %s not found", key)
			}
		}
	})

	t.Run("handles nil span gracefully", func(t *testing.T) {
		AddSpanAttributes(nil, attribute.String("key", "value"))
	})

	t.Run("adds multiple attributes in multiple calls", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		AddSpanAttributes(span, attribute.String("key1", "value1"))
		AddSpanAttributes(span, attribute.String("key2", "value2"))

		span.End()

		spans := exp.GetSpans()
		attrs := spans[0].Attributes

		if len(attrs) < 2 {
			t.Errorf("expected at least 2 attributes, got %d", len(attrs))
		}
	})
}

func TestAddSpanEvent(t *testing.T) {
	t.Run("adds event to span", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		AddSpanEvent(span, "test-event",
			attribute.String("event.key", "event.value"),
		)

		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		events := spans[0].Events
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		if events[0].Name != "test-event" {
			t.Errorf("expected event name 'test-event', got %s", events[0].Name)
		}

		found := false
		for _, attr := range events[0].Attributes {
			if string(attr.Key) == "event.key" && attr.Value.AsString() == "event.value" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected event attribute 'event.key' with value 'event.value' not found")
		}
	})

	t.Run("handles nil span gracefully", func(t *testing.T) {
		AddSpanEvent(nil, "test-event")
	})

	t.Run("adds multiple events", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		AddSpanEvent(span, "event1")
		AddSpanEvent(span, "event2")

		span.End()

		spans := exp.GetSpans()
		events := spans[0].Events

		if len(events) != 2 {
			t.Errorf("expected 2 events, got %d", len(events))
		}
	})
}

func TestRecordSpanError(t *testing.T) {
	t.Run("records error and sets error status", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		testErr := errors.New("test error")
		RecordSpanError(span, testErr)

		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Error {
			t.Errorf("expected status code Error, got %v", spans[0].Status.Code)
		}

		if spans[0].Status.Description != "test error" {
			t.Errorf("expected status description 'test error', got %s", spans[0].Status.Description)
		}

		if len(spans[0].Events) == 0 {
			t.Error("expected error event to be recorded")
		}
	})

	t.Run("handles nil span gracefully", func(t *testing.T) {
		RecordSpanError(nil, errors.New("test error"))
	})

	t.Run("handles nil error gracefully", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		RecordSpanError(span, nil)
		span.End()

		spans := exp.GetSpans()
		if spans[0].Status.Code == codes.Error {
			t.Error("expected status not to be Error when nil error is recorded")
		}
	})

	t.Run("handles both nil span and nil error gracefully", func(t *testing.T) {
		RecordSpanError(nil, nil)
	})
}

func TestSetSpanSuccess(t *testing.T) {
	t.Run("sets span status to OK", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		SetSpanSuccess(span)
		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		if spans[0].Status.Code != codes.Ok {
			t.Errorf("expected status code Ok, got %v", spans[0].Status.Code)
		}

		if spans[0].Status.Description != "" {
			t.Errorf("expected empty status description, got %s", spans[0].Status.Description)
		}
	})

	t.Run("handles nil span gracefully", func(t *testing.T) {
		SetSpanSuccess(nil)
	})

	t.Run("overwrites error status with success", func(t *testing.T) {
		exp, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		_, span := StartSpan(ctx, "test-operation")

		RecordSpanError(span, errors.New("test error"))
		SetSpanSuccess(span)

		span.End()

		spans := exp.GetSpans()
		if spans[0].Status.Code != codes.Ok {
			t.Errorf("expected status code Ok after setting success, got %v", spans[0].Status.Code)
		}
	})
}

func TestTraceID(t *testing.T) {
	t.Run("extracts trace ID from context with span", func(t *testing.T) {
		_, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-operation")
		defer span.End()

		traceID := TraceID(ctx)

		if traceID == "" {
			t.Error("expected trace ID to be non-empty")
		}

		if len(traceID) != 32 {
			t.Errorf("expected trace ID length 32, got %d", len(traceID))
		}

		expectedTraceID := span.SpanContext().TraceID().String()
		if traceID != expectedTraceID {
			t.Errorf("expected trace ID %s, got %s", expectedTraceID, traceID)
		}
	})

	t.Run("returns empty string for context without span", func(t *testing.T) {
		ctx := context.Background()
		traceID := TraceID(ctx)

		if traceID != "" {
			t.Errorf("expected empty trace ID, got %s", traceID)
		}
	})

	t.Run("extracts same trace ID from nested spans", func(t *testing.T) {
		_, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		ctx1, span1 := StartSpan(ctx, "parent")
		ctx2, span2 := StartSpan(ctx1, "child")

		traceID1 := TraceID(ctx1)
		traceID2 := TraceID(ctx2)

		if traceID1 != traceID2 {
			t.Errorf("expected same trace ID for nested spans, got %s and %s", traceID1, traceID2)
		}

		span2.End()
		span1.End()
	})
}

func TestSpanID(t *testing.T) {
	t.Run("extracts span ID from context with span", func(t *testing.T) {
		_, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-operation")
		defer span.End()

		spanID := SpanID(ctx)

		if spanID == "" {
			t.Error("expected span ID to be non-empty")
		}

		if len(spanID) != 16 {
			t.Errorf("expected span ID length 16, got %d", len(spanID))
		}

		expectedSpanID := span.SpanContext().SpanID().String()
		if spanID != expectedSpanID {
			t.Errorf("expected span ID %s, got %s", expectedSpanID, spanID)
		}
	})

	t.Run("returns empty string for context without span", func(t *testing.T) {
		ctx := context.Background()
		spanID := SpanID(ctx)

		if spanID != "" {
			t.Errorf("expected empty span ID, got %s", spanID)
		}
	})

	t.Run("extracts different span IDs from nested spans", func(t *testing.T) {
		_, cleanup := setupTracerProvider(t)
		defer cleanup()

		ctx := context.Background()
		ctx1, span1 := StartSpan(ctx, "parent")
		ctx2, span2 := StartSpan(ctx1, "child")

		spanID1 := SpanID(ctx1)
		spanID2 := SpanID(ctx2)

		if spanID1 == spanID2 {
			t.Error("expected different span IDs for nested spans")
		}

		span2.End()
		span1.End()
	})
}
