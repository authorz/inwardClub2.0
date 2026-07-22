package auth

import "context"

// MemberTokenVersions adapts the member store to authn.TokenVersionChecker so
// the mini access-token middleware can reject tokens minted before a member
// logout (LogoutMember bumps token_version).
type MemberTokenVersions struct{ members MemberRepository }

// NewMemberTokenVersions builds the mini-audience token-version checker.
func NewMemberTokenVersions(members MemberRepository) *MemberTokenVersions {
	return &MemberTokenVersions{members: members}
}

// CurrentTokenVersion returns the member's stored token_version.
func (v *MemberTokenVersions) CurrentTokenVersion(ctx context.Context, id int64) (int64, error) {
	m, err := v.members.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return m.TokenVersion, nil
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
func (v *AccountTokenVersions) CurrentTokenVersion(ctx context.Context, id int64) (int64, error) {
	a, err := v.accounts.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return a.TokenVersion, nil
}
