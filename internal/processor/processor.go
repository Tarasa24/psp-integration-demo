// Package processor contains the background webhook event processor.
package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
	"github.com/Tarasa24/psp-integration-demo/internal/repository"
)

// Processor polls unprocessed webhook events and advances charge state.
type Processor struct {
	ChargeRepo   repository.ChargeRepository
	WebhookRepo  repository.WebhookRepository
	PollInterval time.Duration
}

// New constructs a Processor with the given dependencies and poll interval.
func New(chargeRepo repository.ChargeRepository, webhookRepo repository.WebhookRepository, pollInterval time.Duration) *Processor {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &Processor{
		ChargeRepo:   chargeRepo,
		WebhookRepo:  webhookRepo,
		PollInterval: pollInterval,
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled, then drains
// in-progress work before returning.
func (p *Processor) Run(ctx context.Context) {
	ticker := time.NewTicker(p.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Drain once more on shutdown with a bounded context to avoid hanging.
			drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			p.processAll(drainCtx)
			return
		case <-ticker.C:
			p.processAll(ctx)
		}
	}
}

func (p *Processor) processAll(ctx context.Context) {
	events, err := p.WebhookRepo.ListUnprocessed(ctx)
	if err != nil {
		slog.Error("processor: list unprocessed events", "error", err)
		return
	}

	for _, e := range events {
		if err := p.processEvent(ctx, e); err != nil {
			slog.Error("processor: process event", "event_id", e.ID, "error", err)
		}
	}
}

func (p *Processor) processEvent(ctx context.Context, e *domain.WebhookEvent) error {
	// Decode minimal fields needed from various webhook payload formats.
	var raw struct {
		EventType string `json:"event_type"` // mock format
		ChargeID  string `json:"charge_id"`  // mock format
		Data      struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		} `json:"data"` // stripe format
	}
	if err := json.Unmarshal(e.Payload, &raw); err != nil {
		slog.Warn("processor: could not parse payload, skipping", "event_id", e.ID)
		return p.WebhookRepo.MarkProcessed(ctx, e.ID)
	}

	// Normalise provider ref across mock/stripe formats.
	providerRef := raw.ChargeID
	if providerRef == "" {
		providerRef = raw.Data.Object.ID
	}

	if providerRef == "" {
		slog.Warn("processor: no provider ref in event, skipping", "event_id", e.ID)
		return p.WebhookRepo.MarkProcessed(ctx, e.ID)
	}

	charge, err := p.ChargeRepo.GetByProviderRef(ctx, providerRef)
	if err != nil {
		// Charge may not exist yet (race) — mark processed to avoid infinite retry.
		slog.Warn("processor: charge not found for provider ref",
			"provider_ref", providerRef, "event_id", e.ID)
		return p.WebhookRepo.MarkProcessed(ctx, e.ID)
	}

	switch e.EventType {
	case "payment_intent.succeeded", "charge.succeeded":
		charge.Status = domain.StatusConfirmed
		if !charge.ThreeDSStatus.IsTerminal() {
			if next, err := charge.ThreeDSStatus.Advance(domain.ThreeDSActionConfirm); err == nil {
				charge.ThreeDSStatus = next
			}
		}

	case "payment_intent.payment_failed", "charge.failed":
		charge.Status = domain.StatusFailed
		if !charge.ThreeDSStatus.IsTerminal() {
			if next, err := charge.ThreeDSStatus.Advance(domain.ThreeDSActionFail); err == nil {
				charge.ThreeDSStatus = next
			}
		}

	case "payment_intent.requires_action":
		charge.Status = domain.StatusRequiresAction
		if charge.ThreeDSStatus == domain.ThreeDSPending {
			if next, err := charge.ThreeDSStatus.Advance(domain.ThreeDSActionRequireAction); err == nil {
				charge.ThreeDSStatus = next
			}
		}

	default:
		slog.Debug("processor: unhandled event type", "event_type", e.EventType)
		return p.WebhookRepo.MarkProcessed(ctx, e.ID)
	}

	if err := p.ChargeRepo.Update(ctx, charge); err != nil {
		slog.Error("processor: update charge failed", "charge_id", charge.ID, "error", err)
		// Don't mark processed so we can retry.
		return err
	}

	return p.WebhookRepo.MarkProcessed(ctx, e.ID)
}
