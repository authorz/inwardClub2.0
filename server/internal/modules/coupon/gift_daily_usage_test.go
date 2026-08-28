package coupon

import (
	"database/sql"
	"testing"
)

func TestGiftedCouponDailyLimitSourceAndUnlimitedSemantics(t *testing.T) {
	tests := []struct {
		name          string
		grantedByType string
		configured    sql.NullInt64
		wantLimit     int
		wantLimited   bool
	}{
		{
			name:          "purchased coupon always bypasses configured limit",
			grantedByType: "purchase",
			configured:    sql.NullInt64{Int64: 1, Valid: true},
		},
		{
			name:          "gifted coupon without a configured rule is unrestricted",
			grantedByType: "system",
			configured:    sql.NullInt64{},
		},
		{
			name:          "gifted coupon uses the configured positive daily limit",
			grantedByType: "system",
			configured:    sql.NullInt64{Int64: 2, Valid: true},
			wantLimit:     2,
			wantLimited:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limit, limited := giftedCouponDailyLimit(test.grantedByType, test.configured)
			if limit != test.wantLimit || limited != test.wantLimited {
				t.Fatalf("giftedCouponDailyLimit() = (%d, %v), want (%d, %v)",
					limit, limited, test.wantLimit, test.wantLimited)
			}
		})
	}
}
