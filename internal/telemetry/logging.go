package telemetry

import (
	"context"
	"log/slog"
	"os"
)

func NewLogger(level slog.Level) *slog.Logger {
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	handler := &traceHandler{baseHandler: baseHandler}
	return slog.New(handler)
}

type traceHandler struct {
	baseHandler slog.Handler
	groups      []string
	attrs       []slog.Attr
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.baseHandler.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceAttrs := []slog.Attr{}
	if traceID := TraceID(ctx); traceID != "" {
		traceAttrs = append(traceAttrs, slog.String("trace_id", traceID))
	}
	if spanID := SpanID(ctx); spanID != "" {
		traceAttrs = append(traceAttrs, slog.String("span_id", spanID))
	}

	handler := h.baseHandler

	if len(traceAttrs) > 0 {
		handler = handler.WithAttrs(traceAttrs)
	}

	if len(h.attrs) > 0 {
		handler = handler.WithAttrs(h.attrs)
	}

	for _, group := range h.groups {
		handler = handler.WithGroup(group)
	}

	return handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &traceHandler{
		baseHandler: h.baseHandler,
		groups:      h.groups,
		attrs:       newAttrs,
	}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &traceHandler{
		baseHandler: h.baseHandler,
		groups:      newGroups,
		attrs:       h.attrs,
	}
}
