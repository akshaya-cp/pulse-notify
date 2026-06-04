# Pulse Notify

A backend service for user authentication and asynchronous notifications, built in Go. The goal is a clean, production-style codebase you can run locally, extend incrementally, and reason about in system design discussions.

Inspired by patterns from projects like [go-notify](https://github.com/Harry-027/go-notify), but kept simpler: one API service first, then auth, messaging, and caching in later steps.

## What works today

- HTTP API with [Gin](https://github.com/gin-gonic/gin)
- Environment-based configuration (`.env` for local dev)
- Structured logging with `log/slog` (text in dev, JSON in production)
- `GET /health` — includes PostgreSQL ping
- PostgreSQL user storage with auto-migration on startup
- `POST /api/v1/auth/signup` and `POST /api/v1/auth/login`
- bcrypt password hashing
- JWT access tokens + `GET /api/v1/me` (protected)
- Graceful shutdown on `SIGINT` / `SIGTERM`

## Planned capabilities

| Area | Status |
|------|--------|
| JWT refresh tokens + logout | Upcoming |
| Role-based access control (admin routes) | Upcoming |
| Role-based access control | Upcoming |
| Notification APIs + async workers | Upcoming |
| Kafka (or RabbitMQ) producer/consumer | Upcoming |
| Redis caching, retries, rate limiting | Upcoming |
| Docker Compose for local stack | Upcoming |

## Tech stack

**Current:** Go, Gin, PostgreSQL (pgx), JWT, bcrypt, Docker Compose (Postgres)

**Roadmap:** Redis, Kafka or RabbitMQ, refresh tokens, rate limiting

## Requirements

- Go 1.22+ (project uses 1.26)
- Docker (for local PostgreSQL)

## Quick start

```bash
git clone https://github.com/akshaya-cp/pulse-notify.git
cd pulse-notify

cp .env.example .env

docker compose up -d

go mod download
go run ./cmd/api
```

### Try auth APIs

```bash
# Signup
curl -s -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"password123","name":"Demo User"}'

# Login
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"password123"}'

# Protected profile (replace TOKEN)
curl -s http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer TOKEN"
```

Health check:

```bash
curl http://localhost:8080/health
```

Example response:

```json
{
  "status": "ok",
  "service": "pulse-notify-api",
  "database": "up",
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
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `JWT_SECRET` | — | HMAC secret, min 32 characters (required) |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |

## API overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Liveness + DB ping |
| POST | `/api/v1/auth/signup` | No | Register user |
| POST | `/api/v1/auth/login` | No | Login |
| GET | `/api/v1/me` | Bearer JWT | Current user profile |

## Project layout

```
cmd/api/              Application entrypoint
internal/
  auth/               Password hashing + JWT
  config/             Env configuration
  database/           Postgres pool + migrations
  handler/            HTTP handlers
  middleware/         JWT auth middleware
  model/              Domain models
  repository/         Database access
  service/            Business logic
  logger/             slog setup
  router/             Route registration
  server/             HTTP server lifecycle
migrations/           SQL schema reference
docker-compose.yml    Local PostgreSQL
```

`internal/` keeps application code private to this module—a common Go convention for services meant to evolve without leaking implementation details.

## Development notes

- **Port already in use:** another instance may still be running. Check with `lsof -i :8080` and stop it, or change `HTTP_PORT` in `.env`.
- **Gin debug lines:** expected when `APP_ENV=development`. Production mode disables them via `gin.ReleaseMode`.

## License

MIT (add a `LICENSE` file when you open-source the repo publicly.)
