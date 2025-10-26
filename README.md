# tbd — tiny, but distributed

A **tiny, but real distributed system** built for learning and demonstration purposes.  
It uses **Go**, **PostgreSQL**, **Kafka**, and **Docker Compose** to showcase API design, event-driven processing, observability, and idempotency.

---

## 🧩 Overview

`tbd` simulates a simple **Order Processing System** — small enough to run locally, but rich enough to demonstrate real distributed system concepts:

- RESTful API
- PostgreSQL database
- Kafka-based event stream
- Background worker for async processing
- OpenTelemetry tracing + Prometheus metrics + Grafana dashboards
- Jaeger for distributed tracing
- pgAdmin and Kafka UI for observability

---

## ⚙️ Architecture

```mermaid
flowchart LR
    %% Overall direction
    %% Styles (optional; safe defaults)
    classDef client fill:#eef,stroke:#88a;
    classDef services fill:#eefeef,stroke:#6a6;
    classDef messaging fill:#fff6dd,stroke:#cc9;
    classDef data fill:#e8f2ff,stroke:#59a;
    classDef observ fill:#f5f5f5,stroke:#999;

    %% Client
    subgraph client[Client]
        direction TB
        k6["k6 Load Tester"]:::client
    end

    %% Load Balancer
    subgraph lb[Load Balancer]
        direction TB
        nginx["Nginx<br/>(Round-robin)"]:::services
    end

    %% Core services
    subgraph services[Services]
        direction TB
        api["API (Go)<br/>REST<br/>(scalable)"]:::services
        worker["Worker<br/>(Kafka Consumer/Producer)<br/>(scalable)"]:::services
    end

    %% Messaging
    subgraph msg[Messaging]
        direction TB
        kafka[("(Kafka)<br/>order.created / order.processed")]:::messaging
    end

    %% Data
    subgraph data[Data]
        direction TB
        postgres["Postgres"]:::data
    end

    %% Observability
    subgraph obs[Observability]
        direction TB
        otel["OpenTelemetry Collector"]:::observ
        jaeger["Jaeger"]:::observ
        prometheus["Prometheus"]:::observ
        grafana["Grafana"]:::observ
        pgadmin["pgAdmin"]:::observ
        kafkaui["Kafka UI"]:::observ
    end

    %% Happy-path flow
    k6 --> nginx
    nginx --> api
    api -->|"(1) Write order"| postgres
    api -->|"(2) Publish order.created"| kafka
    kafka --> worker
    worker -->|"(3) Update status"| postgres
    worker -->|"(4) Emit order.processed"| kafka

    %% Telemetry & UIs
    api -->|"Traces & metrics"| otel
    worker -->|"Traces & metrics"| otel
    otel --> jaeger
    otel --> prometheus
    prometheus --> grafana
    postgres --> pgadmin
    kafka --> kafkaui
```

---

## 🧠 Core Concepts

### Entities
**Order**
```json
{
  "id": "uuid",
  "customer_email": "user@example.com",
  "amount_cents": 1299,
  "status": "pending|processing|completed|failed|canceled",
  "created_at": "...",
  "updated_at": "..."
}
```

---

## 🚀 Components (Docker Compose)

| Service         | Purpose |
|-----------------|----------|
| **nginx**       | Load balancer for API instances on port `8080` |
| **api**         | Go REST API, exposes `/v1/orders` endpoints (scalable) |
| **worker**      | Kafka consumer/producer; processes `order.created` events (scalable) |
| **postgres**    | Relational DB for orders |
| **pgadmin**     | Database UI on port `5050` |
| **kafka**       | Message broker (single-node cluster or Redpanda) |
| **kafka-ui**    | Kafka topic browser on port `8082` |
| **otel-collector** | Receives traces/metrics from services |
| **jaeger**      | Distributed tracing UI on port `16686` |
| **prometheus**  | Metrics collection on port `9090` |
| **grafana**     | Dashboards on port `3000` |

---

## 🌐 API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness check |
| `GET` | `/readyz` | Readiness (checks DB + Kafka) |
| `GET` | `/metrics` | Prometheus scrape endpoint |
| `POST` | `/v1/orders` | Create order (requires `Idempotency-Key`; see details below) |
| `GET` | `/v1/orders/{id}` | Retrieve order by ID |
| `GET` | `/v1/orders` | List orders (`?status=&page=&page_size=`) |
| `POST` | `/v1/orders/{id}/cancel` | Cancel pending order |

---

## 🔁 Idempotency for POST /v1/orders

Use an `Idempotency-Key` header for **POST /v1/orders** to ensure safe retries.

**Example:**
```bash
curl -X POST http://localhost:8080/v1/orders   -H "Content-Type: application/json"   -H "Idempotency-Key: $(uuidgen)"   -d '{"customer_email":"a@b.com","amount_cents":1234}'
```

### How it works
- The API stores `{ key, request_hash, response, order_id }` for each key.
- Repeated calls with the same key **replay** the original response.
- Prevents duplicate orders on network retries.
- TTL for dedup cache: 24–72h (configurable).

> **Note:** `Idempotency-Key` ≠ `If-Match`.  
> `If-Match` (with ETags) handles concurrency for updates.  
> `Idempotency-Key` deduplicates **retries** of the same create request.

---

## 🔄 Kafka Topics

| Topic | Description |
|--------|-------------|
| `order.created` | Emitted by API when a new order is created |
| `order.processed` | Emitted by Worker after successful processing |

**Future topics** (for robust error handling):
- `order.failed` — Emitted by Worker on processing failure
- `order.dlq` — Dead letter queue for poison messages after max retries

---

## 🧰 Local Development

### Prerequisites
- Docker + Docker Compose
- Go ≥ 1.22
- make (optional)
- k6 (for load testing)

### Run everything
```bash
docker compose up --build
```

Check UIs:
- API → [http://localhost:8080](http://localhost:8080)
- pgAdmin → [http://localhost:5050](http://localhost:5050)
- Kafka UI → [http://localhost:8082](http://localhost:8082)
- Jaeger → [http://localhost:16686](http://localhost:16686)
- Prometheus → [http://localhost:9090](http://localhost:9090)
- Grafana → [http://localhost:3000](http://localhost:3000)

---

## 🗃️ Database Migrations

This project uses [**golang-migrate/migrate**](https://github.com/golang-migrate/migrate) for database schema versioning and migrations.

### Installation

**macOS:**
```bash
brew install golang-migrate
```

**Linux:**
```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

**Go install:**
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Running Migrations

**Apply all pending migrations:**
```bash
migrate -path migrations \
  -database "postgres://tbd:secret@localhost:5432/tbd?sslmode=disable" \
  up
```

**Rollback last migration:**
```bash
migrate -path migrations \
  -database "postgres://..." \
  down 1
```

**Check migration version:**
```bash
migrate -path migrations \
  -database "postgres://..." \
  version
```

### Creating New Migrations

```bash
migrate create -ext sql -dir migrations -seq create_payments_table
```

This generates:
- `000003_create_payments_table.up.sql` (apply changes)
- `000003_create_payments_table.down.sql` (rollback changes)

### Automated Migrations

The API automatically runs pending migrations on startup (see `cmd/api/main.go` for the initialization logic).

**Disable auto-migration** for production by setting:
```bash
AUTO_MIGRATE=false
```

---

## ⚙️ Configuration

Services are configured via **environment variables**. See `docker-compose.yml` for the complete setup.

### API Service

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `tbd` | Database user |
| `DB_PASSWORD` | `secret` | Database password |
| `DB_NAME` | `tbd` | Database name |
| `DB_MAX_CONNS` | `25` | Maximum database connections |
| `DB_MAX_IDLE_CONNS` | `5` | Maximum idle database connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Maximum connection lifetime |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `KAFKA_TOPIC_ORDER_CREATED` | `order.created` | Topic for order creation events |
| `KAFKA_TOPIC_ORDER_PROCESSED` | `order.processed` | Topic for order processed events |
| `IDEMPOTENCY_TTL` | `72h` | Time-to-live for idempotency keys (24h–168h) |
| `AUTO_MIGRATE` | `true` | Run database migrations on startup |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector endpoint |
| `OTEL_SERVICE_NAME` | `tbd-api` | Service name for traces/metrics |

### Worker Service

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `tbd` | Database user |
| `DB_PASSWORD` | `secret` | Database password |
| `DB_NAME` | `tbd` | Database name |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `KAFKA_CONSUMER_GROUP` | `tbd-workers` | Consumer group ID for worker instances |
| `KAFKA_TOPIC_ORDER_CREATED` | `order.created` | Topic to consume from |
| `KAFKA_TOPIC_ORDER_PROCESSED` | `order.processed` | Topic to publish to |
| `WORKER_CONCURRENCY` | `5` | Number of concurrent message processors |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector endpoint |
| `OTEL_SERVICE_NAME` | `tbd-worker` | Service name for traces/metrics |

### Example `.env` File

```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=tbd
DB_PASSWORD=secret
DB_NAME=tbd

# Kafka
KAFKA_BROKERS=kafka:9092

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
LOG_LEVEL=debug

# API-specific
API_PORT=8080
IDEMPOTENCY_TTL=72h
AUTO_MIGRATE=true

# Worker-specific
KAFKA_CONSUMER_GROUP=tbd-workers
WORKER_CONCURRENCY=5
```

---

## 📈 Observability

| Component | Tool | Notes |
|------------|------|-------|
| Tracing | **OpenTelemetry + Jaeger** | Follow request → event → processing trace |
| Metrics | **Prometheus + Grafana** | HTTP latency, Kafka lag, worker stats |
| Logs | **Structured JSON** | Includes `trace_id`, `span_id`, `order_id` |

**Key Metrics to Monitor:**
- `http_request_duration_seconds` — API endpoint latency (P50, P95, P99)
- `http_requests_total` — Request count by status code
- `kafka_producer_latency_seconds` — Time to publish events
- `kafka_consumer_lag` — Consumer group lag per partition
- `db_query_duration_seconds` — Database query performance
- `orders_created_total` — Business metric: orders created
- `orders_processed_total` — Business metric: orders processed
- `idempotency_hits_total` — Duplicate request prevention rate

---

## ⚠️ Error Handling & Resilience

This project demonstrates production-ready error handling and resilience patterns.

### Retry Strategy

| Operation | Retries | Backoff | Notes |
|-----------|---------|---------|-------|
| **Kafka publish** | 3 | Exponential: 100ms → 500ms → 2s | API fails request if Kafka unavailable |
| **DB queries** | 3 | Linear: 50ms intervals | Automatic retry on transient errors (connection loss, deadlock) |
| **Worker message processing** | ∞ | At-least-once delivery | Kafka consumer auto-commits only on success |

### Timeouts

| Component | Timeout | Rationale |
|-----------|---------|-----------|
| **HTTP request** | 30s | Prevents client hanging indefinitely |
| **DB query** | 5s | Fails fast on slow queries |
| **Kafka publish** | 10s | Allows retries but prevents indefinite blocking |
| **Worker processing** | 60s | Per-message processing limit |
| **Graceful shutdown** | 30s | Finish in-flight requests before terminating |

### Failure Modes & Handling

#### **Kafka Unavailable**
- **API behavior**: Returns `503 Service Unavailable`
- **Readiness check**: `/readyz` fails → load balancer stops routing traffic
- **Worker behavior**: Stops consuming, waits for reconnection
- **Recovery**: Auto-reconnects when Kafka comes back online

#### **Database Unavailable**
- **API behavior**: `/readyz` fails immediately
- **Worker behavior**: Stops processing, retries DB connection
- **Recovery**: Connection pool auto-reconnects

#### **Worker Crash Mid-Processing**
- **Kafka behavior**: Consumer group rebalances partitions
- **Message replay**: Another worker re-processes the message from last commit
- **Idempotency**: Duplicate processing is safe (idempotency keys prevent duplicate orders)

#### **Poison Message** (Future)
- **Current**: Worker retries indefinitely (can cause consumer lag)
- **Planned**: After 3 failed attempts, publish to `order.dlq` topic
- **Manual review**: DLQ messages require investigation

### Idempotency Guarantees

- **POST /v1/orders**: Uses `Idempotency-Key` header to prevent duplicate orders
- **Worker processing**: Updates are idempotent (setting status to "processed" multiple times is safe)
- **Event publishing**: Kafka's at-least-once delivery + idempotent consumers ensure exactly-once semantics

### Circuit Breaker (Future Enhancement)

Planned for external API calls or slow dependencies:
- **Threshold**: 5 consecutive failures → open circuit
- **Half-open retry**: After 30s cooldown
- **Monitoring**: Expose `circuit_breaker_state` metric

### Distributed Tracing for Errors

- All errors include `trace_id` and `span_id` for correlation
- Failed requests are visible in Jaeger with error tags
- Worker failures show full trace: API → Kafka → Worker → DB

---

## 🧪 Load Testing (k6)

Example script: `loadtest/orders.js`
```javascript
import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export let options = { vus: 20, duration: '30s' };

export default function () {
  const headers = {
    'Content-Type': 'application/json',
    'Idempotency-Key': uuidv4(),
  };
  const body = JSON.stringify({
    customer_email: `user${__VU}@example.com`,
    amount_cents: 1999,
  });

  const res = http.post('http://localhost:8080/v1/orders', body, { headers });
  check(res, { 'status 202': (r) => r.status === 202 });
}
```

Run:
```bash
k6 run loadtest/orders.js
```

---

## 🔍 Logs & Monitoring

```bash
# Tail service logs
docker compose logs -f api
docker compose logs -f worker

# Inspect recent Kafka messages
docker exec -it kafka kafka-console-consumer   --bootstrap-server localhost:9092 --topic order.created --from-beginning
```

---

## 🧮 Scaling Locally

You can simulate a true distributed system by running multiple service replicas:

```bash
docker compose up --scale api=3 --scale worker=3
```

**How it works:**
- **API scaling**: Nginx load balancer distributes requests across all API instances (round-robin)
- **Worker scaling**: Kafka consumer groups automatically distribute partitions across all worker instances
- **Database**: Single PostgreSQL instance (write scaling requires read replicas or sharding, beyond this demo's scope)

All requests still go through `http://localhost:8080` (nginx), which transparently load balances to the API instances.

**Observing distributed behavior:**
- Check nginx load balancing: `docker compose logs -f nginx`
- Monitor Kafka partition assignment: Open Kafka UI at http://localhost:8082
- Watch worker coordination: `docker compose logs -f worker`

For advanced simulation:
- Add artificial latency with `tc netem`
- Kill a worker to observe partition rebalancing: `docker compose kill tbd-worker-1`
- Kill an API instance to observe load balancer failover: `docker compose kill tbd-api-1`
- Stop Kafka briefly and observe retry/backpressure: `docker compose stop kafka`

---

## 🔧 Future Extensions

| Feature | Description |
|----------|-------------|
| **gRPC API** | Mirror the REST endpoints using Protobuf |
| **ghz testing** | Benchmark gRPC latency and throughput |
| **Outbox pattern** | Atomic DB write + event publish |
| **Saga orchestration** | Multi-step distributed workflows |
| **Service auth** | mTLS or JWT for inter-service calls |
| **Kubernetes** | Run the same topology with k3d or kind |

---

## 📂 Directory Structure

```
tbd/
├── cmd/
│   ├── api/                       # API entrypoint: wires HTTP server + use cases + adapters
│   └── worker/                    # Worker entrypoint: wires Kafka consumer + use cases
├── internal/
│   ├── orders/                    # Orders bounded context
│   │   ├── domain/
│   │   │   ├── order.go           # Order entity (aggregate root)
│   │   │   ├── status.go          # Status value object
│   │   │   └── events.go          # Domain events (OrderCreated, OrderProcessed, etc.)
│   │   ├── app/
│   │   │   ├── commands/
│   │   │   │   ├── create_order.go       # CreateOrderHandler
│   │   │   │   ├── cancel_order.go       # CancelOrderHandler
│   │   │   │   └── mark_processed.go     # MarkProcessedHandler
│   │   │   └── queries/
│   │   │       ├── get_order.go          # GetOrderHandler
│   │   │       └── list_orders.go        # ListOrdersHandler
│   │   ├── ports/
│   │   │   ├── repository.go      # OrderRepository interface
│   │   │   ├── event_bus.go       # EventBus interface (generic, tech-agnostic)
│   │   │   └── idempotency.go     # IdempotencyStore interface (generic)
│   │   └── adapters/
│   │       ├── http/              # HTTP handlers, routing, validation, DTOs
│   │       │   ├── handlers.go
│   │       │   ├── routes.go
│   │       │   └── dto.go
│   │       ├── grpc/              # gRPC handlers (future)
│   │       ├── postgres/          # OrderRepository impl using pgx/sqlc
│   │       │   ├── repository.go
│   │       │   └── queries.sql
│   │       ├── kafka/             # EventBus impl (Kafka-specific producer/consumer)
│   │       │   ├── producer.go
│   │       │   └── consumer.go
│   │       └── idempotency/       # IdempotencyStore impl (Postgres-backed)
│   │           └── store.go
│   ├── database/                  # Database infrastructure
│   │   ├── postgres.go            # Connection pooling setup
│   │   ├── migrate.go             # golang-migrate runner
│   │   └── health.go              # Health check helper
│   ├── messaging/                 # Messaging infrastructure (tech-agnostic namespace)
│   │   ├── kafka/                 # Kafka client setup, admin operations
│   │   │   ├── client.go
│   │   │   ├── admin.go           # Topic creation, etc.
│   │   │   └── config.go
│   │   └── middleware.go          # Message tracing, logging middleware
│   └── telemetry/                 # Observability setup
│       ├── otel.go                # OpenTelemetry initialization
│       ├── metrics.go             # Prometheus metrics definitions
│       ├── tracing.go             # Jaeger tracer setup
│       └── logging.go             # Structured logger (zerolog/zap)
├── configs/
│   ├── docker/                    # Docker-specific config files
│   └── grafana/                   # Grafana dashboard JSONs
├── loadtest/
│   └── orders.js                  # k6 load test script
├── migrations/
│   ├── 000001_create_orders_table.up.sql
│   ├── 000001_create_orders_table.down.sql
│   ├── 000002_create_idempotency_table.up.sql
│   └── 000002_create_idempotency_table.down.sql
├── docker-compose.yml
├── Makefile
└── README.md
```

**Key Design Principles:**
- **Ports** (interfaces) use generic, tech-agnostic names (`EventBus`, `IdempotencyStore`)
- **Adapters** (implementations) use specific names (`kafka/`, `postgres/`) for clarity
- **Infrastructure** packages (`database/`, `messaging/`) provide shared setup and helpers
- **Single idempotency location** under `orders/adapters/idempotency/` (not duplicated)

---

## 🧭 Design Goals

- **Tiny footprint** – everything runs locally.
- **Real semantics** – async events, retries, DLQs, idempotency.
- **Observability first** – traces, metrics, logs are first-class.
- **Language focus** – idiomatic Go with context propagation.
- **Safe failure** – at-least-once delivery with deduplication.
---

## 🧱 License

MIT License © 2025 – *tbd project contributors*

---

## 🧩 References
- [OpenTelemetry Spec](https://opentelemetry.io/docs/)
- [Kafka Design Docs](https://kafka.apache.org/documentation/)
- [Go Context Propagation](https://pkg.go.dev/context)
- [Prometheus Docs](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/grafana/)
- [Jaeger Tracing](https://www.jaegertracing.io/)
