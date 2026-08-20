package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/referral"
	"github.com/inwardclub/server/internal/modules/wallet"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/outbox"
)

const (
	wechatProvider = "wechat"
	paymentPaid    = "paid"
	paymentPending = "pending"
	// paymentExpired is the terminal state for an unpaid payment order closed by
	// a timeout sweep (spec §11). Settlement refuses any non-pending order, so an
	// expired order can never be resurrected to paid by a late/forged notify.
	paymentExpired = "expired"

	// orderTypeRecharge mirrors business_orders.order_type for wallet top-ups.
	orderTypeRecharge = "recharge"
	// orderTypeActivity mirrors business_orders.order_type for ticket purchases.
	orderTypeActivity = "activity"

	// topicPaymentPostProcess is the outbox topic the settlement hands paid,
	// member-bound orders to for asynchronous entitlement evaluation. It must stay
	// in sync with cmd/worker.TaskPaymentPostProcess.
	topicPaymentPostProcess = "payment:post-process"
)

// postProcessPayload is the transactional-outbox hand-off for a settled,
// member-bound order. The rule worker keys its idempotent evaluation on
// (member_id, payment_order_id, rule_version); this payload carries the source
// facts it needs. It never contains a raw phone number.
type postProcessPayload struct {
	Source          string `json:"source"`
	PaymentOrderID  int64  `json:"paymentOrderId"`
	BusinessOrderID int64  `json:"businessOrderId"`
	MemberID        int64  `json:"memberId"`
	StoreID         int64  `json:"storeId"`
	AmountCent      int64  `json:"amountCent"`
	BusinessType    string `json:"businessType,omitempty"`
}

// writePostProcess appends the post-payment processing event inside the
// settlement transaction so it can never fire on a rolled-back settlement. The
// outbox idem_key is the spec's payment:{id}:post-process; the rule worker adds
// its own per-rule idempotency downstream.
func writePostProcess(ctx context.Context, tx *sql.Tx, p postProcessPayload) error {
	idemKey := fmt.Sprintf("payment:%d:post-process", p.PaymentOrderID)
	return outbox.Write(ctx, tx, topicPaymentPostProcess, p, idemKey)
}

type settlementSQLRepository struct{ db *platdb.DB }

// NewSettlementRepository builds the MySQL settlement repository.
func NewSettlementRepository(db *platdb.DB) SettlementRepository {
	return &settlementSQLRepository{db: db}
}

// SettleWeChat locks the payment order by its number (== notify OutTradeNo),
// re-validates under the lock and advances it to paid exactly once.
func (r *settlementSQLRepository) SettleWeChat(ctx context.Context, n WeChatNotification, now time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			paymentID        int64
			businessID       int64
			amount           int64
			status           string
			payMethod        string
			orderType        string
			memberID         sql.NullInt64
			storeID          sql.NullInt64
			businessNo       string
			collectionStatus string
			businessType     string
		)
		const sel = `SELECT po.id, po.business_order_id, po.amount_cent, po.status, po.pay_method,
				bo.order_type, bo.member_id, bo.store_id, bo.business_order_no,
				COALESCE(oco.status, ''), COALESCE(oco.business_type, '')
			FROM payment_orders po
			JOIN business_orders bo ON bo.id = po.business_order_id
			LEFT JOIN offline_collection_orders oco ON oco.payment_order_id = po.id
			WHERE po.payment_order_no = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, sel, n.OutTradeNo).Scan(
			&paymentID, &businessID, &amount, &status, &payMethod, &orderType, &memberID,
			&storeID, &businessNo, &collectionStatus, &businessType,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("payment order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if status == paymentPaid {
			return nil // idempotent: a duplicate notify for an already-settled order
		}
		if status != paymentPending {
			return apperr.Conflict("payment order is not payable")
		}
		if payMethod != wechatProvider {
			return apperr.Invalid("payment order pay method mismatch")
		}
		if orderType == collectionType && collectionStatus != CollectionPending {
			return apperr.Conflict("collection order is not payable")
		}
		if err := insertTxn(ctx, tx, paymentID, wechatProvider, wechatProvider, n.OutTradeNo, n.TransactionID, n.AmountCent, now); err != nil {
			return err
		}
		if err := markOrderPaid(ctx, tx, paymentID, businessID, now); err != nil {
			return err
		}
		if orderType == collectionType {
			const payCollection = `UPDATE offline_collection_orders SET status = ?, updated_at = ?
				WHERE payment_order_id = ? AND status = ?`
			if _, err := tx.ExecContext(
				ctx, payCollection, CollectionPaid, now, paymentID, CollectionPending,
			); err != nil {
				return apperr.Internal(err)
			}
			if memberID.Valid {
				if err := writePostProcess(ctx, tx, postProcessPayload{
					Source:          collectionType,
					PaymentOrderID:  paymentID,
					BusinessOrderID: businessID,
					MemberID:        memberID.Int64,
					StoreID:         storeID.Int64,
					AmountCent:      amount,
					BusinessType:    businessType,
				}); err != nil {
					return err
				}
			}
		}
		if orderType == "food" {
			if _, err := tx.ExecContext(ctx,
				`UPDATE food_orders SET fulfillment_status = 'completed', updated_at = ?
				 WHERE business_order_id = ?`,
				now, businessID,
			); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE business_orders SET order_status = 'completed', updated_at = ? WHERE id = ?`,
				now, businessID,
			); err != nil {
				return apperr.Internal(err)
			}
		}
		if orderType == orderTypeRecharge {
			if err := creditRechargeBenefit(ctx, tx, paymentID, businessID, memberID, amount, now); err != nil {
				return err
			}
		}
		// Activate the order's tickets on payment (pending -> active); mirrors the
		// coin path in order.SettleByCoin. Idempotent via the status guard.
		if orderType == orderTypeActivity {
			const actTickets = `UPDATE tickets t JOIN activity_orders ao ON ao.id = t.activity_order_id
				SET t.status = 'active', t.updated_at = ?
				WHERE ao.business_order_id = ? AND t.status = 'pending'`
			if _, err := tx.ExecContext(ctx, actTickets, now, businessID); err != nil {
				return apperr.Internal(err)
			}
		}
		lowSpendQualified := false
		if orderType == "food" && memberID.Valid {
			if _, err := wallet.GrantFoodOrderPoints(
				ctx, tx, paymentID, businessID, memberID.Int64, now,
			); err != nil {
				return err
			}
			if storeID.Valid {
				if _, err := wallet.GrantTimedLowSpendReward(
					ctx, tx, businessID, memberID.Int64, storeID.Int64, now,
				); err != nil {
					return err
				}
				lowSpendQualified, err = wallet.InvitationLowSpendQualified(
					ctx, tx, memberID.Int64, storeID.Int64, now,
				)
				if err != nil {
					return err
				}
			}
		}
		if memberID.Valid {
			var rewardStoreID int64
			if storeID.Valid {
				rewardStoreID = storeID.Int64
			}
			if err := referral.GrantWechatPayment(ctx, tx, referral.WeChatPayment{
				PaymentOrderID: paymentID, MemberID: memberID.Int64,
				StoreID: rewardStoreID, OrderType: orderType,
				AmountCent: amount, PaidAt: now, LowSpendQualified: lowSpendQualified,
			}); err != nil {
				return err
			}
		}
		// Every successful member-bound WeChat payment earns growth at ¥1 = 1.
		// Use the business payment-order amount rather than the provider callback
		// amount so payment-debug mode never changes member entitlements.
		if memberID.Valid {
			growthBalance, err := creditGrowthValue(
				ctx, tx, paymentID, businessID, memberID.Int64, wechatGrowthAmount(amount), now,
			)
			if err != nil {
				return err
			}
			if err := applyTierUpgrade(ctx, tx, memberID.Int64, growthBalance, now); err != nil {
				return err
			}
		}
		// A store-bound settled order (food / store activity) prints a receipt on
		// the store's printer; a store-less order (recharge) prints nothing. The
		// event rides this settlement transaction, so it can never fire on a rollback.
		if storeID.Valid {
			if err := printer.WriteReceipt(ctx, tx, printer.Receipt{
				StoreID:         storeID.Int64,
				PaymentOrderID:  paymentID,
				BusinessOrderID: businessID,
				BusinessOrderNo: businessNo,
				OrderType:       orderType,
				AmountCent:      amount,
				PaidAt:          now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

const (
	// coinsAsset is the wallet_accounts.asset_type recharge top-ups credit into.
	coinsAsset = "coins"
	// rechargeSourceType tags quick-recharge wallet credits in the ledger.
	rechargeSourceType       = "recharge_order"
	rechargeCoinsIdemPrefix  = "recharge_order"
	rechargePointsIdemPrefix = "recharge_points"

	firstRechargeRewardSettingKey = "first_recharge_double_points_enabled"
	firstRechargeRewardReason     = "first_recharge_reward"
	firstRechargeRewardSource     = "first_recharge_reward"
	firstRechargeRewardIdemPrefix = "first_recharge_reward"

	rechargeRewardThresholdSettingKey = "recharge_double_points_threshold_amount"
	defaultRechargeRewardThreshold    = int64(1000)
	highValueRechargeRewardReason     = "high_value_recharge_reward"
	highValueRechargeRewardSource     = "high_value_recharge_reward"
	highValueRechargeRewardIdem       = "high_value_recharge_reward"
	rechargePointRewardMultiplier     = int64(2)
)

type rechargePointReward struct {
	amount     int64
	reason     string
	sourceType string
	idemPrefix string
}

// creditRechargeBenefit grants the top-up entitlement for a settled recharge
// order inside the settlement transaction. It matches business_orders.total_amount_cent
// against recharge_products.amount_cent to resolve the configured grant, then:
//   - credits coin_amount into the member's coins wallet;
//   - credits points_amount into the member's points wallet;
//
// Each entitlement is anchored on its own ledger idem_key so a duplicate
// settlement never double-credits, and all of it commits atomically with the
// paid order. A custom amount without a configured tier credits ¥1 as 1 coin.
//
// VIP growth is granted once per successful WeChat payment by SettleWeChat,
// outside this recharge-only entitlement helper.
func creditRechargeBenefit(ctx context.Context, tx *sql.Tx, paymentID, businessID int64, memberID sql.NullInt64, amount int64, now time.Time) error {
	if !memberID.Valid {
		// A recharge order without a member cannot be credited; nothing to grant.
		return nil
	}

	// Resolve configured coin/point grants from the matching quick-recharge tier.
	coins, points := customRechargeCoinAmount(amount), int64(0)
	var couponTemplateID sql.NullInt64
	const selProduct = `SELECT coin_amount, points_amount, coupon_template_id FROM recharge_products
		WHERE amount_cent = ? AND status = 'active' ORDER BY id ASC LIMIT 1`
	switch err := tx.QueryRowContext(ctx, selProduct, amount).Scan(&coins, &points, &couponTemplateID); {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return apperr.Internal(err)
	}
	firstRecharge, err := claimFirstSuccessfulRecharge(ctx, tx, memberID.Int64, businessID, now)
	if err != nil {
		return err
	}
	firstRechargeRewardEnabled, rewardThresholdCent, err := rechargeRewardSettings(ctx, tx)
	if err != nil {
		return err
	}

	if err := creditRechargeAsset(ctx, tx, paymentID, businessID, memberID.Int64, coinsAsset, coins, "recharge", rechargeSourceType, rechargeCoinsIdemPrefix, now); err != nil {
		return err
	}
	if err := creditRechargeAsset(ctx, tx, paymentID, businessID, memberID.Int64, pointsAsset, points, "recharge", rechargeSourceType, rechargePointsIdemPrefix, now); err != nil {
		return err
	}
	for _, reward := range rechargePointRewards(
		coins, amount, rewardThresholdCent, firstRecharge && firstRechargeRewardEnabled,
	) {
		if err := creditRechargeAsset(
			ctx, tx, paymentID, businessID, memberID.Int64, pointsAsset,
			reward.amount, reward.reason, reward.sourceType, reward.idemPrefix, now,
		); err != nil {
			return err
		}
	}
	if couponTemplateID.Valid {
		if err := grantRechargeCoupon(ctx, tx, paymentID, memberID.Int64, couponTemplateID.Int64, now); err != nil {
			return err
		}
	}

	return nil
}

func grantRechargeCoupon(ctx context.Context, tx *sql.Tx, paymentID, memberID, templateID int64, now time.Time) error {
	idemKey := fmt.Sprintf("recharge_coupon:%d", paymentID)
	var existingID int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM coupon_entitlements WHERE idem_key = ?`, idemKey,
	).Scan(&existingID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return apperr.Internal(err)
	}

	var storeID sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT store_id FROM coupon_templates WHERE id = ? FOR UPDATE`, templateID,
	).Scan(&storeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.Conflict("充值档位绑定的优惠券不存在")
		}
		return apperr.Internal(err)
	}

	entitlementNo := fmt.Sprintf("RC%d-%d", paymentID, templateID)
	const insertEntitlement = `INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, member_id, store_id, status, granted_reason,
		 granted_by_type, idem_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', '充值赠券', 'recharge', ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertEntitlement, entitlementNo, templateID, memberID, storeID, idemKey, now, now); err != nil {
		if platdb.IsDuplicate(err) {
			return nil
		}
		return apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE coupon_templates SET issued_quantity = issued_quantity + 1, updated_at = ? WHERE id = ?`,
		now, templateID,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func customRechargeCoinAmount(amountCent int64) int64 {
	return amountCent / 100
}

func rechargeRewardSettings(ctx context.Context, tx *sql.Tx) (bool, int64, error) {
	var enabled string
	const selectSetting = `SELECT setting_value FROM system_settings WHERE setting_key = ?`
	switch err := tx.QueryRowContext(ctx, selectSetting, firstRechargeRewardSettingKey).Scan(&enabled); {
	case errors.Is(err, sql.ErrNoRows):
		enabled = "false"
	case err != nil:
		return false, 0, apperr.Internal(err)
	}

	thresholdAmount := defaultRechargeRewardThreshold
	var rawThreshold string
	switch err := tx.QueryRowContext(ctx, selectSetting, rechargeRewardThresholdSettingKey).Scan(&rawThreshold); {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return false, 0, apperr.Internal(err)
	default:
		parsed, err := strconv.ParseInt(rawThreshold, 10, 64)
		if err != nil || parsed <= 0 {
			return false, 0, apperr.Internal(fmt.Errorf("invalid recharge reward threshold %q", rawThreshold))
		}
		thresholdAmount = parsed
	}
	return enabled == "true", thresholdAmount * 100, nil
}

// claimFirstSuccessfulRecharge owns the platform-wide first-recharge decision.
// member_recharge_history has one row per member and no store dimension, so a
// member can only be first across the whole platform. The member-row lock plus
// the primary key also serializes concurrent payment callbacks.
//
// The legacy_user_id lookup protects members who already recharged in v1 even
// when those old orders were not transformed into v2 business_orders.
func claimFirstSuccessfulRecharge(ctx context.Context, tx *sql.Tx, memberID, businessID int64, now time.Time) (bool, error) {
	var legacyUserID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT legacy_user_id FROM members WHERE id = ? FOR UPDATE`, memberID).Scan(&legacyUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, apperr.NotFound("member not found")
		}
		return false, apperr.Internal(err)
	}

	var existingMemberID int64
	err := tx.QueryRowContext(ctx, `SELECT member_id FROM member_recharge_history WHERE member_id = ?`, memberID).Scan(&existingMemberID)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, apperr.Internal(err)
	}

	if legacyUserID.Valid {
		var firstPaidAt time.Time
		err := tx.QueryRowContext(ctx,
			`SELECT first_paid_at FROM legacy_recharge_members WHERE legacy_user_id = ?`,
			legacyUserID.Int64,
		).Scan(&firstPaidAt)
		if err == nil {
			const insertLegacyHistory = `INSERT INTO member_recharge_history
				(member_id, first_business_order_id, first_paid_at, origin, created_at)
				VALUES (?, NULL, ?, 'legacy_v1', ?)`
			if _, err := tx.ExecContext(ctx, insertLegacyHistory, memberID, firstPaidAt, now); err != nil {
				return false, apperr.Internal(err)
			}
			return false, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, apperr.Internal(err)
		}
	}

	const insertCurrentHistory = `INSERT INTO member_recharge_history
		(member_id, first_business_order_id, first_paid_at, origin, created_at)
		VALUES (?, ?, ?, 'v2', ?)`
	if _, err := tx.ExecContext(ctx, insertCurrentHistory, memberID, businessID, now, now); err != nil {
		if platdb.IsDuplicate(err) {
			return false, nil
		}
		return false, apperr.Internal(err)
	}
	return true, nil
}

func rechargePointRewards(coinAmount, amountCent, thresholdCent int64, firstRecharge bool) []rechargePointReward {
	if coinAmount <= 0 {
		return nil
	}
	rewardAmount := coinAmount * rechargePointRewardMultiplier
	if amountCent >= thresholdCent {
		return []rechargePointReward{{
			amount:     rewardAmount,
			reason:     highValueRechargeRewardReason,
			sourceType: highValueRechargeRewardSource,
			idemPrefix: highValueRechargeRewardIdem,
		}}
	}
	if firstRecharge {
		return []rechargePointReward{{
			amount:     rewardAmount,
			reason:     firstRechargeRewardReason,
			sourceType: firstRechargeRewardSource,
			idemPrefix: firstRechargeRewardIdemPrefix,
		}}
	}
	return nil
}

func creditRechargeAsset(ctx context.Context, tx *sql.Tx, paymentID, businessID, memberID int64, asset string, grant int64, reason, sourceType, idemPrefix string, now time.Time) error {
	if grant <= 0 {
		return nil
	}
	var accountID, available int64
	const selAcct = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selAcct, memberID, asset).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		const insAcct = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		res, err := tx.ExecContext(ctx, insAcct, memberID, asset, now, now)
		if err != nil {
			return apperr.Internal(err)
		}
		if accountID, err = res.LastInsertId(); err != nil {
			return apperr.Internal(err)
		}
		available = 0
	case err != nil:
		return apperr.Internal(err)
	}

	newBalance := available + grant
	idemKey := fmt.Sprintf("%s:%d", idemPrefix, paymentID)
	const insLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insLedger, accountID, memberID, asset, grant, newBalance, reason, sourceType, businessID, idemKey, now); err != nil {
		if platdb.IsDuplicate(err) {
			return nil
		}
		return apperr.Internal(err)
	}

	const updateAcct = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ?
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateAcct, newBalance, now, accountID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// creditGrowthValue accrues a WeChat-payment growth grant into the member's
// growth_value wallet inside the settlement transaction and returns the member's
// authoritative growth balance afterwards. A non-positive grant credits nothing
// but still reports the current balance so the tier can be re-resolved against
// it. Idempotency is anchored on the ledger's unique idem_key
// (wechat_payment_growth:<paymentID>), independent of other credits, so a duplicate
// settlement never double-accrues.
func creditGrowthValue(ctx context.Context, tx *sql.Tx, paymentID, businessID, memberID, grant int64, now time.Time) (int64, error) {
	var accountID, available int64
	const selAcct = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selAcct, memberID, growthAsset).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		if grant <= 0 {
			// No account and nothing to credit: the growth balance is zero.
			return 0, nil
		}
		const insAcct = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		res, err := tx.ExecContext(ctx, insAcct, memberID, growthAsset, now, now)
		if err != nil {
			return 0, apperr.Internal(err)
		}
		if accountID, err = res.LastInsertId(); err != nil {
			return 0, apperr.Internal(err)
		}
		available = 0
	case err != nil:
		return 0, apperr.Internal(err)
	}

	if grant <= 0 {
		// Nothing to accrue for this top-up; report the existing balance so the
		// caller can still re-resolve the tier against it.
		return available, nil
	}

	newBalance := available + grant
	idemKey := fmt.Sprintf("%s:%d", wechatGrowthSourceType, paymentID)
	const insLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, 'wechat_payment_growth', ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insLedger, accountID, memberID, growthAsset, grant, newBalance, wechatGrowthSourceType, businessID, idemKey, now); err != nil {
		if platdb.IsDuplicate(err) {
			// A prior settlement already accrued this growth; return the current
			// balance without re-applying it.
			return available, nil
		}
		return 0, apperr.Internal(err)
	}

	const updateAcct = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ?
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateAcct, newBalance, now, accountID); err != nil {
		return 0, apperr.Internal(err)
	}
	return newBalance, nil
}

// applyTierUpgrade advances the member's persisted VIP tier to the highest active
// tier their growth balance now qualifies for. It is upgrade-only: it never lowers
// a member's tier, since downgrade rules are business-pending (spec §13). It runs
// in the settlement transaction and locks the member row so concurrent recharges
// for the same member cannot lose an upgrade.
func applyTierUpgrade(ctx context.Context, tx *sql.Tx, memberID, growthBalance int64, now time.Time) error {
	const selTiers = `SELECT id, level, threshold FROM membership_tiers WHERE status = 'active'`
	rows, err := tx.QueryContext(ctx, selTiers)
	if err != nil {
		return apperr.Internal(err)
	}
	var tiers []tierRow
	for rows.Next() {
		var t tierRow
		if err := rows.Scan(&t.id, &t.level, &t.threshold); err != nil {
			rows.Close()
			return apperr.Internal(err)
		}
		tiers = append(tiers, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return apperr.Internal(err)
	}
	rows.Close()

	target, ok := resolveTier(tiers, growthBalance)
	if !ok {
		return nil // no configured tier qualifies; leave the member unranked
	}

	// Lock the member row and read the tier currently held so the upgrade is a
	// no-op when the member already sits at or above the resolved tier.
	var currentTierID sql.NullInt64
	const selCurrent = `SELECT current_tier_id FROM members WHERE id = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selCurrent, memberID).Scan(&currentTierID); {
	case errors.Is(err, sql.ErrNoRows):
		return apperr.NotFound("member not found")
	case err != nil:
		return apperr.Internal(err)
	}
	if currentTierID.Valid {
		// Resolve the held tier's level (regardless of status) so a disabled tier
		// still blocks a spurious "upgrade" to a same/lower level.
		var currentLevel int
		switch err := tx.QueryRowContext(ctx, `SELECT level FROM membership_tiers WHERE id = ?`, currentTierID.Int64).Scan(&currentLevel); {
		case errors.Is(err, sql.ErrNoRows):
			// Dangling reference: treat as unranked and let the upgrade proceed.
		case err != nil:
			return apperr.Internal(err)
		default:
			if currentLevel >= target.level {
				return nil // already at or above the qualifying tier
			}
		}
	}

	const upd = `UPDATE members SET current_tier_id = ?, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, upd, target.id, now, memberID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// SettleOffline locks the payment order joined to its offline collection order,
// re-validates under the lock and advances both to paid exactly once.
func (r *settlementSQLRepository) SettleOffline(ctx context.Context, n OfflineNotification, now time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		where, arg := "po.payment_order_no = ?", n.OutTradeNo
		if n.OutTradeNo == "" {
			where, arg = "oco.acquirer_order_no = ?", n.AcquirerOrderNo
		}
		var (
			paymentID        int64
			businessID       int64
			amount           int64
			status           string
			collectionID     int64
			collectionStatus string
			collectionStore  int64
			businessType     string
			memberID         sql.NullInt64
			orderType        string
			businessNo       string
		)
		const selTmpl = `SELECT po.id, po.business_order_id, po.amount_cent, po.status,
				oco.id, oco.status, oco.store_id, oco.business_type, oco.member_id,
				bo.order_type, bo.business_order_no
			FROM payment_orders po
			JOIN offline_collection_orders oco ON oco.payment_order_id = po.id
			JOIN business_orders bo ON bo.id = po.business_order_id
			WHERE `
		err := tx.QueryRowContext(ctx, selTmpl+where+" FOR UPDATE", arg).
			Scan(&paymentID, &businessID, &amount, &status, &collectionID, &collectionStatus,
				&collectionStore, &businessType, &memberID, &orderType, &businessNo)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("collection order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if status == paymentPaid && collectionStatus == CollectionPaid {
			return nil // idempotent: a duplicate notify for an already-settled order
		}
		// Both the payment order and its collection order must still be pending
		// before advancing to paid. A cancelled/expired collection must never be
		// resurrected to paid by a late (or forged) offline notify.
		if status != paymentPending {
			return apperr.Conflict("payment order is not payable")
		}
		if collectionStatus != CollectionPending {
			return apperr.Conflict("collection order is not payable")
		}
		channel := n.Channel
		if err := insertTxn(ctx, tx, paymentID, offlineProvider, channel, poNoOrAcquirer(n), n.ExternalTradeNo, amount, now); err != nil {
			return err
		}
		if err := markOrderPaid(ctx, tx, paymentID, businessID, now); err != nil {
			return err
		}
		const payCollection = `UPDATE offline_collection_orders SET status = ?, updated_at = ? WHERE id = ?`
		if _, err := tx.ExecContext(ctx, payCollection, CollectionPaid, now, collectionID); err != nil {
			return apperr.Internal(err)
		}
		// The counter collection always prints a receipt on the store's printer (the
		// collection is store-bound). Rides this settlement transaction.
		if err := printer.WriteReceipt(ctx, tx, printer.Receipt{
			StoreID:         collectionStore,
			PaymentOrderID:  paymentID,
			BusinessOrderNo: businessNo,
			OrderType:       orderType,
			AmountCent:      amount,
			PaidAt:          now,
		}); err != nil {
			return err
		}
		// A member-bound collection hands off to post-payment entitlement
		// processing; a walk-in (no member_id) grants nothing, per spec §9.3.4. The
		// member binding is read from the collection order, never re-derived here.
		if memberID.Valid {
			if err := writePostProcess(ctx, tx, postProcessPayload{
				Source:          collectionType,
				PaymentOrderID:  paymentID,
				BusinessOrderID: businessID,
				MemberID:        memberID.Int64,
				StoreID:         collectionStore,
				AmountCent:      amount,
				BusinessType:    businessType,
			}); err != nil {
				return err
			}
			if n.Channel == wechatProvider {
				if err := referral.GrantWechatPayment(ctx, tx, referral.WeChatPayment{
					PaymentOrderID: paymentID,
					MemberID:       memberID.Int64,
					StoreID:        collectionStore,
					OrderType:      orderType,
					AmountCent:     amount,
					PaidAt:         now,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// poNoOrAcquirer picks the out_trade_no recorded on the transaction row. The
// payment order number is preferred; the acquirer order number is the fallback
// when the notify omitted it.
func poNoOrAcquirer(n OfflineNotification) string {
	if n.OutTradeNo != "" {
		return n.OutTradeNo
	}
	return n.AcquirerOrderNo
}

// insertTxn records the settled payment_transaction. The (provider, out_trade_no)
// unique key is the durable duplicate guard; empty external numbers are stored as
// NULL so the unique external key is not tripped by repeated blanks.
func insertTxn(ctx context.Context, tx *sql.Tx, paymentID int64, provider, channel, outTradeNo, external string, amount int64, now time.Time) error {
	var externalArg any
	if external != "" {
		externalArg = external
	}
	const ins = `INSERT INTO payment_transactions
		(payment_order_id, provider, channel, out_trade_no, external_transaction_no, amount_cent, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'success', ?)`
	if _, err := tx.ExecContext(ctx, ins, paymentID, provider, channel, outTradeNo, externalArg, amount, now); err != nil {
		if platdb.IsDuplicate(err) {
			// A concurrent notify already recorded this transaction; treat as settled.
			return apperr.Conflict("payment already settled")
		}
		return apperr.Internal(err)
	}
	return nil
}

// markOrderPaid advances the payment order and its business order to paid.
func markOrderPaid(ctx context.Context, tx *sql.Tx, paymentID, businessID int64, now time.Time) error {
	const payPO = `UPDATE payment_orders SET status = 'paid', paid_at = ?, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, payPO, now, now, paymentID); err != nil {
		return apperr.Internal(err)
	}
	const payBO = `UPDATE business_orders SET payment_status = 'paid', updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, payBO, now, businessID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
