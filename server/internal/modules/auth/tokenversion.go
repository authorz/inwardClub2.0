package auth

import (
	"context"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// MemberTokenVersions adapts the member and staff stores to
// authn.TokenVersionChecker so mini access tokens are checked against the
// identity that issued them.
type MemberTokenVersions struct {
	members MemberRepository
	staff   StaffRepository
}

// NewMemberTokenVersions builds the mini-audience token-version checker.
func NewMemberTokenVersions(members MemberRepository, staff ...StaffRepository) *MemberTokenVersions {
	var staffRepo StaffRepository
	if len(staff) > 0 {
		staffRepo = staff[0]
	}
	return &MemberTokenVersions{members: members, staff: staffRepo}
}

// CurrentTokenVersion returns the member or staff binding token_version.
func (v *MemberTokenVersions) CurrentTokenVersion(ctx context.Context, subject authn.SubjectType, id int64) (int64, error) {
	if subject == authn.SubjectStaff {
		if v.staff == nil {
			return 0, apperr.NotFound("staff account not found")
		}
		member, err := v.members.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if member.Status != StatusActive {
			return 0, apperr.NotFound("member not found")
		}
		staff, err := v.staff.GetByMemberID(ctx, id)
		if err != nil {
			return 0, err
		}
		if staff.Status != StatusActive {
			return 0, apperr.NotFound("staff account not found")
		}
		return staff.TokenVersion, nil
	}
	m, err := v.members.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return m.TokenVersion, nil
}

// ExternalID returns the member's WeChat OpenID for internal diagnostics. It is
// never embedded in tokens or returned by the authentication middleware.
func (v *MemberTokenVersions) ExternalID(ctx context.Context, _ authn.SubjectType, id int64) (string, error) {
	m, err := v.members.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return m.WeChatOpenID, nil
}

// AccountTokenVersions adapts the account store to authn.TokenVersionChecker so
// the admin/store access-token middleware can reject tokens minted before an
// account logout, disable or password reset (all of which bump token_version).
type AccountTokenVersions struct{ accounts AccountRepository }

// NewAccountTokenVersions builds the admin/store-audience token-version checker.
func NewAccountTokenVersions(accounts AccountRepository) *AccountTokenVersions {
	return &AccountTokenVersions{accounts: accounts}
}

// CurrentTokenVersion returns the account's stored token_version.
func (v *AccountTokenVersions) CurrentTokenVersion(ctx context.Context, _ authn.SubjectType, id int64) (int64, error) {
	a, err := v.accounts.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return a.TokenVersion, nil
}
