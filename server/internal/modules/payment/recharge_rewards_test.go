package payment

import (
	"database/sql"
	"testing"
)

func TestStoreReceiptEligibleExcludesRecharge(t *testing.T) {
	storeID := sql.NullInt64{Int64: 1, Valid: true}
	if storeReceiptEligible(orderTypeRecharge, storeID) {
		t.Fatal("store-attributed recharge must not print a store receipt")
	}
	if !storeReceiptEligible(orderTypeActivity, storeID) {
		t.Fatal("store activity should remain eligible for a store receipt")
	}
	if storeReceiptEligible(orderTypeActivity, sql.NullInt64{}) {
		t.Fatal("store-less activity must not print a store receipt")
	}
}

func TestRechargePointRewards(t *testing.T) {
	tests := []struct {
		name          string
		coins         int64
		amountCent    int64
		thresholdCent int64
		firstRecharge bool
		wantReasons   []string
		wantAmounts   []int64
	}{
		{
			name:          "ordinary first recharge",
			coins:         200,
			amountCent:    20000,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{400},
		},
		{
			name:          "ordinary repeat recharge",
			coins:         200,
			amountCent:    20000,
			thresholdCent: 100000,
		},
		{
			name:          "repeat recharge at threshold gets no reward",
			coins:         1000,
			amountCent:    100000,
			thresholdCent: 100000,
		},
		{
			name:          "first recharge at threshold gets no reward",
			coins:         1000,
			amountCent:    100000,
			thresholdCent: 100000,
			firstRecharge: true,
		},
		{
			name:          "first recharge below threshold gets reward",
			coins:         1000,
			amountCent:    99999,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{2000},
		},
		{
			name:          "custom threshold is an exclusive upper bound",
			coins:         800,
			amountCent:    79999,
			thresholdCent: 80000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{1600},
		},
		{
			name:          "repeat recharge below threshold gets no reward",
			coins:         800,
			amountCent:    79999,
			thresholdCent: 80000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rechargePointRewards(tt.coins, tt.amountCent, tt.thresholdCent, tt.firstRecharge)
			if len(got) != len(tt.wantReasons) {
				t.Fatalf("got %d rewards, want %d: %+v", len(got), len(tt.wantReasons), got)
			}
			for i := range got {
				if got[i].reason != tt.wantReasons[i] || got[i].amount != tt.wantAmounts[i] {
					t.Fatalf("reward %d = %+v, want reason=%s amount=%d", i, got[i], tt.wantReasons[i], tt.wantAmounts[i])
				}
			}
		})
	}
}
