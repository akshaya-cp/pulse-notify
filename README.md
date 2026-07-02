# Pulse Notify

A distributed authentication & notification platform built in Go. It pairs a
Gin HTTP API for authentication with a Kafka-driven, concurrent worker pipeline
for asynchronous notification delivery, backed by PostgreSQL for persistence and
Redis for caching, rate limiting, and refresh-token storage.

The design separates the **producer** (API) from the **consumers** (worker
pool) behind a message broker, so the write path stays fast and the two tiers
scale independently — the essence of a distributed, event-driven system.

## Architecture

```
                    ┌─────────────┐        ┌──────────┐
  client ──HTTP──▶  │   API (Gin) │──write▶│ Postgres │
                    │             │        └──────────┘
                    │  JWT auth   │──cache▶┌──────────┐
                    │  rate limit │◀───────│  Redis   │
                    └──────┬──────┘        └──────────┘
                           │ publish event
                           ▼
                    ┌─────────────┐
                    │    Kafka    │  topic: notifications
                    └──────┬──────┘
                           │ consume (group)
                           ▼
                    ┌─────────────────────────────┐
                    │   Worker (fan-out pool)      │
                    │  dispatcher → N goroutines   │
                    │  retries + backoff → Postgres│
                    └─────────────────────────────┘
```

1. A client calls `POST /api/v1/notifications`.
2. The API persists a `pending` row in Postgres and publishes a
   `NotificationEvent` to Kafka, then returns `202 Accepted`.
3. The worker's consumer group fetches messages; a single dispatcher fans them
   out to a bounded pool of goroutines over a channel.
4. Each worker "delivers" the notification (simulated provider), retrying with
   exponential backoff, and records the terminal `sent`/`failed` state in
   Postgres before committing the Kafka offset (at-least-once processing).

## Features

- **HTTP API** with [Gin](https://github.com/gin-gonic/gin), structured `slog`
  logging, and graceful shutdown on `SIGINT`/`SIGTERM`.
- **Authentication**: bcrypt password hashing, JWT access tokens, opaque
  **refresh tokens with rotation** stored in Redis, and logout (revocation).
- **Role-based access control** (`RequireRole`) protecting admin routes.
- **Redis-backed rate limiting** (fixed window, per client IP) that stays
  consistent across multiple API instances and fails open.
- **Cache-aside** user profiles and notification lists in Redis.
- **Kafka producer/consumer** via [`segmentio/kafka-go`](https://github.com/segmentio/kafka-go).
- **Concurrent worker pool** using Go concurrency primitives (goroutines +
  channels + `sync.WaitGroup`) with retries, exponential backoff, and
  at-least-once delivery.
- **PostgreSQL** storage with idempotent auto-migration on startup.
- **Docker Compose** stack: Postgres, Redis, Kafka (KRaft, no ZooKeeper), API,
  and worker.

## Tech stack

Go, Gin, PostgreSQL (pgx), Redis (go-redis), Kafka (kafka-go), JWT, bcrypt,
Docker / Docker Compose.

## Requirements

- Go 1.22+ (project uses 1.26)
- Docker + Docker Compose

## Quick start (full stack in Docker)

Brings up Postgres, Redis, Kafka, the API, and the worker together:

```bash
cp .env.example .env
docker compose up --build
```

The API listens on `http://localhost:8080`.

### Run the app locally against Docker infra

If you prefer `go run` for the app while using Docker only for infrastructure:

```bash
cp .env.example .env
docker compose up -d postgres redis kafka

go mod download
go run ./cmd/api      # terminal 1 — HTTP API
go run ./cmd/worker   # terminal 2 — notification worker pool
```

## Demo walkthrough

```bash
# 1. Sign up (returns access_token + refresh_token)
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"password123","name":"Demo User"}'

# Save the access token
TOKEN="paste-access_token-here"

# 2. Enqueue a notification (async — returns 202 with status "pending")
curl -s -X POST http://localhost:8080/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"channel":"email","recipient":"user@example.com","subject":"Hi","body":"Hello from Pulse Notify"}'

# 3. List notifications — watch status flip from "pending" to "sent" (or "failed")
curl -s http://localhost:8080/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN"

# 4. Rotate the refresh token
curl -s -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"paste-refresh_token-here"}'
```

Watch the worker logs to see the concurrent pipeline process each event, retry
on simulated failures, and commit offsets.

### Try an admin route

Signups default to the `user` role. Promote a user to exercise RBAC:

```bash
docker compose exec postgres \
  psql -U pulse -d pulse_notify \
  -c "UPDATE users SET role='admin' WHERE email='demo@example.com';"
```

Log in again to get a token with the new role, then:

```bash
curl -s http://localhost:8080/api/v1/admin/notifications \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## API overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Liveness + Postgres/Redis ping |
| POST | `/api/v1/auth/signup` | No | Register user |
| POST | `/api/v1/auth/login` | No | Login |
| POST | `/api/v1/auth/refresh` | No | Rotate refresh → new token pair |
| POST | `/api/v1/auth/logout` | No | Revoke a refresh token |
| GET | `/api/v1/me` | Bearer JWT | Current user profile (cached) |
| POST | `/api/v1/notifications` | Bearer JWT | Enqueue a notification (202) |
| GET | `/api/v1/notifications` | Bearer JWT | List caller's notifications |
| GET | `/api/v1/notifications/:id` | Bearer JWT | Get one notification |
| GET | `/api/v1/admin/notifications` | Bearer JWT (admin) | List all notifications |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | `development` loads `.env` and uses text logs |
| `HTTP_HOST` / `HTTP_PORT` | `0.0.0.0` / `8080` | Bind address |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `REDIS_URL` | `redis://127.0.0.1:6379/0` | Redis connection |
| `KAFKA_BROKERS` | `127.0.0.1:9092` | Comma-separated broker list |
| `KAFKA_TOPIC` | `notifications` | Notification event topic |
| `KAFKA_GROUP_ID` | `pulse-notify-workers` | Consumer group id |
| `JWT_SECRET` | — | HMAC secret, min 32 chars (required) |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `WORKER_CONCURRENCY` | `4` | Worker goroutines in the pool |
| `WORKER_MAX_RETRIES` | `3` | Delivery retries before marking failed |
| `RATE_LIMIT_REQUESTS` | `100` | Requests per window per IP (`0` disables) |
| `RATE_LIMIT_WINDOW` | `1m` | Rate-limit window |

## Project layout

```
cmd/
  api/                HTTP API entrypoint (producer)
  worker/             Notification worker entrypoint (consumer)
internal/
  auth/               Password hashing, JWT, refresh tokens
  cache/              Redis client + cache/rate-limit primitives
  config/             Env configuration
  database/           Postgres pool + migrations
  events/             Kafka producer/consumer + event schema
  handler/            HTTP handlers
  middleware/         JWT, RBAC, rate limiting
  model/              Domain models
  repository/         Database access
  service/            Business logic (auth, notifications)
  worker/             Concurrent worker pool + notifier
  logger/             slog setup
  router/             Route registration
  server/             HTTP server lifecycle
migrations/           SQL schema reference
Dockerfile            Multi-stage build for api + worker
docker-compose.yml    Full local stack
```

## Design notes

- **At-least-once delivery:** offsets commit only after the terminal state is
  persisted. A crash mid-processing re-delivers the message on restart.
- **Refresh-token rotation:** each refresh invalidates the presented token and
  issues a new pair, limiting the blast radius of a leaked token.
- **Fail-open rate limiting:** a Redis outage logs a warning and allows the
  request rather than hard-failing the API.
- **`internal/`** keeps application code private to the module — a common Go
  convention for services meant to evolve without leaking implementation
  details.

## License

MIT (add a `LICENSE` file when you open-source the repo publicly).
