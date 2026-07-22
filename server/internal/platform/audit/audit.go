// Package audit records every back-office write to audit_logs with actor, role,
// store scope, target, before/after diff and request ID. Handlers build an Entry
// from the request context and pass it to Recorder.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Entry is a single audit record.
type Entry struct {
	ActorType  string
	ActorID    int64
	ActorRole  string
	StoreID    int64 // 0 for global actions
	Action     string
	TargetType string
	TargetID   int64
	Before     any
	After      any
	Reason     string
	RequestID  string
}

// Recorder writes audit entries. It can write standalone or inside a tx.
type Recorder struct {
	db *platdb.DB
}

// NewRecorder builds an audit Recorder.
func NewRecorder(db *platdb.DB) *Recorder { return &Recorder{db: db} }

// FromContext seeds an Entry with actor identity, role, store scope and request
// ID pulled from the authenticated request.
func FromContext(c *gin.Context, action, targetType string, targetID int64) Entry {
	e := Entry{
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		RequestID:  httpx.RequestIDFromContext(c),
	}
	if claims, ok := authn.FromContext(c); ok {
		e.ActorType = string(claims.SubjectType)
		e.ActorID = claims.SubjectID()
		e.ActorRole = string(claims.Role)
		e.StoreID = claims.StoreID
	}
	return e
}

// Record persists an entry outside a transaction.
func (r *Recorder) Record(ctx context.Context, e Entry) error {
	return insert(ctx, r.db, e)
}

// RecordTx persists an entry using an existing transaction so the audit row
// commits atomically with the change it describes.
func RecordTx(ctx context.Context, tx *sql.Tx, e Entry) error {
	return insert(ctx, tx, e)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insert(ctx context.Context, ex execer, e Entry) error {
	before, err := marshalNullable(e.Before)
	if err != nil {
		return err
	}
	after, err := marshalNullable(e.After)
	if err != nil {
		return err
	}
	const q = `INSERT INTO audit_logs
		(actor_type, actor_id, actor_role, store_id, action, target_type, target_id,
		 before_json, after_json, reason, request_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	var storeID any
	if e.StoreID > 0 {
		storeID = e.StoreID
	}
	_, err = ex.ExecContext(ctx, q,
		e.ActorType, e.ActorID, e.ActorRole, storeID, e.Action, e.TargetType, e.TargetID,
		before, after, e.Reason, e.RequestID, time.Now().UTC())
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func marshalNullable(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return string(raw), nil
}
