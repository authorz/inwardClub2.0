package wallet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// GrantFoodOrderPoints credits the reward snapshotted on food_order_items.
// paymentOrderID anchors the unique ledger key, so WeChat callback retries and
// repeated coin-settlement attempts cannot grant twice.
func GrantFoodOrderPoints(
	ctx context.Context,
	tx *sql.Tx,
	paymentOrderID, businessOrderID, memberID int64,
	now time.Time,
) (int64, error) {
	var (
		foodOrderID int64
		points      int64
	)
	const reward = `SELECT fo.id,
			COALESCE(SUM(foi.points_reward_snapshot * foi.quantity), 0)
		FROM food_orders fo
		LEFT JOIN food_order_items foi ON foi.food_order_id = fo.id
		WHERE fo.business_order_id = ?
		GROUP BY fo.id`
	err := tx.QueryRowContext(ctx, reward, businessOrderID).Scan(&foodOrderID, &points)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, apperr.NotFound("food order not found")
	}
	if err != nil {
		return 0, apperr.Internal(err)
	}
	if points <= 0 {
		return 0, nil
	}

	var accountID, available int64
	const account = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, account, memberID, AssetPoints).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		const createAccount = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		res, err := tx.ExecContext(ctx, createAccount, memberID, AssetPoints, now, now)
		if err != nil {
			return 0, apperr.Internal(err)
		}
		accountID, err = res.LastInsertId()
		if err != nil {
			return 0, apperr.Internal(err)
		}
	case err != nil:
		return 0, apperr.Internal(err)
	}

	newBalance := available + points
	idemKey := fmt.Sprintf("food_order_points:%d", paymentOrderID)
	const ledger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after,
		 reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, 'food_order_reward', 'food_order', ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, ledger, accountID, memberID, AssetPoints,
		points, newBalance, businessOrderID, idemKey, now); err != nil {
		if platdb.IsDuplicate(err) {
			return points, nil
		}
		return 0, apperr.Internal(err)
	}

	const updateAccount = `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ?
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateAccount, newBalance, now, accountID); err != nil {
		return 0, apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE food_orders SET points_earned = ?, updated_at = ? WHERE id = ?`,
		points, now, foodOrderID,
	); err != nil {
		return 0, apperr.Internal(err)
	}
	return points, nil
}
