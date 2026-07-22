package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Point-saving review decisions accepted by the store console.
const (
	ReviewApprove = "approve"
	ReviewReject  = "reject"
)

// Point-saving lifecycle statuses.
const (
	PointSavingPending  = "pending"
	PointSavingApproved = "approved"
	PointSavingRejected = "rejected"
)

// PointSavingDirection is fixed for this endpoint: every point_savings row is an
// inbound (points-in) request. Withdrawals live in a separate table.
const PointSavingDirection = "in"

// Verification actors and targets.
const (
	verifiedByStaff    = "staff"
	verificationTicket = "ticket"
)

// A stored verification row is always a completed redemption; these are the
// display status/result reported to the store console for such records.
const (
	verificationStatusUsed    = "used"
	verificationResultSuccess = "success"
)

// Ticket lifecycle statuses relevant to store-console verification.
const (
	ticketActive = "active"
	ticketUsed   = "used"
)

// TicketVerification is the outcome of verifying a ticket at a store.
type TicketVerification struct {
	ID         int64
	StoreID    int64
	TicketID   int64
	TicketNo   string
	ActivityID int64
	Status     string
	VerifiedBy int64
	VerifiedAt time.Time
}

// PointSaving is a member request to convert spend into points, reviewed by the
// store console. It always belongs to exactly one store.
type PointSaving struct {
	ID         int64
	StoreID    int64
	MemberID   int64
	MemberName string
	Phone      string
	StoreName  string
	Points     int64
	Status     string
	Remark     string
	ReviewedBy *int64
	ReviewedAt *time.Time
	CreatedAt  time.Time
}

// Verification is a historical verification record (tickets, coupons, etc.).
type Verification struct {
	ID            int64
	StoreID       int64
	Kind          string
	RefID         int64
	Code          string
	ActivityTitle string
	MemberName    string
	VerifiedBy    int64
	VerifiedAt    time.Time
}

// TodayActivity is a store's activity running today, enriched with the owning
// store name and its sold-vs-verified ticket counts for the staff overview.
type TodayActivity struct {
	ID            int64
	Title         string
	StoreName     string
	StartAt       *time.Time
	EndAt         *time.Time
	PendingVerify int64
	Verified      int64
}

// StaffHomeData is the aggregated store-console staff landing summary, sourced
// from stores + point_savings + verifications + activities for one store.
type StaffHomeData struct {
	StoreName          string
	PendingReview      int64
	TodayVerifications int64
	TodayActivity      *Activity
}

// StoreRepository is the store-console persistence port. Every method is scoped
// by storeID so a console can never read or write another store's rows.
type StoreRepository interface {
	VerifyTicket(ctx context.Context, storeID int64, code string, byID int64, now time.Time) (TicketVerification, error)
	ListPointSavings(ctx context.Context, storeID int64, limit, offset int) ([]PointSaving, int64, error)
	GetPointSaving(ctx context.Context, storeID, requestID int64) (PointSaving, error)
	ReviewPointSaving(ctx context.Context, storeID, requestID int64, decision, remark string, byID int64, now time.Time) (PointSaving, error)
	ListTodayActivities(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]Activity, error)
	ListTodayActivitySummaries(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]TodayActivity, error)
	ListVerifications(ctx context.Context, storeID int64, limit, offset int) ([]Verification, int64, error)
	StaffHome(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) (StaffHomeData, error)
}

type storeSQLRepository struct{ db *platdb.DB }

// NewStoreRepository builds the MySQL store-console repository.
//
// All methods run real queries scoped by storeID: point-saving reads/reviews,
// today's activities, the verification listing (joined to activity + member for
// display) and ticket verification (transactional, single-use).
func NewStoreRepository(db *platdb.DB) StoreRepository { return &storeSQLRepository{db: db} }

// VerifyTicket redeems a ticket by its verification code at the acting store.
// It is single-use: the ticket must currently be 'active' (an already-'used'
// ticket yields a CONFLICT), and the whole redemption runs in one transaction
// so the ticket flip and the verification record commit atomically. The unique
// (target_type, target_id) index on verifications is the last line of defence
// against a concurrent double verify.
func (r *storeSQLRepository) VerifyTicket(ctx context.Context, storeID int64, code string, byID int64, now time.Time) (TicketVerification, error) {
	var v TicketVerification
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			ticketID, activityID, memberID int64
			ticketNo, status               string
		)
		const sel = `SELECT id, ticket_no, activity_id, member_id, status
			FROM tickets WHERE verification_code = ? AND store_id = ? FOR UPDATE`
		switch err := tx.QueryRowContext(ctx, sel, code, storeID).
			Scan(&ticketID, &ticketNo, &activityID, &memberID, &status); {
		case errors.Is(err, sql.ErrNoRows):
			return apperr.NotFound("ticket not found")
		case err != nil:
			return apperr.Internal(err)
		}
		if status == ticketUsed {
			return apperr.Conflict("ticket already verified")
		}
		if status != ticketActive {
			return apperr.Conflict("ticket is not active")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?`,
			ticketUsed, now, ticketID); err != nil {
			return apperr.Internal(err)
		}
		const ins = `INSERT INTO verifications
			(verification_no, target_type, target_id, store_id, verified_by_type, verified_by_id, member_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		verNo := fmt.Sprintf("V%d-%d", ticketID, now.UnixNano())
		res, err := tx.ExecContext(ctx, ins, verNo, verificationTicket, ticketID, storeID, verifiedByStaff, byID, memberID, now)
		if err != nil {
			return apperr.Internal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		v = TicketVerification{
			ID: id, StoreID: storeID, TicketID: ticketID, TicketNo: ticketNo,
			ActivityID: activityID, Status: ticketUsed, VerifiedBy: byID, VerifiedAt: now,
		}
		return nil
	})
	if err != nil {
		return TicketVerification{}, err
	}
	return v, nil
}

// pointSavingSelect reads point_savings joined to members (name/phone) and
// stores (name) for display; COALESCE guards nullable joins and columns.
const pointSavingSelect = `SELECT ps.id, ps.store_id, ps.member_id,
	COALESCE(m.nickname,''), COALESCE(m.phone,''), COALESCE(s.name,''),
	ps.points, ps.status, COALESCE(ps.remark,''), ps.reviewed_by, ps.reviewed_at, ps.created_at
	FROM point_savings ps
	LEFT JOIN members m ON m.id = ps.member_id
	LEFT JOIN stores s ON s.id = ps.store_id`

func scanPointSaving(row interface{ Scan(...any) error }) (PointSaving, error) {
	var p PointSaving
	if err := row.Scan(&p.ID, &p.StoreID, &p.MemberID, &p.MemberName, &p.Phone, &p.StoreName,
		&p.Points, &p.Status, &p.Remark, &p.ReviewedBy, &p.ReviewedAt, &p.CreatedAt); err != nil {
		return PointSaving{}, err
	}
	return p, nil
}

func (r *storeSQLRepository) ListPointSavings(ctx context.Context, storeID int64, limit, offset int) ([]PointSaving, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM point_savings WHERE store_id = ?`, storeID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := pointSavingSelect + ` WHERE ps.store_id = ? ORDER BY ps.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, storeID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []PointSaving
	for rows.Next() {
		p, err := scanPointSaving(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *storeSQLRepository) GetPointSaving(ctx context.Context, storeID, requestID int64) (PointSaving, error) {
	q := pointSavingSelect + ` WHERE ps.id = ? AND ps.store_id = ?`
	p, err := scanPointSaving(r.db.QueryRowContext(ctx, q, requestID, storeID))
	if errors.Is(err, sql.ErrNoRows) {
		return PointSaving{}, apperr.NotFound("point-saving request not found")
	}
	if err != nil {
		return PointSaving{}, apperr.Internal(err)
	}
	return p, nil
}

func (r *storeSQLRepository) ReviewPointSaving(ctx context.Context, storeID, requestID int64, decision, remark string, byID int64, now time.Time) (PointSaving, error) {
	status := PointSavingApproved
	if decision == ReviewReject {
		status = PointSavingRejected
	}
	const q = `UPDATE point_savings SET status = ?, remark = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ?
		WHERE id = ? AND store_id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q, status, remark, byID, now, now, requestID, storeID, PointSavingPending)
	if err != nil {
		return PointSaving{}, apperr.Internal(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return PointSaving{}, apperr.Internal(err)
	}
	if affected == 0 {
		// Distinguish "does not exist for this store" from "already reviewed".
		if _, err := r.GetPointSaving(ctx, storeID, requestID); err != nil {
			return PointSaving{}, err
		}
		return PointSaving{}, apperr.Conflict("point-saving request is not pending review")
	}
	return r.GetPointSaving(ctx, storeID, requestID)
}

func (r *storeSQLRepository) ListTodayActivities(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]Activity, error) {
	q := `SELECT ` + activityColumns + ` FROM activities
		WHERE store_id = ? AND status = 'published'
			AND (start_at IS NULL OR start_at < ?)
			AND (end_at IS NULL OR end_at >= ?)
		ORDER BY start_at ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID, dayEnd, dayStart)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListTodayActivitySummaries returns today's published activities for the store
// with the owning store name and ticket counts: pendingVerify = tickets still
// 'active' (sold, not yet redeemed), verified = tickets already 'used'. Tickets
// are joined and aggregated per activity within the store scope.
func (r *storeSQLRepository) ListTodayActivitySummaries(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]TodayActivity, error) {
	const q = `SELECT a.id, a.title, COALESCE(s.name,''), a.start_at, a.end_at,
			COALESCE(SUM(t.status = 'active'), 0), COALESCE(SUM(t.status = 'used'), 0)
		FROM activities a
		LEFT JOIN stores s ON s.id = a.store_id
		LEFT JOIN tickets t ON t.activity_id = a.id AND t.store_id = a.store_id
		WHERE a.store_id = ? AND a.status = 'published'
			AND (a.start_at IS NULL OR a.start_at < ?)
			AND (a.end_at IS NULL OR a.end_at >= ?)
		GROUP BY a.id, a.title, s.name, a.start_at, a.end_at
		ORDER BY a.start_at ASC, a.id ASC`
	rows, err := r.db.QueryContext(ctx, q, storeID, dayEnd, dayStart)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []TodayActivity
	for rows.Next() {
		var a TodayActivity
		if err := rows.Scan(&a.ID, &a.Title, &a.StoreName, &a.StartAt, &a.EndAt,
			&a.PendingVerify, &a.Verified); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *storeSQLRepository) ListVerifications(ctx context.Context, storeID int64, limit, offset int) ([]Verification, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM verifications WHERE store_id = ?`, storeID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	// Join the ticket→activity chain for the activity title, and members for the
	// member name. Joins are ticket-specific; non-ticket targets simply leave the
	// activity title empty.
	q := `SELECT v.id, v.store_id, v.target_type, v.target_id,
			COALESCE(t.verification_code, v.verification_no),
			COALESCE(a.title,''), COALESCE(m.nickname,''),
			v.verified_by_id, v.created_at
		FROM verifications v
		LEFT JOIN tickets t ON v.target_type = 'ticket' AND t.id = v.target_id
		LEFT JOIN activities a ON a.id = t.activity_id
		LEFT JOIN members m ON m.id = v.member_id
		WHERE v.store_id = ? ORDER BY v.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, storeID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Verification
	for rows.Next() {
		var v Verification
		if err := rows.Scan(&v.ID, &v.StoreID, &v.Kind, &v.RefID, &v.Code,
			&v.ActivityTitle, &v.MemberName, &v.VerifiedBy, &v.VerifiedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *storeSQLRepository) StaffHome(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) (StaffHomeData, error) {
	var data StaffHomeData
	if err := r.db.QueryRowContext(ctx, `SELECT name FROM stores WHERE id = ?`, storeID).Scan(&data.StoreName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StaffHomeData{}, apperr.NotFound("store not found")
		}
		return StaffHomeData{}, apperr.Internal(err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM point_savings WHERE store_id = ? AND status = ?`,
		storeID, PointSavingPending).Scan(&data.PendingReview); err != nil {
		return StaffHomeData{}, apperr.Internal(err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM verifications WHERE store_id = ? AND created_at >= ? AND created_at < ?`,
		storeID, dayStart, dayEnd).Scan(&data.TodayVerifications); err != nil {
		return StaffHomeData{}, apperr.Internal(err)
	}
	acts, err := r.ListTodayActivities(ctx, storeID, dayStart, dayEnd)
	if err != nil {
		return StaffHomeData{}, err
	}
	if len(acts) > 0 {
		data.TodayActivity = &acts[0]
	}
	return data, nil
}

// StoreService provides the store-console operational activity operations. Every
// method takes the store scope explicitly; callers must source it from the token
// (never from client input) via storescope.
type StoreService struct {
	repo   StoreRepository
	assets AssetResolver
	now    func() time.Time
}

// NewStoreService builds the store-console activity service.
func NewStoreService(repo StoreRepository, assets AssetResolver) *StoreService {
	return &StoreService{repo: repo, assets: assets, now: time.Now}
}

// VerifyTicket redeems a ticket by its code at the acting store. byID identifies
// the staff account performing the verification.
func (s *StoreService) VerifyTicket(ctx context.Context, storeID int64, code string, byID int64) (TicketVerificationView, error) {
	if code == "" {
		return TicketVerificationView{}, apperr.Invalid("ticket code is required")
	}
	v, err := s.repo.VerifyTicket(ctx, storeID, code, byID, s.now().UTC())
	if err != nil {
		return TicketVerificationView{}, err
	}
	return ticketVerificationView(v), nil
}

// ListPointSavings returns the store's point-saving requests, newest first.
func (s *StoreService) ListPointSavings(ctx context.Context, storeID int64, page httpx.Page) ([]PointSavingView, int64, error) {
	items, total, err := s.repo.ListPointSavings(ctx, storeID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]PointSavingView, 0, len(items))
	for _, p := range items {
		views = append(views, pointSavingView(p))
	}
	return views, total, nil
}

// GetPointSaving returns one point-saving request owned by the store.
func (s *StoreService) GetPointSaving(ctx context.Context, storeID, requestID int64) (PointSavingView, error) {
	p, err := s.repo.GetPointSaving(ctx, storeID, requestID)
	if err != nil {
		return PointSavingView{}, err
	}
	return pointSavingView(p), nil
}

// ReviewPointSaving approves or rejects a store's point-saving request.
func (s *StoreService) ReviewPointSaving(ctx context.Context, storeID, requestID int64, req ReviewPointSavingRequest, byID int64) (PointSavingView, error) {
	if req.Decision != ReviewApprove && req.Decision != ReviewReject {
		return PointSavingView{}, apperr.Invalid("decision must be approve or reject")
	}
	p, err := s.repo.ReviewPointSaving(ctx, storeID, requestID, req.Decision, req.Remark, byID, s.now().UTC())
	if err != nil {
		return PointSavingView{}, err
	}
	return pointSavingView(p), nil
}

// TodayActivities returns the store's activities running today (store local day
// approximated in UTC).
func (s *StoreService) TodayActivities(ctx context.Context, storeID int64) ([]TodayActivityView, error) {
	now := s.now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	acts, err := s.repo.ListTodayActivitySummaries(ctx, storeID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	views := make([]TodayActivityView, 0, len(acts))
	for _, a := range acts {
		views = append(views, todayActivityView(a))
	}
	return views, nil
}

// ListVerifications returns the store's verification history, newest first.
func (s *StoreService) ListVerifications(ctx context.Context, storeID int64, page httpx.Page) ([]VerificationView, int64, error) {
	items, total, err := s.repo.ListVerifications(ctx, storeID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]VerificationView, 0, len(items))
	for _, v := range items {
		views = append(views, verificationView(v))
	}
	return views, total, nil
}

// StaffHome returns the store-console staff landing summary: the bound store's
// name, the count of point-saving requests still awaiting review, the number of
// verifications performed today, and today's featured activity (nil if none).
func (s *StoreService) StaffHome(ctx context.Context, storeID int64) (StaffHomeView, error) {
	now := s.now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	data, err := s.repo.StaffHome(ctx, storeID, dayStart, dayEnd)
	if err != nil {
		return StaffHomeView{}, err
	}
	view := StaffHomeView{
		Store:              StoreRefView{Name: data.StoreName},
		PendingReview:      data.PendingReview,
		TodayVerifications: data.TodayVerifications,
	}
	if data.TodayActivity != nil {
		av := s.activityView(ctx, *data.TodayActivity)
		view.TodayActivity = &av
	}
	return view, nil
}

func (s *StoreService) activityView(ctx context.Context, a Activity) ActivityView {
	v := ActivityView{
		ID: a.ID, StoreID: a.StoreID, Title: a.Title, Description: a.Description,
		Content: a.Content, StartAt: a.StartAt, EndAt: a.EndAt, PayChannels: a.PayChannels, Status: a.Status,
	}
	if a.AssetID != nil {
		v.ImageURL, _ = s.assets.PublicURLByID(ctx, *a.AssetID)
	}
	return v
}
