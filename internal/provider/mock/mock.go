package mock

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
)

// Magic amounts used to control mock behaviour.
const (
	AmountSuccess     = int64(424242) // $4,242.42 → success
	AmountDecline     = int64(400002) // $4,000.02 → card declined
	AmountRequires3DS = int64(300042) // $3,000.42 → requires 3DS authentication
)

// Provider is a synthetic acquirer for testing and development.
type Provider struct {
	secret string // HMAC-SHA256 webhook signing secret
}

// New constructs a mock provider with the given signing secret.
func New(secret string) *Provider {
	return &Provider{secret: secret}
}

// Name returns the canonical provider name.
func (p *Provider) Name() string { return "mock" }

// Charge returns a synthetic response based on magic amounts.
func (p *Provider) Charge(_ context.Context, req provider.ChargeRequest) (provider.ChargeResponse, error) {
	switch req.Amount {
	case AmountSuccess:
		return provider.ChargeResponse{
			ProviderRef:   "mock_" + uuid.New().String(),
			Status:        domain.StatusConfirmed,
			ThreeDSStatus: domain.ThreeDSNotRequired,
		}, nil
	case AmountDecline:
		return provider.ChargeResponse{}, domain.NewProviderError("card_declined", "Your card was declined")
	case AmountRequires3DS:
		return provider.ChargeResponse{
			ProviderRef:   "mock_" + uuid.New().String(),
			Status:        domain.StatusRequiresAction,
			ThreeDSStatus: domain.ThreeDSRequiresAction,
		}, nil
	default:
		return provider.ChargeResponse{}, domain.NewProviderError("unknown_card", "unknown card")
	}
}

// ValidateWebhookSignature verifies HMAC-SHA256 of payload against sig.
// Expected sig format: hex-encoded HMAC-SHA256(payload, secret).
func (p *Provider) ValidateWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ParseWebhookEvent decodes a mock webhook JSON body.
// Expected format: {"event_type":"...","charge_id":"...","status":"..."}
func (p *Provider) ParseWebhookEvent(payload []byte) (provider.WebhookEventPayload, error) {
	var raw struct {
		EventType string `json:"event_type"`
		ChargeID  string `json:"charge_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return provider.WebhookEventPayload{}, fmt.Errorf("mock: parse webhook: %w", err)
	}
	return provider.WebhookEventPayload{
		EventType: raw.EventType,
		ChargeID:  raw.ChargeID,
		Status:    domain.ChargeStatus(raw.Status),
		RawData:   map[string]interface{}{},
	}, nil
}
