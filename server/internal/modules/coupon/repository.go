package coupon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Repository is the coupon persistence port. Reads are scoped to the owning
// member; redemption writes reserve finite product stock and persist snapshots
// in the same transaction that consumes the entitlement.
type Repository interface {
	ListMemberCoupons(ctx context.Context, memberID int64, status string, limit, offset int) ([]MemberCoupon, int64, error)
	GetEntitlement(ctx context.Context, memberID, entitlementID int64) (MemberCoupon, error)
	// Redeem records a redemption and flips the entitlement to used in one
	// transaction, guarded by the idempotency key and the unique redemption index.
	Redeem(ctx context.Context, in RedeemInput) (MemberCoupon, error)
	// ListRedemptions returns the member's redemption orders, newest first.
	ListRedemptions(ctx context.Context, memberID int64, limit, offset int) ([]RedemptionOrder, int64, error)
	// GetRedemption returns one redemption order owned by the member.
	GetRedemption(ctx context.Context, memberID, id int64) (RedemptionOrder, error)

	// ExpireEntitlements transitions active, dated entitlements whose validity has
	// ended (expires_at < now) to expired — the coupon half of the
	// ticket-coupon:expire sweep (spec §11). Guarded by status='active' so it is
	// idempotent and never downgrades a used/void entitlement; a NULL expires_at
	// never expires. Returns the number expired.
	ExpireEntitlements(ctx context.Context, now time.Time) (int64, error)
}

// RedeemInput is the resolved redemption the repository persists.
type RedeemInput struct {
	MemberID         int64
	EntitlementID    int64
	StoreID          int64
	RedemptionNo     string
	IdemKey          string
	Now              time.Time
	ItemSnapshotJSON []byte
	MatchedRuleJSON  []byte
	Items            []RedemptionItemSnapshot
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL coupon repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

const couponSelect = `SELECT e.id, e.entitlement_no, e.coupon_template_id, t.name,
	COALESCE(t.description,''), t.coupon_type, t.value_cent, e.store_id, e.status, e.expires_at, e.created_at
	FROM coupon_entitlements e
	JOIN coupon_templates t ON t.id = e.coupon_template_id`

func (r *sqlRepository) ListMemberCoupons(ctx context.Context, memberID int64, status string, limit, offset int) ([]MemberCoupon, int64, error) {
	where := `e.member_id = ?`
	args := []any{memberID}
	if status != "" {
		where += ` AND e.status = ?`
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_entitlements e WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := couponSelect + ` WHERE ` + where + ` ORDER BY e.id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []MemberCoupon
	for rows.Next() {
		c, err := scanCoupon(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetEntitlement(ctx context.Context, memberID, entitlementID int64) (MemberCoupon, error) {
	q := couponSelect + ` WHERE e.id = ? AND e.member_id = ?`
	c, err := scanCoupon(r.db.QueryRowContext(ctx, q, entitlementID, memberID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MemberCoupon{}, apperr.NotFound("coupon not found")
		}
		return MemberCoupon{}, apperr.Internal(err)
	}
	return c, nil
}

// Redeem records the redemption and marks the entitlement used atomically. The
// entitlement is locked and re-validated under the lock; the unique index on
// coupon_redemptions.entitlement_id is the final guard against double redemption.
func (r *sqlRepository) Redeem(ctx context.Context, in RedeemInput) (MemberCoupon, error) {
	var out MemberCoupon
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if in.IdemKey != "" {
			if err := idempotency.Claim(ctx, tx, "mini/coupon-redemptions", in.IdemKey, "coupon_entitlement", in.EntitlementID); err != nil {
				return err
			}
		}
		var (
			status     string
			memberID   int64
			templateID int64
			expiresAt  sql.NullTime
		)
		const sel = `SELECT status, member_id, coupon_template_id, expires_at
			FROM coupon_entitlements WHERE id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, sel, in.EntitlementID).Scan(&status, &memberID, &templateID, &expiresAt)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("coupon not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if memberID != in.MemberID {
			// Do not disclose another member's entitlement.
			return apperr.NotFound("coupon not found")
		}
		if status != StatusActive {
			return apperr.Conflict("coupon is not redeemable")
		}
		if expiresAt.Valid && !expiresAt.Time.After(in.Now) {
			return apperr.Conflict("coupon has expired")
		}
		if err := ClaimVIPDailyUsage(ctx, tx, in.MemberID, in.EntitlementID, in.Now); err != nil {
			return err
		}
		for _, item := range in.Items {
			const reserve = `UPDATE catalog_items
				SET stock_quantity = CASE WHEN stock_quantity = 0 THEN 0 ELSE stock_quantity - ? END,
				    updated_at = ?
				WHERE id = ? AND scope_type = 'store' AND store_id = ? AND status = 'published'
				  AND price_cent = ? AND (stock_quantity = 0 OR stock_quantity >= ?)
				  AND JSON_CONTAINS(COALESCE(coupon_template_ids, JSON_ARRAY()), CAST(? AS JSON), '$')`
			res, err := tx.ExecContext(ctx, reserve, item.Quantity, in.Now, item.ItemID, in.StoreID,
				item.UnitPriceCent, item.Quantity, templateID)
			if err != nil {
				return apperr.Internal(err)
			}
			if affected, _ := res.RowsAffected(); affected == 0 {
				return apperr.Conflict("商品库存或兑换设置已变化，请重新选择")
			}
		}
		const ins = `INSERT INTO coupon_redemptions
			(redemption_no, entitlement_id, coupon_template_id, member_id, store_id,
			 matched_rule_json, item_snapshot_json, verified_by_type, idem_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'member', ?, ?)`
		var idem any
		if in.IdemKey != "" {
			idem = in.IdemKey
		}
		insertResult, err := tx.ExecContext(ctx, ins, in.RedemptionNo, in.EntitlementID, templateID,
			in.MemberID, in.StoreID, in.MatchedRuleJSON, in.ItemSnapshotJSON, idem, in.Now)
		if err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("coupon already redeemed")
			}
			return apperr.Internal(err)
		}
		redemptionID, err := insertResult.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		const upd = `UPDATE coupon_entitlements SET status = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		res, err := tx.ExecContext(ctx, upd, StatusUsed, in.Now, in.EntitlementID, StatusActive)
		if err != nil {
			return apperr.Internal(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return apperr.Conflict("coupon is not redeemable")
		}
		out, err = scanCoupon(tx.QueryRowContext(ctx, couponSelect+` WHERE e.id = ? AND e.member_id = ?`, in.EntitlementID, in.MemberID))
		if err != nil {
			return apperr.Internal(err)
		}
		out.RedemptionID = redemptionID
		return nil
	})
	if err != nil {
		return MemberCoupon{}, err
	}
	return out, nil
}

// redemptionSelect joins a redemption with its entitlement (status/expiry),
// template (display name) and store (name). The redemption_no doubles as the
// member-facing 兑换码.
const redemptionSelect = `SELECT r.id, r.redemption_no, e.status, t.name, e.expires_at,
	COALESCE(s.name,''), r.item_snapshot_json, r.created_at
	FROM coupon_redemptions r
	JOIN coupon_entitlements e ON e.id = r.entitlement_id
	JOIN coupon_templates t ON t.id = r.coupon_template_id
	LEFT JOIN stores s ON s.id = r.store_id`

func (r *sqlRepository) ListRedemptions(ctx context.Context, memberID int64, limit, offset int) ([]RedemptionOrder, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_redemptions WHERE member_id = ?`, memberID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := redemptionSelect + ` WHERE r.member_id = ? ORDER BY r.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RedemptionOrder
	for rows.Next() {
		o, err := scanRedemption(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetRedemption(ctx context.Context, memberID, id int64) (RedemptionOrder, error) {
	q := redemptionSelect + ` WHERE r.id = ? AND r.member_id = ?`
	o, err := scanRedemption(r.db.QueryRowContext(ctx, q, id, memberID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RedemptionOrder{}, apperr.NotFound("redemption order not found")
		}
		return RedemptionOrder{}, apperr.Internal(err)
	}
	return o, nil
}

// ExpireEntitlements runs the set-based coupon expiry sweep. status='active' is
// both the idempotency guard (a re-run touches zero rows) and the guard against
// downgrading a used/void entitlement; a NULL expires_at is a never-expiring
// grant and is left untouched.
func (r *sqlRepository) ExpireEntitlements(ctx context.Context, now time.Time) (int64, error) {
	const q = `UPDATE coupon_entitlements SET status = ?, updated_at = ?
		WHERE status = ? AND expires_at IS NOT NULL AND expires_at < ?`
	result, err := r.db.ExecContext(ctx, q, StatusExpired, now, StatusActive, now)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

func scanRedemption(s scanner) (RedemptionOrder, error) {
	var o RedemptionOrder
	var name string
	var snapshotJSON []byte
	if err := s.Scan(&o.ID, &o.RedemptionNo, &o.Status, &name, &o.ValidUntil, &o.StoreName, &snapshotJSON, &o.CreatedAt); err != nil {
		return RedemptionOrder{}, err
	}
	o.Title = name
	o.CouponName = name
	o.Code = o.RedemptionNo
	var items []RedemptionItemSnapshot
	if len(snapshotJSON) > 0 && json.Unmarshal(snapshotJSON, &items) == nil && len(items) > 0 {
		o.Title = items[0].Name
		for _, item := range items {
			o.Qty += item.Quantity
		}
		if len(items) > 1 {
			o.Title += "等商品"
		}
	} else {
		o.Qty = 1
	}
	return o, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCoupon(s scanner) (MemberCoupon, error) {
	var c MemberCoupon
	err := s.Scan(&c.EntitlementID, &c.EntitlementNo, &c.TemplateID, &c.Name, &c.Description,
		&c.CouponType, &c.ValueCent, &c.StoreID, &c.Status, &c.ExpiresAt, &c.CreatedAt)
	return c, err
}
