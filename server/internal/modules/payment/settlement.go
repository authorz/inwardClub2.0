package payment

import (
	"context"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// SettlementRepository applies a verified payment notification to the order
// spine: it records the payment_transaction, marks the payment_order and
// business_order paid (and, for offline collections, the collection order) and
// stays idempotent when the same notification is delivered more than once.
type SettlementRepository interface {
	SettleWeChat(ctx context.Context, n WeChatNotification, now time.Time) error
	SettleOffline(ctx context.Context, n OfflineNotification, now time.Time) error
}

// SettlementService turns a verified callback into an idempotent state
// transition. Signature verification stays in the handler/gateway; this service
// only trusts an already-verified notification.
type SettlementService struct {
	repo SettlementRepository
	now  func() time.Time
}

// NewSettlementService builds the settlement service.
func NewSettlementService(repo SettlementRepository) *SettlementService {
	return &SettlementService{repo: repo, now: time.Now}
}

// SettleWeChat settles a verified WeChat notify. A non-success notification is
// acknowledged without touching state: WeChat only notifies on success today,
// and a failed/closed notify must never mark an order paid.
func (s *SettlementService) SettleWeChat(ctx context.Context, n WeChatNotification) error {
	if !n.Success {
		return nil
	}
	if n.OutTradeNo == "" {
		return apperr.Invalid("wechat notification missing outTradeNo")
	}
	return s.repo.SettleWeChat(ctx, n, s.now().UTC())
}

// SettleOffline settles a verified offline aggregated-collection notify. The
// order is located by outTradeNo (the payment order number) or, as a fallback,
// by the acquirer order number.
func (s *SettlementService) SettleOffline(ctx context.Context, n OfflineNotification) error {
	if !n.Success {
		return nil
	}
	if n.OutTradeNo == "" && n.AcquirerOrderNo == "" {
		return apperr.Invalid("offline notification missing outTradeNo/acquirerOrderNo")
	}
	return s.repo.SettleOffline(ctx, n, s.now().UTC())
}
