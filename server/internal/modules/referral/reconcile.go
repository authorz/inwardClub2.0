package referral

import (
	"context"
	"database/sql"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

type missedRechargePayment struct {
	PaymentOrderID int64
	MemberID       int64
	AmountCent     int64
	PaidAt         time.Time
}

// ReconcileMissedRechargeRewards repairs settled WeChat recharges that were
// skipped while the system-seeded invitation rule was incorrectly dated in the
// future. GrantWechatPayment retains the durable event and wallet idempotency
// guards, so running the migrator repeatedly cannot duplicate a reward.
func ReconcileMissedRechargeRewards(ctx context.Context, database *platdb.DB) error {
	const q = `SELECT po.id, po.member_id, po.amount_cent, po.paid_at
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id AND bo.order_type = 'recharge'
		JOIN members m ON m.id = po.member_id
		LEFT JOIN invitation_reward_events ire
			ON ire.payment_order_id = po.id AND ire.event_type = 'wechat_payment'
		WHERE po.status = 'paid' AND po.pay_method = 'wechat'
		  AND po.paid_at IS NOT NULL
		  AND m.invited_by_member_id IS NOT NULL AND m.invited_at IS NOT NULL
		  AND po.paid_at >= m.invited_at AND ire.id IS NULL
		ORDER BY po.paid_at ASC, po.id ASC`
	rows, err := database.QueryContext(ctx, q)
	if err != nil {
		return apperr.Internal(err)
	}
	var payments []missedRechargePayment
	for rows.Next() {
		var payment missedRechargePayment
		if err := rows.Scan(&payment.PaymentOrderID, &payment.MemberID, &payment.AmountCent, &payment.PaidAt); err != nil {
			rows.Close()
			return apperr.Internal(err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Close(); err != nil {
		return apperr.Internal(err)
	}
	if err := rows.Err(); err != nil {
		return apperr.Internal(err)
	}

	for _, payment := range payments {
		payment := payment
		if err := database.WithinTx(ctx, func(tx *sql.Tx) error {
			return GrantWechatPayment(ctx, tx, WeChatPayment{
				PaymentOrderID: payment.PaymentOrderID,
				MemberID:       payment.MemberID,
				OrderType:      "recharge",
				AmountCent:     payment.AmountCent,
				PaidAt:         payment.PaidAt,
			})
		}); err != nil {
			return err
		}
	}
	return nil
}
