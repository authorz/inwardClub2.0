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
		"苏打水", "2", "15.00元", "合计", "谢谢惠顾！", "<CUT>", "2026-07-18 23:04:05",
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
		Member:          "177****3915",
		VIPLevel:        8,
		CouponName:      "赛事券",
		PaidAt:          time.Date(2026, 8, 27, 4, 0, 0, 0, time.UTC),
	})
	for _, want := range []string{
		"赛事券使用", "手机号  177****3915", "会员等级  VIP8", "券名称  赛事券", "使用数量  1张",
	} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("event coupon receipt missing %q:\n%s", want, job.Content)
		}
	}
	for _, unwanted := range []string{"赛事门票券", "合计", "¥", "?0.00"} {
		if strings.Contains(job.Content, unwanted) {
			t.Fatalf("event coupon receipt contains %q:\n%s", unwanted, job.Content)
		}
	}
}

func TestBuildPointWithdrawalReceipt(t *testing.T) {
	job := BuildPointWithdrawalReceiptJob("SN-POINTS", true, PointWithdrawalReceipt{
		StoreID:      7,
		WithdrawalID: 88,
		StoreName:    "新壹街店",
		Member:       "177****3915",
		Points:       3000,
		BalanceAfter: 7200,
		WithdrawnAt:  time.Date(2026, 8, 27, 15, 20, 30, 0, time.UTC),
	})
	if job.DeviceSN != "SN-POINTS" || job.Template != PointWithdrawalReceiptTemplate || job.Silent {
		t.Fatalf("unexpected point-withdrawal job: %+v", job)
	}
	for _, want := range []string{
		"<IMG></IMG>", "<CB>InwardClub</CB>", "<CB>新壹街店</CB>",
		"2026-08-27 23:20:30", "手机尾号  177****3915", "提取积分  3000",
		"剩余积分  7200", "合计  3000", "<CB>请工作人员仔细检查核验！</CB>", "<CUT>",
	} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("point-withdrawal receipt missing %q:\n%s", want, job.Content)
		}
	}
	for _, unwanted := range []string{"订单号", "谢谢惠顾", "¥", "?"} {
		if strings.Contains(job.Content, unwanted) {
			t.Fatalf("point-withdrawal receipt contains %q:\n%s", unwanted, job.Content)
		}
	}
}

func TestBuildFoodReceiptKeepsQuantityAndAmountOnItemLine(t *testing.T) {
	job := BuildReceiptJob("SN-FOOD", true, Receipt{
		BusinessOrderNo: "BO-FOOD-1",
		OrderType:       "food",
		Items: []ReceiptItem{{
			Name: "88酒券套餐", Quantity: 1, SubtotalCent: 8800,
		}},
	})
	want := "88酒券套餐" + strings.Repeat(" ", 11) + "1" + strings.Repeat(" ", 3) + "88.00元"
	if !strings.Contains(job.Content, want+"\n") {
		t.Fatalf("food item columns are not on one line; want %q:\n%s", want, job.Content)
	}
	if strings.Contains(job.Content, "¥") {
		t.Fatalf("receipt contains unsupported yen sign:\n%s", job.Content)
	}
}

func TestReceiptPlainTextLinesFitPrinterWidth(t *testing.T) {
	job := BuildReceiptJob("SN-EVENT", true, Receipt{
		BusinessOrderNo: "ER322-1787843105234256898",
		OrderType:       "event_coupon",
		Member:          "177****3915",
		VIPLevel:        8,
		CouponName:      "这是一个很长但不能产生孤立字符的赛事券名称",
	})
	for _, line := range strings.Split(job.Content, "\n") {
		if strings.HasPrefix(line, "<") {
			continue
		}
		if width := receiptTextWidth(line); width > receiptLineWidth {
			t.Fatalf("receipt line width = %d, want <= %d: %q\n%s", width, receiptLineWidth, line, job.Content)
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
	if got := MaskedMember("13812345678", "昵称"); got != "138****5678" {
		t.Fatalf("masked phone = %q", got)
	}
	if got := MaskedMember("", "昵称"); got != "昵称" {
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
		0:    "0.00元",
		5:    "0.05元",
		1500: "15.00元",
		1234: "12.34元",
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
