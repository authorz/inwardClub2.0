package admin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/modules/referral"
	"github.com/inwardclub/server/internal/modules/wallet"
	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// StoreProfileProvider returns a single store's profile. It is implemented by an
// adapter over the store module's read service so the store console can fetch
// its own profile without the admin module owning store SQL. Injected at wiring
// time; may be nil when the store-console surface is not mounted.
type StoreProfileProvider interface {
	StoreProfile(ctx context.Context, storeID int64) (StoreProfileView, error)
}

// WalletProvider is implemented by the wallet module's service so the store
// console can read a member's wallet balances and apply admin adjustments
// without this module owning wallet SQL.
type WalletProvider interface {
	GetWallet(ctx context.Context, memberID int64) ([]wallet.Account, error)
	AdjustBalance(ctx context.Context, memberID, storeID int64, req wallet.AdjustmentRequest, idemKey string, auditEntry audit.Entry) (wallet.Account, error)
	// AdjustBalanceForAdmin applies a headquarters-initiated adjustment, not
	// scoped to a single store.
	AdjustBalanceForAdmin(ctx context.Context, memberID int64, req wallet.AdjustmentRequest, idemKey string, auditEntry audit.Entry) (wallet.Account, error)
}

// AssetResolver resolves an asset id to a public URL for console list images.
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// Service provides the headquarters/store console read operations. Each list
// method maps a repository page onto the console view and returns the total for
// pagination meta.
type Service struct {
	repo    Repository
	stores  StoreProfileProvider
	wallets WalletProvider
	assets  AssetResolver
}

// NewService builds the console read service.
func NewService(
	repo Repository,
	stores StoreProfileProvider,
	wallets WalletProvider,
	assets ...AssetResolver,
) *Service {
	var resolver AssetResolver
	if len(assets) > 0 {
		resolver = assets[0]
	}
	return &Service{repo: repo, stores: stores, wallets: wallets, assets: resolver}
}

// ListStores returns a page of store summaries.
func (s *Service) ListStores(ctx context.Context, f ListFilter) ([]StoreView, int64, error) {
	rows, total, err := s.repo.ListStores(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]StoreView, 0, len(rows))
	for _, r := range rows {
		out = append(out, StoreView{
			ID: r.ID, Name: r.Name, Phone: r.Phone, Address: r.Address,
			Latitude: r.Latitude, Longitude: r.Longitude,
			Status: r.Status, CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

// ListCatalogItems returns a page of catalog items.
func (s *Service) ListCatalogItems(ctx context.Context, f ListFilter) ([]CatalogItemView, int64, error) {
	rows, total, err := s.repo.ListCatalogItems(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CatalogItemView, 0, len(rows))
	for _, r := range rows {
		out = append(out, CatalogItemView{
			ID: r.ID, ScopeType: r.ScopeType, StoreID: r.StoreID, CategoryID: r.CategoryID,
			Name: r.Name, ItemType: r.ItemType, PriceCent: r.PriceCent,
			StockQuantity: r.StockQuantity, Status: r.Status, CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

// ListCouponTemplates returns a page of coupon templates.
func (s *Service) ListCouponTemplates(ctx context.Context, f ListFilter) ([]CouponTemplateView, int64, error) {
	rows, total, err := s.repo.ListCouponTemplates(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CouponTemplateView, 0, len(rows))
	for _, r := range rows {
		out = append(out, CouponTemplateView{
			ID: r.ID, ScopeType: r.ScopeType, StoreID: r.StoreID, Name: r.Name,
			Description: r.Description, CouponType: r.CouponType, ValueCent: r.ValueCent,
			PointsPrice: r.PointsPrice, TotalStock: r.TotalStock, StockQuantity: r.TotalStock,
			IssuedCount: r.IssuedCount, IssuedQuantity: r.IssuedCount,
			PerMemberLimit: r.PerMemberLimit, Status: r.Status,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, total, nil
}

// ListActivities returns a page of activities.
func (s *Service) ListActivities(ctx context.Context, f ListFilter) ([]ActivityView, int64, error) {
	rows, total, err := s.repo.ListActivities(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ActivityView, 0, len(rows))
	for _, r := range rows {
		view := ActivityView{
			ID: r.ID, ScopeType: r.ScopeType, StoreID: r.StoreID, Name: r.Name, Title: r.Name,
			Type: r.Type, AssetID: r.AssetID, StartAt: r.StartAt, EndAt: r.EndAt,
			Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
		if s.assets != nil && r.AssetID != nil {
			view.ImageURL, _ = s.assets.PublicURLByID(ctx, *r.AssetID)
		}
		out = append(out, view)
	}
	return out, total, nil
}

// ListOrders returns a page of orders.
func (s *Service) ListOrders(ctx context.Context, f ListFilter) ([]OrderView, int64, error) {
	rows, total, err := s.repo.ListOrders(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]OrderView, 0, len(rows))
	for _, r := range rows {
		out = append(out, OrderView{
			ID: r.ID, BusinessOrderID: r.ID, OrderNo: r.OrderNo, OrderType: r.OrderType,
			StoreID: r.StoreID, StoreName: r.StoreName,
			MemberID: r.MemberID, MemberNickname: r.MemberNickname,
			MemberPhone: r.MemberPhone, MemberAvatarURL: r.MemberAvatarURL,
			MemberPhoneMasked: maskPhone(r.MemberPhone),
			PaymentOrderID:    r.PaymentOrderID,
			TotalCent:         r.TotalCent, AmountCent: r.TotalCent, PayChannel: r.PayChannel,
			RefundStatus: r.RefundStatus, PaymentStatus: r.PaymentStatus,
			OrderStatus: r.OrderStatus, Status: r.OrderStatus,
			CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
		})
	}
	return out, total, nil
}

// maskPhone masks the middle of a phone number, keeping the leading 3 and
// trailing 4 digits. Short or empty values are returned unchanged/empty.
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// ListMembers returns a page of members.
func (s *Service) ListMembers(ctx context.Context, f ListFilter) ([]MemberView, int64, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	f.SortBy = strings.TrimSpace(f.SortBy)
	f.SortOrder = strings.ToLower(strings.TrimSpace(f.SortOrder))
	switch f.SortBy {
	case "", "pointsBalance", "coinsBalance", "vipLevel":
	default:
		return nil, 0, apperr.Invalid("admin: invalid member sortBy")
	}
	switch f.SortOrder {
	case "", "asc", "desc":
	default:
		return nil, 0, apperr.Invalid("admin: invalid member sortOrder")
	}
	rows, total, err := s.repo.ListMembers(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]MemberView, 0, len(rows))
	for _, r := range rows {
		out = append(out, MemberView{
			ID: r.ID, Nickname: r.Nickname, Phone: r.Phone,
			AvatarURL: r.AvatarURL, Gender: r.Gender,
			PointsBalance: r.PointsBalance, CoinsBalance: r.CoinsBalance,
			VIPTierName: r.VIPTierName, VIPLevel: r.VIPLevel,
			Status: r.Status, CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

// CreateWalletAdjustment applies a store-attributed wallet adjustment to a
// platform-wide member.
func (s *Service) CreateWalletAdjustment(ctx context.Context, storeID, memberID int64, req WalletAdjustmentRequest, idemKey string, auditEntry audit.Entry) (WalletAdjustmentView, error) {
	if s.wallets == nil {
		return WalletAdjustmentView{}, apperr.NotImplemented("admin: wallet provider not wired")
	}
	// Members are platform-wide, so a store may adjust any visible member. The
	// store attribution still comes from the authenticated token and is written
	// to both the wallet ledger and audit log.
	if _, err := s.repo.GetMemberByID(ctx, memberID); err != nil {
		return WalletAdjustmentView{}, err
	}
	auditEntry.StoreID = storeID
	acc, err := s.wallets.AdjustBalance(ctx, memberID, storeID, wallet.AdjustmentRequest{
		AssetType: req.AssetType, Direction: req.Direction, Amount: req.Amount, Reason: req.Reason,
	}, idemKey, auditEntry)
	if err != nil {
		return WalletAdjustmentView{}, err
	}
	return WalletAdjustmentView{
		AssetType: acc.AssetType, Direction: req.Direction, Amount: req.Amount,
		BalanceAfter: acc.AvailableAmount, Reason: req.Reason,
	}, nil
}

// GetMemberDetail returns a platform-wide member plus wallet balances. Members
// have no registration-store ownership, so both consoles share this read path.
func (s *Service) GetMemberDetail(ctx context.Context, memberID int64) (MemberDetailView, error) {
	m, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return MemberDetailView{}, err
	}
	view := MemberDetailView{MemberView: MemberView{
		ID: m.ID, Nickname: m.Nickname, Phone: m.Phone,
		AvatarURL: m.AvatarURL, Gender: m.Gender,
		PointsBalance: m.PointsBalance, CoinsBalance: m.CoinsBalance,
		VIPTierName: m.VIPTierName, VIPLevel: m.VIPLevel,
		Status: m.Status, CreatedAt: m.CreatedAt,
	}}
	if s.wallets != nil {
		accounts, err := s.wallets.GetWallet(ctx, memberID)
		if err != nil {
			return MemberDetailView{}, err
		}
		view.Wallet = accounts
	}
	return view, nil
}

// AdminCreateWalletAdjustment applies a headquarters-initiated wallet
// adjustment to any member, not scoped to a store.
func (s *Service) AdminCreateWalletAdjustment(ctx context.Context, memberID int64, req WalletAdjustmentRequest, idemKey string, auditEntry audit.Entry) (WalletAdjustmentView, error) {
	if s.wallets == nil {
		return WalletAdjustmentView{}, apperr.NotImplemented("admin: wallet provider not wired")
	}
	if _, err := s.repo.GetMemberByID(ctx, memberID); err != nil {
		return WalletAdjustmentView{}, err
	}
	auditEntry.StoreID = 0
	acc, err := s.wallets.AdjustBalanceForAdmin(ctx, memberID, wallet.AdjustmentRequest{
		AssetType: req.AssetType, Direction: req.Direction, Amount: req.Amount, Reason: req.Reason,
	}, idemKey, auditEntry)
	if err != nil {
		return WalletAdjustmentView{}, err
	}
	return WalletAdjustmentView{
		AssetType: acc.AssetType, Direction: req.Direction, Amount: req.Amount,
		BalanceAfter: acc.AvailableAmount, Reason: req.Reason,
	}, nil
}

// ListWalletLedger returns a page of wallet ledger entries.
func (s *Service) ListWalletLedger(ctx context.Context, f ListFilter) ([]WalletLedgerEntryView, int64, error) {
	rows, total, err := s.repo.ListWalletLedger(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]WalletLedgerEntryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, WalletLedgerEntryView{
			ID: r.ID, RecordKey: r.RecordKey,
			MemberID: r.MemberID, MemberNickname: r.MemberNickname,
			MemberPhone: r.MemberPhone, MemberAvatarURL: r.MemberAvatarURL,
			StoreID: r.StoreID, StoreName: r.StoreName,
			AssetType: r.AssetType, Direction: r.Direction,
			Amount: r.Amount, BalanceAfter: r.BalanceAfter, Status: r.Status, Reason: r.Reason,
			SourceType: r.SourceType, SourceID: r.SourceID,
			RelatedOrderNo: r.RelatedOrderNo, CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

// ListPaymentTransactions returns a page of payment orders.
func (s *Service) ListPaymentTransactions(ctx context.Context, f ListFilter) ([]PaymentTransactionView, int64, error) {
	rows, total, err := s.repo.ListPaymentTransactions(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PaymentTransactionView, 0, len(rows))
	for _, r := range rows {
		out = append(out, PaymentTransactionView{
			ID: r.ID, PaymentOrderNo: r.PaymentOrderNo, StoreID: r.StoreID, StoreName: r.StoreName,
			BusinessOrderID: r.BusinessOrderID, BusinessOrderNo: r.BusinessOrderNo, OrderType: r.OrderType,
			AmountCent: r.AmountCent, PayMethod: r.PayMethod, Status: r.Status,
			CreatedAt: r.CreatedAt, PaidAt: r.PaidAt,
		})
	}
	return out, total, nil
}

// ListRefunds returns a page of refund orders.
func (s *Service) ListRefunds(ctx context.Context, f ListFilter) ([]RefundView, int64, error) {
	rows, total, err := s.repo.ListRefunds(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]RefundView, 0, len(rows))
	for _, r := range rows {
		out = append(out, RefundView{
			ID: r.ID, RefundOrderNo: r.RefundOrderNo, PaymentOrderID: r.PaymentOrderID,
			BusinessOrderID: r.BusinessOrderID, StoreID: r.StoreID, StoreName: r.StoreName,
			BusinessOrderNo: r.BusinessOrderNo, OrderAmountCent: r.OrderAmountCent,
			MemberID: r.MemberID, MemberNickname: r.MemberNickname,
			MemberPhone: r.MemberPhone, MemberAvatarURL: r.MemberAvatarURL,
			AmountCent: r.AmountCent, Channel: r.Channel, Status: r.Status, Reason: r.Reason,
			OrderCreatedAt: r.OrderCreatedAt, OperatedAt: r.OperatedAt,
		})
	}
	return out, total, nil
}

// ListAuditLogs returns a page of audit-trail rows.
func (s *Service) ListAuditLogs(ctx context.Context, f ListFilter) ([]AuditLogView, int64, error) {
	rows, total, err := s.repo.ListAuditLogs(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AuditLogView, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditLogView{
			ID: r.ID, ActorType: r.ActorType, ActorID: r.ActorID, ActorRole: r.ActorRole, Action: r.Action,
			ActorSnapshot: r.ActorSnapshotJSON, TargetType: r.TargetType, TargetID: r.TargetID,
			TargetSnapshot: r.TargetSnapshotJSON, StoreID: r.StoreID, ScopeSnapshot: r.ScopeSnapshotJSON,
			Before: r.BeforeJSON, After: r.AfterJSON, Reason: r.Reason,
			RequestID: r.RequestID, CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

// ListRuleDefinitions returns a page of business rule definitions.
func (s *Service) ListRuleDefinitions(ctx context.Context, f ListFilter) ([]RuleDefinitionView, int64, error) {
	rows, total, err := s.repo.ListRuleDefinitions(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]RuleDefinitionView, 0, len(rows))
	for _, r := range rows {
		out = append(out, ruleView(r))
	}
	return out, total, nil
}

// UpdateRuleDefinition applies a partial update (config_json / enabled / status)
// to a single rule and returns the refreshed view.
func (s *Service) UpdateRuleDefinition(ctx context.Context, ruleID int64, req RuleDefinitionUpdate) (RuleDefinitionView, error) {
	if len(req.ConfigJSON) == 0 && req.Enabled == nil && req.Status == nil {
		return RuleDefinitionView{}, apperr.Invalid("admin: no rule fields to update")
	}
	if len(req.ConfigJSON) > 0 && !json.Valid(req.ConfigJSON) {
		return RuleDefinitionView{}, apperr.Invalid("admin: configJson is not valid JSON")
	}
	if len(req.ConfigJSON) > 0 {
		current, err := s.repo.GetRuleDefinition(ctx, ruleID)
		if err != nil {
			return RuleDefinitionView{}, err
		}
		if err := validateRuleConfig(current.Key, current.ScopeType, current.StoreID, req.ConfigJSON); err != nil {
			return RuleDefinitionView{}, err
		}
	}
	row, err := s.repo.UpdateRuleDefinition(ctx, ruleID, req)
	if err != nil {
		return RuleDefinitionView{}, err
	}
	return ruleView(row), nil
}

// CreateRuleDefinition inserts a new rule definition version. Rows always land
// in the DB-default 'draft' status, so a freshly created rule cannot take
// effect until explicitly published.
func (s *Service) CreateRuleDefinition(ctx context.Context, req RuleDefinitionCreate) (RuleDefinitionView, error) {
	if req.Key == "" {
		return RuleDefinitionView{}, apperr.Invalid("admin: ruleKey is required")
	}
	if len(req.ConfigJSON) == 0 || !json.Valid(req.ConfigJSON) {
		return RuleDefinitionView{}, apperr.Invalid("admin: configJson is not valid JSON")
	}
	if req.ScopeType == "" {
		req.ScopeType = "global"
	}
	if req.Version == 0 {
		req.Version = 1
	}
	if err := validateRuleConfig(req.Key, req.ScopeType, req.StoreID, req.ConfigJSON); err != nil {
		return RuleDefinitionView{}, err
	}
	row, err := s.repo.CreateRuleDefinition(ctx, req)
	if err != nil {
		return RuleDefinitionView{}, err
	}
	return ruleView(row), nil
}

// DisableRuleDefinition moves a rule definition to the 'disabled' status and
// clears its enabled flag so the rule engine stops applying it immediately.
func (s *Service) DisableRuleDefinition(ctx context.Context, ruleID int64) (RuleDefinitionView, error) {
	enabled := false
	status := "disabled"
	row, err := s.repo.UpdateRuleDefinition(ctx, ruleID, RuleDefinitionUpdate{Enabled: &enabled, Status: &status})
	if err != nil {
		return RuleDefinitionView{}, err
	}
	return ruleView(row), nil
}

// PublishRuleDefinition moves a rule definition to the 'published' status and
// sets its enabled flag so the rule engine starts applying it immediately.
func (s *Service) PublishRuleDefinition(ctx context.Context, ruleID int64) (RuleDefinitionView, error) {
	current, err := s.repo.GetRuleDefinition(ctx, ruleID)
	if err != nil {
		return RuleDefinitionView{}, err
	}
	if err := validateRuleConfig(current.Key, current.ScopeType, current.StoreID, current.ConfigJSON); err != nil {
		return RuleDefinitionView{}, err
	}
	enabled := true
	status := "published"
	row, err := s.repo.UpdateRuleDefinition(ctx, ruleID, RuleDefinitionUpdate{Enabled: &enabled, Status: &status})
	if err != nil {
		return RuleDefinitionView{}, err
	}
	return ruleView(row), nil
}

func validateRuleConfig(key, scopeType string, storeID *int64, raw json.RawMessage) error {
	if key != referral.RuleKey {
		return nil
	}
	if scopeType != "global" || storeID != nil {
		return apperr.Invalid("邀请奖励规则只能使用全局范围")
	}
	if _, err := referral.ParseConfig(raw); err != nil {
		return apperr.Invalid(err.Error())
	}
	return nil
}

// ruleView maps a rule read model onto its console representation.
func ruleView(r RuleDefinition) RuleDefinitionView {
	return RuleDefinitionView{
		ID: r.ID, Key: r.Key, ScopeType: r.ScopeType, StoreID: r.StoreID,
		Version: r.Version, ConfigJSON: r.ConfigJSON, Enabled: r.Enabled,
		Status: r.Status, UpdatedAt: r.UpdatedAt,
	}
}

// ListAdminAccounts returns a page of back-office login accounts (admin_accounts).
func (s *Service) ListAdminAccounts(ctx context.Context, f ListFilter) ([]AdminAccountView, int64, error) {
	rows, total, err := s.repo.ListAdminAccounts(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AdminAccountView, 0, len(rows))
	for _, r := range rows {
		out = append(out, adminAccountView(r))
	}
	return out, total, nil
}

// ListCashiers returns a page of cashier accounts (admin_accounts, role=cashier).
func (s *Service) ListCashiers(ctx context.Context, f ListFilter) ([]AdminAccountView, int64, error) {
	rows, total, err := s.repo.ListCashiers(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AdminAccountView, 0, len(rows))
	for _, r := range rows {
		out = append(out, adminAccountView(r))
	}
	return out, total, nil
}

// ListSuperAdmins returns a page of headquarters super_admin accounts.
func (s *Service) ListSuperAdmins(ctx context.Context, f ListFilter) ([]AdminAccountView, int64, error) {
	rows, total, err := s.repo.ListSuperAdmins(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AdminAccountView, 0, len(rows))
	for _, r := range rows {
		out = append(out, adminAccountView(r))
	}
	return out, total, nil
}

// CreateSuperAdmin creates a non-system headquarters administrator.
func (s *Service) CreateSuperAdmin(ctx context.Context, req AdminAccountCreateRequest) (AdminAccountView, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return AdminAccountView{}, apperr.Invalid("admin: username is required")
	}
	if req.Password == "" {
		return AdminAccountView{}, apperr.Invalid("admin: password is required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AdminAccountView{}, apperr.Internal(err)
	}
	row, err := s.repo.CreateSuperAdmin(ctx, username, string(hash), strings.TrimSpace(req.DisplayName))
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// UpdateSuperAdmin changes the display name and/or password. System
// administrators may be edited; their system status and username are immutable.
func (s *Service) UpdateSuperAdmin(ctx context.Context, id int64, req AdminAccountUpdateRequest) (AdminAccountView, error) {
	if req.DisplayName == nil && req.Password == nil {
		return AdminAccountView{}, apperr.Invalid("admin: no admin account fields to update")
	}
	row, err := s.repo.GetAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	if row.Role != "super_admin" {
		return AdminAccountView{}, apperr.NotFound("admin account not found")
	}
	var displayName *string
	if req.DisplayName != nil {
		value := strings.TrimSpace(*req.DisplayName)
		displayName = &value
	}
	var passwordHash *string
	if req.Password != nil {
		if *req.Password == "" {
			return AdminAccountView{}, apperr.Invalid("admin: password is required")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return AdminAccountView{}, apperr.Internal(err)
		}
		value := string(hash)
		passwordHash = &value
	}
	row, err = s.repo.UpdateSuperAdminByID(ctx, id, displayName, passwordHash)
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// DeleteSuperAdmin permanently removes a non-system headquarters account.
func (s *Service) DeleteSuperAdmin(ctx context.Context, id int64) error {
	row, err := s.repo.GetAdminAccountByID(ctx, id)
	if err != nil {
		return err
	}
	if row.Role != "super_admin" {
		return apperr.NotFound("admin account not found")
	}
	if row.IsSystem {
		return apperr.Forbidden("system administrator cannot be deleted")
	}
	return s.repo.DeleteSuperAdminByID(ctx, id)
}

// DisableAdminAccount disables a super_admin account by id, invalidating any
// outstanding session. It rejects ids that do not belong to a super_admin
// account so this endpoint cannot be used to disable a store_admin.
func (s *Service) DisableAdminAccount(ctx context.Context, id int64) (AdminAccountView, error) {
	row, err := s.repo.GetAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	if row.Role != "super_admin" {
		return AdminAccountView{}, apperr.NotFound("admin account not found")
	}
	if row.IsSystem {
		return AdminAccountView{}, apperr.Forbidden("system administrator cannot be disabled")
	}
	row, err = s.repo.DisableAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// ListStoreAdmins returns a page of store_admin accounts. An optional
// f.StoreID narrows the result to a single store.
func (s *Service) ListStoreAdmins(ctx context.Context, f ListFilter) ([]AdminAccountView, int64, error) {
	rows, total, err := s.repo.ListStoreAdmins(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AdminAccountView, 0, len(rows))
	for _, r := range rows {
		out = append(out, adminAccountView(r))
	}
	return out, total, nil
}

// CreateStoreAdmin creates a store_admin login account pinned to the requested
// store. The caller supplies the initial password; only its bcrypt hash is
// persisted or returned from this layer.
func (s *Service) CreateStoreAdmin(ctx context.Context, req StoreAdminCreateRequest) (AdminAccountView, error) {
	if req.StoreID <= 0 {
		return AdminAccountView{}, apperr.Invalid("admin: storeId is required")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return AdminAccountView{}, apperr.Invalid("admin: username is required")
	}
	if req.Password == "" {
		return AdminAccountView{}, apperr.Invalid("admin: password is required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AdminAccountView{}, apperr.Internal(err)
	}
	row, err := s.repo.CreateStoreAdmin(ctx, req.StoreID, username, string(hash), strings.TrimSpace(req.DisplayName))
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// UpdateStoreAdmin applies a partial update to a store_admin account,
// regardless of its current store.
func (s *Service) UpdateStoreAdmin(ctx context.Context, id int64, req StoreAdminUpdateRequest) (AdminAccountView, error) {
	if req.DisplayName == nil && req.StoreID == nil && req.Password == nil {
		return AdminAccountView{}, apperr.Invalid("admin: no store admin fields to update")
	}
	var displayName *string
	if req.DisplayName != nil {
		trimmed := strings.TrimSpace(*req.DisplayName)
		if trimmed == "" {
			return AdminAccountView{}, apperr.Invalid("admin: displayName is required")
		}
		displayName = &trimmed
	}
	if req.StoreID != nil && *req.StoreID <= 0 {
		return AdminAccountView{}, apperr.Invalid("admin: invalid storeId")
	}
	var passwordHash *string
	if req.Password != nil {
		if *req.Password == "" {
			return AdminAccountView{}, apperr.Invalid("admin: password is required")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return AdminAccountView{}, apperr.Internal(err)
		}
		value := string(hash)
		passwordHash = &value
	}
	row, err := s.repo.UpdateStoreAdminByID(ctx, id, req.StoreID, displayName, passwordHash)
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// DisableStoreAdmin disables a store_admin account by id and invalidates any
// outstanding session. It rejects ids that do not belong to a store_admin
// account so this endpoint cannot be used to disable a super_admin.
func (s *Service) DisableStoreAdmin(ctx context.Context, id int64) (AdminAccountView, error) {
	row, err := s.repo.GetAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	if row.Role != "store_admin" {
		return AdminAccountView{}, apperr.NotFound("store admin account not found")
	}
	row, err = s.repo.DisableAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

func adminAccountView(r AdminAccount) AdminAccountView {
	return AdminAccountView{
		ID: r.ID, Username: r.Username, DisplayName: r.DisplayName, Role: r.Role,
		IsSystem: r.IsSystem, StoreID: r.StoreID, StoreName: r.StoreName,
		Status: r.Status, CreatedAt: r.CreatedAt,
	}
}

// ListStaffAccounts returns a page of WeChat-bound store staff (staff_accounts).
func (s *Service) ListStaffAccounts(ctx context.Context, f ListFilter) ([]StaffAccountView, int64, error) {
	rows, total, err := s.repo.ListStaffAccounts(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]StaffAccountView, 0, len(rows))
	for _, r := range rows {
		out = append(out, staffAccountView(r))
	}
	return out, total, nil
}

func staffAccountView(r StaffAccount) StaffAccountView {
	return StaffAccountView{
		ID: r.ID, MemberID: r.MemberID, Name: r.Name, Phone: maskPhone(r.Phone),
		StoreID: r.StoreID, StoreName: r.StoreName,
		Status: r.Status, CreatedAt: r.CreatedAt,
	}
}

// StoreCreateCashier creates a cashier account (admin_accounts, role=cashier)
// pinned to the caller's own store, generating a random initial password in
// the same bcrypt style the auth module verifies against at login.
func (s *Service) StoreCreateCashier(ctx context.Context, storeID int64, req CashierCreateRequest) (CashierCredentialView, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return CashierCredentialView{}, apperr.Invalid("admin: username is required")
	}
	password, err := generatePassword()
	if err != nil {
		return CashierCredentialView{}, apperr.Internal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return CashierCredentialView{}, apperr.Internal(err)
	}
	row, err := s.repo.CreateCashier(ctx, storeID, username, string(hash), strings.TrimSpace(req.DisplayName))
	if err != nil {
		return CashierCredentialView{}, err
	}
	return CashierCredentialView{AdminAccountView: adminAccountView(row), InitialPassword: password}, nil
}

// StoreUpdateCashier applies a partial update (display name only) to one of the
// caller's own cashier accounts.
func (s *Service) StoreUpdateCashier(ctx context.Context, storeID, id int64, req CashierUpdateRequest) (AdminAccountView, error) {
	if req.DisplayName == nil {
		return AdminAccountView{}, apperr.Invalid("admin: no cashier fields to update")
	}
	row, err := s.repo.UpdateCashier(ctx, storeID, id, strings.TrimSpace(*req.DisplayName))
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// StoreDisableCashier disables one of the caller's own cashier accounts and
// invalidates any outstanding session.
func (s *Service) StoreDisableCashier(ctx context.Context, storeID, id int64) (AdminAccountView, error) {
	row, err := s.repo.DisableCashier(ctx, storeID, id)
	if err != nil {
		return AdminAccountView{}, err
	}
	return adminAccountView(row), nil
}

// StoreResetCashierPassword issues a new random password for one of the
// caller's own cashier accounts and invalidates any outstanding session.
func (s *Service) StoreResetCashierPassword(ctx context.Context, storeID, id int64) (CashierCredentialView, error) {
	password, err := generatePassword()
	if err != nil {
		return CashierCredentialView{}, apperr.Internal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return CashierCredentialView{}, apperr.Internal(err)
	}
	row, err := s.repo.ResetCashierPassword(ctx, storeID, id, string(hash))
	if err != nil {
		return CashierCredentialView{}, err
	}
	return CashierCredentialView{AdminAccountView: adminAccountView(row), InitialPassword: password}, nil
}

// AdminCreateStaffAccount creates a staff_accounts row for any store; unlike
// the store console, the caller supplies the target storeId.
func (s *Service) AdminCreateStaffAccount(ctx context.Context, req AdminStaffAccountCreateRequest) (StaffAccountView, error) {
	if req.StoreID <= 0 {
		return StaffAccountView{}, apperr.Invalid("admin: storeId is required")
	}
	if req.MemberID <= 0 {
		return StaffAccountView{}, apperr.Invalid("admin: memberId is required")
	}
	row, err := s.repo.CreateStaffAccount(ctx, req.StoreID, req.MemberID, strings.TrimSpace(req.Name))
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

// AdminDeleteStaffAccount removes a staff binding for any store (headquarters).
// Only the staff role is revoked; the member's mini-program account is untouched.
func (s *Service) AdminDeleteStaffAccount(ctx context.Context, id int64) error {
	return s.repo.DeleteStaffAccountByID(ctx, id)
}

// SearchMembersByPhone fuzzy-matches registered members by phone fragment (tail
// number supported) so a console can pick one to bind as store staff. Requires
// at least 3 digits to avoid dumping the whole member table.
func (s *Service) SearchMembersByPhone(ctx context.Context, phone string) ([]MemberView, error) {
	phone = strings.TrimSpace(phone)
	if len(phone) < 3 || len(phone) > 11 {
		return nil, apperr.Invalid("admin: 请输入 3 至 11 位手机号")
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return nil, apperr.Invalid("admin: 手机号只能包含数字")
		}
	}
	rows, err := s.repo.SearchMembersByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	out := make([]MemberView, 0, len(rows))
	for _, m := range rows {
		out = append(out, MemberView{
			ID: m.ID, Nickname: m.Nickname, Phone: m.Phone, AvatarURL: m.AvatarURL,
			PointsBalance: m.PointsBalance, Status: m.Status, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

// AdminUpdateStaffAccount applies a partial update (name and/or a store
// reassignment) to any staff account, not scoped to a single store.
func (s *Service) AdminUpdateStaffAccount(ctx context.Context, id int64, req AdminStaffAccountUpdateRequest) (StaffAccountView, error) {
	if req.Name == nil && req.StoreID == nil {
		return StaffAccountView{}, apperr.Invalid("admin: no staff fields to update")
	}
	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return StaffAccountView{}, apperr.Invalid("admin: name is required")
		}
		name = &trimmed
	}
	if req.StoreID != nil && *req.StoreID <= 0 {
		return StaffAccountView{}, apperr.Invalid("admin: invalid storeId")
	}
	row, err := s.repo.UpdateStaffAccountByID(ctx, id, req.StoreID, name)
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

// AdminDisableStaffAccount disables any staff account and invalidates any
// outstanding session, regardless of which store it belongs to.
func (s *Service) AdminDisableStaffAccount(ctx context.Context, id int64) (StaffAccountView, error) {
	row, err := s.repo.DisableStaffAccountByID(ctx, id)
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

// StoreCreateStaffAccount binds an existing active mini-program member to a
// staff_accounts row pinned to the caller's own store.
func (s *Service) StoreCreateStaffAccount(ctx context.Context, storeID int64, req StaffAccountCreateRequest) (StaffAccountView, error) {
	if req.MemberID <= 0 {
		return StaffAccountView{}, apperr.Invalid("admin: memberId is required")
	}
	row, err := s.repo.CreateStaffAccount(ctx, storeID, req.MemberID, strings.TrimSpace(req.Name))
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

// StoreDeleteStaffAccount removes a staff binding scoped to the caller's own
// store. Only the staff role is revoked; the member account is untouched.
func (s *Service) StoreDeleteStaffAccount(ctx context.Context, storeID, id int64) error {
	return s.repo.DeleteStaffAccount(ctx, storeID, id)
}

// StoreUpdateStaffAccount applies a partial update (name only) to one of the
// caller's own staff accounts.
func (s *Service) StoreUpdateStaffAccount(ctx context.Context, storeID, id int64, req StaffAccountUpdateRequest) (StaffAccountView, error) {
	if req.Name == nil {
		return StaffAccountView{}, apperr.Invalid("admin: no staff fields to update")
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" {
		return StaffAccountView{}, apperr.Invalid("admin: name is required")
	}
	row, err := s.repo.UpdateStaffAccount(ctx, storeID, id, name)
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

// StoreDisableStaffAccount disables one of the caller's own staff accounts and
// invalidates any outstanding session.
func (s *Service) StoreDisableStaffAccount(ctx context.Context, storeID, id int64) (StaffAccountView, error) {
	row, err := s.repo.DisableStaffAccount(ctx, storeID, id)
	if err != nil {
		return StaffAccountView{}, err
	}
	return staffAccountView(row), nil
}

const (
	generatedPasswordLength = 12
	// Excludes visually ambiguous characters (0/O, 1/l/I) since this password is
	// read off-screen by a store operator and typed in once.
	passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
)

// generatePassword returns a random initial password for a newly created or
// reset cashier account.
func generatePassword() (string, error) {
	b := make([]byte, generatedPasswordLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = passwordAlphabet[n.Int64()]
	}
	return string(b), nil
}

// GetStoreProfile returns the profile of the given store for the store console.
func (s *Service) GetStoreProfile(ctx context.Context, storeID int64) (StoreProfileView, error) {
	if s.stores == nil {
		return StoreProfileView{}, apperr.NotImplemented("admin: store profile provider not wired")
	}
	return s.stores.StoreProfile(ctx, storeID)
}

// scopedFilter narrows a filter to a single store; used by store-console handlers.
func scopedFilter(f ListFilter, storeID int64) ListFilter {
	f.StoreID = &storeID
	return f
}
