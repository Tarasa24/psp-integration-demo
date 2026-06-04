package stripe

import (
	"errors"

	stripelib "github.com/stripe/stripe-go/v82"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

// mapStripeError converts a stripe SDK error into a domain error.
// No stripe error types leak past this boundary.
func mapStripeError(err error) error {
	if err == nil {
		return nil
	}

	var stripeErr *stripelib.Error
	if !errors.As(err, &stripeErr) {
		// network or unknown error
		return domain.NewProviderError("provider_unavailable", err.Error())
	}

	switch stripeErr.Type {
	case stripelib.ErrorTypeCard:
		code := "card_declined"
		if stripeErr.DeclineCode != "" {
			code = string(stripeErr.DeclineCode)
		}
		return domain.NewProviderError(code, stripeErr.Msg)
	case stripelib.ErrorTypeInvalidRequest:
		return domain.NewProviderError("invalid_request", stripeErr.Msg)
	case stripelib.ErrorTypeAPI:
		return domain.NewProviderError("api_error", stripeErr.Msg)
	case stripelib.ErrorTypeIdempotency:
		return domain.NewProviderError("idempotency_error", stripeErr.Msg)
	default:
		return domain.NewProviderError(string(stripeErr.Type), stripeErr.Msg)
	}
}
