// Console CRUD contracts for categories, items and variants. Reads and writes
// are implemented against catalog_categories/items/variants, with scope_type /
// store_id enforced from the selected store in headquarters or from the
// authenticated store scope in the store console.
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"

	"github.com/gin-gonic/gin"
)

// Variant is a catalog item variant (SKU).
type Variant struct {
	ID            int64
	ItemID        int64
	SKUCode       string
	Name          string
	PriceCent     int64
	StockQuantity int64
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ConsoleScope pins console reads/writes to a store, or nil for admin/HQ which
// sees every scope.
type ConsoleScope struct {
	StoreID *int64
}

// CategoryInput is the create/update body for a console category.
type CategoryInput struct {
	StoreID   *int64 `json:"storeId"`
	Name      string `json:"name" binding:"required"`
	ParentID  *int64 `json:"parentId"`
	AssetID   *int64 `json:"assetId"`
	SortOrder int    `json:"sortOrder"`
	Status    string `json:"status" binding:"required"`
}

// ItemInput is the create/update body for a console item.
type ItemInput struct {
	StoreID       *int64   `json:"storeId"`
	CategoryID    *int64   `json:"categoryId"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	AssetID       *int64   `json:"assetId"`
	ItemType      string   `json:"itemType" binding:"required"`
	PriceCent     int64    `json:"priceCent"`
	StockQuantity int64    `json:"stockQuantity"`
	PayChannels   []string `json:"payChannels"`
	PointsReward  int64    `json:"pointsReward"`
	SortOrder     int      `json:"sortOrder"`
	Status        string   `json:"status" binding:"required"`
}

// VariantInput is the create/update body for a console variant.
type VariantInput struct {
	SKUCode       string `json:"skuCode" binding:"required"`
	Name          string `json:"name" binding:"required"`
	PriceCent     int64  `json:"priceCent"`
	StockQuantity int64  `json:"stockQuantity"`
	Status        string `json:"status" binding:"required"`
}

// ConsoleCategoryView is the console representation of a category.
type ConsoleCategoryView struct {
	ID        int64  `json:"id"`
	ScopeType string `json:"scopeType"`
	StoreID   *int64 `json:"storeId,omitempty"`
	StoreName string `json:"storeName,omitempty"`
	ParentID  *int64 `json:"parentId,omitempty"`
	Name      string `json:"name"`
	AssetID   *int64 `json:"assetId,omitempty"`
	SortOrder int    `json:"sortOrder"`
	Status    string `json:"status"`
}

// ConsoleItemView is the console representation of an item.
type ConsoleItemView struct {
	ID            int64     `json:"id"`
	ScopeType     string    `json:"scopeType"`
	StoreID       *int64    `json:"storeId,omitempty"`
	StoreName     string    `json:"storeName,omitempty"`
	CategoryID    *int64    `json:"categoryId,omitempty"`
	CategoryName  string    `json:"categoryName,omitempty"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	AssetID       *int64    `json:"assetId,omitempty"`
	ImageURL      string    `json:"imageUrl,omitempty"`
	ItemType      string    `json:"itemType"`
	PriceCent     int64     `json:"priceCent"`
	StockQuantity int64     `json:"stockQuantity"`
	PayChannels   []string  `json:"payChannels"`
	PointsReward  int64     `json:"pointsReward"`
	SortOrder     int       `json:"sortOrder"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// VariantView is the console representation of a variant.
type VariantView struct {
	ID            int64     `json:"id"`
	ItemID        int64     `json:"itemId"`
	SKUCode       string    `json:"skuCode"`
	Name          string    `json:"name"`
	PriceCent     int64     `json:"priceCent"`
	StockQuantity int64     `json:"stockQuantity"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ConsoleListFilter contains the catalog list filters shared by the admin and
// store consoles. Store scope still comes from the authenticated store console;
// StoreID is only used by headquarters to select one store.
type ConsoleListFilter struct {
	StoreID    *int64
	CategoryID *int64
	Keyword    string
	Status     string
}

// ConsoleRepository is the catalog console CRUD persistence port.
type ConsoleRepository interface {
	ListCategories(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]Category, int64, error)
	GetCategory(ctx context.Context, scope ConsoleScope, id int64) (Category, error)
	CreateCategory(ctx context.Context, scope ConsoleScope, in CategoryInput) (Category, error)
	UpdateCategory(ctx context.Context, scope ConsoleScope, id int64, in CategoryInput) (Category, error)
	DeleteCategory(ctx context.Context, scope ConsoleScope, id int64) error

	ListItems(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]Item, int64, error)
	GetItem(ctx context.Context, scope ConsoleScope, id int64) (Item, error)
	CreateItem(ctx context.Context, scope ConsoleScope, in ItemInput) (Item, error)
	UpdateItem(ctx context.Context, scope ConsoleScope, id int64, in ItemInput) (Item, error)
	DeleteItem(ctx context.Context, scope ConsoleScope, id int64) error

	ListVariants(ctx context.Context, scope ConsoleScope, itemID int64, page httpx.Page) ([]Variant, int64, error)
	GetVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) (Variant, error)
	CreateVariant(ctx context.Context, scope ConsoleScope, itemID int64, in VariantInput) (Variant, error)
	UpdateVariant(ctx context.Context, scope ConsoleScope, itemID, id int64, in VariantInput) (Variant, error)
	DeleteVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) error
}

type sqlConsoleRepository struct{ db *platdb.DB }

// NewConsoleRepository builds the MySQL catalog console repository.
func NewConsoleRepository(db *platdb.DB) ConsoleRepository { return &sqlConsoleRepository{db: db} }

// scopeWhere returns the store filter clause and its args for a console scope.
// When scope.StoreID is nil (admin/HQ), no filter is applied.
func scopeWhere(scope ConsoleScope) (string, []any) {
	if scope.StoreID == nil {
		return "", nil
	}
	return " AND scope_type = 'store' AND store_id = ?", []any{*scope.StoreID}
}

func scopedAliasWhere(scope ConsoleScope, alias string) (string, []any) {
	if scope.StoreID == nil {
		return "", nil
	}
	return " AND " + alias + ".scope_type = 'store' AND " + alias + ".store_id = ?", []any{*scope.StoreID}
}

func scopeForWrite(scope ConsoleScope, requestedStoreID *int64) (string, *int64) {
	if scope.StoreID != nil {
		return "store", scope.StoreID
	}
	return "store", requestedStoreID
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func consoleListWhere(scope ConsoleScope, filter ConsoleListFilter, alias string) (string, []any) {
	where := "1=1"
	var args []any
	storeColumn := alias + ".store_id"
	if scope.StoreID != nil {
		where += " AND " + alias + ".scope_type = 'store' AND " + storeColumn + " = ?"
		args = append(args, *scope.StoreID)
	} else if filter.StoreID != nil {
		where += " AND " + storeColumn + " = ?"
		args = append(args, *filter.StoreID)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		where += " AND " + alias + `.name LIKE ? ESCAPE '\\'`
		args = append(args, "%"+escapeLike(keyword)+"%")
	}
	if filter.Status != "" {
		where += " AND " + alias + ".status = ?"
		args = append(args, filter.Status)
	}
	return where, args
}

// encodeChannels serialises pay channels to the JSON column, defaulting to an
// empty array so the NOT NULL constraint is satisfied.
func encodeChannels(ch []string) []byte {
	if ch == nil {
		ch = []string{}
	}
	raw, err := json.Marshal(ch)
	if err != nil {
		return []byte("[]")
	}
	return raw
}

func (r *sqlConsoleRepository) ListCategories(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]Category, int64, error) {
	where, args := consoleListWhere(scope, filter, "c")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_categories c WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT c.id, c.scope_type, c.store_id, COALESCE(s.name,''), c.parent_id,
		c.name, c.asset_id, c.sort_order, c.status
		FROM catalog_categories c
		LEFT JOIN stores s ON s.id = c.store_id
		WHERE ` + where + ` ORDER BY c.sort_order ASC, c.id ASC LIMIT ? OFFSET ?`
	qArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, q, qArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ScopeType, &c.StoreID, &c.StoreName, &c.ParentID,
			&c.Name, &c.AssetID, &c.SortOrder, &c.Status); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return out, total, nil
}

func (r *sqlConsoleRepository) GetCategory(ctx context.Context, scope ConsoleScope, id int64) (Category, error) {
	where, args := scopedAliasWhere(scope, "c")
	q := `SELECT c.id, c.scope_type, c.store_id, COALESCE(s.name,''), c.parent_id,
		c.name, c.asset_id, c.sort_order, c.status
		FROM catalog_categories c
		LEFT JOIN stores s ON s.id = c.store_id
		WHERE c.id = ?` + where
	qArgs := append([]any{id}, args...)
	var c Category
	if err := r.db.QueryRowContext(ctx, q, qArgs...).Scan(&c.ID, &c.ScopeType, &c.StoreID, &c.StoreName,
		&c.ParentID, &c.Name, &c.AssetID, &c.SortOrder, &c.Status); err != nil {
		if err == sql.ErrNoRows {
			return Category{}, apperr.NotFound("catalog category not found")
		}
		return Category{}, apperr.Internal(err)
	}
	return c, nil
}

func (r *sqlConsoleRepository) CreateCategory(ctx context.Context, scope ConsoleScope, in CategoryInput) (Category, error) {
	scopeType, storeID := scopeForWrite(scope, in.StoreID)
	res, err := r.db.ExecContext(ctx, `INSERT INTO catalog_categories
		(scope_type, store_id, parent_id, name, asset_id, sort_order, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		scopeType, storeID, in.ParentID, in.Name, in.AssetID, in.SortOrder, in.Status)
	if err != nil {
		return Category{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Category{}, apperr.Internal(err)
	}
	return r.GetCategory(ctx, scope, id)
}

func (r *sqlConsoleRepository) UpdateCategory(ctx context.Context, scope ConsoleScope, id int64, in CategoryInput) (Category, error) {
	if scope.StoreID == nil {
		if _, err := r.db.ExecContext(ctx, `UPDATE catalog_categories
			SET scope_type = 'store', store_id = ?, parent_id = ?, name = ?, asset_id = ?,
			    sort_order = ?, status = ?, updated_at = NOW()
			WHERE id = ?`,
			in.StoreID, in.ParentID, in.Name, in.AssetID, in.SortOrder, in.Status, id); err != nil {
			return Category{}, apperr.Internal(err)
		}
		return r.GetCategory(ctx, scope, id)
	}
	where, args := scopeWhere(scope)
	execArgs := append([]any{in.ParentID, in.Name, in.AssetID, in.SortOrder, in.Status, id}, args...)
	if _, err := r.db.ExecContext(ctx, `UPDATE catalog_categories
		SET parent_id = ?, name = ?, asset_id = ?, sort_order = ?, status = ?, updated_at = NOW()
		WHERE id = ?`+where, execArgs...); err != nil {
		return Category{}, apperr.Internal(err)
	}
	// GetCategory re-applies the scope filter, so an out-of-scope or missing
	// row surfaces as NotFound.
	return r.GetCategory(ctx, scope, id)
}

func (r *sqlConsoleRepository) DeleteCategory(ctx context.Context, scope ConsoleScope, id int64) error {
	where, args := scopeWhere(scope)
	res, err := r.db.ExecContext(ctx, `DELETE FROM catalog_categories WHERE id = ?`+where, append([]any{id}, args...)...)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("catalog category not found")
	}
	return nil
}

func (r *sqlConsoleRepository) ListItems(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]Item, int64, error) {
	where, args := consoleListWhere(scope, filter, "i")
	if filter.CategoryID != nil {
		where += " AND i.category_id = ?"
		args = append(args, *filter.CategoryID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items i WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT i.id, i.scope_type, i.store_id, COALESCE(s.name,''), i.category_id,
		COALESCE(c.name,''), i.name, COALESCE(i.description,''), i.asset_id, i.item_type,
		i.price_cent, i.stock_quantity, i.pay_channels, i.points_reward, i.sort_order,
		i.status, i.created_at, i.updated_at
		FROM catalog_items i
		LEFT JOIN stores s ON s.id = i.store_id
		LEFT JOIN catalog_categories c ON c.id = i.category_id
		WHERE ` + where + ` ORDER BY i.sort_order ASC, i.id ASC LIMIT ? OFFSET ?`
	qArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, q, qArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		var payChannels []byte
		if err := rows.Scan(&it.ID, &it.ScopeType, &it.StoreID, &it.StoreName, &it.CategoryID,
			&it.CategoryName, &it.Name, &it.Description, &it.AssetID, &it.ItemType,
			&it.PriceCent, &it.StockQuantity, &payChannels, &it.PointsReward, &it.SortOrder,
			&it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		it.PayChannels = decodeChannels(payChannels)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return out, total, nil
}

func (r *sqlConsoleRepository) GetItem(ctx context.Context, scope ConsoleScope, id int64) (Item, error) {
	where, args := scopedAliasWhere(scope, "i")
	q := `SELECT i.id, i.scope_type, i.store_id, COALESCE(s.name,''), i.category_id,
		COALESCE(c.name,''), i.name, COALESCE(i.description,''), i.asset_id, i.item_type,
		i.price_cent, i.stock_quantity, i.pay_channels, i.points_reward, i.sort_order,
		i.status, i.created_at, i.updated_at
		FROM catalog_items i
		LEFT JOIN stores s ON s.id = i.store_id
		LEFT JOIN catalog_categories c ON c.id = i.category_id
		WHERE i.id = ?` + where
	qArgs := append([]any{id}, args...)
	var it Item
	var payChannels []byte
	err := r.db.QueryRowContext(ctx, q, qArgs...).Scan(&it.ID, &it.ScopeType, &it.StoreID, &it.StoreName,
		&it.CategoryID, &it.CategoryName, &it.Name, &it.Description, &it.AssetID, &it.ItemType,
		&it.PriceCent, &it.StockQuantity, &payChannels, &it.PointsReward, &it.SortOrder,
		&it.Status, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Item{}, apperr.NotFound("catalog item not found")
		}
		return Item{}, apperr.Internal(err)
	}
	it.PayChannels = decodeChannels(payChannels)
	return it, nil
}

func (r *sqlConsoleRepository) CreateItem(ctx context.Context, scope ConsoleScope, in ItemInput) (Item, error) {
	scopeType, storeID := scopeForWrite(scope, in.StoreID)
	res, err := r.db.ExecContext(ctx, `INSERT INTO catalog_items
		(scope_type, store_id, category_id, name, description, asset_id, item_type,
		 price_cent, stock_quantity, pay_channels, points_reward, sort_order, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		scopeType, storeID, in.CategoryID, in.Name, in.Description, in.AssetID, in.ItemType,
		in.PriceCent, in.StockQuantity, encodeChannels(in.PayChannels), in.PointsReward, in.SortOrder, in.Status)
	if err != nil {
		return Item{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, apperr.Internal(err)
	}
	return r.GetItem(ctx, scope, id)
}

func (r *sqlConsoleRepository) UpdateItem(ctx context.Context, scope ConsoleScope, id int64, in ItemInput) (Item, error) {
	if scope.StoreID == nil {
		if _, err := r.db.ExecContext(ctx, `UPDATE catalog_items
			SET scope_type = 'store', store_id = ?, category_id = ?, name = ?,
			    description = ?, asset_id = ?, item_type = ?, price_cent = ?,
			    stock_quantity = ?, pay_channels = ?, points_reward = ?, sort_order = ?,
			    status = ?, updated_at = NOW()
			WHERE id = ?`,
			in.StoreID, in.CategoryID, in.Name, in.Description, in.AssetID, in.ItemType,
			in.PriceCent, in.StockQuantity, encodeChannels(in.PayChannels), in.PointsReward,
			in.SortOrder, in.Status, id); err != nil {
			return Item{}, apperr.Internal(err)
		}
		return r.GetItem(ctx, scope, id)
	}
	where, args := scopeWhere(scope)
	execArgs := append([]any{in.CategoryID, in.Name, in.Description, in.AssetID, in.ItemType,
		in.PriceCent, in.StockQuantity, encodeChannels(in.PayChannels), in.PointsReward, in.SortOrder, in.Status, id}, args...)
	if _, err := r.db.ExecContext(ctx, `UPDATE catalog_items
		SET category_id = ?, name = ?, description = ?, asset_id = ?, item_type = ?,
		    price_cent = ?, stock_quantity = ?, pay_channels = ?, points_reward = ?,
		    sort_order = ?, status = ?, updated_at = NOW()
		WHERE id = ?`+where, execArgs...); err != nil {
		return Item{}, apperr.Internal(err)
	}
	// GetItem re-applies the scope filter, so an out-of-scope or missing row
	// surfaces as NotFound.
	return r.GetItem(ctx, scope, id)
}

func (r *sqlConsoleRepository) DeleteItem(ctx context.Context, scope ConsoleScope, id int64) error {
	where, args := scopeWhere(scope)
	res, err := r.db.ExecContext(ctx, `DELETE FROM catalog_items WHERE id = ?`+where, append([]any{id}, args...)...)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("catalog item not found")
	}
	return nil
}

func (r *sqlConsoleRepository) ListVariants(ctx context.Context, scope ConsoleScope, itemID int64, page httpx.Page) ([]Variant, int64, error) {
	// Ownership of the item is enforced by scope: the item must be visible
	// under the same scope filter as catalog_items.
	itemWhere, itemArgs := scopeWhere(scope)
	var owned int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items WHERE id = ?`+itemWhere, append([]any{itemID}, itemArgs...)...).Scan(&owned); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if owned == 0 {
		return nil, 0, apperr.NotFound("catalog item not found")
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_variants WHERE item_id = ?`, itemID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, item_id, sku_code, name, price_cent, stock_quantity, status, created_at, updated_at
		FROM catalog_variants WHERE item_id = ? ORDER BY id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, itemID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Variant
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.ItemID, &v.SKUCode, &v.Name, &v.PriceCent, &v.StockQuantity, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return out, total, nil
}

func (r *sqlConsoleRepository) GetVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) (Variant, error) {
	itemWhere, itemArgs := scopeWhere(scope)
	var owned int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items WHERE id = ?`+itemWhere, append([]any{itemID}, itemArgs...)...).Scan(&owned); err != nil {
		return Variant{}, apperr.Internal(err)
	}
	if owned == 0 {
		return Variant{}, apperr.NotFound("catalog item not found")
	}
	q := `SELECT id, item_id, sku_code, name, price_cent, stock_quantity, status, created_at, updated_at
		FROM catalog_variants WHERE id = ? AND item_id = ?`
	var v Variant
	if err := r.db.QueryRowContext(ctx, q, id, itemID).Scan(&v.ID, &v.ItemID, &v.SKUCode, &v.Name, &v.PriceCent, &v.StockQuantity, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Variant{}, apperr.NotFound("catalog variant not found")
		}
		return Variant{}, apperr.Internal(err)
	}
	return v, nil
}

func (r *sqlConsoleRepository) CreateVariant(ctx context.Context, scope ConsoleScope, itemID int64, in VariantInput) (Variant, error) {
	if err := r.assertItemOwned(ctx, scope, itemID); err != nil {
		return Variant{}, err
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO catalog_variants
		(item_id, sku_code, name, price_cent, stock_quantity, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		itemID, in.SKUCode, in.Name, in.PriceCent, in.StockQuantity, in.Status)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return Variant{}, apperr.Conflict("variant sku already exists for this item")
		}
		return Variant{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Variant{}, apperr.Internal(err)
	}
	return r.GetVariant(ctx, scope, itemID, id)
}

func (r *sqlConsoleRepository) UpdateVariant(ctx context.Context, scope ConsoleScope, itemID, id int64, in VariantInput) (Variant, error) {
	if err := r.assertItemOwned(ctx, scope, itemID); err != nil {
		return Variant{}, err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE catalog_variants
		SET sku_code = ?, name = ?, price_cent = ?, stock_quantity = ?, status = ?, updated_at = NOW()
		WHERE id = ? AND item_id = ?`,
		in.SKUCode, in.Name, in.PriceCent, in.StockQuantity, in.Status, id, itemID); err != nil {
		if platdb.IsDuplicate(err) {
			return Variant{}, apperr.Conflict("variant sku already exists for this item")
		}
		return Variant{}, apperr.Internal(err)
	}
	return r.GetVariant(ctx, scope, itemID, id)
}

func (r *sqlConsoleRepository) DeleteVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) error {
	if err := r.assertItemOwned(ctx, scope, itemID); err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM catalog_variants WHERE id = ? AND item_id = ?`, id, itemID)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("catalog variant not found")
	}
	return nil
}

// assertItemOwned verifies that itemID is visible under scope, mirroring the
// ownership check used by the variant reads.
func (r *sqlConsoleRepository) assertItemOwned(ctx context.Context, scope ConsoleScope, itemID int64) error {
	itemWhere, itemArgs := scopeWhere(scope)
	var owned int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_items WHERE id = ?`+itemWhere, append([]any{itemID}, itemArgs...)...).Scan(&owned); err != nil {
		return apperr.Internal(err)
	}
	if owned == 0 {
		return apperr.NotFound("catalog item not found")
	}
	return nil
}

// ConsoleService provides catalog console CRUD operations.
type ConsoleService struct {
	repo   ConsoleRepository
	assets AssetResolver
}

// NewConsoleService builds the catalog console service.
func NewConsoleService(repo ConsoleRepository, assets ...AssetResolver) *ConsoleService {
	var resolver AssetResolver
	if len(assets) > 0 {
		resolver = assets[0]
	}
	return &ConsoleService{repo: repo, assets: resolver}
}

func categoryToView(c Category) ConsoleCategoryView {
	return ConsoleCategoryView{
		ID: c.ID, ScopeType: c.ScopeType, StoreID: c.StoreID, ParentID: c.ParentID,
		StoreName: c.StoreName, Name: c.Name, AssetID: c.AssetID,
		SortOrder: c.SortOrder, Status: c.Status,
	}
}

func (s *ConsoleService) itemToView(ctx context.Context, it Item) ConsoleItemView {
	view := ConsoleItemView{
		ID: it.ID, ScopeType: it.ScopeType, StoreID: it.StoreID, CategoryID: it.CategoryID,
		StoreName: it.StoreName, CategoryName: it.CategoryName, Name: it.Name,
		Description: it.Description, AssetID: it.AssetID, ItemType: it.ItemType,
		PriceCent: it.PriceCent, StockQuantity: it.StockQuantity, PayChannels: it.PayChannels,
		PointsReward: it.PointsReward, SortOrder: it.SortOrder, Status: it.Status,
		CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
	if s.assets != nil && it.AssetID != nil {
		view.ImageURL, _ = s.assets.PublicURLByID(ctx, *it.AssetID)
	}
	return view
}

func variantToView(v Variant) VariantView {
	return VariantView{
		ID: v.ID, ItemID: v.ItemID, SKUCode: v.SKUCode, Name: v.Name,
		PriceCent: v.PriceCent, StockQuantity: v.StockQuantity, Status: v.Status,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

func storeIDForWrite(scope ConsoleScope, requested *int64) (*int64, error) {
	if scope.StoreID != nil {
		return scope.StoreID, nil
	}
	if requested == nil || *requested <= 0 {
		return nil, apperr.Invalid("catalog: storeId is required")
	}
	return requested, nil
}

func (s *ConsoleService) validateItemInput(ctx context.Context, scope ConsoleScope, in *ItemInput) error {
	storeID, err := storeIDForWrite(scope, in.StoreID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.Name) == "" {
		return apperr.Invalid("catalog: item name is required")
	}
	if in.CategoryID == nil || *in.CategoryID <= 0 {
		return apperr.Invalid("catalog: categoryId is required")
	}
	if in.AssetID == nil || *in.AssetID <= 0 {
		return apperr.Invalid("catalog: assetId is required")
	}
	if in.PointsReward < 0 {
		return apperr.Invalid("catalog: pointsReward cannot be negative")
	}
	payChannels, err := normalizePayChannels(in.PayChannels)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetCategory(ctx, ConsoleScope{StoreID: storeID}, *in.CategoryID); err != nil {
		if apperr.From(err).Code == apperr.CodeNotFound {
			return apperr.Invalid("catalog: category must belong to the selected store")
		}
		return err
	}
	in.StoreID = storeID
	in.Name = strings.TrimSpace(in.Name)
	in.PayChannels = payChannels
	if in.ItemType == "" {
		in.ItemType = ItemTypeFood
	}
	return nil
}

// ListCategories returns the categories visible under scope.
func (s *ConsoleService) ListCategories(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]ConsoleCategoryView, int64, error) {
	cats, total, err := s.repo.ListCategories(ctx, scope, filter, page)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ConsoleCategoryView, 0, len(cats))
	for _, c := range cats {
		views = append(views, categoryToView(c))
	}
	return views, total, nil
}

// GetCategory returns a single category visible under scope.
func (s *ConsoleService) GetCategory(ctx context.Context, scope ConsoleScope, id int64) (ConsoleCategoryView, error) {
	c, err := s.repo.GetCategory(ctx, scope, id)
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	return categoryToView(c), nil
}

// CreateCategory creates a category under scope.
func (s *ConsoleService) CreateCategory(ctx context.Context, scope ConsoleScope, in CategoryInput) (ConsoleCategoryView, error) {
	storeID, err := storeIDForWrite(scope, in.StoreID)
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return ConsoleCategoryView{}, apperr.Invalid("catalog: category name is required")
	}
	in.StoreID = storeID
	in.Name = strings.TrimSpace(in.Name)
	c, err := s.repo.CreateCategory(ctx, scope, in)
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	return categoryToView(c), nil
}

// UpdateCategory updates a category under scope.
func (s *ConsoleService) UpdateCategory(ctx context.Context, scope ConsoleScope, id int64, in CategoryInput) (ConsoleCategoryView, error) {
	storeID, err := storeIDForWrite(scope, in.StoreID)
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return ConsoleCategoryView{}, apperr.Invalid("catalog: category name is required")
	}
	in.StoreID = storeID
	in.Name = strings.TrimSpace(in.Name)
	c, err := s.repo.UpdateCategory(ctx, scope, id, in)
	if err != nil {
		return ConsoleCategoryView{}, err
	}
	return categoryToView(c), nil
}

// DeleteCategory deletes a category under scope.
func (s *ConsoleService) DeleteCategory(ctx context.Context, scope ConsoleScope, id int64) error {
	return s.repo.DeleteCategory(ctx, scope, id)
}

// ListItems returns the items visible under scope.
func (s *ConsoleService) ListItems(ctx context.Context, scope ConsoleScope, filter ConsoleListFilter, page httpx.Page) ([]ConsoleItemView, int64, error) {
	items, total, err := s.repo.ListItems(ctx, scope, filter, page)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ConsoleItemView, 0, len(items))
	for _, it := range items {
		views = append(views, s.itemToView(ctx, it))
	}
	return views, total, nil
}

// GetItem returns a single item visible under scope.
func (s *ConsoleService) GetItem(ctx context.Context, scope ConsoleScope, id int64) (ConsoleItemView, error) {
	it, err := s.repo.GetItem(ctx, scope, id)
	if err != nil {
		return ConsoleItemView{}, err
	}
	return s.itemToView(ctx, it), nil
}

// CreateItem creates an item under scope.
func (s *ConsoleService) CreateItem(ctx context.Context, scope ConsoleScope, in ItemInput) (ConsoleItemView, error) {
	if err := s.validateItemInput(ctx, scope, &in); err != nil {
		return ConsoleItemView{}, err
	}
	it, err := s.repo.CreateItem(ctx, scope, in)
	if err != nil {
		return ConsoleItemView{}, err
	}
	return s.itemToView(ctx, it), nil
}

// UpdateItem updates an item under scope.
func (s *ConsoleService) UpdateItem(ctx context.Context, scope ConsoleScope, id int64, in ItemInput) (ConsoleItemView, error) {
	if err := s.validateItemInput(ctx, scope, &in); err != nil {
		return ConsoleItemView{}, err
	}
	it, err := s.repo.UpdateItem(ctx, scope, id, in)
	if err != nil {
		return ConsoleItemView{}, err
	}
	return s.itemToView(ctx, it), nil
}

// DeleteItem deletes an item under scope.
func (s *ConsoleService) DeleteItem(ctx context.Context, scope ConsoleScope, id int64) error {
	return s.repo.DeleteItem(ctx, scope, id)
}

// ListVariants returns the variants of itemID visible under scope.
func (s *ConsoleService) ListVariants(ctx context.Context, scope ConsoleScope, itemID int64, page httpx.Page) ([]VariantView, int64, error) {
	variants, total, err := s.repo.ListVariants(ctx, scope, itemID, page)
	if err != nil {
		return nil, 0, err
	}
	views := make([]VariantView, 0, len(variants))
	for _, v := range variants {
		views = append(views, variantToView(v))
	}
	return views, total, nil
}

// GetVariant returns a single variant of itemID visible under scope.
func (s *ConsoleService) GetVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) (VariantView, error) {
	v, err := s.repo.GetVariant(ctx, scope, itemID, id)
	if err != nil {
		return VariantView{}, err
	}
	return variantToView(v), nil
}

// CreateVariant creates a variant on itemID under scope.
func (s *ConsoleService) CreateVariant(ctx context.Context, scope ConsoleScope, itemID int64, in VariantInput) (VariantView, error) {
	v, err := s.repo.CreateVariant(ctx, scope, itemID, in)
	if err != nil {
		return VariantView{}, err
	}
	return variantToView(v), nil
}

// UpdateVariant updates a variant on itemID under scope.
func (s *ConsoleService) UpdateVariant(ctx context.Context, scope ConsoleScope, itemID, id int64, in VariantInput) (VariantView, error) {
	v, err := s.repo.UpdateVariant(ctx, scope, itemID, id, in)
	if err != nil {
		return VariantView{}, err
	}
	return variantToView(v), nil
}

// DeleteVariant deletes a variant on itemID under scope.
func (s *ConsoleService) DeleteVariant(ctx context.Context, scope ConsoleScope, itemID, id int64) error {
	return s.repo.DeleteVariant(ctx, scope, itemID, id)
}

// ConsoleHandler exposes the admin and store console CRUD endpoints for
// catalog categories, items and variants.
type ConsoleHandler struct {
	svc *ConsoleService
}

// NewConsoleHandler builds the catalog console handler.
func NewConsoleHandler(svc *ConsoleService) *ConsoleHandler { return &ConsoleHandler{svc: svc} }

func storeScopeFromContext(c *gin.Context) (ConsoleScope, bool) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return ConsoleScope{}, false
	}
	return ConsoleScope{StoreID: &storeID}, true
}

func adminScope() ConsoleScope { return ConsoleScope{} }

// --- Admin console (audience: admin, no store filter) ---

// Categories handles GET /admin/catalog/categories.
func (h *ConsoleHandler) Categories(c *gin.Context) {
	h.listCategories(c, adminScope())
}

// GetCategory handles GET /admin/catalog/categories/{id}.
func (h *ConsoleHandler) GetCategory(c *gin.Context) {
	h.getCategory(c, adminScope())
}

// CreateCategory handles POST /admin/catalog/categories.
func (h *ConsoleHandler) CreateCategory(c *gin.Context) {
	h.createCategory(c, adminScope())
}

// UpdateCategory handles PUT /admin/catalog/categories/{id}.
func (h *ConsoleHandler) UpdateCategory(c *gin.Context) {
	h.updateCategory(c, adminScope())
}

// DeleteCategory handles DELETE /admin/catalog/categories/{id}.
func (h *ConsoleHandler) DeleteCategory(c *gin.Context) {
	h.deleteCategory(c, adminScope())
}

// Items handles GET /admin/catalog/items.
func (h *ConsoleHandler) Items(c *gin.Context) {
	h.listItems(c, adminScope())
}

// GetItem handles GET /admin/catalog/items/{id}.
func (h *ConsoleHandler) GetItem(c *gin.Context) {
	h.getItem(c, adminScope())
}

// CreateItem handles POST /admin/catalog/items.
func (h *ConsoleHandler) CreateItem(c *gin.Context) {
	h.createItem(c, adminScope())
}

// UpdateItem handles PUT /admin/catalog/items/{id}.
func (h *ConsoleHandler) UpdateItem(c *gin.Context) {
	h.updateItem(c, adminScope())
}

// DeleteItem handles DELETE /admin/catalog/items/{id}.
func (h *ConsoleHandler) DeleteItem(c *gin.Context) {
	h.deleteItem(c, adminScope())
}

// Variants handles GET /admin/catalog/items/{itemID}/variants.
func (h *ConsoleHandler) Variants(c *gin.Context) {
	h.listVariants(c, adminScope())
}

// GetVariant handles GET /admin/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) GetVariant(c *gin.Context) {
	h.getVariant(c, adminScope())
}

// CreateVariant handles POST /admin/catalog/items/{itemID}/variants.
func (h *ConsoleHandler) CreateVariant(c *gin.Context) {
	h.createVariant(c, adminScope())
}

// UpdateVariant handles PUT /admin/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) UpdateVariant(c *gin.Context) {
	h.updateVariant(c, adminScope())
}

// DeleteVariant handles DELETE /admin/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) DeleteVariant(c *gin.Context) {
	h.deleteVariant(c, adminScope())
}

// --- Store console (audience: store, scope pinned from JWT) ---

// StoreCategories handles GET /store/catalog/categories.
func (h *ConsoleHandler) StoreCategories(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.listCategories(c, scope)
}

// StoreGetCategory handles GET /store/catalog/categories/{id}.
func (h *ConsoleHandler) StoreGetCategory(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.getCategory(c, scope)
}

// StoreCreateCategory handles POST /store/catalog/categories.
func (h *ConsoleHandler) StoreCreateCategory(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.createCategory(c, scope)
}

// StoreUpdateCategory handles PUT /store/catalog/categories/{id}.
func (h *ConsoleHandler) StoreUpdateCategory(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.updateCategory(c, scope)
}

// StoreDeleteCategory handles DELETE /store/catalog/categories/{id}.
func (h *ConsoleHandler) StoreDeleteCategory(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.deleteCategory(c, scope)
}

// StoreItems handles GET /store/catalog/items.
func (h *ConsoleHandler) StoreItems(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.listItems(c, scope)
}

// StoreGetItem handles GET /store/catalog/items/{id}.
func (h *ConsoleHandler) StoreGetItem(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.getItem(c, scope)
}

// StoreCreateItem handles POST /store/catalog/items.
func (h *ConsoleHandler) StoreCreateItem(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.createItem(c, scope)
}

// StoreUpdateItem handles PUT /store/catalog/items/{id}.
func (h *ConsoleHandler) StoreUpdateItem(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.updateItem(c, scope)
}

// StoreDeleteItem handles DELETE /store/catalog/items/{id}.
func (h *ConsoleHandler) StoreDeleteItem(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.deleteItem(c, scope)
}

// StoreVariants handles GET /store/catalog/items/{itemID}/variants.
func (h *ConsoleHandler) StoreVariants(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.listVariants(c, scope)
}

// StoreGetVariant handles GET /store/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) StoreGetVariant(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.getVariant(c, scope)
}

// StoreCreateVariant handles POST /store/catalog/items/{itemID}/variants.
func (h *ConsoleHandler) StoreCreateVariant(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.createVariant(c, scope)
}

// StoreUpdateVariant handles PUT /store/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) StoreUpdateVariant(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.updateVariant(c, scope)
}

// StoreDeleteVariant handles DELETE /store/catalog/items/{itemID}/variants/{id}.
func (h *ConsoleHandler) StoreDeleteVariant(c *gin.Context) {
	scope, ok := storeScopeFromContext(c)
	if !ok {
		return
	}
	h.deleteVariant(c, scope)
}

// --- shared implementations ---

func (h *ConsoleHandler) listCategories(c *gin.Context, scope ConsoleScope) {
	page := httpx.ParsePage(c)
	filter, err := consoleListFilterFromQuery(c, false)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.ListCategories(c.Request.Context(), scope, filter, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) getCategory(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetCategory(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createCategory(c *gin.Context, scope ConsoleScope) {
	var in CategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateCategory(c.Request.Context(), scope, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func (h *ConsoleHandler) updateCategory(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in CategoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateCategory(c.Request.Context(), scope, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteCategory(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), scope, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}

func (h *ConsoleHandler) listItems(c *gin.Context, scope ConsoleScope) {
	page := httpx.ParsePage(c)
	filter, err := consoleListFilterFromQuery(c, true)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.ListItems(c.Request.Context(), scope, filter, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func consoleListFilterFromQuery(c *gin.Context, includeCategory bool) (ConsoleListFilter, error) {
	filter := ConsoleListFilter{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
	}
	storeID, err := positiveQueryID(c, "storeId")
	if err != nil {
		return ConsoleListFilter{}, err
	}
	filter.StoreID = storeID
	if includeCategory {
		categoryID, err := positiveQueryID(c, "categoryId")
		if err != nil {
			return ConsoleListFilter{}, err
		}
		filter.CategoryID = categoryID
	}
	return filter, nil
}

func positiveQueryID(c *gin.Context, key string) (*int64, error) {
	value := c.Query(key)
	if value == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return nil, apperr.Invalid("catalog: invalid " + key)
	}
	return &id, nil
}

func (h *ConsoleHandler) getItem(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetItem(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createItem(c *gin.Context, scope ConsoleScope) {
	var in ItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateItem(c.Request.Context(), scope, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func (h *ConsoleHandler) updateItem(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in ItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateItem(c.Request.Context(), scope, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteItem(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), scope, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}

func (h *ConsoleHandler) listVariants(c *gin.Context, scope ConsoleScope) {
	itemID, err := pathID(c, "itemID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListVariants(c.Request.Context(), scope, itemID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) getVariant(c *gin.Context, scope ConsoleScope) {
	itemID, err := pathID(c, "itemID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetVariant(c.Request.Context(), scope, itemID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createVariant(c *gin.Context, scope ConsoleScope) {
	itemID, err := pathID(c, "itemID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in VariantInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateVariant(c.Request.Context(), scope, itemID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func (h *ConsoleHandler) updateVariant(c *gin.Context, scope ConsoleScope) {
	itemID, err := pathID(c, "itemID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in VariantInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateVariant(c.Request.Context(), scope, itemID, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteVariant(c *gin.Context, scope ConsoleScope) {
	itemID, err := pathID(c, "itemID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteVariant(c.Request.Context(), scope, itemID, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}
