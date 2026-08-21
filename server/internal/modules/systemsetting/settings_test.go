package systemsetting

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func stringPointer(value string) *string { return &value }

type memoryRepository struct{ settings GlobalSettings }

func (r *memoryRepository) GetGlobalSettings(context.Context) (GlobalSettings, error) {
	return r.settings, nil
}

func (r *memoryRepository) UpdateGlobalSettings(_ context.Context, settings GlobalSettings, _ int64, now time.Time) (GlobalSettings, error) {
	settings.UpdatedAt = &now
	r.settings = settings
	return r.settings, nil
}

func TestUpdateGlobalSettingsPersistsBusinessSettings(t *testing.T) {
	svc := NewService(&memoryRepository{})
	got, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		FirstRechargeDoublePointsEnabled:    true,
		RechargeDoublePointsThresholdAmount: 1200,
		RechargeNotice:                      "  首充与满额双倍积分不叠加。  ",
		PhoneChangeIntervalDays:             45,
	}, 1)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if !got.FirstRechargeDoublePointsEnabled {
		t.Fatal("expected first-recharge reward switch to be enabled")
	}
	if got.RechargeDoublePointsThresholdAmount != 1200 {
		t.Fatalf("unexpected recharge threshold %d", got.RechargeDoublePointsThresholdAmount)
	}
	if got.RechargeNotice != "首充与满额双倍积分不叠加。" {
		t.Fatalf("unexpected recharge notice %q", got.RechargeNotice)
	}
	if got.PhoneChangeIntervalDays != 45 {
		t.Fatalf("unexpected phone change interval %d", got.PhoneChangeIntervalDays)
	}
}

func TestUpdateGlobalSettingsRejectsOverlongRechargeNotice(t *testing.T) {
	svc := NewService(&memoryRepository{})
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		RechargeNotice:                      strings.Repeat("字", 201),
		PhoneChangeIntervalDays:             30,
	}, 1); err == nil {
		t.Fatal("expected overlong recharge notice error")
	}
}

func TestUpdateGlobalSettingsRejectsInvalidRechargeThreshold(t *testing.T) {
	svc := NewService(&memoryRepository{})
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{}, 1); err == nil {
		t.Fatal("expected invalid recharge threshold error")
	}
}

func TestUpdateGlobalSettingsRejectsInvalidPhoneChangeInterval(t *testing.T) {
	svc := NewService(&memoryRepository{})
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		PhoneChangeIntervalDays:             0,
	}, 1); err == nil {
		t.Fatal("expected invalid phone change interval error")
	}
}

func TestUpdateGlobalSettingsNormalizesFranchiseSources(t *testing.T) {
	svc := NewService(&memoryRepository{})
	got, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		PhoneChangeIntervalDays:             30,
		FranchiseInquirySources:             []string{" 美团 ", "微信小程序", "美团"},
		FranchiseHotline:                    " 400-888-8888 ",
	}, 1)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if len(got.FranchiseInquirySources) != 2 || got.FranchiseInquirySources[0] != "美团" {
		t.Fatalf("unexpected sources %#v", got.FranchiseInquirySources)
	}
	if got.FranchiseHotline != "400-888-8888" {
		t.Fatalf("unexpected hotline %q", got.FranchiseHotline)
	}
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		PhoneChangeIntervalDays:             30,
		FranchiseInquirySources:             []string{},
	}, 1); err == nil {
		t.Fatal("expected empty source validation error")
	}
}

func TestPrinterProviderSettingsPreserveAndMaskDeveloperKey(t *testing.T) {
	repo := &memoryRepository{}
	svc := NewService(repo)
	got, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		PhoneChangeIntervalDays:             30,
		PrinterDeveloperAccount:             stringPointer(" developer-account "),
		PrinterDeveloperKey:                 stringPointer(" developer-key "),
		PrinterAPIURL:                       stringPointer("https://open.xpyun.net/api/openapi/xprinter/"),
	}, 1)
	if err != nil {
		t.Fatalf("update printer settings: %v", err)
	}
	if got.PrinterDeveloperAccount != "developer-account" || !got.PrinterDeveloperKeyConfigured {
		t.Fatalf("settings = %#v", got)
	}
	if got.PrinterAPIURL != defaultPrinterAPIURL {
		t.Fatalf("api url = %q", got.PrinterAPIURL)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "developer-key") {
		t.Fatalf("developer key leaked in JSON: %s", raw)
	}

	account, key, apiURL, err := svc.PrinterProviderSettings(context.Background())
	if err != nil || account != "developer-account" || key != "developer-key" || apiURL != defaultPrinterAPIURL {
		t.Fatalf("provider settings = %q %q %q, %v", account, key, apiURL, err)
	}
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		RechargeDoublePointsThresholdAmount: 1000,
		PhoneChangeIntervalDays:             30,
	}, 1); err != nil {
		t.Fatalf("omitted printer settings should preserve existing values: %v", err)
	}
	if repo.settings.PrinterDeveloperKey != "developer-key" {
		t.Fatal("omitted key should remain unchanged")
	}
}
