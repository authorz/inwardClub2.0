package member

import (
	"context"
	"fmt"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeRepo is an in-memory Repository for exercising the service mapping and
// invitation rules without a database.
type fakeRepo struct {
	members    map[int64]*Member
	byCode     map[string]int64
	invitees   map[int64][]Invitee
	tiers      []MembershipTier
	products   []RechargeProduct
	rankings   []RankingEntry
	notReady   bool // when true, catalogue reads return NOT_IMPLEMENTED
	lastPhone  string
	lastUpdate ProfileUpdate
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
	if p.AvatarAssetID != nil {
		m.AvatarAssetID = p.AvatarAssetID
	}
	return nil
}

func (r *fakeRepo) UpdatePhone(_ context.Context, id int64, phone string) error {
	r.lastPhone = phone
	r.members[id].Phone = phone
	return nil
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
		ID:          int64(len(r.tiers) + 1),
		Name:        t.Name,
		Level:       t.Level,
		Threshold:   t.Threshold,
		Benefits:    t.Benefits,
		IconAssetID: t.IconAssetID,
		BannerPath:  t.BannerPath,
		Status:      status,
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
		if u.IconAssetID != nil {
			r.tiers[i].IconAssetID = u.IconAssetID
		}
		if u.BannerPath != nil {
			r.tiers[i].BannerPath = *u.BannerPath
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
		ID:           int64(len(r.products) + 1),
		Name:         p.Name,
		Amount:       p.Amount,
		BonusAmount:  p.BonusAmount,
		GrowthAmount: p.GrowthAmount,
		AssetType:    p.AssetType,
		SortOrder:    p.SortOrder,
		Status:       status,
	}
	r.products = append(r.products, product)
	return product, nil
}

func (r *fakeRepo) UpdateRechargeProduct(_ context.Context, id int64, u RechargeProductUpdate) (RechargeProduct, error) {
	for i := range r.products {
		if r.products[i].ID != id {
			continue
		}
		if u.Name != nil {
			r.products[i].Name = *u.Name
		}
		if u.Amount != nil {
			r.products[i].Amount = *u.Amount
		}
		if u.BonusAmount != nil {
			r.products[i].BonusAmount = *u.BonusAmount
		}
		if u.GrowthAmount != nil {
			r.products[i].GrowthAmount = *u.GrowthAmount
		}
		if u.AssetType != nil {
			r.products[i].AssetType = *u.AssetType
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

type fakePhone struct{ phone string }

func (f fakePhone) ResolvePhone(_ context.Context, _ string) (string, error) { return f.phone, nil }

func codeOf(err error) apperr.Code { return apperr.From(err).Code }

func TestUpdateProfileWritesAndMasksPhone(t *testing.T) {
	repo := newFakeRepo()
	repo.members[1] = &Member{ID: 1, Nickname: "old", Phone: "13800001111", Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	nick := "new"
	view, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{Nickname: &nick})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.Nickname != "new" {
		t.Fatalf("expected nickname updated, got %q", view.Nickname)
	}
	if view.Phone != "138****1111" {
		t.Fatalf("expected masked phone, got %q", view.Phone)
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
	avatar := int64(9)
	repo.invitees[1] = []Invitee{{MemberID: 2, Nickname: "b", AvatarAssetID: &avatar}}
	svc := NewService(repo, fakeAssets{}, nil)

	views, total, err := svc.ListInvitations(context.Background(), 1, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 invitation, got %d/%d", total, len(views))
	}
	if views[0].AvatarURL == "" || views[0].JoinedAt == "" {
		t.Fatalf("expected avatar and joinedAt populated: %+v", views[0])
	}
}

func TestRankingPeriodValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.ListRankings(context.Background(), "yearly"); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestListRankingsMapsView(t *testing.T) {
	repo := newFakeRepo()
	avatar := int64(7)
	repo.rankings = []RankingEntry{
		{Rank: 1, MemberID: 42, Nickname: "top", AvatarAssetID: &avatar, Score: 500},
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
	if got.Rank != 1 || got.MemberID != 42 || got.Nickname != "top" || got.Score != 500 || got.AvatarURL == "" {
		t.Fatalf("unexpected ranking view: %+v", got)
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
	if _, err := svc.ListRankings(ctx, "week"); codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("rankings: expected NOT_IMPLEMENTED, got %v", err)
	}
}

func TestMembershipTierMapping(t *testing.T) {
	repo := newFakeRepo()
	icon := int64(3)
	banner := int64(8)
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Level: 2, Threshold: 1000, IconAssetID: &icon, BannerAssetID: &banner, Status: StatusActive}}
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
	if views[0].BannerURL != "https://cdn.test/asset/8" {
		t.Fatalf("expected banner resolved from asset 8, got %q", views[0].BannerURL)
	}
}

func TestCreateMembershipTierPersistsBanner(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CreateMembershipTier(context.Background(), MembershipTierCreateRequest{Name: "Platinum", BannerPath: "vip/banner-42.png"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.BannerURL != "https://cdn.test/vip/banner-42.png" {
		t.Fatalf("expected banner resolved from path, got %q", view.BannerURL)
	}
	if view.BannerPath != "vip/banner-42.png" {
		t.Fatalf("expected BannerPath echoed in view, got %q", view.BannerPath)
	}
	if repo.tiers[0].BannerPath != "vip/banner-42.png" {
		t.Fatalf("expected banner path persisted, got %q", repo.tiers[0].BannerPath)
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
	banner := int64(8)
	tierID := int64(1)
	repo.tiers = []MembershipTier{{ID: 1, Name: "Gold", Level: 2, Threshold: 1000, BannerAssetID: &banner, Status: StatusActive}}
	repo.members[5] = &Member{ID: 5, CurrentTierID: &tierID, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view == nil || view.Name != "Gold" || view.BannerURL != "https://cdn.test/asset/8" {
		t.Fatalf("unexpected current tier view: %+v", view)
	}
}

func TestCurrentTierViewUnranked(t *testing.T) {
	repo := newFakeRepo()
	repo.members[5] = &Member{ID: 5, Status: StatusActive} // no CurrentTierID
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view != nil {
		t.Fatalf("expected nil view for unranked member, got %+v", view)
	}
}

func TestCurrentTierViewDanglingReference(t *testing.T) {
	repo := newFakeRepo()
	missing := int64(99)
	repo.members[5] = &Member{ID: 5, CurrentTierID: &missing, Status: StatusActive}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CurrentTierView(context.Background(), 5)
	if err != nil {
		t.Fatalf("current tier: %v", err)
	}
	if view != nil {
		t.Fatalf("expected nil view for dangling tier reference, got %+v", view)
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
		{ID: 1, Name: "Starter", Amount: 100, Status: StatusActive},
		{ID: 2, Name: "Retired", Amount: 200, Status: StatusDisabled},
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
	repo.products = []RechargeProduct{{ID: 1, Name: "Retired", Amount: 100, Status: StatusDisabled}}
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.AdminGetRechargeProduct(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Status != StatusDisabled {
		t.Fatalf("expected disabled product visible to admin, got %+v", view)
	}
}

func TestCreateRechargeProductRequiresNameAndAmount(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing name, got %v", err)
	}
	if _, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{Name: "Pack", Amount: 0}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for non-positive amount, got %v", err)
	}
}

func TestCreateRechargeProductDefaultsStatusAndAssetType(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	view, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{Name: "Starter", Amount: 100})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Status != StatusActive || view.AssetType != "coin" || view.Name != "Starter" || view.ID == 0 {
		t.Fatalf("unexpected created product: %+v", view)
	}
}

func TestRechargeProductGrowthAmountRoundTrips(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	// growthAmount is the config source the recharge settlement reads to accrue
	// growth_value; it must survive create and partial update unchanged.
	created, err := svc.CreateRechargeProduct(context.Background(), RechargeProductCreateRequest{
		Name: "Growth Pack", Amount: 5000, BonusAmount: 500, GrowthAmount: 5000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GrowthAmount != 5000 {
		t.Fatalf("created growthAmount = %d, want 5000", created.GrowthAmount)
	}

	growth := int64(8000)
	updated, err := svc.UpdateRechargeProduct(context.Background(), created.ID, RechargeProductUpdateRequest{GrowthAmount: &growth})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.GrowthAmount != 8000 {
		t.Fatalf("updated growthAmount = %d, want 8000", updated.GrowthAmount)
	}
	// The coins bonus must be untouched by a growth-only update.
	if updated.BonusAmount != 500 {
		t.Fatalf("bonusAmount changed unexpectedly: %d, want 500", updated.BonusAmount)
	}
}

func TestUpdateRechargeProductRequiresAField(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, Name: "Starter", Amount: 100, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestUpdateRechargeProductRejectsInvalidValues(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, Name: "Starter", Amount: 100, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	empty := ""
	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{Name: &empty}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for empty name, got %v", err)
	}
	zero := int64(0)
	if _, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{Amount: &zero}); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for non-positive amount, got %v", err)
	}
}

func TestUpdateRechargeProductNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeAssets{}, nil)

	name := "Starter"
	if _, err := svc.UpdateRechargeProduct(context.Background(), 99, RechargeProductUpdateRequest{Name: &name}); codeOf(err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestUpdateRechargeProductAppliesFields(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, Name: "Starter", Amount: 100, Status: StatusActive}}
	svc := NewService(repo, fakeAssets{}, nil)

	name := "Starter Plus"
	amount := int64(200)
	view, err := svc.UpdateRechargeProduct(context.Background(), 1, RechargeProductUpdateRequest{Name: &name, Amount: &amount})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.Name != "Starter Plus" || view.Amount != 200 {
		t.Fatalf("unexpected updated product: %+v", view)
	}
}

func TestDisableRechargeProduct(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []RechargeProduct{{ID: 1, Name: "Starter", Amount: 100, Status: StatusActive}}
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
		{ID: 1, Name: "Starter", Amount: 100, BonusAmount: 10, AssetType: "coin", SortOrder: 1, Status: StatusActive},
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
	if got.ID != 1 || got.Name != "Starter" || got.Amount != 100 || got.BonusAmount != 10 || got.AssetType != "coin" || got.SortOrder != 1 {
		t.Fatalf("unexpected recharge product view: %+v", got)
	}
}
