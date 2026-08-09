// Package franchise handles public franchise inquiries and the headquarters read list.
package franchise

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

const (
	StatusUnprocessed = "unprocessed"
	StatusProcessed   = "processed"
)

// Inquiry is one franchise consultation submitted from the mini-program.
type Inquiry struct {
	ID              int64      `json:"id"`
	MemberID        *int64     `json:"memberId,omitempty"`
	MemberNickname  string     `json:"memberNickname,omitempty"`
	MemberPhone     string     `json:"memberPhone,omitempty"`
	MemberAvatarURL string     `json:"memberAvatarUrl,omitempty"`
	ContactName     string     `json:"contactName"`
	Phone           string     `json:"phone"`
	ExpectedRegion  string     `json:"expectedRegion"`
	Source          string     `json:"source"`
	Status          string     `json:"status"`
	ProcessedAt     *time.Time `json:"processedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// CreateInquiryRequest is the public form payload.
type CreateInquiryRequest struct {
	ContactName    string `json:"contactName"`
	Phone          string `json:"phone"`
	ExpectedRegion string `json:"expectedRegion"`
	Source         string `json:"source"`
}

// Config is the public configuration required to render the form.
type Config struct {
	Sources []string `json:"sources"`
	Hotline string   `json:"hotline"`
}

// ListFilter contains headquarters inquiry filters.
type ListFilter struct {
	Keyword string
	Source  string
	Status  string
}

// UpdateStatusRequest changes the headquarters handling state.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// SourceProvider supplies administrator-configured source options.
type SourceProvider interface {
	FranchiseInquirySources(ctx context.Context) ([]string, error)
	FranchiseHotline(ctx context.Context) (string, error)
}

// Repository persists franchise inquiries.
type Repository interface {
	Create(ctx context.Context, memberID *int64, req CreateInquiryRequest, now time.Time) (Inquiry, error)
	List(ctx context.Context, page httpx.Page, filter ListFilter) ([]Inquiry, int64, error)
	UpdateStatus(ctx context.Context, id int64, status string, now time.Time) error
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the SQL-backed repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) Create(ctx context.Context, memberID *int64, req CreateInquiryRequest, now time.Time) (Inquiry, error) {
	const q = `INSERT INTO franchise_inquiries
		(member_id, contact_name, phone, expected_region, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, q, memberID, req.ContactName, req.Phone, req.ExpectedRegion, req.Source, now, now)
	if err != nil {
		return Inquiry{}, apperr.Internal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Inquiry{}, apperr.Internal(err)
	}
	return Inquiry{
		ID: id, MemberID: memberID, ContactName: req.ContactName, Phone: req.Phone,
		ExpectedRegion: req.ExpectedRegion, Source: req.Source, Status: StatusUnprocessed, CreatedAt: now,
	}, nil
}

func (r *sqlRepository) List(ctx context.Context, page httpx.Page, filter ListFilter) ([]Inquiry, int64, error) {
	where := ` WHERE (? = '' OR fi.contact_name LIKE ? OR fi.phone LIKE ? OR fi.expected_region LIKE ?
		OR COALESCE(m.nickname, '') LIKE ? OR COALESCE(m.phone, '') LIKE ?)
		AND (? = '' OR fi.source = ?)
		AND (? = '' OR fi.status = ?)`
	keyword := "%" + filter.Keyword + "%"
	args := []any{filter.Keyword, keyword, keyword, keyword, keyword, keyword, filter.Source, filter.Source, filter.Status, filter.Status}
	joins := ` FROM franchise_inquiries fi LEFT JOIN members m ON m.id = fi.member_id`
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+joins+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT fi.id, fi.member_id,
		COALESCE(m.nickname, ''), COALESCE(m.phone, ''), COALESCE(m.avatar_url, ''),
		fi.contact_name, fi.phone, fi.expected_region, fi.source, fi.status, fi.processed_at, fi.created_at`+
		joins+where+` ORDER BY fi.id DESC LIMIT ? OFFSET ?`, append(args, page.Limit(), page.Offset())...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	items := make([]Inquiry, 0, page.Limit())
	for rows.Next() {
		var item Inquiry
		if err := rows.Scan(
			&item.ID, &item.MemberID, &item.MemberNickname, &item.MemberPhone, &item.MemberAvatarURL,
			&item.ContactName, &item.Phone, &item.ExpectedRegion, &item.Source, &item.Status,
			&item.ProcessedAt, &item.CreatedAt,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return items, total, nil
}

func (r *sqlRepository) UpdateStatus(ctx context.Context, id int64, status string, now time.Time) error {
	var processedAt any
	if status == StatusProcessed {
		processedAt = now
	}
	result, err := r.db.ExecContext(ctx, `UPDATE franchise_inquiries
		SET status = ?, processed_at = ?, updated_at = ? WHERE id = ?`, status, processedAt, now, id)
	if err != nil {
		return apperr.Internal(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if affected == 0 {
		return apperr.NotFound("加盟咨询不存在")
	}
	return nil
}

// Service validates public input and exposes headquarters reads.
type Service struct {
	repo    Repository
	sources SourceProvider
	now     func() time.Time
}

// NewService builds the franchise service.
func NewService(repo Repository, sources SourceProvider) *Service {
	return &Service{repo: repo, sources: sources, now: time.Now}
}

// Config returns the current source options.
func (s *Service) Config(ctx context.Context) (Config, error) {
	sources, err := s.sources.FranchiseInquirySources(ctx)
	if err != nil {
		return Config{}, err
	}
	hotline, err := s.sources.FranchiseHotline(ctx)
	if err != nil {
		return Config{}, err
	}
	return Config{Sources: sources, Hotline: hotline}, nil
}

// Create validates and persists a public inquiry.
func (s *Service) Create(ctx context.Context, memberID *int64, req CreateInquiryRequest) (Inquiry, error) {
	var validationErr error
	req.ContactName, validationErr = inputvalidation.PlainText(req.ContactName, inputvalidation.TextOptions{
		Label: "称呼", MinRunes: 1, MaxRunes: 50,
	})
	if validationErr != nil {
		return Inquiry{}, apperr.Invalid(validationErr.Error())
	}
	req.Phone, validationErr = inputvalidation.Phone(req.Phone)
	if validationErr != nil {
		return Inquiry{}, apperr.Invalid(validationErr.Error())
	}
	req.ExpectedRegion, validationErr = inputvalidation.PlainText(req.ExpectedRegion, inputvalidation.TextOptions{
		Label: "预期开设区域", MinRunes: 1, MaxRunes: 100,
	})
	if validationErr != nil {
		return Inquiry{}, apperr.Invalid(validationErr.Error())
	}
	req.Source = strings.TrimSpace(req.Source)
	sources, err := s.sources.FranchiseInquirySources(ctx)
	if err != nil {
		return Inquiry{}, err
	}
	validSource := false
	for _, source := range sources {
		if source == req.Source {
			validSource = true
			break
		}
	}
	if !validSource {
		return Inquiry{}, apperr.Invalid("请选择有效的信息渠道")
	}
	return s.repo.Create(ctx, memberID, req, s.now().UTC())
}

// List returns inquiries for the headquarters console.
func (s *Service) List(ctx context.Context, page httpx.Page, filter ListFilter) ([]Inquiry, int64, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Source = strings.TrimSpace(filter.Source)
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && filter.Status != StatusUnprocessed && filter.Status != StatusProcessed {
		return nil, 0, apperr.Invalid("加盟咨询状态不正确")
	}
	return s.repo.List(ctx, page, filter)
}

// UpdateStatus marks an inquiry processed or restores it to unprocessed.
func (s *Service) UpdateStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != StatusUnprocessed && status != StatusProcessed {
		return apperr.Invalid("加盟咨询状态不正确")
	}
	return s.repo.UpdateStatus(ctx, id, status, s.now().UTC())
}

// Handler exposes public create/config routes and the headquarters list.
type Handler struct{ svc *Service }

// NewHandler builds the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Config handles GET /mini/franchise-inquiries/config.
func (h *Handler) Config(c *gin.Context) {
	config, err := h.svc.Config(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, config)
}

// Create handles POST /mini/franchise-inquiries.
func (h *Handler) Create(c *gin.Context) {
	var req CreateInquiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	var memberID *int64
	if claims, ok := authn.FromContext(c); ok && claims.SubjectType == authn.SubjectMember {
		id := claims.SubjectID()
		if id > 0 {
			memberID = &id
		}
	}
	item, err := h.svc.Create(c.Request.Context(), memberID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, item)
}

// AdminList handles GET /admin/franchise-inquiries.
func (h *Handler) AdminList(c *gin.Context) {
	page := httpx.ParsePage(c)
	items, total, err := h.svc.List(c.Request.Context(), page, ListFilter{
		Keyword: c.Query("keyword"), Source: c.Query("source"), Status: c.Query("status"),
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, items, httpx.MetaFor(page, total))
}

// AdminUpdateStatus handles PATCH /admin/franchise-inquiries/:inquiryID/status.
func (h *Handler) AdminUpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("inquiryID"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, apperr.Invalid("加盟咨询 ID 不正确"))
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}
