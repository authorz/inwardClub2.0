package printer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/outbox"
)

// ReceiptTopic is the outbox topic — and thus the asynq task type — the worker's
// receipt handler consumes. It must stay in sync with cmd/worker.TaskPrint.
const ReceiptTopic = "print:receipt"

// ReceiptTemplate tags a settled-order receipt Job. The Xpyun print API ignores
// the template (Content is already rendered), but the worker logs it and future
// providers may branch on it.
const ReceiptTemplate = "order-receipt"

var receiptLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

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
	Points     int64
	Items      []ReceiptItem
}

// ReceiptItem is one snapshotted food-order line rendered on the store slip.
type ReceiptItem struct {
	Name         string
	Quantity     int
	SubtotalCent int64
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
	for _, device := range devices {
		if err := outbox.Write(
			ctx,
			tx,
			ReceiptTopic,
			BuildReceiptJob(device.sn, r),
			receiptIdemKey(r.PaymentOrderID, device.id),
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
			COALESCE(m.nickname, ''), fo.points_earned
		FROM food_orders fo
		JOIN stores s ON s.id = fo.store_id
		JOIN members m ON m.id = fo.member_id
		WHERE fo.business_order_id = ?`
	var phone, nickname string
	err := tx.QueryRowContext(ctx, header, r.BusinessOrderID).
		Scan(&r.StoreName, &phone, &nickname, &r.Points)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, apperr.NotFound("food order not found")
	}
	if err != nil {
		return Receipt{}, apperr.Internal(err)
	}
	r.Member = maskedMember(phone, nickname)

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

func maskedMember(phone, nickname string) string {
	if len(phone) >= 7 {
		return phone[:3] + "****" + phone[len(phone)-4:]
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
func BuildReceiptJob(sn string, r Receipt) Job {
	return Job{DeviceSN: sn, Template: ReceiptTemplate, Content: renderReceipt(r)}
}

type activeDevice struct {
	id int64
	sn string
}

// activeDevices returns all active printers bound to the order's store.
func activeDevices(ctx context.Context, tx *sql.Tx, storeID int64) ([]activeDevice, error) {
	const q = `SELECT id, device_sn FROM printer_devices
		WHERE store_id = ? AND status = ? ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, q, storeID, StatusActive)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	var devices []activeDevice
	for rows.Next() {
		var device activeDevice
		if err := rows.Scan(&device.id, &device.sn); err != nil {
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
	const rule = "--------------------------------\n"
	var b strings.Builder
	b.WriteString("<IMG></IMG>\n")
	b.WriteString("<CB>InwardClub</CB>\n")
	if r.StoreName != "" {
		fmt.Fprintf(&b, "<CB>%s</CB>\n", r.StoreName)
	} else {
		b.WriteString(orderTypeLabel(r.OrderType))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "订单号：%s\n", r.BusinessOrderNo)
	fmt.Fprintf(&b, "%s\n", r.PaidAt.In(receiptLocation).Format("2006-01-02 15:04:05"))
	b.WriteString(rule)
	if r.Member != "" {
		fmt.Fprintf(&b, "手机尾号                 %s\n", r.Member)
	}
	if r.OrderType == "food" {
		b.WriteString(rule)
		fmt.Fprintf(&b, "赠送积分                 %d\n", r.Points)
		b.WriteString(rule)
		b.WriteString("商品名称          数量    金额\n")
		for _, item := range r.Items {
			fmt.Fprintf(&b, "%s\n", item.Name)
			fmt.Fprintf(&b, "                  %d      %s\n", item.Quantity, yuan(item.SubtotalCent))
		}
	}
	b.WriteString(rule)
	fmt.Fprintf(&b, "合计  %s\n", yuan(r.AmountCent))
	b.WriteString("谢谢惠顾！\n")
	b.WriteString("<CUT>")
	return b.String()
}

// orderTypeLabel maps a business_orders.order_type to its printed heading.
func orderTypeLabel(orderType string) string {
	switch orderType {
	case "food":
		return "餐饮订单"
	case "activity":
		return "活动订单"
	case "recharge":
		return "充值订单"
	case "offline_collection":
		return "门店收款"
	default:
		return "订单"
	}
}

// yuan renders integer cents as a ¥X.XX amount.
func yuan(cent int64) string {
	sign := ""
	if cent < 0 {
		sign, cent = "-", -cent
	}
	return fmt.Sprintf("¥%s%d.%02d", sign, cent/100, cent%100)
}
