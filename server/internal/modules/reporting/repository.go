package reporting

import (
	"context"
	"database/sql"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the analytics read persistence port.
type Repository interface {
	Overview(ctx context.Context, f OverviewFilter) (Overview, error)
	Revenue(ctx context.Context, f ReportFilter) ([]RevenueRow, int64, error)
	CatalogItems(ctx context.Context, f ReportFilter) ([]CatalogItemStat, int64, error)
	Activities(ctx context.Context, f ReportFilter) ([]ActivityStat, int64, error)
	Coupons(ctx context.Context, f ReportFilter) ([]CouponStat, int64, error)
	Records(ctx context.Context, f ReportFilter) ([]RecordRow, int64, error)
	Members(ctx context.Context, f ReportFilter) ([]MemberStat, int64, error)
	Reservations(ctx context.Context, f ReportFilter) ([]ReservationStat, int64, error)
	Stores(ctx context.Context, f ReportFilter) ([]StoreStat, int64, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL analytics repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

// NewRollupRepository builds the MySQL reporting_daily write repository. It
// shares the read repository's connection; the read and write ports are split so
// the report:rollup worker depends only on the write surface.
func NewRollupRepository(db *platdb.DB) RollupRepository { return &sqlRepository{db: db} }

// scopeDate builds the shared "AND …" fragment applied by every report: an
// optional store scope on storeCol and an optional [From, To] window on dateCol.
// The column names are code-controlled (never request input) so they are safe to
// interpolate; the scope/date values are always bound as parameters. An empty
// column name skips that dimension. The window is inclusive on both edges.
func scopeDate(f ReportFilter, storeCol, dateCol string) (string, []any) {
	clause := ""
	var args []any
	if f.StoreID != nil && storeCol != "" {
		clause += " AND " + storeCol + " = ?"
		args = append(args, *f.StoreID)
	}
	if f.From != nil && dateCol != "" {
		clause += " AND " + dateCol + " >= ?"
		args = append(args, *f.From)
	}
	if f.To != nil && dateCol != "" {
		clause += " AND " + dateCol + " <= ?"
		args = append(args, *f.To)
	}
	return clause, args
}

// Overview computes the dashboard headline counters. A nil StoreID aggregates
// across every store; a set StoreID pins store-owned counters to that store.
// Sales count only paid orders. For a selected store, member counters represent
// members with paid orders there and members whose first paid store order is today.
func (r *sqlRepository) Overview(ctx context.Context, f OverviewFilter) (Overview, error) {
	const dashboardDays = 7
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Now().In(shanghai)
	todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghai)
	todayStart := todayLocal.UTC()
	tomorrowStart := todayLocal.AddDate(0, 0, 1).UTC()
	trendStart := todayLocal.AddDate(0, 0, -(dashboardDays - 1)).UTC()

	orderScope := ""
	var orderScopeArgs []any
	if f.StoreID != nil {
		orderScope = " AND bo.store_id = ?"
		orderScopeArgs = []any{*f.StoreID}
	}
	var o Overview

	if f.StoreID != nil {
		o.StoreCount = 1
	} else if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stores`).Scan(&o.StoreCount); err != nil {
		return Overview{}, apperr.Internal(err)
	}

	if f.StoreID != nil {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*),
				COALESCE(SUM(CASE WHEN first_order_at >= ? AND first_order_at < ? THEN 1 ELSE 0 END), 0)
			FROM (
				SELECT member_id, MIN(created_at) AS first_order_at
				FROM business_orders
				WHERE store_id = ? AND payment_status = 'paid' AND member_id IS NOT NULL
				GROUP BY member_id
			) store_members`,
			todayStart, tomorrowStart, *f.StoreID,
		).Scan(&o.MemberCount, &o.TodayNewMemberCount); err != nil {
			return Overview{}, apperr.Internal(err)
		}
	} else {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*),
				COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0)
			FROM members`,
			todayStart, tomorrowStart,
		).Scan(&o.MemberCount, &o.TodayNewMemberCount); err != nil {
			return Overview{}, apperr.Internal(err)
		}
	}

	paymentArgs := append([]any{todayStart, tomorrowStart}, orderScopeArgs...)
	const paymentTotals = `SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN po.paid_at >= bounds.today_start AND po.paid_at < bounds.tomorrow_start THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(po.amount_cent), 0),
			COALESCE(SUM(CASE WHEN bo.order_type = 'offline_collection' THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND po.paid_at >= bounds.today_start AND po.paid_at < bounds.tomorrow_start THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'recharge' THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'food' THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'activity' THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'recharge' AND po.paid_at >= bounds.today_start AND po.paid_at < bounds.tomorrow_start THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'food' AND po.paid_at >= bounds.today_start AND po.paid_at < bounds.tomorrow_start THEN po.amount_cent ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' AND bo.order_type = 'activity' AND po.paid_at >= bounds.today_start AND po.paid_at < bounds.tomorrow_start THEN po.amount_cent ELSE 0 END), 0)
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		CROSS JOIN (SELECT ? AS today_start, ? AS tomorrow_start) bounds
		WHERE po.status = 'paid'`
	if err := r.db.QueryRowContext(ctx, paymentTotals+orderScope, paymentArgs...).Scan(
		&o.OrderCount,
		&o.TodayOrderCount,
		&o.GrossSalesCent,
		&o.OfflineCollectionRevenueCent,
		&o.WechatRevenue.Total,
		&o.WechatRevenue.Today,
		&o.WechatRevenue.Recharge,
		&o.WechatRevenue.Food,
		&o.WechatRevenue.Activity,
		&o.WechatRevenue.TodayRecharge,
		&o.WechatRevenue.TodayFood,
		&o.WechatRevenue.TodayActivity,
	); err != nil {
		return Overview{}, apperr.Internal(err)
	}
	o.TodayGrossSalesCent = o.WechatRevenue.Today
	o.ActivityRevenueCent = o.WechatRevenue.Activity
	o.TodayActivityRevenueCent = o.WechatRevenue.TodayActivity

	coinArgs := append([]any{todayStart, tomorrowStart}, orderScopeArgs...)
	const coinTotals = `SELECT
			COALESCE(SUM(w.amount), 0),
			COALESCE(SUM(CASE WHEN w.created_at >= bounds.today_start AND w.created_at < bounds.tomorrow_start THEN w.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bo.order_type = 'food' THEN w.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bo.order_type = 'activity' THEN w.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bo.order_type = 'food' AND w.created_at >= bounds.today_start AND w.created_at < bounds.tomorrow_start THEN w.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bo.order_type = 'activity' AND w.created_at >= bounds.today_start AND w.created_at < bounds.tomorrow_start THEN w.amount ELSE 0 END), 0)
		FROM wallet_ledger_entries w
		JOIN payment_orders po ON po.id = w.source_id
		JOIN business_orders bo ON bo.id = po.business_order_id
		CROSS JOIN (SELECT ? AS today_start, ? AS tomorrow_start) bounds
		WHERE w.asset_type = 'coins' AND w.direction = 'debit'
			AND w.reason = 'order_payment' AND w.source_type = 'payment_order'`
	if err := r.db.QueryRowContext(ctx, coinTotals+orderScope, coinArgs...).Scan(
		&o.CoinConsumption.Total,
		&o.CoinConsumption.Today,
		&o.CoinConsumption.Food,
		&o.CoinConsumption.Activity,
		&o.CoinConsumption.TodayFood,
		&o.CoinConsumption.TodayActivity,
	); err != nil {
		return Overview{}, apperr.Internal(err)
	}

	trendIndex := make(map[string]int, dashboardDays)
	for day := 0; day < dashboardDays; day++ {
		date := todayLocal.AddDate(0, 0, day-(dashboardDays-1))
		key := date.Format("2006-01-02")
		trendIndex[key] = len(o.Trend)
		o.Trend = append(o.Trend, OverviewTrendPoint{Date: date})
	}

	trendPaymentArgs := append([]any{trendStart, tomorrowStart}, orderScopeArgs...)
	const paymentTrend = `SELECT
			DATE(CONVERT_TZ(po.paid_at, '+00:00', '+08:00')) AS report_date,
			COALESCE(SUM(CASE WHEN po.pay_method = 'wechat' THEN po.amount_cent ELSE 0 END), 0),
			COUNT(*)
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		WHERE po.status = 'paid' AND po.paid_at >= ? AND po.paid_at < ?`
	rows, err := r.db.QueryContext(ctx, paymentTrend+orderScope+` GROUP BY report_date ORDER BY report_date`, trendPaymentArgs...)
	if err != nil {
		return Overview{}, apperr.Internal(err)
	}
	for rows.Next() {
		var date time.Time
		var revenue, orders int64
		if err := rows.Scan(&date, &revenue, &orders); err != nil {
			rows.Close()
			return Overview{}, apperr.Internal(err)
		}
		if index, ok := trendIndex[date.Format("2006-01-02")]; ok {
			o.Trend[index].WechatRevenueCent = revenue
			o.Trend[index].OrderCount = orders
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Overview{}, apperr.Internal(err)
	}
	rows.Close()

	couponScope := ""
	var couponScopeArgs []any
	if f.StoreID != nil {
		couponScope = " AND store_id = ?"
		couponScopeArgs = []any{*f.StoreID}
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_entitlements WHERE 1 = 1`+couponScope, couponScopeArgs...).Scan(&o.CouponsIssued); err != nil {
		return Overview{}, apperr.Internal(err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_redemptions WHERE 1 = 1`+couponScope, couponScopeArgs...).Scan(&o.CouponsRedeemed); err != nil {
		return Overview{}, apperr.Internal(err)
	}
	return o, nil
}

// Revenue serves the per-day paid order count and gross directly from successful
// payment orders. Using paid_at keeps cross-day payments on their actual receipt
// date and keeps today's report current without waiting for the daily rollup.
// A set StoreID is always sourced from the store token scope at the handler.
func (r *sqlRepository) Revenue(ctx context.Context, f ReportFilter) ([]RevenueRow, int64, error) {
	sd, args := scopeDate(f, "bo.store_id", "po.paid_at")
	const reportDate = "DATE(CONVERT_TZ(po.paid_at, '+00:00', '+08:00'))"
	base := ` FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		WHERE po.status = 'paid'` + sd
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT `+reportDate+`)`+base, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ` + reportDate + `, COUNT(*), COALESCE(SUM(po.amount_cent), 0)` + base +
		` GROUP BY ` + reportDate + ` ORDER BY ` + reportDate + ` DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]RevenueRow, 0)
	for rows.Next() {
		var row RevenueRow
		if err := rows.Scan(&row.Date, &row.OrderCount, &row.GrossCent); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// CatalogItems rolls up food order lines into one row per item: quantity sold and
// gross. Best-selling by gross first.
func (r *sqlRepository) CatalogItems(ctx context.Context, f ReportFilter) ([]CatalogItemStat, int64, error) {
	sd, args := scopeDate(f, "fo.store_id", "fo.created_at")
	base := ` FROM food_order_items foi JOIN food_orders fo ON fo.id = foi.food_order_id WHERE 1 = 1` + sd
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT foi.item_id)`+base, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT foi.item_id, MAX(foi.name_snapshot), COALESCE(SUM(foi.quantity), 0), COALESCE(SUM(foi.subtotal_cent), 0)` +
		base + ` GROUP BY foi.item_id ORDER BY SUM(foi.subtotal_cent) DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]CatalogItemStat, 0)
	for rows.Next() {
		var s CatalogItemStat
		if err := rows.Scan(&s.ItemID, &s.ItemName, &s.SoldQty, &s.GrossCent); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Activities rolls up activity orders into one row per activity: the order count
// and total tickets. Busiest by order count first.
func (r *sqlRepository) Activities(ctx context.Context, f ReportFilter) ([]ActivityStat, int64, error) {
	sd, args := scopeDate(f, "ao.store_id", "ao.created_at")
	base := ` FROM activity_orders ao JOIN activities a ON a.id = ao.activity_id WHERE 1 = 1` + sd
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT ao.activity_id)`+base, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ao.activity_id, MAX(a.title), COUNT(*), COALESCE(SUM(ao.ticket_count), 0)` +
		base + ` GROUP BY ao.activity_id ORDER BY COUNT(*) DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]ActivityStat, 0)
	for rows.Next() {
		var s ActivityStat
		if err := rows.Scan(&s.ActivityID, &s.ActivityName, &s.OrderCount, &s.TicketCount); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Coupons rolls up one row per template: entitlements issued and redemptions,
// each counted within the scope/window. Templates with the most issued first.
func (r *sqlRepository) Coupons(ctx context.Context, f ReportFilter) ([]CouponStat, int64, error) {
	// The scope/window applies to the entitlement and redemption rows; the
	// template list is scoped by its own store_id so a store console only sees
	// its own templates.
	tplScope, tplArgs := scopeDate(f, "ct.store_id", "")
	entSD, entArgs := scopeDate(f, "e.store_id", "e.created_at")
	redSD, redArgs := scopeDate(f, "cr.store_id", "cr.created_at")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_templates ct WHERE 1 = 1`+tplScope, tplArgs...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT ct.id, ct.name,
			(SELECT COUNT(*) FROM coupon_entitlements e WHERE e.coupon_template_id = ct.id` + entSD + `),
			(SELECT COUNT(*) FROM coupon_redemptions cr WHERE cr.coupon_template_id = ct.id` + redSD + `)
		FROM coupon_templates ct WHERE 1 = 1` + tplScope + ` ORDER BY ct.id DESC LIMIT ? OFFSET ?`
	// Argument order matches the SQL text: entitlement subquery, redemption
	// subquery, then the outer template scope and page bounds.
	args := append(append(append([]any{}, entArgs...), redArgs...), tplArgs...)
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]CouponStat, 0)
	for rows.Next() {
		var s CouponStat
		if err := rows.Scan(&s.TemplateID, &s.Name, &s.Issued, &s.Redeemed); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Records lists verification events (ticket/coupon check-ins) as record lines.
// Newest first; Kind is the verified target type.
func (r *sqlRepository) Records(ctx context.Context, f ReportFilter) ([]RecordRow, int64, error) {
	sd, args := scopeDate(f, "store_id", "created_at")
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM verifications WHERE 1 = 1`+sd, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, target_type, created_at FROM verifications WHERE 1 = 1` + sd + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]RecordRow, 0)
	for rows.Next() {
		var row RecordRow
		if err := rows.Scan(&row.ID, &row.Kind, &row.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// Members rolls up one row per member: points balance and order count. Under a
// store scope only members who ordered at that store are listed, and the order
// count is likewise scoped/windowed.
func (r *sqlRepository) Members(ctx context.Context, f ReportFilter) ([]MemberStat, int64, error) {
	orderSD, orderArgs := scopeDate(f, "bo.store_id", "bo.created_at")

	where := "1 = 1"
	var whereArgs []any
	if f.StoreID != nil {
		where += " AND EXISTS (SELECT 1 FROM business_orders bo WHERE bo.member_id = m.id AND bo.store_id = ?)"
		whereArgs = append(whereArgs, *f.StoreID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members m WHERE `+where, whereArgs...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT m.id,
			COALESCE((SELECT wa.available_amount FROM wallet_accounts wa WHERE wa.member_id = m.id AND wa.asset_type = 'points'), 0),
			(SELECT COUNT(*) FROM business_orders bo WHERE bo.member_id = m.id` + orderSD + `)
		FROM members m WHERE ` + where + ` ORDER BY m.id DESC LIMIT ? OFFSET ?`
	// Argument order matches the SQL text: order-count subquery, then the outer
	// EXISTS scope and page bounds.
	args := append(append([]any{}, orderArgs...), whereArgs...)
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]MemberStat, 0)
	for rows.Next() {
		var s MemberStat
		if err := rows.Scan(&s.MemberID, &s.PointsBalance, &s.OrderCount); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Reservations serves the per-day booking rollup from the reporting_daily
// pre-aggregate written by the report:rollup worker. Newest day first; days with
// no rollup row do not appear.
func (r *sqlRepository) Reservations(ctx context.Context, f ReportFilter) ([]ReservationStat, int64, error) {
	sd, scopeArgs := scopeDate(f, "store_id", "report_date")
	args := append([]any{MetricReservations}, scopeArgs...)
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT report_date) FROM reporting_daily WHERE metric = ?`+sd, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT report_date, COALESCE(SUM(quantity), 0) FROM reporting_daily WHERE metric = ?` + sd +
		` GROUP BY report_date ORDER BY report_date DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]ReservationStat, 0)
	for rows.Next() {
		var s ReservationStat
		if err := rows.Scan(&s.Date, &s.Count); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Stores rolls up one row per store with sales, customer, reservation and coupon
// activity. Stores with no activity in the window still appear. Newest store first.
func (r *sqlRepository) Stores(ctx context.Context, f ReportFilter) ([]StoreStat, int64, error) {
	storeScope, storeArgs := scopeDate(f, "s.id", "")
	orderDate, orderArgs := scopeDate(f, "", "bo.created_at")
	reservationDate, reservationArgs := scopeDate(f, "", "r.created_at")
	redemptionDate, redemptionArgs := scopeDate(f, "", "cr.created_at")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stores s WHERE 1 = 1`+storeScope, storeArgs...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT s.id, s.name,
			COALESCE(os.order_count, 0), COALESCE(os.paid_order_count, 0),
			COALESCE(os.gross_cent, 0), COALESCE(os.food_order_count, 0),
			COALESCE(os.food_gross_cent, 0), COALESCE(os.activity_order_count, 0),
			COALESCE(os.activity_gross_cent, 0), COALESCE(os.unique_member_count, 0),
			COALESCE(rs.reservation_count, 0), COALESCE(cs.redemption_count, 0)
		FROM stores s
		LEFT JOIN (
			SELECT bo.store_id,
				COUNT(*) AS order_count,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' THEN 1 ELSE 0 END), 0) AS paid_order_count,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' THEN bo.total_amount_cent ELSE 0 END), 0) AS gross_cent,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' AND bo.order_type = 'food' THEN 1 ELSE 0 END), 0) AS food_order_count,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' AND bo.order_type = 'food' THEN bo.total_amount_cent ELSE 0 END), 0) AS food_gross_cent,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' AND bo.order_type = 'activity' THEN 1 ELSE 0 END), 0) AS activity_order_count,
				COALESCE(SUM(CASE WHEN bo.payment_status = 'paid' AND bo.order_type = 'activity' THEN bo.total_amount_cent ELSE 0 END), 0) AS activity_gross_cent,
				COUNT(DISTINCT CASE WHEN bo.payment_status = 'paid' THEN bo.member_id END) AS unique_member_count
			FROM business_orders bo WHERE bo.store_id IS NOT NULL` + orderDate + ` GROUP BY bo.store_id
		) os ON os.store_id = s.id
		LEFT JOIN (
			SELECT r.store_id, COUNT(*) AS reservation_count
			FROM reservations r WHERE 1 = 1` + reservationDate + ` GROUP BY r.store_id
		) rs ON rs.store_id = s.id
		LEFT JOIN (
			SELECT cr.store_id, COUNT(*) AS redemption_count
			FROM coupon_redemptions cr WHERE 1 = 1` + redemptionDate + ` GROUP BY cr.store_id
		) cs ON cs.store_id = s.id
		WHERE 1 = 1` + storeScope + ` ORDER BY s.id DESC LIMIT ? OFFSET ?`
	args := append(append(append(append([]any{}, orderArgs...), reservationArgs...), redemptionArgs...), storeArgs...)
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]StoreStat, 0)
	for rows.Next() {
		var s StoreStat
		if err := rows.Scan(
			&s.StoreID, &s.StoreName, &s.OrderCount, &s.PaidOrderCount,
			&s.GrossCent, &s.FoodOrderCount, &s.FoodGrossCent,
			&s.ActivityOrderCount, &s.ActivityGrossCent, &s.UniqueMemberCount,
			&s.ReservationCount, &s.CouponRedemptionCount,
		); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// RollupDaily recomputes the reporting_daily pre-aggregates for the requested
// window in one transaction. Each metric is cleared and rewritten for the
// affected (report_date, store) partition, so a run is idempotent and always
// converges to the live aggregate — a full recompute when the request is
// unbounded, a targeted refresh when a date/store is pinned.
func (r *sqlRepository) RollupDaily(ctx context.Context, req RollupRequest) (RollupResult, error) {
	var res RollupResult
	now := time.Now().UTC()
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		// Revenue: order count and paid gross per store/day. store_id is nullable
		// (wallet recharges carry no store), so its NULL group becomes the
		// store-less revenue bucket.
		rev, err := rollupMetric(ctx, tx, req, now, MetricRevenue,
			`SELECT store_id, DATE(created_at) AS d,
				COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN total_amount_cent ELSE 0 END), 0) AS amt,
				COUNT(*) AS qty
			FROM business_orders WHERE 1 = 1`,
			"created_at", "store_id")
		if err != nil {
			return err
		}
		res.RevenueRows = rev

		// Reservations: booking count per store/day (no monetary amount).
		resv, err := rollupMetric(ctx, tx, req, now, MetricReservations,
			`SELECT store_id, DATE(reserved_at) AS d, 0 AS amt, COUNT(*) AS qty
			FROM reservations WHERE 1 = 1`,
			"reserved_at", "store_id")
		if err != nil {
			return err
		}
		res.ReservationRows = resv
		return nil
	})
	if err != nil {
		return RollupResult{}, apperr.Internal(err)
	}
	return res, nil
}

// rollupMetric clears then rewrites one metric's rows for the request window.
// sourceSelect must project (store_id, DATE(dateCol) AS d, <amount> AS amt,
// <count> AS qty) with a trailing "WHERE 1 = 1"; the date/store window is
// appended before grouping. dateCol/storeCol are code-controlled column names
// (never request input), so interpolating them is safe. It returns the number of
// aggregate rows written.
func rollupMetric(ctx context.Context, tx *sql.Tx, req RollupRequest, now time.Time, metric, sourceSelect, dateCol, storeCol string) (int64, error) {
	// Source-side window on the raw table: optional store filter and inclusive
	// [From, To] date bounds compared on the calendar day.
	srcWhere := ""
	var srcArgs []any
	if req.StoreID != nil {
		srcWhere += " AND " + storeCol + " = ?"
		srcArgs = append(srcArgs, *req.StoreID)
	}
	if req.From != nil {
		srcWhere += " AND DATE(" + dateCol + ") >= ?"
		srcArgs = append(srcArgs, *req.From)
	}
	if req.To != nil {
		srcWhere += " AND DATE(" + dateCol + ") <= ?"
		srcArgs = append(srcArgs, *req.To)
	}

	// Matching window on reporting_daily so the delete clears exactly the
	// partition the insert is about to rewrite.
	delWhere := ""
	var delArgs []any
	if req.From != nil {
		delWhere += " AND report_date >= ?"
		delArgs = append(delArgs, *req.From)
	}
	if req.To != nil {
		delWhere += " AND report_date <= ?"
		delArgs = append(delArgs, *req.To)
	}
	if req.StoreID != nil {
		delWhere += " AND store_id = ?"
		delArgs = append(delArgs, *req.StoreID)
	}

	delQ := `DELETE FROM reporting_daily WHERE metric = ?` + delWhere
	if _, err := tx.ExecContext(ctx, delQ, append([]any{metric}, delArgs...)...); err != nil {
		return 0, err
	}

	insQ := `INSERT INTO reporting_daily (store_id, report_date, metric, amount_cent, quantity, created_at, updated_at)
		SELECT store_id, d, ?, amt, qty, ?, ?
		FROM (` + sourceSelect + srcWhere + ` GROUP BY store_id, DATE(` + dateCol + `)) agg`
	insArgs := append([]any{metric, now, now}, srcArgs...)
	result, err := tx.ExecContext(ctx, insQ, insArgs...)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}
