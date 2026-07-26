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

func (r *memoryRepository) UpdateGlobalSettings(_ context.Context, backgroundURL string, _ int64, now time.Time) (GlobalSettings, error) {
	r.settings = GlobalSettings{TableDefaultBackgroundURL: backgroundURL, UpdatedAt: &now}
	return r.settings, nil
}

func TestUpdateGlobalSettingsValidatesBackgroundURL(t *testing.T) {
	svc := NewService(&memoryRepository{})
	if _, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		TableDefaultBackgroundURL: "javascript:alert(1)",
	}, 1); err == nil {
		t.Fatal("expected invalid URL error")
	}
	got, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{
		TableDefaultBackgroundURL: " https://assets.inwardclub.com/table.png ",
	}, 1)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if got.TableDefaultBackgroundURL != "https://assets.inwardclub.com/table.png" {
		t.Fatalf("unexpected background URL %q", got.TableDefaultBackgroundURL)
	}
}

func TestUpdateGlobalSettingsAllowsClearingBackground(t *testing.T) {
	svc := NewService(&memoryRepository{})
	got, err := svc.Update(context.Background(), UpdateGlobalSettingsRequest{}, 1)
	if err != nil {
		t.Fatalf("clear settings: %v", err)
	}
	if got.TableDefaultBackgroundURL != "" {
		t.Fatalf("expected empty background URL, got %q", got.TableDefaultBackgroundURL)
	}
}
