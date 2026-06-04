package domain_test

import (
	"errors"
	"testing"

	"github.com/Tarasa24/psp-integration-demo/internal/domain"
)

func TestThreeDSAdvance_ValidTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     domain.ThreeDSStatus
		action   domain.ThreeDSAction
		expected domain.ThreeDSStatus
	}{
		{
			name:     "pending → requires_action on require_action",
			from:     domain.ThreeDSPending,
			action:   domain.ThreeDSActionRequireAction,
			expected: domain.ThreeDSRequiresAction,
		},
		{
			name:     "requires_action → confirmed on confirm",
			from:     domain.ThreeDSRequiresAction,
			action:   domain.ThreeDSActionConfirm,
			expected: domain.ThreeDSConfirmed,
		},
		{
			name:     "requires_action → failed on fail",
			from:     domain.ThreeDSRequiresAction,
			action:   domain.ThreeDSActionFail,
			expected: domain.ThreeDSFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.from.Advance(tc.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
			// Immutability: original must not change.
			if tc.from == got && tc.from != tc.expected {
				t.Error("Advance mutated the receiver")
			}
		})
	}
}

func TestThreeDSAdvance_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		from   domain.ThreeDSStatus
		action domain.ThreeDSAction
	}{
		{
			name:   "pending → confirm (skip requires_action)",
			from:   domain.ThreeDSPending,
			action: domain.ThreeDSActionConfirm,
		},
		{
			name:   "pending → fail (skip requires_action)",
			from:   domain.ThreeDSPending,
			action: domain.ThreeDSActionFail,
		},
		{
			name:   "confirmed is terminal, cannot advance",
			from:   domain.ThreeDSConfirmed,
			action: domain.ThreeDSActionConfirm,
		},
		{
			name:   "failed is terminal, cannot advance",
			from:   domain.ThreeDSFailed,
			action: domain.ThreeDSActionFail,
		},
		{
			name:   "not_required is terminal, cannot advance",
			from:   domain.ThreeDSNotRequired,
			action: domain.ThreeDSActionRequireAction,
		},
		{
			name:   "requires_action → require_action (no self-loop)",
			from:   domain.ThreeDSRequiresAction,
			action: domain.ThreeDSActionRequireAction,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.from.Advance(tc.action)
			if err == nil {
				t.Fatalf("expected ErrInvalidTransition, got nil (result: %q)", got)
			}
			if !errors.Is(err, domain.ErrInvalidTransition) {
				t.Errorf("expected ErrInvalidTransition, got %v", err)
			}
			// Status must be unchanged on error.
			if got != tc.from {
				t.Errorf("expected unchanged status %q, got %q", tc.from, got)
			}
		})
	}
}

func TestThreeDSIsTerminal(t *testing.T) {
	tests := []struct {
		status   domain.ThreeDSStatus
		terminal bool
	}{
		{domain.ThreeDSPending, false},
		{domain.ThreeDSRequiresAction, false},
		{domain.ThreeDSConfirmed, true},
		{domain.ThreeDSFailed, true},
		{domain.ThreeDSNotRequired, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := tc.status.IsTerminal()
			if got != tc.terminal {
				t.Errorf("IsTerminal(%q) = %v, want %v", tc.status, got, tc.terminal)
			}
		})
	}
}
