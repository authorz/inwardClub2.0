package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// SQLStore is the MySQL-backed Store. It claims due events with
// FOR UPDATE SKIP LOCKED so multiple worker instances relay disjoint batches
// without double-dispatching, and writes each event's terminal state inside the
// same transaction as the claim.
type SQLStore struct{ db *platdb.DB }

// NewSQLStore builds the MySQL outbox store over the shared pool.
func NewSQLStore(db *platdb.DB) *SQLStore { return &SQLStore{db: db} }

// Dispatch claims up to limit pending events whose available_at has passed,
// hands each to handle, and applies the returned Result. The claim and the
// state writes commit together: if the process dies before commit the rows stay
// pending and are redelivered on a later cycle.
func (s *SQLStore) Dispatch(ctx context.Context, limit int, now time.Time, handle func(Event) Result) (int, error) {
	var processed int
	err := s.db.WithinTx(ctx, func(tx *sql.Tx) error {
		processed = 0 // reset: WithinTx may retry the whole closure on deadlock
		events, err := claim(ctx, tx, limit, now.UTC())
		if err != nil {
			return err
		}
		for _, ev := range events {
			if err := applyResult(ctx, tx, ev.ID, handle(ev), now.UTC()); err != nil {
				return err
			}
			processed++
		}
		return nil
	})
	return processed, err
}

// claim reads and row-locks the next batch of due, pending events. The result
// set is fully drained before returning so the caller can issue the follow-up
// UPDATEs on the same connection.
func claim(ctx context.Context, tx *sql.Tx, limit int, now time.Time) ([]Event, error) {
	const q = `SELECT id, topic, payload, idem_key, attempts, available_at
		FROM outbox_events
		WHERE status = ? AND available_at <= ?
		ORDER BY id
		LIMIT ?
		FOR UPDATE SKIP LOCKED`
	rows, err := tx.QueryContext(ctx, q, StatusPending, now, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var (
			ev      Event
			payload []byte
		)
		if err := rows.Scan(&ev.ID, &ev.Topic, &payload, &ev.IdemKey, &ev.Attempts, &ev.AvailableAt); err != nil {
			return nil, apperr.Internal(err)
		}
		ev.Payload = payload
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return events, nil
}

// applyResult writes the terminal state for one claimed event. attempts is
// always incremented so retries and the max-attempts cutoff are durable.
func applyResult(ctx context.Context, tx *sql.Tx, id int64, res Result, now time.Time) error {
	switch res.Status {
	case StatusDispatched:
		const q = `UPDATE outbox_events
			SET status = ?, dispatched_at = ?, attempts = attempts + 1, last_error = NULL
			WHERE id = ?`
		_, err := tx.ExecContext(ctx, q, StatusDispatched, now, id)
		return wrapExec(err)
	case StatusFailed:
		const q = `UPDATE outbox_events
			SET status = ?, attempts = attempts + 1, last_error = ?
			WHERE id = ?`
		_, err := tx.ExecContext(ctx, q, StatusFailed, res.LastError, id)
		return wrapExec(err)
	case StatusPending:
		const q = `UPDATE outbox_events
			SET attempts = attempts + 1, last_error = ?, available_at = ?
			WHERE id = ?`
		_, err := tx.ExecContext(ctx, q, res.LastError, res.AvailableAt.UTC(), id)
		return wrapExec(err)
	default:
		return apperr.Internal(fmt.Errorf("outbox: unknown result status %q", res.Status))
	}
}

func wrapExec(err error) error {
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}
