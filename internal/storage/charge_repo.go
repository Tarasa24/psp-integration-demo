package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

// PostgresChargeRepository implements repository.ChargeRepository.
type PostgresChargeRepository struct {
	pool *pgxpool.Pool
}

// NewChargeRepository constructs a PostgresChargeRepository.
func NewChargeRepository(pool *pgxpool.Pool) *PostgresChargeRepository {
	return &PostgresChargeRepository{pool: pool}
}

const createChargeSQL = `
INSERT INTO charges
    (id, provider_id, provider_ref, amount, currency, status, three_ds_status, idempotency_key, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, provider_id, provider_ref, amount, currency, status, three_ds_status, idempotency_key, metadata, created_at, updated_at`

// Create inserts a new charge. On unique violation of idempotency_key, returns
// the existing charge rather than an error.
func (r *PostgresChargeRepository) Create(ctx context.Context, c *domain.Charge) (*domain.Charge, error) {
	meta, err := json.Marshal(c.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	row := r.pool.QueryRow(ctx, createChargeSQL,
		c.ID, c.ProviderID, c.ProviderRef, c.Amount, c.Currency,
		c.Status, c.ThreeDSStatus, c.IdempotencyKey, meta,
	)

	out, err := scanCharge(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique violation on idempotency_key — return existing.
			return r.GetByIdempotencyKey(ctx, c.IdempotencyKey)
		}
		return nil, fmt.Errorf("create charge: %w", err)
	}
	return out, nil
}

const getChargeSQL = `
SELECT id, provider_id, provider_ref, amount, currency, status, three_ds_status, idempotency_key, metadata, created_at, updated_at
FROM charges WHERE id = $1`

// Get retrieves a charge by UUID.
func (r *PostgresChargeRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Charge, error) {
	row := r.pool.QueryRow(ctx, getChargeSQL, id)
	c, err := scanCharge(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get charge: %w", err)
	}
	return c, nil
}

const updateChargeSQL = `
UPDATE charges
SET status = $2, three_ds_status = $3, provider_ref = $4, updated_at = NOW()
WHERE id = $1`

// Update persists status and 3DS status changes.
func (r *PostgresChargeRepository) Update(ctx context.Context, c *domain.Charge) error {
	ct, err := r.pool.Exec(ctx, updateChargeSQL, c.ID, c.Status, c.ThreeDSStatus, c.ProviderRef)
	if err != nil {
		return fmt.Errorf("update charge: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const getByIdempotencySQL = `
SELECT id, provider_id, provider_ref, amount, currency, status, three_ds_status, idempotency_key, metadata, created_at, updated_at
FROM charges WHERE idempotency_key = $1`

// GetByIdempotencyKey returns a charge matching the given idempotency key.
func (r *PostgresChargeRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Charge, error) {
	row := r.pool.QueryRow(ctx, getByIdempotencySQL, key)
	c, err := scanCharge(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get charge by idempotency key: %w", err)
	}
	return c, nil
}

const getByProviderRefSQL = `
SELECT id, provider_id, provider_ref, amount, currency, status, three_ds_status, idempotency_key, metadata, created_at, updated_at
FROM charges WHERE provider_ref = $1`

// GetByProviderRef returns a charge matching the given provider reference.
func (r *PostgresChargeRepository) GetByProviderRef(ctx context.Context, providerRef string) (*domain.Charge, error) {
	row := r.pool.QueryRow(ctx, getByProviderRefSQL, providerRef)
	c, err := scanCharge(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get charge by provider ref: %w", err)
	}
	return c, nil
}

func scanCharge(row pgx.Row) (*domain.Charge, error) {
	c := &domain.Charge{}
	var meta []byte
	err := row.Scan(
		&c.ID, &c.ProviderID, &c.ProviderRef,
		&c.Amount, &c.Currency, &c.Status, &c.ThreeDSStatus,
		&c.IdempotencyKey, &meta,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &c.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return c, nil
}
