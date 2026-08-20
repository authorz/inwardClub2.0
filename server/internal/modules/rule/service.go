package rule

import (
	"context"
	"log/slog"
	"time"
)

// MonthlyBenefitService drives the daily benefit:vip-monthly task. It resolves
// the VIP monthly-benefit rule and, while that rule stays disabled (the current
// state, spec §13), grants nothing. The idempotency key the grant would carry is
// benefit:{ruleVersion}:{member}:{period} (spec §11) — the version comes from the
// resolved Definition, the period from the business month; those keys are only
// materialised once the grant application is built (see Run).
type MonthlyBenefitService struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

// NewMonthlyBenefitService builds the VIP monthly-benefit driver.
func NewMonthlyBenefitService(repo Repository, log *slog.Logger) *MonthlyBenefitService {
	return &MonthlyBenefitService{repo: repo, log: log, now: time.Now}
}

// Run evaluates the VIP monthly-benefit rule for the current period and returns
// the number of benefit grants applied. With no enabled rule it grants nothing
// and returns 0 — the guaranteed outcome until business enables the rule. If a
// rule is enabled, the grant application (which members, which benefit_type /
// amount, the period boundary and补发/过期 handling) is still pending business
// confirmation, so it is surfaced as a warning rather than applied under an
// unconfirmed policy; it still grants nothing. Either way the sweep is a safe
// no-op today, matching the worker's other idempotent sweeps.
func (s *MonthlyBenefitService) Run(ctx context.Context) (int64, error) {
	def, ok, err := s.repo.ActiveRule(ctx, KeyVIPMonthlyBenefit, s.now().UTC())
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil // no enabled rule: VIP monthly benefits are not configured
	}
	s.log.Warn("benefit:vip-monthly rule is enabled but grant application is pending business confirmation (spec §13); granting nothing",
		"ruleKey", KeyVIPMonthlyBenefit, "ruleVersion", def.Version)
	return 0, nil
}

// EvalInput is the post-payment fact set the rule:post-process evaluator keys on.
// It mirrors the settlement's post-process outbox hand-off (a settled,
// member-bound order) so the eventual enqueue path lines up; it never carries a
// raw phone number.
type EvalInput struct {
	MemberID        int64  `json:"memberId"`
	PaymentOrderID  int64  `json:"paymentOrderId"`
	BusinessOrderID int64  `json:"businessOrderId"`
	StoreID         int64  `json:"storeId"`
	AmountCent      int64  `json:"amountCent"`
	Source          string `json:"source,omitempty"`
}

// PostProcessService keeps the legacy rule:post-process task parseable. All
// WeChat invitation rewards now run synchronously in internal/modules/referral,
// inside payment settlement; no producer enqueues this compatibility task.
type PostProcessService struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

// NewPostProcessService builds the post-payment invite-reward evaluator.
func NewPostProcessService(repo Repository, log *slog.Logger) *PostProcessService {
	return &PostProcessService{repo: repo, log: log, now: time.Now}
}

// Evaluate resolves the rule for diagnostics but deliberately grants nothing;
// the authoritative referral path already ran in the payment transaction.
func (s *PostProcessService) Evaluate(ctx context.Context, in EvalInput) (int64, error) {
	def, ok, err := s.repo.ActiveRule(ctx, KeyInviteReward, s.now().UTC())
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil // no enabled rule: invite rewards are not configured
	}
	s.log.Warn("legacy rule:post-process invite-reward task ignored; payment settlement owns invitation rewards",
		"ruleKey", KeyInviteReward, "ruleVersion", def.Version, "paymentOrderId", in.PaymentOrderID, "memberId", in.MemberID)
	return 0, nil
}
