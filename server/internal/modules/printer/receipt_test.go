package printer

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReceiptJob(t *testing.T) {
	paidAt := time.Date(2026, 7, 18, 15, 4, 5, 0, time.UTC)
	coinBalance := int64(12345)
	job := BuildReceiptJob("SN-42", true, Receipt{
		StoreID:         5,
		PaymentOrderID:  99,
		BusinessOrderNo: "BO-20260718-1",
		OrderType:       "food",
		AmountCent:      1500,
		PaidAt:          paidAt,
		StoreName:       "南滨公园店",
		Member:          "138****5678",
		VIPLevel:        3,
		PayMethod:       "coin",
		Points:          20,
		Remark:          "少冰，不要柠檬",
		CoinBalance:     &coinBalance,
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
		"手机号  138****5678", "会员等级                 VIP3", "消费方式", "金币",
		"赠送积分", "20", "金币余额", "12345", "订单备注：少冰，不要柠檬", "商品名称", "数量", "金额",
		"苏打水", "2", "¥15.00", "合计", "谢谢惠顾！", "<CUT>", "2026-07-18 23:04:05",
	} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("content missing %q:\n%s", want, job.Content)
		}
	}
	for _, unwanted := range []string{"会员下单", "金币支付", "手机尾号"} {
		if strings.Contains(job.Content, unwanted) {
			t.Fatalf("content contains obsolete wording %q:\n%s", unwanted, job.Content)
		}
	}
}

func TestBuildReceiptJobOmitsEmptyRemark(t *testing.T) {
	job := BuildReceiptJob("SN-42", true, Receipt{
		BusinessOrderNo: "BO-NO-REMARK-1",
		OrderType:       "food",
		Remark:          "   ",
		PaidAt:          time.Date(2026, 7, 18, 15, 4, 5, 0, time.UTC),
	})
	if strings.Contains(job.Content, "订单备注") {
		t.Fatalf("empty remark must not take receipt space:\n%s", job.Content)
	}
}

func TestBuildReceiptJobHonoursSoundSetting(t *testing.T) {
	receipt := Receipt{BusinessOrderNo: "BO-SOUND-1", PaidAt: time.Now()}
	if job := BuildReceiptJob("SN-42", true, receipt); job.Silent {
		t.Fatal("sound-enabled receipt must not be silent")
	}
	if job := BuildReceiptJob("SN-42", false, receipt); !job.Silent {
		t.Fatal("sound-disabled receipt must be silent")
	}
}

func TestBuildReceiptJobShowsWechatPaymentAndBalance(t *testing.T) {
	coinBalance := int64(8800)
	job := BuildReceiptJob("SN-42", true, Receipt{
		BusinessOrderNo: "BO-WECHAT-1",
		OrderType:       "food",
		VIPLevel:        1,
		PayMethod:       "wechat",
		CoinBalance:     &coinBalance,
		PaidAt:          time.Date(2026, 7, 18, 15, 4, 5, 0, time.UTC),
	})
	for _, want := range []string{"会员等级                 VIP1", "消费方式                 微信", "金币余额", "8800"} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("wechat receipt missing %q:\n%s", want, job.Content)
		}
	}
}

func TestBuildEventCouponReceipt(t *testing.T) {
	job := BuildReceiptJob("SN-EVENT", true, Receipt{
		BusinessOrderNo: "ER202608270001",
		OrderType:       "event_coupon",
		CouponName:      "周赛赛事券",
		PaidAt:          time.Date(2026, 8, 27, 4, 0, 0, 0, time.UTC),
	})
	for _, want := range []string{"赛事券使用", "周赛赛事券", "使用数量                 1 张", "合计  ¥0.00"} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("event coupon receipt missing %q:\n%s", want, job.Content)
		}
	}
}

func TestPaymentMethodLabel(t *testing.T) {
	cases := map[string]string{
		"wechat":  "微信",
		"coin":    "金币",
		"coupon":  "券",
		"voucher": "券",
		"unknown": "",
	}
	for in, want := range cases {
		if got := paymentMethodLabel(in); got != want {
			t.Fatalf("paymentMethodLabel(%q) = %q, want %q", in, got, want)
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
