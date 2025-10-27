package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFilterLogsByLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		logFunc   func(*slog.Logger, context.Context)
		shouldLog bool
	}{
		{
			name:  "debug level logs debug",
			level: slog.LevelDebug,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.DebugContext(ctx, "debug message")
			},
			shouldLog: true,
		},
		{
			name:  "info level filters debug",
			level: slog.LevelInfo,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.DebugContext(ctx, "debug message")
			},
			shouldLog: false,
		},
		{
			name:  "info level logs info",
			level: slog.LevelInfo,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.InfoContext(ctx, "info message")
			},
			shouldLog: true,
		},
		{
			name:  "warn level filters info",
			level: slog.LevelWarn,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.InfoContext(ctx, "info message")
			},
			shouldLog: false,
		},
		{
			name:  "warn level logs warn",
			level: slog.LevelWarn,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.WarnContext(ctx, "warn message")
			},
			shouldLog: true,
		},
		{
			name:  "error level filters warn",
			level: slog.LevelError,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.WarnContext(ctx, "warn message")
			},
			shouldLog: false,
		},
		{
			name:  "error level logs error",
			level: slog.LevelError,
			logFunc: func(l *slog.Logger, ctx context.Context) {
				l.ErrorContext(ctx, "error message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: tt.level,
			})
			logger := slog.New(&traceHandler{baseHandler: baseHandler})

			ctx := context.Background()
			tt.logFunc(logger, ctx)

			if tt.shouldLog && buf.Len() == 0 {
				t.Error("expected log output but got none")
			}
			if !tt.shouldLog && buf.Len() > 0 {
				t.Errorf("expected no log output but got: %s", buf.String())
			}
		})
	}
}

func TestTraceAndSpanIDInclusion(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.InfoContext(ctx, "test message", "key", "value")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	traceID, ok := logEntry["trace_id"].(string)
	if !ok || traceID == "" {
		t.Error("expected trace_id to be present and non-empty")
	}

	spanID, ok := logEntry["span_id"].(string)
	if !ok || spanID == "" {
		t.Error("expected span_id to be present and non-empty")
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg to be 'test message', got %v", logEntry["msg"])
	}

	if logEntry["key"] != "value" {
		t.Errorf("expected key to be 'value', got %v", logEntry["key"])
	}
}

func TestLogWithoutTraceIDs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	ctx := context.Background()

	logger.InfoContext(ctx, "test message", "key", "value")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, exists := logEntry["trace_id"]; exists {
		t.Error("expected trace_id to not be present")
	}

	if _, exists := logEntry["span_id"]; exists {
		t.Error("expected span_id to not be present")
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg to be 'test message', got %v", logEntry["msg"])
	}
}

func TestRespectLogLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	ctx := context.Background()

	logger.InfoContext(ctx, "info message")
	if buf.Len() > 0 {
		t.Error("expected info message to be filtered out")
	}

	logger.WarnContext(ctx, "warn message")
	if buf.Len() == 0 {
		t.Error("expected warn message to be logged")
	}

	output := buf.String()
	if !strings.Contains(output, "warn message") {
		t.Errorf("expected output to contain 'warn message', got %s", output)
	}
}

func TestLogWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	baseLogger := slog.New(&traceHandler{baseHandler: handler})

	loggerWithAttrs := baseLogger.With("request_id", "req-123")

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	loggerWithAttrs.InfoContext(ctx, "test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["request_id"] != "req-123" {
		t.Errorf("expected request_id to be 'req-123', got %v", logEntry["request_id"])
	}

	traceID, ok := logEntry["trace_id"].(string)
	if !ok || traceID == "" {
		t.Error("expected trace_id to be present and non-empty")
	}
}

func TestLogWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	baseLogger := slog.New(&traceHandler{baseHandler: handler})

	groupLogger := baseLogger.WithGroup("http")

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	groupLogger.InfoContext(ctx, "request", "method", "GET", "path", "/orders")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	traceID, ok := logEntry["trace_id"].(string)
	if !ok || traceID == "" {
		t.Error("expected trace_id to be present at root level")
	}

	spanID, ok := logEntry["span_id"].(string)
	if !ok || spanID == "" {
		t.Error("expected span_id to be present at root level")
	}

	httpGroup, ok := logEntry["http"].(map[string]interface{})
	if !ok {
		t.Fatal("expected http group to be present")
	}

	if httpGroup["method"] != "GET" {
		t.Errorf("expected method to be 'GET', got %v", httpGroup["method"])
	}

	if httpGroup["path"] != "/orders" {
		t.Errorf("expected path to be '/orders', got %v", httpGroup["path"])
	}

	if _, exists := httpGroup["trace_id"]; exists {
		t.Error("trace_id should NOT be in the http group - should be at root level")
	}

	if _, exists := httpGroup["span_id"]; exists {
		t.Error("span_id should NOT be in the http group - should be at root level")
	}
}

func TestLogAllLevelsWithTraceIDs(t *testing.T) {
	levels := []struct {
		name     string
		logFunc  func(*slog.Logger, context.Context, string)
		expected string
	}{
		{"debug", func(l *slog.Logger, ctx context.Context, msg string) { l.DebugContext(ctx, msg) }, "DEBUG"},
		{"info", func(l *slog.Logger, ctx context.Context, msg string) { l.InfoContext(ctx, msg) }, "INFO"},
		{"warn", func(l *slog.Logger, ctx context.Context, msg string) { l.WarnContext(ctx, msg) }, "WARN"},
		{"error", func(l *slog.Logger, ctx context.Context, msg string) { l.ErrorContext(ctx, msg) }, "ERROR"},
	}

	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})
			logger := slog.New(&traceHandler{baseHandler: handler})

			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)
			defer otel.SetTracerProvider(nil)

			ctx := context.Background()
			tracer := otel.Tracer("test")
			ctx, span := tracer.Start(ctx, "test-span")
			defer span.End()

			level.logFunc(logger, ctx, "test message")

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse log output: %v", err)
			}

			if logEntry["level"] != level.expected {
				t.Errorf("expected level to be %s, got %v", level.expected, logEntry["level"])
			}

			if _, ok := logEntry["trace_id"].(string); !ok {
				t.Error("expected trace_id to be present")
			}

			if _, ok := logEntry["span_id"].(string); !ok {
				t.Error("expected span_id to be present")
			}
		})
	}
}

func TestLogLevelEnabled(t *testing.T) {
	tests := []struct {
		name            string
		handlerLevel    slog.Level
		checkLevel      slog.Level
		shouldBeEnabled bool
	}{
		{"debug handler enables debug", slog.LevelDebug, slog.LevelDebug, true},
		{"debug handler enables info", slog.LevelDebug, slog.LevelInfo, true},
		{"info handler disables debug", slog.LevelInfo, slog.LevelDebug, false},
		{"info handler enables info", slog.LevelInfo, slog.LevelInfo, true},
		{"info handler enables warn", slog.LevelInfo, slog.LevelWarn, true},
		{"warn handler disables info", slog.LevelWarn, slog.LevelInfo, false},
		{"warn handler enables warn", slog.LevelWarn, slog.LevelWarn, true},
		{"error handler disables warn", slog.LevelError, slog.LevelWarn, false},
		{"error handler enables error", slog.LevelError, slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: tt.handlerLevel,
			})
			traceHandler := &traceHandler{baseHandler: handler}

			ctx := context.Background()
			enabled := traceHandler.Enabled(ctx, tt.checkLevel)

			if enabled != tt.shouldBeEnabled {
				t.Errorf("expected Enabled() to be %v, got %v", tt.shouldBeEnabled, enabled)
			}
		})
	}
}

func TestLogWithChainedAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.With("attr1", "value1").With("attr2", "value2").InfoContext(ctx, "test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["attr1"] != "value1" {
		t.Errorf("expected attr1 to be 'value1', got %v", logEntry["attr1"])
	}

	if logEntry["attr2"] != "value2" {
		t.Errorf("expected attr2 to be 'value2', got %v", logEntry["attr2"])
	}

	if _, ok := logEntry["trace_id"].(string); !ok {
		t.Error("expected trace_id to be present")
	}

	if _, ok := logEntry["span_id"].(string); !ok {
		t.Error("expected span_id to be present")
	}
}

func TestLogWithNestedGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.WithGroup("http").WithGroup("request").InfoContext(ctx, "nested", "method", "POST")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, ok := logEntry["trace_id"].(string); !ok {
		t.Error("expected trace_id to be present at root level")
	}

	if _, ok := logEntry["span_id"].(string); !ok {
		t.Error("expected span_id to be present at root level")
	}

	httpGroup, ok := logEntry["http"].(map[string]interface{})
	if !ok {
		t.Fatal("expected http group to be present")
	}

	requestGroup, ok := httpGroup["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected request group to be present inside http")
	}

	if requestGroup["method"] != "POST" {
		t.Errorf("expected method to be 'POST', got %v", requestGroup["method"])
	}
}

func TestLogWithAttributesAndGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.With("request_id", "req-123").WithGroup("http").InfoContext(ctx, "request", "method", "GET")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["request_id"] != "req-123" {
		t.Errorf("expected request_id at root level, got %v", logEntry["request_id"])
	}

	if _, ok := logEntry["trace_id"].(string); !ok {
		t.Error("expected trace_id to be present at root level")
	}

	if _, ok := logEntry["span_id"].(string); !ok {
		t.Error("expected span_id to be present at root level")
	}

	httpGroup, ok := logEntry["http"].(map[string]interface{})
	if !ok {
		t.Fatal("expected http group to be present")
	}

	if httpGroup["method"] != "GET" {
		t.Errorf("expected method in http group, got %v", httpGroup["method"])
	}
}

func TestLogWithMultipleAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&traceHandler{baseHandler: handler})

	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nil)

	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.InfoContext(ctx, "multiple attrs",
		"key1", "value1",
		"key2", 42,
		"key3", true,
		"key4", 3.14,
	)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["key1"] != "value1" {
		t.Errorf("expected key1 to be 'value1', got %v", logEntry["key1"])
	}

	if logEntry["key2"] != float64(42) {
		t.Errorf("expected key2 to be 42, got %v", logEntry["key2"])
	}

	if logEntry["key3"] != true {
		t.Errorf("expected key3 to be true, got %v", logEntry["key3"])
	}

	if logEntry["key4"] != 3.14 {
		t.Errorf("expected key4 to be 3.14, got %v", logEntry["key4"])
	}

	if _, ok := logEntry["trace_id"].(string); !ok {
		t.Error("expected trace_id to be present")
	}
}
