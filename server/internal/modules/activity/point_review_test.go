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
		{time.Date(2026, 7, 26, 1, 59, 59, 0, pointReviewLocation), true},
		{time.Date(2026, 7, 26, 2, 0, 0, 0, pointReviewLocation), false},
		{time.Date(2026, 7, 26, 14, 59, 59, 0, pointReviewLocation), false},
		{time.Date(2026, 7, 26, 15, 0, 0, 0, pointReviewLocation), true},
		{time.Date(2026, 7, 26, 16, 29, 0, 0, pointReviewLocation), true},
	}
	for _, tc := range cases {
		window, err := businessWindow(tc.at, "15:00-02:00")
		if err != nil {
			t.Fatal(err)
		}
		if got := window.InBusiness; got != tc.open {
			t.Fatalf("businessWindow(%s) open=%v, want %v", tc.at, got, tc.open)
		}
	}
}

func TestCalculatePointReviewRules(t *testing.T) {
	rule := PointReviewRule{PointsDivisor: 5, CoinPointsDivisor: 2000, Version: 1}
	outside := time.Date(2026, 7, 26, 12, 0, 0, 0, pointReviewLocation)
	inside := time.Date(2026, 7, 26, 20, 0, 0, 0, pointReviewLocation)
	outsideWindow, _ := businessWindow(outside, "15:00-02:00")
	insideWindow, _ := businessWindow(inside, "15:00-02:00")

	cases := []struct {
		name                     string
		requested, base          int64
		window                   pointReviewWindow
		wantPoints, wantCoin     int64
		wantExcess, wantCoinBase int64
	}{
		{name: "outside business excludes known base from coins", window: outsideWindow, requested: 8000, base: 3000, wantPoints: 1600, wantCoin: 2, wantExcess: 5000, wantCoinBase: 5000},
		{name: "inside without base awards coins from saved points", window: insideWindow, requested: 8000, base: 0, wantPoints: 1600, wantCoin: 4, wantCoinBase: 8000},
		{name: "inside without base below coin divisor", window: insideWindow, requested: 1999, base: 0, wantPoints: 399, wantCoin: 0, wantCoinBase: 1999},
		{name: "inside without base at coin divisor", window: insideWindow, requested: 2000, base: 0, wantPoints: 400, wantCoin: 1, wantCoinBase: 2000},
		{name: "inside below base", window: insideWindow, requested: 800, base: 1000, wantPoints: 400, wantCoin: 0},
		{name: "inside equal to base", window: insideWindow, requested: 1000, base: 1000, wantPoints: 1000, wantCoin: 0},
		{name: "inside over base", window: insideWindow, requested: 1600, base: 1000, wantPoints: 1120, wantCoin: 0, wantExcess: 600, wantCoinBase: 600},
		{name: "inside over base awards coins from profit only", window: insideWindow, requested: 8000, base: 3000, wantPoints: 4000, wantCoin: 2, wantExcess: 5000, wantCoinBase: 5000},
		{name: "reported case awards five coins from profit", window: insideWindow, requested: 20000, base: 10000, wantPoints: 12000, wantCoin: 5, wantExcess: 10000, wantCoinBase: 10000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculatePointReview(tc.window, tc.requested, tc.base, rule)
			if got.AwardedPoints != tc.wantPoints || got.AwardedCoins != tc.wantCoin ||
				got.ExcessPoints != tc.wantExcess || got.CoinBasePoints != tc.wantCoinBase {
				t.Fatalf("calculation=%+v", got)
			}
			if !strings.Contains(got.Description, "实际获得积分 =") || !strings.Contains(got.Description, "向下取整") && tc.name != "inside equal to base" {
				t.Fatalf("calculation rule is not explicit: %q", got.Description)
			}
			if !strings.Contains(got.Description, "金币") {
				t.Fatalf("coin calculation rule is not explicit: %q", got.Description)
			}
		})
	}
}

func TestCalculatePointReviewUsesConfiguredRatios(t *testing.T) {
	got := calculatePointReview(
		pointReviewWindow{},
		9000,
		0,
		PointReviewRule{PointsDivisor: 3, CoinPointsDivisor: 1500, Version: 2},
	)
	if got.AwardedPoints != 3000 || got.AwardedCoins != 6 || got.CoinBasePoints != 9000 {
		t.Fatalf("configured calculation=%+v", got)
	}
}

func TestCalculatePointReviewUsesConfiguredBelowBaseRatio(t *testing.T) {
	got := calculatePointReview(
		pointReviewWindow{InBusiness: true},
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
