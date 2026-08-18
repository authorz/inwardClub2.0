package systemsetting

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

const (
	firstRechargeDoublePointsEnabledKey    = "first_recharge_double_points_enabled"
	rechargeDoublePointsThresholdAmountKey = "recharge_double_points_threshold_amount"
	franchiseInquirySourcesKey             = "franchise_inquiry_sources"
	franchiseHotlineKey                    = "franchise_hotline"
	phoneChangeIntervalDaysKey             = "phone_change_interval_days"
	defaultRechargeDoublePointsThreshold   = int64(1000)
	defaultPhoneChangeIntervalDays         = 30
)

var defaultFranchiseInquirySources = []string{"美团", "抖音", "小红书", "店员", "微信小程序"}

// GlobalSettings contains headquarters-level business settings.
type GlobalSettings struct {
	FirstRechargeDoublePointsEnabled    bool       `json:"firstRechargeDoublePointsEnabled"`
	RechargeDoublePointsThresholdAmount int64      `json:"rechargeDoublePointsThresholdAmount"`
	FranchiseInquirySources             []string   `json:"franchiseInquirySources"`
	FranchiseHotline                    string     `json:"franchiseHotline"`
	PhoneChangeIntervalDays             int        `json:"phoneChangeIntervalDays"`
	UpdatedAt                           *time.Time `json:"updatedAt,omitempty"`
}

// UpdateGlobalSettingsRequest is the writable global-settings payload.
type UpdateGlobalSettingsRequest struct {
	FirstRechargeDoublePointsEnabled    bool     `json:"firstRechargeDoublePointsEnabled"`
	RechargeDoublePointsThresholdAmount int64    `json:"rechargeDoublePointsThresholdAmount"`
	FranchiseInquirySources             []string `json:"franchiseInquirySources"`
	FranchiseHotline                    string   `json:"franchiseHotline"`
	PhoneChangeIntervalDays             int      `json:"phoneChangeIntervalDays"`
}

// Repository persists headquarters-level settings.
type Repository interface {
	GetGlobalSettings(ctx context.Context) (GlobalSettings, error)
	UpdateGlobalSettings(ctx context.Context, settings GlobalSettings, updatedBy int64, now time.Time) (GlobalSettings, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the SQL-backed settings repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) GetGlobalSettings(ctx context.Context) (GlobalSettings, error) {
	settings := GlobalSettings{
		RechargeDoublePointsThresholdAmount: defaultRechargeDoublePointsThreshold,
		FranchiseInquirySources:             append([]string(nil), defaultFranchiseInquirySources...),
		PhoneChangeIntervalDays:             defaultPhoneChangeIntervalDays,
	}
	const q = `SELECT setting_key, setting_value, updated_at FROM system_settings
		WHERE setting_key IN (?, ?, ?, ?, ?)`
	rows, err := r.db.QueryContext(
		ctx, q,
		firstRechargeDoublePointsEnabledKey,
		rechargeDoublePointsThresholdAmountKey,
		franchiseInquirySourcesKey,
		franchiseHotlineKey,
		phoneChangeIntervalDaysKey,
	)
	if err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		var updatedAt time.Time
		if err := rows.Scan(&key, &value, &updatedAt); err != nil {
			return GlobalSettings{}, apperr.Internal(err)
		}
		switch key {
		case firstRechargeDoublePointsEnabledKey:
			settings.FirstRechargeDoublePointsEnabled, _ = strconv.ParseBool(value)
		case rechargeDoublePointsThresholdAmountKey:
			if threshold, err := strconv.ParseInt(value, 10, 64); err == nil && threshold > 0 {
				settings.RechargeDoublePointsThresholdAmount = threshold
			}
		case franchiseInquirySourcesKey:
			var sources []string
			if err := json.Unmarshal([]byte(value), &sources); err == nil && len(sources) > 0 {
				settings.FranchiseInquirySources = sources
			}
		case franchiseHotlineKey:
			settings.FranchiseHotline = value
		case phoneChangeIntervalDaysKey:
			if days, err := strconv.Atoi(value); err == nil && days > 0 {
				settings.PhoneChangeIntervalDays = days
			}
		}
		if settings.UpdatedAt == nil || updatedAt.After(*settings.UpdatedAt) {
			t := updatedAt
			settings.UpdatedAt = &t
		}
	}
	if err := rows.Err(); err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	return settings, nil
}

func (r *sqlRepository) UpdateGlobalSettings(
	ctx context.Context,
	settings GlobalSettings,
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
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		sourcesJSON, err := json.Marshal(settings.FranchiseInquirySources)
		if err != nil {
			return apperr.Internal(err)
		}
		values := []struct {
			key   string
			value string
		}{
			{firstRechargeDoublePointsEnabledKey, strconv.FormatBool(settings.FirstRechargeDoublePointsEnabled)},
			{rechargeDoublePointsThresholdAmountKey, strconv.FormatInt(settings.RechargeDoublePointsThresholdAmount, 10)},
			{franchiseInquirySourcesKey, string(sourcesJSON)},
			{franchiseHotlineKey, settings.FranchiseHotline},
			{phoneChangeIntervalDaysKey, strconv.Itoa(settings.PhoneChangeIntervalDays)},
		}
		for _, setting := range values {
			if _, err := tx.ExecContext(ctx, q, setting.key, setting.value, updatedBy, now, now); err != nil {
				return apperr.Internal(err)
			}
		}
		return nil
	})
	if err != nil {
		return GlobalSettings{}, err
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

// FranchiseInquirySources provides the configured public form options.
func (s *Service) FranchiseInquirySources(ctx context.Context) ([]string, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), settings.FranchiseInquirySources...), nil
}

// FranchiseHotline provides the headquarters hotline shown above the public form.
func (s *Service) FranchiseHotline(ctx context.Context) (string, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return "", err
	}
	return settings.FranchiseHotline, nil
}

// PhoneChangeIntervalDays returns the global cooldown used by member phone changes.
func (s *Service) PhoneChangeIntervalDays(ctx context.Context) (int, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return 0, err
	}
	return settings.PhoneChangeIntervalDays, nil
}

// Update validates and persists the headquarters settings.
func (s *Service) Update(ctx context.Context, req UpdateGlobalSettingsRequest, updatedBy int64) (GlobalSettings, error) {
	if req.RechargeDoublePointsThresholdAmount <= 0 {
		return GlobalSettings{}, apperr.Invalid("满额双倍积分门槛必须大于 0")
	}
	if req.PhoneChangeIntervalDays < 1 || req.PhoneChangeIntervalDays > 3650 {
		return GlobalSettings{}, apperr.Invalid("手机号更换间隔必须在 1 到 3650 天之间")
	}
	hotline := strings.TrimSpace(req.FranchiseHotline)
	if hotline != "" && (len(hotline) < 6 || len(hotline) > 32) {
		return GlobalSettings{}, apperr.Invalid("请填写有效的加盟热线")
	}
	var sources []string
	if req.FranchiseInquirySources == nil {
		current, err := s.repo.GetGlobalSettings(ctx)
		if err != nil {
			return GlobalSettings{}, err
		}
		sources = current.FranchiseInquirySources
		if len(sources) == 0 {
			sources = append([]string(nil), defaultFranchiseInquirySources...)
		}
	} else {
		var err error
		sources, err = normalizeFranchiseInquirySources(req.FranchiseInquirySources)
		if err != nil {
			return GlobalSettings{}, err
		}
	}
	return s.repo.UpdateGlobalSettings(ctx, GlobalSettings{
		FirstRechargeDoublePointsEnabled:    req.FirstRechargeDoublePointsEnabled,
		RechargeDoublePointsThresholdAmount: req.RechargeDoublePointsThresholdAmount,
		FranchiseInquirySources:             sources,
		FranchiseHotline:                    hotline,
		PhoneChangeIntervalDays:             req.PhoneChangeIntervalDays,
	}, updatedBy, s.now().UTC())
}

func normalizeFranchiseInquirySources(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, apperr.Invalid("加盟咨询信息渠道至少保留一项")
	}
	if len(raw) > 20 {
		return nil, apperr.Invalid("加盟咨询信息渠道最多可配置 20 项")
	}
	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, apperr.Invalid("加盟咨询信息渠道不能为空")
		}
		if len([]rune(item)) > 20 {
			return nil, apperr.Invalid("加盟咨询信息渠道名称不能超过 20 个字")
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, apperr.Invalid("加盟咨询信息渠道至少保留一项")
	}
	return result, nil
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
