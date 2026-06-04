package provider_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/provider/mock"
	"github.com/Tarasa24/psp-integration-demo/internal/providerfactory"
)

const mockSecret = "test-webhook-secret"

// providerUnderTest returns (name, provider) pairs to run contract tests against.
func providersUnderTest(t *testing.T) []struct {
	name string
	p    provider.Provider
} {
	t.Helper()
	result := []struct {
		name string
		p    provider.Provider
	}{
		{"mock", mock.New(mockSecret)},
	}

	if key := os.Getenv("STRIPE_API_KEY"); key != "" {
		stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		p, err := providerfactory.NewProvider("stripe", providerfactory.Config{
			StripeAPIKey:        key,
			StripeWebhookSecret: stripeWebhookSecret,
		})
		if err != nil {
			t.Fatalf("init stripe provider: %v", err)
		}
		result = append(result, struct {
			name string
			p    provider.Provider
		}{"stripe", p})
	} else {
		t.Log("STRIPE_API_KEY not set — skipping Stripe provider contract tests")
	}

	return result
}

func TestProviderContract_Charge_Success(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "stripe" {
				t.Skip("stripe charge success requires live API; run manually with real key")
			}
			resp, err := tc.p.Charge(context.Background(), provider.ChargeRequest{
				Amount:   mock.AmountSuccess,
				Currency: "usd",
			})
			if err != nil {
				t.Fatalf("Charge error: %v", err)
			}
			if resp.ProviderRef == "" {
				t.Error("expected non-empty ProviderRef")
			}
			if resp.Status != domain.StatusConfirmed {
				t.Errorf("expected StatusConfirmed, got %q", resp.Status)
			}
		})
	}
}

func TestProviderContract_Charge_Decline(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "stripe" {
				t.Skip("stripe decline requires specific test card; run manually")
			}
			_, err := tc.p.Charge(context.Background(), provider.ChargeRequest{
				Amount:   mock.AmountDecline,
				Currency: "usd",
			})
			if err == nil {
				t.Fatal("expected error for declined charge")
			}
			var pe *domain.ErrProviderError
			if !errors.As(err, &pe) {
				t.Errorf("expected ErrProviderError, got %T: %v", err, err)
			}
		})
	}
}

func TestProviderContract_Charge_Requires3DS(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "stripe" {
				t.Skip("stripe 3DS requires live API; run manually with real key")
			}
			resp, err := tc.p.Charge(context.Background(), provider.ChargeRequest{
				Amount:   mock.AmountRequires3DS,
				Currency: "usd",
			})
			if err != nil {
				t.Fatalf("Charge error: %v", err)
			}
			if resp.ProviderRef == "" {
				t.Error("expected non-empty ProviderRef")
			}
			if resp.Status != domain.StatusRequiresAction {
				t.Errorf("expected StatusRequiresAction, got %q", resp.Status)
			}
			if resp.ThreeDSStatus != domain.ThreeDSRequiresAction {
				t.Errorf("expected ThreeDSRequiresAction, got %q", resp.ThreeDSStatus)
			}
		})
	}
}

func TestProviderContract_ValidateWebhookSignature_Valid(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "stripe" {
				t.Skip("stripe webhook sig test requires signed payload; run manually")
			}
			payload := []byte(`{"event_type":"charge.succeeded","charge_id":"mock_123","status":"confirmed"}`)
			sig := mockHMACSig(payload, mockSecret)
			if !tc.p.ValidateWebhookSignature(payload, sig) {
				t.Error("expected valid signature to pass")
			}
		})
	}
}

func TestProviderContract_ValidateWebhookSignature_Invalid(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			payload := []byte(`{"event_type":"charge.succeeded"}`)
			if tc.p.ValidateWebhookSignature(payload, "invalid-sig") {
				t.Error("expected invalid signature to fail")
			}
		})
	}
}

func TestProviderContract_ParseWebhookEvent(t *testing.T) {
	for _, tc := range providersUnderTest(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "stripe" {
				t.Skip("stripe webhook parsing tested separately")
			}
			payload := map[string]string{
				"event_type": "charge.succeeded",
				"charge_id":  "mock_abc123",
				"status":     "confirmed",
			}
			b, _ := json.Marshal(payload)
			ev, err := tc.p.ParseWebhookEvent(b)
			if err != nil {
				t.Fatalf("ParseWebhookEvent error: %v", err)
			}
			if ev.EventType != "charge.succeeded" {
				t.Errorf("EventType = %q, want %q", ev.EventType, "charge.succeeded")
			}
			if ev.ChargeID != "mock_abc123" {
				t.Errorf("ChargeID = %q, want %q", ev.ChargeID, "mock_abc123")
			}
		})
	}
}

func mockHMACSig(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
