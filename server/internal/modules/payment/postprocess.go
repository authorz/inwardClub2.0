package payment

// This file holds the pure, database-free half of the payment:post-process
// worker: the config schema for the consumption-reward rule and the benefit
// computation over one settled, member-bound offline collection. Like the
// recharge growth policy (growth.go) and the sign-in ladder (wallet/sign_in.go),
// it never hard-codes an operational value (spec §1) — every grant amount is
// derived from an admin-published rule_definitions.config_json. Keeping the
// decision pure lets it be unit-tested without a live MySQL, matching the
// settlement spine test convention.

import (
	"encoding/json"
	"errors"
	"fmt"
)

// lowSpendRewardRuleKey is the rule_definitions.rule_key the post-process worker
// evaluates for a member-bound offline collection — spec §13's 低消奖励. It is the
// same key the (separately-added) rule package exposes as rule.KeyLowSpendReward,
// kept as a local literal so this handler carries no dependency on that package.
// It ships disabled and carries no seed: spec §13 lists its qualifying inputs —
// eligible order types/pay methods, amount basis (金额口径), the arrival / 20:30
// tests and refund handling — as still-pending business confirmation, so a
// developer must not self-enable it. See docs/payment-post-process.md for the seam.
const lowSpendRewardRuleKey = "low_spend_reward"

// pointsAsset is the wallet_accounts.asset_type consumption points credit into.
// coinsAsset and growthAsset are defined alongside the recharge settlement; the
// three together are the assets a low_spend_reward rule may earn (spec §9.3.4:
// 金币/积分/成长值). cash_balance is deliberately excluded — it is only moved by
// recharge and refund, never earned by a consumption rule.
const pointsAsset = "points"

// ErrUndecodablePostProcess wraps a task payload that cannot be turned into a
// usable hand-off (bad JSON, or missing member/payment identifiers). The worker
// maps it to asynq.SkipRetry: a message that never parses can never be fixed by a
// retry, so it is dropped rather than retried forever.
var ErrUndecodablePostProcess = errors.New("payment: undecodable post-process payload")

// grantMode is how a single benefit line turns a settled amount into a grant.
type grantMode string

const (
	// grantModeFixed grants Value units flat, independent of the settled amount.
	grantModeFixed grantMode = "fixed"
	// grantModePermille grants floor(amountCent * Value / 1000) units — Value is the
	// units earned per 1000 cent of settled amount. NOTE: the amount basis (金额口径)
	// here is the settled payment amount_cent; spec §13 lists the authoritative
	// basis as pending, so this is a documented, provisional modelling choice that
	// only takes effect once an admin publishes and enables the rule.
	grantModePermille grantMode = "permille"
)

// consumeGrantConfig is one benefit line of the low_spend_reward config: which
// wallet asset to credit and how much, given the settled amount.
type consumeGrantConfig struct {
	Asset string    `json:"asset"` // points | coins | growth_value
	Mode  grantMode `json:"mode"`  // fixed | permille
	Value int64     `json:"value"` // flat units (fixed) or units-per-1000-cent (permille)
}

// consumeRewardConfig is the low_spend_reward rule_definitions.config_json schema.
// It is intentionally minimal: a list of asset grants. The qualifying predicates
// (order type / pay method / arrival / 20:30) are NOT modelled here — they are the
// §13 seam and must be added before the rule is enabled (see the package doc).
type consumeRewardConfig struct {
	Grants []consumeGrantConfig `json:"grants"`
}

// benefitGrant is a computed, ready-to-persist grant: an asset and a positive
// amount to credit the member.
type benefitGrant struct {
	Asset  string
	Amount int64
}

// parseConsumeRewardConfig decodes a rule_definitions.config_json into the
// low_spend_reward schema. A malformed config is an operator error the worker
// surfaces to the caller rather than guessing at.
func parseConsumeRewardConfig(raw []byte) (consumeRewardConfig, error) {
	var cfg consumeRewardConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// computeGrants turns an enabled rule config and a settled amount into the
// concrete grants to apply. A line whose asset is not earnable, whose mode is
// unknown, or that computes to <= 0 (e.g. a permille line on a tiny amount) is
// skipped so it never writes a zero or bogus ledger entry — one bad config line
// can neither wedge the queue nor over-award.
func computeGrants(cfg consumeRewardConfig, amountCent int64) []benefitGrant {
	var out []benefitGrant
	for _, g := range cfg.Grants {
		if !isEarnableAsset(g.Asset) {
			continue
		}
		amount := computeGrantAmount(g, amountCent)
		if amount <= 0 {
			continue
		}
		out = append(out, benefitGrant{Asset: g.Asset, Amount: amount})
	}
	return out
}

// computeGrantAmount evaluates one grant line against the settled amount. An
// unknown mode or a non-positive input yields 0, which computeGrants drops.
func computeGrantAmount(g consumeGrantConfig, amountCent int64) int64 {
	switch g.Mode {
	case grantModeFixed:
		return g.Value
	case grantModePermille:
		if g.Value <= 0 || amountCent <= 0 {
			return 0
		}
		return amountCent * g.Value / 1000
	default:
		return 0
	}
}

// isEarnableAsset reports whether asset is one a low_spend_reward rule may credit.
func isEarnableAsset(asset string) bool {
	switch asset {
	case pointsAsset, coinsAsset, growthAsset:
		return true
	default:
		return false
	}
}

// parsePostProcessPayload decodes a raw payment:post-process task payload (the
// bytes the settlement wrote via the outbox) into the hand-off struct. A decode
// failure is wrapped as ErrUndecodablePostProcess so the worker drops it.
func parsePostProcessPayload(raw []byte) (postProcessPayload, error) {
	var p postProcessPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrUndecodablePostProcess, err)
	}
	return p, nil
}

// valid reports whether the hand-off carries the identifiers the grant needs. A
// walk-in (no member) never reaches here — settlement only writes the event for a
// member-bound collection — but the worker re-checks so a malformed event is
// dropped rather than retried forever.
func (p postProcessPayload) valid() bool {
	return p.MemberID > 0 && p.PaymentOrderID > 0
}

// PostProcessResult reports what the worker did with one post-process event, for
// structured logging. RuleMatched is false when no enabled low_spend_reward rule
// applied (the current production state, spec §13). AlreadyDone is true on an
// idempotent replay of an already-processed event. GrantsApplied counts the
// benefit_grants written on this pass (0 on a replay).
type PostProcessResult struct {
	RuleMatched   bool
	RuleVersion   int
	GrantsApplied int
	AlreadyDone   bool
}
