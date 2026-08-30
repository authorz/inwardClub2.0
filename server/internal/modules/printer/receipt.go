package printer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/outbox"
)

// ReceiptTopic is the outbox topic — and thus the asynq task type — the worker's
// receipt handler consumes. It must stay in sync with cmd/worker.TaskPrint.
const ReceiptTopic = "print:receipt"

// Receipt templates tag print jobs for logs and future provider-specific
// handling. Xpyun receives the already-rendered Content.
const (
	ReceiptTemplate                = "order-receipt"
	PointWithdrawalReceiptTemplate = "point-withdrawal-receipt"
)

var receiptLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

const (
	receiptLineWidth         = 32
	receiptItemNameWidth     = 18
	receiptItemQuantityWidth = 4
	receiptItemAmountWidth   = 10
	receiptRule              = "--------------------------------\n"
)

// Receipt is the settled-order snapshot rendered for a store. Member contains
// only the masked phone number or nickname used on the paper slip.
type Receipt struct {
	StoreID         int64
	PaymentOrderID  int64
	BusinessOrderID int64
	BusinessOrderNo string
	// OrderType mirrors business_orders.order_type (food | activity | recharge |
	// offline_collection); it only picks the printed heading.
	OrderType  string
	AmountCent int64
	PaidAt     time.Time
	StoreName  string
	Member     string
	VIPLevel   int
	PayMethod  string
	Points     int64
	Remark     string
	// CouponTypeName is the coupon_categories.name snapshot resolved when a
	// direct event coupon is consumed.
	CouponTypeName string
	// CoinBalance is the member's available coin balance after this payment.
	CoinBalance *int64
	Items       []ReceiptItem
}

// ReceiptItem is one snapshotted food-order line rendered on the store slip.
type ReceiptItem struct {
	Name         string
	Quantity     int
	SubtotalCent int64
}

// PointWithdrawalReceipt is the committed points-debit snapshot printed by the
// selected store. Member contains a masked phone number or nickname.
type PointWithdrawalReceipt struct {
	StoreID      int64
	WithdrawalID int64
	StoreName    string
	Member       string
	VIPLevel     int
	Points       int64
	BalanceAfter int64
	WithdrawnAt  time.Time
}

// WriteReceipt resolves every active printer bound to the order's store and,
// when at least one exists, appends one print:receipt outbox event per device in
// the caller's transaction — so the receipt commits atomically with the
// settlement and can never fire on a rolled-back one. A store with no active
// device prints nothing (returns nil), so a settlement never fails for the lack
// of a printer.
func WriteReceipt(ctx context.Context, tx *sql.Tx, r Receipt) error {
	devices, err := activeDevices(ctx, tx, r.StoreID)
	if err != nil || len(devices) == 0 {
		return err
	}
	if r.OrderType == "food" {
		r, err = hydrateFoodReceipt(ctx, tx, r)
		if err != nil {
			return err
		}
	}
	return writeReceiptJobs(ctx, tx, r, devices, func(deviceID int64) string {
		return receiptIdemKey(r.PaymentOrderID, deviceID)
	})
}

// WriteEventCouponReceipt queues one receipt per active store printer for a
// direct event-coupon use. Redemption-based keys keep it independent from the
// payment-order receipt namespace.
func WriteEventCouponReceipt(ctx context.Context, tx *sql.Tx, redemptionID int64, r Receipt) error {
	devices, err := activeDevices(ctx, tx, r.StoreID)
	if err != nil || len(devices) == 0 {
		return err
	}
	return writeReceiptJobs(ctx, tx, r, devices, func(deviceID int64) string {
		return fmt.Sprintf("coupon-redemption:%d:printer:%d:print-receipt", redemptionID, deviceID)
	})
}

func writeReceiptJobs(ctx context.Context, tx *sql.Tx, r Receipt, devices []activeDevice, idemKeyFor func(int64) string) error {
	return writePrintJobs(ctx, tx, r.StoreID, r.BusinessOrderNo, devices, idemKeyFor, func(device activeDevice) Job {
		return BuildReceiptJob(device.sn, device.soundEnabled, r)
	})
}

// WritePointWithdrawalReceipt queues the points-withdrawal slip for every
// active printer bound to the selected store. The caller owns the transaction,
// so the debit, ledger, withdrawal record and print event commit atomically.
func WritePointWithdrawalReceipt(ctx context.Context, tx *sql.Tx, r PointWithdrawalReceipt) error {
	devices, err := activeDevices(ctx, tx, r.StoreID)
	if err != nil || len(devices) == 0 {
		return err
	}
	businessOrderNo := fmt.Sprintf("PW%d", r.WithdrawalID)
	return writePrintJobs(ctx, tx, r.StoreID, businessOrderNo, devices, func(deviceID int64) string {
		return fmt.Sprintf("point-withdrawal:%d:printer:%d:print-receipt", r.WithdrawalID, deviceID)
	}, func(device activeDevice) Job {
		return BuildPointWithdrawalReceiptJob(device.sn, device.soundEnabled, r)
	})
}

func writePrintJobs(
	ctx context.Context,
	tx *sql.Tx,
	storeID int64,
	businessOrderNo string,
	devices []activeDevice,
	idemKeyFor func(int64) string,
	jobFor func(activeDevice) Job,
) error {
	for _, device := range devices {
		job := jobFor(device)
		idemKey := idemKeyFor(device.id)
		jobID, err := createPrintJob(ctx, tx, storeID, device.id, businessOrderNo, idemKey, job)
		if err != nil {
			return err
		}
		job.ID = jobID
		if err := outbox.Write(
			ctx,
			tx,
			ReceiptTopic,
			job,
			idemKey,
		); err != nil {
			return err
		}
	}
	return nil
}

// hydrateFoodReceipt loads only order-time snapshots, never the current catalog,
// so edits made after checkout cannot change the printed slip.
func hydrateFoodReceipt(ctx context.Context, tx *sql.Tx, r Receipt) (Receipt, error) {
	const header = `SELECT COALESCE(s.name, ''), COALESCE(m.phone, ''),
			COALESCE(m.nickname, ''),
			COALESCE(mt.level, (
				SELECT base.level FROM membership_tiers base
				WHERE base.status = 'active'
				ORDER BY base.level ASC, base.id ASC LIMIT 1
			), 0),
			COALESCE(po.pay_method, ''),
			COALESCE(coins.available_amount, 0),
			fo.points_earned, COALESCE(fo.remark, '')
		FROM food_orders fo
		JOIN stores s ON s.id = fo.store_id
		JOIN members m ON m.id = fo.member_id
		LEFT JOIN membership_tiers mt ON mt.id = m.current_tier_id
		LEFT JOIN payment_orders po ON po.id = ? AND po.business_order_id = fo.business_order_id
		LEFT JOIN wallet_accounts coins ON coins.member_id = fo.member_id AND coins.asset_type = 'coins'
		WHERE fo.business_order_id = ?`
	var (
		phone, nickname string
		coinBalance     int64
	)
	err := tx.QueryRowContext(ctx, header, r.PaymentOrderID, r.BusinessOrderID).
		Scan(
			&r.StoreName,
			&phone,
			&nickname,
			&r.VIPLevel,
			&r.PayMethod,
			&coinBalance,
			&r.Points,
			&r.Remark,
		)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, apperr.NotFound("food order not found")
	}
	if err != nil {
		return Receipt{}, apperr.Internal(err)
	}
	r.Member = MaskedMember(phone, nickname)
	r.CoinBalance = &coinBalance

	const lines = `SELECT name_snapshot, quantity, subtotal_cent
		FROM food_order_items
		WHERE food_order_id = (SELECT id FROM food_orders WHERE business_order_id = ?)
		ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, lines, r.BusinessOrderID)
	if err != nil {
		return Receipt{}, apperr.Internal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ReceiptItem
		if err := rows.Scan(&item.Name, &item.Quantity, &item.SubtotalCent); err != nil {
			return Receipt{}, apperr.Internal(err)
		}
		r.Items = append(r.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Receipt{}, apperr.Internal(err)
	}
	return r, nil
}

// MaskedMember returns the receipt-safe member identifier shared by payment
// and coupon receipt producers.
func MaskedMember(phone, nickname string) string {
	prefix, digits := "", phone
	if len(phone) > 0 && phone[0] == '+' {
		prefix, digits = "+", phone[1:]
	}
	if len(digits) >= 7 {
		return prefix + digits[:3] + "****" + digits[len(digits)-4:]
	}
	if strings.TrimSpace(nickname) != "" {
		return nickname
	}
	return "会员"
}

// receiptIdemKey is unique per payment and printer device. This allows every
// active device at the store to print once while still deduplicating retries.
func receiptIdemKey(paymentOrderID, deviceID int64) string {
	return fmt.Sprintf("payment:%d:printer:%d:print-receipt", paymentOrderID, deviceID)
}

// BuildReceiptJob renders r onto a printer.Job for device sn. It is pure so the
// receipt body can be unit-tested without a database.
func BuildReceiptJob(sn string, soundEnabled bool, r Receipt) Job {
	return Job{DeviceSN: sn, Template: ReceiptTemplate, Content: renderReceipt(r), Silent: !soundEnabled}
}

// BuildPointWithdrawalReceiptJob renders a points-withdrawal slip without
// touching the database, keeping the paper contract directly testable.
func BuildPointWithdrawalReceiptJob(sn string, soundEnabled bool, r PointWithdrawalReceipt) Job {
	return Job{
		DeviceSN: sn,
		Template: PointWithdrawalReceiptTemplate,
		Content:  renderPointWithdrawalReceipt(r),
		Silent:   !soundEnabled,
	}
}

type activeDevice struct {
	id           int64
	sn           string
	soundEnabled bool
}

// activeDevices returns all active printers bound to the order's store.
func activeDevices(ctx context.Context, tx *sql.Tx, storeID int64) ([]activeDevice, error) {
	const q = `SELECT id, device_sn, sound_enabled FROM printer_devices
		WHERE store_id = ? AND status = ? ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, q, storeID, StatusActive)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	var devices []activeDevice
	for rows.Next() {
		var device activeDevice
		if err := rows.Scan(&device.id, &device.sn, &device.soundEnabled); err != nil {
			return nil, apperr.Internal(err)
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return devices, nil
}

// renderReceipt renders the compact receipt body that lands on paper (the Xpyun
// print API takes pre-rendered content). PaidAt is stored in UTC and rendered in
// the store's operating timezone (Asia/Shanghai).
func renderReceipt(r Receipt) string {
	var b strings.Builder
	writeReceiptHeader(&b, r.StoreName, orderTypeLabel(r.OrderType))
	writeReceiptField(&b, "订单号：", r.BusinessOrderNo)
	fmt.Fprintf(&b, "%s\n", r.PaidAt.In(receiptLocation).Format("2006-01-02 15:04:05"))
	b.WriteString(receiptRule)
	if r.Member != "" {
		writeReceiptField(&b, "手机号", r.Member)
	}
	if r.OrderType == "food" {
		b.WriteString(receiptRule)
		if r.VIPLevel > 0 {
			fmt.Fprintf(&b, "会员等级                 VIP%d\n", r.VIPLevel)
		}
		if payMethod := paymentMethodLabel(r.PayMethod); payMethod != "" {
			fmt.Fprintf(&b, "消费方式                 %s\n", payMethod)
		}
		fmt.Fprintf(&b, "赠送积分                 %d\n", r.Points)
		if r.CoinBalance != nil {
			fmt.Fprintf(&b, "金币余额                 %d\n", *r.CoinBalance)
		}
		if remark := strings.TrimSpace(r.Remark); remark != "" {
			fmt.Fprintf(&b, "订单备注：%s\n", remark)
		}
		b.WriteString(receiptRule)
		b.WriteString(padReceiptRight("商品名称", receiptItemNameWidth))
		b.WriteString(padReceiptLeft("数量", receiptItemQuantityWidth))
		b.WriteString(padReceiptLeft("金额", receiptItemAmountWidth))
		b.WriteByte('\n')
		for _, item := range r.Items {
			writeReceiptItem(&b, item)
		}
	} else if r.OrderType == "event_coupon" && strings.TrimSpace(r.CouponTypeName) != "" {
		if r.VIPLevel > 0 {
			writeReceiptField(&b, "会员等级", fmt.Sprintf("VIP%d", r.VIPLevel))
		}
		writeReceiptField(&b, "券名称", strings.TrimSpace(r.CouponTypeName))
		writeReceiptField(&b, "使用数量", "1张")
	}
	b.WriteString(receiptRule)
	if r.OrderType != "event_coupon" {
		writeReceiptField(&b, "合计", yuan(r.AmountCent))
	}
	b.WriteString("谢谢惠顾！\n")
	b.WriteString("<CUT>")
	return b.String()
}

func renderPointWithdrawalReceipt(r PointWithdrawalReceipt) string {
	var b strings.Builder
	writeReceiptHeader(&b, r.StoreName, "积分提取")
	fmt.Fprintf(&b, "%s\n", r.WithdrawnAt.In(receiptLocation).Format("2006-01-02 15:04:05"))
	b.WriteString(receiptRule)
	writeReceiptField(&b, "手机尾号", r.Member)
	if r.VIPLevel > 0 {
		writeReceiptField(&b, "会员等级", fmt.Sprintf("VIP%d", r.VIPLevel))
	}
	b.WriteString(receiptRule)
	writeReceiptField(&b, "提取积分", strconv.FormatInt(r.Points, 10))
	b.WriteString(receiptRule)
	writeReceiptField(&b, "剩余积分", strconv.FormatInt(r.BalanceAfter, 10))
	b.WriteString(receiptRule)
	writeReceiptField(&b, "合计", strconv.FormatInt(r.Points, 10))
	b.WriteString("请工作人员仔细检查核验！\n")
	b.WriteString("<CUT>")
	return b.String()
}

func writeReceiptHeader(b *strings.Builder, storeName, fallbackTitle string) {
	b.WriteString("<IMG></IMG>\n")
	b.WriteString("<CB>InwardClub</CB>\n")
	if storeName = strings.TrimSpace(storeName); storeName != "" {
		fmt.Fprintf(b, "<CB>%s</CB>\n", storeName)
		return
	}
	if fallbackTitle = strings.TrimSpace(fallbackTitle); fallbackTitle != "" {
		b.WriteString(fallbackTitle)
		b.WriteByte('\n')
	}
}

func writeReceiptField(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	separator := "  "
	if strings.HasSuffix(label, "：") {
		separator = ""
	}
	line := label + separator + value
	if receiptTextWidth(line) <= receiptLineWidth {
		b.WriteString(line)
		b.WriteByte('\n')
		return
	}
	b.WriteString(label)
	b.WriteByte('\n')
	for _, part := range splitReceiptText(value, receiptLineWidth) {
		b.WriteString(part)
		b.WriteByte('\n')
	}
}

func writeReceiptItem(b *strings.Builder, item ReceiptItem) {
	parts := splitReceiptText(strings.TrimSpace(item.Name), receiptItemNameWidth)
	if len(parts) == 0 {
		parts = []string{""}
	}
	for _, part := range parts[:len(parts)-1] {
		b.WriteString(part)
		b.WriteByte('\n')
	}
	b.WriteString(padReceiptRight(parts[len(parts)-1], receiptItemNameWidth))
	b.WriteString(padReceiptLeft(strconv.Itoa(item.Quantity), receiptItemQuantityWidth))
	b.WriteString(padReceiptLeft(yuan(item.SubtotalCent), receiptItemAmountWidth))
	b.WriteByte('\n')
}

func splitReceiptText(value string, width int) []string {
	if value == "" || width <= 0 {
		return nil
	}
	parts := make([]string, 0, 1)
	var part strings.Builder
	partWidth := 0
	for _, r := range value {
		runeWidth := receiptRuneWidth(r)
		if partWidth > 0 && partWidth+runeWidth > width {
			parts = append(parts, part.String())
			part.Reset()
			partWidth = 0
		}
		part.WriteRune(r)
		partWidth += runeWidth
	}
	if part.Len() > 0 {
		parts = append(parts, part.String())
	}
	return parts
}

func padReceiptLeft(value string, width int) string {
	if padding := width - receiptTextWidth(value); padding > 0 {
		return strings.Repeat(" ", padding) + value
	}
	return value
}

func padReceiptRight(value string, width int) string {
	if padding := width - receiptTextWidth(value); padding > 0 {
		return value + strings.Repeat(" ", padding)
	}
	return value
}

func receiptTextWidth(value string) int {
	width := 0
	for _, r := range value {
		width += receiptRuneWidth(r)
	}
	return width
}

func receiptRuneWidth(r rune) int {
	if r <= 0x7f {
		return 1
	}
	return 2
}

// paymentMethodLabel maps persisted payment channels to administrator-facing
// receipt wording. coupon/voucher are accepted for forward compatibility with
// food-order coupon settlement.
func paymentMethodLabel(payMethod string) string {
	switch strings.ToLower(strings.TrimSpace(payMethod)) {
	case "wechat":
		return "微信"
	case "coin", "coins", "balance":
		return "金币"
	case "coupon", "voucher":
		return "券"
	default:
		return ""
	}
}

// orderTypeLabel maps a business_orders.order_type to its printed heading.
func orderTypeLabel(orderType string) string {
	switch orderType {
	case "food":
		return "餐饮订单"
	case "activity":
		return "活动订单"
	case "event_coupon":
		return "赛事券使用"
	case "recharge":
		return "充值订单"
	case "offline_collection":
		return "门店收款"
	default:
		return "订单"
	}
}

// yuan renders integer cents with a printer-safe RMB unit. The target thermal
// printer replaces the half-width yen sign with a question mark.
func yuan(cent int64) string {
	sign := ""
	if cent < 0 {
		sign, cent = "-", -cent
	}
	return fmt.Sprintf("%s%d.%02d元", sign, cent/100, cent%100)
}
