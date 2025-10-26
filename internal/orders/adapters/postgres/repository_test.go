//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dejobratic/tbd/internal/database"
	"github.com/dejobratic/tbd/internal/orders/adapters/postgres"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
	"github.com/jackc/pgx/v5/pgxpool"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := testpostgres.Run(ctx,
		"postgres:16-alpine",
		testpostgres.WithDatabase("test"),
		testpostgres.WithUsername("test"),
		testpostgres.WithPassword("test"),
		testpostgres.BasicWaitStrategies(),
		testpostgres.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	projectRoot := findProjectRoot(t)
	migrationsPath := filepath.Join(projectRoot, "migrations")

	if err := database.RunMigrations(connStr, migrationsPath); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	pool, err := database.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func TestRepositoryCreate(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	order := domain.Order{
		ID:            "test-order-1",
		CustomerEmail: "user@example.com",
		AmountCents:   1999,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	err := repo.Create(ctx, order)
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to retrieve order: %v", err)
	}

	if retrieved.ID != order.ID {
		t.Errorf("expected ID %s, got %s", order.ID, retrieved.ID)
	}
	if retrieved.CustomerEmail != order.CustomerEmail {
		t.Errorf("expected email %s, got %s", order.CustomerEmail, retrieved.CustomerEmail)
	}
	if retrieved.AmountCents != order.AmountCents {
		t.Errorf("expected amount %d, got %d", order.AmountCents, retrieved.AmountCents)
	}
	if retrieved.Status != order.Status {
		t.Errorf("expected status %s, got %s", order.Status, retrieved.Status)
	}
}

func TestRepositoryGetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if err != ports.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryList(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	orders := []domain.Order{
		{
			ID:            "order-1",
			CustomerEmail: "user1@example.com",
			AmountCents:   1000,
			Status:        domain.StatusPending,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		{
			ID:            "order-2",
			CustomerEmail: "user2@example.com",
			AmountCents:   2000,
			Status:        domain.StatusCompleted,
			CreatedAt:     time.Now().UTC().Add(1 * time.Second),
			UpdatedAt:     time.Now().UTC().Add(1 * time.Second),
		},
		{
			ID:            "order-3",
			CustomerEmail: "user3@example.com",
			AmountCents:   3000,
			Status:        domain.StatusPending,
			CreatedAt:     time.Now().UTC().Add(2 * time.Second),
			UpdatedAt:     time.Now().UTC().Add(2 * time.Second),
		},
	}

	for _, order := range orders {
		if err := repo.Create(ctx, order); err != nil {
			t.Fatalf("failed to create order: %v", err)
		}
	}

	t.Run("list all orders", func(t *testing.T) {
		result, err := repo.List(ctx, ports.ListFilter{})
		if err != nil {
			t.Fatalf("failed to list orders: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("expected 3 orders, got %d", len(result))
		}

		if result[0].ID != "order-3" {
			t.Errorf("expected first order to be order-3 (newest), got %s", result[0].ID)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		status := domain.StatusPending
		result, err := repo.List(ctx, ports.ListFilter{Status: &status})
		if err != nil {
			t.Fatalf("failed to list orders: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 pending orders, got %d", len(result))
		}

		for _, order := range result {
			if order.Status != domain.StatusPending {
				t.Errorf("expected status pending, got %s", order.Status)
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		result, err := repo.List(ctx, ports.ListFilter{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatalf("failed to list orders: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 orders (page 1), got %d", len(result))
		}

		result, err = repo.List(ctx, ports.ListFilter{Page: 2, PageSize: 2})
		if err != nil {
			t.Fatalf("failed to list orders: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("expected 1 order (page 2), got %d", len(result))
		}
	})
}

func TestRepositoryUpdateStatus(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	order := domain.Order{
		ID:            "test-order-update",
		CustomerEmail: "user@example.com",
		AmountCents:   1500,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	err := repo.UpdateStatus(ctx, order.ID, domain.StatusProcessing)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	updated, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to retrieve order: %v", err)
	}

	if updated.Status != domain.StatusProcessing {
		t.Errorf("expected status processing, got %s", updated.Status)
	}

	if !updated.UpdatedAt.After(order.UpdatedAt) {
		t.Error("expected updated_at to be updated")
	}
}

func TestRepositoryUpdateStatus_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, "nonexistent-id", domain.StatusCompleted)
	if err != ports.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
