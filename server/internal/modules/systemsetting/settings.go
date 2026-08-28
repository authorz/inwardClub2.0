package systemsetting

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
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
	firstRechargeDoublePointsEnabledKey    = "first_recharge_double_points_enabled"
	rechargeDoublePointsThresholdAmountKey = "recharge_double_points_threshold_amount"
	rechargeNoticeKey                      = "recharge_notice"
	franchiseInquirySourcesKey             = "franchise_inquiry_sources"
	franchiseHotlineKey                    = "franchise_hotline"
	phoneChangeIntervalDaysKey             = "phone_change_interval_days"
	printerDeveloperAccountKey             = "printer_developer_account"
	printerDeveloperKeyKey                 = "printer_developer_key"
	printerAPIURLKey                       = "printer_api_url"
	defaultRechargeDoublePointsThreshold   = int64(1000)
	defaultPhoneChangeIntervalDays         = 30
	defaultRechargeNotice                  = "新用户首充积分赠送双倍，充值一千及以上都赠送双倍积分，不与新用户首充赠送双倍同享。"
	defaultPrinterAPIURL                   = "https://open.xpyun.net/api/openapi/xprinter"
)

var defaultFranchiseInquirySources = []string{"美团", "抖音", "小红书", "店员", "微信小程序"}

// GlobalSettings contains headquarters-level business settings.
type GlobalSettings struct {
	FirstRechargeDoublePointsEnabled    bool                  `json:"firstRechargeDoublePointsEnabled"`
	RechargeDoublePointsThresholdAmount int64                 `json:"rechargeDoublePointsThresholdAmount"`
	RechargeNotice                      string                `json:"rechargeNotice"`
	FranchiseInquirySources             []string              `json:"franchiseInquirySources"`
	FranchiseHotline                    string                `json:"franchiseHotline"`
	PhoneChangeIntervalDays             int                   `json:"phoneChangeIntervalDays"`
	PrinterDeveloperAccount             string                `json:"printerDeveloperAccount"`
	PrinterDeveloperKey                 string                `json:"-"`
	PrinterDeveloperKeyConfigured       bool                  `json:"printerDeveloperKeyConfigured"`
	PrinterAPIURL                       string                `json:"printerApiUrl"`
	GiftCouponUsageRules                []GiftCouponUsageRule `json:"giftCouponUsageRules"`
	UpdatedAt                           *time.Time            `json:"updatedAt,omitempty"`
}

// GiftCouponUsageRule is an independent headquarters rule for gifted coupons.
// A nil DailyLimit explicitly means unrestricted use. Purchased coupon
// products bypass these rules before any limit is evaluated.
type GiftCouponUsageRule struct {
	CouponCategoryID int64 `json:"couponCategoryId"`
	DailyLimit       *int  `json:"dailyLimit"`
}

// UpdateGlobalSettingsRequest is the writable global-settings payload.
type UpdateGlobalSettingsRequest struct {
	FirstRechargeDoublePointsEnabled    bool                   `json:"firstRechargeDoublePointsEnabled"`
	RechargeDoublePointsThresholdAmount int64                  `json:"rechargeDoublePointsThresholdAmount"`
	RechargeNotice                      string                 `json:"rechargeNotice"`
	FranchiseInquirySources             []string               `json:"franchiseInquirySources"`
	FranchiseHotline                    string                 `json:"franchiseHotline"`
	PhoneChangeIntervalDays             int                    `json:"phoneChangeIntervalDays"`
	PrinterDeveloperAccount             *string                `json:"printerDeveloperAccount"`
	PrinterDeveloperKey                 *string                `json:"printerDeveloperKey"`
	PrinterAPIURL                       *string                `json:"printerApiUrl"`
	GiftCouponUsageRules                *[]GiftCouponUsageRule `json:"giftCouponUsageRules"`
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
		RechargeNotice:                      defaultRechargeNotice,
		FranchiseInquirySources:             append([]string(nil), defaultFranchiseInquirySources...),
		PhoneChangeIntervalDays:             defaultPhoneChangeIntervalDays,
		PrinterAPIURL:                       defaultPrinterAPIURL,
		GiftCouponUsageRules:                make([]GiftCouponUsageRule, 0),
	}
	const q = `SELECT setting_key, setting_value, updated_at FROM system_settings
		WHERE setting_key IN (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	rows, err := r.db.QueryContext(
		ctx, q,
		firstRechargeDoublePointsEnabledKey,
		rechargeDoublePointsThresholdAmountKey,
		rechargeNoticeKey,
		franchiseInquirySourcesKey,
		franchiseHotlineKey,
		phoneChangeIntervalDaysKey,
		printerDeveloperAccountKey,
		printerDeveloperKeyKey,
		printerAPIURLKey,
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
		case rechargeNoticeKey:
			settings.RechargeNotice = value
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
		case printerDeveloperAccountKey:
			settings.PrinterDeveloperAccount = value
		case printerDeveloperKeyKey:
			settings.PrinterDeveloperKey = value
		case printerAPIURLKey:
			if value != "" {
				settings.PrinterAPIURL = value
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
	ruleRows, err := r.db.QueryContext(ctx, `SELECT coupon_category_id, daily_limit
		FROM gift_coupon_usage_rules ORDER BY coupon_category_id`)
	if err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var categoryID int64
		var dailyLimit sql.NullInt64
		if err := ruleRows.Scan(&categoryID, &dailyLimit); err != nil {
			return GlobalSettings{}, apperr.Internal(err)
		}
		var limit *int
		if dailyLimit.Valid {
			value := int(dailyLimit.Int64)
			limit = &value
		}
		settings.GiftCouponUsageRules = append(settings.GiftCouponUsageRules, GiftCouponUsageRule{
			CouponCategoryID: categoryID,
			DailyLimit:       limit,
		})
	}
	if err := ruleRows.Err(); err != nil {
		return GlobalSettings{}, apperr.Internal(err)
	}
	settings.PrinterDeveloperKeyConfigured = settings.PrinterDeveloperKey != ""
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
			{rechargeNoticeKey, settings.RechargeNotice},
			{franchiseInquirySourcesKey, string(sourcesJSON)},
			{franchiseHotlineKey, settings.FranchiseHotline},
			{phoneChangeIntervalDaysKey, strconv.Itoa(settings.PhoneChangeIntervalDays)},
			{printerDeveloperAccountKey, settings.PrinterDeveloperAccount},
			{printerDeveloperKeyKey, settings.PrinterDeveloperKey},
			{printerAPIURLKey, settings.PrinterAPIURL},
		}
		for _, setting := range values {
			if _, err := tx.ExecContext(ctx, q, setting.key, setting.value, updatedBy, now, now); err != nil {
				return apperr.Internal(err)
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM gift_coupon_usage_rules`); err != nil {
			return apperr.Internal(err)
		}
		for _, rule := range settings.GiftCouponUsageRules {
			if _, err := tx.ExecContext(ctx, `INSERT INTO gift_coupon_usage_rules
				(coupon_category_id, daily_limit, updated_by, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?)`,
				rule.CouponCategoryID, rule.DailyLimit, updatedBy, now, now,
			); err != nil {
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

// RechargeNotice provides the configurable copy shown in the mini-program recharge sheet.
func (s *Service) RechargeNotice(ctx context.Context) (string, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return "", err
	}
	return settings.RechargeNotice, nil
}

// PrinterProviderSettings returns the unmasked shared Xpyun account used only
// by the server-side printer adapter. Console JSON never exposes the key.
func (s *Service) PrinterProviderSettings(ctx context.Context) (string, string, string, error) {
	settings, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return "", "", "", err
	}
	return settings.PrinterDeveloperAccount, settings.PrinterDeveloperKey, settings.PrinterAPIURL, nil
}

// Update validates and persists the headquarters settings.
func (s *Service) Update(ctx context.Context, req UpdateGlobalSettingsRequest, updatedBy int64) (GlobalSettings, error) {
	if req.RechargeDoublePointsThresholdAmount <= 0 {
		return GlobalSettings{}, apperr.Invalid("满额双倍积分门槛必须大于 0")
	}
	if req.PhoneChangeIntervalDays < 1 || req.PhoneChangeIntervalDays > 3650 {
		return GlobalSettings{}, apperr.Invalid("手机号更换间隔必须在 1 到 3650 天之间")
	}
	rechargeNotice, err := inputvalidation.PlainText(req.RechargeNotice, inputvalidation.TextOptions{
		Label: "充值弹窗提示", MaxRunes: 200, AllowEmpty: true,
	})
	if err != nil {
		return GlobalSettings{}, apperr.Invalid(err.Error())
	}
	hotline := strings.TrimSpace(req.FranchiseHotline)
	if hotline != "" && (len(hotline) < 6 || len(hotline) > 32) {
		return GlobalSettings{}, apperr.Invalid("请填写有效的加盟热线")
	}
	current, err := s.repo.GetGlobalSettings(ctx)
	if err != nil {
		return GlobalSettings{}, err
	}
	rules := current.GiftCouponUsageRules
	if req.GiftCouponUsageRules != nil {
		rules, err = normalizeGiftCouponUsageRules(*req.GiftCouponUsageRules)
		if err != nil {
			return GlobalSettings{}, err
		}
	}
	var sources []string
	if req.FranchiseInquirySources == nil {
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
	printerAccount := current.PrinterDeveloperAccount
	if req.PrinterDeveloperAccount != nil {
		printerAccount = strings.TrimSpace(*req.PrinterDeveloperAccount)
	}
	printerKey := current.PrinterDeveloperKey
	if req.PrinterDeveloperKey != nil && strings.TrimSpace(*req.PrinterDeveloperKey) != "" {
		printerKey = strings.TrimSpace(*req.PrinterDeveloperKey)
	}
	printerAPIURL := current.PrinterAPIURL
	if req.PrinterAPIURL != nil {
		printerAPIURL = strings.TrimRight(strings.TrimSpace(*req.PrinterAPIURL), "/")
	}
	if printerAPIURL == "" {
		printerAPIURL = defaultPrinterAPIURL
	}
	parsedPrinterURL, err := url.ParseRequestURI(printerAPIURL)
	if err != nil || (parsedPrinterURL.Scheme != "https" && parsedPrinterURL.Scheme != "http") || parsedPrinterURL.Host == "" {
		return GlobalSettings{}, apperr.Invalid("请填写有效的打印机接口 URL")
	}
	if (printerAccount == "") != (printerKey == "") {
		return GlobalSettings{}, apperr.Invalid("打印机开发者账号和开发者密钥必须同时配置")
	}
	return s.repo.UpdateGlobalSettings(ctx, GlobalSettings{
		FirstRechargeDoublePointsEnabled:    req.FirstRechargeDoublePointsEnabled,
		RechargeDoublePointsThresholdAmount: req.RechargeDoublePointsThresholdAmount,
		RechargeNotice:                      rechargeNotice,
		FranchiseInquirySources:             sources,
		FranchiseHotline:                    hotline,
		PhoneChangeIntervalDays:             req.PhoneChangeIntervalDays,
		PrinterDeveloperAccount:             printerAccount,
		PrinterDeveloperKey:                 printerKey,
		PrinterDeveloperKeyConfigured:       printerKey != "",
		PrinterAPIURL:                       printerAPIURL,
		GiftCouponUsageRules:                rules,
	}, updatedBy, s.now().UTC())
}

func normalizeGiftCouponUsageRules(raw []GiftCouponUsageRule) ([]GiftCouponUsageRule, error) {
	seen := make(map[int64]struct{}, len(raw))
	result := make([]GiftCouponUsageRule, 0, len(raw))
	for _, rule := range raw {
		if rule.CouponCategoryID <= 0 {
			return nil, apperr.Invalid("请选择正确的券类型")
		}
		if _, exists := seen[rule.CouponCategoryID]; exists {
			return nil, apperr.Invalid("同一券类型不能重复配置")
		}
		seen[rule.CouponCategoryID] = struct{}{}
		if rule.DailyLimit != nil && (*rule.DailyLimit < 1 || *rule.DailyLimit > 999) {
			return nil, apperr.Invalid("赠送券每日使用上限必须在 1 到 999 之间")
		}
		result = append(result, rule)
	}
	return result, nil
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
