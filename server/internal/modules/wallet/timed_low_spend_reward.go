package wallet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const (
	timedLowSpendReason = "low_spend_reward"
	timedLowSpendSource = "low_spend_reward"
)

var chinaStandardTime = time.FixedZone("Asia/Shanghai", 8*60*60)

type timedLowSpendSettings struct {
	Enabled           bool
	ReservationCutoff string
	ConsumptionCutoff string
	MinimumAmountCent int64
	RewardPoints      int64
}

type timedLowSpendWindow struct {
	DayStart          time.Time
	ReservationCutoff time.Time
	ConsumptionCutoff time.Time
}

// GrantTimedLowSpendReward grants the current store's configured points reward
// inside the food-order settlement transaction. Eligibility is scoped to one
// member, store and Beijing calendar day; the ledger idempotency key guarantees
// that concurrent WeChat/coin payments can award at most once that day.
func GrantTimedLowSpendReward(
	ctx context.Context,
	tx *sql.Tx,
	businessOrderID, memberID, storeID int64,
	paidAt time.Time,
) (int64, error) {
	settings, err := loadTimedLowSpendSettings(ctx, tx, storeID)
	if err != nil || !settings.Enabled {
		return 0, err
	}
	window, err := buildTimedLowSpendWindow(paidAt, settings)
	if err != nil {
		return 0, err
	}
	paidAt = paidAt.UTC()
	if paidAt.Before(window.DayStart) || !paidAt.Before(window.ConsumptionCutoff) {
		return 0, nil
	}

	// Serialize all qualifying payments for this member. This makes the
	// cumulative-spend check and once-per-day ledger claim race-safe even when
	// two different food orders settle at the same time.
	var lockedMemberID int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM members WHERE id = ? FOR UPDATE`, memberID,
	).Scan(&lockedMemberID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apperr.NotFound("member not found")
		}
		return 0, apperr.Internal(err)
	}

	qualified, err := hasTimelyReservationOrWaitlist(ctx, tx, memberID, storeID, window)
	if err != nil || !qualified {
		return 0, err
	}
	totalCent, err := cumulativeFoodSpend(ctx, tx, memberID, storeID, window)
	if err != nil {
		return 0, err
	}
	if totalCent < settings.MinimumAmountCent {
		return 0, nil
	}
	if err := markLowSpendMemberArrived(ctx, tx, memberID, storeID, window, paidAt); err != nil {
		return 0, err
	}

	idemKey := fmt.Sprintf("low_spend_reward:%d:%d:%s",
		memberID, storeID, window.DayStart.In(chinaStandardTime).Format("20060102"))
	var alreadyGranted bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM wallet_ledger_entries WHERE idem_key = ?)`, idemKey,
	).Scan(&alreadyGranted); err != nil {
		return 0, apperr.Internal(err)
	}
	if alreadyGranted {
		return 0, nil
	}

	var accountID, available int64
	const accountQuery = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, accountQuery, memberID, AssetPoints).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)
			ON DUPLICATE KEY UPDATE id = id`, memberID, AssetPoints, paidAt, paidAt); err != nil {
			return 0, apperr.Internal(err)
		}
		if err := tx.QueryRowContext(ctx, accountQuery, memberID, AssetPoints).
			Scan(&accountID, &available); err != nil {
			return 0, apperr.Internal(err)
		}
	case err != nil:
		return 0, apperr.Internal(err)
	}

	newBalance := available + settings.RewardPoints
	const insertLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after,
		 reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertLedger, accountID, memberID, AssetPoints,
		settings.RewardPoints, newBalance, timedLowSpendReason, timedLowSpendSource,
		businessOrderID, idemKey, paidAt); err != nil {
		if platdb.IsDuplicate(err) {
			return 0, nil
		}
		return 0, apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		newBalance, paidAt, accountID); err != nil {
		return 0, apperr.Internal(err)
	}
	return settings.RewardPoints, nil
}

func loadTimedLowSpendSettings(ctx context.Context, tx *sql.Tx, storeID int64) (timedLowSpendSettings, error) {
	var settings timedLowSpendSettings
	var raw []byte
	const q = `SELECT enabled, config_json FROM store_rules
		WHERE store_id = ? AND rule_key = 'timed_low_spend_reward'`
	if err := tx.QueryRowContext(ctx, q, storeID).Scan(&settings.Enabled, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return settings, nil
		}
		return timedLowSpendSettings{}, apperr.Internal(err)
	}
	var config struct {
		ReservationCutoff string `json:"reservationCutoff"`
		ConsumptionCutoff string `json:"consumptionCutoff"`
		MinimumAmountCent int64  `json:"minimumAmountCent"`
		RewardPoints      int64  `json:"rewardPoints"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return timedLowSpendSettings{}, apperr.Internal(err)
	}
	settings.ReservationCutoff = config.ReservationCutoff
	settings.ConsumptionCutoff = config.ConsumptionCutoff
	settings.MinimumAmountCent = config.MinimumAmountCent
	settings.RewardPoints = config.RewardPoints
	reservationClock, _ := time.Parse("15:04", settings.ReservationCutoff)
	consumptionClock, _ := time.Parse("15:04", settings.ConsumptionCutoff)
	if !reservationClock.Before(consumptionClock) || settings.MinimumAmountCent <= 0 || settings.RewardPoints <= 0 {
		settings.Enabled = false
	}
	return settings, nil
}

func buildTimedLowSpendWindow(paidAt time.Time, settings timedLowSpendSettings) (timedLowSpendWindow, error) {
	reservationClock, err := time.Parse("15:04", settings.ReservationCutoff)
	if err != nil {
		return timedLowSpendWindow{}, apperr.Invalid("预约截止时间配置不正确")
	}
	consumptionClock, err := time.Parse("15:04", settings.ConsumptionCutoff)
	if err != nil {
		return timedLowSpendWindow{}, apperr.Invalid("消费截止时间配置不正确")
	}
	local := paidAt.In(chinaStandardTime)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, chinaStandardTime)
	reservationCutoff := dayStart.Add(time.Duration(reservationClock.Hour())*time.Hour +
		time.Duration(reservationClock.Minute())*time.Minute)
	consumptionCutoff := dayStart.Add(time.Duration(consumptionClock.Hour())*time.Hour +
		time.Duration(consumptionClock.Minute())*time.Minute)
	if !reservationCutoff.Before(consumptionCutoff) {
		return timedLowSpendWindow{}, apperr.Invalid("消费截止时间必须晚于预约截止时间")
	}
	return timedLowSpendWindow{
		DayStart:          dayStart.UTC(),
		ReservationCutoff: reservationCutoff.UTC(),
		ConsumptionCutoff: consumptionCutoff.UTC(),
	}, nil
}

func hasTimelyReservationOrWaitlist(
	ctx context.Context,
	tx *sql.Tx,
	memberID, storeID int64,
	window timedLowSpendWindow,
) (bool, error) {
	const q = `SELECT EXISTS(
		SELECT 1 FROM reservations
		WHERE member_id = ? AND store_id = ?
		  AND created_at >= ? AND created_at < ?
		  AND status IN ('booked', 'arrived')
		UNION ALL
		SELECT 1 FROM waitlist_entries
		WHERE member_id = ? AND store_id = ?
		  AND created_at >= ? AND created_at < ?
		  AND status IN ('waiting', 'called', 'seated')
	)`
	var exists bool
	if err := tx.QueryRowContext(ctx, q,
		memberID, storeID, window.DayStart, window.ReservationCutoff,
		memberID, storeID, window.DayStart, window.ReservationCutoff,
	).Scan(&exists); err != nil {
		return false, apperr.Internal(err)
	}
	return exists, nil
}

func markLowSpendMemberArrived(
	ctx context.Context,
	tx *sql.Tx,
	memberID, storeID int64,
	window timedLowSpendWindow,
	now time.Time,
) error {
	const recordArrivals = `INSERT INTO arrival_records
		(store_id, member_id, reservation_id, arrived_at, recorded_by_type, recorded_by_id, created_at)
		SELECT r.store_id, r.member_id, r.id, ?, 'system', 0, ?
		FROM reservations r
		WHERE r.member_id = ? AND r.store_id = ?
		  AND r.created_at >= ? AND r.created_at < ?
		  AND r.status = 'booked'
		  AND NOT EXISTS (
			SELECT 1 FROM arrival_records ar WHERE ar.reservation_id = r.id
		  )`
	if _, err := tx.ExecContext(ctx, recordArrivals,
		now, now, memberID, storeID, window.DayStart, window.ReservationCutoff,
	); err != nil {
		return apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE reservations
		SET status = 'arrived', updated_at = ?
		WHERE member_id = ? AND store_id = ?
		  AND created_at >= ? AND created_at < ? AND status = 'booked'`,
		now, memberID, storeID, window.DayStart, window.ReservationCutoff,
	); err != nil {
		return apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE waitlist_entries
		SET status = 'seated', updated_at = ?
		WHERE member_id = ? AND store_id = ?
		  AND created_at >= ? AND created_at < ? AND status IN ('waiting', 'called')`,
		now, memberID, storeID, window.DayStart, window.ReservationCutoff,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func cumulativeFoodSpend(
	ctx context.Context,
	tx *sql.Tx,
	memberID, storeID int64,
	window timedLowSpendWindow,
) (int64, error) {
	const q = `SELECT COALESCE(SUM(bo.total_amount_cent), 0)
		FROM business_orders bo
		WHERE bo.member_id = ? AND bo.store_id = ?
		  AND bo.order_type = 'food' AND bo.payment_status = 'paid'
		  AND EXISTS (
			SELECT 1 FROM payment_orders po
			WHERE po.business_order_id = bo.id
			  AND po.status = 'paid' AND po.pay_method IN ('wechat', 'coin')
			  AND po.paid_at >= ? AND po.paid_at < ?
		  )`
	var total int64
	if err := tx.QueryRowContext(ctx, q,
		memberID, storeID, window.DayStart, window.ConsumptionCutoff,
	).Scan(&total); err != nil {
		return 0, apperr.Internal(err)
	}
	return total, nil
}
