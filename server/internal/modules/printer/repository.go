package printer

import (
	"context"
	"database/sql"
	"errors"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the printer_devices persistence port. A nil storeID on List
// returns every store's devices (admin); a non-nil storeID restricts to that
// store (store console).
type Repository interface {
	List(ctx context.Context, storeID *int64) ([]Device, error)
	Get(ctx context.Context, id int64) (Device, error)
	Create(ctx context.Context, d Device) (Device, error)
	Update(ctx context.Context, d Device) (Device, error)
	Delete(ctx context.Context, id int64) error
}

const deviceColumns = `id, store_id, name, provider, device_sn, device_key, status, created_at, updated_at`

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL-backed printer device repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func scanDevice(row interface{ Scan(...any) error }) (Device, error) {
	var d Device
	err := row.Scan(&d.ID, &d.StoreID, &d.Name, &d.Provider, &d.DeviceSN,
		&d.DeviceKey, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *sqlRepository) List(ctx context.Context, storeID *int64) ([]Device, error) {
	q := `SELECT ` + deviceColumns + ` FROM printer_devices pd
		WHERE EXISTS (SELECT 1 FROM stores live_store WHERE live_store.id = pd.store_id)`
	args := []any{}
	if storeID != nil {
		q += ` AND pd.store_id = ?`
		args = append(args, *storeID)
	}
	q += ` ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Device, 0)
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *sqlRepository) Get(ctx context.Context, id int64) (Device, error) {
	const q = `SELECT ` + deviceColumns + ` FROM printer_devices WHERE id = ?`
	d, err := scanDevice(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, apperr.NotFound("printer device not found")
	}
	if err != nil {
		return Device{}, apperr.Internal(err)
	}
	return d, nil
}

func (r *sqlRepository) Create(ctx context.Context, d Device) (Device, error) {
	const q = `INSERT INTO printer_devices
		(store_id, name, provider, device_sn, device_key, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, d.StoreID, d.Name, d.Provider, d.DeviceSN, d.DeviceKey, d.Status)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return Device{}, apperr.Invalid("device with this provider and SN already exists")
		}
		return Device{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Device{}, apperr.Internal(err)
	}
	return r.Get(ctx, id)
}

// Update applies a full-replace update. A missing row surfaces as NotFound via
// the trailing Get.
func (r *sqlRepository) Update(ctx context.Context, d Device) (Device, error) {
	const q = `UPDATE printer_devices SET name = ?, provider = ?, device_sn = ?,
		device_key = ?, status = ?, updated_at = NOW()
		WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, d.Name, d.Provider, d.DeviceSN, d.DeviceKey, d.Status, d.ID)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return Device{}, apperr.Invalid("device with this provider and SN already exists")
		}
		return Device{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.Get(ctx, d.ID); err != nil {
			return Device{}, err
		}
	}
	return r.Get(ctx, d.ID)
}

func (r *sqlRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM printer_devices WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("printer device not found")
	}
	return nil
}
