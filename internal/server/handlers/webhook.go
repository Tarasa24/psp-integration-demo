package handlers

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/repository"
	"github.com/Tarasa24/psp-integration-demo/internal/server/response"
)

const maxWebhookBodyBytes = 65536

// WebhookHandler handles POST /v1/webhooks/{provider}.
// It validates signature, stores the raw event, and returns 200.
// All business logic is delegated to the async processor.
type WebhookHandler struct {
	Providers   map[string]provider.Provider
	WebhookRepo repository.WebhookRepository
}

// ServeHTTP is the hard-gate webhook ingestion handler.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")

	prov, ok := h.Providers[providerName]
	if !ok {
		response.WriteError(w, &response.ValidationError{
			Field: "provider", Message: "unknown provider: " + providerName,
		})
		return
	}

	// Read raw body before signature validation.
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		response.WriteError(w, &response.ValidationError{Field: "body", Message: "failed to read request body"})
		return
	}

	// Hard gate: validate signature first, before any parsing or storage.
	sig := webhookSignatureHeader(r, providerName)
	if !prov.ValidateWebhookSignature(payload, sig) {
		slog.Warn("webhook signature validation failed",
			"provider", providerName,
			"remote_addr", r.RemoteAddr,
		)
		response.WriteError(w, domain.ErrInvalidSignature)
		return
	}

	// Parse event type for storage metadata — no business logic here.
	parsed, err := prov.ParseWebhookEvent(payload)
	if err != nil {
		slog.Error("failed to parse webhook event", "provider", providerName, "error", err)
		response.WriteError(w, &response.ValidationError{Field: "body", Message: "invalid webhook payload"})
		return
	}

	event := &domain.WebhookEvent{
		Provider:  providerName,
		EventType: parsed.EventType,
		Payload:   payload,
		Verified:  true,
	}

	if err := h.WebhookRepo.Store(r.Context(), event); err != nil {
		slog.Error("failed to store webhook event", "error", err)
		// Return 200 to provider — don't trigger re-delivery on storage failure.
	}

	w.WriteHeader(http.StatusOK)
}

// webhookSignatureHeader returns the appropriate signature header value for the provider.
func webhookSignatureHeader(r *http.Request, providerName string) string {
	switch providerName {
	case "stripe":
		return r.Header.Get("Stripe-Signature")
	default:
		return r.Header.Get("X-Webhook-Signature")
	}
}
