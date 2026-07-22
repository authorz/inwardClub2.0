package wallet

// Daily sign-in points ladder and streak arithmetic. These are the pure,
// database-free pieces of the sign-in flow so the reward semantics can be tested
// directly; the repository (points_repository.go) drives them inside a
// transaction and persists the record + wallet ledger entry.

import (
	"encoding/json"
	"time"
)

// signInDailyDefault is the server-side fallback ladder used when
// rule_definitions has no enabled rule_key=sign_in config: day 1 = 100 points,
// day 2 = 200 ... day 7 = 700. Streaks beyond day 7 keep the capped 700.
var signInDailyDefault = []int64{100, 200, 300, 400, 500, 600, 700}

// Clock is the source of "now" for the sign-in calendar day. It is injected from
// the configured business clock (see platform/config) so the calendar day never
// drifts with the host clock or host zone: Now already returns the time in the
// business zone, and dev/test/acceptance can pin it to a fixed instant.
type Clock interface{ Now() time.Time }

// pointsForStreak returns the points awarded for a consecutive-day streak using
// the ladder. A streak of 7 or more keeps the last (capped) rung.
func pointsForStreak(ladder []int64, streak int) int64 {
	if streak < 1 {
		streak = 1
	}
	if streak > len(ladder) {
		streak = len(ladder)
	}
	return ladder[streak-1]
}

// nextStreak computes the streak count for `today` given the member's most
// recent sign-in. hasPrev is false when the member has never signed in. A gap of
// more than one day resets the streak to 1; a same-day repeat is handled by the
// caller before this is reached.
func nextStreak(hasPrev bool, prevDate, today time.Time, prevStreak int) int {
	if !hasPrev {
		return 1
	}
	if daysBetween(prevDate, today) == 1 {
		return prevStreak + 1
	}
	return 1
}

// daysBetween returns the number of whole calendar days from prev to today,
// ignoring the clock time within each day.
func daysBetween(prev, today time.Time) int {
	p := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return int(t.Sub(p).Hours() / 24)
}

// sameDate reports whether a and b fall on the same calendar day.
func sameDate(a, b time.Time) bool { return daysBetween(a, b) == 0 }

// parseSignInLadder decodes a rule_definitions config_json of the shape
// {"dailyRewards":[100,...,700],"capReward":700}. dailyRewards is the reward per
// consecutive day; capReward (when > 0) clamps every rung so a misconfigured
// ladder can never over-award. The legacy "dailyPoints" key is accepted as a
// fallback for configs seeded before the rename. It returns ok=false for
// malformed or empty config so the caller can fall back to the server default.
func parseSignInLadder(raw []byte) ([]int64, bool) {
	var cfg struct {
		DailyRewards []int64 `json:"dailyRewards"`
		DailyPoints  []int64 `json:"dailyPoints"`
		CapReward    int64   `json:"capReward"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, false
	}
	ladder := cfg.DailyRewards
	if len(ladder) == 0 {
		ladder = cfg.DailyPoints
	}
	if len(ladder) == 0 {
		return nil, false
	}
	if cfg.CapReward > 0 {
		for i, v := range ladder {
			if v > cfg.CapReward {
				ladder[i] = cfg.CapReward
			}
		}
	}
	return ladder, true
}
