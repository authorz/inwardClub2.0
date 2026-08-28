package coupon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/audit"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// ConsoleScope pins a console coupon-template query to a single store, or
// leaves it nil for the admin console (all scopes). Store handlers always
// build this from storescope.MustFromContext; it is never taken from a
// client-supplied storeId.
type ConsoleScope struct {
	StoreID *int64
}

// Template is a coupon template row as stored in coupon_templates.
type Template struct {
	ID             int64
	ScopeType      string
	StoreID        *int64
	Name           string
	Description    string
	CategoryID     int64
	CategoryName   string
	CouponType     string
	AdmissionCount int
	ValueCent      int64
	PointsPrice    int64
	StockQty       int64
	IssuedQty      int64
	PerMemberLim   int64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ApplicableScope is the best-effort decoded shape of coupon_templates.applicable_scope.
type ApplicableScope struct {
	ItemIDs     []int64 `json:"itemIds,omitempty"`
	CategoryIDs []int64 `json:"categoryIds,omitempty"`
}

// TemplateInput is the create/update body for a coupon template.
type TemplateInput struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	CategoryID     int64  `json:"categoryId" binding:"required"`
	AdmissionCount int    `json:"admissionCount"`
}

// GrantRequest grants an entitlement to a member from a template.
type GrantRequest struct {
	TemplateID int64      `json:"templateId" binding:"required"`
	MemberID   int64      `json:"memberId"`
	ScopeType  string     `json:"scopeType"`
	StoreID    *int64     `json:"storeId"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	Reason     string     `json:"reason"`
}

// VoidRequest voids an existing entitlement.
type VoidRequest struct {
	EntitlementID int64  `json:"entitlementId" binding:"required"`
	Reason        string `json:"reason"`
}

// VerifyRequest verifies/redeems an entitlement at a store.
type VerifyRequest struct {
	EntitlementNo string `json:"entitlementNo"`
	EntitlementID int64  `json:"entitlementId"`
	StoreID       int64  `json:"storeId" binding:"required"`
}

// ConsoleTemplateView is the console representation of a coupon template.
type ConsoleTemplateView struct {
	ID             int64     `json:"id"`
	ScopeType      string    `json:"scopeType"`
	StoreID        *int64    `json:"storeId,omitempty"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CategoryID     int64     `json:"categoryId"`
	CategoryName   string    `json:"categoryName"`
	CouponType     string    `json:"couponType"`
	AdmissionCount int       `json:"admissionCount"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ApplicableItemsView is the console representation of a template's
// configured applicable item/category scope.
type ApplicableItemsView struct {
	TemplateID  int64   `json:"templateId"`
	ItemIDs     []int64 `json:"itemIds"`
	CategoryIDs []int64 `json:"categoryIds"`
}

// EntitlementView is the console representation of an entitlement after a
// write action (grant/void/verify).
type EntitlementView struct {
	EntitlementID int64  `json:"entitlementId"`
	EntitlementNo string `json:"entitlementNo"`
	Status        string `json:"status"`
}

// ConsoleEntitlementView is one member-owned coupon shown in the admin console.
type ConsoleEntitlementView struct {
	EntitlementID int64      `json:"entitlementId"`
	EntitlementNo string     `json:"entitlementNo"`
	TemplateID    int64      `json:"templateId"`
	TemplateName  string     `json:"templateName"`
	CouponType    string     `json:"couponType"`
	MemberID      int64      `json:"memberId"`
	StoreID       *int64     `json:"storeId,omitempty"`
	StoreName     string     `json:"storeName,omitempty"`
	Status        string     `json:"status"`
	GrantedReason string     `json:"grantedReason,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// UpdateEntitlementExpiryRequest changes an unused coupon's expiry.
type UpdateEntitlementExpiryRequest struct {
	ExpiresAt time.Time `json:"expiresAt" binding:"required"`
	Reason    string    `json:"reason" binding:"required"`
}

// EntitlementReasonRequest captures the mandatory audit reason for revocation.
type EntitlementReasonRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ConsoleRepository is the console persistence port for coupon templates and
// their write actions. Grant/Void/Verify need a generated entitlement number,
// idempotency handling and a transaction, and land in a later milestone.
type ConsoleRepository interface {
	ListTemplates(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]Template, int64, error)
	GetTemplate(ctx context.Context, scope ConsoleScope, id int64) (Template, error)
	CreateTemplate(ctx context.Context, scope ConsoleScope, in TemplateInput) (Template, error)
	UpdateTemplate(ctx context.Context, scope ConsoleScope, id int64, in TemplateInput) (Template, error)
	SetTemplateStatus(ctx context.Context, scope ConsoleScope, id int64, status string) (Template, error)
	DeleteTemplate(ctx context.Context, scope ConsoleScope, id int64) error
	GetApplicableScope(ctx context.Context, scope ConsoleScope, templateID int64) (ApplicableScope, error)
	ListMemberEntitlements(ctx context.Context, scope ConsoleScope, memberID int64, page httpx.Page) ([]ConsoleEntitlementView, int64, error)
	GrantMemberEntitlement(ctx context.Context, scope ConsoleScope, memberID int64, req GrantRequest, idemKey string, entry audit.Entry) (EntitlementView, error)
	UpdateMemberEntitlementExpiry(ctx context.Context, memberID, entitlementID int64, req UpdateEntitlementExpiryRequest, idemKey string, entry audit.Entry) (EntitlementView, error)
	VoidMemberEntitlement(ctx context.Context, memberID, entitlementID int64, reason, idemKey string, entry audit.Entry) (EntitlementView, error)
	Grant(ctx context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error)
	Void(ctx context.Context, scope ConsoleScope, req VoidRequest) (EntitlementView, error)
	Verify(ctx context.Context, scope ConsoleScope, req VerifyRequest) (EntitlementView, error)
}

type sqlConsoleRepository struct{ db *platdb.DB }

// NewConsoleRepository builds the MySQL console coupon-template repository.
func NewConsoleRepository(db *platdb.DB) ConsoleRepository { return &sqlConsoleRepository{db: db} }

const templateSelect = `SELECT id, scope_type, store_id, name, COALESCE(description,''),
	category_id, COALESCE((SELECT name FROM coupon_categories WHERE id = coupon_templates.category_id), ''),
	coupon_type, admission_count, value_cent, points_price, stock_quantity, issued_quantity,
	per_member_limit, status, created_at, updated_at
	FROM coupon_templates`

func scopeWhere(scope ConsoleScope) (string, []any) {
	if scope.StoreID != nil {
		return ` WHERE scope_type = 'store' AND store_id = ?`, []any{*scope.StoreID}
	}
	return ``, nil
}

func (r *sqlConsoleRepository) ListTemplates(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]Template, int64, error) {
	where, args := scopeWhere(scope)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_templates`+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := templateSelect + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	qArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, q, qArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (r *sqlConsoleRepository) GetTemplate(ctx context.Context, scope ConsoleScope, id int64) (Template, error) {
	where, args := scopeWhere(scope)
	var q string
	if where == "" {
		q = templateSelect + ` WHERE id = ?`
		args = []any{id}
	} else {
		q = templateSelect + where + ` AND id = ?`
		args = append(append([]any{}, args...), id)
	}
	t, err := scanTemplate(r.db.QueryRowContext(ctx, q, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Template{}, apperr.NotFound("coupon template not found")
		}
		return Template{}, apperr.Internal(err)
	}
	return t, nil
}

func (r *sqlConsoleRepository) GetApplicableScope(ctx context.Context, scope ConsoleScope, templateID int64) (ApplicableScope, error) {
	if _, err := r.GetTemplate(ctx, scope, templateID); err != nil {
		return ApplicableScope{}, err
	}
	var raw sql.NullString
	if err := r.db.QueryRowContext(ctx,
		`SELECT applicable_scope FROM coupon_templates WHERE id = ?`, templateID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplicableScope{}, apperr.NotFound("coupon template not found")
		}
		return ApplicableScope{}, apperr.Internal(err)
	}
	var out ApplicableScope
	if raw.Valid && raw.String != "" {
		if err := json.Unmarshal([]byte(raw.String), &out); err != nil {
			// Best-effort: an unparsable scope is reported as empty rather than failing.
			return ApplicableScope{}, nil
		}
	}
	return out, nil
}

func validCouponType(t string) bool {
	switch t {
	case TypeEventTicket, TypeAdmissionTicket, TypeSnack, TypeAlcohol, TypeBeverage, TypeDrink, TypeMeal, TypeGift:
		return true
	default:
		return false
	}
}

func (r *sqlConsoleRepository) templateCategory(ctx context.Context, categoryID int64, includeDisabled bool) (CouponCategory, error) {
	if categoryID <= 0 {
		return CouponCategory{}, apperr.Invalid("请选择券类型")
	}
	q := `SELECT id, name, business_type, sort_order, status, gift_daily_usage_limit, created_at, updated_at
		FROM coupon_categories WHERE id = ?`
	if !includeDisabled {
		q += ` AND status = 'active'`
	}
	var category CouponCategory
	err := r.db.QueryRowContext(ctx, q, categoryID).Scan(
		&category.ID, &category.Name, &category.BusinessType, &category.SortOrder,
		&category.Status, &category.GiftDailyUsageLimit, &category.CreatedAt, &category.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return CouponCategory{}, apperr.Invalid("所选券类型不存在或已停用")
	}
	if err != nil {
		return CouponCategory{}, apperr.Internal(err)
	}
	if !validCouponType(category.BusinessType) {
		return CouponCategory{}, apperr.Invalid("券类型绑定的兑换场景不正确")
	}
	return category, nil
}

func normalizedAdmissionCount(couponType string, admissionCount int) (int, error) {
	if couponType != TypeAdmissionTicket {
		return 1, nil
	}
	if admissionCount < 1 || admissionCount > 99 {
		return 0, apperr.Invalid("门票券可兑人数必须在 1 到 99 之间")
	}
	return admissionCount, nil
}

func (r *sqlConsoleRepository) CreateTemplate(ctx context.Context, scope ConsoleScope, in TemplateInput) (Template, error) {
	if in.Name == "" {
		return Template{}, apperr.Invalid("请填写优惠券名称")
	}
	category, err := r.templateCategory(ctx, in.CategoryID, false)
	if err != nil {
		return Template{}, err
	}
	admissionCount, err := normalizedAdmissionCount(category.BusinessType, in.AdmissionCount)
	if err != nil {
		return Template{}, err
	}
	now := time.Now().UTC()
	// A store console pins scope_type='store' + its own store id; the admin
	// console creates global templates.
	scopeType := "global"
	var storeID any
	if scope.StoreID != nil {
		scopeType = "store"
		storeID = *scope.StoreID
	}
	// All issued coupons have a fixed 30-day validity period.
	const q = `INSERT INTO coupon_templates
		(scope_type, store_id, name, description, coupon_type, category_id, admission_count, value_cent, points_price,
		 stock_quantity, issued_quantity, validity_rule, applicable_scope, per_member_limit,
		 status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, JSON_OBJECT('days', 30), '{}', ?, 'draft', ?, ?)`
	res, err := r.db.ExecContext(ctx, q, scopeType, storeID, in.Name, in.Description, category.BusinessType, category.ID,
		admissionCount, 0, 0, 0, 0, now, now)
	if err != nil {
		return Template{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Template{}, apperr.Internal(err)
	}
	return r.GetTemplate(ctx, scope, id)
}

func (r *sqlConsoleRepository) UpdateTemplate(ctx context.Context, scope ConsoleScope, id int64, in TemplateInput) (Template, error) {
	if in.Name == "" {
		return Template{}, apperr.Invalid("请填写优惠券名称")
	}
	existing, err := r.GetTemplate(ctx, scope, id)
	if err != nil {
		return Template{}, err
	}
	category, err := r.templateCategory(ctx, in.CategoryID, in.CategoryID == existing.CategoryID)
	if err != nil {
		return Template{}, err
	}
	admissionCount, err := normalizedAdmissionCount(category.BusinessType, in.AdmissionCount)
	if err != nil {
		return Template{}, err
	}
	now := time.Now().UTC()
	q := `UPDATE coupon_templates SET name=?, description=?, coupon_type=?, category_id=?, admission_count=?, value_cent=0,
		points_price=0, stock_quantity=0, per_member_limit=0, updated_at=? WHERE id=?`
	args := []any{in.Name, in.Description, category.BusinessType, category.ID, admissionCount, now, id}
	if scope.StoreID != nil {
		q += ` AND scope_type='store' AND store_id=?`
		args = append(args, *scope.StoreID)
	}
	if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
		return Template{}, apperr.Internal(err)
	}
	// Re-read within scope; an out-of-scope id matched no row and surfaces as
	// NOT_FOUND here.
	return r.GetTemplate(ctx, scope, id)
}

func (r *sqlConsoleRepository) DeleteTemplate(ctx context.Context, scope ConsoleScope, id int64) error {
	tmpl, err := r.GetTemplate(ctx, scope, id)
	if err != nil {
		return err
	}
	if tmpl.IssuedQty > 0 {
		return apperr.Conflict("该优惠券已有发放记录，不能删除，可改为停用")
	}
	var bound int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recharge_products WHERE coupon_template_id = ?`, id,
	).Scan(&bound); err != nil {
		return apperr.Internal(err)
	}
	if bound > 0 {
		return apperr.Conflict("该优惠券已绑定充值档位，请先解除绑定")
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM catalog_items WHERE grant_coupon_template_id = ?`, id,
	).Scan(&bound); err != nil {
		return apperr.Internal(err)
	}
	if bound > 0 {
		return apperr.Conflict("该优惠券已绑定在售券商品，请先解除绑定")
	}
	q := `DELETE FROM coupon_templates WHERE id=?`
	args := []any{id}
	if scope.StoreID != nil {
		q += ` AND scope_type='store' AND store_id=?`
		args = append(args, *scope.StoreID)
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("coupon template not found")
	}
	return nil
}

func (r *sqlConsoleRepository) SetTemplateStatus(ctx context.Context, scope ConsoleScope, id int64, status string) (Template, error) {
	if status != "published" && status != "disabled" {
		return Template{}, apperr.Invalid("优惠券状态不正确")
	}
	if _, err := r.GetTemplate(ctx, scope, id); err != nil {
		return Template{}, err
	}
	if status == "disabled" {
		var bound int64
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM recharge_products WHERE coupon_template_id = ? AND status = 'active'`, id,
		).Scan(&bound); err != nil {
			return Template{}, apperr.Internal(err)
		}
		if bound > 0 {
			return Template{}, apperr.Conflict("该优惠券已绑定启用中的充值档位，不能停用")
		}
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM catalog_items WHERE grant_coupon_template_id = ? AND status = 'published'`, id,
		).Scan(&bound); err != nil {
			return Template{}, apperr.Internal(err)
		}
		if bound > 0 {
			return Template{}, apperr.Conflict("该优惠券已绑定已发布的券商品，不能停用")
		}
	}
	q := `UPDATE coupon_templates SET status = ?, updated_at = ? WHERE id = ?`
	args := []any{status, time.Now().UTC(), id}
	if scope.StoreID != nil {
		q += ` AND scope_type = 'store' AND store_id = ?`
		args = append(args, *scope.StoreID)
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return Template{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetTemplate(ctx, scope, id); err != nil {
			return Template{}, err
		}
	}
	return r.GetTemplate(ctx, scope, id)
}

// ListMemberEntitlements returns one member's coupons within the requested
// console scope. Store callers can only see entitlements bound to their store.
func (r *sqlConsoleRepository) ListMemberEntitlements(ctx context.Context, scope ConsoleScope, memberID int64, page httpx.Page) ([]ConsoleEntitlementView, int64, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE id = ?`, memberID).Scan(&exists); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if exists == 0 {
		return nil, 0, apperr.NotFound("member not found")
	}
	where := ` WHERE e.member_id = ?`
	args := []any{memberID}
	if scope.StoreID != nil {
		where += ` AND e.store_id = ?`
		args = append(args, *scope.StoreID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_entitlements e`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const selectSQL = `SELECT e.id, e.entitlement_no, e.coupon_template_id, ct.name, ct.coupon_type,
		e.member_id, e.store_id, COALESCE(s.name, ''), e.status,
		COALESCE(e.granted_reason, ''), e.expires_at, e.created_at, e.updated_at
		FROM coupon_entitlements e
		JOIN coupon_templates ct ON ct.id = e.coupon_template_id
		LEFT JOIN stores s ON s.id = e.store_id`
	queryArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, selectSQL+where+` ORDER BY e.id DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	views := make([]ConsoleEntitlementView, 0, page.Limit())
	for rows.Next() {
		var view ConsoleEntitlementView
		if err := rows.Scan(
			&view.EntitlementID, &view.EntitlementNo, &view.TemplateID, &view.TemplateName,
			&view.CouponType, &view.MemberID, &view.StoreID, &view.StoreName, &view.Status,
			&view.GrantedReason, &view.ExpiresAt, &view.CreatedAt, &view.UpdatedAt,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return views, total, nil
}

// Grant issues one entitlement to a member from a template, enforcing the
// template's stock and per-member limit under a row lock.
func (r *sqlConsoleRepository) Grant(ctx context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error) {
	return r.grantEntitlement(ctx, scope, req, "", nil, false)
}

func (r *sqlConsoleRepository) GrantMemberEntitlement(
	ctx context.Context,
	scope ConsoleScope,
	memberID int64,
	req GrantRequest,
	idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	req.MemberID = memberID
	return r.grantEntitlement(ctx, scope, req, idemKey, &entry, true)
}

func (r *sqlConsoleRepository) grantEntitlement(
	ctx context.Context,
	scope ConsoleScope,
	req GrantRequest,
	idemKey string,
	entry *audit.Entry,
	requirePublished bool,
) (EntitlementView, error) {
	tmpl, err := r.GetTemplate(ctx, scope, req.TemplateID)
	if err != nil {
		return EntitlementView{}, err
	}
	if requirePublished && tmpl.Status != "published" {
		return EntitlementView{}, apperr.Conflict("只能补发已发布的优惠券")
	}
	entitlementStoreID := tmpl.StoreID
	if requirePublished {
		entitlementStoreID, err = resolveMemberGrantStore(tmpl, req)
		if err != nil {
			return EntitlementView{}, err
		}
	}
	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, 30)
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt.UTC()
	}
	if !expiresAt.After(now) {
		return EntitlementView{}, apperr.Invalid("优惠券有效期必须晚于当前时间")
	}
	entNo := fmt.Sprintf("E%d-%d", req.TemplateID, now.UnixNano())
	grantedBy := "admin"
	if scope.StoreID != nil {
		grantedBy = "store"
	}
	var view EntitlementView
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if idemKey != "" {
			idemScope := "admin/member-coupon-grant"
			if scope.StoreID != nil {
				idemScope = "store/member-coupon-grant"
			}
			if err := idempotency.Claim(ctx, tx, idemScope, idemKey, "member", req.MemberID); err != nil {
				return err
			}
		}
		var memberExists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE id = ?`, req.MemberID).Scan(&memberExists); err != nil {
			return apperr.Internal(err)
		}
		if memberExists == 0 {
			return apperr.NotFound("member not found")
		}
		var (
			stock, issued, perLimit int64
			lockedStatus            string
		)
		if err := tx.QueryRowContext(ctx,
			`SELECT stock_quantity, issued_quantity, per_member_limit, status
			 FROM coupon_templates WHERE id = ? FOR UPDATE`,
			req.TemplateID).Scan(&stock, &issued, &perLimit, &lockedStatus); err != nil {
			return apperr.Internal(err)
		}
		if requirePublished && lockedStatus != "published" {
			return apperr.Conflict("只能补发已发布的优惠券")
		}
		if entitlementStoreID != nil {
			var storeExists int
			if err := tx.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM stores WHERE id = ? AND status = 'active'`,
				*entitlementStoreID,
			).Scan(&storeExists); err != nil {
				return apperr.Internal(err)
			}
			if storeExists == 0 {
				return apperr.Invalid("所选门店不存在或未启用")
			}
		}
		if stock > 0 && issued >= stock {
			return apperr.Conflict("coupon template is out of stock")
		}
		if perLimit > 0 {
			var held int64
			if err := tx.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM coupon_entitlements
					WHERE coupon_template_id = ? AND member_id = ? AND status IN ('active','used')`,
				req.TemplateID, req.MemberID).Scan(&held); err != nil {
				return apperr.Internal(err)
			}
			if held >= perLimit {
				return apperr.Conflict("member has reached the coupon limit")
			}
		}
		const ins = `INSERT INTO coupon_entitlements
			(entitlement_no, coupon_template_id, admission_count, member_id, store_id, status, granted_reason,
			 granted_by_type, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, ins, entNo, req.TemplateID, tmpl.AdmissionCount, req.MemberID, entitlementStoreID,
			req.Reason, grantedBy, expiresAt, now, now)
		if err != nil {
			return apperr.Internal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE coupon_templates SET issued_quantity = issued_quantity + 1, updated_at = ? WHERE id = ?`,
			now, req.TemplateID); err != nil {
			return apperr.Internal(err)
		}
		view = EntitlementView{EntitlementID: id, EntitlementNo: entNo, Status: StatusActive}
		if entry != nil {
			entry.TargetType = "member"
			entry.TargetID = req.MemberID
			if entitlementStoreID != nil {
				entry.StoreID = *entitlementStoreID
			}
			entry.Reason = strings.TrimSpace(req.Reason)
			entry.After = map[string]any{
				"entitlementId": id, "entitlementNo": entNo, "templateId": req.TemplateID,
				"scopeType": req.ScopeType, "storeId": entitlementStoreID,
				"status": StatusActive, "expiresAt": expiresAt,
			}
			if err := audit.RecordTx(ctx, tx, *entry); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
}

func resolveMemberGrantStore(tmpl Template, req GrantRequest) (*int64, error) {
	switch req.ScopeType {
	case "global":
		if req.StoreID != nil {
			return nil, apperr.Invalid("全部门店券不能指定门店")
		}
		if tmpl.ScopeType != "global" || tmpl.StoreID != nil {
			return nil, apperr.Invalid("门店专属券不能补发为全部门店券")
		}
		return nil, nil
	case "store":
		if req.StoreID == nil || *req.StoreID <= 0 {
			return nil, apperr.Invalid("请选择适用门店")
		}
		if tmpl.ScopeType == "store" && (tmpl.StoreID == nil || *tmpl.StoreID != *req.StoreID) {
			return nil, apperr.Invalid("门店专属券只能补发到所属门店")
		}
		storeID := *req.StoreID
		return &storeID, nil
	default:
		return nil, apperr.Invalid("请选择全部门店或指定门店")
	}
}

// Void marks an active entitlement void. A store console may only void an
// entitlement bound to its own store; the admin console may void any.
func (r *sqlConsoleRepository) Void(ctx context.Context, scope ConsoleScope, req VoidRequest) (EntitlementView, error) {
	now := time.Now().UTC()
	var view EntitlementView
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			entNo   string
			status  string
			storeID sql.NullInt64
		)
		err := tx.QueryRowContext(ctx,
			`SELECT entitlement_no, status, store_id FROM coupon_entitlements WHERE id = ? FOR UPDATE`,
			req.EntitlementID).Scan(&entNo, &status, &storeID)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("coupon entitlement not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if scope.StoreID != nil && (!storeID.Valid || storeID.Int64 != *scope.StoreID) {
			return apperr.NotFound("coupon entitlement not found")
		}
		if status != StatusActive {
			return apperr.Conflict("coupon entitlement is not active")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE coupon_entitlements SET status = ?, updated_at = ? WHERE id = ?`,
			StatusVoid, now, req.EntitlementID); err != nil {
			return apperr.Internal(err)
		}
		view = EntitlementView{EntitlementID: req.EntitlementID, EntitlementNo: entNo, Status: StatusVoid}
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
}

func (r *sqlConsoleRepository) UpdateMemberEntitlementExpiry(
	ctx context.Context,
	memberID, entitlementID int64,
	req UpdateEntitlementExpiryRequest,
	idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	now := time.Now().UTC()
	expiresAt := req.ExpiresAt.UTC()
	if !expiresAt.After(now) {
		return EntitlementView{}, apperr.Invalid("优惠券有效期必须晚于当前时间")
	}
	var view EntitlementView
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if idemKey != "" {
			if err := idempotency.Claim(
				ctx, tx, "admin/member-coupon-expiry", idemKey, "coupon_entitlement", entitlementID,
			); err != nil {
				return err
			}
		}
		var (
			entNo     string
			status    string
			oldExpiry sql.NullTime
		)
		err := tx.QueryRowContext(ctx,
			`SELECT entitlement_no, status, expires_at FROM coupon_entitlements
			 WHERE id = ? AND member_id = ? FOR UPDATE`, entitlementID, memberID,
		).Scan(&entNo, &status, &oldExpiry)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("coupon entitlement not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if status != StatusActive && status != StatusExpired {
			return apperr.Conflict("已使用或已删除的优惠券不能修改有效期")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE coupon_entitlements SET status = ?, expires_at = ?, updated_at = ?
			 WHERE id = ? AND member_id = ?`,
			StatusActive, expiresAt, now, entitlementID, memberID,
		); err != nil {
			return apperr.Internal(err)
		}
		entry.TargetType = "member"
		entry.TargetID = memberID
		entry.Reason = strings.TrimSpace(req.Reason)
		entry.Before = map[string]any{"entitlementId": entitlementID, "status": status, "expiresAt": nullableTime(oldExpiry)}
		entry.After = map[string]any{"entitlementId": entitlementID, "status": StatusActive, "expiresAt": expiresAt}
		if err := audit.RecordTx(ctx, tx, entry); err != nil {
			return err
		}
		view = EntitlementView{EntitlementID: entitlementID, EntitlementNo: entNo, Status: StatusActive}
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
}

func (r *sqlConsoleRepository) VoidMemberEntitlement(
	ctx context.Context,
	memberID, entitlementID int64,
	reason, idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	now := time.Now().UTC()
	var view EntitlementView
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if idemKey != "" {
			if err := idempotency.Claim(
				ctx, tx, "admin/member-coupon-void", idemKey, "coupon_entitlement", entitlementID,
			); err != nil {
				return err
			}
		}
		var entNo, status string
		err := tx.QueryRowContext(ctx,
			`SELECT entitlement_no, status FROM coupon_entitlements
			 WHERE id = ? AND member_id = ? FOR UPDATE`, entitlementID, memberID,
		).Scan(&entNo, &status)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("coupon entitlement not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if status != StatusActive && status != StatusExpired {
			return apperr.Conflict("已使用或已删除的优惠券不能删除")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE coupon_entitlements SET status = ?, updated_at = ?
			 WHERE id = ? AND member_id = ?`,
			StatusVoid, now, entitlementID, memberID,
		); err != nil {
			return apperr.Internal(err)
		}
		entry.TargetType = "member"
		entry.TargetID = memberID
		entry.Reason = strings.TrimSpace(reason)
		entry.Before = map[string]any{"entitlementId": entitlementID, "status": status}
		entry.After = map[string]any{"entitlementId": entitlementID, "status": StatusVoid}
		if err := audit.RecordTx(ctx, tx, entry); err != nil {
			return err
		}
		view = EntitlementView{EntitlementID: entitlementID, EntitlementNo: entNo, Status: StatusVoid}
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
}

func nullableTime(v sql.NullTime) any {
	if !v.Valid {
		return nil
	}
	return v.Time
}

// Verify redeems an active entitlement at a store, recording a redemption and
// flipping the entitlement to used in one transaction. A store console redeems
// against its own store; the admin console uses the store id from the request.
// A store-bound entitlement can only be redeemed by its owning store.
func (r *sqlConsoleRepository) Verify(ctx context.Context, scope ConsoleScope, req VerifyRequest) (EntitlementView, error) {
	if req.EntitlementID == 0 && req.EntitlementNo == "" {
		return EntitlementView{}, apperr.Invalid("entitlementId or entitlementNo is required")
	}
	storeID := req.StoreID
	if scope.StoreID != nil {
		storeID = *scope.StoreID
	}
	now := time.Now().UTC()
	var view EntitlementView
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		const cols = `id, entitlement_no, status, member_id, coupon_template_id, store_id, expires_at`
		var (
			sel string
			arg any
		)
		if req.EntitlementID != 0 {
			sel = `SELECT ` + cols + ` FROM coupon_entitlements WHERE id = ? FOR UPDATE`
			arg = req.EntitlementID
		} else {
			sel = `SELECT ` + cols + ` FROM coupon_entitlements WHERE entitlement_no = ? FOR UPDATE`
			arg = req.EntitlementNo
		}
		var (
			id         int64
			entNo      string
			status     string
			memberID   int64
			templateID int64
			entStore   sql.NullInt64
			expiresAt  sql.NullTime
		)
		err := tx.QueryRowContext(ctx, sel, arg).
			Scan(&id, &entNo, &status, &memberID, &templateID, &entStore, &expiresAt)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("coupon entitlement not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if scope.StoreID != nil && entStore.Valid && entStore.Int64 != *scope.StoreID {
			return apperr.NotFound("coupon entitlement not found")
		}
		if status != StatusActive {
			return apperr.Conflict("coupon entitlement is not redeemable")
		}
		if expiresAt.Valid && !expiresAt.Time.After(now) {
			return apperr.Conflict("coupon entitlement has expired")
		}
		if err := ClaimGiftDailyUsage(ctx, tx, memberID, id, now); err != nil {
			return err
		}
		redNo := fmt.Sprintf("R%d-%d", id, now.UnixNano())
		const ins = `INSERT INTO coupon_redemptions
			(redemption_no, entitlement_id, coupon_template_id, member_id, store_id, verified_by_type, created_at)
			VALUES (?, ?, ?, ?, ?, 'staff', ?)`
		if _, err := tx.ExecContext(ctx, ins, redNo, id, templateID, memberID, storeID, now); err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("coupon already redeemed")
			}
			return apperr.Internal(err)
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE coupon_entitlements SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
			StatusUsed, now, id, StatusActive)
		if err != nil {
			return apperr.Internal(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return apperr.Conflict("coupon entitlement is not redeemable")
		}
		view = EntitlementView{EntitlementID: id, EntitlementNo: entNo, Status: StatusUsed}
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
}

func scanTemplate(s scanner) (Template, error) {
	var t Template
	err := s.Scan(&t.ID, &t.ScopeType, &t.StoreID, &t.Name, &t.Description,
		&t.CategoryID, &t.CategoryName, &t.CouponType, &t.AdmissionCount,
		&t.ValueCent, &t.PointsPrice, &t.StockQty, &t.IssuedQty,
		&t.PerMemberLim, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// ConsoleService provides console CRUD over coupon templates plus the
// grant/void/verify write actions.
type ConsoleService struct {
	repo ConsoleRepository
}

// NewConsoleService builds the console coupon service.
func NewConsoleService(repo ConsoleRepository) *ConsoleService { return &ConsoleService{repo: repo} }

// ListTemplates returns coupon templates within scope.
func (s *ConsoleService) ListTemplates(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]ConsoleTemplateView, int64, error) {
	templates, total, err := s.repo.ListTemplates(ctx, scope, page)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ConsoleTemplateView, 0, len(templates))
	for _, t := range templates {
		views = append(views, templateView(t))
	}
	return views, total, nil
}

// GetTemplate returns one coupon template within scope.
func (s *ConsoleService) GetTemplate(ctx context.Context, scope ConsoleScope, id int64) (ConsoleTemplateView, error) {
	t, err := s.repo.GetTemplate(ctx, scope, id)
	if err != nil {
		return ConsoleTemplateView{}, err
	}
	return templateView(t), nil
}

// CreateTemplate creates a coupon template within scope.
func (s *ConsoleService) CreateTemplate(ctx context.Context, scope ConsoleScope, in TemplateInput) (ConsoleTemplateView, error) {
	t, err := s.repo.CreateTemplate(ctx, scope, in)
	if err != nil {
		return ConsoleTemplateView{}, err
	}
	return templateView(t), nil
}

// UpdateTemplate updates a coupon template within scope.
func (s *ConsoleService) UpdateTemplate(ctx context.Context, scope ConsoleScope, id int64, in TemplateInput) (ConsoleTemplateView, error) {
	t, err := s.repo.UpdateTemplate(ctx, scope, id, in)
	if err != nil {
		return ConsoleTemplateView{}, err
	}
	return templateView(t), nil
}

func (s *ConsoleService) PublishTemplate(ctx context.Context, scope ConsoleScope, id int64) (ConsoleTemplateView, error) {
	t, err := s.repo.SetTemplateStatus(ctx, scope, id, "published")
	if err != nil {
		return ConsoleTemplateView{}, err
	}
	return templateView(t), nil
}

func (s *ConsoleService) DisableTemplate(ctx context.Context, scope ConsoleScope, id int64) (ConsoleTemplateView, error) {
	t, err := s.repo.SetTemplateStatus(ctx, scope, id, "disabled")
	if err != nil {
		return ConsoleTemplateView{}, err
	}
	return templateView(t), nil
}

// DeleteTemplate deletes a coupon template within scope.
func (s *ConsoleService) DeleteTemplate(ctx context.Context, scope ConsoleScope, id int64) error {
	return s.repo.DeleteTemplate(ctx, scope, id)
}

// GetApplicableItems returns the applicable item/category scope for a template.
func (s *ConsoleService) GetApplicableItems(ctx context.Context, scope ConsoleScope, templateID int64) (ApplicableItemsView, error) {
	a, err := s.repo.GetApplicableScope(ctx, scope, templateID)
	if err != nil {
		return ApplicableItemsView{}, err
	}
	return ApplicableItemsView{TemplateID: templateID, ItemIDs: a.ItemIDs, CategoryIDs: a.CategoryIDs}, nil
}

func (s *ConsoleService) ListMemberEntitlements(ctx context.Context, scope ConsoleScope, memberID int64, page httpx.Page) ([]ConsoleEntitlementView, int64, error) {
	if memberID <= 0 {
		return nil, 0, apperr.Invalid("会员信息不正确")
	}
	return s.repo.ListMemberEntitlements(ctx, scope, memberID, page)
}

func (s *ConsoleService) GrantMemberEntitlement(
	ctx context.Context,
	scope ConsoleScope,
	memberID int64,
	req GrantRequest,
	idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	req.MemberID = memberID
	req.Reason = strings.TrimSpace(req.Reason)
	if memberID <= 0 || req.TemplateID <= 0 {
		return EntitlementView{}, apperr.Invalid("会员或优惠券信息不正确")
	}
	if req.Reason == "" {
		return EntitlementView{}, apperr.Invalid("请填写补发原因")
	}
	if scope.StoreID != nil {
		storeID := *scope.StoreID
		req.ScopeType = "store"
		req.StoreID = &storeID
	} else if req.ScopeType != "global" && req.ScopeType != "store" {
		return EntitlementView{}, apperr.Invalid("请选择全部门店或指定门店")
	}
	if req.ScopeType == "store" && (req.StoreID == nil || *req.StoreID <= 0) {
		return EntitlementView{}, apperr.Invalid("请选择适用门店")
	}
	if req.ScopeType == "global" && req.StoreID != nil {
		return EntitlementView{}, apperr.Invalid("全部门店券不能指定门店")
	}
	return s.repo.GrantMemberEntitlement(ctx, scope, memberID, req, idemKey, entry)
}

func (s *ConsoleService) UpdateMemberEntitlementExpiry(
	ctx context.Context,
	memberID, entitlementID int64,
	req UpdateEntitlementExpiryRequest,
	idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	req.Reason = strings.TrimSpace(req.Reason)
	if memberID <= 0 || entitlementID <= 0 {
		return EntitlementView{}, apperr.Invalid("会员或优惠券信息不正确")
	}
	if req.ExpiresAt.IsZero() || req.Reason == "" {
		return EntitlementView{}, apperr.Invalid("请填写新的有效期和修改原因")
	}
	return s.repo.UpdateMemberEntitlementExpiry(ctx, memberID, entitlementID, req, idemKey, entry)
}

func (s *ConsoleService) VoidMemberEntitlement(
	ctx context.Context,
	memberID, entitlementID int64,
	reason, idemKey string,
	entry audit.Entry,
) (EntitlementView, error) {
	reason = strings.TrimSpace(reason)
	if memberID <= 0 || entitlementID <= 0 {
		return EntitlementView{}, apperr.Invalid("会员或优惠券信息不正确")
	}
	if reason == "" {
		return EntitlementView{}, apperr.Invalid("请填写删除原因")
	}
	return s.repo.VoidMemberEntitlement(ctx, memberID, entitlementID, reason, idemKey, entry)
}

// Grant issues an entitlement to a member from a template.
func (s *ConsoleService) Grant(ctx context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error) {
	return s.repo.Grant(ctx, scope, req)
}

// Void voids an existing entitlement.
func (s *ConsoleService) Void(ctx context.Context, scope ConsoleScope, req VoidRequest) (EntitlementView, error) {
	return s.repo.Void(ctx, scope, req)
}

// Verify verifies/redeems an entitlement at a store.
func (s *ConsoleService) Verify(ctx context.Context, scope ConsoleScope, req VerifyRequest) (EntitlementView, error) {
	return s.repo.Verify(ctx, scope, req)
}

func templateView(t Template) ConsoleTemplateView {
	return ConsoleTemplateView{
		ID: t.ID, ScopeType: t.ScopeType, StoreID: t.StoreID,
		Name: t.Name, Description: t.Description, CategoryID: t.CategoryID, CategoryName: t.CategoryName,
		CouponType: t.CouponType, AdmissionCount: t.AdmissionCount,
		Status: t.Status, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

// ConsoleHandler exposes the console coupon-template endpoints, both the
// admin (unscoped) and store (scoped) variants. Router wiring lives outside
// this module.
type ConsoleHandler struct {
	svc *ConsoleService
}

// NewConsoleHandler builds the console coupon-template handler.
func NewConsoleHandler(svc *ConsoleService) *ConsoleHandler { return &ConsoleHandler{svc: svc} }

func templateID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid id")
	}
	return id, nil
}

func positivePathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}

// --- Admin console (audience: admin, no store filter) ---

// List handles GET /admin/coupon-templates.
func (h *ConsoleHandler) List(c *gin.Context) {
	h.list(c, ConsoleScope{})
}

// Get handles GET /admin/coupon-templates/:id.
func (h *ConsoleHandler) Get(c *gin.Context) {
	h.get(c, ConsoleScope{})
}

// Create handles POST /admin/coupon-templates.
func (h *ConsoleHandler) Create(c *gin.Context) {
	h.create(c, ConsoleScope{})
}

// Update handles PUT /admin/coupon-templates/:id.
func (h *ConsoleHandler) Update(c *gin.Context) {
	h.update(c, ConsoleScope{})
}

func (h *ConsoleHandler) Publish(c *gin.Context) { h.setStatus(c, ConsoleScope{}, true) }

func (h *ConsoleHandler) Disable(c *gin.Context) { h.setStatus(c, ConsoleScope{}, false) }

// Delete handles DELETE /admin/coupon-templates/:id.
func (h *ConsoleHandler) Delete(c *gin.Context) {
	h.delete(c, ConsoleScope{})
}

// ApplicableItems handles GET /admin/coupon-templates/:id/applicable-items.
func (h *ConsoleHandler) ApplicableItems(c *gin.Context) {
	h.applicableItems(c, ConsoleScope{})
}

// ListMemberEntitlements handles GET /admin/members/:memberID/coupon-entitlements.
func (h *ConsoleHandler) ListMemberEntitlements(c *gin.Context) {
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListMemberEntitlements(c.Request.Context(), ConsoleScope{}, memberID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// GrantMemberEntitlement handles POST /admin/members/:memberID/coupon-entitlements.
func (h *ConsoleHandler) GrantMemberEntitlement(c *gin.Context) {
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("优惠券补发参数不正确"))
		return
	}
	entry := audit.FromContext(c, "member.coupon.grant", "member", memberID)
	view, err := h.svc.GrantMemberEntitlement(
		c.Request.Context(), ConsoleScope{}, memberID, req, idempotency.Key(c), entry,
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// UpdateMemberEntitlementExpiry handles PATCH /admin/members/:memberID/coupon-entitlements/:entitlementID.
func (h *ConsoleHandler) UpdateMemberEntitlementExpiry(c *gin.Context) {
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	entitlementID, err := positivePathID(c, "entitlementID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req UpdateEntitlementExpiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("优惠券有效期参数不正确"))
		return
	}
	entry := audit.FromContext(c, "member.coupon.expiry.update", "member", memberID)
	view, err := h.svc.UpdateMemberEntitlementExpiry(
		c.Request.Context(), memberID, entitlementID, req, idempotency.Key(c), entry,
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// VoidMemberEntitlement handles POST /admin/members/:memberID/coupon-entitlements/:entitlementID/void.
func (h *ConsoleHandler) VoidMemberEntitlement(c *gin.Context) {
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	entitlementID, err := positivePathID(c, "entitlementID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req EntitlementReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请填写删除原因"))
		return
	}
	entry := audit.FromContext(c, "member.coupon.delete", "member", memberID)
	view, err := h.svc.VoidMemberEntitlement(
		c.Request.Context(), memberID, entitlementID, req.Reason, idempotency.Key(c), entry,
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// Void handles POST /admin/coupon-voids.
func (h *ConsoleHandler) Void(c *gin.Context) {
	h.void(c, ConsoleScope{})
}

// Verify handles POST /admin/coupon-verifications.
func (h *ConsoleHandler) Verify(c *gin.Context) {
	h.verify(c, ConsoleScope{})
}

// --- Store console (audience: store, scope pinned from JWT) ---

// StoreList handles GET /store/coupon-templates.
func (h *ConsoleHandler) StoreList(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.list(c, ConsoleScope{StoreID: &storeID})
}

// StoreGet handles GET /store/coupon-templates/:id.
func (h *ConsoleHandler) StoreGet(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.get(c, ConsoleScope{StoreID: &storeID})
}

// StoreCreate handles POST /store/coupon-templates.
func (h *ConsoleHandler) StoreCreate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.create(c, ConsoleScope{StoreID: &storeID})
}

// StoreUpdate handles PUT /store/coupon-templates/:id.
func (h *ConsoleHandler) StoreUpdate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.update(c, ConsoleScope{StoreID: &storeID})
}

func (h *ConsoleHandler) StorePublish(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.setStatus(c, ConsoleScope{StoreID: &storeID}, true)
}

func (h *ConsoleHandler) StoreDisable(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.setStatus(c, ConsoleScope{StoreID: &storeID}, false)
}

// StoreDelete handles DELETE /store/coupon-templates/:id.
func (h *ConsoleHandler) StoreDelete(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.delete(c, ConsoleScope{StoreID: &storeID})
}

// StoreApplicableItems handles GET /store/coupon-templates/:id/applicable-items.
func (h *ConsoleHandler) StoreApplicableItems(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.applicableItems(c, ConsoleScope{StoreID: &storeID})
}

// StoreListMemberEntitlements handles GET /store/members/:memberID/coupon-entitlements.
func (h *ConsoleHandler) StoreListMemberEntitlements(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListMemberEntitlements(
		c.Request.Context(), ConsoleScope{StoreID: &storeID}, memberID, page,
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// StoreGrantMemberEntitlement handles POST /store/members/:memberID/coupon-entitlements.
// The store scope is always taken from the authenticated token; body-supplied
// scope fields are ignored so a store can only grant its own templates.
func (h *ConsoleHandler) StoreGrantMemberEntitlement(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	memberID, err := positivePathID(c, "memberID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("优惠券补发参数不正确"))
		return
	}
	entry := audit.FromContext(c, "member.coupon.grant", "member", memberID)
	view, err := h.svc.GrantMemberEntitlement(
		c.Request.Context(), ConsoleScope{StoreID: &storeID}, memberID, req, idempotency.Key(c), entry,
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// StoreVoid handles POST /store/coupon-voids.
func (h *ConsoleHandler) StoreVoid(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.void(c, ConsoleScope{StoreID: &storeID})
}

// StoreVerify handles POST /store/coupon-verifications.
func (h *ConsoleHandler) StoreVerify(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.verify(c, ConsoleScope{StoreID: &storeID})
}

// --- shared implementations ---

func (h *ConsoleHandler) list(c *gin.Context, scope ConsoleScope) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListTemplates(c.Request.Context(), scope, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) get(c *gin.Context, scope ConsoleScope) {
	id, err := templateID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetTemplate(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) create(c *gin.Context, scope ConsoleScope) {
	var in TemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateTemplate(c.Request.Context(), scope, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func (h *ConsoleHandler) update(c *gin.Context, scope ConsoleScope) {
	id, err := templateID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in TemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateTemplate(c.Request.Context(), scope, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) delete(c *gin.Context, scope ConsoleScope) {
	id, err := templateID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteTemplate(c.Request.Context(), scope, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

func (h *ConsoleHandler) setStatus(c *gin.Context, scope ConsoleScope, publish bool) {
	id, err := templateID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var view ConsoleTemplateView
	if publish {
		view, err = h.svc.PublishTemplate(c.Request.Context(), scope, id)
	} else {
		view, err = h.svc.DisableTemplate(c.Request.Context(), scope, id)
	}
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) applicableItems(c *gin.Context, scope ConsoleScope) {
	id, err := templateID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetApplicableItems(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) void(c *gin.Context, scope ConsoleScope) {
	var req VoidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.Void(c.Request.Context(), scope, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) verify(c *gin.Context, scope ConsoleScope) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.Verify(c.Request.Context(), scope, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}
