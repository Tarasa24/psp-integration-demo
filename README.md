# PSP Integration Demo

A production-quality Go service demonstrating a payment service provider (PSP) integration with 3D Secure support, idempotent charge creation, and asynchronous webhook processing.

## Architecture

```
cmd/server/main.go          entry point, wires dependencies
internal/
  domain/                   types, errors, 3DS state machine
  repository/               repository interfaces (context-first, no impl details)
  provider/                 provider abstraction + stripe and mock implementations
  storage/                  PostgreSQL implementations (pgx/v5)
  server/                   HTTP server, routes, handlers, middleware
  processor/                async webhook event processor (background goroutine)
  providerfactory/          constructs providers (avoids import cycles)
migrations/                 SQL migration files (applied at startup)
```

**Why this structure**: Domain types and interfaces are defined without implementation details so the domain layer has zero external dependencies. Storage implements the repository interfaces; the HTTP layer depends only on the interfaces — enabling easy testing with mock repositories.

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.25+

### Run with Docker Compose

```bash
cp .env.example .env
# Edit .env if needed (default uses mock provider, no Stripe key required)
docker compose up --build -d
curl localhost:8080/health
```

### Run locally (requires Postgres)

```bash
export DATABASE_URL=postgres://psp:psp@localhost:5432/psp?sslmode=disable
export ACTIVE_PROVIDER=mock
export MOCK_WEBHOOK_SECRET=dev-secret
go run ./cmd/server
```

## API Reference

### POST /v1/charges

Create a charge.

**Headers**
- `X-Idempotency-Key` (optional) — client-generated key for deduplication. Auto-generated if absent.
- `Content-Type: application/json`

**Request**
```json
{
  "amount": 4242,
  "currency": "usd",
  "metadata": {
    "order_id": "ord_123",
    "3ds": "true"
  }
}
```

**Response 201**
```json
{
  "id": "uuid",
  "provider_id": "mock",
  "provider_ref": "mock_...",
  "amount": 4242,
  "currency": "usd",
  "status": "confirmed",
  "three_ds_status": "not_required",
  "idempotency_key": "...",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Magic amounts (mock provider)**
| Amount (cents) | Behaviour |
|---|---|
| `424242` | Success (confirmed, 3DS not required) |
| `400002` | Card declined |
| `300042` | Requires 3DS authentication |
| Any other | ErrProviderError unknown card |

**Idempotency replay**: Re-sending the same `X-Idempotency-Key` within 24h returns the original response with `X-Idempotent-Replay: true` header.

---

### GET /v1/charges/{id}

Retrieve a charge by UUID.

**Response 200** — same shape as POST response.

---

### POST /v1/webhooks/{provider}

Ingest a webhook event. `{provider}` must be `mock` or `stripe`.

**Mock provider**
- Signature header: `X-Webhook-Signature` (hex-encoded HMAC-SHA256 of body with `MOCK_WEBHOOK_SECRET`)
- Body: `{"event_type":"charge.succeeded","charge_id":"mock_ref","status":"confirmed"}`

**Stripe provider**
- Signature header: `Stripe-Signature` (standard Stripe format)

Returns `200 OK` on success, `400` on invalid signature.

---

### GET /health

Returns `{"status":"ok"}` with HTTP 200.

---

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL DSN |
| `ACTIVE_PROVIDER` | No | `mock` | `mock` or `stripe` |
| `STRIPE_API_KEY` | If stripe | — | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | If stripe | — | Stripe webhook signing secret |
| `MOCK_WEBHOOK_SECRET` | No | `mock-secret` | HMAC secret for mock webhooks |
| `SERVER_ADDR` | No | `:8080` | Listen address |
| `MIGRATIONS_DIR` | No | `./migrations` | Path to SQL migration files |

## Testing

```bash
# Unit + mock provider tests (no DB required)
go test ./...

# With race detector
go test -race ./...

# With Stripe live tests (requires real API key)
STRIPE_API_KEY=sk_test_... go test ./internal/provider/...
```

## Design Decisions

See [docs/adr.md](docs/adr.md) for full architectural decision records.

Key decisions:
- **Idempotency is in PostgreSQL**, not in-memory — survives restarts and horizontal scaling
- **Webhook signature is a hard gate** — false → 400, no further processing
- **Provider errors mapped to domain errors** — `stripe.Error` never leaks out of `provider/stripe`
- **3DS state machine is immutable** — `Advance()` returns a new status, invalid transitions are explicit errors
- **Standard library only** for routing and logging — Go 1.22 ServeMux + `slog`
