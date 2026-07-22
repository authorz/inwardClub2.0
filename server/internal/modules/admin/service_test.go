package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/inwardclub/server/internal/modules/wallet"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeRepo returns fixed pages and records the filter it was called with so the
// tests can assert both mapping and that the store scope is propagated.
type fakeRepo struct {
	lastFilter ListFilter
	err        error

	cashiers      map[int64]AdminAccount
	staffAccounts map[int64]StaffAccount
	adminAccounts map[int64]AdminAccount
	nextID        int64

	members map[int64]Member
}

func (f *fakeRepo) ListStores(_ context.Context, flt ListFilter) ([]StoreSummary, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	return []StoreSummary{{ID: 1, Name: "HQ", Status: StatusActive}}, 3, nil
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
	return []Activity{{ID: 30, Name: "Spring"}}, 1, nil
}
func (f *fakeRepo) ListOrders(_ context.Context, flt ListFilter) ([]Order, int64, error) {
	f.lastFilter = flt
	return []Order{{
		ID: 40, OrderNo: "NO-1", OrderType: "food", StoreName: "三里屯店",
		MemberNickname: "Sam", MemberPhone: "13800001234", TotalCent: 999,
		PayChannel: "wechat", PaymentStatus: "paid", OrderStatus: "completed",
	}}, 1, nil
}
func (f *fakeRepo) ListMembers(_ context.Context, flt ListFilter) ([]Member, int64, error) {
	f.lastFilter = flt
	return []Member{{ID: 50, Nickname: "Sam"}}, 1, nil
}
func (f *fakeRepo) GetMember(_ context.Context, storeID, memberID int64) (Member, error) {
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

func (f *fakeRepo) ListWalletLedger(_ context.Context, flt ListFilter) ([]WalletLedgerEntry, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	sourceID := int64(999)
	return []WalletLedgerEntry{{
		ID: 100, MemberID: 7, AssetType: "points", Direction: "credit", Amount: 50,
		BalanceAfter: 150, Reason: "sign_in", SourceType: "sign_in", SourceID: &sourceID,
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
	return []Refund{{ID: 85, RefundOrderNo: "RF-1", PaymentOrderID: 80, AmountCent: 500, Status: "pending"}}, 1, nil
}
func (f *fakeRepo) ListAuditLogs(_ context.Context, flt ListFilter) ([]AuditLog, int64, error) {
	f.lastFilter = flt
	return []AuditLog{{ID: 60, Action: "login"}}, 1, nil
}
func (f *fakeRepo) ListRuleDefinitions(_ context.Context, flt ListFilter) ([]RuleDefinition, int64, error) {
	f.lastFilter = flt
	return []RuleDefinition{{ID: 70, Key: "sign_in", Enabled: true, Status: "published"}}, 1, nil
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
	return []AdminAccount{{ID: 93, Username: "root", DisplayName: "Root Admin", Role: "super_admin"}}, 1, nil
}
func (f *fakeRepo) ListStoreAdmins(_ context.Context, flt ListFilter) ([]AdminAccount, int64, error) {
	f.lastFilter = flt
	if f.err != nil {
		return nil, 0, f.err
	}
	storeID := int64(42)
	return []AdminAccount{{ID: 94, Username: "store_admin1", DisplayName: "Store Admin One", Role: "store_admin", StoreID: &storeID}}, 1, nil
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
	f.nextID++
	id := f.nextID
	a := AdminAccount{ID: id, Username: username, DisplayName: displayName, Role: "store_admin", StoreID: &storeID, Status: StatusActive}
	f.adminAccounts[id] = a
	return a, nil
}
func (f *fakeRepo) UpdateStoreAdminByID(_ context.Context, id int64, storeID *int64, displayName *string) (AdminAccount, error) {
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

func (f *fakeRepo) CreateStaffAccount(_ context.Context, storeID int64, name string) (StaffAccount, error) {
	if f.err != nil {
		return StaffAccount{}, f.err
	}
	if f.staffAccounts == nil {
		f.staffAccounts = map[int64]StaffAccount{}
	}
	f.nextID++
	id := f.nextID
	a := StaffAccount{ID: id, Name: name, StoreID: storeID, Status: StatusActive}
	f.staffAccounts[id] = a
	return a, nil
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
		v.OrderStatus != "completed" || v.Status != "completed" {
		t.Fatalf("unexpected order-center fields: %+v", v)
	}
	// memberPhone is exposed raw for HQ; memberPhoneMasked hides the middle digits.
	if v.MemberPhone != "13800001234" || v.MemberPhoneMasked != "138****1234" || v.MemberNickname != "Sam" {
		t.Fatalf("unexpected member fields: %+v", v)
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
	if len(views) != 1 || views[0].RefundOrderNo != "RF-1" {
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
	if len(views) != 1 || views[0].MemberID != 7 || views[0].AssetType != "points" || views[0].BalanceAfter != 150 {
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
	if len(views) != 1 || views[0].Role != "super_admin" {
		t.Fatalf("unexpected views: %+v", views)
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

func TestCreateStoreAdminGeneratesPasswordAndScopesToStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)

	v, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{StoreID: 42, Username: "sa1", DisplayName: "SA One"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.InitialPassword == "" || v.StoreID == nil || *v.StoreID != 42 || v.Role != "store_admin" {
		t.Fatalf("unexpected view: %+v", v)
	}
}

func TestCreateStoreAdminRejectsMissingStoreOrUsername(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{Username: "sa1"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing storeId, got %v", err)
	}
	if _, err := svc.CreateStoreAdmin(context.Background(), StoreAdminCreateRequest{StoreID: 42}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing username, got %v", err)
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

func TestUpdateStoreAdminRejectsEmptyBody(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.UpdateStoreAdmin(context.Background(), 1, StoreAdminUpdateRequest{}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
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

func TestStoreCreateStaffAccountScopesToStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	view, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{Name: "Waiter Wang"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "Waiter Wang" || view.StoreID != 42 {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestStoreCreateStaffAccountRejectsEmptyName(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	_, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{Name: " "})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestStoreUpdateStaffAccountCannotReachOtherStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{Name: "Waiter Wang"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "Someone Else"
	_, err = svc.StoreUpdateStaffAccount(context.Background(), 99, created.ID, StaffAccountUpdateRequest{Name: &newName})
	if apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store update, got %v", err)
	}
}

func TestStoreDisableStaffAccountInvalidatesAccount(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{Name: "Waiter Wang"})
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

func TestAdminCreateStaffAccountUsesRequestedStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	view, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42, Name: "Waiter Wang"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "Waiter Wang" || view.StoreID != 42 {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestAdminCreateStaffAccountRejectsMissingStoreOrName(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	if _, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{Name: "Waiter Wang"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing storeId, got %v", err)
	}
	if _, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42, Name: " "}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing name, got %v", err)
	}
}

func TestAdminUpdateStaffAccountCanReassignStore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.AdminCreateStaffAccount(context.Background(), AdminStaffAccountCreateRequest{StoreID: 42, Name: "Waiter Wang"})
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
	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	created, err := svc.StoreCreateStaffAccount(context.Background(), 42, StaffAccountCreateRequest{Name: "Waiter Wang"})
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

// fakeWallets is a stub WalletProvider for GetMemberDetail/CreateWalletAdjustment tests.
type fakeWallets struct {
	accounts []wallet.Account
	adjusted wallet.Account
	err      error

	lastMemberID int64
	lastStoreID  int64
	lastReq      wallet.AdjustmentRequest
	lastIdemKey  string
}

func (f *fakeWallets) GetWallet(_ context.Context, memberID int64) ([]wallet.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts, nil
}

func (f *fakeWallets) AdjustBalance(_ context.Context, memberID, storeID int64, req wallet.AdjustmentRequest, idemKey string) (wallet.Account, error) {
	f.lastMemberID, f.lastStoreID, f.lastReq, f.lastIdemKey = memberID, storeID, req, idemKey
	if f.err != nil {
		return wallet.Account{}, f.err
	}
	return f.adjusted, nil
}

func (f *fakeWallets) AdjustBalanceForAdmin(_ context.Context, memberID int64, req wallet.AdjustmentRequest, idemKey string) (wallet.Account, error) {
	f.lastMemberID, f.lastReq, f.lastIdemKey = memberID, req, idemKey
	if f.err != nil {
		return wallet.Account{}, f.err
	}
	return f.adjusted, nil
}

func TestAdminGetMemberDetailNotScopedToStore(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{accounts: []wallet.Account{{AssetType: wallet.AssetPoints, AvailableAmount: 100}}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.AdminGetMemberDetail(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ID != 7 || len(view.Wallet) != 1 || view.Wallet[0].AvailableAmount != 100 {
		t.Fatalf("unexpected view: %+v", view)
	}

	if _, err := svc.AdminGetMemberDetail(context.Background(), 999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for missing member, got %v", err)
	}
}

func TestAdminCreateWalletAdjustmentDelegatesUnscoped(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.AdminCreateWalletAdjustment(context.Background(), 7, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50, Reason: "goodwill",
	}, "idem-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.BalanceAfter != 150 || wallets.lastMemberID != 7 || wallets.lastIdemKey != "idem-1" {
		t.Fatalf("unexpected adjustment call/result: %+v wallets=%+v", view, wallets)
	}

	if _, err := svc.AdminCreateWalletAdjustment(context.Background(), 999, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50,
	}, "idem-2"); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for missing member, got %v", err)
	}
}

func TestGetMemberDetailScopedToStore(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{accounts: []wallet.Account{{AssetType: wallet.AssetPoints, AvailableAmount: 100}}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.GetMemberDetail(context.Background(), 42, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ID != 7 || len(view.Wallet) != 1 || view.Wallet[0].AvailableAmount != 100 {
		t.Fatalf("unexpected view: %+v", view)
	}

	if _, err := svc.GetMemberDetail(context.Background(), 42, 999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for member outside store scope, got %v", err)
	}
}

func TestCreateWalletAdjustmentScopedAndDelegates(t *testing.T) {
	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)

	view, err := svc.CreateWalletAdjustment(context.Background(), 42, 7, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50, Reason: "goodwill",
	}, "idem-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.BalanceAfter != 150 || wallets.lastMemberID != 7 || wallets.lastStoreID != 42 || wallets.lastIdemKey != "idem-1" {
		t.Fatalf("unexpected adjustment call/result: %+v wallets=%+v", view, wallets)
	}

	// Member outside store scope must be rejected before reaching the wallet.
	if _, err := svc.CreateWalletAdjustment(context.Background(), 42, 999, WalletAdjustmentRequest{
		AssetType: wallet.AssetPoints, Direction: wallet.DirectionCredit, Amount: 50,
	}, "idem-2"); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for member outside store scope, got %v", err)
	}
}
