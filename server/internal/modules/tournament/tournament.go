package tournament

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// Event is a store-bound informational tournament promotion. It intentionally
// has no tickets, prices, payment channels or verification state.
type Event struct {
	ID        int64      `json:"id"`
	StoreID   int64      `json:"storeId"`
	StoreName string     `json:"storeName"`
	Title     string     `json:"title"`
	Summary   string     `json:"summary,omitempty"`
	Content   string     `json:"content,omitempty"`
	AssetID   *int64     `json:"assetId,omitempty"`
	ImageURL  string     `json:"imageUrl,omitempty"`
	StartAt   *time.Time `json:"startAt"`
	EndAt     *time.Time `json:"endAt"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Input struct {
	StoreID int64      `json:"storeId"`
	Title   string     `json:"title"`
	Summary string     `json:"summary"`
	Content string     `json:"content"`
	AssetID *int64     `json:"assetId"`
	StartAt *time.Time `json:"startAt"`
	EndAt   *time.Time `json:"endAt"`
	Status  string     `json:"status"`
}

type Filter struct {
	StoreID int64
	Keyword string
	Status  string
}

type Repository interface {
	List(context.Context, *int64, Filter, httpx.Page) ([]Event, int64, error)
	ListCurrent(context.Context, int64, time.Time) ([]Event, error)
	Get(context.Context, *int64, int64, bool) (Event, error)
	Create(context.Context, *int64, Input, time.Time) (Event, error)
	Update(context.Context, *int64, int64, Input, time.Time) (Event, error)
	Delete(context.Context, *int64, int64) error
}

type sqlRepository struct{ db *platdb.DB }

func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

const columns = `te.id, te.store_id, s.name, te.title, COALESCE(te.summary,''),
	COALESCE(te.content,''), te.asset_id, te.start_at, te.end_at, te.status,
	te.created_at, te.updated_at`
const from = ` FROM tournament_events te JOIN stores s ON s.id = te.store_id `

func scan(row interface{ Scan(...any) error }) (Event, error) {
	var out Event
	err := row.Scan(&out.ID, &out.StoreID, &out.StoreName, &out.Title, &out.Summary,
		&out.Content, &out.AssetID, &out.StartAt, &out.EndAt, &out.Status,
		&out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func listWhere(scope *int64, filter Filter) (string, []any) {
	parts := []string{"1=1"}
	args := make([]any, 0, 4)
	if scope != nil {
		parts = append(parts, "te.store_id=?")
		args = append(args, *scope)
	} else if filter.StoreID > 0 {
		parts = append(parts, "te.store_id=?")
		args = append(args, filter.StoreID)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		parts = append(parts, "te.title LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		parts = append(parts, "te.status=?")
		args = append(args, status)
	}
	return strings.Join(parts, " AND "), args
}

func (r *sqlRepository) List(ctx context.Context, scope *int64, filter Filter, page httpx.Page) ([]Event, int64, error) {
	where, args := listWhere(scope, filter)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from+`WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	queryArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+from+`WHERE `+where+` ORDER BY te.id DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Event, 0, page.Limit())
	for rows.Next() {
		event, err := scan(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return out, total, nil
}

func (r *sqlRepository) ListCurrent(ctx context.Context, storeID int64, now time.Time) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+from+`WHERE te.store_id=? AND te.status='published'
		AND te.start_at<=? AND te.end_at>=? ORDER BY te.start_at ASC, te.id DESC`, storeID, now, now)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Event, 0)
	for rows.Next() {
		event, err := scan(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

func eventWhere(scope *int64, publishedOnly bool) (string, []any) {
	parts := []string{"te.id=?"}
	args := make([]any, 0, 2)
	if scope != nil {
		parts = append(parts, "te.store_id=?")
		args = append(args, *scope)
	}
	if publishedOnly {
		parts = append(parts, "te.status='published'")
	}
	return strings.Join(parts, " AND "), args
}

func (r *sqlRepository) Get(ctx context.Context, scope *int64, id int64, publishedOnly bool) (Event, error) {
	where, args := eventWhere(scope, publishedOnly)
	event, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+from+`WHERE `+where, append([]any{id}, args...)...))
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, apperr.NotFound("赛事活动不存在")
	}
	if err != nil {
		return Event{}, apperr.Internal(err)
	}
	return event, nil
}

func resolveStoreID(scope *int64, requested int64) (int64, error) {
	if scope != nil {
		return *scope, nil
	}
	if requested <= 0 {
		return 0, apperr.Invalid("请选择所属门店")
	}
	return requested, nil
}

func (r *sqlRepository) ensureStore(ctx context.Context, storeID int64) error {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id=?)`, storeID).Scan(&exists); err != nil {
		return apperr.Internal(err)
	}
	if !exists {
		return apperr.Invalid("所属门店不存在")
	}
	return nil
}

func (r *sqlRepository) Create(ctx context.Context, scope *int64, in Input, now time.Time) (Event, error) {
	storeID, err := resolveStoreID(scope, in.StoreID)
	if err != nil {
		return Event{}, err
	}
	if err := r.ensureStore(ctx, storeID); err != nil {
		return Event{}, err
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO tournament_events
		(store_id,title,summary,content,asset_id,start_at,end_at,status,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, storeID, in.Title, in.Summary, in.Content, in.AssetID,
		in.StartAt, in.EndAt, in.Status, now, now)
	if err != nil {
		return Event{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Event{}, apperr.Internal(err)
	}
	return r.Get(ctx, scope, id, false)
}

func (r *sqlRepository) Update(ctx context.Context, scope *int64, id int64, in Input, now time.Time) (Event, error) {
	storeID, err := resolveStoreID(scope, in.StoreID)
	if err != nil {
		return Event{}, err
	}
	if err := r.ensureStore(ctx, storeID); err != nil {
		return Event{}, err
	}
	where := "id=?"
	args := []any{storeID, in.Title, in.Summary, in.Content, in.AssetID, in.StartAt, in.EndAt, in.Status, now, id}
	if scope != nil {
		where += " AND store_id=?"
		args = append(args, *scope)
	}
	res, err := r.db.ExecContext(ctx, `UPDATE tournament_events SET store_id=?,title=?,summary=?,content=?,
		asset_id=?,start_at=?,end_at=?,status=?,updated_at=? WHERE `+where, args...)
	if err != nil {
		return Event{}, apperr.Internal(err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return Event{}, apperr.NotFound("赛事活动不存在")
	}
	return r.Get(ctx, scope, id, false)
}

func (r *sqlRepository) Delete(ctx context.Context, scope *int64, id int64) error {
	query := `DELETE FROM tournament_events WHERE id=?`
	args := []any{id}
	if scope != nil {
		query += ` AND store_id=?`
		args = append(args, *scope)
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return apperr.Internal(err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return apperr.NotFound("赛事活动不存在")
	}
	return nil
}

type AssetResolver interface {
	PublicURLByID(context.Context, int64) (string, error)
}

type Service struct {
	repo   Repository
	assets AssetResolver
	now    func() time.Time
}

func NewService(repo Repository, assets AssetResolver) *Service {
	return &Service{repo: repo, assets: assets, now: time.Now}
}

func (s *Service) decorate(ctx context.Context, events []Event) ([]Event, error) {
	for i := range events {
		if events[i].AssetID == nil {
			continue
		}
		url, err := s.assets.PublicURLByID(ctx, *events[i].AssetID)
		if err != nil {
			return nil, err
		}
		events[i].ImageURL = url
	}
	return events, nil
}

func (s *Service) decorateOne(ctx context.Context, event Event) (Event, error) {
	events, err := s.decorate(ctx, []Event{event})
	if err != nil {
		return Event{}, err
	}
	return events[0], nil
}

func validate(in Input) (Input, error) {
	var err error
	in.Title, err = inputvalidation.PlainText(in.Title, inputvalidation.TextOptions{Label: "赛事标题", MinRunes: 1, MaxRunes: 128})
	if err != nil {
		return Input{}, apperr.Invalid(err.Error())
	}
	in.Summary, err = inputvalidation.PlainText(in.Summary, inputvalidation.TextOptions{Label: "赛事简介", MaxRunes: 500, AllowEmpty: true, AllowNewlines: true})
	if err != nil {
		return Input{}, apperr.Invalid(err.Error())
	}
	in.Content = inputvalidation.SanitizeRichHTML(in.Content)
	if in.StartAt == nil || in.EndAt == nil {
		return Input{}, apperr.Invalid("请设置赛事开始和结束时间")
	}
	if !in.StartAt.Before(*in.EndAt) {
		return Input{}, apperr.Invalid("赛事结束时间必须晚于开始时间")
	}
	if in.AssetID != nil && *in.AssetID <= 0 {
		return Input{}, apperr.Invalid("赛事图片不正确")
	}
	if in.Status == "" {
		in.Status = "published"
	}
	if in.Status != "draft" && in.Status != "published" {
		return Input{}, apperr.Invalid("发布状态不正确")
	}
	return in, nil
}

func (s *Service) List(ctx context.Context, scope *int64, filter Filter, page httpx.Page) ([]Event, int64, error) {
	events, total, err := s.repo.List(ctx, scope, filter, page)
	if err != nil {
		return nil, 0, err
	}
	events, err = s.decorate(ctx, events)
	return events, total, err
}

func (s *Service) ListCurrent(ctx context.Context, storeID int64) ([]Event, error) {
	if storeID <= 0 {
		return nil, apperr.Invalid("门店 ID 不正确")
	}
	events, err := s.repo.ListCurrent(ctx, storeID, s.now().UTC())
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, events)
}

func (s *Service) Get(ctx context.Context, scope *int64, id int64, publishedOnly bool) (Event, error) {
	event, err := s.repo.Get(ctx, scope, id, publishedOnly)
	if err != nil {
		return Event{}, err
	}
	return s.decorateOne(ctx, event)
}

func (s *Service) Create(ctx context.Context, scope *int64, in Input) (Event, error) {
	in, err := validate(in)
	if err != nil {
		return Event{}, err
	}
	event, err := s.repo.Create(ctx, scope, in, s.now().UTC())
	if err != nil {
		return Event{}, err
	}
	return s.decorateOne(ctx, event)
}

func (s *Service) Update(ctx context.Context, scope *int64, id int64, in Input) (Event, error) {
	in, err := validate(in)
	if err != nil {
		return Event{}, err
	}
	event, err := s.repo.Update(ctx, scope, id, in, s.now().UTC())
	if err != nil {
		return Event{}, err
	}
	return s.decorateOne(ctx, event)
}

func (s *Service) Delete(ctx context.Context, scope *int64, id int64) error {
	return s.repo.Delete(ctx, scope, id)
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func positivePathID(c *gin.Context, key string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(key)), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("ID 不正确")
	}
	return id, nil
}

func parseFilter(c *gin.Context) Filter {
	storeID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("storeId")), 10, 64)
	return Filter{StoreID: storeID, Keyword: c.Query("keyword"), Status: c.Query("status")}
}

func (h *Handler) PublicList(c *gin.Context) {
	storeID, err := positivePathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	events, err := h.svc.ListCurrent(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, events)
}

func (h *Handler) PublicDetail(c *gin.Context) {
	id, err := positivePathID(c, "eventID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	event, err := h.svc.Get(c.Request.Context(), nil, id, true)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, event)
}

func (h *Handler) AdminList(c *gin.Context) { h.list(c, nil) }
func (h *Handler) StoreList(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if ok {
		h.list(c, &storeID)
	}
}
func (h *Handler) list(c *gin.Context, scope *int64) {
	page := httpx.ParsePage(c)
	events, total, err := h.svc.List(c.Request.Context(), scope, parseFilter(c), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, events, httpx.MetaFor(page, total))
}

func (h *Handler) AdminGet(c *gin.Context) { h.get(c, nil) }
func (h *Handler) StoreGet(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if ok {
		h.get(c, &storeID)
	}
}
func (h *Handler) get(c *gin.Context, scope *int64) {
	id, err := positivePathID(c, "eventID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	event, err := h.svc.Get(c.Request.Context(), scope, id, false)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, event)
}

func (h *Handler) AdminCreate(c *gin.Context) { h.create(c, nil) }
func (h *Handler) StoreCreate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if ok {
		h.create(c, &storeID)
	}
}
func (h *Handler) create(c *gin.Context, scope *int64) {
	var in Input
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	event, err := h.svc.Create(c.Request.Context(), scope, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, event)
}

func (h *Handler) AdminUpdate(c *gin.Context) { h.update(c, nil) }
func (h *Handler) StoreUpdate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if ok {
		h.update(c, &storeID)
	}
}
func (h *Handler) update(c *gin.Context, scope *int64) {
	id, err := positivePathID(c, "eventID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in Input
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	event, err := h.svc.Update(c.Request.Context(), scope, id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, event)
}

func (h *Handler) AdminDelete(c *gin.Context) { h.remove(c, nil) }
func (h *Handler) StoreDelete(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if ok {
		h.remove(c, &storeID)
	}
}
func (h *Handler) remove(c *gin.Context, scope *int64) {
	id, err := positivePathID(c, "eventID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), scope, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}
