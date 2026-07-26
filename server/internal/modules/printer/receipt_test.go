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
		StoreName:       "南滨公园店",
		Member:          "138****5678",
		Points:          20,
		Items: []ReceiptItem{{
			Name: "苏打水", Quantity: 2, SubtotalCent: 1500,
		}},
	})
	if job.DeviceSN != "SN-42" {
		t.Fatalf("device sn: got %q", job.DeviceSN)
	}
	if job.Template != ReceiptTemplate {
		t.Fatalf("template: got %q want %q", job.Template, ReceiptTemplate)
	}
	for _, want := range []string{
		"<IMG></IMG>", "<CB>InwardClub</CB>", "南滨公园店", "订单号：BO-20260718-1",
		"手机尾号", "138****5678", "赠送积分", "20", "商品名称", "数量", "金额",
		"苏打水", "2", "¥15.00", "合计", "谢谢惠顾！", "<CUT>", "2026-07-18 23:04:05",
	} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("content missing %q:\n%s", want, job.Content)
		}
	}
}

func TestMaskedMember(t *testing.T) {
	if got := maskedMember("13812345678", "昵称"); got != "138****5678" {
		t.Fatalf("masked phone = %q", got)
	}
	if got := maskedMember("", "昵称"); got != "昵称" {
		t.Fatalf("nickname fallback = %q", got)
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

// TestReceiptIdemKey pins the per-payment, per-device exactly-once key format.
func TestReceiptIdemKey(t *testing.T) {
	if got := receiptIdemKey(99, 2); got != "payment:99:printer:2:print-receipt" {
		t.Fatalf("idem key: got %q", got)
	}
}
