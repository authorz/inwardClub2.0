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
	rule := PointReviewRule{PointsDivisor: 5, Version: 1}
	outside := time.Date(2026, 7, 26, 12, 0, 0, 0, pointReviewLocation)
	inside := time.Date(2026, 7, 26, 20, 0, 0, 0, pointReviewLocation)
	outsideWindow, _ := businessWindow(outside, "15:00-02:00")
	insideWindow, _ := businessWindow(inside, "15:00-02:00")

	cases := []struct {
		name            string
		requested, base int64
		window          pointReviewWindow
		wantPoints      int64
		wantExcess      int64
	}{
		{name: "outside business ignores base", window: outsideWindow, requested: 8000, base: 3000, wantPoints: 1600},
		{name: "inside without base uses standard ratio", window: insideWindow, requested: 8000, base: 0, wantPoints: 1600},
		{name: "inside below base ignores base", window: insideWindow, requested: 800, base: 1000, wantPoints: 160},
		{name: "inside equal to base ignores base", window: insideWindow, requested: 1000, base: 1000, wantPoints: 200},
		{name: "inside over base ignores base", window: insideWindow, requested: 1600, base: 1000, wantPoints: 320},
		{name: "inside over base uses standard ratio", window: insideWindow, requested: 8000, base: 3000, wantPoints: 1600},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculatePointReview(tc.window, tc.requested, tc.base, rule)
			if got.AwardedPoints != tc.wantPoints || got.ExcessPoints != tc.wantExcess {
				t.Fatalf("calculation=%+v", got)
			}
			if !strings.Contains(got.Description, "实际获得积分 =") || !strings.Contains(got.Description, "向下取整") || !strings.Contains(got.Description, "不考虑基础积分") {
				t.Fatalf("calculation rule is not explicit: %q", got.Description)
			}
		})
	}
}

func TestCalculatePointReviewUsesConfiguredRatios(t *testing.T) {
	got := calculatePointReview(
		pointReviewWindow{},
		9000,
		0,
		PointReviewRule{PointsDivisor: 3, Version: 2},
	)
	if got.AwardedPoints != 3000 {
		t.Fatalf("configured calculation=%+v", got)
	}
}

func TestCalculatePointReviewIgnoresBelowBaseRatio(t *testing.T) {
	got := calculatePointReview(
		pointReviewWindow{InBusiness: true},
		800,
		1000,
		PointReviewRule{
			PointsDivisor: 5, BelowBasePointsDivisor: 4,
			Version: 2,
		},
	)
	if got.AwardedPoints != 160 {
		t.Fatalf("configured below-base calculation=%+v", got)
	}
}
