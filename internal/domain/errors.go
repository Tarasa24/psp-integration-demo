package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for the domain layer.
var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyProcessed  = errors.New("already processed")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidTransition = errors.New("invalid state transition")
)

// ErrProviderError wraps a provider-specific error with a machine-readable code.
type ErrProviderError struct {
	Code    string
	Message string
}

func (e *ErrProviderError) Error() string {
	return fmt.Sprintf("provider error [%s]: %s", e.Code, e.Message)
}

// NewProviderError constructs an ErrProviderError.
func NewProviderError(code, message string) *ErrProviderError {
	return &ErrProviderError{Code: code, Message: message}
}
