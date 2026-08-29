package vipbenefit

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestCouponExpiryUsesNaturalBoundaries(t *testing.T) {
	now := mustTime(t, "2026-08-24T12:00:00+08:00") // Monday.
	cases := map[string]string{
		"weekday_event": "2026-08-28T16:00:00Z", // Saturday 00:00 CST.
		"weekly_event":  "2026-08-30T16:00:00Z", // Next Monday 00:00 CST.
		"monthly_event": "2026-08-31T16:00:00Z", // Sep 1 00:00 CST.
	}
	for trigger, want := range cases {
		if got := couponExpiry(trigger, now).Format(time.RFC3339); got != want {
			t.Fatalf("%s expiry = %s, want %s", trigger, got, want)
		}
	}
}

func TestWeekdayTicketIsInactiveOnWeekend(t *testing.T) {
	if scheduledTriggerActive("weekday_event", mustTime(t, "2026-08-29T12:00:00+08:00")) {
		t.Fatal("weekday ticket must not be issued on Saturday")
	}
	if !scheduledTriggerActive("weekday_event", mustTime(t, "2026-08-28T12:00:00+08:00")) {
		t.Fatal("weekday ticket must be issuable on Friday")
	}
}

func TestWithinBusinessHoursCapsAtMidnight(t *testing.T) {
	cases := []struct {
		hours string
		now   string
		want  bool
	}{
		{"10:00-22:00", "2026-08-24T10:00:00+08:00", true},
		{"10:00-22:00", "2026-08-24T21:59:00+08:00", true},
		{"10:00-22:00", "2026-08-24T22:00:00+08:00", false},
		{"18:00-02:00", "2026-08-24T23:59:00+08:00", true},
		{"18:00-02:00", "2026-08-25T00:01:00+08:00", false},
		{"bad value", "2026-08-24T12:00:00+08:00", false},
	}
	for _, tc := range cases {
		if got := withinBusinessHours(tc.hours, mustTime(t, tc.now)); got != tc.want {
			t.Fatalf("withinBusinessHours(%q, %s) = %v, want %v", tc.hours, tc.now, got, tc.want)
		}
	}
}

func TestPeriodKeyUsesNaturalWeek(t *testing.T) {
	monday := mustTime(t, "2026-08-24T00:00:00+08:00")
	sunday := mustTime(t, "2026-08-30T23:59:00+08:00")
	for _, now := range []time.Time{monday, sunday} {
		got, ok := periodKey("weekly", now)
		if !ok || got != "20260824" {
			t.Fatalf("weekly key = %q, %v", got, ok)
		}
	}
}

func TestLegacyEventCouponCategoryNameFollowsTrigger(t *testing.T) {
	tests := []struct {
		trigger string
		want    string
	}{
		{trigger: "low_spend", want: "周内低消券"},
		{trigger: "weekday_event", want: "周内低消券"},
		{trigger: "weekly_event", want: "周赛卡券"},
		{trigger: "monthly_event", want: "月赛卡券"},
		{trigger: "visit", want: ""},
	}
	for _, test := range tests {
		benefit := couponBenefit{CouponType: "event_ticket", Trigger: test.trigger}
		if got := legacyEventCouponCategoryName(benefit); got != test.want {
			t.Fatalf("trigger %s category = %q, want %q", test.trigger, got, test.want)
		}
	}
	if got := legacyEventCouponCategoryName(couponBenefit{
		CategoryID: 19, CouponType: "event_ticket", Trigger: "low_spend",
	}); got != "" {
		t.Fatalf("explicit category must not be overridden, got %q", got)
	}
}
