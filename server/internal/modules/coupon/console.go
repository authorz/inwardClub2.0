package coupon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
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
	ID           int64
	ScopeType    string
	StoreID      *int64
	Name         string
	Description  string
	CouponType   string
	ValueCent    int64
	PointsPrice  int64
	StockQty     int64
	IssuedQty    int64
	PerMemberLim int64
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ApplicableScope is the best-effort decoded shape of coupon_templates.applicable_scope.
type ApplicableScope struct {
	ItemIDs     []int64 `json:"itemIds,omitempty"`
	CategoryIDs []int64 `json:"categoryIds,omitempty"`
}

// TemplateInput is the create/update body for a coupon template.
type TemplateInput struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	CouponType   string `json:"couponType" binding:"required"`
	ValueCent    int64  `json:"valueCent"`
	PointsPrice  int64  `json:"pointsPrice"`
	StockQty     int64  `json:"stockQuantity"`
	PerMemberLim int64  `json:"perMemberLimit"`
}

// GrantRequest grants an entitlement to a member from a template.
type GrantRequest struct {
	TemplateID int64  `json:"templateId" binding:"required"`
	MemberID   int64  `json:"memberId" binding:"required"`
	Reason     string `json:"reason"`
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
	ID           int64     `json:"id"`
	ScopeType    string    `json:"scopeType"`
	StoreID      *int64    `json:"storeId,omitempty"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CouponType   string    `json:"couponType"`
	ValueCent    int64     `json:"valueCent"`
	PointsPrice  int64     `json:"pointsPrice"`
	StockQty     int64     `json:"stockQuantity"`
	IssuedQty    int64     `json:"issuedQuantity"`
	PerMemberLim int64     `json:"perMemberLimit"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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
	Grant(ctx context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error)
	Void(ctx context.Context, scope ConsoleScope, req VoidRequest) (EntitlementView, error)
	Verify(ctx context.Context, scope ConsoleScope, req VerifyRequest) (EntitlementView, error)
}

type sqlConsoleRepository struct{ db *platdb.DB }

// NewConsoleRepository builds the MySQL console coupon-template repository.
func NewConsoleRepository(db *platdb.DB) ConsoleRepository { return &sqlConsoleRepository{db: db} }

const templateSelect = `SELECT id, scope_type, store_id, name, COALESCE(description,''),
	coupon_type, value_cent, points_price, stock_quantity, issued_quantity,
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
	return t == TypeExchange || t == TypeDiscount || t == TypeCash
}

func (r *sqlConsoleRepository) CreateTemplate(ctx context.Context, scope ConsoleScope, in TemplateInput) (Template, error) {
	if !validCouponType(in.CouponType) {
		return Template{}, apperr.Invalid("优惠券类型不正确")
	}
	if in.Name == "" {
		return Template{}, apperr.Invalid("请填写优惠券名称")
	}
	if in.ValueCent < 0 || in.PointsPrice < 0 || in.StockQty < 0 || in.PerMemberLim < 0 {
		return Template{}, apperr.Invalid("优惠券金额、积分、库存和限领数量不能小于 0")
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
	// validity_rule / applicable_scope are NOT NULL JSON columns; seed them with
	// empty objects until richer rule editing lands.
	const q = `INSERT INTO coupon_templates
		(scope_type, store_id, name, description, coupon_type, value_cent, points_price,
		 stock_quantity, issued_quantity, validity_rule, applicable_scope, per_member_limit,
		 status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, '{}', '{}', ?, 'draft', ?, ?)`
	res, err := r.db.ExecContext(ctx, q, scopeType, storeID, in.Name, in.Description, in.CouponType,
		in.ValueCent, in.PointsPrice, in.StockQty, in.PerMemberLim, now, now)
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
	if !validCouponType(in.CouponType) {
		return Template{}, apperr.Invalid("优惠券类型不正确")
	}
	if in.Name == "" {
		return Template{}, apperr.Invalid("请填写优惠券名称")
	}
	if in.ValueCent < 0 || in.PointsPrice < 0 || in.StockQty < 0 || in.PerMemberLim < 0 {
		return Template{}, apperr.Invalid("优惠券金额、积分、库存和限领数量不能小于 0")
	}
	current, err := r.GetTemplate(ctx, scope, id)
	if err != nil {
		return Template{}, err
	}
	if in.StockQty > 0 && in.StockQty < current.IssuedQty {
		return Template{}, apperr.Invalid("库存不能小于已发放数量")
	}
	now := time.Now().UTC()
	q := `UPDATE coupon_templates SET name=?, description=?, coupon_type=?, value_cent=?,
		points_price=?, stock_quantity=?, per_member_limit=?, updated_at=? WHERE id=?`
	args := []any{in.Name, in.Description, in.CouponType, in.ValueCent, in.PointsPrice,
		in.StockQty, in.PerMemberLim, now, id}
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

// Grant issues one entitlement to a member from a template, enforcing the
// template's stock and per-member limit under a row lock. The entitlement
// inherits the template's store scope; issued_quantity is bumped atomically.
func (r *sqlConsoleRepository) Grant(ctx context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error) {
	// The template must be visible within the caller's scope.
	tmpl, err := r.GetTemplate(ctx, scope, req.TemplateID)
	if err != nil {
		return EntitlementView{}, err
	}
	now := time.Now().UTC()
	entNo := fmt.Sprintf("E%d-%d", req.TemplateID, now.UnixNano())
	grantedBy := "admin"
	if scope.StoreID != nil {
		grantedBy = "store"
	}
	var view EntitlementView
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var stock, issued, perLimit int64
		if err := tx.QueryRowContext(ctx,
			`SELECT stock_quantity, issued_quantity, per_member_limit FROM coupon_templates WHERE id = ? FOR UPDATE`,
			req.TemplateID).Scan(&stock, &issued, &perLimit); err != nil {
			return apperr.Internal(err)
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
			(entitlement_no, coupon_template_id, member_id, store_id, status, granted_reason,
			 granted_by_type, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'active', ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, ins, entNo, req.TemplateID, req.MemberID, tmpl.StoreID,
			req.Reason, grantedBy, now, now)
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
		return nil
	})
	if err != nil {
		return EntitlementView{}, err
	}
	return view, nil
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
		if expiresAt.Valid && expiresAt.Time.Before(now) {
			return apperr.Conflict("coupon entitlement has expired")
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
		&t.CouponType, &t.ValueCent, &t.PointsPrice, &t.StockQty, &t.IssuedQty,
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
		ID:           t.ID,
		ScopeType:    t.ScopeType,
		StoreID:      t.StoreID,
		Name:         t.Name,
		Description:  t.Description,
		CouponType:   t.CouponType,
		ValueCent:    t.ValueCent,
		PointsPrice:  t.PointsPrice,
		StockQty:     t.StockQty,
		IssuedQty:    t.IssuedQty,
		PerMemberLim: t.PerMemberLim,
		Status:       t.Status,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
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
	if err != nil {
		return 0, apperr.Invalid("invalid id")
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

// Grant handles POST /admin/coupon-grants.
func (h *ConsoleHandler) Grant(c *gin.Context) {
	h.grant(c, ConsoleScope{})
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

// StoreGrant handles POST /store/coupon-grants.
func (h *ConsoleHandler) StoreGrant(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.grant(c, ConsoleScope{StoreID: &storeID})
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

func (h *ConsoleHandler) grant(c *gin.Context, scope ConsoleScope) {
	var req GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.Grant(c.Request.Context(), scope, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
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
