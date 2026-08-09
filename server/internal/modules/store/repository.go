package store

import (
	"context"
	"database/sql"
	"errors"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the store/banner persistence port.
type Repository interface {
	ListActiveStores(ctx context.Context, limit, offset int) ([]Store, int64, error)
	GetStore(ctx context.Context, id int64) (Store, error)
	ListStoreBanners(ctx context.Context, storeID int64) ([]Banner, error)
	UpdateStoreProfile(ctx context.Context, storeID int64, fields UpdateProfileRequest) (Store, error)
	UpdateStoreStatus(ctx context.Context, storeID int64, status string) (Store, error)
	CreateStore(ctx context.Context, input StoreInput) (Store, error)
	UpdateStore(ctx context.Context, id int64, input StoreInput) (Store, error)

	GetStoreSettings(ctx context.Context, storeID int64) (StoreSettings, error)
	UpsertStoreSettings(ctx context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error)

	// Banner CRUD for the admin/store consoles. ListBanners with a nil storeID
	// returns every banner (admin scope); a non-nil storeID restricts to that
	// store's own banners.
	ListBanners(ctx context.Context, storeID *int64) ([]Banner, error)
	GetBanner(ctx context.Context, id int64) (Banner, error)
	CreateBanner(ctx context.Context, b Banner) (Banner, error)
	UpdateBanner(ctx context.Context, b Banner) (Banner, error)
	DeleteBanner(ctx context.Context, id int64) error
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL store repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

const storeColumns = `id, name, logo_asset_id, COALESCE(phone,''), customer_service_qr_asset_id, address,
	latitude, longitude, business_hours, status, created_at, updated_at`

func scanStore(row interface{ Scan(...any) error }) (Store, error) {
	var s Store
	err := row.Scan(&s.ID, &s.Name, &s.LogoAssetID, &s.Phone, &s.CustomerServiceQRAssetID, &s.Address,
		&s.Latitude, &s.Longitude, &s.BusinessHours, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *sqlRepository) ListActiveStores(ctx context.Context, limit, offset int) ([]Store, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stores WHERE status = ?`, StatusActive).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT ` + storeColumns + ` FROM stores WHERE status = ?
		ORDER BY id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, StatusActive, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Store
	for rows.Next() {
		s, err := scanStore(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetStore(ctx context.Context, id int64) (Store, error) {
	const q = `SELECT ` + storeColumns + ` FROM stores WHERE id = ?`
	s, err := scanStore(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Store{}, apperr.NotFound("store not found")
	}
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	return s, nil
}

func (r *sqlRepository) UpdateStoreProfile(ctx context.Context, storeID int64, fields UpdateProfileRequest) (Store, error) {
	if storeID <= 0 {
		return Store{}, apperr.Invalid("invalid storeID")
	}
	const q = `UPDATE stores SET name = ?, phone = ?, customer_service_qr_asset_id = ?, address = ?, business_hours = ?,
		latitude = ?, longitude = ?, logo_asset_id = ?, updated_at = NOW()
		WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, fields.Name, fields.Phone, fields.CustomerServiceQRAssetID, fields.Address, fields.BusinessHours,
		fields.Latitude, fields.Longitude, fields.LogoAssetID, storeID)
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	return r.GetStore(ctx, storeID)
}

// CreateStore inserts a new store profile and returns the persisted row. New
// stores default to active.
func (r *sqlRepository) CreateStore(ctx context.Context, input StoreInput) (Store, error) {
	const q = `INSERT INTO stores
		(name, logo_asset_id, phone, customer_service_qr_asset_id, address, latitude, longitude, business_hours, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, input.Name, input.LogoAssetID, input.Phone, input.CustomerServiceQRAssetID, input.Address,
		input.Latitude, input.Longitude, input.BusinessHours, StatusActive)
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	return r.GetStore(ctx, id)
}

// UpdateStore applies a full-replace update to an existing store profile. A
// missing row surfaces as NotFound via the trailing GetStore.
func (r *sqlRepository) UpdateStore(ctx context.Context, id int64, input StoreInput) (Store, error) {
	const q = `UPDATE stores SET name = ?, phone = ?, customer_service_qr_asset_id = ?, address = ?, business_hours = ?,
		latitude = ?, longitude = ?, logo_asset_id = ?, updated_at = NOW()
		WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, input.Name, input.Phone, input.CustomerServiceQRAssetID, input.Address, input.BusinessHours,
		input.Latitude, input.Longitude, input.LogoAssetID, id)
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Distinguish a genuinely absent row from a no-op update.
		if _, err := r.GetStore(ctx, id); err != nil {
			return Store{}, err
		}
	}
	return r.GetStore(ctx, id)
}

// UpdateStoreStatus updates only the caller's own store's status.
func (r *sqlRepository) UpdateStoreStatus(ctx context.Context, storeID int64, status string) (Store, error) {
	if storeID <= 0 {
		return Store{}, apperr.Invalid("invalid storeID")
	}
	const q = `UPDATE stores SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, status, storeID)
	if err != nil {
		return Store{}, apperr.Internal(err)
	}
	return r.GetStore(ctx, storeID)
}

// GetStoreSettings returns the caller's own store's settings blob. A store
// with no settings row yet returns a zero-value StoreSettings (empty JSON
// object, zero UpdatedAt) rather than an error.
func (r *sqlRepository) GetStoreSettings(ctx context.Context, storeID int64) (StoreSettings, error) {
	const q = `SELECT store_id, settings_json, updated_at FROM store_settings WHERE store_id = ?`
	var s StoreSettings
	err := r.db.QueryRowContext(ctx, q, storeID).Scan(&s.StoreID, &s.SettingsJSON, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return StoreSettings{StoreID: storeID, SettingsJSON: []byte(`{}`)}, nil
	}
	if err != nil {
		return StoreSettings{}, apperr.Internal(err)
	}
	return s, nil
}

// UpsertStoreSettings applies a full-replace update of the store's settings
// blob, creating the row on first write.
func (r *sqlRepository) UpsertStoreSettings(ctx context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error) {
	if storeID <= 0 {
		return StoreSettings{}, apperr.Invalid("invalid storeID")
	}
	const q = `INSERT INTO store_settings (store_id, settings_json, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE settings_json = VALUES(settings_json), updated_at = NOW()`
	if _, err := r.db.ExecContext(ctx, q, storeID, settingsJSON); err != nil {
		return StoreSettings{}, apperr.Internal(err)
	}
	return r.GetStoreSettings(ctx, storeID)
}

const bannerColumns = `id, scope_type, store_id, title, asset_id, link_url, sort_order, status, created_at, updated_at`

func scanBanner(row interface{ Scan(...any) error }) (Banner, error) {
	var b Banner
	err := row.Scan(&b.ID, &b.ScopeType, &b.StoreID, &b.Title, &b.AssetID,
		&b.LinkURL, &b.SortOrder, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

// ListBanners returns banners for the console. A nil storeID lists every
// banner (admin); a non-nil storeID restricts to that store's own banners.
func (r *sqlRepository) ListBanners(ctx context.Context, storeID *int64) ([]Banner, error) {
	q := `SELECT ` + bannerColumns + ` FROM banners`
	args := []any{}
	if storeID != nil {
		q += ` WHERE store_id = ?`
		args = append(args, *storeID)
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Banner
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *sqlRepository) GetBanner(ctx context.Context, id int64) (Banner, error) {
	const q = `SELECT ` + bannerColumns + ` FROM banners WHERE id = ?`
	b, err := scanBanner(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Banner{}, apperr.NotFound("banner not found")
	}
	if err != nil {
		return Banner{}, apperr.Internal(err)
	}
	return b, nil
}

func (r *sqlRepository) CreateBanner(ctx context.Context, b Banner) (Banner, error) {
	const q = `INSERT INTO banners
		(scope_type, store_id, title, asset_id, link_url, sort_order, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, b.ScopeType, b.StoreID, b.Title, b.AssetID,
		b.LinkURL, b.SortOrder, b.Status)
	if err != nil {
		return Banner{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Banner{}, apperr.Internal(err)
	}
	return r.GetBanner(ctx, id)
}

// UpdateBanner applies a full-replace update to an existing banner. A missing
// row surfaces as NotFound via the trailing GetBanner.
func (r *sqlRepository) UpdateBanner(ctx context.Context, b Banner) (Banner, error) {
	const q = `UPDATE banners SET scope_type = ?, store_id = ?, title = ?, asset_id = ?,
		link_url = ?, sort_order = ?, status = ?, updated_at = NOW()
		WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, b.ScopeType, b.StoreID, b.Title, b.AssetID,
		b.LinkURL, b.SortOrder, b.Status, b.ID)
	if err != nil {
		return Banner{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.GetBanner(ctx, b.ID); err != nil {
			return Banner{}, err
		}
	}
	return r.GetBanner(ctx, b.ID)
}

func (r *sqlRepository) DeleteBanner(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM banners WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("banner not found")
	}
	return nil
}

func (r *sqlRepository) ListStoreBanners(ctx context.Context, storeID int64) ([]Banner, error) {
	// Store-scoped banners plus global banners apply to a store's home.
	const q = `SELECT id, scope_type, store_id, title, asset_id, link_url, sort_order, status
		FROM banners
		WHERE status = 'active' AND (scope_type = 'global' OR store_id = ?)
		ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Banner
	for rows.Next() {
		var b Banner
		if err := rows.Scan(&b.ID, &b.ScopeType, &b.StoreID, &b.Title, &b.AssetID, &b.LinkURL, &b.SortOrder, &b.Status); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
