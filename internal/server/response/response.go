// Package response provides HTTP response helpers and shared API types.
package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/google/uuid"
)

// ------------------------------------------------------------------
// Shared API types
// ------------------------------------------------------------------

// CreateChargeRequest is the JSON body for POST /v1/charges.
type CreateChargeRequest struct {
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate performs basic input validation.
func (r *CreateChargeRequest) Validate() error {
	if r.Amount <= 0 {
		return &ValidationError{Field: "amount", Message: "must be a positive integer (cents)"}
	}
	if len(r.Currency) != 3 {
		return &ValidationError{Field: "currency", Message: "must be a 3-letter ISO 4217 code"}
	}
	return nil
}

// ValidationError is returned when request input fails validation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ChargeResponse is the JSON representation returned to clients for a charge.
type ChargeResponse struct {
	ID             uuid.UUID            `json:"id"`
	ProviderID     string               `json:"provider_id"`
	ProviderRef    string               `json:"provider_ref"`
	Amount         int64                `json:"amount"`
	Currency       string               `json:"currency"`
	Status         domain.ChargeStatus  `json:"status"`
	ThreeDSStatus  domain.ThreeDSStatus `json:"three_ds_status"`
	IdempotencyKey string               `json:"idempotency_key"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

// HealthResponse is the JSON body for GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// ErrorBody is the JSON body returned on errors.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ------------------------------------------------------------------
// Error helpers
// ------------------------------------------------------------------

// WriteError writes a JSON error response with the appropriate HTTP status.
func WriteError(w http.ResponseWriter, err error) {
	status, body := MapDomainError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// MapDomainError translates a domain error to HTTP status + ErrorBody.
func MapDomainError(err error) (int, ErrorBody) {
	if err == nil {
		return http.StatusInternalServerError, ErrorBody{
			Code: "internal_error", Message: "unexpected nil error",
		}
	}

	if errors.Is(err, domain.ErrNotFound) {
		return http.StatusNotFound, ErrorBody{Code: "not_found", Message: "resource not found"}
	}
	if errors.Is(err, domain.ErrAlreadyProcessed) {
		return http.StatusConflict, ErrorBody{Code: "already_processed", Message: err.Error()}
	}
	if errors.Is(err, domain.ErrInvalidSignature) {
		return http.StatusBadRequest, ErrorBody{Code: "invalid_signature", Message: "webhook signature invalid"}
	}
	if errors.Is(err, domain.ErrInvalidTransition) {
		return http.StatusUnprocessableEntity, ErrorBody{Code: "invalid_transition", Message: err.Error()}
	}

	var ve *ValidationError
	if errors.As(err, &ve) {
		return http.StatusBadRequest, ErrorBody{Code: "validation_error", Message: err.Error()}
	}

	var pe *domain.ErrProviderError
	if errors.As(err, &pe) {
		return http.StatusPaymentRequired, ErrorBody{Code: pe.Code, Message: pe.Message}
	}

	return http.StatusInternalServerError, ErrorBody{Code: "internal_error", Message: "internal server error"}
}
