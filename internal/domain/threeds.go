package domain

// ThreeDSStatus represents the 3D Secure authentication state for a charge.
type ThreeDSStatus string

const (
	ThreeDSPending        ThreeDSStatus = "pending"
	ThreeDSRequiresAction ThreeDSStatus = "requires_action"
	ThreeDSConfirmed      ThreeDSStatus = "confirmed"
	ThreeDSFailed         ThreeDSStatus = "failed"
	ThreeDSNotRequired    ThreeDSStatus = "not_required"
)

// ThreeDSAction represents an event that triggers a 3DS state transition.
type ThreeDSAction string

const (
	ThreeDSActionRequireAction ThreeDSAction = "require_action"
	ThreeDSActionConfirm       ThreeDSAction = "confirm"
	ThreeDSActionFail          ThreeDSAction = "fail"
)

// valid transitions maps current state → allowed actions → next state.
var validTransitions = map[ThreeDSStatus]map[ThreeDSAction]ThreeDSStatus{
	ThreeDSPending: {
		ThreeDSActionRequireAction: ThreeDSRequiresAction,
	},
	ThreeDSRequiresAction: {
		ThreeDSActionConfirm: ThreeDSConfirmed,
		ThreeDSActionFail:    ThreeDSFailed,
	},
}

// Advance applies the given action to the current 3DS status.
// It is immutable — returns a new ThreeDSStatus without mutating the receiver.
func (s ThreeDSStatus) Advance(action ThreeDSAction) (ThreeDSStatus, error) {
	if s.IsTerminal() {
		return s, ErrInvalidTransition
	}
	actions, ok := validTransitions[s]
	if !ok {
		return s, ErrInvalidTransition
	}
	next, ok := actions[action]
	if !ok {
		return s, ErrInvalidTransition
	}
	return next, nil
}

// IsTerminal returns true if no further transitions are possible.
func (s ThreeDSStatus) IsTerminal() bool {
	return s == ThreeDSConfirmed || s == ThreeDSFailed || s == ThreeDSNotRequired
}
