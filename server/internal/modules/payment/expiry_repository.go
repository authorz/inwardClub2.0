package payment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// collectionExpiryBatch bounds how many due collection orders one sweep run
// closes; the scheduled cadence (spec §11, every minute) drains any remainder.
const collectionExpiryBatch = 1000

// ExpireCollections snapshots the due pending collection orders (bounded,
// unlocked) then closes each in its own transaction, re-checking under a row
// lock. The lock + status guards mean a settlement in flight for the same order
// and this expiry can never both win: whichever commits first, the other reads a
// non-pending status and skips.
func (r *storeSQLRepository) ExpireCollections(ctx context.Context, now time.Time) (int64, error) {
	const sel = `SELECT id FROM offline_collection_orders
		WHERE status = ? AND expires_at < ?
		ORDER BY id
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, sel, CollectionPending, now, collectionExpiryBatch)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, apperr.Internal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, apperr.Internal(err)
	}
	rows.Close()

	var n int64
	for _, id := range ids {
		expired, err := r.expireCollection(ctx, id, now)
		if err != nil {
			return n, err
		}
		n += expired
	}
	return n, nil
}

// expireCollection closes one pending collection order atomically. It locks the
// collection together with its payment order exactly as SettleOffline does, so
// the two paths serialize on the same rows.
func (r *storeSQLRepository) expireCollection(ctx context.Context, id int64, now time.Time) (int64, error) {
	var expired int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			collectionStatus string
			paymentID        int64
			paymentStatus    string
			businessID       int64
		)
		const lock = `SELECT oco.status, po.id, po.status, po.business_order_id
			FROM offline_collection_orders oco
			JOIN payment_orders po ON po.id = oco.payment_order_id
			WHERE oco.id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, lock, id).
			Scan(&collectionStatus, &paymentID, &paymentStatus, &businessID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // gone; nothing to do
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if collectionStatus != CollectionPending || paymentStatus != paymentPending {
			return nil // paid or cancelled since the snapshot; idempotent skip
		}
		const expCol = `UPDATE offline_collection_orders SET status = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		if _, err := tx.ExecContext(ctx, expCol, CollectionExpired, now, id, CollectionPending); err != nil {
			return apperr.Internal(err)
		}
		const expPO = `UPDATE payment_orders SET status = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		if _, err := tx.ExecContext(ctx, expPO, paymentExpired, now, paymentID, paymentPending); err != nil {
			return apperr.Internal(err)
		}
		const closeBO = `UPDATE business_orders SET order_status = 'expired', updated_at = ?
			WHERE id = ? AND payment_status = 'unpaid'`
		if _, err := tx.ExecContext(ctx, closeBO, now, businessID); err != nil {
			return apperr.Internal(err)
		}
		expired = 1
		return nil
	})
	return expired, err
}
