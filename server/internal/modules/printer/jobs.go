package printer

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

const maxPrintJobErrorLength = 512

// PrintJob is one durable receipt-printing attempt shown in the headquarters
// console. Payload content is deliberately not exposed by the API.
type PrintJob struct {
	ID              int64     `json:"id"`
	StoreID         int64     `json:"storeId"`
	StoreName       string    `json:"storeName"`
	DeviceID        *int64    `json:"deviceId,omitempty"`
	DeviceName      string    `json:"deviceName"`
	DeviceSN        string    `json:"deviceSn"`
	Template        string    `json:"template"`
	BusinessOrderNo string    `json:"businessOrderNo"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
	LastError       string    `json:"lastError,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// PrintJobFilter is the headquarters print-log query.
type PrintJobFilter struct {
	Page          httpx.Page
	StoreID       *int64
	Status        string
	Keyword       string
	CreatedFrom   *time.Time
	CreatedBefore *time.Time
}

// JobRecorder updates the durable print job around each worker attempt.
type JobRecorder interface {
	StartAttempt(ctx context.Context, id int64) error
	MarkPrinted(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, cause error) error
}

// JobRepository owns print-job lifecycle writes and the admin read model.
type JobRepository struct{ db *platdb.DB }

func NewJobRepository(db *platdb.DB) *JobRepository { return &JobRepository{db: db} }

func (r *JobRepository) StartAttempt(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE print_jobs
		SET status = 'processing', attempts = attempts + 1, last_error = '', updated_at = NOW()
		WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *JobRepository) MarkPrinted(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE print_jobs
		SET status = 'printed', last_error = '', updated_at = NOW() WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *JobRepository) MarkFailed(ctx context.Context, id int64, cause error) error {
	if id <= 0 {
		return nil
	}
	message := ""
	if cause != nil {
		message = truncatePrintJobError(cause.Error())
	}
	_, err := r.db.ExecContext(ctx, `UPDATE print_jobs
		SET status = 'failed', last_error = ?, updated_at = NOW() WHERE id = ?`, message, id)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *JobRepository) List(ctx context.Context, f PrintJobFilter) ([]PrintJob, int64, error) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 8)
	if f.StoreID != nil {
		clauses = append(clauses, "pj.store_id = ?")
		args = append(args, *f.StoreID)
	}
	if f.Status != "" {
		clauses = append(clauses, "pj.status = ?")
		args = append(args, f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		clauses = append(clauses, `(pj.business_order_no LIKE ? OR s.name LIKE ? OR
			pd.name LIKE ? OR pd.device_sn LIKE ?)`)
		args = append(args, like, like, like, like)
	}
	if f.CreatedFrom != nil {
		clauses = append(clauses, "pj.created_at >= ?")
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedBefore != nil {
		clauses = append(clauses, "pj.created_at < ?")
		args = append(args, f.CreatedBefore.UTC())
	}
	where := strings.Join(clauses, " AND ")
	joins := ` FROM print_jobs pj
		LEFT JOIN stores s ON s.id = pj.store_id
		LEFT JOIN printer_devices pd ON pd.id = pj.device_id`

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+joins+` WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT pj.id, pj.store_id, COALESCE(s.name, ''), pj.device_id,
		COALESCE(pd.name, ''), COALESCE(pd.device_sn,
		JSON_UNQUOTE(JSON_EXTRACT(pj.payload, '$.DeviceSN')), ''),
		pj.template, pj.business_order_no, pj.status, pj.attempts, pj.last_error,
		pj.created_at, pj.updated_at` + joins + ` WHERE ` + where + `
		ORDER BY pj.id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()

	jobs := make([]PrintJob, 0)
	for rows.Next() {
		var job PrintJob
		if err := rows.Scan(&job.ID, &job.StoreID, &job.StoreName, &job.DeviceID,
			&job.DeviceName, &job.DeviceSN, &job.Template, &job.BusinessOrderNo,
			&job.Status, &job.Attempts, &job.LastError, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return jobs, total, nil
}

// createPrintJob persists the log in the same transaction as its outbox event.
func createPrintJob(ctx context.Context, tx *sql.Tx, storeID, deviceID int64, businessOrderNo, idemKey string, job Job) (int64, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO print_jobs
		(store_id, device_id, template, business_order_no, payload, status, idem_key,
		 attempts, last_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?, 0, '', NOW(), NOW())`,
		storeID, deviceID, job.Template, businessOrderNo, payload, idemKey)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func truncatePrintJobError(message string) string {
	runes := []rune(message)
	if len(runes) > maxPrintJobErrorLength {
		return string(runes[:maxPrintJobErrorLength])
	}
	return message
}
