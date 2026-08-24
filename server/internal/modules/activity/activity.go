// Package activity owns activities, sessions, ticket types, orders, tickets and
// verifications. Phase-1 exposes the mini-program public read paths.
package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// Activity is an activity/event template or store instance.
type Activity struct {
	ID             int64
	ScopeType      string
	StoreID        *int64
	StoreName      string
	StoreAddress   string
	StoreLatitude  *float64
	StoreLongitude *float64
	Title          string
	Description    string
	Content        string
	AssetID        *int64
	StartAt        *time.Time
	EndAt          *time.Time
	PayChannels    []string
	PurchaseLimit  int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ActivityView is the public representation. TicketTypes is populated only on
// the detail read (Get); the list read leaves it nil to avoid a per-row lookup.
type ActivityView struct {
	ID            int64                  `json:"id"`
	ScopeType     string                 `json:"scopeType"`
	StoreID       *int64                 `json:"storeId,omitempty"`
	StoreName     string                 `json:"storeName,omitempty"`
	Address       string                 `json:"address,omitempty"`
	Latitude      *float64               `json:"latitude,omitempty"`
	Longitude     *float64               `json:"longitude,omitempty"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	Content       string                 `json:"content,omitempty"`
	ImageURL      string                 `json:"imageUrl,omitempty"`
	StartAt       *time.Time             `json:"startAt,omitempty"`
	EndAt         *time.Time             `json:"endAt,omitempty"`
	PayChannels   []string               `json:"payChannels"`
	PurchaseLimit int                    `json:"purchaseLimit,omitempty"`
	Status        string                 `json:"status"`
	TicketTypes   []PublicTicketTypeView `json:"ticketTypes,omitempty"`
}

// PublicTicketTypeView is a sellable ticket tier as shown on the mini-program
// activity detail purchase sheet (pages/activity-detail). It is the public,
// buyer-facing subset of the console TicketType — no sold count, audit
// timestamps or scope. Stock is the remaining sellable quantity; -1 signals an
// uncapped ticket type (stock_quantity 0).
type PublicTicketTypeView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	PriceCent int64  `json:"priceCent"`
	// Stock is retained for old clients: -1 means unlimited, otherwise it is the
	// remaining finite quantity. New clients use the explicit fields below.
	Stock              int64      `json:"stock"`
	UnlimitedStock     bool       `json:"unlimitedStock"`
	RemainingStock     *int64     `json:"remainingStock"`
	SaleStartAt        *time.Time `json:"saleStartAt,omitempty"`
	SaleEndAt          *time.Time `json:"saleEndAt,omitempty"`
	PayChannels        []string   `json:"payChannels"`
	MaxTicketsPerOrder int        `json:"maxTicketsPerOrder,omitempty"`
}

// AssetResolver resolves an asset id to a public URL.
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// Repository is the activity read persistence port.
type Repository interface {
	ListPublished(ctx context.Context, storeID *int64, endedBefore *time.Time, limit, offset int) ([]Activity, int64, error)
	ListTodayPublished(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]Activity, error)
	GetByID(ctx context.Context, id int64) (Activity, error)
	// ListSellableTicketTypes returns an activity's active ticket types for the
	// buyer-facing detail page, including upcoming, ended and sold-out tiers so
	// the client can show the complete configured tier list with availability.
	ListSellableTicketTypes(ctx context.Context, activityID int64) ([]TicketType, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL activity repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

// activityColumns is alias-qualified because reads LEFT JOIN stores to surface
// the owning store's name (the mini activity list/detail render storeName).
const activityColumns = `a.id, a.scope_type, a.store_id, COALESCE(s.name,''), COALESCE(s.address,''),
	s.latitude, s.longitude, a.title, COALESCE(a.description,''),
	COALESCE(a.content,''), a.asset_id, a.start_at, a.end_at, a.pay_channels, a.purchase_limit_per_member, a.status`

const activityFrom = ` FROM activities a LEFT JOIN stores s ON s.id = a.store_id `

func scanActivity(row interface{ Scan(...any) error }) (Activity, error) {
	var a Activity
	var payChannels []byte
	err := row.Scan(&a.ID, &a.ScopeType, &a.StoreID, &a.StoreName, &a.StoreAddress,
		&a.StoreLatitude, &a.StoreLongitude, &a.Title, &a.Description, &a.Content, &a.AssetID,
		&a.StartAt, &a.EndAt, &payChannels, &a.PurchaseLimit, &a.Status)
	if err != nil {
		return Activity{}, err
	}
	a.PayChannels = decodeChannels(payChannels)
	return a, nil
}

func (r *sqlRepository) ListPublished(ctx context.Context, storeID *int64, endedBefore *time.Time, limit, offset int) ([]Activity, int64, error) {
	where := `a.status = 'published'`
	var args []any
	if storeID != nil {
		where += ` AND (a.scope_type = 'global' OR a.store_id = ?)`
		args = append(args, *storeID)
	}
	if endedBefore != nil {
		where += ` AND a.end_at IS NOT NULL AND a.end_at < ?`
		args = append(args, *endedBefore)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+activityFrom+`WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	orderBy := `a.start_at DESC, a.id DESC`
	if endedBefore != nil {
		orderBy = `a.end_at DESC, a.id DESC`
	}
	q := `SELECT ` + activityColumns + activityFrom + `WHERE ` + where + ` ORDER BY ` + orderBy + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) ListTodayPublished(ctx context.Context, storeID int64, dayStart, dayEnd time.Time) ([]Activity, error) {
	const where = `a.status = 'published'
		AND (a.scope_type = 'global' OR a.store_id = ?)
		AND (a.start_at IS NULL OR a.start_at < ?)
		AND (a.end_at IS NULL OR a.end_at >= ?)`
	rows, err := r.db.QueryContext(ctx, `SELECT `+activityColumns+activityFrom+`WHERE `+where+
		` ORDER BY a.start_at ASC, a.id ASC`, storeID, dayEnd, dayStart)
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
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func (r *sqlRepository) GetByID(ctx context.Context, id int64) (Activity, error) {
	const q = `SELECT ` + activityColumns + activityFrom + `WHERE a.id = ?`
	a, err := scanActivity(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Activity{}, apperr.NotFound("activity not found")
	}
	if err != nil {
		return Activity{}, apperr.Internal(err)
	}
	return a, nil
}

// ListSellableTicketTypes reads the activity's active ticket types for the
// public detail page. Reuses the console ticket-type columns/scanner (same
// package). Inactive rows remain hidden, while timing and stock are returned so
// the client can explain why an active tier is not currently purchasable.
func (r *sqlRepository) ListSellableTicketTypes(ctx context.Context, activityID int64) ([]TicketType, error) {
	const q = `SELECT ` + ticketTypeColumns + ` FROM activity_ticket_types
		WHERE activity_id = ? AND status = 'active'
		ORDER BY price_cent ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, activityID)
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

func decodeChannels(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var ch []string
	if err := json.Unmarshal(raw, &ch); err != nil {
		return []string{}
	}
	normalized, _ := normalizePayChannels(ch)
	return normalized
}

func normalizePayChannels(channels []string) ([]string, error) {
	normalized := make([]string, 0, len(channels))
	seen := make(map[string]struct{}, len(channels))
	var invalid string
	for _, channel := range channels {
		if channel == "balance" {
			channel = "coin"
		}
		if channel != "wechat" && channel != "coin" && channel != "coupon" {
			if invalid == "" {
				invalid = channel
			}
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, channel)
	}
	if invalid != "" {
		return normalized, apperr.Invalid("activity: unsupported pay channel " + invalid)
	}
	return normalized, nil
}

// Service provides activity read operations.
type Service struct {
	repo   Repository
	assets AssetResolver
	now    func() time.Time
}

var publicActivityLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// NewService builds the activity service.
func NewService(repo Repository, assets AssetResolver) *Service {
	return &Service{repo: repo, assets: assets, now: time.Now}
}

// ListToday returns published global or store activities that overlap today's
// China Standard Time business-day window. The list is intentionally small and ordered by the
// activity start time for the reservation-page entry.
func (s *Service) ListToday(ctx context.Context, storeID int64) ([]ActivityView, error) {
	now := s.now().In(publicActivityLocation)
	localDayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, publicActivityLocation)
	dayStart := localDayStart.UTC()
	acts, err := s.repo.ListTodayPublished(ctx, storeID, dayStart, localDayStart.Add(24*time.Hour).UTC())
	if err != nil {
		return nil, err
	}
	views := make([]ActivityView, 0, len(acts))
	for _, activity := range acts {
		views = append(views, s.view(ctx, activity))
	}
	return views, nil
}

// List returns published activities, optionally scoped to a store or limited
// to activities that ended before the current time.
func (s *Service) List(ctx context.Context, storeID *int64, page httpx.Page, history bool) ([]ActivityView, int64, error) {
	var endedBefore *time.Time
	if history {
		now := s.now()
		endedBefore = &now
	}
	acts, total, err := s.repo.ListPublished(ctx, storeID, endedBefore, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]ActivityView, 0, len(acts))
	for _, a := range acts {
		views = append(views, s.view(ctx, a))
	}
	return views, total, nil
}

// Get returns a single published activity view enriched with its sellable
// ticket types so the mini-program detail page can build the purchase sheet.
func (s *Service) Get(ctx context.Context, id int64) (ActivityView, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ActivityView{}, err
	}
	v := s.view(ctx, a)
	tts, err := s.repo.ListSellableTicketTypes(ctx, id)
	if err != nil {
		return ActivityView{}, err
	}
	v.TicketTypes = make([]PublicTicketTypeView, 0, len(tts))
	for _, t := range tts {
		v.TicketTypes = append(v.TicketTypes, publicTicketTypeView(t))
	}
	return v, nil
}

func publicTicketTypeView(t TicketType) PublicTicketTypeView {
	stock := remainingStock(t.StockQuantity, t.SoldQuantity)
	var remaining *int64
	if t.StockQuantity > 0 {
		remaining = &stock
	}
	return PublicTicketTypeView{
		ID: t.ID, Name: t.Name, PriceCent: t.PriceCent,
		Stock:              stock,
		UnlimitedStock:     t.StockQuantity == 0,
		RemainingStock:     remaining,
		SaleStartAt:        t.SaleStartAt,
		SaleEndAt:          t.SaleEndAt,
		PayChannels:        t.PayChannels,
		MaxTicketsPerOrder: t.MaxTicketsPerOrder,
	}
}

// remainingStock reports a ticket type's sellable count: -1 for an uncapped
// type (stock_quantity 0, matching the reserve path's "unlimited" rule),
// otherwise the non-negative remainder of stock minus sold.
func remainingStock(stockQty, soldQty int64) int64 {
	if stockQty == 0 {
		return -1
	}
	if rem := stockQty - soldQty; rem > 0 {
		return rem
	}
	return 0
}

func (s *Service) view(ctx context.Context, a Activity) ActivityView {
	v := ActivityView{
		ID: a.ID, ScopeType: a.ScopeType, StoreID: a.StoreID, StoreName: a.StoreName,
		Address: a.StoreAddress, Latitude: a.StoreLatitude, Longitude: a.StoreLongitude,
		Title: a.Title, Description: a.Description, Content: inputvalidation.SanitizeRichHTML(a.Content),
		StartAt: a.StartAt, EndAt: a.EndAt, PayChannels: a.PayChannels,
		PurchaseLimit: a.PurchaseLimit, Status: a.Status,
	}
	if a.AssetID != nil {
		v.ImageURL, _ = s.assets.PublicURLByID(ctx, *a.AssetID)
	}
	return v
}
