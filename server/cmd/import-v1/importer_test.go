package main

import "testing"

func TestCentsFromString(t *testing.T) {
	tests := map[string]int64{"0": 0, "1": 100, "39.9": 3990, "39.00": 3900, "-1.25": -125}
	for input, want := range tests {
		got, err := centsFromString(input)
		if err != nil || got != want {
			t.Fatalf("centsFromString(%q)=(%d,%v), want %d", input, got, err, want)
		}
	}
	if _, err := centsFromString("1.001"); err == nil {
		t.Fatal("expected excessive precision to fail")
	}
}

func TestLegacyCouponType(t *testing.T) {
	tests := map[string]string{
		"88畅饮酒券套餐":     "alcohol",
		"88酒券套餐":       "alcohol",
		"新客专享套餐（限购一次）": "event_ticket",
		"周赛酒劵":         "event_ticket",
		"月赛酒券":         "event_ticket",
	}
	for name, want := range tests {
		if got := legacyCouponType(name); got != want {
			t.Fatalf("legacyCouponType(%q)=%q, want %q", name, got, want)
		}
	}
}

func TestTruncateUTF8(t *testing.T) {
	if got := truncateUTF8("一二三四", 3); got != "一二三" {
		t.Fatalf("unexpected truncation %q", got)
	}
}

func TestMapCatalogItemStatus(t *testing.T) {
	tests := map[string]string{
		"active":   "published",
		"inactive": "unpublished",
		"sold_out": "unpublished",
	}
	for input, want := range tests {
		if got := mapCatalogItemStatus(input); got != want {
			t.Fatalf("mapCatalogItemStatus(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestResolveLegacyAvatarURL(t *testing.T) {
	tests := map[string]string{
		"":                                      "",
		"avatars/2026/08/member.jpg":            "https://api.inwardclub.com/storage/avatars/2026/08/member.jpg",
		"/avatars/2026/08/member.jpg":           "https://api.inwardclub.com/storage/avatars/2026/08/member.jpg",
		"https://wechat.example.com/member.jpg": "https://wechat.example.com/member.jpg",
	}
	for input, want := range tests {
		got, err := resolveLegacyAvatarURL("https://api.inwardclub.com/storage/", input)
		if err != nil || got != want {
			t.Fatalf("resolveLegacyAvatarURL(%q)=(%q,%v), want %q", input, got, err, want)
		}
	}
	if _, err := resolveLegacyAvatarURL("https://api.inwardclub.com/storage/", "file:///tmp/avatar.jpg"); err == nil {
		t.Fatal("expected non-HTTP absolute avatar URL to fail")
	}
}
