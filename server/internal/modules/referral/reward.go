package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/inwardclub/server/internal/modules/wallet"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// A whole coin equals 100人民币分. Multiplying amountCent by basis points
// leaves a 1/10000 percentage denominator, so this is the numerator required
// for one whole coin. Keeping the remainder makes repeated commissions exact.
const numeratorPerCoin = int64(100 * 10_000)

type WeChatPayment struct {
	PaymentOrderID    int64
	MemberID          int64
	StoreID           int64
	OrderType         string
	AmountCent        int64
	PaidAt            time.Time
	LowSpendQualified bool
}

type WeChatRefund struct {
	RefundOrderID  int64
	PaymentOrderID int64
	AmountCent     int64
	RefundedAt     time.Time
}

type rewardAccount struct {
	InviteeMemberID              int64
	InviterMemberID              int64
	CommissionRemainderNumerator int64
	FirstRewardActive            bool
	FirstRewardPaymentOrderID    sql.NullInt64
	FirstRewardRuleVersion       sql.NullInt64
	FirstRewardStoreID           sql.NullInt64
	FirstRewardPaidAt            sql.NullTime
	FirstRewardCoins             int64
	FirstRewardPoints            int64
}

// GrantWechatPayment credits the inviter for one successfully settled WeChat
// payment. Payment settlement and reward ledger writes share the
// caller's transaction, so a reward cannot outlive a rolled-back payment.
func GrantWechatPayment(ctx context.Context, tx *sql.Tx, in WeChatPayment) error {
	if in.PaymentOrderID <= 0 || in.MemberID <= 0 || in.AmountCent <= 0 {
		return nil
	}
	rule, ok, err := activeRuleTx(ctx, tx, in.PaidAt)
	if err != nil || !ok {
		return err
	}

	inviterID, ok, err := inviterForUpdate(ctx, tx, in.MemberID)
	if err != nil || !ok {
		return err
	}
	account, err := lockRewardAccount(ctx, tx, in.MemberID, inviterID, in.PaidAt)
	if err != nil {
		return err
	}
	idemKey := fmt.Sprintf("invite_reward:payment:%d", in.PaymentOrderID)
	exists, err := rewardEventExists(ctx, tx, idemKey)
	if err != nil || exists {
		return err
	}

	commissionNumerator := in.AmountCent * rule.Config.CommissionRateBasisPoints
	commissionCoins, remainder := accrueCommission(
		account.CommissionRemainderNumerator,
		commissionNumerator,
	)
	if commissionCoins > 0 {
		if err := adjustWallet(ctx, tx, inviterID, "coins", commissionCoins,
			"invitation_commission", "payment_order", in.PaymentOrderID,
			fmt.Sprintf("invitation_commission:payment:%d", in.PaymentOrderID), in.PaidAt); err != nil {
			return err
		}
	}

	firstCoins, firstPoints := int64(0), int64(0)
	if in.LowSpendQualified && !account.FirstRewardActive {
		firstCoins = rule.Config.FirstLowSpendRewardCoins
		firstPoints = rule.Config.FirstLowSpendRewardPoints
		if firstCoins > 0 {
			if err := adjustWallet(ctx, tx, inviterID, "coins", firstCoins,
				"invitation_first_low_spend", "payment_order", in.PaymentOrderID,
				fmt.Sprintf("invitation_first_low_spend:payment:%d:coins", in.PaymentOrderID), in.PaidAt); err != nil {
				return err
			}
		}
		if firstPoints > 0 {
			if err := adjustWallet(ctx, tx, inviterID, "points", firstPoints,
				"invitation_first_low_spend", "payment_order", in.PaymentOrderID,
				fmt.Sprintf("invitation_first_low_spend:payment:%d:points", in.PaymentOrderID), in.PaidAt); err != nil {
				return err
			}
		}
		account.FirstRewardActive = true
		account.FirstRewardPaymentOrderID = sql.NullInt64{Int64: in.PaymentOrderID, Valid: true}
		account.FirstRewardRuleVersion = sql.NullInt64{Int64: int64(rule.Version), Valid: true}
		if in.StoreID > 0 {
			account.FirstRewardStoreID = sql.NullInt64{Int64: in.StoreID, Valid: true}
		}
		account.FirstRewardPaidAt = sql.NullTime{Time: in.PaidAt, Valid: true}
		account.FirstRewardCoins = firstCoins
		account.FirstRewardPoints = firstPoints
	}
	account.CommissionRemainderNumerator = remainder
	if err := updateRewardAccount(ctx, tx, account, in.PaidAt); err != nil {
		return err
	}
	return insertRewardEvent(ctx, tx, rewardEvent{
		EventType: "wechat_payment", InviteeMemberID: in.MemberID, InviterMemberID: inviterID,
		PaymentOrderID: in.PaymentOrderID, OrderType: in.OrderType, AmountCent: in.AmountCent,
		RuleVersion: rule.Version, CommissionRateBasisPoints: rule.Config.CommissionRateBasisPoints,
		CommissionNumeratorDelta: commissionNumerator,
		CoinDelta:                commissionCoins + firstCoins, PointsDelta: firstPoints,
		IdemKey: idemKey, CreatedAt: in.PaidAt,
	})
}

// ReverseWechatRefund reverses the exact commission basis used by the original
// payment. If the refunded food spend makes the original first-low-spend award
// invalid, its configured fixed rewards are also clawed back. Wallet balances
// may become negative so a successful external refund never leaves an
// unreconciled invitation liability.
func ReverseWechatRefund(ctx context.Context, tx *sql.Tx, in WeChatRefund) error {
	if in.RefundOrderID <= 0 || in.PaymentOrderID <= 0 || in.AmountCent <= 0 {
		return nil
	}
	original, ok, err := paymentRewardEvent(ctx, tx, in.PaymentOrderID)
	if err != nil || !ok {
		return err
	}
	idemKey := fmt.Sprintf("invite_reward:refund:%d", in.RefundOrderID)
	exists, err := rewardEventExists(ctx, tx, idemKey)
	if err != nil || exists {
		return err
	}
	account, err := lockRewardAccount(ctx, tx, original.InviteeMemberID, original.InviterMemberID, in.RefundedAt)
	if err != nil {
		return err
	}

	reversedNumerator := in.AmountCent * original.CommissionRateBasisPoints
	commissionCoins, remainder := reverseCommission(
		account.CommissionRemainderNumerator,
		reversedNumerator,
	)
	if commissionCoins > 0 {
		if err := adjustWallet(ctx, tx, original.InviterMemberID, "coins", -commissionCoins,
			"invitation_refund_clawback", "refund_order", in.RefundOrderID,
			fmt.Sprintf("invitation_commission:refund:%d", in.RefundOrderID), in.RefundedAt); err != nil {
			return err
		}
	}

	firstCoins, firstPoints := int64(0), int64(0)
	if account.FirstRewardActive && account.FirstRewardStoreID.Valid && account.FirstRewardPaidAt.Valid {
		qualified, err := wallet.InvitationLowSpendQualified(
			ctx, tx, account.InviteeMemberID, account.FirstRewardStoreID.Int64, account.FirstRewardPaidAt.Time,
		)
		if err != nil {
			return err
		}
		if !qualified {
			firstCoins = account.FirstRewardCoins
			firstPoints = account.FirstRewardPoints
			if firstCoins > 0 {
				if err := adjustWallet(ctx, tx, original.InviterMemberID, "coins", -firstCoins,
					"invitation_refund_clawback", "refund_order", in.RefundOrderID,
					fmt.Sprintf("invitation_first_low_spend:refund:%d:coins", in.RefundOrderID), in.RefundedAt); err != nil {
					return err
				}
			}
			if firstPoints > 0 {
				if err := adjustWallet(ctx, tx, original.InviterMemberID, "points", -firstPoints,
					"invitation_refund_clawback", "refund_order", in.RefundOrderID,
					fmt.Sprintf("invitation_first_low_spend:refund:%d:points", in.RefundOrderID), in.RefundedAt); err != nil {
					return err
				}
			}
			account.FirstRewardActive = false
		}
	}
	account.CommissionRemainderNumerator = remainder
	if err := updateRewardAccount(ctx, tx, account, in.RefundedAt); err != nil {
		return err
	}
	return insertRewardEvent(ctx, tx, rewardEvent{
		EventType: "wechat_refund", InviteeMemberID: original.InviteeMemberID,
		InviterMemberID: original.InviterMemberID, PaymentOrderID: in.PaymentOrderID,
		RefundOrderID: in.RefundOrderID, OrderType: original.OrderType, AmountCent: in.AmountCent,
		RuleVersion:               original.RuleVersion,
		CommissionRateBasisPoints: original.CommissionRateBasisPoints,
		CommissionNumeratorDelta:  -reversedNumerator,
		CoinDelta:                 -(commissionCoins + firstCoins), PointsDelta: -firstPoints,
		IdemKey: idemKey, CreatedAt: in.RefundedAt,
	})
}

func activeRuleTx(ctx context.Context, tx *sql.Tx, now time.Time) (Rule, bool, error) {
	var version int
	var raw []byte
	const q = `SELECT version, config_json FROM rule_definitions
		WHERE rule_key = ? AND scope_type = 'global' AND enabled = 1 AND status = 'published'
		  AND (effective_from IS NULL OR effective_from <= ?)
		  AND (effective_to IS NULL OR effective_to > ?)
		ORDER BY version DESC LIMIT 1`
	switch err := tx.QueryRowContext(ctx, q, RuleKey, now, now).Scan(&version, &raw); {
	case errors.Is(err, sql.ErrNoRows):
		return Rule{}, false, nil
	case err != nil:
		return Rule{}, false, apperr.Internal(err)
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		return Rule{}, false, nil
	}
	return Rule{Version: version, Config: cfg}, true, nil
}

func inviterForUpdate(ctx context.Context, tx *sql.Tx, inviteeID int64) (int64, bool, error) {
	var inviter sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT invited_by_member_id FROM members WHERE id = ? FOR UPDATE`, inviteeID).Scan(&inviter); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, apperr.Internal(err)
	}
	return inviter.Int64, inviter.Valid, nil
}

func lockRewardAccount(ctx context.Context, tx *sql.Tx, inviteeID, inviterID int64, now time.Time) (rewardAccount, error) {
	const insert = `INSERT INTO invitation_reward_accounts
		(invitee_member_id, inviter_member_id, commission_remainder_numerator,
		 first_reward_active, first_reward_coins, first_reward_points, created_at, updated_at)
		VALUES (?, ?, 0, 0, 0, 0, ?, ?)
		ON DUPLICATE KEY UPDATE invitee_member_id = invitee_member_id`
	if _, err := tx.ExecContext(ctx, insert, inviteeID, inviterID, now, now); err != nil {
		return rewardAccount{}, apperr.Internal(err)
	}
	const q = `SELECT invitee_member_id, inviter_member_id, commission_remainder_numerator,
		first_reward_active, first_reward_payment_order_id, first_reward_rule_version,
		first_reward_store_id, first_reward_paid_at, first_reward_coins, first_reward_points
		FROM invitation_reward_accounts WHERE invitee_member_id = ? FOR UPDATE`
	var account rewardAccount
	if err := tx.QueryRowContext(ctx, q, inviteeID).Scan(
		&account.InviteeMemberID, &account.InviterMemberID, &account.CommissionRemainderNumerator,
		&account.FirstRewardActive, &account.FirstRewardPaymentOrderID, &account.FirstRewardRuleVersion,
		&account.FirstRewardStoreID, &account.FirstRewardPaidAt,
		&account.FirstRewardCoins, &account.FirstRewardPoints,
	); err != nil {
		return rewardAccount{}, apperr.Internal(err)
	}
	if account.InviterMemberID != inviterID {
		return rewardAccount{}, apperr.Internal(errors.New("invitation reward inviter mismatch"))
	}
	return account, nil
}

func updateRewardAccount(ctx context.Context, tx *sql.Tx, account rewardAccount, now time.Time) error {
	const q = `UPDATE invitation_reward_accounts SET
		commission_remainder_numerator = ?, first_reward_active = ?,
		first_reward_payment_order_id = ?, first_reward_rule_version = ?,
		first_reward_store_id = ?, first_reward_paid_at = ?,
		first_reward_coins = ?, first_reward_points = ?, updated_at = ?
		WHERE invitee_member_id = ?`
	if _, err := tx.ExecContext(ctx, q,
		account.CommissionRemainderNumerator, account.FirstRewardActive,
		nullInt64(account.FirstRewardPaymentOrderID), nullInt64(account.FirstRewardRuleVersion),
		nullInt64(account.FirstRewardStoreID), nullTime(account.FirstRewardPaidAt),
		account.FirstRewardCoins, account.FirstRewardPoints, now, account.InviteeMemberID,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func accrueCommission(remainder, numerator int64) (int64, int64) {
	total := remainder + numerator
	return total / numeratorPerCoin, total % numeratorPerCoin
}

func reverseCommission(remainder, numerator int64) (int64, int64) {
	total := remainder - numerator
	if total >= 0 {
		return 0, total
	}
	coins := (-total + numeratorPerCoin - 1) / numeratorPerCoin
	return coins, total + coins*numeratorPerCoin
}

func adjustWallet(
	ctx context.Context,
	tx *sql.Tx,
	memberID int64,
	asset string,
	delta int64,
	reason, sourceType string,
	sourceID int64,
	idemKey string,
	now time.Time,
) error {
	if delta == 0 {
		return nil
	}
	const insertAccount = `INSERT INTO wallet_accounts
		(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
		VALUES (?, ?, 0, 0, 0, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`
	if _, err := tx.ExecContext(ctx, insertAccount, memberID, asset, now, now); err != nil {
		return apperr.Internal(err)
	}
	var accountID, available int64
	if err := tx.QueryRowContext(ctx, `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`, memberID, asset).Scan(&accountID, &available); err != nil {
		return apperr.Internal(err)
	}
	newBalance := available + delta
	direction := "credit"
	amount := delta
	if delta < 0 {
		direction = "debit"
		amount = -delta
	}
	const insertLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after,
		 reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertLedger, accountID, memberID, asset, direction,
		amount, newBalance, reason, sourceType, sourceID, idemKey, now); err != nil {
		return apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		newBalance, now, accountID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

type rewardEvent struct {
	EventType                 string
	InviteeMemberID           int64
	InviterMemberID           int64
	PaymentOrderID            int64
	RefundOrderID             int64
	OrderType                 string
	AmountCent                int64
	RuleVersion               int
	CommissionRateBasisPoints int64
	CommissionNumeratorDelta  int64
	CoinDelta                 int64
	PointsDelta               int64
	IdemKey                   string
	CreatedAt                 time.Time
}

func insertRewardEvent(ctx context.Context, tx *sql.Tx, event rewardEvent) error {
	const q = `INSERT INTO invitation_reward_events
		(event_type, invitee_member_id, inviter_member_id, payment_order_id, refund_order_id,
		 order_type, amount_cent, rule_version, commission_rate_basis_points,
		 commission_numerator_delta, coin_delta, points_delta, idem_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, q, event.EventType, event.InviteeMemberID,
		event.InviterMemberID, nullablePositive(event.PaymentOrderID), nullablePositive(event.RefundOrderID),
		event.OrderType, event.AmountCent, event.RuleVersion, event.CommissionRateBasisPoints,
		event.CommissionNumeratorDelta, event.CoinDelta, event.PointsDelta, event.IdemKey, event.CreatedAt,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func rewardEventExists(ctx context.Context, tx *sql.Tx, idemKey string) (bool, error) {
	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM invitation_reward_events WHERE idem_key = ?)`, idemKey,
	).Scan(&exists); err != nil {
		return false, apperr.Internal(err)
	}
	return exists, nil
}

func paymentRewardEvent(ctx context.Context, tx *sql.Tx, paymentOrderID int64) (rewardEvent, bool, error) {
	const q = `SELECT event_type, invitee_member_id, inviter_member_id, payment_order_id,
		order_type, amount_cent, rule_version, commission_rate_basis_points,
		commission_numerator_delta, coin_delta, points_delta, idem_key, created_at
		FROM invitation_reward_events
		WHERE event_type = 'wechat_payment' AND payment_order_id = ? LIMIT 1`
	var event rewardEvent
	if err := tx.QueryRowContext(ctx, q, paymentOrderID).Scan(
		&event.EventType, &event.InviteeMemberID, &event.InviterMemberID, &event.PaymentOrderID,
		&event.OrderType, &event.AmountCent, &event.RuleVersion,
		&event.CommissionRateBasisPoints, &event.CommissionNumeratorDelta,
		&event.CoinDelta, &event.PointsDelta, &event.IdemKey, &event.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rewardEvent{}, false, nil
		}
		return rewardEvent{}, false, apperr.Internal(err)
	}
	return event, true, nil
}

func nullablePositive(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullInt64(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func nullTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}
