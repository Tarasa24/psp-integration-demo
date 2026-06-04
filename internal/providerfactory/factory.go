// Package providerfactory constructs provider.Provider implementations.
// It lives outside the provider package to avoid import cycles.
package providerfactory

import (
	"fmt"

	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/provider/mock"
	stripeProvider "github.com/Tarasa24/psp-integration-demo/internal/provider/stripe"
)

// Config holds provider-specific configuration sourced from environment variables.
type Config struct {
	StripeAPIKey        string
	StripeWebhookSecret string
	MockWebhookSecret   string
}

// NewProvider constructs the named provider using cfg.
// Returns an error if name is unknown or required config is missing.
func NewProvider(name string, cfg Config) (provider.Provider, error) {
	switch name {
	case "stripe":
		if cfg.StripeAPIKey == "" {
			return nil, fmt.Errorf("provider stripe: STRIPE_API_KEY is required")
		}
		return stripeProvider.New(cfg.StripeAPIKey, cfg.StripeWebhookSecret), nil
	case "mock":
		return mock.New(cfg.MockWebhookSecret), nil
	default:
		return nil, fmt.Errorf("provider: unknown provider %q", name)
	}
}
