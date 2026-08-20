package referral

import "testing"

func TestCommissionRemainderAccumulatesWholeCoins(t *testing.T) {
	coins, remainder := accrueCommission(0, 3900*1000)
	if coins != 3 || remainder != 900_000 {
		t.Fatalf("first accrual = %d coins, %d remainder", coins, remainder)
	}
	coins, remainder = accrueCommission(remainder, 1100*1000)
	if coins != 2 || remainder != 0 {
		t.Fatalf("second accrual = %d coins, %d remainder", coins, remainder)
	}
}

func TestCommissionRefundRestoresExactCumulativeResult(t *testing.T) {
	// ¥39 and ¥11 at 10% credited 5 coins in total. Refunding ¥39 must claw
	// back 4 coins and retain 0.1 coin, leaving the exact ¥11 commission state.
	coins, remainder := reverseCommission(0, 3900*1000)
	if coins != 4 || remainder != 100_000 {
		t.Fatalf("refund = %d coins, %d remainder", coins, remainder)
	}
}

func TestCommissionRefundCanUseExistingRemainder(t *testing.T) {
	coins, remainder := reverseCommission(800_000, 300*1000)
	if coins != 0 || remainder != 500_000 {
		t.Fatalf("refund = %d coins, %d remainder", coins, remainder)
	}
}
