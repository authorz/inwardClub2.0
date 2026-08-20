package auth

import (
	"context"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

type memMemberRepo struct {
	seq                     int64
	byID                    map[int64]Member
	byOpen                  map[string]int64
	completeRegistrationErr error
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
func (r *memMemberRepo) FindIDByInviteCode(_ context.Context, inviteCode string) (int64, bool, error) {
	for id, member := range r.byID {
		if member.InviteCode == inviteCode {
			return id, true, nil
		}
	}
	return 0, false, nil
}
func (r *memMemberRepo) Create(_ context.Context, m Member) (int64, error) {
	r.seq++
	m.ID = r.seq
	m.Status = StatusActive
	r.byID[m.ID] = m
	r.byOpen[m.WeChatOpenID] = m.ID
	return m.ID, nil
}
func (r *memMemberRepo) CompleteRegistration(_ context.Context, id int64, update Member) error {
	if r.completeRegistrationErr != nil {
		return r.completeRegistrationErr
	}
	m, ok := r.byID[id]
	if !ok {
		return apperr.NotFound("member not found")
	}
	if m.ProfileCompleted {
		return apperr.Conflict("会员资料已经完善")
	}
	m.Nickname = update.Nickname
	m.AvatarURL = update.AvatarURL
	m.Gender = update.Gender
	m.Phone = update.Phone
	m.InviteCode = update.InviteCode
	m.InvitedByMemberID = update.InvitedByMemberID
	m.ProfileCompleted = true
	m.TokenVersion++
	r.byID[id] = m
	return nil
}

func TestCompletePreRegisteredMemberRejectsDuplicatePhone(t *testing.T) {
	repo := newMemMemberRepo()
	repo.completeRegistrationErr = apperr.Internal(&mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '13800000000' for key 'members.uq_members_phone'",
	})
	svc := &Service{members: repo}

	err := svc.completePreRegisteredMember(context.Background(), 1, "https://example.com/a.jpg", "会员", "other", "13800000000", nil)
	if code := apperr.From(err).Code; code != apperr.CodeConflict {
		t.Fatalf("duplicate phone: got %s, want %s", code, apperr.CodeConflict)
	}
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

type memStaffRepo struct {
	byMember map[int64]Staff
}

func (r *memStaffRepo) GetByMemberID(_ context.Context, memberID int64) (Staff, error) {
	staff, ok := r.byMember[memberID]
	if !ok {
		return Staff{}, apperr.NotFound("staff account not found")
	}
	return staff, nil
}

func (r *memStaffRepo) BumpTokenVersionByMemberID(_ context.Context, memberID int64) error {
	staff, ok := r.byMember[memberID]
	if !ok {
		return apperr.NotFound("staff account not found")
	}
	staff.TokenVersion++
	r.byMember[memberID] = staff
	return nil
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
func (r *memAccountRepo) ListActiveStoreAdminsByStoreID(_ context.Context, storeID int64) ([]Account, error) {
	accounts := make([]Account, 0)
	for _, account := range r.byID {
		if account.StoreID == storeID && account.Role == string(authn.RoleStoreAdmin) && account.Status == StatusActive {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
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
	return NewService(mgr, NewFakeWeChatClient(""), members, accounts, nil, nil), members, accounts
}

func TestVerifyAccountPassword(t *testing.T) {
	svc, _, accounts := newTestService()
	ctx := context.Background()

	if err := svc.VerifyAccountPassword(ctx, 1, "secret"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}
	if err := svc.VerifyAccountPassword(ctx, 1, "wrong"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected forbidden for wrong password, got %v", err)
	}
	if err := svc.VerifyAccountPassword(ctx, 1, " "); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty password, got %v", err)
	}

	account := accounts.byID[1]
	account.Status = StatusDisabled
	accounts.byID[1] = account
	if err := svc.VerifyAccountPassword(ctx, 1, "secret"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected forbidden for disabled account, got %v", err)
	}
}

func TestVerifyStoreAdminPasswordUsesActiveManagerFromSameStore(t *testing.T) {
	svc, _, accounts := newTestService()
	ctx := context.Background()
	cashierHash, _ := bcrypt.GenerateFromPassword([]byte("cashier-secret"), bcrypt.MinCost)
	accounts.byID[4] = Account{
		ID: 4, Username: "cashier", PasswordHash: string(cashierHash),
		Role: string(authn.RoleCashier), StoreID: 7, Status: StatusActive,
	}

	if err := svc.VerifyStoreAdminPassword(ctx, 7, "secret"); err != nil {
		t.Fatalf("expected same-store manager password to pass, got %v", err)
	}
	if err := svc.VerifyStoreAdminPassword(ctx, 8, "secret"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("manager from another store must not pass, got %v", err)
	}
	if err := svc.VerifyStoreAdminPassword(ctx, 7, "wrong"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("wrong manager password must be forbidden, got %v", err)
	}
	if err := svc.VerifyStoreAdminPassword(ctx, 7, "cashier-secret"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("cashier password must not pass, got %v", err)
	}
	if err := svc.VerifyStoreAdminPassword(ctx, 7, " "); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("empty manager password must be invalid, got %v", err)
	}

	manager := accounts.byID[2]
	manager.Status = StatusDisabled
	accounts.byID[2] = manager
	if err := svc.VerifyStoreAdminPassword(ctx, 7, "secret"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("disabled manager password must not pass, got %v", err)
	}
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
	svc := NewService(mgr, NewFakeWeChatClient(""), members, &memAccountRepo{}, stubTierResolver{tier: tier}, nil)

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
	svc := NewService(mgr, NewFakeWeChatClient(""), members, &memAccountRepo{}, stubTierResolver{}, nil)

	profile, err := svc.MemberProfile(context.Background(), id)
	if err != nil {
		t.Fatalf("member profile: %v", err)
	}
	if profile.VipTier != nil {
		t.Fatalf("expected nil vipTier, got %+v", profile.VipTier)
	}
}

func TestMemberProfileIncludesInviterBindingState(t *testing.T) {
	inviterID := int64(9)
	profile := memberProfile(Member{ID: 1, InvitedByMemberID: &inviterID})
	if !profile.InviterBound {
		t.Fatal("expected inviterBound for a member with an inviter")
	}

	profile = memberProfile(Member{ID: 2})
	if profile.InviterBound {
		t.Fatal("expected inviterBound false for a member without an inviter")
	}
}

func TestMemberProfileIncludesGender(t *testing.T) {
	profile := memberProfile(Member{ID: 1, Gender: "female"})
	if profile.Gender != "female" {
		t.Fatalf("expected gender in member profile, got %q", profile.Gender)
	}
}

func TestMiniLoginDefersCreationUntilRegister(t *testing.T) {
	svc, members, _ := newTestService()
	ctx := context.Background()

	// First login for a new user creates NO member — it returns a register ticket.
	first, err := svc.MiniLogin(ctx, "code-abc")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	if !first.IsNew || first.RegisterTicket == "" {
		t.Fatalf("first login must be new with a register ticket, got %+v", first)
	}
	if len(members.byID) != 0 {
		t.Fatalf("no member may exist before registration, got %d", len(members.byID))
	}

	// Authorizing the phone decrypts the one-time code once and re-issues the
	// ticket with the phone embedded.
	_, phoneTicket, err := svc.GetPhoneMask(ctx, first.RegisterTicket, "pc")
	if err != nil {
		t.Fatalf("phone mask: %v", err)
	}

	// Submitting the form is what creates the member.
	reg, err := svc.MiniRegister(ctx, WeChatRegisterRequest{RegisterTicket: phoneTicket, AvatarURL: "https://example.com/avatar.jpg", Nickname: "老大", Gender: "male"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if reg.Token.AccessToken == "" || !reg.IsNew {
		t.Fatalf("register must return a new-member session, got %+v", reg)
	}
	if len(members.byID) != 1 {
		t.Fatalf("expected 1 member after registration, got %d", len(members.byID))
	}

	// A subsequent login for the same openID reuses the member and is not new.
	second, err := svc.MiniLogin(ctx, "code-abc")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if second.IsNew {
		t.Fatal("returning member must not be flagged new")
	}
	if len(members.byID) != 1 {
		t.Fatalf("same openID must reuse member, got %d", len(members.byID))
	}
}

func TestPreRegisterReservesIdentityButStillRequiresProfileCompletion(t *testing.T) {
	svc, members, _ := newTestService()
	ctx := context.Background()

	pre, err := svc.MiniPreRegister(ctx, "silent-code")
	if err != nil {
		t.Fatalf("pre-register: %v", err)
	}
	if pre.SubjectType != authn.SubjectPreMember || pre.Token.AccessToken == "" {
		t.Fatalf("unexpected pre-register response: %+v", pre)
	}
	claims, err := svc.tokens.Parse(pre.Token.AccessToken, authn.AudienceMini)
	if err != nil || claims.SubjectType != authn.SubjectPreMember {
		t.Fatalf("unexpected pre-register claims: %+v, %v", claims, err)
	}
	if len(members.byID) != 1 {
		t.Fatalf("expected one persisted OpenID identity, got %d", len(members.byID))
	}
	memberID := members.byOpen[defaultFakeOpenID]
	pending := members.byID[memberID]
	if pending.ProfileCompleted || pending.WeChatOpenID == "" || pending.Nickname != "inward会员" {
		t.Fatalf("unexpected pending member: %+v", pending)
	}

	login, err := svc.MiniLogin(ctx, "explicit-login-code")
	if err != nil {
		t.Fatalf("login after pre-register: %v", err)
	}
	if !login.IsNew || login.RegisterTicket == "" || login.Token.AccessToken != "" {
		t.Fatalf("pending member must still complete profile: %+v", login)
	}
	_, phoneTicket, err := svc.GetPhoneMask(ctx, login.RegisterTicket, "phone-code")
	if err != nil {
		t.Fatalf("phone mask: %v", err)
	}
	registered, err := svc.MiniRegister(ctx, WeChatRegisterRequest{
		RegisterTicket: phoneTicket,
		AvatarURL:      "https://example.com/avatar.jpg",
		Nickname:       "正式会员",
		Gender:         "female",
	})
	if err != nil {
		t.Fatalf("complete registration: %v", err)
	}
	if registered.Token.AccessToken == "" || len(members.byID) != 1 {
		t.Fatalf("registration must upgrade the same member: %+v", registered)
	}
	completed := members.byID[memberID]
	if !completed.ProfileCompleted || completed.Phone == "" || completed.Nickname != "正式会员" {
		t.Fatalf("profile was not completed: %+v", completed)
	}
	if _, err := svc.Refresh(ctx, pre.Token.RefreshToken, authn.AudienceMini); err == nil {
		t.Fatal("pre-registration refresh token must be invalid after profile completion")
	}

	recovered, err := svc.MiniPreRegister(ctx, "silent-login-for-existing-member")
	if err != nil {
		t.Fatalf("silent login for completed member: %v", err)
	}
	if recovered.SubjectType != authn.SubjectMember || recovered.Profile == nil || recovered.IsNew {
		t.Fatalf("completed member must recover a full login session: %+v", recovered)
	}
}

func TestMiniRegisterBindsInviterFromInviteCode(t *testing.T) {
	svc, members, _ := newTestService()
	ctx := context.Background()
	inviterID, err := members.Create(ctx, Member{
		WeChatOpenID: "inviter-openid",
		Nickname:     "邀请人",
		InviteCode:   "123456",
	})
	if err != nil {
		t.Fatalf("create inviter: %v", err)
	}

	first, err := svc.MiniLogin(ctx, "invitee-code")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	_, phoneTicket, err := svc.GetPhoneMask(ctx, first.RegisterTicket, "phone-code")
	if err != nil {
		t.Fatalf("phone mask: %v", err)
	}
	if _, err := svc.MiniRegister(ctx, WeChatRegisterRequest{
		RegisterTicket: phoneTicket,
		AvatarURL:      "https://example.com/invitee.jpg",
		Nickname:       "被邀请人",
		Gender:         "female",
		InviterCode:    "123456",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	inviteeID := members.byOpen[defaultFakeOpenID]
	invitee := members.byID[inviteeID]
	if invitee.InvitedByMemberID == nil || *invitee.InvitedByMemberID != inviterID {
		t.Fatalf("expected inviter %d, got %+v", inviterID, invitee.InvitedByMemberID)
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

func TestMiniLoginUsesActiveStaffBinding(t *testing.T) {
	svc, members, _ := newTestService()
	ctx := context.Background()

	first, err := svc.MiniLogin(ctx, "staff-code")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	_, phoneTicket, err := svc.GetPhoneMask(ctx, first.RegisterTicket, "phone-code")
	if err != nil {
		t.Fatalf("phone mask: %v", err)
	}
	if _, err := svc.MiniRegister(ctx, WeChatRegisterRequest{
		RegisterTicket: phoneTicket,
		AvatarURL:      "https://example.com/avatar.jpg",
		Nickname:       "员工",
		Gender:         "other",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	var memberID int64
	for id := range members.byID {
		memberID = id
	}
	staff := &memStaffRepo{byMember: map[int64]Staff{
		memberID: {
			ID: 9, MemberID: memberID, StoreID: 42,
			Status: StatusActive, TokenVersion: 3,
		},
	}}
	svc.staff = staff

	login, err := svc.MiniLogin(ctx, "staff-code")
	if err != nil {
		t.Fatalf("staff login: %v", err)
	}
	if login.SubjectType != authn.SubjectStaff || login.StoreID != 42 {
		t.Fatalf("unexpected staff login response: %+v", login)
	}
	claims, err := svc.tokens.Parse(login.Token.AccessToken, authn.AudienceMini)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.SubjectType != authn.SubjectStaff || claims.SubjectID() != memberID ||
		claims.StoreID != 42 || claims.TokenVersion != 3 {
		t.Fatalf("unexpected staff claims: %+v", claims)
	}

	if _, err := svc.Refresh(ctx, login.Token.RefreshToken, authn.AudienceMini); err != nil {
		t.Fatalf("staff refresh: %v", err)
	}
	checker := NewMemberTokenVersions(members, staff)
	if version, err := checker.CurrentTokenVersion(ctx, authn.SubjectStaff, memberID); err != nil || version != 3 {
		t.Fatalf("staff token version = %d, %v; want 3, nil", version, err)
	}
	if err := svc.LogoutMini(ctx, authn.SubjectStaff, memberID); err != nil {
		t.Fatalf("staff logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, login.Token.RefreshToken, authn.AudienceMini); err == nil {
		t.Fatal("staff refresh must fail after logout")
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
	if v, err := accountVersions.CurrentTokenVersion(ctx, authn.SubjectSuperAdmin, 1); err != nil || v != 0 {
		t.Fatalf("account version before logout = %d, %v; want 0, nil", v, err)
	}
	if err := svc.LogoutAccount(ctx, 1); err != nil {
		t.Fatalf("account logout: %v", err)
	}
	if v, err := accountVersions.CurrentTokenVersion(ctx, authn.SubjectSuperAdmin, 1); err != nil || v != 1 {
		t.Fatalf("account version after logout = %d, %v; want 1, nil", v, err)
	}

	// Member (mini audience). Create one via login + register, then log out.
	first, err := svc.MiniLogin(ctx, "code-xyz")
	if err != nil {
		t.Fatalf("mini login: %v", err)
	}
	_, phoneTicket, err := svc.GetPhoneMask(ctx, first.RegisterTicket, "pc")
	if err != nil {
		t.Fatalf("phone mask: %v", err)
	}
	if _, err := svc.MiniRegister(ctx, WeChatRegisterRequest{RegisterTicket: phoneTicket, AvatarURL: "https://example.com/avatar.jpg", Nickname: "会员", Gender: "female"}); err != nil {
		t.Fatalf("mini register: %v", err)
	}
	var memberID int64
	for id := range members.byID {
		memberID = id
	}
	memberVersions := NewMemberTokenVersions(members)
	if v, err := memberVersions.CurrentTokenVersion(ctx, authn.SubjectMember, memberID); err != nil || v != 0 {
		t.Fatalf("member version before logout = %d, %v; want 0, nil", v, err)
	}
	if err := svc.LogoutMini(ctx, authn.SubjectMember, memberID); err != nil {
		t.Fatalf("member logout: %v", err)
	}
	if v, err := memberVersions.CurrentTokenVersion(ctx, authn.SubjectMember, memberID); err != nil || v != 1 {
		t.Fatalf("member version after logout = %d, %v; want 1, nil", v, err)
	}

	// A missing subject surfaces the store's NotFound (mapped to 401 by the
	// middleware), never a silent zero that would masquerade as a valid version.
	if _, err := memberVersions.CurrentTokenVersion(ctx, authn.SubjectMember, 9999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("missing member: want NOT_FOUND, got %v", err)
	}
}
