package payment

import "testing"

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
			name:          "high value repeat recharge",
			coins:         1000,
			amountCent:    100000,
			thresholdCent: 100000,
			wantReasons:   []string{highValueRechargeRewardReason},
			wantAmounts:   []int64{2000},
		},
		{
			name:          "first recharge at threshold only gets high value reward",
			coins:         1000,
			amountCent:    100000,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{highValueRechargeRewardReason},
			wantAmounts:   []int64{2000},
		},
		{
			name:          "threshold is based on paid recharge amount",
			coins:         1000,
			amountCent:    99999,
			thresholdCent: 100000,
			firstRecharge: true,
			wantReasons:   []string{firstRechargeRewardReason},
			wantAmounts:   []int64{2000},
		},
		{
			name:          "custom threshold controls exclusion",
			coins:         800,
			amountCent:    80000,
			thresholdCent: 80000,
			firstRecharge: true,
			wantReasons:   []string{highValueRechargeRewardReason},
			wantAmounts:   []int64{1600},
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
