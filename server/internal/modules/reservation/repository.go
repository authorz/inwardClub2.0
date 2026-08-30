package reservation

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the reservation/waitlist/arrival persistence port. It also reads
// the store-owned tables/seats tables for availability display.
type Repository interface {
	// Availability reads (mini).
	ListTables(ctx context.Context, storeID int64) ([]Table, error)
	ListSeats(ctx context.Context, storeID int64, activeSince time.Time) ([]Seat, error)

	// Reservation lifecycle.
	ListMemberReservations(ctx context.Context, memberID int64, limit, offset int) ([]Reservation, int64, error)
	ListStoreReservations(ctx context.Context, storeID int64, filter StoreReservationFilter, limit, offset int) ([]Reservation, int64, error)
	HasMemberReservation(ctx context.Context, memberID int64, createdFrom, createdBefore time.Time) (bool, error)
	CreateReservation(ctx context.Context, r Reservation, dailyStart, dailyEnd time.Time) (int64, error)
	GetReservation(ctx context.Context, id int64) (Reservation, error)
	CancelReservation(ctx context.Context, id, memberID int64, now time.Time) error
	CancelStoreReservation(ctx context.Context, id, storeID int64, now time.Time) error

	// Waitlist (mini).
	ListWaitingMembers(ctx context.Context, storeID int64, queuedFrom time.Time, limit int) ([]WaitlistEntry, error)
	CreateWaitlistEntry(ctx context.Context, w WaitlistEntry, dailyStart, dailyEnd time.Time) (int64, error)

	// Arrival (store console): mark a booking arrived and record who did it.
	ArriveReservation(ctx context.Context, reservationID, storeID int64, byType string, byID int64, now time.Time) error

	// ExpireBookings deletes table-only reservations after their no-show grace.
	ExpireBookings(ctx context.Context, reservedBefore, now time.Time) (int64, error)
	ClearSeatBookings(ctx context.Context, createdBefore, now time.Time) (int64, error)
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

func (r *sqlRepository) ListSeats(ctx context.Context, storeID int64, activeSince time.Time) ([]Seat, error) {
	const q = `SELECT s.id, s.store_id, s.table_id, s.name,
		CASE WHEN r.id IS NULL THEN s.status ELSE ? END,
		CASE WHEN COALESCE(r.booked_as_guest, 0) = 1 THEN 'inward会员' ELSE COALESCE(m.nickname, '') END,
		CASE WHEN COALESCE(r.booked_as_guest, 0) = 1 THEN '' ELSE COALESCE(m.avatar_url, '') END,
		CASE WHEN COALESCE(r.booked_as_guest, 0) = 1 THEN '' ELSE COALESCE(m.gender, '') END,
		COALESCE(r.booked_as_guest, 0)
		FROM seats s
		LEFT JOIN reservations r ON r.id = (
			SELECT r2.id FROM reservations r2
			WHERE r2.seat_id = s.id AND r2.status IN (?, ?) AND r2.created_at >= ?
			ORDER BY r2.created_at DESC, r2.id DESC LIMIT 1
		)
		LEFT JOIN members m ON m.id = r.member_id
		WHERE s.store_id = ? ORDER BY s.id ASC`
	rows, err := r.db.QueryContext(
		ctx, q, AvailabilityReserved, StatusBooked, StatusArrived, activeSince.UTC(), storeID,
	)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Seat
	for rows.Next() {
		var s Seat
		if err := rows.Scan(
			&s.ID, &s.StoreID, &s.TableID, &s.Name, &s.Status,
			&s.MemberNickname, &s.MemberAvatarURL, &s.MemberGender, &s.BookedAsGuest,
		); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const reservationColumns = `id, reservation_no, store_id, member_id, booked_as_guest, table_id, seat_id,
	party_size, reserved_at, status, remark, created_at, updated_at`

func scanReservation(row interface{ Scan(...any) error }) (Reservation, error) {
	var r Reservation
	err := row.Scan(&r.ID, &r.ReservationNo, &r.StoreID, &r.MemberID, &r.BookedAsGuest, &r.TableID, &r.SeatID,
		&r.PartySize, &r.ReservedAt, &r.Status, &r.Remark, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *sqlRepository) ListMemberReservations(ctx context.Context, memberID int64, limit, offset int) ([]Reservation, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reservations WHERE member_id = ? AND status IN (?, ?)`, memberID, StatusBooked, StatusArrived).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT ` + reservationColumns + ` FROM reservations
		WHERE member_id = ? AND status IN (?, ?) ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, StatusBooked, StatusArrived, limit, offset)
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

func (r *sqlRepository) ListStoreReservations(ctx context.Context, storeID int64, filter StoreReservationFilter, limit, offset int) ([]Reservation, int64, error) {
	joins := ` FROM reservations r
		LEFT JOIN members m ON m.id = r.member_id
		LEFT JOIN tables t ON t.id = r.table_id
		LEFT JOIN seats s ON s.id = r.seat_id`
	where := []string{"r.store_id = ?", "r.status IN (?, ?)"}
	args := []any{storeID, StatusBooked, StatusArrived}
	addLike := func(condition, value string, copies int) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		where = append(where, condition)
		pattern := "%" + value + "%"
		for range copies {
			args = append(args, pattern)
		}
	}
	addLike("(t.code LIKE ? OR t.name LIKE ?)", filter.TableNo, 2)
	addLike("s.name LIKE ?", filter.SeatNo, 1)
	addLike("m.nickname LIKE ?", filter.MemberNickname, 1)
	addLike("m.phone LIKE ?", filter.MemberPhone, 1)
	whereSQL := " WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+joins+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT r.id, r.reservation_no, r.store_id, r.member_id, r.booked_as_guest, r.table_id, r.seat_id,
		r.party_size, r.reserved_at, r.status, r.remark, r.created_at, r.updated_at,
		COALESCE(m.nickname, ''), COALESCE(m.phone, ''), m.avatar_asset_id, COALESCE(m.avatar_url, ''),
		COALESCE(NULLIF(t.code, ''), t.name, ''), COALESCE(s.name, '')` + joins + whereSQL + ` ORDER BY r.id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, q, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Reservation
	for rows.Next() {
		var res Reservation
		if err := rows.Scan(
			&res.ID, &res.ReservationNo, &res.StoreID, &res.MemberID, &res.BookedAsGuest, &res.TableID, &res.SeatID,
			&res.PartySize, &res.ReservedAt, &res.Status, &res.Remark, &res.CreatedAt, &res.UpdatedAt,
			&res.MemberNickname, &res.MemberPhone, &res.MemberAvatarAssetID, &res.MemberAvatarURL, &res.TableNo, &res.SeatNo,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, res)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) HasMemberReservation(
	ctx context.Context,
	memberID int64,
	createdFrom, createdBefore time.Time,
) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM reservation_daily_claims WHERE member_id = ? AND daily_start = ?
			UNION ALL
			SELECT 1 FROM reservations WHERE member_id = ? AND created_at >= ? AND created_at < ?
		)`,
		memberID, createdFrom.UTC(), memberID, createdFrom.UTC(), createdBefore.UTC(),
	).Scan(&exists)
	if err != nil {
		return false, apperr.Internal(err)
	}
	return exists, nil
}

func (r *sqlRepository) CreateReservation(
	ctx context.Context,
	res Reservation,
	dailyStart, dailyEnd time.Time,
) (int64, error) {
	var id int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var lockedMemberID int64
		queryErr := tx.QueryRowContext(ctx,
			`SELECT id FROM members WHERE id = ? FOR UPDATE`,
			res.MemberID,
		).Scan(&lockedMemberID)
		if errors.Is(queryErr, sql.ErrNoRows) {
			return apperr.NotFound("member not found")
		}
		if queryErr != nil {
			return apperr.Internal(queryErr)
		}

		var alreadyReserved bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM reservations
				WHERE member_id = ? AND created_at >= ? AND created_at < ?
			)`,
			res.MemberID, dailyStart.UTC(), dailyEnd.UTC(),
		).Scan(&alreadyReserved); err != nil {
			return apperr.Internal(err)
		}
		if alreadyReserved {
			return apperr.Conflict("你今天已经预约座位了")
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO reservation_daily_claims (member_id, daily_start, created_at) VALUES (?, ?, ?)`,
			res.MemberID, dailyStart.UTC(), res.CreatedAt,
		); err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("你今天已经预约座位了")
			}
			return apperr.Internal(err)
		}

		var tableStoreID int64
		var tableStatus string
		queryErr = tx.QueryRowContext(ctx,
			`SELECT store_id, status FROM tables WHERE id = ? FOR UPDATE`,
			*res.TableID,
		).Scan(&tableStoreID, &tableStatus)
		if errors.Is(queryErr, sql.ErrNoRows) {
			return apperr.NotFound("table not found")
		}
		if queryErr != nil {
			return apperr.Internal(queryErr)
		}
		if tableStoreID != res.StoreID {
			return apperr.Invalid("table does not belong to store")
		}
		if tableStatus != AvailabilityAvailable {
			return apperr.Conflict("该桌暂不可预约")
		}

		if res.SeatID == nil {
			var seatID int64
			queryErr = tx.QueryRowContext(ctx,
				`SELECT s.id
				 FROM seats s
				 WHERE s.store_id = ? AND s.table_id = ? AND s.status = ?
				   AND NOT EXISTS (
				     SELECT 1 FROM reservations r
				     WHERE r.seat_id = s.id AND r.status IN (?, ?)
				       AND r.created_at >= ? AND r.created_at < ?
				   )
				 ORDER BY s.id ASC
				 LIMIT 1 FOR UPDATE`,
				res.StoreID, *res.TableID, AvailabilityAvailable,
				StatusBooked, StatusArrived, dailyStart.UTC(), dailyEnd.UTC(),
			).Scan(&seatID)
			if errors.Is(queryErr, sql.ErrNoRows) {
				return apperr.Conflict("该桌暂时没有空位")
			}
			if queryErr != nil {
				return apperr.Internal(queryErr)
			}
			res.SeatID = &seatID
		}

		var seatStoreID int64
		var seatTableID *int64
		var seatStatus string
		queryErr = tx.QueryRowContext(ctx,
			`SELECT store_id, table_id, status FROM seats WHERE id = ? FOR UPDATE`,
			*res.SeatID,
		).Scan(&seatStoreID, &seatTableID, &seatStatus)
		if errors.Is(queryErr, sql.ErrNoRows) {
			return apperr.NotFound("seat not found")
		}
		if queryErr != nil {
			return apperr.Internal(queryErr)
		}
		if seatStoreID != res.StoreID {
			return apperr.Invalid("seat does not belong to store")
		}
		if res.TableID != nil && (seatTableID == nil || *seatTableID != *res.TableID) {
			return apperr.Invalid("seat does not belong to table")
		}
		if seatStatus != AvailabilityAvailable {
			return apperr.Conflict("seat is not available")
		}

		var occupied bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM reservations
				WHERE seat_id = ? AND status IN (?, ?)
				  AND created_at >= ? AND created_at < ?
			)`,
			*res.SeatID, StatusBooked, StatusArrived, dailyStart.UTC(), dailyEnd.UTC(),
		).Scan(&occupied); err != nil {
			return apperr.Internal(err)
		}
		if occupied {
			return apperr.Conflict("seat is already reserved")
		}

		var insertErr error
		id, insertErr = insertReservation(ctx, tx, res)
		if insertErr != nil {
			return insertErr
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE waitlist_entries SET status = ?, updated_at = ?
			 WHERE store_id = ? AND member_id = ? AND status IN (?, ?)
			   AND queued_at >= ? AND queued_at < ?`,
			WaitlistLeft, res.UpdatedAt, res.StoreID, res.MemberID,
			WaitlistWaiting, WaitlistCalled, dailyStart.UTC(), dailyEnd.UTC(),
		); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
	return id, err
}

type reservationExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertReservation(ctx context.Context, exec reservationExecutor, res Reservation) (int64, error) {
	const q = `INSERT INTO reservations
		(reservation_no, store_id, member_id, booked_as_guest, table_id, seat_id, party_size, reserved_at, status, remark, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := exec.ExecContext(ctx, q, res.ReservationNo, res.StoreID, res.MemberID,
		res.BookedAsGuest, res.TableID, res.SeatID, res.PartySize, res.ReservedAt, res.Status, res.Remark,
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
	const q = `SELECT ` + reservationColumns + ` FROM reservations WHERE id = ? AND status IN (?, ?)`
	res, err := scanReservation(r.db.QueryRowContext(ctx, q, id, StatusBooked, StatusArrived))
	if errors.Is(err, sql.ErrNoRows) {
		return Reservation{}, apperr.NotFound("reservation not found")
	}
	if err != nil {
		return Reservation{}, apperr.Internal(err)
	}
	return res, nil
}

func (r *sqlRepository) CancelReservation(ctx context.Context, id, memberID int64, dailyStart time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var ownerID int64
		var status string
		if err := tx.QueryRowContext(ctx,
			`SELECT member_id, status FROM reservations WHERE id = ? FOR UPDATE`, id,
		).Scan(&ownerID, &status); errors.Is(err, sql.ErrNoRows) {
			// Cancellation is idempotent: a retry after the first request already
			// deleted the booking is still a successful cancellation.
			return nil
		} else if err != nil {
			return apperr.Internal(err)
		}
		if ownerID != memberID {
			return apperr.NotFound("reservation not found")
		}
		if status != StatusBooked {
			return apperr.Conflict("已到店的预约不能取消")
		}
		const q = `DELETE FROM reservations WHERE id = ? AND member_id = ? AND status = ?`
		result, err := tx.ExecContext(ctx, q, id, memberID, StatusBooked)
		if err != nil {
			return apperr.Internal(err)
		}
		if err := affectedOrConflict(result, "reservation cannot be cancelled"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM reservation_daily_claims WHERE member_id = ? AND daily_start = ?`,
			memberID, dailyStart.UTC(),
		); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

func (r *sqlRepository) CancelStoreReservation(ctx context.Context, id, storeID int64, dailyStart time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var memberID int64
		if err := tx.QueryRowContext(ctx,
			`SELECT member_id FROM reservations WHERE id = ? AND store_id = ? AND status = ? FOR UPDATE`,
			id, storeID, StatusBooked,
		).Scan(&memberID); errors.Is(err, sql.ErrNoRows) {
			return apperr.Conflict("reservation cannot be cancelled")
		} else if err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM reservations WHERE id = ?`, id); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM reservation_daily_claims WHERE member_id = ? AND daily_start = ?`,
			memberID, dailyStart.UTC(),
		); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

func (r *sqlRepository) ListWaitingMembers(ctx context.Context, storeID int64, queuedFrom time.Time, limit int) ([]WaitlistEntry, error) {
	const q = `SELECT w.id, w.store_id, w.member_id, m.avatar_asset_id, COALESCE(m.avatar_url, ''),
		w.party_size, w.status, w.queued_at, w.created_at, w.updated_at
		FROM waitlist_entries w
		JOIN members m ON m.id = w.member_id
		WHERE w.store_id = ? AND w.status = ? AND w.queued_at >= ?
		  AND NOT EXISTS (
			SELECT 1 FROM waitlist_entries earlier
			WHERE earlier.store_id = w.store_id AND earlier.member_id = w.member_id
			  AND earlier.status = ? AND earlier.queued_at >= ?
			  AND (earlier.queued_at < w.queued_at OR (earlier.queued_at = w.queued_at AND earlier.id < w.id))
		  )
		ORDER BY w.queued_at ASC, w.id ASC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, storeID, WaitlistWaiting, queuedFrom.UTC(), WaitlistWaiting, queuedFrom.UTC(), limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	entries := make([]WaitlistEntry, 0)
	for rows.Next() {
		var entry WaitlistEntry
		if err := rows.Scan(
			&entry.ID, &entry.StoreID, &entry.MemberID, &entry.MemberAvatarAssetID, &entry.MemberAvatarURL,
			&entry.PartySize, &entry.Status, &entry.QueuedAt, &entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, apperr.Internal(err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return entries, nil
}

func (r *sqlRepository) CreateWaitlistEntry(ctx context.Context, w WaitlistEntry, dailyStart, dailyEnd time.Time) (int64, error) {
	var id int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var lockedMemberID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM members WHERE id = ? FOR UPDATE`, w.MemberID).Scan(&lockedMemberID); errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("member not found")
		} else if err != nil {
			return apperr.Internal(err)
		}
		var hasReservation bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM reservation_daily_claims WHERE member_id = ? AND daily_start = ?
			UNION ALL
			SELECT 1 FROM reservations WHERE member_id = ? AND created_at >= ? AND created_at < ?
		)`, w.MemberID, dailyStart.UTC(), w.MemberID, dailyStart.UTC(), dailyEnd.UTC()).Scan(&hasReservation); err != nil {
			return apperr.Internal(err)
		}
		if hasReservation {
			return apperr.Conflict("你已经预约座位了，如需排队请先取消预约")
		}
		const q = `INSERT INTO waitlist_entries
			(store_id, member_id, party_size, status, queued_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		result, err := tx.ExecContext(ctx, q, w.StoreID, w.MemberID, w.PartySize, w.Status,
			w.QueuedAt, w.CreatedAt, w.UpdatedAt)
		if err != nil {
			return apperr.Internal(err)
		}
		id, err = result.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
	return id, err
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

// ExpireBookings removes table-only no-shows; expired is not a visible state.
func (r *sqlRepository) ExpireBookings(ctx context.Context, reservedBefore, _ time.Time) (int64, error) {
	const q = `DELETE FROM reservations WHERE status = ? AND seat_id IS NULL AND reserved_at < ?`
	result, err := r.db.ExecContext(ctx, q, StatusBooked, reservedBefore)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

// ClearSeatBookings releases seat reservations created before the latest
// business-day 04:00 boundary. The cutoff makes startup catch-up safe: bookings
// created after the boundary remain occupied until the next reset.
func (r *sqlRepository) ClearSeatBookings(ctx context.Context, createdBefore, _ time.Time) (int64, error) {
	const q = `DELETE FROM reservations WHERE status IN (?, ?) AND seat_id IS NOT NULL AND created_at < ?`
	result, err := r.db.ExecContext(ctx, q, StatusBooked, StatusArrived, createdBefore)
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
