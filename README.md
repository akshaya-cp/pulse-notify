# Pulse Notify

A backend service for user authentication and asynchronous notifications, built in Go. The goal is a clean, production-style codebase you can run locally, extend incrementally, and reason about in system design discussions.

Inspired by patterns from projects like [go-notify](https://github.com/Harry-027/go-notify), but kept simpler: one API service first, then auth, messaging, and caching in later steps.

## What works today

- HTTP API with [Gin](https://github.com/gin-gonic/gin)
- Environment-based configuration (`.env` for local dev)
- Structured logging with `log/slog` (text in dev, JSON in production)
- `GET /health` for liveness checks
- Graceful shutdown on `SIGINT` / `SIGTERM`

## Planned capabilities

| Area | Status |
|------|--------|
| PostgreSQL + user signup/login | Upcoming |
| JWT access tokens + refresh flow | Upcoming |
| Role-based access control | Upcoming |
| Notification APIs + async workers | Upcoming |
| Kafka (or RabbitMQ) producer/consumer | Upcoming |
| Redis caching, retries, rate limiting | Upcoming |
| Docker Compose for local stack | Upcoming |

## Tech stack

**Current:** Go, Gin, structured logging (`slog`)

**Roadmap:** PostgreSQL, Redis, Kafka or RabbitMQ, JWT, Docker

## Requirements

- Go 1.22+ (project uses 1.26)
- Make optional; plain `go` commands work fine

## Quick start

```bash
git clone https://github.com/akshaya-cp/pulse-notify.git
cd pulse-notify

cp .env.example .env

go mod download
go run ./cmd/api
```

In another terminal:

```bash
curl http://localhost:8080/health
```

Example response:

```json
{
  "status": "ok",
  "service": "pulse-notify-api",
  "uptime": "12s",
  "timestamp": "2026-05-29T12:00:00Z"
}
```

### Build a binary

```bash
go build -o bin/api ./cmd/api
./bin/api
```

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | `development` loads `.env`; use `production` in deploy |
| `HTTP_HOST` | `0.0.0.0` | Bind address |
| `HTTP_PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Project layout

```
cmd/api/              Application entrypoint
internal/
  config/             Env configuration
  handler/            HTTP handlers (health, auth later, etc.)
  logger/             slog setup
  router/             Route registration + middleware
  server/             HTTP server lifecycle
```

`internal/` keeps application code private to this module—a common Go convention for services meant to evolve without leaking implementation details.

## Development notes

- **Port already in use:** another instance may still be running. Check with `lsof -i :8080` and stop it, or change `HTTP_PORT` in `.env`.
- **Gin debug lines:** expected when `APP_ENV=development`. Production mode disables them via `gin.ReleaseMode`.

## License

MIT (add a `LICENSE` file when you open-source the repo publicly.)
