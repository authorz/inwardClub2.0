package member

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeRepo is an in-memory Repository for exercising the service mapping and
// invitation rules without a database.
type fakeRepo struct {
	members           map[int64]*Member
	byCode            map[string]int64
	invitees          map[int64][]Invitee
	tiers             []MembershipTier
	products          []RechargeProduct
	rankings          []RankingEntry
	notReady          bool // when true, catalogue reads return NOT_IMPLEMENTED
	lastPhone         string
	lastPhoneInterval int
	lastUpdate        ProfileUpdate
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{members: map[int64]*Member{}, byCode: map[string]int64{}, invitees: map[int64][]Invitee{}}
}

func (r *fakeRepo) GetMember(_ context.Context, id int64) (Member, error) {
	m, ok := r.members[id]
	if !ok {
		return Member{}, apperr.New(apperr.CodeMemberNotFound, "member not found")
	}
	return *m, nil
}

func (r *fakeRepo) UpdateProfile(_ context.Context, id int64, p ProfileUpdate) error {
	r.lastUpdate = p
	m := r.members[id]
	if p.Nickname != nil {
		m.Nickname = *p.Nickname
	}
	if p.Gender != nil {
		m.Gender = *p.Gender
	}
	if p.AvatarAssetID != nil {
		m.AvatarAssetID = p.AvatarAssetID
	}
	if p.AvatarURL != nil {
		m.AvatarURL = *p.AvatarURL
	}
	return nil
}

func (r *fakeRepo) UpdatePhone(_ context.Context, id int64, phone string, intervalDays int) (PhoneChangeResult, error) {
	r.lastPhone = phone
	r.lastPhoneInterval = intervalDays
	changed := r.members[id].Phone != phone
	r.members[id].Phone = phone
	return PhoneChangeResult{
		Changed:       changed,
		NextAllowedAt: time.Now().UTC().AddDate(0, 0, intervalDays),
	}, nil
}

func (r *fakeRepo) GetByInviteCode(_ context.Context, code string) (Member, error) {
	id, ok := r.byCode[code]
	if !ok {
		return Member{}, ErrInviteCodeNotFound
	}
	return *r.members[id], nil
}

func (r *fakeRepo) ListInvitees(_ context.Context, inviterID int64, limit, offset int) ([]Invitee, int64, error) {
	all := r.invitees[inviterID]
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *fakeRepo) BindInviter(_ context.Context, inviteeID, inviterID int64) error {
	m := r.members[inviteeID]
	if m.InvitedByMemberID != nil {
		return ErrAlreadyInvited
	}
	m.InvitedByMemberID = &inviterID
	return nil
}

func (r *fakeRepo) ListMembershipTiers(_ context.Context) ([]MembershipTier, error) {
	if r.notReady {
		return nil, apperr.NotImplemented("membership tiers not available")
	}
	return r.tiers, nil
}

func (r *fakeRepo) ListRechargeProducts(_ context.Context) ([]RechargeProduct, error) {
	if r.notReady {
		return nil, apperr.NotImplemented("recharge products not available")
	}
	return r.products, nil
}

func (r *fakeRepo) ListRankings(_ context.Context, _ string, _ int) ([]RankingEntry, error) {
	if r.notReady {
		return nil, apperr.NotImplemented("rankings not available")
	}
	return r.rankings, nil
}

func (r *fakeRepo) ListAllMembershipTiers(_ context.Context) ([]MembershipTier, error) {
	return r.tiers, nil
}

func (r *fakeRepo) GetMembershipTier(_ context.Context, id int64) (MembershipTier, error) {
	for _, t := range r.tiers {
		if t.ID == id {
			return t, nil
		}
	}
	return MembershipTier{}, ErrMembershipTierNotFound
}

func (r *fakeRepo) CreateMembershipTier(_ context.Context, t MembershipTierCreate) (MembershipTier, error) {
	status := t.Status
	if status == "" {
		status = StatusActive
	}
	tier := MembershipTier{
		ID: int64(len(r.tiers) + 1), Name: t.Name, Level: t.Level,
		Threshold: t.Threshold, Benefits: t.Benefits, BenefitConfig: t.BenefitConfig,
		IconAssetID: t.IconAssetID, Status: status,
	}
	r.tiers = append(r.tiers, tier)
	return tier, nil
}

func (r *fakeRepo) UpdateMembershipTier(_ context.Context, id int64, u MembershipTierUpdate) (MembershipTier, error) {
	for i := range r.tiers {
		if r.tiers[i].ID != id {
			continue
		}
		if u.Name != nil {
			r.tiers[i].Name = *u.Name
		}
		if u.Level != nil {
			r.tiers[i].Level = *u.Level
		}
		if u.Threshold != nil {
			r.tiers[i].Threshold = *u.Threshold
		}
		if u.Benefits != nil {
			r.tiers[i].Benefits = *u.Benefits
		}
		if u.BenefitConfig != nil {
			r.tiers[i].BenefitConfig = *u.BenefitConfig
		}
		if u.IconAssetID != nil {
			r.tiers[i].IconAssetID = u.IconAssetID
		}
		if u.Status != nil {
			r.tiers[i].Status = *u.Status
		}
		return r.tiers[i], nil
	}
	return MembershipTier{}, ErrMembershipTierNotFound
}

func (r *fakeRepo) ListAllRechargeProducts(_ context.Context) ([]RechargeProduct, error) {
	return r.products, nil
}

func (r *fakeRepo) GetRechargeProduct(_ context.Context, id int64) (RechargeProduct, error) {
	for _, p := range r.products {
		if p.ID == id {
			return p, nil
		}
	}
	return RechargeProduct{}, ErrRechargeProductNotFound
}

func (r *fakeRepo) CreateRechargeProduct(_ context.Context, p RechargeProductCreate) (RechargeProduct, error) {
	status := p.Status
	if status == "" {
		status = StatusActive
	}
	product := RechargeProduct{
		ID:               int64(len(r.products) + 1),
		AmountCent:       p.AmountCent,
		CoinAmount:       p.CoinAmount,
		PointsAmount:     p.PointsAmount,
		CouponTemplateID: p.CouponTemplateID,
		SortOrder:        p.SortOrder,
		Status:           status,
	}
	r.products = append(r.products, product)
	return product, nil
}

func (r *fakeRepo) UpdateRechargeProduct(_ context.Context, id int64, u RechargeProductUpdate) (RechargeProduct, error) {
	for i := range r.products {
		if r.products[i].ID != id {
			continue
		}
		if u.AmountCent != nil {
			r.products[i].AmountCent = *u.AmountCent
		}
		if u.CoinAmount != nil {
			r.products[i].CoinAmount = *u.CoinAmount
		}
		if u.PointsAmount != nil {
			r.products[i].PointsAmount = *u.PointsAmount
		}
		if u.CouponTemplateID != nil {
			r.products[i].CouponTemplateID = *u.CouponTemplateID
		}
		if u.SortOrder != nil {
			r.products[i].SortOrder = *u.SortOrder
		}
		if u.Status != nil {
			r.products[i].Status = *u.Status
		}
		return r.products[i], nil
	}
	return RechargeProduct{}, ErrRechargeProductNotFound
}

func (r *fakeRepo) ValidateRechargeCouponTemplate(_ context.Context, id int64) error {
	if id == 99 {
		return apperr.Invalid("充值奖励只能绑定已发布的优惠券")
	}
	return nil
}

type fakeAssets struct{}

func (fakeAssets) PublicURLByID(_ context.Context, id int64) (string, error) {
	return fmt.Sprintf("https://cdn.test/asset/%d", id), nil
}

func (fakeAssets) PublicURL(objectKey string) string {
	if objectKey == "" {
		return ""
	}
	return "https://cdn.test/" + objectKey
}

type fakeAvatarAssets struct {
	fakeAssets
	uploaded    string
	contentType string
}

func (a *fakeAvatarAssets) UploadAvatar(_ context.Context, r io.Reader, _ int64, contentType string) (string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	a.uploaded = string(body)
	a.contentType = contentType
	return "https://cdn.test/avatar/member.webp", nil
}

type fakePhone struct {
	phone string
	err   error
}

func (f fakePhone) ResolvePhone(_ context.Context, _ string) (string, error) { return f.phone, f.err }

type fakePhonePolicy struct {
	days   int
	notice string
}

func (p fakePhonePolicy) PhoneChangeIntervalDays(context.Context) (int, error) { return p.days, nil }
func (p fakePhonePolicy) RechargeNotice(context.Context) (string, error)       { return p.notice, nil }

func codeOf(err error) apperr.Code { return apperr.From(err).Code }

func TestUpdateProfileWritesAndMasksPhone(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Nickname: "old", Phone: "13800001111", Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	nick, gender := "new", "female"
	view, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{Nickname: &nick, Gender: &gender})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.Nickname != "new" {
		t.Fatalf("expected nickname updated, got %q", view.Nickname)
	}
	if view.Phone != "138****1111" {
		t.Fatalf("expected masked phone, got %q", view.Phone)
	}
	if view.Gender != "female" || repo.members[1].Gender != "female" {
		t.Fatalf("expected gender updated, got view=%q stored=%q", view.Gender, repo.members[1].Gender)
	}
}

func TestUploadAndPersistAvatarURL(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Nickname: "member", Status: StatusActive}
	assets := &fakeAvatarAssets{}
	svc := NewService(repo, assets, nil)

	avatarURL, err := svc.UploadAvatar(context.Background(), strings.NewReader("avatar-bytes"), 12, "image/webp")
	if err != nil {
		t.Fatalf("upload avatar: %v", err)
	}
	if assets.uploaded != "avatar-bytes" || assets.contentType != "image/webp" {
		t.Fatalf("unexpected upload: body=%q contentType=%q", assets.uploaded, assets.contentType)
	}
	view, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{AvatarURL: &avatarURL})
	if err != nil {
		t.Fatalf("persist avatar: %v", err)
	}
	if view.AvatarURL != avatarURL || repo.members[1].AvatarURL != avatarURL {
		t.Fatalf("expected direct avatar URL in response and repository, view=%q stored=%q", view.AvatarURL, repo.members[1].AvatarURL)
	}
}

func TestUpdateProfileRejectsInvalidGender(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Nickname: "old", Gender: "male", Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)
	gender := "unknown"
	if _, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{Gender: &gender}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid gender rejection, got %v", err)
	}
	if repo.members[1].Gender != "male" {
		t.Fatal("invalid gender reached repository")
	}
}

func TestUpdateProfileRejectsUnsafeNickname(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Nickname: "old", Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)
	nickname := `<img src=x onerror=alert(1)>`
	if _, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{Nickname: &nickname}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected unsafe nickname rejection, got %v", err)
	}
	if repo.members[1].Nickname != "old" {
		t.Fatal("unsafe nickname reached repository")
	}
}

func TestBindPhoneRequiresResolver(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	_, err := svc.BindPhone(context.Background(), 1, BindPhoneRequest{Code: "x"})
	if codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("expected NOT_IMPLEMENTED, got %v", err)
	}
}

func TestBindPhoneWritesMaskedResult(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, fakePhone{phone: "13900002222"})

	view, err := svc.BindPhone(context.Background(), 1, BindPhoneRequest{Code: "x"})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if !view.Bound || view.PhoneMasked != "139****2222" {
		t.Fatalf("unexpected binding view: %+v", view)
	}
	if repo.lastPhone != "13900002222" {
		t.Fatalf("expected raw phone persisted, got %q", repo.lastPhone)
	}
}

func TestBindPhoneUsesConfiguredInterval(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, fakePhone{phone: "13900002222"}, fakePhonePolicy{days: 45})

	view, err := svc.BindPhone(context.Background(), 1, BindPhoneRequest{Code: "x"})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if view.NextChangeAvailableAt == "" {
		t.Fatal("expected the next change time in response")
	}
	if repo.lastPhoneInterval != 45 {
		t.Fatalf("expected configured 45-day interval, got %d", repo.lastPhoneInterval)
	}
}

func TestBindPhoneDoesNotInvalidateLoginOnWeChatFailure(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, fakePhone{err: fmt.Errorf("wechat errcode 40029")})

	_, err := svc.BindPhone(context.Background(), 1, BindPhoneRequest{Code: "x"})
	if codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT rather than UNAUTHENTICATED, got %v", err)
	}
}

func TestBindInvitationRules(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, InviteCode: "CODE1"}
	repo.members[2] = &Member{ID: 2, InviteCode: "CODE2"}
	repo.byCode["CODE1"] = 1
	repo.byCode["CODE2"] = 2
	svc := NewService(repo, fakeAssets{}, nil)
	ctx := context.Background()

	// self-invite rejected
	if err := svc.BindInvitation(ctx, 2, BindInvitationRequest{InviteCode: "CODE2"}); err != ErrSelfInvite {
		t.Fatalf("expected ErrSelfInvite, got %v", err)
	}
	// unknown code
	if err := svc.BindInvitation(ctx, 2, BindInvitationRequest{InviteCode: "NOPE"}); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
	// valid bind
	if err := svc.BindInvitation(ctx, 2, BindInvitationRequest{InviteCode: "CODE1"}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	// second bind rejected
	if err := svc.BindInvitation(ctx, 2, BindInvitationRequest{InviteCode: "CODE1"}); err != ErrAlreadyInvited {
		t.Fatalf("expected ErrAlreadyInvited, got %v", err)
	}
}

func TestListInvitationsMapsView(t *testing.T) {
	repo := newFakeRepo()
	repo.invitees[1] = []Invitee{{
		MemberID:  2,
		Nickname:  "b",
		AvatarURL: "https://cdn.test/avatar.png",
	}}
	svc := NewService(repo, fakeAssets{}, nil)

	views, total, err := svc.ListInvitations(context.Background(), 1, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 invitation, got %d/%d", total, len(views))
	}
	if views[0].AvatarURL != "https://cdn.test/avatar.png" || views[0].JoinedAt == "" {
		t.Fatalf("expected avatar and joinedAt populated: %+v", views[0])
	}
}

func TestRankingPeriodValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	for _, period := range []string{RankingMonth, RankingAll, RankingWater} {
		if _, err := svc.ListRankings(context.Background(), period); err != nil {
			t.Fatalf("period %q should be valid: %v", period, err)
		}
	}
	if _, err := svc.ListRankings(context.Background(), "yearly"); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
	if _, err := svc.ListRankings(context.Background(), "week"); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("legacy week period should be invalid, got %v", err)
	}
}

func TestListRankingsMapsView(t *testing.T) {
	repo := newFakeRepo()
	avatar := int64(7)
	repo.rankings = []RankingEntry{
		{Rank: 1, MemberID: 42, Nickname: "top", AvatarAssetID: &avatar, Gender: "female", Score: 500, GrowthValue: 1200},
	}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.ListRankings(context.Background(), "") // empty period defaults to all
	if err != nil {
		t.Fatalf("rankings: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(views))
	}
	got := views[0]
	if got.Rank != 1 || got.MemberID != 42 || got.Nickname != "top" || got.Gender != "female" || got.Score != 500 || got.GrowthValue != 1200 || got.AvatarURL == "" {
		t.Fatalf("unexpected ranking view: %+v", got)
	}
}

func TestListRankingsPrefersStoredAvatarURL(t *testing.T) {
	repo := newFakeRepo()
	repo.rankings = []RankingEntry{
		{Rank: 1, MemberID: 7, Nickname: "avatar", AvatarURL: "https://cdn.test/direct.png", Score: 100},
	}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.ListRankings(context.Background(), RankingAll)
	if err != nil {
		t.Fatalf("rankings: %v", err)
	}
	if len(views) != 1 || views[0].AvatarURL != "https://cdn.test/direct.png" {
		t.Fatalf("expected direct avatar URL, got %+v", views)
	}
}

func TestCatalogueNotImplementedWhenTablesMissing(t *testing.T) {
	repo := newFakeRepo()
	repo.notReady = true
	svc := NewService(repo, fakeAssets{}, nil)
	ctx := context.Background()

	if _, err := svc.ListMembershipTiers(ctx); codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("tiers: expected NOT_IMPLEMENTED, got %v", err)
	}
	if _, err := svc.ListRechargeProducts(ctx); codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("products: expected NOT_IMPLEMENTED, got %v", err)
	}
	if _, err := svc.ListRankings(ctx, RankingWater); codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("rankings: expected NOT_IMPLEMENTED, got %v", err)
	}
}

func TestMembershipTierMapping(t *testing.T) {
	repo := newFakeRepo()
	icon := int64(3)
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Level: 2, Threshold: 1000, IconAssetID: &icon, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.ListMembershipTiers(context.Background())
	if err != nil {
		t.Fatalf("tiers: %v", err)
	}
	if len(views) != 1 || views[0].Name != "Gold" {
		t.Fatalf("unexpected tier view: %+v", views)
	}
	if views[0].IconURL != "https://cdn.test/asset/3" {
		t.Fatalf("expected icon resolved from asset 3, got %q", views[0].IconURL)
	}
}

func TestCreateMembershipTierPersistsBenefits(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CreateMembershipTier(context.Background(), MembershipTierCreateRequest{
		Name: "Platinum",
		BenefitConfig: TierBenefitConfig{
			Points:  []TierPointBenefit{{Amount: 1000, Period: "daily", Trigger: "low_spend"}},
			Coupons: []TierCouponBenefit{{CouponType: "alcohol", Quantity: 1, Period: "daily", Trigger: "visit"}},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(view.BenefitConfig.Points) != 1 || len(view.BenefitConfig.Coupons) != 1 {
		t.Fatalf("expected structured benefits, got %+v", view.BenefitConfig)
	}
}

func TestVIPShortLabel(t *testing.T) {
	cases := map[int]string{1: "VIP1", 2: "VIP2", 8: "VIP8", 0: "VIP", -1: "VIP"}
	for level, want := range cases {
		if got := VIPShortLabel(level); got != want {
			t.Errorf("VIPShortLabel(%d) = %q, want %q", level, got, want)
		}
	}
}

func TestCurrentTierView(t *testing.T) {
	repo := newFakeRepo()
	tierID := int64(1)
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Level: 2, Threshold: 1000, Status: StatusActive}}
	repo.members[5] = &Member{ID: 5, CurrentTierID: &tierID, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view == nil || view.Name != "Gold" {
		t.Fatalf("unexpected current tier view: %+v", view)
	}
}

func TestCurrentTierViewUnrankedFallsBackToBaseTier(t *testing.T) {
	repo := newFakeRepo()
	// Active tiers ordered by level ASC (as the real query returns them); the
	// first is the base tier (VIP1, threshold 0).
	repo.tiers = []MembershipTier{
		{ID: 1, Name: "VIP1", Level: 1, Threshold: 0, Status: StatusActive},
		{ID: 2, Name: "VIP2", Level: 2, Threshold: 1000, Status: StatusActive},
	}
	repo.members[5] = &Member{ID: 5, Status: StatusActive} // no CurrentTierID
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view == nil || view.Name != "VIP1" {
		t.Fatalf("expected base tier VIP1 for unranked member, got %+v", view)
	}
}

func TestCurrentTierViewUnrankedNoTiersConfigured(t *testing.T) {
	repo := newFakeRepo() // no tiers at all
	repo.members[5] = &Member{ID: 5, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view != nil {
		t.Fatalf("expected nil view when no tiers configured, got %+v", view)
	}
}

func TestCurrentTierViewDanglingReferenceFallsBackToBaseTier(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{{ID: 1, Name: "VIP1", Level: 1, Threshold: 0, Status: StatusActive}}
	missing := int64(99)
	repo.members[5] = &Member{ID: 5, CurrentTierID: &missing, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view == nil || view.Name != "VIP1" {
		t.Fatalf("expected fall back to base tier VIP1 for dangling reference, got %+v", view)
	}
}

func TestAdminListMembershipTiersIncludesDisabled(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{
		{ID: 1, Name: "Gold", Status: StatusActive},
		{ID: 2, Name: "Retired", Status: StatusDisabled},
	}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.AdminListMembershipTiers(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected disabled tiers included, got %+v", views)
	}
}

func TestAdminGetMembershipTierNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.AdminGetMembershipTier(context.Background(), 99); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestAdminGetMembershipTierReturnsDisabled(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{{ID: 1, Name: "Retired", Status: StatusDisabled}}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.AdminGetMembershipTier(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Status != StatusDisabled {
		t.Fatalf("expected disabled tier visible to admin, got %+v", view)
	}
}

func TestCreateMembershipTierRequiresName(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.CreateMembershipTier(context.Background(), MembershipTierCreateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestCreateMembershipTierDefaultsStatusActive(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CreateMembershipTier(context.Background(), MembershipTierCreateRequest{Name: "Platinum", Level: 4, Threshold: 10000})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Status != StatusActive || view.Name != "Platinum" || view.ID == 0 {
		t.Fatalf("unexpected created tier: %+v", view)
	}
}

func TestUpdateMembershipTierRequiresAField(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.UpdateMembershipTier(context.Background(), 1, MembershipTierUpdateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestUpdateMembershipTierNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	name := "Gold"
	if _, err := svc.UpdateMembershipTier(context.Background(), 99, MembershipTierUpdateRequest{Name: &name}); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestUpdateMembershipTierAppliesFields(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Level: 2, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	name := "Gold Plus"
	level := 3
	view, err := svc.UpdateMembershipTier(context.Background(), 1, MembershipTierUpdateRequest{Name: &name, Level: &level})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.Name != "Gold Plus" || view.Level != 3 {
		t.Fatalf("unexpected updated tier: %+v", view)
	}
}

func TestDisableMembershipTier(t *testing.T) {
	repo := newFakeRepo()
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.DisableMembershipTier(context.Background(), 1)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if view.Status != StatusDisabled {
		t.Fatalf("expected disabled status, got %+v", view)
	}
}

func TestAdminListRechargeProductsIncludesDisabled(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{
		{ID: 1, AmountCent: 1000, CoinAmount: 10, Status: StatusActive},
		{ID: 2, AmountCent: 2000, CoinAmount: 20, Status: StatusDisabled},
	}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.AdminListRechargeProducts(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected disabled products included, got %+v", views)
	}
}

func TestAdminGetRechargeProductNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.AdminGetRechargeProduct(context.Background(), 99); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestAdminGetRechargeProductReturnsDisabled(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, AmountCent: 1000, CoinAmount: 10, Status: StatusDisabled}}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.AdminGetRechargeProduct(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Status != StatusDisabled {
		t.Fatalf("expected disabled product visible to admin, got %+v", view)
	}
}

func TestCreateRechargeProductRequiresAmountAndCoins(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing amount, got %v", err)
	}
	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{AmountCent: 50000}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing coins, got %v", err)
	}
	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{AmountCent: 0, CoinAmount: 588}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for non-positive amount, got %v", err)
	}
}

func TestCreateRechargeProductDefaultsStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{
		AmountCent: 50000, CoinAmount: 588, PointsAmount: 10000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Status != StatusActive || view.AmountCent != 50000 || view.CoinAmount != 588 || view.PointsAmount != 10000 || view.ID == 0 {
		t.Fatalf("unexpected created product: %+v", view)
	}
}

func TestRechargeProductAmountsRoundTrip(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	created, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{
		AmountCent: 50000, CoinAmount: 588, PointsAmount: 10000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.CoinAmount != 588 || created.PointsAmount != 10000 {
		t.Fatalf("unexpected created amounts: %+v", created)
	}

	coins := int64(688)
	updated, err := svc.UpdateRechargeProduct(context.Background(), created.ID, RechargeProductUpdateRequest{CoinAmount: &coins})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.CoinAmount != 688 || updated.PointsAmount != 10000 {
		t.Fatalf("unexpected updated amounts: %+v", updated)
	}
}

func TestUpdateRechargeProductRequiresAField(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, AmountCent: 1000, CoinAmount: 10, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestUpdateRechargeProductRejectsInvalidValues(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, AmountCent: 1000, CoinAmount: 10, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	zero := int64(0)
	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{AmountCent: &zero}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for non-positive amount, got %v", err)
	}
	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{CoinAmount: &zero}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for non-positive coins, got %v", err)
	}
	negative := int64(-1)
	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{PointsAmount: &negative}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for negative points, got %v", err)
	}
}

func TestUpdateRechargeProductNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	coins := int64(588)
	if _, err := svc.UpdateRechargeProduct(context.Background(), 99, RechargeProductUpdateRequest{CoinAmount: &coins}); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestUpdateRechargeProductAppliesFields(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, AmountCent: 50000, CoinAmount: 588, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	amount := int64(60000)
	points := int64(12000)
	view, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{AmountCent: &amount, PointsAmount: &points})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.AmountCent != 60000 || view.CoinAmount != 588 || view.PointsAmount != 12000 {
		t.Fatalf("unexpected updated product: %+v", view)
	}
}

func TestDisableRechargeProduct(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, AmountCent: 1000, CoinAmount: 10, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.DisableRechargeProduct(context.Background(), 1)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if view.Status != StatusDisabled {
		t.Fatalf("expected disabled status, got %+v", view)
	}
}

func TestListRechargeProductsMapsView(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{
		{ID: 1, AmountCent: 50000, CoinAmount: 588, PointsAmount: 10000, SortOrder: 1, Status: StatusActive},
	}
	svc := NewService(repo, fakeAssets{}, nil)

	views, err := svc.ListRechargeProducts(context.Background())
	if err != nil {
		t.Fatalf("products: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 product, got %d", len(views))
	}
	got := views[0]
	if got.ID != 1 || got.AmountCent != 50000 || got.CoinAmount != 588 || got.PointsAmount != 10000 || got.SortOrder != 1 {
		t.Fatalf("unexpected recharge product view: %+v", got)
	}
}

func TestRechargeNoticeUsesConfiguredCopy(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeAssets{}, nil, fakePhonePolicy{
		days: 30, notice: "首次充值低于上限赠送双倍积分。",
	})

	notice, err := svc.RechargeNotice(context.Background())
	if err != nil {
		t.Fatalf("recharge notice: %v", err)
	}
	if notice != "首次充值低于上限赠送双倍积分。" {
		t.Fatalf("unexpected recharge notice %q", notice)
	}
}

func TestCreateRechargeProductBindsPublishedCoupon(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, fakePhone{})
	couponID := int64(7)
	view, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{
		AmountCent: 20000, CoinAmount: 200, CouponTemplateID: &couponID,
	})
	if err != nil {
		t.Fatalf("create recharge product: %v", err)
	}
	if view.CouponTemplateID == nil || *view.CouponTemplateID != couponID {
		t.Fatalf("couponTemplateId = %v, want %d", view.CouponTemplateID, couponID)
	}

	invalidID := int64(99)
	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{
		AmountCent: 30000, CoinAmount: 300, CouponTemplateID: &invalidID,
	}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("invalid coupon error = %v, want INVALID_ARGUMENT", err)
	}
}

func TestUpdateRechargeProductClearsCouponBinding(t *testing.T) {
	couponID := int64(7)
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{
		ID: 1, AmountCent: 20000, CoinAmount: 200, CouponTemplateID: &couponID, Status: StatusActive,
	}}
	svc := NewService(repo, fakeAssets{}, fakePhone{})
	clearID := int64(0)
	view, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{
		CouponTemplateID: &clearID,
	})
	if err != nil {
		t.Fatalf("clear coupon binding: %v", err)
	}
	if view.CouponTemplateID != nil {
		t.Fatalf("couponTemplateId = %v, want nil", view.CouponTemplateID)
	}
}

func TestNormalizeBenefitConfigAcceptsRecurringVIPBenefits(t *testing.T) {
	config := TierBenefitConfig{
		Points: []TierPointBenefit{{Amount: 10000, Period: "daily", Trigger: "low_spend"}},
		Coupons: []TierCouponBenefit{
			{CategoryID: 12, Quantity: 2, Period: "once", Trigger: "tier_achieved"},
			{CouponType: "drink", Quantity: 1, Period: "daily", Trigger: "visit"},
			{CouponType: "event_ticket", Quantity: 5, Period: "weekly", Trigger: "weekday_event"},
			{CouponType: "event_ticket", Quantity: 4, Period: "monthly", Trigger: "weekly_event"},
			{CouponType: "event_ticket", Quantity: 2, Period: "monthly", Trigger: "monthly_event"},
			{CouponType: "gift", Quantity: 1, Period: "daily", Trigger: "period_start"},
		},
	}
	if _, err := normalizeBenefitConfig(config); err != nil {
		t.Fatalf("normalize recurring VIP benefits: %v", err)
	}
}
