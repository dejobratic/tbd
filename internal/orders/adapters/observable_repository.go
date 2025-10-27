package adapters

import (
	"context"
	"time"

	"github.com/dejobratic/tbd/internal/database"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
	"github.com/dejobratic/tbd/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type ObservableRepository struct {
	repo    ports.OrderRepository
	metrics *database.Metrics
}

func NewObservableRepository(repo ports.OrderRepository, metrics *database.Metrics) *ObservableRepository {
	return &ObservableRepository{
		repo:    repo,
		metrics: metrics,
	}
}

func (r *ObservableRepository) Create(ctx context.Context, order domain.Order) error {
	ctx, span := telemetry.StartSpan(ctx, "OrderRepository.Create")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", order.ID),
		attribute.String("operation", "create"),
	)

	start := time.Now()
	err := r.repo.Create(ctx, order)
	duration := time.Since(start).Seconds()

	r.metrics.RecordQuery(ctx, "create_order", duration)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}

	telemetry.SetSpanSuccess(span)
	return nil
}

func (r *ObservableRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	ctx, span := telemetry.StartSpan(ctx, "OrderRepository.GetByID")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", id),
		attribute.String("operation", "get_by_id"),
	)

	start := time.Now()
	order, err := r.repo.GetByID(ctx, id)
	duration := time.Since(start).Seconds()

	r.metrics.RecordQuery(ctx, "get_order_by_id", duration)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return nil, err
	}

	telemetry.SetSpanSuccess(span)
	return order, nil
}

func (r *ObservableRepository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	ctx, span := telemetry.StartSpan(ctx, "OrderRepository.List")
	defer span.End()

	attrs := []attribute.KeyValue{
		attribute.String("operation", "list"),
		attribute.Int("page", filter.Page),
		attribute.Int("page_size", filter.PageSize),
	}
	if filter.Status != nil {
		attrs = append(attrs, attribute.String("filter.status", string(*filter.Status)))
	}
	telemetry.AddSpanAttributes(span, attrs...)

	start := time.Now()
	orders, err := r.repo.List(ctx, filter)
	duration := time.Since(start).Seconds()

	r.metrics.RecordQuery(ctx, "list_orders", duration)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return nil, err
	}

	telemetry.AddSpanAttributes(span, attribute.Int("result.count", len(orders)))
	telemetry.SetSpanSuccess(span)
	return orders, nil
}

func (r *ObservableRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	ctx, span := telemetry.StartSpan(ctx, "OrderRepository.UpdateStatus")
	defer span.End()

	telemetry.AddSpanAttributes(span,
		attribute.String("order.id", id),
		attribute.String("order.new_status", string(status)),
		attribute.String("operation", "update_status"),
	)

	start := time.Now()
	err := r.repo.UpdateStatus(ctx, id, status)
	duration := time.Since(start).Seconds()

	r.metrics.RecordQuery(ctx, "update_order_status", duration)

	if err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}

	telemetry.SetSpanSuccess(span)
	return nil
}
