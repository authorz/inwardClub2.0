package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/modules/wallet"
	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeRepo returns fixed pages and records the filter it was called with so the
// tests can assert both mapping and that the store scope is propagated.
type fakeRepo struct {
	lastFilter ListFilter
	err        error

	cashiers         map[int64]AdminAccount
	staffAccounts    map[int64]StaffAccount
	adminAccounts    map[int64]AdminAccount
	nextID           int64
	lastPasswordHash string

	members map[int64]Member

	activities []Activity
}

func (f *fakeRepo) ListStores(_ context.Context, flt ListFilter) ([]StoreSummary, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	lat, lng := 29.56301, 106.55156
	return []StoreSummary{{ID: 1, Name: "HQ", Latitude: &lat, Longitude: &lng, Status: StatusActive}}, 3, nil
}
func (f *fakeRepo) ListCatalogItems(_ context.Context, flt ListFilter) ([]CatalogItem, int64, error) {
	f.lastFilter = flt
	return []CatalogItem{{ID: 10, Name: "Latte", ItemType: "food", PriceCent: 1500}}, 1, nil
}
func (f *fakeRepo) ListCouponTemplates(_ context.Context, flt ListFilter) ([]CouponTemplate, int64, error) {
	f.lastFilter = flt
	return []CouponTemplate{{ID: 20, Name: "5off"}}, 1, nil
}
func (f *fakeRepo) ListActivities(_ context.Context, flt ListFilter) ([]Activity, int64, error) {
	f.lastFilter = flt
	if f.activities != nil {
		return f.activities, int64(len(f.activities)), nil
	}
	return []Activity{{ID: 30, Name: "Spring"}}, 1, nil
}
func (f *fakeRepo) ListOrders(_ context.Context, flt ListFilter) ([]Order, int64, error) {
	f.lastFilter = flt
	return []Order{{
		ID: 40, OrderNo: "NO-1", OrderType: "food", StoreName: "三里屯店",
		MemberNickname: "Sam", MemberPhone: "13800001234",
		MemberAvatarURL: "https://cdn.test/member.webp", PaymentOrderID: 80, TotalCent: 999,
		PayChannel: "wechat", RefundStatus: "processing", PaymentStatus: "paid", OrderStatus: "completed",
	}}, 1, nil
}
func (f *fakeRepo) ListMembers(_ context.Context, flt ListFilter) ([]Member, int64, error) {
	f.lastFilter = flt
	return []Member{{
		ID: 50, Nickname: "Sam", Phone: "13800001234",
		AvatarURL: "https://cdn.test/avatar.webp", Gender: "male",
		PointsBalance: 1200, CoinsBalance: 588,
		VIPTierName: "白银会员", VIPLevel: 2,
		Status: StatusActive,
	}}, 1, nil
}
func (f *fakeRepo) GetMemberByID(_ context.Context, memberID int64) (Member, error) {
	if f.err != nil {
		return Member{}, f.err
	}
	if f.members != nil {
		if m, ok := f.members[memberID]; ok {
			return m, nil
		}
		return Member{}, apperr.NotFound("member not found")
	}
	return Member{ID: memberID, Nickname: "n", Status: StatusActive}, nil
}

func (f *fakeRepo) SearchMembersByPhone(_ context.Context, phone string) ([]Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]Member, 0)
	if f.members != nil {
		for _, m := range f.members {
			if strings.Contains(m.Phone, phone) {
				out = append(out, m)
			}
		}
		return out, nil
	}
	return []Member{{ID: 50, Nickname: "n", Phone: phone, Status: StatusActive}}, nil
}

func (f *fakeRepo) ListWalletLedger(_ context.Context, flt ListFilter) ([]WalletLedgerEntry, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	sourceID := int64(999)
	balanceAfter := int64(150)
	return []WalletLedgerEntry{{
		ID: 100, RecordKey: "ledger:100",
		MemberID: 7, MemberNickname: "Sam", MemberPhone: "13800001234",
		MemberAvatarURL: "https://cdn.test/member.webp",
		AssetType:       "points", Direction: "credit", Amount: 50,
		BalanceAfter: &balanceAfter, Status: "completed",
		Reason: "sign_in", SourceType: "sign_in", SourceID: &sourceID,
		RelatedOrderNo: "BO-1",
	}}, 1, nil
}

func (f *fakeRepo) ListPaymentTransactions(_ context.Context, flt ListFilter) ([]PaymentTransaction, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []PaymentTransaction{{ID: 80, PaymentOrderNo: "PO-1", AmountCent: 500, Status: "paid"}}, 1, nil
}
func (f *fakeRepo) ListRefunds(_ context.Context, flt ListFilter) ([]Refund, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []Refund{{
		ID: 85, RefundOrderNo: "RF-1", PaymentOrderID: 80,
		BusinessOrderNo: "BO-1", OrderAmountCent: 900,
		MemberNickname: "Sam", MemberPhone: "13800001234",
		MemberAvatarURL: "https://cdn.test/member.webp",
		AmountCent:      500, Status: "succeeded",
		OrderCreatedAt: time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC),
		OperatedAt:     time.Date(2026, 7, 21, 11, 0, 0, 0, time.UTC),
	}}, 1, nil
}
func (f *fakeRepo) ListAuditLogs(_ context.Context, flt ListFilter) ([]AuditLog, int64, error) {
	f.lastFilter = flt
	return []AuditLog{{
		ID: 60, ActorType: "store_admin", ActorID: 9, ActorRole: "store_admin",
		Action: "member.wallet.adjust", TargetType: "member", TargetID: "7",
		ActorSnapshotJSON:  json.RawMessage(`{"type":"store_admin","id":9,"username":"store-admin","displayName":"门店管理员"}`),
		TargetSnapshotJSON: json.RawMessage(`{"type":"member","id":7,"nickname":"测试会员","phone":"13800138000"}`),
		ScopeSnapshotJSON:  json.RawMessage(`{"storeId":1,"storeName":"新壹街店"}`),
		BeforeJSON:         json.RawMessage(`{"assetType":"points","availableAmount":100}`),
		AfterJSON:          json.RawMessage(`{"assetType":"points","availableAmount":120}`),
		Reason:             "客服补偿", RequestID: "req-audit-1",
	}}, 1, nil
}
func (f *fakeRepo) ListRuleDefinitions(_ context.Context, flt ListFilter) ([]RuleDefinition, int64, error) {
	f.lastFilter = flt
	return []RuleDefinition{{ID: 70, Key: "sign_in", Enabled: true, Status: "published"}}, 1, nil
}
func (f *fakeRepo) GetRuleDefinition(_ context.Context, ruleID int64) (RuleDefinition, error) {
	if f.err != nil {
		return RuleDefinition{}, f.err
	}
	return RuleDefinition{ID: ruleID, Key: "sign_in", ScopeType: "global", ConfigJSON: json.RawMessage(`{}`)}, nil
}
func (f *fakeRepo) ListAdminAccounts(_ context.Context, flt ListFilter) ([]AdminAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []AdminAccount{{ID: 90, Username: "hq_admin", DisplayName: "HQ Admin", Role: "store_admin"}}, 1, nil
}
func (f *fakeRepo) ListCashiers(_ context.Context, flt ListFilter) ([]AdminAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []AdminAccount{{ID: 91, Username: "cashier1", DisplayName: "Cashier One", Role: "cashier"}}, 1, nil
}
func (f *fakeRepo) ListStaffAccounts(_ context.Context, flt ListFilter) ([]StaffAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []StaffAccount{{ID: 92, Name: "Waiter Wang", StoreID: 42}}, 1, nil
}
func (f *fakeRepo) ListSuperAdmins(_ context.Context, flt ListFilter) ([]AdminAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []AdminAccount{{ID: 93, Username: "superadmin", DisplayName: "System Admin", Role: "super_admin", IsSystem: true}}, 1, nil
}
func (f *fakeRepo) ListStoreAdmins(_ context.Context, flt ListFilter) ([]AdminAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	storeID := int64(42)
	return []AdminAccount{{ID: 94, Username: "store_admin1", DisplayName: "Store Admin One", Role: "store_admin", StoreID: &storeID}}, 1, nil
}

type fakeActivityAssetResolver struct{}

func (fakeActivityAssetResolver) PublicURLByID(_ context.Context, _ int64) (string, error) {
	return "https://cdn.test/activity-cover.webp", nil
}
func (f *fakeRepo) GetAdminAccountByID(_ context.Context, id int64) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.adminAccounts[id]
	if !ok {
		return AdminAccount{}, apperr.NotFound("admin account not found")
	}
	return a, nil
}
func (f *fakeRepo) CreateSuperAdmin(_ context.Context, username, passwordHash, displayName string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	if f.adminAccounts == nil {
		f.adminAccounts = map[int64]AdminAccount{}
	}
	for _, a := range f.adminAccounts {
		if a.Username == username {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
	}
	f.lastPasswordHash = passwordHash
	f.nextID++
	a := AdminAccount{
		ID: f.nextID, Username: username, DisplayName: displayName,
		Role: "super_admin", Status: StatusActive,
	}
	f.adminAccounts[a.ID] = a
	return a, nil
}
func (f *fakeRepo) UpdateSuperAdminByID(_ context.Context, id int64, displayName, passwordHash *string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.adminAccounts[id]
	if !ok || a.Role != "super_admin" {
		return AdminAccount{}, apperr.NotFound("admin account not found")
	}
	if displayName != nil {
		a.DisplayName = *displayName
	}
	if passwordHash != nil {
		f.lastPasswordHash = *passwordHash
	}
	f.adminAccounts[id] = a
	return a, nil
}
func (f *fakeRepo) DeleteSuperAdminByID(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	a, ok := f.adminAccounts[id]
	if !ok || a.Role != "super_admin" {
		return apperr.NotFound("admin account not found")
	}
	if a.IsSystem {
		return apperr.Forbidden("system administrator cannot be deleted")
	}
	delete(f.adminAccounts, id)
	return nil
}
func (f *fakeRepo) CreateStoreAdmin(_ context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	if f.adminAccounts == nil {
		f.adminAccounts = map[int64]AdminAccount{}
	}
	for _, a := range f.adminAccounts {
		if a.Username == username {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
	}
	f.lastPasswordHash = passwordHash
	f.nextID++
	id := f.nextID
	a := AdminAccount{ID: id, Username: username, DisplayName: displayName, Role: "store_admin", StoreID: &storeID, Status: StatusActive}
	f.adminAccounts[id] = a
	return a, nil
}
func (f *fakeRepo) UpdateStoreAdminByID(_ context.Context, id int64, storeID *int64, displayName, passwordHash *string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.adminAccounts[id]
	if !ok || a.Role != "store_admin" {
		return AdminAccount{}, apperr.NotFound("store admin account not found")
	}
	if displayName != nil {
		a.DisplayName = *displayName
	}
	if storeID != nil {
		a.StoreID = storeID
	}
	if passwordHash != nil {
		f.lastPasswordHash = *passwordHash
	}
	f.adminAccounts[id] = a
	return a, nil
}
func (f *fakeRepo) DisableAdminAccountByID(_ context.Context, id int64) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.adminAccounts[id]
	if !ok {
		return AdminAccount{}, apperr.NotFound("admin account not found")
	}
	a.Status = "disabled"
	f.adminAccounts[id] = a
	return a, nil
}

func (f *fakeRepo) CreateRuleDefinition(_ context.Context, req RuleDefinitionCreate) (RuleDefinition, error) {
	if f.err != nil {
		return RuleDefinition{}, f.err
	}
	f.nextID++
	return RuleDefinition{
		ID: f.nextID, Key: req.Key, ScopeType: req.ScopeType, StoreID: req.StoreID,
		Version: req.Version, ConfigJSON: req.ConfigJSON, Enabled: req.Enabled, Status: "draft",
	}, nil
}

func (f *fakeRepo) UpdateRuleDefinition(_ context.Context, ruleID int64, u RuleDefinitionUpdate) (RuleDefinition, error) {
	if f.err != nil {
		return RuleDefinition{}, f.err
	}
	rd := RuleDefinition{ID: ruleID, Key: "sign_in", Status: "published", Enabled: true}
	if len(u.ConfigJSON) > 0 {
		rd.ConfigJSON = u.ConfigJSON
	}
	if u.Enabled != nil {
		rd.Enabled = *u.Enabled
	}
	if u.Status != nil {
		rd.Status = *u.Status
	}
	return rd, nil
}

func (f *fakeRepo) GetCashier(_ context.Context, storeID, id int64) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.cashiers[id]
	if !ok || a.StoreID == nil || *a.StoreID != storeID {
		return AdminAccount{}, apperr.NotFound("cashier not found")
	}
	return a, nil
}

func (f *fakeRepo) CreateCashier(_ context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	if f.cashiers == nil {
		f.cashiers = map[int64]AdminAccount{}
	}
	for _, a := range f.cashiers {
		if a.Username == username {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
	}
	f.nextID++
	id := f.nextID
	a := AdminAccount{ID: id, Username: username, DisplayName: displayName, Role: "cashier", StoreID: &storeID, Status: StatusActive}
	f.cashiers[id] = a
	return a, nil
}

func (f *fakeRepo) UpdateCashier(_ context.Context, storeID, id int64, displayName string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.cashiers[id]
	if !ok || a.StoreID == nil || *a.StoreID != storeID {
		return AdminAccount{}, apperr.NotFound("cashier not found")
	}
	a.DisplayName = displayName
	f.cashiers[id] = a
	return a, nil
}

func (f *fakeRepo) DisableCashier(_ context.Context, storeID, id int64) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.cashiers[id]
	if !ok || a.StoreID == nil || *a.StoreID != storeID {
		return AdminAccount{}, apperr.NotFound("cashier not found")
	}
	a.Status = "disabled"
	f.cashiers[id] = a
	return a, nil
}

func (f *fakeRepo) ResetCashierPassword(_ context.Context, storeID, id int64, passwordHash string) (AdminAccount, error) {
	if f.err != nil {
		return AdminAccount{}, f.err
	}
	a, ok := f.cashiers[id]
	if !ok || a.StoreID == nil || *a.StoreID != storeID {
		return AdminAccount{}, apperr.NotFound("cashier not found")
	}
	return a, nil
}

func (f *fakeRepo) GetStaffAccount(_ context.Context, storeID, id int64) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok || a.StoreID != storeID {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	return a, nil
}

func (f *fakeRepo) CreateStaffAccount(_ context.Context, storeID, memberID int64, name string) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	if f.staffAccounts == nil {
		f.staffAccounts = map[int64]StaffAccount{}
	}
	phone := ""
	if f.members != nil {
		m, ok := f.members[memberID]
		if !ok {
			return StaffAccount{}, apperr.NotFound("member not found")
		}
		if name == "" {
			name = m.Nickname
		}
		phone = m.Phone
	}
	if name == "" {
		name = "会员"
	}
	for id, existing := range f.staffAccounts {
		if existing.MemberID != memberID {
			continue
		}
		if existing.Status != "disabled" {
			return StaffAccount{}, apperr.Conflict("该会员已被绑定为员工")
		}
		existing.StoreID = storeID
		existing.Name = name
		existing.Phone = phone
		existing.Status = StatusActive
		f.staffAccounts[id] = existing
		return existing, nil
	}
	f.nextID++
	id := f.nextID
	a := StaffAccount{ID: id, MemberID: memberID, Name: name, Phone: phone, StoreID: storeID, Status: StatusActive}
	f.staffAccounts[id] = a
	return a, nil
}

func (f *fakeRepo) DeleteStaffAccount(_ context.Context, storeID, id int64) error {
	if f.err != nil {
		return f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok || a.StoreID != storeID {
		return apperr.NotFound("staff account not found")
	}
	delete(f.staffAccounts, id)
	return nil
}

func (f *fakeRepo) DeleteStaffAccountByID(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.staffAccounts[id]; !ok {
		return apperr.NotFound("staff account not found")
	}
	delete(f.staffAccounts, id)
	return nil
}

func (f *fakeRepo) UpdateStaffAccount(_ context.Context, storeID, id int64, name string) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok || a.StoreID != storeID {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	a.Name = name
	f.staffAccounts[id] = a
	return a, nil
}

func (f *fakeRepo) DisableStaffAccount(_ context.Context, storeID, id int64) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok || a.StoreID != storeID {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	a.Status = "disabled"
	f.staffAccounts[id] = a
	return a, nil
}

func (f *fakeRepo) GetStaffAccountByID(_ context.Context, id int64) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	return a, nil
}

func (f *fakeRepo) UpdateStaffAccountByID(_ context.Context, id int64, storeID *int64, name *string) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	if name != nil {
		a.Name = *name
	}
	if storeID != nil {
		a.StoreID = *storeID
	}
	f.staffAccounts[id] = a
	return a, nil
}

func (f *fakeRepo) DisableStaffAccountByID(_ context.Context, id int64) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	a, ok := f.staffAccounts[id]
	if !ok {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	a.Status = "disabled"
	f.staffAccounts[id] = a
	return a, nil
}

type fakeStores struct{}

func (fakeStores) StoreProfile(_ context.Context, id int64) (StoreProfileView, error) {
	return StoreProfileView{ID: id, Name: "Store A", Status: StatusActive}, nil
}

func page() httpx.Page { return httpx.Page{Page: 1, PageSize: 20} }

func TestListStoresMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListStores(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(views) != 1 || views[0].Name != "HQ" {
		t.Fatalf("unexpected views: %+v", views)
	}
	if views[0].Latitude == nil || views[0].Longitude == nil || *views[0].Latitude != 29.56301 || *views[0].Longitude != 106.55156 {
		t.Fatalf("coordinates were not mapped: %+v", views[0])
	}
}

func TestListActivitiesIncludesCoverAndAuditTimes(t *testing.T) {
	assetID := int64(88)
	createdAt := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	repo := &fakeRepo{activities: []Activity{{
		ID: 30, Name: "Spring", AssetID: &assetID,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}}}
	svc := NewService(repo, fakeStores{}, nil, fakeActivityAssetResolver{})

	rows, total, err := svc.ListActivities(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("unexpected page: total=%d rows=%d", total, len(rows))
	}
	if rows[0].AssetID == nil || *rows[0].AssetID != assetID {
		t.Fatalf("unexpected asset id: %+v", rows[0].AssetID)
	}
	if rows[0].ImageURL != "https://cdn.test/activity-cover.webp" {
		t.Fatalf("unexpected image url: %q", rows[0].ImageURL)
	}
	if rows[0].Title != "Spring" {
		t.Fatalf("store activity fields were not mapped: %+v", rows[0])
	}
	if !rows[0].CreatedAt.Equal(createdAt) || !rows[0].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected audit times: created=%v updated=%v", rows[0].CreatedAt, rows[0].UpdatedAt)
	}
}

func TestListPropagatesError(t *testing.T) {
	sentinel := apperr.Internal(errors.New("boom"))
	svc := NewService(&fakeRepo{err: sentinel}, fakeStores{}, nil)

	_, _, err := svc.ListStores(context.Background(), ListFilter{Page: page()})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestScopedFilterPinsStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListCatalogItems(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestListOrdersMapsOrderCenterReadModel(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListOrders(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 view, got total=%d len=%d", total, len(views))
	}
	v := views[0]
	// The order-center pages render amountCent/orderType/paymentStatus/orderStatus/
	// storeName; businessOrderId mirrors id; orderNo/totalCent/status are aliases.
	if v.BusinessOrderID != v.ID || v.AmountCent != v.TotalCent || v.AmountCent != 999 {
		t.Fatalf("unexpected amount/id aliases: %+v", v)
	}
	if v.OrderType != "food" || v.StoreName != "三里屯店" || v.PaymentStatus != "paid" ||
		v.RefundStatus != "processing" || v.OrderStatus != "completed" || v.Status != "completed" {
		t.Fatalf("unexpected order-center fields: %+v", v)
	}
	// memberPhone is exposed raw for HQ; memberPhoneMasked hides the middle digits.
	if v.MemberPhone != "13800001234" || v.MemberPhoneMasked != "138****1234" ||
		v.MemberNickname != "Sam" || v.MemberAvatarURL != "https://cdn.test/member.webp" ||
		v.PaymentOrderID != 80 {
		t.Fatalf("unexpected member fields: %+v", v)
	}
}

func TestListMembersMapsProfileWalletAndVIPFields(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	rows, total, err := svc.ListMembers(context.Background(), ListFilter{
		Keyword:   "  Sam  ",
		SortBy:    "coinsBalance",
		SortOrder: "ASC",
	})
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("unexpected page: total=%d rows=%+v", total, rows)
	}
	got := rows[0]
	if got.AvatarURL == "" || got.Gender != "male" || got.PointsBalance != 1200 ||
		got.CoinsBalance != 588 || got.VIPTierName != "白银会员" || got.VIPLevel != 2 {
		t.Fatalf("unexpected member view: %+v", got)
	}
	if repo.lastFilter.Keyword != "Sam" || repo.lastFilter.SortBy != "coinsBalance" ||
		repo.lastFilter.SortOrder != "asc" {
		t.Fatalf("unexpected normalized filter: %+v", repo.lastFilter)
	}
}

func TestListMembersRejectsUnknownSort(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, _, err := svc.ListMembers(context.Background(), ListFilter{SortBy: "phone"})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid sort error, got %v", err)
	}
}

func TestListPaymentTransactionsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListPaymentTransactions(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].PaymentOrderNo != "PO-1" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestScopedFilterPinsStoreForPaymentTransactions(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListPaymentTransactions(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestListRefundsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListRefunds(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].RefundOrderNo != "RF-1" ||
		views[0].BusinessOrderNo != "BO-1" || views[0].OrderAmountCent != 900 ||
		views[0].MemberNickname != "Sam" ||
		views[0].MemberAvatarURL != "https://cdn.test/member.webp" ||
		views[0].OperatedAt.IsZero() {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestScopedFilterPinsStoreForRefunds(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListRefunds(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestListWalletLedgerMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListWalletLedger(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].MemberID != 7 || views[0].AssetType != "points" ||
		views[0].BalanceAfter == nil || *views[0].BalanceAfter != 150 ||
		views[0].Status != "completed" || views[0].RecordKey != "ledger:100" ||
		views[0].MemberNickname != "Sam" ||
		views[0].MemberAvatarURL != "https://cdn.test/member.webp" ||
		views[0].RelatedOrderNo != "BO-1" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestListWalletLedgerPassesMemberAndAssetFilter(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	memberID := int64(7)
	f := ListFilter{Page: page(), MemberID: &memberID, AssetType: "points"}
	if _, _, err := svc.ListWalletLedger(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.MemberID == nil || *repo.lastFilter.MemberID != 7 || repo.lastFilter.AssetType != "points" {
		t.Fatalf("expected memberId/assetType filter propagated, got %+v", repo.lastFilter)
	}
}

func TestScopedFilterPinsStoreForWalletLedger(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListWalletLedger(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestListAdminAccountsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListAdminAccounts(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Username != "hq_admin" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestListCashiersMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListCashiers(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Role != "cashier" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestScopedFilterPinsStoreForCashiers(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListCashiers(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestListSuperAdminsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListSuperAdmins(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Role != "super_admin" || !views[0].IsSystem {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestCreateSuperAdminHashesProvidedPassword(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	view, err := svc.CreateSuperAdmin(context.Background(), AdminAccountCreateRequest{
		Username: "ops-admin", Password: "secret", DisplayName: "Ops Admin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Username != "ops-admin" || view.Role != "super_admin" || view.IsSystem {
		t.Fatalf("unexpected view: %+v", view)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.lastPasswordHash), []byte("secret")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
}

func TestUpdateSystemAdminAllowsPasswordChange(t *testing.T) {
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "superadmin", Role: "super_admin", IsSystem: true, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)
	password := "new-secret"

	view, err := svc.UpdateSuperAdmin(context.Background(), 1, AdminAccountUpdateRequest{Password: &password})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !view.IsSystem {
		t.Fatalf("system status was lost: %+v", view)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.lastPasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored hash does not match new password: %v", err)
	}
}

func TestDeleteSuperAdminRemovesNonSystemAccount(t *testing.T) {
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		2: {ID: 2, Username: "ops-admin", Role: "super_admin", Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	if err := svc.DeleteSuperAdmin(context.Background(), 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := repo.adminAccounts[2]; ok {
		t.Fatal("expected account to be deleted")
	}
}

func TestSystemAdminCannotBeDisabledOrDeleted(t *testing.T) {
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "superadmin", Role: "super_admin", IsSystem: true, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	if _, err := svc.DisableAdminAccount(context.Background(), 1); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED for disable, got %v", err)
	}
	if err := svc.DeleteSuperAdmin(context.Background(), 1); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED for delete, got %v", err)
	}
}

func TestDisableAdminAccountRejectsNonSuperAdmin(t *testing.T) {
	storeID := int64(42)
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "store_admin1", Role: "store_admin", StoreID: &storeID, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	if _, err := svc.DisableAdminAccount(context.Background(), 1); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestDisableAdminAccountInvalidatesSuperAdmin(t *testing.T) {
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "root", Role: "super_admin", Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	v, err := svc.DisableAdminAccount(context.Background(), 1)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if v.Status != "disabled" {
		t.Fatalf("expected disabled, got %+v", v)
	}
}

func TestListStoreAdminsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListStoreAdmins(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Role != "store_admin" || views[0].StoreID == nil || *views[0].StoreID != 42 {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestCreateStoreAdminHashesProvidedPasswordAndScopesToStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	v, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{
		StoreID: 42, Username: "sa1", Password: "chosen-secret", DisplayName: "SA One",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.StoreID == nil || *v.StoreID != 42 || v.Role != "store_admin" {
		t.Fatalf("unexpected view: %+v", v)
	}
	if repo.lastPasswordHash == "chosen-secret" {
		t.Fatal("plaintext password was passed to the repository")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.lastPasswordHash), []byte("chosen-secret")); err != nil {
		t.Fatalf("stored hash does not match provided password: %v", err)
	}
}

func TestCreateStoreAdminRejectsMissingStoreUsernameOrPassword(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{Username: "sa1", Password: "secret"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing storeId, got %v", err)
	}
	if _, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{StoreID: 42, Password: "secret"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing username, got %v", err)
	}
	if _, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{StoreID: 42, Username: "sa1"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing password, got %v", err)
	}
}

func TestUpdateStoreAdminCanReassignStore(t *testing.T) {
	storeID := int64(42)
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "sa1", DisplayName: "SA One", Role: "store_admin", StoreID: &storeID, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	newStore := int64(7)
	newName := "SA Renamed"
	v, err := svc.UpdateStoreAdmin(context.Background(), 1, StoreAdminUpdateRequest{StoreID: &newStore, DisplayName: &newName})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.DisplayName != "SA Renamed" || v.StoreID == nil || *v.StoreID != 7 {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestUpdateStoreAdminHashesProvidedPassword(t *testing.T) {
	storeID := int64(42)
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "sa1", DisplayName: "SA One", Role: "store_admin", StoreID: &storeID, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	password := "new-secret"
	if _, err := svc.UpdateStoreAdmin(context.Background(), 1, StoreAdminUpdateRequest{Password: &password}); err != nil {
		t.Fatalf("update password: %v", err)
	}
	if repo.lastPasswordHash == password {
		t.Fatal("plaintext password was passed to the repository")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.lastPasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored hash does not match new password: %v", err)
	}
}

func TestUpdateStoreAdminRejectsEmptyBody(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.UpdateStoreAdmin(context.Background(), 1, StoreAdminUpdateRequest{}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
	emptyPassword := ""
	if _, err := svc.UpdateStoreAdmin(context.Background(), 1, StoreAdminUpdateRequest{Password: &emptyPassword}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for empty password, got %v", err)
	}
}

func TestDisableStoreAdminRejectsNonStoreAdmin(t *testing.T) {
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "root", Role: "super_admin", Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	if _, err := svc.DisableStoreAdmin(context.Background(), 1); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestDisableStoreAdminWorksAcrossStores(t *testing.T) {
	storeID := int64(42)
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "sa1", Role: "store_admin", StoreID: &storeID, Status: StatusActive},
	}}
	svc := NewService(repo, fakeStores{}, nil)

	v, err := svc.DisableStoreAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if v.Status != "disabled" {
		t.Fatalf("expected disabled, got %+v", v)
	}
}

func TestListStaffAccountsMapsAndPassesTotal(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	views, total, err := svc.ListStaffAccounts(context.Background(), ListFilter{Page: page()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Name != "Waiter Wang" || views[0].StoreID != 42 {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestScopedFilterPinsStoreForStaffAccounts(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	f := scopedFilter(ListFilter{Page: page()}, 42)
	if _, _, err := svc.ListStaffAccounts(context.Background(), f); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestGetStoreProfile(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	v, err := svc.GetStoreProfile(context.Background(), 42)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if v.ID != 42 {
		t.Fatalf("expected store 42, got %d", v.ID)
	}
}

func TestGetStoreProfileNotWired(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil, nil)
	_, err := svc.GetStoreProfile(context.Background(), 1)
	if apperr.From(err).Code != apperr.CodeNotImplemented {
		t.Fatalf("expected NOT_IMPLEMENTED, got %v", err)
	}
}

func TestUpdateRuleDefinition(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	enabled := false
	v, err := svc.UpdateRuleDefinition(context.Background(), 70, RuleDefinitionUpdate{
		ConfigJSON: []byte(`{"capDay":7}`),
		Enabled:    &enabled,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.ID != 70 || v.Enabled != false || string(v.ConfigJSON) != `{"capDay":7}` {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestUpdateRuleDefinitionRejectsEmpty(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.UpdateRuleDefinition(context.Background(), 70, RuleDefinitionUpdate{})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestUpdateRuleDefinitionRejectsBadJSON(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.UpdateRuleDefinition(context.Background(), 70, RuleDefinitionUpdate{
		ConfigJSON: []byte(`{bad`),
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestCreateRuleDefinition(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	storeID := int64(5)
	v, err := svc.CreateRuleDefinition(context.Background(), RuleDefinitionCreate{
		Key:        "sign_in",
		StoreID:    &storeID,
		ConfigJSON: []byte(`{"capDay":7}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.Key != "sign_in" || v.ScopeType != "global" || v.Version != 1 || v.Status != "draft" {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestCreateRuleDefinitionRejectsMissingKey(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.CreateRuleDefinition(context.Background(), RuleDefinitionCreate{
		ConfigJSON: []byte(`{"capDay":7}`),
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestCreateRuleDefinitionRejectsBadJSON(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.CreateRuleDefinition(context.Background(), RuleDefinitionCreate{
		Key:        "sign_in",
		ConfigJSON: []byte(`{bad`),
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestCreateInvitationRuleValidatesBusinessConfig(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	v, err := svc.CreateRuleDefinition(context.Background(), RuleDefinitionCreate{
		Key:       "invite_reward",
		ScopeType: "global",
		ConfigJSON: []byte(`{
			"firstLowSpendRewardCoins": 50,
			"firstLowSpendRewardPoints": 2000,
			"commissionRateBasisPoints": 1000
		}`),
	})
	if err != nil {
		t.Fatalf("create invitation rule: %v", err)
	}
	if v.Key != "invite_reward" {
		t.Fatalf("unexpected rule: %+v", v)
	}
}

func TestCreateInvitationRuleRejectsEmptyRewards(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.CreateRuleDefinition(context.Background(), RuleDefinitionCreate{
		Key:        "invite_reward",
		ScopeType:  "global",
		ConfigJSON: []byte(`{"firstLowSpendRewardCoins":0,"firstLowSpendRewardPoints":0,"commissionRateBasisPoints":0}`),
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestDisableRuleDefinition(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	v, err := svc.DisableRuleDefinition(context.Background(), 70)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if v.Status != "disabled" || v.Enabled {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestPublishRuleDefinition(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	v, err := svc.PublishRuleDefinition(context.Background(), 70)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if v.Status != "published" || !v.Enabled {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestStoreCreateCashierGeneratesPasswordAndScopesToStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	view, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "cashier1", DisplayName: "Cashier One"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Username != "cashier1" || view.Role != "cashier" || view.StoreID == nil || *view.StoreID != 42 {
		t.Fatalf("unexpected view: %+v", view)
	}
	if len(view.InitialPassword) != generatedPasswordLength {
		t.Fatalf("expected generated password of length %d, got %q", generatedPasswordLength, view.InitialPassword)
	}
}

func TestStoreCreateCashierRejectsEmptyUsername(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "  "})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestStoreCreateCashierRejectsDuplicateUsername(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	if _, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "dup"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "dup"})
	if apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestStoreUpdateCashierAppliesDisplayName(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "cashier1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "Renamed"
	view, err := svc.StoreUpdateCashier(context.Background(), 42, created.ID, CashierUpdateRequest{DisplayName: &newName})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.DisplayName != "Renamed" {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestStoreUpdateCashierCannotReachOtherStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "cashier1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "Renamed"
	_, err = svc.StoreUpdateCashier(context.Background(), 99, created.ID, CashierUpdateRequest{DisplayName: &newName})
	if apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store update, got %v", err)
	}
}

func TestStoreDisableCashierInvalidatesAccount(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "cashier1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	view, err := svc.StoreDisableCashier(context.Background(), 42, created.ID)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if view.Status != "disabled" {
		t.Fatalf("expected disabled status, got %+v", view)
	}
}

func TestStoreResetCashierPasswordGeneratesNewPassword(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateCashier(context.Background(), 42, CashierCreateRequest{Username: "cashier1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	view, err := svc.StoreResetCashierPassword(context.Background(), 42, created.ID)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if len(view.InitialPassword) != generatedPasswordLength {
		t.Fatalf("expected generated password of length %d, got %q", generatedPasswordLength, view.InitialPassword)
	}
}

// staffMemberRepo returns a fakeRepo pre-seeded with one registered member
// (id 7) so staff-binding tests can bind by memberId.
func staffMemberRepo() *fakeRepo {
	return &fakeRepo{members: map[int64]Member{
		7: {ID: 7, Nickname: "Waiter Wang", Phone: "13800000007", AvatarURL: "https://cdn.test/staff.webp", Status: StatusActive},
	}}
}

func TestStoreCreateStaffAccountScopesToStore(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	view, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "Waiter Wang" || view.StoreID != 42 || view.MemberID != 7 {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestStoreCreateStaffAccountRejectsMissingMember(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 0})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestStoreCreateStaffAccountReactivatesDisabledBindingAtCurrentStore(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 99, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create old binding: %v", err)
	}
	if _, err := svc.StoreDisableStaffAccount(context.Background(), 99, created.ID); err != nil {
		t.Fatalf("disable old binding: %v", err)
	}

	reactivated, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if reactivated.ID != created.ID || reactivated.StoreID != 42 || reactivated.Status != StatusActive {
		t.Fatalf("unexpected reactivated binding: %+v", reactivated)
	}
}

func TestStoreCreateStaffAccountKeepsActiveCrossStoreBindingProtected(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	if _, err := svc.StoreCreateStaffAccount(context.Background(), 99, StaffAccountCreateRequest{MemberID: 7}); err != nil {
		t.Fatalf("create old binding: %v", err)
	}

	_, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected active binding conflict, got %v", err)
	}
}

func TestStoreUpdateStaffAccountCannotReachOtherStore(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "Someone Else"
	_, err = svc.StoreUpdateStaffAccount(context.Background(), 99, created.ID, StaffAccountUpdateRequest{Name: &newName})
	if apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store update, got %v", err)
	}
}

func TestStoreDeleteStaffAccountRemovesBinding(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Cross-store delete must not reach another store's binding.
	if err := svc.StoreDeleteStaffAccount(context.Background(), 99, created.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store delete, got %v", err)
	}
	if err := svc.StoreDeleteStaffAccount(context.Background(), 42, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// The binding is gone; the member account itself is not tracked here, but the
	// staff row must no longer resolve.
	if _, err := repo.GetStaffAccount(context.Background(), 42, created.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected staff binding removed, got %v", err)
	}
}

func TestStoreDisableStaffAccountInvalidatesAccount(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	view, err := svc.StoreDisableStaffAccount(context.Background(), 42, created.ID)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if view.Status != "disabled" {
		t.Fatalf("expected disabled status, got %+v", view)
	}
}

func TestSearchMembersByPhone(t *testing.T) {
	repo := staffMemberRepo() // member 7, phone 13800000007
	svc := NewService(repo, fakeStores{}, nil)

	// Tail-number (suffix) fuzzy match hits the member.
	views, err := svc.SearchMembersByPhone(context.Background(), "0007")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(views) != 1 || views[0].ID != 7 || views[0].Nickname != "Waiter Wang" ||
		views[0].Phone != "13800000007" || views[0].AvatarURL != "https://cdn.test/staff.webp" {
		t.Fatalf("unexpected results: %+v", views)
	}
	// Too-short fragment is rejected.
	if _, err := svc.SearchMembersByPhone(context.Background(), "00"); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for <3 chars, got %v", err)
	}
	if _, err := svc.SearchMembersByPhone(context.Background(), "%%%"); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for wildcard input, got %v", err)
	}
	if _, err := svc.SearchMembersByPhone(context.Background(), "138000000071"); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for >11 digits, got %v", err)
	}
	// No match returns an empty list (not an error).
	empty, err := svc.SearchMembersByPhone(context.Background(), "999")
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty result, got %+v err=%v", empty, err)
	}
}

func TestAdminCreateStaffAccountUsesRequestedStore(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	view, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42, MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "Waiter Wang" || view.StoreID != 42 || view.MemberID != 7 {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestAdminCreateStaffAccountRejectsMissingStoreOrMember(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{MemberID: 7}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing storeId, got %v", err)
	}
	if _, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing memberId, got %v", err)
	}
}

func TestAdminUpdateStaffAccountCanReassignStore(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42, MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "Waiter Wang Jr"
	newStore := int64(99)
	view, err := svc.AdminUpdateStaffAccount(context.Background(), created.ID, AdminStaffAccountUpdateRequest{Name: &newName, StoreID: &newStore})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if view.Name != newName || view.StoreID != newStore {
		t.Fatalf("unexpected view after reassignment: %+v", view)
	}
}

func TestAdminUpdateStaffAccountRejectsEmptyBody(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.AdminUpdateStaffAccount(context.Background(), 1, AdminStaffAccountUpdateRequest{}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for empty update, got %v", err)
	}
}

func TestAdminDisableStaffAccountWorksAcrossStores(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	view, err := svc.AdminDisableStaffAccount(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if view.Status != "disabled" {
		t.Fatalf("expected disabled status, got %+v", view)
	}
}

func TestAdminDeleteStaffAccountRemovesBinding(t *testing.T) {
	repo := staffMemberRepo()
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{MemberID: 7})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.AdminDeleteStaffAccount(context.Background(), created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetStaffAccountByID(context.Background(), created.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected staff binding removed, got %v", err)
	}
}

// fakeWallets is a stub WalletProvider for GetMemberDetail/CreateWalletAdjustment tests.
type fakeWallets struct {
	accounts []wallet.Account
	adjusted wallet.Account
	err      error

	lastMemberID int64
	lastStoreID  int64
	lastReq      wallet.AdjustmentRequest
	lastIdemKey  string
	lastAudit    audit.Entry
}

func (f *fakeWallets) GetWallet(_ context.Context, memberID int64) ([]wallet.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts, nil
}

func (f *fakeWallets) AdjustBalance(_ context.Context, memberID, storeID int64, req wallet.AdjustmentRequest, idemKey string, auditEntry audit.Entry) (wallet.Account, error) {
	f.lastMemberID, f.lastStoreID, f.lastReq, f.lastIdemKey = memberID, storeID, req, idemKey
	f.lastAudit = auditEntry
	if f.err != nil {
		return wallet.Account{}, f.err
	}
	return f.adjusted, nil
}

func (f *fakeWallets) AdjustBalanceForAdmin(_ context.Context, memberID int64, req wallet.AdjustmentRequest, idemKey string, auditEntry audit.Entry) (wallet.Account, error) {
	f.lastMemberID, f.lastReq, f.lastIdemKey = memberID, req, idemKey
	f.lastAudit = auditEntry
	if f.err != nil {
		return wallet.Account{}, f.err
	}
	return f.adjusted, nil
}

func TestGetMemberDetailNotScopedToStore(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {
		ID: 7, Nickname: "n", Phone: "13800000007", AvatarURL: "https://cdn.test/avatar.webp",
		PointsBalance: 100, CoinsBalance: 20, VIPTierName: "黄金会员", VIPLevel: 3,
		Status: StatusActive,
	}}}
	wallets := &fakeWallets{accounts: []wallet.Account{{AssetType: wallet.AssetPoints, AvailableAmount: 100}}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.GetMemberDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ID != 7 || view.AvatarURL != "https://cdn.test/avatar.webp" || view.CoinsBalance != 20 ||
		view.VIPLevel != 3 || view.VIPTierName != "黄金会员" || len(view.Wallet) != 1 ||
		view.Wallet[0].AvailableAmount != 100 {
		t.Fatalf("unexpected view: %+v", view)
	}

	if _, err := svc.GetMemberDetail(context.Background(), 999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for missing member, got %v", err)
	}
}

func TestAdminCreateWalletAdjustmentDelegatesUnscoped(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.AdminCreateWalletAdjustment(context.Background(), 7, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50, Reason: "goodwill",
	}, "idem-1", audit.Entry{ActorID: 9})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.BalanceAfter != 150 || wallets.lastMemberID != 7 || wallets.lastIdemKey != "idem-1" {
		t.Fatalf("unexpected adjustment call/result: %+v wallets=%+v", view, wallets)
	}

	if _, err := svc.AdminCreateWalletAdjustment(context.Background(), 999, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50,
	}, "idem-2", audit.Entry{ActorID: 9}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for missing member, got %v", err)
	}
}

func TestCreateWalletAdjustmentAttributesStoreAndAuditActor(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.CreateWalletAdjustment(context.Background(), 42, 7, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50, Reason: "goodwill",
	}, "idem-1", audit.Entry{ActorID: 88, StoreID: 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.BalanceAfter != 150 || wallets.lastMemberID != 7 || wallets.lastStoreID != 42 ||
		wallets.lastIdemKey != "idem-1" || wallets.lastAudit.ActorID != 88 || wallets.lastAudit.StoreID != 42 {
		t.Fatalf("unexpected adjustment call/result: %+v wallets=%+v", view, wallets)
	}

	// A missing platform member is rejected before reaching the wallet.
	if _, err := svc.CreateWalletAdjustment(context.Background(), 42, 999, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50,
	}, "idem-2", audit.Entry{ActorID: 88, StoreID: 42}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for missing member, got %v", err)
	}
}
