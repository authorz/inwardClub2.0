// Package referral implements the configurable invitation reward policy.
// Monetary inputs stay in integer cents and invitation commission is accrued
// with sub-coin precision before whole coins reach the wallet.
package referral

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const RuleKey = "invite_reward"

const (
	maxFirstRewardCoins  = int64(1_000_000)
	maxFirstRewardPoints = int64(100_000_000)
	maxRateBasisPoints   = int64(10_000)
)

// Config is rule_definitions.config_json for the global invite_reward rule.
// One basis point is 0.01%; 1000 basis points is 10%.
type Config struct {
	SchemaVersion             int   `json:"schemaVersion,omitempty"`
	FirstLowSpendRewardCoins  int64 `json:"firstLowSpendRewardCoins"`
	FirstLowSpendRewardPoints int64 `json:"firstLowSpendRewardPoints"`
	CommissionRateBasisPoints int64 `json:"commissionRateBasisPoints"`
}

// Rule is the active policy version plus its validated config.
type Rule struct {
	Version int
	Config  Config
}

// ConfigView is the member-facing effective invitation policy. Disabled rules
// intentionally expose zero values so the mini program never promises an
// inactive reward.
type ConfigView struct {
	Enabled                   bool  `json:"enabled"`
	FirstLowSpendRewardCoins  int64 `json:"firstLowSpendRewardCoins"`
	FirstLowSpendRewardPoints int64 `json:"firstLowSpendRewardPoints"`
	CommissionRateBasisPoints int64 `json:"commissionRateBasisPoints"`
	RewardedMembers           int64 `json:"rewardedMembers"`
	CumulativeRewardCoins     int64 `json:"cumulativeRewardCoins"`
	CumulativeRewardPoints    int64 `json:"cumulativeRewardPoints"`
}

// ParseConfig validates the admin-authored invite reward document.
func ParseConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, errors.New("邀请奖励配置不是有效 JSON")
	}
	if cfg.FirstLowSpendRewardCoins < 0 || cfg.FirstLowSpendRewardCoins > maxFirstRewardCoins {
		return Config{}, errors.New("首次低消奖励金币必须在 0 至 1000000 之间")
	}
	if cfg.FirstLowSpendRewardPoints < 0 || cfg.FirstLowSpendRewardPoints > maxFirstRewardPoints {
		return Config{}, errors.New("首次低消奖励积分必须在 0 至 100000000 之间")
	}
	if cfg.CommissionRateBasisPoints < 0 || cfg.CommissionRateBasisPoints > maxRateBasisPoints {
		return Config{}, errors.New("微信支付返佣比例必须在 0% 至 100% 之间")
	}
	if cfg.FirstLowSpendRewardCoins == 0 && cfg.FirstLowSpendRewardPoints == 0 && cfg.CommissionRateBasisPoints == 0 {
		return Config{}, errors.New("邀请奖励至少需要配置一项正数奖励")
	}
	return cfg, nil
}

type Repository interface {
	ActiveRule(ctx context.Context) (Rule, bool, error)
	RewardSummary(ctx context.Context, inviterMemberID int64) (RewardSummary, error)
}

type RewardSummary struct {
	RewardedMembers int64
	Coins           int64
	Points          int64
}

type sqlRepository struct{ db *platdb.DB }

func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ActiveRule(ctx context.Context) (Rule, bool, error) {
	var version int
	var raw []byte
	const q = `SELECT version, config_json FROM rule_definitions
		WHERE rule_key = ? AND scope_type = 'global' AND enabled = 1 AND status = 'published'
		  AND (effective_from IS NULL OR effective_from <= UTC_TIMESTAMP())
		  AND (effective_to IS NULL OR effective_to > UTC_TIMESTAMP())
		ORDER BY version DESC LIMIT 1`
	switch err := r.db.QueryRowContext(ctx, q, RuleKey).Scan(&version, &raw); {
	case errors.Is(err, sql.ErrNoRows):
		return Rule{}, false, nil
	case err != nil:
		return Rule{}, false, apperr.Internal(err)
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		return Rule{}, false, apperr.Internal(err)
	}
	return Rule{Version: version, Config: cfg}, true, nil
}

func (r *sqlRepository) RewardSummary(ctx context.Context, inviterMemberID int64) (RewardSummary, error) {
	const q = `SELECT COUNT(*), COALESCE(SUM(invitee_rewards.coins), 0),
		COALESCE(SUM(invitee_rewards.points), 0)
		FROM (
			SELECT invitee_member_id, SUM(coin_delta) AS coins, SUM(points_delta) AS points
			FROM invitation_reward_events
			WHERE inviter_member_id = ?
			GROUP BY invitee_member_id
			HAVING coins > 0 OR points > 0
		) AS invitee_rewards`
	var summary RewardSummary
	if err := r.db.QueryRowContext(ctx, q, inviterMemberID).Scan(
		&summary.RewardedMembers, &summary.Coins, &summary.Points,
	); err != nil {
		return RewardSummary{}, apperr.Internal(err)
	}
	return summary, nil
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Config(ctx context.Context, inviterMemberID int64) (ConfigView, error) {
	summary, err := s.repo.RewardSummary(ctx, inviterMemberID)
	if err != nil {
		return ConfigView{}, err
	}
	rule, ok, err := s.repo.ActiveRule(ctx)
	if err != nil {
		return ConfigView{}, err
	}
	view := ConfigView{
		RewardedMembers:        summary.RewardedMembers,
		CumulativeRewardCoins:  summary.Coins,
		CumulativeRewardPoints: summary.Points,
	}
	if !ok {
		return view, nil
	}
	view.Enabled = true
	view.FirstLowSpendRewardCoins = rule.Config.FirstLowSpendRewardCoins
	view.FirstLowSpendRewardPoints = rule.Config.FirstLowSpendRewardPoints
	view.CommissionRateBasisPoints = rule.Config.CommissionRateBasisPoints
	return view, nil
}
