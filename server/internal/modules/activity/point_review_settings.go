package activity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type PointReviewSettings struct {
	PointsDivisor          int64     `json:"pointsDivisor"`
	BelowBasePointsDivisor int64     `json:"belowBasePointsDivisor"`
	CoinPointsDivisor      int64     `json:"coinPointsDivisor"`
	Version                int64     `json:"version"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type UpdatePointReviewSettingsRequest struct {
	PointsDivisor          int64 `json:"pointsDivisor"`
	BelowBasePointsDivisor int64 `json:"belowBasePointsDivisor"`
	CoinPointsDivisor      int64 `json:"coinPointsDivisor"`
}

type PointReviewSettingsRepository interface {
	GetPointReviewSettings(ctx context.Context) (PointReviewSettings, error)
	UpdatePointReviewSettings(
		ctx context.Context,
		pointsDivisor, belowBasePointsDivisor, coinPointsDivisor, updatedBy int64,
		now time.Time,
	) (PointReviewSettings, error)
}

type sqlPointReviewSettingsRepository struct{ db *platdb.DB }

func NewPointReviewSettingsRepository(db *platdb.DB) PointReviewSettingsRepository {
	return &sqlPointReviewSettingsRepository{db: db}
}

func (r *sqlPointReviewSettingsRepository) GetPointReviewSettings(ctx context.Context) (PointReviewSettings, error) {
	var settings PointReviewSettings
	const q = `SELECT points_divisor, below_base_points_divisor, coin_points_divisor, version, updated_at
		FROM point_review_settings WHERE id = 1`
	err := r.db.QueryRowContext(ctx, q).Scan(
		&settings.PointsDivisor, &settings.BelowBasePointsDivisor, &settings.CoinPointsDivisor,
		&settings.Version, &settings.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PointReviewSettings{
			PointsDivisor: defaultPointsDivisor, BelowBasePointsDivisor: defaultBelowBasePointsDivisor,
			CoinPointsDivisor: defaultCoinPointsDivisor, Version: 1,
		}, nil
	}
	if err != nil {
		return PointReviewSettings{}, apperr.Internal(err)
	}
	return settings, nil
}

func (r *sqlPointReviewSettingsRepository) UpdatePointReviewSettings(
	ctx context.Context,
	pointsDivisor, belowBasePointsDivisor, coinPointsDivisor, updatedBy int64,
	now time.Time,
) (PointReviewSettings, error) {
	const q = `INSERT INTO point_review_settings
		(id, points_divisor, below_base_points_divisor, coin_points_divisor, version, updated_by, created_at, updated_at)
		VALUES (1, ?, ?, ?, 1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  points_divisor = VALUES(points_divisor),
		  below_base_points_divisor = VALUES(below_base_points_divisor),
		  coin_points_divisor = VALUES(coin_points_divisor),
		  version = version + 1,
		  updated_by = VALUES(updated_by),
		  updated_at = VALUES(updated_at)`
	if _, err := r.db.ExecContext(
		ctx, q, pointsDivisor, belowBasePointsDivisor, coinPointsDivisor, updatedBy, now, now,
	); err != nil {
		return PointReviewSettings{}, apperr.Internal(err)
	}
	return r.GetPointReviewSettings(ctx)
}

type PointReviewSettingsService struct {
	repo PointReviewSettingsRepository
	now  func() time.Time
}

func NewPointReviewSettingsService(repo PointReviewSettingsRepository) *PointReviewSettingsService {
	return &PointReviewSettingsService{repo: repo, now: time.Now}
}

func (s *PointReviewSettingsService) Get(ctx context.Context) (PointReviewSettings, error) {
	return s.repo.GetPointReviewSettings(ctx)
}

func (s *PointReviewSettingsService) Update(
	ctx context.Context,
	req UpdatePointReviewSettingsRequest,
	updatedBy int64,
) (PointReviewSettings, error) {
	if req.PointsDivisor <= 0 {
		return PointReviewSettings{}, apperr.Invalid("pointsDivisor must be greater than zero")
	}
	if req.BelowBasePointsDivisor <= 0 {
		return PointReviewSettings{}, apperr.Invalid("belowBasePointsDivisor must be greater than zero")
	}
	if req.CoinPointsDivisor <= 0 {
		return PointReviewSettings{}, apperr.Invalid("coinPointsDivisor must be greater than zero")
	}
	return s.repo.UpdatePointReviewSettings(
		ctx, req.PointsDivisor, req.BelowBasePointsDivisor, req.CoinPointsDivisor, updatedBy, s.now().UTC(),
	)
}

type PointReviewSettingsHandler struct{ svc *PointReviewSettingsService }

func NewPointReviewSettingsHandler(svc *PointReviewSettingsService) *PointReviewSettingsHandler {
	return &PointReviewSettingsHandler{svc: svc}
}

func (h *PointReviewSettingsHandler) Get(c *gin.Context) {
	settings, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, settings)
}

func (h *PointReviewSettingsHandler) Update(c *gin.Context) {
	var req UpdatePointReviewSettingsRequest
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
