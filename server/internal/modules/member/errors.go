package member

import apperr "github.com/inwardclub/server/internal/platform/errors"

// Module-level errors. They reuse the central apperr codes so the HTTP layer
// renders a consistent envelope; no module-specific codes are invented.
var (
	// ErrInviteCodeNotFound is returned when a bind targets an unknown code.
	ErrInviteCodeNotFound = apperr.NotFound("invite code not found")
	// ErrSelfInvite is returned when a member binds their own invite code.
	ErrSelfInvite = apperr.Invalid("cannot bind your own invite code")
	// ErrAlreadyInvited is returned when the member already has an inviter.
	ErrAlreadyInvited = apperr.Conflict("invitation already bound")
	// ErrPhoneBindingUnavailable is returned until the WeChat phone exchange is wired.
	ErrPhoneBindingUnavailable = apperr.NotImplemented("phone binding unavailable")
	// ErrMembershipTierNotFound is returned when an admin write targets an
	// unknown membership tier id.
	ErrMembershipTierNotFound = apperr.NotFound("membership tier not found")
	// ErrRechargeProductNotFound is returned when an admin write targets an
	// unknown recharge product id.
	ErrRechargeProductNotFound = apperr.NotFound("recharge product not found")
)
