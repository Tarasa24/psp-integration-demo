package repository

import (
	"context"
	"time"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/google/uuid"
)

// ChargeRepository handles persistence of Charge entities.
type ChargeRepository interface {
	// Create inserts a new charge. Returns existing charge if idempotency key matches.
	Create(ctx context.Context, charge *domain.Charge) (*domain.Charge, error)
	// Get retrieves a charge by its internal UUID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Charge, error)
	// Update persists status and 3DS status changes to a charge.
	Update(ctx context.Context, charge *domain.Charge) error
	// GetByIdempotencyKey returns a charge matching the given idempotency key.
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Charge, error)
	// GetByProviderRef returns a charge matching the provider's reference ID.
	GetByProviderRef(ctx context.Context, providerRef string) (*domain.Charge, error)
}

// WebhookRepository handles persistence of WebhookEvent entities.
type WebhookRepository interface {
	// Store persists a new webhook event. Idempotent — ignores duplicate event IDs.
	Store(ctx context.Context, event *domain.WebhookEvent) error
	// Get retrieves a webhook event by its UUID.
	Get(ctx context.Context, id uuid.UUID) (*domain.WebhookEvent, error)
	// MarkProcessed sets processed_at on the webhook event.
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	// ListUnprocessed returns all verified, unprocessed events.
	ListUnprocessed(ctx context.Context) ([]*domain.WebhookEvent, error)
}

// IdempotencyRepository handles deduplication of HTTP requests.
type IdempotencyRepository interface {
	// Store upserts an idempotency record (insert-or-update on conflict).
	Store(ctx context.Context, key *domain.IdempotencyKey) error
	// Get retrieves a non-expired idempotency record by key string.
	Get(ctx context.Context, key string) (*domain.IdempotencyKey, error)
	// Exists checks whether a non-expired record exists for the given key.
	Exists(ctx context.Context, key string) (bool, error)
	// Delete removes an idempotency record (e.g. on failure before response).
	Delete(ctx context.Context, key string) error
	// Cleanup removes expired records older than the given time.
	Cleanup(ctx context.Context, before time.Time) (int64, error)
}
