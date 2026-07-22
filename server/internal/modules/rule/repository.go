// Package rule holds the worker-side evaluation of the configurable rule engine
// (rule_definitions / benefit_grants, migration 00011). It is the read side of
// the rule model the admin console already manages: the daily VIP monthly-benefit
// driver (benefit:vip-monthly) and the post-payment invite-reward evaluator
// (rule:post-process, 邀请奖励) both resolve their rule here.
//
// Scope note: the 低消奖励 half of the post-payment rewards is owned by the
// payment package's payment:post-process handler (rule_key=low_spend_reward),
// which applies its grants inside the settlement hand-off. This package
// deliberately does not touch low_spend_reward so the two evaluators can never
// double-grant the same order.
//
// Per spec §13 these business rules (VIP 日/月福利, 邀请奖励) may not be
// self-enabled by developers: the row stays disabled until business confirms its
// benefit values, eligibility and补发/过期 semantics. So on any deployment today
// ActiveRule finds no enabled rule and every evaluation resolves to a safe no-op
// — the harness is wired and idempotent, but grants nothing until business turns
// a rule on. The grant-application into benefit_grants is intentionally not built
// yet (see the services), because doing so would bake in unconfirmed policy.
package rule

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Rule keys evaluated by the worker. They index rule_definitions.rule_key; the
// admin console creates/edits/publishes rows under these keys.
const (
	// KeyVIPMonthlyBenefit is the daily-evaluated VIP monthly benefit (spec §13 VIP row).
	KeyVIPMonthlyBenefit = "vip_monthly_benefit"
	// KeyInviteReward is the post-payment 邀请奖励 (spec §13 邀请 row). The sibling
	// 低消奖励 lives under rule_key=low_spend_reward and is evaluated by the
	// payment package's payment:post-process handler, not here.
	KeyInviteReward = "invite_reward"
)

// Definition is the minimal projection of an active rule_definitions row the
// worker needs: the version (part of every idempotency key) and the raw
// config_json the eventual grant would be driven by.
type Definition struct {
	RuleKey    string
	Version    int
	ConfigJSON []byte
}

// Repository is the read port over rule_definitions.
type Repository interface {
	// ActiveRule returns the currently-effective, enabled, published rule for the
	// given key (highest version wins), or ok=false when none is configured. It
	// mirrors the resolution wallet.signInLadder already uses for rule_key=sign_in.
	ActiveRule(ctx context.Context, ruleKey string, now time.Time) (Definition, bool, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL rule repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ActiveRule(ctx context.Context, ruleKey string, now time.Time) (Definition, bool, error) {
	const q = `SELECT rule_key, version, config_json FROM rule_definitions
		WHERE rule_key = ? AND enabled = 1 AND status = 'published'
		  AND (effective_from IS NULL OR effective_from <= ?)
		  AND (effective_to IS NULL OR effective_to > ?)
		ORDER BY version DESC LIMIT 1`
	var d Definition
	switch err := r.db.QueryRowContext(ctx, q, ruleKey, now, now).Scan(&d.RuleKey, &d.Version, &d.ConfigJSON); {
	case errors.Is(err, sql.ErrNoRows):
		return Definition{}, false, nil
	case err != nil:
		return Definition{}, false, apperr.Internal(err)
	}
	return d, true, nil
}
