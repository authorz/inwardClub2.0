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
// slot for the Beijing calendar day. A zero category limit means unlimited;
// entitlements bought as coupon products are always unrestricted.
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
	var dailyLimit int
	err := tx.QueryRowContext(ctx, `SELECT e.granted_by_type, t.category_id, c.name, c.gift_daily_usage_limit
		FROM coupon_entitlements e
		JOIN coupon_templates t ON t.id = e.coupon_template_id
		JOIN coupon_categories c ON c.id = t.category_id
		WHERE e.id = ? AND e.member_id = ? FOR UPDATE`,
		entitlementID, memberID,
	).Scan(&grantedByType, &categoryID, &categoryName, &dailyLimit)
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.NotFound("优惠券不存在")
	}
	if err != nil {
		return apperr.Internal(err)
	}
	if grantedByType == "purchase" || dailyLimit == 0 {
		return nil
	}

	usageDate := now.In(vipUsageLocation).Format("2006-01-02")
	for slot := 1; slot <= dailyLimit; slot++ {
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
	return apperr.Conflict(fmt.Sprintf("%s赠券每天最多使用%d张", categoryName, dailyLimit))
}
