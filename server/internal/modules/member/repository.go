package member

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the member persistence port. Profile/phone/invitation reads and
// writes hit the members table directly; rankings aggregate settled recharge
// business orders across all stores.
type Repository interface {
	// Member self-service (backed by the members table).
	GetMember(ctx context.Context, id int64) (Member, error)
	UpdateProfile(ctx context.Context, id int64, p ProfileUpdate) error
	UpdatePhone(ctx context.Context, id int64, phone string, intervalDays int) (PhoneChangeResult, error)

	// Invitations (backed by members.invite_code / invited_by_member_id).
	GetByInviteCode(ctx context.Context, code string) (Member, error)
	ListInvitees(ctx context.Context, inviterID int64, limit, offset int) ([]Invitee, int64, error)
	// BindInviter sets the invitee's inviter only if it is still unset; it
	// returns ErrAlreadyInvited when the member already has an inviter.
	BindInviter(ctx context.Context, inviteeID, inviterID int64) error

	// Read-only catalogues (membership_tiers / recharge_products / point_savings).
	ListMembershipTiers(ctx context.Context) ([]MembershipTier, error)
	ListRechargeProducts(ctx context.Context) ([]RechargeProduct, error)
	ListRankings(ctx context.Context, period string, limit int) ([]RankingEntry, error)

	// Admin membership tier reads (all statuses; backed by the membership_tiers table).
	ListAllMembershipTiers(ctx context.Context) ([]MembershipTier, error)
	GetMembershipTier(ctx context.Context, id int64) (MembershipTier, error)

	// Admin membership tier writes (backed by the membership_tiers table).
	CreateMembershipTier(ctx context.Context, t MembershipTierCreate) (MembershipTier, error)
	UpdateMembershipTier(ctx context.Context, id int64, u MembershipTierUpdate) (MembershipTier, error)

	// Admin recharge product reads (all statuses; backed by the recharge_products table).
	ListAllRechargeProducts(ctx context.Context) ([]RechargeProduct, error)
	GetRechargeProduct(ctx context.Context, id int64) (RechargeProduct, error)

	// Admin recharge product writes (backed by the recharge_products table).
	CreateRechargeProduct(ctx context.Context, p RechargeProductCreate) (RechargeProduct, error)
	UpdateRechargeProduct(ctx context.Context, id int64, u RechargeProductUpdate) (RechargeProduct, error)
	ValidateRechargeCouponTemplate(ctx context.Context, id int64) error
}

// Clock is the source of the business "now" used to compute the leaderboard
// windows. It is injected from the configured business clock (see
// platform/config) so the weekly/monthly windows are bucketed in the business
// zone and never drift with the host clock. Now already returns the time in the
// business zone.
type Clock interface{ Now() time.Time }

type sqlRepository struct {
	db    *platdb.DB
	clock Clock
}

// NewRepository builds the MySQL member repository. clock supplies the
// business-day "now" used to compute the leaderboard windows.
func NewRepository(db *platdb.DB, clock Clock) Repository {
	return &sqlRepository{db: db, clock: clock}
}

const memberColumns = `id, nickname, COALESCE(gender,''), COALESCE(phone,''), phone_changed_at, COALESCE(invite_code,''),
	avatar_asset_id, invited_by_member_id, current_tier_id, status, created_at`

func scanMember(row interface{ Scan(...any) error }) (Member, error) {
	var m Member
	err := row.Scan(&m.ID, &m.Nickname, &m.Gender, &m.Phone, &m.PhoneChangedAt, &m.InviteCode,
		&m.AvatarAssetID, &m.InvitedByMemberID, &m.CurrentTierID, &m.Status, &m.CreatedAt)
	return m, err
}

func (r *sqlRepository) GetMember(ctx context.Context, id int64) (Member, error) {
	const q = `SELECT ` + memberColumns + ` FROM members WHERE id = ?`
	m, err := scanMember(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, apperr.New(apperr.CodeMemberNotFound, "member not found")
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

func (r *sqlRepository) UpdateProfile(ctx context.Context, id int64, p ProfileUpdate) error {
	sets := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if p.Nickname != nil {
		sets = append(sets, "nickname = ?")
		args = append(args, *p.Nickname)
	}
	if p.Gender != nil {
		sets = append(sets, "gender = ?")
		args = append(args, *p.Gender)
	}
	if p.AvatarAssetID != nil {
		sets = append(sets, "avatar_asset_id = ?")
		args = append(args, *p.AvatarAssetID)
	}
	if p.AvatarURL != nil {
		sets = append(sets, "avatar_url = ?")
		args = append(args, *p.AvatarURL)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC(), id)
	q := `UPDATE members SET ` + strings.Join(sets, ", ") + ` WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlRepository) UpdatePhone(ctx context.Context, id int64, phone string, intervalDays int) (PhoneChangeResult, error) {
	var result PhoneChangeResult
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		const selectMember = `SELECT COALESCE(phone,''), phone_changed_at FROM members WHERE id = ? FOR UPDATE`
		var currentPhone string
		var changedAt sql.NullTime
		if err := tx.QueryRowContext(ctx, selectMember, id).Scan(&currentPhone, &changedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.New(apperr.CodeMemberNotFound, "会员不存在")
			}
			return apperr.Internal(err)
		}

		now := r.clock.Now()
		if currentPhone == phone {
			if changedAt.Valid {
				result.NextAllowedAt = changedAt.Time.In(now.Location()).AddDate(0, 0, intervalDays)
			}
			return nil
		}
		if nextAllowedAt, blocked := phoneChangeCooldown(currentPhone, phone, changedAt, intervalDays, now); blocked {
			return apperr.Conflict(fmt.Sprintf(
				"手机号每 %d 天只能更换一次，请于 %s 后再试",
				intervalDays,
				nextAllowedAt.Format("2006-01-02 15:04"),
			)).WithDetails(map[string]any{
				"intervalDays":  intervalDays,
				"nextAllowedAt": nextAllowedAt.UTC().Format(time.RFC3339),
			})
		}

		const updatePhone = `UPDATE members SET phone = ?, phone_changed_at = ?, updated_at = ? WHERE id = ?`
		nowUTC := now.UTC()
		if _, err := tx.ExecContext(ctx, updatePhone, phone, nowUTC, nowUTC, id); err != nil {
			if platdb.IsDuplicateKey(err, "uq_members_phone") {
				return apperr.Conflict("该手机号已绑定其他会员账号")
			}
			return apperr.Internal(err)
		}
		result.Changed = true
		result.NextAllowedAt = now.AddDate(0, 0, intervalDays)
		return nil
	})
	if err != nil {
		return PhoneChangeResult{}, err
	}
	return result, nil
}

func phoneChangeCooldown(currentPhone, nextPhone string, changedAt sql.NullTime, intervalDays int, now time.Time) (time.Time, bool) {
	if currentPhone == "" || currentPhone == nextPhone || !changedAt.Valid {
		return time.Time{}, false
	}
	nextAllowedAt := changedAt.Time.In(now.Location()).AddDate(0, 0, intervalDays)
	return nextAllowedAt, now.Before(nextAllowedAt)
}

func (r *sqlRepository) GetByInviteCode(ctx context.Context, code string) (Member, error) {
	const q = `SELECT ` + memberColumns + ` FROM members WHERE invite_code = ?`
	m, err := scanMember(r.db.QueryRowContext(ctx, q, code))
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrInviteCodeNotFound
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

func (r *sqlRepository) ListInvitees(ctx context.Context, inviterID int64, limit, offset int) ([]Invitee, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE invited_by_member_id = ?`, inviterID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT id, nickname, avatar_asset_id, COALESCE(avatar_url,''), created_at
		FROM members WHERE invited_by_member_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, inviterID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []Invitee
	for rows.Next() {
		var iv Invitee
		if err := rows.Scan(&iv.MemberID, &iv.Nickname, &iv.AvatarAssetID, &iv.AvatarURL, &iv.JoinedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, iv)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) BindInviter(ctx context.Context, inviteeID, inviterID int64) error {
	// Only bind when no inviter is set yet; a member may bind exactly once.
	const q = `UPDATE members SET invited_by_member_id = ?, invited_at = ?, updated_at = ?
		WHERE id = ? AND invited_by_member_id IS NULL`
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, q, inviterID, now, now, inviteeID)
	if err != nil {
		return apperr.Internal(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if affected == 0 {
		return ErrAlreadyInvited
	}
	return nil
}

func (r *sqlRepository) ListMembershipTiers(ctx context.Context) ([]MembershipTier, error) {
	const q = `SELECT ` + membershipTierColumns + `
		FROM membership_tiers WHERE status = ? ORDER BY level ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, StatusActive)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []MembershipTier
	for rows.Next() {
		t, err := scanMembershipTier(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// ListAllMembershipTiers returns every membership tier regardless of status,
// for admin console use.
func (r *sqlRepository) ListAllMembershipTiers(ctx context.Context) ([]MembershipTier, error) {
	const q = `SELECT ` + membershipTierColumns + ` FROM membership_tiers ORDER BY level ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []MembershipTier
	for rows.Next() {
		t, err := scanMembershipTier(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// GetMembershipTier fetches a single membership tier regardless of status.
func (r *sqlRepository) GetMembershipTier(ctx context.Context, id int64) (MembershipTier, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+membershipTierColumns+` FROM membership_tiers WHERE id = ?`, id)
	t, err := scanMembershipTier(row)
	if errors.Is(err, sql.ErrNoRows) {
		return MembershipTier{}, ErrMembershipTierNotFound
	}
	if err != nil {
		return MembershipTier{}, apperr.Internal(err)
	}
	return t, nil
}

const membershipTierColumns = `id, name, level, threshold, COALESCE(benefits, ''), COALESCE(benefit_config, JSON_OBJECT()), icon_asset_id, status`

func scanMembershipTier(row interface{ Scan(...any) error }) (MembershipTier, error) {
	var t MembershipTier
	err := row.Scan(&t.ID, &t.Name, &t.Level, &t.Threshold, &t.Benefits, &t.BenefitConfig, &t.IconAssetID, &t.Status)
	return t, err
}

func (r *sqlRepository) CreateMembershipTier(ctx context.Context, t MembershipTierCreate) (MembershipTier, error) {
	status := t.Status
	if status == "" {
		status = StatusActive
	}
	const q = `INSERT INTO membership_tiers
		(name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, t.Name, t.Level, t.Threshold, t.Benefits, t.BenefitConfig, t.IconAssetID, status)
	if err != nil {
		return MembershipTier{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return MembershipTier{}, apperr.Internal(err)
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+membershipTierColumns+` FROM membership_tiers WHERE id = ?`, id)
	return scanMembershipTier(row)
}

func (r *sqlRepository) UpdateMembershipTier(ctx context.Context, id int64, u MembershipTierUpdate) (MembershipTier, error) {
	set := make([]string, 0, 7)
	var args []any
	if u.Name != nil {
		set = append(set, "name = ?")
		args = append(args, *u.Name)
	}
	if u.Level != nil {
		set = append(set, "level = ?")
		args = append(args, *u.Level)
	}
	if u.Threshold != nil {
		set = append(set, "threshold = ?")
		args = append(args, *u.Threshold)
	}
	if u.Benefits != nil {
		set = append(set, "benefits = ?")
		args = append(args, *u.Benefits)
	}
	if u.BenefitConfig != nil {
		set = append(set, "benefit_config = ?")
		args = append(args, *u.BenefitConfig)
	}
	if u.IconAssetID != nil {
		set = append(set, "icon_asset_id = ?")
		args = append(args, *u.IconAssetID)
	}
	if u.Status != nil {
		set = append(set, "status = ?")
		args = append(args, *u.Status)
	}
	set = append(set, "updated_at = NOW()")
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, `UPDATE membership_tiers SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return MembershipTier{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// RowsAffected is 0 when the row is missing or the values are unchanged;
		// disambiguate by checking existence below.
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM membership_tiers WHERE id = ?`, id).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return MembershipTier{}, ErrMembershipTierNotFound
			}
			return MembershipTier{}, apperr.Internal(err)
		}
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+membershipTierColumns+` FROM membership_tiers WHERE id = ?`, id)
	return scanMembershipTier(row)
}

func (r *sqlRepository) ListRechargeProducts(ctx context.Context) ([]RechargeProduct, error) {
	const q = `SELECT id, amount_cent, coin_amount, points_amount, coupon_template_id, sort_order, status
		FROM recharge_products WHERE status = ? ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, StatusActive)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RechargeProduct
	for rows.Next() {
		p, err := scanRechargeProduct(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// ListAllRechargeProducts returns every recharge product regardless of
// status, for admin console use.
func (r *sqlRepository) ListAllRechargeProducts(ctx context.Context) ([]RechargeProduct, error) {
	const q = `SELECT ` + rechargeProductColumns + ` FROM recharge_products ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RechargeProduct
	for rows.Next() {
		p, err := scanRechargeProduct(rows)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// GetRechargeProduct fetches a single recharge product regardless of status.
func (r *sqlRepository) GetRechargeProduct(ctx context.Context, id int64) (RechargeProduct, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+rechargeProductColumns+` FROM recharge_products WHERE id = ?`, id)
	p, err := scanRechargeProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return RechargeProduct{}, ErrRechargeProductNotFound
	}
	if err != nil {
		return RechargeProduct{}, apperr.Internal(err)
	}
	return p, nil
}

const rechargeProductColumns = `id, amount_cent, coin_amount, points_amount, coupon_template_id, sort_order, status`

func scanRechargeProduct(row interface{ Scan(...any) error }) (RechargeProduct, error) {
	var p RechargeProduct
	err := row.Scan(&p.ID, &p.AmountCent, &p.CoinAmount, &p.PointsAmount, &p.CouponTemplateID, &p.SortOrder, &p.Status)
	return p, err
}

func (r *sqlRepository) CreateRechargeProduct(ctx context.Context, p RechargeProductCreate) (RechargeProduct, error) {
	status := p.Status
	if status == "" {
		status = StatusActive
	}
	const q = `INSERT INTO recharge_products
		(amount_cent, coin_amount, points_amount, coupon_template_id, sort_order, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, p.AmountCent, p.CoinAmount, p.PointsAmount, p.CouponTemplateID, p.SortOrder, status)
	if err != nil {
		return RechargeProduct{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return RechargeProduct{}, apperr.Internal(err)
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+rechargeProductColumns+` FROM recharge_products WHERE id = ?`, id)
	return scanRechargeProduct(row)
}

func (r *sqlRepository) UpdateRechargeProduct(ctx context.Context, id int64, u RechargeProductUpdate) (RechargeProduct, error) {
	set := make([]string, 0, 5)
	var args []any
	if u.AmountCent != nil {
		set = append(set, "amount_cent = ?")
		args = append(args, *u.AmountCent)
	}
	if u.CoinAmount != nil {
		set = append(set, "coin_amount = ?")
		args = append(args, *u.CoinAmount)
	}
	if u.PointsAmount != nil {
		set = append(set, "points_amount = ?")
		args = append(args, *u.PointsAmount)
	}
	if u.CouponTemplateID != nil {
		set = append(set, "coupon_template_id = ?")
		args = append(args, *u.CouponTemplateID)
	}
	if u.SortOrder != nil {
		set = append(set, "sort_order = ?")
		args = append(args, *u.SortOrder)
	}
	if u.Status != nil {
		set = append(set, "status = ?")
		args = append(args, *u.Status)
	}
	set = append(set, "updated_at = NOW()")
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, `UPDATE recharge_products SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return RechargeProduct{}, apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// RowsAffected is 0 when the row is missing or the values are unchanged;
		// disambiguate by checking existence below.
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM recharge_products WHERE id = ?`, id).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return RechargeProduct{}, ErrRechargeProductNotFound
			}
			return RechargeProduct{}, apperr.Internal(err)
		}
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+rechargeProductColumns+` FROM recharge_products WHERE id = ?`, id)
	return scanRechargeProduct(row)
}

func (r *sqlRepository) ValidateRechargeCouponTemplate(ctx context.Context, id int64) error {
	var status string
	if err := r.db.QueryRowContext(ctx, `SELECT status FROM coupon_templates WHERE id = ?`, id).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.Invalid("请选择有效的优惠券")
		}
		return apperr.Internal(err)
	}
	if status != "published" {
		return apperr.Invalid("充值奖励只能绑定已发布的优惠券")
	}
	return nil
}

// ListRankings sums settled WeChat business-order amounts per member within the
// requested period. ¥1 of successful WeChat payment equals 1 growth value. The
// business-order amount is authoritative so the payment-debug amount never
// changes member entitlements or ranking growth. Successful refunds reduce the
// counted amount. Rank is 1-based by descending growth value.
func (r *sqlRepository) ListRankings(ctx context.Context, period string, limit int) ([]RankingEntry, error) {
	since, hasSince := rankingWindowStart(period, r.clock.Now())

	sourceQuery := `SELECT ro.member_id,
			FLOOR(GREATEST(ro.total_amount_cent - COALESCE(rf.refunded_cent, 0), 0) / 100) AS score
		FROM business_orders ro
		JOIN (
			SELECT business_order_id, MIN(paid_at) AS paid_at
			FROM payment_orders
			WHERE pay_method = 'wechat' AND paid_at IS NOT NULL
				AND status IN ('paid', 'partially_refunded', 'refunded')
			GROUP BY business_order_id
		) paid ON paid.business_order_id = ro.id
		LEFT JOIN (
			SELECT business_order_id, SUM(amount_cent) AS refunded_cent
			FROM refund_orders
			WHERE status = 'succeeded'
			GROUP BY business_order_id
		) rf ON rf.business_order_id = ro.id
		WHERE ro.payment_status IN ('paid', 'partially_refunded')`
	args := make([]any, 0, 2)
	if hasSince {
		sourceQuery += ` AND paid.paid_at >= ?`
		args = append(args, since)
	}
	if period == RankingAll {
		sourceQuery += ` UNION ALL
			SELECT m.id AS member_id, legacy.growth_value AS score
			FROM legacy_recharge_growth_totals legacy
			JOIN members m ON m.legacy_user_id = legacy.legacy_user_id`
	}

	q := `SELECT m.id, m.nickname, m.avatar_asset_id, COALESCE(m.avatar_url, ''),
			COALESCE(m.gender, ''), COALESCE(SUM(src.score), 0) AS score
		FROM (` + sourceQuery + `) src
		JOIN members m ON m.id = src.member_id
		WHERE m.status = 'active'
		GROUP BY m.id, m.nickname, m.avatar_asset_id, m.avatar_url, m.gender
		HAVING score > 0
		ORDER BY score DESC, m.id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RankingEntry
	rank := 0
	for rows.Next() {
		rank++
		e := RankingEntry{Rank: rank}
		if err := rows.Scan(&e.MemberID, &e.Nickname, &e.AvatarAssetID, &e.AvatarURL, &e.Gender, &e.Score); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// rankingWindowStart returns the inclusive lower bound for the given leaderboard
// period, plus whether a bound applies. RankingAll has no bound. now is the
// business "now" (in the business zone); the window edges are anchored to that
// zone's calendar and returned as UTC instants for comparison against paid_at.
func rankingWindowStart(period string, now time.Time) (time.Time, bool) {
	loc := now.Location()
	switch period {
	case RankingWeek:
		// Start of the current ISO week (Monday 00:00 in the business zone).
		weekday := (int(now.Weekday()) + 6) % 7 // Monday=0 ... Sunday=6
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return day.AddDate(0, 0, -weekday).UTC(), true
	case RankingMonth:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).UTC(), true
	default: // RankingAll
		return time.Time{}, false
	}
}
