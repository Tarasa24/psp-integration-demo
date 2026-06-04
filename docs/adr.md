# Architecture Decision Records

## ADR-001: Provider abstraction with domain error mapping

**Decision**: All payment providers implement a single `provider.Provider` interface. Errors from provider SDKs (e.g. `*stripe.Error`) are mapped to `domain.ErrProviderError` inside the provider package before returning to callers.

**Why**: Coupling the HTTP layer or domain logic to Stripe-specific error types creates a dependency inversion problem and makes provider swaps painful. By enforcing mapping at the boundary, the rest of the codebase never needs to import `stripe-go`.

---

## ADR-002: Idempotency at the storage layer, not in Go memory

**Decision**: Idempotency is enforced by the `idempotency_keys` PostgreSQL table using INSERT … ON CONFLICT DO NOTHING (for initial lock) and read-before-process. A Go map would lose state on restart and is not safe under horizontal scaling.

**Why**: PSP operations must be exactly-once. A pod restart or second instance receiving the same request must return the same response without recharging the customer. Only durable storage provides this guarantee.

---

## ADR-003: Webhook signature as a hard gate

**Decision**: `ValidateWebhookSignature` is called before any payload parsing, business logic, or database writes. `false` → immediate HTTP 400 with no further processing.

**Why**: Processing unsigned or tampered webhooks is a security risk (spoofed charge events). Failing fast prevents TOCTOU attacks and avoids unnecessary DB writes on malformed payloads.

---

## ADR-004: Async webhook processing via polling

**Decision**: The HTTP webhook handler stores the verified raw payload and returns 200 immediately. A background goroutine polls for unprocessed events and advances charge state.

**Why**: PSP providers require fast webhook acknowledgement (typically < 5s) to avoid re-delivery. Synchronous processing risks timeouts under load. The poll loop provides retryability — a failed update is simply retried on the next tick.

---

## ADR-005: 3DS state machine — immutable transitions

**Decision**: `ThreeDSStatus.Advance(action)` returns a new `ThreeDSStatus` without mutating the receiver. Invalid transitions return `ErrInvalidTransition`.

**Why**: Immutability makes the state machine easier to test and reason about. Explicit invalid-transition errors prevent silent state corruption — a `Pending → Confirmed` shortcut would silently skip 3DS authentication.

---

## ADR-006: net/http ServeMux with Go 1.22 pattern matching

**Decision**: We use the standard library `net/http` mux (Go 1.22+) rather than third-party routers.

**Why**: Go 1.22 added method-based and wildcard pattern matching (`{id}`) to `ServeMux`, eliminating the primary reason to reach for chi or echo in simple services. Zero added dependencies, no middleware DSL mismatch.

---

## ADR-007: slog for structured logging

**Decision**: All log output uses `log/slog` (stdlib, Go 1.21+).

**Why**: Third-party loggers (zerolog, zap) add dependencies without meaningful benefit for a service at this scale. `slog` integrates cleanly with Go context and outputs JSON by default.
