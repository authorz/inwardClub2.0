package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// The offline collection channel is resolved by the acquirer at callback time,
// so a freshly created payment order carries no concrete pay_method yet.
const (
	offlinePayMethod = "offline"
	offlineProvider  = "offline_acquirer"
	collectionType   = "offline_collection"
)

type storeSQLRepository struct{ db *platdb.DB }

// NewStoreRepository builds the MySQL store-scoped payment repository.
func NewStoreRepository(db *platdb.DB) StoreRepository { return &storeSQLRepository{db: db} }

// ResolveMemberByPhone matches a registered member by exact phone and returns
// only masked identifiers. When several rows share a phone (the members schema
// does not enforce uniqueness) the lowest id wins deterministically. A miss maps
// to MEMBER_NOT_FOUND so the operator can proceed without binding a member. The
// raw phone is neither returned nor logged.
func (r *storeSQLRepository) ResolveMemberByPhone(ctx context.Context, phone string) (MemberMatch, error) {
	const q = `SELECT id, COALESCE(nickname,''), COALESCE(phone,'')
		FROM members WHERE phone = ? ORDER BY id ASC LIMIT 1`
	var (
		id       int64
		nickname string
		stored   string
	)
	err := r.db.QueryRowContext(ctx, q, phone).Scan(&id, &nickname, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		return MemberMatch{}, apperr.New(apperr.CodeMemberNotFound, "member not found")
	}
	if err != nil {
		return MemberMatch{}, apperr.Internal(err)
	}
	return MemberMatch{ID: id, Nickname: nickname, PhoneMasked: maskPhone(stored)}, nil
}

// CreateCollectionOrder inserts the business, payment and offline collection
// rows in one transaction and claims the idempotency key as the duplicate guard.
func (r *storeSQLRepository) CreateCollectionOrder(ctx context.Context, in CollectionOrderCreate) (CollectionOrder, error) {
	var out CollectionOrder
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if in.IdemKey != "" {
			if err := idempotency.Claim(ctx, tx, "store/offline-collection-orders", in.IdemKey, collectionType, 0); err != nil {
				return err
			}
		}
		// A bound member is fixed on the whole order spine so the payment-order read
		// model and the settlement post-processing both see it; nil stays a walk-in.
		var memberArg any
		if in.MemberID != nil {
			memberArg = *in.MemberID
		}
		const insBusiness = `INSERT INTO business_orders
			(business_order_no, order_type, store_id, member_id, total_amount_cent, order_status, payment_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'created', 'unpaid', ?, ?)`
		res, err := tx.ExecContext(ctx, insBusiness, in.BusinessOrderNo, collectionType, in.StoreID, memberArg, in.AmountCent, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		businessID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		const insPayment = `INSERT INTO payment_orders
			(payment_order_no, business_order_id, store_id, member_id, amount_cent, pay_method, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)`
		res, err = tx.ExecContext(ctx, insPayment, in.PaymentOrderNo, businessID, in.StoreID, memberArg, in.AmountCent, offlinePayMethod, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		paymentID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		// Binding audit columns are populated together (the creating operator is the
		// binder) or all left NULL for a walk-in collection.
		var boundByType, phoneMasked any
		var boundByID, boundAt any
		if in.MemberID != nil {
			boundByType = in.CreatedByType
			boundByID = in.CreatedByID
			boundAt = in.Now
			phoneMasked = in.MemberPhoneMasked
		}
		const insCollection = `INSERT INTO offline_collection_orders
			(collection_order_no, store_id, payment_order_id, amount_cent, subject, business_type,
			 member_id, member_phone_masked, bound_by_type, bound_by_id, bound_at,
			 acquirer_order_no, qr_content, status, created_by_type, created_by_id, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		res, err = tx.ExecContext(ctx, insCollection, in.CollectionOrderNo, in.StoreID, paymentID, in.AmountCent,
			in.Subject, in.BusinessType, memberArg, phoneMasked, boundByType, boundByID, boundAt,
			in.AcquirerOrderNo, in.QRContent, CollectionPending,
			in.CreatedByType, in.CreatedByID, in.ExpiresAt, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		out = CollectionOrder{
			ID:                id,
			CollectionOrderNo: in.CollectionOrderNo,
			StoreID:           in.StoreID,
			PaymentOrderID:    paymentID,
			AmountCent:        in.AmountCent,
			Subject:           in.Subject,
			BusinessType:      in.BusinessType,
			Status:            CollectionPending,
			MemberID:          in.MemberID,
			MemberPhoneMasked: in.MemberPhoneMasked,
			AcquirerOrderNo:   in.AcquirerOrderNo,
			QRContent:         in.QRContent,
			ExpiresAt:         in.ExpiresAt,
			CreatedAt:         in.Now,
		}
		return nil
	})
	if err != nil {
		return CollectionOrder{}, err
	}
	return out, nil
}

const collectionColumns = `id, collection_order_no, store_id, payment_order_id, amount_cent,
	subject, business_type, member_id, COALESCE(member_phone_masked,''),
	COALESCE(acquirer_order_no,''), COALESCE(qr_content,''), status, expires_at, created_at`

func scanCollection(row interface{ Scan(...any) error }) (CollectionOrder, error) {
	var o CollectionOrder
	var memberID sql.NullInt64
	err := row.Scan(&o.ID, &o.CollectionOrderNo, &o.StoreID, &o.PaymentOrderID, &o.AmountCent,
		&o.Subject, &o.BusinessType, &memberID, &o.MemberPhoneMasked,
		&o.AcquirerOrderNo, &o.QRContent, &o.Status, &o.ExpiresAt, &o.CreatedAt)
	if memberID.Valid {
		o.MemberID = &memberID.Int64
	}
	return o, err
}

// GetCollectionOrder reads an order pinned to the acting store's scope.
func (r *storeSQLRepository) GetCollectionOrder(ctx context.Context, storeID, id int64) (CollectionOrder, error) {
	const q = `SELECT ` + collectionColumns + ` FROM offline_collection_orders WHERE id = ? AND store_id = ?`
	o, err := scanCollection(r.db.QueryRowContext(ctx, q, id, storeID))
	if errors.Is(err, sql.ErrNoRows) {
		return CollectionOrder{}, apperr.NotFound("collection order not found")
	}
	if err != nil {
		return CollectionOrder{}, apperr.Internal(err)
	}
	return o, nil
}

// CancelCollectionOrder cancels a still-pending order owned by the store.
func (r *storeSQLRepository) CancelCollectionOrder(ctx context.Context, storeID, id int64, now time.Time) error {
	const q = `UPDATE offline_collection_orders SET status = ?, updated_at = ?
		WHERE id = ? AND store_id = ? AND status = ?`
	res, err := r.db.ExecContext(ctx, q, CollectionCancelled, now, id, storeID, CollectionPending)
	if err != nil {
		return apperr.Internal(err)
	}
	return affectedOrConflict(res, "collection order cannot be cancelled")
}

// CreateRefund inserts a pending refund only when the payment order belongs to
// the acting store; the store scope is verified inside the same transaction.
func (r *storeSQLRepository) CreateRefund(ctx context.Context, in RefundCreate) (Refund, error) {
	var out Refund
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if in.IdemKey != "" {
			if err := idempotency.Claim(ctx, tx, "store/refunds", in.IdemKey, "refund", 0); err != nil {
				return err
			}
		}
		var businessID, paidAmount int64
		var channel string
		var poStore sql.NullInt64
		const sel = `SELECT po.business_order_id, po.amount_cent, po.store_id, bo.order_type
			FROM payment_orders po JOIN business_orders bo ON bo.id = po.business_order_id
			WHERE po.id = ?`
		var orderType string
		err := tx.QueryRowContext(ctx, sel, in.PaymentOrderID).Scan(&businessID, &paidAmount, &poStore, &orderType)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("payment order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		// Store scope guard: a store may only refund its own payment orders.
		if !poStore.Valid || poStore.Int64 != in.StoreID {
			return apperr.NotFound("payment order not found")
		}
		if in.AmountCent > paidAmount {
			return apperr.Invalid("refund amount exceeds paid amount")
		}
		channel = channelForOrderType(orderType)
		const ins = `INSERT INTO refund_orders
			(refund_order_no, payment_order_id, business_order_id, store_id, amount_cent, channel, status,
			 reason, requested_by_type, requested_by_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, ins, in.RefundOrderNo, in.PaymentOrderID, businessID, in.StoreID,
			in.AmountCent, channel, RefundPending, in.Reason, in.RequestedByType, in.RequestedByID, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		out = Refund{
			ID:              id,
			RefundOrderNo:   in.RefundOrderNo,
			PaymentOrderID:  in.PaymentOrderID,
			BusinessOrderID: businessID,
			StoreID:         in.StoreID,
			AmountCent:      in.AmountCent,
			Channel:         channel,
			Status:          RefundPending,
			Reason:          in.Reason,
			CreatedAt:       in.Now,
		}
		return nil
	})
	if err != nil {
		return Refund{}, err
	}
	return out, nil
}

// CreateRefundAdmin inserts a pending refund for any store's payment order.
// The store scope is resolved from the payment order row itself rather than
// verified against a caller-supplied store_id, since admin refunds are not
// pinned to a single store.
func (r *storeSQLRepository) CreateRefundAdmin(ctx context.Context, in RefundCreate) (Refund, error) {
	var out Refund
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if in.IdemKey != "" {
			if err := idempotency.Claim(ctx, tx, "admin/refunds", in.IdemKey, "refund", 0); err != nil {
				return err
			}
		}
		var (
			businessID, paidAmount          int64
			poStore                         sql.NullInt64
			paymentOrderNo, payMethod       string
			paymentStatus, businessPayState string
			orderType, acquirerOrderNo      string
		)
		const sel = `SELECT po.business_order_id, po.amount_cent, po.store_id,
				po.payment_order_no, po.pay_method, po.status, bo.payment_status,
				bo.order_type, COALESCE(oco.acquirer_order_no, '')
			FROM payment_orders po
			JOIN business_orders bo ON bo.id = po.business_order_id
			LEFT JOIN offline_collection_orders oco ON oco.payment_order_id = po.id
			WHERE po.id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, sel, in.PaymentOrderID).Scan(
			&businessID, &paidAmount, &poStore, &paymentOrderNo, &payMethod,
			&paymentStatus, &businessPayState, &orderType, &acquirerOrderNo,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("payment order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if paymentStatus != paymentPaid || businessPayState != paymentPaid {
			return apperr.Conflict("只有已支付订单可以退款")
		}
		if in.AmountCent > paidAmount {
			return apperr.Invalid("退款金额不能超过订单实付金额")
		}
		if orderType == orderTypeRecharge {
			return apperr.Conflict("充值订单需先回收已发放权益，暂不支持直接退款")
		}
		var duplicateCount int
		const duplicate = `SELECT COUNT(*) FROM refund_orders
			WHERE payment_order_id = ? AND status IN (?, ?)`
		if err := tx.QueryRowContext(
			ctx, duplicate, in.PaymentOrderID, RefundProcessing, RefundSucceeded,
		).Scan(&duplicateCount); err != nil {
			return apperr.Internal(err)
		}
		if duplicateCount > 0 {
			return apperr.Conflict("该订单已发起退款")
		}

		channel := payMethod
		if channel == "" && acquirerOrderNo != "" {
			channel = "offline"
		}
		if channel != "wechat" && channel != "coin" && channel != "offline" {
			return apperr.Invalid("该支付渠道暂不支持退款")
		}
		var storeArg any
		var storeID int64
		if poStore.Valid {
			storeID = poStore.Int64
			storeArg = storeID
		}
		const ins = `INSERT INTO refund_orders
			(refund_order_no, payment_order_id, business_order_id, store_id, amount_cent, channel, status,
			 reason, requested_by_type, requested_by_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, ins, in.RefundOrderNo, in.PaymentOrderID, businessID, storeArg,
			in.AmountCent, channel, RefundProcessing, in.Reason, in.RequestedByType, in.RequestedByID, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		out = Refund{
			ID:                id,
			RefundOrderNo:     in.RefundOrderNo,
			PaymentOrderID:    in.PaymentOrderID,
			BusinessOrderID:   businessID,
			StoreID:           storeID,
			AmountCent:        in.AmountCent,
			PaymentAmountCent: paidAmount,
			Channel:           channel,
			Status:            RefundProcessing,
			Reason:            in.Reason,
			CreatedAt:         in.Now,
			PaymentOrderNo:    paymentOrderNo,
			PayMethod:         payMethod,
			OrderType:         orderType,
			AcquirerOrderNo:   acquirerOrderNo,
		}
		return nil
	})
	if err != nil {
		return Refund{}, err
	}
	return out, nil
}

// CompleteRefundAdmin atomically completes the internal refund state. Coin
// payments are credited back to the member wallet here; external channels have
// already accepted the refund before this method is called.
func (r *storeSQLRepository) CompleteRefundAdmin(
	ctx context.Context,
	refundID int64,
	externalRefundNo string,
	now time.Time,
) (Refund, error) {
	var out Refund
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var (
			storeID                          sql.NullInt64
			memberID                         sql.NullInt64
			paymentOrderID, businessOrderID  int64
			amountCent, paidAmount           int64
			refundNo, channel, refundStatus  string
			reason, payMethod, paymentStatus string
			businessPayStatus, orderType     string
			createdAt                        time.Time
		)
		const sel = `SELECT ro.refund_order_no, ro.payment_order_id, ro.business_order_id,
				ro.store_id, ro.amount_cent, ro.channel, ro.status, ro.reason, ro.created_at,
				po.member_id, po.pay_method, po.status, po.amount_cent, bo.payment_status, bo.order_type
			FROM refund_orders ro
			JOIN payment_orders po ON po.id = ro.payment_order_id
			JOIN business_orders bo ON bo.id = ro.business_order_id
			WHERE ro.id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, sel, refundID).Scan(
			&refundNo, &paymentOrderID, &businessOrderID, &storeID, &amountCent,
			&channel, &refundStatus, &reason, &createdAt, &memberID, &payMethod,
			&paymentStatus, &paidAmount, &businessPayStatus, &orderType,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("refund order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if refundStatus == RefundSucceeded {
			return nil
		}
		if refundStatus != RefundProcessing || paymentStatus != paymentPaid || businessPayStatus != paymentPaid {
			return apperr.Conflict("退款订单状态已变化")
		}

		if payMethod == "coin" {
			if !memberID.Valid {
				return apperr.Conflict("金币退款缺少会员信息")
			}
			if err := creditCoinRefund(
				ctx, tx, refundID, memberID.Int64, amountCent, now,
			); err != nil {
				return err
			}
		}

		var externalArg any
		if externalRefundNo != "" {
			externalArg = externalRefundNo
		}
		const complete = `UPDATE refund_orders
			SET status = ?, external_refund_no = ?, updated_at = ?
			WHERE id = ? AND status = ?`
		if _, err := tx.ExecContext(
			ctx, complete, RefundSucceeded, externalArg, now, refundID, RefundProcessing,
		); err != nil {
			return apperr.Internal(err)
		}
		refundedState := "refunded"
		if amountCent < paidAmount {
			refundedState = "partially_refunded"
		}
		if _, err := tx.ExecContext(
			ctx, `UPDATE payment_orders SET status = ?, updated_at = ? WHERE id = ?`,
			refundedState, now, paymentOrderID,
		); err != nil {
			return apperr.Internal(err)
		}
		if _, err := tx.ExecContext(
			ctx, `UPDATE business_orders SET payment_status = ?, order_status = ?, updated_at = ? WHERE id = ?`,
			refundedState, refundedState, now, businessOrderID,
		); err != nil {
			return apperr.Internal(err)
		}
		if orderType == orderTypeActivity && amountCent == paidAmount {
			const refundTickets = `UPDATE tickets t
				JOIN activity_orders ao ON ao.id = t.activity_order_id
				SET t.status = 'refunded', t.updated_at = ?
				WHERE ao.business_order_id = ? AND t.status IN ('pending', 'active')`
			if _, err := tx.ExecContext(ctx, refundTickets, now, businessOrderID); err != nil {
				return apperr.Internal(err)
			}
		}
		if channel == "offline" {
			if _, err := tx.ExecContext(
				ctx, `UPDATE offline_collection_orders SET status = ?, updated_at = ? WHERE payment_order_id = ?`,
				refundedState, now, paymentOrderID,
			); err != nil {
				return apperr.Internal(err)
			}
		}
		out = Refund{
			ID: refundID, RefundOrderNo: refundNo, PaymentOrderID: paymentOrderID,
			BusinessOrderID: businessOrderID, AmountCent: amountCent, Channel: channel,
			Status: RefundSucceeded, Reason: reason, CreatedAt: createdAt,
		}
		if storeID.Valid {
			out.StoreID = storeID.Int64
		}
		return nil
	})
	if err != nil {
		return Refund{}, err
	}
	return out, nil
}

func (r *storeSQLRepository) FailRefundAdmin(ctx context.Context, refundID int64, now time.Time) error {
	const q = `UPDATE refund_orders SET status = ?, updated_at = ? WHERE id = ? AND status = ?`
	if _, err := r.db.ExecContext(ctx, q, RefundFailed, now, refundID, RefundProcessing); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func creditCoinRefund(
	ctx context.Context,
	tx *sql.Tx,
	refundID, memberID, amountCent int64,
	now time.Time,
) error {
	var accountID, available int64
	const lock = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = 'coins' FOR UPDATE`
	if err := tx.QueryRowContext(ctx, lock, memberID).Scan(&accountID, &available); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.Conflict("会员金币账户不存在")
		}
		return apperr.Internal(err)
	}
	newBalance := available + amountCent
	const update = `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, update, newBalance, now, accountID); err != nil {
		return apperr.Internal(err)
	}
	const ledger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason,
		 source_type, source_id, idem_key, created_at)
		VALUES (?, ?, 'coins', 'credit', ?, ?, 'refund', 'refund_order', ?, ?, ?)`
	idemKey := fmt.Sprintf("refund_order:%d", refundID)
	if _, err := tx.ExecContext(
		ctx, ledger, accountID, memberID, amountCent, newBalance, refundID, idemKey, now,
	); err != nil {
		return mapWriteErr(err)
	}
	return nil
}

// ListPaymentOrders returns a page of payment_orders rows joined with their
// business order and store. A nil f.StoreID means no scope filter (admin
// console); a set f.StoreID pins the query to one store (store console).
func (r *storeSQLRepository) ListPaymentOrders(ctx context.Context, f PaymentOrderFilter) ([]PaymentOrder, int64, error) {
	where := "1 = 1"
	var args []any
	if f.StoreID != nil {
		where += " AND po.store_id = ?"
		args = append(args, *f.StoreID)
	}
	if f.Status != "" {
		where += " AND po.status = ?"
		args = append(args, f.Status)
	}
	var total int64
	countQ := `SELECT COUNT(*) FROM payment_orders po WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT po.id, po.payment_order_no, po.store_id, COALESCE(s.name,''), po.member_id,
			bo.id, bo.business_order_no, bo.order_type, bo.order_status, bo.payment_status,
			po.amount_cent, po.pay_method, po.status, po.created_at, po.updated_at, po.paid_at
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		LEFT JOIN stores s ON s.id = po.store_id
		WHERE ` + where + ` ORDER BY po.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Page.Limit(), f.Page.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]PaymentOrder, 0)
	for rows.Next() {
		var p PaymentOrder
		if err := rows.Scan(&p.ID, &p.PaymentOrderNo, &p.StoreID, &p.StoreName, &p.MemberID,
			&p.BusinessOrderID, &p.BusinessOrderNo, &p.OrderType, &p.BusinessStatus, &p.PaymentStatus,
			&p.AmountCent, &p.PayMethod, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.PaidAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetPaymentOrder reads a single payment_orders row joined with its business
// order and store. A nil storeID means no scope filter (admin console); a set
// storeID pins the lookup to one store (store console).
func (r *storeSQLRepository) GetPaymentOrder(ctx context.Context, id int64, storeID *int64) (PaymentOrder, error) {
	where := "po.id = ?"
	args := []any{id}
	if storeID != nil {
		where += " AND po.store_id = ?"
		args = append(args, *storeID)
	}
	q := `SELECT po.id, po.payment_order_no, po.store_id, COALESCE(s.name,''), po.member_id,
			bo.id, bo.business_order_no, bo.order_type, bo.order_status, bo.payment_status,
			po.amount_cent, po.pay_method, po.status, po.created_at, po.updated_at, po.paid_at
		FROM payment_orders po
		JOIN business_orders bo ON bo.id = po.business_order_id
		LEFT JOIN stores s ON s.id = po.store_id
		WHERE ` + where
	var p PaymentOrder
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&p.ID, &p.PaymentOrderNo, &p.StoreID, &p.StoreName, &p.MemberID,
		&p.BusinessOrderID, &p.BusinessOrderNo, &p.OrderType, &p.BusinessStatus, &p.PaymentStatus,
		&p.AmountCent, &p.PayMethod, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.PaidAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PaymentOrder{}, apperr.NotFound("payment order not found")
	}
	if err != nil {
		return PaymentOrder{}, apperr.Internal(err)
	}
	return p, nil
}

// channelForOrderType picks the refund channel from the original order type.
// Offline collections resolve their channel at callback, so default to wechat
// otherwise; a later milestone reads the settled transaction's channel.
func channelForOrderType(orderType string) string {
	if orderType == collectionType {
		return "offline"
	}
	return "wechat"
}

func mapWriteErr(err error) error {
	if platdb.IsDuplicate(err) {
		return apperr.Conflict("duplicate payment record")
	}
	return apperr.Internal(err)
}

// affectedOrConflict maps a zero-row update to a conflict.
func affectedOrConflict(result sql.Result, msg string) error {
	n, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if n == 0 {
		return apperr.Conflict(msg)
	}
	return nil
}
