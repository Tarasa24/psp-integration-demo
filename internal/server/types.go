// Package server re-exports types from server/response for convenience.
package server

import "github.com/Tarasa24/psp-integration-demo/internal/server/response"

// Re-export aliases so existing callers of server.ValidationError etc. work.
type (
	ValidationError     = response.ValidationError
	CreateChargeRequest = response.CreateChargeRequest
	ChargeResponse      = response.ChargeResponse
	HealthResponse      = response.HealthResponse
)
