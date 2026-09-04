package coupon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

type purchasedCouponLine struct {
	lineID     int64
	templateID int64
	quantity   int
	storeID    int64
}

// GrantPurchasedCoupons issues one entitlement per purchased coupon product
// unit using the coupon kind's default validity. The order-line snapshot and
// deterministic idempotency key make retries safe and preserve the entitlement
// selected when the order was made.
func GrantPurchasedCoupons(
	ctx context.Context,
	tx *sql.Tx,
	paymentOrderID, businessOrderID, memberID int64,
	now time.Time,
) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT foi.id, foi.coupon_template_id_snapshot, foi.quantity, fo.store_id
		FROM food_order_items foi
		JOIN food_orders fo ON fo.id = foi.food_order_id
		WHERE fo.business_order_id = ? AND foi.coupon_template_id_snapshot IS NOT NULL
		ORDER BY foi.id`, businessOrderID)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	var lines []purchasedCouponLine
	for rows.Next() {
		var line purchasedCouponLine
		if err := rows.Scan(&line.lineID, &line.templateID, &line.quantity, &line.storeID); err != nil {
			rows.Close()
			return 0, apperr.Internal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Close(); err != nil {
		return 0, apperr.Internal(err)
	}
	if err := rows.Err(); err != nil {
		return 0, apperr.Internal(err)
	}

	granted := 0
	for _, line := range lines {
		var (
			admissionCount int
			validityDays   int
		)
		if err := tx.QueryRowContext(ctx,
			`SELECT t.admission_count, COALESCE(c.default_validity_days, 30)
			 FROM coupon_templates t
			 LEFT JOIN coupon_categories c ON c.id = t.category_id
			 WHERE t.id = ? FOR UPDATE`, line.templateID,
		).Scan(&admissionCount, &validityDays); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, apperr.Conflict("购买的券类型已失效")
			}
			return 0, apperr.Internal(err)
		}
		expiresAt := now.AddDate(0, 0, validityDays)
		for sequence := 1; sequence <= line.quantity; sequence++ {
			idemKey := fmt.Sprintf("food_coupon:%d:%d:%d", paymentOrderID, line.lineID, sequence)
			entitlementNo := fmt.Sprintf("FC%d-%d-%d", paymentOrderID, line.lineID, sequence)
			const insertEntitlement = `INSERT INTO coupon_entitlements
				(entitlement_no, coupon_template_id, admission_count, member_id, store_id, status, granted_reason,
				 granted_by_type, granted_by_id, expires_at, idem_key, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, 'active', '购买券商品', 'purchase', ?, ?, ?, ?, ?)`
			if _, err := tx.ExecContext(ctx, insertEntitlement, entitlementNo, line.templateID,
				admissionCount, memberID, line.storeID, line.lineID, expiresAt, idemKey, now, now); err != nil {
				if platdb.IsDuplicate(err) {
					continue
				}
				return 0, apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE coupon_templates
				SET issued_quantity = issued_quantity + 1, updated_at = ? WHERE id = ?`,
				now, line.templateID); err != nil {
				return 0, apperr.Internal(err)
			}
			granted++
		}
	}
	return granted, nil
}
