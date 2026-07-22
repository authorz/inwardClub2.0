package printer

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReceiptJob(t *testing.T) {
	paidAt := time.Date(2026, 7, 18, 15, 4, 5, 0, time.UTC)
	job := BuildReceiptJob("SN-42", Receipt{
		StoreID:         5,
		PaymentOrderID:  99,
		BusinessOrderNo: "BO-20260718-1",
		OrderType:       "food",
		AmountCent:      1500,
		PaidAt:          paidAt,
	})
	if job.DeviceSN != "SN-42" {
		t.Fatalf("device sn: got %q", job.DeviceSN)
	}
	if job.Template != ReceiptTemplate {
		t.Fatalf("template: got %q want %q", job.Template, ReceiptTemplate)
	}
	for _, want := range []string{"餐饮订单", "BO-20260718-1", "¥15.00", "2026-07-18 15:04:05"} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("content missing %q:\n%s", want, job.Content)
		}
	}
}

func TestOrderTypeLabel(t *testing.T) {
	cases := map[string]string{
		"food":               "餐饮订单",
		"activity":           "活动订单",
		"recharge":           "充值订单",
		"offline_collection": "门店收款",
		"something-else":     "订单",
	}
	for in, want := range cases {
		if got := orderTypeLabel(in); got != want {
			t.Fatalf("orderTypeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestYuan(t *testing.T) {
	cases := map[int64]string{
		0:    "¥0.00",
		5:    "¥0.05",
		1500: "¥15.00",
		1234: "¥12.34",
	}
	for cent, want := range cases {
		if got := yuan(cent); got != want {
			t.Fatalf("yuan(%d) = %q, want %q", cent, got, want)
		}
	}
}

// TestReceiptIdemKey pins the exactly-once key format: the print job dedups on
// the payment order id, matching the dispatcher's asynq TaskID contract.
func TestReceiptIdemKey(t *testing.T) {
	if got := receiptIdemKey(99); got != "payment:99:print-receipt" {
		t.Fatalf("idem key: got %q", got)
	}
}
