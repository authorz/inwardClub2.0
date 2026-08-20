// Package rule holds the worker-side evaluation of the configurable rule engine
// (rule_definitions / benefit_grants, migration 00011). It is the read side of
// the rule model the admin console already manages. The daily VIP monthly-benefit
// driver resolves its rule here; rule:post-process remains a legacy no-op reader.
//
// Scope note: the 低消奖励 half of the post-payment rewards is owned by the
// payment package's payment:post-process handler (rule_key=low_spend_reward),
// which applies its grants inside the settlement hand-off. This package
// deliberately does not touch low_spend_reward so the two evaluators can never
// double-grant the same order.
//
// Invitation rewards no longer grant from this package: the confirmed policy is
// implemented by internal/modules/referral in the WeChat settlement transaction.
// VIP monthly benefits remain disabled until that separate policy is confirmed.
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
