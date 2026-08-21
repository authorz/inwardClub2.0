// Package catalog owns store-bound categories, items and variants for both
// console management and mini-program reads.
package catalog

import (
	"context"
	"encoding/json"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Item types.
const (
	ItemTypeFood       = "food"
	ItemTypeCoupon     = "coupon"
	ItemTypeRedeemable = "redeemable"
	ItemTypePhysical   = "physical"
)

// Category types separate ordinary products from coupon products. Coupon
// categories may only contain items that grant one configured coupon template.
const (
	CategoryTypeProduct = "product"
	CategoryTypeCoupon  = "coupon"
)

// Category is a catalog category.
type Category struct {
	ID           int64
	ScopeType    string
	StoreID      *int64
	StoreName    string
	ParentID     *int64
	Name         string
	CategoryType string
	AssetID      *int64
	SortOrder    int
	Status       string
}

// Item is a catalog item (menu/product/redeemable).
type Item struct {
	ID                    int64
	ScopeType             string
	StoreID               *int64
	StoreName             string
	CategoryID            *int64
	CategoryName          string
	Name                  string
	Description           string
	AssetID               *int64
	ItemType              string
	PriceCent             int64
	StockQuantity         int64
	PayChannels           []string
	CouponTemplateIDs     []int64
	GrantCouponTemplateID *int64
	PointsReward          int64
	SortOrder             int
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CategoryView / ItemView are the public representations.
type CategoryView struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	CategoryType string `json:"categoryType"`
	ImageURL     string `json:"imageUrl,omitempty"`
	ParentID     *int64 `json:"parentId,omitempty"`
	SortOrder    int    `json:"sortOrder"`
}

type ItemView struct {
	ID                int64    `json:"id"`
	CategoryID        *int64   `json:"categoryId,omitempty"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	ImageURL          string   `json:"imageUrl,omitempty"`
	ItemType          string   `json:"itemType"`
	PriceCent         int64    `json:"priceCent"`
	StockQuantity     int64    `json:"stockQuantity"`
	PayChannels       []string `json:"payChannels"`
	CouponTemplateIDs []int64  `json:"couponTemplateIds"`
	PointsReward      int64    `json:"pointsReward"`
	SortOrder         int      `json:"sortOrder"`
	Status            string   `json:"status"`
}

// AssetResolver resolves an asset id to a public URL.
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// Repository is the catalog read persistence port.
type Repository interface {
	ListCategoriesForStore(ctx context.Context, storeID int64) ([]Category, error)
	ListItemsForStore(ctx context.Context, storeID int64, categoryID *int64, limit, offset int) ([]Item, int64, error)
	ListCouponRedeemableItemsForStore(ctx context.Context, storeID, couponTemplateID, maxPriceCent int64) ([]Item, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL catalog repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ListCategoriesForStore(ctx context.Context, storeID int64) ([]Category, error) {
	const q = `SELECT id, scope_type, store_id, parent_id, name, category_type, asset_id, sort_order, status
		FROM catalog_categories
		WHERE status = 'active' AND scope_type = 'store' AND store_id = ?
		ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ScopeType, &c.StoreID, &c.ParentID, &c.Name, &c.CategoryType, &c.AssetID, &c.SortOrder, &c.Status); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *sqlRepository) ListItemsForStore(ctx context.Context, storeID int64, categoryID *int64, limit, offset int) ([]Item, int64, error) {
	where := `status = 'published' AND scope_type = 'store' AND store_id = ?
		AND EXISTS (SELECT 1 FROM catalog_categories c
			WHERE c.id = catalog_items.category_id AND c.store_id = catalog_items.store_id)`
	args := []any{storeID}
	if categoryID != nil {
		where += ` AND category_id = ?`
		args = append(args, *categoryID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, scope_type, store_id, category_id, name, COALESCE(description,''),
		asset_id, item_type, price_cent, stock_quantity, pay_channels, coupon_template_ids, grant_coupon_template_id, points_reward,
		sort_order, status, created_at, updated_at
		FROM catalog_items WHERE ` + where + ` ORDER BY sort_order ASC, id ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		var payChannels, couponTemplateIDs []byte
		if err := rows.Scan(&it.ID, &it.ScopeType, &it.StoreID, &it.CategoryID, &it.Name, &it.Description,
			&it.AssetID, &it.ItemType, &it.PriceCent, &it.StockQuantity, &payChannels, &couponTemplateIDs, &it.GrantCouponTemplateID, &it.PointsReward,
			&it.SortOrder, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		it.PayChannels = decodeChannels(payChannels)
		it.CouponTemplateIDs = decodeInt64List(couponTemplateIDs)
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListCouponRedeemableItemsForStore(ctx context.Context, storeID, couponTemplateID, maxPriceCent int64) ([]Item, error) {
	const q = `SELECT id, scope_type, store_id, category_id, name, COALESCE(description,''),
		asset_id, item_type, price_cent, stock_quantity, pay_channels, coupon_template_ids, grant_coupon_template_id, points_reward,
		sort_order, status, created_at, updated_at
		FROM catalog_items
		WHERE status = 'published' AND scope_type = 'store' AND store_id = ?
		  AND EXISTS (SELECT 1 FROM catalog_categories c
			WHERE c.id = catalog_items.category_id AND c.store_id = catalog_items.store_id)
		  AND price_cent > 0 AND price_cent <= ?
		  AND item_type <> 'coupon'
		  AND JSON_CONTAINS(COALESCE(coupon_template_ids, JSON_ARRAY()), CAST(? AS JSON), '$')
		ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID, maxPriceCent, couponTemplateID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		var payChannels, couponTemplateIDs []byte
		if err := rows.Scan(&it.ID, &it.ScopeType, &it.StoreID, &it.CategoryID, &it.Name, &it.Description,
			&it.AssetID, &it.ItemType, &it.PriceCent, &it.StockQuantity, &payChannels, &couponTemplateIDs, &it.GrantCouponTemplateID,
			&it.PointsReward, &it.SortOrder, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		it.PayChannels = decodeChannels(payChannels)
		it.CouponTemplateIDs = decodeInt64List(couponTemplateIDs)
		out = append(out, it)
	}
	return out, rows.Err()
}

func decodeChannels(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var ch []string
	if err := json.Unmarshal(raw, &ch); err != nil {
		return []string{}
	}
	normalized, _ := normalizePayChannels(ch)
	return normalized
}

func decodeInt64List(raw []byte) []int64 {
	if len(raw) == 0 {
		return []int64{}
	}
	var values []int64
	if err := json.Unmarshal(raw, &values); err != nil {
		return []int64{}
	}
	return values
}

func normalizePayChannels(channels []string) ([]string, error) {
	normalized := make([]string, 0, len(channels))
	seen := make(map[string]struct{}, len(channels))
	var invalid string
	for _, channel := range channels {
		if channel == "balance" {
			channel = "coin"
		}
		if channel != "wechat" && channel != "coin" {
			if invalid == "" {
				invalid = channel
			}
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, channel)
	}
	if invalid != "" {
		return normalized, apperr.Invalid("catalog: unsupported pay channel " + invalid)
	}
	return normalized, nil
}

// Service provides catalog read operations.
type Service struct {
	repo   Repository
	assets AssetResolver
}

// NewService builds the catalog service.
func NewService(repo Repository, assets AssetResolver) *Service {
	return &Service{repo: repo, assets: assets}
}

// ListCategories returns the categories visible for a store.
func (s *Service) ListCategories(ctx context.Context, storeID int64) ([]CategoryView, error) {
	cats, err := s.repo.ListCategoriesForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}
	views := make([]CategoryView, 0, len(cats))
	for _, c := range cats {
		v := CategoryView{ID: c.ID, Name: c.Name, CategoryType: c.CategoryType, ParentID: c.ParentID, SortOrder: c.SortOrder}
		if c.AssetID != nil {
			v.ImageURL, _ = s.assets.PublicURLByID(ctx, *c.AssetID)
		}
		views = append(views, v)
	}
	return views, nil
}

// ListItems returns the items visible for a store.
func (s *Service) ListItems(ctx context.Context, storeID int64, categoryID *int64, page httpx.Page) ([]ItemView, int64, error) {
	items, total, err := s.repo.ListItemsForStore(ctx, storeID, categoryID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]ItemView, 0, len(items))
	for _, it := range items {
		itemType := it.ItemType
		if itemType == ItemTypeCoupon && it.GrantCouponTemplateID == nil {
			itemType = ItemTypeFood
		}
		v := ItemView{
			ID: it.ID, CategoryID: it.CategoryID, Name: it.Name, Description: it.Description,
			ItemType: itemType, PriceCent: it.PriceCent, StockQuantity: it.StockQuantity,
			PayChannels: it.PayChannels, CouponTemplateIDs: it.CouponTemplateIDs,
			PointsReward: it.PointsReward, SortOrder: it.SortOrder, Status: it.Status,
		}
		if it.AssetID != nil {
			v.ImageURL, _ = s.assets.PublicURLByID(ctx, *it.AssetID)
		}
		views = append(views, v)
	}
	return views, total, nil
}

// ListCouponRedeemableItems returns published products configured for the
// concrete coupon template whose unit price can fit within its face value.
func (s *Service) ListCouponRedeemableItems(ctx context.Context, storeID, couponTemplateID, maxPriceCent int64) ([]ItemView, error) {
	items, err := s.repo.ListCouponRedeemableItemsForStore(ctx, storeID, couponTemplateID, maxPriceCent)
	if err != nil {
		return nil, err
	}
	views := make([]ItemView, 0, len(items))
	for _, it := range items {
		view := ItemView{
			ID: it.ID, CategoryID: it.CategoryID, Name: it.Name, Description: it.Description,
			ItemType: it.ItemType, PriceCent: it.PriceCent, StockQuantity: it.StockQuantity,
			PayChannels: it.PayChannels, CouponTemplateIDs: it.CouponTemplateIDs,
			PointsReward: it.PointsReward, SortOrder: it.SortOrder, Status: it.Status,
		}
		if it.AssetID != nil {
			view.ImageURL, _ = s.assets.PublicURLByID(ctx, *it.AssetID)
		}
		views = append(views, view)
	}
	return views, nil
}
