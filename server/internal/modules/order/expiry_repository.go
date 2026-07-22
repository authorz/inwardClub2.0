package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// expirySweepBatch bounds how many due activity orders one sweep run closes, so
// a large backlog is drained over successive runs rather than in one long
// transaction burst. The scheduled cadence (spec §11) drains the remainder.
const expirySweepBatch = 1000

// ExpireActivityOrders snapshots the due unpaid activity orders (bounded,
// unlocked) then closes each in its own transaction. Doing the snapshot outside
// the lock keeps the sweep from holding many rows at once; the per-order
// transaction re-reads under a FOR UPDATE lock, so a settlement that paid the
// order since the snapshot is seen and the order is skipped.
func (r *sqlRepository) ExpireActivityOrders(ctx context.Context, createdBefore, now time.Time) (int64, error) {
	const sel = `SELECT ao.id
		FROM activity_orders ao
		JOIN business_orders bo ON bo.id = ao.business_order_id
		JOIN payment_orders po ON po.business_order_id = bo.id
		WHERE ao.status = 'created' AND bo.payment_status = 'unpaid'
		  AND po.status = ? AND bo.created_at < ?
		ORDER BY ao.id
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, sel, PaymentStatusPending, createdBefore, expirySweepBatch)
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
		expired, err := r.expireActivityOrder(ctx, id, now)
		if err != nil {
			return n, err
		}
		n += expired
	}
	return n, nil
}

// expireActivityOrder closes one unpaid activity order atomically. It locks the
// order spine (activity order + business order + payment order) and re-validates
// under the lock; if a settlement paid it in the meantime the payment order is
// no longer pending and the order is left untouched (idempotent skip).
func (r *sqlRepository) expireActivityOrder(ctx context.Context, activityOrderID int64, now time.Time) (int64, error) {
	var expired int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			businessID int64
			paymentID  int64
			aoStatus   string
			poStatus   string
			payStatus  string
		)
		const lock = `SELECT ao.business_order_id, po.id, ao.status, po.status, bo.payment_status
			FROM activity_orders ao
			JOIN business_orders bo ON bo.id = ao.business_order_id
			JOIN payment_orders po ON po.business_order_id = bo.id
			WHERE ao.id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, lock, activityOrderID).
			Scan(&businessID, &paymentID, &aoStatus, &poStatus, &payStatus)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // order (or its spine) is gone; nothing to do
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if aoStatus != "created" || poStatus != PaymentStatusPending || payStatus != "unpaid" {
			return nil // settled or already closed since the snapshot; idempotent skip
		}

		// Release the stock reserved at creation for this order's still-pending
		// tickets, back down to (but never below) zero.
		const release = `UPDATE activity_ticket_types tt
			JOIN (
				SELECT ticket_type_id, COUNT(*) AS c
				FROM tickets
				WHERE activity_order_id = ? AND status = ?
				GROUP BY ticket_type_id
			) g ON g.ticket_type_id = tt.id
			SET tt.sold_quantity = GREATEST(tt.sold_quantity - g.c, 0), tt.updated_at = ?`
		if _, err := tx.ExecContext(ctx, release, activityOrderID, TicketStatusPending, now); err != nil {
			return apperr.Internal(err)
		}
		const expTickets = `UPDATE tickets SET status = ?, updated_at = ?
			WHERE activity_order_id = ? AND status = ?`
		if _, err := tx.ExecContext(ctx, expTickets, TicketStatusExpired, now, activityOrderID, TicketStatusPending); err != nil {
			return apperr.Internal(err)
		}
		const expAO = `UPDATE activity_orders SET status = 'expired', updated_at = ?
			WHERE id = ? AND status = 'created'`
		if _, err := tx.ExecContext(ctx, expAO, now, activityOrderID); err != nil {
			return apperr.Internal(err)
		}
		const expPO = `UPDATE payment_orders SET status = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		if _, err := tx.ExecContext(ctx, expPO, PaymentStatusExpired, now, paymentID, PaymentStatusPending); err != nil {
			return apperr.Internal(err)
		}
		const closeBO = `UPDATE business_orders SET order_status = ?, updated_at = ?
			WHERE id = ? AND payment_status = 'unpaid'`
		if _, err := tx.ExecContext(ctx, closeBO, businessStatusExpired, now, businessID); err != nil {
			return apperr.Internal(err)
		}
		expired = 1
		return nil
	})
	return expired, err
}

// ExpireTickets expires paid-but-unused tickets whose event has ended. The
// deadline is the session end, falling back to the activity end; a ticket with
// neither set never expires here. status='active' keeps the sweep idempotent and
// away from pending (unpaid, owned by activity-order:expire) and terminal states.
func (r *sqlRepository) ExpireTickets(ctx context.Context, now time.Time) (int64, error) {
	const q = `UPDATE tickets t
		LEFT JOIN activity_sessions s ON s.id = t.session_id
		JOIN activities a ON a.id = t.activity_id
		SET t.status = ?, t.updated_at = ?
		WHERE t.status = ?
		  AND COALESCE(s.end_at, a.end_at) IS NOT NULL
		  AND COALESCE(s.end_at, a.end_at) < ?`
	result, err := r.db.ExecContext(ctx, q, TicketStatusExpired, now, TicketStatusActive, now)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}
