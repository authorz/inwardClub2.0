package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// ConsoleScope pins a console request to a single store, or leaves it nil for
// the admin (HQ) console which sees every scope.
type ConsoleScope struct {
	StoreID *int64
}

// Session is an activity session (a scheduled slot within an activity).
type Session struct {
	ID         int64
	ActivityID int64
	StoreID    *int64
	Name       string
	StartAt    time.Time
	EndAt      time.Time
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TicketType is a sellable ticket tier for an activity (optionally scoped to a
// single session).
type TicketType struct {
	ID                 int64
	ActivityID         int64
	SessionID          *int64
	StoreID            *int64
	Name               string
	PriceCent          int64
	StockQuantity      int64
	SoldQuantity       int64
	SaleStartAt        *time.Time
	SaleEndAt          *time.Time
	PayChannels        []string
	MaxTicketsPerOrder int
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ConsoleActivityView is the console representation of an activity. It is
// distinct from ActivityView (the mini-program public read view) because the
// console additionally exposes scope, status and audit timestamps.
type ConsoleActivityView struct {
	ID                     int64      `json:"id"`
	ScopeType              string     `json:"scopeType"`
	StoreID                *int64     `json:"storeId,omitempty"`
	Title                  string     `json:"title"`
	Description            string     `json:"description,omitempty"`
	Content                string     `json:"content,omitempty"`
	AssetID                *int64     `json:"assetId,omitempty"`
	ImageURL               string     `json:"imageUrl,omitempty"`
	StartAt                *time.Time `json:"startAt,omitempty"`
	EndAt                  *time.Time `json:"endAt,omitempty"`
	PayChannels            []string   `json:"payChannels"`
	PurchaseLimitPerMember int        `json:"purchaseLimitPerMember"`
	Status                 string     `json:"status"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

// SessionView is the console representation of an activity session.
type SessionView struct {
	ID         int64     `json:"id"`
	ActivityID int64     `json:"activityId"`
	StoreID    *int64    `json:"storeId,omitempty"`
	Name       string    `json:"name"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// TicketTypeView is the console representation of a ticket type.
type TicketTypeView struct {
	ID                 int64      `json:"id"`
	ActivityID         int64      `json:"activityId"`
	SessionID          *int64     `json:"sessionId,omitempty"`
	StoreID            *int64     `json:"storeId,omitempty"`
	Name               string     `json:"name"`
	PriceCent          int64      `json:"priceCent"`
	StockQuantity      int64      `json:"stockQuantity"`
	SoldQuantity       int64      `json:"soldQuantity"`
	SaleStartAt        *time.Time `json:"saleStartAt,omitempty"`
	SaleEndAt          *time.Time `json:"saleEndAt,omitempty"`
	PayChannels        []string   `json:"payChannels"`
	MaxTicketsPerOrder int        `json:"maxTicketsPerOrder"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// ActivityInput is the console create/update payload for an activity.
type ActivityInput struct {
	StoreID                *int64     `json:"storeId"`
	Title                  string     `json:"title" binding:"required"`
	Description            string     `json:"description"`
	Content                string     `json:"content"`
	AssetID                *int64     `json:"assetId"`
	StartAt                *time.Time `json:"startAt"`
	EndAt                  *time.Time `json:"endAt"`
	PayChannels            []string   `json:"payChannels"`
	PurchaseLimitPerMember int        `json:"purchaseLimitPerMember"`
	Status                 string     `json:"status"`
}

// SessionInput is the console create/update payload for an activity session.
type SessionInput struct {
	Name    string    `json:"name" binding:"required"`
	StartAt time.Time `json:"startAt" binding:"required"`
	EndAt   time.Time `json:"endAt" binding:"required"`
	Status  string    `json:"status"`
}

// TicketTypeInput is the console create/update payload for a ticket type.
type TicketTypeInput struct {
	SessionID          *int64     `json:"sessionId"`
	Name               string     `json:"name" binding:"required"`
	PriceCent          int64      `json:"priceCent" binding:"required,min=0"`
	StockQuantity      int64      `json:"stockQuantity"`
	SaleStartAt        *time.Time `json:"saleStartAt"`
	SaleEndAt          *time.Time `json:"saleEndAt"`
	PayChannels        []string   `json:"payChannels"`
	MaxTicketsPerOrder int        `json:"maxTicketsPerOrder"`
	Status             string     `json:"status"`
}

// ConsoleRepository is the console persistence port for activities, sessions
// and ticket types. Admin (scope.StoreID == nil) sees every scope; a store
// console (scope.StoreID set) is restricted to scope_type = 'store' rows for
// that store.
type ConsoleRepository interface {
	ListActivities(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]Activity, int64, error)
	GetActivity(ctx context.Context, scope ConsoleScope, id int64) (Activity, error)
	CreateActivity(ctx context.Context, scope ConsoleScope, in ActivityInput) (Activity, error)
	UpdateActivity(ctx context.Context, scope ConsoleScope, id int64, in ActivityInput) (Activity, error)
	DeleteActivity(ctx context.Context, scope ConsoleScope, id int64) error

	ListSessions(ctx context.Context, activityID int64) ([]Session, error)
	GetSession(ctx context.Context, activityID, sessionID int64) (Session, error)
	CreateSession(ctx context.Context, activityID int64, in SessionInput) (Session, error)
	UpdateSession(ctx context.Context, activityID, sessionID int64, in SessionInput) (Session, error)
	DeleteSession(ctx context.Context, activityID, sessionID int64) error

	ListTicketTypes(ctx context.Context, activityID int64, sessionID *int64) ([]TicketType, error)
	GetTicketType(ctx context.Context, activityID, ticketTypeID int64) (TicketType, error)
	CreateTicketType(ctx context.Context, activityID int64, in TicketTypeInput) (TicketType, error)
	UpdateTicketType(ctx context.Context, activityID, ticketTypeID int64, in TicketTypeInput) (TicketType, error)
	DeleteTicketType(ctx context.Context, activityID, ticketTypeID int64) error
}

type sqlConsoleRepository struct{ db *platdb.DB }

// NewConsoleRepository builds the MySQL console repository for activities.
func NewConsoleRepository(db *platdb.DB) ConsoleRepository { return &sqlConsoleRepository{db: db} }

const consoleActivityColumns = `id, scope_type, store_id, title, COALESCE(description,''),
	COALESCE(content,''), asset_id, start_at, end_at, pay_channels,
	purchase_limit_per_member, status, created_at, updated_at`

func scanConsoleActivity(row interface{ Scan(...any) error }) (Activity, error) {
	var a Activity
	var payChannels []byte
	err := row.Scan(&a.ID, &a.ScopeType, &a.StoreID, &a.Title, &a.Description,
		&a.Content, &a.AssetID, &a.StartAt, &a.EndAt, &payChannels, &a.PurchaseLimit,
		&a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Activity{}, err
	}
	a.PayChannels = decodeChannels(payChannels)
	return a, nil
}

// scopeWhere builds the WHERE clause + args restricting activities to the
// given console scope. Admin (scope.StoreID == nil) has no restriction.
func scopeWhere(scope ConsoleScope) (string, []any) {
	if scope.StoreID == nil {
		return "1 = 1", nil
	}
	return "scope_type = 'store' AND store_id = ?", []any{*scope.StoreID}
}

func (r *sqlConsoleRepository) ListActivities(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]Activity, int64, error) {
	where, args := scopeWhere(scope)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activities WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ` + consoleActivityColumns + ` FROM activities WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	qArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, q, qArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		a, err := scanConsoleActivity(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlConsoleRepository) GetActivity(ctx context.Context, scope ConsoleScope, id int64) (Activity, error) {
	where, args := scopeWhere(scope)
	q := `SELECT ` + consoleActivityColumns + ` FROM activities WHERE id = ? AND ` + where
	a, err := scanConsoleActivity(r.db.QueryRowContext(ctx, q, append([]any{id}, args...)...))
	if errors.Is(err, sql.ErrNoRows) {
		return Activity{}, apperr.NotFound("activity not found")
	}
	if err != nil {
		return Activity{}, apperr.Internal(err)
	}
	return a, nil
}

// encodeChannels marshals the pay-channel slice for the JSON column, coercing a
// nil slice to an empty JSON array so the NOT NULL column always gets a value.
func encodeChannels(ch []string) []byte {
	if ch == nil {
		ch = []string{}
	}
	b, _ := json.Marshal(ch)
	return b
}

// scopeWrite resolves the scope_type/store_id a write should carry. A store
// console is always pinned to its JWT store; the admin console may explicitly
// bind the activity to a store or leave storeId empty for a global activity.
func scopeWrite(scope ConsoleScope, requestedStoreID *int64) (string, any) {
	if scope.StoreID != nil {
		return "store", *scope.StoreID
	}
	if requestedStoreID != nil {
		return "store", *requestedStoreID
	}
	return "global", nil
}

func (r *sqlConsoleRepository) CreateActivity(ctx context.Context, scope ConsoleScope, in ActivityInput) (Activity, error) {
	now := time.Now().UTC()
	scopeType, storeID := scopeWrite(scope, in.StoreID)
	status := in.Status
	if status == "" {
		status = "published"
	}
	if scope.StoreID == nil && in.StoreID != nil {
		if err := validateActivityStore(ctx, r.db, *in.StoreID); err != nil {
			return Activity{}, err
		}
	}
	const q = `INSERT INTO activities
		(scope_type, store_id, title, description, content, asset_id, start_at, end_at,
		 pay_channels, purchase_limit_per_member, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, scopeType, storeID, in.Title, in.Description, in.Content,
		in.AssetID, in.StartAt, in.EndAt, encodeChannels(in.PayChannels), in.PurchaseLimitPerMember,
		status, now, now)
	if err != nil {
		return Activity{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Activity{}, apperr.Internal(err)
	}
	return r.GetActivity(ctx, scope, id)
}

func (r *sqlConsoleRepository) UpdateActivity(ctx context.Context, scope ConsoleScope, id int64, in ActivityInput) (Activity, error) {
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "published"
	}
	scopeType, storeID := scopeWrite(scope, in.StoreID)
	where, args := scopeWhere(scope)
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if scope.StoreID == nil && in.StoreID != nil {
			if err := validateActivityStore(ctx, tx, *in.StoreID); err != nil {
				return err
			}
		}
		var existingID int64
		lockArgs := append([]any{id}, args...)
		if err := tx.QueryRowContext(
			ctx,
			`SELECT id FROM activities WHERE id=? AND `+where+` FOR UPDATE`,
			lockArgs...,
		).Scan(&existingID); errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("activity not found")
		} else if err != nil {
			return apperr.Internal(err)
		}
		q := `UPDATE activities SET scope_type=?, store_id=?, title=?, description=?, content=?,
			asset_id=?, start_at=?, end_at=?, pay_channels=?, purchase_limit_per_member=?,
			status=?, updated_at=? WHERE id=? AND ` + where
		qArgs := append([]any{scopeType, storeID, in.Title, in.Description, in.Content, in.AssetID,
			in.StartAt, in.EndAt, encodeChannels(in.PayChannels), in.PurchaseLimitPerMember,
			status, now, id}, args...)
		if _, err := tx.ExecContext(ctx, q, qArgs...); err != nil {
			return apperr.Internal(err)
		}
		for _, q := range []string{
			`UPDATE activity_sessions SET store_id=?, updated_at=? WHERE activity_id=?`,
			`UPDATE activity_ticket_types SET store_id=?, updated_at=? WHERE activity_id=?`,
		} {
			if _, err := tx.ExecContext(ctx, q, storeID, now, id); err != nil {
				return apperr.Internal(err)
			}
		}
		return nil
	})
	if err != nil {
		return Activity{}, err
	}
	return r.GetActivity(ctx, scope, id)
}

type activityStoreQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func validateActivityStore(ctx context.Context, q activityStoreQuerier, storeID int64) error {
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id=?)`, storeID).Scan(&exists); err != nil {
		return apperr.Internal(err)
	}
	if !exists {
		return apperr.Invalid("storeId does not reference an existing store")
	}
	return nil
}

func (r *sqlConsoleRepository) DeleteActivity(ctx context.Context, scope ConsoleScope, id int64) error {
	where, args := scopeWhere(scope)
	q := `DELETE FROM activities WHERE id=? AND ` + where
	res, err := r.db.ExecContext(ctx, q, append([]any{id}, args...)...)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("activity not found")
	}
	return nil
}

const sessionColumns = `id, activity_id, store_id, name, start_at, end_at, status, created_at, updated_at`

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.ActivityID, &s.StoreID, &s.Name, &s.StartAt, &s.EndAt, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return Session{}, err
	}
	return s, nil
}

func (r *sqlConsoleRepository) ListSessions(ctx context.Context, activityID int64) ([]Session, error) {
	q := `SELECT ` + sessionColumns + ` FROM activity_sessions WHERE activity_id = ? ORDER BY start_at ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, activityID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *sqlConsoleRepository) GetSession(ctx context.Context, activityID, sessionID int64) (Session, error) {
	q := `SELECT ` + sessionColumns + ` FROM activity_sessions WHERE activity_id = ? AND id = ?`
	s, err := scanSession(r.db.QueryRowContext(ctx, q, activityID, sessionID))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, apperr.NotFound("activity session not found")
	}
	if err != nil {
		return Session{}, apperr.Internal(err)
	}
	return s, nil
}

func (r *sqlConsoleRepository) CreateSession(ctx context.Context, activityID int64, in SessionInput) (Session, error) {
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "active"
	}
	// store_id is inherited from the owning activity so a session can never
	// escape its activity's scope.
	const q = `INSERT INTO activity_sessions
		(activity_id, store_id, name, start_at, end_at, status, created_at, updated_at)
		VALUES (?, (SELECT store_id FROM activities WHERE id = ?), ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, activityID, activityID, in.Name, in.StartAt, in.EndAt, status, now, now)
	if err != nil {
		return Session{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Session{}, apperr.Internal(err)
	}
	return r.GetSession(ctx, activityID, id)
}

func (r *sqlConsoleRepository) UpdateSession(ctx context.Context, activityID, sessionID int64, in SessionInput) (Session, error) {
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "active"
	}
	const q = `UPDATE activity_sessions SET name=?, start_at=?, end_at=?, status=?, updated_at=?
		WHERE id=? AND activity_id=?`
	if _, err := r.db.ExecContext(ctx, q, in.Name, in.StartAt, in.EndAt, status, now, sessionID, activityID); err != nil {
		return Session{}, apperr.Internal(err)
	}
	return r.GetSession(ctx, activityID, sessionID)
}

func (r *sqlConsoleRepository) DeleteSession(ctx context.Context, activityID, sessionID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM activity_sessions WHERE id=? AND activity_id=?`, sessionID, activityID)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("activity session not found")
	}
	return nil
}

const ticketTypeColumns = `id, activity_id, session_id, store_id, name, price_cent, stock_quantity,
	sold_quantity, sale_start_at, sale_end_at, pay_channels, max_tickets_per_order, status, created_at, updated_at`

func scanTicketType(row interface{ Scan(...any) error }) (TicketType, error) {
	var t TicketType
	var payChannels []byte
	err := row.Scan(&t.ID, &t.ActivityID, &t.SessionID, &t.StoreID, &t.Name, &t.PriceCent, &t.StockQuantity,
		&t.SoldQuantity, &t.SaleStartAt, &t.SaleEndAt, &payChannels, &t.MaxTicketsPerOrder, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return TicketType{}, err
	}
	t.PayChannels = decodeChannels(payChannels)
	return t, nil
}

func (r *sqlConsoleRepository) ListTicketTypes(ctx context.Context, activityID int64, sessionID *int64) ([]TicketType, error) {
	where := `activity_id = ?`
	args := []any{activityID}
	if sessionID != nil {
		where += ` AND session_id = ?`
		args = append(args, *sessionID)
	}
	q := `SELECT ` + ticketTypeColumns + ` FROM activity_ticket_types WHERE ` + where + ` ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []TicketType
	for rows.Next() {
		t, err := scanTicketType(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *sqlConsoleRepository) GetTicketType(ctx context.Context, activityID, ticketTypeID int64) (TicketType, error) {
	q := `SELECT ` + ticketTypeColumns + ` FROM activity_ticket_types WHERE activity_id = ? AND id = ?`
	t, err := scanTicketType(r.db.QueryRowContext(ctx, q, activityID, ticketTypeID))
	if errors.Is(err, sql.ErrNoRows) {
		return TicketType{}, apperr.NotFound("ticket type not found")
	}
	if err != nil {
		return TicketType{}, apperr.Internal(err)
	}
	return t, nil
}

func (r *sqlConsoleRepository) CreateTicketType(ctx context.Context, activityID int64, in TicketTypeInput) (TicketType, error) {
	// A session-bound ticket type must reference a session under the same
	// activity; GetSession enforces that (and returns NOT_FOUND otherwise).
	if in.SessionID != nil {
		if _, err := r.GetSession(ctx, activityID, *in.SessionID); err != nil {
			return TicketType{}, err
		}
	}
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "active"
	}
	const q = `INSERT INTO activity_ticket_types
		(activity_id, session_id, store_id, name, price_cent, stock_quantity, sold_quantity,
		 sale_start_at, sale_end_at, pay_channels, max_tickets_per_order, status, created_at, updated_at)
		VALUES (?, ?, (SELECT store_id FROM activities WHERE id = ?), ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, activityID, in.SessionID, activityID, in.Name, in.PriceCent,
		in.StockQuantity, in.SaleStartAt, in.SaleEndAt, encodeChannels(in.PayChannels),
		in.MaxTicketsPerOrder, status, now, now)
	if err != nil {
		return TicketType{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return TicketType{}, apperr.Internal(err)
	}
	return r.GetTicketType(ctx, activityID, id)
}

func (r *sqlConsoleRepository) UpdateTicketType(ctx context.Context, activityID, ticketTypeID int64, in TicketTypeInput) (TicketType, error) {
	if in.SessionID != nil {
		if _, err := r.GetSession(ctx, activityID, *in.SessionID); err != nil {
			return TicketType{}, err
		}
	}
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "active"
	}
	const q = `UPDATE activity_ticket_types SET session_id=?, name=?, price_cent=?, stock_quantity=?,
		sale_start_at=?, sale_end_at=?, pay_channels=?, max_tickets_per_order=?, status=?, updated_at=?
		WHERE id=? AND activity_id=?`
	if _, err := r.db.ExecContext(ctx, q, in.SessionID, in.Name, in.PriceCent, in.StockQuantity,
		in.SaleStartAt, in.SaleEndAt, encodeChannels(in.PayChannels), in.MaxTicketsPerOrder, status, now,
		ticketTypeID, activityID); err != nil {
		return TicketType{}, apperr.Internal(err)
	}
	return r.GetTicketType(ctx, activityID, ticketTypeID)
}

func (r *sqlConsoleRepository) DeleteTicketType(ctx context.Context, activityID, ticketTypeID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM activity_ticket_types WHERE id=? AND activity_id=?`, ticketTypeID, activityID)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("ticket type not found")
	}
	return nil
}

// ConsoleService provides the console CRUD operations for activities,
// sessions and ticket types, mapping repository models onto console views.
type ConsoleService struct {
	repo   ConsoleRepository
	assets AssetResolver
}

// NewConsoleService builds the console service.
func NewConsoleService(repo ConsoleRepository, assets ...AssetResolver) *ConsoleService {
	var resolver AssetResolver
	if len(assets) > 0 {
		resolver = assets[0]
	}
	return &ConsoleService{repo: repo, assets: resolver}
}

func (s *ConsoleService) activityView(ctx context.Context, a Activity) ConsoleActivityView {
	view := ConsoleActivityView{
		ID: a.ID, ScopeType: a.ScopeType, StoreID: a.StoreID, Title: a.Title,
		Description: a.Description, Content: a.Content, AssetID: a.AssetID,
		StartAt: a.StartAt, EndAt: a.EndAt, PayChannels: a.PayChannels,
		PurchaseLimitPerMember: a.PurchaseLimit, Status: a.Status,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
	if s.assets != nil && a.AssetID != nil {
		view.ImageURL, _ = s.assets.PublicURLByID(ctx, *a.AssetID)
	}
	return view
}

func sessionView(s Session) SessionView {
	return SessionView{
		ID: s.ID, ActivityID: s.ActivityID, StoreID: s.StoreID, Name: s.Name,
		StartAt: s.StartAt, EndAt: s.EndAt, Status: s.Status, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func ticketTypeView(t TicketType) TicketTypeView {
	return TicketTypeView{
		ID: t.ID, ActivityID: t.ActivityID, SessionID: t.SessionID, StoreID: t.StoreID, Name: t.Name,
		PriceCent: t.PriceCent, StockQuantity: t.StockQuantity, SoldQuantity: t.SoldQuantity,
		SaleStartAt: t.SaleStartAt, SaleEndAt: t.SaleEndAt, PayChannels: t.PayChannels,
		MaxTicketsPerOrder: t.MaxTicketsPerOrder, Status: t.Status, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

// ListActivities returns a page of activities visible to scope.
func (s *ConsoleService) ListActivities(ctx context.Context, scope ConsoleScope, page httpx.Page) ([]ConsoleActivityView, int64, error) {
	acts, total, err := s.repo.ListActivities(ctx, scope, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ConsoleActivityView, 0, len(acts))
	for _, a := range acts {
		out = append(out, s.activityView(ctx, a))
	}
	return out, total, nil
}

// GetActivity returns a single activity visible to scope.
func (s *ConsoleService) GetActivity(ctx context.Context, scope ConsoleScope, id int64) (ConsoleActivityView, error) {
	a, err := s.repo.GetActivity(ctx, scope, id)
	if err != nil {
		return ConsoleActivityView{}, err
	}
	return s.activityView(ctx, a), nil
}

// CreateActivity creates an activity within scope.
func (s *ConsoleService) CreateActivity(ctx context.Context, scope ConsoleScope, in ActivityInput) (ConsoleActivityView, error) {
	if in.Status == "" {
		in.Status = "published"
	}
	payChannels, err := normalizePayChannels(in.PayChannels)
	if err != nil {
		return ConsoleActivityView{}, err
	}
	in.PayChannels = payChannels
	a, err := s.repo.CreateActivity(ctx, scope, in)
	if err != nil {
		return ConsoleActivityView{}, err
	}
	return s.activityView(ctx, a), nil
}

// UpdateActivity updates an activity within scope.
func (s *ConsoleService) UpdateActivity(ctx context.Context, scope ConsoleScope, id int64, in ActivityInput) (ConsoleActivityView, error) {
	if in.Status == "" {
		in.Status = "published"
	}
	payChannels, err := normalizePayChannels(in.PayChannels)
	if err != nil {
		return ConsoleActivityView{}, err
	}
	in.PayChannels = payChannels
	a, err := s.repo.UpdateActivity(ctx, scope, id, in)
	if err != nil {
		return ConsoleActivityView{}, err
	}
	return s.activityView(ctx, a), nil
}

// DeleteActivity deletes an activity within scope.
func (s *ConsoleService) DeleteActivity(ctx context.Context, scope ConsoleScope, id int64) error {
	return s.repo.DeleteActivity(ctx, scope, id)
}

// ListSessions returns the sessions of an activity, after verifying the
// activity is visible to scope.
func (s *ConsoleService) ListSessions(ctx context.Context, scope ConsoleScope, activityID int64) ([]SessionView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSessions(ctx, activityID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionView, 0, len(rows))
	for _, r := range rows {
		out = append(out, sessionView(r))
	}
	return out, nil
}

// GetSession returns a single session, after verifying the activity is
// visible to scope.
func (s *ConsoleService) GetSession(ctx context.Context, scope ConsoleScope, activityID, sessionID int64) (SessionView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return SessionView{}, err
	}
	sess, err := s.repo.GetSession(ctx, activityID, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	return sessionView(sess), nil
}

// CreateSession creates a session under an activity within scope.
func (s *ConsoleService) CreateSession(ctx context.Context, scope ConsoleScope, activityID int64, in SessionInput) (SessionView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return SessionView{}, err
	}
	sess, err := s.repo.CreateSession(ctx, activityID, in)
	if err != nil {
		return SessionView{}, err
	}
	return sessionView(sess), nil
}

// UpdateSession updates a session under an activity within scope.
func (s *ConsoleService) UpdateSession(ctx context.Context, scope ConsoleScope, activityID, sessionID int64, in SessionInput) (SessionView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return SessionView{}, err
	}
	sess, err := s.repo.UpdateSession(ctx, activityID, sessionID, in)
	if err != nil {
		return SessionView{}, err
	}
	return sessionView(sess), nil
}

// DeleteSession deletes a session under an activity within scope.
func (s *ConsoleService) DeleteSession(ctx context.Context, scope ConsoleScope, activityID, sessionID int64) error {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return err
	}
	return s.repo.DeleteSession(ctx, activityID, sessionID)
}

// ListTicketTypes returns the ticket types of an activity (optionally
// narrowed to a session), after verifying the activity is visible to scope.
func (s *ConsoleService) ListTicketTypes(ctx context.Context, scope ConsoleScope, activityID int64, sessionID *int64) ([]TicketTypeView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListTicketTypes(ctx, activityID, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]TicketTypeView, 0, len(rows))
	for _, r := range rows {
		out = append(out, ticketTypeView(r))
	}
	return out, nil
}

// GetTicketType returns a single ticket type, after verifying the activity is
// visible to scope.
func (s *ConsoleService) GetTicketType(ctx context.Context, scope ConsoleScope, activityID, ticketTypeID int64) (TicketTypeView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return TicketTypeView{}, err
	}
	t, err := s.repo.GetTicketType(ctx, activityID, ticketTypeID)
	if err != nil {
		return TicketTypeView{}, err
	}
	return ticketTypeView(t), nil
}

// CreateTicketType creates a ticket type under an activity within scope.
func (s *ConsoleService) CreateTicketType(ctx context.Context, scope ConsoleScope, activityID int64, in TicketTypeInput) (TicketTypeView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return TicketTypeView{}, err
	}
	if err := validateTicketTypeInput(in); err != nil {
		return TicketTypeView{}, err
	}
	payChannels, err := normalizePayChannels(in.PayChannels)
	if err != nil {
		return TicketTypeView{}, err
	}
	in.PayChannels = payChannels
	t, err := s.repo.CreateTicketType(ctx, activityID, in)
	if err != nil {
		return TicketTypeView{}, err
	}
	return ticketTypeView(t), nil
}

// UpdateTicketType updates a ticket type under an activity within scope.
func (s *ConsoleService) UpdateTicketType(ctx context.Context, scope ConsoleScope, activityID, ticketTypeID int64, in TicketTypeInput) (TicketTypeView, error) {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return TicketTypeView{}, err
	}
	if err := validateTicketTypeInput(in); err != nil {
		return TicketTypeView{}, err
	}
	payChannels, err := normalizePayChannels(in.PayChannels)
	if err != nil {
		return TicketTypeView{}, err
	}
	in.PayChannels = payChannels
	t, err := s.repo.UpdateTicketType(ctx, activityID, ticketTypeID, in)
	if err != nil {
		return TicketTypeView{}, err
	}
	return ticketTypeView(t), nil
}

func validateTicketTypeInput(in TicketTypeInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return apperr.Invalid("请填写票档名称")
	}
	if in.PriceCent <= 0 {
		return apperr.Invalid("票档价格必须大于 0")
	}
	if in.StockQuantity < 0 {
		return apperr.Invalid("票档库存不能小于 0")
	}
	if in.MaxTicketsPerOrder < 0 {
		return apperr.Invalid("单次限购数量不能小于 0")
	}
	if (in.SaleStartAt == nil) != (in.SaleEndAt == nil) {
		return apperr.Invalid("售卖开始时间和结束时间必须同时填写")
	}
	if in.SaleStartAt != nil && !in.SaleEndAt.After(*in.SaleStartAt) {
		return apperr.Invalid("售卖结束时间必须晚于开始时间")
	}
	if (in.Name == "早鸟票" || in.Name == "预售票") && in.SaleStartAt == nil {
		return apperr.Invalid("早鸟票和预售票必须设置售卖时间")
	}
	return nil
}

// DeleteTicketType deletes a ticket type under an activity within scope.
func (s *ConsoleService) DeleteTicketType(ctx context.Context, scope ConsoleScope, activityID, ticketTypeID int64) error {
	if _, err := s.repo.GetActivity(ctx, scope, activityID); err != nil {
		return err
	}
	return s.repo.DeleteTicketType(ctx, activityID, ticketTypeID)
}

// ConsoleHandler exposes the admin and store console CRUD endpoints for
// activities, sessions and ticket types. Router wiring lives outside this
// module; admin methods mount under the admin audience (scope nil) and Store*
// methods under the store audience (scope pinned from the JWT).
type ConsoleHandler struct {
	svc *ConsoleService
}

// NewConsoleHandler builds the console handler.
func NewConsoleHandler(svc *ConsoleService) *ConsoleHandler { return &ConsoleHandler{svc: svc} }

// --- Admin console (audience: admin, no store filter) ---

// Activities handles GET /admin/activities.
func (h *ConsoleHandler) Activities(c *gin.Context) {
	h.listActivities(c, ConsoleScope{})
}

// ActivityDetail handles GET /admin/activities/{activityID}.
func (h *ConsoleHandler) ActivityDetail(c *gin.Context) {
	h.getActivity(c, ConsoleScope{})
}

// CreateActivity handles POST /admin/activities.
func (h *ConsoleHandler) CreateActivity(c *gin.Context) {
	h.createActivity(c, ConsoleScope{})
}

// UpdateActivity handles PUT /admin/activities/{activityID}.
func (h *ConsoleHandler) UpdateActivity(c *gin.Context) {
	h.updateActivity(c, ConsoleScope{})
}

// DeleteActivity handles DELETE /admin/activities/{activityID}.
func (h *ConsoleHandler) DeleteActivity(c *gin.Context) {
	h.deleteActivity(c, ConsoleScope{})
}

// --- Store console (audience: store, scope pinned from JWT) ---

// StoreActivities handles GET /store/activities.
func (h *ConsoleHandler) StoreActivities(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.listActivities(c, scope)
}

// StoreActivityDetail handles GET /store/activities/{activityID}.
func (h *ConsoleHandler) StoreActivityDetail(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.getActivity(c, scope)
}

// StoreCreateActivity handles POST /store/activities.
func (h *ConsoleHandler) StoreCreateActivity(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.createActivity(c, scope)
}

// StoreUpdateActivity handles PUT /store/activities/{activityID}.
func (h *ConsoleHandler) StoreUpdateActivity(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.updateActivity(c, scope)
}

// StoreDeleteActivity handles DELETE /store/activities/{activityID}.
func (h *ConsoleHandler) StoreDeleteActivity(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.deleteActivity(c, scope)
}

func storeScope(c *gin.Context) (ConsoleScope, bool) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return ConsoleScope{}, false
	}
	return ConsoleScope{StoreID: &storeID}, true
}

func (h *ConsoleHandler) listActivities(c *gin.Context, scope ConsoleScope) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListActivities(c.Request.Context(), scope, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) getActivity(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetActivity(c.Request.Context(), scope, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createActivity(c *gin.Context, scope ConsoleScope) {
	var in ActivityInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateActivity(c.Request.Context(), scope, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) updateActivity(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in ActivityInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateActivity(c.Request.Context(), scope, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteActivity(c *gin.Context, scope ConsoleScope) {
	id, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteActivity(c.Request.Context(), scope, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(204)
}

// --- Admin console: sessions (audience: admin, no store filter) ---

// Sessions handles GET /admin/activities/{activityID}/sessions.
func (h *ConsoleHandler) Sessions(c *gin.Context) {
	h.listSessions(c, ConsoleScope{})
}

// SessionDetail handles GET /admin/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) SessionDetail(c *gin.Context) {
	h.getSession(c, ConsoleScope{})
}

// CreateSession handles POST /admin/activities/{activityID}/sessions.
func (h *ConsoleHandler) CreateSession(c *gin.Context) {
	h.createSession(c, ConsoleScope{})
}

// UpdateSession handles PUT /admin/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) UpdateSession(c *gin.Context) {
	h.updateSession(c, ConsoleScope{})
}

// DeleteSession handles DELETE /admin/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) DeleteSession(c *gin.Context) {
	h.deleteSession(c, ConsoleScope{})
}

// --- Admin console: ticket types (audience: admin, no store filter) ---

// TicketTypes handles GET /admin/activities/{activityID}/ticket-types.
func (h *ConsoleHandler) TicketTypes(c *gin.Context) {
	h.listTicketTypes(c, ConsoleScope{})
}

// TicketTypeDetail handles GET /admin/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) TicketTypeDetail(c *gin.Context) {
	h.getTicketType(c, ConsoleScope{})
}

// CreateTicketType handles POST /admin/activities/{activityID}/ticket-types.
func (h *ConsoleHandler) CreateTicketType(c *gin.Context) {
	h.createTicketType(c, ConsoleScope{})
}

// UpdateTicketType handles PUT /admin/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) UpdateTicketType(c *gin.Context) {
	h.updateTicketType(c, ConsoleScope{})
}

// DeleteTicketType handles DELETE /admin/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) DeleteTicketType(c *gin.Context) {
	h.deleteTicketType(c, ConsoleScope{})
}

// --- Store console: sessions (audience: store, scope pinned from JWT) ---

// StoreSessions handles GET /store/activities/{activityID}/sessions.
func (h *ConsoleHandler) StoreSessions(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.listSessions(c, scope)
}

// StoreSessionDetail handles GET /store/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) StoreSessionDetail(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.getSession(c, scope)
}

// StoreCreateSession handles POST /store/activities/{activityID}/sessions.
func (h *ConsoleHandler) StoreCreateSession(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.createSession(c, scope)
}

// StoreUpdateSession handles PUT /store/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) StoreUpdateSession(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.updateSession(c, scope)
}

// StoreDeleteSession handles DELETE /store/activities/{activityID}/sessions/{sessionID}.
func (h *ConsoleHandler) StoreDeleteSession(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.deleteSession(c, scope)
}

// --- Store console: ticket types (audience: store, scope pinned from JWT) ---

// StoreTicketTypes handles GET /store/activities/{activityID}/ticket-types.
func (h *ConsoleHandler) StoreTicketTypes(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.listTicketTypes(c, scope)
}

// StoreTicketTypeDetail handles GET /store/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) StoreTicketTypeDetail(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.getTicketType(c, scope)
}

// StoreCreateTicketType handles POST /store/activities/{activityID}/ticket-types.
func (h *ConsoleHandler) StoreCreateTicketType(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.createTicketType(c, scope)
}

// StoreUpdateTicketType handles PUT /store/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) StoreUpdateTicketType(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.updateTicketType(c, scope)
}

// StoreDeleteTicketType handles DELETE /store/activities/{activityID}/ticket-types/{ticketTypeID}.
func (h *ConsoleHandler) StoreDeleteTicketType(c *gin.Context) {
	scope, ok := storeScope(c)
	if !ok {
		return
	}
	h.deleteTicketType(c, scope)
}

// --- Session handler bodies (scope supplied by the audience wrapper) ---

func (h *ConsoleHandler) listSessions(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListSessions(c.Request.Context(), scope, activityID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

func (h *ConsoleHandler) getSession(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	sessionID, err := pathID(c, "sessionID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetSession(c.Request.Context(), scope, activityID, sessionID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createSession(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in SessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateSession(c.Request.Context(), scope, activityID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) updateSession(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	sessionID, err := pathID(c, "sessionID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in SessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateSession(c.Request.Context(), scope, activityID, sessionID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteSession(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	sessionID, err := pathID(c, "sessionID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteSession(c.Request.Context(), scope, activityID, sessionID); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(204)
}

// --- Ticket-type handler bodies (scope supplied by the audience wrapper) ---

func (h *ConsoleHandler) listTicketTypes(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	sessionID, err := optionalSessionFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListTicketTypes(c.Request.Context(), scope, activityID, sessionID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

func (h *ConsoleHandler) getTicketType(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	ticketTypeID, err := pathID(c, "ticketTypeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetTicketType(c.Request.Context(), scope, activityID, ticketTypeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) createTicketType(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in TicketTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateTicketType(c.Request.Context(), scope, activityID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) updateTicketType(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	ticketTypeID, err := pathID(c, "ticketTypeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in TicketTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateTicketType(c.Request.Context(), scope, activityID, ticketTypeID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *ConsoleHandler) deleteTicketType(c *gin.Context, scope ConsoleScope) {
	activityID, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	ticketTypeID, err := pathID(c, "ticketTypeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteTicketType(c.Request.Context(), scope, activityID, ticketTypeID); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(204)
}

// optionalSessionFilter reads the optional ?sessionId= query used to narrow a
// ticket-type listing to a single session; absent means "all sessions".
func optionalSessionFilter(c *gin.Context) (*int64, error) {
	raw := c.Query("sessionId")
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, apperr.Invalid("invalid sessionId")
	}
	return &id, nil
}
