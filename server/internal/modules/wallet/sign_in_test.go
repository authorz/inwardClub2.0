package wallet

import (
	"testing"
	"time"
)

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

func TestPointsForStreakLadder(t *testing.T) {
	cases := map[int]int64{
		0:  100, // clamped up to day 1
		1:  100,
		2:  200,
		3:  300,
		4:  400,
		5:  500,
		6:  600,
		7:  700,
		8:  700, // day 7 and beyond stay capped at 700
		30: 700,
	}
	for streak, want := range cases {
		if got := pointsForStreak(signInDailyDefault, streak); got != want {
			t.Errorf("streak %d: got %d, want %d", streak, got, want)
		}
	}
}

// TestSignInStreakWeek walks seven consecutive days ending on today (2026-07-17)
// and checks the streak/points grow 1..7 → 100..700, then day 8 (2026-07-18)
// stays at 700.
func TestSignInStreakWeek(t *testing.T) {
	dates := []string{
		"2026-07-11", "2026-07-12", "2026-07-13", "2026-07-14",
		"2026-07-15", "2026-07-16", "2026-07-17", // today
		"2026-07-18",
	}
	wantStreak := []int{1, 2, 3, 4, 5, 6, 7, 8}
	wantPoints := []int64{100, 200, 300, 400, 500, 600, 700, 700}

	hasPrev := false
	var prevDate time.Time
	prevStreak := 0
	for i, ds := range dates {
		today := day(t, ds)
		streak := nextStreak(hasPrev, prevDate, today, prevStreak)
		if streak != wantStreak[i] {
			t.Fatalf("%s: streak got %d, want %d", ds, streak, wantStreak[i])
		}
		points := pointsForStreak(signInDailyDefault, streak)
		if points != wantPoints[i] {
			t.Fatalf("%s: points got %d, want %d", ds, points, wantPoints[i])
		}
		hasPrev, prevDate, prevStreak = true, today, streak
	}
}

func TestNextStreakResetsAfterGap(t *testing.T) {
	prev := day(t, "2026-07-14")
	today := day(t, "2026-07-17") // 3-day gap
	if got := nextStreak(true, prev, today, 5); got != 1 {
		t.Fatalf("gap should reset streak to 1, got %d", got)
	}
}

func TestSameDate(t *testing.T) {
	a := day(t, "2026-07-17").Add(2 * time.Hour)
	b := day(t, "2026-07-17").Add(20 * time.Hour)
	if !sameDate(a, b) {
		t.Fatalf("expected same calendar day")
	}
	if sameDate(a, day(t, "2026-07-18")) {
		t.Fatalf("expected different calendar day")
	}
}

func TestParseSignInLadder(t *testing.T) {
	ladder, ok := parseSignInLadder([]byte(`{"dailyRewards":[10,20,30]}`))
	if !ok || len(ladder) != 3 || ladder[2] != 30 {
		t.Fatalf("valid config not parsed: %v ok=%v", ladder, ok)
	}
	// The seeded default rule config parses to the 100..700 ladder.
	seeded, ok := parseSignInLadder([]byte(`{"dailyRewards":[100,200,300,400,500,600,700],"capDay":7,"capReward":700,"enabled":true}`))
	if !ok || len(seeded) != 7 || seeded[6] != 700 {
		t.Fatalf("seeded config not parsed: %v ok=%v", seeded, ok)
	}
	// capReward clamps every rung so a misconfigured ladder cannot over-award.
	clamped, ok := parseSignInLadder([]byte(`{"dailyRewards":[100,999],"capReward":700}`))
	if !ok || clamped[1] != 700 {
		t.Fatalf("capReward should clamp rungs: %v ok=%v", clamped, ok)
	}
	// The legacy "dailyPoints" key is still accepted for configs seeded before
	// the rename, so an admin ladder saved under the old key keeps working.
	legacy, ok := parseSignInLadder([]byte(`{"dailyPoints":[10,20,30]}`))
	if !ok || len(legacy) != 3 || legacy[2] != 30 {
		t.Fatalf("legacy dailyPoints key not parsed: %v ok=%v", legacy, ok)
	}
	if _, ok := parseSignInLadder([]byte(`{"dailyRewards":[]}`)); ok {
		t.Fatalf("empty ladder should fall back")
	}
	if _, ok := parseSignInLadder([]byte(`not json`)); ok {
		t.Fatalf("malformed config should fall back")
	}
}
