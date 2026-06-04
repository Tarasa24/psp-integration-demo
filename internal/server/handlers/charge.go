package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/repository"
	"github.com/Tarasa24/psp-integration-demo/internal/server/response"
)

// ChargeHandler handles POST /v1/charges.
type ChargeHandler struct {
	Provider        provider.Provider
	ChargeRepo      repository.ChargeRepository
	IdempotencyRepo repository.IdempotencyRepository
	ProviderID      string
}

// ServeHTTP processes charge creation requests.
func (h *ChargeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read idempotency key from header; generate one if absent.
	idemKey := r.Header.Get("X-Idempotency-Key")
	if idemKey == "" {
		idemKey = uuid.New().String()
	}

	ctx := r.Context()

	// Check idempotency store before any processing.
	existing, err := h.IdempotencyRepo.Get(ctx, idemKey)
	if err == nil {
		// Replay cached response.
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Idempotent-Replay", "true")
		w.WriteHeader(existing.StatusCode)
		_, _ = w.Write(existing.Body)
		return
	}

	var req response.CreateChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, &response.ValidationError{Field: "body", Message: "invalid JSON"})
		return
	}
	if err := req.Validate(); err != nil {
		response.WriteError(w, err)
		return
	}

	// Call provider.
	resp, err := h.Provider.Charge(ctx, provider.ChargeRequest{
		Amount:         req.Amount,
		Currency:       req.Currency,
		IdempotencyKey: idemKey,
		Metadata:       req.Metadata,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Persist charge.
	charge := &domain.Charge{
		ProviderID:     h.ProviderID,
		ProviderRef:    resp.ProviderRef,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         resp.Status,
		ThreeDSStatus:  resp.ThreeDSStatus,
		IdempotencyKey: idemKey,
		Metadata:       req.Metadata,
	}
	created, err := h.ChargeRepo.Create(ctx, charge)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	chargeResp := chargeToResponse(created)
	body, _ := json.Marshal(chargeResp)

	// Store idempotency result.
	_ = h.IdempotencyRepo.Store(ctx, &domain.IdempotencyKey{
		Key:        idemKey,
		StatusCode: http.StatusCreated,
		Body:       body,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(body)
}

func chargeToResponse(c *domain.Charge) response.ChargeResponse {
	return response.ChargeResponse{
		ID:             c.ID,
		ProviderID:     c.ProviderID,
		ProviderRef:    c.ProviderRef,
		Amount:         c.Amount,
		Currency:       c.Currency,
		Status:         c.Status,
		ThreeDSStatus:  c.ThreeDSStatus,
		IdempotencyKey: c.IdempotencyKey,
		Metadata:       c.Metadata,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.Format(time.RFC3339),
	}
}
