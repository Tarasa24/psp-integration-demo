package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	stripelib "github.com/stripe/stripe-go/v82"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
)

// Provider implements provider.Provider backed by Stripe.
type Provider struct {
	client        *stripelib.Client
	webhookSecret string
}

// New constructs a Stripe provider. apiKey and webhookSecret must be non-empty.
func New(apiKey, webhookSecret string) *Provider {
	client := stripelib.NewClient(apiKey)
	return &Provider{
		client:        client,
		webhookSecret: webhookSecret,
	}
}

// Name returns the canonical provider name.
func (p *Provider) Name() string { return "stripe" }

// Charge creates a PaymentIntent at Stripe.
// If metadata["3ds"] == "true" the intent is created with confirm=true and an
// authentication-required payment method so Stripe returns requires_action.
func (p *Provider) Charge(ctx context.Context, req provider.ChargeRequest) (provider.ChargeResponse, error) {
	params := &stripelib.PaymentIntentCreateParams{
		Amount:   stripelib.Int64(req.Amount),
		Currency: stripelib.String(req.Currency),
		AutomaticPaymentMethods: &stripelib.PaymentIntentCreateAutomaticPaymentMethodsParams{
			Enabled: stripelib.Bool(true),
		},
	}

	if len(req.Metadata) > 0 {
		params.Metadata = req.Metadata
	}

	if req.IdempotencyKey != "" {
		params.SetIdempotencyKey(req.IdempotencyKey)
	}

	// If 3DS is requested, confirm immediately with an auth-required test card.
	if req.Metadata["3ds"] == "true" {
		params.PaymentMethod = stripelib.String("pm_card_authenticationRequired")
		params.Confirm = stripelib.Bool(true)
	}

	pi, err := p.client.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		return provider.ChargeResponse{}, mapStripeError(err)
	}

	resp := provider.ChargeResponse{
		ProviderRef:   pi.ID,
		Status:        mapPaymentIntentStatus(pi.Status),
		ThreeDSStatus: domain.ThreeDSNotRequired,
	}

	if pi.Status == stripelib.PaymentIntentStatusRequiresAction {
		resp.Status = domain.StatusRequiresAction
		resp.ThreeDSStatus = domain.ThreeDSRequiresAction
	}

	return resp, nil
}

// ValidateWebhookSignature verifies the Stripe-Signature header against the payload.
func (p *Provider) ValidateWebhookSignature(payload []byte, signature string) bool {
	_, err := stripelib.ConstructEvent(payload, signature, p.webhookSecret,
		stripelib.WithIgnoreAPIVersionMismatch())
	return err == nil
}

// ParseWebhookEvent decodes a raw Stripe webhook body into a normalised payload.
func (p *Provider) ParseWebhookEvent(payload []byte) (provider.WebhookEventPayload, error) {
	var event stripelib.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return provider.WebhookEventPayload{}, fmt.Errorf("stripe: parse event: %w", err)
	}

	out := provider.WebhookEventPayload{
		EventType: string(event.Type),
		RawData:   map[string]interface{}{},
	}

	// Extract PaymentIntent ID and map status for supported event types.
	switch event.Type {
	case "payment_intent.succeeded",
		"payment_intent.payment_failed",
		"payment_intent.requires_action":
		var pi stripelib.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return out, fmt.Errorf("stripe: parse payment_intent: %w", err)
		}
		out.ChargeID = pi.ID
		out.Status = mapPaymentIntentStatus(pi.Status)
	}

	return out, nil
}

// mapPaymentIntentStatus converts a stripe status string to a domain ChargeStatus.
func mapPaymentIntentStatus(s stripelib.PaymentIntentStatus) domain.ChargeStatus {
	switch s {
	case stripelib.PaymentIntentStatusSucceeded:
		return domain.StatusConfirmed
	case stripelib.PaymentIntentStatusRequiresAction:
		return domain.StatusRequiresAction
	case stripelib.PaymentIntentStatusCanceled:
		return domain.StatusFailed
	default:
		return domain.StatusPending
	}
}
