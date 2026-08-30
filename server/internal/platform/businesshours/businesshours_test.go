package businesshours

import (
	"testing"
	"time"
)

func TestOvernightScheduleWindowsAndBoundaries(t *testing.T) {
	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
	schedule, err := Parse("15:00-02:00")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		at           string
		open         bool
		wantStart    string
		wantEnd      string
		wantBoundary string
	}{
		{"2026-08-30 14:59", false, "", "", "2026-08-30 15:00"},
		{"2026-08-30 15:00", true, "2026-08-30 15:00", "2026-08-31 02:00", "2026-08-31 02:00"},
		{"2026-08-30 16:29", true, "2026-08-30 15:00", "2026-08-31 02:00", "2026-08-31 02:00"},
		{"2026-08-31 01:59", true, "2026-08-30 15:00", "2026-08-31 02:00", "2026-08-31 02:00"},
		{"2026-08-31 02:00", false, "", "", "2026-08-31 15:00"},
	}
	for _, tc := range cases {
		t.Run(tc.at, func(t *testing.T) {
			now, err := time.ParseInLocation("2006-01-02 15:04", tc.at, loc)
			if err != nil {
				t.Fatal(err)
			}
			start, end, open := schedule.CurrentWindow(now, loc)
			if open != tc.open {
				t.Fatalf("open=%v, want %v", open, tc.open)
			}
			if tc.open {
				if start.Format("2006-01-02 15:04") != tc.wantStart || end.Format("2006-01-02 15:04") != tc.wantEnd {
					t.Fatalf("window=%s..%s", start, end)
				}
			}
			if got := schedule.NextBoundary(now, loc).Format("2006-01-02 15:04"); got != tc.wantBoundary {
				t.Fatalf("next boundary=%s, want %s", got, tc.wantBoundary)
			}
		})
	}
}

func TestDaytimeSchedule(t *testing.T) {
	loc := time.UTC
	schedule, err := Parse("09:30 – 18:00")
	if err != nil {
		t.Fatal(err)
	}
	inside := time.Date(2026, 8, 30, 12, 0, 0, 0, loc)
	closing := time.Date(2026, 8, 30, 18, 0, 0, 0, loc)
	if !schedule.IsOpen(inside, loc) || schedule.IsOpen(closing, loc) {
		t.Fatal("daytime interval must include opening and exclude closing")
	}
}

func TestParseRejectsInvalidSchedule(t *testing.T) {
	for _, value := range []string{"", "9-18", "25:00-02:00", "09:60-18:00", "09:00-09:00"} {
		if _, err := Parse(value); err == nil {
			t.Fatalf("Parse(%q) should fail", value)
		}
	}
}
