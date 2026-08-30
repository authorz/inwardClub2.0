package diagnostics

import (
	"context"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the error-events persistence port. The feed is append-only apart
// from Prune, which enforces the retention cap.
type Repository interface {
	// Insert appends one captured error event.
	Insert(ctx context.Context, e ErrorEvent) error
	// List returns a page of events, newest first, plus the total row count.
	List(ctx context.Context, requestID string, limit, offset int) ([]ErrorEvent, int64, error)
	// Prune deletes all but the newest keep events, bounding table growth.
	Prune(ctx context.Context, keep int) error
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL error-events repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

const errorEventColumns = `id, request_id, method, path, status, message, created_at`

func (r *sqlRepository) Insert(ctx context.Context, e ErrorEvent) error {
	const q = `INSERT INTO error_events
		(request_id, method, path, status, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, q,
		e.RequestID, e.Method, e.Path, e.Status, e.Message, e.CreatedAt); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlRepository) List(ctx context.Context, requestID string, limit, offset int) ([]ErrorEvent, int64, error) {
	where := ""
	args := make([]any, 0, 3)
	if requestID != "" {
		where = ` WHERE request_id LIKE ?`
		args = append(args, "%"+requestID+"%")
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM error_events`+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ` + errorEventColumns + ` FROM error_events` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]ErrorEvent, 0)
	for rows.Next() {
		var e ErrorEvent
		if err := rows.Scan(&e.ID, &e.RequestID, &e.Method, &e.Path, &e.Status, &e.Message, &e.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// Prune keeps only the newest keep events. It finds the id at offset keep in the
// newest-first ordering and deletes every row at or below it; with keep or fewer
// rows the subquery yields NULL and nothing is deleted. The derived-table wrapper
// is required because MySQL forbids referencing the delete target in a direct
// subquery.
func (r *sqlRepository) Prune(ctx context.Context, keep int) error {
	if keep < 0 {
		keep = 0
	}
	const q = `DELETE FROM error_events WHERE id <= (
		SELECT threshold_id FROM (
			SELECT id AS threshold_id FROM error_events ORDER BY id DESC LIMIT 1 OFFSET ?
		) AS t
	)`
	if _, err := r.db.ExecContext(ctx, q, keep); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
