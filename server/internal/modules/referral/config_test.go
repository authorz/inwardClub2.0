package referral

import (
	"context"
	"testing"
)

type fakeRepository struct {
	rule Rule
	ok   bool
	err  error
}

func (f fakeRepository) ActiveRule(context.Context) (Rule, bool, error) {
	return f.rule, f.ok, f.err
}

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{
		"firstLowSpendRewardCoins":50,
		"firstLowSpendRewardPoints":2000,
		"commissionRateBasisPoints":1000
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.FirstLowSpendRewardCoins != 50 || cfg.FirstLowSpendRewardPoints != 2000 || cfg.CommissionRateBasisPoints != 1000 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseConfigRejectsOutOfRangeRate(t *testing.T) {
	_, err := ParseConfig([]byte(`{"commissionRateBasisPoints":10001}`))
	if err == nil {
		t.Fatal("expected an out-of-range error")
	}
}

func TestParseConfigRejectsEmptyRewards(t *testing.T) {
	_, err := ParseConfig([]byte(`{}`))
	if err == nil {
		t.Fatal("expected an empty-reward error")
	}
}

func TestServiceConfigHidesInactiveValues(t *testing.T) {
	svc := NewService(fakeRepository{})
	view, err := svc.Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if view.Enabled || view.FirstLowSpendRewardCoins != 0 || view.CommissionRateBasisPoints != 0 {
		t.Fatalf("inactive config leaked values: %+v", view)
	}
}

func TestServiceConfigReturnsActiveValues(t *testing.T) {
	svc := NewService(fakeRepository{ok: true, rule: Rule{Version: 2, Config: Config{
		FirstLowSpendRewardCoins: 50, FirstLowSpendRewardPoints: 2000, CommissionRateBasisPoints: 1000,
	}}})
	view, err := svc.Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if !view.Enabled || view.FirstLowSpendRewardCoins != 50 || view.FirstLowSpendRewardPoints != 2000 || view.CommissionRateBasisPoints != 1000 {
		t.Fatalf("unexpected view: %+v", view)
	}
}
