package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CheckHealth(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return pool.Ping(ctx)
}
