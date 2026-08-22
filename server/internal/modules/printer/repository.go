package printer

import (
	"context"
	"database/sql"
	"errors"

	"github.com/inwardclub/server/internal/platform/audit"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/idempotency"
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
	AdminCreate(ctx context.Context, d Device, idemKey string, entry audit.Entry) (Device, error)
	AdminUpdate(ctx context.Context, id int64, patch DevicePatch, idemKey string, entry audit.Entry) (Device, error)
	AdminDelete(ctx context.Context, id int64, idemKey string, entry audit.Entry) error
}

const deviceColumns = `id, store_id, name, provider, device_sn, device_key, status, sound_enabled, created_at, updated_at`

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL-backed printer device repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func scanDevice(row interface{ Scan(...any) error }) (Device, error) {
	var d Device
	err := row.Scan(&d.ID, &d.StoreID, &d.Name, &d.Provider, &d.DeviceSN,
		&d.DeviceKey, &d.Status, &d.SoundEnabled, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *sqlRepository) List(ctx context.Context, storeID *int64) ([]Device, error) {
	q := `SELECT ` + deviceColumns + ` FROM printer_devices pd
		WHERE EXISTS (
			SELECT 1 FROM stores live_store
			WHERE live_store.id = pd.store_id AND live_store.status <> 'deleted'
		)`
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
		(store_id, name, provider, device_sn, device_key, status, sound_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, d.StoreID, d.Name, d.Provider, d.DeviceSN, d.DeviceKey, d.Status, d.SoundEnabled)
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
		device_key = ?, status = ?, sound_enabled = ?, updated_at = NOW()
		WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, d.Name, d.Provider, d.DeviceSN, d.DeviceKey, d.Status, d.SoundEnabled, d.ID)
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

func (r *sqlRepository) AdminCreate(ctx context.Context, d Device, idemKey string, entry audit.Entry) (Device, error) {
	var created Device
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := idempotency.Claim(ctx, tx, "admin/printer-device-create", idemKey, "store", d.StoreID); err != nil {
			return err
		}
		var storeExists int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM stores WHERE id = ? AND status <> 'deleted'`, d.StoreID,
		).Scan(&storeExists); err != nil {
			return apperr.Internal(err)
		}
		if storeExists == 0 {
			return apperr.Invalid("selected store does not exist")
		}
		const q = `INSERT INTO printer_devices
			(store_id, name, provider, device_sn, device_key, status, sound_enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
		res, err := tx.ExecContext(ctx, q, d.StoreID, d.Name, d.Provider, d.DeviceSN, d.DeviceKey, d.Status, d.SoundEnabled)
		if err != nil {
			return printerWriteError(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		created, err = getDeviceTx(ctx, tx, id, false)
		if err != nil {
			return err
		}
		entry.StoreID = created.StoreID
		entry.TargetID = created.ID
		entry.After = auditDevice(created)
		return audit.RecordTx(ctx, tx, entry)
	})
	return created, err
}

func (r *sqlRepository) AdminUpdate(ctx context.Context, id int64, patch DevicePatch, idemKey string, entry audit.Entry) (Device, error) {
	var updated Device
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := idempotency.Claim(ctx, tx, "admin/printer-device-update", idemKey, "printer_device", id); err != nil {
			return err
		}
		before, err := getDeviceTx(ctx, tx, id, true)
		if err != nil {
			return err
		}
		updated = before
		applyPatch(&updated, patch)
		const q = `UPDATE printer_devices SET name = ?, provider = ?, device_sn = ?,
			device_key = ?, status = ?, sound_enabled = ?, updated_at = NOW() WHERE id = ?`
		if _, err := tx.ExecContext(ctx, q, updated.Name, updated.Provider, updated.DeviceSN,
			updated.DeviceKey, updated.Status, updated.SoundEnabled, updated.ID); err != nil {
			return printerWriteError(err)
		}
		updated, err = getDeviceTx(ctx, tx, id, false)
		if err != nil {
			return err
		}
		entry.StoreID = updated.StoreID
		entry.TargetID = updated.ID
		entry.Before = auditDevice(before)
		entry.After = auditDevice(updated)
		return audit.RecordTx(ctx, tx, entry)
	})
	return updated, err
}

func (r *sqlRepository) AdminDelete(ctx context.Context, id int64, idemKey string, entry audit.Entry) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := idempotency.Claim(ctx, tx, "admin/printer-device-delete", idemKey, "printer_device", id); err != nil {
			return err
		}
		before, err := getDeviceTx(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM printer_devices WHERE id = ?`, id); err != nil {
			return apperr.Internal(err)
		}
		entry.StoreID = before.StoreID
		entry.TargetID = before.ID
		entry.Before = auditDevice(before)
		entry.After = map[string]any{"deleted": true}
		return audit.RecordTx(ctx, tx, entry)
	})
}

func getDeviceTx(ctx context.Context, tx *sql.Tx, id int64, lock bool) (Device, error) {
	q := `SELECT ` + deviceColumns + ` FROM printer_devices WHERE id = ?`
	if lock {
		q += ` FOR UPDATE`
	}
	d, err := scanDevice(tx.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, apperr.NotFound("printer device not found")
	}
	if err != nil {
		return Device{}, apperr.Internal(err)
	}
	return d, nil
}

func printerWriteError(err error) error {
	if platdb.IsDuplicate(err) {
		return apperr.Invalid("device with this provider and SN already exists")
	}
	return apperr.Internal(err)
}

func auditDevice(d Device) map[string]any {
	return map[string]any{
		"id": d.ID, "storeId": d.StoreID, "name": d.Name, "provider": d.Provider,
		"deviceSn": d.DeviceSN, "deviceKeyConfigured": d.DeviceKey != "", "status": d.Status,
		"soundEnabled": d.SoundEnabled,
	}
}
