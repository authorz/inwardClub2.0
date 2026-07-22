package reservation

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the reservation/waitlist/arrival persistence port. It also reads
// the store-owned tables/seats tables for availability display.
type Repository interface {
	// Availability reads (mini).
	ListTables(ctx context.Context, storeID int64) ([]Table, error)
	ListSeats(ctx context.Context, storeID int64) ([]Seat, error)

	// Reservation lifecycle.
	ListMemberReservations(ctx context.Context, memberID int64, limit, offset int) ([]Reservation, int64, error)
	ListStoreReservations(ctx context.Context, storeID int64, limit, offset int) ([]Reservation, int64, error)
	CreateReservation(ctx context.Context, r Reservation) (int64, error)
	GetReservation(ctx context.Context, id int64) (Reservation, error)
	CancelReservation(ctx context.Context, id, memberID int64, now time.Time) error

	// Waitlist (mini).
	CreateWaitlistEntry(ctx context.Context, w WaitlistEntry) (int64, error)

	// Arrival (store console): mark a booking arrived and record who did it.
	ArriveReservation(ctx context.Context, reservationID, storeID int64, byType string, byID int64, now time.Time) error

	// ExpireBookings transitions still-booked reservations whose reserved time
	// passed before reservedBefore to expired (the reservation:expire sweep,
	// spec §11). Guarded by status='booked' so it is idempotent and never races a
	// concurrent cancel/arrive. Returns the number of reservations expired.
	ExpireBookings(ctx context.Context, reservedBefore, now time.Time) (int64, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL reservation repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ListTables(ctx context.Context, storeID int64) ([]Table, error) {
	const q = `SELECT id, store_id, name, capacity, status FROM tables
		WHERE store_id = ? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Table
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.ID, &t.StoreID, &t.Name, &t.Capacity, &t.Status); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *sqlRepository) ListSeats(ctx context.Context, storeID int64) ([]Seat, error) {
	const q = `SELECT id, store_id, table_id, name, status FROM seats
		WHERE store_id = ? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Seat
	for rows.Next() {
		var s Seat
		if err := rows.Scan(&s.ID, &s.StoreID, &s.TableID, &s.Name, &s.Status); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const reservationColumns = `id, reservation_no, store_id, member_id, table_id, seat_id,
	party_size, reserved_at, status, remark, created_at, updated_at`

func scanReservation(row interface{ Scan(...any) error }) (Reservation, error) {
	var r Reservation
	err := row.Scan(&r.ID, &r.ReservationNo, &r.StoreID, &r.MemberID, &r.TableID, &r.SeatID,
		&r.PartySize, &r.ReservedAt, &r.Status, &r.Remark, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *sqlRepository) ListMemberReservations(ctx context.Context, memberID int64, limit, offset int) ([]Reservation, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reservations WHERE member_id = ?`, memberID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT ` + reservationColumns + ` FROM reservations
		WHERE member_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Reservation
	for rows.Next() {
		res, err := scanReservation(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, res)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListStoreReservations(ctx context.Context, storeID int64, limit, offset int) ([]Reservation, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reservations WHERE store_id = ?`, storeID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT ` + reservationColumns + ` FROM reservations
		WHERE store_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, storeID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Reservation
	for rows.Next() {
		res, err := scanReservation(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, res)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) CreateReservation(ctx context.Context, res Reservation) (int64, error) {
	const q = `INSERT INTO reservations
		(reservation_no, store_id, member_id, table_id, seat_id, party_size, reserved_at, status, remark, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, res.ReservationNo, res.StoreID, res.MemberID,
		res.TableID, res.SeatID, res.PartySize, res.ReservedAt, res.Status, res.Remark,
		res.CreatedAt, res.UpdatedAt)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return 0, apperr.Conflict("duplicate reservation number")
		}
		return 0, apperr.Internal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func (r *sqlRepository) GetReservation(ctx context.Context, id int64) (Reservation, error) {
	const q = `SELECT ` + reservationColumns + ` FROM reservations WHERE id = ?`
	res, err := scanReservation(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Reservation{}, apperr.NotFound("reservation not found")
	}
	if err != nil {
		return Reservation{}, apperr.Internal(err)
	}
	return res, nil
}

func (r *sqlRepository) CancelReservation(ctx context.Context, id, memberID int64, now time.Time) error {
	// Only the owning member may cancel, and only a still-booked reservation.
	const q = `UPDATE reservations SET status = ?, updated_at = ?
		WHERE id = ? AND member_id = ? AND status = ?`
	result, err := r.db.ExecContext(ctx, q, StatusCancelled, now, id, memberID, StatusBooked)
	if err != nil {
		return apperr.Internal(err)
	}
	return affectedOrConflict(result, "reservation cannot be cancelled")
}

func (r *sqlRepository) CreateWaitlistEntry(ctx context.Context, w WaitlistEntry) (int64, error) {
	const q = `INSERT INTO waitlist_entries
		(store_id, member_id, party_size, status, queued_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, w.StoreID, w.MemberID, w.PartySize, w.Status,
		w.QueuedAt, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func (r *sqlRepository) ArriveReservation(ctx context.Context, reservationID, storeID int64, byType string, byID int64, now time.Time) error {
	// Booking transition and arrival record must be atomic, and the reservation
	// must belong to the acting store's scope.
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		const upd = `UPDATE reservations SET status = ?, updated_at = ?
			WHERE id = ? AND store_id = ? AND status = ?`
		result, err := tx.ExecContext(ctx, upd, StatusArrived, now, reservationID, storeID, StatusBooked)
		if err != nil {
			return apperr.Internal(err)
		}
		if err := affectedOrConflict(result, "reservation cannot be marked arrived"); err != nil {
			return err
		}
		var memberID int64
		if err := tx.QueryRowContext(ctx, `SELECT member_id FROM reservations WHERE id = ?`, reservationID).Scan(&memberID); err != nil {
			return apperr.Internal(err)
		}
		const ins = `INSERT INTO arrival_records
			(store_id, member_id, reservation_id, arrived_at, recorded_by_type, recorded_by_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		if _, err := tx.ExecContext(ctx, ins, storeID, memberID, reservationID, now, byType, byID, now); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

// ExpireBookings runs the set-based reservation:expire sweep. A booking is only
// ever moved out of 'booked' by a cancel, an arrival or this sweep, so the
// status='booked' predicate is both the idempotency guard (a re-run touches zero
// rows) and the concurrency guard against those other transitions.
func (r *sqlRepository) ExpireBookings(ctx context.Context, reservedBefore, now time.Time) (int64, error) {
	const q = `UPDATE reservations SET status = ?, updated_at = ?
		WHERE status = ? AND reserved_at < ?`
	result, err := r.db.ExecContext(ctx, q, StatusExpired, now, StatusBooked, reservedBefore)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

// affectedOrConflict maps a zero-row update to a conflict; the row either does
// not exist or is not in the expected state.
func affectedOrConflict(result sql.Result, msg string) error {
	n, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if n == 0 {
		return apperr.Conflict(msg)
	}
	return nil
}
