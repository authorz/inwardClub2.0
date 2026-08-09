package wallet

import (
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
