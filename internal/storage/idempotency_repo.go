package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

// PostgresIdempotencyRepository implements repository.IdempotencyRepository.
type PostgresIdempotencyRepository struct {
	pool *pgxpool.Pool
}

// NewIdempotencyRepository constructs a PostgresIdempotencyRepository.
func NewIdempotencyRepository(pool *pgxpool.Pool) *PostgresIdempotencyRepository {
	return &PostgresIdempotencyRepository{pool: pool}
}

const storeIdempotencySQL = `
INSERT INTO idempotency_keys (key, status_code, body, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE
    SET status_code = EXCLUDED.status_code,
        body        = EXCLUDED.body,
        expires_at  = EXCLUDED.expires_at`

// Store upserts an idempotency record.
func (r *PostgresIdempotencyRepository) Store(ctx context.Context, k *domain.IdempotencyKey) error {
	_, err := r.pool.Exec(ctx, storeIdempotencySQL, k.Key, k.StatusCode, k.Body, k.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store idempotency key: %w", err)
	}
	return nil
}

const getIdempotencySQL = `
SELECT key, status_code, body, created_at, expires_at
FROM idempotency_keys
WHERE key = $1 AND expires_at > NOW()`

// Get retrieves a non-expired idempotency record.
func (r *PostgresIdempotencyRepository) Get(ctx context.Context, key string) (*domain.IdempotencyKey, error) {
	row := r.pool.QueryRow(ctx, getIdempotencySQL, key)
	k := &domain.IdempotencyKey{}
	err := row.Scan(&k.Key, &k.StatusCode, &k.Body, &k.CreatedAt, &k.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return k, nil
}

const existsIdempotencySQL = `
SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE key = $1 AND expires_at > NOW())`

// Exists checks whether a non-expired record exists for the given key.
func (r *PostgresIdempotencyRepository) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, existsIdempotencySQL, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists idempotency key: %w", err)
	}
	return exists, nil
}

const deleteIdempotencySQL = `DELETE FROM idempotency_keys WHERE key = $1`

// Delete removes an idempotency record.
func (r *PostgresIdempotencyRepository) Delete(ctx context.Context, key string) error {
	_, err := r.pool.Exec(ctx, deleteIdempotencySQL, key)
	if err != nil {
		return fmt.Errorf("delete idempotency key: %w", err)
	}
	return nil
}

const cleanupIdempotencySQL = `DELETE FROM idempotency_keys WHERE expires_at < $1`

// Cleanup removes expired records older than the given time.
func (r *PostgresIdempotencyRepository) Cleanup(ctx context.Context, before time.Time) (int64, error) {
	ct, err := r.pool.Exec(ctx, cleanupIdempotencySQL, before)
	if err != nil {
		return 0, fmt.Errorf("cleanup idempotency keys: %w", err)
	}
	return ct.RowsAffected(), nil
}
