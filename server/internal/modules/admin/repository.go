package admin

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// ListFilter is the shared, normalised input for every console list query. A nil
// StoreID means no scope filter (admin console); a set StoreID pins the query to
// a single store (store console, derived from the JWT scope). Status and Keyword
// are optional refinements.
type ListFilter struct {
	Page      httpx.Page
	StoreID   *int64
	Status    string
	Keyword   string
	SortBy    string
	SortOrder string
	// Order-center member refinements are independent so nickname, phone and
	// order number can each be fuzzy-matched.
	MemberNickname string
	MemberPhone    string
	PaymentStatus  string
	PayChannel     string
	RefundID       string
	OperatedFrom   *time.Time
	OperatedBefore *time.Time
	LedgerID       string
	Direction      string
	SourceType     string
	ReasonKeyword  string
	CreatedFrom    *time.Time
	CreatedBefore  *time.Time
	// IncludePointRequests is enabled only by the headquarters wallet ledger.
	// Store-console wallet reads retain their existing settled-ledger semantics.
	IncludePointRequests bool
	// MemberID and AssetType additionally narrow the wallet ledger console read.
	MemberID  *int64
	AssetType string
}

// Repository is the console read persistence port. Each method returns a page of
// rows plus the unfiltered-by-page total so the handler can build pagination
// meta.
type Repository interface {
	ListStores(ctx context.Context, f ListFilter) ([]StoreSummary, int64, error)
	ListCatalogItems(ctx context.Context, f ListFilter) ([]CatalogItem, int64, error)
	ListCouponTemplates(ctx context.Context, f ListFilter) ([]CouponTemplate, int64, error)
	ListActivities(ctx context.Context, f ListFilter) ([]Activity, int64, error)
	ListOrders(ctx context.Context, f ListFilter) ([]Order, int64, error)
	ListMembers(ctx context.Context, f ListFilter) ([]Member, int64, error)
	// GetMember returns a single member, scoped to storeID: the member must have
	// at least one business order at that store, mirroring ListMembers' scope.
	GetMember(ctx context.Context, storeID, memberID int64) (Member, error)
	// GetMemberByID returns a single member by id alone, for the headquarters
	// console which is not pinned to a single store.
	GetMemberByID(ctx context.Context, memberID int64) (Member, error)
	// SearchMembersByPhone fuzzy-matches members by phone fragment (tail number
	// supported) across all members (not store-scoped, not order-gated) so a
	// newly-registered member can be located for staff binding.
	SearchMembersByPhone(ctx context.Context, phone string) ([]Member, error)
	// ListWalletLedger returns settled wallet changes plus point deposit and
	// withdrawal applications. A set f.StoreID filters records attributed to
	// that store.
	ListWalletLedger(ctx context.Context, f ListFilter) ([]WalletLedgerEntry, int64, error)
	ListPaymentTransactions(ctx context.Context, f ListFilter) ([]PaymentTransaction, int64, error)
	ListRefunds(ctx context.Context, f ListFilter) ([]Refund, int64, error)
	ListAuditLogs(ctx context.Context, f ListFilter) ([]AuditLog, int64, error)
	ListRuleDefinitions(ctx context.Context, f ListFilter) ([]RuleDefinition, int64, error)
	CreateRuleDefinition(ctx context.Context, req RuleDefinitionCreate) (RuleDefinition, error)
	UpdateRuleDefinition(ctx context.Context, ruleID int64, u RuleDefinitionUpdate) (RuleDefinition, error)
	ListAdminAccounts(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error)
	ListCashiers(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error)
	ListStaffAccounts(ctx context.Context, f ListFilter) ([]StaffAccount, int64, error)

	// Cashier accounts (admin_accounts, role=cashier), always scoped to the
	// caller's own store so a store console can never reach another store's rows.
	GetCashier(ctx context.Context, storeID, id int64) (AdminAccount, error)
	CreateCashier(ctx context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error)
	UpdateCashier(ctx context.Context, storeID, id int64, displayName string) (AdminAccount, error)
	DisableCashier(ctx context.Context, storeID, id int64) (AdminAccount, error)
	ResetCashierPassword(ctx context.Context, storeID, id int64, passwordHash string) (AdminAccount, error)

	// Headquarters account management (admin_accounts, role=super_admin or
	// role=store_admin). Not scoped to a single store: headquarters manages
	// every account across the platform.
	ListSuperAdmins(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error)
	ListStoreAdmins(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error)
	GetAdminAccountByID(ctx context.Context, id int64) (AdminAccount, error)
	CreateSuperAdmin(ctx context.Context, username, passwordHash, displayName string) (AdminAccount, error)
	UpdateSuperAdminByID(ctx context.Context, id int64, displayName, passwordHash *string) (AdminAccount, error)
	DeleteSuperAdminByID(ctx context.Context, id int64) error
	CreateStoreAdmin(ctx context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error)
	UpdateStoreAdminByID(ctx context.Context, id int64, storeID *int64, displayName, passwordHash *string) (AdminAccount, error)
	DisableAdminAccountByID(ctx context.Context, id int64) (AdminAccount, error)

	// Staff accounts (staff_accounts), always scoped to the caller's own store.
	// CreateStaffAccount binds an existing member (by id) as store staff.
	// DeleteStaffAccount removes only the staff binding row; the member account is
	// untouched.
	GetStaffAccount(ctx context.Context, storeID, id int64) (StaffAccount, error)
	CreateStaffAccount(ctx context.Context, storeID, memberID int64, name string) (StaffAccount, error)
	UpdateStaffAccount(ctx context.Context, storeID, id int64, name string) (StaffAccount, error)
	DisableStaffAccount(ctx context.Context, storeID, id int64) (StaffAccount, error)
	DeleteStaffAccount(ctx context.Context, storeID, id int64) error

	// Staff accounts, admin variants: not scoped to a single store, so
	// headquarters can manage any store's staff.
	GetStaffAccountByID(ctx context.Context, id int64) (StaffAccount, error)
	UpdateStaffAccountByID(ctx context.Context, id int64, storeID *int64, name *string) (StaffAccount, error)
	DisableStaffAccountByID(ctx context.Context, id int64) (StaffAccount, error)
	DeleteStaffAccountByID(ctx context.Context, id int64) error
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL console read repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

// filterClauses builds the shared WHERE fragment used by every list query. The
// column names are code-controlled (never request input) so they are safe to
// interpolate; the status/keyword/scope values are always bound as parameters.
// An empty column name skips that filter (e.g. a table without a status column).
func filterClauses(f ListFilter, scopeCol, statusCol, keywordCol string) (string, []any) {
	where := "1 = 1"
	var args []any
	if f.StoreID != nil && scopeCol != "" {
		where += " AND " + scopeCol + " = ?"
		args = append(args, *f.StoreID)
	}
	if f.Status != "" && statusCol != "" {
		where += " AND " + statusCol + " = ?"
		args = append(args, f.Status)
	}
	if f.Keyword != "" && keywordCol != "" {
		where += " AND " + keywordCol + " LIKE ?"
		args = append(args, "%"+f.Keyword+"%")
	}
	return where, args
}

func (r *sqlRepository) ListStores(ctx context.Context, f ListFilter) ([]StoreSummary, int64, error) {
	where, args := filterClauses(f, "id", "status", "name")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stores WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, name, COALESCE(phone,''), address, status, created_at
		FROM stores WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]StoreSummary, 0)
	for rows.Next() {
		var s StoreSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Phone, &s.Address, &s.Status, &s.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListCatalogItems(ctx context.Context, f ListFilter) ([]CatalogItem, int64, error) {
	where, args := filterClauses(f, "store_id", "status", "name")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, scope_type, store_id, category_id, name, item_type, price_cent, stock_quantity, status, created_at
		FROM catalog_items WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]CatalogItem, 0)
	for rows.Next() {
		var it CatalogItem
		if err := rows.Scan(&it.ID, &it.ScopeType, &it.StoreID, &it.CategoryID, &it.Name,
			&it.ItemType, &it.PriceCent, &it.StockQuantity, &it.Status, &it.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListCouponTemplates(ctx context.Context, f ListFilter) ([]CouponTemplate, int64, error) {
	where, args := filterClauses(f, "store_id", "status", "name")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_templates WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, scope_type, store_id, name, coupon_type, value_cent, stock_quantity, issued_quantity, status, created_at
		FROM coupon_templates WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]CouponTemplate, 0)
	for rows.Next() {
		var ct CouponTemplate
		if err := rows.Scan(&ct.ID, &ct.ScopeType, &ct.StoreID, &ct.Name, &ct.CouponType,
			&ct.ValueCent, &ct.TotalStock, &ct.IssuedCount, &ct.Status, &ct.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, ct)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListActivities(ctx context.Context, f ListFilter) ([]Activity, int64, error) {
	// activities has no dedicated type column; the console shows the title only.
	where, args := filterClauses(f, "store_id", "status", "title")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activities WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, scope_type, store_id, title, asset_id, start_at, end_at, status, created_at, updated_at
		FROM activities WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Activity, 0)
	for rows.Next() {
		var a Activity
		if err := rows.Scan(
			&a.ID, &a.ScopeType, &a.StoreID, &a.Name, &a.AssetID, &a.StartAt,
			&a.EndAt, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListOrders(ctx context.Context, f ListFilter) ([]Order, int64, error) {
	where := "1 = 1"
	var args []any
	if f.StoreID != nil {
		where += " AND bo.store_id = ?"
		args = append(args, *f.StoreID)
	}
	if f.Status != "" {
		where += " AND bo.order_status = ?"
		args = append(args, f.Status)
	}
	if f.PaymentStatus != "" {
		where += " AND bo.payment_status = ?"
		args = append(args, f.PaymentStatus)
	}
	if f.PayChannel != "" {
		where += ` AND COALESCE((SELECT po.pay_method FROM payment_orders po
			WHERE po.business_order_id = bo.id ORDER BY po.id DESC LIMIT 1), '') = ?`
		args = append(args, f.PayChannel)
	}
	if f.Keyword != "" {
		where += " AND bo.business_order_no LIKE ?"
		args = append(args, "%"+f.Keyword+"%")
	}
	if f.MemberNickname != "" {
		where += " AND m.nickname LIKE ?"
		args = append(args, "%"+f.MemberNickname+"%")
	}
	if f.MemberPhone != "" {
		where += " AND m.phone LIKE ?"
		args = append(args, "%"+f.MemberPhone+"%")
	}
	var total int64
	countQ := `SELECT COUNT(*) FROM business_orders bo LEFT JOIN members m ON m.id = bo.member_id WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	// pay_channel is resolved from the latest payment order; completed_at is the
	// last update once the order is paid (business_orders keeps no explicit column).
	// store/member display fields are left-joined so headquarters and store-scoped
	// (member_id may be null) rows both list.
	q := `SELECT bo.id, bo.business_order_no, bo.order_type, bo.store_id, COALESCE(s.name, ''),
			bo.member_id, COALESCE(m.nickname, ''), COALESCE(m.phone, ''), COALESCE(m.avatar_url, ''), bo.total_amount_cent,
			COALESCE((SELECT po.id FROM payment_orders po WHERE po.business_order_id = bo.id ORDER BY po.id DESC LIMIT 1), 0),
			COALESCE((SELECT po.pay_method FROM payment_orders po WHERE po.business_order_id = bo.id ORDER BY po.id DESC LIMIT 1), ''),
			COALESCE((SELECT ro.status FROM refund_orders ro
				WHERE ro.business_order_id = bo.id ORDER BY ro.id DESC LIMIT 1), ''),
			bo.payment_status, bo.order_status, bo.created_at,
			CASE WHEN bo.payment_status = 'paid' THEN bo.updated_at ELSE NULL END
		FROM business_orders bo
		LEFT JOIN stores s ON s.id = bo.store_id
		LEFT JOIN members m ON m.id = bo.member_id
		WHERE ` + where + ` ORDER BY bo.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Order, 0)
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.OrderNo, &o.OrderType, &o.StoreID, &o.StoreName,
			&o.MemberID, &o.MemberNickname, &o.MemberPhone, &o.MemberAvatarURL, &o.TotalCent,
			&o.PaymentOrderID, &o.PayChannel, &o.RefundStatus, &o.PaymentStatus,
			&o.OrderStatus, &o.CreatedAt, &o.CompletedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListPaymentTransactions(ctx context.Context, f ListFilter) ([]PaymentTransaction, int64, error) {
	where, args := filterClauses(f, "po.store_id", "po.status", "po.payment_order_no")
	var total int64
	countQ := `SELECT COUNT(*) FROM payment_orders po WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT po.id, po.payment_order_no, po.store_id, COALESCE(s.name,''),
			bo.id, bo.business_order_no, bo.order_type, po.amount_cent, po.pay_method,
			po.status, po.created_at, po.paid_at
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		LEFT JOIN stores s ON s.id = po.store_id
		WHERE ` + where + ` ORDER BY po.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]PaymentTransaction, 0)
	for rows.Next() {
		var p PaymentTransaction
		if err := rows.Scan(&p.ID, &p.PaymentOrderNo, &p.StoreID, &p.StoreName,
			&p.BusinessOrderID, &p.BusinessOrderNo, &p.OrderType, &p.AmountCent, &p.PayMethod,
			&p.Status, &p.CreatedAt, &p.PaidAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListRefunds(ctx context.Context, f ListFilter) ([]Refund, int64, error) {
	where := "ro.status IN (?, ?)"
	args := []any{"succeeded", "failed"}
	if f.RefundID != "" {
		where += " AND CAST(ro.id AS CHAR) LIKE ?"
		args = append(args, "%"+f.RefundID+"%")
	}
	if f.Keyword != "" {
		where += " AND bo.business_order_no LIKE ?"
		args = append(args, "%"+f.Keyword+"%")
	}
	if f.MemberNickname != "" {
		where += " AND m.nickname LIKE ?"
		args = append(args, "%"+f.MemberNickname+"%")
	}
	if f.MemberPhone != "" {
		where += " AND m.phone LIKE ?"
		args = append(args, "%"+f.MemberPhone+"%")
	}
	if f.StoreID != nil {
		where += " AND ro.store_id = ?"
		args = append(args, *f.StoreID)
	}
	if f.Status != "" {
		where += " AND ro.status = ?"
		args = append(args, f.Status)
	}
	if f.OperatedFrom != nil {
		where += " AND ro.updated_at >= ?"
		args = append(args, f.OperatedFrom.UTC())
	}
	if f.OperatedBefore != nil {
		where += " AND ro.updated_at < ?"
		args = append(args, f.OperatedBefore.UTC())
	}
	var total int64
	countQ := `SELECT COUNT(*)
		FROM refund_orders ro
		JOIN business_orders bo ON bo.id = ro.business_order_id
		LEFT JOIN members m ON m.id = bo.member_id
		WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ro.id, ro.refund_order_no, ro.payment_order_id, ro.business_order_id,
			ro.store_id, COALESCE(s.name,''), bo.business_order_no, bo.total_amount_cent,
			bo.member_id, COALESCE(m.nickname,''), COALESCE(m.phone,''), COALESCE(m.avatar_url,''),
			ro.amount_cent, ro.channel, ro.status, ro.reason, bo.created_at, ro.updated_at
		FROM refund_orders ro
		JOIN business_orders bo ON bo.id = ro.business_order_id
		LEFT JOIN stores s ON s.id = ro.store_id
		LEFT JOIN members m ON m.id = bo.member_id
		WHERE ` + where + ` ORDER BY ro.updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Refund, 0)
	for rows.Next() {
		var rf Refund
		if err := rows.Scan(&rf.ID, &rf.RefundOrderNo, &rf.PaymentOrderID, &rf.BusinessOrderID,
			&rf.StoreID, &rf.StoreName, &rf.BusinessOrderNo, &rf.OrderAmountCent,
			&rf.MemberID, &rf.MemberNickname, &rf.MemberPhone, &rf.MemberAvatarURL,
			&rf.AmountCent, &rf.Channel, &rf.Status, &rf.Reason,
			&rf.OrderCreatedAt, &rf.OperatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, rf)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListMembers(ctx context.Context, f ListFilter) ([]Member, int64, error) {
	// members are not store-scoped in the schema; a store scope selects members
	// who have at least one business order at that store.
	where, args := filterClauses(f, "", "m.status", "")
	if f.Keyword != "" {
		keyword := "%" + f.Keyword + "%"
		where += " AND (m.nickname LIKE ? OR m.phone LIKE ?)"
		args = append(args, keyword, keyword)
	}
	if f.StoreID != nil {
		where += " AND EXISTS (SELECT 1 FROM business_orders bo WHERE bo.member_id = m.id AND bo.store_id = ?)"
		args = append(args, *f.StoreID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members m WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	orderColumn := "m.created_at"
	switch f.SortBy {
	case "pointsBalance":
		orderColumn = "points_balance"
	case "coinsBalance":
		orderColumn = "coins_balance"
	case "vipLevel":
		orderColumn = "vip_level"
	}
	orderDirection := "DESC"
	if strings.EqualFold(f.SortOrder, "asc") {
		orderDirection = "ASC"
	}
	q := `SELECT m.id, m.nickname, COALESCE(m.phone,''), COALESCE(m.avatar_url,''), COALESCE(m.gender,''),
			COALESCE(points.available_amount, 0) AS points_balance,
			COALESCE(coins.available_amount, 0) AS coins_balance,
			COALESCE(current_tier.name, base_tier.name, '') AS vip_tier_name,
			COALESCE(current_tier.level, base_tier.level, 0) AS vip_level,
			m.status, m.created_at
		FROM members m
		LEFT JOIN wallet_accounts points ON points.member_id = m.id AND points.asset_type = 'points'
		LEFT JOIN wallet_accounts coins ON coins.member_id = m.id AND coins.asset_type = 'coins'
		LEFT JOIN membership_tiers current_tier ON current_tier.id = m.current_tier_id
		LEFT JOIN (
			SELECT name, level FROM membership_tiers
			WHERE status = 'active' ORDER BY level ASC, id ASC LIMIT 1
		) base_tier ON 1 = 1
		WHERE ` + where + ` ORDER BY ` + orderColumn + ` ` + orderDirection + `, m.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Member, 0)
	for rows.Next() {
		var m Member
		if err := rows.Scan(
			&m.ID,
			&m.Nickname,
			&m.Phone,
			&m.AvatarURL,
			&m.Gender,
			&m.PointsBalance,
			&m.CoinsBalance,
			&m.VIPTierName,
			&m.VIPLevel,
			&m.Status,
			&m.CreatedAt,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetMember(ctx context.Context, storeID, memberID int64) (Member, error) {
	const q = `SELECT m.id, m.nickname, COALESCE(m.phone,''),
			COALESCE((SELECT wa.available_amount FROM wallet_accounts wa WHERE wa.member_id = m.id AND wa.asset_type = 'points'), 0),
			m.status, m.created_at
		FROM members m
		WHERE m.id = ? AND EXISTS (SELECT 1 FROM business_orders bo WHERE bo.member_id = m.id AND bo.store_id = ?)`
	var m Member
	err := r.db.QueryRowContext(ctx, q, memberID, storeID).Scan(&m.ID, &m.Nickname, &m.Phone, &m.PointsBalance, &m.Status, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return Member{}, apperr.NotFound("member not found")
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

// GetMemberByID looks up a member by id alone, for the headquarters console
// which is not pinned to a single store.
func (r *sqlRepository) GetMemberByID(ctx context.Context, memberID int64) (Member, error) {
	const q = `SELECT m.id, m.nickname, COALESCE(m.phone,''),
			COALESCE((SELECT wa.available_amount FROM wallet_accounts wa WHERE wa.member_id = m.id AND wa.asset_type = 'points'), 0),
			m.status, m.created_at
		FROM members m WHERE m.id = ?`
	var m Member
	err := r.db.QueryRowContext(ctx, q, memberID).Scan(&m.ID, &m.Nickname, &m.Phone, &m.PointsBalance, &m.Status, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return Member{}, apperr.NotFound("member not found")
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

// SearchMembersByPhone fuzzy-matches members whose phone contains the given
// fragment (e.g. a tail number), across all members (not store-scoped, not
// order-gated) so a newly-registered member can be located for staff binding.
// Returns up to 20 matches, newest first; empty slice when none match.
func (r *sqlRepository) SearchMembersByPhone(ctx context.Context, phone string) ([]Member, error) {
	const q = `SELECT m.id, m.nickname, COALESCE(m.phone,''),
			COALESCE((SELECT wa.available_amount FROM wallet_accounts wa WHERE wa.member_id = m.id AND wa.asset_type = 'points'), 0),
			m.status, m.created_at
		FROM members m
		WHERE m.status = 'active' AND m.phone IS NOT NULL AND m.phone LIKE ?
		ORDER BY m.id DESC LIMIT 20`
	rows, err := r.db.QueryContext(ctx, q, "%"+phone+"%")
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Member, 0)
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Nickname, &m.Phone, &m.PointsBalance, &m.Status, &m.CreatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListWalletLedger combines settled wallet balance changes with point deposit
// and withdrawal applications. Pending/rejected applications intentionally
// carry a NULL balance_after because they did not change the member's wallet.
func (r *sqlRepository) ListWalletLedger(ctx context.Context, f ListFilter) ([]WalletLedgerEntry, int64, error) {
	where := "1 = 1"
	var args []any
	if f.LedgerID != "" {
		where += " AND CAST(entry.id AS CHAR) LIKE ?"
		args = append(args, "%"+f.LedgerID+"%")
	}
	if f.MemberID != nil {
		where += " AND entry.member_id = ?"
		args = append(args, *f.MemberID)
	}
	if f.AssetType != "" {
		where += " AND entry.asset_type = ?"
		args = append(args, f.AssetType)
	}
	if f.MemberNickname != "" {
		where += " AND m.nickname LIKE ?"
		args = append(args, "%"+f.MemberNickname+"%")
	}
	if f.MemberPhone != "" {
		where += " AND m.phone LIKE ?"
		args = append(args, "%"+f.MemberPhone+"%")
	}
	if f.Direction != "" {
		where += " AND entry.direction = ?"
		args = append(args, f.Direction)
	}
	if f.SourceType != "" {
		where += " AND entry.source_type LIKE ?"
		args = append(args, "%"+f.SourceType+"%")
	}
	if f.Status != "" {
		where += " AND entry.status = ?"
		args = append(args, f.Status)
	}
	if f.ReasonKeyword != "" {
		where += " AND entry.reason LIKE ?"
		args = append(args, "%"+f.ReasonKeyword+"%")
	}
	if f.CreatedFrom != nil {
		where += " AND entry.created_at >= ?"
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedBefore != nil {
		where += " AND entry.created_at < ?"
		args = append(args, f.CreatedBefore.UTC())
	}
	if f.StoreID != nil {
		where += " AND entry.store_id = ?"
		args = append(args, *f.StoreID)
	}

	const settledEntries = `SELECT wle.id, CONCAT('ledger:', wle.id) AS record_key,
			wle.member_id, wle.asset_type, wle.direction, wle.amount,
			wle.balance_after, 'completed' AS status, wle.reason,
			wle.source_type, wle.source_id,
			COALESCE(payment_bo.store_id, recharge_bo.store_id, refund_bo.store_id,
				food_bo.store_id) AS store_id,
			COALESCE(payment_bo.business_order_no, recharge_bo.business_order_no,
				refund_bo.business_order_no, food_bo.business_order_no, '') AS related_order_no,
			wle.created_at
		FROM wallet_ledger_entries wle
		LEFT JOIN payment_orders po
			ON wle.source_type = 'payment_order' AND po.id = wle.source_id
		LEFT JOIN business_orders payment_bo ON payment_bo.id = po.business_order_id
		LEFT JOIN business_orders recharge_bo
			ON wle.source_type IN ('recharge_order', 'recharge_growth')
			AND recharge_bo.id = wle.source_id
		LEFT JOIN refund_orders ro
			ON wle.source_type = 'refund_order' AND ro.id = wle.source_id
		LEFT JOIN business_orders refund_bo ON refund_bo.id = ro.business_order_id
		LEFT JOIN business_orders food_bo
			ON wle.source_type = 'food_order' AND food_bo.id = wle.source_id`
	const pointRequestEntries = `
		UNION ALL SELECT ps.id, CONCAT('point_saving:', ps.id), ps.member_id, 'points',
			'credit', ps.points, NULL, ps.status,
			CASE WHEN ps.remark = '' THEN 'point_saving' ELSE ps.remark END,
			'point_saving', ps.id, ps.store_id, '', ps.created_at
		FROM point_savings ps
		UNION ALL SELECT pw.id, CONCAT('point_withdrawal:', pw.id), pw.member_id, 'points',
			'debit', pw.points, NULL, pw.status,
			CASE WHEN pw.remark = '' THEN 'point_withdrawal' ELSE pw.remark END,
			'point_withdrawal', pw.id, pw.store_id, '', pw.created_at
		FROM point_withdrawals pw`
	entries := "(" + settledEntries
	if f.IncludePointRequests {
		entries += pointRequestEntries
	}
	entries += ") entry"
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM `+entries+`
		JOIN members m ON m.id = entry.member_id
		LEFT JOIN stores s ON s.id = entry.store_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT entry.id, entry.record_key, entry.member_id,
			COALESCE(m.nickname, ''), COALESCE(m.phone, ''), COALESCE(m.avatar_url, ''),
			entry.store_id, COALESCE(s.name, ''), entry.asset_type, entry.direction,
			entry.amount, entry.balance_after, entry.status, entry.reason,
			entry.source_type, entry.source_id, entry.related_order_no, entry.created_at
		FROM ` + entries + `
		JOIN members m ON m.id = entry.member_id
		LEFT JOIN stores s ON s.id = entry.store_id
		WHERE ` + where + ` ORDER BY entry.created_at DESC, entry.record_key DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any(nil), args...), f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]WalletLedgerEntry, 0)
	for rows.Next() {
		var e WalletLedgerEntry
		if err := rows.Scan(&e.ID, &e.RecordKey, &e.MemberID, &e.MemberNickname, &e.MemberPhone,
			&e.MemberAvatarURL, &e.StoreID, &e.StoreName, &e.AssetType, &e.Direction,
			&e.Amount, &e.BalanceAfter, &e.Status, &e.Reason, &e.SourceType,
			&e.SourceID, &e.RelatedOrderNo, &e.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListAuditLogs(ctx context.Context, f ListFilter) ([]AuditLog, int64, error) {
	// audit_logs has no status column; keyword matches the action.
	where, args := filterClauses(f, "store_id", "", "action")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, actor_type, actor_id, action, target_type, target_id, store_id, request_id, created_at
		FROM audit_logs WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]AuditLog, 0)
	for rows.Next() {
		var a AuditLog
		var targetID int64
		if err := rows.Scan(&a.ID, &a.ActorType, &a.ActorID, &a.Action, &a.TargetType,
			&targetID, &a.StoreID, &a.RequestID, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		a.TargetID = strconv.FormatInt(targetID, 10)
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListRuleDefinitions(ctx context.Context, f ListFilter) ([]RuleDefinition, int64, error) {
	where, args := filterClauses(f, "store_id", "status", "rule_key")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rule_definitions WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, rule_key, scope_type, store_id, version, config_json, enabled, status, updated_at
		FROM rule_definitions WHERE ` + where + ` ORDER BY rule_key ASC, version DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]RuleDefinition, 0)
	for rows.Next() {
		rd, err := scanRule(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, rd)
	}
	return out, total, rows.Err()
}

// CreateRuleDefinition inserts a new rule_definitions row. New rows always
// start in the DB-default 'draft' status regardless of the requested enabled
// flag, so publishing is a distinct, explicit action.
func (r *sqlRepository) CreateRuleDefinition(ctx context.Context, req RuleDefinitionCreate) (RuleDefinition, error) {
	const q = `INSERT INTO rule_definitions
		(rule_key, scope_type, store_id, version, config_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, req.Key, req.ScopeType, req.StoreID, req.Version, []byte(req.ConfigJSON), req.Enabled)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return RuleDefinition{}, apperr.Conflict("admin: rule definition version already exists")
		}
		return RuleDefinition{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return RuleDefinition{}, apperr.Internal(err)
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, rule_key, scope_type, store_id, version, config_json, enabled, status, updated_at
		FROM rule_definitions WHERE id = ?`, id)
	return scanRule(row)
}

func (r *sqlRepository) UpdateRuleDefinition(ctx context.Context, ruleID int64, u RuleDefinitionUpdate) (RuleDefinition, error) {
	set := make([]string, 0, 4)
	var args []any
	if len(u.ConfigJSON) > 0 {
		set = append(set, "config_json = ?")
		args = append(args, []byte(u.ConfigJSON))
	}
	if u.Enabled != nil {
		set = append(set, "enabled = ?")
		args = append(args, *u.Enabled)
	}
	if u.Status != nil {
		set = append(set, "status = ?")
		args = append(args, *u.Status)
	}
	set = append(set, "updated_at = NOW()")
	args = append(args, ruleID)

	res, err := r.db.ExecContext(ctx, `UPDATE rule_definitions SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return RuleDefinition{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// RowsAffected is 0 when the row is missing or the values are unchanged;
		// disambiguate by checking existence below.
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM rule_definitions WHERE id = ?`, ruleID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return RuleDefinition{}, apperr.NotFound("admin: rule definition not found")
			}
			return RuleDefinition{}, apperr.Internal(err)
		}
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, rule_key, scope_type, store_id, version, config_json, enabled, status, updated_at
		FROM rule_definitions WHERE id = ?`, ruleID)
	return scanRule(row)
}

func (r *sqlRepository) ListAdminAccounts(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error) {
	where, args := filterClauses(f, "aa.store_id", "aa.status", "aa.username")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_accounts aa WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.is_system, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE ` + where + ` ORDER BY aa.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]AdminAccount, 0)
	for rows.Next() {
		var a AdminAccount
		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.IsSystem, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListCashiers(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error) {
	where, args := filterClauses(f, "aa.store_id", "aa.status", "aa.username")
	where += " AND aa.role = 'cashier'"
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_accounts aa WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.is_system, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE ` + where + ` ORDER BY aa.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]AdminAccount, 0)
	for rows.Next() {
		var a AdminAccount
		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.IsSystem, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListSuperAdmins(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error) {
	where, args := filterClauses(f, "aa.store_id", "aa.status", "aa.username")
	where += " AND aa.role = 'super_admin'"
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_accounts aa WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.is_system, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE ` + where + ` ORDER BY aa.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]AdminAccount, 0)
	for rows.Next() {
		var a AdminAccount
		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.IsSystem, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListStoreAdmins(ctx context.Context, f ListFilter) ([]AdminAccount, int64, error) {
	where, args := filterClauses(f, "aa.store_id", "aa.status", "aa.username")
	where += " AND aa.role = 'store_admin'"
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_accounts aa WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.is_system, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE ` + where + ` ORDER BY aa.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]AdminAccount, 0)
	for rows.Next() {
		var a AdminAccount
		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.IsSystem, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

// GetAdminAccountByID looks up an admin_accounts row by id alone, regardless of
// role or store, for the headquarters account-management endpoints which are
// not pinned to a single store.
func (r *sqlRepository) GetAdminAccountByID(ctx context.Context, id int64) (AdminAccount, error) {
	const q = `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.is_system, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE aa.id = ?`
	var a AdminAccount
	err := r.db.QueryRowContext(ctx, q, id).Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.IsSystem, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return AdminAccount{}, apperr.NotFound("admin account not found")
	}
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	return a, nil
}

// CreateSuperAdmin creates a non-system headquarters administrator.
func (r *sqlRepository) CreateSuperAdmin(ctx context.Context, username, passwordHash, displayName string) (AdminAccount, error) {
	const q = `INSERT INTO admin_accounts
		(username, password_hash, display_name, role, is_system, store_id, status, token_version, created_at, updated_at)
		VALUES (?, ?, ?, 'super_admin', 0, NULL, 'active', 0, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, username, passwordHash, displayName)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
		return AdminAccount{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	return r.GetAdminAccountByID(ctx, id)
}

// UpdateSuperAdminByID changes the display name and/or password. Password
// changes invalidate all existing sessions for the account.
func (r *sqlRepository) UpdateSuperAdminByID(ctx context.Context, id int64, displayName, passwordHash *string) (AdminAccount, error) {
	set := make([]string, 0, 3)
	var args []any
	if displayName != nil {
		set = append(set, "display_name = ?")
		args = append(args, *displayName)
	}
	if passwordHash != nil {
		set = append(set, "password_hash = ?", "token_version = token_version + 1")
		args = append(args, *passwordHash)
	}
	set = append(set, "updated_at = NOW()")
	args = append(args, id)
	res, err := r.db.ExecContext(ctx, `UPDATE admin_accounts SET `+strings.Join(set, ", ")+` WHERE id = ? AND role = 'super_admin'`, args...)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		row, err := r.GetAdminAccountByID(ctx, id)
		if err != nil {
			return AdminAccount{}, err
		}
		if row.Role != "super_admin" {
			return AdminAccount{}, apperr.NotFound("admin account not found")
		}
	}
	return r.GetAdminAccountByID(ctx, id)
}

// DeleteSuperAdminByID permanently removes a non-system headquarters account.
func (r *sqlRepository) DeleteSuperAdminByID(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM admin_accounts
		WHERE id = ? AND role = 'super_admin' AND is_system = 0`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		row, err := r.GetAdminAccountByID(ctx, id)
		if err != nil {
			return err
		}
		if row.Role != "super_admin" {
			return apperr.NotFound("admin account not found")
		}
		if row.IsSystem {
			return apperr.Forbidden("system administrator cannot be deleted")
		}
	}
	return nil
}

// CreateStoreAdmin creates a store_admin account (admin_accounts) for
// headquarters, pinned to the given store.
func (r *sqlRepository) CreateStoreAdmin(ctx context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error) {
	const q = `INSERT INTO admin_accounts
		(username, password_hash, display_name, role, store_id, status, token_version, created_at, updated_at)
		VALUES (?, ?, ?, 'store_admin', ?, 'active', 0, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, username, passwordHash, displayName, storeID)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
		return AdminAccount{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	return r.GetAdminAccountByID(ctx, id)
}

// UpdateStoreAdminByID applies a partial update to a store_admin account, not
// scoped to a single store. Password changes also invalidate existing tokens.
func (r *sqlRepository) UpdateStoreAdminByID(ctx context.Context, id int64, storeID *int64, displayName, passwordHash *string) (AdminAccount, error) {
	set := make([]string, 0, 4)
	var args []any
	if displayName != nil {
		set = append(set, "display_name = ?")
		args = append(args, *displayName)
	}
	if storeID != nil {
		set = append(set, "store_id = ?")
		args = append(args, *storeID)
	}
	if passwordHash != nil {
		set = append(set, "password_hash = ?", "token_version = token_version + 1")
		args = append(args, *passwordHash)
	}
	set = append(set, "updated_at = NOW()")
	args = append(args, id)
	res, err := r.db.ExecContext(ctx, `UPDATE admin_accounts SET `+strings.Join(set, ", ")+` WHERE id = ? AND role = 'store_admin'`, args...)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.getStoreAdminByID(ctx, id); err != nil {
			return AdminAccount{}, err
		}
	}
	return r.GetAdminAccountByID(ctx, id)
}

// getStoreAdminByID disambiguates a no-op UPDATE (missing row vs. unchanged
// values) for a store_admin account specifically.
func (r *sqlRepository) getStoreAdminByID(ctx context.Context, id int64) (AdminAccount, error) {
	a, err := r.GetAdminAccountByID(ctx, id)
	if err != nil {
		return AdminAccount{}, err
	}
	if a.Role != "store_admin" {
		return AdminAccount{}, apperr.NotFound("store admin account not found")
	}
	return a, nil
}

// DisableAdminAccountByID marks any admin_accounts row disabled and bumps
// token_version so any outstanding session is invalidated on its next refresh.
func (r *sqlRepository) DisableAdminAccountByID(ctx context.Context, id int64) (AdminAccount, error) {
	const q = `UPDATE admin_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW()
		WHERE id = ? AND is_system = 0`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		row, err := r.GetAdminAccountByID(ctx, id)
		if err != nil {
			return AdminAccount{}, err
		}
		if row.IsSystem {
			return AdminAccount{}, apperr.Forbidden("system administrator cannot be disabled")
		}
	}
	return r.GetAdminAccountByID(ctx, id)
}

func (r *sqlRepository) ListStaffAccounts(ctx context.Context, f ListFilter) ([]StaffAccount, int64, error) {
	where, args := filterClauses(f, "sa.store_id", "sa.status", "sa.name")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM staff_accounts sa WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT sa.id, COALESCE(sa.member_id,0), sa.name, COALESCE(m.phone,''), sa.store_id, COALESCE(s.name,''), sa.status, sa.created_at
		FROM staff_accounts sa
		LEFT JOIN stores s ON s.id = sa.store_id
		LEFT JOIN members m ON m.id = sa.member_id
		WHERE ` + where + ` ORDER BY sa.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]StaffAccount, 0)
	for rows.Next() {
		var a StaffAccount
		if err := rows.Scan(&a.ID, &a.MemberID, &a.Name, &a.Phone, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetCashier(ctx context.Context, storeID, id int64) (AdminAccount, error) {
	const q = `SELECT aa.id, aa.username, aa.display_name, aa.role, aa.store_id, COALESCE(s.name,''), aa.status, aa.created_at
		FROM admin_accounts aa LEFT JOIN stores s ON s.id = aa.store_id
		WHERE aa.id = ? AND aa.store_id = ? AND aa.role = 'cashier'`
	var a AdminAccount
	err := r.db.QueryRowContext(ctx, q, id, storeID).Scan(&a.ID, &a.Username, &a.DisplayName, &a.Role, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return AdminAccount{}, apperr.NotFound("cashier not found")
	}
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	return a, nil
}

func (r *sqlRepository) CreateCashier(ctx context.Context, storeID int64, username, passwordHash, displayName string) (AdminAccount, error) {
	const q = `INSERT INTO admin_accounts
		(username, password_hash, display_name, role, store_id, status, token_version, created_at, updated_at)
		VALUES (?, ?, ?, 'cashier', ?, 'active', 0, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, username, passwordHash, displayName, storeID)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return AdminAccount{}, apperr.Conflict("username already exists")
		}
		return AdminAccount{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	return r.GetCashier(ctx, storeID, id)
}

func (r *sqlRepository) UpdateCashier(ctx context.Context, storeID, id int64, displayName string) (AdminAccount, error) {
	const q = `UPDATE admin_accounts SET display_name = ?, updated_at = NOW()
		WHERE id = ? AND store_id = ? AND role = 'cashier'`
	res, err := r.db.ExecContext(ctx, q, displayName, id, storeID)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetCashier(ctx, storeID, id); err != nil {
			return AdminAccount{}, err
		}
	}
	return r.GetCashier(ctx, storeID, id)
}

// DisableCashier marks a cashier account disabled and bumps token_version so any
// outstanding session is invalidated on its next refresh, mirroring
// auth.Service.LogoutAccount.
func (r *sqlRepository) DisableCashier(ctx context.Context, storeID, id int64) (AdminAccount, error) {
	const q = `UPDATE admin_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW()
		WHERE id = ? AND store_id = ? AND role = 'cashier'`
	res, err := r.db.ExecContext(ctx, q, id, storeID)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetCashier(ctx, storeID, id); err != nil {
			return AdminAccount{}, err
		}
	}
	return r.GetCashier(ctx, storeID, id)
}

// ResetCashierPassword replaces the password hash and bumps token_version so any
// outstanding session using the old password is invalidated.
func (r *sqlRepository) ResetCashierPassword(ctx context.Context, storeID, id int64, passwordHash string) (AdminAccount, error) {
	const q = `UPDATE admin_accounts SET password_hash = ?, token_version = token_version + 1, updated_at = NOW()
		WHERE id = ? AND store_id = ? AND role = 'cashier'`
	res, err := r.db.ExecContext(ctx, q, passwordHash, id, storeID)
	if err != nil {
		return AdminAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetCashier(ctx, storeID, id); err != nil {
			return AdminAccount{}, err
		}
	}
	return r.GetCashier(ctx, storeID, id)
}

func (r *sqlRepository) GetStaffAccount(ctx context.Context, storeID, id int64) (StaffAccount, error) {
	const q = `SELECT sa.id, COALESCE(sa.member_id,0), sa.name, COALESCE(m.phone,''), sa.store_id, COALESCE(s.name,''), sa.status, sa.created_at
		FROM staff_accounts sa
		LEFT JOIN stores s ON s.id = sa.store_id
		LEFT JOIN members m ON m.id = sa.member_id
		WHERE sa.id = ? AND sa.store_id = ?`
	var a StaffAccount
	err := r.db.QueryRowContext(ctx, q, id, storeID).Scan(&a.ID, &a.MemberID, &a.Name, &a.Phone, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	return a, nil
}

// CreateStaffAccount binds an existing mini-program member (by id) as store
// staff, copying the member's openid into the staff_accounts row in a single
// INSERT...SELECT. The staff display name is the provided name, falling back to
// the member's nickname when blank. A member who does not exist inserts no row
// (NotFound); a member already bound as staff collides on the member_id/openid
// unique keys (Conflict).
func (r *sqlRepository) CreateStaffAccount(ctx context.Context, storeID, memberID int64, name string) (StaffAccount, error) {
	const q = `INSERT INTO staff_accounts (member_id, wechat_openid, name, store_id, status, token_version, created_at, updated_at)
		SELECT m.id, m.wechat_openid, COALESCE(NULLIF(?,''), NULLIF(m.nickname,''), '会员'), ?, 'active',
			CAST(UNIX_TIMESTAMP(NOW(6)) * 1000000 AS UNSIGNED), NOW(), NOW()
		FROM members m WHERE m.id = ? AND m.status = 'active'`
	res, err := r.db.ExecContext(ctx, q, name, storeID, memberID)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return StaffAccount{}, apperr.Conflict("该会员已被绑定为员工")
		}
		return StaffAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return StaffAccount{}, apperr.NotFound("member not found")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	return r.GetStaffAccount(ctx, storeID, id)
}

func (r *sqlRepository) UpdateStaffAccount(ctx context.Context, storeID, id int64, name string) (StaffAccount, error) {
	const q = `UPDATE staff_accounts SET name = ?, updated_at = NOW() WHERE id = ? AND store_id = ?`
	res, err := r.db.ExecContext(ctx, q, name, id, storeID)
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetStaffAccount(ctx, storeID, id); err != nil {
			return StaffAccount{}, err
		}
	}
	return r.GetStaffAccount(ctx, storeID, id)
}

// DisableStaffAccount marks a staff account disabled and bumps token_version so
// any outstanding session is invalidated on its next refresh.
func (r *sqlRepository) DisableStaffAccount(ctx context.Context, storeID, id int64) (StaffAccount, error) {
	const q = `UPDATE staff_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW()
		WHERE id = ? AND store_id = ?`
	res, err := r.db.ExecContext(ctx, q, id, storeID)
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetStaffAccount(ctx, storeID, id); err != nil {
			return StaffAccount{}, err
		}
	}
	return r.GetStaffAccount(ctx, storeID, id)
}

// GetStaffAccountByID looks up a staff account by id alone, for the admin
// console which is not pinned to a single store.
func (r *sqlRepository) GetStaffAccountByID(ctx context.Context, id int64) (StaffAccount, error) {
	const q = `SELECT sa.id, COALESCE(sa.member_id,0), sa.name, COALESCE(m.phone,''), sa.store_id, COALESCE(s.name,''), sa.status, sa.created_at
		FROM staff_accounts sa
		LEFT JOIN stores s ON s.id = sa.store_id
		LEFT JOIN members m ON m.id = sa.member_id
		WHERE sa.id = ?`
	var a StaffAccount
	err := r.db.QueryRowContext(ctx, q, id).Scan(&a.ID, &a.MemberID, &a.Name, &a.Phone, &a.StoreID, &a.StoreName, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return StaffAccount{}, apperr.NotFound("staff account not found")
	}
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	return a, nil
}

// DeleteStaffAccount removes the staff binding row scoped to the caller's own
// store. Only the staff_accounts row is deleted — the underlying member account
// is untouched. NotFound when no matching row exists.
func (r *sqlRepository) DeleteStaffAccount(ctx context.Context, storeID, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM staff_accounts WHERE id = ? AND store_id = ?`, id, storeID)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("staff account not found")
	}
	return nil
}

// DeleteStaffAccountByID removes any staff binding row (headquarters, not
// store-scoped). Only the staff_accounts row is deleted; the member is untouched.
func (r *sqlRepository) DeleteStaffAccountByID(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM staff_accounts WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("staff account not found")
	}
	return nil
}

// UpdateStaffAccountByID applies a partial update (name and/or store_id,
// reassigning the staff member to a different store) to any staff account.
func (r *sqlRepository) UpdateStaffAccountByID(ctx context.Context, id int64, storeID *int64, name *string) (StaffAccount, error) {
	set := "updated_at = NOW()"
	var args []any
	if name != nil {
		set = "name = ?, " + set
		args = append(args, *name)
	}
	if storeID != nil {
		// A staff token embeds its store scope. Reassignment must invalidate the
		// old token immediately so it cannot continue operating on the old store.
		set = "store_id = ?, token_version = token_version + 1, " + set
		args = append(args, *storeID)
	}
	args = append(args, id)
	res, err := r.db.ExecContext(ctx, `UPDATE staff_accounts SET `+set+` WHERE id = ?`, args...)
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetStaffAccountByID(ctx, id); err != nil {
			return StaffAccount{}, err
		}
	}
	return r.GetStaffAccountByID(ctx, id)
}

// DisableStaffAccountByID marks any staff account disabled and bumps
// token_version so any outstanding session is invalidated on its next refresh.
func (r *sqlRepository) DisableStaffAccountByID(ctx context.Context, id int64) (StaffAccount, error) {
	const q = `UPDATE staff_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW()
		WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return StaffAccount{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetStaffAccountByID(ctx, id); err != nil {
			return StaffAccount{}, err
		}
	}
	return r.GetStaffAccountByID(ctx, id)
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(s rowScanner) (RuleDefinition, error) {
	var rd RuleDefinition
	var cfg []byte
	if err := s.Scan(&rd.ID, &rd.Key, &rd.ScopeType, &rd.StoreID, &rd.Version, &cfg, &rd.Enabled, &rd.Status, &rd.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return RuleDefinition{}, apperr.NotFound("admin: rule definition not found")
		}
		return RuleDefinition{}, apperr.Internal(err)
	}
	rd.ConfigJSON = cfg
	return rd, nil
}
