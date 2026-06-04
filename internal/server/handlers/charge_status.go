package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/Tarasa24/psp-integration-demo/internal/repository"
	"github.com/Tarasa24/psp-integration-demo/internal/server/response"
)

// ChargeStatusHandler handles GET /v1/charges/{id}.
type ChargeStatusHandler struct {
	ChargeRepo repository.ChargeRepository
}

// ServeHTTP returns a charge by ID.
func (h *ChargeStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(w, &response.ValidationError{Field: "id", Message: "must be a valid UUID"})
		return
	}

	charge, err := h.ChargeRepo.Get(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(chargeToResponse(charge))
}
