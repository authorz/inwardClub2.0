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

// Receipt is the settled-order facts a store receipt is rendered from. It is
// produced at a payment settlement point and carries no member PII — only the
// order identity, its store and the amount paid.
type Receipt struct {
	StoreID         int64
	PaymentOrderID  int64
	BusinessOrderNo string
	// OrderType mirrors business_orders.order_type (food | activity | recharge |
	// offline_collection); it only picks the printed heading.
	OrderType  string
	AmountCent int64
	PaidAt     time.Time
}

// WriteReceipt resolves the store's active printer device and, when one exists,
// appends a print:receipt outbox event carrying a ready-to-print printer.Job in
// the caller's transaction — so the receipt commits atomically with the
// settlement and can never fire on a rolled-back one. A store with no active
// device prints nothing (returns nil), so a settlement never fails for the lack
// of a printer.
func WriteReceipt(ctx context.Context, tx *sql.Tx, r Receipt) error {
	sn, ok, err := activeDeviceSN(ctx, tx, r.StoreID)
	if err != nil || !ok {
		return err
	}
	return outbox.Write(ctx, tx, ReceiptTopic, BuildReceiptJob(sn, r), receiptIdemKey(r.PaymentOrderID))
}

// receiptIdemKey is the outbox idem_key (== asynq task id) for a receipt. Keying
// it on the payment order id makes the print exactly-once per settled order: a
// redelivered dispatch collides on the task id instead of printing twice.
func receiptIdemKey(paymentOrderID int64) string {
	return fmt.Sprintf("payment:%d:print-receipt", paymentOrderID)
}

// BuildReceiptJob renders r onto a printer.Job for device sn. It is pure so the
// receipt body can be unit-tested without a database.
func BuildReceiptJob(sn string, r Receipt) Job {
	return Job{DeviceSN: sn, Template: ReceiptTemplate, Content: renderReceipt(r)}
}

// activeDeviceSN returns the SN of the store's first active printer device. A
// store with no active device yields ok=false — nothing to print.
func activeDeviceSN(ctx context.Context, tx *sql.Tx, storeID int64) (sn string, ok bool, err error) {
	const q = `SELECT device_sn FROM printer_devices
		WHERE store_id = ? AND status = ? ORDER BY id ASC LIMIT 1`
	switch scanErr := tx.QueryRowContext(ctx, q, storeID, StatusActive).Scan(&sn); {
	case errors.Is(scanErr, sql.ErrNoRows):
		return "", false, nil
	case scanErr != nil:
		return "", false, apperr.Internal(scanErr)
	}
	return sn, true, nil
}

// renderReceipt renders the compact receipt body that lands on paper (the Xpyun
// print API takes pre-rendered content). PaidAt is the settlement timestamp as
// stored (UTC), matching payment_orders.paid_at.
func renderReceipt(r Receipt) string {
	const rule = "--------------------------------\n"
	var b strings.Builder
	b.WriteString("InwardClub\n")
	b.WriteString(orderTypeLabel(r.OrderType))
	b.WriteByte('\n')
	b.WriteString(rule)
	fmt.Fprintf(&b, "单号  %s\n", r.BusinessOrderNo)
	fmt.Fprintf(&b, "金额  %s\n", yuan(r.AmountCent))
	fmt.Fprintf(&b, "时间  %s\n", r.PaidAt.Format("2006-01-02 15:04:05"))
	b.WriteString(rule)
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
