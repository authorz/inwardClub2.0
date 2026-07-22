package member

import (
	"testing"
	"time"
)

// TestRankingWindowStartBusinessZone verifies the monthly/weekly windows are
// anchored to the business zone's calendar (not the host/UTC calendar) and
// returned as UTC instants. With the business "now" pinned to 2026-07-17 in
// Asia/Shanghai (UTC+8), the month window starts at 2026-07-01 00:00 +08, i.e.
// 2026-06-30 16:00 UTC.
func TestRankingWindowStartBusinessZone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, loc) // Friday

	since, ok := rankingWindowStart(RankingMonth, now)
	if !ok {
		t.Fatal("month window should have a lower bound")
	}
	want := time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC)
	if !since.Equal(want) {
		t.Fatalf("month start: got %s, want %s", since.UTC(), want)
	}

	// Week: Monday 2026-07-13 00:00 +08 = 2026-07-12 16:00 UTC.
	since, ok = rankingWindowStart(RankingWeek, now)
	if !ok {
		t.Fatal("week window should have a lower bound")
	}
	want = time.Date(2026, 7, 12, 16, 0, 0, 0, time.UTC)
	if !since.Equal(want) {
		t.Fatalf("week start: got %s, want %s", since.UTC(), want)
	}

	if _, ok := rankingWindowStart(RankingAll, now); ok {
		t.Fatal("all-time window should have no lower bound")
	}
}
