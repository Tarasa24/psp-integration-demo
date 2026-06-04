package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChargeStatus represents the lifecycle state of a charge.
type ChargeStatus string

const (
	StatusPending        ChargeStatus = "pending"
	StatusRequiresAction ChargeStatus = "requires_action"
	StatusConfirmed      ChargeStatus = "confirmed"
	StatusFailed         ChargeStatus = "failed"
)

// Charge represents a payment charge in the system.
type Charge struct {
	ID             uuid.UUID         `json:"id"`
	ProviderID     string            `json:"provider_id"`
	ProviderRef    string            `json:"provider_ref"`
	Amount         int64             `json:"amount"` // in cents
	Currency       string            `json:"currency"`
	Status         ChargeStatus      `json:"status"`
	ThreeDSStatus  ThreeDSStatus     `json:"three_ds_status"`
	IdempotencyKey string            `json:"idempotency_key"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// WebhookEvent represents a raw webhook event received from a provider.
type WebhookEvent struct {
	ID          uuid.UUID  `json:"id"`
	Provider    string     `json:"provider"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"-"`
	Verified    bool       `json:"verified"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// IdempotencyKey stores the result of a previously processed request.
type IdempotencyKey struct {
	Key        string    `json:"key"`
	StatusCode int       `json:"status_code"`
	Body       []byte    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
