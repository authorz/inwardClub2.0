package systemsetting

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

const timedLowSpendRuleKey = "timed_low_spend_reward"

var reservationRuleLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type StoreLowSpendRuleConfig struct {
	ReservationCutoff string `json:"reservationCutoff"`
	ConsumptionCutoff string `json:"consumptionCutoff"`
	MinimumAmountCent int64  `json:"minimumAmountCent"`
	RewardPoints      int64  `json:"rewardPoints"`
}

type StoreLowSpendRuleView struct {
	StoreID           int64      `json:"storeId"`
	StoreName         string     `json:"storeName"`
	Configured        bool       `json:"configured"`
	Enabled           bool       `json:"enabled"`
	ReservationCutoff string     `json:"reservationCutoff"`
	ConsumptionCutoff string     `json:"consumptionCutoff"`
	MinimumAmount     int64      `json:"minimumAmount"`
	RewardPoints      int64      `json:"rewardPoints"`
	UpdatedAt         *time.Time `json:"updatedAt,omitempty"`
}

type ReservationAvailabilityView struct {
	Reservable        bool       `json:"reservable"`
	ReservationCutoff string     `json:"reservationCutoff"`
	CutoffAt          *time.Time `json:"cutoffAt,omitempty"`
	ServerTime        time.Time  `json:"serverTime"`
	UnavailableReason string     `json:"unavailableReason,omitempty"`
}

type UpdateStoreLowSpendRuleRequest struct {
	Enabled           bool   `json:"enabled"`
	ReservationCutoff string `json:"reservationCutoff"`
	ConsumptionCutoff string `json:"consumptionCutoff"`
	MinimumAmount     int64  `json:"minimumAmount"`
	RewardPoints      int64  `json:"rewardPoints"`
}

type StoreLowSpendRuleRepository interface {
	List(ctx context.Context, keyword string, page httpx.Page) ([]StoreLowSpendRuleView, int64, error)
	Get(ctx context.Context, storeID int64) (StoreLowSpendRuleView, error)
	Upsert(ctx context.Context, storeID int64, config StoreLowSpendRuleConfig, enabled bool, now time.Time) (StoreLowSpendRuleView, error)
	Delete(ctx context.Context, storeID int64) error
}

type storeLowSpendRuleRepository struct{ db *platdb.DB }

func NewStoreLowSpendRuleRepository(db *platdb.DB) StoreLowSpendRuleRepository {
	return &storeLowSpendRuleRepository{db: db}
}

func defaultStoreLowSpendRule(storeID int64, storeName string) StoreLowSpendRuleView {
	return StoreLowSpendRuleView{
		StoreID: storeID, StoreName: storeName, ReservationCutoff: "20:00",
		ConsumptionCutoff: "20:30", MinimumAmount: 88, RewardPoints: 2000,
	}
}

func scanStoreLowSpendRule(row interface{ Scan(...any) error }) (StoreLowSpendRuleView, error) {
	var storeID int64
	var storeName string
	var ruleID sql.NullInt64
	var raw []byte
	var enabled sql.NullBool
	var updatedAt sql.NullTime
	if err := row.Scan(&storeID, &storeName, &ruleID, &raw, &enabled, &updatedAt); err != nil {
		return StoreLowSpendRuleView{}, err
	}
	view := defaultStoreLowSpendRule(storeID, storeName)
	if !ruleID.Valid {
		return view, nil
	}
	var config StoreLowSpendRuleConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return StoreLowSpendRuleView{}, apperr.Internal(err)
	}
	view.Configured = true
	view.Enabled = enabled.Bool
	view.ReservationCutoff = config.ReservationCutoff
	view.ConsumptionCutoff = config.ConsumptionCutoff
	view.MinimumAmount = config.MinimumAmountCent / 100
	view.RewardPoints = config.RewardPoints
	if updatedAt.Valid {
		updated := updatedAt.Time
		view.UpdatedAt = &updated
	}
	return view, nil
}

const storeLowSpendRuleSelect = `SELECT s.id, s.name, sr.id, sr.config_json, sr.enabled, sr.updated_at
	FROM stores s LEFT JOIN store_rules sr
	  ON sr.store_id = s.id AND sr.rule_key = 'timed_low_spend_reward'`

func (r *storeLowSpendRuleRepository) List(ctx context.Context, keyword string, page httpx.Page) ([]StoreLowSpendRuleView, int64, error) {
	where := ""
	var args []any
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = " WHERE s.name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stores s"+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	queryArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := r.db.QueryContext(ctx, storeLowSpendRuleSelect+where+" ORDER BY s.id DESC LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	views := make([]StoreLowSpendRuleView, 0, page.Limit())
	for rows.Next() {
		view, err := scanStoreLowSpendRule(rows)
		if err != nil {
			return nil, 0, err
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return views, total, nil
}

func (r *storeLowSpendRuleRepository) Get(ctx context.Context, storeID int64) (StoreLowSpendRuleView, error) {
	view, err := scanStoreLowSpendRule(r.db.QueryRowContext(ctx, storeLowSpendRuleSelect+" WHERE s.id = ?", storeID))
	if errors.Is(err, sql.ErrNoRows) {
		return StoreLowSpendRuleView{}, apperr.NotFound("门店不存在")
	}
	if err != nil {
		return StoreLowSpendRuleView{}, apperr.Internal(err)
	}
	return view, nil
}

func (r *storeLowSpendRuleRepository) Upsert(ctx context.Context, storeID int64, config StoreLowSpendRuleConfig, enabled bool, now time.Time) (StoreLowSpendRuleView, error) {
	if _, err := r.Get(ctx, storeID); err != nil {
		return StoreLowSpendRuleView{}, err
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return StoreLowSpendRuleView{}, apperr.Internal(err)
	}
	const query = `INSERT INTO store_rules (store_id, rule_key, config_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE config_json = VALUES(config_json), enabled = VALUES(enabled), updated_at = VALUES(updated_at)`
	if _, err := r.db.ExecContext(ctx, query, storeID, timedLowSpendRuleKey, raw, enabled, now, now); err != nil {
		return StoreLowSpendRuleView{}, apperr.Internal(err)
	}
	return r.Get(ctx, storeID)
}

func (r *storeLowSpendRuleRepository) Delete(ctx context.Context, storeID int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM store_rules WHERE store_id = ? AND rule_key = ?`, storeID, timedLowSpendRuleKey)
	if err != nil {
		return apperr.Internal(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return apperr.NotFound("该门店尚未配置预约低消奖励")
	}
	return nil
}

type StoreLowSpendRuleService struct {
	repo StoreLowSpendRuleRepository
	now  func() time.Time
}

func NewStoreLowSpendRuleService(repo StoreLowSpendRuleRepository) *StoreLowSpendRuleService {
	return &StoreLowSpendRuleService{repo: repo, now: time.Now}
}

func (s *StoreLowSpendRuleService) List(ctx context.Context, keyword string, page httpx.Page) ([]StoreLowSpendRuleView, int64, error) {
	return s.repo.List(ctx, keyword, page)
}

func (s *StoreLowSpendRuleService) Get(ctx context.Context, storeID int64) (StoreLowSpendRuleView, error) {
	return s.repo.Get(ctx, storeID)
}

func (s *StoreLowSpendRuleService) ReservationAvailability(ctx context.Context, storeID int64) (ReservationAvailabilityView, error) {
	rule, err := s.repo.Get(ctx, storeID)
	if err != nil {
		return ReservationAvailabilityView{}, err
	}
	now := s.now().In(reservationRuleLocation)
	cutoffClock, parseErr := time.Parse("15:04", rule.ReservationCutoff)
	if parseErr != nil {
		return ReservationAvailabilityView{
			ServerTime: now, UnavailableReason: "门店预约时间配置异常",
		}, nil
	}
	cutoffAt := time.Date(
		now.Year(), now.Month(), now.Day(), cutoffClock.Hour(), cutoffClock.Minute(), 0, 0,
		reservationRuleLocation,
	)
	view := ReservationAvailabilityView{
		Reservable:        rule.Configured && rule.Enabled && now.Before(cutoffAt),
		ReservationCutoff: rule.ReservationCutoff,
		CutoffAt:          &cutoffAt,
		ServerTime:        now,
	}
	if !rule.Configured || !rule.Enabled {
		view.UnavailableReason = "门店暂未开放预约"
	} else if !view.Reservable {
		view.UnavailableReason = "今日预约已截止"
	}
	return view, nil
}

func (s *StoreLowSpendRuleService) ValidateReservationTime(ctx context.Context, storeID int64) error {
	availability, err := s.ReservationAvailability(ctx, storeID)
	if err != nil {
		return err
	}
	if availability.Reservable {
		return nil
	}
	return apperr.Conflict(availability.UnavailableReason)
}

func (s *StoreLowSpendRuleService) Update(ctx context.Context, storeID int64, req UpdateStoreLowSpendRuleRequest) (StoreLowSpendRuleView, error) {
	reservation := strings.TrimSpace(req.ReservationCutoff)
	consumption := strings.TrimSpace(req.ConsumptionCutoff)
	reservationTime, err := time.Parse("15:04", reservation)
	if err != nil {
		return StoreLowSpendRuleView{}, apperr.Invalid("预约或候桌截止时间格式不正确")
	}
	consumptionTime, err := time.Parse("15:04", consumption)
	if err != nil {
		return StoreLowSpendRuleView{}, apperr.Invalid("低消截止时间格式不正确")
	}
	if !reservationTime.Before(consumptionTime) {
		return StoreLowSpendRuleView{}, apperr.Invalid("低消截止时间必须晚于预约或候桌截止时间")
	}
	if req.MinimumAmount < 1 || req.MinimumAmount > 1000000 {
		return StoreLowSpendRuleView{}, apperr.Invalid("低消金额必须在 1 至 1000000 元之间")
	}
	if req.RewardPoints < 1 || req.RewardPoints > 100000000 {
		return StoreLowSpendRuleView{}, apperr.Invalid("赠送积分必须在 1 至 100000000 之间")
	}
	return s.repo.Upsert(ctx, storeID, StoreLowSpendRuleConfig{
		ReservationCutoff: reservation, ConsumptionCutoff: consumption,
		MinimumAmountCent: req.MinimumAmount * 100, RewardPoints: req.RewardPoints,
	}, req.Enabled, s.now().UTC())
}

func (s *StoreLowSpendRuleService) Delete(ctx context.Context, storeID int64) error {
	return s.repo.Delete(ctx, storeID)
}

type StoreLowSpendRuleHandler struct{ svc *StoreLowSpendRuleService }

func NewStoreLowSpendRuleHandler(svc *StoreLowSpendRuleService) *StoreLowSpendRuleHandler {
	return &StoreLowSpendRuleHandler{svc: svc}
}

func (h *StoreLowSpendRuleHandler) AdminList(c *gin.Context) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.List(c.Request.Context(), c.Query("storeName"), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func (h *StoreLowSpendRuleHandler) AdminUpdate(c *gin.Context) {
	storeID, err := pathPositiveID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	h.update(c, storeID)
}

func (h *StoreLowSpendRuleHandler) AdminDelete(c *gin.Context) {
	storeID, err := pathPositiveID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), storeID); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *StoreLowSpendRuleHandler) StoreGet(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	view, err := h.svc.Get(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *StoreLowSpendRuleHandler) MiniReservationAvailability(c *gin.Context) {
	storeID, err := pathPositiveID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.ReservationAvailability(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *StoreLowSpendRuleHandler) StoreUpdate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	h.update(c, storeID)
}

func (h *StoreLowSpendRuleHandler) update(c *gin.Context, storeID int64) {
	var req UpdateStoreLowSpendRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	view, err := h.svc.Update(c.Request.Context(), storeID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func pathPositiveID(c *gin.Context, name string) (int64, error) {
	value := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("门店 ID 不正确")
	}
	return id, nil
}
