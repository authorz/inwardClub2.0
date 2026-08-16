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
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
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
	actorSnapshot, targetSnapshot, scopeSnapshot, err := loadSnapshots(ctx, ex, e)
	if err != nil {
		return err
	}
	const q = `INSERT INTO audit_logs
		(actor_type, actor_id, actor_role, actor_snapshot_json, store_id, scope_snapshot_json,
		 action, target_type, target_id, target_snapshot_json, before_json, after_json,
		 reason, request_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	var storeID any
	if e.StoreID > 0 {
		storeID = e.StoreID
	}
	_, err = ex.ExecContext(ctx, q,
		e.ActorType, e.ActorID, e.ActorRole, actorSnapshot, storeID, scopeSnapshot,
		e.Action, e.TargetType, e.TargetID, targetSnapshot, before, after,
		e.Reason, e.RequestID, time.Now().UTC())
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

type actorSnapshot struct {
	Type        string `json:"type"`
	ID          int64  `json:"id"`
	Role        string `json:"role,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Name        string `json:"name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

type targetSnapshot struct {
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname,omitempty"`
	Phone     string `json:"phone,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

type scopeSnapshot struct {
	StoreID   int64  `json:"storeId"`
	StoreName string `json:"storeName,omitempty"`
}

func loadSnapshots(ctx context.Context, ex execer, e Entry) (any, any, any, error) {
	actor := actorSnapshot{Type: e.ActorType, ID: e.ActorID, Role: e.ActorRole}
	if e.ActorID > 0 {
		switch e.ActorType {
		case "super_admin", "store_admin", "cashier":
			err := ex.QueryRowContext(ctx,
				`SELECT username, display_name FROM admin_accounts WHERE id = ?`, e.ActorID,
			).Scan(&actor.Username, &actor.DisplayName)
			if err != nil && err != sql.ErrNoRows {
				return nil, nil, nil, apperr.Internal(err)
			}
		case "staff":
			query := `SELECT sa.name, COALESCE(m.phone,''), COALESCE(m.avatar_url,'')
				FROM staff_accounts sa JOIN members m ON m.id = sa.member_id
				WHERE sa.member_id = ?`
			args := []any{e.ActorID}
			if e.StoreID > 0 {
				query += ` AND sa.store_id = ?`
				args = append(args, e.StoreID)
			}
			query += ` LIMIT 1`
			err := ex.QueryRowContext(ctx, query, args...).Scan(&actor.Name, &actor.Phone, &actor.AvatarURL)
			if err != nil && err != sql.ErrNoRows {
				return nil, nil, nil, apperr.Internal(err)
			}
		case "member", "pre_member":
			err := ex.QueryRowContext(ctx,
				`SELECT nickname, COALESCE(phone,''), COALESCE(avatar_url,'') FROM members WHERE id = ?`, e.ActorID,
			).Scan(&actor.DisplayName, &actor.Phone, &actor.AvatarURL)
			if err != nil && err != sql.ErrNoRows {
				return nil, nil, nil, apperr.Internal(err)
			}
		}
	}

	target := targetSnapshot{Type: e.TargetType, ID: e.TargetID}
	if e.TargetType == "member" && e.TargetID > 0 {
		err := ex.QueryRowContext(ctx,
			`SELECT nickname, COALESCE(phone,''), COALESCE(avatar_url,'') FROM members WHERE id = ?`, e.TargetID,
		).Scan(&target.Nickname, &target.Phone, &target.AvatarURL)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, nil, apperr.Internal(err)
		}
	}

	var scope any
	if e.StoreID > 0 {
		store := scopeSnapshot{StoreID: e.StoreID}
		err := ex.QueryRowContext(ctx, `SELECT name FROM stores WHERE id = ?`, e.StoreID).Scan(&store.StoreName)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, nil, apperr.Internal(err)
		}
		scope = store
	}

	actorJSON, err := marshalNullable(actor)
	if err != nil {
		return nil, nil, nil, err
	}
	targetJSON, err := marshalNullable(target)
	if err != nil {
		return nil, nil, nil, err
	}
	scopeJSON, err := marshalNullable(scope)
	if err != nil {
		return nil, nil, nil, err
	}
	return actorJSON, targetJSON, scopeJSON, nil
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
