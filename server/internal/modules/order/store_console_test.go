package order

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/modules/payment"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type storeConsoleMemRepo struct {
	storeID    int64
	order      StoreFoodOrderView
	prepared   FoodOrderCancellation
	completed  bool
	rolledBack bool
}

func (r *storeConsoleMemRepo) PrepareFoodOrderCancellation(_ context.Context, in FoodOrderCancellationInput) (FoodOrderCancellation, error) {
	if in.StoreID != r.storeID || in.FoodOrderID != r.order.ID {
		return FoodOrderCancellation{}, apperr.NotFound("food order not found")
	}
	return r.prepared, nil
}
func (r *storeConsoleMemRepo) CompleteFoodOrderCancellation(_ context.Context, _, _ int64, _ time.Time) error {
	r.completed = true
	r.order.Status = "cancelled"
	r.order.PaymentStatus = "refunded"
	return nil
}
func (r *storeConsoleMemRepo) RollbackFoodOrderCancellation(_ context.Context, _ int64, _ string, _ time.Time) error {
	r.rolledBack = true
	return nil
}

type storeRefundFake struct{ err error }

func (f storeRefundFake) CreateFoodOrderCancellationRefund(_ context.Context, _ int64, _ string, _ int64, _ string, _ payment.CreateRefundRequest) (payment.AdminRefundView, error) {
	if f.err != nil {
		return payment.AdminRefundView{}, f.err
	}
	return payment.AdminRefundView{ID: 91, Status: payment.RefundSucceeded}, nil
}

type storePasswordFake struct{ err error }

func (f storePasswordFake) VerifyAccountPassword(_ context.Context, _ int64, _ string) error {
	return f.err
}

func (r *storeConsoleMemRepo) ListStoreFoodOrders(_ context.Context, storeID int64, _ StoreFoodOrderFilter) ([]StoreFoodOrderView, int64, error) {
	if storeID != r.storeID {
		return []StoreFoodOrderView{}, 0, nil
	}
	return []StoreFoodOrderView{r.order}, 1, nil
}

func (r *storeConsoleMemRepo) GetStoreFoodOrder(_ context.Context, storeID, id int64) (StoreFoodOrderView, error) {
	if storeID != r.storeID || id != r.order.ID {
		return StoreFoodOrderView{}, apperr.NotFound("food order not found")
	}
	return r.order, nil
}

func (r *storeConsoleMemRepo) TransitionStoreFoodOrder(_ context.Context, storeID, id int64, from, to string) (bool, error) {
	if storeID != r.storeID || id != r.order.ID || r.order.Status != from {
		return false, nil
	}
	r.order.Status = to
	return true, nil
}

func TestStoreFoodOrderActionIsStoreScoped(t *testing.T) {
	repo := &storeConsoleMemRepo{storeID: 7, order: StoreFoodOrderView{ID: 3, Status: "preparing", PaymentStatus: "paid"}}
	svc := NewStoreConsoleService(repo, nil, nil)

	if _, err := svc.Action(context.Background(), 8, 3, "ready", "store_admin", 1, "i1", ""); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("foreign-store order must be hidden, got %v", err)
	}
	got, err := svc.Action(context.Background(), 7, 3, "ready", "store_admin", 1, "i2", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "ready" {
		t.Fatalf("expected ready, got %q", got.Status)
	}
}

func TestStoreFoodOrderActionValidatesTransitionAndPayment(t *testing.T) {
	repo := &storeConsoleMemRepo{storeID: 7, order: StoreFoodOrderView{ID: 3, Status: "ready", PaymentStatus: "unpaid"}}
	svc := NewStoreConsoleService(repo, nil, nil)

	if _, err := svc.Action(context.Background(), 7, 3, "complete", "store_admin", 1, "i1", ""); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("unpaid order must not be completed, got %v", err)
	}
	repo.order.PaymentStatus = "paid"
	if _, err := svc.Action(context.Background(), 7, 3, "prepare", "store_admin", 1, "i2", ""); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("invalid state transition must conflict, got %v", err)
	}
}

func TestStoreFoodOrderListRejectsUnknownStatus(t *testing.T) {
	svc := NewStoreConsoleService(&storeConsoleMemRepo{}, nil, nil)
	_, _, err := svc.List(context.Background(), 1, StoreFoodOrderFilter{Status: "unknown", Page: httpx.Page{Page: 1, PageSize: 20}})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestStoreFoodOrderListRejectsInvalidSearchEnumsAndRange(t *testing.T) {
	svc := NewStoreConsoleService(&storeConsoleMemRepo{}, nil, nil)
	page := httpx.Page{Page: 1, PageSize: 20}
	for _, filter := range []StoreFoodOrderFilter{
		{PaymentStatus: "expired", Page: page},
		{PayChannel: "alipay", Page: page},
	} {
		if _, _, err := svc.List(context.Background(), 1, filter); apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("expected invalid argument for %+v, got %v", filter, err)
		}
	}
	from := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	to := from.Add(-time.Hour)
	if _, _, err := svc.List(context.Background(), 1, StoreFoodOrderFilter{CreatedFrom: &from, CreatedTo: &to, Page: page}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid date range, got %v", err)
	}
}

func TestStoreFoodOrderWhereBuildsIndependentSearchConditions(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	where, args := storeFoodOrderWhere(7, StoreFoodOrderFilter{
		Status: "completed", PaymentStatus: "paid", PayChannel: "wechat",
		MemberNickname: "张", MemberPhone: "138", OrderNo: "BO2026", ItemName: "咖啡",
		CreatedFrom: &from, CreatedTo: &to,
	})
	for _, clause := range []string{
		"fo.fulfillment_status = ?", "bo.payment_status = ?", "po.pay_method = ?",
		"m.nickname", "m.phone", "bo.business_order_no LIKE", "sx.name_snapshot LIKE",
		"fo.created_at >= ?", "fo.created_at <= ?",
	} {
		if !strings.Contains(where, clause) {
			t.Fatalf("missing clause %q in %s", clause, where)
		}
	}
	if len(args) != 10 {
		t.Fatalf("expected store plus nine filter arguments, got %d: %#v", len(args), args)
	}
}

func TestStoreFoodOrderCancelCompletesRefundAndOrder(t *testing.T) {
	repo := &storeConsoleMemRepo{
		storeID:  7,
		order:    StoreFoodOrderView{ID: 3, Status: "completed", PaymentStatus: "paid"},
		prepared: FoodOrderCancellation{ID: 11, FoodOrderID: 3, PaymentOrderID: 55, AmountCent: 500, PointsEarned: 20, PointsRecovered: 20},
	}
	svc := NewStoreConsoleService(repo, storeRefundFake{}, storePasswordFake{})
	got, err := svc.Action(context.Background(), 7, 3, "cancel", "store_admin", 2, "cancel-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !repo.completed || got.Status != "cancelled" {
		t.Fatalf("cancel not completed: %+v", got)
	}
}

func TestStoreFoodOrderCancelRollsBackPointsWhenRefundFails(t *testing.T) {
	repo := &storeConsoleMemRepo{
		storeID:  7,
		order:    StoreFoodOrderView{ID: 3, Status: "completed", PaymentStatus: "paid"},
		prepared: FoodOrderCancellation{ID: 11, FoodOrderID: 3, PaymentOrderID: 55, AmountCent: 500},
	}
	svc := NewStoreConsoleService(repo, storeRefundFake{err: apperr.Internal(context.Canceled)}, storePasswordFake{})
	if _, err := svc.Action(context.Background(), 7, 3, "cancel", "store_admin", 2, "cancel-2", ""); err == nil {
		t.Fatal("expected refund error")
	}
	if !repo.rolledBack {
		t.Fatal("point clawback must roll back when refund fails")
	}
}

func TestStoreFoodOrderForceCancelRequiresPassword(t *testing.T) {
	repo := &storeConsoleMemRepo{storeID: 7, order: StoreFoodOrderView{ID: 3, Status: "completed", PaymentStatus: "paid"}}
	svc := NewStoreConsoleService(repo, storeRefundFake{}, storePasswordFake{err: apperr.Forbidden("管理员登录密码错误")})
	if _, err := svc.Action(context.Background(), 7, 3, "force-cancel", "store_admin", 2, "cancel-3", "bad"); apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected password rejection, got %v", err)
	}
}
