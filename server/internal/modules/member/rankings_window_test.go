package member

import (
	"testing"
	"time"
)

// TestRankingWindowBusinessZone verifies the natural-month windows are anchored
// to the business zone's calendar (not the host/UTC calendar) and returned as
// UTC instants.
func TestRankingWindowBusinessZone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, loc)

	start, end, ok := rankingWindow(RankingMonth, now)
	if !ok {
		t.Fatal("month ranking should have a natural-month window")
	}
	wantStart := time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("month window: got [%s, %s), want [%s, %s)", start, end, wantStart, wantEnd)
	}

	waterStart, waterEnd, ok := rankingWindow(RankingWater, now)
	if !ok || !waterStart.Equal(start) || !waterEnd.Equal(end) {
		t.Fatalf("water ranking should share the natural-month window, got [%s, %s)", waterStart, waterEnd)
	}

	if _, _, ok := rankingWindow(RankingAll, now); ok {
		t.Fatal("all-time ranking should have no time window")
	}
}
