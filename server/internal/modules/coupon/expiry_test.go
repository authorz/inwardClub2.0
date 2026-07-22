package coupon

import (
	"context"
	"testing"
	"time"
)

// ExpireEntitlements on the in-memory repo mirrors the SQL sweep contract: only
// active entitlements with a dated, elapsed validity flip to expired.
func (r *memRepo) ExpireEntitlements(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for _, coupons := range r.byMember {
		for i := range coupons {
			c := &coupons[i]
			if c.Status == StatusActive && c.ExpiresAt != nil && c.ExpiresAt.Before(now) {
				c.Status = StatusExpired
				n++
			}
		}
	}
	return n, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestSweepExpiredCoupons(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {
			{EntitlementID: 1, Status: StatusActive, ExpiresAt: ptrTime(past)},   // expire
			{EntitlementID: 2, Status: StatusActive, ExpiresAt: ptrTime(future)}, // keep (not yet)
			{EntitlementID: 3, Status: StatusActive, ExpiresAt: nil},             // keep (never expires)
			{EntitlementID: 4, Status: StatusUsed, ExpiresAt: ptrTime(past)},     // keep (already consumed)
			{EntitlementID: 5, Status: StatusExpired, ExpiresAt: ptrTime(past)},  // keep (terminal)
		},
	}}
	svc := NewExpiryService(repo)
	svc.now = func() time.Time { return now }

	n, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count: got %d, want 1", n)
	}
	got := map[int64]string{}
	for _, c := range repo.byMember[10] {
		got[c.EntitlementID] = c.Status
	}
	if got[1] != StatusExpired {
		t.Fatalf("entitlement 1: got %q, want expired", got[1])
	}
	if got[2] != StatusActive || got[3] != StatusActive {
		t.Fatalf("entitlements 2/3 should stay active, got %q/%q", got[2], got[3])
	}
	if got[4] != StatusUsed {
		t.Fatalf("entitlement 4 should stay used, got %q", got[4])
	}

	again, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again != 0 {
		t.Fatalf("second sweep count: got %d, want 0", again)
	}
}
