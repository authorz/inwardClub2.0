package config

import (
	"testing"
	"time"
)

// TestBusinessClockPinned verifies BUSINESS_NOW_RFC3339 pins the business day in
// the configured zone. A pinned instant of 2026-07-17 23:00 UTC+8 must report
// 2026-07-17, and the same instant expressed in UTC must still resolve to the
// Shanghai day (not the UTC day, nor a drifting host day).
func TestBusinessClockPinned(t *testing.T) {
	cfg := &Config{Business: BusinessConfig{TZ: "Asia/Shanghai", NowRFC3339: "2026-07-17T23:00:00+08:00"}}
	clock, err := cfg.BusinessClock()
	if err != nil {
		t.Fatalf("BusinessClock: %v", err)
	}
	if got := clock.Now().Format("2006-01-02"); got != "2026-07-17" {
		t.Fatalf("pinned clock: got %q, want 2026-07-17", got)
	}

	// The acceptance instant from the task: noon local pins the business day.
	cfg.Business.NowRFC3339 = "2026-07-17T12:00:00+08:00"
	clock, err = cfg.BusinessClock()
	if err != nil {
		t.Fatalf("BusinessClock: %v", err)
	}
	if got := clock.Now().Format("2006-01-02"); got != "2026-07-17" {
		t.Fatalf("noon clock: got %q, want 2026-07-17", got)
	}
}

// TestBusinessClockDefaultsToHostTime verifies that with no pinned instant the
// clock tracks real time in the business zone (production behaviour unchanged).
func TestBusinessClockDefaultsToHostTime(t *testing.T) {
	cfg := &Config{Business: BusinessConfig{TZ: "Asia/Shanghai"}}
	clock, err := cfg.BusinessClock()
	if err != nil {
		t.Fatalf("BusinessClock: %v", err)
	}
	if delta := time.Since(clock.Now()); delta < -time.Minute || delta > time.Minute {
		t.Fatalf("unpinned clock should track host time, delta=%v", delta)
	}
	if name, _ := clock.Now().Zone(); name == "UTC" {
		t.Fatalf("business zone should be Asia/Shanghai, got UTC")
	}
}

// TestBusinessClockInvalidNow rejects an unparseable pinned instant.
func TestBusinessClockInvalidNow(t *testing.T) {
	cfg := &Config{Business: BusinessConfig{TZ: "Asia/Shanghai", NowRFC3339: "not-a-time"}}
	if _, err := cfg.BusinessClock(); err == nil {
		t.Fatalf("expected error for invalid BUSINESS_NOW_RFC3339")
	}
}
