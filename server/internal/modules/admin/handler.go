package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// Handler exposes the headquarters console read endpoints, plus the store-scoped
// read variants that back the single-store console. Router wiring lives outside
// this module; the admin methods mount under the admin audience and the Store*
// methods under the store audience (after storescope.Inject).
type Handler struct {
	svc *Service
}

// NewHandler builds the console handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// parseFilter reads pagination plus the optional status/keyword refinements. The
// scope (storeId) is never read from the request: admin endpoints leave it nil,
// store endpoints pin it from the JWT scope.
func parseFilter(c *gin.Context) ListFilter {
	return ListFilter{
		Page:    httpx.ParsePage(c),
		Status:  c.Query("status"),
		Keyword: c.Query("keyword"),
	}
}

// --- Admin console (audience: admin, no store filter) ---

// Stores handles GET /admin/stores.
func (h *Handler) Stores(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListStores(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// CatalogItems handles GET /admin/catalog/items.
func (h *Handler) CatalogItems(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListCatalogItems(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// CouponTemplates handles GET /admin/coupon-templates.
func (h *Handler) CouponTemplates(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListCouponTemplates(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Activities handles GET /admin/activities.
func (h *Handler) Activities(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListActivities(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Orders handles GET /admin/orders.
func (h *Handler) Orders(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListOrders(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Members handles GET /admin/members.
func (h *Handler) Members(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListMembers(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// MemberDetail handles GET /admin/members/:memberID (not scoped to a store).
func (h *Handler) MemberDetail(c *gin.Context) {
	id, err := pathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.AdminGetMemberDetail(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CreateWalletAdjustment handles POST /admin/members/:memberID/wallet-adjustments.
func (h *Handler) CreateWalletAdjustment(c *gin.Context) {
	id, err := pathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req WalletAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid wallet adjustment body"))
		return
	}
	view, err := h.svc.AdminCreateWalletAdjustment(c.Request.Context(), id, req, idempotency.Key(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// parseWalletLedgerFilter reads pagination plus the optional memberId/assetType
// refinements shared by the admin and store wallet-ledger reads. storeId is
// handled by the caller: the admin endpoint accepts an optional query param,
// the store endpoint pins it from the JWT scope instead.
func parseWalletLedgerFilter(c *gin.Context) (ListFilter, error) {
	f := ListFilter{Page: httpx.ParsePage(c), AssetType: c.Query("assetType")}
	if raw := c.Query("memberId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return ListFilter{}, apperr.Invalid("invalid memberId")
		}
		f.MemberID = &id
	}
	return f, nil
}

// WalletLedger handles GET /admin/wallet-ledger. Optional ?memberId, ?assetType
// and ?storeId query params narrow the result.
func (h *Handler) WalletLedger(c *gin.Context) {
	f, err := parseWalletLedgerFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListWalletLedger(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// PaymentTransactions handles GET /admin/payment-transactions. An optional
// ?storeId filter narrows the result to a single store.
func (h *Handler) PaymentTransactions(c *gin.Context) {
	f := parseFilter(c)
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListPaymentTransactions(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Refunds handles GET /admin/refunds. An optional ?storeId filter narrows the
// result to a single store.
func (h *Handler) Refunds(c *gin.Context) {
	f := parseFilter(c)
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListRefunds(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// AuditLogs handles GET /admin/audit-logs.
func (h *Handler) AuditLogs(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListAuditLogs(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// RuleDefinitions handles GET /admin/rule-definitions.
func (h *Handler) RuleDefinitions(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListRuleDefinitions(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// UpdateRuleDefinition handles PATCH /admin/rule-definitions/:ruleID.
func (h *Handler) UpdateRuleDefinition(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("ruleID"), 10, 64)
	if err != nil || ruleID <= 0 {
		httpx.Fail(c, apperr.Invalid("admin: invalid ruleID"))
		return
	}
	var req RuleDefinitionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid rule update body"))
		return
	}
	view, err := h.svc.UpdateRuleDefinition(c.Request.Context(), ruleID, RuleDefinitionUpdate{
		ConfigJSON: req.ConfigJSON,
		Enabled:    req.Enabled,
		Status:     req.Status,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CreateRuleDefinition handles POST /admin/rule-definitions.
func (h *Handler) CreateRuleDefinition(c *gin.Context) {
	var req RuleDefinitionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid rule create body"))
		return
	}
	view, err := h.svc.CreateRuleDefinition(c.Request.Context(), RuleDefinitionCreate{
		Key:        req.RuleKey,
		ScopeType:  req.ScopeType,
		StoreID:    req.StoreID,
		Version:    req.Version,
		ConfigJSON: req.ConfigJSON,
		Enabled:    req.Enabled,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// DisableRuleDefinition handles POST /admin/rule-definitions/:ruleID/disable.
func (h *Handler) DisableRuleDefinition(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("ruleID"), 10, 64)
	if err != nil || ruleID <= 0 {
		httpx.Fail(c, apperr.Invalid("admin: invalid ruleID"))
		return
	}
	view, err := h.svc.DisableRuleDefinition(c.Request.Context(), ruleID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// PublishRuleDefinition handles POST /admin/rule-definitions/:ruleID/publish.
func (h *Handler) PublishRuleDefinition(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("ruleID"), 10, 64)
	if err != nil || ruleID <= 0 {
		httpx.Fail(c, apperr.Invalid("admin: invalid ruleID"))
		return
	}
	view, err := h.svc.PublishRuleDefinition(c.Request.Context(), ruleID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StaffAccounts handles GET /admin/staff-accounts. An optional ?storeId filter
// narrows the result to a single store.
func (h *Handler) StaffAccounts(c *gin.Context) {
	f := parseFilter(c)
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListStaffAccounts(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// AdminCreateStaffAccount handles POST /admin/staff-accounts.
func (h *Handler) AdminCreateStaffAccount(c *gin.Context) {
	var req AdminStaffAccountCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid staff account create body"))
		return
	}
	view, err := h.svc.AdminCreateStaffAccount(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// AdminUpdateStaffAccount handles PATCH /admin/staff-accounts/:staffID.
func (h *Handler) AdminUpdateStaffAccount(c *gin.Context) {
	id, err := pathID(c, "staffID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req AdminStaffAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid staff account update body"))
		return
	}
	view, err := h.svc.AdminUpdateStaffAccount(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminDisableStaffAccount handles POST /admin/staff-accounts/:staffID/disable.
func (h *Handler) AdminDisableStaffAccount(c *gin.Context) {
	id, err := pathID(c, "staffID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.AdminDisableStaffAccount(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminAccounts handles GET /admin/admin-accounts, listing headquarters
// super_admin login accounts.
func (h *Handler) AdminAccounts(c *gin.Context) {
	f := parseFilter(c)
	views, total, err := h.svc.ListSuperAdmins(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// DisableAdminAccount handles POST /admin/admin-accounts/:accountID/disable.
func (h *Handler) DisableAdminAccount(c *gin.Context) {
	id, err := pathID(c, "accountID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.DisableAdminAccount(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreAdminAccounts handles GET /admin/store-admin-accounts. An optional
// ?storeId filter narrows the result to a single store.
func (h *Handler) StoreAdminAccounts(c *gin.Context) {
	f := parseFilter(c)
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListStoreAdmins(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// CreateStoreAdminAccount handles POST /admin/store-admin-accounts.
func (h *Handler) CreateStoreAdminAccount(c *gin.Context) {
	var req StoreAdminCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid store admin create body"))
		return
	}
	view, err := h.svc.CreateStoreAdmin(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// UpdateStoreAdminAccount handles PATCH /admin/store-admin-accounts/:accountID.
func (h *Handler) UpdateStoreAdminAccount(c *gin.Context) {
	id, err := pathID(c, "accountID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req StoreAdminUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid store admin update body"))
		return
	}
	view, err := h.svc.UpdateStoreAdmin(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// DisableStoreAdminAccount handles POST /admin/store-admin-accounts/:accountID/disable.
func (h *Handler) DisableStoreAdminAccount(c *gin.Context) {
	id, err := pathID(c, "accountID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.DisableStoreAdmin(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// --- Store console (audience: store, scope pinned from JWT) ---

// StoreProfile handles GET /store/profile.
func (h *Handler) StoreProfile(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	view, err := h.svc.GetStoreProfile(c.Request.Context(), scope)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreCatalogItems handles GET /store/catalog/items.
func (h *Handler) StoreCatalogItems(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListCatalogItems(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreCouponTemplates handles GET /store/coupon-templates.
func (h *Handler) StoreCouponTemplates(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListCouponTemplates(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreActivities handles GET /store/activities.
func (h *Handler) StoreActivities(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListActivities(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreOrders handles GET /store/orders.
func (h *Handler) StoreOrders(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListOrders(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreRefunds handles GET /store/refunds (and /store/refund-orders), scoped
// strictly to the caller's store.
func (h *Handler) StoreRefunds(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListRefunds(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StorePaymentTransactions handles GET /store/payment-transactions.
func (h *Handler) StorePaymentTransactions(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListPaymentTransactions(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreMembers handles GET /store/members.
func (h *Handler) StoreMembers(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListMembers(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreWalletLedger handles GET /store/wallet-ledger, strictly scoped to the
// caller's own store. Optional ?memberId and ?assetType query params narrow
// the result further; a client-supplied storeId is never honored.
func (h *Handler) StoreWalletLedger(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f, err := parseWalletLedgerFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	f = scopedFilter(f, scope)
	views, total, err := h.svc.ListWalletLedger(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreMemberDetail handles GET /store/members/:memberID.
func (h *Handler) StoreMemberDetail(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetMemberDetail(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreCreateWalletAdjustment handles POST /store/members/:memberID/wallet-adjustments.
func (h *Handler) StoreCreateWalletAdjustment(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req WalletAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid wallet adjustment body"))
		return
	}
	view, err := h.svc.CreateWalletAdjustment(c.Request.Context(), scope, id, req, idempotency.Key(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// StoreCashiers handles GET /store/cashiers.
func (h *Handler) StoreCashiers(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListCashiers(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// StoreStaffAccounts handles GET /store/staff-accounts.
func (h *Handler) StoreStaffAccounts(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	f := scopedFilter(parseFilter(c), scope)
	views, total, err := h.svc.ListStaffAccounts(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// pathID parses a positive int64 path parameter.
func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("admin: invalid " + name)
	}
	return id, nil
}

// StoreCreateCashier handles POST /store/cashiers.
func (h *Handler) StoreCreateCashier(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var req CashierCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid cashier create body"))
		return
	}
	view, err := h.svc.StoreCreateCashier(c.Request.Context(), scope, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// StoreUpdateCashier handles PATCH /store/cashiers/:cashierID.
func (h *Handler) StoreUpdateCashier(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "cashierID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req CashierUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid cashier update body"))
		return
	}
	view, err := h.svc.StoreUpdateCashier(c.Request.Context(), scope, id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreDisableCashier handles POST /store/cashiers/:cashierID/disable.
func (h *Handler) StoreDisableCashier(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "cashierID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.StoreDisableCashier(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreResetCashierPassword handles POST /store/cashiers/:cashierID/password-reset.
func (h *Handler) StoreResetCashierPassword(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "cashierID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.StoreResetCashierPassword(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreCreateStaffAccount handles POST /store/staff-accounts.
func (h *Handler) StoreCreateStaffAccount(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var req StaffAccountCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid staff account create body"))
		return
	}
	view, err := h.svc.StoreCreateStaffAccount(c.Request.Context(), scope, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// StoreUpdateStaffAccount handles PATCH /store/staff-accounts/:staffID.
func (h *Handler) StoreUpdateStaffAccount(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "staffID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req StaffAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("admin: invalid staff account update body"))
		return
	}
	view, err := h.svc.StoreUpdateStaffAccount(c.Request.Context(), scope, id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreDisableStaffAccount handles POST /store/staff-accounts/:staffID/disable.
func (h *Handler) StoreDisableStaffAccount(c *gin.Context) {
	scope, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "staffID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.StoreDisableStaffAccount(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}
