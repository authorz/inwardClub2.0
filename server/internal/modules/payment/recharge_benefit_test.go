package payment

import "testing"

func TestCustomRechargeCoinAmountUsesYuanNotCents(t *testing.T) {
	if got := customRechargeCoinAmount(50000); got != 500 {
		t.Fatalf("customRechargeCoinAmount(50000) = %d, want 500", got)
	}
}
