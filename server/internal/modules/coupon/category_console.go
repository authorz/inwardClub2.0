package coupon

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// CategoryInput is the HQ-console create/update payload. businessType is a
// fixed fulfillment capability; name, order and status are managed content.
type CategoryInput struct {
	Name                string `json:"name" binding:"required"`
	BusinessType        string `json:"businessType" binding:"required"`
	Description         string `json:"description"`
	AdmissionCount      int    `json:"admissionCount"`
	DefaultValidityDays int    `json:"defaultValidityDays"`
	SortOrder           int    `json:"sortOrder"`
	Status              string `json:"status"`
}

type ConsoleCategoryView struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	BusinessType        string    `json:"businessType"`
	Description         string    `json:"description,omitempty"`
	AdmissionCount      int       `json:"admissionCount"`
	DefaultValidityDays int       `json:"defaultValidityDays"`
	CanonicalTemplateID int64     `json:"canonicalTemplateId"`
	SortOrder           int       `json:"sortOrder"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type CategoryRepository interface {
	ListCategories(ctx context.Context, page httpx.Page, status, keyword string, activeOnly bool) ([]CouponCategory, int64, error)
	GetCategory(ctx context.Context, id int64) (CouponCategory, error)
	CreateCategory(ctx context.Context, in CategoryInput) (CouponCategory, error)
	UpdateCategory(ctx context.Context, id int64, in CategoryInput) (CouponCategory, error)
	CountCategoryEntitlements(ctx context.Context, id int64) (int64, error)
	DeleteCategory(ctx context.Context, id int64) error
}

func categoryWhere(status, keyword string, activeOnly bool) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 2)
	if activeOnly {
		clauses = append(clauses, "status = 'active'")
	} else if status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		clauses = append(clauses, "name LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	return strings.Join(clauses, " AND "), args
}

func (r *sqlConsoleRepository) ListCategories(ctx context.Context, page httpx.Page, status, keyword string, activeOnly bool) ([]CouponCategory, int64, error) {
	where, args := categoryWhere(status, keyword, activeOnly)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_categories WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	queryArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, business_type, description, admission_count,
		default_validity_days, canonical_template_id, sort_order, status, created_at, updated_at
		FROM coupon_categories WHERE `+where+` ORDER BY sort_order ASC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]CouponCategory, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, category)
	}
	return out, total, rows.Err()
}

func (r *sqlConsoleRepository) GetCategory(ctx context.Context, id int64) (CouponCategory, error) {
	category, err := scanCategory(r.db.QueryRowContext(ctx, `SELECT id, name, business_type, description, admission_count,
		default_validity_days, canonical_template_id, sort_order, status, created_at, updated_at
		FROM coupon_categories WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return CouponCategory{}, apperr.NotFound("coupon category not found")
	}
	if err != nil {
		return CouponCategory{}, apperr.Internal(err)
	}
	return category, nil
}

func normalizeCategoryInput(in CategoryInput) (CategoryInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" {
		return in, apperr.Invalid("请填写券种名称")
	}
	if !validCouponType(in.BusinessType) {
		return in, apperr.Invalid("请选择正确的兑换场景")
	}
	if len([]rune(in.Description)) > 500 {
		return in, apperr.Invalid("券种说明不能超过 500 个字")
	}
	if in.DefaultValidityDays == 0 {
		in.DefaultValidityDays = 30
	}
	if in.DefaultValidityDays < 1 || in.DefaultValidityDays > 3650 {
		return in, apperr.Invalid("默认有效期必须在 1 到 3650 天之间")
	}
	if in.BusinessType == TypeAdmissionTicket {
		if in.AdmissionCount < 1 || in.AdmissionCount > 99 {
			return in, apperr.Invalid("门票可兑人数必须在 1 到 99 之间")
		}
	} else {
		in.AdmissionCount = 1
	}
	if in.SortOrder < 0 {
		return in, apperr.Invalid("排序值不能小于 0")
	}
	if in.Status == "" {
		in.Status = CategoryStatusActive
	}
	if in.Status != CategoryStatusActive && in.Status != CategoryStatusDisabled {
		return in, apperr.Invalid("券种状态不正确")
	}
	return in, nil
}

func (r *sqlConsoleRepository) CreateCategory(ctx context.Context, in CategoryInput) (CouponCategory, error) {
	in, err := normalizeCategoryInput(in)
	if err != nil {
		return CouponCategory{}, err
	}
	now := time.Now().UTC()
	var id int64
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO coupon_categories
			(name, business_type, description, admission_count, default_validity_days,
			 canonical_template_id, sort_order, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?)`,
			in.Name, in.BusinessType, in.Description, in.AdmissionCount,
			in.DefaultValidityDays, in.SortOrder, in.Status, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		templateStatus := "disabled"
		if in.Status == CategoryStatusActive {
			templateStatus = "published"
		}
		res, err = tx.ExecContext(ctx, `INSERT INTO coupon_templates
			(scope_type, store_id, name, description, coupon_type, category_id,
			 admission_count, value_cent, points_price, stock_quantity, issued_quantity,
			 validity_rule, applicable_scope, per_member_limit, status, created_at, updated_at)
			VALUES ('global', NULL, ?, ?, ?, ?, ?, 0, 0, 0, 0,
			 JSON_OBJECT('days', ?), JSON_OBJECT(), 0, ?, ?, ?)`,
			in.Name, in.Description, in.BusinessType, id, in.AdmissionCount,
			in.DefaultValidityDays, templateStatus, now, now)
		if err != nil {
			return err
		}
		templateID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE coupon_categories SET canonical_template_id = ? WHERE id = ?`, templateID, id)
		return err
	})
	if err != nil {
		if platdb.IsDuplicate(err) {
			return CouponCategory{}, apperr.Conflict("券种名称已存在")
		}
		return CouponCategory{}, apperr.Internal(err)
	}
	return r.GetCategory(ctx, id)
}

func (r *sqlConsoleRepository) UpdateCategory(ctx context.Context, id int64, in CategoryInput) (CouponCategory, error) {
	in, err := normalizeCategoryInput(in)
	if err != nil {
		return CouponCategory{}, err
	}
	existing, err := r.GetCategory(ctx, id)
	if err != nil {
		return CouponCategory{}, err
	}
	if existing.BusinessType != in.BusinessType {
		var used int64
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_templates WHERE category_id = ?`, id).Scan(&used); err != nil {
			return CouponCategory{}, apperr.Internal(err)
		}
		if used > 0 {
			return CouponCategory{}, apperr.Conflict("该券种已被使用，不能修改使用方式")
		}
	}
	now := time.Now().UTC()
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE coupon_categories
			SET name = ?, business_type = ?, description = ?, admission_count = ?,
			    default_validity_days = ?, sort_order = ?, status = ?, updated_at = ? WHERE id = ?`,
			in.Name, in.BusinessType, in.Description, in.AdmissionCount,
			in.DefaultValidityDays, in.SortOrder, in.Status, now, id)
		if err != nil {
			return err
		}
		templateStatus := "disabled"
		if in.Status == CategoryStatusActive {
			templateStatus = "published"
		}
		_, err = tx.ExecContext(ctx, `UPDATE coupon_templates
			SET name = ?, description = ?, coupon_type = ?, admission_count = ?,
			    validity_rule = JSON_OBJECT('days', ?), status = ?, updated_at = ?
			WHERE id = ?`, in.Name, in.Description, in.BusinessType, in.AdmissionCount,
			in.DefaultValidityDays, templateStatus, now, existing.CanonicalTemplateID)
		return err
	})
	if err != nil {
		if platdb.IsDuplicate(err) {
			return CouponCategory{}, apperr.Conflict("券种名称已存在")
		}
		return CouponCategory{}, apperr.Internal(err)
	}
	return r.GetCategory(ctx, id)
}

func (r *sqlConsoleRepository) DeleteCategory(ctx context.Context, id int64) error {
	if _, err := r.GetCategory(ctx, id); err != nil {
		return err
	}
	var used int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_templates WHERE category_id = ?`, id).Scan(&used); err != nil {
		return apperr.Internal(err)
	}
	if used > 0 {
		return apperr.Conflict("该券种已被使用，只能停用")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM coupon_categories WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlConsoleRepository) CountCategoryEntitlements(ctx context.Context, id int64) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM coupon_entitlements AS entitlement
		JOIN coupon_templates AS template ON template.id = entitlement.coupon_template_id
		WHERE template.category_id = ?`, id).Scan(&count); err != nil {
		return 0, apperr.Internal(err)
	}
	return count, nil
}

func scanCategory(scanner scanner) (CouponCategory, error) {
	var category CouponCategory
	err := scanner.Scan(&category.ID, &category.Name, &category.BusinessType, &category.Description,
		&category.AdmissionCount, &category.DefaultValidityDays, &category.CanonicalTemplateID, &category.SortOrder,
		&category.Status, &category.CreatedAt, &category.UpdatedAt)
	return category, err
}

func categoryView(category CouponCategory) ConsoleCategoryView {
	return ConsoleCategoryView{
		ID: category.ID, Name: category.Name, BusinessType: category.BusinessType,
		Description: category.Description, AdmissionCount: category.AdmissionCount,
		DefaultValidityDays: category.DefaultValidityDays, CanonicalTemplateID: category.CanonicalTemplateID,
		SortOrder: category.SortOrder, Status: category.Status,
		CreatedAt: category.CreatedAt, UpdatedAt: category.UpdatedAt,
	}
}

func (s *ConsoleService) categoryRepository() (CategoryRepository, error) {
	repo, ok := s.repo.(CategoryRepository)
	if !ok {
		return nil, apperr.Internal(errors.New("coupon category repository is not configured"))
	}
	return repo, nil
}

func (s *ConsoleService) ListCategories(ctx context.Context, page httpx.Page, status, keyword string, activeOnly bool) ([]ConsoleCategoryView, int64, error) {
	repo, err := s.categoryRepository()
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := repo.ListCategories(ctx, page, status, keyword, activeOnly)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ConsoleCategoryView, 0, len(rows))
	for _, row := range rows {
		views = append(views, categoryView(row))
	}
	return views, total, nil
}

func (s *ConsoleService) GetCategory(ctx context.Context, id int64) (ConsoleCategoryView, error) {
	repo, err := s.categoryRepository()
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	row, err := repo.GetCategory(ctx, id)
	return categoryView(row), err
}

func (s *ConsoleService) CreateCategory(ctx context.Context, in CategoryInput) (ConsoleCategoryView, error) {
	repo, err := s.categoryRepository()
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	row, err := repo.CreateCategory(ctx, in)
	return categoryView(row), err
}

func (s *ConsoleService) UpdateCategory(ctx context.Context, id int64, in CategoryInput) (ConsoleCategoryView, error) {
	repo, err := s.categoryRepository()
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	row, err := repo.UpdateCategory(ctx, id, in)
	return categoryView(row), err
}

func (s *ConsoleService) DeleteCategory(ctx context.Context, id int64) error {
	repo, err := s.categoryRepository()
	if err != nil {
		return err
	}
	entitlementCount, err := repo.CountCategoryEntitlements(ctx, id)
	if err != nil {
		return err
	}
	if entitlementCount > 0 {
		return apperr.Conflict("已有用户持有该券种，不能删除，可改为停用")
	}
	return repo.DeleteCategory(ctx, id)
}

func categoryID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("categoryID"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid categoryID")
	}
	return id, nil
}

func (h *ConsoleHandler) Categories(c *gin.Context) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListCategories(c.Request.Context(), page, c.Query("status"), c.Query("keyword"), false)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) StoreCategories(c *gin.Context) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListCategories(c.Request.Context(), page, "", c.Query("keyword"), true)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) GetCategory(c *gin.Context) {
	id, err := categoryID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetCategory(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) CreateCategory(c *gin.Context) {
	var in CategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid("券种参数不正确"))
		return
	}
	view, err := h.svc.CreateCategory(c.Request.Context(), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func (h *ConsoleHandler) UpdateCategory(c *gin.Context) {
	id, err := categoryID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in CategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid("券种参数不正确"))
		return
	}
	view, err := h.svc.UpdateCategory(c.Request.Context(), id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) DeleteCategory(c *gin.Context) {
	id, err := categoryID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}
