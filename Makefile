.PHONY: help test integration test-all build run clean docker-up docker-down migrations-up migrations-down tidy

help:
	@echo "Available targets:"
	@echo "  make test           - Run unit tests"
	@echo "  make integration    - Run integration tests (requires Docker)"
	@echo "  make test-all       - Run all tests (unit + integration)"
	@echo "  make build          - Build the API binary"
	@echo "  make run            - Run the API locally"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make tidy           - Tidy go.mod dependencies"
	@echo "  make docker-up      - Start all services via docker-compose"
	@echo "  make docker-down    - Stop all services"
	@echo "  make migrations-up  - Run database migrations"
	@echo "  make migrations-down- Rollback last migration"

test:
	go test -v ./...

integration:
	go test -v -tags=integration ./...

test-all: test integration

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

clean:
	rm -rf bin/

tidy:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrations-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/tbd?sslmode=disable" up

migrations-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/tbd?sslmode=disable" down 1
