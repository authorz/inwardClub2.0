package coupon

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const vipBenefitGrantedReason = "VIP等级福利"

var vipUsageLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// ClaimVIPDailyUsage reserves the member's one VIP-benefit coupon redemption
// for the Beijing calendar day. Entitlements obtained from purchases or any
// other source do not claim a slot and remain unrestricted by this rule.
//
// The unique (member_id, usage_date) key is the concurrency guard shared by all
// redemption paths. Callers must invoke this inside the same transaction that
// consumes the entitlement so a failed redemption also rolls back the claim.
func ClaimVIPDailyUsage(
	ctx context.Context,
	tx *sql.Tx,
	memberID, entitlementID int64,
	now time.Time,
) error {
	var grantedReason, grantedByType string
	err := tx.QueryRowContext(ctx, `SELECT granted_reason, granted_by_type
		FROM coupon_entitlements WHERE id = ? AND member_id = ? FOR UPDATE`,
		entitlementID, memberID,
	).Scan(&grantedReason, &grantedByType)
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.NotFound("优惠券不存在")
	}
	if err != nil {
		return apperr.Internal(err)
	}
	if grantedReason != vipBenefitGrantedReason || grantedByType != "system" {
		return nil
	}

	usageDate := now.In(vipUsageLocation).Format("2006-01-02")
	_, err = tx.ExecContext(ctx, `INSERT INTO vip_coupon_daily_usages
		(member_id, usage_date, entitlement_id, created_at) VALUES (?, ?, ?, ?)`,
		memberID, usageDate, entitlementID, now.UTC(),
	)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return apperr.Conflict("VIP权益券每天只能使用一张")
		}
		return apperr.Internal(err)
	}
	return nil
}
