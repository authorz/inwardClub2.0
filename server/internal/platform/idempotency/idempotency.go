// Package idempotency provides the Idempotency-Key middleware and the
// transactional claim used as the final duplicate guard. Payment, wallet,
// redemption, verification, check-in and manual adjustment endpoints require the
// header; the unique index on idempotency_keys is the ultimate protection.
package idempotency

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

const headerName = "Idempotency-Key"

// Require enforces the presence of an Idempotency-Key header and stores it in
// the request context for the service layer.
func Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader(headerName))
		if key == "" {
			httpx.Fail(c, apperr.New(apperr.CodeIdempotencyRequired, "Idempotency-Key header is required"))
			return
		}
		if _, err := inputvalidation.OpaqueToken("幂等请求标识", key, 128); err != nil {
			httpx.Fail(c, apperr.Invalid(err.Error()))
			return
		}
		c.Set(httpx.CtxIdemKey, key)
		c.Next()
	}
}

// Optional captures the Idempotency-Key header when present without requiring it.
func Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		if key := strings.TrimSpace(c.GetHeader(headerName)); key != "" {
			if _, err := inputvalidation.OpaqueToken("幂等请求标识", key, 128); err != nil {
				httpx.Fail(c, apperr.Invalid(err.Error()))
				return
			}
			c.Set(httpx.CtxIdemKey, key)
		}
		c.Next()
	}
}

// Key returns the request's idempotency key, or "" if none was supplied.
func Key(c *gin.Context) string {
	if v, ok := c.Get(httpx.CtxIdemKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Store persists claimed idempotency keys so a repeated request is rejected as a
// conflict rather than re-executing a side effect.
type Store struct {
	db *platdb.DB
}

// NewStore builds an idempotency Store.
func NewStore(db *platdb.DB) *Store { return &Store{db: db} }

// Claim inserts a key within the caller's transaction. A duplicate returns
// CodeIdempotencyConflict. scope namespaces the key (e.g. "mini/point-savings").
func Claim(ctx context.Context, tx *sql.Tx, scope, key, targetType string, targetID int64) error {
	const q = `INSERT INTO idempotency_keys
		(idem_key, scope, target_type, target_id, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q, key, scope, targetType, targetID, time.Now().UTC())
	if err != nil {
		if platdb.IsDuplicate(err) {
			return apperr.New(apperr.CodeIdempotencyConflict, "duplicate request")
		}
		return apperr.Internal(err)
	}
	return nil
}
