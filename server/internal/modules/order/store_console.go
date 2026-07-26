package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/modules/payment"
	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

type StoreFoodOrderItemView struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	UnitPriceCent int64  `json:"unitPriceCent"`
	Quantity      int    `json:"quantity"`
	SubtotalCent  int64  `json:"subtotalCent"`
	PointsReward  int64  `json:"pointsReward"`
	AssetID       *int64 `json:"assetId,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
}

type StoreFoodOrderView struct {
	ID              int64                    `json:"id"`
	BusinessOrderID string                   `json:"businessOrderId"`
	Status          string                   `json:"status"`
	PaymentStatus   string                   `json:"paymentStatus"`
	PayChannel      string                   `json:"payChannel,omitempty"`
	AmountCent      int64                    `json:"amountCent"`
	PaidAmountCent  int64                    `json:"paidAmountCent"`
	PaymentOrderID  int64                    `json:"paymentOrderId"`
	PointsEarned    int64                    `json:"pointsEarned"`
	TableName       string                   `json:"tableName,omitempty"`
	ItemsSummary    string                   `json:"itemsSummary,omitempty"`
	Items           []StoreFoodOrderItemView `json:"items"`
	MemberNickname  string                   `json:"memberNickname,omitempty"`
	MemberPhone     string                   `json:"memberPhone,omitempty"`
	MemberAvatarURL string                   `json:"memberAvatarUrl,omitempty"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

type StoreFoodOrderFilter struct {
	Status         string
	PaymentStatus  string
	PayChannel     string
	MemberNickname string
	MemberPhone    string
	OrderNo        string
	ItemName       string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	Keyword        string
	Page           httpx.Page
}

type StoreConsoleRepository interface {
	ListStoreFoodOrders(context.Context, int64, StoreFoodOrderFilter) ([]StoreFoodOrderView, int64, error)
	GetStoreFoodOrder(context.Context, int64, int64) (StoreFoodOrderView, error)
	TransitionStoreFoodOrder(context.Context, int64, int64, string, string) (bool, error)
	PrepareFoodOrderCancellation(context.Context, FoodOrderCancellationInput) (FoodOrderCancellation, error)
	CompleteFoodOrderCancellation(context.Context, int64, int64, time.Time) error
	RollbackFoodOrderCancellation(context.Context, int64, string, time.Time) error
}

type FoodOrderCancellationInput struct {
	StoreID, FoodOrderID, RequestedByID int64
	RequestedByType, IdemKey            string
	Forced                              bool
	Now                                 time.Time
}

type FoodOrderCancellation struct {
	ID, FoodOrderID, PaymentOrderID, AmountCent    int64
	PointsEarned, PointsRecovered, PointsShortfall int64
}

type sqlStoreConsoleRepository struct{ db *platdb.DB }

func NewStoreConsoleRepository(db *platdb.DB) StoreConsoleRepository {
	return &sqlStoreConsoleRepository{db: db}
}

func storeFoodOrderWhere(storeID int64, f StoreFoodOrderFilter) (string, []any) {
	where := "fo.store_id = ?"
	args := []any{storeID}
	if f.Status != "" {
		where += " AND fo.fulfillment_status = ?"
		args = append(args, f.Status)
	}
	if f.PaymentStatus != "" {
		where += " AND bo.payment_status = ?"
		args = append(args, f.PaymentStatus)
	}
	if f.PayChannel != "" {
		where += " AND po.pay_method = ?"
		args = append(args, f.PayChannel)
	}
	if f.MemberNickname != "" {
		where += " AND COALESCE(m.nickname, '') LIKE ?"
		args = append(args, "%"+f.MemberNickname+"%")
	}
	if f.MemberPhone != "" {
		where += " AND COALESCE(m.phone, '') LIKE ?"
		args = append(args, "%"+f.MemberPhone+"%")
	}
	if f.OrderNo != "" {
		where += " AND bo.business_order_no LIKE ?"
		args = append(args, "%"+f.OrderNo+"%")
	}
	if f.ItemName != "" {
		where += " AND EXISTS (SELECT 1 FROM food_order_items sx WHERE sx.food_order_id = fo.id AND sx.name_snapshot LIKE ?)"
		args = append(args, "%"+f.ItemName+"%")
	}
	if f.CreatedFrom != nil {
		where += " AND fo.created_at >= ?"
		args = append(args, *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		where += " AND fo.created_at <= ?"
		args = append(args, *f.CreatedTo)
	}
	if f.Keyword != "" {
		where += ` AND (bo.business_order_no LIKE ? OR COALESCE(t.name, '') LIKE ?
			OR COALESCE(m.nickname, '') LIKE ? OR COALESCE(m.phone, '') LIKE ?
			OR EXISTS (SELECT 1 FROM food_order_items sx WHERE sx.food_order_id = fo.id AND sx.name_snapshot LIKE ?))`
		like := "%" + f.Keyword + "%"
		args = append(args, like, like, like, like, like)
	}
	return where, args
}

func (r *sqlStoreConsoleRepository) ListStoreFoodOrders(ctx context.Context, storeID int64, f StoreFoodOrderFilter) ([]StoreFoodOrderView, int64, error) {
	where, args := storeFoodOrderWhere(storeID, f)
	countQ := `SELECT COUNT(*) FROM food_orders fo
		JOIN business_orders bo ON bo.id = fo.business_order_id
		JOIN payment_orders po ON po.business_order_id = bo.id
		LEFT JOIN tables t ON t.id = fo.table_id
		LEFT JOIN members m ON m.id = fo.member_id
		WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT fo.id, bo.business_order_no, fo.fulfillment_status, bo.payment_status,
		po.pay_method, fo.total_amount_cent,
		COALESCE((SELECT pt.amount_cent FROM payment_transactions pt WHERE pt.payment_order_id = po.id AND pt.status = 'success' ORDER BY pt.id DESC LIMIT 1), 0),
		po.id, fo.points_earned, COALESCE(t.name, ''),
		COALESCE(m.nickname, ''), COALESCE(m.phone, ''), COALESCE(m.avatar_url, ''), fo.created_at, fo.updated_at
		FROM food_orders fo
		JOIN business_orders bo ON bo.id = fo.business_order_id
		JOIN payment_orders po ON po.business_order_id = bo.id
		LEFT JOIN tables t ON t.id = fo.table_id
		LEFT JOIN members m ON m.id = fo.member_id
		WHERE ` + where + ` ORDER BY fo.id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, queryArgs...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	items := make([]StoreFoodOrderView, 0)
	for rows.Next() {
		var item StoreFoodOrderView
		if err := rows.Scan(&item.ID, &item.BusinessOrderID, &item.Status, &item.PaymentStatus,
			&item.PayChannel, &item.AmountCent, &item.PaidAmountCent, &item.PaymentOrderID,
			&item.PointsEarned, &item.TableName, &item.MemberNickname, &item.MemberPhone,
			&item.MemberAvatarURL, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if err := r.loadStoreFoodOrderItems(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *sqlStoreConsoleRepository) GetStoreFoodOrder(ctx context.Context, storeID, id int64) (StoreFoodOrderView, error) {
	q := `SELECT fo.id, bo.business_order_no, fo.fulfillment_status, bo.payment_status,
		po.pay_method, fo.total_amount_cent,
		COALESCE((SELECT pt.amount_cent FROM payment_transactions pt WHERE pt.payment_order_id = po.id AND pt.status = 'success' ORDER BY pt.id DESC LIMIT 1), 0),
		po.id, fo.points_earned, COALESCE(t.name, ''),
		COALESCE(m.nickname, ''), COALESCE(m.phone, ''), COALESCE(m.avatar_url, ''), fo.created_at, fo.updated_at
		FROM food_orders fo
		JOIN business_orders bo ON bo.id = fo.business_order_id
		JOIN payment_orders po ON po.business_order_id = bo.id
		LEFT JOIN tables t ON t.id = fo.table_id
		LEFT JOIN members m ON m.id = fo.member_id
		WHERE fo.id = ? AND fo.store_id = ?`
	var item StoreFoodOrderView
	err := r.db.QueryRowContext(ctx, q, id, storeID).Scan(
		&item.ID, &item.BusinessOrderID, &item.Status, &item.PaymentStatus,
		&item.PayChannel, &item.AmountCent, &item.PaidAmountCent, &item.PaymentOrderID,
		&item.PointsEarned, &item.TableName, &item.MemberNickname, &item.MemberPhone,
		&item.MemberAvatarURL, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return StoreFoodOrderView{}, apperr.NotFound("food order not found")
	}
	if err != nil {
		return StoreFoodOrderView{}, apperr.Internal(err)
	}
	items := []StoreFoodOrderView{item}
	if err := r.loadStoreFoodOrderItems(ctx, items); err != nil {
		return StoreFoodOrderView{}, err
	}
	item = items[0]
	return item, nil
}

func (r *sqlStoreConsoleRepository) loadStoreFoodOrderItems(ctx context.Context, orders []StoreFoodOrderView) error {
	if len(orders) == 0 {
		return nil
	}
	marks := make([]string, len(orders))
	args := make([]any, len(orders))
	index := make(map[int64]int, len(orders))
	for i := range orders {
		marks[i] = "?"
		args[i] = orders[i].ID
		index[orders[i].ID] = i
		orders[i].Items = []StoreFoodOrderItemView{}
	}
	q := `SELECT foi.id, foi.food_order_id, foi.name_snapshot, foi.unit_price_cent, foi.quantity,
		foi.subtotal_cent, foi.points_reward_snapshot, ci.asset_id
		FROM food_order_items foi
		LEFT JOIN catalog_items ci ON ci.id = foi.item_id
		WHERE foi.food_order_id IN (` + strings.Join(marks, ",") + `) ORDER BY foi.id ASC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return apperr.Internal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var foodOrderID int64
		var line StoreFoodOrderItemView
		if err := rows.Scan(&line.ID, &foodOrderID, &line.Name, &line.UnitPriceCent, &line.Quantity,
			&line.SubtotalCent, &line.PointsReward, &line.AssetID); err != nil {
			return apperr.Internal(err)
		}
		i := index[foodOrderID]
		orders[i].Items = append(orders[i].Items, line)
	}
	for i := range orders {
		parts := make([]string, 0, len(orders[i].Items))
		for _, line := range orders[i].Items {
			parts = append(parts, fmt.Sprintf("%s ×%d", line.Name, line.Quantity))
		}
		orders[i].ItemsSummary = strings.Join(parts, "、")
	}
	return rows.Err()
}

func (r *sqlStoreConsoleRepository) TransitionStoreFoodOrder(ctx context.Context, storeID, id int64, from, to string) (bool, error) {
	var affected int64
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE food_orders fo
			JOIN business_orders bo ON bo.id = fo.business_order_id
			SET fo.fulfillment_status = ?, fo.updated_at = NOW(), bo.order_status = ?, bo.updated_at = NOW()
			WHERE fo.id = ? AND fo.store_id = ? AND fo.fulfillment_status = ?`, to, to, id, storeID, from)
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		if err != nil || affected != 1 {
			return err
		}
		if to == "cancelled" {
			_, err = tx.ExecContext(ctx, `UPDATE payment_orders SET status = 'expired', updated_at = NOW()
				WHERE business_order_id = (SELECT business_order_id FROM food_orders WHERE id = ? AND store_id = ?)
				AND status = 'pending'`, id, storeID)
		}
		return err
	})
	if err != nil {
		return false, apperr.Internal(err)
	}
	return affected == 1, nil
}

func (r *sqlStoreConsoleRepository) PrepareFoodOrderCancellation(ctx context.Context, in FoodOrderCancellationInput) (FoodOrderCancellation, error) {
	var out FoodOrderCancellation
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var memberID, businessID int64
		var originalStatus, paymentStatus, paymentOrderStatus string
		const lock = `SELECT fo.member_id, fo.business_order_id, fo.points_earned, fo.fulfillment_status,
			bo.payment_status, po.id, po.amount_cent, po.status
			FROM food_orders fo
			JOIN business_orders bo ON bo.id = fo.business_order_id
			JOIN payment_orders po ON po.business_order_id = bo.id
			WHERE fo.id = ? AND fo.store_id = ? ORDER BY po.id DESC LIMIT 1 FOR UPDATE`
		err := tx.QueryRowContext(ctx, lock, in.FoodOrderID, in.StoreID).Scan(
			&memberID, &businessID, &out.PointsEarned, &originalStatus,
			&paymentStatus, &out.PaymentOrderID, &out.AmountCent, &paymentOrderStatus,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("点餐订单不存在")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if originalStatus == "cancelled" || paymentStatus == "refunded" || paymentOrderStatus == "refunded" {
			return apperr.Conflict("该订单已经取消")
		}
		if paymentStatus != "paid" || paymentOrderStatus != "paid" {
			return apperr.Conflict("只有已支付订单可以取消并退款")
		}
		var existing int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM food_order_cancellations WHERE food_order_id = ? AND status IN ('processing','succeeded')`, in.FoodOrderID).Scan(&existing); err != nil {
			return apperr.Internal(err)
		}
		if existing > 0 {
			return apperr.Conflict("该订单已经在取消处理中")
		}

		var accountID, available int64
		if out.PointsEarned > 0 {
			err := tx.QueryRowContext(ctx, `SELECT id, available_amount FROM wallet_accounts WHERE member_id = ? AND asset_type = 'points' FOR UPDATE`, memberID).Scan(&accountID, &available)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return apperr.Internal(err)
			}
			if !in.Forced && available < out.PointsEarned {
				return apperr.Conflict("用户当前积分不足，无法取消订单；如需继续请使用强制取消")
			}
			out.PointsRecovered = out.PointsEarned
			if available < out.PointsRecovered {
				out.PointsRecovered = available
			}
			out.PointsShortfall = out.PointsEarned - out.PointsRecovered
		}

		const insertCancel = `INSERT INTO food_order_cancellations
			(food_order_id, payment_order_id, store_id, member_id, original_status,
			 points_earned, points_recovered, points_shortfall, forced, status,
			 requested_by_type, requested_by_id, idem_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'processing', ?, ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertCancel, in.FoodOrderID, out.PaymentOrderID, in.StoreID, memberID,
			originalStatus, out.PointsEarned, out.PointsRecovered, out.PointsShortfall, in.Forced,
			in.RequestedByType, in.RequestedByID, in.IdemKey, in.Now, in.Now)
		if err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("该订单已经在取消处理中")
			}
			return apperr.Internal(err)
		}
		out.ID, err = res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		out.FoodOrderID = in.FoodOrderID
		if out.PointsRecovered > 0 {
			newBalance := available - out.PointsRecovered
			if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`, newBalance, in.Now, accountID); err != nil {
				return apperr.Internal(err)
			}
			idem := fmt.Sprintf("food_order_cancel_points:%d", in.FoodOrderID)
			if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
				(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
				VALUES (?, ?, 'points', 'debit', ?, ?, 'food_order_cancel_clawback', 'food_order', ?, ?, ?)`,
				accountID, memberID, out.PointsRecovered, newBalance, in.FoodOrderID, idem, in.Now); err != nil {
				return apperr.Internal(err)
			}
		}
		_ = businessID
		return nil
	})
	if err != nil {
		return FoodOrderCancellation{}, err
	}
	return out, nil
}

func (r *sqlStoreConsoleRepository) CompleteFoodOrderCancellation(ctx context.Context, cancellationID, refundID int64, now time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var foodOrderID int64
		if err := tx.QueryRowContext(ctx, `SELECT food_order_id FROM food_order_cancellations WHERE id = ? AND status = 'processing' FOR UPDATE`, cancellationID).Scan(&foodOrderID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.Conflict("取消记录状态已变化")
			}
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE food_order_cancellations SET refund_order_id = ?, status = 'succeeded', updated_at = ? WHERE id = ?`, refundID, now, cancellationID); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE food_orders SET fulfillment_status = 'cancelled', updated_at = ? WHERE id = ?`, now, foodOrderID); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

func (r *sqlStoreConsoleRepository) RollbackFoodOrderCancellation(ctx context.Context, cancellationID int64, failure string, now time.Time) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var foodOrderID, memberID, recovered int64
		var originalStatus, status string
		const lock = `SELECT food_order_id, member_id, points_recovered, original_status, status FROM food_order_cancellations WHERE id = ? FOR UPDATE`
		if err := tx.QueryRowContext(ctx, lock, cancellationID).Scan(&foodOrderID, &memberID, &recovered, &originalStatus, &status); err != nil {
			return apperr.Internal(err)
		}
		if status != "processing" {
			return nil
		}
		if recovered > 0 {
			var accountID, available int64
			if err := tx.QueryRowContext(ctx, `SELECT id, available_amount FROM wallet_accounts WHERE member_id = ? AND asset_type = 'points' FOR UPDATE`, memberID).Scan(&accountID, &available); err != nil {
				return apperr.Internal(err)
			}
			newBalance := available + recovered
			if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`, newBalance, now, accountID); err != nil {
				return apperr.Internal(err)
			}
			idem := fmt.Sprintf("food_order_cancel_rollback:%d", cancellationID)
			if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
				(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
				VALUES (?, ?, 'points', 'credit', ?, ?, 'food_order_cancel_rollback', 'food_order', ?, ?, ?)`,
				accountID, memberID, recovered, newBalance, foodOrderID, idem, now); err != nil {
				return apperr.Internal(err)
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE food_order_cancellations SET status = 'failed', failure_reason = ?, updated_at = ? WHERE id = ?`, failure, now, cancellationID); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE food_orders SET fulfillment_status = ?, updated_at = ? WHERE id = ?`, originalStatus, now, foodOrderID); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

func maskStorePhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

type FoodOrderRefundService interface {
	CreateFoodOrderCancellationRefund(context.Context, int64, string, int64, string, payment.CreateRefundRequest) (payment.AdminRefundView, error)
}

type AccountPasswordVerifier interface {
	VerifyAccountPassword(context.Context, int64, string) error
}

type StoreConsoleService struct {
	repo      StoreConsoleRepository
	refunds   FoodOrderRefundService
	passwords AccountPasswordVerifier
	assets    AssetResolver
}

func NewStoreConsoleService(repo StoreConsoleRepository, refunds FoodOrderRefundService, passwords AccountPasswordVerifier, assets AssetResolver) *StoreConsoleService {
	return &StoreConsoleService{repo: repo, refunds: refunds, passwords: passwords, assets: assets}
}

func (s *StoreConsoleService) List(ctx context.Context, storeID int64, f StoreFoodOrderFilter) ([]StoreFoodOrderView, int64, error) {
	if !validFoodStatus(f.Status) {
		return nil, 0, apperr.Invalid("invalid food order status")
	}
	if !validFoodPaymentStatus(f.PaymentStatus) {
		return nil, 0, apperr.Invalid("invalid payment status")
	}
	if f.PayChannel != "" && f.PayChannel != "wechat" && f.PayChannel != "coin" {
		return nil, 0, apperr.Invalid("invalid pay channel")
	}
	if f.CreatedFrom != nil && f.CreatedTo != nil && f.CreatedFrom.After(*f.CreatedTo) {
		return nil, 0, apperr.Invalid("createdFrom must not be after createdTo")
	}
	return s.repo.ListStoreFoodOrders(ctx, storeID, f)
}

func (s *StoreConsoleService) Get(ctx context.Context, storeID, id int64) (StoreFoodOrderView, error) {
	item, err := s.repo.GetStoreFoodOrder(ctx, storeID, id)
	if err != nil {
		return StoreFoodOrderView{}, err
	}
	if s.assets != nil {
		for i := range item.Items {
			if item.Items[i].AssetID != nil {
				item.Items[i].ImageURL, _ = s.assets.PublicURLByID(ctx, *item.Items[i].AssetID)
			}
		}
	}
	return item, nil
}

func (s *StoreConsoleService) Action(ctx context.Context, storeID, id int64, action, byType string, byID int64, idemKey, password string) (StoreFoodOrderView, error) {
	current, err := s.repo.GetStoreFoodOrder(ctx, storeID, id)
	if err != nil {
		return StoreFoodOrderView{}, err
	}
	if action == "cancel" || action == "force-cancel" {
		if current.PaymentStatus != "paid" {
			return StoreFoodOrderView{}, apperr.Conflict("只有已支付订单可以取消并退款")
		}
		forced := action == "force-cancel"
		if forced {
			if s.passwords == nil {
				return StoreFoodOrderView{}, apperr.Internal(fmt.Errorf("password verifier is not configured"))
			}
			if err := s.passwords.VerifyAccountPassword(ctx, byID, password); err != nil {
				return StoreFoodOrderView{}, err
			}
		}
		if s.refunds == nil {
			return StoreFoodOrderView{}, apperr.Internal(fmt.Errorf("refund service is not configured"))
		}
		now := time.Now().UTC()
		prepared, err := s.repo.PrepareFoodOrderCancellation(ctx, FoodOrderCancellationInput{
			StoreID: storeID, FoodOrderID: id, RequestedByType: byType, RequestedByID: byID,
			IdemKey: idemKey, Forced: forced, Now: now,
		})
		if err != nil {
			return StoreFoodOrderView{}, err
		}
		reason := "门店取消点餐订单"
		if forced {
			reason = "门店强制取消点餐订单"
		}
		if prepared.PointsShortfall > 0 {
			reason += fmt.Sprintf("（赠送积分未追回 %d）", prepared.PointsShortfall)
		}
		refundIdemKey := idemKey
		if len(refundIdemKey) > 120 {
			refundIdemKey = refundIdemKey[:120]
		}
		refund, err := s.refunds.CreateFoodOrderCancellationRefund(ctx, storeID, byType, byID,
			refundIdemKey+":refund", payment.CreateRefundRequest{PaymentOrderID: prepared.PaymentOrderID, AmountCent: prepared.AmountCent, Reason: reason})
		if err != nil {
			failureRunes := []rune(err.Error())
			if len(failureRunes) > 255 {
				failureRunes = failureRunes[:255]
			}
			failure := string(failureRunes)
			_ = s.repo.RollbackFoodOrderCancellation(ctx, prepared.ID, failure, now)
			return StoreFoodOrderView{}, err
		}
		if err := s.repo.CompleteFoodOrderCancellation(ctx, prepared.ID, refund.ID, now); err != nil {
			return StoreFoodOrderView{}, err
		}
		return s.Get(ctx, storeID, id)
	}
	to, ok := foodTransition(current.Status, action)
	if !ok {
		return StoreFoodOrderView{}, apperr.Conflict("当前订单状态不能执行该操作")
	}
	if action != "cancel" && current.PaymentStatus != "paid" {
		return StoreFoodOrderView{}, apperr.Conflict("订单尚未支付，不能处理")
	}
	changed, err := s.repo.TransitionStoreFoodOrder(ctx, storeID, id, current.Status, to)
	if err != nil {
		return StoreFoodOrderView{}, err
	}
	if !changed {
		return StoreFoodOrderView{}, apperr.Conflict("订单状态已变化，请刷新后重试")
	}
	return s.Get(ctx, storeID, id)
}

func validFoodStatus(status string) bool {
	switch status {
	case "", "pending", "confirmed", "preparing", "ready", "completed", "cancelled":
		return true
	default:
		return false
	}
}

func validFoodPaymentStatus(status string) bool {
	switch status {
	case "", "unpaid", "paid", "refunded", "partially_refunded":
		return true
	default:
		return false
	}
}

func foodTransition(status, action string) (string, bool) {
	transitions := map[string]map[string]string{
		"pending":   {"confirm": "confirmed", "cancel": "cancelled"},
		"confirmed": {"prepare": "preparing", "cancel": "cancelled"},
		"preparing": {"ready": "ready"},
		"ready":     {"complete": "completed"},
		"completed": {},
	}
	to, ok := transitions[status][action]
	return to, ok
}

type StoreConsoleHandler struct{ svc *StoreConsoleService }

func NewStoreConsoleHandler(svc *StoreConsoleService) *StoreConsoleHandler {
	return &StoreConsoleHandler{svc: svc}
}

func (h *StoreConsoleHandler) List(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	createdFrom, err := optionalQueryTime(c.Query("createdFrom"))
	if err != nil {
		httpx.Fail(c, apperr.Invalid("createdFrom must be RFC3339"))
		return
	}
	createdTo, err := optionalQueryTime(c.Query("createdTo"))
	if err != nil {
		httpx.Fail(c, apperr.Invalid("createdTo must be RFC3339"))
		return
	}
	f := StoreFoodOrderFilter{
		Status:         strings.TrimSpace(c.Query("orderStatus")),
		PaymentStatus:  strings.TrimSpace(c.Query("paymentStatus")),
		PayChannel:     strings.TrimSpace(c.Query("payChannel")),
		MemberNickname: strings.TrimSpace(c.Query("memberNickname")),
		MemberPhone:    strings.TrimSpace(c.Query("memberPhone")),
		OrderNo:        strings.TrimSpace(c.Query("orderNo")),
		ItemName:       strings.TrimSpace(c.Query("itemName")),
		CreatedFrom:    createdFrom,
		CreatedTo:      createdTo,
		Keyword:        strings.TrimSpace(c.Query("keyword")),
		Page:           httpx.ParsePage(c),
	}
	items, total, err := h.svc.List(c.Request.Context(), storeID, f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, items, httpx.MetaFor(f.Page, total))
}

func optionalQueryTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *StoreConsoleHandler) Get(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := positivePathID(c, "orderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	item, err := h.svc.Get(c.Request.Context(), storeID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *StoreConsoleHandler) Action(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := positivePathID(c, "orderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	claims := authn.MustFromContext(c)
	var req struct {
		Password string `json:"password"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, apperr.Invalid("请求参数错误"))
			return
		}
	}
	item, err := h.svc.Action(c.Request.Context(), storeID, id, c.Param("action"), string(claims.SubjectType), claims.SubjectID(), idempotency.Key(c), req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func positivePathID(c *gin.Context, name string) (int64, error) {
	var id int64
	if _, err := fmt.Sscan(c.Param(name), &id); err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
