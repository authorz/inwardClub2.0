package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

type memMemberRepo struct {
	seq    int64
	byID   map[int64]Member
	byOpen map[string]int64
}

func newMemMemberRepo() *memMemberRepo {
	return &memMemberRepo{byID: map[int64]Member{}, byOpen: map[string]int64{}}
}

func (r *memMemberRepo) GetByOpenID(_ context.Context, openID string) (Member, error) {
	id, ok := r.byOpen[openID]
	if !ok {
		return Member{}, apperr.NotFound("member not found")
	}
	return r.byID[id], nil
}
func (r *memMemberRepo) GetByID(_ context.Context, id int64) (Member, error) {
	m, ok := r.byID[id]
	if !ok {
		return Member{}, apperr.NotFound("member not found")
	}
	return m, nil
}
func (r *memMemberRepo) Create(_ context.Context, m Member) (int64, error) {
	r.seq++
	m.ID = r.seq
	m.Status = StatusActive
	r.byID[m.ID] = m
	r.byOpen[m.WeChatOpenID] = m.ID
	return m.ID, nil
}
func (r *memMemberRepo) BumpTokenVersion(_ context.Context, id int64) error {
	m := r.byID[id]
	m.TokenVersion++
	r.byID[id] = m
	return nil
}

type memAccountRepo struct {
	byID   map[int64]Account
	byName map[string]int64
}

func (r *memAccountRepo) GetByUsername(_ context.Context, u string) (Account, error) {
	id, ok := r.byName[u]
	if !ok {
		return Account{}, apperr.NotFound("account not found")
	}
	return r.byID[id], nil
}
func (r *memAccountRepo) GetByID(_ context.Context, id int64) (Account, error) {
	a, ok := r.byID[id]
	if !ok {
		return Account{}, apperr.NotFound("account not found")
	}
	return a, nil
}
func (r *memAccountRepo) BumpTokenVersion(_ context.Context, id int64) error {
	a := r.byID[id]
	a.TokenVersion++
	r.byID[id] = a
	return nil
}

func newTestService() (*Service, *memMemberRepo, *memAccountRepo) {
	mgr := authn.NewManager("k", "inwardclub", time.Hour, 24*time.Hour)
	members := newMemMemberRepo()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	accounts := &memAccountRepo{
		byID: map[int64]Account{
			1: {ID: 1, Username: "boss", PasswordHash: string(hash), Role: string(authn.RoleSuperAdmin), Status: StatusActive},
			2: {ID: 2, Username: "shop", PasswordHash: string(hash), Role: string(authn.RoleStoreAdmin), StoreID: 7, Status: StatusActive},
			3: {ID: 3, Username: "noStore", PasswordHash: string(hash), Role: string(authn.RoleStoreAdmin), Status: StatusActive},
		},
		byName: map[string]int64{"boss": 1, "shop": 2, "noStore": 3},
	}
	return NewService(mgr, NewFakeWeChatClient(), members, accounts, nil), members, accounts
}

// stubTierResolver is a configurable MemberTierResolver for exercising the "me"
// profile tier enrichment without the member module.
type stubTierResolver struct {
	tier *MemberVIPTier
	err  error
}

func (s stubTierResolver) CurrentTier(context.Context, int64) (*MemberVIPTier, error) {
	return s.tier, s.err
}

func TestMemberProfileIncludesVipTier(t *testing.T) {
	mgr := authn.NewManager("k", "inwardclub", time.Hour, 24*time.Hour)
	members := newMemMemberRepo()
	id, _ := members.Create(context.Background(), Member{WeChatOpenID: "open-1", Nickname: "会员"})

	// The member-facing tier exposes only the short VIP label, never the full
	// admin tier name (e.g. "VIP3 黄金会员").
	tier := &MemberVIPTier{ID: 3, Label: "VIP3", Level: 3, Threshold: 5000, BannerURL: "https://cdn.test/banner"}
	svc := NewService(mgr, NewFakeWeChatClient(), members, &memAccountRepo{}, stubTierResolver{tier: tier})

	profile, err := svc.MemberProfile(context.Background(), id)
	if err != nil {
		t.Fatalf("member profile: %v", err)
	}
	if profile.VipTier == nil {
		t.Fatal("expected vipTier populated")
	}
	if profile.VipTier.Label != "VIP3" || profile.VipTier.BannerURL != "https://cdn.test/banner" {
		t.Fatalf("unexpected vipTier: %+v", profile.VipTier)
	}
}

func TestMemberProfileOmitsVipTierWhenUnranked(t *testing.T) {
	mgr := authn.NewManager("k", "inwardclub", time.Hour, 24*time.Hour)
	members := newMemMemberRepo()
	id, _ := members.Create(context.Background(), Member{WeChatOpenID: "open-2", Nickname: "会员"})

	// Resolver reports no tier (nil, nil); the profile must omit vipTier.
	svc := NewService(mgr, NewFakeWeChatClient(), members, &memAccountRepo{}, stubTierResolver{})

	profile, err := svc.MemberProfile(context.Background(), id)
	if err != nil {
		t.Fatalf("member profile: %v", err)
	}
	if profile.VipTier != nil {
		t.Fatalf("expected nil vipTier, got %+v", profile.VipTier)
	}
}

func TestMiniLoginCreatesThenReuses(t *testing.T) {
	svc, members, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.MiniLogin(ctx, "code-abc"); err != nil {
		t.Fatalf("first login: %v", err)
	}
	if len(members.byID) != 1 {
		t.Fatalf("expected 1 member created, got %d", len(members.byID))
	}
	if _, err := svc.MiniLogin(ctx, "code-abc"); err != nil {
		t.Fatalf("second login: %v", err)
	}
	if len(members.byID) != 1 {
		t.Fatalf("same code must reuse member, got %d", len(members.byID))
	}
}

func TestAdminLoginRoleAndPassword(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.AdminLogin(ctx, "boss", "secret"); err != nil {
		t.Fatalf("super_admin login should succeed: %v", err)
	}
	if _, err := svc.AdminLogin(ctx, "boss", "wrong"); err == nil {
		t.Fatal("wrong password must fail")
	}
	// store_admin cannot log into the admin console.
	if _, err := svc.AdminLogin(ctx, "shop", "secret"); err == nil {
		t.Fatal("store_admin must not log into admin console")
	}
}

func TestStoreLoginRequiresStoreBinding(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	resp, err := svc.StoreLogin(ctx, "shop", "secret")
	if err != nil {
		t.Fatalf("store login should succeed: %v", err)
	}
	// The issued token must carry the account's store scope.
	claims, err := svc.tokens.Parse(resp.Token.AccessToken, authn.AudienceStore)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.StoreID != 7 {
		t.Fatalf("expected store 7, got %d", claims.StoreID)
	}
	// An account without a store binding cannot log into the store console.
	if _, err := svc.StoreLogin(ctx, "noStore", "secret"); err == nil {
		t.Fatal("store login without binding must fail")
	}
}

func TestRefreshRejectedAfterLogout(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	resp, err := svc.AdminLogin(ctx, "boss", "secret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	// Logout bumps token_version, invalidating the refresh token.
	if err := svc.LogoutAccount(ctx, 1); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, resp.Token.RefreshToken, authn.AudienceAdmin); err == nil {
		t.Fatal("refresh must fail after logout")
	}
}

// TestTokenVersionCheckersReflectLogout proves the piece the access-token
// middleware relies on for checklist §4.4: after logout the checker reports a
// token_version that no longer matches a token minted before it, so the
// middleware's comparison (current != claims.TokenVersion) rejects the request.
func TestTokenVersionCheckersReflectLogout(t *testing.T) {
	svc, members, accounts := newTestService()
	ctx := context.Background()

	// Account (admin/store audiences). Account #1 starts at version 0.
	accountVersions := NewAccountTokenVersions(accounts)
	if v, err := accountVersions.CurrentTokenVersion(ctx, 1); err != nil || v != 0 {
		t.Fatalf("account version before logout = %d, %v; want 0, nil", v, err)
	}
	if err := svc.LogoutAccount(ctx, 1); err != nil {
		t.Fatalf("account logout: %v", err)
	}
	if v, err := accountVersions.CurrentTokenVersion(ctx, 1); err != nil || v != 1 {
		t.Fatalf("account version after logout = %d, %v; want 1, nil", v, err)
	}

	// Member (mini audience). Create one via MiniLogin, then log out.
	if _, err := svc.MiniLogin(ctx, "code-xyz"); err != nil {
		t.Fatalf("mini login: %v", err)
	}
	var memberID int64
	for id := range members.byID {
		memberID = id
	}
	memberVersions := NewMemberTokenVersions(members)
	if v, err := memberVersions.CurrentTokenVersion(ctx, memberID); err != nil || v != 0 {
		t.Fatalf("member version before logout = %d, %v; want 0, nil", v, err)
	}
	if err := svc.LogoutMember(ctx, memberID); err != nil {
		t.Fatalf("member logout: %v", err)
	}
	if v, err := memberVersions.CurrentTokenVersion(ctx, memberID); err != nil || v != 1 {
		t.Fatalf("member version after logout = %d, %v; want 1, nil", v, err)
	}

	// A missing subject surfaces the store's NotFound (mapped to 401 by the
	// middleware), never a silent zero that would masquerade as a valid version.
	if _, err := memberVersions.CurrentTokenVersion(ctx, 9999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("missing member: want NOT_FOUND, got %v", err)
	}
}
