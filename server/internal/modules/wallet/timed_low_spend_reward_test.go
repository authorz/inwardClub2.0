package wallet

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildTimedLowSpendWindowUsesBeijingCalendarDay(t *testing.T) {
	settings := timedLowSpendSettings{
		ReservationCutoff: "20:00",
		ConsumptionCutoff: "20:30",
	}
	paidAt := time.Date(2026, 8, 8, 12, 15, 0, 0, time.UTC) // 20:15 Beijing
	window, err := buildTimedLowSpendWindow(paidAt, settings)
	if err != nil {
		t.Fatalf("build window: %v", err)
	}
	if got := window.DayStart.In(chinaStandardTime).Format("2006-01-02 15:04"); got != "2026-08-08 00:00" {
		t.Fatalf("day start = %s", got)
	}
	if got := window.ReservationCutoff.In(chinaStandardTime).Format("15:04"); got != "20:00" {
		t.Fatalf("reservation cutoff = %s", got)
	}
	if got := window.ConsumptionCutoff.In(chinaStandardTime).Format("15:04"); got != "20:30" {
		t.Fatalf("consumption cutoff = %s", got)
	}
}

func TestBuildTimedLowSpendWindowRejectsReversedCutoffs(t *testing.T) {
	_, err := buildTimedLowSpendWindow(time.Now(), timedLowSpendSettings{
		ReservationCutoff: "21:00",
		ConsumptionCutoff: "20:30",
	})
	if err == nil {
		t.Fatal("expected reversed cutoff validation error")
	}
}

func TestLowSpendAmountQualifiedIncludesExactThreshold(t *testing.T) {
	tests := []struct {
		name      string
		totalCent int64
		minimum   int64
		qualified bool
	}{
		{name: "below threshold", totalCent: 8799, minimum: 8800, qualified: false},
		{name: "exact threshold", totalCent: 8800, minimum: 8800, qualified: true},
		{name: "above threshold", totalCent: 17600, minimum: 8800, qualified: true},
		{name: "invalid minimum", totalCent: 8800, minimum: 0, qualified: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := lowSpendAmountQualified(test.totalCent, test.minimum); got != test.qualified {
				t.Fatalf("lowSpendAmountQualified(%d, %d) = %t, want %t",
					test.totalCent, test.minimum, got, test.qualified)
			}
		})
	}
}

func TestVIPLowSpendWindowUsesStoreBusinessSession(t *testing.T) {
	raw := json.RawMessage(`{"minimumAmountCent":8800}`)
	tests := []struct {
		name      string
		paidAt    string
		qualified bool
		wantStart string
		wantEnd   string
		wantKey   string
	}{
		{
			name:   "evening belongs to current opening",
			paidAt: "2026-08-31T23:00:00+08:00", qualified: true,
			wantStart: "2026-08-31 15:00", wantEnd: "2026-09-01 08:00",
			wantKey: "20260831",
		},
		{
			name:   "morning belongs to previous opening",
			paidAt: "2026-09-01T07:00:00+08:00", qualified: true,
			wantStart: "2026-08-31 15:00", wantEnd: "2026-09-01 08:00",
			wantKey: "20260831",
		},
		{
			name:   "afternoon starts a new session",
			paidAt: "2026-09-01T16:00:00+08:00", qualified: true,
			wantStart: "2026-09-01 15:00", wantEnd: "2026-09-02 08:00",
			wantKey: "20260901",
		},
		{name: "outside business hours", paidAt: "2026-09-01T10:00:00+08:00"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paidAt, err := time.Parse(time.RFC3339, test.paidAt)
			if err != nil {
				t.Fatal(err)
			}
			window, ok, err := buildVIPLowSpendWindow("15:00-08:00", true, raw, paidAt)
			if err != nil {
				t.Fatalf("build VIP low-spend window: %v", err)
			}
			if ok != test.qualified {
				t.Fatalf("qualified = %v, want %v", ok, test.qualified)
			}
			if !ok {
				return
			}
			if got := window.Start.In(chinaStandardTime).Format("2006-01-02 15:04"); got != test.wantStart {
				t.Fatalf("start = %s, want %s", got, test.wantStart)
			}
			if got := window.End.In(chinaStandardTime).Format("2006-01-02 15:04"); got != test.wantEnd {
				t.Fatalf("end = %s, want %s", got, test.wantEnd)
			}
			if window.PeriodKey != test.wantKey {
				t.Fatalf("period key = %s, want %s", window.PeriodKey, test.wantKey)
			}
			if window.MinimumAmountCent != 8800 {
				t.Fatalf("minimum = %d, want 8800", window.MinimumAmountCent)
			}
		})
	}
}
