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

func TestPeriodKeyUsesShanghaiNaturalDayAndMonth(t *testing.T) {
	tests := []struct {
		name   string
		period string
		now    string
		want   string
	}{
		{name: "day before Shanghai midnight", period: "daily", now: "2026-08-31T15:59:59Z", want: "20260831"},
		{name: "day after Shanghai midnight", period: "daily", now: "2026-08-31T16:00:00Z", want: "20260901"},
		{name: "month before Shanghai midnight", period: "monthly", now: "2026-08-31T15:59:59Z", want: "202608"},
		{name: "month after Shanghai midnight", period: "monthly", now: "2026-08-31T16:00:00Z", want: "202609"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := periodKey(test.period, mustTime(t, test.now))
			if !ok || got != test.want {
				t.Fatalf("periodKey(%q) = %q, %v; want %q, true", test.period, got, ok, test.want)
			}
		})
	}
}

func TestBenefitKeySeparatesIndependentNaturalPeriods(t *testing.T) {
	now := mustTime(t, "2026-09-01T12:00:00+08:00")
	keys := make(map[string]string)
	for _, period := range []string{"daily", "weekly", "monthly"} {
		periodValue, ok := periodKey(period, now)
		if !ok {
			t.Fatalf("periodKey(%q) was rejected", period)
		}
		key := benefitKey("c", 7, 3, period, "low_spend:category:12", periodValue, 0)
		if previousPeriod, exists := keys[key]; exists {
			t.Fatalf("%s and %s benefits share idempotency key %q", previousPeriod, period, key)
		}
		keys[key] = period
	}
}

func TestLowSpendDailyKeyUsesBusinessSessionWhileWeekAndMonthStayNatural(t *testing.T) {
	evening := mustTime(t, "2026-08-31T23:00:00+08:00")
	morning := mustTime(t, "2026-09-01T07:00:00+08:00")
	afternoon := mustTime(t, "2026-09-01T16:00:00+08:00")
	morningSession := "20260831"
	afternoonSession := "20260901"

	morningDaily, ok := benefitPeriodKey("daily", "low_spend", morning, morningSession)
	if !ok {
		t.Fatal("morning daily key was rejected")
	}
	afternoonDaily, ok := benefitPeriodKey("daily", "low_spend", afternoon, afternoonSession)
	if !ok {
		t.Fatal("afternoon daily key was rejected")
	}
	if morningDaily == afternoonDaily {
		t.Fatalf("separate business sessions share daily key %q", morningDaily)
	}
	eveningDaily, _ := benefitPeriodKey("daily", "low_spend", evening, morningSession)
	if eveningDaily != morningDaily {
		t.Fatalf("one overnight business session has two daily keys: %q != %q", eveningDaily, morningDaily)
	}
	legacyAfternoonKey, _ := periodKey("daily", afternoon)
	if afternoonDaily != legacyAfternoonKey {
		t.Fatalf("afternoon session key %q breaks compatibility with existing daily key %q", afternoonDaily, legacyAfternoonKey)
	}

	for _, period := range []string{"weekly", "monthly"} {
		morningKey, _ := benefitPeriodKey(period, "low_spend", morning, morningSession)
		afternoonKey, _ := benefitPeriodKey(period, "low_spend", afternoon, afternoonSession)
		if morningKey != afternoonKey {
			t.Fatalf("%s keys differ within one natural period: %q != %q", period, morningKey, afternoonKey)
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
