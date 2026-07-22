package payment

import "testing"

// tiers mirrors a typical membership_tiers configuration: a base tier at
// threshold 0 plus two paid tiers. resolveTier is the core of the VIP growth
// upgrade path, so it is exercised across the threshold boundaries the settlement
// relies on. Input order is deliberately shuffled to prove the scan is
// order-independent.
var tiers = []tierRow{
	{id: 30, level: 3, threshold: 5000},
	{id: 10, level: 1, threshold: 0},
	{id: 20, level: 2, threshold: 1000},
}

func TestResolveTierBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		balance int64
		wantID  int64
	}{
		{"zero balance -> base tier", 0, 10},
		{"below silver", 999, 10},
		{"exactly silver threshold", 1000, 20},
		{"between silver and gold", 4999, 20},
		{"exactly gold threshold", 5000, 30},
		{"far above gold", 999999, 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveTier(tiers, tc.balance)
			if !ok {
				t.Fatalf("balance %d: expected a qualifying tier, got none", tc.balance)
			}
			if got.id != tc.wantID {
				t.Fatalf("balance %d: got tier id %d, want %d", tc.balance, got.id, tc.wantID)
			}
		})
	}
}

func TestResolveTierNoQualifyingTier(t *testing.T) {
	// Every configured tier requires growth the member has not yet reached: the
	// caller must leave the member unranked, so ok is false.
	paidOnly := []tierRow{
		{id: 20, level: 2, threshold: 1000},
		{id: 30, level: 3, threshold: 5000},
	}
	if _, ok := resolveTier(paidOnly, 500); ok {
		t.Fatalf("balance below every threshold must not resolve a tier")
	}
}

func TestResolveTierEmpty(t *testing.T) {
	if _, ok := resolveTier(nil, 10_000); ok {
		t.Fatalf("no configured tiers must resolve to ok=false")
	}
}

func TestResolveTierThresholdTieBreaksToHigherLevel(t *testing.T) {
	// Two tiers share a threshold (a misconfiguration, but must be deterministic):
	// the higher level wins so an upgrade never lands on the weaker duplicate.
	tie := []tierRow{
		{id: 40, level: 4, threshold: 2000},
		{id: 41, level: 5, threshold: 2000},
	}
	got, ok := resolveTier(tie, 3000)
	if !ok {
		t.Fatalf("expected a qualifying tier")
	}
	if got.level != 5 || got.id != 41 {
		t.Fatalf("tie on threshold must break to the higher level, got level %d id %d", got.level, got.id)
	}
}

func TestResolveTierPicksGreatestThresholdNotLevel(t *testing.T) {
	// The qualifying rule is "greatest threshold met", independent of the order or
	// of a lower-threshold tier happening to carry a higher level number.
	odd := []tierRow{
		{id: 1, level: 9, threshold: 100},  // high level, low threshold
		{id: 2, level: 2, threshold: 3000}, // the real qualifying tier at balance 3000
	}
	got, ok := resolveTier(odd, 3000)
	if !ok || got.id != 2 {
		t.Fatalf("must pick the greatest threshold met (id 2), got ok=%v id=%d", ok, got.id)
	}
}
