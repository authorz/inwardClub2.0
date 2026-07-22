package order

import (
	"context"
	"time"
)

// activityPayWindow is how long an unpaid activity order may stay open before
// the sweep closes it and releases its reserved stock (spec §9.4.2, §11). It is
// a conservative prepay window; the exact value is a business input (spec §13)
// and is tuned here rather than in the SQL.
const activityPayWindow = 15 * time.Minute

// ExpiryService runs the order module's scheduled sweeps (spec §11):
// activity-order:expire and the ticket half of ticket-coupon:expire. Both only
// transition DB state, so the worker builds it from just the repository —
// avoiding the payment/member/asset dependencies of the request-path Service.
type ExpiryService struct {
	repo Repository
	now  func() time.Time
}

// NewExpiryService builds the order expiry sweep service.
func NewExpiryService(repo Repository) *ExpiryService {
	return &ExpiryService{repo: repo, now: time.Now}
}

// SweepExpiredActivityOrders closes unpaid activity orders older than the pay
// window, releasing their stock. It returns the number of orders expired.
func (s *ExpiryService) SweepExpiredActivityOrders(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	return s.repo.ExpireActivityOrders(ctx, now.Add(-activityPayWindow), now)
}

// SweepExpiredTickets expires paid-but-unused tickets whose event has ended. It
// returns the number of tickets expired.
func (s *ExpiryService) SweepExpiredTickets(ctx context.Context) (int64, error) {
	return s.repo.ExpireTickets(ctx, s.now().UTC())
}
