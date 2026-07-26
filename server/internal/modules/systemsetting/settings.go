package systemsetting

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

const tableDefaultBackgroundKey = "table_default_background_url"

// GlobalSettings contains headquarters-level presentation defaults.
type GlobalSettings struct {
	TableDefaultBackgroundURL string     `json:"tableDefaultBackgroundUrl"`
	UpdatedAt                 *time.Time `json:"updatedAt,omitempty"`
}

// UpdateGlobalSettingsRequest is the writable global-settings payload.
type UpdateGlobalSettingsRequest struct {
	TableDefaultBackgroundURL string `json:"tableDefaultBackgroundUrl"`
}

// Repository persists headquarters-level settings.
type Repository interface {
	GetGlobalSettings(ctx context.Context) (GlobalSettings, error)
	UpdateGlobalSettings(ctx context.Context, tableDefaultBackgroundURL string, updatedBy int64, now time.Time) (GlobalSettings, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the SQL-backed settings repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) GetGlobalSettings(ctx context.Context) (GlobalSettings, error) {
	var settings GlobalSettings
	var updatedAt time.Time
	const q = `SELECT setting_value, updated_at FROM system_settings WHERE setting_key = ?`
	err := r.db.QueryRowContext(ctx, q, tableDefaultBackgroundKey).Scan(
		&settings.TableDefaultBackgroundURL, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return GlobalSettings{}, nil
	}
	if err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	settings.UpdatedAt = &updatedAt
	return settings, nil
}

func (r *sqlRepository) UpdateGlobalSettings(
	ctx context.Context,
	tableDefaultBackgroundURL string,
	updatedBy int64,
	now time.Time,
) (GlobalSettings, error) {
	const q = `INSERT INTO system_settings
		(setting_key, setting_value, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  setting_value = VALUES(setting_value),
		  updated_by = VALUES(updated_by),
		  updated_at = VALUES(updated_at)`
	if _, err := r.db.ExecContext(
		ctx, q, tableDefaultBackgroundKey, tableDefaultBackgroundURL, updatedBy, now, now,
	); err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	return r.GetGlobalSettings(ctx)
}

// Service validates and exposes global settings to admin and public read models.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService builds the global settings service.
func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

// Get returns the complete headquarters settings view.
func (s *Service) Get(ctx context.Context) (GlobalSettings, error) {
	return s.repo.GetGlobalSettings(ctx)
}

// TableDefaultBackgroundURL provides the mini-program table fallback.
func (s *Service) TableDefaultBackgroundURL(ctx context.Context) (string, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return "", err
	}
	return settings.TableDefaultBackgroundURL, nil
}

// Update validates and persists the headquarters settings.
func (s *Service) Update(ctx context.Context, req UpdateGlobalSettingsRequest, updatedBy int64) (GlobalSettings, error) {
	backgroundURL := strings.TrimSpace(req.TableDefaultBackgroundURL)
	if err := validateHTTPURL(backgroundURL); err != nil {
		return GlobalSettings{}, err
	}
	return s.repo.UpdateGlobalSettings(ctx, backgroundURL, updatedBy, s.now().UTC())
}

func validateHTTPURL(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > 2048 {
		return apperr.Invalid("tableDefaultBackgroundUrl is too long")
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperr.Invalid("tableDefaultBackgroundUrl must be a valid HTTP or HTTPS URL")
	}
	return nil
}

// Handler exposes headquarters global settings.
type Handler struct{ svc *Service }

// NewHandler builds the global-settings handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Get handles GET /admin/global-settings.
func (h *Handler) Get(c *gin.Context) {
	settings, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, settings)
}

// Update handles PUT /admin/global-settings.
func (h *Handler) Update(c *gin.Context) {
	var req UpdateGlobalSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	claims := authn.MustFromContext(c)
	settings, err := h.svc.Update(c.Request.Context(), req, claims.SubjectID())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, settings)
}
