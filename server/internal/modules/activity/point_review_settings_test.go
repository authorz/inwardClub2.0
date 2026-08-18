package activity

import (
	"context"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

type pointReviewSettingsMemRepo struct{ settings PointReviewSettings }

func (r *pointReviewSettingsMemRepo) GetPointReviewSettings(context.Context) (PointReviewSettings, error) {
	return r.settings, nil
}

func (r *pointReviewSettingsMemRepo) UpdatePointReviewSettings(
	_ context.Context,
	pointsDivisor, belowBasePointsDivisor, coinPointsDivisor, _ int64,
	now time.Time,
) (PointReviewSettings, error) {
	r.settings.PointsDivisor = pointsDivisor
	r.settings.BelowBasePointsDivisor = belowBasePointsDivisor
	r.settings.CoinPointsDivisor = coinPointsDivisor
	r.settings.Version++
	r.settings.UpdatedAt = now
	return r.settings, nil
}

func TestPointReviewSettingsValidation(t *testing.T) {
	svc := NewPointReviewSettingsService(&pointReviewSettingsMemRepo{})
	_, err := svc.Update(context.Background(), UpdatePointReviewSettingsRequest{
		PointsDivisor: 0, BelowBasePointsDivisor: 2, CoinPointsDivisor: 2000,
	}, 1)
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid points divisor, got %v", err)
	}
	_, err = svc.Update(context.Background(), UpdatePointReviewSettingsRequest{
		PointsDivisor: 5, BelowBasePointsDivisor: 0, CoinPointsDivisor: 2000,
	}, 1)
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid below-base points divisor, got %v", err)
	}
}

func TestPointReviewSettingsUpdate(t *testing.T) {
	repo := &pointReviewSettingsMemRepo{settings: PointReviewSettings{Version: 1}}
	svc := NewPointReviewSettingsService(repo)
	got, err := svc.Update(context.Background(), UpdatePointReviewSettingsRequest{
		PointsDivisor: 3, BelowBasePointsDivisor: 4, CoinPointsDivisor: 1500,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.PointsDivisor != 3 || got.BelowBasePointsDivisor != 4 || got.CoinPointsDivisor != 1500 || got.Version != 2 {
		t.Fatalf("settings=%+v", got)
	}
}
