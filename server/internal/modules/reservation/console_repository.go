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

// ConsoleRepository owns headquarters table/seat management persistence.
type ConsoleRepository interface {
	StoreExists(ctx context.Context, storeID int64) (bool, error)
	ListAdminTables(ctx context.Context, filter AdminTableFilter, limit, offset int) ([]Table, int64, error)
	GetAdminTable(ctx context.Context, id int64) (Table, error)
	CreateAdminTable(ctx context.Context, table Table) (Table, error)
	UpdateAdminTable(ctx context.Context, id int64, table Table) (Table, error)
	DeleteAdminTable(ctx context.Context, id int64) error
	ListAdminSeats(ctx context.Context, filter AdminSeatFilter, limit, offset int) ([]Seat, int64, error)
	GetAdminSeat(ctx context.Context, id int64) (Seat, error)
	CreateAdminSeat(ctx context.Context, seat Seat) (Seat, error)
	UpdateAdminSeat(ctx context.Context, id int64, seat Seat) (Seat, error)
	DeleteAdminSeat(ctx context.Context, id int64) error
}

type sqlConsoleRepository struct{ db *platdb.DB }

func NewConsoleRepository(db *platdb.DB) ConsoleRepository {
	return &sqlConsoleRepository{db: db}
}

func (r *sqlConsoleRepository) StoreExists(ctx context.Context, storeID int64) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id = ?)`, storeID).Scan(&exists); err != nil {
		return false, apperr.Internal(err)
	}
	return exists, nil
}

const adminTableColumns = `t.id, t.store_id, s.name, t.name, t.code, t.capacity,
	t.base_points, t.status, COALESCE(sc.seat_count, 0), t.created_at, t.updated_at`

func tableFilterWhere(filter AdminTableFilter) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 4)
	if filter.StoreID != nil {
		where = append(where, "t.store_id = ?")
		args = append(args, *filter.StoreID)
	}
	if filter.Status != "" {
		where = append(where, "t.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where = append(where, "(t.name LIKE ? OR t.code LIKE ?)")
		like := "%" + filter.Keyword + "%"
		args = append(args, like, like)
	}
	return strings.Join(where, " AND "), args
}

func scanAdminTable(row interface{ Scan(...any) error }) (Table, error) {
	var table Table
	err := row.Scan(
		&table.ID, &table.StoreID, &table.StoreName, &table.Name, &table.Code,
		&table.Capacity, &table.BasePoints, &table.Status, &table.SeatCount,
		&table.CreatedAt, &table.UpdatedAt,
	)
	return table, err
}

func (r *sqlConsoleRepository) ListAdminTables(ctx context.Context, filter AdminTableFilter, limit, offset int) ([]Table, int64, error) {
	where, args := tableFilterWhere(filter)
	var total int64
	countQuery := `SELECT COUNT(*) FROM tables t WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	query := `SELECT ` + adminTableColumns + ` FROM tables t
		JOIN stores s ON s.id = t.store_id
		LEFT JOIN (SELECT table_id, COUNT(*) seat_count FROM seats WHERE table_id IS NOT NULL GROUP BY table_id) sc
			ON sc.table_id = t.id
		WHERE ` + where + ` ORDER BY t.id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	items := make([]Table, 0)
	for rows.Next() {
		table, err := scanAdminTable(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		items = append(items, table)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return items, total, nil
}

func (r *sqlConsoleRepository) GetAdminTable(ctx context.Context, id int64) (Table, error) {
	query := `SELECT ` + adminTableColumns + ` FROM tables t
		JOIN stores s ON s.id = t.store_id
		LEFT JOIN (SELECT table_id, COUNT(*) seat_count FROM seats WHERE table_id IS NOT NULL GROUP BY table_id) sc
			ON sc.table_id = t.id WHERE t.id = ?`
	table, err := scanAdminTable(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Table{}, apperr.NotFound("table not found")
	}
	if err != nil {
		return Table{}, apperr.Internal(err)
	}
	return table, nil
}

func (r *sqlConsoleRepository) CreateAdminTable(ctx context.Context, table Table) (Table, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `INSERT INTO tables
		(store_id, name, code, capacity, base_points, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		table.StoreID, table.Name, table.Code, table.Capacity, table.BasePoints,
		table.Status, now, now,
	)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return Table{}, apperr.Conflict("table code already exists in this store")
		}
		return Table{}, apperr.Internal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Table{}, apperr.Internal(err)
	}
	return r.GetAdminTable(ctx, id)
}

func (r *sqlConsoleRepository) UpdateAdminTable(ctx context.Context, id int64, table Table) (Table, error) {
	current, err := r.GetAdminTable(ctx, id)
	if err != nil {
		return Table{}, err
	}
	now := time.Now().UTC()
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE tables
			SET store_id = ?, name = ?, code = ?, capacity = ?, base_points = ?, status = ?, updated_at = ?
			WHERE id = ?`,
			table.StoreID, table.Name, table.Code, table.Capacity, table.BasePoints,
			table.Status, now, id,
		); err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("table code already exists in this store")
			}
			return apperr.Internal(err)
		}
		if current.StoreID == table.StoreID {
			return nil
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE seats SET store_id = ?, updated_at = ? WHERE table_id = ?`,
			table.StoreID, now, id,
		); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE reservations SET store_id = ?, updated_at = ? WHERE table_id = ? AND status = ?`,
			table.StoreID, now, id, StatusBooked,
		); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
	if err != nil {
		return Table{}, err
	}
	return r.GetAdminTable(ctx, id)
}

func (r *sqlConsoleRepository) DeleteAdminTable(ctx context.Context, id int64) error {
	var seatCount, reservationCount int64
	if err := r.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM seats WHERE table_id = ?),
		(SELECT COUNT(*) FROM reservations WHERE table_id = ?)`, id, id).Scan(&seatCount, &reservationCount); err != nil {
		return apperr.Internal(err)
	}
	if seatCount > 0 || reservationCount > 0 {
		return apperr.Conflict("table with seats or reservations cannot be deleted")
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM tables WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return affectedOrNotFound(result, "table not found")
}

const adminSeatColumns = `se.id, se.store_id, st.name, se.table_id, t.name,
	se.name, se.status, se.created_at, se.updated_at`

func seatFilterWhere(filter AdminSeatFilter) (string, []any) {
	where := []string{"se.table_id IS NOT NULL"}
	args := make([]any, 0, 4)
	if filter.StoreID != nil {
		where = append(where, "se.store_id = ?")
		args = append(args, *filter.StoreID)
	}
	if filter.TableID != nil {
		where = append(where, "se.table_id = ?")
		args = append(args, *filter.TableID)
	}
	if filter.Status != "" {
		where = append(where, "se.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where = append(where, "se.name LIKE ?")
		args = append(args, "%"+filter.Keyword+"%")
	}
	return strings.Join(where, " AND "), args
}

func scanAdminSeat(row interface{ Scan(...any) error }) (Seat, error) {
	var seat Seat
	err := row.Scan(
		&seat.ID, &seat.StoreID, &seat.StoreName, &seat.TableID, &seat.TableName,
		&seat.Name, &seat.Status, &seat.CreatedAt, &seat.UpdatedAt,
	)
	return seat, err
}

func (r *sqlConsoleRepository) ListAdminSeats(ctx context.Context, filter AdminSeatFilter, limit, offset int) ([]Seat, int64, error) {
	where, args := seatFilterWhere(filter)
	var total int64
	countQuery := `SELECT COUNT(*) FROM seats se WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	query := `SELECT ` + adminSeatColumns + ` FROM seats se
		JOIN tables t ON t.id = se.table_id
		JOIN stores st ON st.id = se.store_id
		WHERE ` + where + ` ORDER BY se.id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	items := make([]Seat, 0)
	for rows.Next() {
		seat, err := scanAdminSeat(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		items = append(items, seat)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return items, total, nil
}

func (r *sqlConsoleRepository) GetAdminSeat(ctx context.Context, id int64) (Seat, error) {
	query := `SELECT ` + adminSeatColumns + ` FROM seats se
		JOIN tables t ON t.id = se.table_id
		JOIN stores st ON st.id = se.store_id
		WHERE se.id = ? AND se.table_id IS NOT NULL`
	seat, err := scanAdminSeat(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Seat{}, apperr.NotFound("seat not found")
	}
	if err != nil {
		return Seat{}, apperr.Internal(err)
	}
	return seat, nil
}

func (r *sqlConsoleRepository) CreateAdminSeat(ctx context.Context, seat Seat) (Seat, error) {
	if seat.TableID == nil {
		return Seat{}, apperr.Invalid("tableId is required")
	}
	var id int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var storeID int64
		var capacity, seatCount int
		if err := tx.QueryRowContext(ctx, `SELECT store_id, capacity,
			(SELECT COUNT(*) FROM seats WHERE table_id = tables.id)
			FROM tables WHERE id = ? FOR UPDATE`, *seat.TableID).Scan(&storeID, &capacity, &seatCount); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.NotFound("table not found")
			}
			return apperr.Internal(err)
		}
		if seatCount >= capacity {
			return apperr.Conflict("table seat capacity has been reached")
		}
		now := time.Now().UTC()
		result, err := tx.ExecContext(ctx, `INSERT INTO seats
			(store_id, table_id, name, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			storeID, *seat.TableID, seat.Name, seat.Status, now, now,
		)
		if err != nil {
			return apperr.Internal(err)
		}
		id, err = result.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
	if err != nil {
		return Seat{}, err
	}
	return r.GetAdminSeat(ctx, id)
}

func (r *sqlConsoleRepository) UpdateAdminSeat(ctx context.Context, id int64, seat Seat) (Seat, error) {
	if seat.TableID == nil {
		return Seat{}, apperr.Invalid("tableId is required")
	}
	current, err := r.GetAdminSeat(ctx, id)
	if err != nil {
		return Seat{}, err
	}
	table, err := r.GetAdminTable(ctx, *seat.TableID)
	if err != nil {
		return Seat{}, err
	}
	if current.TableID == nil || *current.TableID != *seat.TableID {
		if table.SeatCount >= table.Capacity {
			return Seat{}, apperr.Conflict("table seat capacity has been reached")
		}
	}
	result, err := r.db.ExecContext(ctx, `UPDATE seats
		SET store_id = ?, table_id = ?, name = ?, status = ?, updated_at = ?
		WHERE id = ?`,
		table.StoreID, *seat.TableID, seat.Name, seat.Status, time.Now().UTC(), id,
	)
	if err != nil {
		return Seat{}, apperr.Internal(err)
	}
	if err := affectedOrNotFound(result, "seat not found"); err != nil {
		return Seat{}, err
	}
	return r.GetAdminSeat(ctx, id)
}

func (r *sqlConsoleRepository) DeleteAdminSeat(ctx context.Context, id int64) error {
	var reservationCount int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reservations WHERE seat_id = ?`, id).Scan(&reservationCount); err != nil {
		return apperr.Internal(err)
	}
	if reservationCount > 0 {
		return apperr.Conflict("seat with reservations cannot be deleted")
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM seats WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return affectedOrNotFound(result, "seat not found")
}

func affectedOrNotFound(result sql.Result, message string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if affected == 0 {
		return apperr.NotFound(message)
	}
	return nil
}
