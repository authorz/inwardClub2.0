package activity

import (
	"strings"
	"testing"
	"time"
)

func TestBusinessWindowBoundaries(t *testing.T) {
	cases := []struct {
		at   time.Time
		open bool
	}{
		{time.Date(2026, 7, 26, 9, 59, 59, 0, pointReviewLocation), true},
		{time.Date(2026, 7, 26, 10, 0, 0, 0, pointReviewLocation), false},
		{time.Date(2026, 7, 26, 16, 59, 59, 0, pointReviewLocation), false},
		{time.Date(2026, 7, 26, 17, 0, 0, 0, pointReviewLocation), true},
	}
	for _, tc := range cases {
		if got := businessWindow(tc.at).InBusiness; got != tc.open {
			t.Fatalf("businessWindow(%s) open=%v, want %v", tc.at, got, tc.open)
		}
	}
}

func TestCalculatePointReviewRules(t *testing.T) {
	rule := PointReviewRule{PointsDivisor: 5, CoinPointsDivisor: 2000, Version: 1}
	outside := time.Date(2026, 7, 26, 12, 0, 0, 0, pointReviewLocation)
	inside := time.Date(2026, 7, 26, 20, 0, 0, 0, pointReviewLocation)

	cases := []struct {
		name                 string
		requested, base      int64
		when                 time.Time
		wantPoints, wantCoin int64
		wantExcess           int64
	}{
		{name: "outside business", when: outside, requested: 8000, base: 3000, wantPoints: 1600, wantCoin: 4},
		{name: "inside without base", when: inside, requested: 8000, base: 0, wantPoints: 1600, wantCoin: 4},
		{name: "inside below base", when: inside, requested: 800, base: 1000, wantPoints: 400, wantCoin: 0},
		{name: "inside equal to base", when: inside, requested: 1000, base: 1000, wantPoints: 1000, wantCoin: 0},
		{name: "inside over base", when: inside, requested: 1600, base: 1000, wantPoints: 1120, wantCoin: 0, wantExcess: 600},
		{name: "inside over base awards coins", when: inside, requested: 8000, base: 3000, wantPoints: 4000, wantCoin: 2, wantExcess: 5000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculatePointReview(tc.when, tc.requested, tc.base, rule)
			if got.AwardedPoints != tc.wantPoints || got.AwardedCoins != tc.wantCoin || got.ExcessPoints != tc.wantExcess {
				t.Fatalf("calculation=%+v", got)
			}
			if !strings.Contains(got.Description, "实际获得积分 =") || !strings.Contains(got.Description, "向下取整") && tc.name != "inside equal to base" {
				t.Fatalf("calculation rule is not explicit: %q", got.Description)
			}
		})
	}
}

func TestCalculatePointReviewUsesConfiguredRatios(t *testing.T) {
	got := calculatePointReview(
		time.Date(2026, 7, 26, 12, 0, 0, 0, pointReviewLocation),
		9000,
		0,
		PointReviewRule{PointsDivisor: 3, CoinPointsDivisor: 1500, Version: 2},
	)
	if got.AwardedPoints != 3000 || got.AwardedCoins != 6 {
		t.Fatalf("configured calculation=%+v", got)
	}
}

func TestCalculatePointReviewUsesConfiguredBelowBaseRatio(t *testing.T) {
	got := calculatePointReview(
		time.Date(2026, 7, 26, 20, 0, 0, 0, pointReviewLocation),
		800,
		1000,
		PointReviewRule{
			PointsDivisor: 5, BelowBasePointsDivisor: 4,
			CoinPointsDivisor: 2000, Version: 2,
		},
	)
	if got.AwardedPoints != 200 || got.AwardedCoins != 0 {
		t.Fatalf("configured below-base calculation=%+v", got)
	}
}
