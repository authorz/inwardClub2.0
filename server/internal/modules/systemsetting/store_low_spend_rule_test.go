package systemsetting

import (
	"context"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

type storeLowSpendRuleMemoryRepo struct{ view StoreLowSpendRuleView }

func (r *storeLowSpendRuleMemoryRepo) List(context.Context, string, httpx.Page) ([]StoreLowSpendRuleView, int64, error) {
	return []StoreLowSpendRuleView{r.view}, 1, nil
}
func (r *storeLowSpendRuleMemoryRepo) Get(context.Context, int64) (StoreLowSpendRuleView, error) {
	return r.view, nil
}
func (r *storeLowSpendRuleMemoryRepo) Upsert(_ context.Context, storeID int64, config StoreLowSpendRuleConfig, enabled bool, now time.Time) (StoreLowSpendRuleView, error) {
	r.view = StoreLowSpendRuleView{
		StoreID: storeID, StoreName: "测试门店", Configured: true, Enabled: enabled,
		ReservationCutoff: config.ReservationCutoff, ConsumptionCutoff: config.ConsumptionCutoff,
		MinimumAmount: config.MinimumAmountCent / 100, RewardPoints: config.RewardPoints,
		UpdatedAt: &now,
	}
	return r.view, nil
}
func (r *storeLowSpendRuleMemoryRepo) Delete(context.Context, int64) error { return nil }

func TestStoreLowSpendRuleUpdate(t *testing.T) {
	svc := NewStoreLowSpendRuleService(&storeLowSpendRuleMemoryRepo{})
	got, err := svc.Update(context.Background(), 2, UpdateStoreLowSpendRuleRequest{
		Enabled: true, ReservationCutoff: "20:00", ConsumptionCutoff: "20:30",
		MinimumAmount: 88, RewardPoints: 2000,
	})
	if err != nil {
		t.Fatalf("update rule: %v", err)
	}
	if !got.Configured || !got.Enabled || got.MinimumAmount != 88 || got.RewardPoints != 2000 {
		t.Fatalf("unexpected rule: %+v", got)
	}
}

func TestStoreLowSpendRuleRejectsReversedWindow(t *testing.T) {
	svc := NewStoreLowSpendRuleService(&storeLowSpendRuleMemoryRepo{})
	_, err := svc.Update(context.Background(), 2, UpdateStoreLowSpendRuleRequest{
		Enabled: true, ReservationCutoff: "20:30", ConsumptionCutoff: "20:00",
		MinimumAmount: 88, RewardPoints: 2000,
	})
	if err == nil {
		t.Fatal("expected invalid time window")
	}
}

func TestReservationAvailabilityFollowsConfiguredCutoff(t *testing.T) {
	repo := &storeLowSpendRuleMemoryRepo{view: StoreLowSpendRuleView{
		StoreID: 2, Configured: true, Enabled: true, ReservationCutoff: "20:00",
	}}
	svc := NewStoreLowSpendRuleService(repo)

	tests := []struct {
		name       string
		now        time.Time
		reservable bool
		reason     string
	}{
		{
			name: "before cutoff", now: time.Date(2026, 9, 5, 19, 59, 59, 0, reservationRuleLocation),
			reservable: true,
		},
		{
			name: "at cutoff", now: time.Date(2026, 9, 5, 20, 0, 0, 0, reservationRuleLocation),
			reservable: false, reason: "今日预约已截止",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc.now = func() time.Time { return test.now }
			got, err := svc.ReservationAvailability(context.Background(), 2)
			if err != nil {
				t.Fatalf("reservation availability: %v", err)
			}
			if got.Reservable != test.reservable || got.UnavailableReason != test.reason {
				t.Fatalf("availability = %+v", got)
			}
			if got.ReservationCutoff != "20:00" || got.CutoffAt == nil {
				t.Fatalf("cutoff = %+v", got)
			}
		})
	}
}

func TestReservationAvailabilityRequiresEnabledConfiguredRule(t *testing.T) {
	repo := &storeLowSpendRuleMemoryRepo{view: StoreLowSpendRuleView{
		StoreID: 2, Configured: true, Enabled: false, ReservationCutoff: "20:00",
	}}
	svc := NewStoreLowSpendRuleService(repo)
	svc.now = func() time.Time {
		return time.Date(2026, 9, 5, 12, 0, 0, 0, reservationRuleLocation)
	}

	got, err := svc.ReservationAvailability(context.Background(), 2)
	if err != nil {
		t.Fatalf("reservation availability: %v", err)
	}
	if got.Reservable || got.UnavailableReason != "门店暂未开放预约" {
		t.Fatalf("availability = %+v", got)
	}
}
