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

var vipUsageLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// ClaimGiftDailyUsage reserves one category-specific gifted-coupon redemption
// slot for the Beijing calendar day. A missing rule or NULL daily limit means
// unrestricted use; entitlements bought as coupon products always bypass the
// global gifted-coupon rules.
//
// Unique numbered slots are the concurrency guard shared by all redemption
// paths. Callers invoke this inside the transaction that consumes the coupon,
// so a failed redemption also rolls back its claimed slot.
func ClaimGiftDailyUsage(
	ctx context.Context,
	tx *sql.Tx,
	memberID, entitlementID int64,
	now time.Time,
) error {
	var grantedByType, categoryName string
	var categoryID int64
	var dailyLimit sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT e.granted_by_type, t.category_id, c.name, r.daily_limit
		FROM coupon_entitlements e
		JOIN coupon_templates t ON t.id = e.coupon_template_id
		JOIN coupon_categories c ON c.id = t.category_id
		LEFT JOIN gift_coupon_usage_rules r ON r.coupon_category_id = t.category_id
		WHERE e.id = ? AND e.member_id = ? FOR UPDATE`,
		entitlementID, memberID,
	).Scan(&grantedByType, &categoryID, &categoryName, &dailyLimit)
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.NotFound("优惠券不存在")
	}
	if err != nil {
		return apperr.Internal(err)
	}
	limit, limited := giftedCouponDailyLimit(grantedByType, dailyLimit)
	if !limited {
		return nil
	}

	usageDate := now.In(vipUsageLocation).Format("2006-01-02")
	for slot := 1; slot <= limit; slot++ {
		_, err = tx.ExecContext(ctx, `INSERT INTO gift_coupon_daily_usages
			(member_id, category_id, usage_date, slot_number, entitlement_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			memberID, categoryID, usageDate, slot, entitlementID, now.UTC(),
		)
		if err == nil {
			return nil
		}
		if !platdb.IsDuplicate(err) {
			return apperr.Internal(err)
		}
	}
	return apperr.Conflict(fmt.Sprintf("%s赠券每天最多使用%d张", categoryName, limit))
}

func giftedCouponDailyLimit(grantedByType string, configuredLimit sql.NullInt64) (int, bool) {
	if grantedByType == "purchase" || !configuredLimit.Valid {
		return 0, false
	}
	return int(configuredLimit.Int64), true
}
