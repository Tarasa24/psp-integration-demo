-- 001_init.sql
-- Initial schema for psp-integration-demo

CREATE TABLE IF NOT EXISTS charges (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     TEXT        NOT NULL,
    provider_ref    TEXT        NOT NULL DEFAULT '',
    amount          BIGINT      NOT NULL,
    currency        TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending',
    three_ds_status TEXT        NOT NULL DEFAULT 'not_required',
    idempotency_key TEXT        NOT NULL UNIQUE,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_charges_provider_ref    ON charges (provider_ref);
CREATE INDEX IF NOT EXISTS idx_charges_idempotency_key ON charges (idempotency_key);
CREATE INDEX IF NOT EXISTS idx_charges_status          ON charges (status);

CREATE TABLE IF NOT EXISTS webhook_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider     TEXT        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      BYTEA       NOT NULL,
    verified     BOOLEAN     NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed
    ON webhook_events (verified, processed_at)
    WHERE verified = TRUE AND processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_webhook_events_provider ON webhook_events (provider);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key         TEXT        PRIMARY KEY,
    status_code INT         NOT NULL,
    body        BYTEA       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at ON idempotency_keys (expires_at);
