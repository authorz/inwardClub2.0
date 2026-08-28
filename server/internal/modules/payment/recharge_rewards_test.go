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
		points        int64
		amountCent    int64
		thresholdCent int64
		firstRecharge bool
		wantReasons   []string
		wantAmounts   []int64
	}{
		{
			name:          "first recharge doubles configured points",
			points:        600000,
			amountCent:    20000,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{600000},
		},
		{
			name:          "ordinary repeat recharge",
			points:        600000,
			amountCent:    20000,
			thresholdCent: 100000,
		},
		{
			name:          "repeat recharge at threshold gets no reward",
			points:        600000,
			amountCent:    100000,
			thresholdCent: 100000,
		},
		{
			name:          "first recharge at threshold gets no reward",
			points:        600000,
			amountCent:    100000,
			thresholdCent: 100000,
			firstRecharge: true,
		},
		{
			name:          "first recharge below threshold gets reward",
			points:        35000,
			amountCent:    99999,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{35000},
		},
		{
			name:          "custom threshold is an exclusive upper bound",
			points:        12000,
			amountCent:    79999,
			thresholdCent: 80000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{12000},
		},
		{
			name:          "repeat recharge below threshold gets no reward",
			points:        12000,
			amountCent:    79999,
			thresholdCent: 80000,
		},
		{
			name:          "first recharge without configured points gets no reward",
			amountCent:    50000,
			thresholdCent: 100000,
			firstRecharge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rechargePointRewards(tt.points, tt.amountCent, tt.thresholdCent, tt.firstRecharge)
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
