package auth

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// MemberTierResolver resolves a member's current VIP tier for the "me" surface.
// It is owned by the member module and wired at composition time; when nil, the
// profile is returned without tier info. A nil tier (with nil error) means the
// member is not yet ranked.
type MemberTierResolver interface {
	CurrentTier(ctx context.Context, memberID int64) (*MemberVIPTier, error)
}

// Service implements the authentication flows for all three audiences.
type Service struct {
	tokens   *authn.Manager
	wechat   WeChatClient
	members  MemberRepository
	accounts AccountRepository
	tiers    MemberTierResolver
}

// NewService builds the auth service. tiers may be nil until the member VIP-tier
// resolver is wired; the "me" profile then omits tier info.
func NewService(tokens *authn.Manager, wechat WeChatClient, members MemberRepository, accounts AccountRepository, tiers MemberTierResolver) *Service {
	return &Service{tokens: tokens, wechat: wechat, members: members, accounts: accounts, tiers: tiers}
}

// MiniLogin exchanges a WeChat code for a member session, creating the member on
// first login, and mints a mini-audience access/refresh pair.
func (s *Service) MiniLogin(ctx context.Context, code string) (LoginResponse, error) {
	session, err := s.wechat.Code2Session(ctx, code)
	if err != nil {
		return LoginResponse{}, apperr.Unauthenticated("wechat login failed")
	}
	isNew := false
	member, err := s.members.GetByOpenID(ctx, session.OpenID)
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		id, cerr := s.createMember(ctx, session.OpenID)
		if cerr != nil {
			return LoginResponse{}, cerr
		}
		isNew = true // member did not exist before this login: first-time registration
		member, err = s.members.GetByID(ctx, id)
	}
	if err != nil {
		return LoginResponse{}, err
	}
	if member.Status != StatusActive {
		return LoginResponse{}, apperr.Forbidden("member is disabled")
	}
	pair, err := s.tokens.Issue(authn.Identity{
		SubjectID:    member.ID,
		SubjectType:  authn.SubjectMember,
		Role:         authn.RoleMember,
		Audience:     authn.AudienceMini,
		TokenVersion: member.TokenVersion,
	})
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	return LoginResponse{Token: pair, Profile: memberProfile(member), IsNew: isNew}, nil
}

// createMember inserts a new member on first WeChat login, assigning a unique
// 6-digit numeric invite code. Uniqueness is enforced by the members
// .invite_code UNIQUE constraint; a duplicate-key collision retries with a new
// code (a handful of attempts is ample given the 1e6 code space and small
// membership).
func (s *Service) createMember(ctx context.Context, openID string) (int64, error) {
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := s.members.Create(ctx, Member{
			WeChatOpenID: openID,
			Nickname:     "会员",
			InviteCode:   newInviteCode(),
		})
		if err == nil {
			return id, nil
		}
		if platdb.IsDuplicate(err) {
			continue // invite_code collided — retry with a fresh code
		}
		return 0, err
	}
	return 0, apperr.Internal(fmt.Errorf("could not allocate a unique invite code after %d attempts", maxAttempts))
}

// AdminLogin authenticates a super_admin against the admin console.
func (s *Service) AdminLogin(ctx context.Context, username, password string) (LoginResponse, error) {
	account, err := s.authenticate(ctx, username, password)
	if err != nil {
		return LoginResponse{}, err
	}
	if account.Role != string(authn.RoleSuperAdmin) {
		return LoginResponse{}, apperr.Forbidden("admin console requires super_admin")
	}
	pair, err := s.issueAccountToken(account, authn.AudienceAdmin)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{Token: pair, Profile: accountProfile(account)}, nil
}

// StoreLogin authenticates a store_admin/cashier against the store console. The
// store scope comes from the account binding, never from the request.
func (s *Service) StoreLogin(ctx context.Context, username, password string) (LoginResponse, error) {
	account, err := s.authenticate(ctx, username, password)
	if err != nil {
		return LoginResponse{}, err
	}
	if account.Role != string(authn.RoleStoreAdmin) && account.Role != string(authn.RoleCashier) {
		return LoginResponse{}, apperr.Forbidden("store console requires store_admin or cashier")
	}
	if account.StoreID <= 0 {
		return LoginResponse{}, apperr.New(apperr.CodeStoreScopeRequired, "account has no store binding")
	}
	pair, err := s.issueAccountToken(account, authn.AudienceStore)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{Token: pair, Profile: accountProfile(account)}, nil
}

// Refresh validates a refresh token for the audience and reissues a pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string, audience authn.Audience) (authn.TokenPair, error) {
	claims, err := s.tokens.Parse(refreshToken, audience)
	if err != nil || claims.Kind != authn.TokenRefresh {
		return authn.TokenPair{}, apperr.Unauthenticated("invalid refresh token")
	}
	if audience == authn.AudienceMini {
		member, err := s.members.GetByID(ctx, claims.SubjectID())
		if err != nil {
			return authn.TokenPair{}, err
		}
		if member.TokenVersion != claims.TokenVersion || member.Status != StatusActive {
			return authn.TokenPair{}, apperr.Unauthenticated("session expired")
		}
		return s.tokens.Issue(authn.Identity{
			SubjectID: member.ID, SubjectType: authn.SubjectMember, Role: authn.RoleMember,
			Audience: authn.AudienceMini, TokenVersion: member.TokenVersion,
		})
	}
	account, err := s.accounts.GetByID(ctx, claims.SubjectID())
	if err != nil {
		return authn.TokenPair{}, err
	}
	if account.TokenVersion != claims.TokenVersion || account.Status != StatusActive {
		return authn.TokenPair{}, apperr.Unauthenticated("session expired")
	}
	return s.issueAccountToken(account, audience)
}

// MemberProfile returns the member profile payload, including the member's
// current VIP tier (and its banner URL) when a tier resolver is wired.
func (s *Service) MemberProfile(ctx context.Context, id int64) (MemberProfile, error) {
	member, err := s.members.GetByID(ctx, id)
	if err != nil {
		return MemberProfile{}, err
	}
	profile := memberProfile(member)
	if s.tiers != nil {
		tier, err := s.tiers.CurrentTier(ctx, id)
		if err != nil {
			return MemberProfile{}, err
		}
		profile.VipTier = tier
	}
	return profile, nil
}

// AccountProfile returns the account profile payload.
func (s *Service) AccountProfile(ctx context.Context, id int64) (AccountProfile, error) {
	account, err := s.accounts.GetByID(ctx, id)
	if err != nil {
		return AccountProfile{}, err
	}
	return accountProfile(account), nil
}

// LogoutMember invalidates all of a member's tokens by bumping token_version.
func (s *Service) LogoutMember(ctx context.Context, id int64) error {
	return s.members.BumpTokenVersion(ctx, id)
}

// LogoutAccount invalidates all of an account's tokens.
func (s *Service) LogoutAccount(ctx context.Context, id int64) error {
	return s.accounts.BumpTokenVersion(ctx, id)
}

func (s *Service) authenticate(ctx context.Context, username, password string) (Account, error) {
	account, err := s.accounts.GetByUsername(ctx, strings.TrimSpace(username))
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		return Account{}, apperr.Unauthenticated("invalid credentials")
	}
	if err != nil {
		return Account{}, err
	}
	if account.Status != StatusActive {
		return Account{}, apperr.Forbidden("account is disabled")
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return Account{}, apperr.Unauthenticated("invalid credentials")
	}
	return account, nil
}

func (s *Service) issueAccountToken(account Account, audience authn.Audience) (authn.TokenPair, error) {
	pair, err := s.tokens.Issue(authn.Identity{
		SubjectID:    account.ID,
		SubjectType:  subjectForRole(account.Role),
		Role:         authn.Role(account.Role),
		Audience:     audience,
		StoreID:      account.StoreID,
		TokenVersion: account.TokenVersion,
	})
	if err != nil {
		return authn.TokenPair{}, apperr.Internal(err)
	}
	return pair, nil
}

func subjectForRole(role string) authn.SubjectType {
	switch authn.Role(role) {
	case authn.RoleSuperAdmin:
		return authn.SubjectSuperAdmin
	case authn.RoleStoreAdmin:
		return authn.SubjectStoreAdmin
	case authn.RoleCashier:
		return authn.SubjectCashier
	case authn.RoleStaff:
		return authn.SubjectStaff
	default:
		return authn.SubjectMember
	}
}

func memberProfile(m Member) MemberProfile {
	return MemberProfile{ID: m.ID, Nickname: m.Nickname, MemberNo: formatMemberNo(m.ID), Phone: m.Phone, InviteCode: m.InviteCode, Status: m.Status}
}

// formatMemberNo derives a stable, display-only membership card number from the
// member id (e.g. id 2 -> "6272 0000 002"). It is deterministic and backend
// owned; the mini home card renders it as "NO.<memberNo>".
func formatMemberNo(id int64) string {
	n := fmt.Sprintf("6272%07d", id) // "6272" prefix + 7-digit zero-padded id
	return n[0:4] + " " + n[4:8] + " " + n[8:11]
}

func accountProfile(a Account) AccountProfile {
	return AccountProfile{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Role: a.Role, StoreID: a.StoreID, Status: a.Status}
}
