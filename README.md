# GoTrack — Analytics Instrumentation Service

A production-ready, end-to-end Go-based analytics instrumentation service that collects, validates, and forwards usage/event data from multiple client applications.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                 │
│              REST (POST/GET)        gRPC (Unary/Stream)         │
└──────────┬──────────────────────────────┬───────────────────────┘
           │                              │
           ▼                              ▼
┌─────────────────────┐    ┌─────────────────────────┐
│   Gin REST Server   │    │     gRPC Server          │
│   :8080             │    │     :9090                │
│                     │    │                          │
│ • Auth Middleware    │    │ • IngestEvents (unary)   │
│ • Request ID        │    │ • StreamEvents (stream)  │
│ • Structured Log    │    │                          │
│ • Prometheus /metrics│    │                          │
└────────┬────────────┘    └──────────┬──────────────┘
         │                            │
         ▼                            ▼
┌──────────────────────────────────────────────────┐
│                 Service Layer                     │
│                                                  │
│  • Batch ingestion     • Event validation        │
│  • Dedup via Redis     • Quarantine malformed     │
│  • Query with filters  • Health checks           │
└──────────┬──────────────────────┬────────────────┘
           │                      │
           ▼                      ▼
┌──────────────────┐   ┌──────────────────┐
│   PostgreSQL     │   │     Redis        │
│   :5432          │   │     :6379        │
│                  │   │                  │
│ • events         │   │ • dedup keys     │
│ • quarantined    │   │ • API key cache  │
│ • api_keys       │   │                  │
└──────────────────┘   └──────────────────┘
```

## Features

- REST + gRPC event ingestion (batch and streaming)
- Schema validation with quarantine for malformed events
- Redis-based deduplication and API key caching
- PostgreSQL storage with indexed queries and pagination
- Multi-tenant support via source_id and API key auth
- Prometheus metrics (`events_received_total`, `events_failed_total`, `ingestion_latency_seconds`)
- Structured JSON logging with request IDs (Zap)
- Graceful shutdown handling
- Environment-based config (Viper + .env)

## Project Structure

```
/cmd/server          → Application entrypoint
/internal/api/rest   → REST handlers (Gin)
/internal/api/grpc   → gRPC handlers
/internal/service    → Business logic
/internal/repository → Database layer (PostgreSQL + Redis)
/internal/model      → Data structures
/internal/validator  → Event schema validation
/pkg/middleware      → Auth, logging, metrics middleware
/pkg/logger          → Zap logger setup
/proto               → gRPC proto + generated stubs
/config              → Viper-based configuration
/migrations          → SQL migration files
/docker              → Dockerfile
```

## Quick Start

### Run with Docker Compose

```bash
docker-compose up --build
```

This starts PostgreSQL, Redis, and GoTrack. The migration runs automatically on Postgres startup.

### Run Locally (requires Postgres + Redis running)

```bash
# Apply migration
psql -h localhost -U gotrack -d gotrack -f migrations/000001_init.up.sql

# Run
go run ./cmd/server
```

### Run Tests

```bash
go test ./...
```

## API Reference

### Authentication

All `/api/v1/*` endpoints require the `X-API-Key` header.

Pre-seeded test keys:
| Key | Source ID |
|---|---|
| `test-api-key-1` | `source-alpha` |
| `test-api-key-2` | `source-beta` |

### POST /api/v1/events — Ingest Batch Events

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-api-key-1" \
  -d '{
    "events": [
      {
        "source_id": "source-alpha",
        "type": "page_view",
        "payload": "{\"page\":\"/home\",\"duration\":1200}",
        "timestamp": "2026-03-28T10:00:00Z"
      },
      {
        "source_id": "source-alpha",
        "type": "click",
        "payload": "{\"button\":\"signup\"}"
      }
    ]
  }'
```

Response:
```json
{
  "accepted": 2,
  "rejected": 0,
  "duplicates": 0,
  "rejected_ids": null
}
```

### GET /api/v1/events — Query Events

```bash
curl "http://localhost:8080/api/v1/events?source=source-alpha&type=click&from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z&page=1&page_size=10" \
  -H "X-API-Key: test-api-key-1"
```

Response:
```json
{
  "data": [...],
  "page": 1,
  "page_size": 10,
  "total_count": 1
}
```

### GET /health — Health Check

```bash
curl http://localhost:8080/health
```

### GET /metrics — Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

### gRPC — IngestEvents (Unary)

```bash
grpcurl -plaintext \
  -d '{"events":[{"source_id":"src-1","type":"click","payload":"{\"btn\":\"ok\"}"}],"api_key":"test-api-key-1"}' \
  localhost:9090 gotrack.EventService/IngestEvents
```

### gRPC — StreamEvents (Client Streaming)

Use a gRPC client to stream `EventInput` messages and receive a single `IngestResponse` when the stream closes.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_HTTP_PORT` | `8080` | REST server port |
| `SERVER_GRPC_PORT` | `9090` | gRPC server port |
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `gotrack` | Database user |
| `DB_PASSWORD` | `gotrack_secret` | Database password |
| `DB_NAME` | `gotrack` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | `` | Redis password |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
# Gotrack
