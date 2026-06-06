# PSP Integration Demo

A production-quality Go service demonstrating a payment service provider (PSP) integration with 3D Secure support, idempotent charge creation, and asynchronous webhook processing.

## Demo

**Charge initiation** — create a charge, retrieve by ID:

![Charge flow](demos/charge-flow.gif)

**Idempotency replay** — same `X-Idempotency-Key` returns cached response with `X-Idempotent-Replay: true`:

![Idempotency](demos/idempotency.gif)

**Webhook ingestion** — valid signed event accepted, tampered signature rejected:

![Webhook](demos/webhook.gif)

**Error handling** — declined card (402), not found (404), 3DS required:

![Errors](demos/errors.gif)

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

## API Documentation

Swagger UI available at `http://localhost:8080/docs` when the service is running.
OpenAPI 3.0 spec: `http://localhost:8080/openapi.yaml`

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

---

## Further Reading

Reference material grouped by the concepts this project exercises.

### Idempotency

The hardest part of payments infrastructure — getting this wrong causes double charges or lost money.

- [Stripe: Idempotent Requests](https://stripe.com/docs/api/idempotent_requests) — the canonical API design for idempotency keys; covers key lifetime, replay semantics, and edge cases
- [Designing robust and predictable APIs with idempotency](https://stripe.com/blog/idempotency) — Stripe engineering blog deep-dive: how they store idempotency keys, handle concurrent requests, and what "same response" actually means
- [The Idempotency-Key HTTP Header Field (IETF draft)](https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/) — emerging RFC standardising the header this project uses
- [How Exactly Once Message Delivery Works](https://bravenewgeek.com/you-cannot-have-exactly-once-delivery/) — why "exactly once" is impossible at the network layer and why idempotency at the application layer is the real solution

### Webhook Security

- [Stripe: Check webhook signatures](https://stripe.com/docs/webhooks/signatures) — HMAC-SHA256 + timestamp replay protection; exactly what `ValidateWebhookSignature` implements
- [Standard Webhooks specification](https://www.standardwebhooks.com/) — emerging open standard (backed by Svix, Clerk, etc.) for webhook signatures, retry semantics, and event schemas
- [Webhook Delivery Guarantees](https://docs.svix.com/receiving/introduction) — at-least-once delivery, idempotent handlers, why you must store-then-ack (the pattern this project uses)

### 3DS / EMV 3-D Secure

- [EMVCo 3DS Overview](https://www.emvco.com/emv-technologies/3d-secure/) — the actual spec body; useful for understanding what "frictionless" vs "challenge" flows mean
- [Stripe: 3D Secure authentication](https://stripe.com/docs/payments/3d-secure) — practical walkthrough of the `requires_action` → `use_stripe_sdk` → confirm flow this project models as a state machine
- [3DS2 explained for developers](https://developer.nexigroup.com/monek/en-GB/blog/what-is-3d-secure-2/) — cleaner narrative on the state transitions, liability shift, and why orchestrators need to model this explicitly
- [Liability Shift](https://stripe.com/docs/payments/3d-secure/authentication-flow#disputed-payments) — why 3DS confirmation matters commercially: a confirmed 3DS charge shifts chargeback liability to the issuer

### Payment Orchestration

Why a provider-agnostic interface matters in production payment systems.

- [What is Payment Orchestration?](https://www.adyen.com/knowledge-hub/payment-orchestration) — Adyen's explanation of routing, fallback, and normalisation across acquirers
- [Yuno: How Payment Orchestration Works](https://www.y.uno/blog/what-is-payment-orchestration) — the orchestration model this project's provider interface is designed around
- [Fallback routing and smart retry in payment systems](https://engineering.razorpay.com/building-a-payment-routing-system-for-razorpay-1b1edd3090d0) — Razorpay engineering on building a routing layer; directly analogous to the problem this demo's interface abstraction enables
- [Martin Fowler: Retry Pattern](https://martinfowler.com/articles/patterns-of-distributed-systems/retry.html) — the theory behind idempotent retries with exponential backoff

### API Design for Financial Systems

- [Stripe API design principles](https://dev.to/stripe/designing-apis-for-humans-object-ids-3o5a) — why `ch_xxx` prefixed IDs, versioned endpoints, and expandable objects exist
- [Handling money in software](https://www.joda.org/joda-money/userguide.html) — why this project stores amounts as integer cents, not floats; currency arithmetic pitfalls
- [Idempotency and ordering in distributed systems (Ably)](https://ably.com/blog/message-ordering-and-idempotency) — how ordering and idempotency interact; relevant to webhook processing guarantees

### PostgreSQL Patterns Used Here

- [PostgreSQL: INSERT ON CONFLICT](https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT) — the `upsert` primitive behind the idempotency_keys table
- [SELECT FOR UPDATE and optimistic locking](https://www.2ndquadrant.com/en/blog/postgresql-anti-patterns-read-modify-write-cycles/) — the `updated_at` CAS pattern used in charge updates to prevent lost writes
- [pgx v5 connection pooling](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx#using-a-connection-pool) — why `pgxpool` instead of a single connection; behaviour under concurrent load
