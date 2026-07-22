// Package outbox implements the transactional outbox. External side effects
// (WeChat calls, printing, Qiniu deletes, notifications, rule evaluation) are
// written as events inside the business transaction and dispatched to asynq
// only after the transaction commits, so effects never fire on a rolled-back write.
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Status values for an outbox row.
const (
	StatusPending    = "pending"
	StatusDispatched = "dispatched"
	StatusFailed     = "failed"
)

// Event is a durable intent to perform an external side effect.
type Event struct {
	ID          int64
	Topic       string
	Payload     json.RawMessage
	IdemKey     string
	Attempts    int
	AvailableAt time.Time
}

// Write appends an event within the caller's transaction. topic maps to an asynq
// task type; idemKey is the per-effect business idempotency key.
func Write(ctx context.Context, tx *sql.Tx, topic string, payload any, idemKey string) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return apperr.Internal(err)
	}
	const q = `INSERT INTO outbox_events
		(topic, payload, idem_key, status, available_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, q, topic, raw, idemKey, StatusPending, now, now)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}
