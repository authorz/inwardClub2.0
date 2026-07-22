package asset

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the asset persistence port. Services depend on this interface so
// they can be unit-tested with an in-memory fake.
type Repository interface {
	CreatePending(ctx context.Context, a Asset) (int64, error)
	SetObjectKey(ctx context.Context, id int64, objectKey string) error
	GetByID(ctx context.Context, id int64) (Asset, error)
	MarkUploaded(ctx context.Context, id int64, etag string, size int64, uploadedAt time.Time) error
	MarkFailed(ctx context.Context, id int64) error
	ExpirePending(ctx context.Context, createdBefore time.Time) (int64, error)
}

// sqlRepository is the MySQL-backed Repository.
type sqlRepository struct {
	db *platdb.DB
}

// NewRepository builds the MySQL asset repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) CreatePending(ctx context.Context, a Asset) (int64, error) {
	const q = `INSERT INTO assets
		(bucket, object_key, original_filename, content_type, size_bytes, purpose,
		 visibility, status, uploaded_by_type, uploaded_by_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		a.Bucket, a.ObjectKey, a.OriginalFilename, a.ContentType, a.SizeBytes, a.Purpose,
		a.Visibility, StatusPending, a.UploadedByType, a.UploadedByID, time.Now().UTC())
	if err != nil {
		return 0, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func (r *sqlRepository) SetObjectKey(ctx context.Context, id int64, objectKey string) error {
	const q = `UPDATE assets SET object_key = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, objectKey, id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlRepository) GetByID(ctx context.Context, id int64) (Asset, error) {
	const q = `SELECT id, bucket, object_key, COALESCE(etag,''), original_filename,
		content_type, size_bytes, purpose, visibility, status, uploaded_by_type,
		uploaded_by_id, created_at, uploaded_at
		FROM assets WHERE id = ?`
	var a Asset
	var etag string
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.Bucket, &a.ObjectKey, &etag, &a.OriginalFilename,
		&a.ContentType, &a.SizeBytes, &a.Purpose, &a.Visibility, &a.Status,
		&a.UploadedByType, &a.UploadedByID, &a.CreatedAt, &a.UploadedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, apperr.NotFound("asset not found")
	}
	if err != nil {
		return Asset{}, apperr.Internal(err)
	}
	a.Etag = etag
	return a, nil
}

func (r *sqlRepository) MarkUploaded(ctx context.Context, id int64, etag string, size int64, uploadedAt time.Time) error {
	const q = `UPDATE assets SET status = ?, etag = ?, size_bytes = ?, uploaded_at = ?
		WHERE id = ? AND status = ?`
	if _, err := r.db.ExecContext(ctx, q, StatusUploaded, etag, size, uploadedAt, id, StatusPending); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlRepository) MarkFailed(ctx context.Context, id int64) error {
	const q = `UPDATE assets SET status = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, StatusFailed, id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ExpirePending runs the set-based asset:pending-cleanup sweep (Qiniu spec §8,
// task spec §11): assets stuck in 'pending' since before createdBefore never had
// their upload confirmed (the upload token is only valid for minutes) and are
// abandoned to 'failed'. The status='pending' predicate is both the idempotency
// guard (a re-run or doubled schedule tick touches zero rows) and the concurrency
// guard against a late callback marking the same asset uploaded. Returns the
// number of assets abandoned.
func (r *sqlRepository) ExpirePending(ctx context.Context, createdBefore time.Time) (int64, error) {
	const q = `UPDATE assets SET status = ? WHERE status = ? AND created_at < ?`
	result, err := r.db.ExecContext(ctx, q, StatusFailed, StatusPending, createdBefore)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}
