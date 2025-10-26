package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, order domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_email, amount_cents, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		order.ID,
		order.CustomerEmail,
		order.AmountCents,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_email, amount_cents, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var order domain.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerEmail,
		&order.AmountCents,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, fmt.Errorf("select order: %w", err)
	}

	return &order, nil
}

func (r *Repository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Order, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query := `
		SELECT id, customer_email, amount_cents, status, created_at, updated_at
		FROM orders
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var statusFilter *string
	if filter.Status != nil {
		s := string(*filter.Status)
		statusFilter = &s
	}

	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx, query, statusFilter, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.CustomerEmail,
			&order.AmountCents,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}

	return nil
}
