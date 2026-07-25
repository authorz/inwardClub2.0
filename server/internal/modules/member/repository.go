package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the member persistence port. Profile/phone/invitation reads and
// writes hit the members table directly; the tier/product/ranking reads target
// the membership_tiers, recharge_products and point_savings tables.
type Repository interface {
	// Member self-service (backed by the members table).
	GetMember(ctx context.Context, id int64) (Member, error)
	UpdateProfile(ctx context.Context, id int64, p ProfileUpdate) error
	UpdatePhone(ctx context.Context, id int64, phone string) error

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

const memberColumns = `id, nickname, COALESCE(phone,''), COALESCE(invite_code,''),
	avatar_asset_id, invited_by_member_id, current_tier_id, status, created_at`

func scanMember(row interface{ Scan(...any) error }) (Member, error) {
	var m Member
	err := row.Scan(&m.ID, &m.Nickname, &m.Phone, &m.InviteCode,
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

func (r *sqlRepository) UpdatePhone(ctx context.Context, id int64, phone string) error {
	const q = `UPDATE members SET phone = ?, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, phone, time.Now().UTC(), id); err != nil {
		return apperr.Internal(err)
	}
	return nil
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
	const q = `UPDATE members SET invited_by_member_id = ?, updated_at = ?
		WHERE id = ? AND invited_by_member_id IS NULL`
	res, err := r.db.ExecContext(ctx, q, inviterID, time.Now().UTC(), inviteeID)
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

const membershipTierColumns = `id, name, level, threshold, COALESCE(benefits, ''), icon_asset_id, banner_asset_id, COALESCE(banner_path, ''), status`

func scanMembershipTier(row interface{ Scan(...any) error }) (MembershipTier, error) {
	var t MembershipTier
	err := row.Scan(&t.ID, &t.Name, &t.Level, &t.Threshold, &t.Benefits, &t.IconAssetID, &t.BannerAssetID, &t.BannerPath, &t.Status)
	return t, err
}

func (r *sqlRepository) CreateMembershipTier(ctx context.Context, t MembershipTierCreate) (MembershipTier, error) {
	status := t.Status
	if status == "" {
		status = StatusActive
	}
	const q = `INSERT INTO membership_tiers
		(name, level, threshold, benefits, icon_asset_id, banner_path, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, t.Name, t.Level, t.Threshold, t.Benefits, t.IconAssetID, t.BannerPath, status)
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
	if u.IconAssetID != nil {
		set = append(set, "icon_asset_id = ?")
		args = append(args, *u.IconAssetID)
	}
	if u.BannerPath != nil {
		set = append(set, "banner_path = ?")
		args = append(args, *u.BannerPath)
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
	const q = `SELECT id, amount_cent, coin_amount, points_amount, sort_order, status
		FROM recharge_products WHERE status = ? ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, StatusActive)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RechargeProduct
	for rows.Next() {
		var p RechargeProduct
		if err := rows.Scan(&p.ID, &p.AmountCent, &p.CoinAmount, &p.PointsAmount, &p.SortOrder, &p.Status); err != nil {
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

const rechargeProductColumns = `id, amount_cent, coin_amount, points_amount, sort_order, status`

func scanRechargeProduct(row interface{ Scan(...any) error }) (RechargeProduct, error) {
	var p RechargeProduct
	err := row.Scan(&p.ID, &p.AmountCent, &p.CoinAmount, &p.PointsAmount, &p.SortOrder, &p.Status)
	return p, err
}

func (r *sqlRepository) CreateRechargeProduct(ctx context.Context, p RechargeProductCreate) (RechargeProduct, error) {
	status := p.Status
	if status == "" {
		status = StatusActive
	}
	const q = `INSERT INTO recharge_products
		(amount_cent, coin_amount, points_amount, sort_order, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	res, err := r.db.ExecContext(ctx, q, p.AmountCent, p.CoinAmount, p.PointsAmount, p.SortOrder, status)
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

// ListRankings sums approved point_savings.points per member within the period
// window and returns the top scorers. Per the 2.0 spec the leaderboard is a
// monthly one; week/all are supported as narrower/wider windows over the same
// approved-points measure. Rank is 1-based by descending score.
func (r *sqlRepository) ListRankings(ctx context.Context, period string, limit int) ([]RankingEntry, error) {
	since, hasSince := rankingWindowStart(period, r.clock.Now())

	q := `SELECT m.id, m.nickname, m.avatar_asset_id, COALESCE(SUM(ps.points), 0) AS score
		FROM point_savings ps
		JOIN members m ON m.id = ps.member_id
		WHERE ps.status = ?`
	args := []any{pointSavingApproved}
	if hasSince {
		q += ` AND ps.reviewed_at >= ?`
		args = append(args, since)
	}
	q += ` GROUP BY m.id, m.nickname, m.avatar_asset_id
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
		if err := rows.Scan(&e.MemberID, &e.Nickname, &e.AvatarAssetID, &e.Score); err != nil {
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
// zone's calendar and returned as UTC instants for comparison against the
// stored UTC reviewed_at column.
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
