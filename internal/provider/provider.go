package provider

import (
	"context"
	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

// ChargeRequest is the provider-agnostic input for creating a charge.
type ChargeRequest struct {
	Amount         int64
	Currency       string
	IdempotencyKey string
	Metadata       map[string]string
}

// ChargeResponse is the provider-agnostic result of a charge creation.
type ChargeResponse struct {
	ProviderRef   string
	Status        domain.ChargeStatus
	ThreeDSStatus domain.ThreeDSStatus
}

// WebhookEventPayload is the normalised representation of a provider webhook event.
type WebhookEventPayload struct {
	EventType string
	ChargeID  string
	Status    domain.ChargeStatus
	RawData   map[string]interface{}
}

// Provider is the abstraction over any payment service provider.
type Provider interface {
	// Name returns the canonical provider identifier (e.g. "stripe", "mock").
	Name() string
	// Charge creates a payment charge at the provider.
	Charge(ctx context.Context, req ChargeRequest) (ChargeResponse, error)
	// ValidateWebhookSignature returns true only when the signature is authentic.
	ValidateWebhookSignature(payload []byte, signature string) bool
	// ParseWebhookEvent decodes a raw webhook body into a normalised payload.
	ParseWebhookEvent(payload []byte) (WebhookEventPayload, error)
}
