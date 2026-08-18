package systemsetting

import (
	"context"
	"testing"
	"time"
)

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
	if got.PhoneChangeIntervalDays != 45 {
		t.Fatalf("unexpected phone change interval %d", got.PhoneChangeIntervalDays)
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
