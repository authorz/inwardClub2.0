package rule

import (
	"context"
	"log/slog"
	"time"
)

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
