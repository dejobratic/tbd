//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dejobratic/tbd/internal/database"
	"github.com/dejobratic/tbd/internal/idempotency/postgres"
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

func TestStoreSaveAndGet(t *testing.T) {
	pool := setupTestDB(t)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	key := "test-idempotency-key-1"
	response := ports.StoredResponse{
		StatusCode: 201,
		Body:       []byte(`{"order_id": "test-order-1"}`),
		OrderID:    "test-order-1",
	}

	err := store.Save(ctx, key, response)
	if err != nil {
		t.Fatalf("failed to save idempotency key: %v", err)
	}

	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get idempotency key: %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected response, got nil")
	}

	if retrieved.StatusCode != response.StatusCode {
		t.Errorf("expected status code %d, got %d", response.StatusCode, retrieved.StatusCode)
	}

	if string(retrieved.Body) != string(response.Body) {
		t.Errorf("expected body %s, got %s", response.Body, retrieved.Body)
	}

	if retrieved.OrderID != response.OrderID {
		t.Errorf("expected order ID %s, got %s", response.OrderID, retrieved.OrderID)
	}
}

func TestStoreGet_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "nonexistent-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if retrieved != nil {
		t.Errorf("expected nil response, got %v", retrieved)
	}
}

func TestStoreSave_Conflict(t *testing.T) {
	pool := setupTestDB(t)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	key := "test-idempotency-key-conflict"
	response1 := ports.StoredResponse{
		StatusCode: 201,
		Body:       []byte(`{"order_id": "order-1"}`),
		OrderID:    "order-1",
	}
	response2 := ports.StoredResponse{
		StatusCode: 200,
		Body:       []byte(`{"order_id": "order-2"}`),
		OrderID:    "order-2",
	}

	if err := store.Save(ctx, key, response1); err != nil {
		t.Fatalf("failed to save first response: %v", err)
	}

	if err := store.Save(ctx, key, response2); err != nil {
		t.Fatalf("failed to save second response (conflict): %v", err)
	}

	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get response: %v", err)
	}

	if retrieved.OrderID != response1.OrderID {
		t.Errorf("expected first response to be preserved, got order ID %s", retrieved.OrderID)
	}
}
