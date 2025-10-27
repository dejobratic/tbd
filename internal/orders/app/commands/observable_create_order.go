package commands

import (
	"context"
	"log/slog"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/metrics"
	"github.com/dejobratic/tbd/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type ObservableCommandHandler struct {
	handler CommandHandler
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func NewObservableCommandHandler(handler CommandHandler, logger *slog.Logger, metrics *metrics.Metrics) *ObservableCommandHandler {
	return &ObservableCommandHandler{
		handler: handler,
		logger:  logger,
		metrics: metrics,
	}
}

func (o *ObservableCommandHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	ctx, span := telemetry.StartSpan(ctx, "CreateOrderCommand.Handle")
	defer span.End()

	start := time.Now()
	var success bool
	defer func() {
		duration := time.Since(start).Seconds()
		o.metrics.RecordOrderCreationDuration(ctx, duration)
		o.metrics.RecordOrderCreated(ctx, success)
	}()

	o.logger.InfoContext(ctx, "creating order",
		"customer_email", cmd.CustomerEmail,
		"amount_cents", cmd.AmountCents,
	)

	order, err := o.handler.Handle(ctx, cmd)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		o.logger.ErrorContext(ctx, "failed to create order",
			"error", err,
			"customer_email", cmd.CustomerEmail,
		)
		return nil, err
	}

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", order.ID),
		attribute.String("order.customer_email", order.CustomerEmail),
		attribute.Int64("order.amount_cents", order.AmountCents),
		attribute.String("order.status", string(order.Status)),
	)

	o.logger.InfoContext(ctx, "order created successfully",
		"order_id", order.ID,
		"customer_email", order.CustomerEmail,
	)

	success = true
	telemetry.SetSpanSuccess(span)

	return order, nil
}
