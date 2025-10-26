package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dejobratic/tbd/internal/orders/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Get(ctx context.Context, key string) (*ports.StoredResponse, error) {
	query := `
		SELECT status_code, body, order_id
		FROM idempotency_keys
		WHERE key = $1
	`

	var resp ports.StoredResponse
	err := s.pool.QueryRow(ctx, query, key).Scan(
		&resp.StatusCode,
		&resp.Body,
		&resp.OrderID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select idempotency key: %w", err)
	}

	return &resp, nil
}

func (s *Store) Save(ctx context.Context, key string, response ports.StoredResponse) error {
	query := `
		INSERT INTO idempotency_keys (key, status_code, body, order_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, query, key, response.StatusCode, response.Body, response.OrderID)
	if err != nil {
		return fmt.Errorf("insert idempotency key: %w", err)
	}

	return nil
}
