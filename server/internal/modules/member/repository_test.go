package member

import (
	"database/sql"
	"testing"
	"time"
)

type scanFunc func(...any) error

func (f scanFunc) Scan(dest ...any) error {
	return f(dest...)
}

func TestScanRechargeProductIncludesCouponTemplateID(t *testing.T) {
	product, err := scanRechargeProduct(scanFunc(func(dest ...any) error {
		if len(dest) != 7 {
			t.Fatalf("expected 7 scan destinations, got %d", len(dest))
		}
		couponTemplateID := int64(29)
		*dest[0].(*int64) = 1
		*dest[1].(*int64) = 10000
		*dest[2].(*int64) = 12000
		*dest[3].(*int64) = 300
		*dest[4].(**int64) = &couponTemplateID
		*dest[5].(*int) = 2
		*dest[6].(*string) = StatusActive
		return nil
	}))
	if err != nil {
		t.Fatalf("scan recharge product: %v", err)
	}
	if product.CouponTemplateID == nil || *product.CouponTemplateID != 29 {
		t.Fatalf("expected coupon template ID 29, got %v", product.CouponTemplateID)
	}
}

func TestPhoneChangeCooldown(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	changedAt := time.Date(2026, 8, 1, 10, 0, 0, 0, location)
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, location)

	next, blocked := phoneChangeCooldown(
		"13800001111",
		"13900002222",
		sql.NullTime{Time: changedAt, Valid: true},
		30,
		now,
	)
	if !blocked || !next.Equal(changedAt.AddDate(0, 0, 30)) {
		t.Fatalf("expected cooldown until %v, got blocked=%v next=%v", changedAt.AddDate(0, 0, 30), blocked, next)
	}

	if _, blocked := phoneChangeCooldown("13800001111", "13800001111", sql.NullTime{Time: changedAt, Valid: true}, 30, now); blocked {
		t.Fatal("same phone must not be treated as a change")
	}
	if _, blocked := phoneChangeCooldown("", "13900002222", sql.NullTime{}, 30, now); blocked {
		t.Fatal("first phone binding must not be blocked")
	}
	if _, blocked := phoneChangeCooldown("13800001111", "13900002222", sql.NullTime{Time: changedAt, Valid: true}, 7, now); blocked {
		t.Fatal("change at the exact cooldown boundary must be allowed")
	}
}
