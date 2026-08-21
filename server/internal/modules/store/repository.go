package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/inwardclub/server/internal/platform/audit"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the store persistence port.
type Repository interface {
	ListActiveStores(ctx context.Context, limit, offset int) ([]Store, int64, error)
	GetStore(ctx context.Context, id int64) (Store, error)
	UpdateStoreProfile(ctx context.Context, storeID int64, fields UpdateProfileRequest) (Store, error)
	UpdateStoreStatus(ctx context.Context, storeID int64, status string) (Store, error)
	CreateStore(ctx context.Context, input StoreInput) (Store, error)
	UpdateStore(ctx context.Context, id int64, input StoreInput) (Store, error)
	DeleteStore(ctx context.Context, id int64, auditEntry audit.Entry) error

	GetStoreSettings(ctx context.Context, storeID int64) (StoreSettings, error)
	UpsertStoreSettings(ctx context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error)
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
	const q = `SELECT ` + storeColumns + ` FROM stores WHERE id = ? AND status <> ?`
	s, err := scanStore(r.db.QueryRowContext(ctx, q, id, StatusDeleted))
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

// DeleteStore permanently removes the store row. Historical orders keep their
// immutable store snapshots/ids, while live store-scoped sessions are revoked.
func (r *sqlRepository) DeleteStore(ctx context.Context, id int64, auditEntry audit.Entry) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		const selectStore = `SELECT ` + storeColumns + ` FROM stores WHERE id = ? AND status <> ? FOR UPDATE`
		before, err := scanStore(tx.QueryRowContext(ctx, selectStore, id, StatusDeleted))
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("store not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW() WHERE store_id = ?`, id,
		); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE staff_accounts SET status = 'disabled', token_version = token_version + 1, updated_at = NOW() WHERE store_id = ?`, id,
		); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE printer_devices SET status = 'disabled', updated_at = NOW() WHERE store_id = ?`, id,
		); err != nil {
			return apperr.Internal(err)
		}

		auditEntry.StoreID = id
		auditEntry.Before = map[string]any{"id": before.ID, "name": before.Name, "status": before.Status}
		auditEntry.After = map[string]any{"id": before.ID, "name": before.Name, "deleted": true}
		if err := audit.RecordTx(ctx, tx, auditEntry); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM store_settings WHERE store_id = ?`, id); err != nil {
			return apperr.Internal(err)
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, id)
		if err != nil {
			return apperr.Internal(err)
		}
		if affected, _ := res.RowsAffected(); affected != 1 {
			return apperr.NotFound("store not found")
		}
		return nil
	})
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
