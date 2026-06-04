package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

// PostgresWebhookRepository implements repository.WebhookRepository.
type PostgresWebhookRepository struct {
	pool *pgxpool.Pool
}

// NewWebhookRepository constructs a PostgresWebhookRepository.
func NewWebhookRepository(pool *pgxpool.Pool) *PostgresWebhookRepository {
	return &PostgresWebhookRepository{pool: pool}
}

const storeWebhookSQL = `
INSERT INTO webhook_events (id, provider, event_type, payload, verified)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO NOTHING`

// Store persists a webhook event. Silently ignores duplicate IDs (idempotent).
func (r *PostgresWebhookRepository) Store(ctx context.Context, e *domain.WebhookEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, storeWebhookSQL,
		e.ID, e.Provider, e.EventType, e.Payload, e.Verified)
	if err != nil {
		return fmt.Errorf("store webhook event: %w", err)
	}
	return nil
}

const getWebhookSQL = `
SELECT id, provider, event_type, payload, verified, processed_at, created_at
FROM webhook_events WHERE id = $1`

// Get retrieves a webhook event by UUID.
func (r *PostgresWebhookRepository) Get(ctx context.Context, id uuid.UUID) (*domain.WebhookEvent, error) {
	row := r.pool.QueryRow(ctx, getWebhookSQL, id)
	e, err := scanWebhookEvent(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get webhook event: %w", err)
	}
	return e, nil
}

const markProcessedSQL = `
UPDATE webhook_events SET processed_at = NOW() WHERE id = $1 AND processed_at IS NULL`

// MarkProcessed sets processed_at on the given event. Idempotent.
func (r *PostgresWebhookRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, markProcessedSQL, id)
	if err != nil {
		return fmt.Errorf("mark webhook processed: %w", err)
	}
	return nil
}

const listUnprocessedSQL = `
SELECT id, provider, event_type, payload, verified, processed_at, created_at
FROM webhook_events
WHERE verified = TRUE AND processed_at IS NULL
ORDER BY created_at ASC
LIMIT 100`

// ListUnprocessed returns all verified, unprocessed webhook events.
func (r *PostgresWebhookRepository) ListUnprocessed(ctx context.Context) ([]*domain.WebhookEvent, error) {
	rows, err := r.pool.Query(ctx, listUnprocessedSQL)
	if err != nil {
		return nil, fmt.Errorf("list unprocessed webhooks: %w", err)
	}
	defer rows.Close()

	var events []*domain.WebhookEvent
	for rows.Next() {
		e, err := scanWebhookEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan webhook event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func scanWebhookEvent(row pgx.Row) (*domain.WebhookEvent, error) {
	e := &domain.WebhookEvent{}
	err := row.Scan(
		&e.ID, &e.Provider, &e.EventType, &e.Payload,
		&e.Verified, &e.ProcessedAt, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}
